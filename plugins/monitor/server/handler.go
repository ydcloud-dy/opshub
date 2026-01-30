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

package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"github.com/ydcloud-dy/opshub/plugins/monitor/repository"
	"github.com/ydcloud-dy/opshub/plugins/monitor/service"
	"gorm.io/gorm"
)

// Handler 域名监控处理器
type Handler struct {
	db           *gorm.DB
	repo         *repository.DomainMonitorRepository
	alertService *service.AlertService
}

// NewHandler 创建处理器
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:           db,
		repo:         repository.NewDomainMonitorRepository(db),
		alertService: service.NewAlertService(),
	}
}

// ListDomains 获取域名监控列表
// @Summary 获取域名监控列表
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Success 200 {array} model.DomainMonitor
// @Router /monitor/domains [get]
func (h *Handler) ListDomains(c *gin.Context) {
	monitors, err := h.repo.GetAll()
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取域名列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    monitors,
	})
}

// GetDomain 获取域名监控详情
// @Summary 获取域名监控详情
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id} [get]
func (h *Handler) GetDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    monitor,
	})
}

// CreateDomain 创建域名监控
// @Summary 创建域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param monitor body model.DomainMonitor true "域名监控信息"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains [post]
func (h *Handler) CreateDomain(c *gin.Context) {
	var req model.DomainMonitor
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 检查域名是否已存在
	existing, err := h.repo.GetByDomain(req.Domain)
	if err == nil && existing != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "该域名已在监控中",
		})
		return
	}

	// 设置初始状态
	req.Status = "unknown"
	req.ResponseTime = 0
	now := time.Now()
	req.LastCheck = &now

	// 创建监控记录
	if err := h.repo.Create(&req); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "创建域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	// 创建成功后立即执行一次域名检查
	result, err := h.performCheck(&req)
	if err != nil {
		// 即使检查失败，记录也已创建，只是保持unknown状态
		// 可以选择记录日志或直接忽略
	} else {
		// 保存检查历史
		h.saveCheckHistory(&req, result)

		// 更新检查结果
		req.Status = result.Status
		req.ResponseTime = result.ResponseTime
		req.SSLValid = result.SSLValid
		req.SSLExpiry = result.SSLExpiry
		checkTime := time.Now()
		req.LastCheck = &checkTime

		// 计算下次检查时间
		nextCheck := checkTime.Add(time.Duration(req.CheckInterval) * time.Second)
		req.NextCheck = &nextCheck

		// 保存更新后的状态
		h.repo.Update(&req)
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    req,
	})
}

// UpdateDomain 更新域名监控
// @Summary 更新域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Param monitor body model.DomainMonitor true "域名监控信息"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id} [put]
func (h *Handler) UpdateDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取现有记录
	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	var req model.DomainMonitor
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 更新字段
	monitor.CheckInterval = req.CheckInterval
	monitor.EnableSSL = req.EnableSSL
	monitor.EnableAlert = req.EnableAlert
	monitor.ResponseThreshold = req.ResponseThreshold
	monitor.SSLExpiryDays = req.SSLExpiryDays

	// 如果域名改变，检查新域名是否已存在
	if req.Domain != monitor.Domain {
		existing, err := h.repo.GetByDomain(req.Domain)
		if err == nil && existing != nil && existing.ID != id {
			c.JSON(400, gin.H{
				"code":    400,
				"message": "该域名已在监控中",
			})
			return
		}
		monitor.Domain = req.Domain
	}

	// 保存更新
	if err := h.repo.Update(monitor); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    monitor,
	})
}

// DeleteDomain 删除域名监控
// @Summary 删除域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {string} string "success"
// @Router /monitor/domains/{id} [delete]
func (h *Handler) DeleteDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "删除域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// CheckDomain 立即检查域名
// @Summary 立即检查域名
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id}/check [post]
func (h *Handler) CheckDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取监控记录
	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	// 执行域名检查
	result, err := h.performCheck(monitor)
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "域名检查失败",
			"error":   err.Error(),
		})
		return
	}

	// 保存检查历史
	h.saveCheckHistory(monitor, result)

	// 更新监控记录
	monitor.Status = result.Status
	monitor.ResponseTime = result.ResponseTime
	monitor.SSLValid = result.SSLValid
	monitor.SSLExpiry = result.SSLExpiry
	now := time.Now()
	monitor.LastCheck = &now

	// 计算下次检查时间
	nextCheck := now.Add(time.Duration(monitor.CheckInterval) * time.Second)
	monitor.NextCheck = &nextCheck

	if err := h.repo.Update(monitor); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	// 检查并发送告警
	go h.checkAndSendAlert(monitor, result)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "检查完成",
		"data":    monitor,
	})
}

