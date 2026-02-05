<template>
  <div class="nginx-overview-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DataLine /></el-icon>
        </div>
        <div>
          <h2 class="page-title">访问概况</h2>
          <p class="page-subtitle">按站点查看的访问概况面板</p>
          <p v-if="lastRefreshTime" class="last-refresh-time">
            <el-icon><Clock /></el-icon>
            上次刷新: {{ lastRefreshTime }}
          </p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedSourceId"
          placeholder="请选择站点"
          style="width: 240px"
          @change="onSourceChange"
        >
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-button @click="refreshAll" :loading="loading">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 无站点提示 -->
    <div v-if="!selectedSourceId" class="empty-tip">
      <el-empty description="请选择站点以查看访问概况" />
    </div>

    <template v-else>
      <!-- 区域1+2: 活跃访客 + 核心指标 -->
      <div class="top-section">
        <!-- 左侧: 活跃访客卡片 -->
        <div class="active-visitors-card">
          <div class="av-number">{{ formatNumber(activeVisitors) }}</div>
          <div class="av-label">15分钟活跃访客</div>
          <div class="av-status">活动保持中</div>
        </div>

        <!-- 右侧: 核心指标面板 -->
        <div class="core-metrics-panel" v-loading="loadingMetrics" element-loading-text="加载中...">
          <div class="metrics-grid">
            <div class="metric-column">
              <div class="metric-header">HTTP状态码命中</div>
              <div class="metric-main">{{ formatNumber(coreMetrics.today.statusHits) }}</div>
              <div class="metric-sub">
                <span class="sub-label">昨日</span>
                <span class="sub-value">{{ formatNumber(coreMetrics.yesterday.statusHits) }}</span>
                <span :class="changeClass(coreMetrics.today.statusHits, coreMetrics.yesterday.statusHits)">
                  {{ changePercent(coreMetrics.today.statusHits, coreMetrics.yesterday.statusHits) }}
                </span>
              </div>
              <div class="metric-sub">
                <span class="sub-label">2xx</span>
                <span class="sub-value">{{ formatNumber(coreMetrics.today.status2xx) }}</span>
              </div>
              <div class="metric-sub">
                <span class="sub-label">3xx</span>
                <span class="sub-value">{{ formatNumber(coreMetrics.today.status3xx) }}</span>
              </div>
              <div class="metric-sub">
                <span class="sub-label">4xx</span>
                <span class="sub-value status-4xx">{{ formatNumber(coreMetrics.today.status4xx) }}</span>
              </div>
              <div class="metric-sub">
                <span class="sub-label">5xx</span>
                <span class="sub-value status-5xx">{{ formatNumber(coreMetrics.today.status5xx) }}</span>
              </div>
            </div>
            <div class="metric-column">
              <div class="metric-header">PV浏览量</div>
              <div class="metric-main">{{ formatNumber(coreMetrics.today.pv) }}</div>
              <div class="metric-sub">
                <span class="sub-label">预计今日</span>
                <span :class="predictClass(coreMetrics.predictToday.pv, coreMetrics.yesterday.pv)">
                  {{ formatNumber(coreMetrics.predictToday.pv) }}
                  {{ predictArrow(coreMetrics.predictToday.pv, coreMetrics.yesterday.pv) }}
                </span>
              </div>
            </div>
            <div class="metric-column">
              <div class="metric-header">UV访客数</div>
              <div class="metric-main">{{ formatNumber(coreMetrics.today.uv) }}</div>
              <div class="metric-sub">
                <span class="sub-label">昨日</span>
                <span class="sub-value">{{ formatNumber(coreMetrics.yesterday.uv) }}</span>
              </div>
              <div class="metric-sub">
                <span class="sub-label">预计今日</span>
                <span :class="predictClass(coreMetrics.predictToday.uv, coreMetrics.yesterday.uv)">
                  {{ formatNumber(coreMetrics.predictToday.uv) }}
                  {{ predictArrow(coreMetrics.predictToday.uv, coreMetrics.yesterday.uv) }}
                </span>
              </div>
            </div>
            <div class="metric-column">
              <div class="metric-header">OPS</div>
              <div class="metric-main">{{ formatQPS(coreMetrics.today.realtimeOps) }}</div>
              <div class="metric-sub">
                <span class="sub-label">实时OPS</span>
                <span class="sub-value">{{ formatQPS(coreMetrics.today.realtimeOps) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 区域3+4: 趋势分析 + 新老访客 -->
      <div class="middle-section">
        <!-- 左侧: 趋势分析 -->
        <div class="chart-card trend-card" v-loading="loadingTrend" element-loading-text="加载中...">
          <div class="chart-header">
            <h3 class="chart-title">趋势分析</h3>
            <el-radio-group v-model="trendMode" size="small" @change="onTrendModeChange">
              <el-radio-button label="hour">按时</el-radio-button>
              <el-radio-button label="day">按天</el-radio-button>
            </el-radio-group>
          </div>
          <div class="chart-content" ref="trendChartRef"></div>
        </div>

        <!-- 右侧: 新老访客 -->
        <div class="chart-card visitor-card" v-loading="loadingVisitors" element-loading-text="加载中...">
          <div class="chart-header">
            <h3 class="chart-title">新老访客</h3>
          </div>
          <div class="chart-content" ref="visitorChartRef"></div>
          <div class="visitor-stats">
            <div class="visitor-stat-row">
              <div class="visitor-stat-item">
                <span class="vs-label">今日新访客</span>
                <span class="vs-value">{{ visitorData.todayNew }}</span>
                <span class="vs-pct">({{ visitorData.todayNewPct.toFixed(1) }}%)</span>
              </div>
              <div class="visitor-stat-item">
                <span class="vs-label">今日老访客</span>
                <span class="vs-value">{{ visitorData.todayReturning }}</span>
                <span class="vs-pct">({{ visitorData.todayRetPct.toFixed(1) }}%)</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 区域5: 来路 | 受访页 | 入口页 -->
      <div class="three-column-section">
        <div class="list-card" v-loading="loadingReferers" element-loading-text="加载中...">
          <div class="list-header">
            <h3 class="list-title">来路域名</h3>
          </div>
          <div class="list-body">
            <div v-if="topReferers.length === 0" class="list-empty">暂无数据</div>
            <div v-for="(item, index) in topReferers" :key="index" class="list-item">
              <span class="list-item-name" :title="item.domain">{{ item.domain }}</span>
              <span class="list-item-value">{{ item.visitors }}</span>
              <div class="list-item-bar">
                <div class="list-item-bar-fill" :style="{ width: getBarWidth(item.visitors, topReferers) + '%' }"></div>
              </div>
            </div>
          </div>
        </div>
        <div class="list-card" v-loading="loadingPages" element-loading-text="加载中...">
          <div class="list-header">
            <h3 class="list-title">受访页面</h3>
          </div>
          <div class="list-body">
            <div v-if="topPages.length === 0" class="list-empty">暂无数据</div>
            <div v-for="(item, index) in topPages" :key="index" class="list-item">
              <span class="list-item-name" :title="item.path">{{ item.path }}</span>
              <span class="list-item-value">{{ item.count }}</span>
              <div class="list-item-bar">
                <div class="list-item-bar-fill" :style="{ width: getBarWidthPage(item.count, topPages) + '%' }"></div>
              </div>
            </div>
          </div>
        </div>
        <div class="list-card" v-loading="loadingEntryPages" element-loading-text="加载中...">
          <div class="list-header">
            <h3 class="list-title">入口页面</h3>
          </div>
          <div class="list-body">
            <div v-if="topEntryPages.length === 0" class="list-empty">暂无数据</div>
            <div v-for="(item, index) in topEntryPages" :key="index" class="list-item">
              <span class="list-item-name" :title="item.path">{{ item.path }}</span>
              <span class="list-item-value">{{ item.count }}</span>
              <div class="list-item-bar">
                <div class="list-item-bar-fill" :style="{ width: getBarWidthPage(item.count, topEntryPages) + '%' }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 区域6+7: 地域分布 + 终端设备 -->
      <div class="bottom-section">
        <!-- 左侧: 地域分布 -->
        <div class="chart-card geo-card" v-loading="loadingGeo" element-loading-text="加载中...">
          <div class="chart-header">
            <h3 class="chart-title">地域分布</h3>
          </div>
          <div class="geo-list">
            <div v-if="geoData.length === 0" class="list-empty">暂无数据</div>
            <div v-for="(item, index) in geoData" :key="index" class="geo-item">
              <span class="geo-rank">{{ index + 1 }}</span>
              <span class="geo-name">{{ item.country || '-' }}</span>
              <span class="geo-count">{{ item.count }}</span>
              <span class="geo-pct">{{ item.percent.toFixed(1) }}%</span>
            </div>
          </div>
        </div>

        <!-- 右侧: 终端设备 -->
        <div class="chart-card device-card" v-loading="loadingDevices" element-loading-text="加载中...">
          <div class="chart-header">
            <h3 class="chart-title">终端设备</h3>
          </div>
          <div class="chart-content" ref="deviceChartRef"></div>
          <div class="device-stats">
            <div v-for="(item, index) in deviceData" :key="index" class="device-stat-item">
              <span class="ds-color" :style="{ background: deviceColors[index % deviceColors.length] }"></span>
              <span class="ds-label">{{ item.deviceType }}</span>
              <span class="ds-value">{{ item.count }}</span>
              <span class="ds-pct">({{ item.percent.toFixed(1) }}%)</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { DataLine, Refresh, Clock } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import {
  getNginxSources,
  getActiveVisitors,
  getCoreMetrics,
  getOverviewTrend,
  getNewVsReturning,
  getTopReferers,
  getTopPages,
  getTopEntryPages,
  getOverviewGeo,
  getOverviewDevices,
  collectNginxLogs,
  type NginxSource,
  type CoreMetrics,
  type VisitorComparison,
  type OverviewTrendPoint,
  type RefererItem,
  type PageItem,
  type GeoStats,
  type DeviceStats,
} from '@/api/nginx'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const sources = ref<NginxSource[]>([])
const selectedSourceId = ref<number | null>(null)
const activeVisitors = ref(0)
const lastRefreshTime = ref('')

// 各模块单独的加载状态
const loadingMetrics = ref(false)
const loadingTrend = ref(false)
const loadingVisitors = ref(false)
const loadingReferers = ref(false)
const loadingPages = ref(false)
const loadingEntryPages = ref(false)
const loadingGeo = ref(false)
const loadingDevices = ref(false)

// sessionStorage key
const STORAGE_KEY = 'nginx-overview-state'
const STORAGE_DATA_KEY = 'nginx-overview-data'

// 从 sessionStorage 恢复状态
const restoreState = () => {
  try {
    const saved = sessionStorage.getItem(STORAGE_KEY)
    if (saved) {
      return JSON.parse(saved)
    }
  } catch (e) {
    console.error('恢复状态失败:', e)
  }
  return null
}

// 保存状态到 sessionStorage
const saveState = (state: any) => {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch (e) {
    console.error('保存状态失败:', e)
  }
}

// 从 sessionStorage 恢复数据
const restoreCachedData = () => {
  try {
    if (!selectedSourceId.value) return null
    const key = `${STORAGE_DATA_KEY}-${selectedSourceId.value}`
    const saved = sessionStorage.getItem(key)
    if (saved) {
      const data = JSON.parse(saved)
      // 检查缓存时间戳，如果数据是当前会话的，则使用
      if (data.timestamp && Date.now() - data.timestamp < 24 * 60 * 60 * 1000) {
        return data
      }
    }
  } catch (e) {
    console.error('恢复缓存数据失败:', e)
  }
  return null
}

// 保存数据到 sessionStorage
const saveCachedData = () => {
  try {
    if (!selectedSourceId.value) return
    const key = `${STORAGE_DATA_KEY}-${selectedSourceId.value}`
    const refreshTime = formatRefreshTime(new Date())
    lastRefreshTime.value = refreshTime
    const data = {
      timestamp: Date.now(),
      selectedSourceId: selectedSourceId.value,
      lastRefreshTime: refreshTime,
      activeVisitors: activeVisitors.value,
      coreMetrics: coreMetrics.value,
      trendMode: trendMode.value,
      trendData: trendData.value,
      visitorData: visitorData.value,
      topReferers: topReferers.value,
      topPages: topPages.value,
      topEntryPages: topEntryPages.value,
      geoData: geoData.value,
      deviceData: deviceData.value,
    }
    sessionStorage.setItem(key, JSON.stringify(data))
  } catch (e) {
    console.error('保存缓存数据失败:', e)
  }
}

// 格式化刷新时间
const formatRefreshTime = (date: Date) => {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const emptyMetricSet = () => ({ statusHits: 0, pv: 0, uv: 0, realtimeOps: 0, peakOps: 0, status2xx: 0, status3xx: 0, status4xx: 0, status5xx: 0 })
const coreMetrics = ref<CoreMetrics>({
  today: emptyMetricSet(),
  yesterday: emptyMetricSet(),
  predictToday: emptyMetricSet(),
  yesterdayNow: emptyMetricSet(),
})

const trendMode = ref<'hour' | 'day'>('hour')
const trendData = ref<OverviewTrendPoint[]>([])
const visitorData = ref<VisitorComparison>({
  todayNew: 0, todayReturning: 0, todayNewPct: 0, todayRetPct: 0,
  yesterdayNew: 0, yesterdayReturning: 0, yesterdayNewPct: 0, yesterdayRetPct: 0,
})
const topReferers = ref<RefererItem[]>([])
const topPages = ref<PageItem[]>([])
const topEntryPages = ref<PageItem[]>([])
const geoData = ref<GeoStats[]>([])
const deviceData = ref<DeviceStats[]>([])
const deviceColors = ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de']

const trendChartRef = ref<HTMLElement | null>(null)
const visitorChartRef = ref<HTMLElement | null>(null)
const deviceChartRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null
let visitorChart: echarts.ECharts | null = null
let deviceChart: echarts.ECharts | null = null

// 格式化数字
const formatNumber = (num: number) => {
  if (!num) return '0'
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

// 格式化QPS（每秒查询率）
const formatQPS = (num: number) => {
  if (!num) return '0'
  // QPS通常比较小，直接显示两位小数
  if (num >= 1000) return num.toFixed(0)
  if (num >= 100) return num.toFixed(1)
  return num.toFixed(2)
}

// 预测箭头
const predictArrow = (predict: number, yesterday: number) => {
  if (!yesterday || !predict) return ''
  const pct = ((predict - yesterday) / yesterday * 100).toFixed(1)
  return predict >= yesterday ? `+${pct}%` : `${pct}%`
}

const predictClass = (predict: number, yesterday: number) => {
  if (!yesterday) return 'sub-value'
  return predict >= yesterday ? 'sub-value predict-up' : 'sub-value predict-down'
}

// 变化百分比（今日 vs 昨日）
const changePercent = (today: number, yesterday: number) => {
  if (!yesterday) return ''
  const pct = ((today - yesterday) / yesterday * 100).toFixed(2)
  if (today >= yesterday) {
    return `↑${pct}%`
  } else {
    return `↓${Math.abs(parseFloat(pct)).toFixed(2)}%`
  }
}

const changeClass = (today: number, yesterday: number) => {
  if (!yesterday) return ''
  return today >= yesterday ? 'change-up' : 'change-down'
}

// 条形图宽度
const getBarWidth = (value: number, list: RefererItem[]) => {
  if (!list.length) return 0
  const max = Math.max(...list.map(i => i.visitors))
  return max > 0 ? (value / max * 100) : 0
}

const getBarWidthPage = (value: number, list: PageItem[]) => {
  if (!list.length) return 0
  const max = Math.max(...list.map(i => i.count))
  return max > 0 ? (value / max * 100) : 0
}

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ pageSize: 100 })
    sources.value = res?.list || []

    // 恢复之前保存的状态
    const savedState = restoreState()
    if (savedState && savedState.selectedSourceId) {
      // 检查保存的数据源是否仍在列表中
      const exists = sources.value.some((s: NginxSource) => s.id === savedState.selectedSourceId)
      if (exists) {
        selectedSourceId.value = savedState.selectedSourceId
      } else if (sources.value.length > 0) {
        selectedSourceId.value = sources.value[0].id!
      }
    } else if (sources.value.length > 0 && !selectedSourceId.value) {
      selectedSourceId.value = sources.value[0].id!
    }

    // 尝试恢复缓存的数据（缓存键包含数据源ID，会自动获取正确的缓存）
    const cachedData = restoreCachedData()
    if (cachedData) {
      // 使用缓存数据
      activeVisitors.value = cachedData.activeVisitors || 0
      lastRefreshTime.value = cachedData.lastRefreshTime || ''
      coreMetrics.value = cachedData.coreMetrics || {
        today: emptyMetricSet(),
        yesterday: emptyMetricSet(),
        predictToday: emptyMetricSet(),
        yesterdayNow: emptyMetricSet(),
      }
      trendMode.value = cachedData.trendMode || 'hour'
      trendData.value = cachedData.trendData || []
      visitorData.value = cachedData.visitorData || {
        todayNew: 0, todayReturning: 0, todayNewPct: 0, todayRetPct: 0,
        yesterdayNew: 0, yesterdayReturning: 0, yesterdayNewPct: 0, yesterdayRetPct: 0,
      }
      topReferers.value = cachedData.topReferers || []
      topPages.value = cachedData.topPages || []
      topEntryPages.value = cachedData.topEntryPages || []
      geoData.value = cachedData.geoData || []
      deviceData.value = cachedData.deviceData || []

      // 使用缓存数据后，需要重新渲染图表
      await nextTick()
      initTrendChart()
      initVisitorChart()
      initDeviceChart()

      ElMessage.success('已加载缓存数据')
    }
    // 没有缓存数据时不自动加载，等待用户点击刷新
  } catch (error) {
    console.error('获取数据源列表失败:', error)
  }
}

