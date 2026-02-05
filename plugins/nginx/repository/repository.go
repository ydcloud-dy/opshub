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
	"fmt"
	"math"
	"net/url"
	"strings"
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

	// 确保使用本地时区
	startHour = startHour.Local()
	endHour = endHour.Local()

	fmt.Printf("[DEBUG] ListHourlyStats: sourceID=%d, startHour=%s, endHour=%s\n",
		sourceID, startHour.Format("2006-01-02 15:04:05"), endHour.Format("2006-01-02 15:04:05"))

	// 首先尝试查询指定时间范围
	if r.tableExists("nginx_hourly_stats") {
		err := r.db.Where("source_id = ? AND hour >= ? AND hour <= ?", sourceID, startHour, endHour).
			Order("hour DESC").Find(&stats).Error
		fmt.Printf("[DEBUG] Query result: error=%v, count=%d\n", err, len(stats))
		if err == nil && len(stats) > 0 {
			return stats, nil
		}

		// 如果指定范围没数据，尝试查询最近7天的所有数据
		sevenDaysAgo := time.Now().Local().AddDate(0, 0, -7)
		fmt.Printf("[DEBUG] No data in range, trying fallback: 7 days ago = %s\n",
			sevenDaysAgo.Format("2006-01-02 15:04:05"))
		err = r.db.Where("source_id = ? AND hour >= ?", sourceID, sevenDaysAgo).
			Order("hour DESC").Limit(24).Find(&stats).Error
		fmt.Printf("[DEBUG] Fallback query result: error=%v, count=%d\n", err, len(stats))
		return stats, err
	}

	fmt.Printf("[DEBUG] Table nginx_hourly_stats does not exist\n")
	return stats, nil
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
	// 使用本地时区的今天日期字符串，避免时区问题
	todayStr := time.Now().Local().Format("2006-01-02")
	usedNewTable := false
	foundData := false

	if r.tableExists("nginx_agg_daily") {
		var aggDaily []model.NginxAggDaily
		err := r.db.Where("DATE(date) = ?", todayStr).Find(&aggDaily).Error
		if err == nil && len(aggDaily) > 0 {
			usedNewTable = true
			foundData = true
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

	if !usedNewTable && r.tableExists("nginx_daily_stats") {
		// 回退到旧表
		var dailyStats []model.NginxDailyStats
		r.db.Where("DATE(date) = ?", todayStr).Find(&dailyStats)
		if len(dailyStats) > 0 {
			foundData = true
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
	}

	// 如果今天没有数据，尝试获取最近7天的汇总数据
	if !foundData {
		sevenDaysAgo := time.Now().Local().AddDate(0, 0, -7).Format("2006-01-02")

		if r.tableExists("nginx_daily_stats") {
			var dailyStats []model.NginxDailyStats
			r.db.Where("DATE(date) >= ?", sevenDaysAgo).Find(&dailyStats)
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

	endTime := time.Now().Local()
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
	if r.tableExists("nginx_hourly_stats") {
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
		if err := query.Find(&results).Error; err == nil && len(results) > 0 {
			for _, r := range results {
				trend = append(trend, model.TrendPoint{
					Time:  r.Time.Format("15:04"),
					Value: r.Value,
				})
			}
			return trend, nil
		}

		// 如果最近N小时没数据，尝试获取最近7天的所有小时数据
		sevenDaysAgo := time.Now().Local().AddDate(0, 0, -7)
		query2 := r.db.Model(&model.NginxHourlyStats{}).
			Select("hour as time, SUM(total_requests) as value").
			Where("hour >= ?", sevenDaysAgo).
			Group("hour").
			Order("hour ASC").
			Limit(24) // 只取最近24个有数据的小时

		if sourceID != nil {
			query2 = query2.Where("source_id = ?", *sourceID)
		}

		var results2 []Result
		if err := query2.Find(&results2).Error; err == nil {
			for _, r := range results2 {
				trend = append(trend, model.TrendPoint{
					Time:  r.Time.Format("01-02 15:04"),
					Value: r.Value,
				})
			}
		}
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

	// 回退到旧表 nginx_access_logs (现在有地理信息字段)
	type IPGeoCount struct {
		RemoteAddr string
		Country    string
		Province   string
		City       string
		Count      int64
	}
	var ipCounts []IPGeoCount

	// 使用子查询获取每个IP的地理信息（优先取有省份和城市的记录）
	err := r.db.Raw(`
		SELECT remote_addr,
		       COALESCE(MAX(CASE WHEN country != '' THEN country END), '') as country,
		       COALESCE(MAX(CASE WHEN province != '' THEN province END), '') as province,
		       COALESCE(MAX(CASE WHEN city != '' THEN city END), '') as city,
		       COUNT(*) as count
		FROM nginx_access_logs
		WHERE source_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY remote_addr
		ORDER BY count DESC
		LIMIT ?
	`, sourceID, startTime, endTime, limit).Scan(&ipCounts).Error

	if err != nil {
		return nil, err
	}

	for _, ic := range ipCounts {
		country := ic.Country
		province := ic.Province
		city := ic.City

		// 如果没有地理信息，显示为 "-"
		if country == "" {
			country = "-"
		}
		if province == "" {
			province = "-"
		}
		if city == "" {
			city = "-"
		}

		results = append(results, map[string]interface{}{
			"ip":       ic.RemoteAddr,
			"country":  country,
			"province": province,
			"city":     city,
			"count":    ic.Count,
		})
	}

	return results, nil
}

// GetGeoDistribution 获取地理分布统计
func (r *NginxRepository) GetGeoDistribution(sourceID *uint, startTime, endTime time.Time, level string) ([]model.GeoStats, error) {
	var stats []model.GeoStats

	// 优先使用新表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_ip") {
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
			Select(selectField+", COUNT(DISTINCT f.ip_id) as count").
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
		if err := query.Find(&results).Error; err == nil && len(results) > 0 {
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
	}

	// 回退到旧表 nginx_access_logs (现在有地理信息字段)
	if r.tableExists("nginx_access_logs") {
		var selectField string
		var groupField string
		switch level {
		case "country":
			selectField = "country"
			groupField = "country"
		case "province":
			selectField = "country, province"
			groupField = "country, province"
		default:
			selectField = "country, province, city"
			groupField = "country, province, city"
		}

		query := r.db.Model(&model.NginxAccessLog{}).
			Select(selectField+", COUNT(DISTINCT remote_addr) as count").
			Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
			Where("country != '' AND country IS NOT NULL")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		query = query.Group(groupField).Order("count DESC").Limit(50)

		type GeoResult struct {
			Country  string
			Province string
			City     string
			Count    int64
		}
		var results []GeoResult
		if err := query.Find(&results).Error; err == nil {
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
		}
	}

	return stats, nil
}

// GetBrowserDistribution 获取浏览器分布统计
func (r *NginxRepository) GetBrowserDistribution(sourceID *uint, startTime, endTime time.Time) ([]model.BrowserStats, error) {
	var stats []model.BrowserStats

	// 优先使用新表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_user_agent") {
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
		if err := query.Find(&results).Error; err == nil && len(results) > 0 {
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
	}

	// 回退到旧表 nginx_access_logs (现在有 browser 字段)
	if r.tableExists("nginx_access_logs") {
		query := r.db.Model(&model.NginxAccessLog{}).
			Select("browser, COUNT(*) as count").
			Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
			Where("browser != '' AND browser IS NOT NULL")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		query = query.Group("browser").Order("count DESC").Limit(20)

		type Result struct {
			Browser string
			Count   int64
		}
		var results []Result
		if err := query.Find(&results).Error; err == nil {
			var total int64
			for _, r := range results {
				total += r.Count
			}

			for _, r := range results {
				percent := float64(0)
				if total > 0 {
					percent = float64(r.Count) / float64(total) * 100
				}
				browser := r.Browser
				if browser == "" {
					browser = "Other"
				}
				stats = append(stats, model.BrowserStats{
					Browser: browser,
					Count:   r.Count,
					Percent: percent,
				})
			}
		}
	}

	return stats, nil
}

// GetDeviceDistribution 获取设备分布统计
func (r *NginxRepository) GetDeviceDistribution(sourceID *uint, startTime, endTime time.Time) ([]model.DeviceStats, error) {
	var stats []model.DeviceStats

	// 优先使用新表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_user_agent") {
		query := r.db.Table("nginx_fact_access_logs f").
			Select("ua.device_type, COUNT(DISTINCT f.ip_id) as count").
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
		if err := query.Find(&results).Error; err == nil && len(results) > 0 {
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
	}

	// 回退到旧表 nginx_access_logs (现在有 device_type 字段)
	if r.tableExists("nginx_access_logs") {
		query := r.db.Model(&model.NginxAccessLog{}).
			Select("device_type, COUNT(DISTINCT remote_addr) as count").
			Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
			Where("device_type != '' AND device_type IS NOT NULL")

		if sourceID != nil {
			query = query.Where("source_id = ?", *sourceID)
		}

		query = query.Group("device_type").Order("count DESC")

		type Result struct {
			DeviceType string
			Count      int64
		}
		var results []Result
		if err := query.Find(&results).Error; err == nil {
			var total int64
			for _, r := range results {
				total += r.Count
			}

			// 设备类型映射为中文
			deviceNameMap := map[string]string{
				"desktop": "桌面设备",
				"mobile":  "移动设备",
				"tablet":  "平板设备",
				"bot":     "爬虫/机器人",
				"unknown": "未知",
			}

			for _, r := range results {
				percent := float64(0)
				if total > 0 {
					percent = float64(r.Count) / float64(total) * 100
				}
				deviceType := r.DeviceType
				if name, ok := deviceNameMap[deviceType]; ok {
					deviceType = name
				}
				stats = append(stats, model.DeviceStats{
					DeviceType: deviceType,
					Count:      r.Count,
					Percent:    percent,
				})
			}
		}
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
		if r.tableExists("nginx_hourly_stats") {
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
			if err := query.Find(&results).Error; err == nil {
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
			}
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
		if r.tableExists("nginx_daily_stats") {
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
			if err := query.Find(&results).Error; err == nil {
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
		}
	}

	return points, nil
}

// ============== 概况页面查询方法 ==============

// GetActiveVisitors 获取活跃访客数（最近N分钟内的独立IP数）
func (r *NginxRepository) GetActiveVisitors(sourceID uint, minutes int) (int64, error) {
	since := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var count int64

	fmt.Printf("[DEBUG] GetActiveVisitors: sourceID=%d, minutes=%d, since=%s\n", sourceID, minutes, since.Format("2006-01-02 15:04:05"))

	// 优先使用新表，但如果结果为0则继续尝试旧表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_ip") {
		err := r.db.Table("nginx_fact_access_logs").
			Where("source_id = ? AND timestamp >= ?", sourceID, since).
			Distinct("ip_id").
			Count(&count).Error
		if err == nil && count > 0 {
			fmt.Printf("[DEBUG] GetActiveVisitors: from new table, count=%d\n", count)
			return count, nil
		}
		if err != nil {
			fmt.Printf("[DEBUG] GetActiveVisitors: new table query error: %v\n", err)
		} else {
			fmt.Printf("[DEBUG] GetActiveVisitors: new table returned 0, trying old table\n")
		}
	}

	// 回退到旧表
	if r.tableExists("nginx_access_logs") {
		err := r.db.Model(&model.NginxAccessLog{}).
			Where("source_id = ? AND timestamp >= ?", sourceID, since).
			Distinct("remote_addr").
			Count(&count).Error
		if err == nil {
			fmt.Printf("[DEBUG] GetActiveVisitors: from old table, count=%d\n", count)
			return count, nil
		}
		fmt.Printf("[DEBUG] GetActiveVisitors: old table query error: %v\n", err)
	}

	fmt.Printf("[DEBUG] GetActiveVisitors: no data found, returning 0\n")
	return 0, nil
}

// GetCoreMetrics 获取核心指标（今日/昨日/预计今日/昨日此时）
func (r *NginxRepository) GetCoreMetrics(sourceID uint) (*model.CoreMetrics, error) {
	now := time.Now().Local()
	todayStr := now.Format("2006-01-02")
	yesterdayStr := now.AddDate(0, 0, -1).Format("2006-01-02")
	currentHour := now.Hour()

	metrics := &model.CoreMetrics{}

	// 辅助函数：从日聚合表获取基础 MetricSet
	getFromAggDaily := func(dateStr string) model.MetricSet {
		var ms model.MetricSet
		if r.tableExists("nginx_agg_daily") {
			type DailyResult struct {
				TotalRequests int64
				PVCount       int64
				UniqueIPs     int64
				Status2xx     int64
				Status3xx     int64
				Status4xx     int64
				Status5xx     int64
			}
			var dr DailyResult
			err := r.db.Model(&model.NginxAggDaily{}).
				Select("total_requests, pv_count, unique_ips, status_2xx, status_3xx, status_4xx, status_5xx").
				Where("source_id = ? AND DATE(date) = ?", sourceID, dateStr).
				First(&dr).Error
			if err == nil {
				ms.StatusHits = dr.TotalRequests
				ms.PV = dr.PVCount
				ms.UV = dr.UniqueIPs
				ms.Status2xx = dr.Status2xx
				ms.Status3xx = dr.Status3xx
				ms.Status4xx = dr.Status4xx
				ms.Status5xx = dr.Status5xx
				return ms
			}
		}
		// 回退到旧表
		if r.tableExists("nginx_daily_stats") {
			var ds model.NginxDailyStats
			err := r.db.Where("source_id = ? AND DATE(date) = ?", sourceID, dateStr).First(&ds).Error
			if err == nil {
				ms.StatusHits = ds.TotalRequests
				ms.PV = 0
				ms.UV = ds.UniqueVisitors
				ms.Status2xx = ds.Status2xx
				ms.Status3xx = ds.Status3xx
				ms.Status4xx = ds.Status4xx
				ms.Status5xx = ds.Status5xx
			}
		}
		return ms
	}

	// 今日、昨日基础数据
	metrics.Today = getFromAggDaily(todayStr)
	metrics.Yesterday = getFromAggDaily(yesterdayStr)

	// 计算实时OPS（最近1分钟的请求率）
	oneMinuteAgo := now.Add(-1 * time.Minute)
	var recentCount int64

	// 优先使用新表，但如果结果为0则继续尝试旧表
	if r.tableExists("nginx_fact_access_logs") {
		r.db.Table("nginx_fact_access_logs").
			Where("source_id = ? AND timestamp >= ?", sourceID, oneMinuteAgo).
			Count(&recentCount)
	}
	// 如果新表没有数据或不存在，尝试旧表
	if recentCount == 0 && r.tableExists("nginx_access_logs") {
		r.db.Model(&model.NginxAccessLog{}).
			Where("source_id = ? AND timestamp >= ?", sourceID, oneMinuteAgo).
			Count(&recentCount)
	}
	metrics.Today.RealtimeOps = float64(recentCount) / 60.0 // 每秒请求数

	// 计算峰值OPS（从小时聚合数据中获取最大值）
	if r.tableExists("nginx_agg_hourly") {
		todayStart, _ := time.ParseInLocation("2006-01-02", todayStr, time.Local)
		type HourlyMax struct {
			MaxRequests int64
		}
		var hm HourlyMax
		err := r.db.Model(&model.NginxAggHourly{}).
			Select("MAX(total_requests) as max_requests").
			Where("source_id = ? AND hour >= ?", sourceID, todayStart).
			First(&hm).Error
		if err == nil && hm.MaxRequests > 0 {
			metrics.Today.PeakOps = float64(hm.MaxRequests) / 3600.0 // 小时请求率转为每秒
		}
	}

	// 实时补充今日 PV 和 UV（从原始日志表精确计算）
	todayStart, _ := time.ParseInLocation("2006-01-02", todayStr, time.Local)
	tomorrowStart := todayStart.Add(24 * time.Hour)

	// 优先从旧日志表计算（字段直接可用）
	if r.tableExists("nginx_access_logs") {
		// 计算今日 PV（总请求数）
		var pvCount int64
		r.db.Model(&model.NginxAccessLog{}).
			Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, todayStart, tomorrowStart).
			Count(&pvCount)
		if pvCount > 0 {
			metrics.Today.PV = pvCount
		}

		// 计算今日 UV（独立IP）
		var uvCount int64
		r.db.Model(&model.NginxAccessLog{}).
			Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, todayStart, tomorrowStart).
			Distinct("remote_addr").
			Count(&uvCount)
		if uvCount > 0 {
			metrics.Today.UV = uvCount
		}
	} else if r.tableExists("nginx_fact_access_logs") {
		// 回退到事实表
		var pvCount int64
		r.db.Table("nginx_fact_access_logs").
			Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, todayStart, tomorrowStart).
			Count(&pvCount)
		if pvCount > 0 {
			metrics.Today.PV = pvCount
		}

		if r.tableExists("nginx_dim_ip") {
			var uvCount int64
			r.db.Table("nginx_fact_access_logs").
				Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, todayStart, tomorrowStart).
				Distinct("ip_id").
				Count(&uvCount)
			if uvCount > 0 {
				metrics.Today.UV = uvCount
			}
		}
	}

	// 计算昨日峰值OPS
	if r.tableExists("nginx_agg_hourly") {
		yesterdayStart, _ := time.ParseInLocation("2006-01-02", yesterdayStr, time.Local)
		yesterdayEnd := yesterdayStart.Add(24 * time.Hour)
		type HourlyMax struct {
			MaxRequests int64
		}
		var hm HourlyMax
		err := r.db.Model(&model.NginxAggHourly{}).
			Select("MAX(total_requests) as max_requests").
			Where("source_id = ? AND hour >= ? AND hour < ?", sourceID, yesterdayStart, yesterdayEnd).
			First(&hm).Error
		if err == nil && hm.MaxRequests > 0 {
			metrics.Yesterday.PeakOps = float64(hm.MaxRequests) / 3600.0
		}
	}

	// 昨日此时（昨日0点到昨日当前小时的累计）
	// 注意: 聚合器用 time.Parse (UTC) 存储小时，此处也需匹配
	if r.tableExists("nginx_agg_hourly") {
		yesterdayStart, _ := time.Parse("2006-01-02", yesterdayStr)
		yesterdayNowEnd := yesterdayStart.Add(time.Duration(currentHour) * time.Hour)

		type HourlySum struct {
			TotalRequests int64
			PVCount       int64
			UniqueIPs     int64
			Status2xx     int64
			Status3xx     int64
			Status4xx     int64
			Status5xx     int64
		}
		var hs HourlySum
		err := r.db.Model(&model.NginxAggHourly{}).
			Select("SUM(total_requests) as total_requests, SUM(pv_count) as pv_count, SUM(unique_ips) as unique_ips, SUM(status_2xx) as status_2xx, SUM(status_3xx) as status_3xx, SUM(status_4xx) as status_4xx, SUM(status_5xx) as status_5xx").
			Where("source_id = ? AND hour >= ? AND hour < ?", sourceID, yesterdayStart, yesterdayNowEnd).
			First(&hs).Error
		if err == nil {
			metrics.YesterdayNow.StatusHits = hs.TotalRequests
			if hs.PVCount > 0 {
				metrics.YesterdayNow.PV = hs.PVCount
			} else {
				metrics.YesterdayNow.PV = hs.TotalRequests
			}
			metrics.YesterdayNow.UV = hs.UniqueIPs
			metrics.YesterdayNow.Status2xx = hs.Status2xx
			metrics.YesterdayNow.Status3xx = hs.Status3xx
			metrics.YesterdayNow.Status4xx = hs.Status4xx
			metrics.YesterdayNow.Status5xx = hs.Status5xx
		}
	} else if r.tableExists("nginx_hourly_stats") {
		yesterdayStart, _ := time.Parse("2006-01-02", yesterdayStr)
		yesterdayNowEnd := yesterdayStart.Add(time.Duration(currentHour) * time.Hour)

		type HourlySum struct {
			TotalRequests  int64
			UniqueVisitors int64
			Status2xx      int64
			Status3xx      int64
			Status4xx      int64
			Status5xx      int64
		}
		var hs HourlySum
		err := r.db.Model(&model.NginxHourlyStats{}).
			Select("SUM(total_requests) as total_requests, SUM(unique_visitors) as unique_visitors, SUM(status_2xx) as status_2xx, SUM(status_3xx) as status_3xx, SUM(status_4xx) as status_4xx, SUM(status_5xx) as status_5xx").
			Where("source_id = ? AND hour >= ? AND hour < ?", sourceID, yesterdayStart, yesterdayNowEnd).
			First(&hs).Error
		if err == nil {
			metrics.YesterdayNow.StatusHits = hs.TotalRequests
			metrics.YesterdayNow.PV = hs.TotalRequests
			metrics.YesterdayNow.UV = hs.UniqueVisitors
			metrics.YesterdayNow.Status2xx = hs.Status2xx
			metrics.YesterdayNow.Status3xx = hs.Status3xx
			metrics.YesterdayNow.Status4xx = hs.Status4xx
			metrics.YesterdayNow.Status5xx = hs.Status5xx
		}
	}

	// 预计今日: 基于当前数据预测全天
	hourFactor := float64(24) / math.Max(float64(currentHour), 1.0)
	metrics.PredictToday = model.MetricSet{
		StatusHits:  int64(float64(metrics.Today.StatusHits) * hourFactor),
		PV:          int64(float64(metrics.Today.PV) * hourFactor),
		UV:          int64(float64(metrics.Today.UV) * hourFactor),
		RealtimeOps: metrics.Today.RealtimeOps,
		PeakOps:     metrics.Today.PeakOps,
		Status2xx:   int64(float64(metrics.Today.Status2xx) * hourFactor),
		Status3xx:   int64(float64(metrics.Today.Status3xx) * hourFactor),
		Status4xx:   int64(float64(metrics.Today.Status4xx) * hourFactor),
		Status5xx:   int64(float64(metrics.Today.Status5xx) * hourFactor),
	}

	return metrics, nil
}

// GetOverviewTrend 获取概况趋势（UV+PV）
func (r *NginxRepository) GetOverviewTrend(sourceID uint, mode string, date string) ([]model.OverviewTrendPoint, error) {
	var points []model.OverviewTrendPoint

	if mode == "hour" {
		// 按时：查指定日期的小时聚合
		// 注意: MySQL 连接使用 loc=Local，所以时间存储和查询都用本地时区
		dateTime, err := time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			dateTime, _ = time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), time.Local)
		}
		endTime := dateTime.Add(24 * time.Hour)

		// 用 map 存储查询到的数据
		dataMap := make(map[string]model.OverviewTrendPoint)

		if r.tableExists("nginx_agg_hourly") {
			type HourResult struct {
				Hour          time.Time
				TotalRequests int64
				PVCount       int64
				UniqueIPs     int64
			}
			var results []HourResult
			err := r.db.Model(&model.NginxAggHourly{}).
				Select("hour, total_requests, pv_count, unique_ips").
				Where("source_id = ? AND hour >= ? AND hour < ?", sourceID, dateTime, endTime).
				Order("hour ASC").
				Find(&results).Error
			if err == nil {
				for _, r := range results {
					timeKey := r.Hour.Format("15:00")
					dataMap[timeKey] = model.OverviewTrendPoint{
						Time: timeKey,
						PV:   r.PVCount,
						UV:   r.UniqueIPs,
					}
				}
			}
		}

		// 回退到旧表
		if len(dataMap) == 0 && r.tableExists("nginx_hourly_stats") {
			type HourResult struct {
				Hour           time.Time
				TotalRequests  int64
				UniqueVisitors int64
			}
			var results []HourResult
			err := r.db.Model(&model.NginxHourlyStats{}).
				Select("hour, total_requests, unique_visitors").
				Where("source_id = ? AND hour >= ? AND hour < ?", sourceID, dateTime, endTime).
				Order("hour ASC").
				Find(&results).Error
			if err == nil {
				for _, r := range results {
					timeKey := r.Hour.Format("15:00")
					dataMap[timeKey] = model.OverviewTrendPoint{
						Time: timeKey,
						PV:   r.TotalRequests,
						UV:   r.UniqueVisitors,
					}
				}
			}
		}

		// 聚合表均无数据时，从原始日志生成小时趋势
		if len(dataMap) == 0 {
			type RawHourResult struct {
				HourStr string
				PV      int64
				UV      int64
			}
			// 优先使用旧日志表（直接有 remote_addr 字段，UV 统计更准确）
			if r.tableExists("nginx_access_logs") {
				var results []RawHourResult
				err := r.db.Raw(
					"SELECT DATE_FORMAT(timestamp, '%H:00') as hour_str, COUNT(*) as pv, COUNT(DISTINCT remote_addr) as uv "+
						"FROM nginx_access_logs WHERE source_id = ? AND timestamp >= ? AND timestamp < ? "+
						"GROUP BY hour_str ORDER BY hour_str",
					sourceID, dateTime, endTime,
				).Scan(&results).Error
				if err == nil {
					for _, r := range results {
						dataMap[r.HourStr] = model.OverviewTrendPoint{
							Time: r.HourStr,
							PV:   r.PV,
							UV:   r.UV,
						}
					}
				}
			}
			// 回退到事实表（需要 JOIN 维度表获取正确 UV）
			if len(dataMap) == 0 && r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_ip") {
				var results []RawHourResult
				err := r.db.Raw(
					"SELECT DATE_FORMAT(f.timestamp, '%H:00') as hour_str, COUNT(*) as pv, COUNT(DISTINCT d.ip) as uv "+
						"FROM nginx_fact_access_logs f LEFT JOIN nginx_dim_ip d ON f.ip_id = d.id "+
						"WHERE f.source_id = ? AND f.timestamp >= ? AND f.timestamp < ? "+
						"GROUP BY hour_str ORDER BY hour_str",
					sourceID, dateTime, endTime,
				).Scan(&results).Error
				if err == nil {
					for _, r := range results {
						dataMap[r.HourStr] = model.OverviewTrendPoint{
							Time: r.HourStr,
							PV:   r.PV,
							UV:   r.UV,
						}
					}
				}
			}
		}

		// 生成从 00:00 到当前时间（如果是今天）或 23:00（如果是历史日期）的完整小时列表
		now := time.Now().Local()
		todayStr := now.Format("2006-01-02")
		isToday := date == todayStr

		var maxHour int
		if isToday {
			maxHour = now.Hour() // 今天只显示到当前小时
		} else {
			maxHour = 23 // 历史日期显示全天
		}

		for h := 0; h <= maxHour; h++ {
			timeKey := fmt.Sprintf("%02d:00", h)
			if point, exists := dataMap[timeKey]; exists {
				points = append(points, point)
			} else {
				// 没有数据的小时补0
				points = append(points, model.OverviewTrendPoint{
					Time: timeKey,
					PV:   0,
					UV:   0,
				})
			}
		}

		return points, nil
	} else {
		// 按天：最近30天的日聚合
		endDateStr := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -30)
		startDateStr := startDate.Format("2006-01-02")

		if r.tableExists("nginx_agg_daily") {
			type DayResult struct {
				Date          time.Time
				TotalRequests int64
				PVCount       int64
				UniqueIPs     int64
			}
			var results []DayResult
			err := r.db.Model(&model.NginxAggDaily{}).
				Select("date, total_requests, pv_count, unique_ips").
				Where("source_id = ? AND DATE(date) >= ? AND DATE(date) <= ?", sourceID, startDateStr, endDateStr).
				Order("date ASC").
				Find(&results).Error
			if err == nil && len(results) > 0 {
				for _, r := range results {
					points = append(points, model.OverviewTrendPoint{
						Time: r.Date.Format("01-02"),
						PV:   r.PVCount, // PV 就是实际的页面访问数
						UV:   r.UniqueIPs,
					})
				}
				return points, nil
			}
		}

		// 回退到旧表
		if r.tableExists("nginx_daily_stats") {
			type DayResult struct {
				Date           time.Time
				TotalRequests  int64
				UniqueVisitors int64
			}
			var results []DayResult
			err := r.db.Model(&model.NginxDailyStats{}).
				Select("date, total_requests, unique_visitors").
				Where("source_id = ? AND DATE(date) >= ? AND DATE(date) <= ?", sourceID, startDateStr, endDateStr).
				Order("date ASC").
				Find(&results).Error
			if err == nil && len(results) > 0 {
				for _, r := range results {
					points = append(points, model.OverviewTrendPoint{
						Time: r.Date.Format("01-02"),
						PV:   r.TotalRequests,
						UV:   r.UniqueVisitors,
					})
				}
				return points, nil
			}
		}

		// 聚合表均无数据时，从原始日志生成按天趋势
		if len(points) == 0 {
			startDate := time.Now().AddDate(0, 0, -30)
			type RawDayResult struct {
				DateStr string
				PV      int64
				UV      int64
			}
			// 优先使用旧日志表（直接有 remote_addr 字段，UV 统计更准确）
			if r.tableExists("nginx_access_logs") {
				var results []RawDayResult
				err := r.db.Raw(
					"SELECT DATE_FORMAT(timestamp, '%m-%d') as date_str, COUNT(*) as pv, COUNT(DISTINCT remote_addr) as uv "+
						"FROM nginx_access_logs WHERE source_id = ? AND timestamp >= ? "+
						"GROUP BY date_str ORDER BY date_str",
					sourceID, startDate,
				).Scan(&results).Error
				if err == nil && len(results) > 0 {
					for _, r := range results {
						points = append(points, model.OverviewTrendPoint{
							Time: r.DateStr,
							PV:   r.PV,
							UV:   r.UV,
						})
					}
					return points, nil
				}
			}
			// 回退到事实表（需要 JOIN 维度表获取正确 UV）
			if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_ip") {
				var results []RawDayResult
				err := r.db.Raw(
					"SELECT DATE_FORMAT(f.timestamp, '%m-%d') as date_str, COUNT(*) as pv, COUNT(DISTINCT d.ip) as uv "+
						"FROM nginx_fact_access_logs f LEFT JOIN nginx_dim_ip d ON f.ip_id = d.id "+
						"WHERE f.source_id = ? AND f.timestamp >= ? "+
						"GROUP BY date_str ORDER BY date_str",
					sourceID, startDate,
				).Scan(&results).Error
				if err == nil && len(results) > 0 {
					for _, r := range results {
						points = append(points, model.OverviewTrendPoint{
							Time: r.DateStr,
							PV:   r.PV,
							UV:   r.UV,
						})
					}
					return points, nil
				}
			}
		}
	}

	return points, nil
}

