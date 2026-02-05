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
        <el-button @click="exportToPDF" :loading="exporting" type="primary">
          <el-icon style="margin-right: 6px;"><Download /></el-icon>
          导出PDF
        </el-button>
        <el-button @click="refreshData" :loading="refreshing">
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

    <!-- 导出内容区域（PDF只导出这部分） -->
    <div ref="reportContainerRef" class="export-content">
      <!-- 表格 -->
      <div class="table-wrapper">
      <el-table :data="tableData" v-loading="loading" class="modern-table" :header-cell-style="{ fontWeight: '600', color: '#303133', backgroundColor: '#f8f9fa' }">
        <template #empty>
          <el-empty description="暂无数据，请选择日期范围后点击查询或刷新" />
        </template>
        <el-table-column label="日期" prop="date" width="120" align="center">
          <template #default="{ row }">
            <span class="date-cell">{{ formatDate(row.date) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="总请求数" prop="totalRequests" min-width="150" align="right">
          <template #default="{ row }">
            <span class="number-cell">{{ formatNumber(row.totalRequests) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="独立访客" prop="uniqueVisitors" min-width="150" align="right">
          <template #default="{ row }">
            <span class="number-cell">{{ formatNumber(row.uniqueVisitors) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="总带宽" prop="totalBandwidth" min-width="150" align="right">
          <template #default="{ row }">
            <span class="number-cell">{{ formatBytes(row.totalBandwidth) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="平均响应时间" prop="avgResponseTime" min-width="140" align="right">
          <template #default="{ row }">
            <span class="number-cell">{{ (row.avgResponseTime || 0).toFixed(2) }}s</span>
          </template>
        </el-table-column>
        <el-table-column label="2xx" prop="status2xx" min-width="120" align="center">
          <template #default="{ row }">
            <span class="status-cell status-2xx">{{ formatNumber(row.status2xx) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="3xx" prop="status3xx" min-width="120" align="center">
          <template #default="{ row }">
            <span class="status-cell status-3xx">{{ formatNumber(row.status3xx) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="4xx" prop="status4xx" min-width="120" align="center">
          <template #default="{ row }">
            <span class="status-cell status-4xx">{{ formatNumber(row.status4xx) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="5xx" prop="status5xx" min-width="120" align="center">
          <template #default="{ row }">
            <span class="status-cell status-5xx">{{ formatNumber(row.status5xx) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="错误率" min-width="120" align="right">
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Calendar, Refresh, Download } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getNginxDailyReport, getNginxSources, collectNginxLogs, type NginxDailyStats, type NginxSource } from '@/api/nginx'

const loading = ref(false)
const exporting = ref(false)
const refreshing = ref(false)
const sources = ref<NginxSource[]>([])
const tableData = ref<NginxDailyStats[]>([])
const filterForm = ref({
  sourceId: undefined as number | undefined,
  dateRange: [] as string[],
})

const trendChartRef = ref<HTMLElement | null>(null)
const reportContainerRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null

// sessionStorage key
const STORAGE_KEY = 'nginx-daily-report-state'
const STORAGE_DATA_KEY = 'nginx-daily-report-data'

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

// 从 sessionStorage 恢复缓存数据
const restoreCachedData = () => {
  try {
    if (!filterForm.value.sourceId) return null
    const key = `${STORAGE_DATA_KEY}-${filterForm.value.sourceId}`
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
    if (!filterForm.value.sourceId) return
    const key = `${STORAGE_DATA_KEY}-${filterForm.value.sourceId}`
    const data = {
      timestamp: Date.now(),
      sourceId: filterForm.value.sourceId,
      dateRange: filterForm.value.dateRange,
      tableData: tableData.value,
    }
    sessionStorage.setItem(key, JSON.stringify(data))
  } catch (e) {
    console.error('保存缓存数据失败:', e)
  }
}

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

    // 恢复之前保存的状态
    const savedState = restoreState()
    if (savedState) {
      // 恢复日期范围
      if (savedState.dateRange && savedState.dateRange.length === 2) {
        filterForm.value.dateRange = savedState.dateRange
      }
      // 恢复数据源（如果还在列表中）
      if (savedState.sourceId && sources.value.some((s: NginxSource) => s.id === savedState.sourceId)) {
        filterForm.value.sourceId = savedState.sourceId
      } else if (sources.value.length > 0) {
        // 没有保存的状态或数据源不存在，选择第一个数据源
        filterForm.value.sourceId = sources.value[0].id
      }
    } else if (sources.value.length > 0 && !filterForm.value.sourceId) {
      // 首次进入，自动选择第一个数据源
      filterForm.value.sourceId = sources.value[0].id
    }

    // 尝试从缓存恢复数据（缓存键包含数据源ID）
    const cachedData = restoreCachedData()
    if (cachedData) {
      // 使用缓存数据
      tableData.value = cachedData.tableData || []
      await nextTick()
      initTrendChart()
      ElMessage.success('已加载缓存数据')
    }
    // 没有缓存时不自动加载，等用户手动点击刷新
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
    // 如果没有选择日期范围，默认查询今天
    if (filterForm.value.dateRange?.length === 2) {
      params.startDate = filterForm.value.dateRange[0]
      params.endDate = filterForm.value.dateRange[1]
    } else {
      const today = new Date().toISOString().split('T')[0]
      params.startDate = today
      params.endDate = today
    }

    const res = await getNginxDailyReport(params)
    // request.ts 拦截器已解包响应，直接返回数组
    tableData.value = res || []
    await nextTick()
    initTrendChart()

    // 保存状态（保存用户实际选择的日期范围，不是默认值）
    saveState({
      sourceId: filterForm.value.sourceId,
      dateRange: filterForm.value.dateRange,
    })

    // 保存数据到缓存
    saveCachedData()

    if (tableData.value.length === 0) {
      ElMessage.warning('该日期范围内暂无数据')
    } else {
      ElMessage.success('数据加载成功')
    }
  } catch (error) {
    console.error('获取日报数据失败:', error)
    ElMessage.error('获取日报数据失败')
  } finally {
    loading.value = false
  }
}

// 刷新数据（先采集日志再加载数据）
const refreshData = async () => {
  if (!filterForm.value.sourceId) {
    ElMessage.warning('请选择数据源')
    return
  }

  refreshing.value = true
  try {
    // 先触发日志采集
    ElMessage.info('正在采集日志...')
    await collectNginxLogs(filterForm.value.sourceId)
    ElMessage.success('日志采集完成，正在加载数据...')

    // 采集完成后加载数据
    await loadData()
  } catch (error) {
    console.error('采集日志失败:', error)
    ElMessage.error('采集日志失败')
  } finally {
    refreshing.value = false
  }
}

// 重置筛选
const handleReset = () => {
  filterForm.value = {
    sourceId: sources.value.length > 0 ? sources.value[0].id : undefined,
    dateRange: [],
  }
  // 清除保存的状态
  saveState({
    sourceId: filterForm.value.sourceId,
    dateRange: filterForm.value.dateRange,
  })
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

// 导出PDF
const exportToPDF = async () => {
  if (tableData.value.length === 0) {
    ElMessage.warning('暂无数据可导出')
    return
  }

  if (!reportContainerRef.value) {
    ElMessage.error('页面元素未找到')
    return
  }

  exporting.value = true

  try {
    // 动态导入
    const { jsPDF } = await import('jspdf')
    const html2canvas = (await import('html2canvas')).default

    // 获取数据源名称
    const currentSource = sources.value.find(s => s.id === filterForm.value.sourceId)
    const sourceName = currentSource?.name || '全部数据源'

    // 使用 html2canvas 截图
    const canvas = await html2canvas(reportContainerRef.value, {
      scale: 2, // 提高清晰度
      useCORS: true,
      allowTaint: true,
      backgroundColor: '#f5f7fa',
      logging: false,
    })

    // 创建PDF文档 (A4横向)
    const doc = new jsPDF({
      orientation: 'landscape',
      unit: 'mm',
      format: 'a4'
    })

    const pageWidth = doc.internal.pageSize.getWidth()
    const pageHeight = doc.internal.pageSize.getHeight()

    // 计算图片在PDF中的尺寸（保持宽高比）
    const imgWidth = pageWidth - 20 // 左右各留10mm边距
    const imgHeight = (canvas.height * imgWidth) / canvas.width

    // 将截图添加到PDF
    const imgData = canvas.toDataURL('image/png')

    // 如果图片高度超过一页，需要分页
    let yPosition = 10
    let remainingHeight = imgHeight

    while (remainingHeight > 0) {
      const currentPageHeight = Math.min(remainingHeight, pageHeight - 20)

      // 计算裁剪区域
      const sourceY = (imgHeight - remainingHeight) * (canvas.height / imgHeight)
      const sourceHeight = currentPageHeight * (canvas.height / imgHeight)

      // 创建临时canvas用于裁剪
      const tempCanvas = document.createElement('canvas')
      tempCanvas.width = canvas.width
      tempCanvas.height = sourceHeight
      const tempCtx = tempCanvas.getContext('2d')

      if (tempCtx) {
        tempCtx.drawImage(
          canvas,
          0, sourceY, canvas.width, sourceHeight,
          0, 0, canvas.width, sourceHeight
        )

        const pageImgData = tempCanvas.toDataURL('image/png')
        doc.addImage(pageImgData, 'PNG', 10, yPosition, imgWidth, currentPageHeight)
      }

      remainingHeight -= currentPageHeight

      if (remainingHeight > 0) {
        doc.addPage()
        yPosition = 10
      }
    }

    // 保存PDF
    const fileName = `nginx日报_${sourceName}_${new Date().toISOString().slice(0, 10)}.pdf`
    doc.save(fileName)

    ElMessage.success('导出成功')
  } catch (error) {
    console.error('导出PDF失败:', error)
    ElMessage.error('导出PDF失败，请稍后重试')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  loadSources()
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

/* 导出内容区域 */
.export-content {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 8px;
}

/* 表格 */
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.modern-table {
  border-radius: 8px;
  overflow: hidden;
}

.modern-table :deep(.el-table__header) {
  th {
    white-space: nowrap;
  }
}

.modern-table :deep(.el-table__row) {
  td {
    white-space: nowrap;
  }
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f5f7fa !important;
}

.date-cell {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: #606266;
}

.number-cell {
  font-weight: 500;
  color: #303133;
  font-size: 13px;
}

.status-cell {
  font-weight: 500;
  font-size: 13px;
  padding: 4px 8px;
  border-radius: 4px;
}

.status-2xx {
  color: #67c23a;
}

.status-3xx {
  color: #409eff;
}

.status-4xx {
  color: #e6a23c;
}

.status-5xx {
  color: #f56c6c;
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
