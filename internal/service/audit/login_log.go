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

package audit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type LoginLogService struct {
	useCase *audit.LoginLogUseCase
}

func NewLoginLogService(useCase *audit.LoginLogUseCase) *LoginLogService {
	return &LoginLogService{
		useCase: useCase,
	}
}

// LoginLogListResponse 登录日志列表响应
type LoginLogListResponse struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	RealName    string `json:"realName"`
	LoginType   string `json:"loginType"`
	LoginStatus string `json:"loginStatus"`
	LoginTime   string `json:"loginTime"`
	LogoutTime  string `json:"logoutTime"`
	IP          string `json:"ip"`
	Location    string `json:"location"`
	UserAgent   string `json:"userAgent"`
	FailReason  string `json:"failReason"`
}

func toLoginLogListResponse(log *audit.SysLoginLog) LoginLogListResponse {
	resp := LoginLogListResponse{
		ID:          log.ID,
		UserID:      log.UserID,
		Username:    log.Username,
		RealName:    log.RealName,
		LoginType:   log.LoginType,
		LoginStatus: log.LoginStatus,
		LoginTime:   log.LoginTime.Format("2006-01-02 15:04:05"),
		IP:          log.IP,
		Location:    log.Location,
		UserAgent:   log.UserAgent,
		FailReason:  log.FailReason,
	}
	if log.LogoutTime != nil {
		resp.LogoutTime = log.LogoutTime.Format("2006-01-02 15:04:05")
	}
	return resp
}

// ListLoginLogs 登录日志列表
// @Summary 获取登录日志列表
// @Description 分页获取系统登录日志，支持按用户名、登录类型、登录状态和时间范围筛选
// @Tags 审计管理-登录日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param username query string false "用户名"
// @Param loginType query string false "登录类型"
// @Param loginStatus query string false "登录状态"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/audit/login-logs [get]
func (s *LoginLogService) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	loginType := c.Query("loginType")
	loginStatus := c.Query("loginStatus")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	logs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, username, loginType, loginStatus, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	list := make([]LoginLogListResponse, 0, len(logs))
	for _, log := range logs {
		list = append(list, toLoginLogListResponse(log))
	}

	response.Success(c, gin.H{
		"list":     list,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// GetLoginLog 获取登录日志详情
// @Summary 获取登录日志详情
// @Description 获取单条登录日志的详细信息
// @Tags 审计管理-登录日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "登录日志ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "日志不存在"
// @Router /api/v1/audit/login-logs/{id} [get]
func (s *LoginLogService) GetLoginLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的日志ID")
		return
	}

	log, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志不存在")
		return
	}

	response.Success(c, toLoginLogListResponse(log))
}

// DeleteLoginLog 删除登录日志
// @Summary 删除登录日志
// @Description 删除指定的登录日志记录
// @Tags 审计管理-登录日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "登录日志ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/login-logs/{id} [delete]
func (s *LoginLogService) DeleteLoginLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的日志ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// DeleteLoginLogsBatch 批量删除登录日志
// @Summary 批量删除登录日志
// @Description 批量删除多条登录日志记录
// @Tags 审计管理-登录日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "日志ID列表" example({"ids": [1, 2, 3]})
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/login-logs/batch [delete]
func (s *LoginLogService) DeleteLoginLogsBatch(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.useCase.DeleteBatch(c.Request.Context(), req.IDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}
