// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package monitor

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"github.com/ydcloud-dy/opshub/plugins/monitor/server"
)

const (
	alertRuleSchedulerInterval          = 5 * time.Second
	monitorMaintenanceSchedulerInterval = 1 * time.Minute
	alertRuleSchedulerTimeout           = 4 * time.Minute
	probeTaskSchedulerTimeout           = 2 * time.Minute
)

// Plugin 监控中心插件实现
type Plugin struct {
	db         *gorm.DB
	name       string
	ctx        context.Context
	cancelCtx  context.CancelFunc
	instanceID string
	leader     *monitorLeaderElector
}

var (
	alertRuleSchedulerRunning atomic.Bool
	probeTaskSchedulerRunning atomic.Bool
)

// New 创建插件实例
func New() *Plugin {
	return &Plugin{
		name: "monitor",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "monitor"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "监控中心插件 - 支持域名监控等功能"
}

// Version 返回插件版本
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Author 返回插件作者
func (p *Plugin) Author() string {
	return "J"
}

// Enable 启用插件
func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db

	// 自动迁移所有插件相关的表
	models := []interface{}{
		&model.DomainMonitor{},
		&model.DomainCheckHistory{},
		&model.AlertConfig{},
		&model.AlertChannel{},
		&model.AlertReceiver{},
		&model.AlertReceiverChannel{},
		&model.AlertLog{},
		&model.DataSource{},
		&model.FaultCenter{},
		&model.AlertRuleGroup{},
		&model.AlertRule{},
		&model.AlertEvent{},
		&model.ProbeTask{},
		&model.ProbeHistory{},
		&model.NoticeTemplate{},
		&model.NoticeObject{},
		&model.DutyTable{},
		&model.DutySchedule{},
	}

	// 自动迁移所有插件相关的表
	// GORM 的 AutoMigrate 会自动添加缺失的列，不会删除已有数据
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}
	if err := p.ensureDefaultMonitorData(); err != nil {
		return err
	}

	// 启动定时检查任务
	p.ctx, p.cancelCtx = context.WithCancel(context.Background())
	p.instanceID = buildMonitorSchedulerInstanceID()
	leader, err := newMonitorLeaderElector(p.ctx, p.instanceID, p.startMonitorScheduler)
	if err != nil {
		fmt.Printf("monitor scheduler leader election disabled, fallback to local scheduler: %v\n", err)
		server.SetMonitorSchedulerLeaderStatus(p.instanceID, "local-fallback", true, err)
		go p.startMonitorScheduler(p.ctx)
		return nil
	}
	p.leader = leader
	go p.leader.Start()

	return nil
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 停止定时任务
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	if p.leader != nil {
		p.leader.Stop()
	}
	return nil
}

// startMonitorScheduler 启动监控调度器
func (p *Plugin) startMonitorScheduler(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("monitor scheduler crashed: %v\n", recovered)
		}
	}()

	alertTicker := time.NewTicker(alertRuleSchedulerInterval)
	defer alertTicker.Stop()

	maintenanceTicker := time.NewTicker(monitorMaintenanceSchedulerInterval)
	defer maintenanceTicker.Stop()

	handler := server.NewHandler(p.db)
	dataSourceHandler := server.NewDataSourceHandler(p.db)

	p.runAlertRuleScheduler(ctx, dataSourceHandler)
	p.runProbeTaskScheduler(ctx, dataSourceHandler)

	for {
		select {
		case <-ctx.Done():
			return
		case <-alertTicker.C:
			p.runAlertRuleScheduler(ctx, dataSourceHandler)
			p.runProbeTaskScheduler(ctx, dataSourceHandler)
		case <-maintenanceTicker.C:
			safeMonitorSchedulerCall("domain-maintenance", func() {
				p.checkDueDomains(handler)
			})
		}
	}
}

func (p *Plugin) runAlertRuleScheduler(parentCtx context.Context, dataSourceHandler *server.DataSourceHandler) {
	if !alertRuleSchedulerRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer alertRuleSchedulerRunning.Store(false)
		safeMonitorSchedulerCall("alert-rule-evaluation", func() {
			ctx, cancel := context.WithTimeout(parentCtx, alertRuleSchedulerTimeout)
			defer cancel()
			dataSourceHandler.EvaluateDueAlertRules(ctx)
		})
	}()
}

