package aiops

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateTextKeepsValidUTF8(t *testing.T) {
	input := strings.Repeat("处理建议：检查主机资源、Agent 状态和采集链路。", 120)
	output := truncateText(input, 200)
	if !utf8.ValidString(output) {
		t.Fatalf("truncateText returned invalid UTF-8: %q", output)
	}
	if !strings.Contains(output, "内容已截断") {
		t.Fatalf("expected truncate marker in output")
	}
}

func TestTruncateTextRemovesInvalidUTF8(t *testing.T) {
	input := string([]byte{0xe8, 0xbf, 0x0a}) + "处理建议"
	output := truncateText(input, 100)
	if !utf8.ValidString(output) {
		t.Fatalf("truncateText returned invalid UTF-8: %q", output)
	}
}

func TestCleanContinuationAnswerRemovesRuntimeNotice(t *testing.T) {
	input := "## 5.2 安装 kubelet\n继续写到这里\n\n> 本次流式生成中断，已保留目前收到的内容。"
	output := cleanContinuationAnswer(input)
	if strings.Contains(output, "本次流式生成中断") {
		t.Fatalf("expected runtime notice to be removed: %q", output)
	}
	if !strings.Contains(output, "5.2 安装 kubelet") {
		t.Fatalf("expected original content to remain: %q", output)
	}
}

func TestCleanContinuationAnswerRemovesSyntheticFence(t *testing.T) {
	input := "```bash\nkubeadm init --config kubeadm.yaml\n```\n\n> 本次流式生成中断，已保留目前收到的内容。"
	output := cleanContinuationAnswer(input)
	if strings.HasSuffix(output, "```") {
		t.Fatalf("expected synthetic fence closure to be removed: %q", output)
	}
	if !strings.Contains(output, "kubeadm init") {
		t.Fatalf("expected code content to remain: %q", output)
	}
}

func TestCleanContinuationAnswerKeepsRealFenceWithoutRuntimeNotice(t *testing.T) {
	input := "```bash\nkubectl get node\n```"
	output := cleanContinuationAnswer(input)
	if !strings.HasSuffix(output, "```") {
		t.Fatalf("expected real fence closure to remain: %q", output)
	}
}

func TestContinuationAnchorFindsLastNumberedSection(t *testing.T) {
	input := "# kubeadm 部署文档\n\n## 5.1 安装组件\n内容\n\n### 5.2 配置 kubelet\n这里还没写完"
	output := continuationAnchor(input)
	if !strings.Contains(output, "5.2") {
		t.Fatalf("expected last section anchor to contain 5.2: %q", output)
	}
}

func TestIsLengthFinishReason(t *testing.T) {
	if !isLengthFinishReason(" length ") {
		t.Fatalf("expected length finish reason to be detected")
	}
	if isLengthFinishReason("stop") {
		t.Fatalf("did not expect stop finish reason to be detected as length")
	}
}

func TestDetectAssistantIntentForOpsHubClusterData(t *testing.T) {
	intent := detectAssistantIntent("帮我看下有哪些 k8s 集群，他们每个集群有哪些节点，每个节点资源使用情况")
	if intent != "kubernetes" && intent != "platform" {
		t.Fatalf("expected OpsHub cluster data question to use platform context, got %q", intent)
	}
}

func TestDetectAssistantIntentKeepsGenericKubeadmQuestionGeneral(t *testing.T) {
	intent := detectAssistantIntent("给我一个使用 kubeadm 部署 Kubernetes 集群的详细文档")
	if intent != "general" {
		t.Fatalf("expected generic Kubernetes tutorial to remain general, got %q", intent)
	}
}

func TestHeuristicOpsHubAgentToolPlanSelectsKubernetesAndAlerts(t *testing.T) {
	calls := normalizeOpsHubAgentToolCalls("帮我看下有哪些 k8s 集群和 P0 告警", heuristicOpsHubAgentToolPlan("帮我看下有哪些 k8s 集群和 P0 告警"))
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
	}
	for _, name := range []string{opsHubAgentToolModuleKnowledge, opsHubAgentToolPlatformOverview, opsHubAgentToolPlatformQuery, opsHubAgentToolK8sSummary, opsHubAgentToolAlertsRecent} {
		if !names[name] {
			t.Fatalf("expected tool %s to be selected, got %#v", name, calls)
		}
	}
}

