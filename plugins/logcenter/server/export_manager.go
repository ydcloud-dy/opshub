package server

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	exportPageSize          = 500
	exportIdlePollInterval  = 10 * time.Second
	exportErrorPollInterval = 5 * time.Second
)

type logExportManager struct {
	db            *gorm.DB
	query         *logsvc.InternalQueryService
	directory     string
	maxRows       int
	maxAttempts   int
	queueCapacity int
	workers       int
	lease         time.Duration
	retention     time.Duration
	wake          chan struct{}
	instanceID    string
}

type logExportPayload struct {
	SchemaVersion           int                          `json:"schemaVersion"`
	Request                 logsvc.InternalQueryRequest  `json:"request"`
	AllowedPolicyIDs        []uint64                     `json:"allowedPolicyIds,omitempty"`
	AllowedHostIDs          []uint64                     `json:"allowedHostIds,omitempty"`
	AllowedKubernetesScopes map[uint64][]string          `json:"allowedKubernetesScopes,omitempty"`
	DeniedFields            []string                     `json:"deniedFields,omitempty"`
	MaskFields              []string                     `json:"maskFields,omitempty"`
	DenyAll                 bool                         `json:"denyAll,omitempty"`
	RequiredFilters         []logsvc.InternalQueryFilter `json:"requiredFilters,omitempty"`
}

func newLogExportManager(db *gorm.DB, query *logsvc.InternalQueryService) *logExportManager {
	if db == nil || query == nil {
		return nil
	}
	directory := strings.TrimSpace(os.Getenv("OPSHUB_LOG_EXPORT_DIR"))
	if directory == "" {
		directory = "data/log-exports"
	}
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	manager := &logExportManager{
		db:            db,
		query:         query,
		directory:     directory,
		maxRows:       positiveEnvInt("OPSHUB_LOG_EXPORT_MAX_ROWS", 1000000),
		maxAttempts:   positiveEnvInt("OPSHUB_LOG_EXPORT_MAX_ATTEMPTS", 3),
		queueCapacity: positiveEnvInt("OPSHUB_LOG_EXPORT_QUEUE_CAPACITY", 100),
		workers:       positiveEnvInt("OPSHUB_LOG_EXPORT_WORKERS", 2),
		lease:         time.Duration(positiveEnvInt("OPSHUB_LOG_EXPORT_LEASE_SECONDS", 300)) * time.Second,
		retention:     time.Duration(positiveEnvInt("OPSHUB_LOG_EXPORT_RETENTION_HOURS", 24)) * time.Hour,
		wake:          make(chan struct{}, 1),
		instanceID:    fmt.Sprintf("%s-%d", hostname, os.Getpid()),
	}
	if manager.lease < 30*time.Second {
		manager.lease = 30 * time.Second
	}
	if err := os.MkdirAll(manager.directory, 0o700); err != nil {
		return nil
	}
	for index := 0; index < manager.workers; index++ {
		go manager.runWorker(index)
	}
	go manager.runCleanup()
	manager.notifyWorkers()
	return manager
}

func encodeLogExportRequest(request logsvc.InternalQueryRequest) (string, error) {
	return encodeLogExportPayload(logExportPayload{
		SchemaVersion: 2, Request: request,
		AllowedPolicyIDs: request.AllowedPolicyIDs,
		AllowedHostIDs:   request.AllowedHostIDs, AllowedKubernetesScopes: request.AllowedKubernetesScopes,
		DeniedFields: request.DeniedFields, MaskFields: request.MaskFields, DenyAll: request.DenyAll,
		RequiredFilters: request.RequiredFilters,
	})
}

func encodeLogExportPayload(payload logExportPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeLogExportPayload(raw string) (logExportPayload, error) {
	var payload logExportPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return payload, err
	}
	if payload.SchemaVersion != 2 {
		return payload, fmt.Errorf("日志导出任务格式过旧，请重新创建导出任务")
	}
	return payload, nil
}

func (m *logExportManager) notifyWorkers() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *logExportManager) queueFull(ctx context.Context) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).Model(&logmodel.LogExportTask{}).
		Where("status IN ?", []string{"pending", "running"}).Count(&count).Error
	return count >= int64(m.queueCapacity), err
}

func (m *logExportManager) runWorker(index int) {
	owner := fmt.Sprintf("%s-%d", m.instanceID, index)
	for {
		task, claimed, err := m.claimTask(owner)
		if err == nil && claimed {
			m.process(owner, task)
			continue
		}
		delay := exportIdlePollInterval
		if err != nil {
			delay = exportErrorPollInterval
		}
		wait := time.NewTimer(delay)
		select {
		case <-m.wake:
			if !wait.Stop() {
				<-wait.C
			}
		case <-wait.C:
		}
	}
}

