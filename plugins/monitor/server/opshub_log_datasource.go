package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	logcenterserver "github.com/ydcloud-dy/opshub/plugins/logcenter/server"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
)

const opsHubLogDataSourceType = "opshub_logs"

type opsHubLogQueryAST struct {
	Version       int                          `json:"version"`
	StorageID     uint                         `json:"storageId"`
	WindowSeconds int                          `json:"windowSeconds"`
	Keyword       string                       `json:"keyword"`
	Scope         logsvc.InternalQueryScope    `json:"scope"`
	Filters       []logsvc.InternalQueryFilter `json:"filters"`
	GroupBy       []string                     `json:"groupBy"`
	Aggregation   string                       `json:"aggregation"`
	SampleLimit   int                          `json:"sampleLimit"`
}

func (h *DataSourceHandler) queryOpsHubLogs(ctx context.Context, datasource *model.DataSource, request dataSourceQueryRequest) (interface{}, int, error) {
	ast, err := parseOpsHubLogQueryAST(request.Query)
	if err != nil {
		return nil, 0, err
	}
	if ast.StorageID == 0 {
		ast.StorageID = opsHubStorageIDFromURL(datasource.URL)
	}
	storage, password, err := logcenterserver.LoadInternalStorage(h.db, ast.StorageID)
	if err != nil {
		return nil, 0, err
	}
	end := parseMonitorQueryTime(request.End)
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := parseMonitorQueryTime(request.Start)
	if start.IsZero() || !start.Before(end) {
		start = end.Add(-time.Duration(ast.WindowSeconds) * time.Second)
	}
	queryRequest := logsvc.InternalQueryRequest{
		StorageID: storage.ID, Start: start.Format(time.RFC3339Nano), End: end.Format(time.RFC3339Nano),
		Query: firstNonEmpty(ast.Keyword, "*"), Scope: ast.Scope, Filters: ast.Filters, Sort: "desc", Limit: ast.SampleLimit, SkipHistory: true,
	}
	samples, err := logsvc.NewInternalQueryService().AlertSamples(ctx, storage, password, queryRequest, ast.GroupBy, ast.SampleLimit)
	if err != nil {
		return nil, 0, err
	}
	results := make([]interface{}, 0, len(samples))
	for _, sample := range samples {
		value := sample.Value
		if ast.Aggregation == "rate" {
			value = value / float64(ast.WindowSeconds)
		}
		labels := cloneStringMap(sample.Labels)
		labels["storageId"] = strconv.FormatUint(uint64(storage.ID), 10)
		labels["logQuery"] = ast.Keyword
		labels["logWindowSeconds"] = strconv.Itoa(ast.WindowSeconds)
		if len(ast.Scope.HostIDs) == 1 {
			labels["hostId"] = strconv.FormatUint(ast.Scope.HostIDs[0], 10)
		}
		if len(ast.Scope.ClusterIDs) == 1 {
			labels["clusterId"] = strconv.FormatUint(ast.Scope.ClusterIDs[0], 10)
		}
		logs := make([]interface{}, 0, len(sample.Logs))
		for _, item := range sample.Logs {
			logs = append(logs, map[string]interface{}{"timestamp": item.Timestamp, "line": item.Message, "labels": item.Labels})
		}
		results = append(results, map[string]interface{}{
			"metric":          labels,
			"value":           []interface{}{float64(end.Unix()), strconv.FormatFloat(value, 'f', -1, 64)},
			"matchedLogs":     logs,
			"matchedLogCount": int(sample.Value),
			"matchedLogQuery": request.Query,
		})
	}
	return map[string]interface{}{
		"status": "success",
		"data":   map[string]interface{}{"resultType": "vector", "result": results},
	}, 200, nil
}

func parseOpsHubLogQueryAST(raw string) (opsHubLogQueryAST, error) {
	var ast opsHubLogQueryAST
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &ast); err != nil {
		return ast, fmt.Errorf("OpsHub 日志告警查询必须是结构化 Query AST: %w", err)
	}
	if ast.Version == 0 {
		ast.Version = 1
	}
	if ast.Version != 1 {
		return ast, fmt.Errorf("不支持的日志 Query AST 版本: %d", ast.Version)
	}
	if ast.WindowSeconds <= 0 {
		ast.WindowSeconds = 300
	}
	if ast.WindowSeconds < 10 || ast.WindowSeconds > 7*24*60*60 {
		return ast, fmt.Errorf("日志告警窗口必须在 10 秒到 7 天之间")
	}
	ast.Aggregation = strings.ToLower(strings.TrimSpace(ast.Aggregation))
	if ast.Aggregation == "" {
		ast.Aggregation = "count"
	}
	if ast.Aggregation != "count" && ast.Aggregation != "rate" {
		return ast, fmt.Errorf("日志告警聚合仅支持 count 或 rate")
	}
	if ast.SampleLimit <= 0 {
		ast.SampleLimit = 5
	}
	if ast.SampleLimit > 20 {
		ast.SampleLimit = 20
	}
	if strings.TrimSpace(ast.Keyword) == "" {
		ast.Keyword = "*"
	}
	return ast, nil
}

