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

package repository

import (
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NginxRepository Nginx 数据仓库
type NginxRepository struct {
	db *gorm.DB

	// 维度缓存
	ipCache      sync.Map // map[string]uint64
	urlCache     sync.Map // map[string]uint64
	refererCache sync.Map // map[string]uint64
	uaCache      sync.Map // map[string]uint64

	// 表存在性缓存
	tableExistsCache sync.Map // map[string]bool
}

// NewNginxRepository 创建仓库实例
func NewNginxRepository(db *gorm.DB) *NginxRepository {
	return &NginxRepository{db: db}
}

// tableExists 检查表是否存在
func (r *NginxRepository) tableExists(tableName string) bool {
	// 先从缓存查找
	if exists, ok := r.tableExistsCache.Load(tableName); ok {
		return exists.(bool)
	}

	// 查询数据库
	var count int64
	r.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
	exists := count > 0
	r.tableExistsCache.Store(tableName, exists)
	return exists
}

// ============== 数据源操作 ==============

// CreateSource 创建数据源
func (r *NginxRepository) CreateSource(source *model.NginxSource) error {
	return r.db.Create(source).Error
}

// UpdateSource 更新数据源
func (r *NginxRepository) UpdateSource(source *model.NginxSource) error {
	return r.db.Save(source).Error
}

// DeleteSource 删除数据源及其关联数据
func (r *NginxRepository) DeleteSource(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 删除关联的访问日志 (旧表)
		if r.tableExists("nginx_access_logs") {
			tx.Where("source_id = ?", id).Delete(&model.NginxAccessLog{})
		}
		// 删除关联的访问日志 (新表)
		if r.tableExists("nginx_fact_access_logs") {
			tx.Where("source_id = ?", id).Delete(&model.NginxFactAccessLog{})
		}
		// 删除关联的日统计 (旧表)
		if r.tableExists("nginx_daily_stats") {
			tx.Where("source_id = ?", id).Delete(&model.NginxDailyStats{})
		}
		// 删除关联的小时统计 (旧表)
		if r.tableExists("nginx_hourly_stats") {
			tx.Where("source_id = ?", id).Delete(&model.NginxHourlyStats{})
		}
		// 删除关联的聚合 (新表)
		if r.tableExists("nginx_agg_hourly") {
			tx.Where("source_id = ?", id).Delete(&model.NginxAggHourly{})
		}
		if r.tableExists("nginx_agg_daily") {
			tx.Where("source_id = ?", id).Delete(&model.NginxAggDaily{})
		}
		// 删除数据源
		if err := tx.Delete(&model.NginxSource{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetSourceByID 根据ID获取数据源
func (r *NginxRepository) GetSourceByID(id uint) (*model.NginxSource, error) {
	var source model.NginxSource
	err := r.db.First(&source, id).Error
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// ListSources 获取数据源列表
func (r *NginxRepository) ListSources(page, pageSize int, sourceType string, status *int) ([]model.NginxSource, int64, error) {
	var sources []model.NginxSource
	var total int64

	query := r.db.Model(&model.NginxSource{})

	if sourceType != "" {
		query = query.Where("type = ?", sourceType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&sources).Error
	if err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// GetActiveSources 获取活跃数据源
func (r *NginxRepository) GetActiveSources() ([]model.NginxSource, error) {
	var sources []model.NginxSource
	err := r.db.Where("status = ?", 1).Find(&sources).Error
	return sources, err
}

// UpdateSourceCollectStatus 更新数据源采集状态
func (r *NginxRepository) UpdateSourceCollectStatus(id uint, logsCount int64, err string) error {
	now := time.Now()
	return r.db.Model(&model.NginxSource{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_collect_at":   &now,
		"last_collect_logs": logsCount,
		"last_error":        err,
	}).Error
}

// ============== 维度表操作 ==============

// GetOrCreateDimIP 获取或创建 IP 维度
func (r *NginxRepository) GetOrCreateDimIP(ip string) (uint64, error) {
	// 先从缓存查找
	if id, ok := r.ipCache.Load(ip); ok {
		return id.(uint64), nil
	}

	var dim model.NginxDimIP
	err := r.db.Where("ip_address = ?", ip).First(&dim).Error
	if err == nil {
		r.ipCache.Store(ip, dim.ID)
		return dim.ID, nil
	}

	if err == gorm.ErrRecordNotFound {
		dim = model.NginxDimIP{
			IPAddress: ip,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "ip_address"}},
			DoNothing: true,
		}).Create(&dim).Error
		if err != nil {
			return 0, err
		}

		// 如果是 DoNothing，需要重新查询获取 ID
		if dim.ID == 0 {
			err = r.db.Where("ip_address = ?", ip).First(&dim).Error
			if err != nil {
				return 0, err
			}
		}
		r.ipCache.Store(ip, dim.ID)
		return dim.ID, nil
	}
	return 0, err
}

// UpdateDimIPGeo 更新 IP 地理位置信息
func (r *NginxRepository) UpdateDimIPGeo(id uint64, country, province, city, isp string, isBot bool) error {
	return r.db.Model(&model.NginxDimIP{}).Where("id = ?", id).Updates(map[string]interface{}{
		"country":    country,
		"province":   province,
		"city":       city,
		"isp":        isp,
		"is_bot":     isBot,
		"updated_at": time.Now(),
	}).Error
}

// GetDimIP 获取 IP 维度
func (r *NginxRepository) GetDimIP(id uint64) (*model.NginxDimIP, error) {
	var dim model.NginxDimIP
	err := r.db.First(&dim, id).Error
	if err != nil {
		return nil, err
	}
	return &dim, nil
}

// GetDimIPByAddress 根据 IP 地址获取维度
func (r *NginxRepository) GetDimIPByAddress(ip string) (*model.NginxDimIP, error) {
	var dim model.NginxDimIP
	err := r.db.Where("ip_address = ?", ip).First(&dim).Error
	if err != nil {
		return nil, err
	}
	return &dim, nil
}

// GetOrCreateDimURL 获取或创建 URL 维度
func (r *NginxRepository) GetOrCreateDimURL(urlHash, urlPath, urlNormalized, host string) (uint64, error) {
	if id, ok := r.urlCache.Load(urlHash); ok {
		return id.(uint64), nil
	}

	var dim model.NginxDimURL
	err := r.db.Where("url_hash = ?", urlHash).First(&dim).Error
	if err == nil {
		r.urlCache.Store(urlHash, dim.ID)
		return dim.ID, nil
	}

	if err == gorm.ErrRecordNotFound {
		dim = model.NginxDimURL{
			URLHash:       urlHash,
			URLPath:       urlPath,
			URLNormalized: urlNormalized,
			Host:          host,
			CreatedAt:     time.Now(),
		}
		err = r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "url_hash"}},
			DoNothing: true,
		}).Create(&dim).Error
		if err != nil {
			return 0, err
		}

		if dim.ID == 0 {
			err = r.db.Where("url_hash = ?", urlHash).First(&dim).Error
			if err != nil {
				return 0, err
			}
		}
		r.urlCache.Store(urlHash, dim.ID)
		return dim.ID, nil
	}
	return 0, err
}