func (m *logExportManager) claimTask(owner string) (logmodel.LogExportTask, bool, error) {
	var task logmodel.LogExportTask
	claimed := false
	now := time.Now()
	err := m.db.Transaction(func(tx *gorm.DB) error {
		query := tx.WithContext(context.Background()).Where(
			"((status = ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)) OR (status = ? AND ((lease_expires_at IS NOT NULL AND lease_expires_at < ?) OR (lease_expires_at IS NULL AND updated_at < ?)))) AND (max_attempts <= 0 OR attempt_count < max_attempts)",
			"pending", now, "running", now, now.Add(-m.lease),
		).Order("created_at ASC, id ASC").Limit(1).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		result := query.Find(&task)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true
		attempts := task.AttemptCount + 1
		if task.MaxAttempts <= 0 {
			task.MaxAttempts = m.maxAttempts
		}
		updates := map[string]interface{}{
			"status":           "running",
			"attempt_count":    attempts,
			"max_attempts":     task.MaxAttempts,
			"lease_owner":      owner,
			"lease_expires_at": now.Add(m.lease),
			"next_attempt_at":  nil,
			"progress":         1,
			"exported_rows":    0,
			"file_name":        "",
			"file_path":        "",
			"file_size":        0,
			"completed_at":     nil,
			"error_message":    "",
		}
		if task.StartedAt == nil {
			updates["started_at"] = now
		}
		if err := tx.Model(&task).Updates(updates).Error; err != nil {
			return err
		}
		task.Status = "running"
		task.AttemptCount = attempts
		task.LeaseOwner = owner
		task.LeaseExpiresAt = timePtr(now.Add(m.lease))
		return nil
	})
	return task, claimed && err == nil, err
}

func (m *logExportManager) process(owner string, task logmodel.LogExportTask) {
	defer func() {
		if recovered := recover(); recovered != nil {
			m.fail(owner, task, fmt.Errorf("日志导出任务异常: %v", recovered), "")
		}
	}()
	payload, err := decodeLogExportPayload(task.QueryPayload)
	if err != nil {
		m.fail(owner, task, err, "")
		return
	}
	request := payload.Request
	request.AllowedPolicyIDs = payload.AllowedPolicyIDs
	request.AllowedHostIDs = payload.AllowedHostIDs
	request.AllowedKubernetesScopes = payload.AllowedKubernetesScopes
	request.DeniedFields = payload.DeniedFields
	request.MaskFields = payload.MaskFields
	request.DenyAll = payload.DenyAll
	request.RequiredFilters = payload.RequiredFilters
	request.StorageID = task.StorageID
	cluster, err := m.loadStorage(task.StorageID)
	if err != nil {
		m.fail(owner, task, err, "")
		return
	}
	password, err := decryptStorageSecret(cluster.PasswordEncrypted)
	if err != nil {
		m.fail(owner, task, err, "")
		return
	}
	leaseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.heartbeat(leaseCtx, owner, task.ID, cancel)
	ctx, timeoutCancel := context.WithTimeout(leaseCtx, time.Duration(positiveEnvInt("OPSHUB_LOG_EXPORT_TIMEOUT_MINUTES", 120))*time.Minute)
	defer timeoutCancel()
	extension := "ndjson"
	if task.Format == "csv" {
		extension = "csv"
	}
	fileName := fmt.Sprintf("opshub-logs-%d-%d.%s", task.ID, time.Now().UnixNano(), extension)
	finalPath := filepath.Join(m.directory, fileName)
	temporaryPath := finalPath + ".part"
	file, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		m.fail(owner, task, err, temporaryPath)
		return
	}
	buffered := bufio.NewWriterSize(file, 1024*1024)
	rows, exportErr := m.writeRows(ctx, buffered, owner, cluster, password, request, task)
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
		m.fail(owner, task, exportErr, temporaryPath)
		return
	}
	info, statErr := os.Stat(finalPath)
	if statErr != nil {
		m.fail(owner, task, statErr, finalPath)
		return
	}
	relativePath, pathErr := filepath.Rel(m.directory, finalPath)
	if pathErr != nil {
		m.fail(owner, task, pathErr, finalPath)
		return
	}
	completedAt := time.Now()
	expiresAt := completedAt.Add(m.retention)
	result := m.db.Model(&logmodel.LogExportTask{}).Where("id = ? AND status = ? AND lease_owner = ?", task.ID, "running", owner).Updates(map[string]interface{}{
		"status": "completed", "progress": 100, "exported_rows": rows, "file_name": fileName,
		"file_path": relativePath, "file_size": info.Size(), "completed_at": completedAt, "expires_at": expiresAt,
		"lease_owner": "", "lease_expires_at": nil, "error_message": "",
	})
	if result.Error != nil || result.RowsAffected == 0 {
		_ = os.Remove(finalPath)
	}
}

