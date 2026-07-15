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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	assetmodel "github.com/ydcloud-dy/opshub/internal/biz/asset"
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	rbacmodel "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	aiopsdata "github.com/ydcloud-dy/opshub/internal/data/aiops"
	k8smodels "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	monitormodel "github.com/ydcloud-dy/opshub/plugins/monitor/model"
	sslmodel "github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	taskmodel "github.com/ydcloud-dy/opshub/plugins/task/model"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type assistantPlatformContext struct {
	UserID          uint             `json:"-"`
	CollectedAt     string           `json:"collectedAt"`
	Intent          string           `json:"intent"`
	Question        string           `json:"question"`
	PlatformSummary map[string]any   `json:"platformSummary,omitempty"`
	ModuleKnowledge []map[string]any `json:"moduleKnowledge,omitempty"`
	SystemSummary   map[string]any   `json:"systemSummary,omitempty"`
	HostSummary     map[string]any   `json:"hostSummary,omitempty"`
	Hosts           []map[string]any `json:"hosts,omitempty"`
	ClusterSummary  map[string]any   `json:"clusterSummary,omitempty"`
	Clusters        []map[string]any `json:"clusters,omitempty"`
	MonitorSummary  map[string]any   `json:"monitorSummary,omitempty"`
	AlertSummary    map[string]any   `json:"alertSummary,omitempty"`
	RecentAlerts    []map[string]any `json:"recentAlerts,omitempty"`
	CertSummary     map[string]any   `json:"certSummary,omitempty"`
	TaskSummary     map[string]any   `json:"taskSummary,omitempty"`
	AuditSummary    map[string]any   `json:"auditSummary,omitempty"`
	AISummary       map[string]any   `json:"aiSummary,omitempty"`
	Errors          []string         `json:"errors,omitempty"`
}

func (s *Service) collectAssistantPlatformContext(ctx context.Context, sessionID, excludeMessageID uint, question string) assistantPlatformContext {
	resolvedQuestion := strings.TrimSpace(question)
	if sessionID != 0 && shouldReuseAssistantContext(resolvedQuestion) {
		if previous := s.latestSessionUserQuestion(ctx, sessionID, excludeMessageID); previous != "" {
			resolvedQuestion = previous
		}
	}

	intent := detectAssistantIntent(resolvedQuestion)
	result := assistantPlatformContext{
		CollectedAt: time.Now().Format("2006-01-02 15:04:05"),
		Intent:      intent,
		Question:    truncateText(resolvedQuestion, 1000),
	}
	result.ModuleKnowledge = selectAssistantModuleKnowledge(resolvedQuestion)
	if intent == "general" && isOpsHubPlatformQuestion(resolvedQuestion) {
		result.Intent = "platform"
		intent = "platform"
		if len(result.ModuleKnowledge) == 0 {
			result.ModuleKnowledge = selectAssistantModuleKnowledge("opshub 平台 所有模块")
		}
	}
	if intent == "general" {
		return result
	}
	s.collectAssistantOverviewContext(ctx, &result)
	if intent == "platform" || intent == "system" {
		s.collectAssistantSystemContext(ctx, &result)
	}
	if intent == "platform" || intent == "hosts" {
		s.collectAssistantHostContext(ctx, &result)
	}
	if intent == "platform" || intent == "kubernetes" {
		s.collectAssistantClusterContext(ctx, &result)
	}
	if intent == "platform" || intent == "monitor" || intent == "alerts" {
		s.collectAssistantMonitorContext(ctx, &result)
		s.collectAssistantAlertContext(ctx, &result)
	}
	if intent == "platform" || intent == "certificates" {
		s.collectAssistantCertContext(ctx, &result)
	}
	if intent == "platform" || intent == "tasks" {
		s.collectAssistantTaskContext(ctx, &result)
	}
	if intent == "platform" || intent == "audit" {
		s.collectAssistantAuditContext(ctx, &result)
	}
	if intent == "platform" || intent == "aiops" {
		s.collectAssistantAIContext(ctx, &result)
	}
	return result
}

func (s *Service) latestSessionUserQuestion(ctx context.Context, sessionID, excludeMessageID uint) string {
	if sessionID == 0 {
		return ""
	}
	query := s.db.WithContext(ctx).
		Model(&aiopsdata.Message{}).
		Where("session_id = ? AND role = ? AND status = ?", sessionID, "user", "success")
	if excludeMessageID > 0 {
		query = query.Where("id < ?", excludeMessageID)
	}
	var msg aiopsdata.Message
	if err := query.Order("id DESC").First(&msg).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(msg.Content)
}

func shouldReuseAssistantContext(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return true
	}
	if containsAnyText(text, []string{
		"继续生成",
		"继续输出",
		"继续回答",
		"继续分析",
		"继续",
		"接着",
		"上次",
		"上文",
		"前面",
		"刚才",
		"后面",
		"补充",
		"展开",
		"续写",
		"再来",
		"再说",
		"还有吗",
		"还有其他",
		"详细点",
		"说下去",
	}) {
		return true
	}
	if len([]rune(text)) <= 12 && containsAnyText(text, []string{
		"为什么",
		"怎么",
		"如何",
		"啥原因",
		"啥情况",
		"详细",
		"还有",
	}) {
		return true
	}
	return false
}

func detectAssistantIntent(question string) string {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return "general"
	}
	hostHit := containsAnyText(text, []string{"机器", "主机", "服务器", "资产", "云主机", "主机配置", "cpu", "内存", "磁盘", "agent", "凭据", "云账号", "终端"})
	k8sHit := containsAnyText(text, []string{"k8s", "kubernetes", "容器管理", "集群", "节点", "pod", "deployment", "namespace", "命名空间", "工作负载", "service", "ingress", "configmap", "secret", "pvc"})
	alertHit := containsAnyText(text, []string{"告警", "报警", "alert", "p0", "p1", "p2", "critical", "warning", "firing"})
	certHit := containsAnyText(text, []string{"证书", "ssl", "tls", "域名", "过期"})
	monitorHit := containsAnyText(text, []string{"监控", "监控中心", "数据源", "prometheus", "victoriametrics", "loki", "elasticsearch", "告警规则", "拨测"})
	systemHit := containsAnyText(text, []string{"系统管理", "用户", "角色", "权限", "菜单", "部门", "岗位", "rbac", "登录", "mfa"})
	taskHit := containsAnyText(text, []string{"任务中心", "作业", "任务", "ansible", "脚本", "文件分发", "模板"})
	auditHit := containsAnyText(text, []string{"审计", "操作日志", "登录日志", "数据日志", "终端审计"})
	aiopsHit := containsAnyText(text, []string{"智能运维", "ai助手", "ai 助手", "智能诊断", "日志分析", "ai配置", "会话记录"})
	moduleHit := containsAnyText(text, []string{"流程", "模块", "菜单", "功能", "怎么用", "如何使用", "使用说明", "平台能力", "页面", "入口"})
	if !isOpsHubPlatformQuestion(text) {
		return "general"
	}
	hitCount := 0
	for _, hit := range []bool{hostHit, k8sHit, alertHit, certHit, monitorHit, systemHit, taskHit, auditHit, aiopsHit} {
		if hit {
			hitCount++
		}
	}
	if hitCount > 1 || moduleHit || containsAnyText(text, []string{"opshub", "平台", "概览", "总览"}) {
		return "platform"
	}
	switch {
	case certHit:
		return "certificates"
	case monitorHit:
		return "monitor"
	case alertHit:
		return "alerts"
	case k8sHit:
		return "kubernetes"
	case hostHit:
		return "hosts"
	case systemHit:
		return "system"
	case taskHit:
		return "tasks"
	case auditHit:
		return "audit"
	case aiopsHit:
		return "aiops"
	case containsAnyText(text, []string{"opshub", "平台", "当前", "现在", "多少", "有哪些", "统计", "概览"}):
		return "platform"
	default:
		return "general"
	}
}

func isOpsHubPlatformQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return false
	}
	if containsAnyText(text, []string{
		"opshub", "平台", "本系统", "当前系统", "这个系统", "这个项目", "这个页面", "当前页面",
		"资产管理", "主机管理", "agent管理", "云账号", "终端审计",
		"容器管理", "集群管理", "命名空间", "工作负载", "网络管理", "配置管理", "存储管理", "访问控制", "应用诊断",
		"监控中心", "数据源管理", "告警规则", "告警事件", "拨测",
		"ssl证书", "证书管理", "dns配置", "部署配置", "任务记录",
		"统一认证", "认证源", "oauth", "oidc", "ldap", "sso", "应用门户",
		"智能运维", "ai助手", "智能诊断", "日志分析", "告警分析", "ai配置",
		"系统管理", "用户管理", "角色管理", "菜单管理", "部门管理", "岗位管理", "操作审计",
		"任务中心", "文件分发",
	}) {
		return true
	}
	dataQueryHit := containsAnyText(text, []string{"有哪些", "多少", "几个", "当前", "现在", "列表", "统计", "资源占用", "使用情况", "状态", "查一下", "看下", "列出"})
	entityHit := containsAnyText(text, []string{
		"主机", "服务器", "机器", "资产", "agent", "k8s", "kubernetes", "集群", "节点", "pod", "deployment", "namespace", "命名空间",
		"告警", "证书", "域名", "数据源", "用户", "角色", "菜单", "任务", "审计", "oauth", "oidc", "ldap", "sso", "应用", "认证源",
	})
	return dataQueryHit && entityHit
}

