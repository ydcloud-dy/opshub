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
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	assetmodel "github.com/ydcloud-dy/opshub/internal/biz/asset"
	aiopsdata "github.com/ydcloud-dy/opshub/internal/data/aiops"
	sshclient "github.com/ydcloud-dy/opshub/pkg/ssh"
	k8sservice "github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Service struct {
	db                *gorm.DB
	clusterService    *k8sservice.ClusterService
	generationMu      sync.Mutex
	activeGenerations map[uint]activeGeneration
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:                db,
		clusterService:    k8sservice.NewClusterService(db),
		activeGenerations: make(map[uint]activeGeneration),
	}
}

type activeGeneration struct {
	UserID    uint
	SessionID uint
	Cancel    context.CancelFunc
}

type ProviderRequest struct {
	Name            string  `json:"name" binding:"required"`
	Provider        string  `json:"provider"`
	BaseURL         string  `json:"baseUrl" binding:"required"`
	APIKey          string  `json:"apiKey"`
	Model           string  `json:"model" binding:"required"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"maxTokens"`
	Timeout         int     `json:"timeout"`
	ReasoningEffort string  `json:"reasoningEffort"`
	Enabled         bool    `json:"enabled"`
	IsDefault       bool    `json:"isDefault"`
	Remark          string  `json:"remark"`
}

type ProviderOption struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	IsDefault bool   `json:"isDefault"`
	Remark    string `json:"remark,omitempty"`
}

type ChatRequest struct {
	SessionID             uint   `json:"sessionId"`
	ProviderID            uint   `json:"providerId"`
	Message               string `json:"message" binding:"required"`
	Continue              bool   `json:"continue"`
	ContinueFromMessageID uint   `json:"continueFromMessageId"`
	OriginalQuestion      string `json:"originalQuestion"`
	PreviousAnswer        string `json:"previousAnswer"`
}

type StopChatRequest struct {
	SessionID uint `json:"sessionId" binding:"required"`
	MessageID uint `json:"messageId"`
}

type ChatResponse struct {
	SessionID     uint               `json:"sessionId"`
	Answer        string             `json:"answer"`
	Model         string             `json:"model"`
	Fallback      bool               `json:"fallback"`
	FinishReason  string             `json:"finishReason,omitempty"`
	ThinkingSteps []string           `json:"thinkingSteps,omitempty"`
	Message       *aiopsdata.Message `json:"message,omitempty"`
}

type ChatStreamEvent struct {
	Type          string             `json:"type"`
	SessionID     uint               `json:"sessionId,omitempty"`
	Delta         string             `json:"delta,omitempty"`
	Answer        string             `json:"answer,omitempty"`
	Model         string             `json:"model,omitempty"`
	Fallback      bool               `json:"fallback,omitempty"`
	FinishReason  string             `json:"finishReason,omitempty"`
	ThinkingSteps []string           `json:"thinkingSteps,omitempty"`
	Message       *aiopsdata.Message `json:"message,omitempty"`
	Error         string             `json:"error,omitempty"`
}

type DiagnoseRequest struct {
	ProviderID uint   `json:"providerId"`
	ObjectType string `json:"objectType" binding:"required"`
	ClusterID  uint   `json:"clusterId" binding:"required"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name" binding:"required"`
	Container  string `json:"container"`
	TailLines  int64  `json:"tailLines"`
}

type LogAnalyzeRequest struct {
	Logs          string `json:"logs"`
	Source        string `json:"source"`
	SourceType    string `json:"sourceType"`
	ProviderID    uint   `json:"providerId"`
	K8sObjectType string `json:"k8sObjectType"`
	Title         string `json:"title"`
	SessionID     uint   `json:"sessionId"`
	HostID        uint   `json:"hostId"`
	LogPath       string `json:"logPath"`
	ClusterID     uint   `json:"clusterId"`
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	Container     string `json:"container"`
	TailLines     int64  `json:"tailLines"`
}

type HostDiagnoseRequest struct {
	HostID     uint   `json:"hostId" binding:"required"`
	ProviderID uint   `json:"providerId"`
	Focus      string `json:"focus"`
}

type DiagnosisResponse struct {
	SessionID  uint   `json:"sessionId"`
	TaskID     uint   `json:"taskId"`
	Conclusion string `json:"conclusion"`
	Evidence   any    `json:"evidence"`
	Suggestion string `json:"suggestion"`
	Model      string `json:"model"`
	Fallback   bool   `json:"fallback"`
}

type SessionListItem struct {
	aiopsdata.Session
	MessageCount int64 `json:"messageCount"`
}

func (s *Service) ListProviders(ctx context.Context) ([]aiopsdata.Provider, error) {
	var providers []aiopsdata.Provider
	err := s.db.WithContext(ctx).Order("is_default DESC, id DESC").Find(&providers).Error
	for i := range providers {
		providers[i].APIKey = maskSecret(providers[i].APIKey)
	}
	return providers, err
}

func (s *Service) ListProviderOptions(ctx context.Context) ([]ProviderOption, error) {
	var providers []aiopsdata.Provider
	err := s.db.WithContext(ctx).
		Select("id, name, provider, model, is_default, remark").
		Where("enabled = ?", true).
		Order("is_default DESC, id DESC").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	options := make([]ProviderOption, 0, len(providers))
	for _, provider := range providers {
		options = append(options, ProviderOption{
			ID:        provider.ID,
			Name:      provider.Name,
			Model:     provider.Model,
			Provider:  provider.Provider,
			IsDefault: provider.IsDefault,
			Remark:    provider.Remark,
		})
	}
	return options, nil
}

