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

package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"github.com/ydcloud-dy/opshub/plugins/nginx/repository"
	"github.com/ydcloud-dy/opshub/plugins/nginx/service"
	"gorm.io/gorm"
)

type Handler struct {
	db       *gorm.DB
	repo     *repository.NginxRepository
	geoSvc   *service.GeolocationService
	uaParser *service.UAParser
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:       db,
		repo:     repository.NewNginxRepository(db),
		geoSvc:   service.NewGeolocationService(),
		uaParser: service.NewUAParser(),
	}
}

// ==================== 数据源管理 ====================

// ListSources 获取数据源列表
// @Summary 获取数据源列表
// @Description 分页获取Nginx数据源列表
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param type query string false "数据源类型"
// @Param status query int false "状态"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/sources [get]
func (h *Handler) ListSources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	sourceType := c.Query("type")
	statusStr := c.Query("status")

	var status *int
	if statusStr != "" {
		s, _ := strconv.Atoi(statusStr)
		status = &s
	}

	sources, total, err := h.repo.ListSources(page, pageSize, sourceType, status)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取数据源列表失败")
		return
	}

	response.Success(c, gin.H{
		"list":     sources,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetSource 获取数据源详情
// @Summary 获取数据源详情
// @Description 获取指定数据源的详细信息
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "数据源ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "数据源不存在"
// @Router /nginx/sources/{id} [get]
func (h *Handler) GetSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	source, err := h.repo.GetSourceByID(uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}
	response.Success(c, source)
}

// CreateSource 创建数据源
// @Summary 创建数据源
// @Description 创建新的Nginx数据源
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body model.NginxSource true "数据源信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /nginx/sources [post]
func (h *Handler) CreateSource(c *gin.Context) {
	var source model.NginxSource
	if err := c.ShouldBindJSON(&source); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.repo.CreateSource(&source); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建数据源失败: "+err.Error())
		return
	}
	response.Success(c, source)
}

// UpdateSource 更新数据源
// @Summary 更新数据源
// @Description 更新指定数据源的信息
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "数据源ID"
// @Param body body model.NginxSource true "数据源信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 404 {object} response.Response "数据源不存在"
// @Router /nginx/sources/{id} [put]
func (h *Handler) UpdateSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	source, err := h.repo.GetSourceByID(uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}

	if err := c.ShouldBindJSON(source); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.repo.UpdateSource(source); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新数据源失败: "+err.Error())
		return
	}
	response.Success(c, source)
}

// DeleteSource 删除数据源
// @Summary 删除数据源
// @Description 删除指定的数据源
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "数据源ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /nginx/sources/{id} [delete]
func (h *Handler) DeleteSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.repo.DeleteSource(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除数据源失败")
		return
	}
	response.Success(c, nil)
}

// ==================== 概况统计 ====================

// GetOverview 获取概况统计
// @Summary 获取概况统计
// @Description 获取今日概况统计数据
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview [get]
func (h *Handler) GetOverview(c *gin.Context) {
	overview, err := h.repo.GetTodayOverview()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取概况失败")
		return
	}

	// 获取请求趋势（最近24小时）
	trend, err := h.repo.GetRequestsTrend(nil, 24)
	if err == nil {
		overview.RequestsTrend = trend
	}

	response.Success(c, overview)
}

// GetRequestsTrend 获取请求趋势
// @Summary 获取请求趋势
// @Description 获取指定时间范围的请求趋势数据
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID"
// @Param hours query int false "小时数" default(24)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/trend [get]
func (h *Handler) GetRequestsTrend(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	sourceIDStr := c.Query("sourceId")

	var sourceID *uint
	if sourceIDStr != "" {
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		sid := uint(id)
		sourceID = &sid
	}

	trend, err := h.repo.GetRequestsTrend(sourceID, hours)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取趋势数据失败")
		return
	}
	response.Success(c, trend)
}

// ==================== 数据日报 ====================

