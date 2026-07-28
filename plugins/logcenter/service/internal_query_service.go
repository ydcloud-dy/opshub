package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

const (
	defaultInternalLimit = 200
	maxInternalLimit     = 2000
)

type InternalQueryService struct {
	clickhouse *ClickHouseService
}

func NewInternalQueryService() *InternalQueryService {
	return &InternalQueryService{clickhouse: NewClickHouseService()}
}

func ValidateInternalQueryRequest(req InternalQueryRequest) error {
	_, err := buildInternalWhere(req)
	return err
}

type InternalQueryScope struct {
	AssetTypes   []string `json:"assetTypes"`
	AssetIDs     []uint64 `json:"assetIds"`
	HostIDs      []uint64 `json:"hostIds"`
	ClusterIDs   []uint64 `json:"clusterIds"`
	Namespaces   []string `json:"namespaces"`
	Services     []string `json:"services"`
	Workloads    []string `json:"workloads"`
	Pods         []string `json:"pods"`
	Containers   []string `json:"containers"`
	Nodes        []string `json:"nodes"`
	Levels       []string `json:"levels"`
	Environments []string `json:"environments"`
}

type InternalQueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type InternalQueryRequest struct {
	StorageID               uint                  `json:"storageId"`
	Start                   string                `json:"start"`
	End                     string                `json:"end"`
	Query                   string                `json:"query"`
	Scope                   InternalQueryScope    `json:"scope"`
	Filters                 []InternalQueryFilter `json:"filters"`
	FilterLogic             string                `json:"filterLogic"`
	Sort                    string                `json:"sort"`
	Limit                   int                   `json:"limit"`
	Cursor                  string                `json:"cursor"`
	SkipHistory             bool                  `json:"skipHistory"`
	AllowedPolicyIDs        []uint64              `json:"-"`
	AllowedHostIDs          []uint64              `json:"-"`
	AllowedKubernetesScopes map[uint64][]string   `json:"-"`
	DeniedFields            []string              `json:"-"`
	MaskFields              []string              `json:"-"`
	DenyAll                 bool                  `json:"-"`
	RequiredFilters         []InternalQueryFilter `json:"-"`
}

type InternalContextRequest struct {
	StorageID               uint                   `json:"storageId"`
	Timestamp               string                 `json:"timestamp"`
	Message                 string                 `json:"message"`
	Level                   string                 `json:"level"`
	Fingerprint             uint64                 `json:"fingerprint"`
	Sequence                uint64                 `json:"sequence"`
	Labels                  map[string]string      `json:"labels"`
	Fields                  map[string]interface{} `json:"fields"`
	BeforeSeconds           int                    `json:"beforeSeconds"`
	AfterSeconds            int                    `json:"afterSeconds"`
	Limit                   int                    `json:"limit"`
	AllowedPolicyIDs        []uint64               `json:"-"`
	AllowedHostIDs          []uint64               `json:"-"`
	AllowedKubernetesScopes map[uint64][]string    `json:"-"`
	DeniedFields            []string               `json:"-"`
	MaskFields              []string               `json:"-"`
}

type internalCursor struct {
	Timestamp      string `json:"timestamp"`
	TimestampNanos int64  `json:"timestampNanos,omitempty"`
	Fingerprint    uint64 `json:"fingerprint"`
	Sequence       uint64 `json:"sequence"`
}

type internalSQL struct {
	Where  string
	Params map[string]string
	Start  time.Time
	End    time.Time
}

type InternalResourceOptions struct {
	HostIDs             []string                           `json:"hostIds"`
	ClusterIDs          []string                           `json:"clusterIds"`
	Environments        []string                           `json:"environments"`
	Services            []string                           `json:"services"`
	Namespaces          []string                           `json:"namespaces"`
	Workloads           []string                           `json:"workloads"`
	Pods                []string                           `json:"pods"`
	Containers          []string                           `json:"containers"`
	Nodes               []string                           `json:"nodes"`
	KubernetesResources []InternalKubernetesResourceOption `json:"kubernetesResources"`
}

