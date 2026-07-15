package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	k8smodels "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	opsHubAgentToolPlatformQuery     = "opshub.platform.query"
	opsHubAgentToolPlatformOverview  = "opshub.platform.overview"
	opsHubAgentToolModuleKnowledge   = "opshub.module.knowledge"
	opsHubAgentToolDataCatalog       = "opshub.data.catalog"
	opsHubAgentToolDataQuery         = "opshub.data.query"
	opsHubAgentToolHostsSummary      = "opshub.hosts.summary"
	opsHubAgentToolHostTopProcesses  = "opshub.hosts.top_processes"
	opsHubAgentToolHostSoftwareProbe = "opshub.hosts.software_probe"
	opsHubAgentToolK8sSummary        = "opshub.kubernetes.summary"
	opsHubAgentToolK8sNamespaces     = "opshub.kubernetes.namespaces"
	opsHubAgentToolK8sPods           = "opshub.kubernetes.pods"
	opsHubAgentToolMonitorSummary    = "opshub.monitor.summary"
	opsHubAgentToolAlertsRecent      = "opshub.alerts.recent"
	opsHubAgentToolSSLSummary        = "opshub.ssl.summary"
	opsHubAgentToolSystemSummary     = "opshub.system.summary"
	opsHubAgentToolTasksSummary      = "opshub.tasks.summary"
	opsHubAgentToolAuditSummary      = "opshub.audit.summary"
	opsHubAgentToolAIopsSummary      = "opshub.aiops.summary"
)

type opsHubAgentToolCall struct {
	Name   string         `json:"name"`
	Reason string         `json:"reason,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

type opsHubAgentToolResult struct {
	Name      string         `json:"name"`
	Reason    string         `json:"reason,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Success   bool           `json:"success"`
	Empty     bool           `json:"empty,omitempty"`
	DataState string         `json:"dataState,omitempty"`
	Data      any            `json:"data,omitempty"`
	Warnings  []string       `json:"warnings,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duration  int64          `json:"durationMs"`
}

type opsHubAgentAction struct {
	Action            string         `json:"action"`
	Tool              string         `json:"tool,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	Params            map[string]any `json:"params,omitempty"`
	AnswerInstruction string         `json:"answerInstruction,omitempty"`
}

type opsHubAgentTurn struct {
	Round          int                    `json:"round"`
	ThoughtSummary string                 `json:"thoughtSummary,omitempty"`
	Action         opsHubAgentAction      `json:"action"`
	Observation    *opsHubAgentToolResult `json:"observation,omitempty"`
}

type opsHubAgentTrace struct {
	Mode             string                  `json:"mode"`
	Question         string                  `json:"question"`
	CollectedAt      string                  `json:"collectedAt"`
	Planner          string                  `json:"planner"`
	ToolCatalog      []map[string]string     `json:"toolCatalog"`
	Turns            []opsHubAgentTurn       `json:"turns,omitempty"`
	ToolCalls        []opsHubAgentToolCall   `json:"toolCalls"`
	ToolResults      []opsHubAgentToolResult `json:"toolResults"`
	ModuleKnowledge  []map[string]any        `json:"moduleKnowledge,omitempty"`
	AnswerPolicy     []string                `json:"answerPolicy"`
	ThinkingSteps    []string                `json:"thinkingSteps,omitempty"`
	FinalInstruction string                  `json:"finalInstruction,omitempty"`
	Errors           []string                `json:"errors,omitempty"`
}

type opsHubAgentPlannerResponse struct {
	ToolCalls []opsHubAgentToolCall `json:"toolCalls"`
}

type opsHubAgentQueryPlan struct {
	Domains             []string `json:"domains"`
	KubernetesResources []string `json:"kubernetesResources,omitempty"`
	Reason              string   `json:"reason"`
}

func (s *Service) buildOpsHubAgentExecutionPlan(ctx context.Context, userID, sessionID, excludeMessageID uint, req ChatRequest, userContent string, onUpdate func([]string, any)) chatExecutionPlan {
	question := strings.TrimSpace(userContent)
	if sessionID != 0 && shouldReuseAssistantContext(question) {
		if previous := s.latestSessionUserQuestion(ctx, sessionID, excludeMessageID); previous != "" {
			question = previous
		}
	}
	moduleKnowledge := selectAssistantModuleKnowledge(question)
	if len(moduleKnowledge) == 0 {
		moduleKnowledge = selectAssistantModuleKnowledge("opshub 平台 所有模块")
	}

	trace, steps := s.runOpsHubAgentLoop(ctx, userID, req.ProviderID, question, moduleKnowledge, onUpdate)

	return chatExecutionPlan{
		SystemPrompt:  opsHubAgentSystemPrompt(),
		Prompt:        buildOpsHubAgentPrompt(userContent, trace),
		ThinkingSteps: steps,
		Context:       trace,
		ToolName:      "opshub_platform_agent",
		ToolParams: map[string]any{
			"question": question,
			"planner":  trace.Planner,
			"tools":    trace.ToolCalls,
			"turns":    trace.Turns,
			"userId":   userID,
		},
		UseHistory: true,
	}
}

func (s *Service) runOpsHubAgentLoop(ctx context.Context, userID, providerID uint, question string, moduleKnowledge []map[string]any, onUpdate func([]string, any)) (opsHubAgentTrace, []string) {
	trace := opsHubAgentTrace{
		Mode:            "opshub_platform_agent_react",
		Question:        truncateText(question, 1000),
		CollectedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Planner:         "model-react",
		ToolCatalog:     opsHubAgentToolCatalog(),
		ModuleKnowledge: moduleKnowledge,
		AnswerPolicy: []string{
			"OpsHub 智能体采用多轮只读工具调用：先理解问题，再查询平台数据，观察结果后决定是否继续查询。",
			"展示给用户的是高层思考摘要和工具调用过程，不输出模型隐藏推理链。",
			"工具结果为空或失败时，回答必须说明具体失败工具、缺少的数据类型和建议查看入口。",
			"智能体不能执行删除、重启、扩缩容、部署等变更操作；只允许调用后端预置的只读查询工具，涉及变更只能给风险说明和人工确认建议。",
		},
	}
	steps := []string{
		"理解问题：识别为 OpsHub 平台相关问题，启用 Platform Agent",
		"进入智能体循环：按“思考摘要 -> 工具调用 -> 观察结果 -> 继续判断”的方式查询",
	}
	emit := func() {
		trace.ThinkingSteps = append([]string(nil), steps...)
		if onUpdate != nil {
			onUpdate(append([]string(nil), steps...), trace)
		}
	}
	emit()

	const maxRounds = 5
	for round := 1; round <= maxRounds; round++ {
		action, thoughtSummary, planner := s.planOpsHubAgentNextAction(ctx, providerID, question, moduleKnowledge, trace, round, maxRounds)
		if planner == "rules" {
			trace.Planner = "rules-react"
		}
		action = normalizeOpsHubAgentAction(question, trace, action, round)
		thoughtSummary = defaultString(strings.TrimSpace(thoughtSummary), defaultOpsHubAgentThoughtSummary(action, round))
		if action.Action == "final" {
			trace.FinalInstruction = defaultString(action.AnswerInstruction, "已有工具观察足够回答用户问题")
			trace.Turns = append(trace.Turns, opsHubAgentTurn{
				Round:          round,
				ThoughtSummary: thoughtSummary,
				Action:         action,
			})
			steps = append(steps, fmt.Sprintf("第 %d 轮判断：%s", round, thoughtSummary))
			steps = append(steps, "生成回答：工具观察已足够，开始组织中文 Markdown 结论")
			emit()
			break
		}

		if action.Tool == "" {
			action.Action = "final"
			action.AnswerInstruction = "没有找到可继续调用的合适只读工具，基于已有观察回答"
			trace.FinalInstruction = action.AnswerInstruction
			trace.Turns = append(trace.Turns, opsHubAgentTurn{
				Round:          round,
				ThoughtSummary: thoughtSummary,
				Action:         action,
			})
			steps = append(steps, "生成回答：没有更多合适工具，基于已有观察回答")
			emit()
			break
		}

		call := opsHubAgentToolCall{
			Name:   action.Tool,
			Reason: defaultString(action.Reason, "本轮需要读取该平台只读数据"),
			Params: action.Params,
		}
		if call.Params == nil {
			call.Params = map[string]any{}
		}
		steps = append(steps, fmt.Sprintf("第 %d 轮思考摘要：%s", round, thoughtSummary))
		steps = append(steps, fmt.Sprintf("第 %d 轮工具调用：%s（%s）", round, call.Name, truncateText(call.Reason, 120)))
		emit()

		result := s.executeOpsHubAgentTool(ctx, userID, question, call)
		trace.ToolCalls = append(trace.ToolCalls, call)
		trace.ToolResults = append(trace.ToolResults, result)
		resultCopy := result
		trace.Turns = append(trace.Turns, opsHubAgentTurn{
			Round:          round,
			ThoughtSummary: thoughtSummary,
			Action:         action,
			Observation:    &resultCopy,
		})
		if !result.Success && strings.TrimSpace(result.Error) != "" {
			trace.Errors = append(trace.Errors, fmt.Sprintf("%s: %s", result.Name, result.Error))
		}
		steps = append(steps, fmt.Sprintf("第 %d 轮观察结果：%s", round, opsHubAgentObservationSummary(result)))
		emit()
	}

	if strings.TrimSpace(trace.FinalInstruction) == "" {
		trace.FinalInstruction = "达到最大工具轮次，基于已有观察回答"
		steps = append(steps, "生成回答：达到最大工具轮次，基于已有观察汇总")
		emit()
	}
	trace.ThinkingSteps = append([]string(nil), steps...)
	return trace, steps
}

