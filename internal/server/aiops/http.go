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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	rbacbiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	aiopsservice "github.com/ydcloud-dy/opshub/internal/service/aiops"
	rbacservice "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"gorm.io/gorm"
)

type HTTPServer struct {
	service        *aiopsservice.Service
	authMiddleware *rbacservice.AuthMiddleware
}

func NewHTTPServer(db *gorm.DB, authMiddleware *rbacservice.AuthMiddleware) *HTTPServer {
	return &HTTPServer{
		service:        aiopsservice.NewService(db),
		authMiddleware: authMiddleware,
	}
}

func (s *HTTPServer) RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/aiops")
	{
		group.GET("/providers/options", s.ListProviderOptions)

		providers := group.Group("/providers")
		providers.Use(s.authMiddleware.RequireAdmin())
		{
			providers.GET("", s.ListProviders)
			providers.POST("", s.SaveProvider)
			providers.PUT("/:id", s.SaveProvider)
			providers.DELETE("/:id", s.DeleteProvider)
			providers.POST("/:id/test", s.TestProvider)
		}

		group.POST("/chat", s.Chat)
		group.POST("/chat/stream", s.ChatStream)
		group.POST("/chat/stop", s.StopChat)
		group.POST("/logs/analyze", s.AnalyzeLogs)
		group.POST("/diagnosis/kubernetes", s.DiagnoseKubernetes)
		group.POST("/diagnosis/hosts/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionView),
			s.DiagnoseHost)
		group.GET("/alerts/events", s.ListAlertEvents)
		group.POST("/alerts/analyze", s.AnalyzeAlertRootCause)
		group.GET("/alerts/analyses", s.ListRootCauseAnalyses)
		group.GET("/sessions", s.ListSessions)
		group.GET("/sessions/:id", s.GetSession)
		group.DELETE("/sessions/:id", s.DeleteSession)
		group.GET("/diagnosis/tasks", s.ListDiagnosisTasks)
	}
}

func (s *HTTPServer) ListProviders(c *gin.Context) {
	items, err := s.service.ListProviders(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询AI配置失败: "+err.Error())
		return
	}
	response.Success(c, items)
}

func (s *HTTPServer) ListProviderOptions(c *gin.Context) {
	items, err := s.service.ListProviderOptions(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询AI模型列表失败: "+err.Error())
		return
	}
	response.Success(c, items)
}

func (s *HTTPServer) SaveProvider(c *gin.Context) {
	var req aiopsservice.ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	id := parseUintParam(c, "id")
	item, err := s.service.SaveProvider(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存AI配置失败: "+err.Error())
		return
	}
	response.Success(c, item)
}

func (s *HTTPServer) DeleteProvider(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的配置ID")
		return
	}
	if err := s.service.DeleteProvider(c.Request.Context(), id); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除AI配置失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (s *HTTPServer) TestProvider(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的配置ID")
		return
	}
	result, err := s.service.TestProvider(c.Request.Context(), id)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) Chat(c *gin.Context) {
	var req aiopsservice.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	result, err := s.service.Chat(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "AI问答失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) ChatStream(c *gin.Context) {
	var req aiopsservice.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	// Some proxies/browsers buffer tiny SSE frames. An initial padded comment makes
	// the stream visible immediately, so meta events can update the UI in real time.
	_, _ = fmt.Fprintf(c.Writer, ": %s\n\n", strings.Repeat(" ", 2048))
	c.Writer.Flush()

	var emitMu sync.Mutex
	emit := func(event aiopsservice.ChatStreamEvent) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		emitMu.Lock()
		defer emitMu.Unlock()
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-done:
				return
			case <-ticker.C:
				_ = emit(aiopsservice.ChatStreamEvent{Type: "ping"})
			}
		}
	}()

	if err := s.service.ChatStream(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req, emit); err != nil {
		_ = emit(aiopsservice.ChatStreamEvent{Type: "error", Error: "AI问答失败: " + err.Error()})
	}
}

func (s *HTTPServer) StopChat(c *gin.Context) {
	var req aiopsservice.StopChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if err := s.service.StopChat(c.Request.Context(), rbacservice.GetUserID(c), req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "停止生成失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"stopped": true})
}

func (s *HTTPServer) AnalyzeLogs(c *gin.Context) {
	var req aiopsservice.LogAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	result, err := s.service.AnalyzeLogs(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "日志分析失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) DiagnoseKubernetes(c *gin.Context) {
	var req aiopsservice.DiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	result, err := s.service.DiagnoseKubernetes(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "智能诊断失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) DiagnoseHost(c *gin.Context) {
	hostID := parseUintParam(c, "id")
	if hostID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}
	var req aiopsservice.HostDiagnoseRequest
	_ = c.ShouldBindJSON(&req)
	req.HostID = hostID
	result, err := s.service.DiagnoseHost(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "主机智能诊断失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) ListSessions(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "pageSize", 20)
	sessionType := c.Query("type")
	items, total, err := s.service.ListSessions(c.Request.Context(), rbacservice.GetUserID(c), page, pageSize, sessionType)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询会话失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, items)
}

func (s *HTTPServer) GetSession(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}
	session, messages, tools, err := s.service.GetSession(c.Request.Context(), rbacservice.GetUserID(c), id)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		return
	}
	response.Success(c, gin.H{
		"session":  session,
		"messages": messages,
		"tools":    tools,
	})
}

func (s *HTTPServer) DeleteSession(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}
	if err := s.service.DeleteSession(c.Request.Context(), rbacservice.GetUserID(c), id); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除会话失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (s *HTTPServer) ListDiagnosisTasks(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "pageSize", 20)
	items, total, err := s.service.ListDiagnosisTasks(c.Request.Context(), rbacservice.GetUserID(c), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询诊断记录失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, items)
}

func parseUintParam(c *gin.Context, name string) uint {
	raw := c.Param(name)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0
	}
	return uint(value)
}

func parseIntQuery(c *gin.Context, name string, fallback int) int {
	raw := c.Query(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
