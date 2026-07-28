package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/conf"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"gorm.io/gorm"
)

const (
	encryptedStorageSecretPrefix = "enc:v1:"
	builtinStorageName           = "OpsHub 内置日志存储"
)

// BootstrapStorageFromEnvironment provisions the built-in storage for Compose and Helm installs.
func BootstrapStorageFromEnvironment(db *gorm.DB) error {
	endpoint := strings.TrimSpace(os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT"))
	if endpoint == "" || db == nil {
		return nil
	}
	payload := storageClusterPayload{
		Name:                 builtinStorageName,
		StorageType:          "clickhouse",
		Endpoints:            endpoint,
		DatabaseName:         firstNonEmpty(os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_DATABASE"), "opshub_logs"),
		Username:             os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_USERNAME"),
		Password:             os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_PASSWORD"),
		Timeout:              300,
		QueueMode:            firstNonEmpty(strings.ToLower(strings.TrimSpace(os.Getenv("OPSHUB_LOG_QUEUE_MODE"))), "direct"),
		QueueEndpoints:       strings.TrimSpace(os.Getenv("OPSHUB_LOG_KAFKA_BROKERS")),
		DefaultRetentionDays: 30,
		Enabled:              true,
	}
	desired, err := storageClusterFromPayload(payload)
	if err != nil {
		return fmt.Errorf("解析内置日志存储配置失败: %w", err)
	}
	desired.IsPrimary = true
	var item logmodel.StorageCluster
	findErr := db.Where("name = ? OR (is_primary = ? AND endpoints = ?)", builtinStorageName, true, desired.Endpoints).First(&item).Error
	switch {
	case findErr == nil:
		updates := map[string]interface{}{
			"name": desired.Name, "storage_type": desired.StorageType, "endpoints": desired.Endpoints,
			"database_name": desired.DatabaseName, "username": desired.Username, "skip_tls_verify": desired.SkipTLSVerify,
			"timeout": desired.Timeout, "queue_mode": desired.QueueMode, "queue_endpoints": desired.QueueEndpoints,
			"default_retention_days": desired.DefaultRetentionDays, "enabled": true, "is_primary": true,
		}
		if payload.Password != "" {
			currentPassword, decryptErr := decryptStorageSecret(item.PasswordEncrypted)
			if decryptErr != nil {
				return fmt.Errorf("读取内置日志存储凭据失败: %w", decryptErr)
			}
			if currentPassword != payload.Password {
				updates["password_encrypted"], err = encryptStorageSecret(payload.Password)
				if err != nil {
					return fmt.Errorf("加密内置日志存储凭据失败: %w", err)
				}
			}
		}
		if err := db.Model(&item).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新内置日志存储失败: %w", err)
		}
		if err := db.First(&item, item.ID).Error; err != nil {
			return fmt.Errorf("重新读取内置日志存储失败: %w", err)
		}
	case errors.Is(findErr, gorm.ErrRecordNotFound):
		var count int64
		if err := db.Model(&logmodel.StorageCluster{}).Count(&count).Error; err != nil {
			return fmt.Errorf("统计日志存储失败: %w", err)
		}
		if count > 0 {
			return nil
		}
		item = desired
		item.PasswordEncrypted, err = encryptStorageSecret(payload.Password)
		if err != nil {
			return fmt.Errorf("加密内置日志存储凭据失败: %w", err)
		}
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("创建内置日志存储失败: %w", err)
		}
	default:
		return fmt.Errorf("读取内置日志存储失败: %w", findErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	clickhouse := logsvc.NewClickHouseService()
	if err := clickhouse.Initialize(ctx, item, payload.Password); err != nil {
		now := time.Now()
		_ = db.Model(&item).Updates(map[string]interface{}{"status": "error", "last_test_at": &now, "last_error": err.Error()}).Error
		return nil
	}
	now := time.Now()
	if err := db.Model(&item).Updates(map[string]interface{}{
		"status": "healthy", "last_test_at": &now, "last_error": "", "initialized_at": &now,
	}).Error; err != nil {
		return fmt.Errorf("更新内置日志存储状态失败: %w", err)
	}
	return nil
}

type storageClusterPayload struct {
	Name                 string `json:"name"`
	StorageType          string `json:"storageType"`
	Endpoints            string `json:"endpoints"`
	DatabaseName         string `json:"databaseName"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	SkipTLSVerify        bool   `json:"skipTlsVerify"`
	Timeout              int    `json:"timeout"`
	QueueMode            string `json:"queueMode"`
	QueueEndpoints       string `json:"queueEndpoints"`
	DefaultRetentionDays int    `json:"defaultRetentionDays"`
	Enabled              bool   `json:"enabled"`
}

func (h *Handler) ListStorageClusters(c *gin.Context) {
	isAdmin, ok := h.logAdminStatus(c)
	if !ok {
		return
	}
	var items []logmodel.StorageCluster
	query := h.db.Order("is_primary DESC, created_at ASC")
	if c.Query("enabled") != "" {
		query = query.Where("enabled = ?", c.Query("enabled") == "true" || c.Query("enabled") == "1")
	}
	if err := query.Find(&items).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询日志存储失败: "+err.Error())
		return
	}
	for index := range items {
		prepareStorageForResponse(&items[index])
		if !isAdmin {
			prepareStorageForQueryResponse(&items[index])
		}
	}
	response.Success(c, items)
}

func (h *Handler) GetStorageCluster(c *gin.Context) {
	isAdmin, ok := h.logAdminStatus(c)
	if !ok {
		return
	}
	item, ok := h.loadStorageCluster(c, parseUint(c.Param("id")))
	if !ok {
		return
	}
	prepareStorageForResponse(&item)
	if !isAdmin {
		prepareStorageForQueryResponse(&item)
	}
	response.Success(c, item)
}

func (h *Handler) CreateStorageCluster(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志存储") {
		return
	}
	var payload storageClusterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item, err := storageClusterFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	item.PasswordEncrypted, err = encryptStorageSecret(payload.Password)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "加密 ClickHouse 密码失败: "+err.Error())
		return
	}
	var count int64
	if err := h.db.Model(&logmodel.StorageCluster{}).Count(&count).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志存储配置失败: "+err.Error())
		return
	}
	item.IsPrimary = count == 0
	if err := h.db.Create(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志存储失败: "+err.Error())
		return
	}
	SyncInternalMonitorDataSources(h.db)
	prepareStorageForResponse(&item)
	response.Success(c, item)
}

func (h *Handler) UpdateStorageCluster(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志存储") {
		return
	}
	existing, ok := h.loadStorageCluster(c, parseUint(c.Param("id")))
	if !ok {
		return
	}
	var payload storageClusterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item, err := storageClusterFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	passwordEncrypted := existing.PasswordEncrypted
	if payload.Password != "" {
		passwordEncrypted, err = encryptStorageSecret(payload.Password)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "加密 ClickHouse 密码失败: "+err.Error())
			return
		}
	}
	updates := map[string]interface{}{
		"name":                   item.Name,
		"storage_type":           item.StorageType,
		"endpoints":              item.Endpoints,
		"database_name":          item.DatabaseName,
		"username":               item.Username,
		"password_encrypted":     passwordEncrypted,
		"skip_tls_verify":        item.SkipTLSVerify,
		"timeout":                item.Timeout,
		"queue_mode":             item.QueueMode,
		"queue_endpoints":        item.QueueEndpoints,
		"default_retention_days": item.DefaultRetentionDays,
		"enabled":                item.Enabled,
	}
	if !item.Enabled {
		updates["status"] = "disabled"
	}
	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志存储失败: "+err.Error())
		return
	}
	SyncInternalMonitorDataSources(h.db)
	updated, _ := h.loadStorageClusterByID(existing.ID)
	prepareStorageForResponse(&updated)
	response.Success(c, updated)
}

func (h *Handler) DeleteStorageCluster(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志存储") {
		return
	}
	item, ok := h.loadStorageCluster(c, parseUint(c.Param("id")))
	if !ok {
		return
	}
	if item.IsPrimary && item.Enabled {
		response.ErrorCode(c, http.StatusBadRequest, "正在使用的主存储不能删除，请先停用")
		return
	}
	if err := h.db.Delete(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除日志存储失败: "+err.Error())
		return
	}
	SyncInternalMonitorDataSources(h.db)
	response.Success(c, gin.H{"id": item.ID})
}

func (h *Handler) TestStorageCluster(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志存储") {
		return
	}
	item, ok := h.loadStorageCluster(c, parseUint(c.Param("id")))
	if !ok {
		return
	}
	password, err := decryptStorageSecret(item.PasswordEncrypted)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 ClickHouse 密码失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(item.Timeout)*time.Second)
	defer cancel()
	err = h.clickhouse.Ping(ctx, item, password)
	now := time.Now()
	updates := map[string]interface{}{"last_test_at": &now}
	if err != nil {
		updates["status"] = "error"
		updates["last_error"] = err.Error()
		_ = h.db.Model(&item).Updates(updates).Error
		response.ErrorCode(c, http.StatusBadRequest, "ClickHouse 连接测试失败: "+err.Error())
		return
	}
	updates["status"] = "healthy"
	updates["last_error"] = ""
	_ = h.db.Model(&item).Updates(updates).Error
	SyncInternalMonitorDataSources(h.db)
	response.Success(c, gin.H{"status": "healthy", "testedAt": now})
}

func (h *Handler) InitializeStorageCluster(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志存储") {
		return
	}
	item, ok := h.loadStorageCluster(c, parseUint(c.Param("id")))
	if !ok {
		return
	}
	if !item.Enabled {
		response.ErrorCode(c, http.StatusBadRequest, "请先启用该日志存储")
		return
	}
	password, err := decryptStorageSecret(item.PasswordEncrypted)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 ClickHouse 密码失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(maxInt(item.Timeout, 60))*time.Second)
	defer cancel()
	if err := h.clickhouse.Initialize(ctx, item, password); err != nil {
		now := time.Now()
		_ = h.db.Model(&item).Updates(map[string]interface{}{"status": "error", "last_test_at": &now, "last_error": err.Error()}).Error
		response.ErrorCode(c, http.StatusBadRequest, "初始化 ClickHouse 日志表失败: "+err.Error())
		return
	}
	now := time.Now()
	if err := h.db.Model(&logmodel.StorageCluster{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"status":         "healthy",
		"last_test_at":   &now,
		"last_error":     "",
		"initialized_at": &now,
	}).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存内置日志库配置失败: "+err.Error())
		return
	}
	SyncInternalMonitorDataSources(h.db)
	response.Success(c, gin.H{
		"status":        "healthy",
		"initializedAt": now,
		"database":      item.DatabaseName,
		"tables":        []string{"opshub_logs", "opshub_log_metrics_1m"},
	})
}

func (h *Handler) QueryInternalLogs(c *gin.Context) {
	var req logsvc.InternalQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalPermissions(c, &req, "query") {
		return
	}
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	started := time.Now()
	result, err := h.internalQuery.Query(c.Request.Context(), cluster, password, req)
	if err != nil {
		h.saveInternalQueryHistory(c, req, time.Since(started).Milliseconds(), 0, "failed", err.Error())
		response.ErrorCode(c, http.StatusBadRequest, "查询内置日志库失败: "+err.Error())
		return
	}
	if !req.SkipHistory {
		h.saveInternalQueryHistory(c, req, result.DurationMS, len(result.Items), "success", "")
	}
	response.Success(c, result)
}

func (h *Handler) QueryInternalHistogram(c *gin.Context) {
	var req logsvc.InternalQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalPermissions(c, &req, "query") {
		return
	}
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	started := time.Now()
	histogram, err := h.internalQuery.Histogram(c.Request.Context(), cluster, password, req)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "查询内置日志趋势失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"histogram": histogram, "durationMs": time.Since(started).Milliseconds()})
}

func (h *Handler) QueryInternalContext(c *gin.Context) {
	var req logsvc.InternalContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalContextPermissions(c, &req) {
		return
	}
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	result, err := h.internalQuery.Context(c.Request.Context(), cluster, password, req)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "查询内置日志上下文失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) QueryInternalResourceOptions(c *gin.Context) {
	var req logsvc.InternalQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalPermissions(c, &req, "query") {
		return
	}
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	options, err := h.internalQuery.ResourceOptions(c.Request.Context(), cluster, password, req)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "读取日志资源选项失败: "+err.Error())
		return
	}
	response.Success(c, options)
}

func (h *Handler) ListInternalFields(c *gin.Context) {
	storageID := parseUint(c.Query("storageId"))
	if _, _, ok := h.internalStorage(c, storageID); !ok {
		return
	}
	decision, ok := h.authorizeInternalAction(c, "query", storageID)
	if !ok {
		return
	}
	response.Success(c, logsvc.FilterInternalFieldOptions(logsvc.InternalFieldOptions(), decision.DeniedFields, decision.MaskFields))
}

func storageClusterFromPayload(payload storageClusterPayload) (logmodel.StorageCluster, error) {
	item := logmodel.StorageCluster{
		Name:                 strings.TrimSpace(payload.Name),
		StorageType:          strings.ToLower(strings.TrimSpace(payload.StorageType)),
		Endpoints:            strings.TrimSpace(payload.Endpoints),
		DatabaseName:         strings.TrimSpace(payload.DatabaseName),
		Username:             strings.TrimSpace(payload.Username),
		SkipTLSVerify:        payload.SkipTLSVerify,
		Timeout:              payload.Timeout,
		QueueMode:            strings.ToLower(strings.TrimSpace(payload.QueueMode)),
		QueueEndpoints:       strings.TrimSpace(payload.QueueEndpoints),
		DefaultRetentionDays: payload.DefaultRetentionDays,
		Enabled:              payload.Enabled,
		Status:               "unknown",
	}
	if item.Name == "" {
		return item, fmt.Errorf("存储名称不能为空")
	}
	if item.StorageType == "" {
		item.StorageType = "clickhouse"
	}
	if item.StorageType != "clickhouse" {
		return item, fmt.Errorf("当前仅支持 ClickHouse 日志存储")
	}
	if _, err := logsvc.NormalizeClickHouseEndpoint(item.Endpoints); err != nil {
		return item, err
	}
	if item.DatabaseName == "" {
		item.DatabaseName = "opshub_logs"
	}
	if item.Timeout <= 0 {
		item.Timeout = 300
	}
	if item.Timeout > 1800 {
		return item, fmt.Errorf("请求超时时间不能超过 1800 秒")
	}
	if item.QueueMode == "" {
		item.QueueMode = "direct"
	}
	if item.QueueMode != "direct" && item.QueueMode != "redpanda" && item.QueueMode != "kafka" {
		return item, fmt.Errorf("队列模式仅支持 direct、redpanda 或 kafka")
	}
	if item.DefaultRetentionDays <= 0 {
		item.DefaultRetentionDays = 30
	}
	if item.DefaultRetentionDays > 3650 {
		return item, fmt.Errorf("默认保留天数不能超过 3650 天")
	}
	return item, nil
}

func prepareStorageForResponse(item *logmodel.StorageCluster) {
	item.PasswordConfigured = item.PasswordEncrypted != ""
	item.PasswordEncrypted = ""
	item.QueueAuthEncrypted = ""
}

func prepareStorageForQueryResponse(item *logmodel.StorageCluster) {
	item.Endpoints = ""
	item.DatabaseName = ""
	item.Username = ""
	item.PasswordConfigured = false
	item.SkipTLSVerify = false
	item.Timeout = 0
	item.QueueMode = ""
	item.QueueEndpoints = ""
	item.LastError = ""
}

func (h *Handler) loadStorageCluster(c *gin.Context, id uint) (logmodel.StorageCluster, bool) {
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "日志存储 ID 无效")
		return logmodel.StorageCluster{}, false
	}
	item, err := h.loadStorageClusterByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "日志存储不存在")
		} else {
			response.ErrorCode(c, http.StatusInternalServerError, "读取日志存储失败: "+err.Error())
		}
		return logmodel.StorageCluster{}, false
	}
	return item, true
}

func (h *Handler) loadStorageClusterByID(id uint) (logmodel.StorageCluster, error) {
	var item logmodel.StorageCluster
	err := h.db.First(&item, id).Error
	return item, err
}

func (h *Handler) internalStorage(c *gin.Context, storageID uint) (logmodel.StorageCluster, string, bool) {
	var item logmodel.StorageCluster
	query := h.db.Where("enabled = ? AND storage_type = ?", true, "clickhouse")
	if storageID > 0 {
		query = query.Where("id = ?", storageID)
	} else {
		query = query.Order("is_primary DESC, created_at ASC")
	}
	if err := query.First(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "没有可用的 ClickHouse 内置日志存储，请先在日志库中完成初始化")
		return item, "", false
	}
	if item.InitializedAt == nil {
		response.ErrorCode(c, http.StatusBadRequest, "ClickHouse 日志存储尚未初始化")
		return item, "", false
	}
	password, err := decryptStorageSecret(item.PasswordEncrypted)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 ClickHouse 密码失败: "+err.Error())
		return item, "", false
	}
	return item, password, true
}

func (h *Handler) saveInternalQueryHistory(c *gin.Context, req logsvc.InternalQueryRequest, durationMS int64, count int, status, errorMessage string) {
	if req.SkipHistory {
		return
	}
	start, end := parseRangeForHistory(req.Start, req.End)
	queryJSON, _ := json.Marshal(gin.H{"query": req.Query, "scope": req.Scope, "filters": req.Filters, "filterLogic": req.FilterLogic, "cursor": req.Cursor})
	scopeJSON, _ := json.Marshal(req.Scope)
	_ = h.db.Create(&logmodel.QueryHistory{
		UserID: rbacsvc.GetUserID(c), DataSourceID: 0, DataSourceType: "internal_clickhouse",
		QueryLanguage: "structured", Query: string(queryJSON), StartTime: start, EndTime: end, Limit: req.Limit,
		DurationMS: durationMS, ResultCount: count, Status: status, ErrorMessage: errorMessage,
		SourceMode: "internal", AssetScope: string(scopeJSON),
	}).Error
}

func (h *Handler) userAccessibleHostIDs(c *gin.Context) ([]uint64, bool, error) {
	userID := rbacsvc.GetUserID(c)
	if userID == 0 {
		return []uint64{}, false, nil
	}
	var adminCount int64
	if strings.EqualFold(strings.TrimSpace(rbacsvc.GetUsername(c)), "admin") {
		return nil, true, nil
	}
	if err := h.db.WithContext(c.Request.Context()).Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error; err != nil {
		return nil, false, err
	}
	if adminCount > 0 {
		return nil, true, nil
	}
	var hostIDs []uint64
	err := h.db.WithContext(c.Request.Context()).Raw(`
		SELECT DISTINCT h.id
		FROM hosts AS h
		JOIN sys_role_asset_permission AS p ON p.asset_group_id = h.group_id
		JOIN sys_user_role AS ur ON p.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND h.deleted_at IS NULL
		AND p.deleted_at IS NULL
		AND (
			JSON_LENGTH(COALESCE(p.host_ids, JSON_ARRAY())) = 0
			OR JSON_CONTAINS(p.host_ids, CAST(h.id AS JSON))
		)
	`, userID).Scan(&hostIDs).Error
	return hostIDs, false, err
}

func encryptStorageSecret(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	key, err := storageEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return encryptedStorageSecretPrefix + base64.RawStdEncoding.EncodeToString(cipherText), nil
}

func decryptStorageSecret(value string) (string, error) {
	plainText, _, err := decryptStorageSecretWithKeyIndex(value)
	return plainText, err
}

func decryptStorageSecretWithKeyIndex(value string) (string, int, error) {
	if value == "" {
		return "", 0, nil
	}
	if !strings.HasPrefix(value, encryptedStorageSecretPrefix) {
		return value, -1, nil
	}
	keys, err := storageEncryptionKeys()
	if err != nil {
		return "", 0, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, encryptedStorageSecretPrefix))
	if err != nil {
		return "", 0, err
	}
	for index, key := range keys {
		block, blockErr := aes.NewCipher(key)
		if blockErr != nil {
			continue
		}
		gcm, gcmErr := cipher.NewGCM(block)
		if gcmErr != nil || len(raw) < gcm.NonceSize() {
			continue
		}
		plainText, openErr := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
		if openErr == nil {
			return string(plainText), index, nil
		}
	}
	return "", 0, fmt.Errorf("日志存储凭据解密失败，请检查当前密钥和旧密钥列表")
}

func storageEncryptionKey() ([]byte, error) {
	keys, err := storageEncryptionKeys()
	if err != nil {
		return nil, err
	}
	return keys[0], nil
}

func storageEncryptionKeys() ([][]byte, error) {
	secret := strings.TrimSpace(os.Getenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY"))
	if secret == "" && conf.Get() != nil {
		secret = strings.TrimSpace(conf.Get().Server.JWTSecret)
	}
	if secret == "" {
		return nil, fmt.Errorf("未配置 OPSHUB_LOGCENTER_ENCRYPTION_KEY 或 JWT Secret")
	}
	secrets := []string{secret}
	secrets = append(secrets, splitStorageEncryptionSecrets(os.Getenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS"))...)
	seen := make(map[string]struct{}, len(secrets))
	keys := make([][]byte, 0, len(secrets))
	for _, candidate := range secrets {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		sum := sha256.Sum256([]byte(candidate))
		key := make([]byte, len(sum))
		copy(key, sum[:])
		keys = append(keys, key)
	}
	return keys, nil
}

func splitStorageEncryptionSecrets(value string) []string {
	return strings.FieldsFunc(value, func(character rune) bool {
		return character == ',' || character == '\n' || character == '\r'
	})
}

// RotateStorageSecretsFromEnvironment rewrites plaintext or legacy-key credentials
// with the current key. It is idempotent and safe when multiple API replicas start.
func RotateStorageSecretsFromEnvironment(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var items []logmodel.StorageCluster
	if err := db.Where("password_encrypted IS NOT NULL AND password_encrypted <> ?", "").Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		plainText, keyIndex, err := decryptStorageSecretWithKeyIndex(item.PasswordEncrypted)
		if err != nil {
			return fmt.Errorf("日志存储 %d 凭据迁移失败: %w", item.ID, err)
		}
		if keyIndex == 0 {
			continue
		}
		encrypted, err := encryptStorageSecret(plainText)
		if err != nil {
			return fmt.Errorf("日志存储 %d 凭据重新加密失败: %w", item.ID, err)
		}
		if err := db.Model(&logmodel.StorageCluster{}).
			Where("id = ? AND password_encrypted = ?", item.ID, item.PasswordEncrypted).
			Update("password_encrypted", encrypted).Error; err != nil {
			return fmt.Errorf("日志存储 %d 凭据保存失败: %w", item.ID, err)
		}
	}
	return nil
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
