package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestClickHousePingUsesBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "opshub" || password != "secret" {
			t.Fatalf("BasicAuth = %q/%q/%v", username, password, ok)
		}
		_, _ = writer.Write([]byte("Ok.\n"))
	}))
	defer server.Close()

	err := NewClickHouseService().Ping(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL,
		Username:  "opshub",
		Timeout:   5,
	}, "secret")
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestClickHouseInitializeCreatesRequiredTables(t *testing.T) {
	t.Setenv("OPSHUB_LOG_TTL_MERGE_TIMEOUT_SECONDS", "1800")
	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_CLUSTER", "")
	var mutex sync.Mutex
	statements := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		mutex.Lock()
		statements = append(statements, string(raw))
		mutex.Unlock()
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := NewClickHouseService().Initialize(context.Background(), logmodel.StorageCluster{
		Endpoints:            server.URL,
		DatabaseName:         "opshub_logs",
		DefaultRetentionDays: 30,
		Timeout:              5,
	}, "")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	joined := strings.Join(statements, "\n")
	for _, expected := range []string{"CREATE DATABASE", "opshub_logs.opshub_logs", "opshub_log_metrics_1m", "MATERIALIZED VIEW", "non_replicated_deduplication_window = 100000", "merge_with_ttl_timeout = 1800"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("schema does not contain %q", expected)
		}
	}
}

func TestClickHouseInitializeCreatesReplicatedTablesForInternalCluster(t *testing.T) {
	var mutex sync.Mutex
	statements := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		mutex.Lock()
		statements = append(statements, string(raw))
		mutex.Unlock()
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT", server.URL)
	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_CLUSTER", "opshub_logs_cluster")
	err := NewClickHouseService().Initialize(context.Background(), logmodel.StorageCluster{
		Endpoints:            server.URL,
		DatabaseName:         "opshub_logs",
		DefaultRetentionDays: 30,
		Timeout:              5,
	}, "")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	joined := strings.Join(statements, "\n")
	for _, expected := range []string{
		"ON CLUSTER `opshub_logs_cluster`",
		"ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')",
		"ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')",
		"replicated_deduplication_window = 100000",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("replicated schema does not contain %q", expected)
		}
	}
	if strings.Contains(joined, "non_replicated_deduplication_window") {
		t.Fatalf("replicated schema contains standalone deduplication setting")
	}
}

func TestClickHouseInitializeDoesNotApplyInternalClusterToExternalStorage(t *testing.T) {
	var statements strings.Builder
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		statements.Write(raw)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT", "http://internal-clickhouse:8123")
	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_CLUSTER", "opshub_logs_cluster")
	err := NewClickHouseService().Initialize(context.Background(), logmodel.StorageCluster{
		Endpoints:            server.URL,
		DatabaseName:         "external_logs",
		DefaultRetentionDays: 30,
		Timeout:              5,
	}, "")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if strings.Contains(statements.String(), "ON CLUSTER") || strings.Contains(statements.String(), "ReplicatedMergeTree") {
		t.Fatalf("external storage unexpectedly used the internal ClickHouse cluster")
	}
}

func TestClickHouseInitializeRequiresEndpointWithClusterName(t *testing.T) {
	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT", "")
	t.Setenv("OPSHUB_LOGCENTER_CLICKHOUSE_CLUSTER", "opshub_logs_cluster")
	err := NewClickHouseService().Initialize(context.Background(), logmodel.StorageCluster{
		Endpoints:            "http://internal-clickhouse:8123",
		DatabaseName:         "opshub_logs",
		DefaultRetentionDays: 30,
		Timeout:              5,
	}, "")
	if err == nil || !strings.Contains(err.Error(), "必须同时配置内置存储地址") {
		t.Fatalf("missing internal endpoint error = %v", err)
	}
}

func TestRetentionHealthReportsTTLBacklogAndMerge(t *testing.T) {
	t.Setenv("OPSHUB_LOG_TTL_MERGE_TIMEOUT_SECONDS", "3600")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		switch {
		case strings.Contains(query, "FROM system.parts"):
			_, _ = writer.Write([]byte(`{"expiredParts":"2","oldestExpiredAt":"2026-07-27T01:00:00Z","ttlLagSeconds":"10800"}` + "\n"))
		case strings.Contains(query, "FROM system.merges"):
			_, _ = writer.Write([]byte(`{"mergeCount":"1","mergeProgress":"0.42"}` + "\n"))
		default:
			t.Fatalf("unexpected query: %s", query)
		}
	}))
	defer server.Close()

	result, err := NewClickHouseService().RetentionHealth(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "")
	if err != nil {
		t.Fatalf("RetentionHealth failed: %v", err)
	}
	if result.Status != "warning" || result.ExpiredParts != 2 || result.TTLLagSeconds != 10800 {
		t.Fatalf("unexpected retention health: %+v", result)
	}
	if !result.TTLMergeActive || result.TTLMergeProgress != 0.42 {
		t.Fatalf("unexpected merge state: %+v", result)
	}
}