// GetDailyReport 获取日报数据（从数据库读取已聚合的统计数据）
// @Summary 获取日报数据
// @Description 从数据库获取指定日期范围的Nginx统计日报
// @Tags Nginx统计-数据日报
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param startDate query string false "开始日期"
// @Param endDate query string false "结束日期"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/daily-report [get]
func (h *Handler) GetDailyReport(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// 必须选择数据源
	if sourceIDStr == "" {
		response.Success(c, []interface{}{})
		return
	}

	sourceID, _ := strconv.ParseUint(sourceIDStr, 10, 32)
	_, err := h.repo.GetSourceByID(uint(sourceID))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}

	// 解析日期范围
	var startDate, endDate time.Time
	now := time.Now()

	if startDateStr != "" {
		startDate, _ = time.ParseInLocation("2006-01-02", startDateStr, time.Local)
	} else {
		startDate = now.AddDate(0, 0, -7) // 默认最近7天
	}

	if endDateStr != "" {
		endDate, _ = time.ParseInLocation("2006-01-02", endDateStr, time.Local)
	} else {
		endDate = now
	}

	// 从数据库查询已聚合的统计数据
	stats, err := h.repo.ListDailyStats(uint(sourceID), startDate, endDate)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取数据失败: "+err.Error())
		return
	}

	// 转换为前端期望的格式
	result := make([]map[string]interface{}, 0, len(stats))
	for _, s := range stats {
		result = append(result, map[string]interface{}{
			"date":            s.Date.Format("2006-01-02"),
			"totalRequests":   s.TotalRequests,
			"uniqueVisitors":  s.UniqueVisitors,
			"totalBandwidth":  s.TotalBandwidth,
			"avgResponseTime": s.AvgResponseTime,
			"status2xx":       s.Status2xx,
			"status3xx":       s.Status3xx,
			"status4xx":       s.Status4xx,
			"status5xx":       s.Status5xx,
		})
	}

	response.Success(c, result)
}

// ==================== 访问明细 ====================

// ListAccessLogs 获取访问日志列表
// @Summary 获取访问日志列表
// @Description 分页获取访问日志列表
// @Tags Nginx统计-访问明细
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param remoteAddr query string false "客户端IP"
// @Param uri query string false "请求URI"
// @Param status query int false "状态码"
// @Param method query string false "请求方法"
// @Param host query string false "请求主机"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/access-logs [get]
func (h *Handler) ListAccessLogs(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = &t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = &t
		}
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if v := c.Query("remoteAddr"); v != "" {
		filters["remoteAddr"] = v
	}
	if v := c.Query("uri"); v != "" {
		filters["uri"] = v
	}
	if v := c.Query("status"); v != "" {
		if s, err := strconv.Atoi(v); err == nil {
			filters["status"] = s
		}
	}
	if v := c.Query("method"); v != "" {
		filters["method"] = v
	}
	if v := c.Query("host"); v != "" {
		filters["host"] = v
	}

	logs, total, err := h.repo.ListAccessLogs(uint(sourceID), page, pageSize, startTime, endTime, filters)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取访问日志失败")
		return
	}

	response.Success(c, gin.H{
		"list":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetTopURIs 获取 Top URI
// @Summary 获取 Top URI
// @Description 获取访问量最高的URI列表
// @Tags Nginx统计-访问明细
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/access-logs/top-uris [get]
func (h *Handler) GetTopURIs(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 默认最近24小时
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		}
	}

	results, err := h.repo.GetTopURIs(uint(sourceID), startTime, endTime, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取Top URI失败")
		return
	}
	response.Success(c, results)
}

// GetTopIPs 获取 Top IP
// @Summary 获取 Top IP
// @Description 获取访问量最高的IP列表
// @Tags Nginx统计-访问明细
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/access-logs/top-ips [get]
func (h *Handler) GetTopIPs(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 默认最近24小时
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		}
	}

	results, err := h.repo.GetTopIPs(uint(sourceID), startTime, endTime, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取Top IP失败")
		return
	}
	response.Success(c, results)
}

