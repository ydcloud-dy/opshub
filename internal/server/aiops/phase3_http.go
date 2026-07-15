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
	"net/http"

	"github.com/gin-gonic/gin"
	aiopsservice "github.com/ydcloud-dy/opshub/internal/service/aiops"
	rbacservice "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

func (s *HTTPServer) ListAlertEvents(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "pageSize", 20)
	items, total, err := s.service.ListAlertEvents(c.Request.Context(), page, pageSize, c.Query("state"), c.Query("severity"))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询告警事件失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, items)
}

func (s *HTTPServer) AnalyzeAlertRootCause(c *gin.Context) {
	var req aiopsservice.AlertAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	result, err := s.service.AnalyzeAlertRootCause(c.Request.Context(), rbacservice.GetUserID(c), rbacservice.GetUsername(c), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "告警根因分析失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (s *HTTPServer) ListRootCauseAnalyses(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "pageSize", 20)
	items, total, err := s.service.ListRootCauseAnalyses(c.Request.Context(), rbacservice.GetUserID(c), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询根因分析记录失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, items)
}
