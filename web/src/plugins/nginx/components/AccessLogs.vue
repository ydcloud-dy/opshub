<template>
  <div class="nginx-access-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><List /></el-icon>
        </div>
        <div>
          <h2 class="page-title">访问明细</h2>
          <p class="page-subtitle">查看 Nginx 访问日志详细记录</p>
        </div>
      </div>
    </div>

    <!-- 筛选条件 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select v-model="filterForm.sourceId" placeholder="选择数据源" class="search-input">
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-date-picker
          v-model="filterForm.timeRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          value-format="YYYY-MM-DD HH:mm:ss"
          class="date-picker"
        />
        <el-input
          v-model="filterForm.remoteAddr"
          placeholder="客户端IP"
          clearable
          class="search-input-sm"
        />
        <el-input
          v-model="filterForm.uri"
          placeholder="请求URI"
          clearable
          class="search-input"
        />
        <el-select v-model="filterForm.status" placeholder="状态码" clearable class="search-input-xs">
          <el-option label="2xx" :value="2" />
          <el-option label="3xx" :value="3" />
          <el-option label="4xx" :value="4" />
          <el-option label="5xx" :value="5" />
        </el-select>
        <el-select v-model="filterForm.method" placeholder="请求方法" clearable class="search-input-xs">
          <el-option label="GET" value="GET" />
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
          <el-option label="DELETE" value="DELETE" />
          <el-option label="PATCH" value="PATCH" />
          <el-option label="HEAD" value="HEAD" />
          <el-option label="OPTIONS" value="OPTIONS" />
        </el-select>
      </div>
      <div class="search-actions">
        <el-button class="black-button" @click="loadData">查询</el-button>
        <el-button class="reset-btn" @click="handleReset">重置</el-button>
      </div>
    </div>

    <!-- Top 统计 -->
    <div class="top-stats" v-if="filterForm.sourceId">
      <div class="top-card">
        <div class="top-header">
          <h4 class="top-title">Top 10 URI</h4>
        </div>
        <div class="top-list">
          <div v-for="(item, index) in topURIs" :key="index" class="top-item">
            <span class="top-rank" :class="'rank-' + (index + 1)">{{ index + 1 }}</span>
            <span class="top-name" :title="item.uri">{{ item.uri }}</span>
            <span class="top-count">{{ formatNumber(item.count) }}</span>
          </div>
          <el-empty v-if="topURIs.length === 0" description="暂无数据" :image-size="60" />
        </div>
      </div>
      <div class="top-card">
        <div class="top-header">
          <h4 class="top-title">Top 10 IP</h4>
        </div>
        <div class="top-list">
          <div v-for="(item, index) in topIPs" :key="index" class="top-item">
            <span class="top-rank" :class="'rank-' + (index + 1)">{{ index + 1 }}</span>
            <span class="top-name">{{ item.ip }}</span>
            <span class="top-count">{{ formatNumber(item.count) }}</span>
          </div>
          <el-empty v-if="topIPs.length === 0" description="暂无数据" :image-size="60" />
        </div>
      </div>
    </div>

    <!-- 表格 -->
    <div class="table-wrapper">
      <el-table :data="tableData" v-loading="loading" class="access-log-table" stripe>
        <el-table-column label="时间" prop="timestamp" width="165" fixed>
          <template #default="{ row }">
            <span class="time-cell">{{ formatTime(row.timestamp) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="客户端IP" prop="remoteAddr" width="145" fixed>
          <template #default="{ row }">
            <span class="ip-cell">{{ row.remoteAddr }}</span>
          </template>
        </el-table-column>
        <el-table-column label="方法" prop="method" width="75" align="center">
          <template #default="{ row }">
            <el-tag :type="getMethodTagType(row.method)" size="small" class="method-tag">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="URI" prop="uri" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="uri-cell">{{ row.uri }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态码" prop="status" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)" size="small" effect="light">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="响应大小" prop="bodyBytesSent" width="95" align="right">
          <template #default="{ row }">
            <span class="size-cell">{{ formatBytes(row.bodyBytesSent) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="耗时" prop="requestTime" width="80" align="right">
          <template #default="{ row }">
            <span :class="getResponseTimeClass(row.requestTime)">
              {{ row.requestTime.toFixed(3) }}s
            </span>
          </template>
        </el-table-column>
        <el-table-column label="Host" prop="host" width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="host-cell">{{ row.host }}</span>
          </template>
        </el-table-column>
        <el-table-column label="User-Agent" prop="httpUserAgent" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="ua-cell">{{ row.httpUserAgent }}</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[20, 50, 100, 200]"
          layout="total, sizes, prev, pager, next, jumper"
        />
      </div>
    </div>

    <!-- 没有数据源提示 -->
    <div v-if="!filterForm.sourceId" class="empty-tip">
      <el-empty description="请先选择数据源" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { List } from '@element-plus/icons-vue'
import {
  getNginxAccessLogs,
  getNginxSources,
  getNginxTopURIs,
  getNginxTopIPs,
  type NginxAccessLog,
  type NginxSource,
} from '@/api/nginx'

// sessionStorage key
const STORAGE_KEY = 'nginx-access-logs-state'
const STORAGE_DATA_KEY = 'nginx-access-logs-data'

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
const saveState = () => {
  try {
    const state = {
      sourceId: filterForm.value.sourceId,
      timeRange: filterForm.value.timeRange,
      remoteAddr: filterForm.value.remoteAddr,
      uri: filterForm.value.uri,
      status: filterForm.value.status,
      method: filterForm.value.method,
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
    }
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
      // 检查缓存时间戳，24小时内有效
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
      tableData: tableData.value,
      topURIs: topURIs.value,
      topIPs: topIPs.value,
      total: pagination.value.total,
    }
    sessionStorage.setItem(key, JSON.stringify(data))
  } catch (e) {
    console.error('保存缓存数据失败:', e)
  }
}

const loading = ref(false)
const sources = ref<NginxSource[]>([])
const tableData = ref<NginxAccessLog[]>([])
const topURIs = ref<{ uri: string; count: number }[]>([])
const topIPs = ref<{ ip: string; count: number }[]>([])

// 标记是否是初始化加载
const isInitialLoad = ref(true)

const filterForm = ref({
  sourceId: undefined as number | undefined,
  timeRange: [] as string[],
  remoteAddr: '',
  uri: '',
  status: undefined as number | undefined,
  method: '',
})

const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
})

// 格式化时间
const formatTime = (timeStr: string) => {
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
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
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 获取请求方法标签类型
const getMethodTagType = (method: string) => {
  const types: Record<string, string> = {
    GET: '',
    POST: 'success',
    PUT: 'warning',
    DELETE: 'danger',
    PATCH: 'warning',
    HEAD: 'info',
    OPTIONS: 'info',
  }
  return types[method] || ''
}

// 获取状态码标签类型
const getStatusTagType = (status: number) => {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'info'
  if (status >= 400 && status < 500) return 'warning'
  if (status >= 500) return 'danger'
  return ''
}

// 获取响应时间样式
const getResponseTimeClass = (time: number) => {
  if (time < 0.1) return 'response-fast'
  if (time < 0.5) return 'response-normal'
  if (time < 1) return 'response-slow'
  return 'response-very-slow'
}

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ status: 1 })
    sources.value = res.list || res || []

    // 恢复之前保存的状态
    const savedState = restoreState()
    if (savedState) {
      // 恢复筛选条件
      if (savedState.sourceId && sources.value.some((s: NginxSource) => s.id === savedState.sourceId)) {
        filterForm.value.sourceId = savedState.sourceId
      } else if (sources.value.length > 0) {
        filterForm.value.sourceId = sources.value[0].id
      }
      if (savedState.timeRange) {
        filterForm.value.timeRange = savedState.timeRange
      }
      if (savedState.remoteAddr) {
        filterForm.value.remoteAddr = savedState.remoteAddr
      }
      if (savedState.uri) {
        filterForm.value.uri = savedState.uri
      }
      if (savedState.status) {
        filterForm.value.status = savedState.status
      }
      if (savedState.method) {
        filterForm.value.method = savedState.method
      }
      if (savedState.page) {
        pagination.value.page = savedState.page
      }
      if (savedState.pageSize) {
        pagination.value.pageSize = savedState.pageSize
      }
    } else if (sources.value.length > 0) {
      filterForm.value.sourceId = sources.value[0].id
    }

    // 尝试从缓存恢复数据
    const cachedData = restoreCachedData()
    if (cachedData) {
      tableData.value = cachedData.tableData || []
      topURIs.value = cachedData.topURIs || []
      topIPs.value = cachedData.topIPs || []
      pagination.value.total = cachedData.total || 0
      ElMessage.success('已加载缓存数据')
    }
    // 没有缓存时不自动加载，等用户点击查询

    // 初始化完成
    isInitialLoad.value = false
  } catch (error) {
    console.error('获取数据源列表失败:', error)
    isInitialLoad.value = false
  }
}