func TestRetentionHealthStatusThresholds(t *testing.T) {
	for _, testCase := range []struct {
		parts   int64
		lag     int64
		timeout int
		want    string
	}{
		{parts: 0, lag: 99999, timeout: 3600, want: "healthy"},
		{parts: 1, lag: 7200, timeout: 3600, want: "healthy"},
		{parts: 1, lag: 7201, timeout: 3600, want: "warning"},
		{parts: 1, lag: 21601, timeout: 3600, want: "critical"},
	} {
		if got := retentionHealthStatus(testCase.parts, testCase.lag, testCase.timeout); got != testCase.want {
			t.Fatalf("retentionHealthStatus(%d, %d, %d) = %q, want %q", testCase.parts, testCase.lag, testCase.timeout, got, testCase.want)
		}
	}
}

func TestClickHouseTTLMergeTimeoutSecondsUsesSafeDefault(t *testing.T) {
	for _, value := range []string{"", "invalid", "299", "86401"} {
		t.Setenv("OPSHUB_LOG_TTL_MERGE_TIMEOUT_SECONDS", value)
		if got := clickHouseTTLMergeTimeoutSeconds(); got != 3600 {
			t.Fatalf("value %q produced %d", value, got)
		}
	}
}

func TestBuildInternalWhereKeepsValuesOutOfSQL(t *testing.T) {
	malicious := `timeout') OR 1 = 1 --`
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-13T00:00:00Z",
		End:   "2026-07-13T01:00:00Z",
		Query: malicious,
		Filters: []InternalQueryFilter{
			{Field: "service", Operator: "eq", Value: malicious},
		},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if strings.Contains(built.Where, malicious) {
		t.Fatalf("unsafe value leaked into SQL: %s", built.Where)
	}
	if built.Params["keyword"] != malicious || built.Params["filter_0"] != malicious {
		t.Fatalf("query params were not preserved: %#v", built.Params)
	}
}

func TestBuildInternalWhereGroupsFiltersWithSelectedLogic(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start:       "2026-07-13T00:00:00Z",
		End:         "2026-07-13T01:00:00Z",
		FilterLogic: "or",
		Scope:       InternalQueryScope{ClusterIDs: []uint64{7}, Levels: []string{"ERROR"}},
		Filters: []InternalQueryFilter{
			{Field: "service", Operator: "eq", Value: "frontend"},
			{Field: "body", Operator: "contains", Value: "timeout"},
		},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, "cluster_id IN (7) AND level IN ({scope_0:String}) AND (") {
		t.Fatalf("scope filters were not kept outside the custom filter group: %s", built.Where)
	}
	if !strings.Contains(built.Where, " OR ") || strings.Count(built.Where, " OR ") != 1 {
		t.Fatalf("custom filters were not joined with OR: %s", built.Where)
	}
	if built.Params["filter_1"] != "frontend" || built.Params["filter_2"] != "timeout" {
		t.Fatalf("custom filter params = %#v", built.Params)
	}
}

func TestBuildInternalWhereDefaultsFilterLogicToAnd(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-13T00:00:00Z", End: "2026-07-13T01:00:00Z",
		Filters: []InternalQueryFilter{
			{Field: "service", Operator: "eq", Value: "frontend"},
			{Field: "namespace", Operator: "eq", Value: "production"},
		},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, " AND ") || strings.Contains(built.Where, " OR ") {
		t.Fatalf("default filter logic was not AND: %s", built.Where)
	}
}

func TestBuildInternalWhereSupportsMultipleValuesForAnyField(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-13T00:00:00Z",
		End:   "2026-07-13T01:00:00Z",
		Filters: []InternalQueryFilter{
			{Field: "traceId", Operator: "in", Value: []string{"trace-a", "trace-b", "trace-c"}},
		},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, "toString(trace_id) IN ({filter_0:String},{filter_1:String},{filter_2:String})") {
		t.Fatalf("multi-value trace filter was not generated: %s", built.Where)
	}
	if built.Params["filter_0"] != "trace-a" || built.Params["filter_1"] != "trace-b" || built.Params["filter_2"] != "trace-c" {
		t.Fatalf("multi-value trace params = %#v", built.Params)
	}
}

