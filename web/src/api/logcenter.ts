import request from '@/utils/request'

const LOG_QUERY_TIMEOUT_MS = 5 * 60 * 1000
const LOG_HISTOGRAM_TIMEOUT_MS = 5 * 60 * 1000

export interface LogStorageCluster {
  id?: number
  name: string
  storageType: 'clickhouse' | string
  endpoints: string
  databaseName: string
  username?: string
  password?: string
  passwordConfigured?: boolean
  skipTlsVerify?: boolean
  timeout?: number
  queueMode?: 'direct' | 'redpanda' | string
  queueEndpoints?: string
  defaultRetentionDays?: number
  status?: string
  lastTestAt?: string
  lastError?: string
  initializedAt?: string
  enabled: boolean
  isPrimary?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface InternalLogQueryScope {
  assetTypes?: string[]
  assetIds?: number[]
  hostIds?: number[]
  clusterIds?: number[]
  namespaces?: string[]
  services?: string[]
  workloads?: string[]
  pods?: string[]
  containers?: string[]
  nodes?: string[]
  levels?: string[]
  environments?: string[]
}

export interface InternalLogQueryRequest {
  storageId?: number
  start: string
  end: string
  query?: string
  scope?: InternalLogQueryScope
  filters?: Array<{ field: string; operator: string; value: any }>
  filterLogic?: 'and' | 'or'
  sort?: 'asc' | 'desc'
  limit?: number
  cursor?: string
  skipHistory?: boolean
}

export interface LogItem {
  timestamp: string
  message: string
  level: string
  labels: Record<string, string>
  fields: Record<string, any>
  raw?: any
  contextSelected?: boolean
}

export interface LogQueryResponse {
  items: LogItem[]
  total: number
  durationMs: number
  histogram: Array<{ time: string; count: number }>
  fields: Array<{ name: string; type: string; count: number; sample: string }>
  nextCursor?: string
  hasMore?: boolean
  scannedBytes?: number
}

export interface LogHistogramResponse {
  histogram: Array<{ time: string; count: number }>
  durationMs: number
}

export interface InternalKubernetesResourceOption {
  clusterId: string
  namespace: string
  workloadKind: string
  workloadName: string
  podName: string
  containerName: string
  nodeName: string
}

export interface InternalLogResourceOptions {
  hostIds: string[]
  clusterIds: string[]
  environments: string[]
  services: string[]
  namespaces: string[]
  workloads: string[]
  pods: string[]
  containers: string[]
  nodes: string[]
  kubernetesResources: InternalKubernetesResourceOption[]
}

export interface LogExportTask {
  id: number
  userId: number
  storageId: number
  format: 'ndjson' | 'csv' | string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'expired' | string
  progress: number
  exportedRows: number
  maxRows: number
	attemptCount: number
	maxAttempts: number
	nextAttemptAt?: string
  fileName?: string
  fileSize: number
  errorMessage?: string
  startedAt?: string
  completedAt?: string
  expiresAt?: string
  createdAt: string
  updatedAt: string
}

export interface LogQueryTemplate {
  id?: number
  name: string
  category?: string
  datasourceType: string
  datasourceId?: number
  queryLanguage: string
  query: string
  index?: string
  timeRange?: string
  variables?: string
  description?: string
  isPublic?: boolean
  ownerId?: number
  sort?: number
  createdAt?: string
  updatedAt?: string
}

export interface LogSavedView {
  id?: number
  name: string
  userId?: number
  datasourceId?: number
  queryLanguage?: string
  query?: string
  index?: string
  filters?: string
  columns?: string
  timeRange?: string
  displayOptions?: string
  isPublic?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface LogAlertContext {
  id?: number
  ruleId?: number
  sampleQuery?: string
  sampleLimit?: number
  contextBefore?: number
  contextAfter?: number
  includeLogInNotice?: boolean
  jumpTimeWindow?: number
  highlightKeywords?: string
  createdAt?: string
  updatedAt?: string
}

export interface LogLibraryItem {
  id?: number
  datasourceId: number
  datasourceType: string
  itemType: string
  name: string
  displayName?: string
  description?: string
  owner?: string
  environment?: string
  retentionDays?: number
  docCount?: number
  storeSize?: string
  lastSyncAt?: string
  status?: string
  rawMeta?: string
  createdAt?: string
  updatedAt?: string
}

export interface LogFieldCatalog {
  id?: number
  datasourceId: number
  libraryItemId: number
  fieldName: string
  fieldType?: string
  displayName?: string
  sampleValue?: string
  isTimeField?: boolean
  isMessageField?: boolean
  isLevelField?: boolean
  isSensitive?: boolean
}

export interface LogIngestComponentStatus {
  name: string
  instanceId?: string
  status: string
  reachable: boolean
  queueMode?: string
  queueTopic?: string
  consumerGroup?: string
  brokerCount?: number
  queueHealthy?: boolean
  queueLag?: number
  startedAt?: string
  uptimeSeconds?: number
  acceptedBatches?: number
  acceptedRecords?: number
  rejectedBatches?: number
  duplicateBatches?: number
  failedBatches?: number
  queueDepth?: number
  queueCapacity?: number
  inflight?: number
  inflightLimit?: number
  publishLatencyMs?: number
  writeLatencyMs?: number
  deadletterBatches?: number
  queueLastError?: string
  lastSuccessAt?: string
  lastErrorAt?: string
  lastError?: string
  writerUrl?: string
  httpAddress?: string
  grpcAddress?: string
}

export interface LogIngestQueueStatus {
  enabled: boolean
  status: string
  reachable: boolean
  topic?: string
  consumerGroup?: string
  brokerCount?: number
  lag?: number
  deadletterBatches?: number
  lastError?: string
}

export interface LogIngestReadinessCheck {
  id: string
  title: string
  status: 'passed' | 'warning' | 'failed' | string
  description: string
  recommendation?: string
}

export interface LogIngestReadinessSummary {
  passed: number
  warnings: number
  failed: number
  total: number
}

export interface LogIngestStatus {
  mode: string
  gateway: LogIngestComponentStatus
  queue: LogIngestQueueStatus
  writer: LogIngestComponentStatus
  storage: {
    id?: number
    name?: string
    status: string
    reachable: boolean
    lastTestAt?: string
    lastError?: string
    initializedAt?: string
	retentionStatus?: 'healthy' | 'warning' | 'critical' | 'error' | string
	retentionError?: string
	expiredParts: number
	ttlLagSeconds: number
	ttlMergeActive: boolean
	ttlMergeProgress: number
	ttlMergeTimeoutSeconds: number
  }
  gatewayUrl: string
  writerUrl: string
  publicGatewayUrl: string
  readiness: LogIngestReadinessCheck[]
  readinessSummary: LogIngestReadinessSummary
  checkedAt: string
}

export interface LogPolicyTarget {
	targetType: 'host' | 'host_group' | 'cluster' | string
  targetId: number
  namespace?: string
  workloadKind?: string
  workloadName?: string
  labelSelector?: string
  containerInclude?: string[]
  containerExclude?: string[]
}

export interface LogCollectionPolicyPayload {
  name: string
	sourceMode: 'host' | 'kubernetes' | string
  description?: string
  paths: string[]
  excludePaths: string[]
  readFrom: 'latest' | 'beginning' | string
  encoding: string
  environment?: string
  service?: string
  stream?: string
  maxLineBytes: number
  parser: {
    type: 'raw' | 'json' | 'regex' | string
    pattern?: string
    messageField?: string
    timestampField?: string
    levelField?: string
    timestampLayout?: string
  }
  multiline: {
    enabled: boolean
    preset?: 'java' | 'go' | 'python' | 'custom' | string
    startPattern?: string
    maxLines?: number
    maxBytes?: number
    flushSeconds?: number
  }
  redaction: {
    configured: boolean
    enabled: boolean
    useDefaultRules: boolean
    sensitiveFields: string[]
    rules: Array<{
      name?: string
      target: 'field' | 'json_path' | 'regex' | string
      field?: string
      pattern?: string
      action: 'replace' | 'hash' | 'drop_field' | string
      replacement?: string
    }>
  }
  retentionPolicyId?: number
  retention: {
    defaultDays: number
    levelDays: Record<string, number>
  }
  retentionDays: number
  walMaxBytes: number
  targets: LogPolicyTarget[]
}

export interface LogRetentionPolicy {
  id: number
  boundPolicyCount?: number
  updatedPolicyCount?: number
  createdBy?: number
  updatedBy?: number
  createdAt?: string
  updatedAt?: string
  payload: {
    name: string
    description?: string
    storageId?: number
    defaultDays: number
    levelDays: Record<string, number>
    priority: number
    enabled: boolean
  }
}

export interface LogAccessPolicy {
  id: number
  createdBy?: number
  updatedBy?: number
  createdAt?: string
  updatedAt?: string
  payload: {
    name: string
    description?: string
    subjectType: 'user' | 'role'
    subjectId: number
    storageId?: number
    libraryItemPattern?: string
		scopeMode: 'all' | 'collection_policy'
		collectionPolicyIds: number[]
    allowedActions: Array<'query' | 'tail' | 'export' | string>
    deniedFields: string[]
    maskFields: string[]
    enabled: boolean
  }
}

export interface InternalLogAccessCapabilities {
  isAdmin: boolean
  canQuery: boolean
  canTail: boolean
  canExport: boolean
}

export interface LogCapacityEstimate {
  storageId: number
  storageName: string
  currentRows: number
  currentCompressedBytes: number
  currentStoredBytes: number
  currentUncompressedBytes: number
  compressionRatio: number
  logsLast24Hours: number
  rawBytesLast24Hours: number
  averageRecordBytes: number
  averageStoredRecordBytes: number
  dailyStoredBytes: number
  forecastBasis: 'stored_bytes_per_row' | 'raw_bytes_compression' | 'retention_average' | 'insufficient_data'
  safetyFactor: number
  retentionDays: number
  projectedStoredBytes: number
  recommendedBytes: number
  diskTotalBytes: number
  diskFreeBytes: number
  diskReservedBytes: number
  usableDiskBytes: number
  usableFreeBytes: number
  projectedUsagePercent: number
  daysUntilFull: number
	retention: LogRetentionHealth
}

export interface LogRetentionHealth {
	status: 'healthy' | 'warning' | 'critical' | string
	expiredRows: number
	expiredParts: number
	oldestExpiredAt?: string
	ttlLagSeconds: number
	ttlMergeActive: boolean
	ttlMergeProgress: number
	ttlMergeTimeoutSeconds: number
}

export interface LogPolicyTargetHost {
  id: number
  name: string
  ip: string
  groupId: number
  agentId?: string
  agentVersion?: string
  agentStatus?: string
}

export interface LogPolicyTargetGroup {
  id: number
  name: string
  parentId: number
  hostCount: number
}

export interface LogPolicyTargetCluster {
  id: number
  name: string
  alias?: string
  version?: string
  nodeCount: number
  status: number
}

export interface LogKubernetesWorkloadOption {
  namespace: string
  kind: string
  name: string
}

export interface LogKubernetesCollectorStatus {
  clusterId: number
  clusterName: string
  installed: boolean
  credentialConfigured: boolean
  tokenHint?: string
  desiredNodes: number
  readyNodes: number
  availableNodes: number
  instanceTotal: number
  instanceOnline: number
  policyCount: number
  image?: string
  imagePullPolicy?: string
  serverUrl?: string
  instances: Array<{
    instanceId: string
    nodeName: string
    podName: string
    status: string
    version: string
    configVersion: number
    lastHeartbeatAt?: string
    lastIngestAt?: string
    walBytes: number
    inputEps: number
    outputEps: number
    lastError?: string
  }>
  lastError?: string
}

export interface LogCollectionPolicy {
  id: number
  status: 'draft' | 'published' | 'disabled' | 'archived' | string
  version: number
  createdBy: number
  updatedBy: number
  createdAt: string
  updatedAt: string
  payload: LogCollectionPolicyPayload
  draftPayload?: LogCollectionPolicyPayload
  hasUnpublishedChanges?: boolean
  targetCount: number
  targetExpected: number
  instanceTotal: number
  instanceOnline: number
  instanceApplied: number
  instancePending: number
  errorInstances: number
	targetHosts: LogPolicyTargetHost[]
	targetClusters: LogPolicyTargetCluster[]
  collectorShutdown?: Array<{
    clusterId: number
    clusterName: string
    status: 'uninstalled' | 'skipped' | 'failed' | string
    activePolicyCount: number
    message: string
  }>
}

export interface LogPolicyRevision {
  id: number
  policyId: number
  policyName?: string
  version: number
  checksum: string
  changeSummary: string
  createdBy: number
  createdAt: string
}

export interface LogPolicyRevisionPage {
  total: number
  page: number
  pageSize: number
  data: LogPolicyRevision[]
}

export interface LogCollectorAssignment {
  id: number
  instanceId: string
  policyId: number
  policyVersion: number
  desiredState: string
  applyStatus: string
  appliedAt?: string
  lastError?: string
}

export interface LogCollectorInstanceView {
  status: string
  runtimeStatus?: string
  lifecycleStatus?: 'active' | 'idle' | 'retired' | string
  collectorCredentialStatus?: string
  activePolicyCount?: number
  instance: {
    id: number
    instanceId: string
    agentId: string
    mode: string
    hostId: number
    clusterId?: number
    hostname: string
    podName?: string
    namespace?: string
    nodeName?: string
    collectorType: string
    version: string
    configVersion: number
    status: string
    lastHeartbeatAt?: string
    lastIngestAt?: string
    walBytes: number
    inputEps: number
    outputEps: number
    droppedTotal: number
    retryTotal: number
    lastError?: string
  }
  assignments: LogCollectorAssignment[]
}

export interface LogOverviewPoint {
  name: string
  value: number
}

export interface LogOverviewTrendPoint {
  time: string
  count: number
  bytes: number
}

export interface LogOverview {
  logs24h: number
  bytes24h: number
  errors24h: number
  averageEps5m: number
  activeServices: number
  trend: LogOverviewTrendPoint[]
  levels: LogOverviewPoint[]
  topServices: LogOverviewPoint[]
  sources: LogOverviewPoint[]
  collectors: {
    total: number
    online: number
    errors: number
    inputEps: number
    outputEps: number
    walBytes: number
    droppedTotal: number
    retryTotal: number
    lastIngestAt?: string
  }
  storage: {
    id?: number
    name?: string
    status?: string
    available: boolean
    error?: string
  }
  trendRange?: '24h' | '30d' | '12m'
  updatedAt: string
}

export const getLogOverview = (params?: { trendRange?: '24h' | '30d' | '12m' }) =>
  request.get<LogOverview>('/api/v1/plugins/logcenter/overview', { params })

export const getLogIngestStatus = () => {
  return request.get<LogIngestStatus>('/api/v1/plugins/logcenter/ingest/status')
}

export const testLogIngest = (data?: { message?: string; level?: string }) => {
  return request.post('/api/v1/plugins/logcenter/ingest/test', data || {})
}

export const getLogCollectionPolicies = (params?: { keyword?: string; status?: string }) => {
  return request.get<LogCollectionPolicy[]>('/api/v1/plugins/logcenter/policies', { params })
}

export const getLogCollectionPolicy = (id: number) => {
  return request.get<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}`)
}

export const createLogCollectionPolicy = (data: LogCollectionPolicyPayload) => {
  return request.post<LogCollectionPolicy>('/api/v1/plugins/logcenter/policies', data)
}

export const updateLogCollectionPolicy = (id: number, data: LogCollectionPolicyPayload) => {
  return request.put<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}`, data)
}

