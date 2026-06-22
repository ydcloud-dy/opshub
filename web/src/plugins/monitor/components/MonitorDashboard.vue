<template>
  <div class="monitor-dashboard-page">
    <section class="dashboard-hero">
      <div class="dashboard-title-group">
        <div class="dashboard-title-icon">
          <el-icon><DataAnalysis /></el-icon>
        </div>
        <div>
          <h2>监控面板</h2>
          <p>实时监控系统状态，聚合规则、故障中心、数据源和告警事件</p>
        </div>
      </div>
      <div class="hero-clock">
        <span>{{ todayText }}</span>
        <strong>{{ clockText }}</strong>
      </div>
    </section>

    <section class="dashboard-stat-grid">
      <article class="dashboard-stat-card">
        <div class="stat-icon stat-blue"><el-icon><Warning /></el-icon></div>
        <div>
          <span>告警规则</span>
          <strong>{{ stats.totalRules || rules.length }}</strong>
          <em>{{ stats.enabledRules || enabledRuleCount }} 条启用</em>
        </div>
      </article>
      <article class="dashboard-stat-card">
        <div class="stat-icon stat-amber"><el-icon><FirstAidKit /></el-icon></div>
        <div>
          <span>故障中心</span>
          <strong>{{ faultCenters.length }}</strong>
          <em>{{ activeFaultCenterCount }} 个存在活跃事件</em>
        </div>
      </article>
      <article class="dashboard-stat-card">
        <div class="stat-icon stat-green"><el-icon><Connection /></el-icon></div>
        <div>
          <span>数据源</span>
          <strong>{{ dataSources.length }}</strong>
          <em>{{ normalSourceCount }} 个连接正常</em>
        </div>
      </article>
      <article class="dashboard-stat-card distribution-card">
        <div class="stat-icon stat-red"><el-icon><Bell /></el-icon></div>
        <div>
          <span>告警分布</span>
          <div class="stat-distribution">
            <em v-for="item in severityDistribution" :key="item.name">
              <i :class="`dot-${item.name.toLowerCase()}`"></i>
              {{ item.name }}
              <b>{{ item.value }}</b>
            </em>
          </div>
        </div>
      </article>
    </section>

    <section class="dashboard-main-grid">
      <article class="dashboard-panel trend-panel">
        <div class="panel-head">
          <div>
            <h3>告警通知趋势</h3>
            <p>按最近 7 天和等级聚合事件数量</p>
          </div>
          <el-button @click="loadAll">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
        <div ref="trendChartRef" class="trend-chart"></div>
      </article>

      <article class="dashboard-panel active-panel">
        <div class="panel-head stacked">
          <div>
            <h3>最近活跃告警</h3>
            <p>实时事件监控 · {{ stats.unresolvedEvents || activeEvents.length }} 条未恢复</p>
          </div>
          <el-select v-model="selectedFaultCenterId" clearable placeholder="全部故障中心" @change="loadActiveEvents">
            <el-option v-for="center in faultCenters" :key="center.id" :label="center.name" :value="center.id" />
          </el-select>
        </div>
        <div v-loading="activeLoading" class="active-list">
          <div v-if="!activeEvents.length" class="empty-active">暂无活跃告警</div>
          <article v-for="event in activeEvents" :key="event.id" class="active-event">
            <el-tag :type="getSeverityTag(event.severity)" effect="dark">{{ getSeverityName(event.severity) }}</el-tag>
            <div>
              <strong>{{ event.ruleName || '-' }}</strong>
              <p>{{ getFaultCenterName(event.faultCenterId) }} · {{ formatDateTime(event.startedAt) }}</p>
            </div>
          </article>
        </div>
      </article>
    </section>

    <section class="dashboard-bottom-grid">
      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <h3>数据源状态</h3>
            <p>接入端连接健康度</p>
          </div>
        </div>
        <div class="source-status-list">
          <div v-for="source in dataSources.slice(0, 6)" :key="source.id" class="source-status-item">
            <span>{{ source.name }}</span>
            <em>{{ getTypeName(source.type) }}</em>
            <el-tag :type="source.status === 'normal' ? 'success' : source.status === 'abnormal' ? 'danger' : 'info'" effect="light">
              {{ source.status === 'normal' ? '正常' : source.status === 'abnormal' ? '异常' : '未测试' }}
            </el-tag>
          </div>
          <div v-if="!dataSources.length" class="empty-active">暂无数据源</div>
        </div>
      </article>

      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <h3>等级分布</h3>
            <p>近期事件按 P0/P1/P2 统计</p>
          </div>
        </div>
        <div class="severity-distribution">
          <div v-for="item in severityDistribution" :key="item.name" class="severity-row">
            <span><i :class="`dot-${item.name.toLowerCase()}`"></i>{{ item.name }}</span>
            <strong>{{ item.value }}</strong>
            <em><b :style="{ width: `${item.percent}%` }"></b></em>
          </div>
        </div>
      </article>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import {
  Bell,
  Connection,
  DataAnalysis,
  FirstAidKit,
  Refresh,
  Warning
} from '@element-plus/icons-vue'
import {
  getMonitorAlertEventStats,
  getMonitorAlertEvents,
  getMonitorAlertRules,
  getMonitorDataSources,
  getMonitorFaultCenters,
  type MonitorAlertEvent,
  type MonitorAlertEventStats,
  type MonitorAlertRule,
  type MonitorDataSource,
  type MonitorFaultCenter
} from '@/api/monitor-datasource'

