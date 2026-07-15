package aiops

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	opsHubAgentDataQueryDefaultLimit     = 20
	opsHubAgentDataQueryMaxLimit         = 80
	opsHubAgentDataQueryDefaultResources = 8
	opsHubAgentDataQueryMaxResources     = 12
)

type opsHubAgentDataResource struct {
	Key           string
	Module        string
	Title         string
	Description   string
	Table         string
	Aliases       []string
	Columns       []string
	SearchColumns []string
	OrderBy       string
}

type opsHubAgentDataQueryResourceResult struct {
	Key        string           `json:"key"`
	Module     string           `json:"module"`
	Title      string           `json:"title"`
	Table      string           `json:"table"`
	TotalRows  int64            `json:"totalRows"`
	Returned   int              `json:"returned"`
	Columns    []string         `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	Truncated  bool             `json:"truncated"`
	Error      string           `json:"error,omitempty"`
	QueryHint  string           `json:"queryHint,omitempty"`
	SafePolicy string           `json:"safePolicy,omitempty"`
}

func requiredOpsHubAgentDataTool(question string, trace opsHubAgentTrace) string {
	if !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolPlatformQuery) {
		return ""
	}
	if isOpsHubDataCatalogQuestion(question) && !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolDataCatalog) {
		return opsHubAgentToolDataCatalog
	}
	if shouldUseOpsHubAgentDataQuery(question, trace) && !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolDataQuery) {
		return opsHubAgentToolDataQuery
	}
	return ""
}

func enrichOpsHubAgentDataToolParams(question, tool string, params map[string]any) {
	if params == nil || tool != opsHubAgentToolDataQuery {
		return
	}
	if len(stringSliceParam(params, "resources", "resource", "tables", "table")) == 0 {
		keys := inferOpsHubAgentDataResourceKeys(question, params)
		if len(keys) > 0 {
			params["resources"] = keys
		}
	}
	if _, ok := params["limit"]; !ok {
		params["limit"] = opsHubAgentDataQueryDefaultLimit
	}
}

func isOpsHubDataCatalogQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return false
	}
	return containsAnyText(text, []string{
		"能查什么", "可以查什么", "查询范围", "数据目录", "资源目录", "接口目录", "所有接口", "所有数据",
		"哪些接口", "哪些数据", "所有信息", "全平台数据", "all api", "all interface", "catalog",
	})
}

func shouldUseOpsHubAgentDataQuery(question string, trace opsHubAgentTrace) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" || !isOpsHubPlatformQuestion(text) {
		return false
	}
	if isHostSoftwareProbeQuestion(text) || isHostTopProcessQuestion(text) {
		return false
	}
	if isOpsHubDataCatalogQuestion(text) {
		return false
	}
	dataIntent := containsAnyText(text, []string{
		"当前", "现在", "有哪些", "多少", "几个", "列表", "明细", "详情", "状态", "最近", "记录", "统计", "配置",
		"账号", "证书", "域名", "任务", "日志", "审计", "应用", "权限", "用户", "角色", "菜单", "主机", "集群",
		"告警", "规则", "数据源", "拨测", "nginx", "oauth", "ldap", "mfa", "sso", "agent",
	})
	if dataIntent {
		return true
	}
	return len(inferOpsHubAgentDataResourceKeys(question, nil)) > 0 && !isOpsHubCapabilityQuestion(text)
}

func collectOpsHubAgentDataCatalog(question string) map[string]any {
	resources := opsHubAgentDataCatalog()
	modules := map[string]int{}
	items := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		modules[resource.Module]++
		items = append(items, resource.catalogMap())
	}
	return map[string]any{
		"dataCatalogSummary": map[string]any{
			"resourceCount": len(resources),
			"moduleCount":   modules,
			"mode":          "read_only_whitelist",
			"question":      truncateText(question, 200),
		},
		"resources": items,
		"safePolicy": []string{
			"只允许查询白名单资源和安全字段，不允许模型提交原始 SQL。",
			"密码、私钥、Token、AccessKey、Secret、证书私钥、授权码、刷新令牌等敏感字段不会返回。",
			"仅执行 SELECT/COUNT 类只读查询，不执行新增、修改、删除、导入、安装、重启、部署等变更操作。",
		},
	}
}

func (r opsHubAgentDataResource) catalogMap() map[string]any {
	return map[string]any{
		"key":           r.Key,
		"module":        r.Module,
		"title":         r.Title,
		"description":   r.Description,
		"table":         r.Table,
		"aliases":       r.Aliases,
		"columns":       r.Columns,
		"searchColumns": r.SearchColumns,
	}
}

func (s *Service) collectOpsHubAgentDataQuery(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	_ = userID
	limit := boundedIntParam(params, opsHubAgentDataQueryDefaultLimit, 1, opsHubAgentDataQueryMaxLimit, "limit", "pageSize")
	keyword := strings.TrimSpace(firstStringParam(params, "keyword", "q", "search"))
	resourceKeys := inferOpsHubAgentDataResourceKeys(question, params)
	resources := resourcesByKeys(resourceKeys)
	warnings := make([]string, 0)

	if len(resources) == 0 {
		return map[string]any{
			"dataQuerySummary": map[string]any{
				"matchedResources": 0,
				"queriedResources": 0,
				"keyword":          keyword,
				"mode":             "read_only_whitelist",
			},
			"resources":      []opsHubAgentDataQueryResourceResult{},
			"catalogPreview": compactOpsHubAgentDataCatalogForModel(collectOpsHubAgentDataCatalog(question)),
		}, []string{"未能从问题中匹配到可查询的 OpsHub 只读资源，可换成更明确的对象名称，例如主机、证书、告警、用户、OAuth 应用、任务记录等"}
	}

	results := make([]opsHubAgentDataQueryResourceResult, 0, len(resources))
	totalRows := int64(0)
	returnedRows := 0
	for _, resource := range resources {
		item := s.queryOpsHubAgentDataResource(ctx, resource, keyword, limit)
		if item.Error != "" && len(warnings) < 12 {
			warnings = append(warnings, fmt.Sprintf("%s 查询失败: %s", resource.Title, item.Error))
		}
		totalRows += item.TotalRows
		returnedRows += item.Returned
		results = append(results, item)
	}

	return map[string]any{
		"dataQuerySummary": map[string]any{
			"matchedResources": len(resources),
			"queriedResources": countSuccessfulDataQueryResources(results),
			"totalRows":        totalRows,
			"returnedRows":     returnedRows,
			"keyword":          keyword,
			"limitPerResource": limit,
			"mode":             "read_only_whitelist",
			"truncated":        returnedRows >= len(resources)*limit,
		},
		"resources": results,
		"safePolicy": []string{
			"本次查询只读取白名单资源的安全字段。",
			"敏感字段和写操作接口未开放给智能体。",
		},
	}, uniqueStrings(warnings)
}

func (s *Service) queryOpsHubAgentDataResource(ctx context.Context, resource opsHubAgentDataResource, keyword string, limit int) opsHubAgentDataQueryResourceResult {
	result := opsHubAgentDataQueryResourceResult{
		Key:        resource.Key,
		Module:     resource.Module,
		Title:      resource.Title,
		Table:      resource.Table,
		QueryHint:  resource.Description,
		SafePolicy: "read_only_selected_columns",
	}
	available, err := s.safeColumnsForTable(resource.Table)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	columns := intersectColumns(resource.Columns, available)
	if len(columns) == 0 {
		result.Error = "白名单字段在当前数据库表中不存在"
		return result
	}
	result.Columns = columns
	db := s.db.WithContext(ctx).Table(resource.Table)
	if available["deleted_at"] {
		db = db.Where("deleted_at IS NULL")
	}
	searchColumns := intersectColumns(resource.SearchColumns, available)
	if keyword != "" && len(searchColumns) > 0 {
		whereParts := make([]string, 0, len(searchColumns))
		args := make([]any, 0, len(searchColumns))
		for _, column := range searchColumns {
			whereParts = append(whereParts, fmt.Sprintf("`%s` LIKE ?", column))
			args = append(args, "%"+keyword+"%")
		}
		db = db.Where(strings.Join(whereParts, " OR "), args...)
	}
	if err := db.Count(&result.TotalRows).Error; err != nil {
		result.Error = err.Error()
		return result
	}
	if result.TotalRows == 0 {
		result.Rows = []map[string]any{}
		return result
	}
	orderBy := safeOrderBy(resource.OrderBy, available)
	if orderBy == "" {
		if available["updated_at"] {
			orderBy = "updated_at DESC"
		} else if available["created_at"] {
			orderBy = "created_at DESC"
		} else if available["id"] {
			orderBy = "id DESC"
		}
	}
	query := db.Select(columns)
	if orderBy != "" {
		query = query.Order(orderBy)
	}
	var rows []map[string]any
	if err := query.Limit(limit).Find(&rows).Error; err != nil {
		result.Error = err.Error()
		return result
	}
	result.Rows = normalizeDataQueryRows(rows)
	result.Returned = len(result.Rows)
	result.Truncated = result.TotalRows > int64(result.Returned)
	return result
}

func (s *Service) safeColumnsForTable(table string) (map[string]bool, error) {
	if !safeSQLIdentifier(table) {
		return nil, fmt.Errorf("非法表名")
	}
	types, err := s.db.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("表不存在或没有可读取字段")
	}
	result := make(map[string]bool, len(types))
	for _, column := range types {
		name := strings.ToLower(strings.TrimSpace(column.Name()))
		if safeSQLIdentifier(name) {
			result[name] = true
		}
	}
	return result, nil
}

func inferOpsHubAgentDataResourceKeys(question string, params map[string]any) []string {
	explicit := stringSliceParam(params, "resources", "resource", "tables", "table", "keys", "key")
	if len(explicit) > 0 {
		return normalizeOpsHubAgentDataResourceKeys(explicit, opsHubAgentDataQueryMaxResources)
	}
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return nil
	}
	type scored struct {
		key   string
		score int
	}
	var matches []scored
	for _, resource := range opsHubAgentDataCatalog() {
		score := scoreOpsHubAgentDataResource(text, resource)
		if score > 0 {
			matches = append(matches, scored{key: resource.Key, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].key < matches[j].key
		}
		return matches[i].score > matches[j].score
	})
	keys := make([]string, 0, minInt(len(matches), opsHubAgentDataQueryMaxResources))
	for _, match := range matches {
		if !containsString(keys, match.key) {
			keys = append(keys, match.key)
		}
		if len(keys) >= opsHubAgentDataQueryMaxResources {
			break
		}
	}
	if len(keys) > 0 {
		return keys
	}
	if containsAnyText(text, []string{"平台", "opshub", "所有", "全部", "总览", "概览", "整体", "当前系统"}) {
		return []string{"hosts", "k8s_clusters", "monitor_alert_events", "monitor_alert_rules", "monitor_datasources", "ssl_certificates", "sys_users", "job_tasks"}
	}
	return nil
}

func scoreOpsHubAgentDataResource(text string, resource opsHubAgentDataResource) int {
	score := 0
	candidates := append([]string{resource.Key, resource.Table, resource.Title, resource.Module}, resource.Aliases...)
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		switch {
		case strings.Contains(text, candidate):
			score += 12 + len([]rune(candidate))/2
		case strings.Contains(candidate, text) && len([]rune(text)) >= 3:
			score += 4
		}
	}
	if resource.Module != "" && strings.Contains(text, strings.ToLower(resource.Module)) {
		score += 4
	}
	if score > 0 && containsAnyText(text, []string{"最近", "记录", "日志", "历史"}) {
		if containsAnyText(strings.ToLower(resource.Title+" "+resource.Key), []string{"log", "日志", "history", "record", "记录", "event", "事件"}) {
			score += 5
		}
	}
	return score
}

func normalizeOpsHubAgentDataResourceKeys(values []string, limit int) []string {
	byAlias := dataResourceAliasIndex()
	keys := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.Trim(value, "`'\" ")
		if key := byAlias[value]; key != "" && !containsString(keys, key) {
			keys = append(keys, key)
		}
		if len(keys) >= limit {
			break
		}
	}
	return keys
}