func (p *Plugin) runProbeTaskScheduler(parentCtx context.Context, dataSourceHandler *server.DataSourceHandler) {
	if !probeTaskSchedulerRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer probeTaskSchedulerRunning.Store(false)
		safeMonitorSchedulerCall("probe-task-evaluation", func() {
			ctx, cancel := context.WithTimeout(parentCtx, probeTaskSchedulerTimeout)
			defer cancel()
			dataSourceHandler.RunDueProbeTasks(ctx)
		})
	}()
}

func safeMonitorSchedulerCall(name string, fn func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("monitor scheduler task %s crashed: %v\n", name, recovered)
		}
	}()
	fn()
}

func buildMonitorSchedulerInstanceID() string {
	hostname, _ := os.Hostname()
	if strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}
	return fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
}

// checkDueDomains 检查到期需要检查的域名
func (p *Plugin) checkDueDomains(handler *server.Handler) {
	var monitors []model.DomainMonitor
	now := time.Now()

	// 查找需要检查的域名：状态为正常或异常，且下次检查时间已过
	p.db.Where("status IN ? AND next_check <= ?", []string{"normal", "abnormal"}, now).
		Find(&monitors)

	for _, monitor := range monitors {
		// 在后台执行检查，避免阻塞
		go handler.CheckDomainByID(monitor.ID)
	}
}

func (p *Plugin) ensureDefaultMonitorData() error {
	var faultCenterCount int64
	if err := p.db.Model(&model.FaultCenter{}).Count(&faultCenterCount).Error; err != nil {
		return err
	}
	if faultCenterCount == 0 {
		if err := p.db.Create(&model.FaultCenter{
			Name:                 "默认故障中心",
			Description:          "系统默认故障中心，用于承接未指定路由的告警事件",
			NoticeObjectIDs:      "[]",
			RecoverNotify:        true,
			AggregationType:      "Rule",
			SilenceEnabled:       false,
			SilenceRules:         "[]",
			RecoverWaitSeconds:   30,
			RepeatNoticeInterval: `{"p0":30,"p1":60,"p2":120}`,
			UpgradableSeverities: `["p0","p1"]`,
			UpgradeStrategy:      `{"enabled":false,"timeout":30,"repeatInterval":60,"noticeObjectIds":[]}`,
		}).Error; err != nil {
			return err
		}
	}

	var groupCount int64
	if err := p.db.Model(&model.AlertRuleGroup{}).Count(&groupCount).Error; err != nil {
		return err
	}
	if groupCount == 0 {
		if err := p.db.Create(&model.AlertRuleGroup{
			Name:        "默认规则组",
			Description: "系统默认规则组",
			Sort:        1,
		}).Error; err != nil {
			return err
		}
	}
	if err := p.ensureDefaultNoticeTemplates(); err != nil {
		return err
	}
	return p.ensureDefaultDutyAndNoticeObject()
}

