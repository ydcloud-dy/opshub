<template>
  <div class="log-center-page internal-query-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon"><el-icon><Search /></el-icon></div>
        <div>
          <h2>日志查询</h2>
          <p>检索由 OpsHub Log Agent 采集并写入 ClickHouse 的主机与 Kubernetes 日志</p>
        </div>
      </div>
      <div class="head-actions">
        <el-button :type="tailing ? 'danger' : 'default'" :disabled="!form.storageId" @click="toggleTail">
          <el-icon><SwitchButton v-if="tailing" /><VideoPlay v-else /></el-icon>
          {{ tailing ? '停止 Tail' : '实时 Tail' }}
        </el-button>
        <el-button :disabled="!form.storageId" @click="openExportDialog">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <el-button :loading="loading" :disabled="tailing" @click="runQuery">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <section class="panel query-panel">
      <div class="query-context-row">
        <div class="context-control storage-context">
          <span class="control-label">日志存储</span>
          <el-select v-model="form.storageId" class="storage-select" placeholder="选择 ClickHouse 存储" @change="loadFields">
            <el-option
              v-for="item in storages"
              :key="item.id"
              :label="`${item.name}${item.isPrimary ? '（主存储）' : ''}`"
              :value="item.id"
              :disabled="!item.enabled || !item.initializedAt"
            />
          </el-select>
          <el-tag size="small" :type="activeStorage?.status === 'healthy' ? 'success' : 'warning'">
            {{ activeStorage?.status === 'healthy' ? '连接正常' : '待检测' }}
          </el-tag>
          <el-tag v-if="routeHostId" size="small" type="info" closable @close="clearHostScope">{{ hostLabel(routeHostId) }}</el-tag>
          <el-tag v-if="routeClusterId" size="small" type="info" closable @close="clearClusterScope">{{ clusterLabel(routeClusterId) }}</el-tag>
        </div>
        <div class="context-control time-context">
          <span class="control-label">时间范围</span>
          <el-select v-model="quickRange" style="width: 112px" @change="applyQuickRange">
            <el-option label="最近 15 分钟" :value="15" />
            <el-option label="最近 1 小时" :value="60" />
            <el-option label="最近 6 小时" :value="360" />
            <el-option label="最近 24 小时" :value="1440" />
            <el-option label="最近 7 天" :value="10080" />
          </el-select>
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            :clearable="false"
            class="time-picker"
          />
        </div>
      </div>

      <div class="query-search-row">
        <el-input v-model="form.query" class="query-text-input" clearable :disabled="tailing" placeholder="搜索日志正文，输入 * 查询全部日志" @keyup.enter="runQuery">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="form.limit" class="page-size-select" :disabled="tailing" @change="runQuery">
          <el-option v-for="size in pageSizeOptions" :key="size" :label="`${size} 条/页`" :value="size" />
        </el-select>
        <el-button type="primary" class="primary-action" :loading="loading" :disabled="tailing" @click="runQuery">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
      </div>

      <div v-loading="resourceLoading" class="filter-section">
        <div class="filter-section-head">
          <div class="filter-title">
            <el-icon><Filter /></el-icon>
            <strong>范围筛选</strong>
            <el-tag v-if="activeFilterCount" size="small" type="info">{{ activeFilterCount }} 项已选</el-tag>
          </div>
          <el-button link class="advanced-toggle" @click="advancedOpen = !advancedOpen">
            {{ advancedOpen ? '收起高级筛选' : '更多筛选' }}
            <el-icon :class="{ rotated: advancedOpen }"><ArrowDown /></el-icon>
          </el-button>
        </div>

        <div class="primary-filter-grid">
          <div class="filter-control source-filter-control">
            <span>日志来源</span>
            <el-segmented v-model="sourceMode" :options="sourceModeOptions" :disabled="tailing" />
          </div>
          <div v-if="sourceMode !== 'kubernetes'" class="filter-control">
            <span>主机</span>
            <el-select v-model="scope.hostIds" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部主机">
              <el-option v-for="host in availableHosts" :key="host.id" :label="host.label" :value="host.id" />
            </el-select>
          </div>
          <div v-if="sourceMode !== 'host'" class="filter-control">
            <span>Kubernetes 集群</span>
            <el-select v-model="scope.clusterIds" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部集群">
              <el-option v-for="cluster in availableClusters" :key="cluster.id" :label="cluster.label" :value="cluster.id" />
            </el-select>
          </div>
          <div class="filter-control">
            <span>环境</span>
            <el-select v-model="scope.environments" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部环境">
              <el-option v-for="value in resourceOptions.environments" :key="value" :label="value" :value="value" />
            </el-select>
          </div>
          <div class="filter-control">
            <span>服务</span>
            <el-select v-model="scope.services" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部服务">
              <el-option v-for="value in resourceOptions.services" :key="value" :label="value" :value="value" />
            </el-select>
          </div>
          <div class="filter-control level-filter-control">
            <span>日志级别</span>
            <el-select v-model="scope.level" clearable :disabled="tailing" placeholder="全部级别">
              <el-option v-for="level in ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']" :key="level" :label="level" :value="level" />
            </el-select>
          </div>
        </div>

        <el-collapse-transition>
          <div v-show="advancedOpen" class="advanced-filter-area">
            <div class="advanced-filter-grid">
              <div v-if="sourceMode !== 'host'" class="filter-control">
                <span>Namespace</span>
                <el-select v-model="scope.namespaces" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部 Namespace">
                  <el-option v-for="value in resourceOptions.namespaces" :key="value" :label="value" :value="value" />
                </el-select>
              </div>
              <div v-if="sourceMode !== 'host'" class="filter-control">
                <span>工作负载</span>
                <el-select v-model="scope.workloads" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部工作负载">
                  <el-option v-for="value in resourceOptions.workloads" :key="value" :label="value" :value="value" />
                </el-select>
              </div>
              <div v-if="sourceMode !== 'host'" class="filter-control">
                <span>Pod</span>
                <el-select v-model="scope.pods" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部 Pod">
                  <el-option v-for="value in resourceOptions.pods" :key="value" :label="value" :value="value" />
                </el-select>
              </div>
              <div class="filter-control">
                <span>容器</span>
                <el-select v-model="scope.containers" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部容器">
                  <el-option v-for="value in resourceOptions.containers" :key="value" :label="value" :value="value" />
                </el-select>
              </div>
              <div class="filter-control">
                <span>节点</span>
                <el-select v-model="scope.nodes" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部节点">
                  <el-option v-for="value in resourceOptions.nodes" :key="value" :label="value" :value="value" />
                </el-select>
              </div>
            </div>

            <div class="condition-builder">
              <div class="condition-head">
                <strong>字段条件</strong>
                <el-button link :disabled="tailing" @click="addCondition"><el-icon><Plus /></el-icon>添加条件</el-button>
              </div>
              <div v-if="conditions.length" class="condition-list">
                <div v-for="condition in conditions" :key="condition.id" class="condition-row">
                  <el-select v-model="condition.field" filterable :disabled="tailing" placeholder="字段">
                    <el-option v-for="field in conditionFields" :key="field.name" :label="field.displayName || field.name" :value="field.name" />
                  </el-select>
                  <el-select v-model="condition.operator" :disabled="tailing" placeholder="操作符">
                    <el-option v-for="operator in conditionOperators" :key="operator.value" :label="operator.label" :value="operator.value" />
                  </el-select>
                  <el-input v-model="condition.value" :disabled="tailing" :placeholder="condition.operator === 'in' ? '多个值使用逗号分隔' : '条件值'" @keyup.enter="runQuery" />
                  <el-button circle text :disabled="tailing" title="删除条件" @click="removeCondition(condition.id)"><el-icon><Delete /></el-icon></el-button>
                </div>
              </div>
              <div v-else class="empty-condition">未添加字段条件</div>
            </div>
          </div>
        </el-collapse-transition>
      </div>
    </section>

    <section class="panel histogram-panel">
      <div class="section-head">
        <div>
          <strong>日志趋势</strong>
          <span>{{ totalCount.toLocaleString() }} 条趋势统计 · 查询耗时 {{ durationMs }} ms</span>
        </div>
        <span>{{ formatRangeLabel }}</span>
      </div>
      <div ref="chartRef" class="histogram-chart"></div>
    </section>

    <section class="query-result-grid">
      <aside class="panel field-panel">
        <div class="section-head compact">
          <strong>可用字段</strong>
          <span>{{ fieldOptions.length }}</span>
        </div>
        <el-input v-model="fieldKeyword" size="small" clearable placeholder="搜索字段" />
        <div class="field-list">
          <button
            v-for="field in filteredFields"
            :key="field.name"
            type="button"
            :class="{ active: selectedFields.includes(field.name) }"
            @click="toggleField(field.name)"
          >
            <span>{{ field.displayName || field.name }}</span>
            <em>{{ field.type }}</em>
          </button>
        </div>
      </aside>

      <div class="panel log-panel">
        <div class="section-head log-head">
          <div>
            <strong>日志明细</strong>
            <span v-if="tailing" class="tail-status"><i :class="{ connected: tailConnected }"></i>{{ tailConnected ? '实时接收中' : '正在连接' }} · 新增 {{ tailReceived.toLocaleString() }} 条 · 当前 {{ items.length.toLocaleString() }} 条</span>
            <span v-else-if="loading">正在查询 ClickHouse...</span>
            <span v-else>第 {{ currentPage }} 页 · 本页 {{ items.length.toLocaleString() }} 条</span>
          </div>
          <el-segmented v-model="form.sort" :options="sortOptions" :disabled="tailing" size="small" />
        </div>

        <el-alert v-if="errorMessage" type="error" :closable="false" show-icon :title="errorMessage" />
        <div v-loading="(loading && !items.length) || loadingMore" ref="logStreamRef" class="log-stream">
          <article
            v-for="(item, index) in items"
            :key="logKey(item, index)"
            class="log-row"
            :class="{ expanded: expandedRows.has(logKey(item, index)) }"
          >
            <div class="log-line">
              <button class="expand-button" type="button" title="展开日志详情" @click="toggleExpanded(item, index)">
                <el-icon><ArrowRight /></el-icon>
              </button>
              <time>{{ formatTimestamp(item.timestamp) }}</time>
              <span class="level-badge" :class="levelClass(item.level)">{{ levelText(item.level) }}</span>
              <p v-html="highlightMessage(item.message)"></p>
              <div class="row-actions">
                <el-button link size="small" @click="openContext(item)">上下文</el-button>
                <el-button link size="small" @click="copyLog(item)">复制</el-button>
              </div>
            </div>
            <div v-if="expandedRows.has(logKey(item, index))" class="log-detail">
              <div class="detail-section">
                <strong>索引字段</strong>
                <dl>
                  <template v-for="entry in detailEntries(item)" :key="entry.key">
                    <dt>{{ entry.key }}</dt><dd>{{ formatValue(entry.value) }}</dd>
                  </template>
                </dl>
              </div>
              <div class="detail-section message-section">
                <strong>日志正文</strong>
                <pre>{{ item.message }}</pre>
              </div>
            </div>
          </article>

          <el-empty v-if="!loading && !items.length && !errorMessage" description="当前条件下没有日志" :image-size="88" />
        </div>

        <div v-if="items.length" class="result-footer">
          <span v-if="tailing">已保留当前结果，最多显示最近 2,000 条</span>
          <template v-else>
            <span>共 {{ totalCount.toLocaleString() }} 条 · 当前 {{ pageStart.toLocaleString() }}-{{ pageEnd.toLocaleString() }} 条</span>
            <div class="page-actions">
              <el-button :disabled="currentPage <= 1 || loadingMore" @click="goPreviousPage">上一页</el-button>
              <span>第 {{ currentPage }} / {{ totalPages }} 页</span>
              <el-button :disabled="!hasMore || loadingMore" @click="goNextPage">下一页</el-button>
            </div>
          </template>
        </div>
      </div>
    </section>

    <el-drawer v-model="contextVisible" title="日志上下文" size="70%" destroy-on-close @opened="centerContextLog">
      <div class="context-toolbar">
        <span>当前日志前后 5 分钟</span>
        <el-tag size="small" :type="levelTagType(activeLog?.level)">{{ levelText(activeLog?.level) }}</el-tag>
      </div>
      <div v-loading="contextLoading" ref="contextListRef" class="context-list">
        <div
          v-for="(item, index) in contextItems"
          :key="`context-${index}-${logKey(item, index)}`"
          :data-active="isActiveContext(item) ? 'true' : 'false'"
          class="context-row"
          :class="{ active: isActiveContext(item) }"
        >
          <time>{{ formatTimestamp(item.timestamp) }}</time>
          <span class="level-badge" :class="levelClass(item.level)">{{ levelText(item.level) }}</span>
          <pre>{{ item.message }}</pre>
        </div>
        <el-empty v-if="!contextLoading && !contextItems.length" description="未查询到上下文日志" />
      </div>
    </el-drawer>

    <el-dialog v-model="exportVisible" title="异步导出日志" width="560px" destroy-on-close>
      <div v-if="!exportTask" class="export-form">
        <el-form label-position="top">
          <el-form-item label="文件格式">
            <el-segmented v-model="exportForm.format" :options="exportFormatOptions" />
          </el-form-item>
          <el-form-item label="最大导出条数">
            <el-input-number v-model="exportForm.maxRows" :min="1000" :max="1000000" :step="10000" controls-position="right" />
          </el-form-item>
        </el-form>
      </div>
      <div v-else class="export-progress">
        <div class="export-status-line">
          <el-tag :type="exportStatusType(exportTask.status)">{{ exportStatusText(exportTask.status) }}</el-tag>
          <span>{{ exportTask.exportedRows.toLocaleString() }} / {{ exportTask.maxRows.toLocaleString() }} 条</span>
        </div>
        <el-progress :percentage="exportTask.progress" :status="exportTask.status === 'failed' ? 'exception' : exportTask.status === 'completed' ? 'success' : undefined" />
        <dl>
          <dt>文件</dt><dd>{{ exportTask.fileName || '生成中' }}</dd>
          <dt>大小</dt><dd>{{ formatBytes(exportTask.fileSize) }}</dd>
          <dt>过期时间</dt><dd>{{ exportTask.expiresAt ? new Date(exportTask.expiresAt).toLocaleString() : '-' }}</dd>
        </dl>
        <el-alert v-if="exportTask.errorMessage" type="error" :closable="false" :title="exportTask.errorMessage" />
      </div>
      <template #footer>
        <el-button @click="exportVisible = false">关闭</el-button>
        <el-button v-if="!exportTask" type="primary" class="primary-action" :loading="exportCreating" @click="createExport">创建导出任务</el-button>
        <el-button v-else-if="exportTask.status === 'completed'" type="primary" class="primary-action" @click="downloadExport">下载文件</el-button>
        <el-button v-else-if="exportTask.status === 'failed' || exportTask.status === 'expired'" @click="resetExport">重新创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowRight, Delete, Download, Filter, Plus, Refresh, Search, SwitchButton, VideoPlay } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import {
  createInternalLogExport,
  downloadInternalLogExport,
  getInternalLogAssets,
  getInternalLogExport,
  getInternalLogFields,
  getLogStorageClusters,
  queryInternalLogContext,
  queryInternalLogHistogram,
  queryInternalLogResourceOptions,
  queryInternalLogs,
  streamInternalLogs,
  type InternalLogQueryRequest,
  type InternalLogResourceOptions,
  type LogExportTask,
  type LogItem,
  type LogPolicyTargetCluster,
  type LogPolicyTargetHost,
  type LogQueryResponse,
  type LogStorageCluster,
} from '@/api/logcenter'