func selectAssistantModuleKnowledge(question string) []map[string]any {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return nil
	}
	docs := []map[string]any{
		{
			"key":     "asset",
			"title":   "资产管理",
			"aliases": []string{"资产", "主机", "服务器", "agent", "云主机", "终端"},
			"capabilities": []string{
				"维护主机资产、业务分组、凭据、云账号、Agent 管理和 Web 终端审计。",
				"内网主机可通过 Agent 主动采集，云主机默认通过 SSH 采集，避免公网机器反连内网 OpsHub。",
				"支持一键安装 Agent、批量解除 Agent、主机资源与在线状态查看。",
			},
			"flow": []string{
				"先录入云账号或主机凭据，再同步/新增主机资产。",
				"根据主机来源选择采集方式：内网 Agent、云主机 SSH。",
				"采集结果进入主机列表、Agent 管理和终端分组，后续可用于诊断和权限控制。",
			},
			"notes": []string{"涉及安装 Agent、SSH 执行命令和终端登录时，需要校验用户权限和凭据可用性。"},
		},
		{
			"key":     "kubernetes",
			"title":   "容器管理",
			"aliases": []string{"容器", "k8s", "kubernetes", "集群", "命名空间", "deployment", "service", "ingress"},
			"capabilities": []string{
				"管理 Kubernetes 集群、节点、命名空间、工作负载、网络、存储、配置、访问控制、终端审计和应用诊断。",
				"支持 YAML 与表单方式创建/编辑资源，支持对象详情、事件、日志和资源状态查看。",
				"AI 助手可按问题读取集群、节点、命名空间、Pod、工作负载和资源占用等只读数据。",
			},
			"flow": []string{
				"注册集群并保存 kubeconfig 凭证。",
				"同步/读取集群资源，包括节点、命名空间、工作负载、Service、Ingress、PVC、ConfigMap、Secret 等。",
				"在详情、诊断、日志分析或终端入口做只读排查，变更类操作需要人工确认。",
			},
			"notes": []string{"如果 metrics-server 或 Prometheus 不可用，CPU/内存实时占用可能为空，只能返回对象数量和状态。"},
		},
		{
			"key":     "monitor",
			"title":   "监控中心",
			"aliases": []string{"监控", "监控中心", "prometheus", "victoriametrics", "loki", "elasticsearch", "告警规则", "拨测", "数据源"},
			"capabilities": []string{
				"统一管理 Prometheus、VictoriaMetrics、Loki、Elasticsearch 等数据源。",
				"支持数据源查询、仪表盘展示、告警规则、告警事件、通知渠道、拨测任务和调度状态。",
				"告警分析会读取告警事件、规则、数据源和近期上下文生成根因分析。",
			},
			"flow": []string{
				"先在数据源管理中新增并测试 Prometheus、VictoriaMetrics、Loki 或 Elasticsearch。",
				"基于数据源配置查询、仪表盘或告警规则，规则定时调度评估后生成告警事件。",
				"告警事件按 P0/P1/P2 展示，可关联通知渠道发送，并可进入智能运维的告警分析做根因判断。",
			},
			"notes": []string{"监控中心本身不直接生成指标，指标来自外部数据源；数据源地址、认证、查询语句和调度状态会影响展示结果。"},
		},
		{
			"key":     "ssl-cert",
			"title":   "SSL 证书",
			"aliases": []string{"ssl", "tls", "证书", "域名", "dns", "cas", "let's encrypt", "阿里云证书"},
			"capabilities": []string{
				"管理证书、DNS 配置、部署配置和任务记录，支持 Let's Encrypt、云厂商证书服务和手动导入。",
				"支持证书申请、同步、续期、下载和部署到 Nginx 或 Kubernetes Secret 等目标。",
				"AI 助手可读取证书状态、自动续期配置、过期时间和最近错误。",
			},
			"flow": []string{
				"先配置 DNS Provider 或云账号，用于域名验证或云厂商证书申请。",
				"创建证书申请任务，任务完成后证书进入证书管理列表。",
				"配置自动续期和部署目标后，由任务调度触发续期/部署；任务记录保存执行状态和错误信息。",
			},
			"notes": []string{"自动续期和自动部署依赖 DNS/云账号权限、目标部署配置和后台任务调度；任务一直执行中时应优先查看任务日志和云端证书状态。"},
		},
		{
			"key":     "aiops",
			"title":   "智能运维",
			"aliases": []string{"ai", "智能运维", "ai助手", "智能诊断", "日志分析", "告警分析"},
			"capabilities": []string{
				"提供 AI 助手、智能诊断、日志分析、告警分析、AI 会话记录和 AI 配置。",
				"AI 助手通过平台知识和只读数据上下文回答问题，诊断/日志/告警分析会采集对应证据链。",
				"支持 Markdown 输出、流式回复、会话保存和模型配置选择。",
			},
			"flow": []string{
				"先在 AI 配置中配置 OpenAI-compatible 模型和默认模型。",
				"用户提问后，系统识别意图并收集主机、集群、告警、证书或模块流程等只读上下文。",
				"模型基于上下文生成回答，过程与会话会落库，便于追溯。",
			},
			"notes": []string{"AI 不应该编造平台没有返回的数据；实时数据以 OpsHub 接口和当前用户权限可见范围为准。"},
		},
		{
			"key":     "system",
			"title":   "系统管理",
			"aliases": []string{"系统管理", "用户", "角色", "权限", "菜单", "部门", "岗位", "rbac", "账号", "登录"},
			"capabilities": []string{
				"维护用户、角色、部门、岗位、菜单和资产权限，是 OpsHub 后台访问控制的基础。",
				"角色绑定菜单权限和资产权限后，用户通过角色获得页面入口、按钮操作和主机访问范围。",
				"支持用户状态、部门组织、岗位信息和菜单可见性管理，适合演示账号和最小权限账号隔离。",
			},
			"flow": []string{
				"先创建部门/岗位等组织信息，再创建用户并分配角色。",
				"在角色中配置菜单权限和资产权限，控制用户能看到哪些模块、能操作哪些主机。",
				"用户登录后前端按当前用户菜单生成导航，后端接口仍需要权限中间件兜底校验。",
			},
			"notes": []string{"演示环境建议使用独立 test/demo 账号，只给只读菜单和必要资产权限，不直接暴露 admin。"},
		},
		{
			"key":     "task",
			"title":   "任务中心",
			"aliases": []string{"任务", "任务中心", "作业", "脚本", "ansible", "文件分发", "模板"},
			"capabilities": []string{
				"管理脚本作业、Ansible 任务、任务模板和文件分发，用于批量执行与自动化运维。",
				"任务可以选择目标主机、模板参数和执行方式，执行结果会记录状态、输出和错误。",
				"文件分发用于将文件批量推送到指定主机路径，适合配置下发、包分发等场景。",
			},
			"flow": []string{
				"先维护主机资产和可用凭据，确保目标主机可连接。",
				"创建任务模板或临时作业，选择目标主机并填写参数。",
				"执行后查看任务状态、执行输出和失败原因，必要时按主机逐个排查。",
			},
			"notes": []string{"任务中心属于变更执行能力，AI 助手只能生成建议和只读分析，不应直接声称已执行任务。"},
		},
		{
			"key":     "identity",
			"title":   "统一认证",
			"aliases": []string{"统一认证", "sso", "oauth", "ldap", "mfa", "应用门户", "单点登录"},
			"capabilities": []string{
				"提供 OAuth2/OIDC、LDAP、MFA、第三方登录和应用门户能力。",
				"可把外部系统接入 OpsHub 作为统一身份入口，也可将 OpsHub 用户同步到本地账号体系。",
				"认证日志会记录登录来源、应用访问和失败原因，便于安全审计。",
			},
			"flow": []string{
				"配置身份源或 OAuth 应用参数。",
				"为用户分配系统角色和应用访问权限。",
				"用户通过统一登录入口访问 OpsHub 或已接入的第三方应用。",
			},
			"notes": []string{"认证能力和 RBAC 菜单权限互补：认证解决谁登录，RBAC 决定登录后能看和能做什么。"},
		},
		{
			"key":     "nginx",
			"title":   "Nginx 统计",
			"aliases": []string{"nginx", "访问日志", "网站统计", "流量"},
			"capabilities": []string{
				"面向 Nginx 日志和站点访问情况做统计展示。",
				"通常用于查看访问趋势、状态码、URL、来源 IP、流量和异常请求。",
			},
			"flow": []string{
				"配置日志来源或采集路径。",
				"解析访问日志并写入统计数据。",
				"在页面中查看趋势、Top 列表和异常分布。",
			},
			"notes": []string{"统计准确性依赖日志格式、采集路径和解析规则。"},
		},
		{
			"key":     "audit",
			"title":   "操作审计",
			"aliases": []string{"审计", "操作日志", "登录日志", "数据日志", "终端审计"},
			"capabilities": []string{
				"记录用户登录、操作、数据变更和终端会话审计。",
				"用于问题追溯、权限审查和安全合规。",
			},
			"flow": []string{
				"用户访问系统或执行操作时生成审计记录。",
				"审计页面按用户、模块、时间、动作等条件查询。",
				"终端审计关联资产终端会话和命令记录。",
			},
			"notes": []string{"审计只记录已接入审计中间件或对应模块显式写入的行为。"},
		},
	}
	broad := containsAnyText(text, []string{"平台", "opshub", "所有模块", "全部模块", "有哪些功能", "平台能力", "菜单"})
	selected := make([]map[string]any, 0)
	for _, doc := range docs {
		aliases, _ := doc["aliases"].([]string)
		if broad || containsAnyText(text, aliases) || strings.Contains(text, strings.ToLower(fmt.Sprint(doc["title"]))) {
			selected = append(selected, doc)
		}
	}
	if len(selected) == 0 && containsAnyText(text, []string{"流程", "怎么用", "如何使用", "功能"}) {
		selected = docs
	}
	return selected
}

func (s *Service) collectAssistantOverviewContext(ctx context.Context, result *assistantPlatformContext) {
	entityCounts := map[string]any{
		"hosts":           s.assistantCount(ctx, result, "主机数量", &assetmodel.Host{}, ""),
		"cloudAccounts":   s.assistantCount(ctx, result, "云账号数量", &assetmodel.CloudAccount{}, ""),
		"credentials":     s.assistantCount(ctx, result, "主机凭据数量", &assetmodel.Credential{}, ""),
		"k8sClusters":     s.assistantCount(ctx, result, "Kubernetes 集群数量", &k8smodels.Cluster{}, ""),
		"alertEvents":     s.assistantCount(ctx, result, "告警事件数量", &monitormodel.AlertEvent{}, ""),
		"sslCertificates": s.assistantCount(ctx, result, "SSL 证书数量", &sslmodel.SSLCertificate{}, ""),
		"users":           s.assistantCount(ctx, result, "用户数量", &rbacmodel.SysUser{}, ""),
		"roles":           s.assistantCount(ctx, result, "角色数量", &rbacmodel.SysRole{}, ""),
		"menus":           s.assistantCount(ctx, result, "菜单数量", &rbacmodel.SysMenu{}, ""),
		"jobTasks":        s.assistantCount(ctx, result, "任务作业数量", &taskmodel.JobTask{}, ""),
	}

	menuItems := make([]map[string]any, 0)
	var menus []rbacmodel.SysMenu
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "code", "type", "parent_id", "path", "component", "icon", "sort", "visible", "status"}).
		Where("status = ? AND visible = ? AND type IN ?", 1, 1, []int{1, 2}).
		Order("parent_id ASC, sort ASC, id ASC").
		Limit(220).
		Find(&menus).Error; err != nil {
		result.Errors = append(result.Errors, "平台菜单查询失败: "+err.Error())
	} else {
		for _, menu := range menus {
			menuItems = append(menuItems, map[string]any{
				"id":        menu.ID,
				"name":      menu.Name,
				"code":      menu.Code,
				"type":      menu.Type,
				"parentId":  menu.ParentID,
				"path":      menu.Path,
				"component": menu.Component,
				"icon":      menu.Icon,
			})
		}
	}

	result.PlatformSummary = map[string]any{
		"answerPolicy": []string{
			"只要问题和 OpsHub 平台相关，就优先结合平台能力目录、菜单和已采集到的只读数据回答。",
			"涉及实时状态时，使用上下文中的主机、集群、告警、证书、任务、审计等数据；缺少数据时说明具体缺少哪类数据和建议查看入口。",
			"AI 助手只做只读查询、解释和建议，不声称已执行删除、重启、扩缩容、部署等变更。",
		},
		"moduleCatalog": assistantModuleCatalog(),
		"entityCounts":  entityCounts,
		"visibleMenus":  menuItems,
	}
}

