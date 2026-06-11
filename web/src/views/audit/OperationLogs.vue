<template>
  <div class="operation-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">操作日志</h2>
          <p class="page-subtitle">记录系统内的所有操作行为</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleSearch">
          <el-icon style="margin-right: 6px;"><Search /></el-icon>
          查询
        </el-button>
        <el-button class="black-button" @click="handleReset">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          重置
        </el-button>
        <el-button class="black-button danger" @click="handleBatchDelete" :disabled="selectedIds.length === 0">
          <el-icon style="margin-right: 6px;"><Delete /></el-icon>
          批量删除
        </el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-input
        v-model="searchForm.username"
        placeholder="搜索用户名..."
        clearable
        class="filter-input"
      >
        <template #prefix>
          <el-icon class="filter-icon"><User /></el-icon>
        </template>
      </el-input>
      <el-select
        v-model="searchForm.module"
        placeholder="选择模块"
        clearable
        class="filter-select"
      >
        <el-option label="系统管理" value="系统管理" />
        <el-option label="个人信息" value="个人信息" />
        <el-option label="操作审计" value="操作审计" />
        <el-option label="资产管理" value="资产管理" />
        <el-option label="容器管理" value="容器管理" />
        <el-option label="监控中心" value="监控中心" />
        <el-option label="任务中心" value="任务中心" />
      </el-select>
      <el-select
        v-model="searchForm.action"
        placeholder="操作类型"
        clearable
        class="filter-select"
      >
        <el-option label="查询" value="查询" />
        <el-option label="创建" value="创建" />
        <el-option label="更新" value="更新" />
        <el-option label="删除" value="删除" />
        <el-option label="登录" value="登录" />
        <el-option label="登出" value="登出" />
      </el-select>
      <el-select
        v-model="searchForm.status"
        placeholder="状态码"
        clearable
        class="filter-select"
      >
        <el-option label="成功 (2xx)" value="2xx" />
        <el-option label="重定向 (3xx)" value="3xx" />
        <el-option label="客户端错误 (4xx)" value="4xx" />
        <el-option label="服务器错误 (5xx)" value="5xx" />
      </el-select>
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        value-format="YYYY-MM-DD"
        class="filter-date"
      />
    </div>

    <!-- 数据表格 -->
    <div class="table-wrapper">
      <el-table
        :data="logList"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        class="modern-table"
        size="default"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column label="ID" prop="id" width="80" align="center">
          <template #default="{ row }">
            <span class="id-text">#{{ row.id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作用户" prop="username" min-width="140">
          <template #default="{ row }">
            <div class="user-cell">
              <el-icon class="user-icon"><User /></el-icon>
              <span>{{ row.realName || row.username || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="模块" prop="module" min-width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.module }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" prop="action" width="90">
          <template #default="{ row }">
            <el-tag :type="getActionType(row.action)" size="small">
              {{ row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作描述" prop="description" min-width="200" show-overflow-tooltip />
        <el-table-column label="请求方法" prop="method" width="100">
          <template #default="{ row }">
            <el-tag :type="getMethodType(row.method)" size="small">
              {{ row.method }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="请求路径" prop="path" min-width="180" show-overflow-tooltip />
        <el-table-column label="状态" prop="status" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status >= 200 && row.status < 300 ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" prop="costTime" width="100" align="right">
          <template #default="{ row }">
            <span :class="['cost-text', { 'slow-cost': row.costTime > 1000 }]">
              {{ row.costTime }}ms
            </span>
          </template>
        </el-table-column>
        <el-table-column label="IP地址" prop="ip" width="130" />
        <el-table-column label="操作时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn danger" @click="handleDelete(row)">
                  <el-icon :size="18"><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLogList"
          @current-change="loadLogList"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Delete, Search, Refresh, User } from '@element-plus/icons-vue'
import { getOperationLogList, deleteOperationLog, deleteOperationLogsBatch } from '@/api/audit'

// 搜索表单
const searchForm = reactive({
  username: '',
  module: '',
  action: '',
  status: '',
  startTime: '',
  endTime: ''
})

// 日期范围
const dateRange = ref<[string, string]>([])

// 监听日期范围变化
watch(dateRange, (newVal) => {
  if (newVal && newVal.length === 2) {
    searchForm.startTime = newVal[0]
    searchForm.endTime = newVal[1]
  } else {
    searchForm.startTime = ''
    searchForm.endTime = ''
  }
})

// 日志列表
const logList = ref<any[]>([])
const loading = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 选中的ID
const selectedIds = ref<number[]>([])

// 加载日志列表
const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getOperationLogList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchForm
    })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取日志列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadLogList()
}

// 重置
const handleReset = () => {
  searchForm.username = ''
  searchForm.module = ''
  searchForm.action = ''
  searchForm.status = ''
  searchForm.startTime = ''
  searchForm.endTime = ''
  dateRange.value = []
  pagination.page = 1
  loadLogList()
}

// 删除
const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定要删除这条日志吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteOperationLog(row.id)
      ElMessage.success('删除成功')
      loadLogList()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

// 批量删除
const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 条日志吗?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteOperationLogsBatch(selectedIds.value)
      ElMessage.success('删除成功')
      selectedIds.value = []
      loadLogList()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

// 选择变化
const handleSelectionChange = (selection: any[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 获取操作类型标签样式
const getActionType = (action: string) => {
  const map: Record<string, string> = {
    '查询': 'info',
    '创建': 'success',
    '更新': 'warning',
    '删除': 'danger',
    '登录': 'success',
    '登出': 'info'
  }
  return map[action] || 'info'
}

// 获取请求方法标签样式
const getMethodType = (method: string) => {
  const map: Record<string, string> = {
    'GET': 'info',
    'POST': 'success',
    'PUT': 'warning',
    'DELETE': 'danger'
  }
  return map[method] || 'info'
}

// 实时搜索
watch([() => searchForm.username, () => searchForm.module, () => searchForm.action, () => searchForm.status], () => {
  pagination.page = 1
  loadLogList()
})

onMounted(() => {
  loadLogList()
})
</script>

<style scoped>
.operation-logs-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  background: #eff6ff;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2563eb;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #bfdbfe;
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

.black-button.danger {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
}

.black-button.danger:hover {
  background-color: #f78989 !important;
}

.black-button:disabled {
  background-color: #c0c4cc !important;
  border-color: #c0c4cc !important;
}

/* 筛选栏 */
.filter-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-input {
  width: 200px;
}

.filter-select {
  width: 140px;
}

.filter-date {
  width: 260px;
}

.filter-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.id-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #909399;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-icon {
  color: #d4af37;
  font-size: 16px;
}

.cost-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
}

.cost-text.slow-cost {
  color: #f56c6c;
  font-weight: 600;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: #d4af37;
  padding: 4px;
}

.action-btn:hover {
  color: #bfa13f;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}
</style>