func resourcesByKeys(keys []string) []opsHubAgentDataResource {
	index := dataResourceIndex()
	resources := make([]opsHubAgentDataResource, 0, len(keys))
	for _, key := range keys {
		if resource, ok := index[key]; ok {
			resources = append(resources, resource)
		}
	}
	if len(resources) > opsHubAgentDataQueryMaxResources {
		resources = resources[:opsHubAgentDataQueryMaxResources]
	}
	return resources
}

func dataResourceIndex() map[string]opsHubAgentDataResource {
	resources := opsHubAgentDataCatalog()
	index := make(map[string]opsHubAgentDataResource, len(resources))
	for _, resource := range resources {
		index[resource.Key] = resource
	}
	return index
}

func dataResourceAliasIndex() map[string]string {
	index := make(map[string]string)
	for _, resource := range opsHubAgentDataCatalog() {
		values := append([]string{resource.Key, resource.Table, resource.Title}, resource.Aliases...)
		for _, value := range values {
			value = strings.ToLower(strings.TrimSpace(value))
			if value != "" {
				index[value] = resource.Key
			}
		}
	}
	return index
}

func safeSQLIdentifier(value string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(strings.TrimSpace(value))
}

func intersectColumns(columns []string, available map[string]bool) []string {
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		column = strings.ToLower(strings.TrimSpace(column))
		if safeSQLIdentifier(column) && available[column] && !containsString(result, column) {
			result = append(result, column)
		}
	}
	return result
}

