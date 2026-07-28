package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"gorm.io/gorm"
)

type Handler struct {
	db            *gorm.DB
	clickhouse    *logsvc.ClickHouseService
	internalQuery *logsvc.InternalQueryService
	tailSlots     chan struct{}
	exporter      *logExportManager
}

func NewHandler(db *gorm.DB) *Handler {
	handler := &Handler{
		db:            db,
		clickhouse:    logsvc.NewClickHouseService(),
		internalQuery: logsvc.NewInternalQueryService(),
		tailSlots:     make(chan struct{}, positiveEnvInt("OPSHUB_LOG_TAIL_MAX_CONNECTIONS", 50)),
	}
	handler.exporter = newLogExportManager(db, handler.internalQuery)
	return handler
}

var logOverviewDatabasePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type logOverviewCollectorStats struct {
	Total        int64      `json:"total"`
	Online       int64      `json:"online"`
	Errors       int64      `json:"errors"`
	InputEPS     float64    `json:"inputEps"`
	OutputEPS    float64    `json:"outputEps"`
	WALBytes     int64      `json:"walBytes"`
	DroppedTotal uint64     `json:"droppedTotal"`
	RetryTotal   uint64     `json:"retryTotal"`
	LastIngestAt *time.Time `json:"lastIngestAt,omitempty"`
}

type logOverviewPoint struct {
	Name  string `json:"name"`
	Value uint64 `json:"value"`
}

type logOverviewTrendPoint struct {
	Time  string `json:"time"`
	Count uint64 `json:"count"`
	Bytes uint64 `json:"bytes"`
}

type logOverviewTrendConfig struct {
	Range    string
	Interval string
	Bucket   string
	Key      string
}

func logOverviewTrendConfigFor(value string) logOverviewTrendConfig {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "30d":
		return logOverviewTrendConfig{
			Range: "30d", Interval: "30 DAY",
			Bucket: "toStartOfDay(minute, 'Asia/Shanghai')",
			Key:    "formatDateTime(toStartOfDay(minute, 'Asia/Shanghai'), '%Y-%m-%d', 'Asia/Shanghai')",
		}
	case "12m":
		return logOverviewTrendConfig{
			Range: "12m", Interval: "12 MONTH",
			Bucket: "toStartOfMonth(minute, 'Asia/Shanghai')",
			Key:    "formatDateTime(toStartOfMonth(minute, 'Asia/Shanghai'), '%Y-%m', 'Asia/Shanghai')",
		}
	default:
		return logOverviewTrendConfig{
			Range: "24h", Interval: "24 HOUR",
			Bucket: "toStartOfHour(minute)",
			Key:    "formatDateTime(toStartOfHour(minute), '%Y-%m-%dT%H:00:00Z', 'UTC')",
		}
	}
}

type logOverviewSnapshot struct {
	Logs24H        uint64                  `json:"logs24h"`
	Bytes24H       uint64                  `json:"bytes24h"`
	Errors24H      uint64                  `json:"errors24h"`
	AverageEPS5M   float64                 `json:"averageEps5m"`
	ActiveServices uint64                  `json:"activeServices"`
	Trend          []logOverviewTrendPoint `json:"trend"`
	Levels         []logOverviewPoint      `json:"levels"`
	TopServices    []logOverviewPoint      `json:"topServices"`
	Sources        []logOverviewPoint      `json:"sources"`
}

