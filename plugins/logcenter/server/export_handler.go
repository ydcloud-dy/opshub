package server

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"gorm.io/gorm"
)

const exportPageSize = 5000

type internalLogExportRequest struct {
	logsvc.InternalQueryRequest
	Format  string `json:"format"`
	MaxRows int    `json:"maxRows"`
}

type logExportJob struct {
	TaskID   uint
	Request  logsvc.InternalQueryRequest
	Cluster  logmodel.StorageCluster
	Password string
}

type logExportManager struct {
	db        *gorm.DB
	query     *logsvc.InternalQueryService
	jobs      chan logExportJob
	directory string
	maxRows   int
}

func newLogExportManager(db *gorm.DB, query *logsvc.InternalQueryService) *logExportManager {
	if db == nil || query == nil {
		return nil
	}
	directory := strings.TrimSpace(os.Getenv("OPSHUB_LOG_EXPORT_DIR"))
	if directory == "" {
		directory = "data/log-exports"
	}
	manager := &logExportManager{
		db: db, query: query, jobs: make(chan logExportJob, positiveEnvInt("OPSHUB_LOG_EXPORT_QUEUE_CAPACITY", 100)),
		directory: directory, maxRows: positiveEnvInt("OPSHUB_LOG_EXPORT_MAX_ROWS", 1000000),
	}
	_ = os.MkdirAll(manager.directory, 0o700)
	for index := 0; index < positiveEnvInt("OPSHUB_LOG_EXPORT_WORKERS", 2); index++ {
		go manager.runWorker()
	}
	go manager.runCleanup()
	return manager
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
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
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
	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = 100000
	}
	if maxRows > h.exporter.maxRows {
		maxRows = h.exporter.maxRows
	}
	req.Sort = "asc"
	req.Cursor = ""
	req.SkipHistory = true
	queryPayload, _ := json.Marshal(req.InternalQueryRequest)
	now := time.Now()
	expiresAt := now.Add(time.Duration(positiveEnvInt("OPSHUB_LOG_EXPORT_RETENTION_HOURS", 24)) * time.Hour)
	task := logmodel.LogExportTask{
		UserID: rbacsvc.GetUserID(c), StorageID: cluster.ID, Format: format, QueryPayload: string(queryPayload),
		Status: "pending", Progress: 0, MaxRows: maxRows, ExpiresAt: &expiresAt,
	}
	if task.UserID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未获取到当前用户，无法创建导出任务")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&task).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志导出任务失败: "+err.Error())
		return
	}
	job := logExportJob{TaskID: task.ID, Request: req.InternalQueryRequest, Cluster: cluster, Password: password}
	select {
	case h.exporter.jobs <- job:
		response.Success(c, task)
	default:
		errorMessage := "日志导出队列已满，请稍后重试"
		_ = h.db.Model(&task).Updates(map[string]interface{}{"status": "failed", "error_message": errorMessage}).Error
		response.ErrorCode(c, http.StatusServiceUnavailable, errorMessage)
	}
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

func (m *logExportManager) runWorker() {
	for job := range m.jobs {
		m.process(job)
	}
}

func (m *logExportManager) process(job logExportJob) {
	startedAt := time.Now()
	_ = m.db.Model(&logmodel.LogExportTask{}).Where("id = ?", job.TaskID).Updates(map[string]interface{}{
		"status": "running", "progress": 1, "started_at": &startedAt, "error_message": "",
	}).Error
	var task logmodel.LogExportTask
	if err := m.db.First(&task, job.TaskID).Error; err != nil {
		return
	}
	extension := "ndjson"
	if task.Format == "csv" {
		extension = "csv"
	}
	fileName := fmt.Sprintf("opshub-logs-%d-%s.%s", task.ID, startedAt.Format("20060102-150405"), extension)
	finalPath := filepath.Join(m.directory, fileName)
	temporaryPath := finalPath + ".part"
	file, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		m.fail(task.ID, err, temporaryPath)
		return
	}
	buffered := bufio.NewWriterSize(file, 1024*1024)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(positiveEnvInt("OPSHUB_LOG_EXPORT_TIMEOUT_MINUTES", 120))*time.Minute)
	defer cancel()
	rows, exportErr := m.writeRows(ctx, buffered, job, task)
	if flushErr := buffered.Flush(); exportErr == nil {
		exportErr = flushErr
	}
	if closeErr := file.Close(); exportErr == nil {
		exportErr = closeErr
	}
	if exportErr == nil {
		exportErr = os.Rename(temporaryPath, finalPath)
	}
	if exportErr != nil {
		m.fail(task.ID, exportErr, temporaryPath)
		return
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		m.fail(task.ID, err, finalPath)
		return
	}
	completedAt := time.Now()
	_ = m.db.Model(&logmodel.LogExportTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		"status": "completed", "progress": 100, "exported_rows": rows, "file_name": fileName,
		"file_path": finalPath, "file_size": info.Size(), "completed_at": &completedAt, "error_message": "",
	}).Error
}

