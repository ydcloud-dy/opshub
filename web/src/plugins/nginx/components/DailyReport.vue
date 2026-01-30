<template>
  <div class="nginx-daily-report-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Calendar /></el-icon>
        </div>
        <div>
          <h2 class="page-title">数据日报</h2>
          <p class="page-subtitle">查看 Nginx 每日统计数据汇总</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="loadData" :loading="loading">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 筛选条件 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select v-model="filterForm.sourceId" placeholder="选择数据源" clearable class="search-input">
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-date-picker
          v-model="filterForm.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          class="date-picker"
        />
      </div>
      <div class="search-actions">
        <el-button class="black-button" @click="loadData">查询</el-button>
        <el-button class="reset-btn" @click="handleReset">重置</el-button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="table-wrapper">
      <el-table :data="tableData" v-loading="loading" class="modern-table">
        <el-table-column label="日期" prop="date" width="120">
          <template #default="{ row }">
            {{ formatDate(row.date) }}
          </template>
        </el-table-column>
        <el-table-column label="总请求数" prop="totalRequests" width="120" align="right">
          <template #default="{ row }">
            {{ formatNumber(row.totalRequests) }}
          </template>
        </el-table-column>
        <el-table-column label="独立访客" prop="uniqueVisitors" width="120" align="right">
          <template #default="{ row }">
            {{ formatNumber(row.uniqueVisitors) }}
          </template>
        </el-table-column>
        <el-table-column label="总带宽" prop="totalBandwidth" width="120" align="right">
          <template #default="{ row }">
            {{ formatBytes(row.totalBandwidth) }}
          </template>
        </el-table-column>
        <el-table-column label="平均响应时间" prop="avgResponseTime" width="140" align="right">
          <template #default="{ row }">
            {{ (row.avgResponseTime || 0).toFixed(3) }}s
          </template>
        </el-table-column>
        <el-table-column label="2xx" prop="status2xx" width="100" align="right">
          <template #default="{ row }">
            <el-tag type="success" size="small" effect="dark">{{ formatNumber(row.status2xx) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="3xx" prop="status3xx" width="100" align="right">
          <template #default="{ row }">
            <el-tag type="info" size="small" effect="dark">{{ formatNumber(row.status3xx) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="4xx" prop="status4xx" width="100" align="right">
          <template #default="{ row }">
            <el-tag type="warning" size="small" effect="dark">{{ formatNumber(row.status4xx) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="5xx" prop="status5xx" width="100" align="right">
          <template #default="{ row }">
            <el-tag type="danger" size="small" effect="dark">{{ formatNumber(row.status5xx) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="错误率" width="100" align="right">
          <template #default="{ row }">
            <span :class="getErrorRateClass(row)">{{ calculateErrorRate(row) }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 图表区域 -->
    <div class="chart-card">
      <div class="chart-header">
        <h3 class="chart-title">请求趋势</h3>
      </div>
      <div class="chart-content" ref="trendChartRef"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Calendar, Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getNginxDailyReport, getNginxSources, type NginxDailyStats, type NginxSource } from '@/api/nginx'

const loading = ref(false)
const sources = ref<NginxSource[]>([])
const tableData = ref<NginxDailyStats[]>([])
const filterForm = ref({
  sourceId: undefined as number | undefined,
  dateRange: [] as string[],
})

const trendChartRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null

// 格式化日期
const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  // 处理 "2026-01-30" 或 "2026-01-30T00:00:00+08:00" 格式
  const dateOnly = dateStr.split('T')[0]
  return dateOnly
}

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

// 计算错误率
const calculateErrorRate = (row: NginxDailyStats) => {
  const total = row.status2xx + row.status3xx + row.status4xx + row.status5xx
  if (total === 0) return 0
  const errors = row.status4xx + row.status5xx
  return ((errors / total) * 100).toFixed(2)
}

// 获取错误率样式
const getErrorRateClass = (row: NginxDailyStats) => {
  const rate = parseFloat(calculateErrorRate(row))
  if (rate < 1) return 'error-rate-low'
  if (rate < 5) return 'error-rate-medium'
  return 'error-rate-high'
}

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ status: 1 })
    // request.ts 拦截器已解包响应
    sources.value = res.list || res || []
  } catch (error) {
    console.error('获取数据源列表失败:', error)
  }
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const params: any = {}
    if (filterForm.value.sourceId) {
      params.sourceId = filterForm.value.sourceId
    }
    if (filterForm.value.dateRange?.length === 2) {
      params.startDate = filterForm.value.dateRange[0]
      params.endDate = filterForm.value.dateRange[1]
    }

    const res = await getNginxDailyReport(params)
    // request.ts 拦截器已解包响应，直接返回数组
    tableData.value = res || []
    await nextTick()
    initTrendChart()
  } catch (error) {
    console.error('获取日报数据失败:', error)
    ElMessage.error('获取日报数据失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选
const handleReset = () => {
  filterForm.value = {
    sourceId: undefined,
    dateRange: [],
  }
  loadData()
}

// 初始化趋势图表
const initTrendChart = () => {
  if (!trendChartRef.value) return

  if (trendChart) {
    trendChart.dispose()
  }

  trendChart = echarts.init(trendChartRef.value)
  const data = [...tableData.value].reverse()

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
    },
    legend: {
      data: ['请求数', '访客数'],
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
      data: data.map(item => formatDate(item.date)),
    },
    yAxis: [
      {
        type: 'value',
        name: '请求数',
      },
      {
        type: 'value',
        name: '访客数',
      },
    ],
    series: [
      {
        name: '请求数',
        type: 'line',
        smooth: true,
        data: data.map(item => item.totalRequests),
        lineStyle: { color: '#d4af37' },
        itemStyle: { color: '#d4af37' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(212, 175, 55, 0.3)' },
            { offset: 1, color: 'rgba(212, 175, 55, 0.1)' }
          ])
        },
      },
      {
        name: '访客数',
        type: 'line',
        smooth: true,
        yAxisIndex: 1,
        data: data.map(item => item.uniqueVisitors),
        lineStyle: { color: '#67c23a' },
        itemStyle: { color: '#67c23a' },
      },
    ],
  }

  trendChart.setOption(option)
}

// 监听窗口大小变化
const handleResize = () => {
  trendChart?.resize()
}

onMounted(() => {
  loadSources()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
})
</script>

<style scoped>
.nginx-daily-report-container {
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

.date-picker {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
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

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
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

/* 表格 */
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

/* 图表 */
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
  height: 300px;
}

.error-rate-low {
  color: #67c23a;
  font-weight: 500;
}

.error-rate-medium {
  color: #e6a23c;
  font-weight: 500;
}

.error-rate-high {
  color: #f56c6c;
  font-weight: 500;
}
</style>