type FieldOption = { name: string; type: string; displayName?: string }
type DetailEntry = { key: string; value: any }
type ConditionRow = { id: number; field: string; operator: string; value: string }
type AssetOption = { id: number; label: string }

const pageSizeOptions = [50, 100, 200, 500]
const route = useRoute()
const routeNumberList = (plural: string, singular: string) => String(route.query[plural] || route.query[singular] || '').split(',').map(Number).filter(Boolean)
const routeStringList = (name: string) => String(route.query[name] || '').split(',').map(item => item.trim()).filter(Boolean)
const routeHostIds = ref(routeNumberList('hostIds', 'hostId'))
const routeClusterIds = ref(routeNumberList('clusterIds', 'clusterId'))
const routeHostId = ref(routeHostIds.value[0] || 0)
const routeClusterId = ref(routeClusterIds.value[0] || 0)
const routeStorageId = Number(route.query.storageId || 0)
const routePageSize = Number(route.query.pageSize || 0)
const routeSort = String(route.query.sort || '').toLowerCase() === 'asc' ? 'asc' : 'desc'
const routeSourceMode = ['host', 'kubernetes'].includes(String(route.query.sourceMode)) ? String(route.query.sourceMode) as 'host' | 'kubernetes' : undefined
const routeStart = route.query.start ? new Date(String(route.query.start)) : undefined
const routeEnd = route.query.end ? new Date(String(route.query.end)) : undefined
const chartRef = ref<HTMLElement>()
const contextListRef = ref<HTMLElement>()
const logStreamRef = ref<HTMLElement>()
const storages = ref<LogStorageCluster[]>([])
const assetHosts = ref<LogPolicyTargetHost[]>([])
const assetClusters = ref<LogPolicyTargetCluster[]>([])
const fieldOptions = ref<FieldOption[]>([])
const resourceOptions = ref<InternalLogResourceOptions>({
  hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [],
})
const items = ref<LogItem[]>([])
const histogram = ref<Array<{ time: string; count: number }>>([])
const loading = ref(false)
const loadingMore = ref(false)
const resourceLoading = ref(false)
const contextLoading = ref(false)
const errorMessage = ref('')
const nextCursor = ref('')
const hasMore = ref(false)
const currentPage = ref(1)
const pageCursors = ref<string[]>([''])
const durationMs = ref(0)
const quickRange = ref(15)
const timeRange = ref<[Date, Date]>(routeStart && routeEnd && !Number.isNaN(routeStart.getTime()) && !Number.isNaN(routeEnd.getTime()) ? [routeStart, routeEnd] : [new Date(Date.now() - 15 * 60 * 1000), new Date()])
const fieldKeyword = ref('')
const selectedFields = ref<string[]>(['service', 'namespace', 'podName', 'containerName', 'filePath'])
const expandedRows = ref(new Set<string>())
const contextVisible = ref(false)
const contextItems = ref<LogItem[]>([])
const activeLog = ref<LogItem>()
const sourceMode = ref<'all' | 'host' | 'kubernetes'>(routeSourceMode || (routeHostId.value ? 'host' : routeClusterId.value ? 'kubernetes' : 'all'))
const conditions = ref<ConditionRow[]>([])
const tailing = ref(false)
const tailConnected = ref(false)
const tailReceived = ref(0)
const exportVisible = ref(false)
const exportCreating = ref(false)
const exportTask = ref<LogExportTask>()
const exportForm = reactive({ format: 'ndjson' as 'ndjson' | 'csv', maxRows: 100000 })
let chart: echarts.ECharts | undefined
let queryController: AbortController | undefined
let histogramController: AbortController | undefined
let resourceController: AbortController | undefined
let tailController: AbortController | undefined
let exportTimer: number | undefined
let conditionSequence = 0