// GetNewVsReturningVisitors 获取新老访客对比
func (r *NginxRepository) GetNewVsReturningVisitors(sourceID uint) (*model.VisitorComparison, error) {
	now := time.Now().Local()
	todayStart, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	fmt.Printf("[DEBUG] GetNewVsReturningVisitors: sourceID=%d, todayStart=%s, yesterdayStart=%s\n",
		sourceID, todayStart.Format("2006-01-02 15:04:05"), yesterdayStart.Format("2006-01-02 15:04:05"))

	vc := &model.VisitorComparison{}

	calcForDate := func(dateStart, dateEnd time.Time) (newCount, retCount int64) {
		fmt.Printf("[DEBUG] calcForDate: dateStart=%s, dateEnd=%s\n",
			dateStart.Format("2006-01-02 15:04:05"), dateEnd.Format("2006-01-02 15:04:05"))

		// 优先使用新表，但如果没有数据则回退到旧表
		if r.tableExists("nginx_fact_access_logs") {
			// 当日所有IP
			type IPRow struct {
				IPID uint64
			}
			var dayIPs []IPRow
			r.db.Table("nginx_fact_access_logs").
				Select("DISTINCT ip_id").
				Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, dateStart, dateEnd).
				Find(&dayIPs)

			totalIPs := int64(len(dayIPs))
			fmt.Printf("[DEBUG] nginx_fact_access_logs: found %d IPs\n", totalIPs)
			if totalIPs > 0 {
				// 统计新访客: 这些IP在当日之前没有出现过
				ipIDs := make([]uint64, len(dayIPs))
				for i, ip := range dayIPs {
					ipIDs[i] = ip.IPID
				}

				var oldCount int64
				r.db.Table("nginx_fact_access_logs").
					Where("source_id = ? AND timestamp < ? AND ip_id IN ?", sourceID, dateStart, ipIDs).
					Distinct("ip_id").
					Count(&oldCount)

				retCount = oldCount
				newCount = totalIPs - oldCount
				if newCount < 0 {
					newCount = 0
				}
				return newCount, retCount
			}
			// totalIPs == 0, 继续尝试旧表
		}

		// 回退到旧表
		if r.tableExists("nginx_access_logs") {
			type IPRow struct {
				RemoteAddr string
			}
			var dayIPs []IPRow
			r.db.Model(&model.NginxAccessLog{}).
				Select("DISTINCT remote_addr").
				Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, dateStart, dateEnd).
				Find(&dayIPs)

			totalIPs := int64(len(dayIPs))
			fmt.Printf("[DEBUG] nginx_access_logs: found %d IPs\n", totalIPs)
			if totalIPs == 0 {
				return 0, 0
			}

			addrs := make([]string, len(dayIPs))
			for i, ip := range dayIPs {
				addrs[i] = ip.RemoteAddr
			}

			var oldCount int64
			r.db.Model(&model.NginxAccessLog{}).
				Where("source_id = ? AND timestamp < ? AND remote_addr IN ?", sourceID, dateStart, addrs).
				Distinct("remote_addr").
				Count(&oldCount)

			retCount = oldCount
			newCount = totalIPs - oldCount
			if newCount < 0 {
				newCount = 0
			}
			fmt.Printf("[DEBUG] nginx_access_logs: newCount=%d, retCount=%d\n", newCount, retCount)
			return newCount, retCount
		}

		fmt.Printf("[DEBUG] no tables available\n")
		return 0, 0
	}

	// 今日
	todayNew, todayRet := calcForDate(todayStart, todayStart.Add(24*time.Hour))
	vc.TodayNew = todayNew
	vc.TodayReturning = todayRet
	todayTotal := todayNew + todayRet
	if todayTotal > 0 {
		vc.TodayNewPct = float64(todayNew) / float64(todayTotal) * 100
		vc.TodayRetPct = float64(todayRet) / float64(todayTotal) * 100
	}

	// 昨日
	yestNew, yestRet := calcForDate(yesterdayStart, todayStart)
	vc.YestNew = yestNew
	vc.YestReturning = yestRet
	yestTotal := yestNew + yestRet
	if yestTotal > 0 {
		vc.YestNewPct = float64(yestNew) / float64(yestTotal) * 100
		vc.YestRetPct = float64(yestRet) / float64(yestTotal) * 100
	}

	return vc, nil
}

