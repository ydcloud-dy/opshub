package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestAlertRuleLabelsAnnotationsAndCallbackVariables(t *testing.T) {
	raw := map[string]interface{}{
		"data": map[string]interface{}{
			"result": []interface{}{
				map[string]interface{}{
					"metric": map[string]interface{}{
						"instance": "10.0.0.1:9100",
						"job":      "node",
					},
					"value": []interface{}{float64(1710000000), "95.2"},
				},
			},
		},
	}
	result := &alertRuleEvaluationResult{
		RuleID:         7,
		RuleName:       "CPU 使用率",
		DataSourceName: "Prometheus",
		DataSourceType: "prometheus",
		Severity:       "p1",
		Value:          95.2,
		Condition:      "gt",
		Threshold:      90,
		Labels:         extractRuleResultLabels("prometheus", raw),
		Message:        "CPU 使用率过高",
		EvaluatedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.Local),
	}
	rule := &model.AlertRule{
		ID:             7,
		Name:           "CPU 使用率",
		Query:          `node_cpu_seconds_total{mode!="idle"}`,
		Labels:         `{"team":"ops","target":"${labels.instance}"}`,
		Annotations:    `{"summary":"${labels.instance} CPU 高于 ${threshold}","value":"${labels.value}","watchValue":"{{ $labels.value }}","labelText":"{{ $labels }}"}`,
		DetailTemplate: `节点：${labels.instance}，CPU 使用率过高，当前：{{ $labels.value }}%，旧变量：{{ $value }}%，阈值：${threshold}%`,
	}

	labels := buildRuleLabelMap(rule, result)
	if labels["instance"] != "10.0.0.1:9100" {
		t.Fatalf("expected datasource instance label, got %q", labels["instance"])
	}
	if labels["target"] != "10.0.0.1:9100" {
		t.Fatalf("expected rendered rule label, got %q", labels["target"])
	}
	if labels["team"] != "ops" {
		t.Fatalf("expected rule label override/addition, got %q", labels["team"])
	}
	if labels["value"] != "95.2" {
		t.Fatalf("expected current value to be exposed as labels.value, got %q", labels["value"])
	}

	annotations := parseStringMap(buildRuleAnnotations(rule, result, labels, "fingerprint-1"))
	if annotations["summary"] != "10.0.0.1:9100 CPU 高于 90" {
		t.Fatalf("expected rendered annotation summary, got %q", annotations["summary"])
	}
	if annotations["value"] != "95.2" {
		t.Fatalf("expected rendered annotation value, got %q", annotations["value"])
	}
	if annotations["watchValue"] != "95.2" {
		t.Fatalf("expected WatchAlert-style labels.value to render, got %q", annotations["watchValue"])
	}
	if !strings.Contains(annotations["labelText"], "instance=10.0.0.1:9100") || !strings.Contains(annotations["labelText"], "value=95.2") {
		t.Fatalf("expected WatchAlert-style labels map to render, got %q", annotations["labelText"])
	}
	if annotations["description"] != "节点：10.0.0.1:9100，CPU 使用率过高，当前：95.2%，旧变量：95.2%，阈值：90%" {
		t.Fatalf("expected rendered alert detail, got %q", annotations["description"])
	}
	if noticeAnnotationText("", annotations) != annotations["description"] {
		t.Fatalf("expected notice annotation text to prefer alert detail, got %q", noticeAnnotationText("", annotations))
	}
	if replaced := replaceNoticeAnnotationVariables("事件：${annotations.summary}", annotations, false); replaced != "事件："+annotations["summary"] {
		t.Fatalf("expected notice annotation variable replacement, got %q", replaced)
	}
	if escaped := noticeTemplateValue("第一行\n第二行", true); escaped != `第一行\n第二行` {
		t.Fatalf("expected json-safe template value, got %q", escaped)
	}

	event := model.AlertEvent{
		RuleID:         rule.ID,
		ID:             99,
		RuleName:       rule.Name,
		FaultCenterID:  3,
		DataSourceName: result.DataSourceName,
		DataSourceType: result.DataSourceType,
		Severity:       result.Severity,
		Value:          result.Value,
		Condition:      result.Condition,
		Threshold:      result.Threshold,
		Message:        result.Message,
		Labels:         marshalStringMap(labels),
		Annotations:    marshalStringMap(annotations),
		Fingerprint:    "fingerprint-1",
		LastEvalAt:     result.EvaluatedAt,
	}
	query := renderAlertCallbackQuery(`up{instance="${labels.instance}"} # ${annotations.summary} ${fingerprint}`, *rule, event)
	for _, expected := range []string{`up{instance="10.0.0.1:9100"}`, "10.0.0.1:9100 CPU 高于 90", "fingerprint-1"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("expected rendered callback query to contain %q, got %q", expected, query)
		}
	}

	payload := buildRuleNotificationPayload(&event)
	eventURL := buildNoticeEventURL(payload)
	if !strings.Contains(eventURL, "/monitor/fault-centers/3") || !strings.Contains(eventURL, "eventId=99") {
		t.Fatalf("expected notice event URL to include fault center and event id, got %q", eventURL)
	}
	callbacks := notificationCallbackContext{
		Items: []notificationCallbackItem{{
			Name:      "CPU 曲线",
			QueryMode: "range",
			Status:    "success",
			ValueText: "95.2",
		}},
	}
	callbacks.Summary = formatCallbackSummary(callbacks.Items)
	callbacks.DetailText = formatCallbackDetailText(callbacks.Items, eventURL)
	route := noticeObjectRouteConfig{NoticeType: "FeiShu"}
	text := (&DataSourceHandler{}).renderNoticeRouteText(
		model.NoticeObject{},
		route,
		payload,
		callbacks,
	)
	for _, expected := range []string{"CPU 曲线=95.2", "查看事件回调查询"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected rendered notice text to include %q, got %q", expected, text)
		}
	}
}