const form = reactive({
  storageId: undefined as number | undefined,
  query: '*',
  limit: pageSizeOptions.includes(routePageSize) ? routePageSize : 200,
  sort: routeSort as 'asc' | 'desc',
})
const scope = reactive({
  hostIds: [...routeHostIds.value] as number[],
  clusterIds: [...routeClusterIds.value] as number[],
  environments: routeStringList('environments'), services: routeStringList('services'), namespaces: routeStringList('namespaces'), workloads: routeStringList('workloads'),
  pods: routeStringList('pods'), containers: routeStringList('containers'), nodes: routeStringList('nodes'), level: '',
})
const advancedOpen = ref(Boolean(scope.namespaces.length || scope.workloads.length || scope.pods.length || scope.containers.length || scope.nodes.length))
const sortOptions = [{ label: '最新优先', value: 'desc' }, { label: '最早优先', value: 'asc' }]
const sourceModeOptions = [{ label: '全部来源', value: 'all' }, { label: '主机', value: 'host' }, { label: 'Kubernetes', value: 'kubernetes' }]
const conditionOperators = [
  { label: '等于', value: 'eq' }, { label: '不等于', value: 'neq' }, { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' }, { label: '属于', value: 'in' },
]
const exportFormatOptions = [{ label: 'NDJSON', value: 'ndjson' }, { label: 'CSV', value: 'csv' }]
const activeStorage = computed(() => storages.value.find(item => item.id === form.storageId))
const totalCount = computed(() => histogram.value.reduce((sum, item) => sum + Number(item.count || 0), 0))
const totalPages = computed(() => Math.max(1, totalCount.value ? Math.ceil(totalCount.value / form.limit) : currentPage.value + (hasMore.value ? 1 : 0)))
const pageStart = computed(() => items.value.length ? (currentPage.value - 1) * form.limit + 1 : 0)
const pageEnd = computed(() => items.value.length ? pageStart.value + items.value.length - 1 : 0)
const conditionFields = computed(() => fieldOptions.value.filter(item => item.name !== 'timestamp'))
const activeFilterCount = computed(() => (
  (sourceMode.value === 'all' ? 0 : 1)
  + scope.hostIds.length + scope.clusterIds.length + scope.environments.length + scope.services.length
  + scope.namespaces.length + scope.workloads.length + scope.pods.length + scope.containers.length + scope.nodes.length
  + (scope.level ? 1 : 0)
  + conditions.value.filter(item => item.field && item.value.trim()).length
))
const filteredFields = computed(() => {
  const keyword = fieldKeyword.value.trim().toLowerCase()
  return keyword ? fieldOptions.value.filter(item => `${item.name} ${item.displayName || ''}`.toLowerCase().includes(keyword)) : fieldOptions.value
})
const formatRangeLabel = computed(() => `${timeRange.value[0].toLocaleString()} - ${timeRange.value[1].toLocaleString()}`)
const availableHosts = computed<AssetOption[]>(() => mergeAssetOptions(
  assetHosts.value.map(item => ({ id: Number(item.id), label: `${item.name || item.ip || `主机 #${item.id}`}${item.ip ? ` · ${item.ip}` : ''}` })),
  resourceOptions.value.hostIds,
  '主机',
))
const availableClusters = computed<AssetOption[]>(() => mergeAssetOptions(
  assetClusters.value.map(item => ({ id: Number(item.id), label: item.alias || item.name || `集群 #${item.id}` })),
  resourceOptions.value.clusterIds,
  '集群',
))

const buildPayload = (cursor = '', skipHistory = false): InternalLogQueryRequest => {
  const queryFilters: InternalLogQueryRequest['filters'] = []
  if (scope.level) queryFilters.push({ field: 'level', operator: 'eq', value: scope.level })
  if (scope.nodes.length) queryFilters.push({ field: 'nodeName', operator: 'in', value: [...scope.nodes] })
  for (const condition of conditions.value) {
    const value = condition.value.trim()
    if (!condition.field || !value) continue
    queryFilters.push({
      field: condition.field,
      operator: condition.operator,
      value: condition.operator === 'in' ? value.split(/[,，]/).map(item => item.trim()).filter(Boolean) : value,
    })
  }
  return {
    storageId: form.storageId,
    start: timeRange.value[0].toISOString(),
    end: timeRange.value[1].toISOString(),
    query: form.query.trim() || '*',
    scope: {
      assetTypes: sourceMode.value === 'all' ? [] : [sourceMode.value],
      hostIds: sourceMode.value === 'kubernetes' ? [] : [...scope.hostIds],
      clusterIds: sourceMode.value === 'host' ? [] : [...scope.clusterIds],
      environments: [...scope.environments], services: [...scope.services], namespaces: sourceMode.value === 'host' ? [] : [...scope.namespaces],
      workloads: sourceMode.value === 'host' ? [] : [...scope.workloads], pods: sourceMode.value === 'host' ? [] : [...scope.pods],
      containers: [...scope.containers],
    },
    filters: queryFilters,
    sort: form.sort,
    limit: form.limit,
    cursor,
    skipHistory,
  }
}

const mergeAssetOptions = (items: AssetOption[], ids: string[], prefix: string) => {
  const result = new Map<number, string>()
  for (const item of items) result.set(item.id, item.label)
  for (const value of ids) {
    const id = Number(value)
    if (id && !result.has(id)) result.set(id, `${prefix} #${id}`)
  }
  return Array.from(result, ([id, label]) => ({ id, label })).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
}
const hostLabel = (id: number) => availableHosts.value.find(item => item.id === id)?.label || `主机 #${id}`
const clusterLabel = (id: number) => availableClusters.value.find(item => item.id === id)?.label || `集群 #${id}`
const clearHostScope = () => { scope.hostIds = scope.hostIds.filter(id => id !== routeHostId.value); routeHostId.value = 0 }
const clearClusterScope = () => { scope.clusterIds = scope.clusterIds.filter(id => id !== routeClusterId.value); routeClusterId.value = 0 }

const loadInitialData = async () => {
  const [storageResult, assetResult] = await Promise.all([
    getLogStorageClusters({ enabled: true }) as any,
    getInternalLogAssets() as any,
  ])
  storages.value = storageResult || []
  assetHosts.value = assetResult?.hosts || []
  assetClusters.value = assetResult?.clusters || []
  const available = storages.value.find(item => item.isPrimary && item.initializedAt) || storages.value.find(item => item.initializedAt)
  form.storageId = storages.value.find(item => item.id === routeStorageId)?.id || available?.id
  if (route.query.q) form.query = String(route.query.q)
  await loadFields()
  if (route.query.filters) {
    try {
      const filters = JSON.parse(String(route.query.filters)) as Array<{ field?: string; operator?: string; value?: any }>
      conditions.value = filters.filter(item => item.field && !['level', 'nodeName'].includes(String(item.field))).map(item => ({ id: ++conditionSequence, field: String(item.field), operator: String(item.operator || 'eq'), value: Array.isArray(item.value) ? item.value.join(',') : String(item.value ?? '') }))
      const levelFilter = filters.find(item => item.field === 'level')
      if (levelFilter) scope.level = String(levelFilter.value || '')
      const nodeFilter = filters.find(item => item.field === 'nodeName')
      if (nodeFilter) scope.nodes = Array.isArray(nodeFilter.value) ? nodeFilter.value.map(String) : [String(nodeFilter.value || '')].filter(Boolean)
      if (conditions.value.length || scope.nodes.length) advancedOpen.value = true
    } catch {}
  }
  if (form.storageId) await runQuery()
}

const loadFields = async () => {
  if (!form.storageId) { fieldOptions.value = []; return }
  fieldOptions.value = (await getInternalLogFields({ storageId: form.storageId }) as any) || []
}

const loadResourceOptions = async () => {
  if (!form.storageId) return
  resourceController?.abort()
  resourceController = new AbortController()
  resourceLoading.value = true
  try {
    const result = await queryInternalLogResourceOptions(buildPayload('', true), resourceController.signal) as any
    resourceOptions.value = {
      hostIds: result?.hostIds || [], clusterIds: result?.clusterIds || [], environments: result?.environments || [],
      services: result?.services || [], namespaces: result?.namespaces || [], workloads: result?.workloads || [],
      pods: result?.pods || [], containers: result?.containers || [], nodes: result?.nodes || [],
    }
  } catch (error: any) {
    if (error?.name !== 'CanceledError' && error?.name !== 'AbortError') console.warn('读取日志资源选项失败', error)
  } finally { resourceLoading.value = false }
}

const runQuery = async () => {
  if (!form.storageId) { ElMessage.warning('请先在日志库中配置并初始化 ClickHouse'); return }
  if (tailing.value) return
  queryController?.abort(); histogramController?.abort()
  queryController = new AbortController(); histogramController = new AbortController()
  loading.value = true; errorMessage.value = ''; expandedRows.value = new Set()
  try {
    const [queryResult, histogramResult] = await Promise.all([
      queryInternalLogs(buildPayload(), queryController.signal) as any,
      queryInternalLogHistogram(buildPayload('', true), histogramController.signal) as any,
    ])
    const result = queryResult as LogQueryResponse
    pageCursors.value = ['']
    applyPageResult(result, 1)
    durationMs.value = Number(result.durationMs || 0)
    histogram.value = histogramResult?.histogram || []
    renderChart()
    void loadResourceOptions()
  } catch (error: any) {
    if (error?.name !== 'CanceledError') errorMessage.value = error?.message || error?.response?.data?.message || '日志查询失败'
  } finally { loading.value = false }
}

const applyPageResult = (result: LogQueryResponse, page: number) => {
  items.value = result.items || []
  currentPage.value = page
  nextCursor.value = result.nextCursor || ''
  hasMore.value = Boolean(result.hasMore)
  expandedRows.value = new Set()
  if (result.nextCursor) pageCursors.value[page] = result.nextCursor
  else pageCursors.value.splice(page)
  nextTick(() => { if (logStreamRef.value) logStreamRef.value.scrollTop = 0 })
}

const loadPage = async (page: number, cursor: string) => {
  if (loadingMore.value || page < 1) return
  queryController?.abort()
  queryController = new AbortController()
  loadingMore.value = true
  errorMessage.value = ''
  try {
    const result = await queryInternalLogs(buildPayload(cursor, true), queryController.signal) as any as LogQueryResponse
    applyPageResult(result, page)
    durationMs.value = Number(result.durationMs || 0)
  } catch (error: any) {
    if (error?.name !== 'CanceledError') errorMessage.value = error?.message || error?.response?.data?.message || '日志分页查询失败'
  } finally {
    loadingMore.value = false
  }
}

const goPreviousPage = () => {
  if (currentPage.value <= 1) return
  const targetPage = currentPage.value - 1
  void loadPage(targetPage, pageCursors.value[targetPage - 1] || '')
}

const goNextPage = () => {
  if (!hasMore.value || !nextCursor.value) return
  const targetPage = currentPage.value + 1
  pageCursors.value[targetPage - 1] = nextCursor.value
  void loadPage(targetPage, nextCursor.value)
}

const applyQuickRange = () => {
  const end = new Date()
  timeRange.value = [new Date(end.getTime() - quickRange.value * 60 * 1000), end]
}

const addCondition = () => {
  conditionSequence += 1
  conditions.value.push({ id: conditionSequence, field: 'service', operator: 'eq', value: '' })
}
const removeCondition = (id: number) => { conditions.value = conditions.value.filter(item => item.id !== id) }

const renderChart = () => {
  nextTick(() => {
    if (!chartRef.value) return
    chart ||= echarts.init(chartRef.value)
    chart.setOption({
      animation: false,
      grid: { left: 46, right: 18, top: 18, bottom: 30 },
      tooltip: { trigger: 'axis', formatter: (params: any) => `${params?.[0]?.axisValueLabel || ''}<br/>日志数：${params?.[0]?.value || 0}` },
      xAxis: { type: 'category', boundaryGap: false, data: histogram.value.map(item => formatChartTime(item.time)), axisLine: { lineStyle: { color: '#d8dee8' } }, axisLabel: { color: '#7b8494', hideOverlap: true } },
      yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: '#edf0f5' } }, axisLabel: { color: '#7b8494' } },
      series: [{ type: 'line', data: histogram.value.map(item => item.count), showSymbol: false, smooth: 0.25, lineStyle: { color: '#2563eb', width: 2 }, areaStyle: { color: 'rgba(37,99,235,.10)' } }],
    }, true)
  })
}

const toggleField = (name: string) => {
  selectedFields.value = selectedFields.value.includes(name) ? selectedFields.value.filter(item => item !== name) : [...selectedFields.value, name]
}
const toggleExpanded = (item: LogItem, index: number) => {
  const key = logKey(item, index)
  const next = new Set(expandedRows.value)
  next.has(key) ? next.delete(key) : next.add(key)
  expandedRows.value = next
}
const detailEntries = (item: LogItem): DetailEntry[] => {
  const all = { ...item.labels, ...item.fields }
  const entries = Object.entries(all).map(([key, value]) => ({ key, value }))
  if (!selectedFields.value.length) return entries
  return entries.sort((left, right) => Number(selectedFields.value.includes(right.key)) - Number(selectedFields.value.includes(left.key)))
}
const openContext = async (item: LogItem) => {
  activeLog.value = item; contextVisible.value = true; contextLoading.value = true; contextItems.value = []
  try {
    const result = await queryInternalLogContext({
      storageId: form.storageId, timestamp: item.timestamp, message: item.message, level: item.level,
      labels: item.labels, fields: item.fields, beforeSeconds: 300, afterSeconds: 300, limit: 201,
    }) as any as LogQueryResponse
    contextItems.value = result.items || []
  } finally { contextLoading.value = false; await nextTick(); centerContextLog() }
}
const centerContextLog = () => {
  nextTick(() => contextListRef.value?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'center' }))
}
const isActiveContext = (item: LogItem) => item.contextSelected === true
const copyLog = async (item: LogItem) => { await navigator.clipboard.writeText(`${item.timestamp} ${levelText(item.level)} ${item.message}`); ElMessage.success('日志已复制') }