// 站点切换
const onSourceChange = async (newSourceId: number) => {
  // 先更新 selectedSourceId
  selectedSourceId.value = newSourceId

  // 保存状态
  saveState({ selectedSourceId: newSourceId })

  // 先尝试从缓存加载数据（缓存键包含数据源ID，所以会自动获取正确的缓存）
  const cachedData = restoreCachedData()
  if (cachedData) {
    // 使用缓存数据
    activeVisitors.value = cachedData.activeVisitors || 0
    lastRefreshTime.value = cachedData.lastRefreshTime || ''
    coreMetrics.value = cachedData.coreMetrics || {
      today: emptyMetricSet(),
      yesterday: emptyMetricSet(),
      predictToday: emptyMetricSet(),
      yesterdayNow: emptyMetricSet(),
    }
    trendMode.value = cachedData.trendMode || 'hour'
    trendData.value = cachedData.trendData || []
    visitorData.value = cachedData.visitorData || {
      todayNew: 0, todayReturning: 0, todayNewPct: 0, todayRetPct: 0,
      yesterdayNew: 0, yesterdayReturning: 0, yesterdayNewPct: 0, yesterdayRetPct: 0,
    }
    topReferers.value = cachedData.topReferers || []
    topPages.value = cachedData.topPages || []
    topEntryPages.value = cachedData.topEntryPages || []
    geoData.value = cachedData.geoData || []
    deviceData.value = cachedData.deviceData || []

    // 确保 coreMetrics 中的所有字段都有值
    if (coreMetrics.value.today) {
      coreMetrics.value.today = {
        ...emptyMetricSet(),
        ...coreMetrics.value.today
      }
    }
    if (coreMetrics.value.yesterday) {
      coreMetrics.value.yesterday = {
        ...emptyMetricSet(),
        ...coreMetrics.value.yesterday
      }
    }
    if (coreMetrics.value.predictToday) {
      coreMetrics.value.predictToday = {
        ...emptyMetricSet(),
        ...coreMetrics.value.predictToday
      }
    }
    if (coreMetrics.value.yesterdayNow) {
      coreMetrics.value.yesterdayNow = {
        ...emptyMetricSet(),
        ...coreMetrics.value.yesterdayNow
      }
    }

    // 使用缓存数据后，需要重新渲染图表
    await nextTick()
    initTrendChart()
    initVisitorChart()
    initDeviceChart()

    ElMessage.success('已加载缓存数据')
  } else {
    // 没有缓存，清空数据，等待用户点击刷新
    activeVisitors.value = 0
    lastRefreshTime.value = ''
    coreMetrics.value = {
      today: emptyMetricSet(),
      yesterday: emptyMetricSet(),
      predictToday: emptyMetricSet(),
      yesterdayNow: emptyMetricSet(),
    }
    trendData.value = []
    visitorData.value = {
      todayNew: 0, todayReturning: 0, todayNewPct: 0, todayRetPct: 0,
      yesterdayNew: 0, yesterdayReturning: 0, yesterdayNewPct: 0, yesterdayRetPct: 0,
    }
    topReferers.value = []
    topPages.value = []
    topEntryPages.value = []
    geoData.value = []
    deviceData.value = []

    // 清空图表
    await nextTick()
    initTrendChart()
    initVisitorChart()
    initDeviceChart()

    ElMessage.info('请点击刷新按钮加载数据')
  }
}

