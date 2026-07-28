package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

func TestBuildLogOverviewSnapshot(t *testing.T) {
	rows := []map[string]interface{}{
		{"category": "service", "metric_key": "order-api", "metric_value": "120"},
		{"category": "service", "metric_key": "web", "metric_value": float64(480)},
		{"category": "summary", "metric_key": "logs24h", "metric_value": "600"},
		{"category": "summary", "metric_key": "bytes24h", "metric_value": float64(4096)},
		{"category": "summary", "metric_key": "errors24h", "metric_value": "12"},
		{"category": "summary", "metric_key": "averageEps5m", "metric_value": 2.5},
		{"category": "summary", "metric_key": "activeServices", "metric_value": float64(2)},
		{"category": "level", "metric_key": "INFO", "metric_value": "580"},
		{"category": "level", "metric_key": "ERROR", "metric_value": float64(20)},
		{"category": "trend", "metric_key": "2026-07-23T11:00:00Z", "metric_value": "400", "metric_bytes": "4096"},
		{"category": "trend", "metric_key": "2026-07-23T10:00:00Z", "metric_value": float64(200), "metric_bytes": float64(2048)},
	}

	snapshot := buildLogOverviewSnapshot(rows)
	if snapshot.Logs24H != 600 || snapshot.Bytes24H != 4096 || snapshot.Errors24H != 12 {
		t.Fatalf("unexpected summary: %#v", snapshot)
	}
	if snapshot.AverageEPS5M != 2.5 || snapshot.ActiveServices != 2 {
		t.Fatalf("unexpected rate or services: %#v", snapshot)
	}
	if len(snapshot.Trend) != 2 || snapshot.Trend[0].Time != "2026-07-23T10:00:00Z" {
		t.Fatalf("trend was not sorted: %#v", snapshot.Trend)
	}
	if snapshot.Trend[0].Bytes != 2048 || snapshot.Trend[1].Bytes != 4096 {
		t.Fatalf("trend bytes were not parsed: %#v", snapshot.Trend)
	}
	if len(snapshot.TopServices) != 2 || snapshot.TopServices[0].Name != "web" {
		t.Fatalf("services were not sorted: %#v", snapshot.TopServices)
	}
}

func TestLogOverviewNumberRejectsNegativeValues(t *testing.T) {
	if value := logOverviewNumber("-12"); value != 0 {
		t.Fatalf("expected negative value to be clamped, got %v", value)
	}
	if value := logOverviewNumber("3.25"); value != 3.25 {
		t.Fatalf("expected decimal value, got %v", value)
	}
}

func TestQueryLogOverviewSnapshotUsesMinuteMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query := string(raw)
		if strings.HasPrefix(strings.TrimSpace(query), "SELECT formatDateTime") {
			_, _ = writer.Write([]byte("{\"metric_key\":\"2026-07-23T10:00:00Z\",\"metric_value\":\"1024\"}\n"))
			return
		}
		for _, expected := range []string{"opshub_log_metrics_1m", "INTERVAL 24 HOUR", "averageEps5m", "GROUP BY service"} {
			if !strings.Contains(query, expected) {
				t.Fatalf("overview query is missing %q: %s", expected, query)
			}
		}
		_, _ = writer.Write([]byte("{\"category\":\"summary\",\"metric_key\":\"logs24h\",\"metric_value\":\"42\"}\n"))
	}))
	defer server.Close()

	handler := &Handler{clickhouse: logsvc.NewClickHouseService()}
	snapshot, err := handler.queryLogOverviewSnapshot(context.Background(), logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "", logOverviewTrendConfigFor("24h"))
	if err != nil {
		t.Fatalf("queryLogOverviewSnapshot failed: %v", err)
	}
	if snapshot.Logs24H != 42 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestLogOverviewTrendConfigUsesLocalCalendarBuckets(t *testing.T) {
	daily := logOverviewTrendConfigFor("30d")
	if daily.Range != "30d" || daily.Interval != "30 DAY" || !strings.Contains(daily.Key, "Asia/Shanghai") {
		t.Fatalf("unexpected daily trend config: %#v", daily)
	}
	monthly := logOverviewTrendConfigFor("12m")
	if monthly.Range != "12m" || monthly.Interval != "12 MONTH" || !strings.Contains(monthly.Key, "toStartOfMonth") {
		t.Fatalf("unexpected monthly trend config: %#v", monthly)
	}
	if fallback := logOverviewTrendConfigFor("invalid"); fallback.Range != "24h" {
		t.Fatalf("invalid trend range should fall back to 24h: %#v", fallback)
	}
}