export const deleteLogCollectionPolicy = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/policies/${id}`)
}

export const restoreLogCollectionPolicy = (id: number) => {
  return request.post<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}/restore`)
}

export const publishLogCollectionPolicy = (id: number, changeSummary?: string) => {
  return request.post<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}/publish`, { changeSummary })
}

export const disableLogCollectionPolicy = (id: number, uninstallCollectors = false) => {
  return request.post<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}/disable`, { uninstallCollectors }, { timeout: 90000 })
}

export const rollbackLogCollectionPolicy = (id: number, version: number) => {
  return request.post<LogCollectionPolicy>(`/api/v1/plugins/logcenter/policies/${id}/rollback/${version}`)
}

export const getLogPolicyTargetOptions = () => {
	return request.get<{ hosts: LogPolicyTargetHost[]; groups: LogPolicyTargetGroup[]; clusters: LogPolicyTargetCluster[] }>('/api/v1/plugins/logcenter/policies/target-options')
}

export const getLogKubernetesPolicyOptions = (clusterId: number) => {
  return request.get<{ namespaces: string[]; workloads: LogKubernetesWorkloadOption[] }>(`/api/v1/plugins/logcenter/kubernetes/clusters/${clusterId}/options`)
}

export const getLogKubernetesCollectorStatus = (clusterId: number) => {
  return request.get<LogKubernetesCollectorStatus>(`/api/v1/plugins/logcenter/kubernetes/clusters/${clusterId}/collector/status`)
}