func (s *Service) SaveProvider(ctx context.Context, id uint, req ProviderRequest) (*aiopsdata.Provider, error) {
	provider := aiopsdata.Provider{
		Name:            strings.TrimSpace(req.Name),
		Provider:        defaultString(req.Provider, "openai-compatible"),
		BaseURL:         strings.TrimSpace(req.BaseURL),
		Model:           strings.TrimSpace(req.Model),
		Temperature:     req.Temperature,
		MaxTokens:       req.MaxTokens,
		Timeout:         req.Timeout,
		ReasoningEffort: normalizeReasoningEffort(req.ReasoningEffort),
		Enabled:         req.Enabled,
		IsDefault:       req.IsDefault,
		Remark:          strings.TrimSpace(req.Remark),
	}
	if provider.Temperature == 0 {
		provider.Temperature = 0.2
	}
	if provider.MaxTokens <= 0 {
		provider.MaxTokens = defaultAIModelMaxTokens
	}
	if provider.Timeout <= 0 {
		provider.Timeout = defaultAIModelTimeoutSeconds
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if provider.IsDefault {
			if err := tx.Model(&aiopsdata.Provider{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if id == 0 {
			provider.APIKey = strings.TrimSpace(req.APIKey)
			if err := tx.Create(&provider).Error; err != nil {
				return err
			}
			return nil
		}
		var existing aiopsdata.Provider
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}
		provider.ID = existing.ID
		provider.CreatedAt = existing.CreatedAt
		if strings.TrimSpace(req.APIKey) == "" || isMaskedSecret(req.APIKey) {
			provider.APIKey = existing.APIKey
		} else {
			provider.APIKey = strings.TrimSpace(req.APIKey)
		}
		return tx.Model(&existing).Updates(map[string]any{
			"name":             provider.Name,
			"provider":         provider.Provider,
			"base_url":         provider.BaseURL,
			"api_key":          provider.APIKey,
			"model":            provider.Model,
			"temperature":      provider.Temperature,
			"max_tokens":       provider.MaxTokens,
			"timeout":          provider.Timeout,
			"reasoning_effort": provider.ReasoningEffort,
			"enabled":          provider.Enabled,
			"is_default":       provider.IsDefault,
			"remark":           provider.Remark,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	var saved aiopsdata.Provider
	if err := s.db.WithContext(ctx).First(&saved, func() uint {
		if id != 0 {
			return id
		}
		return provider.ID
	}()).Error; err != nil {
		return nil, err
	}
	saved.APIKey = maskSecret(saved.APIKey)
	return &saved, nil
}

func (s *Service) DeleteProvider(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&aiopsdata.Provider{}, id).Error
}

func (s *Service) TestProvider(ctx context.Context, id uint) (*ChatResult, error) {
	var provider aiopsdata.Provider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return nil, err
	}
	result, err := callModel(ctx, &provider, []ChatMessage{
		{Role: "system", Content: "你是 OpsHub 智能运维模块的连通性测试助手。"},
		{Role: "user", Content: "请用一句中文回复：AI 模型连接正常。"},
	})
	now := time.Now()
	msg := "连接正常"
	if err != nil {
		msg = err.Error()
	}
	_ = s.db.WithContext(ctx).Model(&provider).Updates(map[string]any{
		"last_test_at":  &now,
		"last_test_msg": msg,
	}).Error
	return result, err
}

func (s *Service) registerActiveGeneration(messageID, userID, sessionID uint, cancel context.CancelFunc) {
	if messageID == 0 || cancel == nil {
		return
	}
	s.generationMu.Lock()
	defer s.generationMu.Unlock()
	if existing, ok := s.activeGenerations[messageID]; ok && existing.Cancel != nil {
		existing.Cancel()
	}
	s.activeGenerations[messageID] = activeGeneration{
		UserID:    userID,
		SessionID: sessionID,
		Cancel:    cancel,
	}
}

func (s *Service) unregisterActiveGeneration(messageID uint) {
	if messageID == 0 {
		return
	}
	s.generationMu.Lock()
	defer s.generationMu.Unlock()
	delete(s.activeGenerations, messageID)
}

func (s *Service) cancelSessionGenerations(ctx context.Context, userID, sessionID uint, excludeMessageID uint, reason string) {
	_ = ctx
	if sessionID == 0 {
		return
	}
	s.generationMu.Lock()
	for messageID, item := range s.activeGenerations {
		if item.UserID == userID && item.SessionID == sessionID && messageID != excludeMessageID {
			if item.Cancel != nil {
				item.Cancel()
			}
			delete(s.activeGenerations, messageID)
		}
	}
	s.generationMu.Unlock()
	statusReason := defaultString(reason, "已被新的提问中断")
	saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := s.db.WithContext(saveCtx).
		Model(&aiopsdata.Message{}).
		Where("user_id = ? AND session_id = ? AND status = ?", userID, sessionID, "generating")
	if excludeMessageID > 0 {
		query = query.Where("id <> ?", excludeMessageID)
	}
	_ = query.Updates(map[string]any{
		"status":     "interrupted",
		"error":      statusReason,
		"updated_at": time.Now(),
	}).Error
}

func (s *Service) StopChat(ctx context.Context, userID uint, req StopChatRequest) error {
	if req.SessionID == 0 {
		return fmt.Errorf("无效的会话ID")
	}
	if req.MessageID > 0 {
		s.generationMu.Lock()
		item, ok := s.activeGenerations[req.MessageID]
		if ok && item.UserID == userID && item.SessionID == req.SessionID {
			if item.Cancel != nil {
				item.Cancel()
			}
			delete(s.activeGenerations, req.MessageID)
		}
		s.generationMu.Unlock()
	}
	if req.MessageID == 0 {
		s.cancelSessionGenerations(ctx, userID, req.SessionID, 0, "用户手动停止生成")
	} else {
		_ = s.db.WithContext(ctx).
			Model(&aiopsdata.Message{}).
			Where("user_id = ? AND session_id = ? AND id = ? AND status = ?", userID, req.SessionID, req.MessageID, "generating").
			Updates(map[string]any{
				"status":     "interrupted",
				"error":      "用户手动停止生成",
				"updated_at": time.Now(),
			}).Error
	}
	return nil
}

type chatExecutionPlan struct {
	SystemPrompt  string
	Prompt        string
	ThinkingSteps []string
	Context       any
	ToolName      string
	ToolParams    map[string]any
	UseHistory    bool
}

func (s *Service) buildChatExecutionPlan(ctx context.Context, userID, sessionID, excludeMessageID uint, req ChatRequest, userContent string, onUpdate func([]string, any)) chatExecutionPlan {
	if req.Continue || isContinuePrompt(userContent) {
		originalQuestion := sanitizeText(req.OriginalQuestion, 6000)
		if originalQuestion == "" {
			originalQuestion = s.latestNonContinueUserQuestion(ctx, sessionID, excludeMessageID)
		}
		previousAnswer := cleanContinuationAnswer(req.PreviousAnswer)
		if previousAnswer == "" {
			previousAnswer = s.resolvePreviousAssistantAnswer(ctx, sessionID, req.ContinueFromMessageID)
		}
		answerTail := truncateRunesFromEnd(previousAnswer, 12000)
		anchor := continuationAnchor(answerTail)
		return chatExecutionPlan{
			SystemPrompt: continuationSystemPrompt(),
			Prompt:       buildContinuationPrompt(originalQuestion, answerTail, anchor),
			ThinkingSteps: []string{
				"识别为长回答续写请求",
				"读取上一段回答末尾，定位中断小节和代码块",
				"只从中断位置继续输出，避免重复或跳章",
			},
			Context: map[string]any{
				"mode":                  "continue",
				"continueFromMessageId": req.ContinueFromMessageID,
				"originalQuestion":      originalQuestion,
				"previousAnswerTail":    answerTail,
				"continuationAnchor":    anchor,
			},
			ToolName: "continue_ai_answer",
			ToolParams: map[string]any{
				"continueFromMessageId": req.ContinueFromMessageID,
				"question":              originalQuestion,
			},
			UseHistory: false,
		}
	}

	if isOpsHubPlatformQuestion(userContent) {
		return s.buildOpsHubAgentExecutionPlan(ctx, userID, sessionID, excludeMessageID, req, userContent, onUpdate)
	}

	platformContext := s.collectAssistantPlatformContext(ctx, sessionID, excludeMessageID, userContent)
	return chatExecutionPlan{
		SystemPrompt:  assistantSystemPrompt(),
		Prompt:        buildAssistantPrompt(userContent, platformContext),
		ThinkingSteps: buildAssistantThinkingSteps(platformContext),
		Context:       platformContext,
		ToolName:      "collect_platform_context",
		ToolParams: map[string]any{
			"question": userContent,
		},
		UseHistory: true,
	}
}

func (s *Service) Chat(ctx context.Context, userID uint, username string, req ChatRequest) (*ChatResponse, error) {
	session, err := s.ensureSession(ctx, req.SessionID, userID, username, "chat", titleFromText(req.Message, "AI 助手会话"))
	if err != nil {
		return nil, err
	}
	isContinuation := req.Continue || isContinuePrompt(req.Message)
	messageRole := "user"
	if isContinuation {
		messageRole = "system"
	}
	req.Continue = isContinuation
	userMsg := aiopsdata.Message{
		SessionID: session.ID,
		UserID:    userID,
		Role:      messageRole,
		Content:   sanitizeText(req.Message, 6000),
		Status:    "success",
	}
	if err := s.db.WithContext(ctx).Create(&userMsg).Error; err != nil {
		return nil, err
	}

	plan := s.buildChatExecutionPlan(ctx, userID, session.ID, userMsg.ID, req, userMsg.Content, nil)
	messages := s.buildChatModelMessages(ctx, session.ID, userMsg.ID, plan)
	result, err := s.callSelectedModel(ctx, req.ProviderID, messages)
	if err != nil {
		result = fallbackChatForPlan(userMsg.Content, plan, err)
	}
	result.Content = stripModelThinking(result.Content)
	messageStatus := "success"
	if isLengthFinishReason(result.FinishReason) {
		messageStatus = "truncated"
	}
	assistantMsg := aiopsdata.Message{
		SessionID: session.ID,
		UserID:    userID,
		Role:      "assistant",
		Content:   result.Content,
		Model:     result.Model,
		Status:    messageStatus,
		TokensIn:  result.TokensIn,
		TokensOut: result.TokensOut,
		LatencyMS: result.LatencyMS,
	}
	if data, marshalErr := json.Marshal(plan.Context); marshalErr == nil {
		assistantMsg.ContextRef = string(data)
	}
	if err := s.db.WithContext(ctx).Create(&assistantMsg).Error; err != nil {
		return nil, err
	}
	s.recordToolCall(ctx, session.ID, assistantMsg.ID, userID, defaultString(plan.ToolName, "ai_chat"), plan.ToolParams, plan.Context)
	_ = s.db.WithContext(ctx).Model(session).Updates(map[string]any{
		"summary":    truncateText(stripMarkdown(result.Content), 500),
		"updated_at": time.Now(),
	}).Error

	return &ChatResponse{
		SessionID:     session.ID,
		Answer:        result.Content,
		Model:         result.Model,
		Fallback:      result.Fallback,
		FinishReason:  result.FinishReason,
		ThinkingSteps: plan.ThinkingSteps,
		Message:       &assistantMsg,
	}, nil
}

func (s *Service) ChatStream(ctx context.Context, userID uint, username string, req ChatRequest, emit func(ChatStreamEvent) error) error {
	if emit == nil {
		return errors.New("未配置流式输出处理器")
	}
	session, err := s.ensureSession(ctx, req.SessionID, userID, username, "chat", titleFromText(req.Message, "AI 助手会话"))
	if err != nil {
		return err
	}
	s.cancelSessionGenerations(context.Background(), userID, session.ID, 0, "已被新的提问中断")
	isContinuation := req.Continue || isContinuePrompt(req.Message)
	messageRole := "user"
	if isContinuation {
		messageRole = "system"
	}
	req.Continue = isContinuation
	userMsg := aiopsdata.Message{
		SessionID: session.ID,
		UserID:    userID,
		Role:      messageRole,
		Content:   sanitizeText(req.Message, 6000),
		Status:    "success",
	}
	if err := s.db.WithContext(ctx).Create(&userMsg).Error; err != nil {
		return err
	}

	assistantMsg := aiopsdata.Message{
		SessionID: session.ID,
		UserID:    userID,
		Role:      "assistant",
		Status:    "generating",
	}
	if err := s.db.WithContext(ctx).Create(&assistantMsg).Error; err != nil {
		return err
	}
	generationCtx, cancelGeneration := context.WithCancel(context.Background())
	s.registerActiveGeneration(assistantMsg.ID, userID, session.ID, cancelGeneration)
	defer func() {
		cancelGeneration()
		s.unregisterActiveGeneration(assistantMsg.ID)
	}()
	clientDisconnected := false
	emitIfConnected := func(event ChatStreamEvent) error {
		if clientDisconnected || ctx.Err() != nil {
			clientDisconnected = true
			return nil
		}
		if err := emit(event); err != nil {
			clientDisconnected = true
		}
		return nil
	}
	_ = emitIfConnected(ChatStreamEvent{
		Type:          "meta",
		SessionID:     session.ID,
		ThinkingSteps: []string{"接收问题：正在启动 OpsHub AI 执行链路"},
		Message:       &assistantMsg,
	})

	planCtx, cancelPlan := context.WithTimeout(generationCtx, 10*time.Minute)
	defer cancelPlan()
	plan := s.buildChatExecutionPlan(planCtx, userID, session.ID, userMsg.ID, req, userMsg.Content, func(steps []string, contextValue any) {
		if data, marshalErr := json.Marshal(contextValue); marshalErr == nil {
			assistantMsg.ContextRef = string(data)
			saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = s.db.WithContext(saveCtx).Model(&aiopsdata.Message{}).Where("id = ?", assistantMsg.ID).Update("context_ref", assistantMsg.ContextRef).Error
			cancel()
		}
		_ = emitIfConnected(ChatStreamEvent{
			Type:          "meta",
			SessionID:     session.ID,
			ThinkingSteps: steps,
			Message:       &assistantMsg,
		})
	})
	if data, marshalErr := json.Marshal(plan.Context); marshalErr == nil {
		assistantMsg.ContextRef = string(data)
		saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = s.db.WithContext(saveCtx).Model(&aiopsdata.Message{}).Where("id = ?", assistantMsg.ID).Update("context_ref", assistantMsg.ContextRef).Error
		cancel()
	}
	messages := s.buildChatModelMessages(planCtx, session.ID, userMsg.ID, plan)
	_ = emitIfConnected(ChatStreamEvent{
		Type:          "meta",
		SessionID:     session.ID,
		ThinkingSteps: plan.ThinkingSteps,
		Message:       &assistantMsg,
	})

	persistAssistant := func(content, status string, result *ChatResult, errorText string) {
		saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		now := time.Now()
		updates := map[string]any{
			"content":    content,
			"status":     status,
			"updated_at": now,
		}
		if result != nil {
			updates["model"] = result.Model
			updates["tokens_in"] = result.TokensIn
			updates["tokens_out"] = result.TokensOut
			updates["latency_ms"] = result.LatencyMS
		}
		if strings.TrimSpace(errorText) != "" {
			updates["error"] = truncateText(errorText, 1000)
		}
		_ = s.db.WithContext(saveCtx).Model(&aiopsdata.Message{}).Where("id = ?", assistantMsg.ID).Updates(updates).Error
		if strings.TrimSpace(content) != "" {
			_ = s.db.WithContext(saveCtx).Model(&aiopsdata.Session{}).Where("id = ?", session.ID).Updates(map[string]any{
				"summary":    truncateText(stripMarkdown(content), 500),
				"updated_at": now,
			}).Error
		} else {
			_ = s.db.WithContext(saveCtx).Model(&aiopsdata.Session{}).Where("id = ?", session.ID).Update("updated_at", now).Error
		}
	}

	emitted := false
	var visibleContent strings.Builder
	lastPersistAt := time.Now()
	thinkFilter := newThinkingDeltaFilter()
	if generationCtx.Err() != nil {
		persistAssistant("", "interrupted", nil, "用户手动停止生成")
		assistantMsg.Status = "interrupted"
		assistantMsg.Error = "用户手动停止生成"
		_ = emitIfConnected(ChatStreamEvent{Type: "error", Error: "用户手动停止生成", Message: &assistantMsg})
		return nil
	}

	modelCtx, cancelModel := context.WithTimeout(generationCtx, 30*time.Minute)
	defer cancelModel()
	result, err := s.callSelectedModelStream(modelCtx, req.ProviderID, messages, func(delta string) error {
		cleanDelta := thinkFilter.Push(delta)
		if cleanDelta == "" {
			return nil
		}
		emitted = true
		visibleContent.WriteString(cleanDelta)
		if time.Since(lastPersistAt) >= 700*time.Millisecond {
			persistAssistant(visibleContent.String(), "generating", nil, "")
			lastPersistAt = time.Now()
		}
		return emitIfConnected(ChatStreamEvent{Type: "delta", Delta: cleanDelta})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || modelCtx.Err() != nil || generationCtx.Err() != nil {
			content := strings.TrimSpace(visibleContent.String())
			statusErr := "用户手动停止生成"
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(modelCtx.Err(), context.DeadlineExceeded) {
				statusErr = "生成超时中断"
			}
			persistAssistant(content, "interrupted", nil, statusErr)
			assistantMsg.Content = content
			assistantMsg.Status = "interrupted"
			assistantMsg.Error = statusErr
			_ = emitIfConnected(ChatStreamEvent{Type: "error", Error: statusErr, Message: &assistantMsg})
			return nil
		}
		if emitted {
			if content := strings.TrimSpace(visibleContent.String()); content != "" {
				persistAssistant(content, "interrupted", nil, err.Error())
				assistantMsg.Content = content
			}
			assistantMsg.Status = "interrupted"
			assistantMsg.Error = err.Error()
			_ = emitIfConnected(ChatStreamEvent{Type: "error", Error: err.Error(), Message: &assistantMsg})
			return nil
		}
		thinkFilter = newThinkingDeltaFilter()
		result = fallbackChatForPlan(userMsg.Content, plan, err)
		result.Content = stripModelThinking(result.Content)
		visibleContent.WriteString(result.Content)
		persistAssistant(result.Content, "generating", result, "")
		if err := emitTextDeltas(result.Content, emitIfConnected); err != nil {
			persistAssistant(result.Content, "interrupted", result, err.Error())
			return nil
		}
	} else if tail := thinkFilter.Flush(); tail != "" {
		emitted = true
		visibleContent.WriteString(tail)
		persistAssistant(visibleContent.String(), "generating", nil, "")
		if err := emitIfConnected(ChatStreamEvent{Type: "delta", Delta: tail}); err != nil {
			persistAssistant(visibleContent.String(), "interrupted", nil, err.Error())
			return nil
		}
	}

	result.Content = stripModelThinking(result.Content)
	if strings.TrimSpace(result.Content) == "" && strings.TrimSpace(visibleContent.String()) != "" {
		result.Content = strings.TrimSpace(visibleContent.String())
	}
	messageStatus := "success"
	if isLengthFinishReason(result.FinishReason) {
		messageStatus = "truncated"
	}
	persistAssistant(result.Content, messageStatus, result, "")
	assistantMsg.Content = result.Content
	assistantMsg.Model = result.Model
	assistantMsg.Status = messageStatus
	assistantMsg.TokensIn = result.TokensIn
	assistantMsg.TokensOut = result.TokensOut
	assistantMsg.LatencyMS = result.LatencyMS
	s.recordToolCall(context.Background(), session.ID, assistantMsg.ID, userID, defaultString(plan.ToolName, "ai_chat"), plan.ToolParams, plan.Context)

	return emitIfConnected(ChatStreamEvent{
		Type:         "done",
		SessionID:    session.ID,
		Answer:       result.Content,
		Model:        result.Model,
		Fallback:     result.Fallback,
		FinishReason: result.FinishReason,
		Message:      &assistantMsg,
	})
}

func (s *Service) buildAssistantModelMessages(ctx context.Context, sessionID uint, excludeMessageID uint, prompt string) []ChatMessage {
	messages := []ChatMessage{{Role: "system", Content: assistantSystemPrompt()}}
	messages = append(messages, s.recentSessionChatMessages(ctx, sessionID, excludeMessageID, 6)...)
	messages = append(messages, ChatMessage{Role: "user", Content: prompt})
	return messages
}

func (s *Service) buildChatModelMessages(ctx context.Context, sessionID uint, excludeMessageID uint, plan chatExecutionPlan) []ChatMessage {
	systemPrompt := defaultString(plan.SystemPrompt, assistantSystemPrompt())
	messages := []ChatMessage{{Role: "system", Content: systemPrompt}}
	if plan.UseHistory {
		messages = append(messages, s.recentSessionChatMessages(ctx, sessionID, excludeMessageID, 6)...)
	}
	messages = append(messages, ChatMessage{Role: "user", Content: plan.Prompt})
	return messages
}

func (s *Service) recentSessionChatMessages(ctx context.Context, sessionID uint, excludeMessageID uint, limit int) []ChatMessage {
	if sessionID == 0 || limit <= 0 {
		return nil
	}
	var records []aiopsdata.Message
	query := s.db.WithContext(ctx).
		Where("session_id = ? AND role IN ? AND status IN ?", sessionID, []string{"user", "assistant"}, []string{"success", "truncated"}).
		Order("id DESC").
		Limit(limit)
	if excludeMessageID > 0 {
		query = query.Where("id <> ?", excludeMessageID)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil
	}

	messages := make([]ChatMessage, 0, len(records))
	for i := len(records) - 1; i >= 0; i-- {
		role := strings.TrimSpace(records[i].Role)
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(records[i].Content)
		if content == "" {
			continue
		}
		messages = append(messages, ChatMessage{
			Role:    role,
			Content: compactHistoryContent(role, content),
		})
	}
	return messages
}

func (s *Service) latestNonContinueUserQuestion(ctx context.Context, sessionID, excludeMessageID uint) string {
	if sessionID == 0 {
		return ""
	}
	var records []aiopsdata.Message
	query := s.db.WithContext(ctx).
		Where("session_id = ? AND role = ? AND status IN ?", sessionID, "user", []string{"success", "truncated"}).
		Order("id DESC").
		Limit(20)
	if excludeMessageID > 0 {
		query = query.Where("id < ?", excludeMessageID)
	}
	if err := query.Find(&records).Error; err != nil {
		return ""
	}
	for _, record := range records {
		content := strings.TrimSpace(record.Content)
		if content != "" && !isContinuePrompt(content) {
			return content
		}
	}
	return ""
}

func (s *Service) resolvePreviousAssistantAnswer(ctx context.Context, sessionID, messageID uint) string {
	if sessionID == 0 {
		return ""
	}
	var msg aiopsdata.Message
	query := s.db.WithContext(ctx).Where("session_id = ? AND role = ?", sessionID, "assistant")
	if messageID > 0 {
		query = query.Where("id = ?", messageID)
	} else {
		query = query.Where("status IN ?", []string{"success", "generating", "interrupted", "truncated"}).Order("id DESC")
	}
	if err := query.First(&msg).Error; err != nil {
		return ""
	}
	return cleanContinuationAnswer(msg.Content)
}

func compactHistoryContent(role string, content string) string {
	limit := 2000
	if role == "assistant" {
		limit = 6000
	}
	return truncateRunesFromEnd(content, limit)
}

func truncateRunesFromEnd(text string, max int) string {
	runes := []rune(strings.TrimSpace(text))
	if max <= 0 || len(runes) <= max {
		return string(runes)
	}
	return "（前文省略）\n" + string(runes[len(runes)-max:])
}

func (s *Service) AnalyzeLogs(ctx context.Context, userID uint, username string, req LogAnalyzeRequest) (*DiagnosisResponse, error) {
	logs, resolvedSource, err := s.resolveAnalyzeLogs(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	logs = sanitizeText(logs, 16000)
	if strings.TrimSpace(logs) == "" {
		return nil, fmt.Errorf("未获取到可分析的日志内容")
	}
	title := defaultString(req.Title, defaultString(resolvedSource, "日志智能分析"))
	session, err := s.ensureSession(ctx, req.SessionID, userID, username, "log", title)
	if err != nil {
		return nil, err
	}

	evidence := buildLogEvidence(logs, resolvedSource)
	evidence["sourceType"] = defaultString(req.SourceType, "manual")
	evidenceJSON, _ := json.Marshal(evidence)
	tool := s.recordToolCall(ctx, session.ID, 0, userID, "analyze_log_locally", map[string]any{
		"source":     resolvedSource,
		"sourceType": defaultString(req.SourceType, "manual"),
		"length":     len(logs),
	}, evidence)

	prompt := fmt.Sprintf(`请作为资深 SRE 分析下面日志。请输出：
1. 异常摘要
2. 疑似根因
3. 证据链
4. 处理建议
5. 推荐排查命令

日志来源：%s
日志内容：
%s`, defaultString(resolvedSource, "手动输入"), logs)

	result, modelErr := s.callSelectedModel(ctx, req.ProviderID, []ChatMessage{
		{Role: "system", Content: diagnosisSystemPrompt()},
		{Role: "user", Content: prompt},
	})
	if modelErr != nil {
		result = fallbackLogAnalysis(logs, modelErr)
	}
	assistantMsg := aiopsdata.Message{
		SessionID:  session.ID,
		UserID:     userID,
		Role:       "assistant",
		Content:    result.Content,
		Model:      result.Model,
		Status:     "success",
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
		LatencyMS:  result.LatencyMS,
		ContextRef: string(evidenceJSON),
	}
	if err := s.db.WithContext(ctx).Create(&assistantMsg).Error; err != nil {
		return nil, err
	}
	if tool != nil {
		_ = s.db.WithContext(ctx).Model(tool).Update("message_id", assistantMsg.ID).Error
	}
	task := aiopsdata.DiagnosisTask{
		UserID:       userID,
		Username:     username,
		SessionID:    session.ID,
		ObjectType:   "log",
		Status:       "success",
		Conclusion:   result.Content,
		EvidenceJSON: string(evidenceJSON),
		Suggestion:   extractSuggestions(result.Content),
	}
	if err := s.db.WithContext(ctx).Create(&task).Error; err != nil {
		return nil, err
	}
	return &DiagnosisResponse{
		SessionID:  session.ID,
		TaskID:     task.ID,
		Conclusion: result.Content,
		Evidence:   evidence,
		Suggestion: task.Suggestion,
		Model:      result.Model,
		Fallback:   result.Fallback,
	}, nil
}

func (s *Service) DiagnoseKubernetes(ctx context.Context, userID uint, username string, req DiagnoseRequest) (*DiagnosisResponse, error) {
	if req.TailLines <= 0 {
		req.TailLines = 120
	}
	objectType := strings.ToLower(strings.TrimSpace(req.ObjectType))
	session, err := s.ensureSession(ctx, 0, userID, username, "diagnosis", fmt.Sprintf("%s/%s 智能诊断", req.Namespace, req.Name))
	if err != nil {
		return nil, err
	}

	evidence, err := s.collectKubernetesEvidence(ctx, userID, objectType, req)
	if err != nil {
		task := aiopsdata.DiagnosisTask{
			UserID:     userID,
			Username:   username,
			SessionID:  session.ID,
			ObjectType: objectType,
			ClusterID:  req.ClusterID,
			Namespace:  req.Namespace,
			ObjectName: req.Name,
			Container:  req.Container,
			Status:     "failed",
			Error:      err.Error(),
		}
		_ = s.db.WithContext(ctx).Create(&task).Error
		return nil, err
	}
	evidenceJSON, _ := json.Marshal(evidence)
	tool := s.recordToolCall(ctx, session.ID, 0, userID, "collect_kubernetes_evidence", map[string]any{
		"objectType": objectType,
		"clusterId":  req.ClusterID,
		"namespace":  req.Namespace,
		"name":       req.Name,
		"container":  req.Container,
	}, evidence)

	prompt := fmt.Sprintf(`请诊断 Kubernetes %s 异常。请输出：
1. 健康结论
2. 疑似原因，按可能性排序
3. 证据链，引用状态、事件、日志
4. 影响范围
5. 建议处理步骤
6. 推荐 kubectl 命令

诊断上下文 JSON：
%s`, objectType, string(evidenceJSON))

	result, modelErr := s.callSelectedModel(ctx, req.ProviderID, []ChatMessage{
		{Role: "system", Content: diagnosisSystemPrompt()},
		{Role: "user", Content: prompt},
	})
	if modelErr != nil {
		result = fallbackKubernetesDiagnosis(objectType, evidence, modelErr)
	}
	assistantMsg := aiopsdata.Message{
		SessionID:  session.ID,
		UserID:     userID,
		Role:       "assistant",
		Content:    result.Content,
		Model:      result.Model,
		Status:     "success",
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
		LatencyMS:  result.LatencyMS,
		ContextRef: string(evidenceJSON),
	}
	if err := s.db.WithContext(ctx).Create(&assistantMsg).Error; err != nil {
		return nil, err
	}
	if tool != nil {
		_ = s.db.WithContext(ctx).Model(tool).Update("message_id", assistantMsg.ID).Error
	}
	task := aiopsdata.DiagnosisTask{
		UserID:       userID,
		Username:     username,
		SessionID:    session.ID,
		ObjectType:   objectType,
		ClusterID:    req.ClusterID,
		Namespace:    req.Namespace,
		ObjectName:   req.Name,
		Container:    req.Container,
		Status:       "success",
		Conclusion:   result.Content,
		EvidenceJSON: string(evidenceJSON),
		Suggestion:   extractSuggestions(result.Content),
	}
	if err := s.db.WithContext(ctx).Create(&task).Error; err != nil {
		return nil, err
	}
	return &DiagnosisResponse{
		SessionID:  session.ID,
		TaskID:     task.ID,
		Conclusion: result.Content,
		Evidence:   evidence,
		Suggestion: task.Suggestion,
		Model:      result.Model,
		Fallback:   result.Fallback,
	}, nil
}

func (s *Service) resolveAnalyzeLogs(ctx context.Context, userID uint, req LogAnalyzeRequest) (string, string, error) {
	sourceType := strings.ToLower(strings.TrimSpace(req.SourceType))
	if sourceType == "" {
		sourceType = "manual"
	}
	tailLines := req.TailLines
	if tailLines <= 0 {
		tailLines = 200
	}
	if tailLines > 5000 {
		tailLines = 5000
	}
	switch sourceType {
	case "manual":
		return req.Logs, defaultString(req.Source, "手动输入"), nil
	case "host":
		logs, source, err := s.collectHostLog(ctx, req.HostID, req.LogPath, tailLines)
		return logs, source, err
	case "kubernetes", "k8s":
		if req.ClusterID == 0 || strings.TrimSpace(req.Namespace) == "" || strings.TrimSpace(req.PodName) == "" {
			return "", "", fmt.Errorf("Kubernetes 日志分析需要选择集群、命名空间和资源对象")
		}
		clientset, err := s.clusterService.GetClientsetForUser(ctx, req.ClusterID, userID)
		if err != nil {
			return "", "", fmt.Errorf("获取集群连接失败: %w", err)
		}
		return s.collectKubernetesLog(ctx, clientset, req, tailLines)
	default:
		return "", "", fmt.Errorf("不支持的日志来源类型: %s", req.SourceType)
	}
}

func (s *Service) collectKubernetesLog(ctx context.Context, clientset *kubernetes.Clientset, req LogAnalyzeRequest, tailLines int64) (string, string, error) {
	objectType := strings.ToLower(strings.TrimSpace(req.K8sObjectType))
	if objectType == "" {
		objectType = "pod"
	}
	namespace := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.PodName)
	podName := name
	sourceKind := "Pod"
	switch objectType {
	case "pod":
		sourceKind = "Pod"
	case "deployment":
		item, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("获取 Deployment 失败: %w", err)
		}
		podName, err = firstPodBySelector(ctx, clientset, namespace, metav1.FormatLabelSelector(item.Spec.Selector))
		if err != nil {
			return "", "", err
		}
		sourceKind = "Deployment"
	case "statefulset":
		item, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("获取 StatefulSet 失败: %w", err)
		}
		podName, err = firstPodBySelector(ctx, clientset, namespace, metav1.FormatLabelSelector(item.Spec.Selector))
		if err != nil {
			return "", "", err
		}
		sourceKind = "StatefulSet"
	case "daemonset":
		item, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("获取 DaemonSet 失败: %w", err)
		}
		podName, err = firstPodBySelector(ctx, clientset, namespace, metav1.FormatLabelSelector(item.Spec.Selector))
		if err != nil {
			return "", "", err
		}
		sourceKind = "DaemonSet"
	case "job":
		item, err := clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("获取 Job 失败: %w", err)
		}
		podName, err = firstPodBySelector(ctx, clientset, namespace, metav1.FormatLabelSelector(item.Spec.Selector))
		if err != nil {
			return "", "", err
		}
		sourceKind = "Job"
	case "cronjob":
		podName, err := firstCronJobPod(ctx, clientset, namespace, name)
		if err != nil {
			return "", "", err
		}
		sourceKind = "CronJob"
		return s.collectPodLog(ctx, clientset, namespace, podName, req.Container, tailLines, fmt.Sprintf("K8s %s %s/%s -> Pod %s", sourceKind, namespace, name, podName))
	default:
		return "", "", fmt.Errorf("不支持的 Kubernetes 日志对象类型: %s", req.K8sObjectType)
	}
	return s.collectPodLog(ctx, clientset, namespace, podName, req.Container, tailLines, fmt.Sprintf("K8s %s %s/%s -> Pod %s", sourceKind, namespace, name, podName))
}

