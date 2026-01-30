<template>
  <div class="nginx-overview-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DataLine /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Nginx 概况</h2>
          <p class="page-subtitle">实时查看 Nginx 流量统计和性能概览</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" :loading="collecting" @click="handleCollect">
          <el-icon style="margin-right: 6px;"><Upload /></el-icon>
          采集日志
        </el-button>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Connection /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">数据源总数</div>
          <div class="stat-value">{{ overview.totalSources }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">活跃数据源</div>
          <div class="stat-value">{{ overview.activeSources }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-info">
          <el-icon><View /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">今日请求数</div>
          <div class="stat-value">{{ formatNumber(overview.todayRequests) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><User /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">今日访客</div>
          <div class="stat-value">{{ formatNumber(overview.todayVisitors) }}</div>
        </div>
      </div>
    </div>

    <!-- 第二排统计卡片 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Download /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">今日带宽</div>
          <div class="stat-value">{{ formatBytes(overview.todayBandwidth) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-info">
          <el-icon><Document /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">今日PV</div>
          <div class="stat-value">{{ formatNumber(overview.todayPv || 0) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Timer /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">平均响应时间</div>
          <div class="stat-value">{{ formatResponseTime(overview.avgResponseTime || 0) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">错误率</div>
          <div class="stat-value">{{ overview.todayErrorRate.toFixed(2) }}%</div>
        </div>
      </div>
    </div>

    <!-- 第三排统计卡片 - 状态码 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><SuccessFilled /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">2xx 成功</div>
          <div class="stat-value">{{ formatNumber(overview.statusDistribution?.['2xx'] || 0) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-info">
          <el-icon><Promotion /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">3xx 重定向</div>
          <div class="stat-value">{{ formatNumber(overview.statusDistribution?.['3xx'] || 0) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><WarnTriangleFilled /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">4xx 客户端错误</div>
          <div class="stat-value">{{ formatNumber(overview.statusDistribution?.['4xx'] || 0) }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">5xx 服务端错误</div>
          <div class="stat-value">{{ formatNumber(overview.statusDistribution?.['5xx'] || 0) }}</div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-container">
      <div class="chart-card chart-card-wide">
        <div class="chart-header">
          <h3 class="chart-title">请求趋势（最近24小时）</h3>
        </div>
        <div class="chart-content" ref="requestsChartRef"></div>
      </div>
    </div>

    <!-- 第二排图表 -->
    <div class="charts-container charts-container-second">
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">带宽趋势（最近24小时）</h3>
        </div>
        <div class="chart-content" ref="bandwidthChartRef"></div>
      </div>
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">状态码分布</h3>
        </div>
        <div class="chart-content" ref="statusChartRef"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import {
  DataLine,
  Refresh,
  Connection,
  CircleCheck,
  View,
  User,
  Download,
  Warning,
  SuccessFilled,
  CircleClose,
  Upload,
  Document,
  Timer,
  Promotion,
  WarnTriangleFilled,
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getNginxOverview, collectNginxLogs, type OverviewStats } from '@/api/nginx'

const loading = ref(false)
const collecting = ref(false)
const overview = ref<OverviewStats>({
  totalSources: 0,
  activeSources: 0,
  todayRequests: 0,
  todayVisitors: 0,
  todayBandwidth: 0,
  todayPv: 0,
  todayErrorRate: 0,
  avgResponseTime: 0,
  statusDistribution: {},
})

const requestsChartRef = ref<HTMLElement | null>(null)
const bandwidthChartRef = ref<HTMLElement | null>(null)
const statusChartRef = ref<HTMLElement | null>(null)
let requestsChart: echarts.ECharts | null = null
let bandwidthChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null

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

// 格式化响应时间
const formatResponseTime = (ms: number) => {
  if (ms < 1) return (ms * 1000).toFixed(0) + ' μs'
  if (ms < 1000) return ms.toFixed(0) + ' ms'
  return (ms / 1000).toFixed(2) + ' s'
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getNginxOverview()
    // request.ts 拦截器已解包响应，直接使用返回数据
    overview.value = res || {
      totalSources: 0,
      activeSources: 0,
      todayRequests: 0,
      todayVisitors: 0,
      todayBandwidth: 0,
      todayPv: 0,
      todayErrorRate: 0,
      avgResponseTime: 0,
      statusDistribution: {},
    }
    await nextTick()
    initCharts()
  } catch (error) {
    console.error('获取概况数据失败:', error)
    ElMessage.error('获取概况数据失败')
  } finally {
    loading.value = false
  }
}

// 手动采集日志
const handleCollect = async () => {
  collecting.value = true
  try {
    const res = await collectNginxLogs()
    const results = res.results || []
    const successCount = results.filter((r: any) => r.status === 'success').length
    const failedCount = results.filter((r: any) => r.status === 'failed').length

    if (failedCount === 0) {
      ElMessage.success(`日志采集完成，共采集 ${results.length} 个数据源`)
    } else {
      ElMessage.warning(`日志采集完成: ${successCount} 成功, ${failedCount} 失败`)
    }

    // 刷新数据
    await loadData()
  } catch (error: any) {
    console.error('日志采集失败:', error)
    ElMessage.error(error.message || '日志采集失败')
  } finally {
    collecting.value = false
  }
}

// 初始化图表
const initCharts = () => {
  initRequestsChart()
  initBandwidthChart()
  initStatusChart()
}

// 初始化请求趋势图表
const initRequestsChart = () => {
  if (!requestsChartRef.value) return

  if (requestsChart) {
    requestsChart.dispose()
  }

  requestsChart = echarts.init(requestsChartRef.value)
  const trend = overview.value.requestsTrend || []

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: trend.map(item => item.time),
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '请求数',
        type: 'line',
        smooth: true,
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(212, 175, 55, 0.5)' },
            { offset: 1, color: 'rgba(212, 175, 55, 0.1)' }
          ])
        },
        lineStyle: {
          color: '#d4af37',
          width: 2
        },
        itemStyle: {
          color: '#d4af37'
        },
        data: trend.map(item => item.value),
      },
    ],
  }

  requestsChart.setOption(option)
}

// 初始化带宽趋势图表
const initBandwidthChart = () => {
  if (!bandwidthChartRef.value) return

  if (bandwidthChart) {
    bandwidthChart.dispose()
  }

  bandwidthChart = echarts.init(bandwidthChartRef.value)
  const trend = overview.value.bandwidthTrend || []

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
      formatter: (params: any) => {
        const time = params[0]?.axisValue || ''
        const value = params[0]?.value || 0
        return `${time}<br/>带宽: ${formatBytes(value)}`
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: trend.map(item => item.time),
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: (value: number) => formatBytes(value)
      }
    },
    series: [
      {
        name: '带宽',
        type: 'line',
        smooth: true,
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 158, 255, 0.5)' },
            { offset: 1, color: 'rgba(64, 158, 255, 0.1)' }
          ])
        },
        lineStyle: {
          color: '#409eff',
          width: 2
        },
        itemStyle: {
          color: '#409eff'
        },
        data: trend.map(item => item.value),
      },
    ],
  }

  bandwidthChart.setOption(option)
}

// 初始化状态码分布图表
const initStatusChart = () => {
  if (!statusChartRef.value) return

  if (statusChart) {
    statusChart.dispose()
  }

  statusChart = echarts.init(statusChartRef.value)
  const distribution = overview.value.statusDistribution || {}

  const data = [
    { value: distribution['2xx'] || 0, name: '2xx', itemStyle: { color: '#67c23a' } },
    { value: distribution['3xx'] || 0, name: '3xx', itemStyle: { color: '#409eff' } },
    { value: distribution['4xx'] || 0, name: '4xx', itemStyle: { color: '#e6a23c' } },
    { value: distribution['5xx'] || 0, name: '5xx', itemStyle: { color: '#f56c6c' } },
  ].filter(item => item.value > 0)

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)',
    },
    legend: {
      orient: 'vertical',
      left: 'left',
    },
    series: [
      {
        name: '状态码',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        label: {
          show: false,
          position: 'center',
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold',
          },
        },
        labelLine: {
          show: false,
        },
        data: data,
      },
    ],
  }

  statusChart.setOption(option)
}

// 监听窗口大小变化
const handleResize = () => {
  requestsChart?.resize()
  bandwidthChart?.resize()
  statusChart?.resize()
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  requestsChart?.dispose()
  bandwidthChart?.dispose()
  statusChart?.dispose()
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
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
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

.stat-icon-danger {
  background: linear-gradient(135deg, #f56c6c 0%, #f4534a 100%);
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
  grid-template-columns: 1fr;
  gap: 12px;
  margin-bottom: 12px;
}

.charts-container-second {
  grid-template-columns: 1fr 1fr;
}

.chart-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.chart-card-wide {
  grid-column: span 1;
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

@media (max-width: 1200px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-container,
  .charts-container-second {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: 1fr;
  }
}
</style>