func (s *Service) planOpsHubAgentNextAction(ctx context.Context, providerID uint, question string, moduleKnowledge []map[string]any, trace opsHubAgentTrace, round, maxRounds int) (opsHubAgentAction, string, string) {
	plannerCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	catalogJSON, _ := json.MarshalIndent(opsHubAgentToolCatalog(), "", "  ")
	moduleJSON, _ := json.MarshalIndent(moduleKnowledge, "", "  ")
	observationJSON, _ := json.MarshalIndent(buildOpsHubAgentObservationDigest(trace), "", "  ")
	prompt := fmt.Sprintf(`你是 OpsHub Platform Agent 的 ReAct 工具决策器。请只返回严格 JSON，不要解释，不要 Markdown。

用户问题：
%s

当前轮次：%d / %d

可用只读工具：
%s

相关模块知识：
%s

历史工具观察：
%s

返回格式（二选一）：
1. 继续查询：
{"thoughtSummary":"一句话高层思考摘要，不要输出隐藏推理链","action":"tool","tool":"工具名","reason":"为什么此刻需要这个工具","params":{}}
2. 信息足够：
{"thoughtSummary":"一句话说明为什么足够回答","action":"final","answerInstruction":"回答时应该重点覆盖什么"}

决策要求：
1. 只能选择一个工具，不能一次返回多个工具。
2. 所有工具都是只读查询，禁止选择或编造任何变更动作。
3. 如果还没有任何工具观察，必须先调用 opshub.platform.query，并尽量在 params 中带上 domains、kubernetesResources、clusterName、namespace、limit 等你能从问题中识别出的参数。
4. 如果 opshub.platform.query 返回的信息不足，再根据观察结果选择更具体工具继续查，例如 Kubernetes 用 opshub.kubernetes.summary、opshub.kubernetes.namespaces 或 opshub.kubernetes.pods，告警用 opshub.alerts.recent，主机用 opshub.hosts.summary；没有专用工具覆盖的平台数据问题必须使用 opshub.data.query 查询只读数据白名单。
5. 如果用户询问某个命名空间的 Pod 是否正常、异常 Pod、Pod 重启或运行状态，必须选择 opshub.kubernetes.pods，并在 params 中尽量带 clusterName 和 namespace。
6. 如果用户询问主机/服务器/机器中的进程、top、CPU/内存占用最高、占用资源最多的进程，必须选择 opshub.hosts.top_processes；它会通过后端预置只读 SSH 采集器现场查询，不能只用 opshub.hosts.summary。
7. 如果用户询问哪台主机部署/安装/运行了某个软件或服务，例如 nginx、redis、mysql、docker、java、tomcat，必须选择 opshub.hosts.software_probe，并在 params.target 中写入软件名；不能只查进程或主机摘要。
8. 如果用户问的是 OpsHub 功能、页面、流程，优先结合 opshub.module.knowledge 和 opshub.platform.overview。
9. 如果用户问当前有哪些、多少、状态、列表、最近记录、配置、任务、账号、证书、应用、权限、审计等平台数据，必须主动查对应只读工具；如果没有专用工具，选择 opshub.data.query，不要直接回答“上下文没有”。
10. 如果用户问 OpsHub 支持查询哪些对象、智能体能查什么，选择 opshub.data.catalog。
11. 只要还有未调用且明显相关的只读工具，就不能直接 final；工具失败后再基于失败原因回答。
12. 如果已有观察可以回答，就返回 action=final。`, question, round, maxRounds, string(catalogJSON), truncateText(string(moduleJSON), 8000), truncateText(string(observationJSON), 30000))

	result, err := s.callSelectedModel(plannerCtx, providerID, []ChatMessage{
		{Role: "system", Content: "你只负责为 OpsHub 智能体做下一步只读工具决策，必须返回严格 JSON。"},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return heuristicOpsHubAgentNextAction(question, trace), "模型决策不可用，改用规则选择下一个只读工具", "rules"
	}
	action, thoughtSummary, ok := parseOpsHubAgentNextAction(result.Content)
	if !ok {
		return heuristicOpsHubAgentNextAction(question, trace), "模型决策格式不可用，改用规则选择下一个只读工具", "rules"
	}
	return action, thoughtSummary, "model"
}

func parseOpsHubAgentNextAction(content string) (opsHubAgentAction, string, bool) {
	text := strings.TrimSpace(stripModelThinking(content))
	if text == "" {
		return opsHubAgentAction{}, "", false
	}
	text = strings.Trim(text, "` \n\r\t")
	re := regexp.MustCompile(`(?s)\{.*\}`)
	if match := re.FindString(text); match != "" {
		text = match
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return opsHubAgentAction{}, "", false
	}
	thoughtSummary := firstStringFromMap(raw, "thoughtSummary", "thought", "summary", "reasoningSummary")
	action := opsHubAgentAction{
		Action:            firstStringFromMap(raw, "action", "type"),
		Tool:              firstStringFromMap(raw, "tool", "name"),
		Reason:            firstStringFromMap(raw, "reason"),
		Params:            mapAnyFromMap(raw, "params", "arguments"),
		AnswerInstruction: firstStringFromMap(raw, "answerInstruction", "finalAnswer", "instruction"),
	}
	if nested, ok := raw["action"].(map[string]any); ok {
		action.Action = firstStringFromMap(nested, "action", "type")
		action.Tool = defaultString(firstStringFromMap(nested, "tool", "name"), action.Tool)
		action.Reason = defaultString(firstStringFromMap(nested, "reason"), action.Reason)
		if params := mapAnyFromMap(nested, "params", "arguments"); len(params) > 0 {
			action.Params = params
		}
		action.AnswerInstruction = defaultString(firstStringFromMap(nested, "answerInstruction", "finalAnswer", "instruction"), action.AnswerInstruction)
	}
	action.Action = strings.ToLower(strings.TrimSpace(action.Action))
	action.Tool = strings.TrimSpace(action.Tool)
	if action.Params == nil {
		action.Params = map[string]any{}
	}
	if action.Action == "" && action.Tool != "" {
		action.Action = "tool"
	}
	if action.Action == "finish" || action.Action == "answer" || action.Action == "done" {
		action.Action = "final"
	}
	if action.Action != "tool" && action.Action != "final" {
		return opsHubAgentAction{}, thoughtSummary, false
	}
	return action, thoughtSummary, true
}

func normalizeOpsHubAgentAction(question string, trace opsHubAgentTrace, action opsHubAgentAction, round int) opsHubAgentAction {
	action.Action = strings.ToLower(strings.TrimSpace(action.Action))
	action.Tool = strings.TrimSpace(action.Tool)
	if action.Params == nil {
		action.Params = map[string]any{}
	}
	if round == 1 && isOpsHubPlatformQuestion(question) && !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolPlatformQuery) {
		enrichOpsHubAgentPlatformQueryParams(question, action.Params)
		return opsHubAgentAction{
			Action: "tool",
			Tool:   opsHubAgentToolPlatformQuery,
			Reason: defaultString(action.Reason, "先统一理解问题并路由查询 OpsHub 平台数据"),
			Params: action.Params,
		}
	}
	if forced := requiredOpsHubAgentLiveTool(question, trace); forced != "" {
		params := action.Params
		if params == nil {
			params = map[string]any{}
		}
		enrichOpsHubAgentLiveToolParams(question, forced, params)
		return opsHubAgentAction{
			Action: "tool",
			Tool:   forced,
			Reason: "问题需要现场只读采集，不能仅依赖平台已有汇总数据",
			Params: params,
		}
	}
	if forced := requiredOpsHubAgentDataTool(question, trace); forced != "" {
		params := action.Params
		if params == nil {
			params = map[string]any{}
		}
		enrichOpsHubAgentDataToolParams(question, forced, params)
		return opsHubAgentAction{
			Action: "tool",
			Tool:   forced,
			Reason: "问题需要查询 OpsHub 全平台只读数据白名单，不能只依赖少量摘要上下文",
			Params: params,
		}
	}
	if action.Action != "tool" {
		action.Action = "final"
		return action
	}
	valid := map[string]bool{}
	for _, item := range opsHubAgentToolCatalog() {
		valid[item["name"]] = true
	}
	if !valid[action.Tool] {
		if next := heuristicOpsHubAgentNextAction(question, trace); next.Action == "tool" {
			return next
		}
		return opsHubAgentAction{Action: "final", AnswerInstruction: "没有可用的合法只读工具，基于已有观察回答"}
	}
	if (action.Tool == opsHubAgentToolModuleKnowledge || action.Tool == opsHubAgentToolPlatformOverview) && !isOpsHubCapabilityQuestion(question) {
		if next := heuristicOpsHubAgentNextAction(question, trace); next.Action == "tool" {
			return next
		}
		return opsHubAgentAction{Action: "final", AnswerInstruction: "实时数据问题不再查询模块说明，基于已有业务工具观察回答"}
	}
	enrichOpsHubAgentKubernetesParams(question, action.Params)
	enrichOpsHubAgentLiveToolParams(question, action.Tool, action.Params)
	enrichOpsHubAgentDataToolParams(question, action.Tool, action.Params)
	if opsHubAgentToolAlreadyCalled(trace, action.Tool) {
		if next := heuristicOpsHubAgentNextAction(question, trace); next.Action == "tool" {
			return next
		}
		return opsHubAgentAction{Action: "final", AnswerInstruction: "需要的数据工具已经查询过，基于已有观察回答"}
	}
	return action
}