func safeOrderBy(orderBy string, available map[string]bool) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return ""
	}
	fields := strings.Fields(orderBy)
	if len(fields) == 0 || len(fields) > 2 {
		return ""
	}
	column := strings.ToLower(fields[0])
	if !safeSQLIdentifier(column) || !available[column] {
		return ""
	}
	direction := "DESC"
	if len(fields) == 2 {
		direction = strings.ToUpper(fields[1])
	}
	if direction != "ASC" && direction != "DESC" {
		return ""
	}
	return column + " " + direction
}

func normalizeDataQueryRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		normalized := make(map[string]any, len(row))
		for key, value := range row {
			normalized[key] = normalizeDataQueryValue(value)
		}
		result = append(result, normalized)
	}
	return result
}

func normalizeDataQueryValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return truncateText(string(v), 1000)
	case string:
		return truncateText(v, 1000)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case *time.Time:
		if v == nil {
			return nil
		}
		return v.Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

func countSuccessfulDataQueryResources(items []opsHubAgentDataQueryResourceResult) int {
	total := 0
	for _, item := range items {
		if item.Error == "" {
			total++
		}
	}
	return total
}

func compactOpsHubAgentDataCatalogForModel(data any) any {
	value, ok := data.(map[string]any)
	if !ok {
		return data
	}
	resources, _ := value["resources"].([]map[string]any)
	if len(resources) > 40 {
		resources = resources[:40]
	}
	return map[string]any{
		"dataCatalogSummary": value["dataCatalogSummary"],
		"resources":          resources,
		"safePolicy":         value["safePolicy"],
	}
}