// GetStats 获取统计数据
// @Summary 获取域名监控统计数据
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int64
// @Router /monitor/domains/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.repo.GetStats()
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取统计数据失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// CheckResult 检查结果
type CheckResult struct {
	Status       string     `json:"status"`       // normal, abnormal
	ResponseTime int        `json:"responseTime"` // 响应时间(ms)
	SSLValid     bool       `json:"sslValid"`     // SSL是否有效
	SSLExpiry    *time.Time `json:"sslExpiry"`    // SSL过期时间
	StatusCode   int        `json:"statusCode"`   // HTTP状态码
	ErrorMessage string     `json:"errorMessage"` // 错误信息
}

// performCheck 执行域名检查
func (h *Handler) performCheck(monitor *model.DomainMonitor) (*CheckResult, error) {
	result := &CheckResult{
		Status:       "abnormal",
		ResponseTime: 0,
		SSLValid:     false,
		SSLExpiry:    nil,
		StatusCode:   0,
		ErrorMessage: "",
	}

	// 解析域名和端口
	host := monitor.Domain
	port := 443
	if strings.Contains(monitor.Domain, ":") {
		parts := strings.Split(monitor.Domain, ":")
		host = parts[0]
		p, err := strconv.Atoi(parts[1])
		if err == nil && p > 0 {
			port = p
		}
	}

	// 构造目标 URL
	url := fmt.Sprintf("https://%s", host)
	if port != 443 {
		url = fmt.Sprintf("https://%s:%d", host, port)
	}

	// 记录开始时间
	start := time.Now()

	// 创建HTTP客户端，跳过SSL验证(用于测试连接)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证，只测试连接
			},
		},
	}

	// 发送请求
	resp, err := client.Get(url)
	if err != nil {
		// 检查是否是网络错误
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.ResponseTime = int(10 * time.Second / time.Millisecond)
			result.ErrorMessage = "请求超时"
		} else {
			result.ErrorMessage = err.Error()
		}
		return result, nil
	}
	defer resp.Body.Close()

	// 计算响应时间
	result.ResponseTime = int(time.Since(start).Milliseconds())
	result.StatusCode = resp.StatusCode

	// 检查响应状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "normal"
	} else {
		result.Status = "abnormal"
		result.ErrorMessage = fmt.Sprintf("HTTP状态码异常: %d", resp.StatusCode)
	}

	// 如果启用SSL检查，验证SSL证书
	if monitor.EnableSSL {
		// 重新发送请求获取证书信息
		addr := fmt.Sprintf("%s:%d", host, port)
		conn, err := tls.Dial("tcp", addr, &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host, // 保证 SNI 正确
		})
		if err != nil {
			result.SSLValid = false
		} else {
			state := conn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				cert := state.PeerCertificates[0]
				result.SSLValid = true
				result.SSLExpiry = &cert.NotAfter

				// 检查证书是否即将过期(30天内)
				if time.Until(cert.NotAfter) < 30*24*time.Hour {
					result.SSLValid = false
				}
			}
			conn.Close()
		}
	}

	return result, nil
}

// checkAndSendAlert 检查并发送告警
func (h *Handler) checkAndSendAlert(monitor *model.DomainMonitor, result *CheckResult) {
	// 如果未启用告警，直接返回
	if !monitor.EnableAlert {
		return
	}

	// 检查告警频率控制
	if h.shouldSuppressAlert(monitor.ID, result.Status) {
		return
	}

	// 检查各种告警条件
	alerts := h.checkAlertConditions(monitor, result)

	// 如果有告警，发送通知
	if len(alerts) > 0 {
		for _, alert := range alerts {
			h.sendAlert(monitor, alert)
		}
	}
}

