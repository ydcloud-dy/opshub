package server

import (
	"net/http"
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

func (h *Handler) GetOverview(c *gin.Context) {
	var storageCount int64
	h.db.Model(&logmodel.StorageCluster{}).Where("enabled = ?", true).Count(&storageCount)

	startOfDay := time.Now().Truncate(24 * time.Hour)
	var todayQueries int64
	h.db.Model(&logmodel.QueryHistory{}).Where("created_at >= ?", startOfDay).Count(&todayQueries)

	var todayFailures int64
	h.db.Model(&logmodel.QueryHistory{}).Where("created_at >= ? AND status = ?", startOfDay, "failed").Count(&todayFailures)

	var templateCount int64
	h.db.Model(&logmodel.QueryTemplate{}).Count(&templateCount)

	var libraryCount int64
	h.db.Model(&logmodel.LibraryItem{}).Count(&libraryCount)

	var recentHistories []logmodel.QueryHistory
	h.db.Order("created_at DESC").Limit(8).Find(&recentHistories)

	var hotTemplates []logmodel.QueryTemplate
	h.db.Order("sort ASC, updated_at DESC").Limit(8).Find(&hotTemplates)

	var recentLibraries []logmodel.LibraryItem
	h.db.Order("updated_at DESC").Limit(8).Find(&recentLibraries)

	response.Success(c, gin.H{
		"storageCount":  storageCount,
		"todayQueries":  todayQueries,
		"todayFailures": todayFailures,
		"templateCount": templateCount,
		"libraryCount":  libraryCount,
		"recentQueries": recentHistories,
		"hotTemplates":  hotTemplates,
		"recentLibrary": recentLibraries,
	})
}

func (h *Handler) ListHistories(c *gin.Context) {
	page, pageSize := parsePage(c)
	query := h.db.Model(&logmodel.QueryHistory{}).Order("created_at DESC")
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
	if err := h.db.Delete(&logmodel.QueryHistory{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除查询历史失败: "+err.Error())
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
	if len(req.IDs) > 0 {
		if err := h.db.Delete(&logmodel.QueryHistory{}, req.IDs).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "批量删除查询历史失败: "+err.Error())
			return
		}
	}
	response.Success(c, gin.H{"ids": req.IDs})
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
	if err := h.db.Delete(&logmodel.QueryTemplate{}, id).Error; err != nil {
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
	item.ID = 0
	item.Name = item.Name + " 副本"
	item.OwnerID = rbacsvc.GetUserID(c)
	if err := h.db.Create(&item).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "克隆查询模板失败: "+err.Error())
		return
	}
	response.Success(c, item)
}

func (h *Handler) ListLibrary(c *gin.Context) {
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
	var item logmodel.LibraryItem
	if err := h.db.First(&item, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志库不存在")
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateLibraryItem(c *gin.Context) {
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
	id := parseUint(c.Param("id"))
	if err := h.db.Delete(&logmodel.LibraryItem{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除日志库缓存失败: "+err.Error())
		return
	}
	h.db.Where("library_item_id = ?", id).Delete(&logmodel.FieldCatalog{})
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ListLibraryFields(c *gin.Context) {
	id := parseUint(c.Param("id"))
	var fields []logmodel.FieldCatalog
	if err := h.db.Where("library_item_id = ?", id).Order("field_name ASC").Find(&fields).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询字段失败: "+err.Error())
		return
	}
	response.Success(c, fields)
}

func (h *Handler) UpdateLibraryField(c *gin.Context) {
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