// GetTopReferersByVisitors 获取来路域名排行
func (r *NginxRepository) GetTopReferersByVisitors(sourceID uint, start, end time.Time, limit int) ([]model.RefererItem, error) {
	var items []model.RefererItem

	// 优先使用新表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_referer") {
		type RefResult struct {
			RefererDomain string
			Visitors      int64
		}
		var results []RefResult
		err := r.db.Table("nginx_fact_access_logs f").
			Select("ref.referer_domain, COUNT(DISTINCT f.ip_id) as visitors").
			Joins("LEFT JOIN nginx_dim_referer ref ON f.referer_id = ref.id").
			Where("f.source_id = ? AND f.timestamp >= ? AND f.timestamp < ?", sourceID, start, end).
			Where("ref.referer_domain != '' AND ref.referer_domain IS NOT NULL AND ref.referer_domain != '-'").
			Group("ref.referer_domain").
			Order("visitors DESC").
			Limit(limit).
			Find(&results).Error
		if err == nil && len(results) > 0 {
			for _, r := range results {
				items = append(items, model.RefererItem{
					Domain:   r.RefererDomain,
					Visitors: r.Visitors,
				})
			}
			return items, nil
		}
	}

	// 回退到旧表
	if r.tableExists("nginx_access_logs") {
		type RefResult struct {
			HTTPReferer string
			Visitors    int64
		}
		var results []RefResult
		err := r.db.Model(&model.NginxAccessLog{}).
			Select("http_referer, COUNT(DISTINCT remote_addr) as visitors").
			Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, start, end).
			Where("http_referer != '' AND http_referer != '-' AND http_referer IS NOT NULL").
			Group("http_referer").
			Order("visitors DESC").
			Limit(limit * 3). // 获取更多，后续提取域名聚合
			Find(&results).Error
		if err == nil {
			domainMap := make(map[string]int64)
			for _, r := range results {
				domain := extractDomain(r.HTTPReferer)
				if domain != "" && domain != "-" {
					domainMap[domain] += r.Visitors
				}
			}
			// 排序
			type kv struct {
				Key   string
				Value int64
			}
			var sorted []kv
			for k, v := range domainMap {
				sorted = append(sorted, kv{k, v})
			}
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[j].Value > sorted[i].Value {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
			count := limit
			if count > len(sorted) {
				count = len(sorted)
			}
			for i := 0; i < count; i++ {
				items = append(items, model.RefererItem{
					Domain:   sorted[i].Key,
					Visitors: sorted[i].Value,
				})
			}
		}
	}

	return items, nil
}