func compactOpsHubAgentDataQueryForModel(data any) any {
	value, ok := data.(map[string]any)
	if !ok {
		return data
	}
	resources, _ := value["resources"].([]opsHubAgentDataQueryResourceResult)
	compactResources := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		rows := resource.Rows
		if len(rows) > 12 {
			rows = rows[:12]
		}
		compactResources = append(compactResources, map[string]any{
			"key":        resource.Key,
			"module":     resource.Module,
			"title":      resource.Title,
			"table":      resource.Table,
			"totalRows":  resource.TotalRows,
			"returned":   resource.Returned,
			"columns":    resource.Columns,
			"rows":       rows,
			"truncated":  resource.Truncated,
			"error":      resource.Error,
			"safePolicy": resource.SafePolicy,
		})
	}
	return map[string]any{
		"dataQuerySummary": value["dataQuerySummary"],
		"resources":        compactResources,
		"safePolicy":       value["safePolicy"],
		"answerHint":       "这些数据来自 OpsHub 后端只读白名单查询；如果 resources 中有 rows，应直接基于 rows 回答，不要说没有平台数据。",
	}
}

func opsHubAgentDataCatalog() []opsHubAgentDataResource {
	return []opsHubAgentDataResource{
		dataResource("hosts", "资产管理", "主机", "OpsHub 主机资产、Agent 状态和资源使用", "hosts", []string{"主机", "服务器", "机器", "资产", "云主机", "agent", "host", "server"}, []string{"id", "name", "hostname", "ip", "group_id", "type", "cloud_provider", "status", "agent_status", "cpu_cores", "cpu_usage", "memory_total", "memory_usage", "disk_total", "disk_usage", "os", "arch", "last_seen", "agent_last_seen", "created_at", "updated_at"}, []string{"name", "hostname", "ip", "os", "cloud_provider"}, "id DESC"),
		dataResource("asset_groups", "资产管理", "资产分组", "主机和资产分组树", "asset_group", []string{"资产分组", "主机分组", "分组", "group"}, []string{"id", "name", "code", "parent_id", "description", "sort", "created_at", "updated_at"}, []string{"name", "code", "description"}, "sort ASC"),
		dataResource("credentials", "资产管理", "主机凭据", "主机 SSH 凭据元数据，不返回密码和私钥", "credentials", []string{"凭据", "ssh凭据", "密钥", "credential"}, []string{"id", "name", "type", "username", "description", "created_at", "updated_at"}, []string{"name", "type", "username", "description"}, "id DESC"),
		dataResource("cloud_accounts", "资产管理", "云账号", "公有云账号元数据，不返回 AccessKey/Secret", "cloud_accounts", []string{"云账号", "云账户", "阿里云账号", "腾讯云账号", "华为云账号", "cloud account"}, []string{"id", "name", "provider", "region", "status", "description", "created_at", "updated_at"}, []string{"name", "provider", "region", "description"}, "id DESC"),
		dataResource("terminal_sessions", "资产管理", "终端会话", "资产 SSH 终端审计会话", "ssh_terminal_sessions", []string{"终端", "终端审计", "ssh会话", "terminal"}, []string{"id", "host_id", "host_name", "username", "client_ip", "status", "started_at", "ended_at", "duration", "created_at"}, []string{"host_name", "username", "client_ip", "status"}, "id DESC"),

		dataResource("k8s_clusters", "容器管理", "Kubernetes 集群", "集群管理中的集群元数据和同步状态", "k8s_clusters", []string{"k8s集群", "kubernetes集群", "集群", "容器集群", "cluster"}, []string{"id", "name", "alias", "version", "status", "region", "provider", "node_count", "pod_count", "status_synced_at", "created_at", "updated_at"}, []string{"name", "alias", "version", "region", "provider"}, "id DESC"),
		dataResource("k8s_role_bindings", "容器管理", "K8s 用户授权", "Kubernetes 用户角色绑定", "k8s_user_role_bindings", []string{"k8s权限", "k8s授权", "访问控制", "rolebinding", "用户授权"}, []string{"id", "user_id", "cluster_id", "role_name", "namespace", "created_at", "updated_at"}, []string{"role_name", "namespace"}, "id DESC"),
		dataResource("k8s_inspections", "容器管理", "K8s 巡检", "Kubernetes 集群巡检记录", "k8s_cluster_inspections", []string{"k8s巡检", "集群巡检", "容器巡检", "inspection"}, []string{"id", "cluster_id", "cluster_name", "status", "score", "summary", "created_at", "updated_at"}, []string{"cluster_name", "status", "summary"}, "id DESC"),
		dataResource("k8s_terminal_sessions", "容器管理", "K8s 终端会话", "Kubernetes 终端审计会话", "k8s_terminal_sessions", []string{"k8s终端", "容器终端", "pod终端", "终端审计"}, []string{"id", "cluster_id", "namespace", "pod_name", "container_name", "username", "status", "started_at", "ended_at", "created_at"}, []string{"namespace", "pod_name", "container_name", "username", "status"}, "id DESC"),

		dataResource("monitor_datasources", "监控中心", "监控数据源", "Prometheus、Loki、VictoriaMetrics、Elasticsearch 等数据源配置，不返回密码 Token", "monitor_datasources", []string{"数据源", "prometheus", "victoriametrics", "loki", "elasticsearch", "monitor datasource"}, []string{"id", "name", "type", "url", "auth_type", "timeout", "skip_tls_verify", "enabled", "remote_write_enabled", "remote_write_url", "status", "last_test_at", "last_error", "description", "created_at", "updated_at"}, []string{"name", "type", "url", "status", "last_error", "description"}, "id DESC"),
		dataResource("monitor_alert_rules", "监控中心", "告警规则", "监控告警规则配置和最近评估状态", "monitor_alert_rules", []string{"告警规则", "报警规则", "alert rule", "规则"}, []string{"id", "name", "rule_group_id", "fault_center_id", "data_source_id", "data_source_type", "query", "query_mode", "condition", "threshold", "severity", "enabled", "last_state", "last_value", "last_eval_at", "last_error", "created_at", "updated_at"}, []string{"name", "data_source_type", "query", "severity", "last_state", "last_error"}, "id DESC"),
		dataResource("monitor_alert_events", "监控中心", "告警事件", "告警生命周期事件和当前状态", "monitor_alert_events", []string{"告警事件", "报警事件", "告警", "报警", "p0", "p1", "p2", "alert event"}, []string{"id", "rule_id", "rule_name", "data_source_id", "data_source_name", "data_source_type", "severity", "state", "value", "condition", "threshold", "message", "labels", "acknowledged", "notify_status", "started_at", "ended_at", "last_eval_at", "created_at", "updated_at"}, []string{"rule_name", "data_source_name", "data_source_type", "severity", "state", "message", "labels"}, "last_eval_at DESC"),
		dataResource("monitor_probe_tasks", "监控中心", "拨测任务", "HTTP/TCP/ICMP 等拨测任务配置和状态", "monitor_probe_tasks", []string{"拨测", "探测", "黑盒", "probe", "domain monitor"}, []string{"id", "name", "protocol", "endpoint", "method", "frequency_seconds", "timeout_seconds", "enabled", "status", "last_status", "last_probe_at", "last_duration_ms", "last_error", "description", "created_at", "updated_at"}, []string{"name", "protocol", "endpoint", "status", "last_status", "last_error", "description"}, "id DESC"),
		dataResource("monitor_probe_histories", "监控中心", "拨测历史", "拨测执行历史和错误信息", "monitor_probe_histories", []string{"拨测历史", "探测历史", "probe history"}, []string{"id", "probe_task_id", "protocol", "endpoint", "success", "status_code", "duration_ms", "ssl_expire_at", "ssl_days_left", "error", "message", "checked_at", "created_at"}, []string{"protocol", "endpoint", "error", "message"}, "checked_at DESC"),
		dataResource("monitor_fault_centers", "监控中心", "故障中心", "告警聚合、通知和升级策略", "monitor_fault_centers", []string{"故障中心", "告警中心", "fault center"}, []string{"id", "name", "description", "aggregation_type", "silence_enabled", "recover_notify", "upgrade_enabled", "created_at", "updated_at"}, []string{"name", "description", "aggregation_type"}, "id DESC"),
		dataResource("domain_monitors", "监控中心", "域名监控", "域名/站点监控任务", "domain_monitors", []string{"域名监控", "站点监控", "domain"}, []string{"id", "domain", "enable_ssl", "enable_alert", "status", "response_time", "ssl_valid", "ssl_expiry", "last_check", "created_at", "updated_at"}, []string{"domain", "status"}, "id DESC"),

		dataResource("ssl_certificates", "SSL证书", "SSL 证书", "证书管理中的证书、过期时间和自动续期状态，不返回私钥内容", "ssl_certificates", []string{"ssl证书", "证书", "cas证书", "tls证书", "certificate"}, []string{"id", "name", "domain", "domains", "provider", "cert_type", "status", "issuer", "not_before", "not_after", "auto_renew", "renew_days", "last_renew_at", "created_at", "updated_at"}, []string{"name", "domain", "domains", "provider", "cert_type", "status", "issuer"}, "id DESC"),
		dataResource("ssl_dns_providers", "SSL证书", "DNS 配置", "DNS 解析服务商配置元数据，不返回密钥", "ssl_dns_providers", []string{"dns配置", "dns账号", "dns provider", "解析配置"}, []string{"id", "name", "provider", "status", "created_at", "updated_at"}, []string{"name", "provider", "status"}, "id DESC"),
		dataResource("ssl_deploy_configs", "SSL证书", "证书部署配置", "证书自动部署目标配置，不返回敏感凭据", "ssl_deploy_configs", []string{"部署配置", "证书部署", "自动部署", "deploy config"}, []string{"id", "name", "cert_id", "deploy_type", "target_id", "status", "auto_deploy", "last_deploy_at", "created_at", "updated_at"}, []string{"name", "deploy_type", "status"}, "id DESC"),
		dataResource("ssl_renew_tasks", "SSL证书", "证书任务记录", "证书申请、续期、部署任务记录", "ssl_renew_tasks", []string{"任务记录", "续期任务", "证书任务", "renew task"}, []string{"id", "cert_id", "task_type", "status", "message", "started_at", "finished_at", "created_at", "updated_at"}, []string{"task_type", "status", "message"}, "id DESC"),

		dataResource("sys_users", "系统管理", "用户", "系统用户和登录状态，不返回密码", "sys_user", []string{"用户", "账号", "用户管理", "user"}, []string{"id", "username", "real_name", "email", "phone", "status", "source", "department_id", "last_login_at", "created_at", "updated_at"}, []string{"username", "real_name", "email", "phone", "source"}, "id DESC"),
		dataResource("sys_roles", "系统管理", "角色", "系统角色", "sys_role", []string{"角色", "角色管理", "role"}, []string{"id", "name", "code", "description", "status", "sort", "created_at", "updated_at"}, []string{"name", "code", "description"}, "id DESC"),
		dataResource("sys_departments", "系统管理", "部门", "组织部门", "sys_department", []string{"部门", "组织", "department"}, []string{"id", "name", "code", "parent_id", "dept_type", "sort", "status", "created_at", "updated_at"}, []string{"name", "code"}, "sort ASC"),
		dataResource("sys_menus", "系统管理", "菜单", "系统菜单和路由配置", "sys_menu", []string{"菜单", "路由", "页面", "menu"}, []string{"id", "name", "code", "type", "parent_id", "path", "component", "icon", "sort", "visible", "status", "created_at", "updated_at"}, []string{"name", "code", "path", "component"}, "sort ASC"),
		dataResource("sys_positions", "系统管理", "岗位", "系统岗位", "sys_position", []string{"岗位", "职位", "position"}, []string{"id", "post_code", "post_name", "post_status", "sort", "remark", "created_at", "updated_at"}, []string{"post_code", "post_name", "remark"}, "id DESC"),
		dataResource("asset_permissions", "系统管理", "资产权限", "角色资产权限规则", "sys_role_asset_permission", []string{"资产权限", "主机权限", "权限规则"}, []string{"id", "role_id", "asset_group_id", "permission_type", "created_at", "updated_at"}, []string{"permission_type"}, "id DESC"),

		dataResource("job_tasks", "任务中心", "脚本作业", "任务中心作业执行记录", "job_tasks", []string{"任务", "作业", "脚本作业", "执行记录", "job task"}, []string{"id", "name", "task_type", "status", "execute_time", "error_message", "created_at", "updated_at"}, []string{"name", "task_type", "status", "error_message"}, "id DESC"),
		dataResource("job_templates", "任务中心", "任务模板", "任务模板和脚本模板，不返回敏感参数", "job_templates", []string{"任务模板", "脚本模板", "template"}, []string{"id", "name", "task_type", "description", "created_at", "updated_at"}, []string{"name", "task_type", "description"}, "id DESC"),
		dataResource("ansible_tasks", "任务中心", "Ansible 任务", "Ansible 自动化任务", "ansible_tasks", []string{"ansible", "ansible任务"}, []string{"id", "name", "status", "last_run_time", "error_message", "created_at", "updated_at"}, []string{"name", "status", "error_message"}, "id DESC"),

		dataResource("operation_logs", "操作审计", "操作日志", "系统操作审计日志", "sys_operation_log", []string{"操作日志", "操作审计", "审计日志", "operation log"}, []string{"id", "user_id", "username", "module", "action", "description", "method", "path", "status", "error_msg", "cost_time", "ip", "created_at"}, []string{"username", "module", "action", "description", "method", "path", "error_msg", "ip"}, "id DESC"),
		dataResource("login_logs", "操作审计", "登录日志", "用户登录日志", "sys_login_log", []string{"登录日志", "登录记录", "login log"}, []string{"id", "user_id", "username", "ip", "user_agent", "status", "message", "created_at"}, []string{"username", "ip", "user_agent", "status", "message"}, "id DESC"),
		dataResource("data_logs", "操作审计", "数据日志", "数据变更审计日志", "sys_data_log", []string{"数据日志", "数据审计", "data log"}, []string{"id", "user_id", "username", "table_name", "record_id", "action", "old_data", "new_data", "created_at"}, []string{"username", "table_name", "record_id", "action"}, "id DESC"),

		dataResource("identity_sources", "统一认证", "认证源", "OAuth2/OIDC/LDAP 等认证源元数据，不返回密钥", "identity_sources", []string{"认证源", "身份源", "oauth源", "ldap源", "identity source"}, []string{"id", "name", "type", "enabled", "is_default", "description", "created_at", "updated_at"}, []string{"name", "type", "description"}, "id DESC"),
		dataResource("sso_applications", "统一认证", "SSO 应用", "统一认证应用门户中的应用", "sso_applications", []string{"sso应用", "oauth应用", "应用门户", "统一认证应用", "应用", "oauth app"}, []string{"id", "name", "code", "type", "category", "enabled", "homepage_url", "description", "created_at", "updated_at"}, []string{"name", "code", "type", "category", "homepage_url", "description"}, "id DESC"),
		dataResource("user_credentials", "统一认证", "用户第三方凭据", "第三方用户凭据元数据，不返回密钥", "user_credentials", []string{"用户凭据", "第三方凭据", "credential"}, []string{"id", "user_id", "source_id", "external_id", "username", "display_name", "email", "created_at", "updated_at"}, []string{"external_id", "username", "display_name", "email"}, "id DESC"),
		dataResource("app_permissions", "统一认证", "应用权限", "SSO 应用访问权限", "app_permissions", []string{"应用权限", "app permission"}, []string{"id", "app_id", "subject_type", "subject_id", "permission", "created_at", "updated_at"}, []string{"subject_type", "permission"}, "id DESC"),
		dataResource("auth_logs", "统一认证", "认证日志", "统一认证登录、授权、访问日志", "auth_logs", []string{"认证日志", "授权日志", "oauth日志", "auth log"}, []string{"id", "user_id", "username", "app_id", "app_name", "source_id", "source_name", "action", "status", "ip", "user_agent", "message", "created_at"}, []string{"username", "app_name", "source_name", "action", "status", "ip", "message"}, "id DESC"),
		dataResource("ldap_sync_jobs", "统一认证", "LDAP 同步任务", "LDAP 用户同步任务记录", "ldap_sync_jobs", []string{"ldap同步", "ldap任务", "同步任务"}, []string{"id", "source_id", "status", "total", "success", "failed", "message", "started_at", "finished_at", "created_at"}, []string{"status", "message"}, "id DESC"),

		dataResource("nginx_sources", "Nginx统计", "Nginx 日志源", "Nginx 日志采集源配置", "nginx_sources", []string{"nginx源", "nginx日志源", "nginx source"}, []string{"id", "name", "source_type", "host_id", "log_path", "enabled", "last_collect_at", "last_error", "created_at", "updated_at"}, []string{"name", "source_type", "log_path", "last_error"}, "id DESC"),
		dataResource("nginx_access_logs", "Nginx统计", "Nginx 访问日志", "Nginx 原始访问日志样本", "nginx_access_logs", []string{"nginx日志", "访问日志", "access log"}, []string{"id", "source_id", "remote_addr", "time_local", "method", "uri", "status", "body_bytes_sent", "request_time", "http_referer", "http_user_agent", "created_at"}, []string{"remote_addr", "method", "uri", "status", "http_referer", "http_user_agent"}, "id DESC"),
		dataResource("nginx_daily_stats", "Nginx统计", "Nginx 日统计", "Nginx 每日统计指标", "nginx_daily_stats", []string{"nginx日统计", "每日统计", "daily stats"}, []string{"id", "source_id", "date", "pv", "uv", "ip_count", "error_count", "avg_response_time", "created_at", "updated_at"}, []string{"date"}, "date DESC"),
		dataResource("nginx_hourly_stats", "Nginx统计", "Nginx 小时统计", "Nginx 小时级统计指标", "nginx_hourly_stats", []string{"nginx小时统计", "小时统计", "hourly stats"}, []string{"id", "source_id", "hour", "pv", "uv", "ip_count", "error_count", "avg_response_time", "created_at", "updated_at"}, []string{"hour"}, "hour DESC"),

		dataResource("ai_providers", "智能运维", "AI 模型配置", "AI Provider 模型配置，不返回 API Key", "ai_providers", []string{"ai配置", "模型配置", "provider", "模型"}, []string{"id", "name", "provider", "base_url", "model", "temperature", "max_tokens", "timeout", "enabled", "is_default", "last_test_at", "last_test_msg", "created_at", "updated_at"}, []string{"name", "provider", "base_url", "model", "last_test_msg"}, "id DESC"),
		dataResource("ai_sessions", "智能运维", "AI 会话", "AI 助手和分析会话", "ai_sessions", []string{"ai会话", "会话记录", "session"}, []string{"id", "user_id", "username", "title", "type", "status", "summary", "created_at", "updated_at"}, []string{"username", "title", "type", "status", "summary"}, "id DESC"),
		dataResource("ai_messages", "智能运维", "AI 消息", "AI 会话消息内容摘要", "ai_messages", []string{"ai消息", "聊天记录", "message"}, []string{"id", "session_id", "user_id", "role", "content", "model", "status", "tokens_in", "tokens_out", "latency_ms", "error", "created_at", "updated_at"}, []string{"role", "content", "model", "status", "error"}, "id DESC"),
		dataResource("ai_diagnosis_tasks", "智能运维", "AI 诊断任务", "智能诊断任务状态和结论", "ai_diagnosis_tasks", []string{"智能诊断", "诊断任务", "diagnosis"}, []string{"id", "user_id", "username", "session_id", "object_type", "cluster_id", "namespace", "object_name", "container", "status", "conclusion", "suggestion", "error", "created_at", "updated_at"}, []string{"username", "object_type", "namespace", "object_name", "container", "status", "conclusion", "suggestion", "error"}, "id DESC"),
		dataResource("ai_root_cause_analyses", "智能运维", "AI 根因分析", "告警根因分析记录", "ai_root_cause_analyses", []string{"根因分析", "告警分析", "root cause"}, []string{"id", "alert_event_id", "rule_id", "rule_name", "severity", "state", "summary", "root_cause", "suggestion", "model", "fallback", "status", "created_at", "updated_at"}, []string{"rule_name", "severity", "state", "summary", "root_cause", "suggestion", "status"}, "id DESC"),
	}
}

func dataResource(key, module, title, description, table string, aliases, columns, searchColumns []string, orderBy string) opsHubAgentDataResource {
	return opsHubAgentDataResource{
		Key:           key,
		Module:        module,
		Title:         title,
		Description:   description,
		Table:         table,
		Aliases:       aliases,
		Columns:       columns,
		SearchColumns: searchColumns,
		OrderBy:       orderBy,
	}
}