func TestOpsHubAgentToolPlanSelectsNamespaceTool(t *testing.T) {
	calls := normalizeOpsHubAgentToolCalls("帮我看下现在 test 的 k8s 集群中有多少个命名空间", heuristicOpsHubAgentToolPlan("帮我看下现在 test 的 k8s 集群中有多少个命名空间"))
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
	}
	if !names[opsHubAgentToolK8sNamespaces] {
		t.Fatalf("expected namespace tool to be selected, got %#v", calls)
	}
}

func TestInferOpsHubAgentQueryPlanRoutesMultipleDomains(t *testing.T) {
	plan := inferOpsHubAgentQueryPlan("帮我看下 OpsHub 平台有多少台主机、SSL证书任务和用户角色")
	domains := map[string]bool{}
	for _, domain := range plan.Domains {
		domains[domain] = true
	}
	for _, domain := range []string{"module", "overview", "hosts", "ssl", "system"} {
		if !domains[domain] {
			t.Fatalf("expected domain %s in plan, got %#v", domain, plan)
		}
	}
}

func TestParseOpsHubAgentPlannerToolCallsFromMarkdownJSON(t *testing.T) {
	content := "```json\n{\"toolCalls\":[{\"name\":\"opshub.hosts.summary\",\"reason\":\"查主机\"}]}\n```"
	calls := parseOpsHubAgentPlannerToolCalls(content)
	if len(calls) != 1 || calls[0].Name != opsHubAgentToolHostsSummary {
		t.Fatalf("expected host summary tool call, got %#v", calls)
	}
}

func TestParseOpsHubAgentNextActionFromJSON(t *testing.T) {
	content := "```json\n{\"thoughtSummary\":\"先查平台路由\",\"action\":\"tool\",\"tool\":\"opshub.platform.query\",\"reason\":\"需要平台数据\",\"params\":{\"domains\":[\"kubernetes\"],\"clusterName\":\"test\"}}\n```"
	action, thought, ok := parseOpsHubAgentNextAction(content)
	if !ok {
		t.Fatalf("expected action to parse")
	}
	if thought != "先查平台路由" {
		t.Fatalf("unexpected thought summary: %q", thought)
	}
	if action.Action != "tool" || action.Tool != opsHubAgentToolPlatformQuery {
		t.Fatalf("unexpected action: %#v", action)
	}
	if action.Params["clusterName"] != "test" {
		t.Fatalf("expected clusterName param, got %#v", action.Params)
	}
}

func TestNormalizeOpsHubAgentActionForcesInitialPlatformQuery(t *testing.T) {
	action := normalizeOpsHubAgentAction("帮我看下 test 的 k8s 集群有多少命名空间", opsHubAgentTrace{}, opsHubAgentAction{
		Action: "tool",
		Tool:   opsHubAgentToolK8sNamespaces,
		Params: map[string]any{},
	}, 1)
	if action.Tool != opsHubAgentToolPlatformQuery {
		t.Fatalf("expected first round to use platform query, got %#v", action)
	}
	if action.Params["clusterName"] != "test" {
		t.Fatalf("expected inferred clusterName, got %#v", action.Params)
	}
	resources, _ := action.Params["kubernetesResources"].([]string)
	if !containsString(resources, "namespaces") {
		t.Fatalf("expected namespaces resource hint, got %#v", action.Params)
	}
}

func TestHeuristicOpsHubAgentActionPlanKeepsRealtimeSSLFocused(t *testing.T) {
	calls := heuristicOpsHubAgentActionPlan("帮我看下 SSL 证书中本月会过期的域名")
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
	}
	if !names[opsHubAgentToolPlatformQuery] || !names[opsHubAgentToolSSLSummary] {
		t.Fatalf("expected platform query and SSL summary, got %#v", calls)
	}
	if names[opsHubAgentToolModuleKnowledge] || names[opsHubAgentToolPlatformOverview] {
		t.Fatalf("did not expect module/overview tools for realtime SSL query, got %#v", calls)
	}
}

