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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package identity

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// AuthLogService 认证日志服务
type AuthLogService struct {
	useCase *identity.AuthLogUseCase
}

func NewAuthLogService(useCase *identity.AuthLogUseCase) *AuthLogService {
	return &AuthLogService{useCase: useCase}
}

// ListLogs 获取认证日志列表
// @Summary 获取认证日志列表
// @Description 分页获取认证日志列表
// @Tags 身份认证-认证日志
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param userId query int false "用户ID"
// @Param action query string false "动作(login/logout/access_app)"
// @Param result query string false "结果(success/failed)"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/logs [get]
func (s *AuthLogService) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	action := c.Query("action")
	result := c.Query("result")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	var userID *uint
	if id := c.Query("userId"); id != "" {
		idVal, _ := strconv.ParseUint(id, 10, 64)
		uidVal := uint(idVal)
		userID = &uidVal
	}

	logs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, userID, action, result, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取日志列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  logs,
		"total": total,
	})
}

// GetStats 获取认证统计
// @Summary 获取认证统计
// @Description 获取认证日志统计信息
// @Tags 身份认证-认证日志
// @Accept json
// @Produce json
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/logs/stats [get]
func (s *AuthLogService) GetStats(c *gin.Context) {
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	stats, err := s.useCase.GetStats(c.Request.Context(), startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取统计信息失败: "+err.Error())
		return
	}

	response.Success(c, stats)
}

// GetLoginTrend 获取登录趋势
// @Summary 获取登录趋势
// @Description 获取近N天的登录趋势数据
// @Tags 身份认证-认证日志
// @Accept json
// @Produce json
// @Param days query int false "天数" default(7)
// @Success 200 {object} response.Response
// @Router /api/v1/identity/logs/trend [get]
func (s *AuthLogService) GetLoginTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 30 {
		days = 7
	}

	trend, err := s.useCase.GetLoginTrend(c.Request.Context(), days)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取趋势数据失败: "+err.Error())
		return
	}

	response.Success(c, trend)
}