func heuristicOpsHubAgentNextAction(question string, trace opsHubAgentTrace) opsHubAgentAction {
	for _, call := range heuristicOpsHubAgentActionPlan(question) {
		if opsHubAgentToolAlreadyCalled(trace, call.Name) {
			continue
		}
		if call.Params == nil {
			call.Params = map[string]any{}
		}
		if call.Name == opsHubAgentToolPlatformQuery {
			enrichOpsHubAgentPlatformQueryParams(question, call.Params)
		}
		return opsHubAgentAction{
			Action: "tool",
			Tool:   call.Name,
			Reason: call.Reason,
			Params: call.Params,
		}
	}
	return opsHubAgentAction{
		Action:            "final",
		AnswerInstruction: "规则判断已有工具观察足够回答，或没有更多未查询的只读工具",
	}
}

func heuristicOpsHubAgentActionPlan(question string) []opsHubAgentToolCall {
	text := strings.ToLower(strings.TrimSpace(question))
	calls := []opsHubAgentToolCall{
		{Name: opsHubAgentToolPlatformQuery, Reason: "先统一理解问题并路由查询 OpsHub 平台数据", Params: map[string]any{}},
	}
	add := func(name, reason string) {
		calls = append(calls, opsHubAgentToolCall{Name: name, Reason: reason, Params: map[string]any{}})
	}
	if containsAnyText(text, []string{"ssl", "tls", "证书", "域名", "dns", "续期", "部署配置", "任务记录", "过期"}) {
		add(opsHubAgentToolSSLSummary, "问题涉及 SSL 证书、DNS、续期、部署或过期时间")
	}
	if containsAnyText(text, []string{"k8s", "kubernetes", "容器", "集群", "节点", "pod", "deployment", "namespace", "命名空间", "工作负载", "service", "ingress", "configmap", "secret", "pvc", "事件"}) {
		add(opsHubAgentToolK8sSummary, "问题涉及 Kubernetes 集群或资源对象实时状态")
	}
	if containsAnyText(text, []string{"pod"}) && containsAnyText(text, []string{"正常", "运行", "异常", "健康", "重启", "事件", "状态", "命名空间", "namespace"}) {
		add(opsHubAgentToolK8sPods, "问题需要精确查询 Kubernetes Pod 运行状态、Ready 和重启情况")
	}
	if isNamespaceQuestion(text) {
		add(opsHubAgentToolK8sNamespaces, "问题明确询问命名空间数量或列表")
	}
	if containsAnyText(text, []string{"告警", "报警", "p0", "p1", "p2", "alert", "firing", "critical"}) {
		add(opsHubAgentToolAlertsRecent, "问题涉及当前或最近告警")
	}
	if containsAnyText(text, []string{"监控", "数据源", "prometheus", "victoriametrics", "loki", "elasticsearch", "告警规则", "拨测", "通知"}) {
		add(opsHubAgentToolMonitorSummary, "问题涉及监控中心配置或数据源")
	}
	if containsAnyText(text, []string{"主机", "服务器", "机器", "资产", "agent", "云主机", "凭据", "云账号", "终端"}) {
		if isHostSoftwareProbeQuestion(text) {
			add(opsHubAgentToolHostSoftwareProbe, "问题需要主动进入主机现场探测软件/服务是否部署、安装或运行")
		}
		if isHostTopProcessQuestion(text) {
			add(opsHubAgentToolHostTopProcesses, "问题需要主动进入主机现场查询 CPU/内存占用最高的进程")
		}
		add(opsHubAgentToolHostsSummary, "问题涉及主机资产、Agent 或云主机数据")
	}
	if containsAnyText(text, []string{"用户", "角色", "权限", "菜单", "部门", "岗位", "rbac", "登录", "演示账号"}) {
		add(opsHubAgentToolSystemSummary, "问题涉及系统管理和权限数据")
	}
	if containsAnyText(text, []string{"任务", "作业", "脚本", "ansible", "文件分发", "模板"}) {
		add(opsHubAgentToolTasksSummary, "问题涉及任务中心")
	}
	if containsAnyText(text, []string{"审计", "操作日志", "登录日志", "数据日志", "终端审计"}) {
		add(opsHubAgentToolAuditSummary, "问题涉及审计日志")
	}
	if containsAnyText(text, []string{"ai助手", "ai 助手", "智能运维", "智能诊断", "日志分析", "ai配置", "会话记录"}) {
		add(opsHubAgentToolAIopsSummary, "问题涉及智能运维自身配置或数据")
	}
	if isOpsHubDataCatalogQuestion(text) {
		add(opsHubAgentToolDataCatalog, "问题需要了解 OpsHub 智能体可查询的只读数据目录")
	}
	if shouldUseOpsHubAgentDataQuery(question, opsHubAgentTrace{}) {
		add(opsHubAgentToolDataQuery, "问题需要从 OpsHub 全平台只读数据白名单中主动查询明细")
	}
	if isOpsHubCapabilityQuestion(text) {
		add(opsHubAgentToolModuleKnowledge, "问题涉及 OpsHub 模块能力、页面入口或使用流程")
		add(opsHubAgentToolPlatformOverview, "问题涉及 OpsHub 平台菜单和关键对象数量")
	}
	return normalizeOpsHubAgentToolCallsForActionPlan(question, calls)
}

func normalizeOpsHubAgentToolCallsForActionPlan(question string, calls []opsHubAgentToolCall) []opsHubAgentToolCall {
	valid := map[string]bool{}
	for _, item := range opsHubAgentToolCatalog() {
		valid[item["name"]] = true
	}
	normalized := make([]opsHubAgentToolCall, 0, len(calls))
	seen := map[string]bool{}
	for _, call := range calls {
		call.Name = strings.TrimSpace(call.Name)
		if !valid[call.Name] || seen[call.Name] {
			continue
		}
		if call.Params == nil {
			call.Params = map[string]any{}
		}
		if call.Name == opsHubAgentToolPlatformQuery {
			enrichOpsHubAgentPlatformQueryParams(question, call.Params)
		}
		if call.Name == opsHubAgentToolK8sPods || call.Name == opsHubAgentToolK8sNamespaces || call.Name == opsHubAgentToolK8sSummary {
			enrichOpsHubAgentKubernetesParams(question, call.Params)
		}
		enrichOpsHubAgentLiveToolParams(question, call.Name, call.Params)
		enrichOpsHubAgentDataToolParams(question, call.Name, call.Params)
		if strings.TrimSpace(call.Reason) == "" {
			call.Reason = "回答用户问题需要该平台只读数据"
		}
		seen[call.Name] = true
		normalized = append(normalized, call)
	}
	return normalized
}

func isOpsHubCapabilityQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	return containsAnyText(text, []string{"怎么用", "如何使用", "流程", "功能", "菜单", "入口", "在哪里", "支持什么", "有什么", "模块", "说明", "介绍", "文档"})
}

func opsHubAgentToolAlreadyCalled(trace opsHubAgentTrace, name string) bool {
	for _, call := range trace.ToolCalls {
		if call.Name == name {
			return true
		}
	}
	return false
}

func defaultOpsHubAgentThoughtSummary(action opsHubAgentAction, round int) string {
	if action.Action == "tool" && action.Tool != "" {
		return fmt.Sprintf("第 %d 轮需要通过 %s 获取平台事实", round, action.Tool)
	}
	return "已有观察足够生成回答"
}

func buildOpsHubAgentObservationDigest(trace opsHubAgentTrace) []map[string]any {
	items := make([]map[string]any, 0, len(trace.ToolResults))
	for _, result := range trace.ToolResults {
		dataJSON, _ := json.Marshal(compactOpsHubAgentToolDataForModel(result.Name, result.Data))
		items = append(items, map[string]any{
			"tool":       result.Name,
			"success":    result.Success,
			"params":     result.Params,
			"warnings":   result.Warnings,
			"error":      result.Error,
			"durationMs": result.Duration,
			"summary":    opsHubAgentObservationSummary(result),
			"data":       truncateText(string(dataJSON), 14000),
		})
	}
	return items
}

func opsHubAgentObservationSummary(result opsHubAgentToolResult) string {
	if !result.Success {
		return fmt.Sprintf("%s 查询失败：%s", result.Name, defaultString(result.Error, "没有返回可用数据"))
	}
	parts := []string{fmt.Sprintf("%s 查询成功", result.Name)}
	if result.Empty {
		parts = append(parts, "未查询到匹配数据")
	}
	if result.Duration > 0 {
		parts = append(parts, fmt.Sprintf("耗时 %dms", result.Duration))
	}
	if summary := summarizeOpsHubAgentData(result.Data); summary != "" {
		parts = append(parts, summary)
	}
	if len(result.Warnings) > 0 {
		parts = append(parts, "警告："+truncateText(strings.Join(result.Warnings, "；"), 180))
	}
	return strings.Join(parts, "，")
}

func summarizeOpsHubAgentData(data any) string {
	value, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	if subQueries, ok := value["subQueries"]; ok {
		return "已查询子域：" + truncateText(fmt.Sprint(subQueries), 120)
	}
	if total, ok := value["totalNamespaces"]; ok {
		return fmt.Sprintf("命名空间总数：%v，匹配集群：%v", total, value["matchedClusters"])
	}
	if summary, ok := value["summary"]; ok {
		return "返回摘要：" + truncateText(fmt.Sprint(summary), 160)
	}
	if hostSummary, ok := value["hostSummary"]; ok {
		return "返回主机摘要：" + truncateText(fmt.Sprint(hostSummary), 160)
	}
	if topSummary, ok := value["topProcessSummary"]; ok {
		return "返回主机现场进程：" + truncateText(fmt.Sprint(topSummary), 180)
	}
	if softwareSummary, ok := value["softwareProbeSummary"]; ok {
		return "返回主机软件现场探测：" + truncateText(fmt.Sprint(softwareSummary), 180)
	}
	if catalogSummary, ok := value["dataCatalogSummary"]; ok {
		return "返回只读数据目录：" + truncateText(fmt.Sprint(catalogSummary), 180)
	}
	if querySummary, ok := value["dataQuerySummary"]; ok {
		return "返回全平台只读数据：" + truncateText(fmt.Sprint(querySummary), 180)
	}
	if clusterSummary, ok := value["clusterSummary"]; ok {
		return "返回集群摘要：" + truncateText(fmt.Sprint(clusterSummary), 160)
	}
	if moduleKnowledge, ok := value["moduleKnowledge"]; ok {
		return "返回模块知识：" + truncateText(fmt.Sprint(moduleKnowledge), 160)
	}
	return ""
}

