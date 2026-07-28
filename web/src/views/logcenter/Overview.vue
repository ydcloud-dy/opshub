<template>
  <div class="log-center-page log-overview-page">
    <header class="overview-header">
      <div class="overview-title">
        <div class="title-line">
          <h2>日志总览</h2>
          <span class="pipeline-badge" :class="pipelineState.tone"><i />{{ pipelineState.label }}</span>
        </div>
        <p>聚焦日志采集、写入、存储与数据质量，指标范围默认为最近 24 小时</p>
      </div>
      <div class="header-actions">
        <span class="updated-at">{{ updatedAtText }}</span>
        <el-button @click="$router.push('/logs/query')"><el-icon><Search /></el-icon>日志查询</el-button>
        <el-tooltip content="刷新总览" placement="bottom">
          <el-button class="icon-button" :loading="loading" aria-label="刷新总览" @click="loadData(false)"><el-icon><Refresh /></el-icon></el-button>
        </el-tooltip>
      </div>
    </header>

    <el-alert
      v-if="overview.storage.error"
      class="storage-alert"
      type="warning"
      :closable="false"
      show-icon
      title="日志指标暂不可用"
      :description="overview.storage.error"
    />

    <section v-loading="loading" class="metrics-grid">
      <article class="metric-card metric-health">
        <div class="metric-icon"><el-icon><Connection /></el-icon></div>
        <div class="metric-content">
          <span>采集链路</span>
          <strong :class="`metric-${pipelineState.tone}`">{{ pipelineState.shortLabel }}</strong>
          <small>{{ ingest?.readinessSummary.passed || 0 }}/{{ ingest?.readinessSummary.total || 0 }} 项检查通过</small>
        </div>
      </article>
      <article class="metric-card metric-volume">
        <div class="metric-icon"><el-icon><Document /></el-icon></div>
        <div class="metric-content">
          <span>近 24 小时日志</span>
          <strong>{{ formatCompact(overview.logs24h) }}</strong>
          <small>原始日志量 {{ formatBytes(overview.bytes24h) }}</small>
        </div>
      </article>
      <article class="metric-card metric-rate">
        <div class="metric-icon"><el-icon><Odometer /></el-icon></div>
        <div class="metric-content">
          <span>近 5 分钟平均速率</span>
          <strong>{{ formatRate(overview.averageEps5m) }}</strong>
          <small>采集器当前输出 {{ formatRate(overview.collectors.outputEps) }}</small>
        </div>
      </article>
      <article class="metric-card metric-collector">
        <div class="metric-icon"><el-icon><Monitor /></el-icon></div>
        <div class="metric-content">
          <span>在线采集器</span>
          <strong>{{ overview.collectors.online }}<em>/{{ overview.collectors.total }}</em></strong>
          <small>{{ overview.activeServices }} 个活跃服务</small>
        </div>
      </article>
      <article class="metric-card metric-error">
        <div class="metric-icon"><el-icon><Warning /></el-icon></div>
        <div class="metric-content">
          <span>错误日志</span>
          <strong>{{ formatCompact(overview.errors24h) }}</strong>
          <small>错误率 {{ errorRateText }}</small>
        </div>
      </article>
    </section>

    <section class="chart-grid">
      <article class="panel trend-panel">
        <div class="panel-head">
          <div>
            <h3>日志写入趋势</h3>
            <p>{{ trendRangeDescription }}，不受查询页分页上限影响</p>
          </div>
          <div class="trend-tools">
            <el-segmented v-model="trendRange" :options="trendRangeOptions" />
            <el-segmented v-model="trendMetric" :options="trendMetricOptions" />
          </div>
          <div class="panel-summary"><strong>{{ formatTrendValue(trendTotal) }}</strong><span>{{ trendMetricLabel }} · {{ trendRangeLabel }}</span></div>
        </div>
        <div ref="trendChartRef" class="chart trend-chart" />
      </article>

      <article class="panel level-panel">
        <div class="panel-head">
          <div>
            <h3>日志级别分布</h3>
            <p>识别结果占比与错误日志规模</p>
          </div>
        </div>
        <div ref="levelChartRef" class="chart level-chart" />
      </article>
    </section>

    <section class="detail-grid">
      <article class="panel ranking-panel">
        <div class="panel-head ranking-head">
          <div>
            <h3>{{ rankingMode === 'service' ? '高频服务' : '日志来源' }}</h3>
            <p>{{ rankingMode === 'service' ? '最近 24 小时日志量最高的服务' : '主机与 Kubernetes 等来源占比' }}</p>
          </div>
          <el-segmented v-model="rankingMode" :options="rankingOptions" />
        </div>
        <div v-if="rankingItems.length" class="ranking-list">
          <div v-for="(item, index) in rankingItems" :key="item.name" class="ranking-row">
            <span class="ranking-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="ranking-main">
              <div><strong :title="item.name">{{ rankingName(item.name) }}</strong><span>{{ formatCompact(item.value) }}</span></div>
              <div class="ranking-track"><i :style="{ width: `${rankingPercent(item.value)}%` }" /></div>
            </div>
          </div>
        </div>
        <el-empty v-else :image-size="72" description="最近 24 小时暂无日志数据" />
      </article>

      <article class="panel pipeline-panel">
        <div class="panel-head">
          <div>
            <h3>采集链路状态</h3>
            <p>组件连通性、队列积压与存储状态</p>
          </div>
          <span class="check-summary">{{ pipelineState.label }}</span>
        </div>
        <div class="stage-list">
          <div v-for="stage in pipelineStages" :key="stage.key" class="stage-row">
            <i class="stage-dot" :class="stage.tone" />
            <div class="stage-main">
              <div><strong>{{ stage.name }}</strong><span>{{ stage.description }}</span></div>
              <small>{{ stage.meta }}</small>
            </div>
            <span class="stage-status" :class="stage.tone">{{ stage.status }}</span>
          </div>
        </div>
        <div class="pipeline-foot">
          <span>WAL {{ formatBytes(overview.collectors.walBytes) }}</span>
          <span>重试 {{ formatCompact(overview.collectors.retryTotal) }}</span>
          <span>丢弃 {{ formatCompact(overview.collectors.droppedTotal) }}</span>
          <span>最近写入 {{ formatRelativeTime(overview.collectors.lastIngestAt) }}</span>
        </div>
      </article>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { Connection, Document, Monitor, Odometer, Refresh, Search, Warning } from '@element-plus/icons-vue'