// 并发加载所有数据
const loadAllData = async (sourceId?: number) => {
  // 如果没有传入sourceId，使用selectedSourceId.value
  const sid = sourceId ?? selectedSourceId.value
  if (!sid) return
  loading.value = true
  try {
    await Promise.all([
      loadActiveVisitors(sid),
      loadCoreMetrics(sid),
      loadTrend(sid),
      loadVisitors(sid),
      loadReferers(sid),
      loadPages(sid),
      loadEntryPages(sid),
      loadGeo(sid),
      loadDevices(sid),
    ])

    // 保存数据到缓存
    saveCachedData()

    ElMessage.success('数据加载成功')
  } catch (error) {
    console.error('加载数据失败:', error)
    ElMessage.error('数据加载失败')
  } finally {
    loading.value = false
  }
}

const refreshAll = async () => {
  if (!selectedSourceId.value) return
  loading.value = true
  try {
    // 触发日志采集（不等待完成，后台执行）
    collectNginxLogs(selectedSourceId.value).then(() => {
      // 采集完成后自动刷新数据
      loadAllData()
    }).catch(err => {
      console.error('采集失败:', err)
    })

    // 提示用户
    ElMessage.info('正在后台采集日志，数据会逐步更新...')

    // 立即加载当前已有数据
    await loadAllData()
  } catch (error) {
    console.error('刷新失败:', error)
  } finally {
    loading.value = false
  }
}