func (h *Handler) GetOverview(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以查看日志总览") {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	trendConfig := logOverviewTrendConfigFor(c.Query("trendRange"))

	collectorStats, err := h.loadLogOverviewCollectorStats(ctx)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志采集状态失败: "+err.Error())
		return
	}

	snapshot := logOverviewSnapshot{
		Trend: []logOverviewTrendPoint{}, Levels: []logOverviewPoint{},
		TopServices: []logOverviewPoint{}, Sources: []logOverviewPoint{},
	}
	storageInfo := gin.H{"available": false}
	var storage logmodel.StorageCluster
	storageErr := h.db.WithContext(ctx).Where("enabled = ?", true).Order("is_primary DESC, id ASC").First(&storage).Error
	if storageErr == nil {
		storageInfo["id"] = storage.ID
		storageInfo["name"] = storage.Name
		storageInfo["status"] = storage.Status
		password, decryptErr := decryptStorageSecret(storage.PasswordEncrypted)
		if decryptErr != nil {
			storageInfo["error"] = "ClickHouse 凭据解密失败"
		} else if overview, queryErr := h.queryLogOverviewSnapshot(ctx, storage, password, trendConfig); queryErr != nil {
			storageInfo["error"] = queryErr.Error()
		} else {
			snapshot = overview
			storageInfo["available"] = true
		}
	} else if errors.Is(storageErr, gorm.ErrRecordNotFound) {
		storageInfo["error"] = "尚未配置可用的 ClickHouse 日志存储"
	} else {
		storageInfo["error"] = storageErr.Error()
	}

	response.Success(c, gin.H{
		"logs24h": snapshot.Logs24H, "bytes24h": snapshot.Bytes24H,
		"errors24h": snapshot.Errors24H, "averageEps5m": snapshot.AverageEPS5M,
		"activeServices": snapshot.ActiveServices, "trend": snapshot.Trend,
		"levels": snapshot.Levels, "topServices": snapshot.TopServices,
		"sources": snapshot.Sources, "collectors": collectorStats,
		"storage": storageInfo, "trendRange": trendConfig.Range, "updatedAt": time.Now(),
	})
}

func (h *Handler) loadLogOverviewCollectorStats(ctx context.Context) (logOverviewCollectorStats, error) {
	var result logOverviewCollectorStats
	cutoff := time.Now().Add(-90 * time.Second)
	err := h.db.WithContext(ctx).Model(&logmodel.CollectorInstance{}).
		Select(`COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN last_heartbeat_at >= ? THEN 1 ELSE 0 END), 0) AS online,
			COALESCE(SUM(CASE WHEN last_heartbeat_at >= ? AND last_error <> '' THEN 1 ELSE 0 END), 0) AS errors,
			COALESCE(SUM(CASE WHEN last_heartbeat_at >= ? THEN input_eps ELSE 0 END), 0) AS input_eps,
			COALESCE(SUM(CASE WHEN last_heartbeat_at >= ? THEN output_eps ELSE 0 END), 0) AS output_eps,
			COALESCE(SUM(wal_bytes), 0) AS wal_bytes,
			COALESCE(SUM(dropped_total), 0) AS dropped_total,
			COALESCE(SUM(retry_total), 0) AS retry_total,
			MAX(last_ingest_at) AS last_ingest_at`, cutoff, cutoff, cutoff, cutoff).
		Scan(&result).Error
	return result, err
}

func (h *Handler) queryLogOverviewSnapshot(ctx context.Context, storage logmodel.StorageCluster, password string, trendConfig logOverviewTrendConfig) (logOverviewSnapshot, error) {
	database := strings.TrimSpace(storage.DatabaseName)
	if database == "" {
		database = "opshub_logs"
	}
	if !logOverviewDatabasePattern.MatchString(database) {
		return logOverviewSnapshot{}, fmt.Errorf("ClickHouse 数据库名称无效")
	}
	query := fmt.Sprintf(`
SELECT 'summary' AS category, 'logs24h' AS metric_key, toFloat64(sum(log_count)) AS metric_value
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
UNION ALL
SELECT 'summary', 'bytes24h', toFloat64(sum(byte_count))
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
UNION ALL
SELECT 'summary', 'errors24h', toFloat64(sumIf(log_count, upperUTF8(level) IN ('ERROR', 'FATAL')))
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
UNION ALL
SELECT 'summary', 'averageEps5m', toFloat64(sum(log_count)) / 300
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 5 MINUTE
UNION ALL
SELECT 'summary', 'activeServices', toFloat64(uniqExactIf(service, service != ''))
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
UNION ALL
SELECT 'trend', %[2]s, toFloat64(sum(log_count))
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL %[3]s GROUP BY %[4]s
UNION ALL
SELECT 'level', if(level = '', 'UNKNOWN', upperUTF8(level)), toFloat64(sum(log_count))
FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR GROUP BY level
UNION ALL
SELECT 'service', metric_key, metric_value FROM (
	SELECT if(service = '', '未标注服务', service) AS metric_key, toFloat64(sum(log_count)) AS metric_value
	FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
	GROUP BY service ORDER BY metric_value DESC LIMIT 8
)
UNION ALL
SELECT 'source', metric_key, metric_value FROM (
	SELECT if(asset_type = '', 'unknown', asset_type) AS metric_key, toFloat64(sum(log_count)) AS metric_value
	FROM %[1]s.opshub_log_metrics_1m WHERE minute >= now() - INTERVAL 24 HOUR
	GROUP BY asset_type ORDER BY metric_value DESC LIMIT 8
)`, database, trendConfig.Key, trendConfig.Interval, trendConfig.Bucket)
	rows, err := h.clickhouse.QueryJSONEachRow(ctx, storage, password, query, nil)
	if err != nil {
		return logOverviewSnapshot{}, fmt.Errorf("读取 ClickHouse 日志指标失败: %w", err)
	}
	trendBytesQuery := fmt.Sprintf(`
SELECT %[2]s AS metric_key, toFloat64(sum(byte_count)) AS metric_value
FROM %[1]s.opshub_log_metrics_1m
WHERE minute >= now() - INTERVAL %[3]s
GROUP BY %[4]s`, database, trendConfig.Key, trendConfig.Interval, trendConfig.Bucket)
	trendBytesRows, err := h.clickhouse.QueryJSONEachRow(ctx, storage, password, trendBytesQuery, nil)
	if err != nil {
		return logOverviewSnapshot{}, fmt.Errorf("读取 ClickHouse 日志趋势数据量失败: %w", err)
	}
	for _, row := range trendBytesRows {
		row["category"] = "trendBytes"
		rows = append(rows, row)
	}
	return buildLogOverviewSnapshot(rows), nil
}

