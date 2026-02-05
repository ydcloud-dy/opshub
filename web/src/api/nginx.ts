import request from '@/utils/request'

// ==================== 类型定义 ====================

export interface NginxSource {
  id?: number
  name: string
  type: 'host' | 'k8s_ingress'
  description?: string
  status: number
  hostId?: number
  logPath?: string
  logFormat?: string
  clusterId?: number
  namespace?: string
  ingressName?: string
  k8sPodSelector?: string
  k8sContainerName?: string
  logFormatConfig?: string
  geoEnabled?: boolean
  sessionEnabled?: boolean
  collectInterval: number
  retentionDays: number
  lastCollectAt?: string
  lastCollectLogs?: number
  lastError?: string
  createdAt?: string
  updatedAt?: string
}

export interface NginxAccessLog {
  id: number
  sourceId: number
  timestamp: string
  remoteAddr: string
  remoteUser?: string
  request: string
  method: string
  uri: string
  protocol: string
  status: number
  bodyBytesSent: number
  httpReferer?: string
  httpUserAgent?: string
  requestTime: number
  upstreamTime?: number
  host: string
  ingressName?: string
  serviceName?: string
  createdAt: string
}

export interface AccessLogView {
  id: number
  timestamp: string
  remoteAddr: string
  country?: string
  city?: string
  method: string
  uri: string
  host: string
  protocol: string
  status: number
  bodyBytesSent: number
  requestTime: number
  httpReferer?: string
  browser?: string
  os?: string
  deviceType?: string
  isBot?: boolean
}

export interface NginxDailyStats {
  id: number
  sourceId: number
  date: string
  totalRequests: number
  uniqueVisitors: number
  totalBandwidth: number
  avgResponseTime: number
  status2xx: number
  status3xx: number
  status4xx: number
  status5xx: number
  topURIs?: string
  topIPs?: string
  topReferers?: string
  topUserAgents?: string
  createdAt: string
  updatedAt: string
}

export interface NginxHourlyStats {
  id: number
  sourceId: number
  hour: string
  totalRequests: number
  uniqueVisitors: number
  totalBandwidth: number
  avgResponseTime: number
  status2xx: number
  status3xx: number
  status4xx: number
  status5xx: number
  createdAt: string
}

export interface OverviewStats {
  totalSources: number
  activeSources: number
  todayRequests: number
  todayVisitors: number
  todayBandwidth: number
  todayPv?: number
  todayErrorRate: number
  avgResponseTime?: number
  requestsTrend?: TrendPoint[]
  bandwidthTrend?: TrendPoint[]
  statusDistribution: Record<string, number>
}

export interface TrendPoint {
  time: string
  value: number
}

export interface TimeSeriesPoint {
  time: string
  requests: number
  bandwidth: number
  uniqueIps: number
  avgResponseTime: number
  errorRate: number
}

export interface GeoStats {
  country: string
  province?: string
  city?: string
  count: number
  percent: number
}

export interface BrowserStats {
  browser: string
  version?: string
  count: number
  percent: number
}

export interface DeviceStats {
  deviceType: string
  count: number
  percent: number
}

export interface TopIPWithGeo {
  ip: string
  country?: string
  province?: string
  city?: string
  count: number
}

// ==================== 数据源管理 ====================

// 获取数据源列表
export const getNginxSources = (params?: { page?: number; pageSize?: number; type?: string; status?: number }) => {
  return request.get('/api/v1/plugins/nginx/sources', { params })
}

// 获取数据源详情
export const getNginxSource = (id: number) => {
  return request.get(`/api/v1/plugins/nginx/sources/${id}`)
}

// 创建数据源
export const createNginxSource = (data: NginxSource) => {
  return request.post('/api/v1/plugins/nginx/sources', data)
}

// 更新数据源
export const updateNginxSource = (id: number, data: NginxSource) => {
  return request.put(`/api/v1/plugins/nginx/sources/${id}`, data)
}

// 删除数据源
export const deleteNginxSource = (id: number) => {
  return request.delete(`/api/v1/plugins/nginx/sources/${id}`)
}

// ==================== 概况统计 ====================

// 获取概况统计
export const getNginxOverview = () => {
  return request.get('/api/v1/plugins/nginx/overview')
}

// 获取请求趋势
export const getNginxRequestsTrend = (params?: { sourceId?: number; hours?: number }) => {
  return request.get('/api/v1/plugins/nginx/overview/trend', { params })
}

// ==================== 概况页面新增接口 ====================

// 核心指标类型
export interface CoreMetrics {
  today: MetricSet
  yesterday: MetricSet
  predictToday: MetricSet
  yesterdayNow: MetricSet
}

export interface MetricSet {
  statusHits: number
  pv: number
  uv: number
  realtimeOps: number
  peakOps: number
  status2xx: number
  status3xx: number
  status4xx: number
  status5xx: number
}