const loadActiveVisitors = async (sid: number) => {
  try {
    const res = await getActiveVisitors(sid)
    activeVisitors.value = res?.count || 0
  } catch { activeVisitors.value = 0 }
}

const loadCoreMetrics = async (sid: number) => {
  loadingMetrics.value = true
  try {
    const res = await getCoreMetrics(sid)
    if (res) {
      coreMetrics.value = res
    }
  } catch {
    coreMetrics.value = {
      today: emptyMetricSet(), yesterday: emptyMetricSet(),
      predictToday: emptyMetricSet(), yesterdayNow: emptyMetricSet(),
    }
  } finally {
    loadingMetrics.value = false
  }
}

const loadTrend = async (sid?: number) => {
  const sourceId = sid ?? selectedSourceId.value
  if (!sourceId) return
  loadingTrend.value = true
  try {
    const res = await getOverviewTrend({
      sourceId: sourceId,
      mode: trendMode.value,
    })
    trendData.value = res || []
    await nextTick()
    initTrendChart()
  } catch {
    trendData.value = []
  } finally {
    loadingTrend.value = false
  }
}

// 趋势模式切换（按时/按天）
const onTrendModeChange = () => {
  if (!selectedSourceId.value) return
  loadTrend(selectedSourceId.value)
}

const loadVisitors = async (sid: number) => {
  loadingVisitors.value = true
  try {
    const res = await getNewVsReturning(sid)
    if (res) {
      visitorData.value = res
    }
    await nextTick()
    initVisitorChart()
  } catch {
    visitorData.value = {
      todayNew: 0, todayReturning: 0, todayNewPct: 0, todayRetPct: 0,
      yesterdayNew: 0, yesterdayReturning: 0, yesterdayNewPct: 0, yesterdayRetPct: 0,
    }
  } finally {
    loadingVisitors.value = false
  }
}