import { getLogIngestStatus, getLogOverview, type LogIngestStatus, type LogOverview } from '@/api/logcenter'

type StageTone = 'success' | 'warning' | 'danger' | 'neutral'
type TrendRange = '24h' | '30d' | '12m'
type TrendMetric = 'count' | 'bytes'

interface PipelineStage {
  key: string
  name: string
  status: string
  tone: StageTone
  description: string
  meta: string
}

const emptyOverview = (): LogOverview => ({
  logs24h: 0,
  bytes24h: 0,
  errors24h: 0,
  averageEps5m: 0,
  activeServices: 0,
  trend: [],
  levels: [],
  topServices: [],
  sources: [],
  collectors: {
    total: 0,
    online: 0,
    errors: 0,
    inputEps: 0,
    outputEps: 0,
    walBytes: 0,
    droppedTotal: 0,
    retryTotal: 0,
  },
  storage: { available: false },
  updatedAt: '',
})

const loading = ref(false)
const overview = ref<LogOverview>(emptyOverview())
const ingest = ref<LogIngestStatus>()
const trendChartRef = ref<HTMLElement>()
const levelChartRef = ref<HTMLElement>()
const rankingMode = ref<'service' | 'source'>('service')
const rankingOptions = [
  { label: '服务排行', value: 'service' },
  { label: '来源分布', value: 'source' },
]
const trendRange = ref<TrendRange>('24h')
const trendMetric = ref<TrendMetric>('count')
const trendRangeOptions = [
  { label: '24 小时', value: '24h' },
  { label: '30 天', value: '30d' },
  { label: '12 个月', value: '12m' },
]
const trendMetricOptions = [
  { label: '日志条数', value: 'count' },
  { label: '数据量', value: 'bytes' },
]
const sourceNames: Record<string, string> = { host: '主机日志', kubernetes: 'Kubernetes', 'kubernetes-node': 'Kubernetes' }

let trendChart: echarts.ECharts | undefined
let levelChart: echarts.ECharts | undefined
let refreshTimer: ReturnType<typeof setInterval> | undefined