// GetTopVisitedPages 获取受访页面排行
func (r *NginxRepository) GetTopVisitedPages(sourceID uint, start, end time.Time, limit int) ([]model.PageItem, error) {
	var items []model.PageItem

	// 优先使用新表
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_url") {
		type PageResult struct {
			URLNormalized string
			Count         int64
		}
		var results []PageResult
		err := r.db.Table("nginx_fact_access_logs f").
			Select("u.url_normalized, COUNT(*) as count").
			Joins("LEFT JOIN nginx_dim_url u ON f.url_id = u.id").
			Where("f.source_id = ? AND f.timestamp >= ? AND f.timestamp < ? AND f.is_pv = ?", sourceID, start, end, true).
			Group("u.url_normalized").
			Order("count DESC").
			Limit(limit).
			Find(&results).Error
		if err == nil && len(results) > 0 {
			for _, r := range results {
				items = append(items, model.PageItem{
					Path:  r.URLNormalized,
					Count: r.Count,
				})
			}
			return items, nil
		}
	}

	// 回退到旧表
	if r.tableExists("nginx_access_logs") {
		type PageResult struct {
			URI   string
			Count int64
		}
		var results []PageResult
		err := r.db.Model(&model.NginxAccessLog{}).
			Select("uri, COUNT(*) as count").
			Where("source_id = ? AND timestamp >= ? AND timestamp < ?", sourceID, start, end).
			Group("uri").
			Order("count DESC").
			Limit(limit).
			Find(&results).Error
		if err == nil {
			for _, r := range results {
				items = append(items, model.PageItem{
					Path:  r.URI,
					Count: r.Count,
				})
			}
		}
	}

	return items, nil
}