const loadReferers = async (sid: number) => {
  loadingReferers.value = true
  try {
    const res = await getTopReferers({ sourceId: sid, limit: 10 })
    topReferers.value = res || []
  } catch { topReferers.value = [] }
  finally { loadingReferers.value = false }
}

const loadPages = async (sid: number) => {
  loadingPages.value = true
  try {
    const res = await getTopPages({ sourceId: sid, limit: 10 })
    topPages.value = res || []
  } catch { topPages.value = [] }
  finally { loadingPages.value = false }
}

const loadEntryPages = async (sid: number) => {
  loadingEntryPages.value = true
  try {
    const res = await getTopEntryPages({ sourceId: sid, limit: 10 })
    topEntryPages.value = res || []
  } catch { topEntryPages.value = [] }
  finally { loadingEntryPages.value = false }
}

const loadGeo = async (sid?: number) => {
  const sourceId = sid ?? selectedSourceId.value
  if (!sourceId) return
  loadingGeo.value = true
  try {
    const res = await getOverviewGeo({
      sourceId: sourceId,
      scope: 'global',
    })
    geoData.value = res || []
  } catch { geoData.value = [] }
  finally { loadingGeo.value = false }
}

const loadDevices = async (sid: number) => {
  loadingDevices.value = true
  try {
    const res = await getOverviewDevices(sid)
    deviceData.value = res || []
    await nextTick()
    initDeviceChart()
  } catch { deviceData.value = [] }
  finally { loadingDevices.value = false }
}