const pipelineHealthy = computed(() => Boolean(
  ingest.value?.gateway.reachable && ingest.value?.writer.reachable && ingest.value?.storage.reachable
  && (!ingest.value.queue.enabled || ingest.value.queue.reachable),
))

const pipelineState = computed(() => {
  const failed = ingest.value?.readinessSummary.failed || 0
  const warnings = ingest.value?.readinessSummary.warnings || 0
  if (failed > 0 || (!pipelineHealthy.value && ingest.value)) return { tone: 'danger', label: '采集链路异常', shortLabel: '异常' }
  if (warnings > 0) return { tone: 'warning', label: `${warnings} 项配置需关注`, shortLabel: '需关注' }
  if (!ingest.value) return { tone: 'neutral', label: '正在检查链路', shortLabel: '检查中' }
  return { tone: 'success', label: '采集链路正常', shortLabel: '正常' }
})

const errorRateText = computed(() => {
  if (!overview.value.logs24h) return '0%'
  const rate = overview.value.errors24h / overview.value.logs24h * 100
  return `${rate < 0.01 && rate > 0 ? '<0.01' : rate.toFixed(rate >= 10 ? 1 : 2)}%`
})

const updatedAtText = computed(() => overview.value.updatedAt
  ? `更新于 ${new Date(overview.value.updatedAt).toLocaleTimeString('zh-CN', { hour12: false })}`
  : '尚未更新')

const rankingItems = computed(() => rankingMode.value === 'service' ? overview.value.topServices : overview.value.sources)
const rankingMaximum = computed(() => Math.max(...rankingItems.value.map(item => Number(item.value) || 0), 1))

const trendRangeLabel = computed(() => ({ '24h': '24 小时', '30d': '30 天', '12m': '12 个月' }[trendRange.value]))
const trendRangeDescription = computed(() => ({
  '24h': '最近 24 小时按小时聚合',
  '30d': '最近 30 天按自然日聚合',
  '12m': '最近 12 个月按自然月聚合',
}[trendRange.value]))
const trendMetricLabel = computed(() => trendMetric.value === 'bytes' ? '数据量' : '日志条数')

const localDayKey = (date: Date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
const localMonthKey = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
const utcHourKey = (date: Date) => `${date.toISOString().slice(0, 13)}:00:00Z`

const trendBuckets = computed(() => {
  const values = new Map<string, { count: number; bytes: number }>()
  overview.value.trend.forEach(item => values.set(item.time, {
    count: Number(item.count) || 0,
    bytes: Number(item.bytes) || 0,
  }))
  if (trendRange.value === '24h') {
    const end = new Date()
    end.setUTCMinutes(0, 0, 0)
    return Array.from({ length: 24 }, (_, index) => {
      const date = new Date(end.getTime() - (23 - index) * 60 * 60 * 1000)
      const key = utcHourKey(date)
      return {
        key, label: date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }),
        tooltip: date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }),
        ...(values.get(key) || { count: 0, bytes: 0 }),
      }
    })
  }
  if (trendRange.value === '30d') {
    const end = new Date()
    end.setHours(0, 0, 0, 0)
    return Array.from({ length: 30 }, (_, index) => {
      const date = new Date(end)
      date.setDate(end.getDate() - (29 - index))
      const key = localDayKey(date)
      return {
        key, label: `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`,
        tooltip: key, ...(values.get(key) || { count: 0, bytes: 0 }),
      }
    })
  }
  const end = new Date(new Date().getFullYear(), new Date().getMonth(), 1)
  return Array.from({ length: 12 }, (_, index) => {
    const date = new Date(end.getFullYear(), end.getMonth() - (11 - index), 1)
    const key = localMonthKey(date)
    return { key, label: key, tooltip: `${key} 月`, ...(values.get(key) || { count: 0, bytes: 0 }) }
  })
})
const trendValue = (item: { count: number; bytes: number }) => trendMetric.value === 'bytes' ? item.bytes : item.count
const trendTotal = computed(() => trendBuckets.value.reduce((total, item) => total + trendValue(item), 0))