func firstStringFromMap(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case fmt.Stringer:
			if text := strings.TrimSpace(v.String()); text != "" {
				return text
			}
		}
	}
	return ""
}

func mapAnyFromMap(values map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if mapped, ok := value.(map[string]any); ok {
			return mapped
		}
	}
	return nil
}

func enrichOpsHubAgentPlatformQueryParams(question string, params map[string]any) {
	if params == nil {
		return
	}
	plan := inferOpsHubAgentQueryPlan(question)
	if len(plan.Domains) > 0 {
		params["domains"] = plan.Domains
	}
	if len(plan.KubernetesResources) > 0 {
		params["kubernetesResources"] = plan.KubernetesResources
	}
	if clusterName := inferClusterNameFromQuestion(question); clusterName != "" {
		params["clusterName"] = clusterName
	}
	if namespace := inferNamespaceFromQuestion(question); namespace != "" {
		params["namespace"] = namespace
	}
}

func enrichOpsHubAgentKubernetesParams(question string, params map[string]any) {
	if params == nil {
		return
	}
	if firstStringParam(params, "clusterName", "cluster", "clusterAlias", "clusterID", "clusterId") == "" {
		if clusterName := inferClusterNameFromQuestion(question); clusterName != "" {
			params["clusterName"] = clusterName
		}
	}
	if firstStringParam(params, "namespace", "ns") == "" {
		if namespace := inferNamespaceFromQuestion(question); namespace != "" {
			params["namespace"] = namespace
		}
	}
}

func inferClusterNameFromQuestion(question string) string {
	text := strings.TrimSpace(question)
	if text == "" {
		return ""
	}
	patterns := []string{
		`(?i)(?:集群|cluster)\s*[:：]?\s*([a-zA-Z0-9._-]{2,64})`,
		`(?i)([a-zA-Z0-9._-]{2,64})\s*(?:的)?\s*(?:k8s|kubernetes)\s*集群`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			value := strings.Trim(match[1], " ，,。:：")
			if value != "" && !containsAnyText(strings.ToLower(value), []string{"k8s", "kubernetes", "cluster"}) {
				return value
			}
		}
	}
	return ""
}

func inferNamespaceFromQuestion(question string) string {
	text := strings.TrimSpace(question)
	if text == "" {
		return ""
	}
	patterns := []string{
		`(?i)([a-zA-Z0-9._-]{2,64})\s*(?:的)?\s*(?:namespace|命名空间)`,
		`(?i)(?:namespace|命名空间)\s*[:：]?\s*([a-zA-Z0-9._-]{2,64})`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			value := strings.Trim(match[1], " ，,。:：")
			if value != "" && !containsAnyText(strings.ToLower(value), []string{"namespace", "命名空间"}) {
				return value
			}
		}
	}
	return ""
}

func assistantPodReadyContainers(pod *corev1.Pod) (int, int) {
	if pod == nil {
		return 0, 0
	}
	ready := 0
	total := len(pod.Status.ContainerStatuses)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
	}
	return ready, total
}

func (s *Service) planOpsHubAgentTools(ctx context.Context, providerID uint, question string, moduleKnowledge []map[string]any) ([]opsHubAgentToolCall, string) {
	plannerCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	catalogJSON, _ := json.MarshalIndent(opsHubAgentToolCatalog(), "", "  ")
	moduleJSON, _ := json.MarshalIndent(moduleKnowledge, "", "  ")
	prompt := fmt.Sprintf(`你是 OpsHub Platform Agent 的工具规划器。请只返回 JSON，不要解释。

用户问题：
%s

可用只读工具：
%s

相关模块知识：
%s

返回格式：
{"toolCalls":[{"name":"工具名","reason":"为什么需要这个工具","params":{}}]}

规划要求：
1. 只选择回答问题必要的工具，最多 8 个。
2. 只要是 OpsHub 平台相关问题，必须优先选择 opshub.platform.query，让后端统一路由并按需查询平台数据。
3. OpsHub 平台流程、菜单、模块能力类问题再选择 opshub.module.knowledge 和 opshub.platform.overview。
4. 主机/服务器/Agent/云主机可追加 opshub.hosts.summary；如果问题涉及进程、top、最高 CPU/内存占用，必须追加 opshub.hosts.top_processes；如果问题涉及某个软件/服务部署、安装或运行在哪些主机，必须追加 opshub.hosts.software_probe。
5. Kubernetes/集群/节点/工作负载/Service/Ingress/ConfigMap/Secret/PVC/Pod 可追加 opshub.kubernetes.summary。
6. 如果问题问“命名空间有多少/有哪些/namespace 列表/namespace 数量”，必须追加 opshub.kubernetes.namespaces。
7. 监控中心/数据源/告警规则/拨测选 opshub.monitor.summary；最近告警/P0/P1/P2 选 opshub.alerts.recent。
8. SSL/证书/DNS/续期/部署/任务记录选 opshub.ssl.summary。
9. 用户/角色/菜单/权限/部门/岗位选 opshub.system.summary。
10. 任务中心/脚本/Ansible/文件分发选 opshub.tasks.summary。
11. 审计/操作日志/登录日志/数据日志选 opshub.audit.summary。
12. AI助手/智能诊断/AI配置/会话记录选 opshub.aiops.summary。
13. 任何 OpsHub 平台数据问题如果没有更精确专用工具，选择 opshub.data.query；如果用户问可查询范围，选择 opshub.data.catalog。`, question, string(catalogJSON), truncateText(string(moduleJSON), 10000))
	result, err := s.callSelectedModel(plannerCtx, providerID, []ChatMessage{
		{Role: "system", Content: "你只负责为 OpsHub 智能体选择只读工具，必须返回严格 JSON。"},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return heuristicOpsHubAgentToolPlan(question), "rules"
	}
	toolCalls := parseOpsHubAgentPlannerToolCalls(result.Content)
	if len(toolCalls) == 0 {
		return heuristicOpsHubAgentToolPlan(question), "rules"
	}
	return toolCalls, "model"
}

func parseOpsHubAgentPlannerToolCalls(content string) []opsHubAgentToolCall {
	text := strings.TrimSpace(stripModelThinking(content))
	if text == "" {
		return nil
	}
	text = strings.Trim(text, "` \n\r\t")
	re := regexp.MustCompile(`(?s)\{.*\}`)
	if match := re.FindString(text); match != "" {
		text = match
	}
	var parsed opsHubAgentPlannerResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}
	return parsed.ToolCalls
}

func heuristicOpsHubAgentToolPlan(question string) []opsHubAgentToolCall {
	text := strings.ToLower(strings.TrimSpace(question))
	calls := []opsHubAgentToolCall{
		{Name: opsHubAgentToolModuleKnowledge, Reason: "读取 OpsHub 模块能力和流程说明"},
		{Name: opsHubAgentToolPlatformOverview, Reason: "读取平台菜单和关键对象数量"},
		{Name: opsHubAgentToolPlatformQuery, Reason: "统一理解问题并主动查询相关 OpsHub 平台数据"},
	}
	add := func(name, reason string) {
		calls = append(calls, opsHubAgentToolCall{Name: name, Reason: reason})
	}
	broadDataQuery := containsAnyText(text, []string{"平台", "opshub", "总览", "概览", "所有", "全部", "多少", "有哪些", "统计"})
	if containsAnyText(text, []string{"主机", "服务器", "机器", "资产", "agent", "云主机", "凭据", "云账号", "终端"}) {
		add(opsHubAgentToolHostsSummary, "问题涉及主机资产、Agent 或云主机数据")
		if isHostSoftwareProbeQuestion(text) {
			add(opsHubAgentToolHostSoftwareProbe, "问题需要主动进入主机现场探测软件/服务部署情况")
		}
		if isHostTopProcessQuestion(text) {
			add(opsHubAgentToolHostTopProcesses, "问题需要主动进入主机现场查询 CPU/内存占用最高的进程")
		}
	}
	if containsAnyText(text, []string{"k8s", "kubernetes", "容器", "集群", "节点", "pod", "deployment", "namespace", "命名空间", "工作负载", "service", "ingress", "configmap", "secret", "pvc"}) {
		add(opsHubAgentToolK8sSummary, "问题涉及 Kubernetes 集群或资源对象")
	}
	if isNamespaceQuestion(text) {
		add(opsHubAgentToolK8sNamespaces, "问题需要精确查询 Kubernetes 命名空间数量或列表")
	}
	if containsAnyText(text, []string{"监控", "数据源", "prometheus", "victoriametrics", "loki", "elasticsearch", "告警规则", "拨测"}) {
		add(opsHubAgentToolMonitorSummary, "问题涉及监控中心配置或数据源")
	}
	if containsAnyText(text, []string{"告警", "报警", "p0", "p1", "p2", "alert", "firing", "critical"}) {
		add(opsHubAgentToolAlertsRecent, "问题涉及当前或最近告警")
	}
	if containsAnyText(text, []string{"ssl", "tls", "证书", "域名", "dns", "续期", "部署配置", "任务记录"}) {
		add(opsHubAgentToolSSLSummary, "问题涉及 SSL 证书、DNS、续期或部署任务")
	}
	if containsAnyText(text, []string{"用户", "角色", "权限", "菜单", "部门", "岗位", "rbac", "演示账号"}) {
		add(opsHubAgentToolSystemSummary, "问题涉及系统管理和权限数据")
	}
	if containsAnyText(text, []string{"任务", "作业", "脚本", "ansible", "文件分发", "模板"}) {
		add(opsHubAgentToolTasksSummary, "问题涉及任务中心")
	}
	if containsAnyText(text, []string{"审计", "操作日志", "登录日志", "数据日志", "终端审计"}) {
		add(opsHubAgentToolAuditSummary, "问题涉及审计日志")
	}
	if containsAnyText(text, []string{"ai助手", "ai 助手", "智能运维", "智能诊断", "日志分析", "ai配置", "会话记录"}) {
		add(opsHubAgentToolAIopsSummary, "问题涉及智能运维自身配置或数据")
	}
	if isOpsHubDataCatalogQuestion(text) {
		add(opsHubAgentToolDataCatalog, "问题需要了解 OpsHub 智能体可查询的只读数据目录")
	}
	if shouldUseOpsHubAgentDataQuery(question, opsHubAgentTrace{}) {
		add(opsHubAgentToolDataQuery, "问题需要从 OpsHub 全平台只读数据白名单中主动查询明细")
	}
	if broadDataQuery && len(calls) <= 2 {
		add(opsHubAgentToolHostsSummary, "平台总览需要主机摘要")
		add(opsHubAgentToolK8sSummary, "平台总览需要 Kubernetes 摘要")
		add(opsHubAgentToolMonitorSummary, "平台总览需要监控摘要")
		add(opsHubAgentToolAlertsRecent, "平台总览需要告警摘要")
		add(opsHubAgentToolSSLSummary, "平台总览需要证书摘要")
		add(opsHubAgentToolSystemSummary, "平台总览需要系统权限摘要")
	}
	return calls
}