func (s *Service) collectAssistantSystemContext(ctx context.Context, result *assistantPlatformContext) {
	userStatus := map[string]int64{
		"enabled":  s.assistantCount(ctx, result, "启用用户数量", &rbacmodel.SysUser{}, "status = ?", 1),
		"disabled": s.assistantCount(ctx, result, "禁用用户数量", &rbacmodel.SysUser{}, "status = ?", 0),
	}
	roleStatus := map[string]int64{
		"enabled":  s.assistantCount(ctx, result, "启用角色数量", &rbacmodel.SysRole{}, "status = ?", 1),
		"disabled": s.assistantCount(ctx, result, "禁用角色数量", &rbacmodel.SysRole{}, "status = ?", 0),
	}
	menuStatus := map[string]int64{
		"visible": s.assistantCount(ctx, result, "可见菜单数量", &rbacmodel.SysMenu{}, "status = ? AND visible = ?", 1, 1),
		"hidden":  s.assistantCount(ctx, result, "隐藏菜单数量", &rbacmodel.SysMenu{}, "visible = ?", 0),
	}

	recentUsers := make([]map[string]any, 0)
	var users []rbacmodel.SysUser
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "username", "real_name", "email", "status", "source", "department_id", "last_login_at", "created_at"}).
		Order("id DESC").
		Limit(50).
		Find(&users).Error; err != nil {
		result.Errors = append(result.Errors, "用户列表摘要查询失败: "+err.Error())
	} else {
		for _, user := range users {
			recentUsers = append(recentUsers, map[string]any{
				"id":           user.ID,
				"username":     user.Username,
				"realName":     user.RealName,
				"email":        user.Email,
				"status":       user.Status,
				"source":       user.Source,
				"departmentId": user.DepartmentID,
				"lastLoginAt":  formatTimePtr(user.LastLoginAt),
			})
		}
	}

	result.SystemSummary = map[string]any{
		"userStatus":       userStatus,
		"roleStatus":       roleStatus,
		"menuStatus":       menuStatus,
		"departments":      s.assistantCount(ctx, result, "部门数量", &rbacmodel.SysDepartment{}, ""),
		"positions":        s.assistantCount(ctx, result, "岗位数量", &rbacmodel.SysPosition{}, ""),
		"assetPermissions": s.assistantCount(ctx, result, "资产权限规则数量", &rbacmodel.SysRoleAssetPermission{}, ""),
		"recentUsers":      recentUsers,
	}
}

func (s *Service) collectAssistantMonitorContext(ctx context.Context, result *assistantPlatformContext) {
	var dataSources []monitormodel.DataSource
	typeCount := map[string]int{}
	statusCount := map[string]int{}
	dataSourceItems := make([]map[string]any, 0)
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "type", "url", "enabled", "remote_write_enabled", "status", "last_test_at", "last_error", "updated_at"}).
		Order("id DESC").
		Limit(80).
		Find(&dataSources).Error; err != nil {
		result.Errors = append(result.Errors, "监控数据源查询失败: "+err.Error())
	} else {
		for _, ds := range dataSources {
			typeCount[ds.Type]++
			statusCount[ds.Status]++
			dataSourceItems = append(dataSourceItems, map[string]any{
				"id":                 ds.ID,
				"name":               ds.Name,
				"type":               ds.Type,
				"url":                ds.URL,
				"enabled":            ds.Enabled,
				"remoteWriteEnabled": ds.RemoteWriteEnabled,
				"status":             ds.Status,
				"lastTestAt":         formatTimePtr(ds.LastTestAt),
				"lastError":          truncateText(ds.LastError, 200),
			})
		}
	}

	ruleStateCount := map[string]int{}
	var rules []monitormodel.AlertRule
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "data_source_type", "severity", "enabled", "last_state", "last_eval_at", "last_error"}).
		Order("id DESC").
		Limit(100).
		Find(&rules).Error; err != nil {
		result.Errors = append(result.Errors, "监控告警规则查询失败: "+err.Error())
	} else {
		for _, rule := range rules {
			state := strings.TrimSpace(rule.LastState)
			if state == "" {
				state = "unknown"
			}
			ruleStateCount[state]++
		}
	}

	probeStatusCount := map[string]int{}
	var probes []monitormodel.ProbeTask
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "protocol", "endpoint", "enabled", "status", "last_status", "last_probe_at", "last_error"}).
		Order("id DESC").
		Limit(100).
		Find(&probes).Error; err != nil {
		result.Errors = append(result.Errors, "拨测任务查询失败: "+err.Error())
	} else {
		for _, probe := range probes {
			status := strings.TrimSpace(probe.Status)
			if status == "" {
				status = "unknown"
			}
			probeStatusCount[status]++
		}
	}

	result.MonitorSummary = map[string]any{
		"dataSourceSample":      len(dataSources),
		"dataSourceTypeCount":   typeCount,
		"dataSourceStatusCount": statusCount,
		"dataSources":           dataSourceItems,
		"alertRules": map[string]any{
			"sample":     len(rules),
			"total":      s.assistantCount(ctx, result, "告警规则数量", &monitormodel.AlertRule{}, ""),
			"enabled":    s.assistantCount(ctx, result, "启用告警规则数量", &monitormodel.AlertRule{}, "enabled = ?", true),
			"stateCount": ruleStateCount,
		},
		"probeTasks": map[string]any{
			"sample":      len(probes),
			"total":       s.assistantCount(ctx, result, "拨测任务数量", &monitormodel.ProbeTask{}, ""),
			"enabled":     s.assistantCount(ctx, result, "启用拨测任务数量", &monitormodel.ProbeTask{}, "enabled = ?", true),
			"statusCount": probeStatusCount,
		},
		"noticeObjects": s.assistantCount(ctx, result, "通知对象数量", &monitormodel.NoticeObject{}, ""),
		"faultCenters":  s.assistantCount(ctx, result, "故障中心数量", &monitormodel.FaultCenter{}, ""),
	}
}

func (s *Service) collectAssistantTaskContext(ctx context.Context, result *assistantPlatformContext) {
	statusCount := map[string]int{}
	var tasks []taskmodel.JobTask
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "task_type", "status", "execute_time", "error_message", "created_at", "updated_at"}).
		Order("id DESC").
		Limit(100).
		Find(&tasks).Error; err != nil {
		result.Errors = append(result.Errors, "任务作业查询失败: "+err.Error())
	} else {
		for _, task := range tasks {
			status := strings.TrimSpace(task.Status)
			if status == "" {
				status = "unknown"
			}
			statusCount[status]++
		}
	}
	ansibleStatusCount := map[string]int{}
	var ansibleTasks []taskmodel.AnsibleTask
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "status", "last_run_time", "updated_at"}).
		Order("id DESC").
		Limit(100).
		Find(&ansibleTasks).Error; err != nil {
		result.Errors = append(result.Errors, "Ansible 任务查询失败: "+err.Error())
	} else {
		for _, task := range ansibleTasks {
			status := strings.TrimSpace(task.Status)
			if status == "" {
				status = "unknown"
			}
			ansibleStatusCount[status]++
		}
	}
	result.TaskSummary = map[string]any{
		"jobTasks": map[string]any{
			"sample":      len(tasks),
			"total":       s.assistantCount(ctx, result, "任务作业数量", &taskmodel.JobTask{}, ""),
			"statusCount": statusCount,
		},
		"templates": s.assistantCount(ctx, result, "任务模板数量", &taskmodel.JobTemplate{}, ""),
		"ansibleTasks": map[string]any{
			"sample":      len(ansibleTasks),
			"total":       s.assistantCount(ctx, result, "Ansible 任务数量", &taskmodel.AnsibleTask{}, ""),
			"statusCount": ansibleStatusCount,
		},
	}
}

func (s *Service) collectAssistantAuditContext(ctx context.Context, result *assistantPlatformContext) {
	moduleCount := map[string]int{}
	statusCount := map[string]int{}
	recentOps := make([]map[string]any, 0)
	var ops []auditmodel.SysOperationLog
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "user_id", "username", "module", "action", "description", "method", "path", "status", "error_msg", "cost_time", "ip", "created_at"}).
		Order("id DESC").
		Limit(80).
		Find(&ops).Error; err != nil {
		result.Errors = append(result.Errors, "操作审计查询失败: "+err.Error())
	} else {
		for _, op := range ops {
			module := strings.TrimSpace(op.Module)
			if module == "" {
				module = "unknown"
			}
			moduleCount[module]++
			statusCount[fmt.Sprintf("%d", op.Status)]++
			if len(recentOps) < 30 {
				recentOps = append(recentOps, map[string]any{
					"id":          op.ID,
					"username":    op.Username,
					"module":      op.Module,
					"action":      op.Action,
					"description": op.Description,
					"method":      op.Method,
					"path":        op.Path,
					"status":      op.Status,
					"errorMsg":    truncateText(op.ErrorMsg, 160),
					"costTime":    op.CostTime,
					"ip":          op.IP,
					"createdAt":   op.CreatedAt.Format("2006-01-02 15:04:05"),
				})
			}
		}
	}
	result.AuditSummary = map[string]any{
		"operationLogs": map[string]any{
			"total":       s.assistantCount(ctx, result, "操作日志数量", &auditmodel.SysOperationLog{}, ""),
			"sample":      len(ops),
			"moduleCount": moduleCount,
			"statusCount": statusCount,
			"recent":      recentOps,
		},
		"loginLogs": s.assistantCount(ctx, result, "登录日志数量", &auditmodel.SysLoginLog{}, ""),
		"dataLogs":  s.assistantCount(ctx, result, "数据日志数量", &auditmodel.SysDataLog{}, ""),
	}
}

func (s *Service) collectAssistantAIContext(ctx context.Context, result *assistantPlatformContext) {
	var providers []aiopsdata.Provider
	providerItems := make([]map[string]any, 0)
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "provider", "base_url", "model", "enabled", "is_default", "max_tokens", "timeout", "last_test_at", "last_test_msg"}).
		Order("is_default DESC, id DESC").
		Limit(30).
		Find(&providers).Error; err != nil {
		result.Errors = append(result.Errors, "AI 模型配置查询失败: "+err.Error())
	} else {
		for _, provider := range providers {
			providerItems = append(providerItems, map[string]any{
				"id":          provider.ID,
				"name":        provider.Name,
				"provider":    provider.Provider,
				"baseURL":     provider.BaseURL,
				"model":       provider.Model,
				"enabled":     provider.Enabled,
				"isDefault":   provider.IsDefault,
				"maxTokens":   provider.MaxTokens,
				"timeout":     provider.Timeout,
				"lastTestAt":  formatTimePtr(provider.LastTestAt),
				"lastTestMsg": truncateText(provider.LastTestMsg, 180),
			})
		}
	}
	result.AISummary = map[string]any{
		"providers": map[string]any{
			"total":   s.assistantCount(ctx, result, "AI 模型配置数量", &aiopsdata.Provider{}, ""),
			"enabled": s.assistantCount(ctx, result, "启用 AI 模型配置数量", &aiopsdata.Provider{}, "enabled = ?", true),
			"items":   providerItems,
		},
		"sessions":       s.assistantCount(ctx, result, "AI 会话数量", &aiopsdata.Session{}, ""),
		"messages":       s.assistantCount(ctx, result, "AI 消息数量", &aiopsdata.Message{}, ""),
		"diagnosisTasks": s.assistantCount(ctx, result, "AI 诊断任务数量", &aiopsdata.DiagnosisTask{}, ""),
	}
}