func (s *Service) collectPodLog(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName, container string, tailLines int64, source string) (string, string, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("获取 Pod 失败: %w", err)
	}
	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}
	if container == "" {
		return "", "", fmt.Errorf("Pod 中没有可读取日志的容器")
	}
	return s.getPodLogs(ctx, clientset, namespace, podName, container, tailLines), source + "/" + container, nil
}

func firstPodBySelector(ctx context.Context, clientset *kubernetes.Clientset, namespace, selector string) (string, error) {
	if strings.TrimSpace(selector) == "" {
		return "", fmt.Errorf("资源没有 selector，无法自动定位 Pod")
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", fmt.Errorf("查询关联 Pod 失败: %w", err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("未找到 selector=%s 对应的 Pod", selector)
	}
	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
	})
	return pods.Items[0].Name, nil
}

func firstCronJobPod(ctx context.Context, clientset *kubernetes.Clientset, namespace, cronJobName string) (string, error) {
	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("查询 CronJob 关联 Job 失败: %w", err)
	}
	matchedJobs := make([]batchv1.Job, 0)
	for _, job := range jobs.Items {
		for _, owner := range job.OwnerReferences {
			if strings.EqualFold(owner.Kind, "CronJob") && owner.Name == cronJobName {
				matchedJobs = append(matchedJobs, job)
				break
			}
		}
	}
	if len(matchedJobs) == 0 {
		return "", fmt.Errorf("未找到 CronJob %s 产生的 Job", cronJobName)
	}
	sort.Slice(matchedJobs, func(i, j int) bool {
		return matchedJobs[i].CreationTimestamp.After(matchedJobs[j].CreationTimestamp.Time)
	})
	selector := metav1.FormatLabelSelector(matchedJobs[0].Spec.Selector)
	return firstPodBySelector(ctx, clientset, namespace, selector)
}