type InternalKubernetesResourceOption struct {
	ClusterID     string `json:"clusterId"`
	Namespace     string `json:"namespace"`
	WorkloadKind  string `json:"workloadKind"`
	WorkloadName  string `json:"workloadName"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	NodeName      string `json:"nodeName"`
}

type InternalAlertSample struct {
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
	Logs   []LogItem         `json:"logs"`
}

var internalAlertGroupColumns = map[string]string{
	"assetType": "asset_type", "assetId": "asset_id", "hostId": "host_id", "clusterId": "cluster_id",
	"environment": "environment", "service": "service", "level": "level", "namespace": "namespace",
	"workloadKind": "workload_kind", "workloadName": "workload_name", "podName": "pod_name",
	"containerName": "container_name", "nodeName": "node_name",
}

func (s *InternalQueryService) Query(ctx context.Context, cluster logmodel.StorageCluster, password string, req InternalQueryRequest) (*LogQueryResponse, error) {
	started := time.Now()
	built, err := buildInternalWhere(req)
	if err != nil {
		return nil, err
	}
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return nil, err
	}
	limit := normalizeInternalLimit(req.Limit)
	sortDirection := "DESC"
	if strings.EqualFold(req.Sort, "asc") {
		sortDirection = "ASC"
	}
	query := fmt.Sprintf(`SELECT
    formatDateTime(timestamp, '%%Y-%%m-%%dT%%H:%%i:%%S.%%fZ', 'UTC') AS timestampText,
	toString(toUnixTimestamp64Nano(timestamp)) AS timestampNanos,
    formatDateTime(observed_at, '%%Y-%%m-%%dT%%H:%%i:%%S.%%fZ', 'UTC') AS observedAt,
    formatDateTime(ingested_at, '%%Y-%%m-%%dT%%H:%%i:%%S.%%fZ', 'UTC') AS ingestedAt,
    source_type AS sourceType,
    asset_type AS assetType,
    toString(asset_id) AS assetId,
    toString(host_id) AS hostId,
    toString(cluster_id) AS clusterId,
    environment,
    service,
    level,
    namespace,
    workload_kind AS workloadKind,
    workload_name AS workloadName,
    pod_name AS podName,
    pod_uid AS podUid,
    container_name AS containerName,
    container_image AS containerImage,
    node_name AS nodeName,
    file_path AS filePath,
    stream,
    body,
    attributes,
    resource_attributes AS resourceAttributes,
    trace_id AS traceId,
    span_id AS spanId,
    agent_id AS agentId,
    toString(policy_id) AS policyId,
    toString(fingerprint) AS fingerprintText,
    toString(sequence) AS sequenceText
FROM %s.opshub_logs
WHERE %s
ORDER BY timestamp %s, fingerprint %s, sequence %s
LIMIT %d`, database, built.Where, sortDirection, sortDirection, sortDirection, limit+1)
	rows, err := s.clickhouse.QueryJSONEachRow(ctx, cluster, password, query, built.Params)
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items := make([]LogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, internalRowToLogItem(row))
	}
	items = applyLogFieldSecurity(items, req.DeniedFields, req.MaskFields)
	result := &LogQueryResponse{
		Items:      items,
		Total:      -1,
		DurationMS: time.Since(started).Milliseconds(),
		Fields:     summarizeFields(items),
		HasMore:    hasMore,
	}
	if len(rows) > 0 {
		last := rows[len(rows)-1]
		result.NextCursor = encodeInternalCursor(internalCursor{
			Timestamp:      asString(last["timestampText"]),
			TimestampNanos: parseInternalInt64(last["timestampNanos"]),
			Fingerprint:    parseInternalUint(last["fingerprintText"]),
			Sequence:       parseInternalUint(last["sequenceText"]),
		})
	}
	return result, nil
}

func (s *InternalQueryService) AlertSamples(ctx context.Context, cluster logmodel.StorageCluster, password string, req InternalQueryRequest, groupBy []string, sampleLimit int) ([]InternalAlertSample, error) {
	built, err := buildInternalWhere(req)
	if err != nil {
		return nil, err
	}
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return nil, err
	}
	groupFields := make([]string, 0, len(groupBy))
	groupColumns := make([]string, 0, len(groupBy))
	seen := make(map[string]struct{}, len(groupBy))
	for _, field := range groupBy {
		field = strings.TrimSpace(field)
		column, ok := internalAlertGroupColumns[field]
		if !ok {
			return nil, fmt.Errorf("日志告警不支持按字段 %s 分组", field)
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		groupFields = append(groupFields, field)
		groupColumns = append(groupColumns, fmt.Sprintf("toString(%s) AS %s", column, field))
	}
	selectParts := append([]string{}, groupColumns...)
	selectParts = append(selectParts, "toString(count()) AS value")
	query := fmt.Sprintf("SELECT %s FROM %s.opshub_logs WHERE %s", strings.Join(selectParts, ", "), database, built.Where)
	if len(groupColumns) > 0 {
		plainColumns := make([]string, 0, len(groupFields))
		for _, field := range groupFields {
			plainColumns = append(plainColumns, internalAlertGroupColumns[field])
		}
		query += " GROUP BY " + strings.Join(plainColumns, ", ")
	}
	query += " ORDER BY toUInt64(value) DESC LIMIT 20"
	rows, err := s.clickhouse.QueryJSONEachRow(ctx, cluster, password, query, built.Params)
	if err != nil {
		return nil, err
	}
	if sampleLimit <= 0 {
		sampleLimit = 5
	}
	if sampleLimit > 20 {
		sampleLimit = 20
	}
	result := make([]InternalAlertSample, 0, len(rows))
	for _, row := range rows {
		labels := make(map[string]string, len(groupFields)+3)
		sampleRequest := req
		sampleRequest.RequiredFilters = append([]InternalQueryFilter(nil), req.RequiredFilters...)
		for _, field := range groupFields {
			value := asString(row[field])
			if value == "" || value == "0" {
				continue
			}
			labels[field] = value
			sampleRequest.RequiredFilters = append(sampleRequest.RequiredFilters, InternalQueryFilter{Field: field, Operator: "eq", Value: value})
		}
		sampleRequest.Sort = "desc"
		sampleRequest.Limit = sampleLimit
		sampleRequest.Cursor = ""
		sampleRequest.SkipHistory = true
		logs, err := s.Query(ctx, cluster, password, sampleRequest)
		if err != nil {
			return nil, err
		}
		result = append(result, InternalAlertSample{Value: float64(parseInternalUint(row["value"])), Labels: labels, Logs: logs.Items})
	}
	return result, nil
}

func (s *InternalQueryService) Histogram(ctx context.Context, cluster logmodel.StorageCluster, password string, req InternalQueryRequest) ([]HistogramBar, error) {
	rawBuilt, err := buildInternalWhere(req)
	if err != nil {
		return nil, err
	}
	_, useMetrics, err := buildInternalMetricsWhere(req)
	if err != nil {
		return nil, err
	}
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return nil, err
	}
	step := chooseInternalBucket(rawBuilt.End.Sub(rawBuilt.Start))
	bars := emptyHistogram(rawBuilt.Start, rawBuilt.End, step)
	fullStart := rawBuilt.Start.Truncate(time.Minute)
	if !fullStart.Equal(rawBuilt.Start) {
		fullStart = fullStart.Add(time.Minute)
	}
	fullEnd := rawBuilt.End.Truncate(time.Minute)
	if !useMetrics || !fullStart.Before(fullEnd) {
		rows, queryErr := s.queryInternalHistogramRange(ctx, cluster, password, database, rawBuilt, step, false)
		if queryErr != nil {
			return nil, queryErr
		}
		mergeInternalHistogramRows(bars, rawBuilt.Start, step, rows)
		return bars, nil
	}

	if rawBuilt.Start.Before(fullStart) {
		edgeReq := req
		edgeReq.Start = rawBuilt.Start.Format(time.RFC3339Nano)
		edgeReq.End = fullStart.Format(time.RFC3339Nano)
		edgeBuilt, buildErr := buildInternalWhere(edgeReq)
		if buildErr != nil {
			return nil, buildErr
		}
		rows, queryErr := s.queryInternalHistogramRange(ctx, cluster, password, database, edgeBuilt, step, false)
		if queryErr != nil {
			return nil, queryErr
		}
		mergeInternalHistogramRows(bars, rawBuilt.Start, step, rows)
	}

	metricReq := req
	metricReq.Start = fullStart.Format(time.RFC3339Nano)
	metricReq.End = fullEnd.Format(time.RFC3339Nano)
	metricBuilt, _, buildErr := buildInternalMetricsWhere(metricReq)
	if buildErr != nil {
		return nil, buildErr
	}
	rows, queryErr := s.queryInternalHistogramRange(ctx, cluster, password, database, metricBuilt, step, true)
	if queryErr != nil {
		return nil, queryErr
	}
	mergeInternalHistogramRows(bars, rawBuilt.Start, step, rows)

	if fullEnd.Before(rawBuilt.End) {
		edgeReq := req
		edgeReq.Start = fullEnd.Format(time.RFC3339Nano)
		edgeReq.End = rawBuilt.End.Format(time.RFC3339Nano)
		edgeBuilt, edgeBuildErr := buildInternalWhere(edgeReq)
		if edgeBuildErr != nil {
			return nil, edgeBuildErr
		}
		edgeRows, edgeQueryErr := s.queryInternalHistogramRange(ctx, cluster, password, database, edgeBuilt, step, false)
		if edgeQueryErr != nil {
			return nil, edgeQueryErr
		}
		mergeInternalHistogramRows(bars, rawBuilt.Start, step, edgeRows)
	}
	return bars, nil
}

func (s *InternalQueryService) queryInternalHistogramRange(ctx context.Context, cluster logmodel.StorageCluster, password, database string, built internalSQL, step time.Duration, useMetrics bool) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`SELECT
    formatDateTime(toStartOfInterval(timestamp, INTERVAL %d SECOND), '%%Y-%%m-%%dT%%H:%%i:%%SZ', 'UTC') AS time,
    count() AS count
FROM %s.opshub_logs
WHERE %s
GROUP BY time
ORDER BY time ASC`, int(step.Seconds()), database, built.Where)
	if useMetrics {
		query = fmt.Sprintf(`SELECT
    formatDateTime(toStartOfInterval(minute, INTERVAL %d SECOND), '%%Y-%%m-%%dT%%H:%%i:%%SZ', 'UTC') AS time,
    sum(log_count) AS count
FROM %s.opshub_log_metrics_1m
WHERE %s
GROUP BY time
ORDER BY time ASC`, int(step.Seconds()), database, built.Where)
	}
	return s.clickhouse.QueryJSONEachRow(ctx, cluster, password, query, built.Params)
}

func mergeInternalHistogramRows(bars []HistogramBar, start time.Time, step time.Duration, rows []map[string]interface{}) {
	for _, row := range rows {
		addCountToHistogram(bars, start, step, parseFlexibleTime(asString(row["time"])), int(parseInternalUint(row["count"])))
	}
}

func (s *InternalQueryService) Context(ctx context.Context, cluster logmodel.StorageCluster, password string, req InternalContextRequest) (*LogQueryResponse, error) {
	started := time.Now()
	timestamp := parseFlexibleTime(req.Timestamp)
	if timestamp.IsZero() {
		return nil, fmt.Errorf("日志时间无效，无法查询上下文")
	}
	before := req.BeforeSeconds
	if before <= 0 {
		before = 300
	}
	after := req.AfterSeconds
	if after <= 0 {
		after = 300
	}
	filters := make([]InternalQueryFilter, 0, 6)
	for _, field := range []string{"assetType", "assetId", "hostId", "clusterId", "podName", "containerName", "filePath"} {
		value := firstInternalValue(req.Labels, req.Fields, field)
		if value != "" && value != "0" {
			filters = append(filters, InternalQueryFilter{Field: field, Operator: "eq", Value: value})
		}
	}
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 101
	}
	fingerprint := req.Fingerprint
	if fingerprint == 0 {
		fingerprint = parseInternalUint(req.Fields["fingerprint"])
	}
	sequence := req.Sequence
	if sequence == 0 {
		sequence = parseInternalUint(req.Fields["sequence"])
	}
	selectedCursor := encodeInternalCursor(internalCursor{
		Timestamp: timestamp.UTC().Format(time.RFC3339Nano), Fingerprint: fingerprint, Sequence: sequence,
	})
	beforeLimit := limit / 2
	afterLimit := limit - beforeLimit - 1
	base := InternalQueryRequest{
		StorageID: req.StorageID, Filters: filters, SkipHistory: true, AllowedPolicyIDs: req.AllowedPolicyIDs, AllowedHostIDs: req.AllowedHostIDs,
		AllowedKubernetesScopes: req.AllowedKubernetesScopes, DeniedFields: req.DeniedFields, MaskFields: req.MaskFields,
	}
	beforeReq := base
	beforeReq.Start = timestamp.Add(-time.Duration(before) * time.Second).Format(time.RFC3339Nano)
	beforeReq.End = timestamp.Format(time.RFC3339Nano)
	beforeReq.Sort = "desc"
	beforeReq.Limit = beforeLimit
	beforeReq.Cursor = selectedCursor
	afterReq := base
	afterReq.Start = timestamp.Format(time.RFC3339Nano)
	afterReq.End = timestamp.Add(time.Duration(after) * time.Second).Format(time.RFC3339Nano)
	afterReq.Sort = "asc"
	afterReq.Limit = afterLimit
	afterReq.Cursor = selectedCursor

	beforeResult := &LogQueryResponse{}
	var err error
	if beforeLimit > 0 {
		beforeResult, err = s.Query(ctx, cluster, password, beforeReq)
		if err != nil {
			return nil, err
		}
	}
	afterResult := &LogQueryResponse{}
	if afterLimit > 0 {
		afterResult, err = s.Query(ctx, cluster, password, afterReq)
		if err != nil {
			return nil, err
		}
	}
	for left, right := 0, len(beforeResult.Items)-1; left < right; left, right = left+1, right-1 {
		beforeResult.Items[left], beforeResult.Items[right] = beforeResult.Items[right], beforeResult.Items[left]
	}
	selectedFields := make(map[string]interface{}, len(req.Fields)+2)
	for key, value := range req.Fields {
		selectedFields[key] = value
	}
	selectedFields["fingerprint"] = strconv.FormatUint(fingerprint, 10)
	selectedFields["sequence"] = strconv.FormatUint(sequence, 10)
	selectedLabels := make(map[string]string, len(req.Labels))
	for key, value := range req.Labels {
		selectedLabels[key] = value
	}
	selected := LogItem{
		Timestamp: timestamp.UTC().Format(time.RFC3339Nano), Message: req.Message,
		Level: firstNonEmpty(strings.TrimSpace(req.Level), "UNKNOWN"), Labels: selectedLabels, Fields: selectedFields,
		ContextSelected: true,
	}
	items := make([]LogItem, 0, len(beforeResult.Items)+1+len(afterResult.Items))
	items = appendInternalContextNeighbors(items, beforeResult.Items, selected)
	items = append(items, selected)
	items = appendInternalContextNeighbors(items, afterResult.Items, selected)
	items = applyLogFieldSecurity(items, req.DeniedFields, req.MaskFields)
	return &LogQueryResponse{
		Items: items, Total: len(items), DurationMS: time.Since(started).Milliseconds(), Fields: summarizeFields(items),
	}, nil
}

func (s *InternalQueryService) ResourceOptions(ctx context.Context, cluster logmodel.StorageCluster, password string, req InternalQueryRequest) (InternalResourceOptions, error) {
	built, err := buildInternalWhere(req)
	if err != nil {
		return InternalResourceOptions{}, err
	}
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return InternalResourceOptions{}, err
	}
	query := fmt.Sprintf(`SELECT
	    groupUniqArrayIf(200)(toString(host_id), asset_type = 'host' AND host_id > 0) AS hostIds,
	    groupUniqArrayIf(200)(toString(cluster_id), cluster_id > 0) AS clusterIds,
    groupUniqArrayIf(200)(environment, environment != '') AS environments,
    groupUniqArrayIf(200)(service, service != '') AS services,
    groupUniqArrayIf(200)(namespace, namespace != '') AS namespaces,
    groupUniqArrayIf(200)(workload_name, workload_name != '') AS workloads,
    groupUniqArrayIf(200)(pod_name, pod_name != '') AS pods,
	    groupUniqArrayIf(200)(container_name, container_name != '') AS containers,
	    groupUniqArrayIf(200)(node_name, node_name != '') AS nodes,
	    groupUniqArrayIf(5000)(concat(toString(cluster_id), '|', namespace, '|', workload_kind, '|', workload_name, '|', pod_name, '|', container_name, '|', node_name), asset_type = 'kubernetes' AND cluster_id > 0 AND pod_name != '') AS kubernetesResources
	FROM %s.opshub_logs
WHERE %s`, database, built.Where)
	rows, err := s.clickhouse.QueryJSONEachRow(ctx, cluster, password, query, built.Params)
	if err != nil {
		return InternalResourceOptions{}, err
	}
	if len(rows) == 0 {
		return InternalResourceOptions{}, nil
	}
	row := rows[0]
	return InternalResourceOptions{
		HostIDs: internalStringSlice(row["hostIds"]), ClusterIDs: internalStringSlice(row["clusterIds"]),
		Environments: internalStringSlice(row["environments"]), Services: internalStringSlice(row["services"]),
		Namespaces: internalStringSlice(row["namespaces"]), Workloads: internalStringSlice(row["workloads"]),
		Pods: internalStringSlice(row["pods"]), Containers: internalStringSlice(row["containers"]),
		Nodes: internalStringSlice(row["nodes"]), KubernetesResources: parseInternalKubernetesResources(row["kubernetesResources"]),
	}, nil
}

func InternalFieldOptions() []FieldOption {
	return []FieldOption{
		{Name: "timestamp", Type: "datetime", DisplayName: "日志时间", IsTimeField: true},
		{Name: "body", Type: "text", DisplayName: "日志正文", IsMessageField: true},
		{Name: "level", Type: "keyword", DisplayName: "日志级别", IsLevelField: true},
		{Name: "sourceType", Type: "keyword", DisplayName: "来源类型"},
		{Name: "assetType", Type: "keyword", DisplayName: "资产类型"},
		{Name: "assetId", Type: "uint64", DisplayName: "资产"},
		{Name: "hostId", Type: "uint64", DisplayName: "主机"},
		{Name: "clusterId", Type: "uint64", DisplayName: "集群"},
		{Name: "environment", Type: "keyword", DisplayName: "环境"},
		{Name: "service", Type: "keyword", DisplayName: "服务"},
		{Name: "namespace", Type: "keyword", DisplayName: "命名空间"},
		{Name: "workloadKind", Type: "keyword", DisplayName: "工作负载类型"},
		{Name: "workloadName", Type: "keyword", DisplayName: "工作负载"},
		{Name: "podName", Type: "keyword", DisplayName: "Pod"},
		{Name: "podUid", Type: "keyword", DisplayName: "Pod UID"},
		{Name: "containerName", Type: "keyword", DisplayName: "容器"},
		{Name: "containerImage", Type: "keyword", DisplayName: "容器镜像"},
		{Name: "nodeName", Type: "keyword", DisplayName: "节点"},
		{Name: "filePath", Type: "keyword", DisplayName: "文件路径"},
		{Name: "stream", Type: "keyword", DisplayName: "输出流"},
		{Name: "traceId", Type: "keyword", DisplayName: "Trace ID"},
		{Name: "spanId", Type: "keyword", DisplayName: "Span ID"},
		{Name: "agentId", Type: "keyword", DisplayName: "Agent ID"},
		{Name: "policyId", Type: "uint64", DisplayName: "策略 ID"},
		{Name: "fingerprint", Type: "uint64", DisplayName: "日志指纹"},
		{Name: "sequence", Type: "uint64", DisplayName: "日志序列"},
	}
}

func buildInternalWhere(req InternalQueryRequest) (internalSQL, error) {
	start := parseFlexibleTime(req.Start)
	end := parseFlexibleTime(req.End)
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return internalSQL{}, fmt.Errorf("必须提供有效的日志查询开始和结束时间")
	}
	params := map[string]string{
		"start": start.UTC().Format(time.RFC3339Nano),
		"end":   end.UTC().Format(time.RFC3339Nano),
	}
	conditions := []string{
		"timestamp >= parseDateTime64BestEffort({start:String}, 9, 'UTC')",
		"timestamp < parseDateTime64BestEffort({end:String}, 9, 'UTC')",
	}
	if req.DenyAll {
		conditions = append(conditions, "0 = 1")
	}
	appendUintCondition := func(column string, values []uint64) {
		if len(values) == 0 {
			return
		}
		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, strconv.FormatUint(value, 10))
		}
		conditions = append(conditions, fmt.Sprintf("%s IN (%s)", column, strings.Join(parts, ",")))
	}
	paramIndex := 0
	appendStringCondition := func(column string, values []string) {
		cleaned := make([]string, 0, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				cleaned = append(cleaned, strings.TrimSpace(value))
			}
		}
		if len(cleaned) == 0 {
			return
		}
		placeholders := make([]string, 0, len(cleaned))
		for _, value := range cleaned {
			name := fmt.Sprintf("scope_%d", paramIndex)
			paramIndex++
			params[name] = value
			placeholders = append(placeholders, fmt.Sprintf("{%s:String}", name))
		}
		conditions = append(conditions, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	appendStringCondition("asset_type", req.Scope.AssetTypes)
	if req.AllowedPolicyIDs != nil {
		if len(req.AllowedPolicyIDs) == 0 {
			conditions = append(conditions, "0 = 1")
		} else {
			appendUintCondition("policy_id", req.AllowedPolicyIDs)
		}
	}
	appendUintCondition("asset_id", req.Scope.AssetIDs)
	appendUintCondition("host_id", req.Scope.HostIDs)
	appendUintCondition("cluster_id", req.Scope.ClusterIDs)
	if req.AllowedHostIDs != nil {
		if len(req.AllowedHostIDs) == 0 {
			conditions = append(conditions, "asset_type != 'host'")
		} else {
			parts := make([]string, 0, len(req.AllowedHostIDs))
			for _, value := range req.AllowedHostIDs {
				parts = append(parts, strconv.FormatUint(value, 10))
			}
			conditions = append(conditions, fmt.Sprintf("(asset_type != 'host' OR host_id IN (%s))", strings.Join(parts, ",")))
		}
	}
	appendKubernetesAccessCondition(&conditions, params, req.AllowedKubernetesScopes, "acl")
	appendStringCondition("namespace", req.Scope.Namespaces)
	appendStringCondition("service", req.Scope.Services)
	appendStringCondition("workload_name", req.Scope.Workloads)
	appendStringCondition("pod_name", req.Scope.Pods)
	appendStringCondition("container_name", req.Scope.Containers)
	appendStringCondition("node_name", req.Scope.Nodes)
	appendStringCondition("level", req.Scope.Levels)
	appendStringCondition("environment", req.Scope.Environments)
	if keyword := strings.TrimSpace(req.Query); keyword != "" && keyword != "*" {
		params["keyword"] = keyword
		conditions = append(conditions, "positionCaseInsensitiveUTF8(body, {keyword:String}) > 0")
	}
	for _, filter := range req.RequiredFilters {
		condition, err := buildInternalFilter(filter, params, &paramIndex)
		if err != nil {
			return internalSQL{}, err
		}
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}
	filterGroup, err := buildInternalFilterGroup(req.Filters, req.FilterLogic, params, &paramIndex)
	if err != nil {
		return internalSQL{}, err
	}
	if filterGroup != "" {
		conditions = append(conditions, filterGroup)
	}
	if strings.TrimSpace(req.Cursor) != "" {
		cursor, err := decodeInternalCursor(req.Cursor)
		if err != nil {
			return internalSQL{}, fmt.Errorf("分页游标无效")
		}
		operator := "<"
		if strings.EqualFold(req.Sort, "asc") {
			operator = ">"
		}
		if cursor.TimestampNanos > 0 {
			params["cursor_nanos"] = strconv.FormatInt(cursor.TimestampNanos, 10)
			conditions = append(conditions, fmt.Sprintf("(timestamp, fingerprint, sequence) %s (fromUnixTimestamp64Nano({cursor_nanos:Int64}), %d, %d)", operator, cursor.Fingerprint, cursor.Sequence))
		} else {
			params["cursor_time"] = cursor.Timestamp
			conditions = append(conditions, fmt.Sprintf("(timestamp, fingerprint, sequence) %s (parseDateTime64BestEffort({cursor_time:String}, 9, 'UTC'), %d, %d)", operator, cursor.Fingerprint, cursor.Sequence))
		}
	}
	return internalSQL{Where: strings.Join(conditions, " AND "), Params: params, Start: start.UTC(), End: end.UTC()}, nil
}

func buildInternalMetricsWhere(req InternalQueryRequest) (internalSQL, bool, error) {
	start := parseFlexibleTime(req.Start)
	end := parseFlexibleTime(req.End)
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return internalSQL{}, false, fmt.Errorf("必须提供有效的日志查询开始和结束时间")
	}
	if keyword := strings.TrimSpace(req.Query); keyword != "" && keyword != "*" {
		return internalSQL{}, false, nil
	}
	// The minute metrics table has no policy_id dimension. Policy-scoped users
	// must aggregate the raw table or the histogram could include denied logs.
	if req.AllowedPolicyIDs != nil {
		return internalSQL{}, false, nil
	}
	filterLogic, err := normalizeInternalFilterLogic(req.FilterLogic)
	if err != nil {
		return internalSQL{}, false, err
	}
	if req.Cursor != "" || len(req.Scope.Environments) > 0 || len(req.Scope.Workloads) > 0 || len(req.Scope.Pods) > 0 || len(req.Scope.Containers) > 0 || len(req.Scope.Nodes) > 0 {
		return internalSQL{}, false, nil
	}
	params := map[string]string{
		"start": start.UTC().Format(time.RFC3339Nano),
		"end":   end.UTC().Format(time.RFC3339Nano),
	}
	conditions := []string{
		"minute >= parseDateTime64BestEffort({start:String}, 9, 'UTC')",
		"minute < parseDateTime64BestEffort({end:String}, 9, 'UTC')",
	}
	if req.DenyAll {
		conditions = append(conditions, "0 = 1")
	}
	appendUintCondition := func(column string, values []uint64) {
		if len(values) == 0 {
			return
		}
		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, strconv.FormatUint(value, 10))
		}
		conditions = append(conditions, fmt.Sprintf("%s IN (%s)", column, strings.Join(parts, ",")))
	}
	paramIndex := 0
	appendStringCondition := func(column string, values []string) {
		cleaned := normalizeInternalStrings(values)
		if len(cleaned) == 0 {
			return
		}
		placeholders := make([]string, 0, len(cleaned))
		for _, value := range cleaned {
			name := fmt.Sprintf("metric_%d", paramIndex)
			paramIndex++
			params[name] = value
			placeholders = append(placeholders, fmt.Sprintf("{%s:String}", name))
		}
		conditions = append(conditions, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	appendStringCondition("asset_type", req.Scope.AssetTypes)
	appendUintCondition("asset_id", req.Scope.AssetIDs)
	if len(req.Scope.HostIDs) > 0 {
		parts := make([]string, 0, len(req.Scope.HostIDs))
		for _, value := range req.Scope.HostIDs {
			parts = append(parts, strconv.FormatUint(value, 10))
		}
		conditions = append(conditions, fmt.Sprintf("asset_type = 'host' AND asset_id IN (%s)", strings.Join(parts, ",")))
	}
	appendUintCondition("cluster_id", req.Scope.ClusterIDs)
	appendStringCondition("namespace", req.Scope.Namespaces)
	appendStringCondition("service", req.Scope.Services)
	appendStringCondition("level", req.Scope.Levels)
	if req.AllowedHostIDs != nil {
		if len(req.AllowedHostIDs) == 0 {
			conditions = append(conditions, "asset_type != 'host'")
		} else {
			parts := make([]string, 0, len(req.AllowedHostIDs))
			for _, value := range req.AllowedHostIDs {
				parts = append(parts, strconv.FormatUint(value, 10))
			}
			conditions = append(conditions, fmt.Sprintf("(asset_type != 'host' OR asset_id IN (%s))", strings.Join(parts, ",")))
		}
	}
	appendKubernetesAccessCondition(&conditions, params, req.AllowedKubernetesScopes, "metric_acl")
	for _, filter := range req.RequiredFilters {
		condition, supported := buildInternalMetricFilter(filter, params, &paramIndex)
		if !supported {
			return internalSQL{}, false, nil
		}
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}
	filterConditions := make([]string, 0, len(req.Filters))
	for _, filter := range req.Filters {
		condition, supported := buildInternalMetricFilter(filter, params, &paramIndex)
		if !supported {
			return internalSQL{}, false, nil
		}
		if condition != "" {
			filterConditions = append(filterConditions, condition)
		}
	}
	if len(filterConditions) > 0 {
		conditions = append(conditions, "("+strings.Join(filterConditions, " "+filterLogic+" ")+")")
	}
	return internalSQL{Where: strings.Join(conditions, " AND "), Params: params, Start: start.UTC(), End: end.UTC()}, true, nil
}

func normalizeInternalFilterLogic(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "and":
		return "AND", nil
	case "or":
		return "OR", nil
	default:
		return "", fmt.Errorf("字段条件关系仅支持 AND 或 OR")
	}
}

func buildInternalFilterGroup(filters []InternalQueryFilter, logic string, params map[string]string, index *int) (string, error) {
	joiner, err := normalizeInternalFilterLogic(logic)
	if err != nil {
		return "", err
	}
	conditions := make([]string, 0, len(filters))
	for _, filter := range filters {
		condition, buildErr := buildInternalFilter(filter, params, index)
		if buildErr != nil {
			return "", buildErr
		}
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return "(" + strings.Join(conditions, " "+joiner+" ") + ")", nil
}

func buildInternalMetricFilter(filter InternalQueryFilter, params map[string]string, index *int) (string, bool) {
	columns := map[string]string{
		"level": "level", "assetType": "asset_type", "assetId": "asset_id", "hostId": "asset_id",
		"clusterId": "cluster_id", "namespace": "namespace", "service": "service",
	}
	column, supported := columns[strings.TrimSpace(filter.Field)]
	operator := strings.ToLower(strings.TrimSpace(filter.Operator))
	if !supported || (operator != "" && operator != "eq" && operator != "in") {
		return "", false
	}
	values := normalizeInternalStrings(internalStringValues(filter.Value))
	if len(values) == 0 {
		return "", true
	}
	if operator != "in" {
		values = values[:1]
	}
	placeholders := make([]string, 0, len(values))
	for _, value := range values {
		name := fmt.Sprintf("metric_%d", *index)
		(*index)++
		params[name] = value
		placeholders = append(placeholders, fmt.Sprintf("{%s:String}", name))
	}
	condition := fmt.Sprintf("toString(%s) IN (%s)", column, strings.Join(placeholders, ","))
	if filter.Field == "hostId" {
		condition = "(asset_type = 'host' AND " + condition + ")"
	}
	return condition, true
}

func buildInternalFilter(filter InternalQueryFilter, params map[string]string, index *int) (string, error) {
	columns := map[string]string{
		"body": "body", "level": "level", "sourceType": "source_type", "assetType": "asset_type",
		"assetId": "asset_id", "hostId": "host_id", "clusterId": "cluster_id", "environment": "environment",
		"service": "service", "namespace": "namespace", "workloadKind": "workload_kind", "workloadName": "workload_name",
		"podName": "pod_name", "podUid": "pod_uid", "containerName": "container_name", "containerImage": "container_image",
		"nodeName": "node_name", "filePath": "file_path", "stream": "stream", "traceId": "trace_id", "spanId": "span_id",
		"agentId": "agent_id", "policyId": "policy_id", "fingerprint": "fingerprint", "sequence": "sequence",
	}
	column, ok := columns[strings.TrimSpace(filter.Field)]
	if !ok {
		return "", fmt.Errorf("不支持的日志筛选字段: %s", filter.Field)
	}
	operator := strings.ToLower(strings.TrimSpace(filter.Operator))
	values := internalStringValues(filter.Value)
	if len(values) == 0 {
		return "", nil
	}
	if operator == "in" {
		placeholders := make([]string, 0, len(values))
		for _, value := range values {
			name := fmt.Sprintf("filter_%d", *index)
			(*index)++
			params[name] = value
			placeholders = append(placeholders, fmt.Sprintf("{%s:String}", name))
		}
		return fmt.Sprintf("toString(%s) IN (%s)", column, strings.Join(placeholders, ",")), nil
	}
	name := fmt.Sprintf("filter_%d", *index)
	(*index)++
	params[name] = values[0]
	switch operator {
	case "eq", "":
		return fmt.Sprintf("toString(%s) = {%s:String}", column, name), nil
	case "neq":
		return fmt.Sprintf("toString(%s) != {%s:String}", column, name), nil
	case "contains":
		return fmt.Sprintf("positionCaseInsensitiveUTF8(toString(%s), {%s:String}) > 0", column, name), nil
	case "not_contains":
		return fmt.Sprintf("positionCaseInsensitiveUTF8(toString(%s), {%s:String}) = 0", column, name), nil
	default:
		return "", fmt.Errorf("不支持的日志筛选操作: %s", filter.Operator)
	}
}

func internalRowToLogItem(row map[string]interface{}) LogItem {
	normalized := make(map[string]interface{}, len(row)+2)
	for key, value := range row {
		normalized[key] = value
	}
	row = normalized
	if value, ok := row["fingerprintText"]; ok {
		row["fingerprint"] = value
		delete(row, "fingerprintText")
	}
	if value, ok := row["sequenceText"]; ok {
		row["sequence"] = value
		delete(row, "sequenceText")
	}
	delete(row, "timestampNanos")
	labels := map[string]string{}
	for _, key := range []string{"sourceType", "assetType", "assetId", "hostId", "clusterId", "environment", "service", "namespace", "workloadKind", "workloadName", "podName", "containerName", "nodeName", "filePath", "stream"} {
		if value := asString(row[key]); value != "" && value != "0" {
			labels[key] = value
		}
	}
	fields := make(map[string]interface{}, len(row))
	for key, value := range row {
		if key == "body" || key == "timestampText" || key == "level" {
			continue
		}
		fields[key] = value
	}
	return LogItem{
		Timestamp: asString(row["timestampText"]),
		Message:   asString(row["body"]),
		Level:     firstNonEmpty(asString(row["level"]), "UNKNOWN"),
		Labels:    labels,
		Fields:    fields,
		Raw:       row,
	}
}

func normalizeInternalLimit(limit int) int {
	if limit <= 0 {
		return defaultInternalLimit
	}
	if limit > maxInternalLimit {
		return maxInternalLimit
	}
	return limit
}

func appendInternalContextNeighbors(items []LogItem, neighbors []LogItem, selected LogItem) []LogItem {
	for _, item := range neighbors {
		if sameInternalContextLog(item, selected) {
			continue
		}
		items = append(items, item)
	}
	return items
}

func sameInternalContextLog(left, right LogItem) bool {
	leftTime := parseFlexibleTime(left.Timestamp)
	rightTime := parseFlexibleTime(right.Timestamp)
	sameTime := !leftTime.IsZero() && !rightTime.IsZero() && leftTime.Equal(rightTime)
	if !sameTime {
		return false
	}
	leftFingerprint := parseInternalUint(left.Fields["fingerprint"])
	rightFingerprint := parseInternalUint(right.Fields["fingerprint"])
	leftSequence := parseInternalUint(left.Fields["sequence"])
	rightSequence := parseInternalUint(right.Fields["sequence"])
	if leftFingerprint > 0 && rightFingerprint > 0 && leftSequence > 0 && rightSequence > 0 &&
		leftFingerprint == rightFingerprint && leftSequence == rightSequence {
		return true
	}
	return left.Message == right.Message && strings.EqualFold(left.Level, right.Level)
}

func chooseInternalBucket(duration time.Duration) time.Duration {
	step := time.Minute
	for _, candidate := range []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour} {
		step = candidate
		if duration/candidate <= 120 {
			break
		}
	}
	return step
}

func internalStringValues(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, fmt.Sprint(item))
		}
		return result
	case nil:
		return nil
	default:
		return []string{fmt.Sprint(typed)}
	}
}

func internalStringSlice(value interface{}) []string {
	values := internalStringValues(value)
	values = normalizeInternalStrings(values)
	sort.Strings(values)
	return values
}

func parseInternalKubernetesResources(value interface{}) []InternalKubernetesResourceOption {
	encoded := internalStringSlice(value)
	result := make([]InternalKubernetesResourceOption, 0, len(encoded))
	for _, item := range encoded {
		parts := strings.SplitN(item, "|", 7)
		if len(parts) != 7 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[4]) == "" {
			continue
		}
		result = append(result, InternalKubernetesResourceOption{
			ClusterID: strings.TrimSpace(parts[0]), Namespace: strings.TrimSpace(parts[1]),
			WorkloadKind: strings.TrimSpace(parts[2]), WorkloadName: strings.TrimSpace(parts[3]),
			PodName: strings.TrimSpace(parts[4]), ContainerName: strings.TrimSpace(parts[5]), NodeName: strings.TrimSpace(parts[6]),
		})
	}
	sort.Slice(result, func(left, right int) bool {
		leftKey := result[left].ClusterID + "\x00" + result[left].Namespace + "\x00" + result[left].WorkloadKind + "\x00" + result[left].WorkloadName + "\x00" + result[left].PodName + "\x00" + result[left].ContainerName
		rightKey := result[right].ClusterID + "\x00" + result[right].Namespace + "\x00" + result[right].WorkloadKind + "\x00" + result[right].WorkloadName + "\x00" + result[right].PodName + "\x00" + result[right].ContainerName
		return leftKey < rightKey
	})
	return result
}

func normalizeInternalStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func encodeInternalCursor(cursor internalCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeInternalCursor(value string) (internalCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return internalCursor{}, err
	}
	var cursor internalCursor
	err = json.Unmarshal(raw, &cursor)
	return cursor, err
}

func parseInternalUint(value interface{}) uint64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case uint64:
		return typed
	case uint:
		return uint64(typed)
	case uint32:
		return uint64(typed)
	case uint16:
		return uint64(typed)
	case uint8:
		return uint64(typed)
	case int64:
		if typed < 0 {
			return 0
		}
		return uint64(typed)
	case int:
		if typed < 0 {
			return 0
		}
		return uint64(typed)
	case int32:
		if typed < 0 {
			return 0
		}
		return uint64(typed)
	case int16:
		if typed < 0 {
			return 0
		}
		return uint64(typed)
	case int8:
		if typed < 0 {
			return 0
		}
		return uint64(typed)
	case float64:
		if typed <= 0 {
			return 0
		}
		return uint64(typed + 0.5)
	case float32:
		if typed <= 0 {
			return 0
		}
		return uint64(typed + 0.5)
	case json.Number:
		if parsed, err := strconv.ParseUint(strings.TrimSpace(typed.String()), 10, 64); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed.String()), 64); err == nil && parsed > 0 {
			return uint64(parsed + 0.5)
		}
		return 0
	}
	raw := strings.TrimSpace(fmt.Sprint(value))
	if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return parsed
	}
	if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed > 0 {
		return uint64(parsed + 0.5)
	}
	return 0
}

func parseInternalInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(typed.String()), 10, 64); err == nil {
			return parsed
		}
	}
	raw := strings.TrimSpace(fmt.Sprint(value))
	parsed, _ := strconv.ParseInt(raw, 10, 64)
	return parsed
}

func firstInternalValue(labels map[string]string, fields map[string]interface{}, key string) string {
	if value := strings.TrimSpace(labels[key]); value != "" {
		return value
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
