<template>
  <div class="nginx-analytics-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><TrendCharts /></el-icon>
        </div>
        <div>
          <h2 class="page-title">数据分析</h2>
          <p class="page-subtitle">查看详细的流量分析和访问者分布</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select v-model="selectedSourceId" placeholder="选择数据源" clearable style="width: 200px">
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          :shortcuts="dateShortcuts"
          value-format="YYYY-MM-DD"
          style="width: 280px"
        />
        <el-button class="black-button" @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 时间序列图表 -->
    <div class="chart-section">
      <div class="section-header">
        <h3>流量趋势</h3>
        <el-radio-group v-model="timeInterval" size="small" @change="loadTimeSeries">
          <el-radio-button label="hour">按小时</el-radio-button>
          <el-radio-button label="day">按天</el-radio-button>
        </el-radio-group>
      </div>
      <div ref="timeSeriesChartRef" class="chart" style="height: 350px" v-loading="loadingTimeSeries"></div>
    </div>

    <!-- 分布图表区域 -->
    <div class="chart-row">
      <!-- 地理分布 -->
      <div class="chart-card half">
        <div class="section-header">
          <h3>地理分布</h3>
          <el-select v-model="geoLevel" size="small" style="width: 100px" @change="loadGeoDistribution">
            <el-option label="国家" value="country" />
            <el-option label="省份" value="province" />
            <el-option label="城市" value="city" />
          </el-select>
        </div>
        <div ref="geoChartRef" class="chart" style="height: 300px" v-loading="loadingGeo"></div>
      </div>

      <!-- 浏览器分布 -->
      <div class="chart-card half">
        <div class="section-header">
          <h3>浏览器分布</h3>
        </div>
        <div ref="browserChartRef" class="chart" style="height: 300px" v-loading="loadingBrowser"></div>
      </div>
    </div>

    <div class="chart-row">
      <!-- 设备分布 -->
      <div class="chart-card half">
        <div class="section-header">
          <h3>设备类型分布</h3>
        </div>
        <div ref="deviceChartRef" class="chart" style="height: 300px" v-loading="loadingDevice"></div>
      </div>

      <!-- 响应时间分布 -->
      <div class="chart-card half">
        <div class="section-header">
          <h3>状态码分布</h3>
        </div>
        <div ref="statusChartRef" class="chart" style="height: 300px" v-loading="loadingStatus"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { TrendCharts, Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import {
  getNginxSources,
  getNginxTimeSeries,
  getNginxGeoDistribution,
  getNginxBrowserDistribution,
  getNginxDeviceDistribution,
  getNginxOverview,
  type NginxSource,
  type TimeSeriesPoint,
  type GeoStats,
  type BrowserStats,
  type DeviceStats,
} from '@/api/nginx'

const selectedSourceId = ref<number | undefined>(undefined)
const sources = ref<NginxSource[]>([])
const dateRange = ref<[string, string] | null>(null)
const timeInterval = ref<'hour' | 'day'>('hour')
const geoLevel = ref<'country' | 'province' | 'city'>('country')

const loadingTimeSeries = ref(false)
const loadingGeo = ref(false)
const loadingBrowser = ref(false)
const loadingDevice = ref(false)
const loadingStatus = ref(false)

const timeSeriesChartRef = ref<HTMLElement | null>(null)
const geoChartRef = ref<HTMLElement | null>(null)
const browserChartRef = ref<HTMLElement | null>(null)
const deviceChartRef = ref<HTMLElement | null>(null)
const statusChartRef = ref<HTMLElement | null>(null)

let timeSeriesChart: echarts.ECharts | null = null
let geoChart: echarts.ECharts | null = null
let browserChart: echarts.ECharts | null = null
let deviceChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null

const dateShortcuts = [
  { text: '今天', value: () => { const d = new Date(); return [d, d] } },
  { text: '最近7天', value: () => { const d = new Date(); return [new Date(d.getTime() - 7 * 24 * 3600 * 1000), d] } },
  { text: '最近30天', value: () => { const d = new Date(); return [new Date(d.getTime() - 30 * 24 * 3600 * 1000), d] } },
]

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ pageSize: 100 })
    sources.value = res.list || res || []
  } catch (error) {
    console.error('获取数据源列表失败:', error)
  }
}