func (s *Service) collectHostLog(ctx context.Context, hostID uint, logPath string, tailLines int64) (string, string, error) {
	if hostID == 0 {
		return "", "", fmt.Errorf("主机日志分析需要选择主机")
	}
	cleanPath := filepath.Clean(strings.TrimSpace(logPath))
	if cleanPath == "." || !strings.HasPrefix(cleanPath, "/") {
		return "", "", fmt.Errorf("日志路径必须是绝对路径，例如 /var/log/messages")
	}
	if strings.Contains(cleanPath, "\x00") || strings.ContainsAny(cleanPath, "\n\r") {
		return "", "", fmt.Errorf("日志路径包含非法字符")
	}
	var host assetmodel.Host
	if err := s.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return "", "", fmt.Errorf("获取主机失败: %w", err)
	}
	if host.CredentialID == 0 {
		return "", "", fmt.Errorf("主机未绑定 SSH 凭证，无法自动读取日志")
	}
	var credential assetmodel.Credential
	if err := s.db.WithContext(ctx).Table("credentials").First(&credential, host.CredentialID).Error; err != nil {
		return "", "", fmt.Errorf("获取主机凭证失败: %w", err)
	}
	if err := decryptCredentialForAI(&credential); err != nil {
		return "", "", err
	}
	client, err := newAIHostSSHClient(host, credential)
	if err != nil {
		return "", "", err
	}
	defer client.Close()
	const logErrorMarker = "__OPSHUB_LOG_ERROR__:"
	command := fmt.Sprintf(
		"p=%s; if [ ! -e \"$p\" ]; then printf '%sMISSING:%%s\\n' \"$p\"; elif [ -d \"$p\" ]; then printf '%sDIRECTORY:%%s\\n' \"$p\"; elif [ -r \"$p\" ]; then tail -n %d -- \"$p\"; elif command -v sudo >/dev/null 2>&1; then sudo -n tail -n %d -- \"$p\" 2>/dev/null || printf '%sUNREADABLE:%%s\\n' \"$p\"; else printf '%sUNREADABLE:%%s\\n' \"$p\"; fi",
		shellQuote(cleanPath),
		logErrorMarker,
		logErrorMarker,
		tailLines,
		tailLines,
		logErrorMarker,
		logErrorMarker,
	)
	output, err := client.ExecuteWithTimeout(command, 20*time.Second)
	if err != nil {
		return "", "", fmt.Errorf("读取主机日志失败，请检查 SSH 连通性和日志路径权限: %w", err)
	}
	if strings.HasPrefix(strings.TrimSpace(output), logErrorMarker) {
		parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(output), logErrorMarker), ":", 2)
		reason := parts[0]
		path := cleanPath
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			path = strings.TrimSpace(parts[1])
		}
		switch reason {
		case "MISSING":
			return "", "", fmt.Errorf("日志文件不存在: %s", path)
		case "DIRECTORY":
			return "", "", fmt.Errorf("日志路径是目录，不是文件: %s", path)
		case "UNREADABLE":
			return "", "", fmt.Errorf("当前 SSH 用户无权限读取日志文件: %s，请换用有权限的凭证、调整日志权限，或确认 sudo -n tail 可免密执行", path)
		default:
			return "", "", fmt.Errorf("读取主机日志失败: %s", output)
		}
	}
	return output, fmt.Sprintf("主机 %s(%s) %s", host.Name, host.IP, cleanPath), nil
}

