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

package model

import (
	"time"
)

// ============== Source Types ==============

// NginxSourceType Nginx 数据源类型
type NginxSourceType string

const (
	SourceTypeHost       NginxSourceType = "host"        // 主机上的 Nginx
	SourceTypeK8sIngress NginxSourceType = "k8s_ingress" // K8s Ingress-Nginx
)

// LogFormat 日志格式类型
type LogFormat string

const (
	LogFormatCombined LogFormat = "combined" // 标准 combined 格式
	LogFormatJSON     LogFormat = "json"     // JSON 格式
	LogFormatCustom   LogFormat = "custom"   // 自定义格式
)

// NginxSource Nginx 数据源配置
type NginxSource struct {
	ID          uint            `gorm:"primarykey" json:"id"`
	Name        string          `gorm:"type:varchar(100);not null" json:"name"`
	Type        NginxSourceType `gorm:"type:varchar(20);not null" json:"type"`
	Description string          `gorm:"type:varchar(500)" json:"description"`
	Status      int             `gorm:"type:tinyint;default:1" json:"status"`

	// 主机类型配置
	HostID    *uint  `gorm:"index" json:"hostId"`
	LogPath   string `gorm:"type:varchar(500)" json:"logPath"`
	LogFormat string `gorm:"type:varchar(50);default:'combined'" json:"logFormat"`

	// K8s Ingress 类型配置
	ClusterID        *uint  `gorm:"index" json:"clusterId"`
	Namespace        string `gorm:"type:varchar(100)" json:"namespace"`
	IngressName      string `gorm:"type:varchar(100)" json:"ingressName"`
	K8sPodSelector   string `gorm:"type:varchar(200)" json:"k8sPodSelector"`   // Pod 标签选择器
	K8sContainerName string `gorm:"type:varchar(100)" json:"k8sContainerName"` // 容器名称

	// 高级配置
	LogFormatConfig string `gorm:"type:text" json:"logFormatConfig"`             // 自定义日志格式配置
	GeoEnabled      bool   `gorm:"type:tinyint;default:1" json:"geoEnabled"`     // 是否启用地理位置解析
	SessionEnabled  bool   `gorm:"type:tinyint;default:0" json:"sessionEnabled"` // 是否启用会话跟踪

	// 通用配置
	CollectInterval int `gorm:"type:int;default:60" json:"collectInterval"`
	RetentionDays   int `gorm:"type:int;default:30" json:"retentionDays"`

	// 采集状态
	LastCollectAt   *time.Time `gorm:"type:datetime" json:"lastCollectAt"`
	LastCollectLogs int64      `gorm:"type:bigint;default:0" json:"lastCollectLogs"`
	LastError       string     `gorm:"type:varchar(500)" json:"lastError"`

	// 文件偏移量追踪（用于增量采集大文件）
	LastFileSize   int64  `gorm:"type:bigint;default:0" json:"lastFileSize"`   // 上次采集时文件大小
	LastFileOffset int64  `gorm:"type:bigint;default:0" json:"lastFileOffset"` // 上次读取的字节偏移量
	LastFileInode  uint64 `gorm:"type:bigint;default:0" json:"lastFileInode"`  // 文件inode（检测日志轮转）

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `gorm:"index" json:"deletedAt,omitempty"`
}

func (NginxSource) TableName() string {
	return "nginx_sources"
}

// ============== Dimension Tables ==============

