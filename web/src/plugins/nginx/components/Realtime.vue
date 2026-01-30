<template>
  <div class="nginx-realtime-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Timer /></el-icon>
        </div>
        <div>
          <h2 class="page-title">实时统计</h2>
          <p class="page-subtitle">查看 Nginx 实时流量和性能数据</p>
        </div>
      </div>
      <div class="header-actions">
        <el-switch
          v-model="autoRefresh"
          active-text="自动刷新"
          inactive-text=""
          style="margin-right: 16px"
        />
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 筛选条件 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select v-model="filterForm.sourceId" placeholder="选择数据源" class="search-input" @change="loadData">
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-select v-model="filterForm.hours" placeholder="时间范围" class="search-input" @change="loadData">
          <el-option label="最近1小时" :value="1" />
          <el-option label="最近3小时" :value="3" />
          <el-option label="最近6小时" :value="6" />
          <el-option label="最近12小时" :value="12" />
          <el-option label="最近24小时" :value="24" />
        </el-select>
      </div>
    </div>

    <!-- 实时统计卡片 -->
    <div class="stats-cards" v-if="latestStats">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><View /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">当前小时请求数</div>
          <div class="stat-value">{{ formatNumber(latestStats.totalRequests) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><User /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">当前小时访客</div>
          <div class="stat-value">{{ formatNumber(latestStats.uniqueVisitors) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Download /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">当前小时带宽</div>
          <div class="stat-value">{{ formatBytes(latestStats.totalBandwidth) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-info">
          <el-icon><Stopwatch /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">平均响应时间</div>
          <div class="stat-value">{{ latestStats.avgResponseTime.toFixed(3) }}s</div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-container">
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">请求趋势</h3>
        </div>
        <div class="chart-content" ref="requestsChartRef"></div>
      </div>
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">响应时间趋势</h3>
        </div>
        <div class="chart-content" ref="responseChartRef"></div>
      </div>
    </div>

    <!-- 状态码分布图表 -->
    <div class="chart-card full-width">
      <div class="chart-header">
        <h3 class="chart-title">状态码分布趋势</h3>
      </div>
      <div class="chart-content" ref="statusChartRef"></div>
    </div>

    <!-- 没有数据源提示 -->
    <div v-if="!filterForm.sourceId" class="empty-tip">
      <el-empty description="请先选择数据源" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Timer, Refresh, View, User, Download, Stopwatch } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getNginxRealTimeStats, getNginxSources, type NginxHourlyStats, type NginxSource } from '@/api/nginx'

const loading = ref(false)
const autoRefresh = ref(true)
const sources = ref<NginxSource[]>([])
const hourlyData = ref<NginxHourlyStats[]>([])
const filterForm = ref({
  sourceId: undefined as number | undefined,
  hours: 6,
})

let refreshTimer: ReturnType<typeof setInterval> | null = null

const requestsChartRef = ref<HTMLElement | null>(null)
const responseChartRef = ref<HTMLElement | null>(null)
const statusChartRef = ref<HTMLElement | null>(null)
let requestsChart: echarts.ECharts | null = null
let responseChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null

// 最新一小时的统计
const latestStats = computed(() => {
  if (hourlyData.value.length === 0) return null
  return hourlyData.value[0]
})

// 格式化数字
const formatNumber = (num: number) => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

// 格式化字节
const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化小时
const formatHour = (hourStr: string) => {
  const date = new Date(hourStr)
  return `${date.getHours().toString().padStart(2, '0')}:00`
}

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ status: 1 })
    // request.ts 拦截器已解包响应
    sources.value = res.list || res || []
    if (sources.value.length > 0 && !filterForm.value.sourceId) {
      filterForm.value.sourceId = sources.value[0].id
    }
  } catch (error) {
    console.error('获取数据源列表失败:', error)
  }
}