// GetDimURL 获取 URL 维度
func (r *NginxRepository) GetDimURL(id uint64) (*model.NginxDimURL, error) {
	var dim model.NginxDimURL
	err := r.db.First(&dim, id).Error
	if err != nil {
		return nil, err
	}
	return &dim, nil
}

// GetOrCreateDimReferer 获取或创建 Referer 维度
func (r *NginxRepository) GetOrCreateDimReferer(refererHash, refererURL, refererDomain, refererType string) (uint64, error) {
	if id, ok := r.refererCache.Load(refererHash); ok {
		return id.(uint64), nil
	}

	var dim model.NginxDimReferer
	err := r.db.Where("referer_hash = ?", refererHash).First(&dim).Error
	if err == nil {
		r.refererCache.Store(refererHash, dim.ID)
		return dim.ID, nil
	}

	if err == gorm.ErrRecordNotFound {
		dim = model.NginxDimReferer{
			RefererHash:   refererHash,
			RefererURL:    refererURL,
			RefererDomain: refererDomain,
			RefererType:   refererType,
			CreatedAt:     time.Now(),
		}
		err = r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "referer_hash"}},
			DoNothing: true,
		}).Create(&dim).Error
		if err != nil {
			return 0, err
		}

		if dim.ID == 0 {
			err = r.db.Where("referer_hash = ?", refererHash).First(&dim).Error
			if err != nil {
				return 0, err
			}
		}
		r.refererCache.Store(refererHash, dim.ID)
		return dim.ID, nil
	}
	return 0, err
}

// GetDimReferer 获取 Referer 维度
func (r *NginxRepository) GetDimReferer(id uint64) (*model.NginxDimReferer, error) {
	var dim model.NginxDimReferer
	err := r.db.First(&dim, id).Error
	if err != nil {
		return nil, err
	}
	return &dim, nil
}

// GetOrCreateDimUserAgent 获取或创建 UserAgent 维度
func (r *NginxRepository) GetOrCreateDimUserAgent(uaHash, userAgent, browser, browserVersion, os, osVersion, deviceType string, isBot bool) (uint64, error) {
	if id, ok := r.uaCache.Load(uaHash); ok {
		return id.(uint64), nil
	}

	var dim model.NginxDimUserAgent
	err := r.db.Where("ua_hash = ?", uaHash).First(&dim).Error
	if err == nil {
		r.uaCache.Store(uaHash, dim.ID)
		return dim.ID, nil
	}

	if err == gorm.ErrRecordNotFound {
		dim = model.NginxDimUserAgent{
			UAHash:         uaHash,
			UserAgent:      userAgent,
			Browser:        browser,
			BrowserVersion: browserVersion,
			OS:             os,
			OSVersion:      osVersion,
			DeviceType:     deviceType,
			IsBot:          isBot,
			CreatedAt:      time.Now(),
		}
		err = r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "ua_hash"}},
			DoNothing: true,
		}).Create(&dim).Error
		if err != nil {
			return 0, err
		}

		if dim.ID == 0 {
			err = r.db.Where("ua_hash = ?", uaHash).First(&dim).Error
			if err != nil {
				return 0, err
			}
		}
		r.uaCache.Store(uaHash, dim.ID)
		return dim.ID, nil
	}
	return 0, err
}

// GetDimUserAgent 获取 UserAgent 维度
func (r *NginxRepository) GetDimUserAgent(id uint64) (*model.NginxDimUserAgent, error) {
	var dim model.NginxDimUserAgent
	err := r.db.First(&dim, id).Error
	if err != nil {
		return nil, err
	}
	return &dim, nil
}

