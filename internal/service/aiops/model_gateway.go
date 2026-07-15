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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	aiopsdata "github.com/ydcloud-dy/opshub/internal/data/aiops"
)

const (
	defaultAIModelMaxTokens      = 8192
	defaultAIModelTimeoutSeconds = 180
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResult struct {
	Content      string
	Model        string
	TokensIn     int
	TokensOut    int
	LatencyMS    int64
	Fallback     bool
	FinishReason string
}

type openAIChatRequest struct {
	Model           string        `json:"model"`
	Messages        []ChatMessage `json:"messages"`
	Temperature     float64       `json:"temperature,omitempty"`
	MaxTokens       int           `json:"max_tokens,omitempty"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
	Stream          bool          `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type openAIChatStreamResponse struct {
	Choices []struct {
		Delta        ChatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func callModel(ctx context.Context, provider *aiopsdata.Provider, messages []ChatMessage) (*ChatResult, error) {
	start := time.Now()
	if provider == nil || !provider.Enabled || strings.TrimSpace(provider.APIKey) == "" || strings.TrimSpace(provider.BaseURL) == "" {
		return nil, fmt.Errorf("未配置可用的 AI 模型，请先在 AI 配置中配置 OpenAI-compatible 模型")
	}

	timeout := normalizedModelTimeout(provider)

	reqBody := openAIChatRequest{
		Model:           provider.Model,
		Messages:        messages,
		Temperature:     provider.Temperature,
		MaxTokens:       provider.MaxTokens,
		ReasoningEffort: normalizeReasoningEffort(provider.ReasoningEffort),
		Stream:          false,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	client := newModelHTTPClient(timeout, false)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("解析模型响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("模型调用失败: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("模型调用失败，HTTP状态码: %d", resp.StatusCode)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("模型调用失败: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("模型响应为空")
	}

	return &ChatResult{
		Content:      strings.TrimSpace(parsed.Choices[0].Message.Content),
		Model:        provider.Model,
		TokensIn:     parsed.Usage.PromptTokens,
		TokensOut:    parsed.Usage.CompletionTokens,
		LatencyMS:    time.Since(start).Milliseconds(),
		FinishReason: parsed.Choices[0].FinishReason,
	}, nil
}

func callModelStream(ctx context.Context, provider *aiopsdata.Provider, messages []ChatMessage, onDelta func(string) error) (*ChatResult, error) {
	start := time.Now()
	if provider == nil || !provider.Enabled || strings.TrimSpace(provider.APIKey) == "" || strings.TrimSpace(provider.BaseURL) == "" {
		return nil, fmt.Errorf("未配置可用的 AI 模型，请先在 AI 配置中配置 OpenAI-compatible 模型")
	}
	if onDelta == nil {
		return nil, fmt.Errorf("未配置流式输出处理器")
	}

	timeout := normalizedModelTimeout(provider)

	reqBody := openAIChatRequest{
		Model:           provider.Model,
		Messages:        messages,
		Temperature:     provider.Temperature,
		MaxTokens:       provider.MaxTokens,
		ReasoningEffort: normalizeReasoningEffort(provider.ReasoningEffort),
		Stream:          true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	client := newModelHTTPClient(timeout, true)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		if readErr != nil {
			return nil, readErr
		}
		var parsed openAIChatResponse
		if err := json.Unmarshal(respBytes, &parsed); err == nil && parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("模型调用失败: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("模型调用失败，HTTP状态码: %d", resp.StatusCode)
	}

	var content strings.Builder
	var tokensIn int
	var tokensOut int
	var finishReason string
	sawDone := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 16*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			sawDone = true
			break
		}

		var parsed openAIChatStreamResponse
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			continue
		}
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("模型调用失败: %s", parsed.Error.Message)
		}
		if parsed.Usage.PromptTokens > 0 {
			tokensIn = parsed.Usage.PromptTokens
		}
		if parsed.Usage.CompletionTokens > 0 {
			tokensOut = parsed.Usage.CompletionTokens
		}
		if len(parsed.Choices) == 0 {
			continue
		}
		if parsed.Choices[0].FinishReason != "" {
			finishReason = parsed.Choices[0].FinishReason
		}
		delta := parsed.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		content.WriteString(delta)
		if err := onDelta(delta); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取模型流式响应失败: %w", err)
	}

	finalContent := strings.TrimSpace(content.String())
	if finalContent == "" {
		return nil, fmt.Errorf("模型响应为空")
	}
	if !sawDone && strings.TrimSpace(finishReason) == "" {
		return nil, fmt.Errorf("模型流式响应提前结束，未收到完成标记 [DONE] 或 finish_reason")
	}

	return &ChatResult{
		Content:      finalContent,
		Model:        provider.Model,
		TokensIn:     tokensIn,
		TokensOut:    tokensOut,
		LatencyMS:    time.Since(start).Milliseconds(),
		FinishReason: finishReason,
	}, nil
}

func normalizedModelTimeout(provider *aiopsdata.Provider) int {
	if provider != nil && provider.Timeout > 0 {
		return provider.Timeout
	}
	return defaultAIModelTimeoutSeconds
}

func newModelHTTPClient(timeoutSeconds int, stream bool) *http.Client {
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultAIModelTimeoutSeconds
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if !stream {
		return &http.Client{Timeout: timeout}
	}

	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		streamTransport := transport.Clone()
		streamTransport.ResponseHeaderTimeout = timeout
		return &http.Client{Transport: streamTransport}
	}
	return &http.Client{}
}

func normalizeReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal", "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}