func normalizeOpsHubAgentToolCalls(question string, calls []opsHubAgentToolCall) []opsHubAgentToolCall {
	valid := map[string]bool{}
	for _, item := range opsHubAgentToolCatalog() {
		valid[item["name"]] = true
	}
	normalized := make([]opsHubAgentToolCall, 0, len(calls)+2)
	seen := map[string]bool{}
	add := func(call opsHubAgentToolCall) {
		call.Name = strings.TrimSpace(call.Name)
		if !valid[call.Name] || seen[call.Name] {
			return
		}
		if call.Params == nil {
			call.Params = map[string]any{}
		}
		if strings.TrimSpace(call.Reason) == "" {
			call.Reason = "回答用户问题需要该平台只读数据"
		}
		enrichOpsHubAgentLiveToolParams(question, call.Name, call.Params)
		enrichOpsHubAgentDataToolParams(question, call.Name, call.Params)
		seen[call.Name] = true
		normalized = append(normalized, call)
	}
	add(opsHubAgentToolCall{Name: opsHubAgentToolModuleKnowledge, Reason: "读取 OpsHub 平台模块能力"})
	add(opsHubAgentToolCall{Name: opsHubAgentToolPlatformOverview, Reason: "读取 OpsHub 平台菜单和关键对象数量"})
	if isOpsHubPlatformQuestion(question) {
		add(opsHubAgentToolCall{Name: opsHubAgentToolPlatformQuery, Reason: "统一理解问题并主动查询相关 OpsHub 平台数据"})
	}
	if isNamespaceQuestion(question) {
		add(opsHubAgentToolCall{Name: opsHubAgentToolK8sNamespaces, Reason: "问题明确询问命名空间数量或列表，主动查询命名空间"})
	}
	for _, call := range calls {
		add(call)
	}
	if len(normalized) == 2 && isOpsHubPlatformQuestion(question) {
		for _, call := range heuristicOpsHubAgentToolPlan(question) {
			add(call)
		}
	}
	if len(normalized) > 8 {
		normalized = normalized[:8]
	}
	return normalized
}

func isNamespaceQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	return containsAnyText(text, []string{"命名空间", "namespace"}) &&
		containsAnyText(text, []string{"多少", "几个", "数量", "有哪些", "列表", "列出", "查看", "看下", "查一下"})
}