export const generateLogKubernetesCollectorManifest = (clusterId: number, data?: { image?: string; serverUrl?: string }) => {
  return request.post<{ yaml: string; tokenHint: string; warning?: string }>(`/api/v1/plugins/logcenter/kubernetes/clusters/${clusterId}/collector/manifest`, data || {})
}

export const installLogKubernetesCollector = (clusterId: number, data?: { image?: string; serverUrl?: string }) => {
  return request.post(`/api/v1/plugins/logcenter/kubernetes/clusters/${clusterId}/collector/install`, data || {}, { timeout: 60000 })
}

export const uninstallLogKubernetesCollector = (clusterId: number) => {
  return request.delete(`/api/v1/plugins/logcenter/kubernetes/clusters/${clusterId}/collector`, { timeout: 60000 })
}

export const getLogPolicyRevisions = (params?: { policyId?: number; page?: number; pageSize?: number }) => {
  return request.get<LogPolicyRevisionPage>('/api/v1/plugins/logcenter/policies/revisions', { params })
}

export const getLogCollectorInstances = () => {
  return request.get<LogCollectorInstanceView[]>('/api/v1/plugins/logcenter/collectors/instances')
}

export const deleteLogCollectorInstance = (instanceId: string) => {
  return request.delete(`/api/v1/plugins/logcenter/collectors/instances/${encodeURIComponent(instanceId)}`)
}

