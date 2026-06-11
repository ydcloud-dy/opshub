import request from '@/utils/request'

export type DataSourceType = 'prometheus' | 'victoriametrics' | 'loki' | 'elasticsearch'
export type DataSourceAuthType = 'none' | 'basic' | 'bearer'
export type ProbeProtocol = 'http' | 'icmp' | 'tcp' | 'ssl'

export interface MonitorDataSource {
  id?: number
  name: string
  type: DataSourceType
  url: string
  authType: DataSourceAuthType
  username?: string
  password?: string
  token?: string
  headers?: string
  timeout: number
  skipTlsVerify?: boolean
  enabled: boolean
  remoteWriteEnabled?: boolean
  remoteWriteUrl?: string
  remoteWriteAuthType?: DataSourceAuthType
  remoteWriteUsername?: string
  remoteWritePassword?: string
  remoteWriteToken?: string
  remoteWriteHeaders?: string
  remoteWriteSkipTlsVerify?: boolean
  status?: string
  lastTestAt?: string
  lastError?: string
  description?: string
  createdAt?: string
  updatedAt?: string
}

export interface DataSourceQueryRequest {
  query: string
  queryMode?: 'instant' | 'range'
  start?: string
  end?: string
  step?: string
  index?: string
  limit?: number
}

export interface MonitorDataSourceIndex {
  name: string
  type: 'index' | 'alias' | string
  status?: string
  docsCount?: string
  storeSize?: string
}

export interface MonitorAlertRule {
  id?: number
  name: string
  ruleGroupId?: number
  faultCenterId?: number
  dataSourceId: number
  dataSourceIds?: string
  dataSourceType?: string
  query: string
  queryMode: 'instant' | 'range'
  index?: string
  condition: 'gt' | 'gte' | 'lt' | 'lte' | 'eq' | 'neq'
  threshold: number
  severityRules?: string
  forSeconds: number
  evaluateInterval: number
  severity: 'info' | 'warning' | 'critical' | 'p0' | 'p1' | 'p2'
  enabled: boolean
  channelIds?: string
  notifyRecovery?: boolean
  repeatInterval?: number
  labels?: string
  annotations?: string
  detailTemplate?: string
  callbackQueries?: string
  effectiveTime?: string
  lastState?: string
  lastValue?: number
  lastEvalAt?: string
  pendingSince?: string
  firingSince?: string
  lastNotifyAt?: string
  lastError?: string
  createdAt?: string
  updatedAt?: string
}

export interface MonitorAlertRuleBatchUpdatePayload {
  ids: number[]
  ruleGroupId?: number
  dataSourceId?: number
  faultCenterId?: number
  enabled?: boolean
}

export interface MonitorAlertEvent {
  id: number
  ruleId: number
  ruleGroupId?: number
  faultCenterId?: number
  ruleName: string
  dataSourceId: number
  dataSourceName: string
  dataSourceType: string
  severity: string
  state: string
  value: number
  condition: string
  threshold: number
  message: string
  labels?: string
  annotations?: string
  fingerprint: string
  acknowledged?: boolean
  acknowledgedBy?: string
  acknowledgedAt?: string
  notifyStatus: string
  notifyError?: string
  escalated?: boolean
  escalatedAt?: string
  lastEscalateAt?: string
  escalateStatus?: string
  escalateError?: string
  startedAt: string
  endedAt?: string
  lastEvalAt: string
  createdAt: string
  updatedAt: string
}

export interface MonitorAlertEventStats {
  totalRules: number
  enabledRules: number
  firingRules: number
  pendingRules: number
  todayEvents: number
  unresolvedEvents: number
}

export interface MonitorAlertCallbackResult {
  key?: string
  name: string
  query: string
  renderedQuery?: string
  queryMode?: string
  dataSourceId?: number
  dataSourceName?: string
  dataSourceType?: string
  statusCode?: number
  duration?: number
  result?: any
  error?: string
}

export interface MonitorFaultCenter {
  id?: number
  name: string
  description?: string
  noticeObjectIds?: string
  noticeChannelIds?: string
  noticeRoutes?: string
  repeatNoticeInterval?: string
  recoverNotify?: boolean
  aggregationType?: string
  silenceEnabled?: boolean
  silenceRules?: string
  recoverWaitSeconds?: number
  upgradeEnabled?: boolean
  upgradableSeverities?: string
  upgradeStrategy?: string
  currentPreAlertNumber?: number
  currentAlertNumber?: number
  currentRecoverNumber?: number
  createdAt?: string
  updatedAt?: string
}