func (s *Service) executeOpsHubAgentTool(ctx context.Context, userID uint, question string, call opsHubAgentToolCall) opsHubAgentToolResult {
	start := time.Now()
	result := opsHubAgentToolResult{
		Name:   call.Name,
		Reason: call.Reason,
		Params: map[string]any{
			"question": question,
			"userId":   userID,
		},
	}
	for key, value := range call.Params {
		result.Params[key] = value
	}
	platformContext := assistantPlatformContext{
		UserID:      userID,
		CollectedAt: time.Now().Format("2006-01-02 15:04:05"),
		Intent:      "platform",
		Question:    truncateText(question, 1000),
	}
	defer func() {
		result.Duration = time.Since(start).Milliseconds()
		useful := hasOpsHubAgentUsefulData(result.Data)
		if len(platformContext.Errors) > 0 && result.Error == "" {
			result.Warnings = append(result.Warnings, platformContext.Errors...)
		}
		if result.Error == "" {
			result.Success = true
			if useful {
				result.DataState = "available"
			} else {
				result.Empty = true
				result.DataState = "empty"
				result.Warnings = append(result.Warnings, "工具执行成功，但未查询到匹配数据")
			}
		} else {
			result.DataState = "error"
		}
		result.Warnings = uniqueStrings(result.Warnings)
	}()

	switch call.Name {
	case opsHubAgentToolPlatformQuery:
		data, warnings := s.collectOpsHubAgentPlatformQuery(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolPlatformOverview:
		s.collectAssistantOverviewContext(ctx, &platformContext)
		result.Data = platformContext.PlatformSummary
	case opsHubAgentToolModuleKnowledge:
		result.Data = map[string]any{
			"moduleKnowledge": selectAssistantModuleKnowledge(question),
			"moduleCatalog":   assistantModuleCatalog(),
		}
	case opsHubAgentToolDataCatalog:
		result.Data = collectOpsHubAgentDataCatalog(question)
	case opsHubAgentToolDataQuery:
		data, warnings := s.collectOpsHubAgentDataQuery(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolHostsSummary:
		s.collectAssistantHostContext(ctx, &platformContext)
		result.Data = map[string]any{
			"hostSummary": platformContext.HostSummary,
			"hosts":       platformContext.Hosts,
		}
	case opsHubAgentToolHostTopProcesses:
		data, warnings := s.collectOpsHubAgentHostTopProcesses(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolHostSoftwareProbe:
		data, warnings := s.collectOpsHubAgentHostSoftwareProbe(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolK8sSummary:
		s.collectAssistantClusterContext(ctx, &platformContext)
		result.Data = map[string]any{
			"clusterSummary": platformContext.ClusterSummary,
			"clusters":       platformContext.Clusters,
		}
	case opsHubAgentToolK8sNamespaces:
		data, warnings := s.collectOpsHubAgentKubernetesNamespaces(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolK8sPods:
		data, warnings := s.collectOpsHubAgentKubernetesPods(ctx, userID, question, call.Params)
		result.Data = data
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	case opsHubAgentToolMonitorSummary:
		s.collectAssistantMonitorContext(ctx, &platformContext)
		result.Data = platformContext.MonitorSummary
	case opsHubAgentToolAlertsRecent:
		s.collectAssistantAlertContext(ctx, &platformContext)
		result.Data = map[string]any{
			"alertSummary": platformContext.AlertSummary,
			"recentAlerts": platformContext.RecentAlerts,
		}
	case opsHubAgentToolSSLSummary:
		s.collectAssistantCertContext(ctx, &platformContext)
		result.Data = platformContext.CertSummary
	case opsHubAgentToolSystemSummary:
		s.collectAssistantSystemContext(ctx, &platformContext)
		result.Data = platformContext.SystemSummary
	case opsHubAgentToolTasksSummary:
		s.collectAssistantTaskContext(ctx, &platformContext)
		result.Data = platformContext.TaskSummary
	case opsHubAgentToolAuditSummary:
		s.collectAssistantAuditContext(ctx, &platformContext)
		result.Data = platformContext.AuditSummary
	case opsHubAgentToolAIopsSummary:
		s.collectAssistantAIContext(ctx, &platformContext)
		result.Data = platformContext.AISummary
	default:
		result.Error = "未知工具"
	}
	return result
}

func (s *Service) collectOpsHubAgentKubernetesNamespaces(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	var clusters []k8smodels.Cluster
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "alias", "version", "status", "region", "provider", "node_count", "pod_count", "status_synced_at"}).
		Order("id DESC").
		Find(&clusters).Error; err != nil {
		return map[string]any{"clusters": []any{}, "totalNamespaces": 0}, []string{"Kubernetes 集群查询失败: " + err.Error()}
	}

	clusterHint := firstStringParam(params, "clusterName", "cluster", "clusterAlias", "clusterID", "clusterId")
	matchText := strings.TrimSpace(strings.Join([]string{question, clusterHint}, " "))
	hasMention := hasMentionedCluster(matchText, clusters)
	matched := make([]k8smodels.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if !hasMention || clusterMatchesQuestion(matchText, &cluster) {
			matched = append(matched, cluster)
		}
	}

	warnings := make([]string, 0)
	items := make([]map[string]any, 0, len(matched))
	totalNamespaces := 0
	for _, cluster := range matched {
		item := map[string]any{
			"id":             cluster.ID,
			"name":           cluster.Name,
			"alias":          cluster.Alias,
			"displayName":    clusterDisplayName(&cluster),
			"status":         clusterStatusText(cluster.Status),
			"version":        cluster.Version,
			"provider":       cluster.Provider,
			"region":         cluster.Region,
			"nodeCount":      cluster.NodeCount,
			"podCount":       cluster.PodCount,
			"statusSyncedAt": formatTimePtr(cluster.StatusSyncedAt),
		}
		clientset, err := s.getAssistantClusterClientset(ctx, userID, cluster.ID)
		if err != nil {
			item["querySuccess"] = false
			item["error"] = err.Error()
			warnings = append(warnings, fmt.Sprintf("集群 %s 命名空间查询失败: %s", clusterDisplayName(&cluster), err.Error()))
			items = append(items, item)
			continue
		}
		namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			item["querySuccess"] = false
			item["error"] = err.Error()
			warnings = append(warnings, fmt.Sprintf("集群 %s 命名空间列表查询失败: %s", clusterDisplayName(&cluster), err.Error()))
			items = append(items, item)
			continue
		}

		namespaceItems := make([]map[string]any, 0, len(namespaces.Items))
		for _, ns := range namespaces.Items {
			namespaceItems = append(namespaceItems, map[string]any{
				"name":   ns.Name,
				"status": string(ns.Status.Phase),
				"age":    humanDurationSince(ns.CreationTimestamp.Time),
			})
		}
		sort.Slice(namespaceItems, func(i, j int) bool {
			return fmt.Sprint(namespaceItems[i]["name"]) < fmt.Sprint(namespaceItems[j]["name"])
		})
		namespaceCount := len(namespaceItems)
		totalNamespaces += namespaceCount
		if len(namespaceItems) > 200 {
			namespaceItems = namespaceItems[:200]
			item["namespacesTruncated"] = true
		}
		item["querySuccess"] = true
		item["namespaceCount"] = namespaceCount
		item["namespaces"] = namespaceItems
		items = append(items, item)
	}

	if hasMention && len(matched) == 0 {
		warnings = append(warnings, "未匹配到问题中提到的 Kubernetes 集群，请确认集群名称或别名是否正确")
	}

	return map[string]any{
		"matchedByQuestion": hasMention,
		"clusterHint":       clusterHint,
		"matchedClusters":   len(matched),
		"totalClusters":     len(clusters),
		"totalNamespaces":   totalNamespaces,
		"clusters":          items,
	}, warnings
}

func (s *Service) collectOpsHubAgentKubernetesPods(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	var clusters []k8smodels.Cluster
	if err := s.db.WithContext(ctx).
		Select([]string{"id", "name", "alias", "version", "status", "region", "provider", "node_count", "pod_count", "status_synced_at"}).
		Order("id DESC").
		Find(&clusters).Error; err != nil {
		return map[string]any{"clusters": []any{}, "totalPods": 0}, []string{"Kubernetes 集群查询失败: " + err.Error()}
	}

	clusterHint := firstStringParam(params, "clusterName", "cluster", "clusterAlias", "clusterID", "clusterId")
	namespaceHint := firstStringParam(params, "namespace", "ns")
	if namespaceHint == "" {
		namespaceHint = inferNamespaceFromQuestion(question)
	}
	matchText := strings.TrimSpace(strings.Join([]string{question, clusterHint}, " "))
	hasMention := hasMentionedCluster(matchText, clusters)
	matched := make([]k8smodels.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if !hasMention || clusterMatchesQuestion(matchText, &cluster) {
			matched = append(matched, cluster)
		}
	}

	warnings := make([]string, 0)
	items := make([]map[string]any, 0, len(matched))
	totalPods := 0
	totalRunning := 0
	totalAbnormal := 0
	totalRestarts := 0
	for _, cluster := range matched {
		item := map[string]any{
			"id":             cluster.ID,
			"name":           cluster.Name,
			"alias":          cluster.Alias,
			"displayName":    clusterDisplayName(&cluster),
			"status":         clusterStatusText(cluster.Status),
			"version":        cluster.Version,
			"provider":       cluster.Provider,
			"region":         cluster.Region,
			"namespace":      namespaceHint,
			"statusSyncedAt": formatTimePtr(cluster.StatusSyncedAt),
		}
		clientset, err := s.getAssistantClusterClientset(ctx, userID, cluster.ID)
		if err != nil {
			item["querySuccess"] = false
			item["error"] = err.Error()
			warnings = append(warnings, fmt.Sprintf("集群 %s Pod 查询失败: %s", clusterDisplayName(&cluster), err.Error()))
			items = append(items, item)
			continue
		}
		pods, err := clientset.CoreV1().Pods(namespaceHint).List(ctx, metav1.ListOptions{})
		if err != nil {
			item["querySuccess"] = false
			item["error"] = err.Error()
			warnings = append(warnings, fmt.Sprintf("集群 %s Pod 列表查询失败: %s", clusterDisplayName(&cluster), err.Error()))
			items = append(items, item)
			continue
		}

		podItems := make([]map[string]any, 0, len(pods.Items))
		running := 0
		abnormal := 0
		restarts := 0
		for _, pod := range pods.Items {
			restartCount := assistantPodRestartCount(&pod)
			restarts += restartCount
			readyContainers, totalContainers := assistantPodReadyContainers(&pod)
			ready := totalContainers > 0 && readyContainers == totalContainers && string(pod.Status.Phase) == "Running"
			reason := assistantPodAbnormalReason(&pod)
			if string(pod.Status.Phase) == "Running" {
				running++
			}
			if !ready || reason != "" || restartCount > 0 {
				abnormal++
			}
			podItems = append(podItems, map[string]any{
				"namespace":       pod.Namespace,
				"name":            pod.Name,
				"phase":           string(pod.Status.Phase),
				"ready":           ready,
				"readyContainers": readyContainers,
				"totalContainers": totalContainers,
				"restartCount":    restartCount,
				"reason":          reason,
				"nodeName":        pod.Spec.NodeName,
				"podIP":           pod.Status.PodIP,
				"age":             humanDurationSince(pod.CreationTimestamp.Time),
			})
		}
		sort.Slice(podItems, func(i, j int) bool {
			leftReady, _ := podItems[i]["ready"].(bool)
			rightReady, _ := podItems[j]["ready"].(bool)
			if leftReady != rightReady {
				return !leftReady
			}
			leftRestarts := intFromAny(podItems[i]["restartCount"])
			rightRestarts := intFromAny(podItems[j]["restartCount"])
			if leftRestarts != rightRestarts {
				return leftRestarts > rightRestarts
			}
			return fmt.Sprint(podItems[i]["name"]) < fmt.Sprint(podItems[j]["name"])
		})
		podCount := len(podItems)
		totalPods += podCount
		totalRunning += running
		totalAbnormal += abnormal
		totalRestarts += restarts
		if len(podItems) > 200 {
			podItems = podItems[:200]
			item["podsTruncated"] = true
		}
		item["querySuccess"] = true
		item["podCount"] = podCount
		item["runningPods"] = running
		item["abnormalPods"] = abnormal
		item["restartCount"] = restarts
		item["pods"] = podItems
		items = append(items, item)
	}

	if hasMention && len(matched) == 0 {
		warnings = append(warnings, "未匹配到问题中提到的 Kubernetes 集群，请确认集群名称或别名是否正确")
	}
	return map[string]any{
		"matchedByQuestion": hasMention,
		"clusterHint":       clusterHint,
		"namespaceHint":     namespaceHint,
		"matchedClusters":   len(matched),
		"totalClusters":     len(clusters),
		"totalPods":         totalPods,
		"totalRunningPods":  totalRunning,
		"totalAbnormalPods": totalAbnormal,
		"totalRestarts":     totalRestarts,
		"clusters":          items,
	}, warnings
}

func (s *Service) collectOpsHubAgentPlatformQuery(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	plan := inferOpsHubAgentQueryPlan(question)
	if domains := stringSliceParam(params, "domains", "domain"); len(domains) > 0 {
		plan.Domains = normalizeOpsHubAgentDomains(domains)
	}
	if resources := stringSliceParam(params, "kubernetesResources", "k8sResources", "resources", "resource"); len(resources) > 0 {
		plan.KubernetesResources = normalizeOpsHubAgentKubernetesResources(resources)
	}
	platformContext := assistantPlatformContext{
		UserID:      userID,
		CollectedAt: time.Now().Format("2006-01-02 15:04:05"),
		Intent:      "platform",
		Question:    truncateText(question, 1000),
	}
	data := map[string]any{
		"routerPlan": plan,
		"subQueries": []string{},
	}
	subQueries := make([]string, 0)
	warnings := make([]string, 0)
	appendWarnings := func() {
		if len(platformContext.Errors) == 0 {
			return
		}
		warnings = append(warnings, platformContext.Errors...)
		platformContext.Errors = nil
	}

	for _, domain := range plan.Domains {
		switch domain {
		case "overview":
			s.collectAssistantOverviewContext(ctx, &platformContext)
			data["platformOverview"] = platformContext.PlatformSummary
			subQueries = append(subQueries, "platformOverview")
			appendWarnings()
		case "module":
			data["moduleKnowledge"] = selectAssistantModuleKnowledge(question)
			data["moduleCatalog"] = assistantModuleCatalog()
			subQueries = append(subQueries, "moduleKnowledge")
		case "system":
			s.collectAssistantSystemContext(ctx, &platformContext)
			data["system"] = platformContext.SystemSummary
			subQueries = append(subQueries, "system")
			appendWarnings()
		case "hosts":
			s.collectAssistantHostContext(ctx, &platformContext)
			data["hosts"] = map[string]any{
				"summary": platformContext.HostSummary,
				"items":   platformContext.Hosts,
			}
			subQueries = append(subQueries, "hosts")
			appendWarnings()
		case "kubernetes":
			s.collectAssistantClusterContext(ctx, &platformContext)
			data["kubernetes"] = map[string]any{
				"summary":  platformContext.ClusterSummary,
				"clusters": platformContext.Clusters,
			}
			subQueries = append(subQueries, "kubernetes")
			appendWarnings()
			if containsString(plan.KubernetesResources, "namespaces") {
				namespaceData, namespaceWarnings := s.collectOpsHubAgentKubernetesNamespaces(ctx, userID, question, params)
				data["kubernetesNamespaces"] = namespaceData
				subQueries = append(subQueries, "kubernetesNamespaces")
				warnings = append(warnings, namespaceWarnings...)
			}
			if containsString(plan.KubernetesResources, "pods") {
				podData, podWarnings := s.collectOpsHubAgentKubernetesPods(ctx, userID, question, params)
				data["kubernetesPods"] = podData
				subQueries = append(subQueries, "kubernetesPods")
				warnings = append(warnings, podWarnings...)
			}
		case "monitor":
			s.collectAssistantMonitorContext(ctx, &platformContext)
			data["monitor"] = platformContext.MonitorSummary
			subQueries = append(subQueries, "monitor")
			appendWarnings()
		case "alerts":
			s.collectAssistantAlertContext(ctx, &platformContext)
			data["alerts"] = map[string]any{
				"summary": platformContext.AlertSummary,
				"recent":  platformContext.RecentAlerts,
			}
			subQueries = append(subQueries, "alerts")
			appendWarnings()
		case "ssl":
			s.collectAssistantCertContext(ctx, &platformContext)
			data["ssl"] = platformContext.CertSummary
			subQueries = append(subQueries, "ssl")
			appendWarnings()
		case "tasks":
			s.collectAssistantTaskContext(ctx, &platformContext)
			data["tasks"] = platformContext.TaskSummary
			subQueries = append(subQueries, "tasks")
			appendWarnings()
		case "audit":
			s.collectAssistantAuditContext(ctx, &platformContext)
			data["audit"] = platformContext.AuditSummary
			subQueries = append(subQueries, "audit")
			appendWarnings()
		case "aiops":
			s.collectAssistantAIContext(ctx, &platformContext)
			data["aiops"] = platformContext.AISummary
			subQueries = append(subQueries, "aiops")
			appendWarnings()
		}
	}

	data["subQueries"] = subQueries
	return data, uniqueStrings(warnings)
}

func inferOpsHubAgentQueryPlan(question string) opsHubAgentQueryPlan {
	text := strings.ToLower(strings.TrimSpace(question))
	domains := []string{"module", "overview"}
	k8sResources := make([]string, 0)
	addDomain := func(domain string) {
		if !containsString(domains, domain) {
			domains = append(domains, domain)
		}
	}
	addK8sResource := func(resource string) {
		if !containsString(k8sResources, resource) {
			k8sResources = append(k8sResources, resource)
		}
	}

	if containsAnyText(text, []string{"主机", "服务器", "机器", "资产", "agent", "云主机", "凭据", "云账号", "终端"}) {
		addDomain("hosts")
	}
	if containsAnyText(text, []string{"k8s", "kubernetes", "容器", "集群", "节点", "pod", "deployment", "namespace", "命名空间", "工作负载", "service", "ingress", "configmap", "secret", "pvc", "存储", "网络管理", "配置管理"}) {
		addDomain("kubernetes")
	}
	if isNamespaceQuestion(text) {
		addDomain("kubernetes")
		addK8sResource("namespaces")
	}
	if containsAnyText(text, []string{"节点", "node", "资源占用", "cpu", "内存"}) {
		addK8sResource("nodes")
	}
	if containsAnyText(text, []string{"pod", "异常", "重启", "事件"}) {
		addK8sResource("pods")
	}
	if containsAnyText(text, []string{"deployment", "工作负载", "statefulset", "daemonset", "job", "cronjob"}) {
		addK8sResource("workloads")
	}
	if containsAnyText(text, []string{"service", "ingress", "endpoint", "networkpolicy", "网络"}) {
		addK8sResource("network")
	}
	if containsAnyText(text, []string{"configmap", "secret", "配置"}) {
		addK8sResource("configs")
	}
	if containsAnyText(text, []string{"pvc", "pv", "storageclass", "存储"}) {
		addK8sResource("storage")
	}
	if containsAnyText(text, []string{"监控", "数据源", "prometheus", "victoriametrics", "loki", "elasticsearch", "告警规则", "拨测", "通知"}) {
		addDomain("monitor")
	}
	if containsAnyText(text, []string{"告警", "报警", "p0", "p1", "p2", "alert", "firing", "critical"}) {
		addDomain("alerts")
	}
	if containsAnyText(text, []string{"ssl", "tls", "证书", "域名", "dns", "续期", "部署配置", "任务记录"}) {
		addDomain("ssl")
	}
	if containsAnyText(text, []string{"用户", "角色", "权限", "菜单", "部门", "岗位", "rbac", "登录", "演示账号"}) {
		addDomain("system")
	}
	if containsAnyText(text, []string{"任务", "作业", "脚本", "ansible", "文件分发", "模板"}) {
		addDomain("tasks")
	}
	if containsAnyText(text, []string{"审计", "操作日志", "登录日志", "数据日志", "终端审计"}) {
		addDomain("audit")
	}
	if containsAnyText(text, []string{"ai助手", "ai 助手", "智能运维", "智能诊断", "日志分析", "ai配置", "会话记录"}) {
		addDomain("aiops")
	}
	if containsAnyText(text, []string{"平台", "opshub", "总览", "概览", "所有", "全部模块", "整体", "有哪些功能"}) && len(domains) <= 2 {
		for _, domain := range []string{"hosts", "kubernetes", "monitor", "alerts", "ssl", "system", "tasks", "audit", "aiops"} {
			addDomain(domain)
		}
	}

	return opsHubAgentQueryPlan{
		Domains:             domains,
		KubernetesResources: k8sResources,
		Reason:              "根据用户问题中的模块名、资源类型和查询词自动选择平台只读数据源",
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstStringParam(params map[string]any, keys ...string) string {
	if params == nil {
		return ""
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case float64:
			if v > 0 {
				return fmt.Sprintf("%.0f", v)
			}
		case int:
			if v > 0 {
				return fmt.Sprintf("%d", v)
			}
		case uint:
			if v > 0 {
				return fmt.Sprintf("%d", v)
			}
		}
	}
	return ""
}

func stringSliceParam(params map[string]any, keys ...string) []string {
	if params == nil {
		return nil
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case []string:
			return uniqueStrings(v)
		case []any:
			items := make([]string, 0, len(v))
			for _, item := range v {
				if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
					items = append(items, text)
				}
			}
			return uniqueStrings(items)
		case string:
			parts := strings.FieldsFunc(v, func(r rune) bool {
				return r == ',' || r == '，' || r == ' ' || r == ';' || r == '；'
			})
			items := make([]string, 0, len(parts))
			for _, part := range parts {
				if text := strings.TrimSpace(part); text != "" {
					items = append(items, text)
				}
			}
			return uniqueStrings(items)
		}
	}
	return nil
}

func normalizeOpsHubAgentDomains(values []string) []string {
	aliases := map[string]string{
		"platform":    "overview",
		"overview":    "overview",
		"module":      "module",
		"modules":     "module",
		"host":        "hosts",
		"hosts":       "hosts",
		"asset":       "hosts",
		"assets":      "hosts",
		"k8s":         "kubernetes",
		"kubernetes":  "kubernetes",
		"container":   "kubernetes",
		"monitor":     "monitor",
		"monitoring":  "monitor",
		"alert":       "alerts",
		"alerts":      "alerts",
		"ssl":         "ssl",
		"cert":        "ssl",
		"certificate": "ssl",
		"system":      "system",
		"rbac":        "system",
		"task":        "tasks",
		"tasks":       "tasks",
		"audit":       "audit",
		"ai":          "aiops",
		"aiops":       "aiops",
	}
	domains := []string{"module", "overview"}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if domain := aliases[key]; domain != "" && !containsString(domains, domain) {
			domains = append(domains, domain)
		}
	}
	return domains
}