const toggleTail = () => tailing.value ? stopTail() : startTail()
const startTail = () => {
  if (!form.storageId) { ElMessage.warning('请先选择日志存储'); return }
  stopTail(true)
  queryController?.abort()
  histogramController?.abort()
  loading.value = false
  const controller = new AbortController()
  tailController = controller
  tailing.value = true; tailConnected.value = false; tailReceived.value = 0; errorMessage.value = ''
  void streamInternalLogs(buildPayload('', true), {
    signal: controller.signal,
    onReady: () => { if (tailController === controller) tailConnected.value = true },
    onLogs: payload => {
      if (tailController !== controller) return
      const known = new Set(items.value.map(item => logIdentity(item)))
      const incoming: LogItem[] = []
      for (const item of payload.items || []) {
        const identity = logIdentity(item)
        if (known.has(identity)) continue
        known.add(identity)
        incoming.push(item)
      }
      if (!incoming.length) return
      tailReceived.value += incoming.length
      if (form.sort === 'desc') {
        items.value.unshift(...incoming.slice().reverse())
        if (items.value.length > 2000) items.value.splice(2000)
      } else {
        items.value.push(...incoming)
        if (items.value.length > 2000) items.value.splice(0, items.value.length - 2000)
      }
    },
    onError: payload => { if (tailController === controller) errorMessage.value = payload.message || '实时 Tail 查询失败' },
    onEnd: () => { if (tailController === controller) { tailConnected.value = false; tailing.value = false } },
  }).then(() => {
    if (!controller.signal.aborted && tailController === controller) { tailConnected.value = false; tailing.value = false }
  }).catch((error: any) => {
    if (tailController !== controller) return
    if (error?.name !== 'AbortError') {
      errorMessage.value = error?.message || '实时 Tail 连接失败'
      ElMessage.error(errorMessage.value)
    }
    tailConnected.value = false; tailing.value = false
  })
}
const stopTail = (silent = false) => {
  tailController?.abort(); tailController = undefined; tailing.value = false; tailConnected.value = false
  if (!silent) ElMessage.success('实时 Tail 已停止')
}