func TestShouldSendRecoveryNotificationOnlyForNotifiedFiringEvents(t *testing.T) {
	cases := []struct {
		name  string
		event *model.AlertEvent
		want  bool
	}{
		{name: "nil event", event: nil, want: false},
		{name: "pending with successful notification status", event: &model.AlertEvent{State: "pending", NotifyStatus: "success"}, want: false},
		{name: "silenced with successful notification status", event: &model.AlertEvent{State: "silenced", NotifyStatus: "success"}, want: false},
		{name: "processing with successful notification status", event: &model.AlertEvent{State: "processing", NotifyStatus: "success"}, want: true},
		{name: "firing without notification", event: &model.AlertEvent{State: "firing", NotifyStatus: "none"}, want: false},
		{name: "firing with failed notification", event: &model.AlertEvent{State: "firing", NotifyStatus: "failed"}, want: false},
		{name: "firing with successful notification", event: &model.AlertEvent{State: "firing", NotifyStatus: "success"}, want: true},
		{name: "firing with partial notification", event: &model.AlertEvent{State: "firing", NotifyStatus: "partial"}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSendRecoveryNotification(tc.event); got != tc.want {
				t.Fatalf("shouldSendRecoveryNotification() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRecoveryWaitRemainingStartsWhenEventFirstRecovers(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.Local)
	wait := 30 * time.Second

	if remaining := recoveryWaitRemainingForDuration(&model.AlertEvent{State: "firing", LastEvalAt: now.Add(-5 * time.Minute)}, now, wait); remaining != wait {
		t.Fatalf("expected first recovery to wait the full window, got %s", remaining)
	}
	if remaining := recoveryWaitRemainingForDuration(&model.AlertEvent{State: "recovering", LastEvalAt: now.Add(-10 * time.Second)}, now, wait); remaining != 20*time.Second {
		t.Fatalf("expected recovering event to keep the remaining window, got %s", remaining)
	}
	if remaining := recoveryWaitRemainingForDuration(&model.AlertEvent{State: "recovering", LastEvalAt: now.Add(-35 * time.Second)}, now, wait); remaining != 0 {
		t.Fatalf("expected recovering event past the wait window to be ready, got %s", remaining)
	}
	if remaining := recoveryWaitRemainingForDuration(&model.AlertEvent{State: "firing", LastEvalAt: now}, now, 0); remaining != 0 {
		t.Fatalf("expected zero wait to recover immediately, got %s", remaining)
	}
}

func TestLokiMatchedLogsAnnotationsAndNoticeVariables(t *testing.T) {
	raw := map[string]interface{}{
		"data": map[string]interface{}{
			"resultType": "streams",
			"result": []interface{}{
				map[string]interface{}{
					"stream": map[string]interface{}{
						"app": "sage",
						"pod": "sage-0",
					},
					"values": []interface{}{
						[]interface{}{"1781130000000000000", `{"level":"error","message":"sage接口报错","trace_id":"abc"}`},
						[]interface{}{"1781130001000000000", `ERROR sage接口报错 request_id=def`},
					},
				},
			},
		},
	}
	samples, err := extractRuleEvaluationSamples("loki", raw)
	if err != nil {
		t.Fatalf("extract loki samples: %v", err)
	}
	if len(samples) != 1 || samples[0].Value != 2 || samples[0].MatchedLogCount != 2 || len(samples[0].MatchedLogs) != 2 {
		t.Fatalf("expected one loki sample with two matched logs, got %#v", samples)
	}
	if samples[0].Labels["pod"] != "sage-0" || !strings.Contains(samples[0].MatchedLogs[0].Line, "trace_id") {
		t.Fatalf("expected stream labels and log line to be retained, got %#v", samples[0])
	}

	rule := &model.AlertRule{
		ID:             18,
		Name:           "sage接口报错",
		Query:          `sum by (pod) (count_over_time({app="sage"} |= "ERROR" [1m]))`,
		DataSourceType: "loki",
		DetailTemplate: "最近 ${matched_log_count} 行日志命中：\n${matched_logs}",
	}
	result := &alertRuleEvaluationResult{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		DataSourceName:  "Loki",
		DataSourceType:  "loki",
		Severity:        "p1",
		State:           "firing",
		Value:           samples[0].Value,
		Condition:       "gt",
		Threshold:       0,
		Labels:          samples[0].Labels,
		MatchedLogs:     samples[0].MatchedLogs,
		MatchedLogCount: samples[0].MatchedLogCount,
		MatchedLogQuery: deriveLokiMatchedLogQuery(rule.Query),
		Message:         "日志命中",
		EvaluatedAt:     time.Date(2026, 6, 11, 10, 0, 0, 0, time.Local),
	}
	labels := buildRuleLabelMap(rule, result)
	annotations := parseStringMap(buildRuleAnnotations(rule, result, labels, "fp-loki"))
	if annotations["matched_log_count"] != "2" {
		t.Fatalf("expected matched log count annotation, got %#v", annotations)
	}
	if !strings.Contains(annotations["matched_logs"], "sage接口报错") || !strings.Contains(annotations["description"], "最近 2 行日志命中") {
		t.Fatalf("expected matched logs in annotations and detail, got %#v", annotations)
	}
	if annotations["matched_log_query"] != `{app="sage"} |= "ERROR"` {
		t.Fatalf("expected derived log query, got %q", annotations["matched_log_query"])
	}
	if annotationText := noticeAnnotationText(marshalStringMap(annotations), annotations); !strings.Contains(annotationText, "```text\n") || !strings.Contains(annotationText, "sage接口报错") {
		t.Fatalf("expected matched logs in annotation text to be rendered as a code block, got %q", annotationText)
	}

	event := model.AlertEvent{
		ID:             180,
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		FaultCenterID:  9,
		DataSourceName: "Loki",
		DataSourceType: "loki",
		Severity:       "p1",
		State:          "firing",
		Value:          2,
		Condition:      "gt",
		Threshold:      0,
		Message:        "日志命中",
		Labels:         marshalStringMap(labels),
		Annotations:    marshalStringMap(annotations),
		Fingerprint:    "fp-loki",
		StartedAt:      result.EvaluatedAt,
		LastEvalAt:     result.EvaluatedAt,
	}
	payload := buildRuleNotificationPayload(&event)
	text := (&DataSourceHandler{}).renderNoticeRouteText(
		model.NoticeObject{},
		noticeObjectRouteConfig{NoticeType: "DingTalk"},
		payload,
		notificationCallbackContext{},
	)
	for _, expected := range []string{"命中日志", "sage接口报错"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected notice text to contain %q, got %s", expected, text)
		}
	}
	if replaced := replaceNoticeAnnotationVariables("查询：${annotations.matched_log_query}", annotations, false); replaced != `查询：{app="sage"} |= "ERROR"` {
		t.Fatalf("expected matched log query annotation replacement, got %q", replaced)
	}
}

func TestLokiMatchedLogLookbackUsesLogQLRange(t *testing.T) {
	rule := &model.AlertRule{
		Query:            `sum(count_over_time({project="bibf",env="prod"} |= "### Error updating database" [100h]))`,
		ForSeconds:       20,
		EvaluateInterval: 10,
	}

	if got, want := lokiMatchedLogLookbackSeconds(rule), 100*60*60; got != want {
		t.Fatalf("expected Loki matched log lookback to follow LogQL range, got %d want %d", got, want)
	}

	rule.Query = `sum(count_over_time({project="bibf",env="prod"} |= "ERROR" [30d]))`
	if got, want := lokiMatchedLogLookbackSeconds(rule), maxLokiMatchedLogLookbackSeconds; got != want {
		t.Fatalf("expected Loki matched log lookback to be capped, got %d want %d", got, want)
	}
}

func TestLegacyMarkdownNoticeAppendsMatchedLogsBlock(t *testing.T) {
	logs := "2026-06-11 10:40:10 ERROR sage接口报错\n2026-06-11 10:40:11 ERROR 查询文章详情失败"
	text := appendMatchedLogsBlockIfMissing("### OpsHub 告警中\n\n> 事件说明：日志过多", logs, noticeMatchedLogsBlock(logs), false)
	if !strings.Contains(text, "命中日志：") || !strings.Contains(text, "```text\n") || !strings.Contains(text, "sage接口报错") {
		t.Fatalf("expected legacy markdown text to append matched logs block, got %q", text)
	}
	text = appendMatchedLogsBlockIfMissing(text, logs, noticeMatchedLogsBlock(logs), false)
	if strings.Count(text, "命中日志：") != 1 {
		t.Fatalf("expected matched logs block not to be duplicated, got %q", text)
	}
}

func TestAggregatedNotificationPreservesMatchedLogs(t *testing.T) {
	startedAt := time.Date(2026, 6, 11, 10, 40, 24, 0, time.Local)
	lastEvalAt := startedAt.Add(20 * time.Second)
	annotations := marshalStringMap(map[string]string{
		"description":       "platform-website-biz-prod ERROR 日志过多",
		"matched_log_count": "2",
		"matched_log_query": `{app="platform-website-biz-prod"} |= "ERROR"`,
		"matched_logs":      "2026-06-11 10:40:10 ERROR 1 --- trace_id=abc 查询文章详情失败\n2026-06-11 10:40:11 ERROR 1 --- trace_id=def 查询文章详情失败",
	})
	events := []*model.AlertEvent{
		{
			ID:             31,
			RuleID:         28,
			RuleName:       "loki告警",
			FaultCenterID:  5,
			DataSourceID:   2,
			DataSourceName: "Loki",
			DataSourceType: "loki",
			Severity:       "p1",
			State:          "firing",
			Value:          113,
			Condition:      "gt",
			Threshold:      100,
			Labels:         `{"pod":"platform-website-biz-prod-5679797474-7l46c"}`,
			Annotations:    annotations,
			Fingerprint:    "fp-loki-1",
			Message:        "Loki 日志告警",
			StartedAt:      startedAt,
			LastEvalAt:     lastEvalAt,
		},
		{
			ID:             32,
			RuleID:         28,
			RuleName:       "loki告警",
			FaultCenterID:  5,
			DataSourceID:   2,
			DataSourceName: "Loki",
			DataSourceType: "loki",
			Severity:       "p1",
			State:          "firing",
			Value:          118,
			Condition:      "gt",
			Threshold:      100,
			Labels:         `{"pod":"platform-website-biz-prod-5679797474-n9fq7"}`,
			Annotations:    annotations,
			Fingerprint:    "fp-loki-2",
			Message:        "Loki 日志告警",
			StartedAt:      startedAt,
			LastEvalAt:     lastEvalAt,
		},
	}
	payload := buildAggregatedRuleNotificationPayload(events)
	renderedAnnotations := parseStringMap(payload.Annotations)
	if renderedAnnotations["matched_logs"] == "" || renderedAnnotations["matched_log_query"] == "" || renderedAnnotations["matched_log_count"] != "2" {
		t.Fatalf("expected aggregated payload to preserve matched log annotations, got %#v", renderedAnnotations)
	}
	text := (&DataSourceHandler{}).renderNoticeRouteText(
		model.NoticeObject{},
		noticeObjectRouteConfig{NoticeType: "DingDing"},
		payload,
		notificationCallbackContext{},
	)
	if !strings.Contains(text, "命中日志：") || !strings.Contains(text, "trace_id=abc") {
		t.Fatalf("expected aggregated DingTalk text to include matched logs, got %s", text)
	}
}

func TestRecoveredAnnotationsCarryPreviousMatchedLogs(t *testing.T) {
	previous := marshalStringMap(map[string]string{
		"description":       "触发时详情",
		"matched_log_count": "2",
		"matched_log_query": `{app="sage"} |= "ERROR"`,
		"matched_logs":      "2026-06-11 ERROR sage接口报错",
	})
	next := marshalStringMap(map[string]string{
		"description": "恢复详情\n最近命中日志：",
	})
	merged := parseStringMap(mergeRecoveredMatchedLogAnnotations(previous, next))
	if merged["matched_logs"] == "" || merged["matched_log_query"] == "" || merged["matched_log_count"] != "2" {
		t.Fatalf("expected recovered annotations to carry previous matched logs, got %#v", merged)
	}
}

func TestExtractRuleEvaluationSamplesEvaluatesEveryPrometheusSeries(t *testing.T) {
	raw := map[string]interface{}{
		"data": map[string]interface{}{
			"resultType": "vector",
			"result": []interface{}{
				map[string]interface{}{
					"metric": map[string]interface{}{"instance": "10.0.0.1:9100", "job": "node"},
					"value":  []interface{}{float64(1710000000), "2.89"},
				},
				map[string]interface{}{
					"metric": map[string]interface{}{"instance": "10.0.0.2:9100", "job": "node"},
					"value":  []interface{}{float64(1710000000), "9.96"},
				},
				map[string]interface{}{
					"metric": map[string]interface{}{"instance": "10.0.0.3:9100", "job": "node"},
					"value":  []interface{}{float64(1710000000), "4.58"},
				},
			},
		},
	}
	rule := &model.AlertRule{
		Name:       "CPU 使用率过低",
		Condition:  "lt",
		Threshold:  5,
		Severity:   "p1",
		ForSeconds: 60,
	}

	samples, err := extractRuleEvaluationSamples("prometheus", raw)
	if err != nil {
		t.Fatalf("extract samples: %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("expected every Prometheus series to become a sample, got %d", len(samples))
	}

	matchedByInstance := map[string]bool{}
	for _, sample := range samples {
		condition := selectSeverityCondition(rule, sample.Value)
		matchedByInstance[sample.Labels["instance"]] = condition.Matched
	}
	if !matchedByInstance["10.0.0.1:9100"] || !matchedByInstance["10.0.0.3:9100"] {
		t.Fatalf("expected values below threshold to match independently, got %#v", matchedByInstance)
	}
	if matchedByInstance["10.0.0.2:9100"] {
		t.Fatalf("expected value above threshold not to match, got %#v", matchedByInstance)
	}
}

func TestSelectSeverityConditionMatchesEqualThreshold(t *testing.T) {
	rule := &model.AlertRule{
		Name:       "Nginx 服务异常",
		Condition:  "eq",
		Threshold:  1,
		Severity:   "p1",
		ForSeconds: 60,
	}

	condition := selectSeverityCondition(rule, 1)
	if !condition.Matched {
		t.Fatalf("expected value equal to threshold to match, got %#v", condition)
	}
	if condition.Condition != "eq" || condition.Threshold != 1 {
		t.Fatalf("expected eq condition to be preserved, got %#v", condition)
	}
}

func TestExtractRuleEvaluationSamplesAllowsEmptyPrometheusVector(t *testing.T) {
	raw := map[string]interface{}{
		"data": map[string]interface{}{
			"resultType": "vector",
			"result":     []interface{}{},
		},
	}

	samples, err := extractRuleEvaluationSamples("prometheus", raw)
	if err != nil {
		t.Fatalf("expected empty vector to be a valid inactive evaluation, got %v", err)
	}
	if len(samples) != 0 {
		t.Fatalf("expected no samples for empty vector, got %d", len(samples))
	}
}

func TestBuildAlertFingerprintUsesSeriesLabels(t *testing.T) {
	first := buildAlertFingerprint(7, "CPU 使用率", "p1", map[string]string{
		"instance": "10.0.0.1:9100",
		"job":      "node",
	})
	sameDifferentOrder := buildAlertFingerprint(7, "CPU 使用率", "p1", map[string]string{
		"job":      "node",
		"instance": "10.0.0.1:9100",
	})
	otherInstance := buildAlertFingerprint(7, "CPU 使用率", "p1", map[string]string{
		"instance": "10.0.0.2:9100",
		"job":      "node",
	})
	otherSeverity := buildAlertFingerprint(7, "CPU 使用率", "p0", map[string]string{
		"instance": "10.0.0.1:9100",
		"job":      "node",
	})
	otherValue := buildAlertFingerprint(7, "CPU 使用率", "p1", map[string]string{
		"instance": "10.0.0.1:9100",
		"job":      "node",
		"value":    "99.9",
	})

	if first != sameDifferentOrder {
		t.Fatalf("expected fingerprint to be stable regardless of label order")
	}
	if first == otherInstance {
		t.Fatalf("expected different instance labels to produce different fingerprints")
	}
	if first == otherSeverity {
		t.Fatalf("expected severity to participate in fingerprint like WatchAlert")
	}
	if first != otherValue {
		t.Fatalf("expected dynamic labels.value to be ignored by fingerprint")
	}
}

func TestBuildAggregatedRuleNotificationPayload(t *testing.T) {
	startedAt := time.Date(2026, 6, 9, 13, 57, 46, 0, time.Local)
	lastEvalAt := startedAt.Add(10 * time.Second)
	events := []*model.AlertEvent{
		{
			ID:             11,
			RuleID:         7,
			RuleName:       "testsss",
			FaultCenterID:  3,
			DataSourceID:   1,
			DataSourceName: "Prometheus",
			DataSourceType: "prometheus",
			Severity:       "p0",
			State:          "firing",
			Value:          2.824,
			Condition:      "lt",
			Threshold:      5,
			Labels:         `{"instance":"10.122.28.12","job":"node"}`,
			Annotations:    `{"description":"生产主机 10.122.28.12 CPU 持续高于阈值"}`,
			Fingerprint:    "fp-12",
			Message:        "CPU 告警",
			StartedAt:      startedAt,
			LastEvalAt:     lastEvalAt,
		},
		{
			ID:             12,
			RuleID:         7,
			RuleName:       "testsss",
			FaultCenterID:  3,
			DataSourceID:   1,
			DataSourceName: "Prometheus",
			DataSourceType: "prometheus",
			Severity:       "p0",
			State:          "firing",
			Value:          4.466,
			Condition:      "lt",
			Threshold:      5,
			Labels:         `{"instance":"10.122.28.22","job":"node"}`,
			Annotations:    `{"description":"生产主机 10.122.28.22 CPU 持续高于阈值"}`,
			Fingerprint:    "fp-22",
			Message:        "CPU 告警",
			StartedAt:      startedAt,
			LastEvalAt:     lastEvalAt,
		},
	}

	payload := buildAggregatedRuleNotificationPayload(events)
	if !payload.Aggregated || payload.EventCount != 2 {
		t.Fatalf("expected aggregated payload with 2 events, got aggregated=%v count=%d", payload.Aggregated, payload.EventCount)
	}
	labels := parseStringMap(payload.Labels)
	if labels["instance"] != "10.122.28.12" {
		t.Fatalf("expected representative instance label, got %q", labels["instance"])
	}
	if labels["instances"] != "10.122.28.12、10.122.28.22" {
		t.Fatalf("expected aggregated instances helper label, got %q", labels["instances"])
	}
	annotations := parseStringMap(payload.Annotations)
	if !strings.Contains(annotations["description"], "10.122.28.12") || !strings.Contains(annotations["description"], "聚合 2 条消息") {
		t.Fatalf("expected representative annotation with WatchAlert-style aggregation notice, got %q", annotations["description"])
	}
	if strings.Contains(annotations["description"], "10.122.28.22 CPU 持续高于阈值") {
		t.Fatalf("expected aggregated notification not to expand every event detail, got %q", annotations["description"])
	}
	if payload.Fingerprint != "fp-12" || payload.Value != 2.824 {
		t.Fatalf("expected representative fingerprint/value, got fingerprint=%q value=%v", payload.Fingerprint, payload.Value)
	}
	if text := noticeEventLinkText(payload); text != "查看活跃告警" {
		t.Fatalf("expected aggregated firing link text to point at active alerts, got %q", text)
	}
	if url := buildNoticeEventURL(payload); strings.Contains(url, "eventId=") || !strings.Contains(url, "tab=active") || !strings.Contains(url, "query=testsss") {
		t.Fatalf("expected aggregated event URL to jump to active alert list without event id, got %q", url)
	}
}

func TestAggregatedCallbackContextDoesNotAppendDuplicateEventLink(t *testing.T) {
	items := []notificationCallbackItem{{
		Name:      "CPU 使用率曲线",
		QueryMode: "range",
		Status:    "success",
		ValueText: "2.82",
	}}
	text := fmt.Sprintf("- **10.122.28.12**：%s", formatCallbackItemsInline(items))
	if strings.Contains(text, "查看事件") || strings.Contains(text, "/monitor/fault-centers/") {
		t.Fatalf("expected aggregated callback text not to include event link, got %q", text)
	}
}

func TestFaultCenterRepeatNoticeInterval(t *testing.T) {
	raw := `{"p0":30,"p1":"120","p2":360,"critical":45}`

	interval, ok := faultCenterRepeatNoticeInterval(raw, "p0")
	if !ok || interval != 30*time.Minute {
		t.Fatalf("expected P0 interval 30m, got %s, ok=%v", interval, ok)
	}

	interval, ok = faultCenterRepeatNoticeInterval(raw, "warning")
	if !ok || interval != 120*time.Minute {
		t.Fatalf("expected P1/warning interval 120m, got %s, ok=%v", interval, ok)
	}

	_, ok = faultCenterRepeatNoticeInterval(`{"p0":0}`, "p0")
	if ok {
		t.Fatalf("expected zero interval to be ignored")
	}
}

func TestClearSilenceAcknowledgement(t *testing.T) {
	ackAt := time.Date(2026, 6, 9, 16, 25, 44, 0, time.Local)
	event := &model.AlertEvent{
		Acknowledged:   true,
		AcknowledgedBy: "Codex P0 静默测试",
		AcknowledgedAt: &ackAt,
	}

	clearSilenceAcknowledgement(event)
	if event.Acknowledged || event.AcknowledgedBy != "" || event.AcknowledgedAt != nil {
		t.Fatalf("expected silence acknowledgement fields to be cleared, got acknowledged=%v by=%q at=%v", event.Acknowledged, event.AcknowledgedBy, event.AcknowledgedAt)
	}
}

func TestSplitPrometheusAlertExpression(t *testing.T) {
	query, condition, threshold, ok := splitPrometheusAlertExpression(`up{job!="blackbox"} == 0`)
	if !ok || query != `up{job!="blackbox"}` || condition != "eq" || threshold != 0 {
		t.Fatalf("expected expression to split outside label matcher, got query=%q condition=%q threshold=%v ok=%v", query, condition, threshold, ok)
	}

	query, condition, threshold, ok = splitPrometheusAlertExpression(`(1 - avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100 < 5`)
	if !ok || !strings.Contains(query, "node_cpu_seconds_total") || condition != "lt" || threshold != 5 {
		t.Fatalf("expected CPU expression to split into lt 5, got query=%q condition=%q threshold=%v ok=%v", query, condition, threshold, ok)
	}

	query, condition, threshold, ok = splitPrometheusAlertExpression(`sum(rate(http_requests_total[5m]))`)
	if ok || query != `sum(rate(http_requests_total[5m]))` || condition != "gt" || threshold != 0 {
		t.Fatalf("expected expression without comparison to fall back to gt 0, got query=%q condition=%q threshold=%v ok=%v", query, condition, threshold, ok)
	}
}

func TestParsePrometheusDurationSeconds(t *testing.T) {
	if got := parsePrometheusDurationSeconds("1h30m"); got != 5400 {
		t.Fatalf("expected 1h30m to be 5400s, got %d", got)
	}
	if got := parsePrometheusDurationSeconds("50w"); got != 50*7*24*3600 {
		t.Fatalf("expected 50w to be converted to seconds, got %d", got)
	}
}

func TestNormalizePrometheusRuleSeverityMapsSeriousToP1(t *testing.T) {
	if got := normalizePrometheusRuleSeverity("serious"); got != "p1" {
		t.Fatalf("expected serious severity to map to p1, got %q", got)
	}
}

func TestNoticeDutyUserFeishuAtText(t *testing.T) {
	object := model.NoticeObject{
		CurrentDutyUsers: []model.DutyUser{
			{RealName: "杜杰", NotifyUserID: "ou_123"},
			{RealName: "张强"},
			{RealName: "王伟", NotifyUserID: "7643995081296350389"},
		},
	}

	got := noticeDutyUserFeishuAtText(object)
	want := `<at id="ou_123">杜杰</at> @张强 @王伟`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFeishuRouteUsesWatchAlertDutyUserVariable(t *testing.T) {
	object := model.NoticeObject{
		CurrentDutyUsers: []model.DutyUser{
			{RealName: "杜杰", NotifyUserID: "7643995081296350389"},
		},
	}

	text := noticeRouteDutyUserText(object, "FeiShu")
	want := `@杜杰`
	if text != want {
		t.Fatalf("expected FeiShu ${duty_user} to render as %q, got %q", want, text)
	}
	if name := noticeRouteDutyUserText(object, "DingDing"); name != "杜杰" {
		t.Fatalf("expected non-FeiShu ${duty_user} to stay display name, got %q", name)
	}
}

func TestBuildFeishuWebhookPayloadAndMarshalKeepAtTag(t *testing.T) {
	card := map[string]interface{}{
		"schema": "2.0",
		"body": map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{"tag": "markdown", "content": `**值班人员:** <at id="7643995081296350389">杜杰</at>`},
			},
		},
	}

	payload := buildFeishuWebhookPayload(card)
	if payload["msg_type"] != "interactive" {
		t.Fatalf("expected interactive payload, got %#v", payload)
	}
	if _, ok := payload["card"]; !ok {
		t.Fatalf("expected card wrapper, got %#v", payload)
	}

	body, err := marshalJSONBody(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	bodyText := string(body)
	if !strings.Contains(bodyText, `<at id=\"7643995081296350389\">杜杰</at>`) {
		t.Fatalf("expected raw at tag in JSON body, got %s", bodyText)
	}
	if strings.Contains(bodyText, `\u003c`) || strings.Contains(bodyText, `\u003e`) {
		t.Fatalf("expected JSON body not to html-escape at tag, got %s", bodyText)
	}

	fullMessage := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}
	wrapped := buildFeishuWebhookPayload(fullMessage)
	data, _ := json.Marshal(wrapped)
	if strings.Count(string(data), "msg_type") != 1 {
		t.Fatalf("expected full FeiShu message not to be wrapped twice, got %s", string(data))
	}
}

func TestWebhookBusinessResponseError(t *testing.T) {
	if err := webhookBusinessResponseError([]byte(`{"StatusCode":0,"StatusMessage":"success"}`)); err != nil {
		t.Fatalf("expected Feishu success response to pass, got %v", err)
	}
	err := webhookBusinessResponseError([]byte(`{"code":9499,"msg":"bad card"}`))
	if err == nil || !strings.Contains(err.Error(), "9499") || !strings.Contains(err.Error(), "bad card") {
		t.Fatalf("expected business error to include code and message, got %v", err)
	}
	err = webhookBusinessResponseError([]byte(`{"errcode":310000,"errmsg":"keywords not in content"}`))
	if err == nil || !strings.Contains(err.Error(), "关键词安全设置") {
		t.Fatalf("expected DingTalk keyword error to include friendly hint, got %v", err)
	}
}

func TestBuildDingTalkWebhookURLWithSecret(t *testing.T) {
	now := time.Unix(1710000000, 123000000)
	secret := "SEC0123456789"
	rawURL := "https://oapi.dingtalk.com/robot/send?access_token=token"
	signedURL, err := buildDingTalkWebhookURL(rawURL, secret, now)
	if err != nil {
		t.Fatalf("expected signed url, got error: %v", err)
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatalf("expected valid url, got %v", err)
	}
	query := parsed.Query()
	timestamp := query.Get("timestamp")
	if timestamp != "1710000000123" {
		t.Fatalf("expected millisecond timestamp, got %q", timestamp)
	}
	if query.Get("access_token") != "token" {
		t.Fatalf("expected original access token to be preserved")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	expectedSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if query.Get("sign") != expectedSign {
		t.Fatalf("expected sign %q, got %q", expectedSign, query.Get("sign"))
	}
}

func TestBuildDingTalkWebhookURLWithoutSecretKeepsURL(t *testing.T) {
	rawURL := "https://oapi.dingtalk.com/robot/send?access_token=token"
	got, err := buildDingTalkWebhookURL(rawURL, "", time.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != rawURL {
		t.Fatalf("expected unsigned url to stay unchanged, got %q", got)
	}
}

func TestBuildNoticeEmailBodyKeepsHTMLFragment(t *testing.T) {
	body := buildNoticeEmailBody(`<div><h2>OpsHub 告警中</h2><p>CPU 使用率过高</p></div>`)
	if !strings.Contains(body, "<!doctype html>") {
		t.Fatalf("expected html fragment to be wrapped as full email html, got %q", body)
	}
	if !strings.Contains(body, `<div><h2>OpsHub 告警中</h2><p>CPU 使用率过高</p></div>`) {
		t.Fatalf("expected html fragment to stay unescaped, got %q", body)
	}
	if strings.Contains(body, "&lt;div") || strings.Contains(body, "&lt;h2") {
		t.Fatalf("expected html fragment not to be escaped, got %q", body)
	}
}

func TestBuildNoticeEmailBodyEscapesPlainText(t *testing.T) {
	body := buildNoticeEmailBody("第一行\n<script>alert(1)</script>")
	if !strings.Contains(body, "第一行<br>") {
		t.Fatalf("expected newlines in plain text to become br tags, got %q", body)
	}
	if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("expected plain text html to be escaped, got %q", body)
	}
}

func TestFormatDataSourceResponseErrorExtractsQueryError(t *testing.T) {
	err := formatDataSourceResponseError(422, []byte(`{"status":"error","errorType":"bad_data","error":"invalid parameter query: parse error"}`))
	if err == nil {
		t.Fatalf("expected query error")
	}
	text := err.Error()
	for _, expected := range []string{"HTTP 422", "bad_data", "parse error"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected datasource error to contain %q, got %q", expected, text)
		}
	}
	if strings.Contains(text, `{"status"`) {
		t.Fatalf("expected datasource error to be user-readable instead of raw json, got %q", text)
	}
}

func TestBuildFeishuCallbackImageElementsShowsUploadWarning(t *testing.T) {
	elements := buildFeishuCallbackImageElements(notificationCallbackContext{
		Images:        []notificationCallbackImage{{Title: "CPU 使用率趋势", PNG: []byte{1, 2, 3}}},
		ImageWarnings: []string{"飞书上传图片失败：权限不足"},
	})
	if len(elements) == 0 {
		t.Fatalf("expected warning elements")
	}
	data, _ := json.Marshal(elements)
	text := string(data)
	for _, expected := range []string{"图表上传失败", "权限不足"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected warning card to contain %q, got %s", expected, text)
		}
	}
}

func TestRenderCallbackChartPNGIncludesMetricMeta(t *testing.T) {
	item := notificationCallbackItem{
		Key:           "cpu",
		Name:          "CPU 使用率趋势",
		RenderedQuery: `(1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100`,
	}
	unit := inferCallbackMetricUnit(item.Name + " " + item.RenderedQuery)
	if unit.Label != "%" {
		t.Fatalf("expected percent unit, got %q", unit.Label)
	}
	imageBytes, err := renderCallbackChartPNG([]callbackChartSeries{
		{
			Name: "192.168.10.111:9100",
			Points: []callbackChartPoint{
				{Time: 1781074800, Value: 72.5},
				{Time: 1781075100, Value: 76.2},
				{Time: 1781075400, Value: 96.26},
			},
		},
	}, item, model.AlertEvent{Condition: "gte", Threshold: 80})
	if err != nil {
		t.Fatalf("render chart png failed: %v", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		t.Fatalf("decode chart png failed: %v", err)
	}
	if config.Width != 940 || config.Height != 430 {
		t.Fatalf("expected compact chart image with meta area, got %dx%d", config.Width, config.Height)
	}
}

func TestFeishuUploadErrorUsesReadablePermissionHint(t *testing.T) {
	err := feishuAPIError("飞书上传图片失败", http.StatusBadRequest, []byte(`{"code":99991672,"msg":"Access denied. One of the following scopes is required: [im:resource:upload, im:resource]. please contact your administrator"}`))
	message := feishuUploadUserMessage(err)
	for _, expected := range []string{"应用身份", "im:resource", "用户身份权限"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected readable error to contain %q, got %q", expected, message)
		}
	}
	for _, unwanted := range []string{"https://open.feishu.cn/app", "troubleshooter", "Access denied"} {
		if strings.Contains(message, unwanted) {
			t.Fatalf("expected raw FeiShu error details to be hidden, got %q", message)
		}
	}
}

func TestFeishuIdentifierLooksResolvable(t *testing.T) {
	for _, id := range []string{"ou_123", "on_abc", "un_xyz"} {
		if !feishuIdentifierLooksResolvable(id) {
			t.Fatalf("expected %s to be treated as resolvable FeiShu identifier", id)
		}
	}
	if feishuIdentifierLooksResolvable("7643995081296350389") {
		t.Fatalf("expected pure numeric ID to fall back to display name for this webhook")
	}
}

func TestNoticeRouteSkipReasonExplainsSeverityMismatch(t *testing.T) {
	enabled := true
	route := noticeObjectRouteConfig{
		NoticeType: "DingDing",
		Severitys:  []string{"P0"},
		Enabled:    &enabled,
	}

	reason := noticeRouteSkipReason(route, "p1")
	for _, expected := range []string{"P1", "P0", "不在适用级别"} {
		if !strings.Contains(reason, expected) {
			t.Fatalf("expected skip reason to contain %q, got %q", expected, reason)
		}
	}
}

func TestNoticeRouteSkipReasonExplainsDisabledRoute(t *testing.T) {
	enabled := false
	route := noticeObjectRouteConfig{
		NoticeType: "DingDing",
		Severitys:  []string{"P0", "P1"},
		Enabled:    &enabled,
	}

	if reason := noticeRouteSkipReason(route, "p1"); reason != "通知路由已停用" {
		t.Fatalf("expected disabled route reason, got %q", reason)
	}
}

func TestGenericWebhookNoticeObjectIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("OPSHUB_INTEGRATION_DSN"))
	if dsn == "" {
		t.Skip("set OPSHUB_INTEGRATION_DSN to run the real MySQL generic webhook integration test")
	}

	type capturedRequest struct {
		Header string
		Body   string
	}
	received := make(chan capturedRequest, 1)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST webhook request, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		received <- capturedRequest{
			Header: r.Header.Get("X-OpsHub-Test"),
			Body:   string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer webhook.Close()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(
		&model.NoticeTemplate{},
		&model.NoticeObject{},
		&model.FaultCenter{},
		&model.AlertRule{},
		&model.AlertEvent{},
	); err != nil {
		t.Fatalf("auto migrate integration tables: %v", err)
	}

	prefix := fmt.Sprintf("codex-webhook-%d", time.Now().UnixNano())
	cleanup := func() {
		_ = db.Where("rule_name LIKE ?", prefix+"%").Delete(&model.AlertEvent{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.AlertRule{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.FaultCenter{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.NoticeObject{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.NoticeTemplate{}).Error
	}
	cleanup()
	defer cleanup()

	template := model.NoticeTemplate{
		Name:           prefix + "-template",
		NoticeType:     "WebHook",
		Description:    "generic webhook integration test template",
		TemplateFiring: `{"status":"firing","platform":"OpsHub","ruleName":"${rule_name}","severity":"${severity}","instance":"${labels.instance}","value":"{{value}}","eventId":"{{eventId}}","eventUrl":"${event_url}","annotations":"${annotations}"}`,
		Enabled:        true,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create webhook template: %v", err)
	}

	routeEnabled := true
	routes, _ := json.Marshal([]noticeObjectRouteConfig{{
		NoticeType:       "WebHook",
		NoticeTemplateID: template.ID,
		Severitys:        []string{"P1"},
		Hook:             webhook.URL,
		Headers:          map[string]string{"X-OpsHub-Test": "generic-webhook"},
		Enabled:          &routeEnabled,
	}})
	noticeObject := model.NoticeObject{
		UUID:        prefix + "-notice-object",
		Name:        prefix + "-notice-object",
		Description: "generic webhook integration test object",
		Routes:      string(routes),
		Enabled:     true,
		LastStatus:  "ready",
	}
	if err := db.Create(&noticeObject).Error; err != nil {
		t.Fatalf("create notice object: %v", err)
	}

	noticeObjectIDs, _ := json.Marshal([]uint{noticeObject.ID})
	center := model.FaultCenter{
		Name:                 prefix + "-fault-center",
		Description:          "generic webhook integration test center",
		NoticeObjectIDs:      string(noticeObjectIDs),
		RecoverNotify:        true,
		AggregationType:      "Rule",
		RepeatNoticeInterval: `{"p0":1,"p1":1,"p2":1}`,
	}
	if err := db.Create(&center).Error; err != nil {
		t.Fatalf("create fault center: %v", err)
	}

	rule := model.AlertRule{
		Name:             prefix + "-rule",
		FaultCenterID:    center.ID,
		DataSourceID:     1,
		DataSourceIDs:    `[1]`,
		DataSourceType:   "prometheus",
		Query:            "up == 0",
		QueryMode:        "instant",
		Condition:        "eq",
		Threshold:        0,
		Severity:         "p1",
		Enabled:          true,
		NotifyRecovery:   true,
		RepeatInterval:   3600,
		EvaluateInterval: 60,
		ForSeconds:       60,
		LastState:        "firing",
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create alert rule: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	event := model.AlertEvent{
		RuleID:         rule.ID,
		FaultCenterID:  center.ID,
		RuleName:       rule.Name,
		DataSourceID:   1,
		DataSourceName: "Prometheus",
		DataSourceType: "prometheus",
		Severity:       "p1",
		State:          "firing",
		Value:          1,
		Condition:      "eq",
		Threshold:      0,
		Message:        "generic webhook firing event",
		Labels:         `{"instance":"opshub-webhook-target"}`,
		Annotations:    `{"description":"通用 WebHook 集成测试事件"}`,
		Fingerprint:    prefix + "-fingerprint",
		StartedAt:      now.Add(-2 * time.Minute),
		LastEvalAt:     now,
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create alert event: %v", err)
	}

	handler := NewDataSourceHandler(db)
	if !handler.sendAndRecordRuleNotifications(context.Background(), &rule, []*model.AlertEvent{&event}) {
		t.Fatalf("expected generic webhook notification to be sent")
	}

	var captured capturedRequest
	select {
	case captured = <-received:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected generic webhook request to be captured")
	}
	if captured.Header != "generic-webhook" {
		t.Fatalf("expected custom header to be forwarded, got %q", captured.Header)
	}

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(captured.Body), &body); err != nil {
		t.Fatalf("expected rendered webhook body to be JSON, got %q: %v", captured.Body, err)
	}
	for key, expected := range map[string]string{
		"status":   "firing",
		"platform": "OpsHub",
		"ruleName": rule.Name,
		"severity": "P1",
		"instance": "opshub-webhook-target",
	} {
		if fmt.Sprint(body[key]) != expected {
			t.Fatalf("expected webhook body %s=%q, got %#v in %s", key, expected, body[key], captured.Body)
		}
	}

	var saved model.AlertEvent
	if err := db.First(&saved, event.ID).Error; err != nil {
		t.Fatalf("load saved webhook event: %v", err)
	}
	if saved.NotifyStatus != "success" || saved.NotifyError != "" {
		t.Fatalf("expected event notification success, got status=%q error=%q", saved.NotifyStatus, saved.NotifyError)
	}
}

func TestFaultCenterEscalationIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("OPSHUB_INTEGRATION_DSN"))
	if dsn == "" {
		t.Skip("set OPSHUB_INTEGRATION_DSN to run the real MySQL escalation integration test")
	}

	var webhookCount atomic.Int64
	receivedBodies := make(chan string, 8)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST webhook request, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		webhookCount.Add(1)
		receivedBodies <- string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer webhook.Close()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(
		&model.NoticeTemplate{},
		&model.NoticeObject{},
		&model.FaultCenter{},
		&model.AlertRule{},
		&model.AlertEvent{},
	); err != nil {
		t.Fatalf("auto migrate integration tables: %v", err)
	}

	prefix := fmt.Sprintf("codex-escalation-%d", time.Now().UnixNano())
	cleanup := func() {
		_ = db.Where("rule_name LIKE ?", prefix+"%").Delete(&model.AlertEvent{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.AlertRule{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.FaultCenter{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.NoticeObject{}).Error
		_ = db.Where("name LIKE ?", prefix+"%").Delete(&model.NoticeTemplate{}).Error
	}
	cleanup()
	defer cleanup()

	template := model.NoticeTemplate{
		Name:                 prefix + "-feishu-template",
		NoticeType:           "FeiShu",
		Description:          "integration test template",
		TemplateFiring:       `{"schema":"2.0","config":{"width_mode":"fill"},"header":{"template":"red","title":{"tag":"plain_text","content":"【告警升级】- OpsHub 集成测试"}},"body":{"elements":[{"tag":"markdown","content":"**规则:** ${rule_name}\n**升级:** ${escalation_text}\n**值:** ${value}\n[${event_link_text}](${event_url})"}]}}`,
		TemplateRecover:      `{"schema":"2.0","config":{"width_mode":"fill"},"header":{"template":"green","title":{"tag":"plain_text","content":"【已恢复】- OpsHub 集成测试"}},"body":{"elements":[{"tag":"markdown","content":"${rule_name} 已恢复"}]}}`,
		EnableFeiShuJSONCard: true,
		Enabled:              true,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create notice template: %v", err)
	}

	routeEnabled := true
	routes, _ := json.Marshal([]noticeObjectRouteConfig{{
		NoticeType:       "FeiShu",
		NoticeTemplateID: template.ID,
		Severitys:        []string{"P0"},
		Hook:             webhook.URL,
		Enabled:          &routeEnabled,
	}})
	noticeObject := model.NoticeObject{
		UUID:        prefix + "-notice-object",
		Name:        prefix + "-notice-object",
		Description: "integration test object",
		Routes:      string(routes),
		Enabled:     true,
		LastStatus:  "ready",
	}
	if err := db.Create(&noticeObject).Error; err != nil {
		t.Fatalf("create notice object: %v", err)
	}

	noticeObjectIDs, _ := json.Marshal([]uint{noticeObject.ID})
	upgradableSeverities, _ := json.Marshal([]string{"p0"})
	strategyEnabled := true
	upgradeStrategy, _ := json.Marshal(faultCenterUpgradeStrategyConfig{
		Enabled:         &strategyEnabled,
		Timeout:         1,
		RepeatInterval:  5,
		NoticeObjectIDs: []interface{}{float64(noticeObject.ID)},
	})
	center := model.FaultCenter{
		Name:                 prefix + "-fault-center",
		Description:          "integration test center",
		NoticeObjectIDs:      string(noticeObjectIDs),
		RecoverNotify:        true,
		AggregationType:      "Rule",
		UpgradeEnabled:       true,
		UpgradableSeverities: string(upgradableSeverities),
		UpgradeStrategy:      string(upgradeStrategy),
	}
	if err := db.Create(&center).Error; err != nil {
		t.Fatalf("create fault center: %v", err)
	}

	rule := model.AlertRule{
		Name:             prefix + "-rule",
		FaultCenterID:    center.ID,
		DataSourceID:     1,
		DataSourceIDs:    `[1]`,
		DataSourceType:   "prometheus",
		Query:            "up == 0",
		QueryMode:        "instant",
		Condition:        "eq",
		Threshold:        0,
		Severity:         "p0",
		Enabled:          true,
		NotifyRecovery:   true,
		RepeatInterval:   3600,
		EvaluateInterval: 60,
		ForSeconds:       60,
		DetailTemplate:   "集成测试告警详情",
		LastState:        "firing",
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create alert rule: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	event := model.AlertEvent{
		RuleID:         rule.ID,
		FaultCenterID:  center.ID,
		RuleName:       rule.Name,
		DataSourceID:   1,
		DataSourceName: "Prometheus",
		DataSourceType: "prometheus",
		Severity:       "p0",
		State:          "firing",
		Value:          0,
		Condition:      "eq",
		Threshold:      0,
		Message:        "integration firing event",
		Labels:         `{"instance":"127.0.0.1:9100"}`,
		Annotations:    `{"description":"集成测试事件"}`,
		Fingerprint:    prefix + "-fingerprint",
		StartedAt:      now.Add(-2 * time.Minute),
		LastEvalAt:     now,
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create alert event: %v", err)
	}

	handler := NewDataSourceHandler(db)
	handler.processFaultCenterEscalations(context.Background(), &rule, []*model.AlertEvent{&event}, now)
	if got := webhookCount.Load(); got != 1 {
		t.Fatalf("expected first escalation to send one webhook, got %d", got)
	}
	select {
	case body := <-receivedBodies:
		for _, expected := range []string{"msg_type", "告警升级", rule.Name, "告警已持续超过 1 分钟"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("expected webhook body to contain %q, got %s", expected, body)
			}
		}
	default:
		t.Fatalf("expected webhook body to be captured")
	}

	var saved model.AlertEvent
	if err := db.First(&saved, event.ID).Error; err != nil {
		t.Fatalf("load escalated event: %v", err)
	}
	if !saved.Escalated || saved.EscalatedAt == nil || saved.LastEscalateAt == nil || saved.EscalateStatus != "success" || saved.EscalateError != "" {
		t.Fatalf("expected successful escalation state, got escalated=%v escalatedAt=%v last=%v status=%q error=%q", saved.Escalated, saved.EscalatedAt, saved.LastEscalateAt, saved.EscalateStatus, saved.EscalateError)
	}

	handler.processFaultCenterEscalations(context.Background(), &rule, []*model.AlertEvent{&saved}, now.Add(time.Minute))
	if got := webhookCount.Load(); got != 1 {
		t.Fatalf("expected repeat interval to suppress second webhook, got %d", got)
	}

	lastEscalateAt := now.Add(-6 * time.Minute)
	if err := db.Model(&model.AlertEvent{}).Where("id = ?", saved.ID).Updates(map[string]interface{}{
		"last_escalate_at": lastEscalateAt,
	}).Error; err != nil {
		t.Fatalf("move last escalate time: %v", err)
	}
	if err := db.First(&saved, event.ID).Error; err != nil {
		t.Fatalf("reload event after repeat update: %v", err)
	}
	handler.processFaultCenterEscalations(context.Background(), &rule, []*model.AlertEvent{&saved}, now)
	if got := webhookCount.Load(); got != 2 {
		t.Fatalf("expected webhook after repeat interval elapsed, got %d", got)
	}
}