func (m *logExportManager) writeRows(ctx context.Context, writer *bufio.Writer, job logExportJob, task logmodel.LogExportTask) (int64, error) {
	req := job.Request
	req.Sort = "asc"
	req.Cursor = ""
	req.SkipHistory = true
	var csvWriter *csv.Writer
	if task.Format == "csv" {
		csvWriter = csv.NewWriter(writer)
		if err := csvWriter.Write([]string{"timestamp", "level", "message", "sourceType", "hostId", "clusterId", "namespace", "workload", "pod", "container", "node", "service", "environment", "labels", "fields"}); err != nil {
			return 0, err
		}
	}
	encoder := json.NewEncoder(writer)
	var exported int64
	for exported < int64(task.MaxRows) {
		remaining := int64(task.MaxRows) - exported
		req.Limit = exportPageSize
		if remaining < exportPageSize {
			req.Limit = int(remaining)
		}
		result, err := m.query.Query(ctx, job.Cluster, job.Password, req)
		if err != nil {
			return exported, err
		}
		for _, item := range result.Items {
			if csvWriter != nil {
				labels, _ := json.Marshal(item.Labels)
				fields, _ := json.Marshal(item.Fields)
				if err := csvWriter.Write([]string{
					item.Timestamp, item.Level, item.Message, item.Labels["sourceType"], item.Labels["hostId"],
					item.Labels["clusterId"], item.Labels["namespace"], item.Labels["workloadName"], item.Labels["podName"],
					item.Labels["containerName"], item.Labels["nodeName"], item.Labels["service"], item.Labels["environment"],
					string(labels), string(fields),
				}); err != nil {
					return exported, err
				}
			} else if err := encoder.Encode(map[string]interface{}{
				"timestamp": item.Timestamp, "level": item.Level, "message": item.Message,
				"labels": item.Labels, "fields": item.Fields,
			}); err != nil {
				return exported, err
			}
			exported++
		}
		if csvWriter != nil {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return exported, err
			}
		}
		progress := int(exported * 95 / int64(task.MaxRows))
		if progress < 1 {
			progress = 1
		}
		_ = m.db.Model(&logmodel.LogExportTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"progress": progress, "exported_rows": exported,
		}).Error
		if len(result.Items) == 0 || !result.HasMore || result.NextCursor == "" {
			break
		}
		req.Cursor = result.NextCursor
	}
	return exported, nil
}

func (m *logExportManager) fail(taskID uint, err error, paths ...string) {
	for _, path := range paths {
		if path != "" {
			_ = os.Remove(path)
		}
	}
	completedAt := time.Now()
	_ = m.db.Model(&logmodel.LogExportTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": "failed", "error_message": err.Error(), "completed_at": &completedAt,
	}).Error
}

func (m *logExportManager) secureArtifactPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty export path")
	}
	absDirectory, err := filepath.Abs(m.directory)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absDirectory, absPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid export path")
	}
	return absPath, nil
}

func (m *logExportManager) runCleanup() {
	m.cleanupExpired()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanupExpired()
	}
}

func (m *logExportManager) cleanupExpired() {
	var tasks []logmodel.LogExportTask
	if err := m.db.Where("expires_at IS NOT NULL AND expires_at < ? AND status = ?", time.Now(), "completed").Limit(500).Find(&tasks).Error; err != nil {
		return
	}
	for _, task := range tasks {
		if path, err := m.secureArtifactPath(task.FilePath); err == nil {
			_ = os.Remove(path)
		}
		_ = m.db.Model(&task).Updates(map[string]interface{}{
			"status": "expired", "file_path": "", "file_size": 0,
		}).Error
	}
}