// GetTopEntryPages 获取入口页面排行
func (r *NginxRepository) GetTopEntryPages(sourceID uint, start, end time.Time, limit int) ([]model.PageItem, error) {
	var items []model.PageItem

	// 优先使用新表: 每个IP第一次PV请求的url
	if r.tableExists("nginx_fact_access_logs") && r.tableExists("nginx_dim_url") {
		type EntryResult struct {
			URLNormalized string
			Count         int64
		}
		var results []EntryResult
		// 子查询：每个ip_id最早的PV请求
		err := r.db.Raw(`
			SELECT u.url_normalized, COUNT(*) as count
			FROM nginx_dim_url u
			INNER JOIN (
				SELECT f.url_id
				FROM nginx_fact_access_logs f
				INNER JOIN (
					SELECT ip_id, MIN(timestamp) as min_ts
					FROM nginx_fact_access_logs
					WHERE source_id = ? AND timestamp >= ? AND timestamp < ? AND is_pv = 1
					GROUP BY ip_id
				) first_pv ON f.ip_id = first_pv.ip_id AND f.timestamp = first_pv.min_ts
				WHERE f.source_id = ? AND f.is_pv = 1
			) entries ON u.id = entries.url_id
			GROUP BY u.url_normalized
			ORDER BY count DESC
			LIMIT ?
		`, sourceID, start, end, sourceID, limit).Find(&results).Error
		if err == nil && len(results) > 0 {
			for _, r := range results {
				items = append(items, model.PageItem{
					Path:  r.URLNormalized,
					Count: r.Count,
				})
			}
			return items, nil
		}
	}

	// 回退到旧表
	if r.tableExists("nginx_access_logs") {
		type EntryResult struct {
			URI   string
			Count int64
		}
		var results []EntryResult
		err := r.db.Raw(`
			SELECT sub.uri, COUNT(*) as count
			FROM (
				SELECT a.uri
				FROM nginx_access_logs a
				INNER JOIN (
					SELECT remote_addr, MIN(timestamp) as min_ts
					FROM nginx_access_logs
					WHERE source_id = ? AND timestamp >= ? AND timestamp < ?
					GROUP BY remote_addr
				) first_req ON a.remote_addr = first_req.remote_addr AND a.timestamp = first_req.min_ts
				WHERE a.source_id = ?
			) sub
			GROUP BY sub.uri
			ORDER BY count DESC
			LIMIT ?
		`, sourceID, start, end, sourceID, limit).Find(&results).Error
		if err == nil {
			for _, r := range results {
				items = append(items, model.PageItem{
					Path:  r.URI,
					Count: r.Count,
				})
			}
		}
	}

	return items, nil
}

// GetOverviewGeo 获取概况页地域分布（复用现有逻辑，加sourceID和date过滤）
func (r *NginxRepository) GetOverviewGeo(sourceID uint, start, end time.Time, scope string) ([]model.GeoStats, error) {
	sid := sourceID
	level := "province"
	if scope == "global" {
		level = "country"
	}
	return r.GetGeoDistribution(&sid, start, end, level)
}

// GetOverviewDevices 获取概况页终端设备分布
func (r *NginxRepository) GetOverviewDevices(sourceID uint, start, end time.Time) ([]model.DeviceStats, error) {
	sid := sourceID
	return r.GetDeviceDistribution(&sid, start, end)
}

// extractDomain 从URL中提取域名
func extractDomain(rawURL string) string {
	if rawURL == "" || rawURL == "-" {
		return ""
	}
	// 确保有协议前缀
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	return host
}

// DeleteOldAggDaily 删除过期日聚合
func (r *NginxRepository) DeleteOldAggDaily(sourceID uint, beforeTime time.Time) error {
	return r.db.Where("source_id = ? AND date < ?", sourceID, beforeTime).Delete(&model.NginxAggDaily{}).Error
}