export interface MonitorRuleGroup {
  id?: number
  name: string
  description?: string
  sort?: number
  createdAt?: string
  updatedAt?: string
}

export interface MonitorNoticeTemplate {
  id?: number
  name: string
  noticeType: string
  description?: string
  template?: string
  templateFiring?: string
  templateRecover?: string
  enableFeiShuJsonCard?: boolean
  enabled?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface MonitorDutyUser {
  id?: number
  username: string
  realName?: string
  email?: string
  phone?: string
  notifyUserId?: string
  feishuUserId?: string
  feishuOpenId?: string
  dingtalkUserId?: string
  wecomUserId?: string
}

export interface MonitorDutyTable {
  id?: number
  name: string
  description?: string
  managerUserId?: number
  managerUsername?: string
  enabled?: boolean
  currentDutyUsers?: MonitorDutyUser[]
  createdAt?: string
  updatedAt?: string
}

export interface MonitorDutySchedule {
  id?: number
  dutyTableId: number
  dutyDate: string
  users: string
  status?: string
  createdAt?: string
  updatedAt?: string
}

export interface MonitorNoticeObject {
  id?: number
  uuid?: string
  name: string
  description?: string
  dutyTableId?: number
  dutyTableName?: string
  routes?: string
  enabled?: boolean
  lastStatus?: string
  currentDutyUsers?: MonitorDutyUser[]
  createdAt?: string
  updatedAt?: string
}

export interface MonitorQuerySuggestion {
  value: string
  insertText: string
  kind: string
  type?: string
  help?: string
}

export interface MonitorProbeTask {
  id?: number
  name: string
  protocol: ProbeProtocol
  endpoint: string
  method?: string
  headers?: string
  body?: string
  frequencySeconds: number
  timeoutSeconds: number
  icmpCount?: number
  icmpIntervalMs?: number
  dataSourceId?: number
  writeRuleEnabled?: boolean
  enabled: boolean
  status?: string
  lastStatus?: string
  lastProbeAt?: string
  nextProbeAt?: string
  lastDurationMs?: number
  lastError?: string
  lastWriteAt?: string
  lastWriteError?: string
  description?: string
  operator?: string
  dataSourceName?: string
  dataSourceRemoteOk?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface MonitorProbeResult {
  protocol: ProbeProtocol
  endpoint: string
  success: boolean
  statusCode?: number
  durationMs: number
  sslExpireAt?: string
  sslDaysLeft?: number
  message?: string
  error?: string
  checkedAt: string
}

export interface MonitorProbeRunSummary {
  success: boolean
  status: string
  durationMs: number
  remoteWriteOk?: boolean
  remoteWriteErr?: string
  results: MonitorProbeResult[]
}

export const getMonitorDataSources = (params?: { type?: string; keyword?: string }) => {
  return request.get('/api/v1/plugins/monitor/datasources', { params })
}

export const getMonitorDataSource = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/datasources/${id}`)
}

export const createMonitorDataSource = (data: MonitorDataSource) => {
  return request.post('/api/v1/plugins/monitor/datasources', data)
}

export const updateMonitorDataSource = (id: number, data: MonitorDataSource) => {
  return request.put(`/api/v1/plugins/monitor/datasources/${id}`, data)
}

export const deleteMonitorDataSource = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/datasources/${id}`)
}

export const testMonitorDataSource = (id: number) => {
  return request.post(`/api/v1/plugins/monitor/datasources/${id}/test`)
}

export const testMonitorDataSourceRemoteWrite = (id: number) => {
  return request.post(`/api/v1/plugins/monitor/datasources/${id}/remote-write-test`)
}

export const testMonitorDataSourceRemoteWriteConfig = (data: MonitorDataSource) => {
  return request.post('/api/v1/plugins/monitor/datasource-remote-write-test', data)
}

export const queryMonitorDataSource = (id: number, data: DataSourceQueryRequest) => {
  return request.post(`/api/v1/plugins/monitor/datasources/${id}/query`, data)
}

export const getMonitorDataSourceIndices = (
  id: number,
  params?: { keyword?: string; limit?: number }
) => {
  return request.get(`/api/v1/plugins/monitor/datasources/${id}/indices`, { params })
}

export const getMonitorDataSourceSuggestions = (
  id: number,
  params?: { keyword?: string; index?: string; limit?: number }
) => {
  return request.get(`/api/v1/plugins/monitor/datasources/${id}/suggestions`, { params })
}