// 初始化趋势图表
const initTrendChart = () => {
  if (!trendChartRef.value) return
  if (trendChart) trendChart.dispose()
  trendChart = echarts.init(trendChartRef.value)

  const data = trendData.value
  const option = {
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: ['PV', 'UV'],
      top: 0,
      right: 0,
    },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data.map(i => i.time),
    },
    yAxis: { type: 'value' },
    series: [
      {
        name: 'PV',
        type: 'line',
        smooth: true,
        data: data.map(i => i.pv),
        lineStyle: { color: '#d4af37', width: 2 },
        itemStyle: { color: '#d4af37' },
        areaStyle: {
          opacity: 0.15,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(212, 175, 55, 0.4)' },
            { offset: 1, color: 'rgba(212, 175, 55, 0.05)' },
          ]),
        },
      },
      {
        name: 'UV',
        type: 'line',
        smooth: true,
        data: data.map(i => i.uv),
        lineStyle: { color: '#b8860b', width: 2 },
        itemStyle: { color: '#b8860b' },
        areaStyle: {
          opacity: 0.15,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(184, 134, 11, 0.4)' },
            { offset: 1, color: 'rgba(184, 134, 11, 0.05)' },
          ]),
        },
      },
    ],
  }
  trendChart.setOption(option)
}