// NginxDimIP IP 维度表 (含地理位置信息)
type NginxDimIP struct {
	ID        uint64 `gorm:"primarykey" json:"id"`
	IPAddress string `gorm:"type:varchar(50);uniqueIndex;not null" json:"ipAddress"`
	Country   string `gorm:"type:varchar(50)" json:"country"`
	Province  string `gorm:"type:varchar(50)" json:"province"`
	City      string `gorm:"type:varchar(50)" json:"city"`
	ISP       string `gorm:"type:varchar(100)" json:"isp"`
	IsBot     bool   `gorm:"type:tinyint;default:0" json:"isBot"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NginxDimIP) TableName() string {
	return "nginx_dim_ip"
}

// NginxDimURL URL 维度表
type NginxDimURL struct {
	ID            uint64 `gorm:"primarykey" json:"id"`
	URLHash       string `gorm:"type:varchar(64);uniqueIndex;not null" json:"urlHash"`
	URLPath       string `gorm:"type:varchar(2000)" json:"urlPath"`
	URLNormalized string `gorm:"type:varchar(500);index" json:"urlNormalized"` // 去除参数的规范化路径
	Host          string `gorm:"type:varchar(255);index" json:"host"`

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxDimURL) TableName() string {
	return "nginx_dim_url"
}

// NginxDimReferer Referer 维度表
type NginxDimReferer struct {
	ID            uint64 `gorm:"primarykey" json:"id"`
	RefererHash   string `gorm:"type:varchar(64);uniqueIndex;not null" json:"refererHash"`
	RefererURL    string `gorm:"type:varchar(2000)" json:"refererUrl"`
	RefererDomain string `gorm:"type:varchar(255);index" json:"refererDomain"`
	RefererType   string `gorm:"type:varchar(20)" json:"refererType"` // direct, search, social, other

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxDimReferer) TableName() string {
	return "nginx_dim_referer"
}

// NginxDimUserAgent User-Agent 维度表
type NginxDimUserAgent struct {
	ID             uint64 `gorm:"primarykey" json:"id"`
	UAHash         string `gorm:"type:varchar(64);uniqueIndex;not null" json:"uaHash"`
	UserAgent      string `gorm:"type:varchar(500)" json:"userAgent"`
	Browser        string `gorm:"type:varchar(50);index" json:"browser"`
	BrowserVersion string `gorm:"type:varchar(20)" json:"browserVersion"`
	OS             string `gorm:"type:varchar(50);index" json:"os"`
	OSVersion      string `gorm:"type:varchar(20)" json:"osVersion"`
	DeviceType     string `gorm:"type:varchar(20);index" json:"deviceType"` // desktop, mobile, tablet, bot
	IsBot          bool   `gorm:"type:tinyint;default:0;index" json:"isBot"`

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxDimUserAgent) TableName() string {
	return "nginx_dim_user_agent"
}

// ============== Fact Table ==============

// NginxFactAccessLog Nginx 访问日志事实表
type NginxFactAccessLog struct {
	ID        uint64    `gorm:"primarykey" json:"id"`
	SourceID  uint      `gorm:"index:idx_source_time;not null" json:"sourceId"`
	Timestamp time.Time `gorm:"type:datetime;index:idx_source_time;not null" json:"timestamp"`

	// 维度外键
	IPID      uint64 `gorm:"index" json:"ipId"`
	URLID     uint64 `gorm:"index" json:"urlId"`
	RefererID uint64 `gorm:"index" json:"refererId"`
	UAID      uint64 `gorm:"index" json:"uaId"`

	// 度量字段
	Method        string  `gorm:"type:varchar(20);index" json:"method"`
	Protocol      string  `gorm:"type:varchar(50)" json:"protocol"`
	Status        int     `gorm:"type:int;index" json:"status"`
	BodyBytesSent int64   `gorm:"type:bigint" json:"bodyBytesSent"`
	RequestTime   float64 `gorm:"type:decimal(10,3)" json:"requestTime"`
	UpstreamTime  float64 `gorm:"type:decimal(10,3)" json:"upstreamTime"`

	// K8s 特有字段
	IngressName string `gorm:"type:varchar(100)" json:"ingressName"`
	ServiceName string `gorm:"type:varchar(100)" json:"serviceName"`
	PodName     string `gorm:"type:varchar(100)" json:"podName"`

	// 扩展字段
	IsPV      bool   `gorm:"type:tinyint;default:1" json:"isPv"` // 是否为页面访问
	SessionID string `gorm:"type:varchar(64)" json:"sessionId"`  // 会话ID

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxFactAccessLog) TableName() string {
	return "nginx_fact_access_logs"
}

// ============== Legacy Table (保持兼容) ==============

// NginxAccessLog Nginx 访问日志 (旧表，保持兼容)
type NginxAccessLog struct {
	ID            uint64    `gorm:"primarykey" json:"id"`
	SourceID      uint      `gorm:"index;not null" json:"sourceId"`
	Timestamp     time.Time `gorm:"type:datetime;index;not null" json:"timestamp"`
	RemoteAddr    string    `gorm:"type:varchar(50);index" json:"remoteAddr"`
	RemoteUser    string    `gorm:"type:varchar(100)" json:"remoteUser"`
	Request       string    `gorm:"type:varchar(2000)" json:"request"`
	Method        string    `gorm:"type:varchar(20);index" json:"method"`
	URI           string    `gorm:"type:varchar(1000)" json:"uri"`
	Protocol      string    `gorm:"type:varchar(50)" json:"protocol"`
	Status        int       `gorm:"type:int;index" json:"status"`
	BodyBytesSent int64     `gorm:"type:bigint" json:"bodyBytesSent"`
	HTTPReferer   string    `gorm:"type:varchar(1000)" json:"httpReferer"`
	HTTPUserAgent string    `gorm:"type:varchar(500)" json:"httpUserAgent"`
	RequestTime   float64   `gorm:"type:decimal(10,3)" json:"requestTime"`
	UpstreamTime  float64   `gorm:"type:decimal(10,3)" json:"upstreamTime"`
	Host          string    `gorm:"type:varchar(255);index" json:"host"`

	// 地理位置字段
	Country  string `gorm:"type:varchar(50)" json:"country"`
	Province string `gorm:"type:varchar(50)" json:"province"`
	City     string `gorm:"type:varchar(50)" json:"city"`
	ISP      string `gorm:"type:varchar(100)" json:"isp"`

	// UA 解析字段
	Browser        string `gorm:"type:varchar(50);index" json:"browser"`
	BrowserVersion string `gorm:"type:varchar(20)" json:"browserVersion"`
	OS             string `gorm:"type:varchar(50);index" json:"os"`
	OSVersion      string `gorm:"type:varchar(20)" json:"osVersion"`
	DeviceType     string `gorm:"type:varchar(20);index" json:"deviceType"` // desktop, mobile, tablet, bot

	// K8s Ingress 特有字段
	IngressName string `gorm:"type:varchar(100)" json:"ingressName"`
	ServiceName string `gorm:"type:varchar(100)" json:"serviceName"`

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxAccessLog) TableName() string {
	return "nginx_access_logs"
}

// ============== Aggregation Tables ==============

// NginxAggHourly 小时聚合统计
type NginxAggHourly struct {
	ID       uint      `gorm:"primarykey" json:"id"`
	SourceID uint      `gorm:"uniqueIndex:idx_source_hour;not null" json:"sourceId"`
	Hour     time.Time `gorm:"type:datetime;uniqueIndex:idx_source_hour;not null" json:"hour"`

	// 基础指标
	TotalRequests   int64   `gorm:"type:bigint;default:0" json:"totalRequests"`
	PVCount         int64   `gorm:"type:bigint;default:0" json:"pvCount"`
	UniqueIPs       int64   `gorm:"type:bigint;default:0" json:"uniqueIps"`
	TotalBandwidth  int64   `gorm:"type:bigint;default:0" json:"totalBandwidth"`
	AvgResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"avgResponseTime"`
	MaxResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"maxResponseTime"`
	MinResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"minResponseTime"`

	// 状态码统计
	Status2xx int64 `gorm:"type:bigint;default:0" json:"status2xx"`
	Status3xx int64 `gorm:"type:bigint;default:0" json:"status3xx"`
	Status4xx int64 `gorm:"type:bigint;default:0" json:"status4xx"`
	Status5xx int64 `gorm:"type:bigint;default:0" json:"status5xx"`

	// 方法分布 (JSON)
	MethodDistribution string `gorm:"type:text" json:"methodDistribution"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NginxAggHourly) TableName() string {
	return "nginx_agg_hourly"
}

// NginxAggDaily 日聚合统计
type NginxAggDaily struct {
	ID       uint      `gorm:"primarykey" json:"id"`
	SourceID uint      `gorm:"uniqueIndex:idx_source_date;not null" json:"sourceId"`
	Date     time.Time `gorm:"type:date;uniqueIndex:idx_source_date;not null" json:"date"`

	// 基础指标
	TotalRequests   int64   `gorm:"type:bigint;default:0" json:"totalRequests"`
	PVCount         int64   `gorm:"type:bigint;default:0" json:"pvCount"`
	UniqueIPs       int64   `gorm:"type:bigint;default:0" json:"uniqueIps"`
	TotalBandwidth  int64   `gorm:"type:bigint;default:0" json:"totalBandwidth"`
	AvgResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"avgResponseTime"`
	MaxResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"maxResponseTime"`
	MinResponseTime float64 `gorm:"type:decimal(10,3);default:0" json:"minResponseTime"`

	// 状态码统计
	Status2xx int64 `gorm:"type:bigint;default:0" json:"status2xx"`
	Status3xx int64 `gorm:"type:bigint;default:0" json:"status3xx"`
	Status4xx int64 `gorm:"type:bigint;default:0" json:"status4xx"`
	Status5xx int64 `gorm:"type:bigint;default:0" json:"status5xx"`

	// Top N 数据 (JSON)
	TopURLs      string `gorm:"type:text" json:"topUrls"`
	TopIPs       string `gorm:"type:text" json:"topIps"`
	TopReferers  string `gorm:"type:text" json:"topReferers"`
	TopCountries string `gorm:"type:text" json:"topCountries"`
	TopBrowsers  string `gorm:"type:text" json:"topBrowsers"`
	TopDevices   string `gorm:"type:text" json:"topDevices"`

	// 每小时流量分布 (JSON)
	HourlyTraffic string `gorm:"type:text" json:"hourlyTraffic"`

	// 方法分布 (JSON)
	MethodDistribution string `gorm:"type:text" json:"methodDistribution"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NginxAggDaily) TableName() string {
	return "nginx_agg_daily"
}