export const getMonitorProbeTasks = (params?: { keyword?: string; protocol?: string; status?: string }) => {
  return request.get('/api/v1/plugins/monitor/probe-tasks', { params })
}

export const getMonitorProbeTask = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/probe-tasks/${id}`)
}

export const createMonitorProbeTask = (data: MonitorProbeTask) => {
  return request.post('/api/v1/plugins/monitor/probe-tasks', data)
}

export const updateMonitorProbeTask = (id: number, data: MonitorProbeTask) => {
  return request.put(`/api/v1/plugins/monitor/probe-tasks/${id}`, data)
}

export const deleteMonitorProbeTask = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/probe-tasks/${id}`)
}

export const runMonitorProbeTask = (id: number) => {
  return request.post(`/api/v1/plugins/monitor/probe-tasks/${id}/run`)
}

export const runMonitorInstantProbe = (data: {
  protocol: ProbeProtocol
  endpoint: string
  method?: string
  headers?: string
  body?: string
  timeoutSeconds?: number
  icmpCount?: number
  icmpIntervalMs?: number
}) => {
  return request.post('/api/v1/plugins/monitor/instant-probe', data)
}

export const getMonitorFaultCenters = (params?: { keyword?: string }) => {
  return request.get('/api/v1/plugins/monitor/fault-centers', { params })
}

export const getMonitorFaultCenter = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/fault-centers/${id}`)
}

export const createMonitorFaultCenter = (data: MonitorFaultCenter) => {
  return request.post('/api/v1/plugins/monitor/fault-centers', data)
}

export const updateMonitorFaultCenter = (id: number, data: MonitorFaultCenter) => {
  return request.put(`/api/v1/plugins/monitor/fault-centers/${id}`, data)
}

export const deleteMonitorFaultCenter = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/fault-centers/${id}`)
}

export const getMonitorFaultCenterSLO = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/fault-centers/${id}/slo`)
}

export const getMonitorNoticeTemplates = (params?: { keyword?: string; noticeType?: string }) => {
  return request.get('/api/v1/plugins/monitor/notice-templates', { params })
}

export const getMonitorNoticeTemplate = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/notice-templates/${id}`)
}

export const createMonitorNoticeTemplate = (data: MonitorNoticeTemplate) => {
  return request.post('/api/v1/plugins/monitor/notice-templates', data)
}

export const updateMonitorNoticeTemplate = (id: number, data: MonitorNoticeTemplate) => {
  return request.put(`/api/v1/plugins/monitor/notice-templates/${id}`, data)
}

export const deleteMonitorNoticeTemplate = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/notice-templates/${id}`)
}

export const getMonitorNoticeObjects = (params?: { keyword?: string; enabled?: boolean }) => {
  return request.get('/api/v1/plugins/monitor/notice-objects', { params })
}

export const getMonitorNoticeObject = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/notice-objects/${id}`)
}

export const createMonitorNoticeObject = (data: MonitorNoticeObject) => {
  return request.post('/api/v1/plugins/monitor/notice-objects', data)
}

export const updateMonitorNoticeObject = (id: number, data: MonitorNoticeObject) => {
  return request.put(`/api/v1/plugins/monitor/notice-objects/${id}`, data)
}

export const deleteMonitorNoticeObject = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/notice-objects/${id}`)
}

export const getMonitorAlertEventCallbacks = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/alert-events/${id}/callback-queries`)
}

export const testMonitorNoticeObject = (data: {
  noticeObject: MonitorNoticeObject
  routeIndex: number
  severity?: string
  state?: string
}) => {
  return request.post('/api/v1/plugins/monitor/notice-objects/test', data)
}

export const getMonitorDutyTables = (params?: { keyword?: string }) => {
  return request.get('/api/v1/plugins/monitor/duty-tables', { params })
}

export const getMonitorDutyTable = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/duty-tables/${id}`)
}

export const createMonitorDutyTable = (data: MonitorDutyTable) => {
  return request.post('/api/v1/plugins/monitor/duty-tables', data)
}

export const updateMonitorDutyTable = (id: number, data: MonitorDutyTable) => {
  return request.put(`/api/v1/plugins/monitor/duty-tables/${id}`, data)
}

export const deleteMonitorDutyTable = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/duty-tables/${id}`)
}

export const getMonitorDutySchedules = (params?: { dutyTableId?: number; month?: string }) => {
  return request.get('/api/v1/plugins/monitor/duty-schedules', { params })
}