export interface VisitorComparison {
  todayNew: number
  todayReturning: number
  todayNewPct: number
  todayRetPct: number
  yesterdayNew: number
  yesterdayReturning: number
  yesterdayNewPct: number
  yesterdayRetPct: number
}

export interface OverviewTrendPoint {
  time: string
  pv: number
  uv: number
}

export interface RefererItem {
  domain: string
  visitors: number
}

export interface PageItem {
  path: string
  count: number
}

// 获取活跃访客
export const getActiveVisitors = (sourceId: number) => {
  return request.get('/api/v1/plugins/nginx/overview/active-visitors', { params: { sourceId } })
}

// 获取核心指标
export const getCoreMetrics = (sourceId: number) => {
  return request.get('/api/v1/plugins/nginx/overview/core-metrics', { params: { sourceId } })
}

// 获取概况趋势（UV+PV）
export const getOverviewTrend = (params: { sourceId: number; mode?: 'hour' | 'day'; date?: string }) => {
  return request.get('/api/v1/plugins/nginx/overview/overview-trend', { params })
}

// 获取新老访客对比
export const getNewVsReturning = (sourceId: number) => {
  return request.get('/api/v1/plugins/nginx/overview/new-vs-returning', { params: { sourceId } })
}

// 获取来路排行
export const getTopReferers = (params: { sourceId: number; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/overview/top-referers', { params })
}

// 获取受访页面排行
export const getTopPages = (params: { sourceId: number; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/overview/top-pages', { params })
}

// 获取入口页面排行
export const getTopEntryPages = (params: { sourceId: number; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/overview/top-entry-pages', { params })
}

// 获取概况地域分布
export const getOverviewGeo = (params: { sourceId: number; scope?: string }) => {
  return request.get('/api/v1/plugins/nginx/overview/geo', { params })
}

// 获取概况终端设备分布
export const getOverviewDevices = (sourceId: number) => {
  return request.get('/api/v1/plugins/nginx/overview/devices', { params: { sourceId } })
}

// ==================== 数据日报 ====================

// 获取日报数据
export const getNginxDailyReport = (params: { sourceId?: number; startDate?: string; endDate?: string }) => {
  return request.get('/api/v1/plugins/nginx/daily-report', { params })
}

// ==================== 访问明细 ====================

// 获取访问日志列表
export const getNginxAccessLogs = (params: {
  sourceId: number
  page?: number
  pageSize?: number
  startTime?: string
  endTime?: string
  remoteAddr?: string
  uri?: string
  status?: number
  method?: string
  host?: string
}) => {
  return request.get('/api/v1/plugins/nginx/access-logs', { params })
}

// 获取带维度信息的访问日志
export const getNginxLogs = (params: {
  sourceId: number
  page?: number
  pageSize?: number
  startTime?: string
  endTime?: string
  remoteAddr?: string
  uri?: string
  status?: number
  method?: string
  host?: string
}) => {
  return request.get('/api/v1/plugins/nginx/logs', { params })
}

// 获取 Top URI
export const getNginxTopURIs = (params: { sourceId: number; startTime?: string; endTime?: string; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/access-logs/top-uris', { params })
}

// 获取 Top IP
export const getNginxTopIPs = (params: { sourceId: number; startTime?: string; endTime?: string; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/access-logs/top-ips', { params })
}

// ==================== 统计分析 ====================

// 获取时间序列数据
export const getNginxTimeSeries = (params?: {
  sourceId?: number
  startTime?: string
  endTime?: string
  interval?: 'hour' | 'day'
}) => {
  return request.get('/api/v1/plugins/nginx/stats/timeseries', { params })
}

// 获取地理分布统计
export const getNginxGeoDistribution = (params?: {
  sourceId?: number
  startTime?: string
  endTime?: string
  level?: 'country' | 'province' | 'city'
}) => {
  return request.get('/api/v1/plugins/nginx/stats/geo', { params })
}

// 获取浏览器分布统计
export const getNginxBrowserDistribution = (params?: {
  sourceId?: number
  startTime?: string
  endTime?: string
}) => {
  return request.get('/api/v1/plugins/nginx/stats/browsers', { params })
}

// 获取设备分布统计
export const getNginxDeviceDistribution = (params?: {
  sourceId?: number
  startTime?: string
  endTime?: string
}) => {
  return request.get('/api/v1/plugins/nginx/stats/devices', { params })
}

// 获取 Top URLs (新接口)
export const getNginxTopURLs = (params: { sourceId: number; startTime?: string; endTime?: string; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/stats/top-urls', { params })
}

// 获取 Top IPs 带地理信息
export const getNginxTopIPsWithGeo = (params: { sourceId: number; startTime?: string; endTime?: string; limit?: number }) => {
  return request.get('/api/v1/plugins/nginx/stats/top-ips', { params })
}

// ==================== 日志采集 ====================

// 手动触发日志采集
export const collectNginxLogs = (sourceId?: number) => {
  const params = sourceId ? { sourceId } : {}
  return request.post('/api/v1/plugins/nginx/collect', null, { params })
}