func TestBuildInternalWhereRejectsInvalidFilterLogic(t *testing.T) {
	_, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-13T00:00:00Z", End: "2026-07-13T01:00:00Z", FilterLogic: "AND 1 = 1",
	})
	if err == nil || !strings.Contains(err.Error(), "AND 或 OR") {
		t.Fatalf("invalid filter logic error = %v", err)
	}
}

func TestBuildInternalWhereRequiresTimeRange(t *testing.T) {
	if _, err := buildInternalWhere(InternalQueryRequest{}); err == nil {
		t.Fatal("buildInternalWhere succeeded without a time range")
	}
}

func TestBuildInternalWhereAppliesHostPermissions(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start:          "2026-07-13T00:00:00Z",
		End:            "2026-07-13T01:00:00Z",
		AllowedHostIDs: []uint64{2, 9},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, "asset_type != 'host' OR host_id IN (2,9)") {
		t.Fatalf("host permission condition missing: %s", built.Where)
	}
}

func TestInternalAlertSamplesAggregatesAndReturnsMatchingLogs(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		writer.Header().Set("Content-Type", "application/json")
		if strings.Contains(query, "toString(count()) AS value") {
			if !strings.Contains(query, "GROUP BY service, namespace") {
				t.Fatalf("alert aggregation did not group by selected fields: %s", query)
			}
			_, _ = writer.Write([]byte(`{"service":"api","namespace":"production","value":"3"}` + "\n"))
			return
		}
		if !strings.Contains(query, "service") || !strings.Contains(query, "namespace") {
			t.Fatalf("sample query did not retain group filters: %s", query)
		}
		_, _ = writer.Write([]byte(`{"timestampText":"2026-07-15T02:00:00.000000000Z","body":"ERROR database timeout","level":"ERROR","service":"api","namespace":"production","fingerprintText":"11","sequenceText":"12"}` + "\n"))
	}))
	defer server.Close()

	samples, err := NewInternalQueryService().AlertSamples(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalQueryRequest{
		Start: "2026-07-15T01:55:00Z",
		End:   "2026-07-15T02:00:00Z",
		Query: "ERROR",
	}, []string{"service", "namespace"}, 5)
	if err != nil {
		t.Fatalf("AlertSamples failed: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("AlertSamples made %d requests, want 2", requestCount)
	}
	if len(samples) != 1 || samples[0].Value != 3 {
		t.Fatalf("unexpected alert samples: %#v", samples)
	}
	if samples[0].Labels["service"] != "api" || samples[0].Labels["namespace"] != "production" {
		t.Fatalf("alert labels were not preserved: %#v", samples[0].Labels)
	}
	if len(samples[0].Logs) != 1 || samples[0].Logs[0].Message != "ERROR database timeout" {
		t.Fatalf("matching log samples were not returned: %#v", samples[0].Logs)
	}
}

func TestNormalizeInternalLimitCapsLargeRequests(t *testing.T) {
	if got := normalizeInternalLimit(50000); got != 2000 {
		t.Fatalf("normalizeInternalLimit = %d, want 2000", got)
	}
	if got := normalizeInternalLimit(0); got != 200 {
		t.Fatalf("normalizeInternalLimit default = %d, want 200", got)
	}
}