func assistantModuleCatalog() []map[string]any {
	return []map[string]any{
		{"key": "asset", "title": "资产管理", "entries": []string{"主机管理", "Agent 管理", "云账号", "凭据", "终端/终端审计"}},
		{"key": "kubernetes", "title": "容器管理", "entries": []string{"集群管理", "节点", "命名空间", "工作负载", "网络管理", "配置管理", "存储管理", "访问控制", "应用诊断"}},
		{"key": "monitor", "title": "监控中心", "entries": []string{"数据源", "仪表盘/查询", "告警规则", "告警事件", "通知对象", "拨测", "故障中心"}},
		{"key": "ssl-cert", "title": "SSL 证书", "entries": []string{"证书管理", "DNS 配置", "部署配置", "任务记录", "自动续期/部署"}},
		{"key": "task", "title": "任务中心", "entries": []string{"脚本作业", "Ansible 任务", "任务模板", "文件分发", "执行记录"}},
		{"key": "system", "title": "系统管理", "entries": []string{"用户", "角色", "菜单", "部门", "岗位", "资产权限"}},
		{"key": "identity", "title": "统一认证", "entries": []string{"OAuth2/OIDC", "LDAP", "MFA", "应用门户", "认证日志"}},
		{"key": "audit", "title": "操作审计", "entries": []string{"操作日志", "登录日志", "数据日志", "终端审计"}},
		{"key": "aiops", "title": "智能运维", "entries": []string{"AI 助手", "智能诊断", "日志分析", "告警分析", "会话记录", "AI 配置"}},
	}
}

func (s *Service) assistantCount(ctx context.Context, result *assistantPlatformContext, label string, model any, query string, args ...any) int64 {
	var count int64
	db := s.db.WithContext(ctx).Model(model)
	if strings.TrimSpace(query) != "" {
		db = db.Where(query, args...)
	}
	if err := db.Count(&count).Error; err != nil {
		result.Errors = append(result.Errors, label+"查询失败: "+err.Error())
		return 0
	}
	return count
}

func shouldCollectNamespaceResources(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if !containsAnyText(text, []string{"namespace", "命名空间"}) {
		return false
	}
	return containsAnyText(text, []string{"资源", "占用", "使用", "统计", "有哪些", "列表", "cpu", "内存", "pod", "工作负载", "pvc", "service"})
}

func shouldCollectKubernetesObjectInventory(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if !containsAnyText(text, []string{"有哪些", "列表", "列出", "查看", "看下", "看看", "查一下", "资源", "对象", "清单", "详情", "状态", "正常", "运行", "异常", "健康", "重启", "事件", "告警"}) {
		return false
	}
	return containsAnyText(text, []string{
		"k8s", "kubernetes", "集群", "namespace", "命名空间", "pod", "deployment", "工作负载", "statefulset", "daemonset", "job", "cronjob",
		"service", "svc", "ingress", "configmap", "secret", "pvc", "存储", "网络", "配置",
	})
}

func hasMentionedCluster(question string, clusters []k8smodels.Cluster) bool {
	for i := range clusters {
		if clusterMatchesQuestion(question, &clusters[i]) {
			return true
		}
	}
	return false
}

func clusterMatchesQuestion(question string, cluster *k8smodels.Cluster) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" || cluster == nil {
		return false
	}
	for _, name := range []string{cluster.Name, cluster.Alias, fmt.Sprintf("%d", cluster.ID)} {
		name = strings.ToLower(strings.TrimSpace(name))
		if len([]rune(name)) >= 2 && strings.Contains(text, name) {
			return true
		}
	}
	return false
}

func (s *Service) collectAssistantHostContext(ctx context.Context, result *assistantPlatformContext) {
	var hosts []assetmodel.Host
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "hostname", "ip", "type", "cloud_provider", "status", "agent_status", "cpu_cores", "cpu_usage", "memory_total", "memory_used", "memory_usage", "disk_total", "disk_used", "disk_usage", "os", "arch", "last_seen", "agent_last_seen", "updated_at"}).
		Order("id DESC").
		Limit(500).
		Find(&hosts).Error; err != nil {
		result.Errors = append(result.Errors, "主机资产查询失败: "+err.Error())
		return
	}
	statusCount := map[string]int{"online": 0, "offline": 0, "unknown": 0}
	typeCount := map[string]int{"self": 0, "cloud": 0}
	agentCount := map[string]int{"online": 0, "offline": 0, "pending": 0, "none": 0}
	var totalCPU int
	var totalMemory uint64
	var totalDisk uint64
	hostItems := make([]map[string]any, 0, minInt(len(hosts), 80))
	hotHosts := make([]assetmodel.Host, 0)
	for _, host := range hosts {
		totalCPU += host.CPUCores
		totalMemory += host.MemoryTotal
		totalDisk += host.DiskTotal
		statusCount[hostStatusText(host.Status)]++
		hostType := strings.TrimSpace(host.Type)
		if hostType == "" {
			hostType = "self"
		}
		typeCount[hostType]++
		agentStatus := strings.TrimSpace(host.AgentStatus)
		if agentStatus == "" {
			agentStatus = "none"
		}
		agentCount[agentStatus]++
		if host.CPUUsage >= 85 || host.MemoryUsage >= 85 || host.DiskUsage >= 85 {
			hotHosts = append(hotHosts, host)
		}
		if len(hostItems) < 80 {
			hostItems = append(hostItems, map[string]any{
				"id":            host.ID,
				"name":          host.Name,
				"hostname":      host.Hostname,
				"ip":            host.IP,
				"type":          host.Type,
				"cloudProvider": host.CloudProvider,
				"status":        hostStatusText(host.Status),
				"agentStatus":   agentStatus,
				"cpuCores":      host.CPUCores,
				"cpuUsage":      roundFloat(host.CPUUsage),
				"memoryTotal":   formatBytesCN(host.MemoryTotal),
				"memoryUsage":   roundFloat(host.MemoryUsage),
				"diskTotal":     formatBytesCN(host.DiskTotal),
				"diskUsage":     roundFloat(host.DiskUsage),
				"os":            host.OS,
				"arch":          host.Arch,
				"lastSeen":      formatTimePtr(host.LastSeen),
				"agentLastSeen": formatTimePtr(host.AgentLastSeen),
			})
		}
	}
	sort.Slice(hotHosts, func(i, j int) bool {
		return maxFloat(hotHosts[i].CPUUsage, hotHosts[i].MemoryUsage, hotHosts[i].DiskUsage) > maxFloat(hotHosts[j].CPUUsage, hotHosts[j].MemoryUsage, hotHosts[j].DiskUsage)
	})
	hotItems := make([]map[string]any, 0, minInt(len(hotHosts), 8))
	for i := 0; i < minInt(len(hotHosts), 8); i++ {
		host := hotHosts[i]
		hotItems = append(hotItems, map[string]any{
			"id":          host.ID,
			"name":        host.Name,
			"ip":          host.IP,
			"cpuUsage":    roundFloat(host.CPUUsage),
			"memoryUsage": roundFloat(host.MemoryUsage),
			"diskUsage":   roundFloat(host.DiskUsage),
		})
	}
	result.HostSummary = map[string]any{
		"total":          len(hosts),
		"statusCount":    statusCount,
		"typeCount":      typeCount,
		"agentCount":     agentCount,
		"totalCPU":       totalCPU,
		"totalMemory":    formatBytesCN(totalMemory),
		"totalDisk":      formatBytesCN(totalDisk),
		"highUsageHosts": hotItems,
		"truncated":      len(hosts) >= 500,
	}
	result.Hosts = hostItems
}

func (s *Service) collectAssistantClusterContext(ctx context.Context, result *assistantPlatformContext) {
	var clusters []k8smodels.Cluster
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "alias", "version", "status", "region", "provider", "node_count", "pod_count", "status_synced_at", "updated_at"}).
		Order("id DESC").
		Find(&clusters).Error; err != nil {
		result.Errors = append(result.Errors, "Kubernetes 集群查询失败: "+err.Error())
		return
	}
	statusCount := map[string]int{"normal": 0, "failed": 0, "disabled": 0, "unknown": 0}
	var totalNodes int
	var totalPods int
	items := make([]map[string]any, 0, len(clusters))
	needNamespaceResources := shouldCollectNamespaceResources(result.Question)
	needObjectInventory := shouldCollectKubernetesObjectInventory(result.Question)
	hasClusterMention := hasMentionedCluster(result.Question, clusters)
	for _, cluster := range clusters {
		status := clusterStatusText(cluster.Status)
		statusCount[status]++
		totalNodes += cluster.NodeCount
		totalPods += cluster.PodCount
		item := map[string]any{
			"id":             cluster.ID,
			"name":           cluster.Name,
			"alias":          cluster.Alias,
			"version":        cluster.Version,
			"status":         status,
			"provider":       cluster.Provider,
			"region":         cluster.Region,
			"nodeCount":      cluster.NodeCount,
			"podCount":       cluster.PodCount,
			"statusSyncedAt": formatTimePtr(cluster.StatusSyncedAt),
		}
		s.collectAssistantClusterNodeContext(ctx, &cluster, item, result)
		if needNamespaceResources && (!hasClusterMention || clusterMatchesQuestion(result.Question, &cluster)) {
			s.collectAssistantClusterNamespaceContext(ctx, &cluster, item, result)
		}
		if needObjectInventory && (!hasClusterMention || clusterMatchesQuestion(result.Question, &cluster)) {
			s.collectAssistantClusterObjectInventory(ctx, &cluster, item, result)
		}
		items = append(items, item)
	}
	result.ClusterSummary = map[string]any{
		"total":       len(clusters),
		"statusCount": statusCount,
		"totalNodes":  totalNodes,
		"totalPods":   totalPods,
	}
	result.Clusters = items
}