// 加载数据
const loadData = async () => {
  if (!filterForm.value.sourceId) return

  // 保存状态
  saveState()

  loading.value = true
  try {
    const params: any = {
      sourceId: filterForm.value.sourceId,
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
    }

    if (filterForm.value.timeRange?.length === 2) {
      params.startTime = filterForm.value.timeRange[0]
      params.endTime = filterForm.value.timeRange[1]
    }
    if (filterForm.value.remoteAddr) {
      params.remoteAddr = filterForm.value.remoteAddr
    }
    if (filterForm.value.uri) {
      params.uri = filterForm.value.uri
    }
    if (filterForm.value.status) {
      // 将状态码类别转换为具体状态码范围
      params.status = filterForm.value.status * 100
    }
    if (filterForm.value.method) {
      params.method = filterForm.value.method
    }

    const res = await getNginxAccessLogs(params)
    tableData.value = res.list || []
    pagination.value.total = res.total || 0

    // 同时加载 Top 统计
    await Promise.all([loadTopURIs(), loadTopIPs()])

    // 保存缓存数据
    saveCachedData()
  } catch (error) {
    console.error('获取访问日志失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载 Top URI
const loadTopURIs = async () => {
  if (!filterForm.value.sourceId) return

  try {
    const params: any = { sourceId: filterForm.value.sourceId, limit: 10 }
    if (filterForm.value.timeRange?.length === 2) {
      params.startTime = filterForm.value.timeRange[0]
      params.endTime = filterForm.value.timeRange[1]
    }

    const res = await getNginxTopURIs(params)
    // request.ts 拦截器已解包响应
    topURIs.value = res || []
  } catch (error) {
    console.error('获取 Top URI 失败:', error)
  }
}

// 加载 Top IP
const loadTopIPs = async () => {
  if (!filterForm.value.sourceId) return

  try {
    const params: any = { sourceId: filterForm.value.sourceId, limit: 10 }
    if (filterForm.value.timeRange?.length === 2) {
      params.startTime = filterForm.value.timeRange[0]
      params.endTime = filterForm.value.timeRange[1]
    }

    const res = await getNginxTopIPs(params)
    // request.ts 拦截器已解包响应
    topIPs.value = res || []
  } catch (error) {
    console.error('获取 Top IP 失败:', error)
  }
}

// 重置筛选
const handleReset = () => {
  filterForm.value = {
    sourceId: filterForm.value.sourceId,
    timeRange: [],
    remoteAddr: '',
    uri: '',
    status: undefined,
    method: '',
  }
  pagination.value.page = 1
  loadData()
}

// 监听数据源变化
watch(() => filterForm.value.sourceId, (newVal, oldVal) => {
  // 初始化期间不触发
  if (isInitialLoad.value) return
  if (!newVal) return

  // 保存状态
  saveState()

  // 切换数据源时，尝试从缓存加载
  const cachedData = restoreCachedData()
  if (cachedData) {
    tableData.value = cachedData.tableData || []
    topURIs.value = cachedData.topURIs || []
    topIPs.value = cachedData.topIPs || []
    pagination.value.total = cachedData.total || 0
    ElMessage.success('已加载缓存数据')
  } else {
    // 没有缓存，清空数据
    tableData.value = []
    topURIs.value = []
    topIPs.value = []
    pagination.value.total = 0
    ElMessage.info('请点击查询按钮加载数据')
  }
})

// 监听分页变化
watch([() => pagination.value.page, () => pagination.value.pageSize], ([newPage, newSize], [oldPage, oldSize]) => {
  // 初始化期间不触发
  if (isInitialLoad.value) return
  // 如果是 pageSize 变化，重置页码
  if (newSize !== oldSize) {
    pagination.value.page = 1
  }
  loadData()
})

onMounted(async () => {
  await loadSources()
  // 不自动加载数据，等用户点击查询（loadSources 中会从缓存恢复数据）
})
</script>

<style scoped>
.nginx-access-logs-container {
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
  flex-wrap: wrap;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
  flex-wrap: wrap;
}

.search-input {
  width: 180px;
}

.search-input-sm {
  width: 150px;
}

.search-input-xs {
  width: 120px;
}

.date-picker {
  width: 380px;
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

/* Top 统计 */
.top-stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.top-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.top-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.top-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.top-list {
  max-height: 280px;
  overflow-y: auto;
}

.top-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f5f7fa;
}

.top-item:last-child {
  border-bottom: none;
}

.top-rank {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #909399;
  margin-right: 12px;
}

.top-rank.rank-1 {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.top-rank.rank-2 {
  background: linear-gradient(135deg, #1a1a1a 0%, #333 100%);
  color: #d4af37;
}

.top-rank.rank-3 {
  background: linear-gradient(135deg, #333 0%, #4a4a4a 100%);
  color: #d4af37;
}

.top-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: #606266;
}

.top-count {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  margin-left: 12px;
}

/* 表格 */
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

/* 访问日志表格样式 */
.access-log-table {
  border-radius: 8px;
  overflow: hidden;
}

.access-log-table :deep(.el-table__header th) {
  background-color: #fafafa;
  font-weight: 600;
  font-size: 13px;
  color: #606266;
  padding: 12px 0;
}

.access-log-table :deep(.el-table__row) {
  transition: background-color 0.15s;
}

.access-log-table :deep(.el-table__row td) {
  padding: 10px 0;
}

.access-log-table :deep(.el-table__row:hover td) {
  background-color: #f5f7fa !important;
}

/* 表格单元格样式 */
.time-cell {
  font-size: 13px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  white-space: nowrap;
}

.ip-cell {
  font-size: 13px;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-weight: 500;
  white-space: nowrap;
}

.method-tag {
  font-weight: 600;
}

.uri-cell {
  font-size: 13px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.size-cell {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
}

.host-cell {
  font-size: 13px;
  color: #606266;
}

.ua-cell {
  font-size: 12px;
  color: #909399;
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.response-fast {
  color: #67c23a;
  font-weight: 500;
}

.response-normal {
  color: #409eff;
  font-weight: 500;
}

.response-slow {
  color: #e6a23c;
  font-weight: 500;
}

.response-very-slow {
  color: #f56c6c;
  font-weight: 500;
}

.empty-tip {
  background: #fff;
  border-radius: 8px;
  padding: 60px 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

@media (max-width: 1200px) {
  .top-stats {
    grid-template-columns: 1fr;
  }
}
</style>