// 初始化新老访客图表
const initVisitorChart = () => {
  if (!visitorChartRef.value) return
  if (visitorChart) visitorChart.dispose()
  visitorChart = echarts.init(visitorChartRef.value)

  const vd = visitorData.value
  const option = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['新访客', '老访客'], top: 0, right: 0 },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: ['今日', '昨日'],
    },
    yAxis: { type: 'value' },
    series: [
      {
        name: '新访客',
        type: 'bar',
        data: [vd.todayNew, vd.yesterdayNew],
        itemStyle: { color: '#409eff' },
      },
      {
        name: '老访客',
        type: 'bar',
        data: [vd.todayReturning, vd.yesterdayReturning],
        itemStyle: { color: '#67c23a' },
      },
    ],
  }
  visitorChart.setOption(option)
}

// 初始化终端设备图表
const initDeviceChart = () => {
  if (!deviceChartRef.value) return
  if (deviceChart) deviceChart.dispose()
  deviceChart = echarts.init(deviceChartRef.value)

  const data = deviceData.value.map((item, index) => ({
    value: item.count,
    name: item.deviceType,
    itemStyle: { color: deviceColors[index % deviceColors.length] },
  }))

  const option = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: false,
        label: { show: false, position: 'center' },
        emphasis: {
          label: { show: true, fontSize: 16, fontWeight: 'bold' },
        },
        labelLine: { show: false },
        data: data,
      },
    ],
  }
  deviceChart.setOption(option)
}

const handleResize = () => {
  trendChart?.resize()
  visitorChart?.resize()
  deviceChart?.resize()
}

onMounted(() => {
  loadSources()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  visitorChart?.dispose()
  deviceChart?.dispose()
})
</script>

<style scoped>
.nginx-overview-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
}