func normalizeOpsHubAgentKubernetesResources(values []string) []string {
	aliases := map[string]string{
		"namespace":      "namespaces",
		"namespaces":     "namespaces",
		"ns":             "namespaces",
		"node":           "nodes",
		"nodes":          "nodes",
		"pod":            "pods",
		"pods":           "pods",
		"workload":       "workloads",
		"workloads":      "workloads",
		"deployment":     "workloads",
		"deployments":    "workloads",
		"service":        "network",
		"services":       "network",
		"ingress":        "network",
		"ingresses":      "network",
		"network":        "network",
		"config":         "configs",
		"configs":        "configs",
		"configmap":      "configs",
		"configmaps":     "configs",
		"secret":         "configs",
		"secrets":        "configs",
		"storage":        "storage",
		"pvc":            "storage",
		"pv":             "storage",
		"storageclass":   "storage",
		"storageclasses": "storage",
	}
	resources := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if resource := aliases[key]; resource != "" && !containsString(resources, resource) {
			resources = append(resources, resource)
		}
	}
	return resources
}

func hasOpsHubAgentUsefulData(data any) bool {
	if data == nil {
		return false
	}
	switch v := data.(type) {
	case map[string]any:
		return len(v) > 0
	case []map[string]any:
		return len(v) > 0
	case []any:
		return len(v) > 0
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return true
	}
}