const openExportDialog = () => {
  clearExportTimer(); exportTask.value = undefined; exportVisible.value = true
}
const createExport = async () => {
  if (!form.storageId) return
  exportCreating.value = true
  try {
    exportTask.value = await createInternalLogExport({ ...buildPayload('', true), format: exportForm.format, maxRows: exportForm.maxRows }) as any
    startExportPolling()
  } finally { exportCreating.value = false }
}
const startExportPolling = () => {
  clearExportTimer()
  exportTimer = window.setInterval(async () => {
    if (!exportTask.value) return
    const task = await getInternalLogExport(exportTask.value.id) as any as LogExportTask
    exportTask.value = task
    if (['completed', 'failed', 'expired'].includes(task.status)) clearExportTimer()
  }, 1500)
}
const clearExportTimer = () => { if (exportTimer) window.clearInterval(exportTimer); exportTimer = undefined }
const resetExport = () => { clearExportTimer(); exportTask.value = undefined }
const downloadExport = async () => {
  if (!exportTask.value) return
  const blob = await downloadInternalLogExport(exportTask.value.id) as any as Blob
  const href = URL.createObjectURL(blob)
  const anchor = document.createElement('a'); anchor.href = href; anchor.download = exportTask.value.fileName || `opshub-logs-${exportTask.value.id}.${exportTask.value.format}`; anchor.click()
  URL.revokeObjectURL(href)
}
const exportStatusText = (status: string) => ({ pending: '等待中', running: '导出中', completed: '已完成', failed: '失败', expired: '已过期' }[status] || status)
const exportStatusType = (status: string) => status === 'completed' ? 'success' : status === 'failed' ? 'danger' : status === 'expired' ? 'info' : 'warning'
const formatBytes = (value?: number) => !value ? '0 B' : value >= 1073741824 ? `${(value / 1073741824).toFixed(2)} GiB` : value >= 1048576 ? `${(value / 1048576).toFixed(2)} MiB` : value >= 1024 ? `${(value / 1024).toFixed(1)} KiB` : `${value} B`
const highlightMessage = (message: string) => {
  const escaped = escapeHTML(message)
  const keyword = form.query.trim()
  if (!keyword || keyword === '*') return escaped
  return escaped.replace(new RegExp(escapeRegExp(escapeHTML(keyword)), 'gi'), value => `<mark>${value}</mark>`)
}
const escapeHTML = (value: string) => value.replace(/[&<>"]/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[char] || char))
const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const logIdentity = (item: LogItem) => `${item.timestamp}-${item.fields?.fingerprint || ''}-${item.fields?.sequence || ''}-${item.message}`
const logKey = (item: LogItem, index: number) => `${item.timestamp}-${item.fields?.fingerprint || index}-${item.fields?.sequence || ''}`
const levelText = (value?: string) => (value || 'INFO').toUpperCase()
const levelClass = (value?: string) => `level-${levelText(value).toLowerCase()}`
const levelTagType = (value?: string) => levelText(value) === 'ERROR' ? 'danger' : levelText(value) === 'WARN' ? 'warning' : 'info'
const formatTimestamp = (value: string) => value ? new Date(value).toLocaleString() : '-'
const formatChartTime = (value: string) => new Date(value).toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
const formatValue = (value: any) => typeof value === 'object' ? JSON.stringify(value) : String(value ?? '')

watch(() => form.sort, () => items.value.length && !tailing.value && runQuery())
onMounted(() => { loadInitialData(); window.addEventListener('resize', resizeChart) })
onBeforeUnmount(() => {
  queryController?.abort(); histogramController?.abort(); resourceController?.abort(); stopTail(true); clearExportTimer(); chart?.dispose(); window.removeEventListener('resize', resizeChart)
})
const resizeChart = () => chart?.resize()
</script>

<style scoped>
.internal-query-page { display: flex; flex-direction: column; gap: 12px; }
.head-actions, .query-context-row, .context-control, .filter-section-head, .filter-title, .section-head, .section-head > div { display: flex; align-items: center; }
.head-actions, .context-control { gap: 10px; }
.query-panel { padding: 0; overflow: hidden; }
.query-context-row { justify-content: space-between; gap: 20px; min-height: 64px; padding: 13px 18px; border-bottom: 1px solid #edf0f4; background: #fbfcfe; }
.control-label { flex: 0 0 auto; color: #667085; font-size: 12px; font-weight: 600; }
.storage-context { min-width: 0; }.storage-select { width: 250px; }.time-context { justify-content: flex-end; }.time-picker { width: 360px; }
.query-search-row { display: grid; grid-template-columns: minmax(300px, 1fr) 126px auto; gap: 10px; align-items: center; padding: 16px 18px; }
.query-text-input :deep(.el-input__wrapper) { min-height: 38px; padding-left: 12px; }
.query-text-input :deep(.el-input__prefix) { color: #667085; font-size: 16px; }
.page-size-select { width: 126px; }
.query-search-row .primary-action { min-width: 92px; height: 38px; }
.filter-section { padding: 0 18px 16px; }
.filter-section-head { justify-content: space-between; min-height: 38px; padding-top: 2px; border-top: 1px solid #edf0f4; }
.filter-title { gap: 8px; color: #344054; }.filter-title > .el-icon { color: #667085; }.filter-title strong { font-size: 13px; }
.advanced-toggle { color: #475467; }.advanced-toggle .el-icon { margin-left: 4px; transition: transform .2s; }.advanced-toggle .el-icon.rotated { transform: rotate(180deg); }
.primary-filter-grid, .advanced-filter-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: 10px; }
.filter-control { min-width: 0; }.filter-control > span { display: block; margin-bottom: 6px; color: #7b8494; font-size: 11px; font-weight: 600; }
.filter-control :deep(.el-select), .filter-control :deep(.el-segmented) { width: 100%; }
.source-filter-control { grid-column: span 2; }.source-filter-control :deep(.el-segmented) { min-height: 32px; }
.advanced-filter-area { margin-top: 14px; padding-top: 14px; border-top: 1px dashed #dfe5ee; }
.condition-builder { margin-top: 14px; padding: 12px 14px; border: 1px solid #e7ebf1; border-radius: 6px; background: #fafbfc; }
.condition-head { display: flex; align-items: center; justify-content: space-between; min-height: 28px; }
.condition-head strong { color: #344054; font-size: 12px; }
.condition-list { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
.condition-row { display: grid; grid-template-columns: minmax(160px, .8fr) 130px minmax(220px, 1.5fr) 34px; gap: 8px; align-items: center; }
.empty-condition { padding: 6px 0 2px; color: #98a2b3; font-size: 12px; }
.histogram-panel { padding: 14px 18px 8px; }
.section-head { justify-content: space-between; gap: 12px; margin-bottom: 10px; color: #667085; font-size: 12px; }
.section-head > div { gap: 10px; }
.section-head strong { color: #1f2937; font-size: 14px; }
.section-head.compact { margin-bottom: 12px; }
.histogram-chart { height: 190px; }
.query-result-grid { display: grid; grid-template-columns: 230px minmax(0, 1fr); gap: 12px; min-height: 480px; }
.field-panel { padding: 16px; overflow: hidden; }
.field-list { display: flex; flex-direction: column; gap: 2px; max-height: calc(100vh - 470px); min-height: 380px; margin-top: 10px; overflow-y: auto; }
.field-list button { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; padding: 7px 8px; border: 0; border-radius: 5px; background: transparent; color: #475467; cursor: pointer; text-align: left; }
.field-list button:hover, .field-list button.active { background: #eff6ff; color: #1d4ed8; }
.field-list span { overflow: hidden; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.field-list em { color: #98a2b3; font-size: 10px; font-style: normal; }
.log-panel { min-width: 0; overflow: hidden; }
.log-head { min-height: 50px; margin: 0; padding: 0 16px; border-bottom: 1px solid #edf0f4; }
.tail-status { display: inline-flex; align-items: center; gap: 6px; }
.tail-status i { width: 7px; height: 7px; border-radius: 50%; background: #f59e0b; }
.tail-status i.connected { background: #16a34a; box-shadow: 0 0 0 3px rgba(22, 163, 74, .12); }
.log-stream { min-height: 390px; max-height: calc(100vh - 440px); overflow: auto; }
.log-row { border-bottom: 1px solid #f0f2f5; background: #fff; }
.log-row:hover { background: #fafcff; }
.log-line { display: grid; grid-template-columns: 24px 160px 58px minmax(200px, 1fr) auto; align-items: start; gap: 8px; min-height: 40px; padding: 9px 12px; }
.expand-button { display: grid; width: 24px; height: 24px; place-items: center; border: 0; background: transparent; color: #98a2b3; cursor: pointer; transition: transform .15s; }
.log-row.expanded .expand-button { transform: rotate(90deg); }
.log-line time { padding-top: 3px; color: #667085; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; white-space: nowrap; }
.level-badge { align-self: start; padding: 3px 6px; border-radius: 4px; background: #f2f4f7; color: #475467; font-size: 10px; font-weight: 700; text-align: center; }
.level-error, .level-fatal { background: #fef2f2; color: #dc2626; }
.level-warn, .level-warning { background: #fffbeb; color: #d97706; }
.level-info { background: #eff6ff; color: #2563eb; }
.level-debug, .level-trace { background: #f5f3ff; color: #7c3aed; }
.log-line p { margin: 1px 0 0; color: #1f2937; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.55; overflow-wrap: anywhere; white-space: pre-wrap; }
.log-line :deep(mark) { padding: 0 2px; background: #fde68a; color: inherit; }
.row-actions { display: flex; opacity: 0; transition: opacity .15s; }
.log-row:hover .row-actions { opacity: 1; }
.log-detail { display: grid; grid-template-columns: minmax(300px, .9fr) minmax(360px, 1.1fr); gap: 16px; padding: 14px 44px 18px; border-top: 1px dashed #e5e7eb; background: #f9fafb; }
.detail-section > strong { display: block; margin-bottom: 10px; color: #344054; font-size: 12px; }
.detail-section dl { display: grid; grid-template-columns: 145px minmax(0, 1fr); margin: 0; font-size: 11px; }
.detail-section dt, .detail-section dd { margin: 0; padding: 5px 7px; border-bottom: 1px solid #edf0f4; }
.detail-section dt { color: #667085; }
.detail-section dd { color: #1f2937; overflow-wrap: anywhere; }
.message-section pre, .context-row pre { margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; overflow-wrap: anywhere; }
.message-section pre { max-height: 260px; padding: 12px; overflow: auto; border: 1px solid #e5e7eb; border-radius: 6px; background: #fff; color: #111827; font-size: 12px; line-height: 1.55; }
.result-footer { display: flex; align-items: center; justify-content: space-between; gap: 14px; min-height: 54px; padding: 0 16px; border-top: 1px solid #edf0f4; color: #667085; font-size: 12px; }
.page-actions { display: flex; align-items: center; gap: 10px; }
.context-toolbar { display: flex; justify-content: space-between; margin-bottom: 12px; color: #667085; font-size: 12px; }
.context-list { height: calc(100vh - 150px); overflow: auto; border: 1px solid #e5e7eb; }
.context-row { display: grid; grid-template-columns: 165px 58px minmax(0, 1fr); gap: 10px; padding: 9px 12px; border-bottom: 1px solid #f0f2f5; }
.context-row.active { position: relative; background: #eff6ff; box-shadow: inset 3px 0 #2563eb; }
.context-row time { color: #667085; font-size: 11px; }
.context-row pre { color: #1f2937; font-size: 12px; line-height: 1.5; }
.export-form :deep(.el-input-number) { width: 100%; }
.export-status-line { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; color: #667085; font-size: 12px; }
.export-progress dl { display: grid; grid-template-columns: 90px minmax(0, 1fr); margin: 18px 0; border-top: 1px solid #edf0f4; font-size: 12px; }
.export-progress dt, .export-progress dd { margin: 0; padding: 9px 6px; border-bottom: 1px solid #edf0f4; }
.export-progress dt { color: #667085; }.export-progress dd { color: #1f2937; overflow-wrap: anywhere; }
@media (max-width: 900px) {
  .query-context-row { align-items: flex-start; flex-direction: column; }
  .time-context { justify-content: flex-start; flex-wrap: wrap; }
  .query-search-row { grid-template-columns: minmax(0, 1fr) 126px; }
  .query-search-row .primary-action { grid-column: 1 / -1; }
  .condition-row { grid-template-columns: 1fr 120px; }
  .condition-row :deep(.el-input) { grid-column: 1 / -1; }
  .query-result-grid { grid-template-columns: 1fr; }
  .field-panel { display: none; }
  .log-detail { grid-template-columns: 1fr; padding-left: 20px; }
}
@media (max-width: 640px) {
  .head-actions { flex-wrap: wrap; }
  .context-control { align-items: flex-start; flex-wrap: wrap; width: 100%; }
  .storage-select, .time-picker { width: 100%; }
  .query-search-row { grid-template-columns: 1fr; }
  .query-search-row .primary-action { grid-column: auto; }
  .page-size-select { width: 100%; }
  .source-filter-control { grid-column: span 1; }
  .condition-row { grid-template-columns: 1fr; }
  .condition-row :deep(.el-input) { grid-column: auto; }
  .log-line { grid-template-columns: 24px 1fr 58px; }
  .log-line p, .row-actions { grid-column: 2 / -1; }
}
</style>
