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

type DataLogService struct {
	useCase *audit.DataLogUseCase
}

func NewDataLogService(useCase *audit.DataLogUseCase) *DataLogService {
	return &DataLogService{
		useCase: useCase,
	}
}

// DataLogListResponse 数据日志列表响应
type DataLogListResponse struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"userId"`
	Username   string `json:"username"`
	RealName   string `json:"realName"`
	TableName  string `json:"tableName"`
	RecordID   uint   `json:"recordId"`
	Action     string `json:"action"`
	OldData    string `json:"oldData"`
	NewData    string `json:"newData"`
	DiffFields string `json:"diffFields"`
	IP         string `json:"ip"`
	CreatedAt  string `json:"createdAt"`
}

func toDataLogListResponse(log *audit.SysDataLog) DataLogListResponse {
	return DataLogListResponse{
		ID:         log.ID,
		UserID:     log.UserID,
		Username:   log.Username,
		RealName:   log.RealName,
		TableName:  log.TableName,
		RecordID:   log.RecordID,
		Action:     log.Action,
		OldData:    log.OldData,
		NewData:    log.NewData,
		DiffFields: log.DiffFields,
		IP:         log.IP,
		CreatedAt:  log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListDataLogs 数据日志列表
// @Summary 获取数据日志列表
// @Description 分页获取数据变更日志，支持按用户名、表名、操作类型和时间范围筛选
// @Tags 审计管理-数据日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param username query string false "用户名"
// @Param tableName query string false "表名"
// @Param action query string false "操作类型"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/audit/data-logs [get]
func (s *DataLogService) ListDataLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	tableName := c.Query("tableName")
	action := c.Query("action")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	logs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, username, tableName, action, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	list := make([]DataLogListResponse, 0, len(logs))
	for _, log := range logs {
		list = append(list, toDataLogListResponse(log))
	}

	response.Success(c, gin.H{
		"list":     list,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// GetDataLog 获取数据日志详情
// @Summary 获取数据日志详情
// @Description 获取单条数据变更日志的详细信息
// @Tags 审计管理-数据日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "数据日志ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "日志不存在"
// @Router /api/v1/audit/data-logs/{id} [get]
func (s *DataLogService) GetDataLog(c *gin.Context) {
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

	response.Success(c, toDataLogListResponse(log))
}

// DeleteDataLog 删除数据日志
// @Summary 删除数据日志
// @Description 删除指定的数据变更日志记录
// @Tags 审计管理-数据日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "数据日志ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/data-logs/{id} [delete]
func (s *DataLogService) DeleteDataLog(c *gin.Context) {
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

// DeleteDataLogsBatch 批量删除数据日志
// @Summary 批量删除数据日志
// @Description 批量删除多条数据变更日志记录
// @Tags 审计管理-数据日志
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "日志ID列表" example({"ids": [1, 2, 3]})
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/audit/data-logs/batch [delete]
func (s *DataLogService) DeleteDataLogsBatch(c *gin.Context) {
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