export const upsertMonitorDutySchedule = (data: MonitorDutySchedule) => {
  return request.post('/api/v1/plugins/monitor/duty-schedules', data)
}

export const updateMonitorDutySchedule = (id: number, data: MonitorDutySchedule) => {
  return request.put(`/api/v1/plugins/monitor/duty-schedules/${id}`, data)
}

export const deleteMonitorDutySchedule = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/duty-schedules/${id}`)
}

export const getMonitorRuleGroups = () => {
  return request.get('/api/v1/plugins/monitor/rule-groups')
}

export const createMonitorRuleGroup = (data: MonitorRuleGroup) => {
  return request.post('/api/v1/plugins/monitor/rule-groups', data)
}

export const updateMonitorRuleGroup = (id: number, data: MonitorRuleGroup) => {
  return request.put(`/api/v1/plugins/monitor/rule-groups/${id}`, data)
}

export const deleteMonitorRuleGroup = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/rule-groups/${id}`)
}

export const getMonitorAlertRules = (params?: {
  dataSourceId?: number
  ruleGroupId?: number
  faultCenterId?: number
  dataSourceType?: string
  enabled?: boolean
  keyword?: string
}) => {
  return request.get('/api/v1/plugins/monitor/rules', { params })
}

export const getMonitorAlertRule = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/rules/${id}`)
}

export const createMonitorAlertRule = (data: MonitorAlertRule) => {
  return request.post('/api/v1/plugins/monitor/rules', data)
}

export const updateMonitorAlertRule = (id: number, data: MonitorAlertRule) => {
  return request.put(`/api/v1/plugins/monitor/rules/${id}`, data)
}

export const deleteMonitorAlertRule = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/rules/${id}`)
}

export const batchUpdateMonitorAlertRules = (data: MonitorAlertRuleBatchUpdatePayload) => {
  return request.post('/api/v1/plugins/monitor/rules/batch-update', data)
}

export const batchDeleteMonitorAlertRules = (ids: number[]) => {
  return request.post('/api/v1/plugins/monitor/rules/batch-delete', { ids })
}

export const exportMonitorAlertRules = (ids: number[]) => {
  return request.post('/api/v1/plugins/monitor/rules/export', { ids }, { responseType: 'blob' })
}

export const importMonitorAlertRules = (data: {
  content: string
  dataSourceId?: number
  ruleGroupId?: number
  faultCenterId?: number
  defaultSeverity?: string
}) => {
  return request.post('/api/v1/plugins/monitor/rules/import', data)
}

export const evaluateMonitorAlertRule = (id: number) => {
  return request.post(`/api/v1/plugins/monitor/rules/${id}/evaluate`)
}

export const getMonitorAlertEvents = (params?: {
  page?: number
  pageSize?: number
  ruleId?: number
  ruleGroupId?: number
  faultCenterId?: number
  dataSourceType?: string
  scope?: 'active' | 'history'
  keyword?: string
  state?: string
  severity?: string
  startDate?: string
  endDate?: string
}) => {
  return request.get('/api/v1/plugins/monitor/alert-events', { params })
}

export const getMonitorAlertEvent = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/alert-events/${id}`)
}

export const getMonitorAlertEventStats = () => {
  return request.get('/api/v1/plugins/monitor/alert-events/stats')
}

export const acknowledgeMonitorAlertEvent = (id: number, data?: { username?: string }) => {
  return request.post(`/api/v1/plugins/monitor/alert-events/${id}/ack`, data || {})
}

export const batchAcknowledgeMonitorAlertEvents = (ids: number[], data?: { username?: string }) => {
  return request.post('/api/v1/plugins/monitor/alert-events/batch-ack', { ids, ...(data || {}) })
}

export const silenceMonitorAlertEvent = (id: number, data?: { username?: string }) => {
  return request.post(`/api/v1/plugins/monitor/alert-events/${id}/silence`, data || {})
}

export const deleteMonitorAlertEvent = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/alert-events/${id}`)
}

export const batchDeleteMonitorAlertEvents = (ids: number[]) => {
  return request.post('/api/v1/plugins/monitor/alert-events/batch-delete', { ids })
}

export const importPrometheusRuleYaml = (data: {
  yaml: string
  dataSourceId: number
  ruleGroupId?: number
  faultCenterId?: number
  defaultSeverity?: string
}) => {
  return request.post('/api/v1/plugins/monitor/rules/import-prometheus-yaml', data)
}