func TestHostTopProcessQuestionSelectsLiveTool(t *testing.T) {
	question := "帮我找出每个主机中内存占用最高的进程"
	calls := heuristicOpsHubAgentActionPlan(question)
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
	}
	if !names[opsHubAgentToolPlatformQuery] || !names[opsHubAgentToolHostTopProcesses] {
		t.Fatalf("expected platform query and host top process live tool, got %#v", calls)
	}
	if !isHostTopProcessQuestion(question) {
		t.Fatalf("expected question to be detected as host top process query")
	}
	if isHostTopProcessQuestion("帮我看下 OpsHub 平台资源总览") {
		t.Fatalf("did not expect broad OpsHub resource overview to be treated as host process query")
	}
}

func TestHostSoftwareProbeQuestionSelectsLiveTool(t *testing.T) {
	question := "帮我看下哪台主机部署了 nginx"
	calls := heuristicOpsHubAgentActionPlan(question)
	names := map[string]bool{}
	var probeCall opsHubAgentToolCall
	for _, call := range calls {
		names[call.Name] = true
		if call.Name == opsHubAgentToolHostSoftwareProbe {
			probeCall = call
		}
	}
	if !names[opsHubAgentToolPlatformQuery] || !names[opsHubAgentToolHostSoftwareProbe] {
		t.Fatalf("expected platform query and host software probe live tool, got %#v", calls)
	}
	if !isHostSoftwareProbeQuestion(question) {
		t.Fatalf("expected question to be detected as host software probe query")
	}
	if inferHostSoftwareTarget(question) != "nginx" {
		t.Fatalf("expected nginx target, got %q", inferHostSoftwareTarget(question))
	}
	if probeCall.Params["target"] != "nginx" {
		t.Fatalf("expected target hint nginx, got %#v", probeCall.Params)
	}
}

func TestNormalizeOpsHubAgentActionForcesLiveToolBeforeFinal(t *testing.T) {
	trace := opsHubAgentTrace{
		ToolCalls: []opsHubAgentToolCall{{Name: opsHubAgentToolPlatformQuery}},
	}
	action := normalizeOpsHubAgentAction("找出每台服务器 CPU 占用最高的进程", trace, opsHubAgentAction{
		Action:            "final",
		AnswerInstruction: "基于已有上下文回答",
	}, 2)
	if action.Action != "tool" || action.Tool != opsHubAgentToolHostTopProcesses {
		t.Fatalf("expected live top process tool before final, got %#v", action)
	}
	if action.Params["metric"] != "cpu" {
		t.Fatalf("expected cpu metric hint, got %#v", action.Params)
	}
}

func TestNormalizeOpsHubAgentActionForcesSoftwareProbeBeforeFinal(t *testing.T) {
	trace := opsHubAgentTrace{
		ToolCalls: []opsHubAgentToolCall{{Name: opsHubAgentToolPlatformQuery}},
	}
	action := normalizeOpsHubAgentAction("帮我看下哪台主机部署了 nginx", trace, opsHubAgentAction{
		Action:            "final",
		AnswerInstruction: "基于已有上下文回答",
	}, 2)
	if action.Action != "tool" || action.Tool != opsHubAgentToolHostSoftwareProbe {
		t.Fatalf("expected live software probe tool before final, got %#v", action)
	}
	if action.Params["target"] != "nginx" {
		t.Fatalf("expected nginx target hint, got %#v", action.Params)
	}
}

func TestOpsHubAgentDataQueryCoversGenericPlatformData(t *testing.T) {
	question := "帮我看下现在有哪些 OAuth 应用"
	if !shouldUseOpsHubAgentDataQuery(question, opsHubAgentTrace{}) {
		t.Fatalf("expected generic OpsHub data question to use universal data query")
	}
	keys := inferOpsHubAgentDataResourceKeys(question, nil)
	if !containsString(keys, "sso_applications") {
		t.Fatalf("expected sso_applications resource, got %#v", keys)
	}
	trace := opsHubAgentTrace{ToolCalls: []opsHubAgentToolCall{{Name: opsHubAgentToolPlatformQuery}}}
	action := normalizeOpsHubAgentAction(question, trace, opsHubAgentAction{
		Action:            "final",
		AnswerInstruction: "基于已有上下文回答",
	}, 2)
	if action.Action != "tool" || action.Tool != opsHubAgentToolDataQuery {
		t.Fatalf("expected universal readonly data query before final, got %#v", action)
	}
	resources, _ := action.Params["resources"].([]string)
	if !containsString(resources, "sso_applications") {
		t.Fatalf("expected sso_applications resource hint, got %#v", action.Params)
	}
}