const pipelineStages = computed<PipelineStage[]>(() => {
  const gateway = ingest.value?.gateway
  const queue = ingest.value?.queue
  const writer = ingest.value?.writer
  const storage = ingest.value?.storage
  return [
    {
      key: 'gateway', name: 'Log Gateway',
      ...componentStatus(gateway?.reachable, gateway?.status),
      description: gateway?.reachable ? '接收 Agent 批次并完成入口校验' : '控制面无法连接 Gateway',
      meta: `接收 ${formatCompact(gateway?.acceptedRecords || 0)} 条 · 拒绝 ${formatCompact(gateway?.rejectedBatches || 0)} 批`,
    },
    {
      key: 'queue', name: queue?.enabled ? 'Kafka / Redpanda' : '直写通道',
      ...(queue?.enabled ? componentStatus(queue.reachable, queue.status) : { status: '直写', tone: 'neutral' as StageTone }),
      description: queue?.enabled ? '持久化缓冲并解耦写入峰值' : 'Gateway 直接向 Writer 发送日志',
      meta: queue?.enabled ? `消费积压 ${formatCompact(queue.lag || 0)} · ${queue.brokerCount || 0} 个 Broker` : '适合当前规模，存储异常时由 Agent WAL 背压',
    },
    {
      key: 'writer', name: 'Log Writer',
      ...componentStatus(writer?.reachable, writer?.status),
      description: writer?.reachable ? '批量写入 ClickHouse 并返回 ACK' : '控制面无法连接 Writer',
      meta: `写入 ${formatCompact(writer?.acceptedRecords || 0)} 条 · 失败 ${formatCompact(writer?.failedBatches || 0)} 批`,
    },
    {
      key: 'storage', name: storage?.name || overview.value.storage.name || 'ClickHouse',
      ...componentStatus(storage?.reachable, storage?.status),
      description: storage?.reachable ? '分钟聚合与日志明细存储可用' : (overview.value.storage.error || '存储连接不可用'),
      meta: overview.value.storage.available ? `近 24 小时写入 ${formatBytes(overview.value.bytes24h)}` : '等待恢复后自动继续写入',
    },
  ]
})

const componentStatus = (reachable?: boolean, status?: string): { status: string; tone: StageTone } => {
  if (reachable === undefined) return { status: '检查中', tone: 'neutral' }
  if (!reachable) return { status: '异常', tone: 'danger' }
  if (status && !['healthy', 'passed', 'bypassed'].includes(status.toLowerCase())) return { status: '降级', tone: 'warning' }
  return { status: '正常', tone: 'success' }
}

const formatCompact = (value?: number) => new Intl.NumberFormat('zh-CN', {
  notation: 'compact', maximumFractionDigits: 1,
}).format(Number(value) || 0)

const formatBytes = (value?: number) => {
  const bytes = Number(value) || 0
  if (bytes < 1024) return `${bytes.toFixed(0)} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let amount = bytes / 1024
  let index = 0
  while (amount >= 1024 && index < units.length - 1) { amount /= 1024; index++ }
  return `${amount.toFixed(amount >= 100 ? 0 : amount >= 10 ? 1 : 2)} ${units[index]}`
}

const formatRate = (value?: number) => {
  const rate = Number(value) || 0
  return `${rate.toLocaleString('zh-CN', { maximumFractionDigits: rate >= 100 ? 0 : 1 })} 条/秒`
}

const formatRelativeTime = (value?: string) => {
  if (!value) return '暂无'
  const seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000))
  if (seconds < 60) return `${seconds} 秒前`
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} 小时前`
  return new Date(value).toLocaleString('zh-CN')
}

const rankingPercent = (value: number) => Math.max(3, Math.round((Number(value) || 0) / rankingMaximum.value * 100))
const rankingName = (value: string) => rankingMode.value === 'source' ? (sourceNames[value] || value) : value

const levelColor = (level: string) => ({
  FATAL: '#b42318', ERROR: '#e5484d', WARN: '#f59e0b', INFO: '#22a06b', DEBUG: '#2f6fed', TRACE: '#7c5ce7', UNKNOWN: '#98a2b3',
}[level.toUpperCase()] || '#64748b')