func (s *Service) DiagnoseHost(ctx context.Context, userID uint, username string, req HostDiagnoseRequest) (*DiagnosisResponse, error) {
	session, err := s.ensureSession(ctx, 0, userID, username, "diagnosis", fmt.Sprintf("主机 #%d 智能诊断", req.HostID))
	if err != nil {
		return nil, err
	}

	evidence, err := s.collectHostEvidence(ctx, req.HostID, req.Focus)
	if err != nil {
		task := aiopsdata.DiagnosisTask{
			UserID:     userID,
			Username:   username,
			SessionID:  session.ID,
			ObjectType: "host",
			Status:     "failed",
			Error:      err.Error(),
		}
		_ = s.db.WithContext(ctx).Create(&task).Error
		return nil, err
	}
	evidenceJSON, _ := json.Marshal(evidence)
	tool := s.recordToolCall(ctx, session.ID, 0, userID, "collect_host_evidence", map[string]any{
		"hostId": req.HostID,
		"focus":  req.Focus,
	}, evidence)

	prompt := fmt.Sprintf(`请诊断 OpsHub 主机资产状态。请输出：
1. 健康结论
2. 资源风险，重点关注 CPU、内存、磁盘、在线状态和 Agent 状态
3. 证据链
4. 处理建议
5. 推荐只读排查命令
6. 需要人工确认的高风险操作提醒

诊断关注点：%s
主机上下文 JSON：
%s`, defaultString(req.Focus, "整体健康状态"), string(evidenceJSON))

	result, modelErr := s.callSelectedModel(ctx, req.ProviderID, []ChatMessage{
		{Role: "system", Content: diagnosisSystemPrompt()},
		{Role: "user", Content: prompt},
	})
	if modelErr != nil {
		result = fallbackHostDiagnosis(evidence, modelErr)
	}
	assistantMsg := aiopsdata.Message{
		SessionID:  session.ID,
		UserID:     userID,
		Role:       "assistant",
		Content:    result.Content,
		Model:      result.Model,
		Status:     "success",
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
		LatencyMS:  result.LatencyMS,
		ContextRef: string(evidenceJSON),
	}
	if err := s.db.WithContext(ctx).Create(&assistantMsg).Error; err != nil {
		return nil, err
	}
	if tool != nil {
		_ = s.db.WithContext(ctx).Model(tool).Update("message_id", assistantMsg.ID).Error
	}
	objectName := fmt.Sprintf("host-%d", req.HostID)
	if host, ok := evidence["host"].(map[string]any); ok {
		if name, ok := host["name"].(string); ok && name != "" {
			objectName = name
		}
	}
	task := aiopsdata.DiagnosisTask{
		UserID:       userID,
		Username:     username,
		SessionID:    session.ID,
		ObjectType:   "host",
		ObjectName:   objectName,
		Status:       "success",
		Conclusion:   result.Content,
		EvidenceJSON: string(evidenceJSON),
		Suggestion:   extractSuggestions(result.Content),
	}
	if err := s.db.WithContext(ctx).Create(&task).Error; err != nil {
		return nil, err
	}
	return &DiagnosisResponse{
		SessionID:  session.ID,
		TaskID:     task.ID,
		Conclusion: result.Content,
		Evidence:   evidence,
		Suggestion: task.Suggestion,
		Model:      result.Model,
		Fallback:   result.Fallback,
	}, nil
}

func (s *Service) ListSessions(ctx context.Context, userID uint, page, pageSize int, sessionType string) ([]SessionListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.WithContext(ctx).Model(&aiopsdata.Session{}).Where("user_id = ?", userID)
	if strings.TrimSpace(sessionType) != "" {
		query = query.Where("type = ?", sessionType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var sessions []aiopsdata.Session
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	items := make([]SessionListItem, 0, len(sessions))
	for _, session := range sessions {
		var count int64
		_ = s.db.WithContext(ctx).Model(&aiopsdata.Message{}).
			Where("session_id = ? AND role IN ?", session.ID, []string{"user", "assistant"}).
			Count(&count).Error
		items = append(items, SessionListItem{Session: session, MessageCount: count})
	}
	return items, total, nil
}

func (s *Service) GetSession(ctx context.Context, userID, sessionID uint) (*aiopsdata.Session, []aiopsdata.Message, []aiopsdata.ToolCall, error) {
	var session aiopsdata.Session
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, nil, nil, err
	}
	var messages []aiopsdata.Message
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("id ASC").Find(&messages).Error; err != nil {
		return nil, nil, nil, err
	}
	var tools []aiopsdata.ToolCall
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("id ASC").Find(&tools).Error; err != nil {
		return nil, nil, nil, err
	}
	return &session, messages, tools, nil
}

func (s *Service) DeleteSession(ctx context.Context, userID, sessionID uint) error {
	var session aiopsdata.Session
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", sessionID).Delete(&aiopsdata.ToolCall{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&aiopsdata.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&aiopsdata.DiagnosisTask{}).Error; err != nil {
			return err
		}
		return tx.Delete(&session).Error
	})
}

func (s *Service) ListDiagnosisTasks(ctx context.Context, userID uint, page, pageSize int) ([]aiopsdata.DiagnosisTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.WithContext(ctx).Model(&aiopsdata.DiagnosisTask{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []aiopsdata.DiagnosisTask
	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (s *Service) ensureSession(ctx context.Context, sessionID, userID uint, username, typ, title string) (*aiopsdata.Session, error) {
	if sessionID != 0 {
		var session aiopsdata.Session
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
			return nil, err
		}
		return &session, nil
	}
	session := aiopsdata.Session{
		UserID:   userID,
		Username: username,
		Title:    title,
		Type:     typ,
		Status:   "active",
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Service) getDefaultProvider(ctx context.Context) (*aiopsdata.Provider, error) {
	var provider aiopsdata.Provider
	err := s.db.WithContext(ctx).Where("enabled = ? AND is_default = ?", true, true).First(&provider).Error
	if err == nil {
		return &provider, nil
	}
	err = s.db.WithContext(ctx).Where("enabled = ?", true).Order("id DESC").First(&provider).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (s *Service) getSelectedProvider(ctx context.Context, providerID uint) (*aiopsdata.Provider, error) {
	if providerID > 0 {
		var provider aiopsdata.Provider
		if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
			return nil, err
		}
		if !provider.Enabled {
			return nil, fmt.Errorf("所选 AI 模型已停用")
		}
		return &provider, nil
	}
	return s.getDefaultProvider(ctx)
}

func (s *Service) callSelectedModel(ctx context.Context, providerID uint, messages []ChatMessage) (*ChatResult, error) {
	provider, err := s.getSelectedProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return callModel(ctx, provider, messages)
}

func (s *Service) callSelectedModelStream(ctx context.Context, providerID uint, messages []ChatMessage, onDelta func(string) error) (*ChatResult, error) {
	provider, err := s.getSelectedProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return callModelStream(ctx, provider, messages, onDelta)
}

func (s *Service) recordToolCall(ctx context.Context, sessionID, messageID, userID uint, toolName string, params any, result any) *aiopsdata.ToolCall {
	paramsJSON, _ := json.Marshal(params)
	resultJSON, _ := json.Marshal(result)
	call := aiopsdata.ToolCall{
		SessionID: sessionID,
		MessageID: messageID,
		UserID:    userID,
		ToolName:  toolName,
		Params:    string(paramsJSON),
		Result:    truncateText(string(resultJSON), 60000),
		Status:    "success",
	}
	if err := s.db.WithContext(ctx).Create(&call).Error; err != nil {
		return nil
	}
	return &call
}

func (s *Service) collectKubernetesEvidence(ctx context.Context, userID uint, objectType string, req DiagnoseRequest) (map[string]any, error) {
	clientset, err := s.clusterService.GetClientsetForUser(ctx, req.ClusterID, userID)
	if err != nil {
		return nil, fmt.Errorf("获取集群连接失败: %w", err)
	}
	if !isClusterScopeDiagnosisType(objectType) && strings.TrimSpace(req.Namespace) == "" {
		return nil, fmt.Errorf("%s 诊断需要选择命名空间", objectType)
	}
	evidence := map[string]any{
		"objectType":  objectType,
		"clusterId":   req.ClusterID,
		"namespace":   req.Namespace,
		"name":        req.Name,
		"collectedAt": time.Now().Format(time.RFC3339),
	}
	switch objectType {
	case "pod":
		pod, err := clientset.CoreV1().Pods(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Pod 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name),
		})
		containerName := req.Container
		if containerName == "" && len(pod.Spec.Containers) > 0 {
			containerName = pod.Spec.Containers[0].Name
		}
		logs := ""
		if containerName != "" {
			logs = s.getPodLogs(ctx, clientset, pod.Namespace, pod.Name, containerName, req.TailLines)
		}
		evidence["pod"] = summarizePod(pod)
		evidence["events"] = summarizeEvents(events.Items, 20)
		evidence["container"] = containerName
		evidence["logs"] = sanitizeText(logs, 10000)
		return evidence, nil
	case "deployment":
		deploy, err := clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Deployment 失败: %w", err)
		}
		selector := metav1.FormatLabelSelector(deploy.Spec.Selector)
		pods, _ := clientset.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name),
		})
		podSummaries := make([]map[string]any, 0, len(pods.Items))
		for i := range pods.Items {
			podSummaries = append(podSummaries, summarizePod(&pods.Items[i]))
		}
		evidence["deployment"] = summarizeDeployment(deploy)
		evidence["selector"] = selector
		evidence["pods"] = podSummaries
		evidence["events"] = summarizeEvents(events.Items, 20)
		s.attachSamplePodLogs(ctx, clientset, evidence, pods.Items, req.Container, req.TailLines)
		return evidence, nil
	case "statefulset":
		item, err := clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 StatefulSet 失败: %w", err)
		}
		selector := metav1.FormatLabelSelector(item.Spec.Selector)
		pods, _ := clientset.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["statefulSet"] = summarizeStatefulSet(item)
		evidence["selector"] = selector
		evidence["pods"] = summarizePods(pods.Items)
		evidence["events"] = summarizeEvents(events.Items, 20)
		s.attachSamplePodLogs(ctx, clientset, evidence, pods.Items, req.Container, req.TailLines)
		return evidence, nil
	case "daemonset":
		item, err := clientset.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 DaemonSet 失败: %w", err)
		}
		selector := metav1.FormatLabelSelector(item.Spec.Selector)
		pods, _ := clientset.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["daemonSet"] = summarizeDaemonSet(item)
		evidence["selector"] = selector
		evidence["pods"] = summarizePods(pods.Items)
		evidence["events"] = summarizeEvents(events.Items, 20)
		s.attachSamplePodLogs(ctx, clientset, evidence, pods.Items, req.Container, req.TailLines)
		return evidence, nil
	case "job":
		item, err := clientset.BatchV1().Jobs(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Job 失败: %w", err)
		}
		selector := metav1.FormatLabelSelector(item.Spec.Selector)
		pods, _ := clientset.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["job"] = summarizeJob(item)
		evidence["selector"] = selector
		evidence["pods"] = summarizePods(pods.Items)
		evidence["events"] = summarizeEvents(events.Items, 20)
		s.attachSamplePodLogs(ctx, clientset, evidence, pods.Items, req.Container, req.TailLines)
		return evidence, nil
	case "cronjob":
		item, err := clientset.BatchV1().CronJobs(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 CronJob 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["cronJob"] = summarizeCronJob(item)
		evidence["events"] = summarizeEvents(events.Items, 20)
		return evidence, nil
	case "service":
		item, err := clientset.CoreV1().Services(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Service 失败: %w", err)
		}
		endpoints, _ := clientset.CoreV1().Endpoints(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["service"] = summarizeService(item)
		if endpoints != nil {
			evidence["endpoints"] = summarizeEndpoints(endpoints)
		}
		evidence["events"] = summarizeEvents(events.Items, 20)
		return evidence, nil
	case "ingress":
		item, err := clientset.NetworkingV1().Ingresses(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Ingress 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["ingress"] = summarizeIngress(item)
		evidence["events"] = summarizeEvents(events.Items, 20)
		return evidence, nil
	case "node":
		item, err := clientset.CoreV1().Nodes().Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Node 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		pods, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("spec.nodeName=%s", req.Name)})
		evidence["node"] = summarizeNode(item)
		evidence["pods"] = summarizePods(pods.Items)
		evidence["events"] = summarizeEvents(events.Items, 30)
		return evidence, nil
	case "namespace":
		item, err := clientset.CoreV1().Namespaces().Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Namespace 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events(req.Name).List(ctx, metav1.ListOptions{})
		evidence["namespace"] = summarizeNamespace(item)
		evidence["events"] = summarizeEvents(events.Items, 30)
		return evidence, nil
	case "configmap":
		item, err := clientset.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 ConfigMap 失败: %w", err)
		}
		evidence["configMap"] = summarizeConfigMap(item)
		return evidence, nil
	case "secret":
		item, err := clientset.CoreV1().Secrets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Secret 失败: %w", err)
		}
		evidence["secret"] = summarizeSecret(item)
		return evidence, nil
	case "persistentvolumeclaim":
		item, err := clientset.CoreV1().PersistentVolumeClaims(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 PVC 失败: %w", err)
		}
		events, _ := clientset.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.Name)})
		evidence["persistentVolumeClaim"] = summarizePVC(item)
		evidence["events"] = summarizeEvents(events.Items, 20)
		return evidence, nil
	case "persistentvolume":
		item, err := clientset.CoreV1().PersistentVolumes().Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 PV 失败: %w", err)
		}
		evidence["persistentVolume"] = summarizePV(item)
		return evidence, nil
	case "storageclass":
		item, err := clientset.StorageV1().StorageClasses().Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 StorageClass 失败: %w", err)
		}
		evidence["storageClass"] = summarizeStorageClass(item)
		return evidence, nil
	case "endpoints":
		item, err := clientset.CoreV1().Endpoints(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Endpoints 失败: %w", err)
		}
		evidence["endpoints"] = summarizeEndpoints(item)
		return evidence, nil
	case "networkpolicy":
		item, err := clientset.NetworkingV1().NetworkPolicies(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 NetworkPolicy 失败: %w", err)
		}
		evidence["networkPolicy"] = summarizeNetworkPolicy(item)
		return evidence, nil
	default:
		return nil, fmt.Errorf("暂不支持的诊断对象类型: %s", objectType)
	}
}