func TestParseInternalUintAcceptsClickHouseJSONNumbers(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
		want  uint64
	}{
		{name: "integer string", value: "1042198", want: 1042198},
		{name: "json float", value: float64(1042198), want: 1042198},
		{name: "scientific string", value: "1.042198e+06", want: 1042198},
		{name: "small float", value: float64(796610), want: 796610},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInternalUint(tt.value); got != tt.want {
				t.Fatalf("parseInternalUint(%#v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestBuildInternalWhereSupportsDisplayedFields(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-13T00:00:00Z",
		End:   "2026-07-13T01:00:00Z",
		Filters: []InternalQueryFilter{
			{Field: "workloadKind", Operator: "eq", Value: "Deployment"},
			{Field: "agentId", Operator: "contains", Value: "agent-"},
			{Field: "policyId", Operator: "eq", Value: "3"},
		},
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	for _, expected := range []string{"workload_kind", "agent_id", "policy_id"} {
		if !strings.Contains(built.Where, expected) {
			t.Fatalf("where clause missing %s: %s", expected, built.Where)
		}
	}
}

func TestEstimateCapacityUsesActualStoredBytesAndSafetyReserve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		switch {
		case strings.Contains(query, "FROM system.parts"):
			_, _ = writer.Write([]byte(`{"rows":"10000","compressedBytes":"1000000","storedBytes":"3000000","uncompressedBytes":"5000000","expiredParts":"1","oldestExpiredAt":"2026-07-27T01:00:00Z","ttlLagSeconds":"1800"}` + "\n"))
		case strings.Contains(query, "FROM system.merges"):
			_, _ = writer.Write([]byte(`{"mergeCount":"1","mergeProgress":"0.25"}` + "\n"))
		case strings.Contains(query, "countIf(expire_at"):
			_, _ = writer.Write([]byte(`{"expiredRows":"321"}` + "\n"))
		case strings.Contains(query, "opshub_log_metrics_1m"):
			_, _ = writer.Write([]byte(`{"logs":"1000","rawBytes":"1000000"}` + "\n"))
		case strings.Contains(query, "FROM system.disks"):
			_, _ = writer.Write([]byte(`{"totalBytes":"1000000000","freeBytes":"400000000"}` + "\n"))
		default:
			t.Fatalf("unexpected capacity query: %s", query)
		}
	}))
	defer server.Close()

	estimate, err := NewClickHouseService().EstimateCapacity(context.Background(), logmodel.StorageCluster{
		ID: 7, Name: "primary", Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", 30)
	if err != nil {
		t.Fatalf("EstimateCapacity failed: %v", err)
	}
	if estimate.CompressionRatio != 5 || estimate.AverageRecordBytes != 1000 || estimate.AverageStoredBytes != 300 {
		t.Fatalf("unexpected compression estimate: %#v", estimate)
	}
	if estimate.DailyStoredBytes != 300000 || estimate.ProjectedStoredBytes != 9000000 || estimate.RecommendedBytes != 11700000 {
		t.Fatalf("unexpected retention projection: %#v", estimate)
	}
	if estimate.ForecastBasis != "stored_bytes_per_row" || estimate.DiskReservedBytes != 200000000 || estimate.UsableFreeBytes != 200000000 {
		t.Fatalf("unexpected forecast safeguards: %#v", estimate)
	}
	if estimate.ProjectedUsage < 1.462 || estimate.ProjectedUsage > 1.463 || estimate.DaysUntilFull < 512 || estimate.DaysUntilFull > 513 {
		t.Fatalf("unexpected projected disk usage or writable days: usage=%f days=%f", estimate.ProjectedUsage, estimate.DaysUntilFull)
	}
	if estimate.Retention.Status != "healthy" || estimate.Retention.ExpiredRows != 321 || estimate.Retention.TTLMergeProgress != 0.25 {
		t.Fatalf("unexpected retention health: %+v", estimate.Retention)
	}
}

func TestInternalQueryDoesNotShadowTimestampColumn(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query = string(raw)
		_, _ = writer.Write([]byte(`{"timestampText":"2026-07-14T00:15:00.000000000Z","timestampNanos":"1720916100000000000","body":"test log","level":"INFO","fingerprintText":"1","sequenceText":"2"}` + "\n"))
	}))
	defer server.Close()

	result, err := NewInternalQueryService().Query(context.Background(), logmodel.StorageCluster{
		Endpoints:    server.URL,
		DatabaseName: "opshub_logs",
		Timeout:      5,
	}, "", InternalQueryRequest{
		Start: "2026-07-14T00:00:00Z",
		End:   "2026-07-14T01:00:00Z",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if strings.Contains(query, "AS timestamp,") {
		t.Fatalf("formatted timestamp shadows source column: %s", query)
	}
	if !strings.Contains(query, "AS timestampText") || !strings.Contains(query, "AS timestampNanos") || !strings.Contains(query, "WHERE timestamp >=") {
		t.Fatalf("timestamp projection or predicate is invalid: %s", query)
	}
	if strings.Contains(query, "AS fingerprint,") || strings.Contains(query, "AS sequence\n") {
		t.Fatalf("cursor projection shadows source columns: %s", query)
	}
	if !strings.Contains(query, "AS fingerprintText") || !strings.Contains(query, "AS sequenceText") {
		t.Fatalf("cursor projection aliases are missing: %s", query)
	}
	if len(result.Items) != 1 || result.Items[0].Timestamp != "2026-07-14T00:15:00.000000000Z" {
		t.Fatalf("unexpected query result: %#v", result.Items)
	}
	if result.Items[0].Fields["fingerprint"] != "1" || result.Items[0].Fields["sequence"] != "2" {
		t.Fatalf("cursor fields were not normalized: %#v", result.Items[0].Fields)
	}
	cursor, err := decodeInternalCursor(result.NextCursor)
	if err != nil || cursor.TimestampNanos != 1720916100000000000 || cursor.Fingerprint != 1 || cursor.Sequence != 2 {
		t.Fatalf("next cursor = %#v, err = %v", cursor, err)
	}
}

func TestBuildInternalWhereUsesExactNanosecondCursor(t *testing.T) {
	cursor := encodeInternalCursor(internalCursor{
		Timestamp:      "2026-07-14T00:15:00.123456Z",
		TimestampNanos: 1720916100123456789,
		Fingerprint:    11,
		Sequence:       12,
	})
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-14T00:00:00Z", End: "2026-07-14T01:00:00Z",
		Sort: "asc", Cursor: cursor,
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, "fromUnixTimestamp64Nano({cursor_nanos:Int64})") {
		t.Fatalf("nanosecond cursor predicate missing: %s", built.Where)
	}
	if strings.Contains(built.Where, "parseDateTime64BestEffort({cursor_time:String}") {
		t.Fatalf("nanosecond cursor fell back to lossy timestamp text: %s", built.Where)
	}
	if built.Params["cursor_nanos"] != "1720916100123456789" {
		t.Fatalf("cursor nanos parameter = %q", built.Params["cursor_nanos"])
	}
}