const renderCharts = () => nextTick(() => {
  if (trendChartRef.value) {
    trendChart ||= echarts.init(trendChartRef.value)
    trendChart.setOption({
      animationDuration: 300,
      grid: { left: 12, right: 18, top: 18, bottom: 34, containLabel: true },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          const index = params?.[0]?.dataIndex || 0
          return `${trendBuckets.value[index]?.tooltip || ''}<br/>${trendMetricLabel.value}：${formatTrendValue(Number(params?.[0]?.value || 0))}`
        },
      },
      xAxis: {
        type: 'category', boundaryGap: false, data: trendBuckets.value.map(item => item.label),
        axisLine: { lineStyle: { color: '#d8dee8' } }, axisTick: { show: false },
        axisLabel: { color: '#7b8494', interval: trendRange.value === '24h' ? 3 : trendRange.value === '30d' ? 4 : 1, hideOverlap: true },
      },
      yAxis: {
        type: 'value', minInterval: 1, splitNumber: 4,
        axisLabel: { color: '#7b8494', margin: 10, formatter: (value: number) => trendMetric.value === 'bytes' ? formatBytes(value) : formatCompact(value) },
        splitLine: { lineStyle: { color: '#edf0f5' } },
      },
      series: [{
        name: trendMetricLabel.value, type: 'line', data: trendBuckets.value.map(trendValue),
        showSymbol: false, smooth: 0.22, lineStyle: { color: '#2f6fed', width: 2 },
        areaStyle: { color: 'rgba(47,111,237,.10)' },
      }],
    }, true)
  }
  if (levelChartRef.value) {
    levelChart ||= echarts.init(levelChartRef.value)
    const levels = overview.value.levels.filter(item => item.value > 0)
    levelChart.setOption({
      animationDuration: 300,
      color: levels.map(item => levelColor(item.name)),
      tooltip: { trigger: 'item', formatter: (item: any) => `${item.name}<br/>${Number(item.value).toLocaleString('zh-CN')} 条（${item.percent}%）` },
      legend: {
        bottom: 0, left: 'center', itemWidth: 9, itemHeight: 9, icon: 'circle',
        textStyle: { color: '#667085', fontSize: 11 },
      },
      graphic: levels.length ? [{
        type: 'text', left: 'center', top: '35%', silent: true,
        style: { text: `${formatCompact(overview.value.logs24h)}\n日志`, textAlign: 'center', fill: '#101828', font: '600 16px sans-serif', lineHeight: 24 },
      }] : [{
        type: 'text', left: 'center', top: '44%', silent: true,
        style: { text: '暂无日志数据', textAlign: 'center', fill: '#98a2b3', font: '13px sans-serif' },
      }],
      series: [{
        type: 'pie', radius: ['54%', '72%'], center: ['50%', '42%'],
        avoidLabelOverlap: true, label: { show: false }, labelLine: { show: false },
        data: levels.map(item => ({ name: item.name, value: item.value })),
        itemStyle: { borderColor: '#fff', borderWidth: 2 },
      }],
    }, true)
  }
})

const loadData = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const [overviewResult, ingestResult] = await Promise.all([getLogOverview({ trendRange: trendRange.value }), getLogIngestStatus()])
    overview.value = overviewResult as unknown as LogOverview
    ingest.value = ingestResult as unknown as LogIngestStatus
    renderCharts()
  } finally {
    if (!silent) loading.value = false
  }
}

const formatTrendValue = (value?: number) => trendMetric.value === 'bytes' ? formatBytes(value) : formatCompact(value)

const resizeCharts = () => { trendChart?.resize(); levelChart?.resize() }

watch(trendRange, () => loadData(false))
watch(trendMetric, () => renderCharts())

onMounted(() => {
  loadData(false)
  refreshTimer = setInterval(() => loadData(true), 30000)
  window.addEventListener('resize', resizeCharts)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  window.removeEventListener('resize', resizeCharts)
  trendChart?.dispose()
  levelChart?.dispose()
})
</script>