func isClusterScopeDiagnosisType(objectType string) bool {
	switch strings.ToLower(strings.TrimSpace(objectType)) {
	case "node", "namespace", "persistentvolume", "storageclass":
		return true
	default:
		return false
	}
}

func (s *Service) collectHostEvidence(ctx context.Context, hostID uint, focus string) (map[string]any, error) {
	var host assetmodel.Host
	if err := s.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return nil, fmt.Errorf("获取主机失败: %w", err)
	}
	statusText := "未知"
	switch host.Status {
	case 1:
		statusText = "在线"
	case 0:
		statusText = "离线"
	case -1:
		statusText = "未知"
	}
	collectMode := "ssh"
	if host.Type != "cloud" && host.AgentID != "" {
		collectMode = "agent"
	}
	agentFresh := false
	if host.AgentLastSeen != nil && time.Since(*host.AgentLastSeen) <= 2*time.Minute {
		agentFresh = true
	}
	risks := make([]string, 0)
	if host.Status != 1 {
		risks = append(risks, "主机状态不是在线")
	}
	if collectMode == "agent" && !agentFresh {
		risks = append(risks, "Agent 心跳可能已超时")
	}
	if host.CPUUsage >= 90 {
		risks = append(risks, "CPU 使用率超过 90%")
	} else if host.CPUUsage >= 70 {
		risks = append(risks, "CPU 使用率偏高")
	}
	if host.MemoryUsage >= 90 {
		risks = append(risks, "内存使用率超过 90%")
	} else if host.MemoryUsage >= 70 {
		risks = append(risks, "内存使用率偏高")
	}
	if host.DiskUsage >= 90 {
		risks = append(risks, "磁盘使用率超过 90%")
	} else if host.DiskUsage >= 80 {
		risks = append(risks, "磁盘使用率偏高")
	}
	return map[string]any{
		"objectType":  "host",
		"focus":       focus,
		"collectedAt": time.Now().Format(time.RFC3339),
		"host": map[string]any{
			"id":                   host.ID,
			"name":                 host.Name,
			"hostname":             host.Hostname,
			"ip":                   host.IP,
			"port":                 host.Port,
			"type":                 host.Type,
			"cloudProvider":        host.CloudProvider,
			"cloudInstanceId":      host.CloudInstanceID,
			"status":               host.Status,
			"statusText":           statusText,
			"tags":                 splitTags(host.Tags),
			"os":                   host.OS,
			"kernel":               host.Kernel,
			"arch":                 host.Arch,
			"uptime":               host.Uptime,
			"collectMode":          collectMode,
			"agentId":              host.AgentID,
			"agentVersion":         host.AgentVersion,
			"agentStatus":          host.AgentStatus,
			"agentLastSeen":        formatTimePtr(host.AgentLastSeen),
			"agentLastCollectAt":   formatTimePtr(host.AgentLastCollectAt),
			"agentHeartbeatFresh":  agentFresh,
			"lastSeen":             formatTimePtr(host.LastSeen),
			"cpuCores":             host.CPUCores,
			"cpuUsage":             roundFloat(host.CPUUsage),
			"memoryTotal":          host.MemoryTotal,
			"memoryUsed":           host.MemoryUsed,
			"memoryUsage":          roundFloat(host.MemoryUsage),
			"diskTotal":            host.DiskTotal,
			"diskUsed":             host.DiskUsed,
			"diskUsage":            roundFloat(host.DiskUsage),
			"resourceSummary":      hostResourceSummary(host),
			"detectedRiskSummary":  risks,
			"recommendedDataFresh": "如数据为空或时间过旧，请先在主机管理中执行采集信息或确认 Agent 在线。",
		},
	}, nil
}

func (s *Service) getPodLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName, container string, tailLines int64) string {
	if tailLines <= 0 {
		tailLines = 120
	}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  container,
		Timestamps: true,
		TailLines:  &tailLines,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "获取日志失败: " + err.Error()
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, 1024*1024))
	if err != nil {
		return "读取日志失败: " + err.Error()
	}
	return string(data)
}

func (s *Service) attachSamplePodLogs(ctx context.Context, clientset *kubernetes.Clientset, evidence map[string]any, pods []corev1.Pod, container string, tailLines int64) {
	if len(pods) == 0 {
		return
	}
	pod := pods[0]
	containerName := container
	if containerName == "" && len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}
	if containerName == "" {
		return
	}
	evidence["samplePod"] = pod.Name
	evidence["container"] = containerName
	evidence["logs"] = sanitizeText(s.getPodLogs(ctx, clientset, pod.Namespace, pod.Name, containerName, tailLines), 10000)
}

func summarizePod(pod *corev1.Pod) map[string]any {
	containers := make([]map[string]any, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		state := "unknown"
		reason := ""
		if c.State.Waiting != nil {
			state = "waiting"
			reason = c.State.Waiting.Reason + " " + c.State.Waiting.Message
		} else if c.State.Terminated != nil {
			state = "terminated"
			reason = c.State.Terminated.Reason + " " + c.State.Terminated.Message
		} else if c.State.Running != nil {
			state = "running"
		}
		containers = append(containers, map[string]any{
			"name":         c.Name,
			"ready":        c.Ready,
			"restartCount": c.RestartCount,
			"state":        state,
			"reason":       strings.TrimSpace(reason),
			"image":        c.Image,
		})
	}
	conditions := make([]map[string]any, 0, len(pod.Status.Conditions))
	for _, c := range pod.Status.Conditions {
		conditions = append(conditions, map[string]any{
			"type":    c.Type,
			"status":  c.Status,
			"reason":  c.Reason,
			"message": c.Message,
		})
	}
	return map[string]any{
		"name":       pod.Name,
		"namespace":  pod.Namespace,
		"phase":      pod.Status.Phase,
		"reason":     pod.Status.Reason,
		"message":    pod.Status.Message,
		"nodeName":   pod.Spec.NodeName,
		"podIP":      pod.Status.PodIP,
		"labels":     pod.Labels,
		"conditions": conditions,
		"containers": containers,
	}
}

func summarizeDeployment(deploy *appsv1.Deployment) map[string]any {
	conditions := make([]map[string]any, 0, len(deploy.Status.Conditions))
	for _, c := range deploy.Status.Conditions {
		conditions = append(conditions, map[string]any{
			"type":    c.Type,
			"status":  c.Status,
			"reason":  c.Reason,
			"message": c.Message,
		})
	}
	containers := make([]map[string]any, 0, len(deploy.Spec.Template.Spec.Containers))
	for _, c := range deploy.Spec.Template.Spec.Containers {
		containers = append(containers, map[string]any{
			"name":  c.Name,
			"image": c.Image,
		})
	}
	return map[string]any{
		"name":                deploy.Name,
		"namespace":           deploy.Namespace,
		"replicas":            valueInt32(deploy.Spec.Replicas),
		"updatedReplicas":     deploy.Status.UpdatedReplicas,
		"readyReplicas":       deploy.Status.ReadyReplicas,
		"availableReplicas":   deploy.Status.AvailableReplicas,
		"unavailableReplicas": deploy.Status.UnavailableReplicas,
		"observedGeneration":  deploy.Status.ObservedGeneration,
		"generation":          deploy.Generation,
		"conditions":          conditions,
		"containers":          containers,
	}
}

func summarizeStatefulSet(item *appsv1.StatefulSet) map[string]any {
	return map[string]any{
		"name":              item.Name,
		"namespace":         item.Namespace,
		"replicas":          valueInt32(item.Spec.Replicas),
		"readyReplicas":     item.Status.ReadyReplicas,
		"currentReplicas":   item.Status.CurrentReplicas,
		"updatedReplicas":   item.Status.UpdatedReplicas,
		"availableReplicas": item.Status.AvailableReplicas,
		"serviceName":       item.Spec.ServiceName,
		"updateRevision":    item.Status.UpdateRevision,
		"currentRevision":   item.Status.CurrentRevision,
	}
}

func summarizeDaemonSet(item *appsv1.DaemonSet) map[string]any {
	return map[string]any{
		"name":                   item.Name,
		"namespace":              item.Namespace,
		"desiredNumberScheduled": item.Status.DesiredNumberScheduled,
		"currentNumberScheduled": item.Status.CurrentNumberScheduled,
		"numberReady":            item.Status.NumberReady,
		"numberAvailable":        item.Status.NumberAvailable,
		"numberUnavailable":      item.Status.NumberUnavailable,
		"updatedNumberScheduled": item.Status.UpdatedNumberScheduled,
	}
}

func summarizeJob(item *batchv1.Job) map[string]any {
	conditions := make([]map[string]any, 0, len(item.Status.Conditions))
	for _, c := range item.Status.Conditions {
		conditions = append(conditions, map[string]any{
			"type":    c.Type,
			"status":  c.Status,
			"reason":  c.Reason,
			"message": c.Message,
		})
	}
	return map[string]any{
		"name":        item.Name,
		"namespace":   item.Namespace,
		"parallelism": valueInt32(item.Spec.Parallelism),
		"completions": valueInt32(item.Spec.Completions),
		"active":      item.Status.Active,
		"succeeded":   item.Status.Succeeded,
		"failed":      item.Status.Failed,
		"conditions":  conditions,
	}
}

func summarizeCronJob(item *batchv1.CronJob) map[string]any {
	active := make([]string, 0, len(item.Status.Active))
	for _, ref := range item.Status.Active {
		active = append(active, ref.Name)
	}
	return map[string]any{
		"name":               item.Name,
		"namespace":          item.Namespace,
		"schedule":           item.Spec.Schedule,
		"suspend":            valueBool(item.Spec.Suspend),
		"activeJobs":         active,
		"lastScheduleTime":   formatMetaTime(item.Status.LastScheduleTime),
		"lastSuccessfulTime": formatMetaTime(item.Status.LastSuccessfulTime),
	}
}