func opsHubAgentToolCatalog() []map[string]string {
	return []map[string]string{
		{"name": opsHubAgentToolPlatformQuery, "description": "OpsHub 通用平台查询工具：先理解问题，再路由到资产、K8s、监控、告警、SSL、系统、任务、审计、AI配置等只读数据源。"},
		{"name": opsHubAgentToolPlatformOverview, "description": "读取 OpsHub 菜单、模块目录和关键对象数量。"},
		{"name": opsHubAgentToolModuleKnowledge, "description": "读取 OpsHub 模块能力、流程和注意事项。"},
		{"name": opsHubAgentToolDataCatalog, "description": "读取 OpsHub 智能体可主动查询的只读数据资源目录，包括资源名、模块、别名、安全字段和能力边界。"},
		{"name": opsHubAgentToolDataQuery, "description": "OpsHub 全平台只读数据查询：根据资源目录主动查询主机、资产、K8s、监控、告警、SSL、任务、审计、系统、统一认证、Nginx、AI 运维等安全白名单字段；禁止写入和敏感字段。"},
		{"name": opsHubAgentToolHostsSummary, "description": "读取主机资产、Agent、云主机、资源配置和高负载主机摘要。"},
		{"name": opsHubAgentToolHostTopProcesses, "description": "主动现场查询主机进程 Top：通过后端预置只读 SSH 采集命令查询每台可访问主机 CPU/内存占用最高的进程，并返回每台主机的成功或失败原因。"},
		{"name": opsHubAgentToolHostSoftwareProbe, "description": "主动现场探测主机软件/服务部署情况：通过后端预置只读 SSH 探测进程、systemd/service、二进制、rpm/dpkg 包、配置目录和容器线索。"},
		{"name": opsHubAgentToolK8sSummary, "description": "读取 Kubernetes 集群、节点、Pod 健康、命名空间资源和对象清单。"},
		{"name": opsHubAgentToolK8sNamespaces, "description": "按问题中的集群名/别名精确查询 Kubernetes 命名空间数量和命名空间列表。"},
		{"name": opsHubAgentToolK8sPods, "description": "按集群名和命名空间精确查询 Kubernetes Pod 列表、运行状态、Ready、重启次数和异常原因。"},
		{"name": opsHubAgentToolMonitorSummary, "description": "读取监控数据源、告警规则、拨测、通知对象和故障中心摘要。"},
		{"name": opsHubAgentToolAlertsRecent, "description": "读取告警等级、状态分布和最近告警事件。"},
		{"name": opsHubAgentToolSSLSummary, "description": "读取 SSL 证书、DNS 配置、部署配置、自动续期和最近任务状态。"},
		{"name": opsHubAgentToolSystemSummary, "description": "读取用户、角色、菜单、部门、岗位和资产权限摘要。"},
		{"name": opsHubAgentToolTasksSummary, "description": "读取任务作业、任务模板和 Ansible 任务摘要。"},
		{"name": opsHubAgentToolAuditSummary, "description": "读取操作审计、登录日志和数据日志摘要。"},
		{"name": opsHubAgentToolAIopsSummary, "description": "读取 AI 模型配置、会话、消息、诊断任务和风险规则摘要。"},
	}
}

func opsHubAgentSystemPrompt() string {
	return "你是 OpsHub Platform Agent，专门回答 OpsHub 平台相关问题。你会基于后端已经执行的只读工具结果回答，不要编造工具结果中不存在的实时数据。输出中文 Markdown，结构清晰。遇到缺少上下文时，必须优先检查工具轨迹里是否已有可用的主动查询工具；如果工具已经失败，要说明具体失败原因。涉及删除、重启、扩缩容、部署、执行变更命令等操作时，必须明确说明你没有执行，只能给风险和人工确认建议。不要输出模型内部思考链或 <think> 内容。"
}

func buildOpsHubAgentPrompt(question string, trace opsHubAgentTrace) string {
	answerTrace := compactOpsHubAgentTraceForAnswer(trace)
	traceJSON, _ := json.MarshalIndent(answerTrace, "", "  ")
	return fmt.Sprintf(`用户问题：
%s

OpsHub Platform Agent 已执行的只读工具轨迹 JSON（已压缩为回答专用结构，保留关键事实）：
%s

请基于以上工具结果回答。
要求：
1. 这是 OpsHub 专属智能体回答，不要说“我没有接入平台数据”；工具结果就是你本次可用的平台数据。
2. 优先查看 opshub.platform.query 的 routerPlan、subQueries 和对应域数据，它代表智能体主动理解问题并查询平台后的结果。
3. 如果用户问平台功能/流程/页面入口，优先使用 moduleKnowledge、moduleCatalog、platformOverview 的菜单和能力说明。
4. 如果用户问当前资源/状态/有哪些/多少，优先使用 opshub.platform.query 中对应域的实时摘要和列表；再结合专用工具补充。
5. 如果某个工具失败或结果为空，要说明具体工具、缺少的数据类型、可能原因，以及用户可去 OpsHub 哪个模块查看。
6. 如果问题询问命名空间数量或列表，必须优先使用 kubernetesNamespaces 或 opshub.kubernetes.namespaces 的 namespaceCount、totalNamespaces 和 namespaces，不要只看 kubernetes.summary。
7. 如果问题询问 Pod 是否正常、异常 Pod、Pod 重启或某命名空间 Pod 状态，必须优先使用 kubernetesPods 或 opshub.kubernetes.pods 的 podCount、runningPods、abnormalPods、restartCount 和 pods 列表。
8. Kubernetes 节点资源使用优先使用 kubernetes.summary 中的 clusters[].nodes；命名空间资源使用 namespaceResources；对象列表使用 objectInventory；Pod 异常使用 podHealthSummary、abnormalPods、highRestartPods、recentWarningEvents。
9. 告警等级统一显示 P0/P1/P2；不要直接用 critical/warning/info 作为中文页面展示。
10. 不要输出 JSON 原文，除非用户明确要求。优先给结论、关键统计、表格和下一步建议。
11. 如果问题涉及主机进程、CPU/内存占用最高进程、资源占用最多进程，必须优先使用 hostTopProcesses/opshub.hosts.top_processes 的现场查询结果；不要回答“上下文没有进程数据”。
12. 对 opshub.hosts.top_processes：只要 topProcessSummary.queriedHosts > 0，就按“已现场查询，部分主机失败”回答；不要写“工具顶层 success=false”，也不要写“工具轨迹被截断”。失败主机只需要按 error 字段说明原因。
13. 如果问题涉及某个软件/服务部署在哪些主机，必须优先使用 softwareProbe/opshub.hosts.software_probe 的 deployed、running、evidence 字段；不要只根据进程是否存在下结论。
14. 如果问题涉及任意 OpsHub 平台数据，优先使用 dataQuery/opshub.data.query 的 resources、rows、totalRows；这是后端白名单只读查询结果，不要说“没有接入平台所有接口”。
15. dataQuery 只代表数据库安全字段；Kubernetes 实时对象、主机现场命令、外部云厂商实时数据仍以专用只读工具结果为准。
16. 只读回答，不声称已经执行任何变更操作。`, question, truncateText(string(traceJSON), 120000))
}

func compactOpsHubAgentTraceForAnswer(trace opsHubAgentTrace) opsHubAgentTrace {
	trace.ToolCatalog = nil
	trace.ModuleKnowledge = nil
	if len(trace.ToolCalls) > 0 {
		trace.ToolCalls = append([]opsHubAgentToolCall(nil), trace.ToolCalls...)
	}
	if len(trace.Errors) > 0 {
		trace.Errors = append([]string(nil), trace.Errors...)
	}
	if len(trace.AnswerPolicy) > 0 {
		trace.AnswerPolicy = append([]string(nil), trace.AnswerPolicy...)
	}
	if len(trace.ThinkingSteps) > 0 {
		trace.ThinkingSteps = append([]string(nil), trace.ThinkingSteps...)
	}
	if len(trace.ToolResults) > 0 {
		trace.ToolResults = append([]opsHubAgentToolResult(nil), trace.ToolResults...)
	}
	for i := range trace.ToolResults {
		trace.ToolResults[i].Data = compactOpsHubAgentToolDataForModel(trace.ToolResults[i].Name, trace.ToolResults[i].Data)
	}
	if len(trace.Turns) > 0 {
		trace.Turns = append([]opsHubAgentTurn(nil), trace.Turns...)
	}
	for i := range trace.Turns {
		if trace.Turns[i].Observation != nil {
			observation := *trace.Turns[i].Observation
			observation.Data = compactOpsHubAgentToolDataForModel(observation.Name, observation.Data)
			trace.Turns[i].Observation = &observation
		}
	}
	return trace
}

func compactOpsHubAgentToolDataForModel(toolName string, data any) any {
	if toolName == opsHubAgentToolHostTopProcesses {
		return compactHostTopProcessDataForModel(data)
	}
	if toolName == opsHubAgentToolHostSoftwareProbe {
		return compactHostSoftwareProbeDataForModel(data)
	}
	if toolName == opsHubAgentToolDataCatalog {
		return compactOpsHubAgentDataCatalogForModel(data)
	}
	if toolName == opsHubAgentToolDataQuery {
		return compactOpsHubAgentDataQueryForModel(data)
	}
	return data
}

func opsHubAgentToolNames(calls []opsHubAgentToolCall) string {
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name)
	}
	sort.Strings(names)
	return strings.Join(names, "、")
}

func fallbackAgentChat(question string, trace opsHubAgentTrace, cause error) *ChatResult {
	summaries := make([]string, 0, len(trace.ToolResults))
	for _, result := range trace.ToolResults {
		if result.Success {
			summaries = append(summaries, "- "+result.Name+"：已获取")
		} else {
			summaries = append(summaries, "- "+result.Name+"：失败，"+result.Error)
		}
	}
	answer := fmt.Sprintf(`当前外部 AI 模型调用失败，但 OpsHub Agent 已完成只读工具查询。

问题摘要：%s

工具执行结果：
%s

本地建议：
1. 如果你问的是平台功能或流程，可先查看对应模块菜单和模块说明。
2. 如果你问的是实时资源，请优先确认对应工具是否成功返回；失败项通常与数据库、集群连接、监控数据源或权限有关。
3. 需要更完整的自然语言分析，请检查“智能运维 -> AI 配置”的模型可用性。

模型调用失败原因：%s`, truncateText(question, 300), strings.Join(summaries, "\n"), cause.Error())
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}