export const restartLogCollectorInstance = (instanceId: string) => {
  return request.post(`/api/v1/plugins/logcenter/collectors/instances/${encodeURIComponent(instanceId)}/restart`)
}

export const getLogStorageClusters = (params?: any) => {
  return request.get<LogStorageCluster[]>('/api/v1/plugins/logcenter/storages', { params })
}

export const getLogRetentionPolicies = (params?: { storageId?: number }) => {
  return request.get<any, LogRetentionPolicy[]>('/api/v1/plugins/logcenter/retention-policies', { params })
}

export const createLogRetentionPolicy = (data: LogRetentionPolicy['payload']) => {
  return request.post<any, LogRetentionPolicy>('/api/v1/plugins/logcenter/retention-policies', data)
}

export const updateLogRetentionPolicy = (id: number, data: LogRetentionPolicy['payload']) => {
  return request.put<any, LogRetentionPolicy>(`/api/v1/plugins/logcenter/retention-policies/${id}`, data)
}

export const deleteLogRetentionPolicy = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/retention-policies/${id}`)
}

export const getLogAccessPolicies = () => {
  return request.get<any, LogAccessPolicy[]>('/api/v1/plugins/logcenter/access-policies')
}

export const getLogAccessPolicyOptions = () => {
	return request.get<any, {
		users: Array<{ id: number; username: string; realName?: string }>
		roles: Array<{ id: number; name: string; code: string }>
		collectionPolicies: Array<{ id: number; name: string; sourceMode: string; status: string; environment?: string }>
	}>('/api/v1/plugins/logcenter/access-policies/options')
}

export const createLogAccessPolicy = (data: LogAccessPolicy['payload']) => {
  return request.post<any, LogAccessPolicy>('/api/v1/plugins/logcenter/access-policies', data)
}

export const updateLogAccessPolicy = (id: number, data: LogAccessPolicy['payload']) => {
  return request.put<any, LogAccessPolicy>(`/api/v1/plugins/logcenter/access-policies/${id}`, data)
}

export const deleteLogAccessPolicy = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/access-policies/${id}`)
}