func summarizeService(item *corev1.Service) map[string]any {
	ports := make([]map[string]any, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		ports = append(ports, map[string]any{
			"name":       port.Name,
			"protocol":   port.Protocol,
			"port":       port.Port,
			"targetPort": port.TargetPort.String(),
			"nodePort":   port.NodePort,
		})
	}
	return map[string]any{
		"name":        item.Name,
		"namespace":   item.Namespace,
		"type":        item.Spec.Type,
		"clusterIP":   item.Spec.ClusterIP,
		"externalIPs": item.Spec.ExternalIPs,
		"selector":    item.Spec.Selector,
		"ports":       ports,
	}
}

func summarizeEndpoints(item *corev1.Endpoints) map[string]any {
	subsets := make([]map[string]any, 0, len(item.Subsets))
	for _, subset := range item.Subsets {
		addresses := make([]string, 0, len(subset.Addresses))
		for _, address := range subset.Addresses {
			addresses = append(addresses, address.IP)
		}
		notReady := make([]string, 0, len(subset.NotReadyAddresses))
		for _, address := range subset.NotReadyAddresses {
			notReady = append(notReady, address.IP)
		}
		ports := make([]string, 0, len(subset.Ports))
		for _, port := range subset.Ports {
			ports = append(ports, fmt.Sprintf("%s/%d", port.Protocol, port.Port))
		}
		subsets = append(subsets, map[string]any{"addresses": addresses, "notReadyAddresses": notReady, "ports": ports})
	}
	return map[string]any{"name": item.Name, "namespace": item.Namespace, "subsets": subsets}
}

func summarizeIngress(item *networkingv1.Ingress) map[string]any {
	rules := make([]map[string]any, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		paths := make([]map[string]any, 0)
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				backend := ""
				if path.Backend.Service != nil {
					backend = fmt.Sprintf("%s:%s", path.Backend.Service.Name, path.Backend.Service.Port.String())
				}
				paths = append(paths, map[string]any{"path": path.Path, "pathType": path.PathType, "backend": backend})
			}
		}
		rules = append(rules, map[string]any{"host": rule.Host, "paths": paths})
	}
	tls := make([]map[string]any, 0, len(item.Spec.TLS))
	for _, itemTLS := range item.Spec.TLS {
		tls = append(tls, map[string]any{"hosts": itemTLS.Hosts, "secretName": itemTLS.SecretName})
	}
	className := ""
	if item.Spec.IngressClassName != nil {
		className = *item.Spec.IngressClassName
	}
	return map[string]any{"name": item.Name, "namespace": item.Namespace, "ingressClass": className, "rules": rules, "tls": tls}
}

func summarizeNamespace(item *corev1.Namespace) map[string]any {
	conditions := make([]map[string]any, 0, len(item.Status.Conditions))
	for _, condition := range item.Status.Conditions {
		conditions = append(conditions, map[string]any{
			"type":    condition.Type,
			"status":  condition.Status,
			"reason":  condition.Reason,
			"message": condition.Message,
		})
	}
	return map[string]any{
		"name":              item.Name,
		"phase":             item.Status.Phase,
		"labels":            item.Labels,
		"annotations":       item.Annotations,
		"conditions":        conditions,
		"deletionTimestamp": formatMetaTimePtr(item.DeletionTimestamp),
	}
}

func summarizeConfigMap(item *corev1.ConfigMap) map[string]any {
	keys := make([]string, 0, len(item.Data)+len(item.BinaryData))
	for key := range item.Data {
		keys = append(keys, key)
	}
	for key := range item.BinaryData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return map[string]any{
		"name":            item.Name,
		"namespace":       item.Namespace,
		"labels":          item.Labels,
		"dataKeys":        keys,
		"dataCount":       len(item.Data),
		"binaryDataCount": len(item.BinaryData),
	}
}

func summarizeSecret(item *corev1.Secret) map[string]any {
	keys := make([]string, 0, len(item.Data)+len(item.StringData))
	for key := range item.Data {
		keys = append(keys, key)
	}
	for key := range item.StringData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return map[string]any{
		"name":      item.Name,
		"namespace": item.Namespace,
		"type":      item.Type,
		"labels":    item.Labels,
		"dataKeys":  keys,
		"dataCount": len(item.Data),
	}
}

func summarizePVC(item *corev1.PersistentVolumeClaim) map[string]any {
	return map[string]any{
		"name":         item.Name,
		"namespace":    item.Namespace,
		"phase":        item.Status.Phase,
		"volumeName":   item.Spec.VolumeName,
		"storageClass": valueStringPtr(item.Spec.StorageClassName),
		"accessModes":  item.Spec.AccessModes,
		"capacity":     item.Status.Capacity,
		"requests":     item.Spec.Resources.Requests,
		"conditions":   item.Status.Conditions,
		"labels":       item.Labels,
	}
}

func summarizePV(item *corev1.PersistentVolume) map[string]any {
	claim := ""
	if item.Spec.ClaimRef != nil {
		claim = fmt.Sprintf("%s/%s", item.Spec.ClaimRef.Namespace, item.Spec.ClaimRef.Name)
	}
	return map[string]any{
		"name":          item.Name,
		"phase":         item.Status.Phase,
		"reason":        item.Status.Reason,
		"capacity":      item.Spec.Capacity,
		"accessModes":   item.Spec.AccessModes,
		"reclaimPolicy": item.Spec.PersistentVolumeReclaimPolicy,
		"storageClass":  item.Spec.StorageClassName,
		"claim":         claim,
		"labels":        item.Labels,
	}
}

func summarizeStorageClass(item *storagev1.StorageClass) map[string]any {
	return map[string]any{
		"name":                 item.Name,
		"provisioner":          item.Provisioner,
		"reclaimPolicy":        valueReclaimPolicyPtr(item.ReclaimPolicy),
		"volumeBindingMode":    valueVolumeBindingModePtr(item.VolumeBindingMode),
		"allowVolumeExpansion": valueBoolPtr(item.AllowVolumeExpansion),
		"parameters":           item.Parameters,
		"labels":               item.Labels,
	}
}

func summarizeNetworkPolicy(item *networkingv1.NetworkPolicy) map[string]any {
	policyTypes := make([]string, 0, len(item.Spec.PolicyTypes))
	for _, policyType := range item.Spec.PolicyTypes {
		policyTypes = append(policyTypes, string(policyType))
	}
	return map[string]any{
		"name":        item.Name,
		"namespace":   item.Namespace,
		"podSelector": item.Spec.PodSelector.MatchLabels,
		"policyTypes": policyTypes,
		"ingressRule": len(item.Spec.Ingress),
		"egressRule":  len(item.Spec.Egress),
		"labels":      item.Labels,
	}
}

func summarizeNode(item *corev1.Node) map[string]any {
	conditions := make([]map[string]any, 0, len(item.Status.Conditions))
	for _, c := range item.Status.Conditions {
		conditions = append(conditions, map[string]any{"type": c.Type, "status": c.Status, "reason": c.Reason, "message": c.Message})
	}
	taints := make([]map[string]any, 0, len(item.Spec.Taints))
	for _, taint := range item.Spec.Taints {
		taints = append(taints, map[string]any{"key": taint.Key, "value": taint.Value, "effect": taint.Effect})
	}
	return map[string]any{
		"name":          item.Name,
		"unschedulable": item.Spec.Unschedulable,
		"podCIDR":       item.Spec.PodCIDR,
		"providerID":    item.Spec.ProviderID,
		"labels":        item.Labels,
		"taints":        taints,
		"conditions":    conditions,
		"capacity":      item.Status.Capacity,
		"allocatable":   item.Status.Allocatable,
	}
}

func summarizePods(pods []corev1.Pod) []map[string]any {
	items := make([]map[string]any, 0, len(pods))
	for i := range pods {
		items = append(items, summarizePod(&pods[i]))
	}
	return items
}

func summarizeEvents(events []corev1.Event, limit int) []map[string]any {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].LastTimestamp.Time.After(events[j].LastTimestamp.Time)
	})
	if len(events) > limit {
		events = events[:limit]
	}
	result := make([]map[string]any, 0, len(events))
	for _, e := range events {
		result = append(result, map[string]any{
			"type":    e.Type,
			"reason":  e.Reason,
			"message": e.Message,
			"count":   e.Count,
			"time":    e.LastTimestamp.Time.Format(time.RFC3339),
		})
	}
	return result
}

func buildLogEvidence(logs, source string) map[string]any {
	lines := splitLines(logs)
	keywords := []string{"error", "exception", "failed", "panic", "fatal", "oom", "timeout", "refused", "denied", "unauthorized", "forbidden", "not found"}
	hits := make([]string, 0)
	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				hits = append(hits, line)
				break
			}
		}
		if len(hits) >= 30 {
			break
		}
	}
	return map[string]any{
		"source":       source,
		"lineCount":    len(lines),
		"errorSamples": hits,
		"tail":         tailLines(lines, 80),
	}
}

func assistantSystemPrompt() string {
	return "你是 OpsHub 智能运维助手。请使用中文回答，面向运维和开发人员，回答要直接、结构化、可执行。你可以基于传入的 OpsHub 只读平台上下文回答主机、Kubernetes 集群、监控告警、SSL 证书等查询，但不能编造上下文中不存在的数据。涉及危险命令或变更操作时必须提示风险，并要求人工确认。不要输出模型内部思考链或 <think> 内容。"
}

func diagnosisSystemPrompt() string {
	return "你是资深 SRE 和 Kubernetes 排障专家。请基于给定上下文诊断，不要臆造事实。输出中文，包含结论、证据、建议和推荐命令。涉及删除、重启、扩缩容、修改配置时必须提示风险和人工确认。"
}

func commandSystemPrompt() string {
	return "你是 OpsHub AI 任务助手。你只能生成命令或脚本草稿，不能声称已经执行。请用中文说明用途、风险、执行前检查项。优先给只读命令，危险操作必须提示人工确认。"
}

func continuationSystemPrompt() string {
	return "你是长文档续写助手。用户已经拿到前半段回答，现在只需要从中断处继续写。必须使用中文。严禁重新开始、严禁跳章、严禁重复已输出内容、严禁输出与上一段无关的平台分析过程。请根据上一段回答的末尾判断应该继续的小节、代码块或句子。"
}

func buildContinuationPrompt(originalQuestion, previousAnswerTail, anchor string) string {
	return fmt.Sprintf(`请从上一段回答真正中断的位置继续输出。

原始问题：
%s

系统检测到的上一段末尾位置：
%s

上一段回答末尾：
%s

续写要求：
1. 只输出后续正文，不要解释你正在续写。
2. 不要重复“上一段回答末尾”中已经出现的标题、段落或命令。
3. 如果上一段停在未完成的标题、小节、句子、表格或代码块中，请先补完这个位置。
4. 小节编号必须沿用“系统检测到的上一段末尾位置”，不要跳到后面的章节。
5. 如果上一段末尾已经打开了代码块，请继续代码内容并在合适位置闭合代码块。
6. 如果上一段末尾停在 5.2，就必须从 5.2 未完成处继续，不能直接跳到 6、7、8 或重新生成目录。
7. 如果无法准确判断中断点，请从上一段最后一个未完成小节继续，而不是重新规划目录。`, defaultString(originalQuestion, "用户要求生成长文档"), defaultString(anchor, "未识别到明确编号，请以上一段末尾文字判断"), defaultString(previousAnswerTail, "上一段内容为空"))
}