func (p *Plugin) ensureDefaultNoticeTemplates() error {
	var count int64
	if err := p.db.Model(&model.NoticeTemplate{}).Count(&count).Error; err != nil {
		return err
	}
	templates := defaultNoticeTemplates()
	if count == 0 {
		return p.db.Create(&templates).Error
	}
	for _, template := range templates {
		var existing model.NoticeTemplate
		if err := p.db.Where("name = ? AND notice_type = ?", template.Name, template.NoticeType).First(&existing).Error; err == nil {
			existing.Description = template.Description
			existing.Template = ""
			existing.TemplateFiring = template.TemplateFiring
			existing.TemplateRecover = template.TemplateRecover
			existing.EnableFeiShuJSONCard = template.EnableFeiShuJSONCard
			existing.Enabled = true
			if err := p.db.Save(&existing).Error; err != nil {
				return err
			}
			continue
		}
		var typeCount int64
		if err := p.db.Model(&model.NoticeTemplate{}).Where("notice_type = ?", template.NoticeType).Count(&typeCount).Error; err != nil {
			return err
		}
		if typeCount == 0 {
			if err := p.db.Create(&template).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func defaultNoticeTemplates() []model.NoticeTemplate {
	return []model.NoticeTemplate{
		{
			Name:                 "默认飞书模板",
			NoticeType:           "FeiShu",
			Description:          "飞书高级消息卡片模板",
			TemplateFiring:       feishuFiringTemplate(),
			TemplateRecover:      feishuRecoverTemplate(),
			EnableFeiShuJSONCard: true,
			Enabled:              true,
		},
		{
			Name:            "默认钉钉模板",
			NoticeType:      "DingDing",
			Description:     "钉钉机器人 Markdown 模板",
			TemplateFiring:  dingTalkFiringTemplate(),
			TemplateRecover: dingTalkRecoverTemplate(),
			Enabled:         true,
		},
		{
			Name:            "默认企业微信模板",
			NoticeType:      "WeChat",
			Description:     "企业微信机器人 Markdown 模板",
			TemplateFiring:  "## 🔥 OpsHub 告警中\n> 规则：${rule_name}\n> 指纹：${fingerprint}\n> 等级：<font color=\"warning\">${severity}</font>\n> 实例：${labels.instance}\n> 当前值：{{value}}\n> 开始时间：{{ .FirstTriggerTime | formatTime }}\n> 值班人员：${duty_user}\n> 通知对象：${duty_user_mentions}\n> 事件说明：${annotations}\n\n${matched_logs_block}\n\n[查看事件](${event_url})",
			TemplateRecover: "## ✅ OpsHub 已恢复\n> 规则：${rule_name}\n> 指纹：${fingerprint}\n> 等级：<font color=\"info\">${severity}</font>\n> 实例：${labels.instance}\n> 开始时间：{{ .FirstTriggerTime | formatTime }}\n> 恢复时间：{{ .RecoverTime | formatTime }}\n> 值班人员：${duty_user}\n> 通知对象：${duty_user_mentions}\n> 事件说明：${annotations}\n\n${matched_logs_block}\n\n[查看事件](${event_url})",
			Enabled:         true,
		},
		{
			Name:            "默认邮件模板",
			NoticeType:      "Email",
			Description:     "邮件告警卡片模板",
			TemplateFiring:  emailFiringTemplate(),
			TemplateRecover: emailRecoverTemplate(),
			Enabled:         true,
		},
		{
			Name:            "默认 Webhook 模板",
			NoticeType:      "WebHook",
			Description:     "通用 Webhook JSON 模板",
			TemplateFiring:  `{"status":"firing","platform":"OpsHub","ruleName":"${rule_name}","fingerprint":"${fingerprint}","severity":"${severity}","instance":"${labels.instance}","value":"{{value}}","startedAt":"{{ .FirstTriggerTime | formatTime }}","dutyUser":"${duty_user}","dutyUserMentions":"${duty_user_mentions}","annotations":"${annotations}","eventUrl":"${event_url}"}`,
			TemplateRecover: `{"status":"recovered","platform":"OpsHub","ruleName":"${rule_name}","fingerprint":"${fingerprint}","severity":"${severity}","instance":"${labels.instance}","startedAt":"{{ .FirstTriggerTime | formatTime }}","recoveredAt":"{{ .RecoverTime | formatTime }}","dutyUser":"${duty_user}","dutyUserMentions":"${duty_user_mentions}","annotations":"${annotations}","eventUrl":"${event_url}"}`,
			Enabled:         true,
		},
	}
}

func dingTalkFiringTemplate() string {
	return `### 🔥 OpsHub 告警中

> **${rule_name}**
> 等级：**${severity}**  ｜ 当前值：**${value}**
> 实例：${labels.instance}

**基础信息**

- 开始时间：{{ .FirstTriggerTime | formatTime }}
- 值班人员：${duty_user}
- 通知对象：${duty_user_dingtalk_at}
- 事件指纹：${fingerprint}

**告警详情**

> ${annotations}

${matched_logs_block}

[🔎 查看事件](${event_url})

---
OpsHub 监控中心`
}

func dingTalkRecoverTemplate() string {
	return `### ✅ OpsHub 已恢复

> **${rule_name}**
> 等级：**${severity}**
> 实例：${labels.instance}

**恢复信息**

- 开始时间：{{ .FirstTriggerTime | formatTime }}
- 恢复时间：{{ .RecoverTime | formatTime }}
- 值班人员：${duty_user}
- 通知对象：${duty_user_dingtalk_at}
- 事件指纹：${fingerprint}

**恢复详情**

> ${annotations}

${matched_logs_block}

[🔎 查看事件](${event_url})

---
OpsHub 监控中心`
}

func emailFiringTemplate() string {
	return `<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;font-family:Arial,'PingFang SC','Microsoft YaHei',sans-serif;color:#111827;">
  <tr>
    <td>
      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0">
        <tr>
          <td style="vertical-align:top;">
            <div style="font-size:13px;color:#6b7280;margin-bottom:6px;">OpsHub 监控中心</div>
            <div style="font-size:22px;font-weight:700;line-height:1.35;color:#111827;">${rule_name}</div>
          </td>
          <td align="right" style="vertical-align:top;">
            <span style="display:inline-block;background:#fef2f2;color:#b91c1c;border:1px solid #fecaca;border-radius:999px;padding:6px 12px;font-size:13px;font-weight:700;">告警中 · ${severity}</span>
          </td>
        </tr>
      </table>
      <div style="height:1px;background:#e5e7eb;margin:20px 0;"></div>
      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="border-collapse:separate;border-spacing:0 10px;">
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">告警实例</td>
          <td style="color:#111827;font-size:14px;font-weight:600;">${labels.instance}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">当前数值</td>
          <td style="color:#dc2626;font-size:18px;font-weight:700;">${value}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">开始时间</td>
          <td style="color:#111827;font-size:14px;">{{ .FirstTriggerTime | formatTime }}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">值班人员</td>
          <td style="color:#111827;font-size:14px;">${duty_user}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">事件数量</td>
          <td style="color:#111827;font-size:14px;">${event_count}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">事件指纹</td>
          <td style="color:#475569;font-size:13px;font-family:Menlo,Consolas,monospace;word-break:break-all;">${fingerprint}</td>
        </tr>
      </table>
      <div style="background:#f9fafb;border:1px solid #e5e7eb;border-radius:10px;padding:14px 16px;margin-top:18px;">
        <div style="font-size:14px;font-weight:700;color:#111827;margin-bottom:8px;">告警详情</div>
        <div style="font-size:14px;line-height:1.7;color:#374151;white-space:pre-line;">${annotations}</div>
      </div>
      <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:14px 16px;margin-top:12px;">
        <div style="font-size:14px;font-weight:700;color:#111827;margin-bottom:8px;">回调查询</div>
        <div style="font-size:13px;line-height:1.7;color:#475569;white-space:pre-line;">${callback_summary}
${callback_details}</div>
      </div>
      <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin-top:22px;">
        <tr>
          <td bgcolor="#111827" style="border-radius:8px;">
            <a href="${event_url}" style="display:inline-block;padding:11px 18px;color:#ffffff;text-decoration:none;font-size:14px;font-weight:700;">${event_link_text}</a>
          </td>
        </tr>
      </table>
      <div style="height:1px;background:#e5e7eb;margin:22px 0 14px;"></div>
      <div style="font-size:12px;color:#94a3b8;">OpsHub 运维平台自动发送，请勿直接回复。</div>
    </td>
  </tr>
</table>`
}

func emailRecoverTemplate() string {
	return `<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;font-family:Arial,'PingFang SC','Microsoft YaHei',sans-serif;color:#111827;">
  <tr>
    <td>
      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0">
        <tr>
          <td style="vertical-align:top;">
            <div style="font-size:13px;color:#6b7280;margin-bottom:6px;">OpsHub 监控中心</div>
            <div style="font-size:22px;font-weight:700;line-height:1.35;color:#111827;">${rule_name}</div>
          </td>
          <td align="right" style="vertical-align:top;">
            <span style="display:inline-block;background:#ecfdf5;color:#047857;border:1px solid #bbf7d0;border-radius:999px;padding:6px 12px;font-size:13px;font-weight:700;">已恢复 · ${severity}</span>
          </td>
        </tr>
      </table>
      <div style="height:1px;background:#e5e7eb;margin:20px 0;"></div>
      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="border-collapse:separate;border-spacing:0 10px;">
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">告警实例</td>
          <td style="color:#111827;font-size:14px;font-weight:600;">${labels.instance}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">开始时间</td>
          <td style="color:#111827;font-size:14px;">{{ .FirstTriggerTime | formatTime }}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">恢复时间</td>
          <td style="color:#047857;font-size:14px;font-weight:700;">{{ .RecoverTime | formatTime }}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">值班人员</td>
          <td style="color:#111827;font-size:14px;">${duty_user}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">事件数量</td>
          <td style="color:#111827;font-size:14px;">${event_count}</td>
        </tr>
        <tr>
          <td style="width:92px;color:#6b7280;font-size:13px;">事件指纹</td>
          <td style="color:#475569;font-size:13px;font-family:Menlo,Consolas,monospace;word-break:break-all;">${fingerprint}</td>
        </tr>
      </table>
      <div style="background:#f9fafb;border:1px solid #e5e7eb;border-radius:10px;padding:14px 16px;margin-top:18px;">
        <div style="font-size:14px;font-weight:700;color:#111827;margin-bottom:8px;">恢复详情</div>
        <div style="font-size:14px;line-height:1.7;color:#374151;white-space:pre-line;">${annotations}</div>
      </div>
      <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:14px 16px;margin-top:12px;">
        <div style="font-size:14px;font-weight:700;color:#111827;margin-bottom:8px;">回调查询</div>
        <div style="font-size:13px;line-height:1.7;color:#475569;white-space:pre-line;">${callback_summary}
${callback_details}</div>
      </div>
      <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin-top:22px;">
        <tr>
          <td bgcolor="#047857" style="border-radius:8px;">
            <a href="${event_url}" style="display:inline-block;padding:11px 18px;color:#ffffff;text-decoration:none;font-size:14px;font-weight:700;">${event_link_text}</a>
          </td>
        </tr>
      </table>
      <div style="height:1px;background:#e5e7eb;margin:22px 0 14px;"></div>
      <div style="font-size:12px;color:#94a3b8;">OpsHub 运维平台自动发送，请勿直接回复。</div>
    </td>
  </tr>
</table>`
}

func feishuFiringTemplate() string {
	return `{
  "schema": "2.0",
  "config": {
    "width_mode": "fill",
    "enable_forward": true
  },
  "header": {
    "template": "red",
    "title": {
      "tag": "plain_text",
      "content": "【告警中】- OpsHub 业务系统 🔥"
    }
  },
  "body": {
    "elements": [
      { "tag": "markdown", "content": "**🤖 告警类型:** ${rule_name}\n**📌 告警等级:** ${severity}\n**🖥 告警主机:** ${labels.instance}\n**📈 当前数值:** ${value}\n**🕘 开始时间:** {{ .FirstTriggerTime | formatTime }}\n**👤 值班人员:** ${duty_user_feishu_at}\n**🫧 告警指纹:** ${fingerprint}" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "**📝 告警事件**\n${annotations}\n${matched_logs_block}" },
      { "tag": "markdown", "content": "**📊 回调查询**\n${callback_summary}\n${callback_links}" },
      { "tag": "markdown", "content": "[${event_link_text}](${event_url})" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队" }
    ]
  }
}`
}

func feishuRecoverTemplate() string {
	return `{
  "schema": "2.0",
  "config": {
    "width_mode": "fill",
    "enable_forward": true
  },
  "header": {
    "template": "green",
    "title": {
      "tag": "plain_text",
      "content": "【已恢复】- OpsHub 业务系统 ✨"
    }
  },
  "body": {
    "elements": [
      { "tag": "markdown", "content": "**🤖 告警类型:** ${rule_name}\n**📌 告警等级:** ${severity}\n**🖥 告警主机:** ${labels.instance}\n**🕘 开始时间:** {{ .FirstTriggerTime | formatTime }}\n**🕘 恢复时间:** {{ .RecoverTime | formatTime }}\n**👤 值班人员:** ${duty_user_feishu_at}\n**🫧 告警指纹:** ${fingerprint}" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "**📝 告警事件**\n${annotations}\n${matched_logs_block}" },
      { "tag": "markdown", "content": "**📊 回调查询**\n${callback_summary}\n${callback_links}" },
      { "tag": "markdown", "content": "[${event_link_text}](${event_url})" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队" }
    ]
  }
}`
}

func (p *Plugin) ensureDefaultDutyAndNoticeObject() error {
	var duty model.DutyTable
	if err := p.db.Where("name = ?", "默认值班表").First(&duty).Error; err != nil {
		if err := p.db.Create(&model.DutyTable{
			Name:            "默认值班表",
			Description:     "默认值班表，可在值班表页面维护每日值班人员",
			ManagerUsername: "admin",
			Enabled:         true,
		}).Error; err != nil {
			return err
		}
		if err := p.db.Where("name = ?", "默认值班表").First(&duty).Error; err != nil {
			return err
		}
	}

	var scheduleCount int64
	if err := p.db.Model(&model.DutySchedule{}).Where("duty_table_id = ?", duty.ID).Count(&scheduleCount).Error; err != nil {
		return err
	}
	if scheduleCount == 0 {
		if err := p.db.Create(&model.DutySchedule{
			DutyTableID: duty.ID,
			DutyDate:    time.Now().Format("2006-01-02"),
			Users:       `[{"id":1,"username":"admin","realName":"管理员"}]`,
			Status:      "active",
		}).Error; err != nil {
			return err
		}
	}

	var noticeObject model.NoticeObject
	if err := p.db.Where("name = ?", "默认通知对象").First(&noticeObject).Error; err != nil {
		var template model.NoticeTemplate
		_ = p.db.Where("notice_type = ?", "FeiShu").Order("id ASC").First(&template).Error
		templateID := ""
		if template.ID > 0 {
			templateID = uintToString(template.ID)
		}
		route := `[{"noticeType":"FeiShu","noticeTemplateId":"` + templateID + `","severitys":["P0","P1","P2"],"hook":"","headers":{},"sign":"","subject":"","to":[],"cc":[],"effectiveTime":{"week":[],"startTime":"","endTime":""},"enabled":true}]`
		if err := p.db.Create(&model.NoticeObject{
			UUID:        "default-notice-object",
			Name:        "默认通知对象",
			Description: "默认通知对象，绑定默认值班表和默认飞书模板",
			DutyTableID: duty.ID,
			Routes:      route,
			Enabled:     true,
			LastStatus:  "ready",
		}).Error; err != nil {
			return err
		}
		if err := p.db.Where("name = ?", "默认通知对象").First(&noticeObject).Error; err != nil {
			return err
		}
	}

	if noticeObject.ID > 0 {
		objectIDs := `[` + uintToString(noticeObject.ID) + `]`
		if err := p.db.Model(&model.FaultCenter{}).
			Where("notice_object_ids = '' OR notice_object_ids IS NULL OR notice_object_ids = '[]'").
			Update("notice_object_ids", objectIDs).Error; err != nil {
			return err
		}
	}
	return nil
}

func uintToString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:       "监控中心",
			Path:       "/monitor",
			Icon:       "Monitor",
			Sort:       20,
			ParentPath: "",
		},
		{
			Name:       "故障中心",
			Path:       "/monitor/fault-centers",
			Icon:       "Operation",
			Sort:       1,
			ParentPath: "/monitor",
		},
		{
			Name:       "数据源管理",
			Path:       "/monitor/datasources",
			Icon:       "Connection",
			Sort:       2,
			ParentPath: "/monitor",
		},
		{
			Name:       "告警规则",
			Path:       "/monitor/rules",
			Icon:       "Warning",
			Sort:       3,
			ParentPath: "/monitor",
		},
		{
			Name:       "拨测任务",
			Path:       "/monitor/probe-tasks",
			Icon:       "Odometer",
			Sort:       4,
			ParentPath: "/monitor",
		},
		{
			Name:       "即时拨测",
			Path:       "/monitor/instant-probe",
			Icon:       "VideoPlay",
			Sort:       5,
			ParentPath: "/monitor",
		},
		{
			Name:       "通知对象",
			Path:       "/monitor/notice-objects",
			Icon:       "Message",
			Sort:       6,
			ParentPath: "/monitor",
		},
		{
			Name:       "通知模板",
			Path:       "/monitor/notice-templates",
			Icon:       "DocumentCopy",
			Sort:       7,
			ParentPath: "/monitor",
		},
		{
			Name:       "值班表",
			Path:       "/monitor/duty-tables",
			Icon:       "Calendar",
			Sort:       8,
			ParentPath: "/monitor",
		},
	}
}