func TestBuildInternalWhereSupportsLegacyTimestampCursor(t *testing.T) {
	cursor := encodeInternalCursor(internalCursor{
		Timestamp: "2026-07-14T00:15:00.123456Z", Fingerprint: 11, Sequence: 12,
	})
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-14T00:00:00Z", End: "2026-07-14T01:00:00Z",
		Sort: "desc", Cursor: cursor,
	})
	if err != nil {
		t.Fatalf("buildInternalWhere failed: %v", err)
	}
	if !strings.Contains(built.Where, "parseDateTime64BestEffort({cursor_time:String}, 9, 'UTC')") {
		t.Fatalf("legacy cursor predicate missing: %s", built.Where)
	}
}

func TestInternalHistogramUsesMinuteMetricsWhenCompatible(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query = string(raw)
		_, _ = writer.Write([]byte(`{"time":"2026-07-14T00:15:00Z","count":"42"}` + "\n"))
	}))
	defer server.Close()

	bars, err := NewInternalQueryService().Histogram(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalQueryRequest{
		Start: "2026-07-14T00:00:00Z", End: "2026-07-14T01:00:00Z", Query: "*",
		Scope:   InternalQueryScope{ClusterIDs: []uint64{7}, Namespaces: []string{"production"}},
		Filters: []InternalQueryFilter{{Field: "level", Operator: "eq", Value: "ERROR"}},
	})
	if err != nil {
		t.Fatalf("Histogram failed: %v", err)
	}
	if !strings.Contains(query, "opshub_log_metrics_1m") || !strings.Contains(query, "sum(log_count)") {
		t.Fatalf("histogram did not use minute metrics: %s", query)
	}
	found := false
	for _, bar := range bars {
		if bar.Count == 42 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("histogram count missing: %#v", bars)
	}
}

func TestInternalHistogramFallsBackToRawLogsForKeyword(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query = string(raw)
	}))
	defer server.Close()

	_, err := NewInternalQueryService().Histogram(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalQueryRequest{Start: "2026-07-14T00:00:00Z", End: "2026-07-14T01:00:00Z", Query: "timeout"})
	if err != nil {
		t.Fatalf("Histogram failed: %v", err)
	}
	if !strings.Contains(query, "opshub_logs") || strings.Contains(query, "opshub_log_metrics_1m") {
		t.Fatalf("keyword histogram did not fall back to raw logs: %s", query)
	}
}