func buildLogOverviewSnapshot(rows []map[string]interface{}) logOverviewSnapshot {
	result := logOverviewSnapshot{
		Trend: []logOverviewTrendPoint{}, Levels: []logOverviewPoint{},
		TopServices: []logOverviewPoint{}, Sources: []logOverviewPoint{},
	}
	for _, row := range rows {
		category := strings.TrimSpace(fmt.Sprint(row["category"]))
		name := strings.TrimSpace(fmt.Sprint(row["metric_key"]))
		value := logOverviewNumber(row["metric_value"])
		switch category {
		case "summary":
			switch name {
			case "logs24h":
				result.Logs24H = uint64(value)
			case "bytes24h":
				result.Bytes24H = uint64(value)
			case "errors24h":
				result.Errors24H = uint64(value)
			case "averageEps5m":
				result.AverageEPS5M = value
			case "activeServices":
				result.ActiveServices = uint64(value)
			}
		case "trend":
			result.Trend = append(result.Trend, logOverviewTrendPoint{
				Time: name, Count: uint64(value), Bytes: uint64(logOverviewNumber(row["metric_bytes"])),
			})
		case "trendBytes":
			for index := range result.Trend {
				if result.Trend[index].Time == name {
					result.Trend[index].Bytes = uint64(value)
					break
				}
			}
		case "level":
			result.Levels = append(result.Levels, logOverviewPoint{Name: name, Value: uint64(value)})
		case "service":
			result.TopServices = append(result.TopServices, logOverviewPoint{Name: name, Value: uint64(value)})
		case "source":
			result.Sources = append(result.Sources, logOverviewPoint{Name: name, Value: uint64(value)})
		}
	}
	sort.Slice(result.Trend, func(left, right int) bool { return result.Trend[left].Time < result.Trend[right].Time })
	sort.Slice(result.Levels, func(left, right int) bool { return result.Levels[left].Value > result.Levels[right].Value })
	sort.Slice(result.TopServices, func(left, right int) bool { return result.TopServices[left].Value > result.TopServices[right].Value })
	sort.Slice(result.Sources, func(left, right int) bool { return result.Sources[left].Value > result.Sources[right].Value })
	return result
}

func logOverviewNumber(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return typed
		}
	case float32:
		if typed > 0 {
			return float64(typed)
		}
	case int:
		if typed > 0 {
			return float64(typed)
		}
	case int64:
		if typed > 0 {
			return float64(typed)
		}
	case uint64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if parsed > 0 {
			return parsed
		}
	}
	return 0
}

func (h *Handler) ListHistories(c *gin.Context) {
	page, pageSize := parsePage(c)
	query := h.db.Model(&logmodel.QueryHistory{}).
		Where("user_id = ?", rbacsvc.GetUserID(c)).
		Order("created_at DESC")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("query LIKE ?", "%"+keyword+"%")
	}
	if ds := parseUint(c.Query("datasourceId")); ds > 0 {
		query = query.Where("data_source_id = ?", ds)
	}
	if typ := strings.TrimSpace(c.Query("datasourceType")); typ != "" {
		query = query.Where("data_source_type = ?", typ)
	}
	var total int64
	query.Count(&total)
	var items []logmodel.QueryHistory
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询历史失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": items})
}