func (s *Service) collectAssistantClusterNamespaceContext(ctx context.Context, cluster *k8smodels.Cluster, item map[string]any, result *assistantPlatformContext) {
	if cluster == nil || cluster.ID == 0 {
		return
	}
	clientset, err := s.getAssistantClusterClientset(ctx, result.UserID, cluster.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s 命名空间资源查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s 命名空间列表查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return
	}
	stats := make(map[string]map[string]any, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		stats[ns.Name] = map[string]any{
			"name":               ns.Name,
			"status":             string(ns.Status.Phase),
			"podCount":           0,
			"runningPods":        0,
			"pendingPods":        0,
			"failedPods":         0,
			"deploymentCount":    0,
			"statefulSetCount":   0,
			"daemonSetCount":     0,
			"jobCount":           0,
			"cronJobCount":       0,
			"serviceCount":       0,
			"ingressCount":       0,
			"configMapCount":     0,
			"secretCount":        0,
			"pvcCount":           0,
			"networkPolicyCount": 0,
			"cpuUsed":            "0m",
			"memoryUsed":         "0 B",
			"cpuUsedMilli":       int64(0),
			"memoryUsedBytes":    int64(0),
		}
	}
	ensureNS := func(name string) map[string]any {
		if strings.TrimSpace(name) == "" {
			name = "default"
		}
		if stats[name] == nil {
			stats[name] = map[string]any{"name": name}
		}
		return stats[name]
	}
	addCount := func(namespace, key string, delta int) {
		ns := ensureNS(namespace)
		ns[key] = intFromAny(ns[key]) + delta
	}

	if pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, pod := range pods.Items {
			ns := ensureNS(pod.Namespace)
			ns["podCount"] = intFromAny(ns["podCount"]) + 1
			switch pod.Status.Phase {
			case corev1.PodRunning:
				ns["runningPods"] = intFromAny(ns["runningPods"]) + 1
			case corev1.PodPending:
				ns["pendingPods"] = intFromAny(ns["pendingPods"]) + 1
			case corev1.PodFailed:
				ns["failedPods"] = intFromAny(ns["failedPods"]) + 1
			}
		}
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Pod 统计失败: %s", clusterDisplayName(cluster), err.Error()))
	}
	if deployments, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range deployments.Items {
			addCount(obj.Namespace, "deploymentCount", 1)
		}
	}
	if statefulSets, err := clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range statefulSets.Items {
			addCount(obj.Namespace, "statefulSetCount", 1)
		}
	}
	if daemonSets, err := clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range daemonSets.Items {
			addCount(obj.Namespace, "daemonSetCount", 1)
		}
	}
	if jobs, err := clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range jobs.Items {
			addCount(obj.Namespace, "jobCount", 1)
		}
	}
	if cronJobs, err := clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range cronJobs.Items {
			addCount(obj.Namespace, "cronJobCount", 1)
		}
	}
	if services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range services.Items {
			addCount(obj.Namespace, "serviceCount", 1)
		}
	}
	if ingresses, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range ingresses.Items {
			addCount(obj.Namespace, "ingressCount", 1)
		}
	}
	if configMaps, err := clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range configMaps.Items {
			addCount(obj.Namespace, "configMapCount", 1)
		}
	}
	if secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range secrets.Items {
			addCount(obj.Namespace, "secretCount", 1)
		}
	}
	if pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range pvcs.Items {
			addCount(obj.Namespace, "pvcCount", 1)
		}
	}
	if policies, err := clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{}); err == nil {
		for _, obj := range policies.Items {
			addCount(obj.Namespace, "networkPolicyCount", 1)
		}
	}

	metricsAvailable := false
	metricsMessage := ""
	if result.UserID > 0 {
		metricsMessage = "用户级权限模式下未使用管理员 metrics client，CPU/内存实时占用可能为空"
	} else if metricsClient, metricsErr := s.clusterService.GetCachedMetricsClientset(ctx, cluster.ID); metricsErr != nil {
		metricsMessage = metricsErr.Error()
	} else if podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{}); err != nil {
		metricsMessage = err.Error()
	} else {
		metricsAvailable = true
		for _, metric := range podMetrics.Items {
			var cpuMilli int64
			var memoryBytes int64
			for _, container := range metric.Containers {
				cpuMilli += container.Usage.Cpu().MilliValue()
				memoryBytes += container.Usage.Memory().Value()
			}
			ns := ensureNS(metric.Namespace)
			ns["cpuUsedMilli"] = int64FromAny(ns["cpuUsedMilli"]) + cpuMilli
			ns["memoryUsedBytes"] = int64FromAny(ns["memoryUsedBytes"]) + memoryBytes
		}
	}

	namespaceItems := make([]map[string]any, 0, len(stats))
	for _, ns := range stats {
		cpuMilli := int64FromAny(ns["cpuUsedMilli"])
		memoryBytes := int64FromAny(ns["memoryUsedBytes"])
		ns["cpuUsed"] = formatMilliCPU(cpuMilli)
		ns["memoryUsed"] = formatBytesCN(uint64(memoryBytes))
		delete(ns, "cpuUsedMilli")
		delete(ns, "memoryUsedBytes")
		namespaceItems = append(namespaceItems, ns)
	}
	sort.Slice(namespaceItems, func(i, j int) bool {
		leftPods := intFromAny(namespaceItems[i]["podCount"])
		rightPods := intFromAny(namespaceItems[j]["podCount"])
		if leftPods == rightPods {
			return fmt.Sprint(namespaceItems[i]["name"]) < fmt.Sprint(namespaceItems[j]["name"])
		}
		return leftPods > rightPods
	})
	if len(namespaceItems) > 120 {
		namespaceItems = namespaceItems[:120]
		item["namespaceResourcesTruncated"] = true
	}
	item["namespaceResources"] = namespaceItems
	item["namespaceResourceSummary"] = map[string]any{
		"namespaceCount":   len(stats),
		"metricsAvailable": metricsAvailable,
		"metricsMessage":   metricsMessage,
	}
}

func (s *Service) collectAssistantClusterObjectInventory(ctx context.Context, cluster *k8smodels.Cluster, item map[string]any, result *assistantPlatformContext) {
	if cluster == nil || cluster.ID == 0 {
		return
	}
	clientset, err := s.getAssistantClusterClientset(ctx, result.UserID, cluster.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s 对象清单查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return
	}
	inventory := map[string]any{}
	counts := map[string]int{}

	if namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(namespaces.Items), 120))
		for i := 0; i < minInt(len(namespaces.Items), 120); i++ {
			ns := namespaces.Items[i]
			items = append(items, map[string]any{
				"name":   ns.Name,
				"status": string(ns.Status.Phase),
				"age":    humanDurationSince(ns.CreationTimestamp.Time),
			})
		}
		counts["namespaces"] = len(namespaces.Items)
		inventory["namespaces"] = items
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Namespace 清单查询失败: %s", clusterDisplayName(cluster), err.Error()))
	}
	if pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(pods.Items), 120))
		for i := 0; i < minInt(len(pods.Items), 120); i++ {
			pod := pods.Items[i]
			items = append(items, map[string]any{
				"namespace":    pod.Namespace,
				"name":         pod.Name,
				"phase":        string(pod.Status.Phase),
				"nodeName":     pod.Spec.NodeName,
				"restartCount": assistantPodRestartCount(&pod),
				"age":          humanDurationSince(pod.CreationTimestamp.Time),
			})
		}
		counts["pods"] = len(pods.Items)
		inventory["pods"] = items
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Pod 清单查询失败: %s", clusterDisplayName(cluster), err.Error()))
	}
	if deployments, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(deployments.Items), 120))
		for i := 0; i < minInt(len(deployments.Items), 120); i++ {
			obj := deployments.Items[i]
			items = append(items, map[string]any{
				"namespace":         obj.Namespace,
				"name":              obj.Name,
				"replicas":          int32PtrValue(obj.Spec.Replicas),
				"readyReplicas":     obj.Status.ReadyReplicas,
				"availableReplicas": obj.Status.AvailableReplicas,
				"updatedReplicas":   obj.Status.UpdatedReplicas,
				"age":               humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["deployments"] = len(deployments.Items)
		inventory["deployments"] = items
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Deployment 清单查询失败: %s", clusterDisplayName(cluster), err.Error()))
	}
	if statefulSets, err := clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(statefulSets.Items), 80))
		for i := 0; i < minInt(len(statefulSets.Items), 80); i++ {
			obj := statefulSets.Items[i]
			items = append(items, map[string]any{
				"namespace":     obj.Namespace,
				"name":          obj.Name,
				"replicas":      int32PtrValue(obj.Spec.Replicas),
				"readyReplicas": obj.Status.ReadyReplicas,
				"age":           humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["statefulSets"] = len(statefulSets.Items)
		inventory["statefulSets"] = items
	}
	if daemonSets, err := clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(daemonSets.Items), 80))
		for i := 0; i < minInt(len(daemonSets.Items), 80); i++ {
			obj := daemonSets.Items[i]
			items = append(items, map[string]any{
				"namespace":        obj.Namespace,
				"name":             obj.Name,
				"desiredScheduled": obj.Status.DesiredNumberScheduled,
				"ready":            obj.Status.NumberReady,
				"age":              humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["daemonSets"] = len(daemonSets.Items)
		inventory["daemonSets"] = items
	}
	if jobs, err := clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(jobs.Items), 80))
		for i := 0; i < minInt(len(jobs.Items), 80); i++ {
			obj := jobs.Items[i]
			items = append(items, map[string]any{
				"namespace": obj.Namespace,
				"name":      obj.Name,
				"succeeded": obj.Status.Succeeded,
				"failed":    obj.Status.Failed,
				"active":    obj.Status.Active,
				"age":       humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["jobs"] = len(jobs.Items)
		inventory["jobs"] = items
	}
	if cronJobs, err := clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(cronJobs.Items), 80))
		for i := 0; i < minInt(len(cronJobs.Items), 80); i++ {
			obj := cronJobs.Items[i]
			items = append(items, map[string]any{
				"namespace": obj.Namespace,
				"name":      obj.Name,
				"schedule":  obj.Spec.Schedule,
				"suspend":   boolPtrValue(obj.Spec.Suspend),
				"age":       humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["cronJobs"] = len(cronJobs.Items)
		inventory["cronJobs"] = items
	}
	if services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(services.Items), 120))
		for i := 0; i < minInt(len(services.Items), 120); i++ {
			obj := services.Items[i]
			items = append(items, map[string]any{
				"namespace":  obj.Namespace,
				"name":       obj.Name,
				"type":       string(obj.Spec.Type),
				"clusterIP":  obj.Spec.ClusterIP,
				"externalIP": strings.Join(obj.Spec.ExternalIPs, ","),
				"ports":      servicePortSummary(obj.Spec.Ports),
				"age":        humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["services"] = len(services.Items)
		inventory["services"] = items
	}
	if ingresses, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(ingresses.Items), 120))
		for i := 0; i < minInt(len(ingresses.Items), 120); i++ {
			obj := ingresses.Items[i]
			items = append(items, map[string]any{
				"namespace":    obj.Namespace,
				"name":         obj.Name,
				"className":    stringPtrValue(obj.Spec.IngressClassName),
				"hosts":        ingressHosts(obj),
				"loadBalancer": ingressLoadBalancer(obj),
				"age":          humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["ingresses"] = len(ingresses.Items)
		inventory["ingresses"] = items
	}
	if configMaps, err := clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(configMaps.Items), 80))
		for i := 0; i < minInt(len(configMaps.Items), 80); i++ {
			obj := configMaps.Items[i]
			items = append(items, map[string]any{
				"namespace":  obj.Namespace,
				"name":       obj.Name,
				"dataKeys":   len(obj.Data),
				"binaryKeys": len(obj.BinaryData),
				"age":        humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["configMaps"] = len(configMaps.Items)
		inventory["configMaps"] = items
	}
	if secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(secrets.Items), 80))
		for i := 0; i < minInt(len(secrets.Items), 80); i++ {
			obj := secrets.Items[i]
			items = append(items, map[string]any{
				"namespace": obj.Namespace,
				"name":      obj.Name,
				"type":      string(obj.Type),
				"keyCount":  len(obj.Data),
				"age":       humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["secrets"] = len(secrets.Items)
		inventory["secrets"] = items
	}
	if pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{}); err == nil {
		items := make([]map[string]any, 0, minInt(len(pvcs.Items), 80))
		for i := 0; i < minInt(len(pvcs.Items), 80); i++ {
			obj := pvcs.Items[i]
			storage := ""
			if q, ok := obj.Status.Capacity[corev1.ResourceStorage]; ok {
				storage = q.String()
			}
			items = append(items, map[string]any{
				"namespace":    obj.Namespace,
				"name":         obj.Name,
				"status":       string(obj.Status.Phase),
				"storage":      storage,
				"storageClass": stringPtrValue(obj.Spec.StorageClassName),
				"age":          humanDurationSince(obj.CreationTimestamp.Time),
			})
		}
		counts["pvcs"] = len(pvcs.Items)
		inventory["pvcs"] = items
	}

	item["objectInventorySummary"] = map[string]any{
		"counts": counts,
		"limits": map[string]any{
			"namespacesPodsDeploymentsServicesIngresses": 120,
			"otherResourceSamples":                       80,
		},
	}
	item["objectInventory"] = inventory
}