// fetchNginxStatsFromHost 实时从主机获取Nginx统计数据
func (h *Handler) fetchNginxStatsFromHost(source *model.NginxSource, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	if source.HostID == nil {
		return nil, fmt.Errorf("数据源未关联主机")
	}

	// 获取主机信息
	var host assetbiz.Host
	if err := h.db.First(&host, *source.HostID).Error; err != nil {
		return nil, fmt.Errorf("获取主机信息失败: %w", err)
	}

	// 获取凭证信息
	var credential assetbiz.Credential
	if err := h.db.First(&credential, host.CredentialID).Error; err != nil {
		return nil, fmt.Errorf("获取凭证信息失败: %w", err)
	}

	// 解密凭证
	if err := decryptCredential(&credential); err != nil {
		return nil, fmt.Errorf("解密凭证失败: %w", err)
	}

	// 创建SSH连接
	sshClient, err := createSSHClient(&host, &credential)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}
	defer sshClient.Close()

	// 获取日志路径
	logPath := source.LogPath
	if logPath == "" {
		logPath = "/var/log/nginx/access.log"
	}

	// 读取日志文件
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 读取整个日志文件
	output, err := session.CombinedOutput(fmt.Sprintf("cat %s 2>&1", logPath))
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %s", string(output))
	}

	logContent := string(output)
	if len(logContent) == 0 {
		return nil, fmt.Errorf("日志文件为空: %s", logPath)
	}

	// 解析日志并按日期分组统计
	dailyStats := make(map[string]*DailyStatsData)
	lineCount := 0
	parsedCount := 0
	matchedCount := 0

	scanner := bufio.NewScanner(strings.NewReader(logContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineCount++

		// 根据日志格式选择解析方式
		var entry *LogEntry
		var err error
		if source.LogFormat == "json" {
			entry, err = parseJSONLogLine(line)
		} else {
			entry, err = parseLogLine(line)
		}
		if err != nil {
			continue
		}
		parsedCount++

		// 检查是否在日期范围内
		if entry.Timestamp.Before(startDate) || entry.Timestamp.After(endDate) {
			continue
		}
		matchedCount++

		// 按日期分组
		dateKey := entry.Timestamp.Format("2006-01-02")
		if _, ok := dailyStats[dateKey]; !ok {
			dailyStats[dateKey] = &DailyStatsData{
				Date:      dateKey,
				UniqueIPs: make(map[string]bool),
			}
		}

		stats := dailyStats[dateKey]
		stats.TotalRequests++
		stats.TotalBandwidth += entry.BodyBytesSent
		stats.UniqueIPs[entry.RemoteAddr] = true

		switch {
		case entry.Status >= 200 && entry.Status < 300:
			stats.Status2xx++
		case entry.Status >= 300 && entry.Status < 400:
			stats.Status3xx++
		case entry.Status >= 400 && entry.Status < 500:
			stats.Status4xx++
		case entry.Status >= 500:
			stats.Status5xx++
		}

		stats.TotalResponseTime += entry.RequestTime
	}

	// 如果没有匹配的数据，返回空数组（不再返回错误）
	if matchedCount == 0 {
		return make([]map[string]interface{}, 0), nil
	}

	// 转换为返回格式 - 使用 make 确保返回空数组而不是 null
	result := make([]map[string]interface{}, 0)
	for dateKey, stats := range dailyStats {
		avgResponseTime := float64(0)
		if stats.TotalRequests > 0 {
			avgResponseTime = stats.TotalResponseTime / float64(stats.TotalRequests)
		}

		result = append(result, map[string]interface{}{
			"date":            dateKey,
			"totalRequests":   stats.TotalRequests,
			"uniqueVisitors":  len(stats.UniqueIPs),
			"totalBandwidth":  stats.TotalBandwidth,
			"avgResponseTime": avgResponseTime,
			"status2xx":       stats.Status2xx,
			"status3xx":       stats.Status3xx,
			"status4xx":       stats.Status4xx,
			"status5xx":       stats.Status5xx,
		})
	}

	// 按日期倒序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i]["date"].(string) > result[j]["date"].(string)
	})

	return result, nil
}

// DailyStatsData 每日统计数据
type DailyStatsData struct {
	Date              string
	TotalRequests     int64
	UniqueIPs         map[string]bool
	TotalBandwidth    int64
	TotalResponseTime float64
	Status2xx         int64
	Status3xx         int64
	Status4xx         int64
	Status5xx         int64
}

// parseLogLine 解析单行Nginx日志 (combined 格式)
func parseLogLine(line string) (*LogEntry, error) {
	// combined 格式: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
	pattern := regexp.MustCompile(`^(\S+)\s+-\s+(\S+)\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\d+)\s+"([^"]*)"\s+"([^"]*)"(?:\s+(\S+))?`)

	matches := pattern.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("无法解析日志行")
	}

	entry := &LogEntry{
		RemoteAddr: matches[1],
	}

	// 解析时间
	timeStr := matches[3]
	// 先尝试带时区的格式
	t, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeStr)
	if err != nil {
		// 不带时区，使用本地时区解析（Nginx日志通常是服务器本地时间）
		t, err = time.ParseInLocation("02/Jan/2006:15:04:05", timeStr, time.Local)
		if err != nil {
			return nil, fmt.Errorf("解析时间失败")
		}
	}
	entry.Timestamp = t

	// 状态码
	entry.Status, _ = strconv.Atoi(matches[5])

	// 响应体大小
	entry.BodyBytesSent, _ = strconv.ParseInt(matches[6], 10, 64)

	// 请求时间 (可选)
	if len(matches) > 9 && matches[9] != "" && matches[9] != "-" {
		entry.RequestTime, _ = strconv.ParseFloat(matches[9], 64)
	}

	return entry, nil
}