const rules = ref<MonitorAlertRule[]>([])
const dataSources = ref<MonitorDataSource[]>([])
const faultCenters = ref<MonitorFaultCenter[]>([])
const activeEvents = ref<MonitorAlertEvent[]>([])
const recentEvents = ref<MonitorAlertEvent[]>([])
const activeLoading = ref(false)
const selectedFaultCenterId = ref<number>()
const trendChartRef = ref<HTMLElement>()
const now = ref(new Date())
let trendChart: echarts.ECharts | null = null
let clockTimer: ReturnType<typeof setInterval> | undefined

const stats = ref<MonitorAlertEventStats>({
  totalRules: 0,
  enabledRules: 0,
  firingRules: 0,
  pendingRules: 0,
  todayEvents: 0,
  unresolvedEvents: 0
})

const enabledRuleCount = computed(() => rules.value.filter(item => item.enabled).length)
const normalSourceCount = computed(() => dataSources.value.filter(item => item.status === 'normal').length)
const activeFaultCenterCount = computed(() => new Set(activeEvents.value.map(item => item.faultCenterId).filter(Boolean)).size)
const todayEventCount = computed(() => {
  const today = formatDate(new Date())
  return recentEvents.value.filter(item => String(item.startedAt || '').startsWith(today)).length
})