export const getLogCapacityEstimate = (params: { storageId?: number; retentionDays?: number }) => {
  return request.get<any, LogCapacityEstimate>('/api/v1/plugins/logcenter/capacity', { params, timeout: LOG_QUERY_TIMEOUT_MS })
}

export const getLogStorageCluster = (id: number) => {
  return request.get<LogStorageCluster>(`/api/v1/plugins/logcenter/storages/${id}`)
}

export const createLogStorageCluster = (data: Partial<LogStorageCluster>) => {
  return request.post<LogStorageCluster>('/api/v1/plugins/logcenter/storages', data)
}

export const updateLogStorageCluster = (id: number, data: Partial<LogStorageCluster>) => {
  return request.put<LogStorageCluster>(`/api/v1/plugins/logcenter/storages/${id}`, data)
}

export const deleteLogStorageCluster = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/storages/${id}`)
}

export const testLogStorageCluster = (id: number) => {
  return request.post(`/api/v1/plugins/logcenter/storages/${id}/test`)
}

export const initializeLogStorageCluster = (id: number) => {
  return request.post(`/api/v1/plugins/logcenter/storages/${id}/initialize`, undefined, { timeout: LOG_QUERY_TIMEOUT_MS })
}

export const queryInternalLogs = (data: InternalLogQueryRequest, signal?: AbortSignal) => {
  return request.post<LogQueryResponse>('/api/v1/plugins/logcenter/internal/query', data, {
    timeout: LOG_QUERY_TIMEOUT_MS,
    signal,
    headers: { 'X-Silent-Error': '1' },
  })
}

export const queryInternalLogHistogram = (data: InternalLogQueryRequest, signal?: AbortSignal) => {
  return request.post<LogHistogramResponse>('/api/v1/plugins/logcenter/internal/query/histogram', data, {
    timeout: LOG_HISTOGRAM_TIMEOUT_MS,
    signal,
    headers: { 'X-Silent-Error': '1' },
  })
}

export const queryInternalLogContext = (data: {
  storageId?: number
  timestamp: string
  message?: string
  level?: string
  fingerprint?: number
  sequence?: number
  labels?: Record<string, string>
  fields?: Record<string, any>
  beforeSeconds?: number
  afterSeconds?: number
  limit?: number
}) => {
  return request.post<LogQueryResponse>('/api/v1/plugins/logcenter/internal/query/context', data, {
    timeout: LOG_QUERY_TIMEOUT_MS,
    headers: { 'X-Silent-Error': '1' },
  })
}

export const queryInternalLogResourceOptions = (data: InternalLogQueryRequest, signal?: AbortSignal) => {
  return request.post<InternalLogResourceOptions>('/api/v1/plugins/logcenter/internal/query/options', data, {
    timeout: LOG_QUERY_TIMEOUT_MS,
    signal,
    headers: { 'X-Silent-Error': '1' },
  })
}

export const getInternalLogAssets = () => {
  return request.get<{ hosts: LogPolicyTargetHost[]; clusters: LogPolicyTargetCluster[] }>('/api/v1/plugins/logcenter/internal/assets')
}

export const getInternalLogAccessCapabilities = () => {
  return request.get<InternalLogAccessCapabilities>('/api/v1/plugins/logcenter/internal/access-capabilities', {
    headers: { 'X-Silent-Error': '1' },
  })
}

export const createInternalLogExport = (data: InternalLogQueryRequest & { format: 'ndjson' | 'csv'; maxRows: number }) => {
  return request.post<LogExportTask>('/api/v1/plugins/logcenter/internal/exports', data, { timeout: LOG_QUERY_TIMEOUT_MS })
}

export const getInternalLogExports = () => {
  return request.get<LogExportTask[]>('/api/v1/plugins/logcenter/internal/exports')
}

export const getInternalLogExport = (id: number) => {
  return request.get<LogExportTask>(`/api/v1/plugins/logcenter/internal/exports/${id}`)
}

export const downloadInternalLogExport = (id: number) => {
  return request.get(`/api/v1/plugins/logcenter/internal/exports/${id}/download`, {
    responseType: 'blob',
    timeout: LOG_QUERY_TIMEOUT_MS,
  })
}

export const streamInternalLogs = async (
  data: InternalLogQueryRequest,
  options: {
    signal: AbortSignal
    onReady?: (payload: any) => void
    onLogs: (payload: { items: LogItem[]; cursor?: string; receivedAt?: string }) => void
    onError?: (payload: { message?: string; retrying?: boolean }) => void
    onEnd?: (payload: { reason?: string }) => void
  },
) => {
  const token = localStorage.getItem('token')
  const response = await fetch('/api/v1/plugins/logcenter/internal/tail', {
    method: 'POST',
    credentials: 'include',
    signal: options.signal,
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(data),
  })
  const contentType = response.headers.get('content-type') || ''
  if (!response.ok || !contentType.includes('text/event-stream')) {
    const payload = await response.json().catch(() => ({}))
    const error = new Error(payload?.message || `Tail 连接失败（HTTP ${response.status}）`)
    const typedError = error as Error & { status?: number }
    typedError.status = response.status
    throw typedError
  }
  if (!response.body) throw new Error('浏览器不支持流式日志响应')
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  const dispatch = (block: string) => {
    let event = 'message'
    const dataLines: string[] = []
    for (const line of block.split(/\r?\n/)) {
      if (line.startsWith('event:')) event = line.slice(6).trim()
      if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
    }
    if (!dataLines.length) return
    const payload = JSON.parse(dataLines.join('\n'))
    if (event === 'ready') options.onReady?.(payload)
    if (event === 'logs') options.onLogs(payload)
    if (event === 'error') options.onError?.(payload)
    if (event === 'end') options.onEnd?.(payload)
  }
  while (!options.signal.aborted) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = blocks.pop() || ''
    for (const block of blocks) dispatch(block)
  }
}

export const getInternalLogFields = (params?: { storageId?: number }) => {
  return request.get('/api/v1/plugins/logcenter/internal/fields', { params })
}

export const getLogHistories = (params?: any) => {
  return request.get('/api/v1/plugins/logcenter/histories', { params })
}

export const deleteLogHistory = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/histories/${id}`)
}