func TestInternalHistogramUsesRawEdgesAroundMinuteMetrics(t *testing.T) {
	var mutex sync.Mutex
	queries := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		mutex.Lock()
		queries = append(queries, query)
		mutex.Unlock()
		if strings.Contains(query, "opshub_log_metrics_1m") {
			_, _ = writer.Write([]byte(`{"time":"2026-07-14T00:01:00Z","count":"10"}` + "\n"))
			return
		}
		timestamp := "2026-07-14T00:00:00Z"
		if strings.HasPrefix(request.URL.Query().Get("param_start"), "2026-07-14T00:03:00") {
			timestamp = "2026-07-14T00:03:00Z"
		}
		_, _ = writer.Write([]byte(fmt.Sprintf(`{"time":%q,"count":"2"}`+"\n", timestamp)))
	}))
	defer server.Close()

	bars, err := NewInternalQueryService().Histogram(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalQueryRequest{Start: "2026-07-14T00:00:15Z", End: "2026-07-14T00:03:15Z", Query: "*"})
	if err != nil {
		t.Fatalf("Histogram failed: %v", err)
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(queries) != 3 {
		t.Fatalf("histogram queries = %d, want 3: %#v", len(queries), queries)
	}
	if !strings.Contains(queries[1], "opshub_log_metrics_1m") {
		t.Fatalf("middle range did not use minute metrics: %#v", queries)
	}
	if !strings.Contains(queries[0], "opshub_logs") || !strings.Contains(queries[2], "opshub_logs") {
		t.Fatalf("partial minute edges did not use raw logs: %#v", queries)
	}
	var total int
	for _, bar := range bars {
		total += bar.Count
	}
	if total != 14 {
		t.Fatalf("histogram total = %d, want 14: %#v", total, bars)
	}
}

func TestInternalContextCentersSelectedLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		if strings.Contains(query, "ORDER BY timestamp DESC") {
			_, _ = writer.Write([]byte(
				`{"timestampText":"2026-07-14T00:09:59.000000000Z","body":"before-2","level":"INFO","fingerprint":"8","sequence":"8"}` + "\n" +
					`{"timestampText":"2026-07-14T00:09:58.000000000Z","body":"before-1","level":"INFO","fingerprint":"7","sequence":"7"}` + "\n",
			))
			return
		}
		_, _ = writer.Write([]byte(
			`{"timestampText":"2026-07-14T00:10:00.000000000Z","body":"selected","level":"ERROR","fingerprintText":"10","sequenceText":"10"}` + "\n" +
				`{"timestampText":"2026-07-14T00:10:01.000000000Z","body":"after-1","level":"WARN","fingerprintText":"11","sequenceText":"11"}` + "\n",
		))
	}))
	defer server.Close()

	result, err := NewInternalQueryService().Context(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalContextRequest{
		Timestamp: "2026-07-14T00:10:00Z", Message: "selected", Level: "ERROR", Limit: 5,
		Fields: map[string]interface{}{"fingerprint": "9", "sequence": "9"},
	})
	if err != nil {
		t.Fatalf("Context failed: %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("context items = %#v", result.Items)
	}
	if result.Items[0].Message != "before-1" || result.Items[1].Message != "before-2" || result.Items[2].Message != "selected" || result.Items[3].Message != "after-1" {
		t.Fatalf("selected log was not centered: %#v", result.Items)
	}
	selectedCount := 0
	for _, item := range result.Items {
		if item.ContextSelected {
			selectedCount++
		}
	}
	if selectedCount != 1 || !result.Items[2].ContextSelected {
		t.Fatalf("context selected markers = %d, items = %#v", selectedCount, result.Items)
	}
}

func TestInternalResourceOptionsReturnsSortedValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"hostIds":["9","2"],"clusterIds":["3"],"environments":["prod"],"services":["web","api"],"namespaces":["default"],"workloads":["frontend"],"pods":["frontend-1"],"containers":["nginx"],"nodes":["node-1"],"kubernetesResources":["3|default|Deployment|frontend|frontend-2|nginx|node-2","3|default|Deployment|frontend|frontend-1|nginx|node-1"]}` + "\n"))
	}))
	defer server.Close()

	options, err := NewInternalQueryService().ResourceOptions(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", InternalQueryRequest{Start: "2026-07-14T00:00:00Z", End: "2026-07-14T01:00:00Z"})
	if err != nil {
		t.Fatalf("ResourceOptions failed: %v", err)
	}
	if strings.Join(options.HostIDs, ",") != "2,9" || strings.Join(options.Services, ",") != "api,web" {
		t.Fatalf("resource options were not normalized: %#v", options)
	}
	if len(options.KubernetesResources) != 2 || options.KubernetesResources[0].PodName != "frontend-1" || options.KubernetesResources[1].NodeName != "node-2" {
		t.Fatalf("kubernetes resource paths were not parsed and sorted: %#v", options.KubernetesResources)
	}
}