func isContinuePrompt(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	keywords := []string{"继续生成", "继续输出", "接着输出", "接着写", "从上次中断", "不要重复已经输出", "continue"}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func isLengthFinishReason(reason string) bool {
	return strings.EqualFold(strings.TrimSpace(reason), "length")
}

func continuationAnchor(value string) string {
	lines := strings.Split(value, "\n")
	sectionPattern := regexp.MustCompile(`^\s*(?:#{1,6}\s*)?(\d+(?:\.\d+)*)(?:[\.、．\s-]+)(.+)$`)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(strings.Trim(lines[i], "*`# "))
		if line == "" {
			continue
		}
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			return strings.TrimSpace(matches[1] + " " + matches[2])
		}
	}
	return ""
}

func cleanContinuationAnswer(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	markers := []string{
		"> 本次流式生成中断",
		"> 本次回答已到达模型最大输出长度",
		"生成被中断，当前内容已作为本地草稿保留",
		"回答被模型最大输出长度截断",
	}
	removedRuntimeNotice := false
	for _, marker := range markers {
		if idx := strings.Index(text, marker); idx >= 0 {
			text = strings.TrimSpace(text[:idx])
			removedRuntimeNotice = true
		}
	}
	if removedRuntimeNotice {
		text = removeSyntheticFenceClosure(text)
	}
	return text
}

func removeSyntheticFenceClosure(value string) string {
	text := strings.TrimRight(value, " \t\r\n")
	if !strings.HasSuffix(text, "```") {
		return value
	}
	withoutFence := strings.TrimRight(strings.TrimSuffix(text, "```"), " \t\r\n")
	if strings.Count(withoutFence, "```")%2 == 1 {
		return withoutFence
	}
	return value
}

func fallbackChatForPlan(question string, plan chatExecutionPlan, cause error) *ChatResult {
	if plan.ToolName == "continue_ai_answer" {
		answer := fmt.Sprintf("续写时模型调用失败，已保留前面生成的内容。请稍后点击“继续生成”重试。\n\n失败原因：%s", cause.Error())
		return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
	}
	if contextValue, ok := plan.Context.(assistantPlatformContext); ok {
		return fallbackChat(question, contextValue, cause)
	}
	if agentTrace, ok := plan.Context.(opsHubAgentTrace); ok {
		return fallbackAgentChat(question, agentTrace, cause)
	}
	return fallbackChat(question, assistantPlatformContext{}, cause)
}

func fallbackChat(question string, platformContext assistantPlatformContext, cause error) *ChatResult {
	contextSummary := fallbackPlatformContextSummary(platformContext)
	answer := fmt.Sprintf(`当前未能调用外部 AI 模型，我先给出本地助手建议。

问题摘要：%s
%s

建议：
1. 如果这是排障问题，优先收集对象状态、最近事件、最近日志和相关变更记录。
2. 如果需要 AI 深度分析，请先到“AI 配置”中配置 OpenAI-compatible 模型，例如 DeepSeek、通义千问、OpenAI 或 Ollama。
3. 以上平台数据只来自 OpsHub 当前数据库的只读摘要，不包含凭证、Token 或私钥。
4. 模型调用失败原因：%s`, truncateText(question, 300), contextSummary, cause.Error())
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}

func emitTextDeltas(content string, emit func(ChatStreamEvent) error) error {
	runes := []rune(content)
	const chunkSize = 18
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		if err := emit(ChatStreamEvent{Type: "delta", Delta: string(runes[start:end])}); err != nil {
			return err
		}
	}
	return nil
}

func fallbackPlatformContextSummary(platformContext assistantPlatformContext) string {
	if platformContext.Intent == "general" {
		return ""
	}
	lines := []string{"", "平台只读数据摘要："}
	if platformContext.HostSummary != nil {
		if total, ok := platformContext.HostSummary["total"]; ok {
			lines = append(lines, fmt.Sprintf("- 主机总数：%v 台", total))
		}
		if totalCPU, ok := platformContext.HostSummary["totalCPU"]; ok {
			lines = append(lines, fmt.Sprintf("- 主机资源：CPU %v 核，内存 %v，磁盘 %v", totalCPU, platformContext.HostSummary["totalMemory"], platformContext.HostSummary["totalDisk"]))
		}
	}
	if platformContext.ClusterSummary != nil {
		lines = append(lines, fmt.Sprintf("- Kubernetes 集群：%v 个，节点 %v 个，Pod %v 个", platformContext.ClusterSummary["total"], platformContext.ClusterSummary["totalNodes"], platformContext.ClusterSummary["totalPods"]))
	}
	if platformContext.AlertSummary != nil {
		lines = append(lines, fmt.Sprintf("- 监控告警：总计 %v 条，等级分布 %v，状态分布 %v", platformContext.AlertSummary["total"], platformContext.AlertSummary["severityCount"], platformContext.AlertSummary["stateCount"]))
	}
	if platformContext.CertSummary != nil {
		lines = append(lines, fmt.Sprintf("- SSL 证书：样本 %v 张，状态分布 %v", platformContext.CertSummary["sample"], platformContext.CertSummary["statusCount"]))
	}
	if len(platformContext.Errors) > 0 {
		lines = append(lines, "- 数据查询提示："+strings.Join(platformContext.Errors, "；"))
	}
	return strings.Join(lines, "\n")
}

func fallbackHostDiagnosis(evidence map[string]any, cause error) *ChatResult {
	data, _ := json.MarshalIndent(evidence, "", "  ")
	answer := fmt.Sprintf(`主机本地诊断结果：

健康结论：
已完成主机资产和资源指标上下文收集，但未能调用外部 AI 模型进行深度推理。

建议处理：
1. 优先确认主机在线状态、最近采集时间和 Agent 心跳是否正常。
2. CPU、内存、磁盘任一使用率超过 90%% 时，建议先排查高占用进程、异常日志和容量增长趋势。
3. 如果资源指标为空或明显过旧，请先在主机管理中执行采集信息，或确认 Agent 是否在线。
4. 需要执行清理、重启、删除等操作时，必须先确认业务影响范围和回滚方案。

模型调用失败原因：%s

Evidence:
%s`, cause.Error(), truncateText(string(data), 12000))
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}

func fallbackLogAnalysis(logs string, cause error) *ChatResult {
	evidence := buildLogEvidence(logs, "local")
	samples, _ := json.MarshalIndent(evidence["errorSamples"], "", "  ")
	answer := fmt.Sprintf(`日志本地分析结果：

异常摘要：
检测到 %d 行日志，其中命中错误关键词的样本如下。

证据样本：
%s

处理建议：
1. 先按时间点确认错误是否集中爆发，关联发布、配置变更或流量突增。
2. 如果存在 exception、panic、timeout、refused 等关键词，优先检查应用堆栈、依赖服务、网络连通性和配置。
3. 建议补充完整上下文后重新分析，或在 AI 配置中配置模型获得更完整结论。

模型调用失败原因：%s`, len(splitLines(logs)), string(samples), cause.Error())
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}

func fallbackKubernetesDiagnosis(objectType string, evidence map[string]any, cause error) *ChatResult {
	data, _ := json.MarshalIndent(evidence, "", "  ")
	answer := fmt.Sprintf(`Kubernetes %s 本地诊断结果：

健康结论：
已完成对象状态、事件和日志上下文收集，但未能调用外部 AI 模型进行深度推理。

建议处理：
1. 优先查看 Evidence 中的 phase、conditions、container state、restartCount 和 events。
2. 如果 Pod 为 CrashLoopBackOff，重点看容器日志、启动命令、环境变量和探针配置。
3. 如果 Deployment 不可用，重点看 readyReplicas、unavailableReplicas、selector 匹配的 Pod 状态。
4. 配置 AI 模型后可获得更完整的根因排序和操作建议。

模型调用失败原因：%s

Evidence:
%s`, objectType, cause.Error(), truncateText(string(data), 12000))
	return &ChatResult{Content: answer, Model: "local-fallback", Fallback: true}
}

func sanitizeText(input string, maxLen int) string {
	text := strings.TrimSpace(input)
	secretPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pwd|token|secret|access[_-]?key|api[_-]?key)\s*[:=]\s*['"]?[^'"\s]+`),
		regexp.MustCompile(`(?i)authorization:\s*bearer\s+[a-z0-9._\-]+`),
	}
	for _, re := range secretPatterns {
		text = re.ReplaceAllString(text, "$1: ******")
	}
	return truncateText(text, maxLen)
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "********"
	}
	return secret[:4] + "********" + secret[len(secret)-4:]
}

func isMaskedSecret(secret string) bool {
	return strings.Contains(secret, "********") || strings.Trim(secret, "*") == ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func titleFromText(text, fallback string) string {
	title := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if title == "" {
		return fallback
	}
	return truncateText(title, 36)
}

func truncateText(text string, max int) string {
	text = strings.ToValidUTF8(text, "")
	if max <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "\n...（内容已截断）"
}

func stripMarkdown(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "#", ""), "*", "")
}

func extractSuggestions(text string) string {
	if idx := strings.Index(text, "建议"); idx >= 0 {
		return truncateText(text[idx:], 2000)
	}
	return truncateText(text, 1000)
}

func splitLines(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func tailLines(lines []string, limit int) []string {
	if len(lines) <= limit {
		return lines
	}
	return lines[len(lines)-limit:]
}

func splitTags(tags string) []string {
	parts := strings.Split(tags, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatMetaTime(t *metav1.Time) string {
	if t == nil {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

func formatMetaTimePtr(t *metav1.Time) string {
	return formatMetaTime(t)
}

func valueBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func valueBoolPtr(v *bool) bool {
	return valueBool(v)
}

func valueStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueReclaimPolicyPtr(v *corev1.PersistentVolumeReclaimPolicy) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func valueVolumeBindingModePtr(v *storagev1.VolumeBindingMode) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func decryptCredentialForAI(credential *assetmodel.Credential) error {
	var err error
	if credential.Password != "" {
		credential.Password, err = decryptAISecret(credential.Password)
		if err != nil {
			return fmt.Errorf("解密密码失败: %w", err)
		}
	}
	if credential.PrivateKey != "" {
		credential.PrivateKey, err = decryptAISecret(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密私钥失败: %w", err)
		}
	}
	if credential.Passphrase != "" {
		credential.Passphrase, err = decryptAISecret(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("解密私钥密码失败: %w", err)
		}
	}
	return nil
}

func decryptAISecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte("opshub-enc-key-32-bytes-long!!!!"))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func newAIHostSSHClient(host assetmodel.Host, credential assetmodel.Credential) (*sshclient.Client, error) {
	var privateKey []byte
	if credential.Type == "key" {
		privateKey = []byte(credential.PrivateKey)
	}
	client, err := sshclient.NewClient(host.IP, host.Port, host.SSHUser, credential.Password, privateKey, credential.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("创建 SSH 客户端失败: %w", err)
	}
	return client, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func roundFloat(v float64) float64 {
	return float64(int(v*100)) / 100
}

func hostResourceSummary(host assetmodel.Host) string {
	parts := []string{
		fmt.Sprintf("CPU %.1f%% / %d核", host.CPUUsage, host.CPUCores),
		fmt.Sprintf("内存 %.1f%%", host.MemoryUsage),
		fmt.Sprintf("磁盘 %.1f%%", host.DiskUsage),
	}
	return strings.Join(parts, "，")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func valueInt32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

var errNoClientset = errors.New("clientset unavailable")

var _ = errNoClientset