func (m *logExportManager) writeRows(ctx context.Context, writer *bufio.Writer, owner string, cluster logmodel.StorageCluster, password string, request logsvc.InternalQueryRequest, task logmodel.LogExportTask) (int64, error) {
	request.Sort = "asc"
	request.Cursor = ""
	request.SkipHistory = true
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
		request.Limit = exportPageSize
		if remaining < exportPageSize {
			request.Limit = int(remaining)
		}
		result, err := m.query.Query(ctx, cluster, password, request)
		if err != nil {
			return exported, err
		}
		for _, item := range result.Items {
			if csvWriter != nil {
				labels, _ := json.Marshal(item.Labels)
				fields, _ := json.Marshal(item.Fields)
				row := []string{item.Timestamp, item.Level, item.Message, item.Labels["sourceType"], item.Labels["hostId"], item.Labels["clusterId"], item.Labels["namespace"], item.Labels["workloadName"], item.Labels["podName"], item.Labels["containerName"], item.Labels["nodeName"], item.Labels["service"], item.Labels["environment"], string(labels), string(fields)}
				if err := csvWriter.Write(sanitizeCSVRow(row)); err != nil {
					return exported, err
				}
			} else if err := encoder.Encode(map[string]interface{}{"timestamp": item.Timestamp, "level": item.Level, "message": item.Message, "labels": item.Labels, "fields": item.Fields}); err != nil {
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
		if err := m.updateProgress(owner, task.ID, exported, int(exported*95/int64(maxInt(task.MaxRows, 1)))); err != nil {
			return exported, err
		}
		if len(result.Items) == 0 || !result.HasMore || result.NextCursor == "" {
			break
		}
		request.Cursor = result.NextCursor
	}
	return exported, nil
}

func sanitizeCSVRow(row []string) []string {
	result := make([]string, len(row))
	for index, value := range row {
		result[index] = sanitizeCSVCell(value)
	}
	return result
}

func sanitizeCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func (m *logExportManager) updateProgress(owner string, taskID uint, rows int64, progress int) error {
	if progress < 1 {
		progress = 1
	}
	result := m.db.Model(&logmodel.LogExportTask{}).Where("id = ? AND status = ? AND lease_owner = ?", taskID, "running", owner).Updates(map[string]interface{}{"progress": progress, "exported_rows": rows, "lease_expires_at": time.Now().Add(m.lease)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("日志导出任务租约已失效")
	}
	return nil
}

func (m *logExportManager) heartbeat(ctx context.Context, owner string, taskID uint, cancel context.CancelFunc) {
	ticker := time.NewTicker(m.lease / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := m.db.Model(&logmodel.LogExportTask{}).Where("id = ? AND status = ? AND lease_owner = ?", taskID, "running", owner).Update("lease_expires_at", time.Now().Add(m.lease))
			if result.Error != nil || result.RowsAffected == 0 {
				cancel()
				return
			}
		}
	}
}

func (m *logExportManager) fail(owner string, task logmodel.LogExportTask, err error, path string) {
	if path != "" {
		_ = os.Remove(path)
	}
	now := time.Now()
	status := "pending"
	updates := map[string]interface{}{"status": status, "error_message": err.Error(), "next_attempt_at": now.Add(exportRetryDelay(task.AttemptCount)), "lease_owner": "", "lease_expires_at": nil}
	maxAttempts := task.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = m.maxAttempts
	}
	if task.AttemptCount >= maxAttempts {
		status = "failed"
		updates["status"] = status
		updates["completed_at"] = now
		updates["next_attempt_at"] = nil
	}
	_ = m.db.Model(&logmodel.LogExportTask{}).Where("id = ? AND status = ? AND lease_owner = ?", task.ID, "running", owner).Updates(updates)
	m.notifyWorkers()
}

func exportRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

func (m *logExportManager) loadStorage(id uint) (logmodel.StorageCluster, error) {
	var cluster logmodel.StorageCluster
	if err := m.db.Where("id = ? AND enabled = ? AND storage_type = ?", id, true, "clickhouse").First(&cluster).Error; err != nil {
		return cluster, fmt.Errorf("日志存储不可用: %w", err)
	}
	if cluster.InitializedAt == nil {
		return cluster, fmt.Errorf("日志存储尚未初始化")
	}
	return cluster, nil
}

func (m *logExportManager) secureArtifactPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty export path")
	}
	absDirectory, err := filepath.Abs(m.directory)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filepath.Join(m.directory, path))
	if err != nil {
		return "", err
	}
	// Accept legacy absolute paths only when they are already inside the directory.
	if filepath.IsAbs(path) {
		absPath, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}
	relative, err := filepath.Rel(absDirectory, absPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid export path")
	}
	return absPath, nil
}

func (m *logExportManager) runCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		m.failExhausted()
		m.cleanupExpired()
		<-ticker.C
	}
}

func (m *logExportManager) failExhausted() {
	now := time.Now()
	_ = m.db.Model(&logmodel.LogExportTask{}).Where(
		"max_attempts > 0 AND attempt_count >= max_attempts AND ((status = ?) OR (status = ? AND ((lease_expires_at IS NOT NULL AND lease_expires_at < ?) OR (lease_expires_at IS NULL AND updated_at < ?))))",
		"pending", "running", now, now.Add(-m.lease),
	).Updates(map[string]interface{}{
		"status": "failed", "completed_at": now, "next_attempt_at": nil,
		"lease_owner": "", "lease_expires_at": nil,
	})
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
		_ = m.db.Model(&task).Updates(map[string]interface{}{"status": "expired", "file_path": "", "file_size": 0})
	}
}

func timePtr(value time.Time) *time.Time { return &value }