// checkAlertConditions 检查告警条件
func (h *Handler) checkAlertConditions(monitor *model.DomainMonitor, result *CheckResult) []service.AlertMessage {
	var alerts []service.AlertMessage
	now := time.Now()

	// 1. 域名无法访问告警
	if result.Status == "abnormal" && result.ResponseTime == 0 {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "domain_down",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "域名无法访问或响应超时",
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	// 2. 响应时间过高告警
	if monitor.ResponseThreshold != nil && result.ResponseTime > *monitor.ResponseThreshold {
		alerts = append(alerts, service.AlertMessage{
			AlertType:    "high_response_time",
			Domain:       monitor.Domain,
			Status:       result.Status,
			Message:      fmt.Sprintf("响应时间 %dms 超过阈值 %dms", result.ResponseTime, *monitor.ResponseThreshold),
			ResponseTime: result.ResponseTime,
			Timestamp:    now.Format("2006-01-02 15:04:05"),
		})
	}

	// 3. SSL证书即将过期告警
	if monitor.EnableSSL && result.SSLExpiry != nil {
		daysUntilExpiry := int(time.Until(*result.SSLExpiry).Hours() / 24)
		if monitor.SSLExpiryDays != nil && daysUntilExpiry <= *monitor.SSLExpiryDays && daysUntilExpiry > 0 {
			alerts = append(alerts, service.AlertMessage{
				AlertType: "ssl_expiring",
				Domain:    monitor.Domain,
				Status:    result.Status,
				Message:   fmt.Sprintf("SSL证书将在 %d 天后过期", daysUntilExpiry),
				SSLExpiry: result.SSLExpiry.Format("2006-01-02 15:04:05"),
				Timestamp: now.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 4. SSL证书已过期告警
	if monitor.EnableSSL && result.SSLExpiry != nil && result.SSLExpiry.Before(now) {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "ssl_expired",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "SSL证书已过期",
			SSLExpiry: result.SSLExpiry.Format("2006-01-02 15:04:05"),
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	// 5. SSL证书无效告警
	if monitor.EnableSSL && !result.SSLValid {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "ssl_invalid",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "SSL证书无效或无法验证",
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	return alerts
}

// sendAlert 发送告警
func (h *Handler) sendAlert(monitor *model.DomainMonitor, alert service.AlertMessage) {
	// 1. 获取启用的告警通道
	var channels []model.AlertChannel
	if err := h.db.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		h.logAlert(monitor.ID, alert, "failed", "", fmt.Sprintf("获取告警通道失败: %v", err))
		return
	}

	// 2. 获取启用的告警接收人
	var receivers []model.AlertReceiver
	if err := h.db.Find(&receivers).Error; err != nil {
		h.logAlert(monitor.ID, alert, "failed", "", fmt.Sprintf("获取告警接收人失败: %v", err))
		return
	}

	// 如果没有配置通道或接收人，记录失败日志
	if len(channels) == 0 {
		h.logAlert(monitor.ID, alert, "failed", "", "未配置启用的告警通道")
		return
	}
	if len(receivers) == 0 {
		h.logAlert(monitor.ID, alert, "failed", "", "未配置告警接收人")
		return
	}

	// 3. 构建告警通道配置
	var channelConfig service.AlertChannelConfig
	var emailReceivers []string

	for _, channel := range channels {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
			continue
		}

		switch channel.ChannelType {
		case "email":
			if smtpHost, ok := config["smtpHost"].(string); ok {
				channelConfig.SMTPHost = smtpHost
			}
			if smtpPort, ok := config["smtpPort"].(float64); ok {
				channelConfig.SMTPPort = int(smtpPort)
			}
			if smtpUser, ok := config["smtpUser"].(string); ok {
				channelConfig.SMTPUser = smtpUser
			}
			if smtpPassword, ok := config["smtpPassword"].(string); ok {
				channelConfig.SMTPPassword = smtpPassword
			}
			if fromEmail, ok := config["fromEmail"].(string); ok {
				channelConfig.FromEmail = fromEmail
			}
			if fromName, ok := config["fromName"].(string); ok {
				channelConfig.FromName = fromName
			}
		case "webhook":
			if webhookURL, ok := config["webhookUrl"].(string); ok {
				channelConfig.WebhookURL = webhookURL
			}
		case "wechat":
			if wechatWebhook, ok := config["wechatWebhook"].(string); ok {
				channelConfig.WeChatWebhook = wechatWebhook
			}
		case "dingtalk":
			if dingtalkWebhook, ok := config["dingtalkWebhook"].(string); ok {
				channelConfig.DingTalkWebhook = dingtalkWebhook
			}
			if dingtalkSecret, ok := config["dingtalkSecret"].(string); ok {
				channelConfig.DingTalkSecret = dingtalkSecret
			}
		case "feishu":
			if feishuWebhook, ok := config["feishuWebhook"].(string); ok {
				channelConfig.FeishuWebhook = feishuWebhook
			}
		}
	}

	// 4. 获取接收人-通道关联关系（用于@提醒）
	var receiverChannels []model.AlertReceiverChannel
	if err := h.db.Find(&receiverChannels).Error; err != nil {
		// 如果查询关联失败，继续使用旧的方式发送（向后兼容）
		receiverChannels = []model.AlertReceiverChannel{}
	}

	// 5. 构建接收人-通道关联信息
	receiverMap := make(map[uint]*model.AlertReceiver)
	for i := range receivers {
		receiverMap[receivers[i].ID] = &receivers[i]
	}

	channelMap := make(map[uint]*model.AlertChannel)
	for i := range channels {
		channelMap[channels[i].ID] = &channels[i]
	}

	// 构建有关联关系的接收人-通道列表
	var relations []service.ReceiverChannelRelation
	for _, rc := range receiverChannels {
		receiver, receiverExists := receiverMap[rc.ReceiverID]
		channel, channelExists := channelMap[rc.ChannelID]

		if !receiverExists || !channelExists {
			continue
		}

		// 只添加有效的关联（接收人启用对应通道）
		shouldAdd := false
		switch channel.ChannelType {
		case "email":
			shouldAdd = receiver.EnableEmail && receiver.Email != ""
		case "feishu":
			shouldAdd = receiver.EnableFeishu && receiver.FeishuID != ""
		case "dingtalk":
			shouldAdd = receiver.EnableDingTalk && (receiver.DingTalkID != "" || receiver.Phone != "")
		case "wechat":
			shouldAdd = receiver.EnableWeChat && receiver.WeChatID != ""
		}

		if shouldAdd {
			relations = append(relations, service.ReceiverChannelRelation{
				ReceiverID:  rc.ReceiverID,
				ChannelID:   rc.ChannelID,
				ChannelType: channel.ChannelType,
				Receiver: service.ReceiverInfo{
					ID:         receiver.ID,
					Name:       receiver.Name,
					Email:      receiver.Email,
					Phone:      receiver.Phone,
					FeishuID:   receiver.FeishuID,
					DingTalkID: receiver.DingTalkID,
					WeChatID:   receiver.WeChatID,
				},
				ChannelConfig: rc.Config,
			})
		}
	}

	// 6. 发送告警
	var err error
	if len(relations) > 0 {
		// 如果有关联关系，使用新的方式发送（支持@提醒）
		err = h.alertService.SendAlert(alert, channelConfig, emailReceivers, relations)
	} else {
		// 否则使用旧的方式发送（向后兼容）
		err = h.alertService.SendAlert(alert, channelConfig, emailReceivers)
	}

	// 7. 记录发送结果
	if err != nil {
		h.logAlert(monitor.ID, alert, "failed", channels[0].ChannelType, err.Error())
	} else {
		h.logAlert(monitor.ID, alert, "success", channels[0].ChannelType, "")
	}
}

// logAlert 记录告警日志
func (h *Handler) logAlert(domainMonitorID uint, alert service.AlertMessage, status, channel, errorMsg string) {
	alertLog := &model.AlertLog{
		DomainMonitorID: domainMonitorID,
		Domain:          alert.Domain,
		AlertType:       alert.AlertType,
		Status:          status,
		Message:         alert.Message,
		ChannelType:     channel,
		ErrorMsg:        errorMsg,
		SentAt:          time.Now(),
	}

	h.db.Create(alertLog)
}

// shouldSuppressAlert 检查是否应该抑制告警（告警频率控制）
func (h *Handler) shouldSuppressAlert(domainMonitorID uint, status string) bool {
	// 查询最近的告警日志
	var recentAlertCount int64
	h.db.Model(&model.AlertLog{}).
		Where("domain_monitor_id = ? AND alert_type = ? AND status = ? AND sent_at > ?",
			domainMonitorID, "domain_down", status, time.Now().Add(-10*time.Minute)).
		Count(&recentAlertCount)

	return recentAlertCount > 0
}

// CheckDomainByID 定时任务调用的域名检查方法
func (h *Handler) CheckDomainByID(id uint) {
	// 获取监控记录
	monitor, err := h.repo.GetByID(id)
	if err != nil {
		return
	}

	// 只检查非暂停状态的监控
	if monitor.Status == "paused" {
		return
	}

	// 执行域名检查
	result, err := h.performCheck(monitor)
	if err != nil {
		return
	}

	// 保存检查历史
	h.saveCheckHistory(monitor, result)

	// 更新监控记录
	monitor.Status = result.Status
	monitor.ResponseTime = result.ResponseTime
	monitor.SSLValid = result.SSLValid
	monitor.SSLExpiry = result.SSLExpiry
	now := time.Now()
	monitor.LastCheck = &now

	// 计算下次检查时间
	nextCheck := now.Add(time.Duration(monitor.CheckInterval) * time.Second)
	monitor.NextCheck = &nextCheck

	h.repo.Update(monitor)

	// 检查并发送告警
	h.checkAndSendAlert(monitor, result)
}

// 每个域名保留的最大历史记录数
const maxHistoryPerDomain = 100

// saveCheckHistory 保存检查历史记录
func (h *Handler) saveCheckHistory(monitor *model.DomainMonitor, result *CheckResult) {
	history := &model.DomainCheckHistory{
		DomainID:     monitor.ID,
		Domain:       monitor.Domain,
		Status:       result.Status,
		ResponseTime: result.ResponseTime,
		SSLValid:     result.SSLValid,
		SSLExpiry:    result.SSLExpiry,
		StatusCode:   result.StatusCode,
		ErrorMessage: result.ErrorMessage,
		CheckedAt:    time.Now(),
	}

	h.db.Create(history)

	// 清理旧的历史记录，只保留最近的 maxHistoryPerDomain 条
	h.cleanupOldHistory(monitor.ID)
}

// cleanupOldHistory 清理旧的检查历史记录
func (h *Handler) cleanupOldHistory(domainID uint) {
	// 统计该域名的历史记录数量
	var count int64
	h.db.Model(&model.DomainCheckHistory{}).Where("domain_id = ?", domainID).Count(&count)

	// 如果超过限制，删除最旧的记录
	if count > maxHistoryPerDomain {
		// 找到要保留的最小ID（保留最新的 maxHistoryPerDomain 条）
		var minKeepID uint
		h.db.Model(&model.DomainCheckHistory{}).
			Where("domain_id = ?", domainID).
			Order("checked_at DESC").
			Offset(maxHistoryPerDomain).
			Limit(1).
			Pluck("id", &minKeepID)

		// 删除比这个ID更旧的记录
		if minKeepID > 0 {
			h.db.Where("domain_id = ? AND id <= ?", domainID, minKeepID).
				Delete(&model.DomainCheckHistory{})
		}
	}
}

// GetCheckHistory 获取域名检查历史
// @Summary 获取域名检查历史
// @Description 获取指定域名的检查历史记录
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/monitor/domains/{id}/history [get]
func (h *Handler) GetCheckHistory(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 20
	if p, err := fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &page); err != nil || p == 0 {
		page = 1
	}
	if ps, err := fmt.Sscanf(c.DefaultQuery("pageSize", "20"), "%d", &pageSize); err != nil || ps == 0 {
		pageSize = 20
	}

	// 限制每页最大数量
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询检查历史
	var histories []model.DomainCheckHistory
	var total int64

	// 统计总数
	h.db.Model(&model.DomainCheckHistory{}).Where("domain_id = ?", id).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	if err := h.db.Where("domain_id = ?", id).
		Order("checked_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&histories).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取检查历史失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     histories,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