const todayText = computed(() => {
  const date = now.value
  const week = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'][date.getDay()]
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日${week}`
})

const clockText = computed(() => {
  const date = now.value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
})

const severityDistribution = computed(() => {
  const counts = { P0: 0, P1: 0, P2: 0 }
  recentEvents.value.forEach(event => {
    const key = normalizeSeverity(event.severity)
    counts[key] += 1
  })
  const max = Math.max(1, ...Object.values(counts))
  return Object.entries(counts).map(([name, value]) => ({ name, value, percent: Math.round(value / max * 100) }))
})

const normalizeSeverity = (severity?: string): 'P0' | 'P1' | 'P2' => {
  const value = String(severity || '').toLowerCase()
  if (value === 'p0' || value === 'critical') return 'P0'
  if (value === 'p2' || value === 'info') return 'P2'
  return 'P1'
}

const getSeverityName = (severity?: string) => normalizeSeverity(severity)
const getSeverityTag = (severity?: string) => {
  const value = normalizeSeverity(severity)
  if (value === 'P0') return 'danger'
  if (value === 'P1') return 'warning'
  return 'success'
}

const getTypeName = (type?: string) => {
  const map: Record<string, string> = {
    prometheus: 'Prometheus',
    victoriametrics: 'VictoriaMetrics',
    loki: 'Loki',
    elasticsearch: 'Elasticsearch'
  }
  return type ? map[type] || type : '-'
}

const getFaultCenterName = (id?: number) => {
  return faultCenters.value.find(item => item.id === id)?.name || '-'
}

const getEventTimeValue = (value?: string) => {
  const date = value ? new Date(value) : undefined
  return date && !Number.isNaN(date.getTime()) ? date.getTime() : 0
}

const sortEventsByStartedAtDesc = (events: MonitorAlertEvent[]) => {
  return [...events].sort((a, b) => {
    const diff = getEventTimeValue(b.startedAt) - getEventTimeValue(a.startedAt)
    if (diff !== 0) return diff
    return (b.id || 0) - (a.id || 0)
  })
}

const formatDate = (date: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

const formatDateTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const recentDays = () => {
  return Array.from({ length: 7 }).map((_, index) => {
    const date = new Date()
    date.setDate(date.getDate() - (6 - index))
    return formatDate(date)
  })
}

const renderTrendChart = async () => {
  await nextTick()
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)
  const days = recentDays()
  const seriesData = {
    P0: days.map(() => 0),
    P1: days.map(() => 0),
    P2: days.map(() => 0)
  }
  recentEvents.value.forEach(event => {
    const day = String(event.startedAt || '').slice(0, 10)
    const index = days.indexOf(day)
    if (index < 0) return
    seriesData[normalizeSeverity(event.severity)][index] += 1
  })
  trendChart.setOption({
    color: ['#ef4444', '#f59e0b', '#22c55e'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#fff',
      borderColor: '#e5e9f2',
      borderWidth: 1,
      textStyle: { color: '#111827' },
      extraCssText: 'box-shadow: 0 12px 28px rgba(15, 23, 42, .12); border-radius: 8px;'
    },
    grid: { left: 42, right: 24, top: 28, bottom: 46 },
    legend: {
      bottom: 0,
      icon: 'circle',
      textStyle: { color: '#667085' }
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: days.map(day => day.slice(5)),
      axisLine: { lineStyle: { color: '#d8dee9' } },
      axisTick: { show: false },
      axisLabel: { color: '#667085' }
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: '#667085' },
      splitLine: { lineStyle: { color: '#edf1f7' } }
    },
    series: [
      { name: 'P0', type: 'line', smooth: true, symbolSize: 7, lineStyle: { width: 2 }, areaStyle: { opacity: 0.18 }, data: seriesData.P0 },
      { name: 'P1', type: 'line', smooth: true, symbolSize: 7, lineStyle: { width: 2 }, areaStyle: { opacity: 0.2 }, data: seriesData.P1 },
      { name: 'P2', type: 'line', smooth: true, symbolSize: 7, lineStyle: { width: 2 }, areaStyle: { opacity: 0.08 }, data: seriesData.P2 }
    ]
  })
  trendChart.resize()
}

const loadActiveEvents = async () => {
  activeLoading.value = true
  try {
    const result = await getMonitorAlertEvents({
      page: 1,
      pageSize: 8,
      scope: 'active',
      faultCenterId: selectedFaultCenterId.value,
      sort: 'started_at'
    })
    activeEvents.value = sortEventsByStartedAtDesc(result?.list || [])
  } finally {
    activeLoading.value = false
  }
}

const loadAll = async () => {
  const start = new Date()
  start.setDate(start.getDate() - 6)
  const startDate = formatDate(start)
  const endDate = formatDate(new Date())
  const [sourceResult, ruleResult, centerResult, statsResult, eventResult] = await Promise.all([
    getMonitorDataSources(),
    getMonitorAlertRules(),
    getMonitorFaultCenters(),
    getMonitorAlertEventStats().catch(() => null),
    getMonitorAlertEvents({ page: 1, pageSize: 500, startDate, endDate, sort: 'started_at' })
  ])
  dataSources.value = sourceResult || []
  rules.value = ruleResult || []
  faultCenters.value = centerResult || []
  if (statsResult) stats.value = statsResult
  recentEvents.value = sortEventsByStartedAtDesc(eventResult?.list || [])
  await loadActiveEvents()
  renderTrendChart()
}

const handleResize = () => {
  trendChart?.resize()
}

onMounted(() => {
  loadAll()
  clockTimer = setInterval(() => {
    now.value = new Date()
  }, 1000)
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
})
</script>

<style scoped>
.monitor-dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0;
  background: transparent;
  color: #111827;
}

.dashboard-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.dashboard-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.dashboard-title-icon {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 22px;
}

.dashboard-hero h2 {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.3;
}

.dashboard-hero p {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
  font-weight: 400;
}

.hero-clock {
  text-align: right;
  color: #344054;
  white-space: nowrap;
}

.hero-clock span {
  display: block;
  color: #667085;
  font-size: 13px;
  font-weight: 500;
}

.hero-clock strong {
  display: block;
  margin-top: 6px;
  color: #111827;
  font-size: 26px;
  font-weight: 780;
  line-height: 1;
}

.dashboard-stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.dashboard-stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 108px;
  padding: 18px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.stat-icon {
  width: 46px;
  height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  font-size: 23px;
}

.stat-blue { background: #eef4ff; border: 1px solid #dbe8ff; color: #2563eb; }
.stat-amber { background: #fffbeb; border: 1px solid #fde68a; color: #d97706; }
.stat-green { background: #ecfdf3; border: 1px solid #bbf7d0; color: #16a34a; }
.stat-red { background: #fff1f2; border: 1px solid #fecdd3; color: #e11d48; }

.dashboard-stat-card span,
.dashboard-stat-card em {
  display: block;
  color: #667085;
  font-size: 13px;
  font-style: normal;
  font-weight: 500;
}

.dashboard-stat-card strong {
  display: block;
  margin: 7px 0;
  color: #111827;
  font-size: 32px;
  font-weight: 780;
  line-height: 1;
}

.stat-distribution {
  display: grid;
  gap: 8px;
  margin-top: 7px;
  min-width: 132px;
}

.stat-distribution em {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #344054;
  font-size: 13px;
}

.stat-distribution b {
  margin-left: auto;
  color: #111827;
  font-size: 14px;
  font-weight: 760;
}

.dashboard-main-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(340px, 1fr);
  gap: 12px;
}

.dashboard-bottom-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.dashboard-panel {
  min-width: 0;
  padding: 18px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}

.panel-head.stacked {
  align-items: stretch;
  flex-direction: column;
}

.panel-head h3 {
  margin: 0;
  color: #111827;
  font-size: 17px;
  font-weight: 780;
}

.panel-head p {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
  font-weight: 400;
}

.trend-chart {
  height: 356px;
}

.active-list,
.source-status-list,
.severity-distribution {
  display: grid;
  gap: 10px;
}

.active-event,
.source-status-item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 12px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.active-event div {
  min-width: 0;
}

.active-event strong,
.source-status-item span {
  display: block;
  overflow: hidden;
  color: #111827;
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.active-event p,
.source-status-item em {
  margin: 4px 0 0;
  color: #667085;
  font-size: 12px;
  font-style: normal;
}

.source-status-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 140px auto;
}

.severity-row {
  display: grid;
  grid-template-columns: 80px 44px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  color: #344054;
  font-size: 14px;
}

.severity-row span {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
}

.severity-row i {
  width: 8px;
  height: 8px;
  display: inline-block;
  border-radius: 999px;
}

.dot-p0 { background: #ef4444; }
.dot-p1 { background: #f59e0b; }
.dot-p2 { background: #22c55e; }

.severity-row strong {
  color: #111827;
  font-size: 15px;
}

.severity-row em {
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: #f2f4f7;
}

.severity-row b {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #1677ff;
}

.empty-active {
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #98a2b3;
  border: 1px dashed #d0d5dd;
  border-radius: 8px;
  background: #fcfcfd;
}

@media (max-width: 1100px) {
  .dashboard-stat-grid,
  .dashboard-main-grid,
  .dashboard-bottom-grid {
    grid-template-columns: 1fr;
  }

  .monitor-dashboard-page {
    padding: 0;
  }

  .dashboard-hero {
    align-items: flex-start;
    flex-direction: column;
  }

  .hero-clock {
    text-align: left;
  }
}
</style>