// 加载数据
const loadData = async () => {
  if (!filterForm.value.sourceId) return

  loading.value = true
  try {
    const res = await getNginxRealTimeStats({
      sourceId: filterForm.value.sourceId,
      hours: filterForm.value.hours,
    })
    // request.ts 拦截器已解包响应
    hourlyData.value = res || []
    await nextTick()
    initCharts()
  } catch (error) {
    console.error('获取实时数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 初始化所有图表
const initCharts = () => {
  initRequestsChart()
  initResponseChart()
  initStatusChart()
}

// 初始化请求趋势图表
const initRequestsChart = () => {
  if (!requestsChartRef.value) return

  if (requestsChart) {
    requestsChart.dispose()
  }

  requestsChart = echarts.init(requestsChartRef.value)
  const data = [...hourlyData.value].reverse()

  const option = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => formatHour(item.hour)),
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '请求数',
        type: 'bar',
        data: data.map(item => item.totalRequests),
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#d4af37' },
            { offset: 1, color: '#b8960c' }
          ]),
          borderRadius: [4, 4, 0, 0]
        },
      },
    ],
  }

  requestsChart.setOption(option)
}

// 初始化响应时间图表
const initResponseChart = () => {
  if (!responseChartRef.value) return

  if (responseChart) {
    responseChart.dispose()
  }

  responseChart = echarts.init(responseChartRef.value)
  const data = [...hourlyData.value].reverse()

  const option = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => formatHour(item.hour)),
    },
    yAxis: {
      type: 'value',
      name: '响应时间(s)',
    },
    series: [
      {
        name: '平均响应时间',
        type: 'line',
        smooth: true,
        data: data.map(item => item.avgResponseTime),
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(103, 194, 58, 0.5)' },
            { offset: 1, color: 'rgba(103, 194, 58, 0.1)' }
          ])
        },
        lineStyle: {
          color: '#67c23a',
          width: 2
        },
        itemStyle: {
          color: '#67c23a',
        },
      },
    ],
  }

  responseChart.setOption(option)
}

// 初始化状态码分布图表
const initStatusChart = () => {
  if (!statusChartRef.value) return

  if (statusChart) {
    statusChart.dispose()
  }

  statusChart = echarts.init(statusChartRef.value)
  const data = [...hourlyData.value].reverse()

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
    },
    legend: {
      data: ['2xx', '3xx', '4xx', '5xx'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => formatHour(item.hour)),
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '2xx',
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        data: data.map(item => item.status2xx),
        itemStyle: { color: '#67c23a' },
      },
      {
        name: '3xx',
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        data: data.map(item => item.status3xx),
        itemStyle: { color: '#409eff' },
      },
      {
        name: '4xx',
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        data: data.map(item => item.status4xx),
        itemStyle: { color: '#e6a23c' },
      },
      {
        name: '5xx',
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        data: data.map(item => item.status5xx),
        itemStyle: { color: '#f56c6c' },
      },
    ],
  }

  statusChart.setOption(option)
}

// 监听窗口大小变化
const handleResize = () => {
  requestsChart?.resize()
  responseChart?.resize()
  statusChart?.resize()
}

// 监听自动刷新状态
watch(autoRefresh, (val) => {
  if (val) {
    refreshTimer = setInterval(loadData, 60000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
})

onMounted(async () => {
  await loadSources()
  loadData()
  window.addEventListener('resize', handleResize)

  if (autoRefresh.value) {
    refreshTimer = setInterval(loadData, 60000)
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  requestsChart?.dispose()
  responseChart?.dispose()
  statusChart?.dispose()

  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.nginx-realtime-container {
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
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 200px;
}

.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-icon-primary {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-success {
  background: linear-gradient(135deg, #4caf50 0%, #45a049 100%);
  color: #fff;
}

.stat-icon-warning {
  background: linear-gradient(135deg, #e6a23c 0%, #d9972c 100%);
  color: #fff;
}

.stat-icon-info {
  background: linear-gradient(135deg, #909399 0%, #7d8086 100%);
  color: #fff;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

/* 图表容器 */
.charts-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.chart-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.chart-card.full-width {
  grid-column: span 2;
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
  height: 300px;
}

.empty-tip {
  background: #fff;
  border-radius: 8px;
  padding: 60px 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

@media (max-width: 1200px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-container {
    grid-template-columns: 1fr;
  }

  .chart-card.full-width {
    grid-column: span 1;
  }
}

@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: 1fr;
  }
}
</style>