export const batchDeleteLogHistories = (ids: number[]) => {
  return request.post('/api/v1/plugins/logcenter/histories/batch-delete', { ids })
}

export const getLogSavedViews = (params?: any) => {
  return request.get('/api/v1/plugins/logcenter/views', { params })
}

export const getLogSavedView = (id: number) => {
  return request.get<LogSavedView>(`/api/v1/plugins/logcenter/views/${id}`)
}

export const createLogSavedView = (data: LogSavedView) => {
  return request.post<LogSavedView>('/api/v1/plugins/logcenter/views', data)
}

export const updateLogSavedView = (id: number, data: LogSavedView) => {
  return request.put<LogSavedView>(`/api/v1/plugins/logcenter/views/${id}`, data)
}

export const deleteLogSavedView = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/views/${id}`)
}

export const getLogTemplates = (params?: any) => {
  return request.get('/api/v1/plugins/logcenter/templates', { params })
}

export const getLogTemplate = (id: number) => {
  return request.get<LogQueryTemplate>(`/api/v1/plugins/logcenter/templates/${id}`)
}

export const createLogTemplate = (data: LogQueryTemplate) => {
  return request.post<LogQueryTemplate>('/api/v1/plugins/logcenter/templates', data)
}

export const updateLogTemplate = (id: number, data: LogQueryTemplate) => {
  return request.put<LogQueryTemplate>(`/api/v1/plugins/logcenter/templates/${id}`, data)
}

export const deleteLogTemplate = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/templates/${id}`)
}

