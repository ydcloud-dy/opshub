package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

type internalLogExportRequest struct {
	logsvc.InternalQueryRequest
	Format  string `json:"format"`
	MaxRows int    `json:"maxRows"`
}

func (h *Handler) CreateInternalLogExport(c *gin.Context) {
	if h.exporter == nil {
		response.ErrorCode(c, http.StatusServiceUnavailable, "日志导出服务未启用")
		return
	}
	var req internalLogExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalPermissions(c, &req.InternalQueryRequest, "export") {
		return
	}
	if err := logsvc.ValidateInternalQueryRequest(req.InternalQueryRequest); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	cluster, _, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	full, err := h.exporter.queueFull(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志导出队列失败: "+err.Error())
		return
	}
	if full {
		response.ErrorCode(c, http.StatusServiceUnavailable, "日志导出队列已满，请稍后重试")
		return
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "ndjson"
	}
	if format != "ndjson" && format != "csv" {
		response.ErrorCode(c, http.StatusBadRequest, "日志导出格式仅支持 ndjson 或 csv")
		return
	}
	maxRows, err := validateExportMaxRows(req.MaxRows, h.exporter.maxRows)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	req.Sort = "asc"
	req.Cursor = ""
	req.SkipHistory = true
	req.StorageID = cluster.ID
	queryPayload, err := encodeLogExportRequest(req.InternalQueryRequest)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存日志导出条件失败: "+err.Error())
		return
	}
	task := logmodel.LogExportTask{
		UserID: rbacsvc.GetUserID(c), StorageID: cluster.ID, Format: format, QueryPayload: queryPayload,
		Status: "pending", Progress: 0, MaxRows: maxRows, MaxAttempts: h.exporter.maxAttempts,
		ExpiresAt: timePtr(time.Now().Add(h.exporter.retention)),
	}
	if task.UserID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未获取到当前用户，无法创建导出任务")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&task).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志导出任务失败: "+err.Error())
		return
	}
	h.exporter.notifyWorkers()
	response.Success(c, task)
}

func validateExportMaxRows(requested, maximum int) (int, error) {
	if requested < 1000 || requested > maximum {
		return 0, fmt.Errorf("最大导出条数必须在 1000 到 %s 之间", strconv.Itoa(maximum))
	}
	return requested, nil
}

func (h *Handler) ListInternalLogExports(c *gin.Context) {
	userID := rbacsvc.GetUserID(c)
	var tasks []logmodel.LogExportTask
	if err := h.db.WithContext(c.Request.Context()).Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&tasks).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志导出任务失败: "+err.Error())
		return
	}
	response.Success(c, tasks)
}

func (h *Handler) GetInternalLogExport(c *gin.Context) {
	task, ok := h.userLogExportTask(c)
	if !ok {
		return
	}
	response.Success(c, task)
}

func (h *Handler) DownloadInternalLogExport(c *gin.Context) {
	task, ok := h.userLogExportTask(c)
	if !ok {
		return
	}
	if task.Status != "completed" {
		response.ErrorCode(c, http.StatusConflict, "日志导出任务尚未完成")
		return
	}
	if task.ExpiresAt != nil && time.Now().After(*task.ExpiresAt) {
		response.ErrorCode(c, http.StatusGone, "日志导出文件已过期")
		return
	}
	path, err := h.exporter.secureArtifactPath(task.FilePath)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志导出文件不存在")
		return
	}
	if _, err := os.Stat(path); err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志导出文件不存在或当前副本未挂载共享导出目录")
		return
	}
	c.FileAttachment(path, task.FileName)
}

func (h *Handler) userLogExportTask(c *gin.Context) (logmodel.LogExportTask, bool) {
	var task logmodel.LogExportTask
	id := parseUint(c.Param("id"))
	if id == 0 || h.db.WithContext(c.Request.Context()).Where("id = ? AND user_id = ?", id, rbacsvc.GetUserID(c)).First(&task).Error != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志导出任务不存在")
		return task, false
	}
	return task, true
}