.last-refresh-time {
  margin: 6px 0 0 0;
  font-size: 12px;
  color: #b0b0b0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.empty-tip {
  padding: 80px 0;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

/* 顶部区域: 活跃访客 + 核心指标 */
.top-section {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 12px;
  margin-bottom: 12px;
}

.active-visitors-card {
  background: linear-gradient(135deg, #409eff 0%, #337ecc 100%);
  border-radius: 8px;
  padding: 24px 20px;
  color: #fff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.25);
}

.av-number {
  font-size: 42px;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 8px;
}

.av-label {
  font-size: 14px;
  opacity: 0.9;
  margin-bottom: 4px;
}

.av-status {
  font-size: 12px;
  opacity: 0.7;
}

.core-metrics-panel {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.metric-column {
  text-align: center;
  padding: 0 8px;
  border-right: 1px solid #f0f0f0;
}

.metric-column:last-child {
  border-right: none;
}

.metric-header {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.metric-main {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 10px;
}

.metric-sub {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  margin-bottom: 4px;
  padding: 0 4px;
}

.sub-label {
  color: #909399;
}

.sub-value {
  color: #606266;
}

.predict-up {
  color: #67c23a;
}

.predict-down {
  color: #f56c6c;
}

.change-up {
  color: #67c23a;
  font-size: 11px;
  margin-left: 6px;
}

.change-down {
  color: #f56c6c;
  font-size: 11px;
  margin-left: 6px;
}

.status-4xx {
  color: #e6a23c;
}

.status-5xx {
  color: #f56c6c;
}

/* 中间区域: 趋势 + 新老访客 */
.middle-section {
  display: grid;
  grid-template-columns: 3fr 2fr;
  gap: 12px;
  margin-bottom: 12px;
}

.chart-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.chart-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.chart-content {
  height: 280px;
}

.visitor-stats {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.visitor-stat-row {
  display: flex;
  gap: 24px;
  justify-content: center;
}

.visitor-stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.vs-label { color: #909399; }
.vs-value { font-weight: 600; color: #303133; }
.vs-pct { color: #909399; font-size: 12px; }

/* 三列区域: 来路 | 受访页 | 入口页 */
.three-column-section {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.list-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.list-header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.list-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.list-body {
  max-height: 300px;
  overflow-y: auto;
}

.list-empty {
  text-align: center;
  color: #909399;
  padding: 40px 0;
  font-size: 13px;
}

.list-item {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #fafafa;
}

.list-item:last-child {
  border-bottom: none;
}

.list-item-name {
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  grid-column: 1;
}

.list-item-value {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  text-align: right;
  grid-column: 2;
}

.list-item-bar {
  grid-column: 1 / -1;
  height: 4px;
  background: #f5f7fa;
  border-radius: 2px;
}

.list-item-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #d4af37, #c9a227);
  border-radius: 2px;
  transition: width 0.3s;
}

/* 底部区域: 地域分布 + 终端设备 */
.bottom-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 12px;
  align-items: stretch;
}

.bottom-section .chart-card {
  display: flex;
  flex-direction: column;
  height: 400px;
}

.geo-card .chart-header {
  flex-shrink: 0;
}

.geo-list {
  flex: 1;
  overflow-y: auto;
}

.geo-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 4px;
  border-bottom: 1px solid #fafafa;
  font-size: 13px;
}

.geo-item:last-child {
  border-bottom: none;
}

.geo-rank {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #909399;
  font-weight: 600;
  flex-shrink: 0;
}

.geo-item:nth-child(1) .geo-rank {
  background: linear-gradient(135deg, #f6c052 0%, #e6a723 100%);
  color: #fff;
}

.geo-item:nth-child(2) .geo-rank {
  background: linear-gradient(135deg, #a0a0a0 0%, #808080 100%);
  color: #fff;
}

.geo-item:nth-child(3) .geo-rank {
  background: linear-gradient(135deg, #cd7f32 0%, #b87333 100%);
  color: #fff;
}

.geo-name {
  flex: 1;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.geo-count {
  font-weight: 600;
  color: #303133;
}

.geo-pct {
  color: #909399;
  width: 50px;
  text-align: right;
}

.device-stats {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.device-stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
}

.ds-color {
  width: 12px;
  height: 12px;
  border-radius: 3px;
  flex-shrink: 0;
}

.ds-label {
  flex: 1;
  color: #606266;
}

.ds-value {
  font-weight: 600;
  color: #303133;
}

.ds-pct {
  color: #909399;
  font-size: 12px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .top-section {
    grid-template-columns: 1fr;
  }

  .middle-section,
  .bottom-section {
    grid-template-columns: 1fr;
  }

  .three-column-section {
    grid-template-columns: 1fr;
  }

  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }
}
</style>
