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

type OperationLogService struct {
	useCase *audit.OperationLogUseCase
}

func NewOperationLogService(useCase *audit.OperationLogUseCase) *OperationLogService {
	return &OperationLogService{
		useCase: useCase,
	}
}

// OperationLogListResponse 操作日志列表响应
type OperationLogListResponse struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	RealName    string `json:"realName"`
	Module      string `json:"module"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Status      int    `json:"status"`
	ErrorMsg    string `json:"errorMsg"`
	CostTime    int64  `json:"costTime"`
	IP          string `json:"ip"`
	CreatedAt   string `json:"createdAt"`
}

func toOperationLogListResponse(log *audit.SysOperationLog) OperationLogListResponse {
	return OperationLogListResponse{
		ID:          log.ID,
		UserID:      log.UserID,
		Username:    log.Username,
		RealName:    log.RealName,
		Module:      log.Module,
		Action:      log.Action,
		Description: log.Description,
		Method:      log.Method,
		Path:        log.Path,
		Status:      log.Status,
		ErrorMsg:    log.ErrorMsg,
		CostTime:    log.CostTime,
		IP:          log.IP,
		CreatedAt:   log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListOperationLogs 操作日志列表
// @Summary 获取操作日志列表
// @Description 分页获取系统操作日志，支持按用户、模块、操作、状态和时间范围筛选
// @Tags 审计管理-操作日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param username query string false "用户名"
// @Param module query string false "模块名"
// @Param action query string false "操作"
// @Param status query string false "状态"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/audit/operation-logs [get]
func (s *OperationLogService) ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	module := c.Query("module")
	action := c.Query("action")
	status := c.Query("status")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	logs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, username, module, action, status, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	list := make([]OperationLogListResponse, 0, len(logs))
	for _, log := range logs {
		list = append(list, toOperationLogListResponse(log))
	}

	response.Success(c, gin.H{
		"list":     list,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// GetOperationLog 获取操作日志详情
// @Summary 获取操作日志详情
// @Description 获取单条操作日志的详细信息
// @Tags 审计管理-操作日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "操作日志ID"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/audit/operation-logs/{id} [get]
func (s *OperationLogService) GetOperationLog(c *gin.Context) {
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

	response.Success(c, toOperationLogListResponse(log))
}

// DeleteOperationLog 删除操作日志
// @Summary 删除操作日志
// @Description 删除指定的操作日志记录
// @Tags 审计管理-操作日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "操作日志ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/operation-logs/{id} [delete]
func (s *OperationLogService) DeleteOperationLog(c *gin.Context) {
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

// DeleteOperationLogsBatch 批量删除操作日志
// @Summary 批量删除操作日志
// @Description 批量删除多条操作日志记录
// @Tags 审计管理-操作日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "日志ID列表" example({"ids": [1, 2, 3]})
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/operation-logs/batch [delete]
func (s *OperationLogService) DeleteOperationLogsBatch(c *gin.Context) {
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