func TestOpsHubAgentDataCatalogQuestionUsesCatalogTool(t *testing.T) {
	question := "这个智能体能查询 OpsHub 的哪些接口和数据"
	if !isOpsHubDataCatalogQuestion(question) {
		t.Fatalf("expected catalog question to be detected")
	}
	trace := opsHubAgentTrace{ToolCalls: []opsHubAgentToolCall{{Name: opsHubAgentToolPlatformQuery}}}
	action := normalizeOpsHubAgentAction(question, trace, opsHubAgentAction{Action: "final"}, 2)
	if action.Action != "tool" || action.Tool != opsHubAgentToolDataCatalog {
		t.Fatalf("expected data catalog tool before final, got %#v", action)
	}
}

func TestOpsHubAgentDataCatalogHidesSensitiveFields(t *testing.T) {
	catalog := opsHubAgentDataCatalog()
	if len(catalog) < 20 {
		t.Fatalf("expected broad readonly resource catalog, got %d", len(catalog))
	}
	for _, resource := range catalog {
		for _, column := range resource.Columns {
			lower := strings.ToLower(column)
			if strings.Contains(lower, "password") || strings.Contains(lower, "private_key") || strings.Contains(lower, "access_key_secret") || lower == "token" || strings.HasSuffix(lower, "_token") || lower == "secret" {
				t.Fatalf("sensitive column %q leaked in resource %#v", column, resource.Key)
			}
		}
	}
}

func TestParseHostTopProcessOutput(t *testing.T) {
	output := `__OPSHUB_MEM__
123 1 root java 12.5 3.4 204800 /usr/bin/java -jar app.jar
456 1 mysql mysqld 6.2 1.1 102400 /usr/sbin/mysqld
__OPSHUB_CPU__
789 1 root nginx 80.1 0.5 51200 nginx: worker process
123 1 root java 30.4 12.5 204800 /usr/bin/java -jar app.jar`
	mem, cpu := parseHostTopProcessOutput(output, 2)
	if len(mem) != 2 || len(cpu) != 2 {
		t.Fatalf("expected two memory and cpu processes, got mem=%#v cpu=%#v", mem, cpu)
	}
	if mem[0].PID != 123 || mem[0].MemoryPercent != 12.5 || !strings.Contains(mem[0].Command, "app.jar") {
		t.Fatalf("unexpected memory process parse: %#v", mem[0])
	}
	if cpu[0].PID != 789 || cpu[0].CPUPercent < 80 || cpu[0].CPUPercent > 80.2 || !strings.Contains(cpu[0].Command, "worker process") {
		t.Fatalf("unexpected cpu process parse: %#v", cpu[0])
	}
}

func TestParseHostSoftwareProbeOutput(t *testing.T) {
	output := "__OPSHUB_SOFTWARE__\tprocess\t123 nginx nginx: master process\n" +
		"__OPSHUB_SOFTWARE__\tsystemd_unit\tnginx.service loaded active running A high performance web server\n" +
		"__OPSHUB_SOFTWARE__\tpackage\tnginx-1.24.0-1.el7.x86_64\n" +
		"__OPSHUB_SOFTWARE__\tconfig\t/etc/nginx/nginx.conf\n" +
		"__OPSHUB_SOFTWARE__\tcontainer\tabc123 web-nginx nginx:latest Up 3 days"
	evidence := parseHostSoftwareProbeOutput(output)
	if len(evidence) != 5 {
		t.Fatalf("expected five evidence items, got %#v", evidence)
	}
	running := 0
	for _, item := range evidence {
		if item.Running {
			running++
		}
	}
	if running < 3 {
		t.Fatalf("expected process/systemd/container evidence to be running, got %#v", evidence)
	}
}

func TestBuildHostSoftwareProbeCommandEscapesPrintfPlaceholders(t *testing.T) {
	command := buildHostSoftwareProbeCommand("nginx")
	if strings.Contains(command, "%!s(MISSING)") {
		t.Fatalf("expected command to escape remote printf placeholders, got %q", command)
	}
	if !strings.Contains(command, "T='nginx'") {
		t.Fatalf("expected command to include quoted nginx target, got %q", command)
	}
}