func (s *Service) collectAssistantClusterNodeContext(ctx context.Context, cluster *k8smodels.Cluster, item map[string]any, result *assistantPlatformContext) {
	if cluster == nil || cluster.ID == 0 {
		return
	}
	clientset, err := s.getAssistantClusterClientset(ctx, result.UserID, cluster.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s 节点查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s 节点列表查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return
	}
	pods, podErr := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if podErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Pod 分布查询失败: %s", clusterDisplayName(cluster), podErr.Error()))
	}
	podCountByNode := map[string]int{}
	podPhaseCount := map[string]int{}
	abnormalPods := make([]map[string]any, 0)
	highRestartPods := make([]map[string]any, 0)
	totalPodCountForHealth := 0
	abnormalPodCount := 0
	highRestartPodCount := 0
	if pods != nil {
		totalPodCountForHealth = len(pods.Items)
		for _, pod := range pods.Items {
			phase := string(pod.Status.Phase)
			if phase == "" {
				phase = "Unknown"
			}
			podPhaseCount[phase]++
			if pod.Spec.NodeName != "" {
				podCountByNode[pod.Spec.NodeName]++
			}
			reason := assistantPodAbnormalReason(&pod)
			totalRestarts := assistantPodRestartCount(&pod)
			if reason != "" {
				abnormalPodCount++
				if len(abnormalPods) < 50 {
					abnormalPods = append(abnormalPods, assistantPodContextItem(&pod, reason, totalRestarts))
				}
			}
			if totalRestarts >= 3 {
				highRestartPodCount++
				if len(highRestartPods) < 30 {
					highRestartPods = append(highRestartPods, assistantPodContextItem(&pod, "restartCount>=3", totalRestarts))
				}
			}
		}
	}

	metricsAvailable := false
	metricsMessage := ""
	nodeMetrics := map[string]*metricsv1beta1.NodeMetrics{}
	if result.UserID > 0 {
		metricsMessage = "用户级权限模式下未使用管理员 metrics client，CPU/内存实时占用可能为空"
	} else if metricsClient, metricsErr := s.clusterService.GetCachedMetricsClientset(ctx, cluster.ID); metricsErr != nil {
		metricsMessage = metricsErr.Error()
	} else {
		metricList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err != nil {
			metricsMessage = err.Error()
		} else {
			metricsAvailable = true
			for i := range metricList.Items {
				nodeMetrics[metricList.Items[i].Name] = &metricList.Items[i]
			}
		}
	}

	nodesInfo := make([]map[string]any, 0, len(nodes.Items))
	var readyNodes int
	var unschedulableNodes int
	var totalCPUMilli int64
	var usedCPUMilli int64
	var totalMemoryBytes int64
	var usedMemoryBytes int64
	var totalPodCapacity int64
	var totalPodCount int64
	for _, node := range nodes.Items {
		cpuAllocatable := node.Status.Allocatable.Cpu().MilliValue()
		if cpuAllocatable == 0 {
			cpuAllocatable = node.Status.Capacity.Cpu().MilliValue()
		}
		memoryAllocatable := node.Status.Allocatable.Memory().Value()
		if memoryAllocatable == 0 {
			memoryAllocatable = node.Status.Capacity.Memory().Value()
		}
		podCapacity := node.Status.Allocatable.Pods().Value()
		if podCapacity == 0 {
			podCapacity = node.Status.Capacity.Pods().Value()
		}
		podCount := podCountByNode[node.Name]
		cpuUsed := int64(0)
		memoryUsed := int64(0)
		if metric := nodeMetrics[node.Name]; metric != nil {
			cpuUsed = metric.Usage.Cpu().MilliValue()
			memoryUsed = metric.Usage.Memory().Value()
		}
		ready := isK8sNodeReady(node)
		if ready {
			readyNodes++
		}
		if node.Spec.Unschedulable {
			unschedulableNodes++
		}
		totalCPUMilli += cpuAllocatable
		usedCPUMilli += cpuUsed
		totalMemoryBytes += memoryAllocatable
		usedMemoryBytes += memoryUsed
		totalPodCapacity += podCapacity
		totalPodCount += int64(podCount)
		nodesInfo = append(nodesInfo, map[string]any{
			"name":              node.Name,
			"status":            mapBoolText(ready, "Ready", "NotReady"),
			"roles":             k8sNodeRoles(node),
			"internalIP":        k8sNodeInternalIP(node),
			"version":           node.Status.NodeInfo.KubeletVersion,
			"osImage":           node.Status.NodeInfo.OSImage,
			"containerRuntime":  node.Status.NodeInfo.ContainerRuntimeVersion,
			"cpuAllocatable":    formatMilliCPU(cpuAllocatable),
			"cpuUsed":           formatMilliCPU(cpuUsed),
			"cpuUsagePercent":   percentOf(float64(cpuUsed), float64(cpuAllocatable)),
			"memoryAllocatable": formatBytesCN(uint64(memoryAllocatable)),
			"memoryUsed":        formatBytesCN(uint64(memoryUsed)),
			"memoryPercent":     percentOf(float64(memoryUsed), float64(memoryAllocatable)),
			"podCount":          podCount,
			"podCapacity":       podCapacity,
			"podPercent":        percentOf(float64(podCount), float64(podCapacity)),
			"schedulable":       !node.Spec.Unschedulable,
			"taintCount":        len(node.Spec.Taints),
		})
	}

	item["nodes"] = nodesInfo
	item["nodeSummary"] = map[string]any{
		"actualNodeCount":     len(nodes.Items),
		"readyNodes":          readyNodes,
		"notReadyNodes":       len(nodes.Items) - readyNodes,
		"unschedulableNodes":  unschedulableNodes,
		"metricsAvailable":    metricsAvailable,
		"metricsMessage":      metricsMessage,
		"totalCPU":            formatMilliCPU(totalCPUMilli),
		"usedCPU":             formatMilliCPU(usedCPUMilli),
		"cpuUsagePercent":     percentOf(float64(usedCPUMilli), float64(totalCPUMilli)),
		"totalMemory":         formatBytesCN(uint64(totalMemoryBytes)),
		"usedMemory":          formatBytesCN(uint64(usedMemoryBytes)),
		"memoryUsagePercent":  percentOf(float64(usedMemoryBytes), float64(totalMemoryBytes)),
		"totalPodCount":       totalPodCount,
		"totalPodCapacity":    totalPodCapacity,
		"podUsagePercent":     percentOf(float64(totalPodCount), float64(totalPodCapacity)),
		"nodeMetricsReturned": len(nodeMetrics),
	}
	item["podHealthSummary"] = map[string]any{
		"totalPods":             totalPodCountForHealth,
		"phaseCount":            podPhaseCount,
		"abnormalPodCount":      abnormalPodCount,
		"highRestartPodCount":   highRestartPodCount,
		"abnormalPodsTruncated": abnormalPodCount > len(abnormalPods),
	}
	item["abnormalPods"] = abnormalPods
	item["highRestartPods"] = highRestartPods
	item["recentWarningEvents"] = s.collectAssistantClusterWarningEvents(ctx, cluster, result)
}

func (s *Service) collectAssistantAlertContext(ctx context.Context, result *assistantPlatformContext) {
	var events []monitormodel.AlertEvent
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "rule_id", "rule_name", "data_source_name", "data_source_type", "severity", "state", "value", "message", "labels", "last_eval_at", "started_at"}).
		Order("last_eval_at DESC, id DESC").
		Limit(80).
		Find(&events).Error; err != nil {
		result.Errors = append(result.Errors, "监控告警查询失败: "+err.Error())
		return
	}
	var total int64
	_ = s.db.WithContext(ctx).Model(&monitormodel.AlertEvent{}).Count(&total).Error
	severityCount := map[string]int{"P0": 0, "P1": 0, "P2": 0, "其他": 0}
	stateCount := map[string]int{"firing": 0, "pending": 0, "recovered": 0, "其他": 0}
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		severityCount[alertSeverityLabel(event.Severity)]++
		state := strings.TrimSpace(event.State)
		if _, ok := stateCount[state]; !ok {
			state = "其他"
		}
		stateCount[state]++
		items = append(items, map[string]any{
			"id":             event.ID,
			"ruleName":       event.RuleName,
			"dataSourceName": event.DataSourceName,
			"dataSourceType": event.DataSourceType,
			"severity":       alertSeverityLabel(event.Severity),
			"rawSeverity":    event.Severity,
			"state":          event.State,
			"value":          roundFloat(event.Value),
			"message":        truncateText(event.Message, 240),
			"labels":         truncateText(event.Labels, 240),
			"lastEvalAt":     event.LastEvalAt.Format("2006-01-02 15:04:05"),
		})
	}
	result.AlertSummary = map[string]any{
		"total":         total,
		"recentSample":  len(events),
		"severityCount": severityCount,
		"stateCount":    stateCount,
	}
	result.RecentAlerts = items
}