// NginxJSONLog JSON格式的Nginx日志结构
type NginxJSONLog struct {
	TimeLocal     string `json:"time_local"`
	Time          string `json:"time"`       // 备用时间字段
	Timestamp     string `json:"@timestamp"` // 备用时间字段
	RemoteAddr    string `json:"remote_addr"`
	ClientIP      string `json:"client_ip"` // 备用 IP 字段
	Request       string `json:"request"`
	Status        int    `json:"status"`
	StatusStr     string `json:"status_code"` // 备用状态码字段（字符串）
	Bytes         int64  `json:"bytes"`
	BodyBytes     int64  `json:"body_bytes_sent"` // 兼容另一种字段名
	BytesSent     int64  `json:"bytes_sent"`      // 备用字节字段
	RequestTime   string `json:"request_time"`
	UpstreamTime  string `json:"upstream_time"`
	UserAgent     string `json:"user_agent"`
	HTTPUserAgent string `json:"http_user_agent"` // 备用 UA 字段
	Referer       string `json:"referer"`
	HTTPReferer   string `json:"http_referer"` // 备用 referer 字段
	Host          string `json:"host"`
	ServerName    string `json:"server_name"` // 备用 host 字段
	XForwarded    string `json:"x_forwarded"`
	XForwardedFor string `json:"x_forwarded_for"`
}

// parseJSONLogLine 解析JSON格式的Nginx日志
func parseJSONLogLine(line string) (*LogEntry, error) {
	var jsonLog NginxJSONLog
	if err := json.Unmarshal([]byte(line), &jsonLog); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	entry := &LogEntry{
		RemoteAddr: jsonLog.RemoteAddr,
		Status:     jsonLog.Status,
	}

	// 备用 IP 字段
	if entry.RemoteAddr == "" {
		entry.RemoteAddr = jsonLog.ClientIP
	}
	// 处理 X-Forwarded-For
	if entry.RemoteAddr == "" || strings.HasPrefix(entry.RemoteAddr, "10.") || strings.HasPrefix(entry.RemoteAddr, "192.168.") || strings.HasPrefix(entry.RemoteAddr, "172.") {
		xff := jsonLog.XForwarded
		if xff == "" {
			xff = jsonLog.XForwardedFor
		}
		if xff != "" && xff != "-" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				realIP := strings.TrimSpace(parts[0])
				if realIP != "" && realIP != "-" {
					entry.RemoteAddr = realIP
				}
			}
		}
	}

	// 备用状态码
	if entry.Status == 0 && jsonLog.StatusStr != "" {
		entry.Status, _ = strconv.Atoi(jsonLog.StatusStr)
	}

	// 响应体大小
	if jsonLog.Bytes > 0 {
		entry.BodyBytesSent = jsonLog.Bytes
	} else if jsonLog.BodyBytes > 0 {
		entry.BodyBytesSent = jsonLog.BodyBytes
	} else {
		entry.BodyBytesSent = jsonLog.BytesSent
	}

	// 解析时间 - 支持多种字段和格式
	timeStr := jsonLog.TimeLocal
	if timeStr == "" {
		timeStr = jsonLog.Time
	}
	if timeStr == "" {
		timeStr = jsonLog.Timestamp
	}

	if timeStr == "" {
		return nil, fmt.Errorf("缺少时间字段")
	}

	// 尝试多种时间格式
	var t time.Time
	var parseErr error
	parsed := false

	// 先尝试带时区的格式
	formatsWithTZ := []string{
		"02/Jan/2006:15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05+08:00",
		"2006-01-02T15:04:05-07:00",
	}
	for _, f := range formatsWithTZ {
		t, parseErr = time.Parse(f, timeStr)
		if parseErr == nil {
			parsed = true
			break
		}
	}

	// 不带时区的格式，使用本地时区解析（Nginx日志通常是服务器本地时间）
	if !parsed {
		formatsWithoutTZ := []string{
			"02/Jan/2006:15:04:05",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, f := range formatsWithoutTZ {
			t, parseErr = time.ParseInLocation(f, timeStr, time.Local)
			if parseErr == nil {
				parsed = true
				break
			}
		}
	}

	if !parsed {
		return nil, fmt.Errorf("解析时间失败: %s", timeStr)
	}
	entry.Timestamp = t

	// 请求时间
	if jsonLog.RequestTime != "" && jsonLog.RequestTime != "-" {
		entry.RequestTime, _ = strconv.ParseFloat(jsonLog.RequestTime, 64)
	}

	return entry, nil
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp     time.Time
	RemoteAddr    string
	Status        int
	BodyBytesSent int64
	RequestTime   float64
}