// 加载所有数据
const loadData = () => {
  loadTimeSeries()
  loadGeoDistribution()
  loadBrowserDistribution()
  loadDeviceDistribution()
  loadStatusDistribution()
}

// 构建查询参数
const buildParams = () => {
  const params: any = {}
  if (selectedSourceId.value) {
    params.sourceId = selectedSourceId.value
  }
  if (dateRange.value) {
    params.startTime = dateRange.value[0]
    params.endTime = dateRange.value[1]
  }
  return params
}

// 加载时间序列数据
const loadTimeSeries = async () => {
  loadingTimeSeries.value = true
  try {
    const params = { ...buildParams(), interval: timeInterval.value }
    const res = await getNginxTimeSeries(params)
    const data: TimeSeriesPoint[] = res || []

    await nextTick()
    if (timeSeriesChartRef.value) {
      if (!timeSeriesChart) {
        timeSeriesChart = echarts.init(timeSeriesChartRef.value)
      }

      const option = {
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'cross' }
        },
        legend: {
          data: ['请求数', '带宽(KB)', '响应时间(ms)'],
          bottom: 0
        },
        grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
        xAxis: {
          type: 'category',
          boundaryGap: false,
          data: data.map(d => d.time)
        },
        yAxis: [
          { type: 'value', name: '请求数', position: 'left' },
          { type: 'value', name: '带宽/响应时间', position: 'right' }
        ],
        series: [
          {
            name: '请求数',
            type: 'line',
            smooth: true,
            data: data.map(d => d.requests),
            itemStyle: { color: '#d4af37' },
            areaStyle: { color: 'rgba(212, 175, 55, 0.1)' }
          },
          {
            name: '带宽(KB)',
            type: 'line',
            smooth: true,
            yAxisIndex: 1,
            data: data.map(d => Math.round(d.bandwidth / 1024)),
            itemStyle: { color: '#67C23A' }
          },
          {
            name: '响应时间(ms)',
            type: 'line',
            smooth: true,
            yAxisIndex: 1,
            data: data.map(d => Math.round(d.avgResponseTime * 1000)),
            itemStyle: { color: '#E6A23C' }
          }
        ]
      }
      timeSeriesChart.setOption(option)
    }
  } catch (error) {
    console.error('获取时间序列数据失败:', error)
  } finally {
    loadingTimeSeries.value = false
  }
}

// 加载地理分布
const loadGeoDistribution = async () => {
  loadingGeo.value = true
  try {
    const params = { ...buildParams(), level: geoLevel.value }
    const res = await getNginxGeoDistribution(params)
    const data: GeoStats[] = res || []

    await nextTick()
    if (geoChartRef.value) {
      if (!geoChart) {
        geoChart = echarts.init(geoChartRef.value)
      }

      const option = {
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          right: 10,
          top: 'center',
          type: 'scroll'
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 4 },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          labelLine: { show: false },
          data: data.slice(0, 10).map((d, i) => ({
            name: d.country || d.province || d.city || '未知',
            value: d.count,
            itemStyle: { color: getColorByIndex(i) }
          }))
        }]
      }
      geoChart.setOption(option)
    }
  } catch (error) {
    console.error('获取地理分布数据失败:', error)
  } finally {
    loadingGeo.value = false
  }
}

// 加载浏览器分布
const loadBrowserDistribution = async () => {
  loadingBrowser.value = true
  try {
    const params = buildParams()
    const res = await getNginxBrowserDistribution(params)
    const data: BrowserStats[] = res || []

    await nextTick()
    if (browserChartRef.value) {
      if (!browserChart) {
        browserChart = echarts.init(browserChartRef.value)
      }

      const option = {
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          right: 10,
          top: 'center'
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 4 },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          labelLine: { show: false },
          data: data.slice(0, 8).map((d, i) => ({
            name: d.browser || '其他',
            value: d.count,
            itemStyle: { color: getBrowserColor(d.browser) }
          }))
        }]
      }
      browserChart.setOption(option)
    }
  } catch (error) {
    console.error('获取浏览器分布数据失败:', error)
  } finally {
    loadingBrowser.value = false
  }
}