export const cloneLogTemplate = (id: number) => {
  return request.post(`/api/v1/plugins/logcenter/templates/${id}/clone`)
}

export const getLogAlertContext = (ruleId: number) => {
  return request.get<LogAlertContext>(`/api/v1/plugins/logcenter/alerts/${ruleId}/context`)
}

export const updateLogAlertContext = (ruleId: number, data: LogAlertContext) => {
  return request.put<LogAlertContext>(`/api/v1/plugins/logcenter/alerts/${ruleId}/context`, data)
}

export const getLogLibrary = (params?: any) => {
  return request.get('/api/v1/plugins/logcenter/library', { params })
}

export const updateLogLibraryItem = (id: number, data: Partial<LogLibraryItem>) => {
  return request.put<LogLibraryItem>(`/api/v1/plugins/logcenter/library/${id}`, data)
}

export const deleteLogLibraryItem = (id: number) => {
  return request.delete(`/api/v1/plugins/logcenter/library/${id}`)
}

export const getLogLibraryFields = (id: number) => {
  return request.get<LogFieldCatalog[]>(`/api/v1/plugins/logcenter/library/${id}/fields`)
}

export const updateLogLibraryField = (libraryId: number, fieldId: number, data: Partial<LogFieldCatalog>) => {
  return request.put<LogFieldCatalog>(`/api/v1/plugins/logcenter/library/${libraryId}/fields/${fieldId}`, data)
}