func (h *Handler) DeleteHistory(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "ID 无效")
		return
	}
	result := h.db.Where("id = ? AND user_id = ?", id, rbacsvc.GetUserID(c)).Delete(&logmodel.QueryHistory{})
	if result.Error != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除查询历史失败: "+result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		response.ErrorCode(c, http.StatusNotFound, "查询历史不存在")
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) BatchDeleteHistories(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ownedIDs := make([]uint, 0, len(req.IDs))
	if len(req.IDs) > 0 {
		if err := h.db.Model(&logmodel.QueryHistory{}).
			Where("user_id = ? AND id IN ?", rbacsvc.GetUserID(c), req.IDs).
			Pluck("id", &ownedIDs).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取查询历史失败: "+err.Error())
			return
		}
		if len(ownedIDs) > 0 {
			if err := h.db.Where("user_id = ? AND id IN ?", rbacsvc.GetUserID(c), ownedIDs).Delete(&logmodel.QueryHistory{}).Error; err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "批量删除查询历史失败: "+err.Error())
				return
			}
		}
	}
	response.Success(c, gin.H{"ids": ownedIDs})
}

func (h *Handler) ListSavedViews(c *gin.Context) {
	page, pageSize := parsePage(c)
	query := h.db.Model(&logmodel.SavedView{}).Order("updated_at DESC")
	userID := rbacsvc.GetUserID(c)
	query = query.Where("user_id = ? OR is_public = ?", userID, true)
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR query LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if ds := parseUint(c.Query("datasourceId")); ds > 0 {
		query = query.Where("data_source_id = ?", ds)
	}
	var total int64
	query.Count(&total)
	var items []logmodel.SavedView
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询保存视图失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": items})
}

func (h *Handler) GetSavedView(c *gin.Context) {
	var item logmodel.SavedView
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "保存视图不存在")
		return
	}
	if item.UserID != rbacsvc.GetUserID(c) && !item.IsPublic {
		response.ErrorCode(c, http.StatusForbidden, "没有权限查看该保存视图")
		return
	}
	response.Success(c, item)
}

func (h *Handler) CreateSavedView(c *gin.Context) {
	var item logmodel.SavedView
	if err := c.ShouldBindJSON(&item); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item.ID = 0
	item.UserID = rbacsvc.GetUserID(c)
	fillSavedViewDefaults(&item)
	if err := h.db.Create(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存视图失败: "+err.Error())
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateSavedView(c *gin.Context) {
	var existing logmodel.SavedView
	if err := h.db.First(&existing, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "保存视图不存在")
		return
	}
	if existing.UserID != rbacsvc.GetUserID(c) {
		response.ErrorCode(c, http.StatusForbidden, "只能修改自己创建的保存视图")
		return
	}
	var req logmodel.SavedView
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	req.ID = existing.ID
	req.UserID = existing.UserID
	req.CreatedAt = existing.CreatedAt
	fillSavedViewDefaults(&req)
	if err := h.db.Save(&req).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新保存视图失败: "+err.Error())
		return
	}
	response.Success(c, req)
}

func (h *Handler) DeleteSavedView(c *gin.Context) {
	var item logmodel.SavedView
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "保存视图不存在")
		return
	}
	if item.UserID != rbacsvc.GetUserID(c) {
		response.ErrorCode(c, http.StatusForbidden, "只能删除自己创建的保存视图")
		return
	}
	if err := h.db.Delete(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除保存视图失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": item.ID})
}

func (h *Handler) GetAlertContext(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志告警上下文") {
		return
	}
	ruleID := parseUint(c.Param("ruleId"))
	var context logmodel.AlertContext
	if err := h.db.Where("rule_id = ?", ruleID).First(&context).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			context = logmodel.AlertContext{
				RuleID:             ruleID,
				SampleLimit:        5,
				IncludeLogInNotice: true,
				JumpTimeWindow:     900,
			}
			response.Success(c, context)
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志告警上下文失败: "+err.Error())
		return
	}
	response.Success(c, context)
}