// ==================== 新增统计接口 ====================

// GetTimeSeries 获取时间序列数据
// @Summary 获取时间序列数据
// @Description 获取指定时间范围的时间序列统计数据
// @Tags Nginx统计-分析
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param interval query string false "间隔" default(hour)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/stats/timeseries [get]
func (h *Handler) GetTimeSeries(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")
	interval := c.DefaultQuery("interval", "hour")

	var sourceID *uint
	if sourceIDStr != "" {
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		sid := uint(id)
		sourceID = &sid
	}

	// 默认时间范围
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	if interval == "day" {
		startTime = endTime.AddDate(0, 0, -30)
	}

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", et); err == nil {
			endTime = t.Add(24*time.Hour - time.Second)
		}
	}

	results, err := h.repo.GetTimeSeries(sourceID, startTime, endTime, interval)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取时间序列数据失败")
		return
	}
	response.Success(c, results)
}

// GetGeoDistribution 获取地理分布统计
// @Summary 获取地理分布统计
// @Description 获取访问者地理位置分布统计
// @Tags Nginx统计-分析
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param level query string false "统计级别 country/province/city" default(country)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/stats/geo [get]
func (h *Handler) GetGeoDistribution(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")
	level := c.DefaultQuery("level", "country")

	var sourceID *uint
	if sourceIDStr != "" {
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		sid := uint(id)
		sourceID = &sid
	}

	// 默认最近7天
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", et); err == nil {
			endTime = t.Add(24*time.Hour - time.Second)
		}
	}

	results, err := h.repo.GetGeoDistribution(sourceID, startTime, endTime, level)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取地理分布数据失败")
		return
	}
	response.Success(c, results)
}

// GetBrowserDistribution 获取浏览器分布统计
// @Summary 获取浏览器分布统计
// @Description 获取访问者浏览器分布统计
// @Tags Nginx统计-分析
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/stats/browsers [get]
func (h *Handler) GetBrowserDistribution(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")

	var sourceID *uint
	if sourceIDStr != "" {
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		sid := uint(id)
		sourceID = &sid
	}

	// 默认最近7天
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", et); err == nil {
			endTime = t.Add(24*time.Hour - time.Second)
		}
	}

	results, err := h.repo.GetBrowserDistribution(sourceID, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取浏览器分布数据失败")
		return
	}
	response.Success(c, results)
}

// GetDeviceDistribution 获取设备分布统计
// @Summary 获取设备分布统计
// @Description 获取访问者设备类型分布统计
// @Tags Nginx统计-分析
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/stats/devices [get]
func (h *Handler) GetDeviceDistribution(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")

	var sourceID *uint
	if sourceIDStr != "" {
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		sid := uint(id)
		sourceID = &sid
	}

	// 默认最近7天
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", et); err == nil {
			endTime = t.Add(24*time.Hour - time.Second)
		}
	}

	results, err := h.repo.GetDeviceDistribution(sourceID, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取设备分布数据失败")
		return
	}
	response.Success(c, results)
}

// GetTopIPsWithGeo 获取带地理信息的 Top IP
// @Summary 获取带地理信息的 Top IP
// @Description 获取访问量最高的IP列表，包含地理位置信息
// @Tags Nginx统计-分析
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/stats/top-ips [get]
func (h *Handler) GetTopIPsWithGeo(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 默认最近24小时
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = t
		}
	}

	results, err := h.repo.GetTopIPsWithGeo(uint(sourceID), startTime, endTime, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取Top IP失败")
		return
	}
	response.Success(c, results)
}