// ClearDimCache 清除维度缓存
func (r *NginxRepository) ClearDimCache() {
	r.ipCache = sync.Map{}
	r.urlCache = sync.Map{}
	r.refererCache = sync.Map{}
	r.uaCache = sync.Map{}
}

// ============== 事实表操作 ==============

// CreateFactAccessLog 创建访问日志事实
func (r *NginxRepository) CreateFactAccessLog(log *model.NginxFactAccessLog) error {
	return r.db.Create(log).Error
}

// BatchCreateFactAccessLogs 批量创建访问日志事实
func (r *NginxRepository) BatchCreateFactAccessLogs(logs []model.NginxFactAccessLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(logs, 1000).Error
}

// ListFactAccessLogs 获取访问日志事实列表
func (r *NginxRepository) ListFactAccessLogs(sourceID uint, page, pageSize int, startTime, endTime *time.Time, filters map[string]interface{}) ([]model.NginxFactAccessLog, int64, error) {
	var logs []model.NginxFactAccessLog
	var total int64

	query := r.db.Model(&model.NginxFactAccessLog{}).Where("source_id = ?", sourceID)

	if startTime != nil {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", endTime)
	}

	if filters != nil {
		if status, ok := filters["status"]; ok && status != 0 {
			query = query.Where("status = ?", status)
		}
		if method, ok := filters["method"]; ok && method != "" {
			query = query.Where("method = ?", method)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("timestamp DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListFactAccessLogsWithDimensions 获取带维度信息的访问日志
func (r *NginxRepository) ListFactAccessLogsWithDimensions(sourceID uint, page, pageSize int, startTime, endTime *time.Time, filters map[string]interface{}) ([]model.AccessLogView, int64, error) {
	var views []model.AccessLogView
	var total int64

	// 检查新表是否存在
	if r.tableExists("nginx_fact_access_logs") {
		baseQuery := r.db.Table("nginx_fact_access_logs f").
			Select(`f.id, f.timestamp, f.method, f.protocol, f.status, f.body_bytes_sent, f.request_time,
				i.ip_address as remote_addr, i.country, i.city,
				u.url_path as uri, u.host,
				ref.referer_url as http_referer,
				ua.browser, ua.os, ua.device_type, ua.is_bot`).
			Joins("LEFT JOIN nginx_dim_ip i ON f.ip_id = i.id").
			Joins("LEFT JOIN nginx_dim_url u ON f.url_id = u.id").
			Joins("LEFT JOIN nginx_dim_referer ref ON f.referer_id = ref.id").
			Joins("LEFT JOIN nginx_dim_user_agent ua ON f.ua_id = ua.id").
			Where("f.source_id = ?", sourceID)

		if startTime != nil {
			baseQuery = baseQuery.Where("f.timestamp >= ?", startTime)
		}
		if endTime != nil {
			baseQuery = baseQuery.Where("f.timestamp <= ?", endTime)
		}

		if filters != nil {
			if ip, ok := filters["remoteAddr"]; ok && ip != "" {
				baseQuery = baseQuery.Where("i.ip_address LIKE ?", "%"+ip.(string)+"%")
			}
			if uri, ok := filters["uri"]; ok && uri != "" {
				baseQuery = baseQuery.Where("u.url_path LIKE ?", "%"+uri.(string)+"%")
			}
			if status, ok := filters["status"]; ok && status != 0 {
				baseQuery = baseQuery.Where("f.status = ?", status)
			}
			if method, ok := filters["method"]; ok && method != "" {
				baseQuery = baseQuery.Where("f.method = ?", method)
			}
			if host, ok := filters["host"]; ok && host != "" {
				baseQuery = baseQuery.Where("u.host LIKE ?", "%"+host.(string)+"%")
			}
		}

		countQuery := r.db.Table("(?) as sub", baseQuery).Count(&total)
		if countQuery.Error == nil && total > 0 {
			offset := (page - 1) * pageSize
			err := baseQuery.Offset(offset).Limit(pageSize).Order("f.timestamp DESC").Find(&views).Error
			if err == nil {
				return views, total, nil
			}
		}
	}

	// 回退到旧表
	var logs []model.NginxAccessLog
	query := r.db.Model(&model.NginxAccessLog{}).Where("source_id = ?", sourceID)

	if startTime != nil {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", endTime)
	}

	if filters != nil {
		if ip, ok := filters["remoteAddr"]; ok && ip != "" {
			query = query.Where("remote_addr LIKE ?", "%"+ip.(string)+"%")
		}
		if uri, ok := filters["uri"]; ok && uri != "" {
			query = query.Where("uri LIKE ?", "%"+uri.(string)+"%")
		}
		if status, ok := filters["status"]; ok && status != 0 {
			query = query.Where("status = ?", status)
		}
		if method, ok := filters["method"]; ok && method != "" {
			query = query.Where("method = ?", method)
		}
		if host, ok := filters["host"]; ok && host != "" {
			query = query.Where("host LIKE ?", "%"+host.(string)+"%")
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("timestamp DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	// 转换为 AccessLogView
	for _, log := range logs {
		views = append(views, model.AccessLogView{
			ID:            log.ID,
			Timestamp:     log.Timestamp,
			RemoteAddr:    log.RemoteAddr,
			Method:        log.Method,
			URI:           log.URI,
			Host:          log.Host,
			Protocol:      log.Protocol,
			Status:        log.Status,
			BodyBytesSent: log.BodyBytesSent,
			RequestTime:   log.RequestTime,
			HTTPReferer:   log.HTTPReferer,
		})
	}

	return views, total, nil
}

// DeleteOldFactAccessLogs 删除过期访问日志事实
func (r *NginxRepository) DeleteOldFactAccessLogs(sourceID uint, beforeTime time.Time) error {
	return r.db.Where("source_id = ? AND timestamp < ?", sourceID, beforeTime).Delete(&model.NginxFactAccessLog{}).Error
}

// ============== 旧版访问日志操作 (兼容) ==============

// CreateAccessLog 创建访问日志
func (r *NginxRepository) CreateAccessLog(log *model.NginxAccessLog) error {
	return r.db.Create(log).Error
}

// BatchCreateAccessLogs 批量创建访问日志
func (r *NginxRepository) BatchCreateAccessLogs(logs []model.NginxAccessLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(logs, 1000).Error
}

// ListAccessLogs 获取访问日志列表
func (r *NginxRepository) ListAccessLogs(sourceID uint, page, pageSize int, startTime, endTime *time.Time, filters map[string]interface{}) ([]model.NginxAccessLog, int64, error) {
	var logs []model.NginxAccessLog
	var total int64

	query := r.db.Model(&model.NginxAccessLog{}).Where("source_id = ?", sourceID)

	if startTime != nil {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", endTime)
	}

	if filters != nil {
		if ip, ok := filters["remoteAddr"]; ok && ip != "" {
			query = query.Where("remote_addr LIKE ?", "%"+ip.(string)+"%")
		}
		if uri, ok := filters["uri"]; ok && uri != "" {
			query = query.Where("uri LIKE ?", "%"+uri.(string)+"%")
		}
		if status, ok := filters["status"]; ok && status != 0 {
			query = query.Where("status = ?", status)
		}
		if method, ok := filters["method"]; ok && method != "" {
			query = query.Where("method = ?", method)
		}
		if host, ok := filters["host"]; ok && host != "" {
			query = query.Where("host LIKE ?", "%"+host.(string)+"%")
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("timestamp DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// DeleteOldAccessLogs 删除过期访问日志
func (r *NginxRepository) DeleteOldAccessLogs(sourceID uint, beforeTime time.Time) error {
	return r.db.Where("source_id = ? AND timestamp < ?", sourceID, beforeTime).Delete(&model.NginxAccessLog{}).Error
}

// ============== 新版聚合表操作 ==============

// CreateOrUpdateAggHourly 创建或更新小时聚合
func (r *NginxRepository) CreateOrUpdateAggHourly(agg *model.NginxAggHourly) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "hour"}},
		UpdateAll: true,
	}).Create(agg).Error
}

// GetAggHourly 获取小时聚合数据
func (r *NginxRepository) GetAggHourly(sourceID uint, hour time.Time) (*model.NginxAggHourly, error) {
	var agg model.NginxAggHourly
	err := r.db.Where("source_id = ? AND hour = ?", sourceID, hour).First(&agg).Error
	if err != nil {
		return nil, err
	}
	return &agg, nil
}

// ListAggHourly 获取小时聚合列表
func (r *NginxRepository) ListAggHourly(sourceID uint, startHour, endHour time.Time) ([]model.NginxAggHourly, error) {
	var aggs []model.NginxAggHourly
	err := r.db.Where("source_id = ? AND hour >= ? AND hour <= ?", sourceID, startHour, endHour).
		Order("hour ASC").Find(&aggs).Error
	return aggs, err
}

// ListAllAggHourly 获取所有数据源的小时聚合列表
func (r *NginxRepository) ListAllAggHourly(startHour, endHour time.Time) ([]model.NginxAggHourly, error) {
	var aggs []model.NginxAggHourly
	err := r.db.Where("hour >= ? AND hour <= ?", startHour, endHour).
		Order("hour ASC").Find(&aggs).Error
	return aggs, err
}

// DeleteOldAggHourly 删除过期小时聚合
func (r *NginxRepository) DeleteOldAggHourly(sourceID uint, beforeTime time.Time) error {
	return r.db.Where("source_id = ? AND hour < ?", sourceID, beforeTime).Delete(&model.NginxAggHourly{}).Error
}

// CreateOrUpdateAggDaily 创建或更新日聚合
func (r *NginxRepository) CreateOrUpdateAggDaily(agg *model.NginxAggDaily) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "date"}},
		UpdateAll: true,
	}).Create(agg).Error
}

// GetAggDaily 获取日聚合数据
func (r *NginxRepository) GetAggDaily(sourceID uint, date time.Time) (*model.NginxAggDaily, error) {
	var agg model.NginxAggDaily
	err := r.db.Where("source_id = ? AND date = ?", sourceID, date).First(&agg).Error
	if err != nil {
		return nil, err
	}
	return &agg, nil
}

// ListAggDaily 获取日聚合列表
func (r *NginxRepository) ListAggDaily(sourceID uint, startDate, endDate time.Time) ([]model.NginxAggDaily, error) {
	var aggs []model.NginxAggDaily
	err := r.db.Where("source_id = ? AND date >= ? AND date <= ?", sourceID, startDate, endDate).
		Order("date DESC").Find(&aggs).Error
	return aggs, err
}

// ListAllAggDaily 获取所有数据源的日聚合列表
func (r *NginxRepository) ListAllAggDaily(startDate, endDate time.Time) ([]model.NginxAggDaily, error) {
	var aggs []model.NginxAggDaily
	err := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date DESC").Find(&aggs).Error
	return aggs, err
}

// ============== 旧版统计操作 (兼容) ==============

// CreateOrUpdateDailyStats 创建或更新日统计
func (r *NginxRepository) CreateOrUpdateDailyStats(stats *model.NginxDailyStats) error {
	return r.db.Save(stats).Error
}

// GetDailyStats 获取日统计数据
func (r *NginxRepository) GetDailyStats(sourceID uint, date time.Time) (*model.NginxDailyStats, error) {
	var stats model.NginxDailyStats
	err := r.db.Where("source_id = ? AND date = ?", sourceID, date).First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// ListDailyStats 获取日统计列表
func (r *NginxRepository) ListDailyStats(sourceID uint, startDate, endDate time.Time) ([]model.NginxDailyStats, error) {
	var stats []model.NginxDailyStats
	err := r.db.Where("source_id = ? AND date >= ? AND date <= ?", sourceID, startDate, endDate).
		Order("date DESC").Find(&stats).Error
	return stats, err
}

// GetDailyStatsRange 获取日期范围内的统计数据
func (r *NginxRepository) GetDailyStatsRange(startDate, endDate time.Time) ([]model.NginxDailyStats, error) {
	var stats []model.NginxDailyStats
	err := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Where("source_id IN (SELECT id FROM nginx_sources WHERE deleted_at IS NULL)").
		Order("date DESC").Find(&stats).Error
	return stats, err
}

// CreateOrUpdateHourlyStats 创建或更新小时统计
func (r *NginxRepository) CreateOrUpdateHourlyStats(stats *model.NginxHourlyStats) error {
	return r.db.Save(stats).Error
}

// GetHourlyStats 获取小时统计数据
func (r *NginxRepository) GetHourlyStats(sourceID uint, hour time.Time) (*model.NginxHourlyStats, error) {
	var stats model.NginxHourlyStats
	err := r.db.Where("source_id = ? AND hour = ?", sourceID, hour).First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// ListHourlyStats 获取小时统计列表
func (r *NginxRepository) ListHourlyStats(sourceID uint, startHour, endHour time.Time) ([]model.NginxHourlyStats, error) {
	var stats []model.NginxHourlyStats
	err := r.db.Where("source_id = ? AND hour >= ? AND hour <= ?", sourceID, startHour, endHour).
		Order("hour DESC").Find(&stats).Error
	return stats, err
}

// DeleteOldHourlyStats 删除过期小时统计
func (r *NginxRepository) DeleteOldHourlyStats(sourceID uint, beforeTime time.Time) error {
	return r.db.Where("source_id = ? AND hour < ?", sourceID, beforeTime).Delete(&model.NginxHourlyStats{}).Error
}

// ============== 统计查询 ==============

// GetTodayOverview 获取今日概况
func (r *NginxRepository) GetTodayOverview() (*model.OverviewStats, error) {
	overview := &model.OverviewStats{
		StatusDistribution: make(map[string]int64),
	}

	// 获取数据源统计
	r.db.Model(&model.NginxSource{}).Count(&overview.TotalSources)
	r.db.Model(&model.NginxSource{}).Where("status = ?", 1).Count(&overview.ActiveSources)

	// 获取今日聚合数据 (优先使用新表)
	today := time.Now().Truncate(24 * time.Hour)
	usedNewTable := false

	if r.tableExists("nginx_agg_daily") {
		var aggDaily []model.NginxAggDaily
		err := r.db.Where("date = ?", today).Find(&aggDaily).Error
		if err == nil && len(aggDaily) > 0 {
			usedNewTable = true
			var totalResponseTime float64
			var responseTimeCount int64
			for _, agg := range aggDaily {
				overview.TodayRequests += agg.TotalRequests
				overview.TodayVisitors += agg.UniqueIPs
				overview.TodayBandwidth += agg.TotalBandwidth
				overview.TodayPV += agg.PVCount
				overview.StatusDistribution["2xx"] += agg.Status2xx
				overview.StatusDistribution["3xx"] += agg.Status3xx
				overview.StatusDistribution["4xx"] += agg.Status4xx
				overview.StatusDistribution["5xx"] += agg.Status5xx
				if agg.AvgResponseTime > 0 {
					totalResponseTime += agg.AvgResponseTime * float64(agg.TotalRequests)
					responseTimeCount += agg.TotalRequests
				}
			}
			if responseTimeCount > 0 {
				overview.AvgResponseTime = totalResponseTime / float64(responseTimeCount)
			}
		}
	}

	if !usedNewTable {
		// 回退到旧表
		var dailyStats []model.NginxDailyStats
		r.db.Where("date = ?", today).Find(&dailyStats)
		for _, stats := range dailyStats {
			overview.TodayRequests += stats.TotalRequests
			overview.TodayVisitors += stats.UniqueVisitors
			overview.TodayBandwidth += stats.TotalBandwidth
			overview.StatusDistribution["2xx"] += stats.Status2xx
			overview.StatusDistribution["3xx"] += stats.Status3xx
			overview.StatusDistribution["4xx"] += stats.Status4xx
			overview.StatusDistribution["5xx"] += stats.Status5xx
		}
	}

	// 计算错误率
	totalStatus := overview.StatusDistribution["2xx"] + overview.StatusDistribution["3xx"] +
		overview.StatusDistribution["4xx"] + overview.StatusDistribution["5xx"]
	if totalStatus > 0 {
		errorCount := overview.StatusDistribution["4xx"] + overview.StatusDistribution["5xx"]
		overview.TodayErrorRate = float64(errorCount) / float64(totalStatus) * 100
	}

	return overview, nil
}

// GetRequestsTrend 获取请求趋势
func (r *NginxRepository) GetRequestsTrend(sourceID *uint, hours int) ([]model.TrendPoint, error) {
	var trend []model.TrendPoint

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	// 优先使用新表
	if r.tableExists("nginx_agg_hourly") {
		query := r.db.Model(&model.NginxAggHourly{}).
			Select("hour as time, SUM(total_requests) as value").
			Where("hour >= ? AND hour <= ?", startTime, endTime).
			Group("hour").
			Order("hour ASC")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		type Result struct {
			Time  time.Time
			Value int64
		}
		var results []Result
		err := query.Find(&results).Error
		if err == nil && len(results) > 0 {
			for _, r := range results {
				trend = append(trend, model.TrendPoint{
					Time:  r.Time.Format("15:04"),
					Value: r.Value,
				})
			}
			return trend, nil
		}
	}

	// 回退到旧表
	query := r.db.Model(&model.NginxHourlyStats{}).
		Select("hour as time, SUM(total_requests) as value").
		Where("hour >= ? AND hour <= ?", startTime, endTime).
		Group("hour").
		Order("hour ASC")

	if sourceID != nil {
		query = query.Where("source_id = ?", *sourceID)
	}

	type Result struct {
		Time  time.Time
		Value int64
	}
	var results []Result
	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	for _, r := range results {
		trend = append(trend, model.TrendPoint{
			Time:  r.Time.Format("15:04"),
			Value: r.Value,
		})
	}

	return trend, nil
}

// GetTopURIs 获取 Top URI
func (r *NginxRepository) GetTopURIs(sourceID uint, startTime, endTime time.Time, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	type URICount struct {
		URI   string
		Count int64
	}
	var uriCounts []URICount

	err := r.db.Model(&model.NginxAccessLog{}).
		Select("uri, COUNT(*) as count").
		Where("source_id = ? AND timestamp >= ? AND timestamp <= ?", sourceID, startTime, endTime).
		Group("uri").
		Order("count DESC").
		Limit(limit).
		Find(&uriCounts).Error

	if err != nil {
		return nil, err
	}

	for _, uc := range uriCounts {
		results = append(results, map[string]interface{}{
			"uri":   uc.URI,
			"count": uc.Count,
		})
	}

	return results, nil
}

// GetTopIPs 获取 Top IP
func (r *NginxRepository) GetTopIPs(sourceID uint, startTime, endTime time.Time, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	type IPCount struct {
		RemoteAddr string
		Count      int64
	}
	var ipCounts []IPCount

	err := r.db.Model(&model.NginxAccessLog{}).
		Select("remote_addr, COUNT(*) as count").
		Where("source_id = ? AND timestamp >= ? AND timestamp <= ?", sourceID, startTime, endTime).
		Group("remote_addr").
		Order("count DESC").
		Limit(limit).
		Find(&ipCounts).Error

	if err != nil {
		return nil, err
	}

	for _, ic := range ipCounts {
		results = append(results, map[string]interface{}{
			"ip":    ic.RemoteAddr,
			"count": ic.Count,
		})
	}

	return results, nil
}

// GetTopIPsWithGeo 获取带地理信息的 Top IP
func (r *NginxRepository) GetTopIPsWithGeo(sourceID uint, startTime, endTime time.Time, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 检查新表是否存在
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_ip") {
		type IPGeoCount struct {
			IPAddress string
			Country   string
			Province  string
			City      string
			Count     int64
		}
		var ipCounts []IPGeoCount

		err := r.db.Table("nginx_fact_access_logs f").
			Select("i.ip_address, i.country, i.province, i.city, COUNT(*) as count").
			Joins("LEFT JOIN nginx_dim_ip i ON f.ip_id = i.id").
			Where("f.source_id = ? AND f.timestamp >= ? AND f.timestamp <= ?", sourceID, startTime, endTime).
			Group("f.ip_id").
			Order("count DESC").
			Limit(limit).
			Find(&ipCounts).Error

		if err == nil && len(ipCounts) > 0 {
			for _, ic := range ipCounts {
				results = append(results, map[string]interface{}{
					"ip":       ic.IPAddress,
					"country":  ic.Country,
					"province": ic.Province,
					"city":     ic.City,
					"count":    ic.Count,
				})
			}
			return results, nil
		}
	}

	// 回退到旧表 (没有地理信息)
	type IPCount struct {
		RemoteAddr string
		Count      int64
	}
	var ipCounts []IPCount

	err := r.db.Model(&model.NginxAccessLog{}).
		Select("remote_addr, COUNT(*) as count").
		Where("source_id = ? AND timestamp >= ? AND timestamp <= ?", sourceID, startTime, endTime).
		Group("remote_addr").
		Order("count DESC").
		Limit(limit).
		Find(&ipCounts).Error

	if err != nil {
		return nil, err
	}

	for _, ic := range ipCounts {
		results = append(results, map[string]interface{}{
			"ip":       ic.RemoteAddr,
			"country":  "",
			"province": "",
			"city":     "",
			"count":    ic.Count,
		})
	}

	return results, nil
}

// GetGeoDistribution 获取地理分布统计
func (r *NginxRepository) GetGeoDistribution(sourceID *uint, startTime, endTime time.Time, level string) ([]model.GeoStats, error) {
	var stats []model.GeoStats

	// 检查新表是否存在
	if !r.tableExists("nginx_fact_access_logs") || !r.tableExists("nginx_dim_ip") {
		// 新表不存在，返回空数据 (旧表没有地理信息)
		return stats, nil
	}

	var selectField string
	switch level {
	case "country":
		selectField = "i.country"
	case "province":
		selectField = "i.country, i.province"
	default:
		selectField = "i.country, i.province, i.city"
	}

	query := r.db.Table("nginx_fact_access_logs f").
		Select(selectField+", COUNT(*) as count").
		Joins("LEFT JOIN nginx_dim_ip i ON f.ip_id = i.id").
		Where("f.timestamp >= ? AND f.timestamp <= ?", startTime, endTime)

	if sourceID != nil {
		query = query.Where("f.source_id = ?", *sourceID)
	}

	query = query.Group(selectField).Order("count DESC")

	type GeoResult struct {
		Country  string
		Province string
		City     string
		Count    int64
	}
	var results []GeoResult
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	var total int64
	for _, r := range results {
		total += r.Count
	}

	for _, r := range results {
		percent := float64(0)
		if total > 0 {
			percent = float64(r.Count) / float64(total) * 100
		}
		stats = append(stats, model.GeoStats{
			Country:  r.Country,
			Province: r.Province,
			City:     r.City,
			Count:    r.Count,
			Percent:  percent,
		})
	}

	return stats, nil
}

// GetBrowserDistribution 获取浏览器分布统计
func (r *NginxRepository) GetBrowserDistribution(sourceID *uint, startTime, endTime time.Time) ([]model.BrowserStats, error) {
	var stats []model.BrowserStats

	// 检查新表是否存在
	if !r.tableExists("nginx_fact_access_logs") || !r.tableExists("nginx_dim_user_agent") {
		// 新表不存在，返回空数据 (旧表没有浏览器解析信息)
		return stats, nil
	}

	query := r.db.Table("nginx_fact_access_logs f").
		Select("ua.browser, COUNT(*) as count").
		Joins("LEFT JOIN nginx_dim_user_agent ua ON f.ua_id = ua.id").
		Where("f.timestamp >= ? AND f.timestamp <= ?", startTime, endTime).
		Where("ua.is_bot = ?", false)

	if sourceID != nil {
		query = query.Where("f.source_id = ?", *sourceID)
	}

	query = query.Group("ua.browser").Order("count DESC")

	type Result struct {
		Browser string
		Count   int64
	}
	var results []Result
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	var total int64
	for _, r := range results {
		total += r.Count
	}

	for _, r := range results {
		percent := float64(0)
		if total > 0 {
			percent = float64(r.Count) / float64(total) * 100
		}
		stats = append(stats, model.BrowserStats{
			Browser: r.Browser,
			Count:   r.Count,
			Percent: percent,
		})
	}

	return stats, nil
}

// GetDeviceDistribution 获取设备分布统计
func (r *NginxRepository) GetDeviceDistribution(sourceID *uint, startTime, endTime time.Time) ([]model.DeviceStats, error) {
	var stats []model.DeviceStats

	// 检查新表是否存在
	if !r.tableExists("nginx_fact_access_logs") || !r.tableExists("nginx_dim_user_agent") {
		// 新表不存在，返回空数据 (旧表没有设备解析信息)
		return stats, nil
	}

	query := r.db.Table("nginx_fact_access_logs f").
		Select("ua.device_type, COUNT(*) as count").
		Joins("LEFT JOIN nginx_dim_user_agent ua ON f.ua_id = ua.id").
		Where("f.timestamp >= ? AND f.timestamp <= ?", startTime, endTime).
		Where("ua.is_bot = ?", false)

	if sourceID != nil {
		query = query.Where("f.source_id = ?", *sourceID)
	}

	query = query.Group("ua.device_type").Order("count DESC")

	type Result struct {
		DeviceType string
		Count      int64
	}
	var results []Result
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	var total int64
	for _, r := range results {
		total += r.Count
	}

	for _, r := range results {
		percent := float64(0)
		if total > 0 {
			percent = float64(r.Count) / float64(total) * 100
		}
		stats = append(stats, model.DeviceStats{
			DeviceType: r.DeviceType,
			Count:      r.Count,
			Percent:    percent,
		})
	}

	return stats, nil
}

// GetTimeSeries 获取时间序列数据
func (r *NginxRepository) GetTimeSeries(sourceID *uint, startTime, endTime time.Time, interval string) ([]model.TimeSeriesPoint, error) {
	var points []model.TimeSeriesPoint

	// 根据间隔选择聚合表
	if interval == "hour" {
		// 检查新表是否存在
		if r.tableExists("nginx_agg_hourly") {
			query := r.db.Model(&model.NginxAggHourly{}).
				Select(`hour as time,
					SUM(total_requests) as requests,
					SUM(total_bandwidth) as bandwidth,
					SUM(unique_ips) as unique_ips,
					AVG(avg_response_time) as avg_response_time,
					SUM(status_4xx) + SUM(status_5xx) as errors,
					SUM(total_requests) as total`).
				Where("hour >= ? AND hour <= ?", startTime, endTime).
				Group("hour").
				Order("hour ASC")

			if sourceID != nil {
				query = query.Where("source_id = ?", *sourceID)
			}

			type Result struct {
				Time            time.Time
				Requests        int64
				Bandwidth       int64
				UniqueIPs       int64
				AvgResponseTime float64
				Errors          int64
				Total           int64
			}
			var results []Result
			if err := query.Find(&results).Error; err == nil && len(results) > 0 {
				for _, r := range results {
					errorRate := float64(0)
					if r.Total > 0 {
						errorRate = float64(r.Errors) / float64(r.Total) * 100
					}
					points = append(points, model.TimeSeriesPoint{
						Time:            r.Time.Format("2006-01-02 15:04"),
						Requests:        r.Requests,
						Bandwidth:       r.Bandwidth,
						UniqueIPs:       r.UniqueIPs,
						AvgResponseTime: r.AvgResponseTime,
						ErrorRate:       errorRate,
					})
				}
				return points, nil
			}
		}

		// 回退到旧表
		query := r.db.Model(&model.NginxHourlyStats{}).
			Select(`hour as time,
				SUM(total_requests) as requests,
				SUM(total_bandwidth) as bandwidth,
				SUM(unique_visitors) as unique_ips,
				AVG(avg_response_time) as avg_response_time,
				SUM(status_4xx) + SUM(status_5xx) as errors,
				SUM(total_requests) as total`).
			Where("hour >= ? AND hour <= ?", startTime, endTime).
			Group("hour").
			Order("hour ASC")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		type Result struct {
			Time            time.Time
			Requests        int64
			Bandwidth       int64
			UniqueIPs       int64
			AvgResponseTime float64
			Errors          int64
			Total           int64
		}
		var results []Result
		if err := query.Find(&results).Error; err != nil {
			return nil, err
		}

		for _, r := range results {
			errorRate := float64(0)
			if r.Total > 0 {
				errorRate = float64(r.Errors) / float64(r.Total) * 100
			}
			points = append(points, model.TimeSeriesPoint{
				Time:            r.Time.Format("2006-01-02 15:04"),
				Requests:        r.Requests,
				Bandwidth:       r.Bandwidth,
				UniqueIPs:       r.UniqueIPs,
				AvgResponseTime: r.AvgResponseTime,
				ErrorRate:       errorRate,
			})
		}
	} else {
		// 日级别聚合
		// 检查新表是否存在
		if r.tableExists("nginx_agg_daily") {
			query := r.db.Model(&model.NginxAggDaily{}).
				Select(`date as time,
					SUM(total_requests) as requests,
					SUM(total_bandwidth) as bandwidth,
					SUM(unique_ips) as unique_ips,
					AVG(avg_response_time) as avg_response_time,
					SUM(status_4xx) + SUM(status_5xx) as errors,
					SUM(total_requests) as total`).
				Where("date >= ? AND date <= ?", startTime, endTime).
				Group("date").
				Order("date ASC")

			if sourceID != nil {
				query = query.Where("source_id = ?", *sourceID)
			}

			type Result struct {
				Time            time.Time
				Requests        int64
				Bandwidth       int64
				UniqueIPs       int64
				AvgResponseTime float64
				Errors          int64
				Total           int64
			}
			var results []Result
			if err := query.Find(&results).Error; err == nil && len(results) > 0 {
				for _, r := range results {
					errorRate := float64(0)
					if r.Total > 0 {
						errorRate = float64(r.Errors) / float64(r.Total) * 100
					}
					points = append(points, model.TimeSeriesPoint{
						Time:            r.Time.Format("2006-01-02"),
						Requests:        r.Requests,
						Bandwidth:       r.Bandwidth,
						UniqueIPs:       r.UniqueIPs,
						AvgResponseTime: r.AvgResponseTime,
						ErrorRate:       errorRate,
					})
				}
				return points, nil
			}
		}

		// 回退到旧表
		query := r.db.Model(&model.NginxDailyStats{}).
			Select(`date as time,
				SUM(total_requests) as requests,
				SUM(total_bandwidth) as bandwidth,
				SUM(unique_visitors) as unique_ips,
				AVG(avg_response_time) as avg_response_time,
				SUM(status_4xx) + SUM(status_5xx) as errors,
				SUM(total_requests) as total`).
			Where("date >= ? AND date <= ?", startTime, endTime).
			Group("date").
			Order("date ASC")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		type Result struct {
			Time            time.Time
			Requests        int64
			Bandwidth       int64
			UniqueIPs       int64
			AvgResponseTime float64
			Errors          int64
			Total           int64
		}
		var results []Result
		if err := query.Find(&results).Error; err != nil {
			return nil, err
		}

		for _, r := range results {
			errorRate := float64(0)
			if r.Total > 0 {
				errorRate = float64(r.Errors) / float64(r.Total) * 100
			}
			points = append(points, model.TimeSeriesPoint{
				Time:            r.Time.Format("2006-01-02"),
				Requests:        r.Requests,
				Bandwidth:       r.Bandwidth,
				UniqueIPs:       r.UniqueIPs,
				AvgResponseTime: r.AvgResponseTime,
				ErrorRate:       errorRate,
			})
		}
	}

	return points, nil
}