// 加载设备分布
const loadDeviceDistribution = async () => {
  loadingDevice.value = true
  try {
    const params = buildParams()
    const res = await getNginxDeviceDistribution(params)
    const data: DeviceStats[] = res || []

    await nextTick()
    if (deviceChartRef.value) {
      if (!deviceChart) {
        deviceChart = echarts.init(deviceChartRef.value)
      }

      const deviceColors: Record<string, string> = {
        'desktop': '#409EFF',
        'mobile': '#67C23A',
        'tablet': '#E6A23C',
        'bot': '#909399'
      }

      const option = {
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          right: 10,
          top: 'center'
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 4 },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          labelLine: { show: false },
          data: data.map(d => ({
            name: getDeviceName(d.deviceType),
            value: d.count,
            itemStyle: { color: deviceColors[d.deviceType] || '#909399' }
          }))
        }]
      }
      deviceChart.setOption(option)
    }
  } catch (error) {
    console.error('获取设备分布数据失败:', error)
  } finally {
    loadingDevice.value = false
  }
}

// 加载状态码分布
const loadStatusDistribution = async () => {
  loadingStatus.value = true
  try {
    const res = await getNginxOverview()
    const statusDistribution = res?.statusDistribution || {}

    await nextTick()
    if (statusChartRef.value) {
      if (!statusChart) {
        statusChart = echarts.init(statusChartRef.value)
      }

      const statusColors: Record<string, string> = {
        '2xx': '#67C23A',
        '3xx': '#409EFF',
        '4xx': '#E6A23C',
        '5xx': '#F56C6C'
      }

      const data = Object.entries(statusDistribution).map(([key, value]) => ({
        name: key,
        value: value as number,
        itemStyle: { color: statusColors[key] || '#909399' }
      }))

      const option = {
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          right: 10,
          top: 'center'
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 4 },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          labelLine: { show: false },
          data
        }]
      }
      statusChart.setOption(option)
    }
  } catch (error) {
    console.error('获取状态码分布数据失败:', error)
  } finally {
    loadingStatus.value = false
  }
}

// 辅助函数
const getColorByIndex = (index: number): string => {
  const colors = ['#d4af37', '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399', '#9B59B6', '#1ABC9C', '#3498DB', '#E74C3C']
  return colors[index % colors.length]
}

const getBrowserColor = (browser: string): string => {
  const colors: Record<string, string> = {
    'Chrome': '#4285F4',
    'Firefox': '#FF7139',
    'Safari': '#000000',
    'Edge': '#0078D7',
    'IE': '#0078D7',
    'Opera': '#FF1B2D'
  }
  return colors[browser] || '#909399'
}

const getDeviceName = (type: string): string => {
  const names: Record<string, string> = {
    'desktop': '桌面端',
    'mobile': '移动端',
    'tablet': '平板',
    'bot': '机器人'
  }
  return names[type] || type
}

// 监听筛选条件变化
watch([selectedSourceId, dateRange], () => {
  loadData()
})

// 窗口大小变化时调整图表
const handleResize = () => {
  timeSeriesChart?.resize()
  geoChart?.resize()
  browserChart?.resize()
  deviceChart?.resize()
  statusChart?.resize()
}

onMounted(() => {
  loadSources()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  timeSeriesChart?.dispose()
  geoChart?.dispose()
  browserChart?.dispose()
  deviceChart?.dispose()
  statusChart?.dispose()
})
</script>

<style scoped>
.nginx-analytics-container {
  padding: 0;
  background-color: transparent;
}

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

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

.chart-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.chart {
  width: 100%;
}

.chart-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.chart-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.chart-card.half {
  flex: 1;
}

@media (max-width: 1200px) {
  .chart-row {
    flex-direction: column;
  }

  .header-actions {
    flex-wrap: wrap;
  }
}
</style>