// ListAccessLogsWithDimensions 获取带维度信息的访问日志
// @Summary 获取带维度信息的访问日志
// @Description 获取带地理位置、浏览器等维度信息的访问日志
// @Tags Nginx统计-访问明细
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/logs [get]
func (h *Handler) ListAccessLogsWithDimensions(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", st); err == nil {
			startTime = &t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", et); err == nil {
			endTime = &t
		}
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if v := c.Query("remoteAddr"); v != "" {
		filters["remoteAddr"] = v
	}
	if v := c.Query("uri"); v != "" {
		filters["uri"] = v
	}
	if v := c.Query("status"); v != "" {
		if s, err := strconv.Atoi(v); err == nil {
			filters["status"] = s
		}
	}
	if v := c.Query("method"); v != "" {
		filters["method"] = v
	}
	if v := c.Query("host"); v != "" {
		filters["host"] = v
	}

	logs, total, err := h.repo.ListFactAccessLogsWithDimensions(uint(sourceID), page, pageSize, startTime, endTime, filters)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取访问日志失败")
		return
	}

	response.Success(c, gin.H{
		"list":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// BackfillGeoData 回填地理位置数据
// @Summary 回填地理位置数据
// @Description 为现有的访问日志回填地理位置信息
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID（不指定则处理所有）"
// @Param batchSize query int false "批量处理数量" default(1000)
// @Success 200 {object} response.Response "回填成功"
// @Router /nginx/backfill-geo [post]
func (h *Handler) BackfillGeoData(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")
	batchSize, _ := strconv.Atoi(c.DefaultQuery("batchSize", "1000"))
	if batchSize <= 0 || batchSize > 5000 {
		batchSize = 1000
	}

	var totalUpdated int64
	var lastID uint64 = 0

	// 分批处理，使用 ID 分页避免重复处理
	for {
		var logs []model.NginxAccessLog

		// 构建查询条件：country 为空或为 "-" 的日志，按 ID 分页
		query := h.db.Model(&model.NginxAccessLog{}).
			Where("(country = '' OR country = '-' OR country IS NULL)").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(batchSize)

		if sourceIDStr != "" {
			sourceID, _ := strconv.ParseUint(sourceIDStr, 10, 32)
			query = query.Where("source_id = ?", sourceID)
		}

		if err := query.Find(&logs).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "查询日志失败: "+err.Error())
			return
		}

		if len(logs) == 0 {
			break
		}

		// 更新 lastID 用于下一批
		lastID = logs[len(logs)-1].ID

		// 更新每条日志的地理位置信息
		for _, log := range logs {
			geoInfo, _ := h.geoSvc.Lookup(log.RemoteAddr)
			uaInfo := h.uaParser.Parse(log.HTTPUserAgent)

			updates := map[string]interface{}{}

			// 地理位置信息
			if geoInfo != nil {
				if geoInfo.Country != "" {
					updates["country"] = geoInfo.Country
				}
				if geoInfo.Province != "" {
					updates["province"] = geoInfo.Province
				}
				if geoInfo.City != "" {
					updates["city"] = geoInfo.City
				}
				if geoInfo.ISP != "" {
					updates["isp"] = geoInfo.ISP
				}
			}

			// UA 信息（如果为空也一起更新）
			if log.Browser == "" || log.Browser == "-" {
				if uaInfo.Browser != "" {
					updates["browser"] = uaInfo.Browser
				}
				if uaInfo.BrowserVersion != "" {
					updates["browser_version"] = uaInfo.BrowserVersion
				}
				if uaInfo.OS != "" {
					updates["os"] = uaInfo.OS
				}
				if uaInfo.OSVersion != "" {
					updates["os_version"] = uaInfo.OSVersion
				}
				if uaInfo.DeviceType != "" {
					updates["device_type"] = uaInfo.DeviceType
				}
			}

			if len(updates) > 0 {
				if err := h.db.Model(&model.NginxAccessLog{}).Where("id = ?", log.ID).Updates(updates).Error; err != nil {
					fmt.Printf("更新日志 %d 失败: %v\n", log.ID, err)
					continue
				}
				totalUpdated++
			}
		}
	}

	response.Success(c, gin.H{
		"message":      "回填完成",
		"totalUpdated": totalUpdated,
	})
}

// ==================== 概况页面接口 ====================