func TestCompactHostTopProcessDataMarksPartialSuccess(t *testing.T) {
	data := map[string]any{
		"topProcessSummary": map[string]any{
			"totalHosts":   int64(2),
			"queriedHosts": 1,
			"failedHosts":  1,
		},
		"hostTopProcesses": []opsHubAgentHostTopProcessItem{
			{
				HostID:         1,
				Name:           "host-a",
				IP:             "10.0.0.1",
				GroupName:      "生产环境",
				QuerySuccess:   true,
				StoredCPUUsage: 12.3,
				TopMemoryProcesses: []opsHubAgentHostProcessInfo{{
					PID:           100,
					Name:          "java",
					CPUPercent:    1.2,
					MemoryPercent: 54.3,
					RSS:           "16.9 GB",
					Command:       strings.Repeat("java ", 80),
				}},
			},
			{
				HostID:       2,
				Name:         "host-b",
				IP:           "10.0.0.2",
				QuerySuccess: false,
				Error:        "主机未绑定 SSH 凭证，无法现场查询进程",
			},
		},
	}
	compact, ok := compactHostTopProcessDataForModel(data).(map[string]any)
	if !ok {
		t.Fatalf("expected compact map")
	}
	if compact["resultState"] != "partial_success" {
		t.Fatalf("expected partial_success, got %#v", compact["resultState"])
	}
	items, ok := compact["hostTopProcesses"].([]map[string]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected compact host items, got %#v", compact["hostTopProcesses"])
	}
	process, ok := items[0]["topMemoryProcess"].(map[string]any)
	if !ok || len(process["command"].(string)) > 220 {
		t.Fatalf("expected compact process command, got %#v", process)
	}
	if items[1]["error"] == "" {
		t.Fatalf("expected failed host error to be preserved")
	}
}

func TestCompactHostSoftwareProbeDataKeepsEvidence(t *testing.T) {
	data := map[string]any{
		"softwareProbeSummary": map[string]any{
			"target":        "nginx",
			"totalHosts":    int64(2),
			"queriedHosts":  2,
			"failedHosts":   0,
			"deployedHosts": 1,
			"runningHosts":  1,
		},
		"softwareProbe": []opsHubAgentHostSoftwareProbeItem{
			{
				HostID:       1,
				Name:         "host-a",
				IP:           "10.0.0.1",
				GroupName:    "生产环境",
				Target:       "nginx",
				QuerySuccess: true,
				Deployed:     true,
				Running:      true,
				Evidence: []opsHubAgentHostSoftwareEvidence{{
					Type:    "process",
					Detail:  strings.Repeat("nginx ", 80),
					Running: true,
				}},
			},
			{
				HostID:       2,
				Name:         "host-b",
				IP:           "10.0.0.2",
				Target:       "nginx",
				QuerySuccess: true,
			},
		},
	}
	compact, ok := compactHostSoftwareProbeDataForModel(data).(map[string]any)
	if !ok {
		t.Fatalf("expected compact map")
	}
	if compact["resultState"] != "success_with_deployments" {
		t.Fatalf("expected success_with_deployments, got %#v", compact["resultState"])
	}
	items, ok := compact["softwareProbe"].([]map[string]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected compact software probe items, got %#v", compact["softwareProbe"])
	}
	if items[0]["deployed"] != true || items[0]["running"] != true {
		t.Fatalf("expected deployed and running flags, got %#v", items[0])
	}
	evidence, ok := items[0]["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || len(evidence[0]["detail"].(string)) > 220 {
		t.Fatalf("expected compact evidence, got %#v", items[0]["evidence"])
	}
}

func TestHeuristicOpsHubAgentActionPlanSelectsPodTool(t *testing.T) {
	calls := heuristicOpsHubAgentActionPlan("帮我看看 test 集群 platform-test 命名空间的 pod 是否都正常运行")
	names := map[string]bool{}
	var podCall opsHubAgentToolCall
	for _, call := range calls {
		names[call.Name] = true
		if call.Name == opsHubAgentToolK8sPods {
			podCall = call
		}
	}
	if !names[opsHubAgentToolPlatformQuery] || !names[opsHubAgentToolK8sPods] {
		t.Fatalf("expected platform query and pod tool, got %#v", calls)
	}
	if podCall.Params["namespace"] != "platform-test" {
		t.Fatalf("expected namespace hint platform-test, got %#v", podCall.Params)
	}
}