func (h *Handler) UpdateAlertContext(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志告警上下文") {
		return
	}
	ruleID := parseUint(c.Param("ruleId"))
	var req logmodel.AlertContext
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	var context logmodel.AlertContext
	err := h.db.Where("rule_id = ?", ruleID).First(&context).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志告警上下文失败: "+err.Error())
		return
	}
	if err == gorm.ErrRecordNotFound {
		context.RuleID = ruleID
	}
	context.SampleQuery = req.SampleQuery
	context.SampleLimit = normalizePositive(req.SampleLimit, 5)
	context.ContextBefore = req.ContextBefore
	context.ContextAfter = req.ContextAfter
	context.IncludeLogInNotice = req.IncludeLogInNotice
	context.JumpTimeWindow = normalizePositive(req.JumpTimeWindow, 900)
	context.HighlightKeywords = req.HighlightKeywords
	if context.ID == 0 {
		err = h.db.Create(&context).Error
	} else {
		err = h.db.Save(&context).Error
	}
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存日志告警上下文失败: "+err.Error())
		return
	}
	response.Success(c, context)
}

func (h *Handler) ListTemplates(c *gin.Context) {
	page, pageSize := parsePage(c)
	query := h.db.Model(&logmodel.QueryTemplate{}).Order("sort ASC, updated_at DESC")
	isAdmin, ok := h.logAdminStatus(c)
	if !ok {
		return
	}
	if !isAdmin {
		query = query.Where("owner_id = ? OR is_public = ?", rbacsvc.GetUserID(c), true)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR query LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	if typ := strings.TrimSpace(c.Query("datasourceType")); typ != "" {
		query = query.Where("data_source_type = ?", typ)
	}
	var total int64
	query.Count(&total)
	var items []logmodel.QueryTemplate
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询模板失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": items})
}

func (h *Handler) GetTemplate(c *gin.Context) {
	var item logmodel.QueryTemplate
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "查询模板不存在")
		return
	}
	if !h.authorizeTemplate(c, item, false) {
		return
	}
	response.Success(c, item)
}

func (h *Handler) CreateTemplate(c *gin.Context) {
	var item logmodel.QueryTemplate
	if err := c.ShouldBindJSON(&item); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item.ID = 0
	item.OwnerID = rbacsvc.GetUserID(c)
	fillTemplateDefaults(&item)
	if err := h.db.Create(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建查询模板失败: "+err.Error())
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateTemplate(c *gin.Context) {
	var existing logmodel.QueryTemplate
	if err := h.db.First(&existing, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "查询模板不存在")
		return
	}
	if !h.authorizeTemplate(c, existing, true) {
		return
	}
	var req logmodel.QueryTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	req.ID = existing.ID
	req.OwnerID = existing.OwnerID
	req.CreatedAt = existing.CreatedAt
	fillTemplateDefaults(&req)
	if err := h.db.Save(&req).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新查询模板失败: "+err.Error())
		return
	}
	response.Success(c, req)
}

func (h *Handler) DeleteTemplate(c *gin.Context) {
	id := parseUint(c.Param("id"))
	var item logmodel.QueryTemplate
	if err := h.db.First(&item, id).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "查询模板不存在")
		return
	}
	if !h.authorizeTemplate(c, item, true) {
		return
	}
	if err := h.db.Delete(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除查询模板失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) CloneTemplate(c *gin.Context) {
	var item logmodel.QueryTemplate
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "查询模板不存在")
		return
	}
	if !h.authorizeTemplate(c, item, false) {
		return
	}
	item.ID = 0
	item.Name = item.Name + " 副本"
	item.OwnerID = rbacsvc.GetUserID(c)
	if err := h.db.Create(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "克隆查询模板失败: "+err.Error())
		return
	}
	response.Success(c, item)
}

func (h *Handler) authorizeTemplate(c *gin.Context, item logmodel.QueryTemplate, write bool) bool {
	isAdmin, ok := h.logAdminStatus(c)
	if !ok {
		return false
	}
	allowed := templateAccessAllowed(item, rbacsvc.GetUserID(c), isAdmin, write)
	if !allowed {
		response.ErrorCode(c, http.StatusNotFound, "查询模板不存在")
		return false
	}
	return true
}

func templateAccessAllowed(item logmodel.QueryTemplate, userID uint, isAdmin, write bool) bool {
	return isAdmin || item.OwnerID == userID || (!write && item.IsPublic)
}