func extractOpsHubLogSamples(raw interface{}) ([]ruleEvaluationSample, error) {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("OpsHub 日志响应不是 JSON 对象")
	}
	data, _ := root["data"].(map[string]interface{})
	results, _ := data["result"].([]interface{})
	samples := make([]ruleEvaluationSample, 0, len(results))
	for _, rawResult := range results {
		series, ok := rawResult.(map[string]interface{})
		if !ok {
			continue
		}
		value, err := valueFromPair(series["value"])
		if err != nil {
			continue
		}
		labels := labelsFromInterfaceMap(series["metric"])
		logs := opsHubMatchedLogs(series["matchedLogs"], labels)
		samples = append(samples, ruleEvaluationSample{
			Value: value, Labels: labels, MatchedLogs: logs,
			MatchedLogCount: int(interfaceFloat(series["matchedLogCount"])), MatchedLogQuery: strings.TrimSpace(fmt.Sprint(series["matchedLogQuery"])),
		})
	}
	if len(samples) == 0 {
		return []ruleEvaluationSample{{Value: 0, Labels: map[string]string{}}}, nil
	}
	return samples, nil
}

func opsHubMatchedLogs(raw interface{}, fallbackLabels map[string]string) []matchedLogEntry {
	values, _ := raw.([]interface{})
	result := make([]matchedLogEntry, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		labels := labelsFromInterfaceMap(item["labels"])
		if len(labels) == 0 {
			labels = cloneStringMap(fallbackLabels)
		}
		result = append(result, matchedLogEntry{
			Timestamp: strings.TrimSpace(fmt.Sprint(item["timestamp"])),
			Line:      clipPlainText(strings.TrimSpace(fmt.Sprint(item["line"])), maxMatchedLogLineChars), Labels: labels,
		})
	}
	return result
}

func opsHubStorageIDFromURL(raw string) uint {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	id, _ := strconv.ParseUint(parts[len(parts)-1], 10, 64)
	return uint(id)
}

func parseMonitorQueryTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC()
		}
	}
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		seconds := int64(value)
		nanos := int64((value - float64(seconds)) * float64(time.Second))
		if seconds > 1_000_000_000_000 {
			return time.UnixMilli(seconds).UTC()
		}
		return time.Unix(seconds, nanos).UTC()
	}
	return time.Time{}
}

func interfaceFloat(value interface{}) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	return parsed
}

func buildOpsHubLogNoticeURL(payload ruleNotificationPayload) string {
	annotations := parseStringMap(payload.Annotations)
	rawQuery := strings.TrimSpace(firstNonEmpty(annotations["matched_log_query"], annotations["matchedLogQuery"]))
	ast, err := parseOpsHubLogQueryAST(rawQuery)
	if err != nil {
		return ""
	}
	labels := parseStringMap(payload.Labels)
	end := payload.Time
	if payload.EndedAt != nil && !payload.EndedAt.IsZero() {
		end = *payload.EndedAt
	}
	if end.IsZero() {
		end = time.Now()
	}
	start := end.Add(-time.Duration(ast.WindowSeconds) * time.Second)
	values := url.Values{}
	values.Set("fromAlert", "1")
	values.Set("start", start.UTC().Format(time.RFC3339Nano))
	values.Set("end", end.UTC().Format(time.RFC3339Nano))
	values.Set("q", ast.Keyword)
	storageID := ast.StorageID
	if storageID == 0 {
		parsed, _ := strconv.ParseUint(labels["storageId"], 10, 64)
		storageID = uint(parsed)
	}
	if storageID > 0 {
		values.Set("storageId", strconv.FormatUint(uint64(storageID), 10))
	}
	setUintQueryValues(values, "hostIds", ast.Scope.HostIDs, labels["hostId"])
	setUintQueryValues(values, "clusterIds", ast.Scope.ClusterIDs, labels["clusterId"])
	setStringQueryValues(values, "environments", ast.Scope.Environments, labels["environment"])
	setStringQueryValues(values, "services", ast.Scope.Services, labels["service"])
	setStringQueryValues(values, "namespaces", ast.Scope.Namespaces, labels["namespace"])
	setStringQueryValues(values, "workloads", ast.Scope.Workloads, labels["workloadName"])
	setStringQueryValues(values, "pods", ast.Scope.Pods, labels["podName"])
	setStringQueryValues(values, "containers", ast.Scope.Containers, labels["containerName"])
	if len(ast.Filters) > 0 {
		if raw, marshalErr := json.Marshal(ast.Filters); marshalErr == nil {
			values.Set("filters", string(raw))
		}
	}
	path := "/logs/query?" + values.Encode()
	base := noticeFrontendBaseURL()
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + path
}

func setUintQueryValues(values url.Values, key string, configured []uint64, fallback string) {
	items := make([]string, 0, len(configured)+1)
	for _, value := range configured {
		items = append(items, strconv.FormatUint(value, 10))
	}
	if len(items) == 0 && strings.TrimSpace(fallback) != "" {
		items = append(items, strings.TrimSpace(fallback))
	}
	if len(items) > 0 {
		values.Set(key, strings.Join(items, ","))
	}
}

func setStringQueryValues(values url.Values, key string, configured []string, fallback string) {
	items := append([]string(nil), configured...)
	if len(items) == 0 && strings.TrimSpace(fallback) != "" {
		items = append(items, strings.TrimSpace(fallback))
	}
	if len(items) > 0 {
		values.Set(key, strings.Join(items, ","))
	}
}