<style scoped>
.log-overview-page { display:flex;flex-direction:column;gap:12px;padding:0;color:#101828; }
.overview-header { display:flex;align-items:center;justify-content:space-between;gap:20px;padding:18px 20px;border:1px solid #e4e8f0;border-radius:8px;background:#fff; }
.title-line { display:flex;align-items:center;gap:12px; }.title-line h2 { margin:0;font-size:21px;font-weight:700;line-height:1.3;letter-spacing:0; }
.overview-title p { margin:5px 0 0;color:#667085;font-size:13px;line-height:1.5; }
.pipeline-badge { display:inline-flex;align-items:center;gap:7px;color:#475467;font-size:12px;font-weight:600; }.pipeline-badge i { width:8px;height:8px;border-radius:50%;background:#98a2b3;box-shadow:0 0 0 3px #f2f4f7; }
.pipeline-badge.success { color:#067647; }.pipeline-badge.success i { background:#12b76a;box-shadow:0 0 0 3px #d1fadf; }
.pipeline-badge.warning { color:#b54708; }.pipeline-badge.warning i { background:#f79009;box-shadow:0 0 0 3px #fef0c7; }
.pipeline-badge.danger { color:#b42318; }.pipeline-badge.danger i { background:#f04438;box-shadow:0 0 0 3px #fee4e2; }
.header-actions { display:flex;align-items:center;gap:8px; }.updated-at { margin-right:4px;color:#98a2b3;font-size:12px;white-space:nowrap; }
.header-actions .el-button { border-color:#dfe3eb;color:#344054; }.header-actions .el-button:hover { border-color:#111827;color:#111827;background:#f8fafc; }.header-actions .icon-button { width:34px;padding:0; }
.storage-alert { margin:0; }
.metrics-grid { display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px; }
.metric-card { position:relative;display:flex;align-items:center;gap:14px;min-width:0;min-height:116px;padding:17px 18px;border:1px solid #e4e8f0;border-radius:8px;background:#fff;overflow:hidden; }
.metric-icon { display:flex;align-items:center;justify-content:center;width:42px;height:42px;flex:0 0 auto;border:1px solid #e4e7ec;border-radius:7px;background:#f8fafc;color:#475467;font-size:21px; }
.metric-health .metric-icon { border-color:#abefc6;background:#ecfdf3;color:#067647; }
.metric-volume .metric-icon { border-color:#c7d7fe;background:#eff4ff;color:#2f6fed; }
.metric-rate .metric-icon { border-color:#ddd6fe;background:#f5f3ff;color:#6941c6; }
.metric-collector .metric-icon { border-color:#a5f3fc;background:#ecfeff;color:#0e7490; }
.metric-error .metric-icon { border-color:#fecdca;background:#fef3f2;color:#d92d20; }
.metric-content { min-width:0; }.metric-content>span { display:block;color:#667085;font-size:12px;font-weight:600;white-space:nowrap; }
.metric-content strong { display:block;margin:7px 0 6px;color:#101828;font-size:27px;font-weight:700;line-height:1;letter-spacing:0;white-space:nowrap; }.metric-content strong em { margin-left:3px;color:#98a2b3;font-size:16px;font-style:normal;font-weight:500; }
.metric-content strong.metric-success { color:#067647; }.metric-content strong.metric-warning { color:#b54708; }.metric-content strong.metric-danger { color:#b42318; }.metric-content strong.metric-neutral { color:#475467; }
.metric-content small { display:block;overflow:hidden;color:#98a2b3;font-size:12px;text-overflow:ellipsis;white-space:nowrap; }
.chart-grid { display:grid;grid-template-columns:minmax(0,1.85fr) minmax(320px,.85fr);gap:12px; }
.detail-grid { display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.12fr);gap:12px; }
.panel { min-width:0;padding:18px 20px;border:1px solid #e4e8f0;border-radius:8px;background:#fff; }
.panel-head { display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:10px; }.panel-head h3 { margin:0;color:#101828;font-size:15px;font-weight:700;letter-spacing:0; }.panel-head p { margin:5px 0 0;color:#98a2b3;font-size:12px;line-height:1.5; }
.trend-panel .panel-head { align-items:center; }.trend-tools { display:flex;align-items:center;gap:8px;margin-left:auto; }.trend-tools :deep(.el-segmented) { --el-segmented-item-selected-bg-color:#111827;--el-segmented-item-selected-color:#fff; }
.panel-summary { text-align:right; }.panel-summary strong,.panel-summary span { display:block; }.panel-summary strong { color:#101828;font-size:20px; }.panel-summary span { margin-top:3px;color:#98a2b3;font-size:11px; }
.chart { width:100%;height:286px; }.level-chart { height:286px; }
.ranking-head { align-items:center;margin-bottom:4px; }.ranking-head :deep(.el-segmented) { --el-segmented-item-selected-bg-color:#111827;--el-segmented-item-selected-color:#fff; }
.ranking-list { display:flex;flex-direction:column; }.ranking-row { display:flex;align-items:center;gap:13px;min-height:51px;border-bottom:1px solid #eef1f5; }.ranking-row:last-child { border-bottom:0; }
.ranking-index { width:23px;flex:0 0 auto;color:#98a2b3;font:600 11px/1 ui-monospace,SFMono-Regular,Menlo,monospace; }.ranking-main { min-width:0;flex:1; }.ranking-main>div:first-child { display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:7px; }.ranking-main strong { overflow:hidden;color:#344054;font-size:12px;font-weight:600;text-overflow:ellipsis;white-space:nowrap; }.ranking-main span { flex:0 0 auto;color:#667085;font-size:12px;font-weight:600; }
.ranking-track { width:100%;height:4px;background:#eef2f6;overflow:hidden; }.ranking-track i { display:block;height:100%;background:#2f6fed;transition:width .25s ease; }
.check-summary { color:#667085;font-size:12px;font-weight:600;white-space:nowrap; }.stage-list { display:flex;flex-direction:column; }.stage-row { display:grid;grid-template-columns:10px minmax(0,1fr) auto;align-items:center;gap:12px;min-height:61px;border-bottom:1px solid #eef1f5; }.stage-row:last-child { border-bottom:0; }
.stage-dot { width:8px;height:8px;border-radius:50%;background:#98a2b3; }.stage-dot.success { background:#12b76a;box-shadow:0 0 0 3px #d1fadf; }.stage-dot.warning { background:#f79009;box-shadow:0 0 0 3px #fef0c7; }.stage-dot.danger { background:#f04438;box-shadow:0 0 0 3px #fee4e2; }
.stage-main { min-width:0; }.stage-main>div { display:flex;align-items:baseline;gap:10px;min-width:0; }.stage-main strong { flex:0 0 auto;color:#344054;font-size:12px; }.stage-main span { overflow:hidden;color:#667085;font-size:12px;text-overflow:ellipsis;white-space:nowrap; }.stage-main small { display:block;margin-top:5px;overflow:hidden;color:#98a2b3;font-size:11px;text-overflow:ellipsis;white-space:nowrap; }
.stage-status { padding-left:10px;color:#667085;font-size:12px;font-weight:600; }.stage-status.success { color:#067647; }.stage-status.warning { color:#b54708; }.stage-status.danger { color:#b42318; }
.pipeline-foot { display:grid;grid-template-columns:repeat(4,1fr);gap:1px;margin-top:12px;border:1px solid #e4e8f0;background:#e4e8f0; }.pipeline-foot span { min-width:0;padding:9px 8px;overflow:hidden;background:#f8fafc;color:#667085;font-size:11px;text-align:center;text-overflow:ellipsis;white-space:nowrap; }
@media (max-width:1500px) { .metrics-grid { grid-template-columns:repeat(3,minmax(0,1fr)); }.metric-card:nth-child(4),.metric-card:nth-child(5) { min-height:104px; } }
@media (max-width:1100px) { .chart-grid,.detail-grid { grid-template-columns:1fr; }.level-chart { height:320px; } }
@media (max-width:1100px) { .trend-panel .panel-head { align-items:flex-start;flex-wrap:wrap; }.trend-tools { order:3;width:100%;margin-left:0; }.trend-tools :deep(.el-segmented) { max-width:100%; } }
@media (max-width:760px) { .overview-header { align-items:flex-start;flex-direction:column; }.header-actions { width:100%; }.updated-at { flex:1; }.metrics-grid { grid-template-columns:1fr 1fr; }.pipeline-foot { grid-template-columns:1fr 1fr; }.stage-main>div { align-items:flex-start;flex-direction:column;gap:2px; }.chart { height:250px; }.trend-tools { flex-wrap:wrap; }.trend-tools :deep(.el-segmented) { flex:1; } }
@media (max-width:520px) { .metrics-grid { grid-template-columns:1fr; }.header-actions .el-button:not(.icon-button) { flex:1; }.metric-card { min-height:102px; } }
</style>