// ============== Legacy Aggregation Tables (保持兼容) ==============

// NginxDailyStats Nginx 日统计数据 (旧表)
type NginxDailyStats struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	SourceID        uint      `gorm:"index;not null" json:"sourceId"`
	Date            time.Time `gorm:"type:date;index;not null" json:"date"`
	TotalRequests   int64     `gorm:"type:bigint;default:0" json:"totalRequests"`
	UniqueVisitors  int64     `gorm:"type:bigint;default:0" json:"uniqueVisitors"`
	TotalBandwidth  int64     `gorm:"type:bigint;default:0" json:"totalBandwidth"`
	AvgResponseTime float64   `gorm:"type:decimal(10,3);default:0" json:"avgResponseTime"`
	Status2xx       int64     `gorm:"type:bigint;default:0" json:"status2xx"`
	Status3xx       int64     `gorm:"type:bigint;default:0" json:"status3xx"`
	Status4xx       int64     `gorm:"type:bigint;default:0" json:"status4xx"`
	Status5xx       int64     `gorm:"type:bigint;default:0" json:"status5xx"`

	// Top 数据 (JSON)
	TopURIs       string `gorm:"type:text" json:"topURIs"`
	TopIPs        string `gorm:"type:text" json:"topIPs"`
	TopReferers   string `gorm:"type:text" json:"topReferers"`
	TopUserAgents string `gorm:"type:text" json:"topUserAgents"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NginxDailyStats) TableName() string {
	return "nginx_daily_stats"
}

// NginxHourlyStats Nginx 小时统计数据 (旧表)
type NginxHourlyStats struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	SourceID        uint      `gorm:"index;not null" json:"sourceId"`
	Hour            time.Time `gorm:"type:datetime;index;not null" json:"hour"`
	TotalRequests   int64     `gorm:"type:bigint;default:0" json:"totalRequests"`
	UniqueVisitors  int64     `gorm:"type:bigint;default:0" json:"uniqueVisitors"`
	TotalBandwidth  int64     `gorm:"type:bigint;default:0" json:"totalBandwidth"`
	AvgResponseTime float64   `gorm:"type:decimal(10,3);default:0" json:"avgResponseTime"`
	Status2xx       int64     `gorm:"type:bigint;default:0" json:"status2xx"`
	Status3xx       int64     `gorm:"type:bigint;default:0" json:"status3xx"`
	Status4xx       int64     `gorm:"type:bigint;default:0" json:"status4xx"`
	Status5xx       int64     `gorm:"type:bigint;default:0" json:"status5xx"`

	CreatedAt time.Time `json:"createdAt"`
}

func (NginxHourlyStats) TableName() string {
	return "nginx_hourly_stats"
}

// ============== Response/DTO Models ==============

// NginxRealTimeStats 实时统计数据 (内存中)
type NginxRealTimeStats struct {
	SourceID        uint      `json:"sourceId"`
	Timestamp       time.Time `json:"timestamp"`
	RequestsPerSec  float64   `json:"requestsPerSec"`
	BandwidthPerSec int64     `json:"bandwidthPerSec"`
	ActiveConns     int64     `json:"activeConns"`
	AvgResponseTime float64   `json:"avgResponseTime"`
	Status2xxRate   float64   `json:"status2xxRate"`
	Status4xxRate   float64   `json:"status4xxRate"`
	Status5xxRate   float64   `json:"status5xxRate"`
}

// OverviewStats 概况统计
type OverviewStats struct {
	TotalSources       int64            `json:"totalSources"`
	ActiveSources      int64            `json:"activeSources"`
	TodayRequests      int64            `json:"todayRequests"`
	TodayVisitors      int64            `json:"todayVisitors"`
	TodayBandwidth     int64            `json:"todayBandwidth"`
	TodayPV            int64            `json:"todayPv"`
	TodayErrorRate     float64          `json:"todayErrorRate"`
	AvgResponseTime    float64          `json:"avgResponseTime"`
	RequestsTrend      []TrendPoint     `json:"requestsTrend"`
	BandwidthTrend     []TrendPoint     `json:"bandwidthTrend"`
	StatusDistribution map[string]int64 `json:"statusDistribution"`
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Time  string `json:"time"`
	Value int64  `json:"value"`
}

// TopItem Top 统计项
type TopItem struct {
	Name    string  `json:"name"`
	Value   int64   `json:"value"`
	Percent float64 `json:"percent,omitempty"`
}

// GeoStats 地理位置统计
type GeoStats struct {
	Country  string  `json:"country"`
	Province string  `json:"province,omitempty"`
	City     string  `json:"city,omitempty"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