func (s *Service) collectAssistantCertContext(ctx context.Context, result *assistantPlatformContext) {
	var certs []sslmodel.SSLCertificate
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "domain", "source_type", "status", "auto_renew", "not_after", "last_renew_at", "last_error"}).
		Order("not_after ASC, id DESC").
		Limit(100).
		Find(&certs).Error; err != nil {
		result.Errors = append(result.Errors, "SSL 证书查询失败: "+err.Error())
		return
	}
	now := time.Now()
	statusCount := map[string]int{}
	expiringItems := make([]map[string]any, 0)
	for _, cert := range certs {
		statusCount[cert.Status]++
		if cert.NotAfter != nil && cert.NotAfter.Before(now.AddDate(0, 0, 30)) {
			expiringItems = append(expiringItems, map[string]any{
				"id":          cert.ID,
				"name":        cert.Name,
				"domain":      cert.Domain,
				"status":      cert.Status,
				"sourceType":  cert.SourceType,
				"autoRenew":   cert.AutoRenew,
				"notAfter":    cert.NotAfter.Format("2006-01-02 15:04:05"),
				"daysLeft":    int(cert.NotAfter.Sub(now).Hours() / 24),
				"lastRenewAt": formatTimePtr(cert.LastRenewAt),
				"lastError":   truncateText(cert.LastError, 240),
			})
		}
	}
	deployStatus := map[string]int64{
		"total":       s.assistantCount(ctx, result, "SSL 部署配置数量", &sslmodel.DeployConfig{}, ""),
		"enabled":     s.assistantCount(ctx, result, "启用 SSL 部署配置数量", &sslmodel.DeployConfig{}, "enabled = ?", true),
		"autoDeploy":  s.assistantCount(ctx, result, "自动部署 SSL 配置数量", &sslmodel.DeployConfig{}, "auto_deploy = ?", true),
		"lastSuccess": s.assistantCount(ctx, result, "最近成功 SSL 部署配置数量", &sslmodel.DeployConfig{}, "last_deploy_ok = ?", true),
	}
	dnsStatus := map[string]int64{
		"total":   s.assistantCount(ctx, result, "DNS 配置数量", &sslmodel.DNSProvider{}, ""),
		"enabled": s.assistantCount(ctx, result, "启用 DNS 配置数量", &sslmodel.DNSProvider{}, "enabled = ?", true),
	}
	taskStatus := map[string]int{}
	recentTasks := make([]map[string]any, 0)
	var tasks []sslmodel.RenewTask
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "certificate_id", "task_type", "status", "trigger_type", "started_at", "finished_at", "error_message", "updated_at"}).
		Order("id DESC").
		Limit(50).
		Find(&tasks).Error; err != nil {
		result.Errors = append(result.Errors, "SSL 任务记录查询失败: "+err.Error())
	} else {
		for _, task := range tasks {
			status := strings.TrimSpace(task.Status)
			if status == "" {
				status = "unknown"
			}
			taskStatus[status]++
			recentTasks = append(recentTasks, map[string]any{
				"id":            task.ID,
				"certificateId": task.CertificateID,
				"taskType":      task.TaskType,
				"status":        task.Status,
				"triggerType":   task.TriggerType,
				"startedAt":     formatTimePtr(task.StartedAt),
				"finishedAt":    formatTimePtr(task.FinishedAt),
				"errorMessage":  truncateText(task.ErrorMessage, 240),
			})
		}
	}
	result.CertSummary = map[string]any{
		"sample":           len(certs),
		"statusCount":      statusCount,
		"expiringIn30Days": expiringItems,
		"dnsProviders":     dnsStatus,
		"deployConfigs":    deployStatus,
		"recentTaskStatus": taskStatus,
		"recentTasks":      recentTasks,
	}
}

func buildAssistantPrompt(question string, context assistantPlatformContext) string {
	contextJSON, _ := json.MarshalIndent(context, "", "  ")
	question = strings.TrimSpace(question)
	resolvedQuestion := strings.TrimSpace(context.Question)
	if context.Intent == "general" {
		return fmt.Sprintf(`用户问题：
%s

说明：该问题未命中 OpsHub 平台数据查询意图。请按通用智能运维助手方式回答，如涉及平台实时数据，请提示用户补充更具体的问题。`, question)
	}
	if resolvedQuestion != "" && resolvedQuestion != question {
		return fmt.Sprintf(`用户本次问题：
%s

关联的上一轮问题：
%s

OpsHub 平台只读上下文 JSON：
%s

请基于以上 OpsHub 当前平台数据回答。
要求：
1. 只要问题和 OpsHub 平台相关，优先结合 platformSummary、moduleKnowledge、visibleMenus 和各模块 Summary 回答，不要把平台问题当成普通闲聊。
2. 实时数据只能使用上下文中出现的数据，不要编造不存在的主机、集群、节点、Pod、命名空间、告警、证书、任务或用户。
3. 如果数据为空或查询失败，要明确说明缺少的是哪类数据、可能原因和平台内可查看入口；不要只说“上下文没有返回”。
4. 你只能做只读查询、分析和建议，不要声称已经执行删除、重启、扩缩容、变更配置等操作。
5. 输出中文 Markdown，优先给结论和关键统计，再给必要明细。
6. 如果用户询问 Kubernetes 集群资源占用、节点或每个节点占用，必须优先使用 clusters[].nodeSummary 和 clusters[].nodes，按集群列出节点名称、状态、CPU、内存、Pod 使用情况。
7. 如果用户询问 Pod 健康、异常、重启或事件，必须优先使用 clusters[].podHealthSummary、clusters[].abnormalPods、clusters[].highRestartPods 和 clusters[].recentWarningEvents。
8. 如果用户询问命名空间资源占用，必须优先使用 clusters[].namespaceResourceSummary 和 clusters[].namespaceResources，用 Markdown 表格列出命名空间、Pod、工作负载、Service、PVC、CPU、内存。
9. 如果用户询问 Kubernetes 对象列表、资源清单或某类对象有哪些，必须优先使用 clusters[].objectInventorySummary 和 clusters[].objectInventory。
10. 如果用户询问模块流程、页面入口或平台功能，优先使用 moduleKnowledge、platformSummary.moduleCatalog 和 platformSummary.visibleMenus。`, question, resolvedQuestion, truncateText(string(contextJSON), 60000))
	}
	return fmt.Sprintf(`用户问题：
%s

OpsHub 平台只读上下文 JSON：
%s

请基于以上 OpsHub 当前平台数据回答。
要求：
1. 只要问题和 OpsHub 平台相关，优先结合 platformSummary、moduleKnowledge、visibleMenus 和各模块 Summary 回答，不要把平台问题当成普通闲聊。
2. 实时数据只能使用上下文中出现的数据，不要编造不存在的主机、集群、节点、Pod、命名空间、告警、证书、任务或用户。
3. 如果数据为空或查询失败，要明确说明缺少的是哪类数据、可能原因和平台内可查看入口；不要只说“上下文没有返回”。
4. 你只能做只读查询、分析和建议，不要声称已经执行删除、重启、扩缩容、变更配置等操作。
5. 输出中文 Markdown，优先给结论和关键统计，再给必要明细。
6. 如果用户询问 Kubernetes 集群资源占用、节点或每个节点占用，必须优先使用 clusters[].nodeSummary 和 clusters[].nodes，按集群列出节点名称、状态、CPU、内存、Pod 使用情况。
7. 如果用户询问 Pod 健康、异常、重启或事件，必须优先使用 clusters[].podHealthSummary、clusters[].abnormalPods、clusters[].highRestartPods 和 clusters[].recentWarningEvents。
8. 如果用户询问命名空间资源占用，必须优先使用 clusters[].namespaceResourceSummary 和 clusters[].namespaceResources，用 Markdown 表格列出命名空间、Pod、工作负载、Service、PVC、CPU、内存。
9. 如果用户询问 Kubernetes 对象列表、资源清单或某类对象有哪些，必须优先使用 clusters[].objectInventorySummary 和 clusters[].objectInventory。
10. 如果用户询问模块流程、页面入口或平台功能，优先使用 moduleKnowledge、platformSummary.moduleCatalog 和 platformSummary.visibleMenus。`, question, truncateText(string(contextJSON), 60000))
}

func buildAssistantThinkingSteps(context assistantPlatformContext) []string {
	steps := []string{"识别用户问题类型：" + assistantIntentText(context.Intent)}
	if context.Intent == "general" {
		return append(steps, "未触发平台数据查询，按通用运维问答生成建议")
	}
	if context.PlatformSummary != nil {
		steps = append(steps, "读取 OpsHub 平台能力目录、菜单入口和关键对象数量")
	}
	if len(context.ModuleKnowledge) > 0 {
		steps = append(steps, "检索 OpsHub 模块流程与平台能力说明")
	}
	if context.SystemSummary != nil {
		steps = append(steps, "读取用户、角色、菜单、部门和资产权限摘要")
	}
	if context.HostSummary != nil {
		steps = append(steps, "读取主机资产摘要与资源配置")
	}
	if context.ClusterSummary != nil {
		steps = append(steps, "读取 Kubernetes 集群状态、节点明细、Pod 健康和 CPU/内存/Pod 使用数据")
		for _, cluster := range context.Clusters {
			if _, ok := cluster["namespaceResources"]; ok {
				steps = append(steps, "读取匹配集群的命名空间资源统计")
				break
			}
		}
		for _, cluster := range context.Clusters {
			if _, ok := cluster["objectInventory"]; ok {
				steps = append(steps, "读取匹配集群的 Kubernetes 对象清单")
				break
			}
		}
	}
	if context.MonitorSummary != nil {
		steps = append(steps, "读取监控数据源、告警规则、拨测和通知配置摘要")
	}
	if context.AlertSummary != nil {
		steps = append(steps, "读取监控告警等级、状态和最近告警事件")
	}
	if context.CertSummary != nil {
		steps = append(steps, "读取 SSL 证书、DNS、部署配置和最近任务状态")
	}
	if context.TaskSummary != nil {
		steps = append(steps, "读取任务中心作业、模板和 Ansible 任务摘要")
	}
	if context.AuditSummary != nil {
		steps = append(steps, "读取操作审计、登录日志和数据日志摘要")
	}
	if context.AISummary != nil {
		steps = append(steps, "读取 AI 模型配置、会话和诊断任务摘要")
	}
	if len(context.Errors) > 0 {
		steps = append(steps, "记录部分数据查询失败项并在回答中保留边界")
	}
	steps = append(steps, "汇总只读数据并生成回答，不执行任何变更操作")
	return steps
}

func assistantIntentText(intent string) string {
	switch intent {
	case "platform":
		return "OpsHub 平台概览"
	case "hosts":
		return "主机资产查询"
	case "kubernetes":
		return "Kubernetes 集群查询"
	case "monitor":
		return "监控中心查询"
	case "alerts":
		return "监控告警查询"
	case "certificates":
		return "SSL 证书查询"
	case "system":
		return "系统权限查询"
	case "tasks":
		return "任务中心查询"
	case "audit":
		return "操作审计查询"
	case "aiops":
		return "智能运维查询"
	default:
		return "通用运维问答"
	}
}