// GetActiveVisitors 获取活跃访客数
// @Summary 获取活跃访客数
// @Description 获取指定站点最近15分钟的活跃访客数
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/active-visitors [get]
func (h *Handler) GetActiveVisitors(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	count, err := h.repo.GetActiveVisitors(uint(sourceID), 15)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取活跃访客失败")
		return
	}
	response.Success(c, gin.H{"count": count})
}

// GetCoreMetrics 获取核心指标
// @Summary 获取核心指标
// @Description 获取今日/昨日/预计今日/昨日此时的核心指标
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/core-metrics [get]
func (h *Handler) GetCoreMetrics(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	metrics, err := h.repo.GetCoreMetrics(uint(sourceID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取核心指标失败")
		return
	}
	response.Success(c, metrics)
}

// GetOverviewTrend 获取概况趋势（UV+PV）
// @Summary 获取概况趋势
// @Description 获取按时/按天的UV和PV趋势数据
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param mode query string false "模式: hour 或 day" default(hour)
// @Param date query string false "日期(hour模式下指定)" default(今天)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/overview-trend [get]
func (h *Handler) GetOverviewTrend(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	mode := c.DefaultQuery("mode", "hour")
	date := c.DefaultQuery("date", time.Now().Local().Format("2006-01-02"))

	points, err := h.repo.GetOverviewTrend(uint(sourceID), mode, date)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取趋势数据失败")
		return
	}
	response.Success(c, points)
}

// GetNewVsReturning 获取新老访客对比
// @Summary 获取新老访客对比
// @Description 获取今日和昨日的新老访客对比数据
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/new-vs-returning [get]
func (h *Handler) GetNewVsReturning(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	vc, err := h.repo.GetNewVsReturningVisitors(uint(sourceID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取新老访客数据失败")
		return
	}
	response.Success(c, vc)
}

// GetTopReferers 获取来路排行
// @Summary 获取来路排行
// @Description 获取访客来路域名排行
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/top-referers [get]
func (h *Handler) GetTopReferers(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	// 默认今日
	now := time.Now().Local()
	start, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	end := start.Add(24 * time.Hour)

	items, err := h.repo.GetTopReferersByVisitors(uint(sourceID), start, end, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取来路排行失败")
		return
	}
	response.Success(c, items)
}

// GetTopPages 获取受访页面排行
// @Summary 获取受访页面排行
// @Description 获取访问量最高的页面排行
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/top-pages [get]
func (h *Handler) GetTopPages(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	now := time.Now().Local()
	start, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	end := start.Add(24 * time.Hour)

	items, err := h.repo.GetTopVisitedPages(uint(sourceID), start, end, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取受访页面排行失败")
		return
	}
	response.Success(c, items)
}

// GetTopEntryPages 获取入口页面排行
// @Summary 获取入口页面排行
// @Description 获取用户首次访问的入口页面排行
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/top-entry-pages [get]
func (h *Handler) GetTopEntryPages(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	now := time.Now().Local()
	start, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	end := start.Add(24 * time.Hour)

	items, err := h.repo.GetTopEntryPages(uint(sourceID), start, end, limit)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取入口页面排行失败")
		return
	}
	response.Success(c, items)
}

// GetOverviewGeo 获取概况页地域分布
// @Summary 获取概况页地域分布
// @Description 获取指定站点的地域分布数据
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Param scope query string false "范围: domestic 或 global" default(domestic)
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/geo [get]
func (h *Handler) GetOverviewGeo(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	scope := c.DefaultQuery("scope", "domestic")
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	now := time.Now().Local()
	start, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	end := start.Add(24 * time.Hour)

	stats, err := h.repo.GetOverviewGeo(uint(sourceID), start, end, scope)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取地域分布失败")
		return
	}
	response.Success(c, stats)
}

// GetOverviewDevices 获取概况页终端设备分布
// @Summary 获取概况页终端设备分布
// @Description 获取指定站点的终端设备类型分布
// @Tags Nginx统计-概况
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int true "数据源ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /nginx/overview/devices [get]
func (h *Handler) GetOverviewDevices(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 32)
	if sourceID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请指定数据源ID")
		return
	}

	now := time.Now().Local()
	start, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	end := start.Add(24 * time.Hour)

	stats, err := h.repo.GetOverviewDevices(uint(sourceID), start, end)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取终端设备分布失败")
		return
	}
	response.Success(c, stats)
}