func (h *Handler) ListLibrary(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	page, pageSize := parsePage(c)
	query := h.db.Model(&logmodel.LibraryItem{}).Order("updated_at DESC")
	if ds := parseUint(c.Query("datasourceId")); ds > 0 {
		query = query.Where("data_source_id = ?", ds)
	}
	if typ := strings.TrimSpace(c.Query("datasourceType")); typ != "" {
		query = query.Where("data_source_type = ?", typ)
	}
	if itemType := strings.TrimSpace(c.Query("itemType")); itemType != "" {
		query = query.Where("item_type = ?", itemType)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	query.Count(&total)
	var items []logmodel.LibraryItem
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询日志库失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": items})
}

func (h *Handler) GetLibraryItem(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	var item logmodel.LibraryItem
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志库不存在")
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateLibraryItem(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	var existing logmodel.LibraryItem
	if err := h.db.First(&existing, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志库不存在")
		return
	}
	var req logmodel.LibraryItem
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	updates := map[string]interface{}{
		"display_name":   req.DisplayName,
		"description":    req.Description,
		"owner":          req.Owner,
		"environment":    req.Environment,
		"retention_days": req.RetentionDays,
		"status":         firstNonEmpty(req.Status, existing.Status),
	}
	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志库失败: "+err.Error())
		return
	}
	h.db.First(&existing, existing.ID)
	response.Success(c, existing)
}

func (h *Handler) DeleteLibraryItem(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	id := parseUint(c.Param("id"))
	if err := h.db.Delete(&logmodel.LibraryItem{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除日志库缓存失败: "+err.Error())
		return
	}
	h.db.Where("library_item_id = ?", id).Delete(&logmodel.FieldCatalog{})
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ListLibraryFields(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	id := parseUint(c.Param("id"))
	var fields []logmodel.FieldCatalog
	if err := h.db.Where("library_item_id = ?", id).Order("field_name ASC").Find(&fields).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询字段失败: "+err.Error())
		return
	}
	response.Success(c, fields)
}

func (h *Handler) UpdateLibraryField(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以管理日志库") {
		return
	}
	var field logmodel.FieldCatalog
	if err := h.db.Where("id = ? AND library_item_id = ?", parseUint(c.Param("fieldId")), parseUint(c.Param("id"))).First(&field).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "字段不存在")
		return
	}
	var req logmodel.FieldCatalog
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	updates := map[string]interface{}{
		"display_name":     req.DisplayName,
		"is_time_field":    req.IsTimeField,
		"is_message_field": req.IsMessageField,
		"is_level_field":   req.IsLevelField,
		"is_sensitive":     req.IsSensitive,
	}
	if err := h.db.Model(&field).Updates(updates).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新字段失败: "+err.Error())
		return
	}
	h.db.First(&field, field.ID)
	response.Success(c, field)
}

func fillTemplateDefaults(item *logmodel.QueryTemplate) {
	item.Name = strings.TrimSpace(item.Name)
	item.Query = strings.TrimSpace(item.Query)
	item.DataSourceType = "internal_clickhouse"
	item.QueryLanguage = "structured"
	item.TimeRange = firstNonEmpty(item.TimeRange, "15m")
	if item.Category == "" {
		item.Category = "常用查询"
	}
}

func fillSavedViewDefaults(item *logmodel.SavedView) {
	item.Name = strings.TrimSpace(item.Name)
	item.Query = strings.TrimSpace(item.Query)
	item.QueryLanguage = "structured"
	item.TimeRange = firstNonEmpty(item.TimeRange, "15m")
	if strings.TrimSpace(item.DisplayOptions) == "" {
		item.DisplayOptions = `{"wrapLog":true,"showLabels":true,"showFields":true}`
	}
}

func parsePage(c *gin.Context) (int, int) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 20)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func parseRangeForHistory(startRaw, endRaw string) (time.Time, time.Time) {
	end := parseFlexibleTime(endRaw)
	if end.IsZero() {
		end = time.Now()
	}
	start := parseFlexibleTime(startRaw)
	if start.IsZero() {
		start = end.Add(-15 * time.Minute)
	}
	return start, end
}

func parseFlexibleTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizePositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parseUint(raw string) uint {
	value, _ := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	return uint(value)
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