func stripModelThinking(content string) string {
	text := strings.TrimSpace(content)
	if text == "" {
		return text
	}
	re := regexp.MustCompile(`(?is)<think>.*?</think>`)
	text = strings.TrimSpace(re.ReplaceAllString(text, ""))
	re = regexp.MustCompile(`(?is)<thinking>.*?</thinking>`)
	text = strings.TrimSpace(re.ReplaceAllString(text, ""))
	re = regexp.MustCompile(`(?is)<think>.*$`)
	text = strings.TrimSpace(re.ReplaceAllString(text, ""))
	re = regexp.MustCompile(`(?is)<thinking>.*$`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

type thinkingDeltaFilter struct {
	buffer  string
	inThink bool
}

func newThinkingDeltaFilter() *thinkingDeltaFilter {
	return &thinkingDeltaFilter{}
}

func (f *thinkingDeltaFilter) Push(delta string) string {
	if delta == "" {
		return ""
	}
	f.buffer += delta
	var output strings.Builder
	for {
		if f.buffer == "" {
			return output.String()
		}
		lower := strings.ToLower(f.buffer)
		if f.inThink {
			index, tag := firstTagIndex(lower, []string{"</think>", "</thinking>"})
			if index >= 0 {
				f.buffer = f.buffer[index+len(tag):]
				f.inThink = false
				continue
			}
			keep := longestTagPrefixSuffix(f.buffer, []string{"</think>", "</thinking>"})
			if keep > 0 {
				f.buffer = f.buffer[len(f.buffer)-keep:]
			} else {
				f.buffer = ""
			}
			return output.String()
		}

		index, tag := firstTagIndex(lower, []string{"<think>", "<thinking>"})
		if index >= 0 {
			output.WriteString(f.buffer[:index])
			f.buffer = f.buffer[index+len(tag):]
			f.inThink = true
			continue
		}
		keep := longestTagPrefixSuffix(f.buffer, []string{"<think>", "<thinking>"})
		flushLen := len(f.buffer) - keep
		if flushLen > 0 {
			output.WriteString(f.buffer[:flushLen])
			f.buffer = f.buffer[flushLen:]
		}
		return output.String()
	}
}

func (f *thinkingDeltaFilter) Flush() string {
	if f.inThink {
		f.buffer = ""
		return ""
	}
	rest := f.buffer
	f.buffer = ""
	return rest
}

func firstTagIndex(lower string, tags []string) (int, string) {
	bestIndex := -1
	bestTag := ""
	for _, tag := range tags {
		index := strings.Index(lower, tag)
		if index >= 0 && (bestIndex < 0 || index < bestIndex) {
			bestIndex = index
			bestTag = tag
		}
	}
	return bestIndex, bestTag
}

func longestTagPrefixSuffix(value string, tags []string) int {
	lower := strings.ToLower(value)
	best := 0
	for _, tag := range tags {
		maxLen := minInt(len(lower), len(tag)-1)
		for size := maxLen; size > best; size-- {
			if strings.HasSuffix(lower, tag[:size]) {
				best = size
				break
			}
		}
	}
	return best
}

func containsAnyText(text string, values []string) bool {
	for _, value := range values {
		if strings.Contains(text, strings.ToLower(value)) {
			return true
		}
	}
	return false
}

func hostStatusText(status int) string {
	switch status {
	case 1:
		return "online"
	case 0:
		return "offline"
	default:
		return "unknown"
	}
}

func clusterStatusText(status int) string {
	switch status {
	case k8smodels.ClusterStatusNormal:
		return "normal"
	case k8smodels.ClusterStatusFailed:
		return "failed"
	case k8smodels.ClusterStatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

func alertSeverityLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "p0", "critical":
		return "P0"
	case "p1", "warning", "high":
		return "P1"
	case "p2", "info", "medium", "low":
		return "P2"
	default:
		return "其他"
	}
}

func alertSeverityQueryValues(value string) []string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "p0", "critical":
		return []string{"p0", "P0", "critical"}
	case "p1", "warning", "high":
		return []string{"p1", "P1", "warning", "high"}
	case "p2", "info", "medium", "low":
		return []string{"p2", "P2", "info", "medium", "low"}
	default:
		return []string{strings.TrimSpace(value)}
	}
}

func clusterDisplayName(cluster *k8smodels.Cluster) string {
	if cluster == nil {
		return "未知集群"
	}
	if strings.TrimSpace(cluster.Alias) != "" {
		return cluster.Alias
	}
	if strings.TrimSpace(cluster.Name) != "" {
		return cluster.Name
	}
	return fmt.Sprintf("#%d", cluster.ID)
}

func (s *Service) getAssistantClusterClientset(ctx context.Context, userID, clusterID uint) (*kubernetes.Clientset, error) {
	if userID > 0 {
		return s.clusterService.GetClientsetForUser(ctx, clusterID, userID)
	}
	return s.clusterService.GetCachedClientset(ctx, clusterID)
}

func isK8sNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func k8sNodeRoles(node corev1.Node) string {
	roles := make([]string, 0)
	for key := range node.Labels {
		const prefix = "node-role.kubernetes.io/"
		if strings.HasPrefix(key, prefix) {
			role := strings.TrimPrefix(key, prefix)
			if role == "" {
				role = "worker"
			}
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	if len(roles) == 0 {
		return "worker"
	}
	return strings.Join(roles, ",")
}

func k8sNodeInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	if len(node.Status.Addresses) > 0 {
		return node.Status.Addresses[0].Address
	}
	return ""
}

func int32PtrValue(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func servicePortSummary(ports []corev1.ServicePort) []string {
	items := make([]string, 0, len(ports))
	for _, port := range ports {
		protocol := string(port.Protocol)
		if protocol == "" {
			protocol = "TCP"
		}
		value := fmt.Sprintf("%s:%d/%s", port.Name, port.Port, protocol)
		if strings.TrimPrefix(value, ":") != value {
			value = fmt.Sprintf("%d/%s", port.Port, protocol)
		}
		if port.NodePort > 0 {
			value = fmt.Sprintf("%s->%d", value, port.NodePort)
		}
		items = append(items, value)
	}
	return items
}

func ingressHosts(ingress networkingv1.Ingress) []string {
	hosts := make([]string, 0, len(ingress.Spec.Rules))
	for _, rule := range ingress.Spec.Rules {
		if strings.TrimSpace(rule.Host) != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	sort.Strings(hosts)
	return hosts
}

func ingressLoadBalancer(ingress networkingv1.Ingress) []string {
	items := make([]string, 0)
	for _, item := range ingress.Status.LoadBalancer.Ingress {
		if strings.TrimSpace(item.IP) != "" {
			items = append(items, item.IP)
			continue
		}
		if strings.TrimSpace(item.Hostname) != "" {
			items = append(items, item.Hostname)
		}
	}
	sort.Strings(items)
	return items
}

func assistantPodRestartCount(pod *corev1.Pod) int {
	if pod == nil {
		return 0
	}
	total := 0
	for _, status := range pod.Status.InitContainerStatuses {
		total += int(status.RestartCount)
	}
	for _, status := range pod.Status.ContainerStatuses {
		total += int(status.RestartCount)
	}
	return total
}

func assistantPodAbnormalReason(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if pod.DeletionTimestamp != nil {
		return ""
	}
	switch pod.Status.Phase {
	case corev1.PodFailed:
		return nonEmptyReason("Failed", pod.Status.Reason, pod.Status.Message)
	case corev1.PodUnknown:
		return nonEmptyReason("Unknown", pod.Status.Reason, pod.Status.Message)
	case corev1.PodPending:
		if reason := firstContainerWaitingReason(pod); reason != "" {
			return reason
		}
		return nonEmptyReason("Pending", pod.Status.Reason, pod.Status.Message)
	}
	if reason := firstContainerWaitingReason(pod); reason != "" {
		return reason
	}
	for _, status := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			return nonEmptyReason("Terminated", status.State.Terminated.Reason, status.State.Terminated.Message)
		}
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse && (condition.Type == corev1.PodReady || condition.Type == corev1.ContainersReady || condition.Type == corev1.PodScheduled) {
			return nonEmptyReason(string(condition.Type), condition.Reason, condition.Message)
		}
	}
	return ""
}

func firstContainerWaitingReason(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	for _, status := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if status.State.Waiting != nil {
			return nonEmptyReason("Waiting", status.State.Waiting.Reason, status.State.Waiting.Message)
		}
	}
	return ""
}

func nonEmptyReason(prefix, reason, message string) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(prefix) != "" {
		parts = append(parts, strings.TrimSpace(prefix))
	}
	if strings.TrimSpace(reason) != "" {
		parts = append(parts, strings.TrimSpace(reason))
	}
	if strings.TrimSpace(message) != "" {
		parts = append(parts, truncateText(strings.TrimSpace(message), 180))
	}
	return strings.Join(parts, ": ")
}

func assistantPodContextItem(pod *corev1.Pod, reason string, totalRestarts int) map[string]any {
	if pod == nil {
		return map[string]any{}
	}
	containers := make([]map[string]any, 0, len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.ContainerStatuses {
		state := "unknown"
		stateReason := ""
		switch {
		case status.State.Running != nil:
			state = "running"
		case status.State.Waiting != nil:
			state = "waiting"
			stateReason = status.State.Waiting.Reason
		case status.State.Terminated != nil:
			state = "terminated"
			stateReason = status.State.Terminated.Reason
		}
		containers = append(containers, map[string]any{
			"name":         status.Name,
			"ready":        status.Ready,
			"restartCount": status.RestartCount,
			"state":        state,
			"reason":       stateReason,
		})
	}
	return map[string]any{
		"namespace":    pod.Namespace,
		"name":         pod.Name,
		"nodeName":     pod.Spec.NodeName,
		"phase":        string(pod.Status.Phase),
		"reason":       reason,
		"restartCount": totalRestarts,
		"age":          humanDurationSince(pod.CreationTimestamp.Time),
		"containers":   containers,
	}
}

func (s *Service) collectAssistantClusterWarningEvents(ctx context.Context, cluster *k8smodels.Cluster, result *assistantPlatformContext) []map[string]any {
	if cluster == nil || cluster.ID == 0 {
		return nil
	}
	clientset, err := s.getAssistantClusterClientset(ctx, result.UserID, cluster.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Warning 事件查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return nil
	}
	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{FieldSelector: "type=Warning"})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("集群 %s Warning 事件列表查询失败: %s", clusterDisplayName(cluster), err.Error()))
		return nil
	}
	sort.Slice(events.Items, func(i, j int) bool {
		return assistantEventTime(events.Items[i]).After(assistantEventTime(events.Items[j]))
	})
	items := make([]map[string]any, 0, minInt(len(events.Items), 30))
	for i := 0; i < minInt(len(events.Items), 30); i++ {
		event := events.Items[i]
		involved := event.InvolvedObject
		items = append(items, map[string]any{
			"namespace":       event.Namespace,
			"reason":          event.Reason,
			"message":         truncateText(event.Message, 240),
			"count":           event.Count,
			"lastTimestamp":   assistantEventTime(event).Format("2006-01-02 15:04:05"),
			"involvedKind":    involved.Kind,
			"involvedName":    involved.Name,
			"involvedField":   involved.FieldPath,
			"reportingSource": event.Source.Component,
		})
	}
	return items
}

func assistantEventTime(event corev1.Event) time.Time {
	if !event.EventTime.Time.IsZero() {
		return event.EventTime.Time
	}
	if !event.LastTimestamp.Time.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.Time.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}

func mapBoolText(value bool, trueText, falseText string) string {
	if value {
		return trueText
	}
	return falseText
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func percentOf(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundFloat(used / total * 100)
}

func formatMilliCPU(value int64) string {
	if value == 0 {
		return "0m"
	}
	if value%1000 == 0 {
		return fmt.Sprintf("%d 核", value/1000)
	}
	return fmt.Sprintf("%.2f 核", float64(value)/1000)
}

func formatBytesCN(value uint64) string {
	if value == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	n := float64(value)
	idx := 0
	for n >= 1024 && idx < len(units)-1 {
		n /= 1024
		idx++
	}
	return fmt.Sprintf("%.1f %s", n, units[idx])
}

func humanDurationSince(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	duration := time.Since(t)
	if duration < 0 {
		duration = 0
	}
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%d 秒", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%d 分钟", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%d 小时", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d 天", int(duration.Hours()/24))
	}
}

func maxFloat(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
