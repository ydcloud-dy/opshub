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
	"strconv"
	"strings"
	"time"

	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	aiopsdata "github.com/ydcloud-dy/opshub/internal/data/aiops"
	monitormodel "github.com/ydcloud-dy/opshub/plugins/monitor/model"
)

type AlertAnalyzeRequest struct {
	AlertEventID uint   `json:"alertEventId"`
	Query        string `json:"query"`
	ProviderID   uint   `json:"providerId"`
}

type AlertEventSummary struct {
	ID             uint    `json:"id"`
	RuleID         uint    `json:"ruleId"`
	RuleName       string  `json:"ruleName"`
	DataSourceName string  `json:"dataSourceName"`
	DataSourceType string  `json:"dataSourceType"`
	Severity       string  `json:"severity"`
	State          string  `json:"state"`
	Value          float64 `json:"value"`
	Message        string  `json:"message"`
	Labels         string  `json:"labels"`
	StartedAt      string  `json:"startedAt"`
	LastEvalAt     string  `json:"lastEvalAt"`
}

func (s *Service) ListAlertEvents(ctx context.Context, page, pageSize int, state, severity string) ([]AlertEventSummary, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.WithContext(ctx).Model(&monitormodel.AlertEvent{})
	if strings.TrimSpace(state) != "" {
		query = query.Where("state = ?", strings.TrimSpace(state))
	}
	if strings.TrimSpace(severity) != "" {
		query = query.Where("severity IN ?", alertSeverityQueryValues(severity))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var events []monitormodel.AlertEvent
	if err := query.Order("last_eval_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	result := make([]AlertEventSummary, 0, len(events))
	for _, event := range events {
		result = append(result, summarizeAlertEvent(event))
	}
	return result, total, nil
}

func (s *Service) AnalyzeAlertRootCause(ctx context.Context, userID uint, username string, req AlertAnalyzeRequest) (*aiopsdata.RootCauseAnalysis, error) {
	evidence, err := s.collectAlertEvidence(ctx, req)
	if err != nil {
		return nil, err
	}
	evidenceJSON, _ := json.Marshal(evidence)
	prompt := fmt.Sprintf(`请对 OpsHub 监控告警做根因分析。
要求：
1. 输出告警摘要
2. 给出疑似根因并按可能性排序
3. 明确证据链，关联告警规则、最近事件和最近变更/操作
4. 给出只读排查命令和处置建议
5. 涉及重启、删除、扩缩容、配置修改时必须提示人工确认

用户补充：%s
告警证据 JSON：
%s`, defaultString(req.Query, "无"), truncateText(string(evidenceJSON), 28000))

	result, modelErr := s.callSelectedModel(ctx, req.ProviderID, []ChatMessage{
		{Role: "system", Content: diagnosisSystemPrompt()},
		{Role: "user", Content: prompt},
	})
	if modelErr != nil {
		result = fallbackAlertAnalysis(evidence, modelErr)
	}
	analysis := aiopsdata.RootCauseAnalysis{
		UserID:       userID,
		Username:     username,
		AlertEventID: uintFromAny(evidence["alertEventId"]),
		RuleID:       uintFromAny(evidence["ruleId"]),
		RuleName:     stringFromAny(evidence["ruleName"]),
		Severity:     stringFromAny(evidence["severity"]),
		State:        stringFromAny(evidence["state"]),
		Summary:      truncateText(stripMarkdown(result.Content), 1200),
		RootCause:    result.Content,
		EvidenceJSON: string(evidenceJSON),
		Suggestion:   extractSuggestions(result.Content),
		Model:        result.Model,
		Fallback:     result.Fallback,
		Status:       "success",
	}
	if err := s.db.WithContext(ctx).Create(&analysis).Error; err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (s *Service) ListRootCauseAnalyses(ctx context.Context, userID uint, page, pageSize int) ([]aiopsdata.RootCauseAnalysis, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.WithContext(ctx).Model(&aiopsdata.RootCauseAnalysis{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []aiopsdata.RootCauseAnalysis
	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (s *Service) collectAlertEvidence(ctx context.Context, req AlertAnalyzeRequest) (map[string]any, error) {
	evidence := map[string]any{"collectedAt": time.Now().Format(time.RFC3339), "query": req.Query}
	var event monitormodel.AlertEvent
	if req.AlertEventID > 0 {
		if err := s.db.WithContext(ctx).First(&event, req.AlertEventID).Error; err != nil {
			return nil, fmt.Errorf("获取告警事件失败: %w", err)
		}
		evidence["alertEventId"] = event.ID
		evidence["ruleId"] = event.RuleID
		evidence["ruleName"] = event.RuleName
		evidence["severity"] = event.Severity
		evidence["state"] = event.State
		evidence["event"] = summarizeAlertEvent(event)
	} else {
		var events []monitormodel.AlertEvent
		query := s.db.WithContext(ctx).Order("last_eval_at DESC").Limit(8)
		if strings.TrimSpace(req.Query) != "" {
			like := "%" + strings.TrimSpace(req.Query) + "%"
			query = query.Where("rule_name LIKE ? OR message LIKE ? OR labels LIKE ?", like, like, like)
		}
		if err := query.Find(&events).Error; err != nil {
			return nil, err
		}
		if len(events) > 0 {
			event = events[0]
			evidence["alertEventId"] = event.ID
			evidence["ruleId"] = event.RuleID
			evidence["ruleName"] = event.RuleName
			evidence["severity"] = event.Severity
			evidence["state"] = event.State
		}
		items := make([]AlertEventSummary, 0, len(events))
		for _, item := range events {
			items = append(items, summarizeAlertEvent(item))
		}
		evidence["matchedEvents"] = items
	}
	if event.RuleID > 0 {
		var rule monitormodel.AlertRule
		if err := s.db.WithContext(ctx).First(&rule, event.RuleID).Error; err == nil {
			evidence["rule"] = map[string]any{
				"id": rule.ID, "name": rule.Name, "query": rule.Query, "condition": rule.Condition,
				"threshold": rule.Threshold, "severity": rule.Severity, "lastState": rule.LastState,
				"lastValue": rule.LastValue, "lastError": rule.LastError,
			}
		}
		var related []monitormodel.AlertEvent
		_ = s.db.WithContext(ctx).Where("rule_id = ?", event.RuleID).Order("last_eval_at DESC").Limit(10).Find(&related).Error
		relatedSummaries := make([]AlertEventSummary, 0, len(related))
		for _, item := range related {
			relatedSummaries = append(relatedSummaries, summarizeAlertEvent(item))
		}
		evidence["relatedEvents"] = relatedSummaries
	}
	var ops []auditmodel.SysOperationLog
	_ = s.db.WithContext(ctx).Where("created_at >= ?", time.Now().Add(-24*time.Hour)).Order("id DESC").Limit(20).Find(&ops).Error
	evidence["recentOperations"] = summarizeOperationLogs(ops)
	return evidence, nil
}

func summarizeAlertEvent(event monitormodel.AlertEvent) AlertEventSummary {
	return AlertEventSummary{
		ID: event.ID, RuleID: event.RuleID, RuleName: event.RuleName, DataSourceName: event.DataSourceName,
		DataSourceType: event.DataSourceType, Severity: event.Severity, State: event.State, Value: event.Value,
		Message: truncateText(event.Message, 600), Labels: truncateText(event.Labels, 600),
		StartedAt: event.StartedAt.Format("2006-01-02 15:04:05"), LastEvalAt: event.LastEvalAt.Format("2006-01-02 15:04:05"),
	}
}

func summarizeOperationLogs(logs []auditmodel.SysOperationLog) []map[string]any {
	result := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		result = append(result, map[string]any{
			"id": log.ID, "username": log.Username, "module": log.Module, "action": log.Action,
			"description": log.Description, "path": log.Path, "status": log.Status, "error": truncateText(log.ErrorMsg, 300),
			"time": log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result
}

func fallbackAlertAnalysis(evidence map[string]any, cause error) *ChatResult {
	data, _ := json.MarshalIndent(evidence, "", "  ")
	answer := fmt.Sprintf(`告警本地根因分析：

结论：
已收集告警事件、规则配置、关联事件和最近 24 小时操作记录。模型暂不可用，先给出本地判断。

建议：
1. 优先确认告警状态、规则查询语句、阈值和最近一次评估值。
2. 如果最近 24 小时存在发布、配置变更、资源调整，请优先关联变更时间点。
3. 先执行只读排查命令确认影响范围，再决定是否修复。
4. 涉及重启、扩缩容、删除和配置修改时必须人工确认。

模型调用失败原因：%s

Evidence:
%s`, cause.Error(), truncateText(string(data), 12000))
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprint(value)
}

func uintFromAny(value any) uint {
	switch v := value.(type) {
	case uint:
		return v
	case uint64:
		return uint(v)
	case int:
		if v > 0 {
			return uint(v)
		}
	case string:
		parsed, _ := strconv.ParseUint(v, 10, 32)
		return uint(parsed)
	}
	return 0
}