// BrowserStats 浏览器统计
type BrowserStats struct {
	Browser string  `json:"browser"`
	Version string  `json:"version,omitempty"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// DeviceStats 设备统计
type DeviceStats struct {
	DeviceType string  `json:"deviceType"`
	Count      int64   `json:"count"`
	Percent    float64 `json:"percent"`
}

// ResponseTimeStats 响应时间统计
type ResponseTimeStats struct {
	Avg float64 `json:"avg"`
	Max float64 `json:"max"`
	Min float64 `json:"min"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// TimeSeriesPoint 时间序列点
type TimeSeriesPoint struct {
	Time            string  `json:"time"`
	Requests        int64   `json:"requests"`
	Bandwidth       int64   `json:"bandwidth"`
	UniqueIPs       int64   `json:"uniqueIps"`
	AvgResponseTime float64 `json:"avgResponseTime"`
	ErrorRate       float64 `json:"errorRate"`
}

// AccessLogView 访问日志视图 (带维度信息)
type AccessLogView struct {
	ID            uint64    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	RemoteAddr    string    `json:"remoteAddr"`
	Country       string    `json:"country"`
	City          string    `json:"city"`
	Method        string    `json:"method"`
	URI           string    `json:"uri"`
	Host          string    `json:"host"`
	Protocol      string    `json:"protocol"`
	Status        int       `json:"status"`
	BodyBytesSent int64     `json:"bodyBytesSent"`
	RequestTime   float64   `json:"requestTime"`
	HTTPReferer   string    `json:"httpReferer"`
	Browser       string    `json:"browser"`
	OS            string    `json:"os"`
	DeviceType    string    `json:"deviceType"`
	IsBot         bool      `json:"isBot"`
}

// ============== Overview DTO Models ==============

// CoreMetrics 核心指标（今日/昨日/预计今日/昨日此时）
type CoreMetrics struct {
	Today        MetricSet `json:"today"`
	Yesterday    MetricSet `json:"yesterday"`
	PredictToday MetricSet `json:"predictToday"`
	YesterdayNow MetricSet `json:"yesterdayNow"`
}

// MetricSet 指标集合
type MetricSet struct {
	StatusHits  int64   `json:"statusHits"`
	PV          int64   `json:"pv"`
	UV          int64   `json:"uv"`
	RealtimeOps float64 `json:"realtimeOps"`
	PeakOps     float64 `json:"peakOps"`
	Status2xx   int64   `json:"status2xx"`
	Status3xx   int64   `json:"status3xx"`
	Status4xx   int64   `json:"status4xx"`
	Status5xx   int64   `json:"status5xx"`
}

// VisitorComparison 新老访客对比
type VisitorComparison struct {
	TodayNew       int64   `json:"todayNew"`
	TodayReturning int64   `json:"todayReturning"`
	TodayNewPct    float64 `json:"todayNewPct"`
	TodayRetPct    float64 `json:"todayRetPct"`
	YestNew        int64   `json:"yesterdayNew"`
	YestReturning  int64   `json:"yesterdayReturning"`
	YestNewPct     float64 `json:"yesterdayNewPct"`
	YestRetPct     float64 `json:"yesterdayRetPct"`
}

// OverviewTrendPoint 概况趋势数据点（含UV和PV）
type OverviewTrendPoint struct {
	Time string `json:"time"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

// RefererItem 来路项
type RefererItem struct {
	Domain   string `json:"domain"`
	Visitors int64  `json:"visitors"`
}

// PageItem 页面项
type PageItem struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

// ParsedLogEntry 解析后的日志条目
type ParsedLogEntry struct {
	Timestamp     time.Time
	RemoteAddr    string
	RemoteUser    string
	Method        string
	URI           string
	Protocol      string
	Status        int
	BodyBytesSent int64
	HTTPReferer   string
	HTTPUserAgent string
	RequestTime   float64
	UpstreamTime  float64
	Host          string

	// K8s 字段
	IngressName string
	ServiceName string
	PodName     string
}
