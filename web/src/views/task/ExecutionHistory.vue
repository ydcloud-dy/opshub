<template>
  <div class="history-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Notebook /></el-icon>
        </div>
        <div>
          <h2 class="page-title">执行记录</h2>
          <p class="page-subtitle">查看用户执行任务和文件分发的历史记录</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button
          v-if="selectedIds.length > 0"
          type="danger"
          @click="handleBatchDelete"
        >
          <el-icon style="margin-right: 4px;"><Delete /></el-icon>
          删除选中 ({{ selectedIds.length }})
        </el-button>
        <el-button @click="handleExport">
          <el-icon style="margin-right: 4px;"><Download /></el-icon>
          {{ selectedIds.length > 0 ? `导出选中 (${selectedIds.length})` : '导出全部' }}
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索任务名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.taskType"
          placeholder="任务类型"
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <el-option label="手动执行" value="manual" />
          <el-option label="脚本执行" value="script" />
          <el-option label="文件分发" value="file" />
          <el-option label="系统命令" value="command" />
        </el-select>

        <el-select
          v-model="searchForm.status"
          placeholder="执行状态"
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <el-option label="等待中" value="pending" />
          <el-option label="执行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
        </el-select>

        <el-date-picker
          v-model="searchForm.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          class="search-input date-picker"
          @change="handleSearch"
        />
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        ref="tableRef"
        :data="historyList"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column label="任务名称" prop="name" min-width="180">
          <template #default="{ row }">
            <div class="task-name-cell">
              <el-icon class="task-icon" :class="getTaskIconClass(row.taskType)">
                <component :is="getTaskIcon(row.taskType)" />
              </el-icon>
              <span class="task-name">{{ row.name || '-' }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="任务类型" prop="taskType" width="120">
          <template #default="{ row }">
            <el-tag :type="getTaskTypeColor(row.taskType)" effect="plain">
              {{ getTaskTypeLabel(row.taskType) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" prop="status" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" effect="dark">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="目标主机" prop="targetHostsDisplay" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              v-if="row.targetHostsDisplay && row.targetHostsDisplay.length > 30"
              :content="row.targetHostsDisplay"
              placement="top"
              :show-after="300"
            >
              <span class="description-text hosts-ellipsis">{{ row.targetHostsDisplay }}</span>
            </el-tooltip>
            <span v-else class="description-text">{{ row.targetHostsDisplay || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="执行用户" prop="createdByName" width="120">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="24" class="user-avatar">
                {{ (row.createdByName || 'U').charAt(0).toUpperCase() }}
              </el-avatar>
              <span>{{ row.createdByName || '未知用户' }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="执行时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.createdAt) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="查看详情" placement="top">
                <el-button link class="action-btn action-view" @click="handleView(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadHistory"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 查看详情对话框 -->
    <el-dialog
      v-model="viewDialogVisible"
      title="执行记录详情"
      width="80%"
      class="history-view-dialog responsive-dialog"
    >
      <el-descriptions :column="2" border>
        <el-descriptions-item label="任务ID">{{ currentRecord.id }}</el-descriptions-item>
        <el-descriptions-item label="任务名称">{{ currentRecord.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="任务类型">
          <el-tag :type="getTaskTypeColor(currentRecord.taskType)" effect="plain">
            {{ getTaskTypeLabel(currentRecord.taskType) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="执行状态">
          <el-tag :type="getStatusType(currentRecord.status)" effect="dark">
            {{ getStatusLabel(currentRecord.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="执行用户">{{ currentRecord.createdByName || '未知用户' }}</el-descriptions-item>
        <el-descriptions-item label="执行时间">{{ formatDateTime(currentRecord.createdAt) }}</el-descriptions-item>
        <el-descriptions-item label="目标主机" :span="2">
          <div class="target-hosts-detail">
            {{ currentRecord.targetHostsDisplay || '-' }}
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <!-- 执行结果 -->
      <div v-if="currentRecord.result" class="result-section">
        <div class="result-header">
          <span class="result-title">执行结果</span>
        </div>
        <div class="result-content">
          <div
            v-for="(hostResult, index) in parsedResults"
            :key="index"
            class="host-result"
          >
            <div class="host-header" :class="hostResult.status">
              <span class="host-name">{{ hostResult.hostName }}</span>
              <span class="host-ip">({{ hostResult.hostIp }})</span>
              <el-tag
                :type="hostResult.status === 'success' ? 'success' : 'danger'"
                size="small"
                effect="dark"
              >
                {{ hostResult.status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </div>
            <div v-if="hostResult.error" class="host-error">
              <span class="error-label">错误:</span> {{ hostResult.error }}
            </div>
            <div class="host-output">
              <div class="output-header">
                <span>输出:</span>
                <el-button size="small" link @click="copyOutput(hostResult.output)">
                  <el-icon><CopyDocument /></el-icon>
                  复制
                </el-button>
              </div>
              <pre class="output-content">{{ hostResult.output || '(无输出)' }}</pre>
            </div>
          </div>
        </div>
      </div>

      <div v-if="currentRecord.errorMessage" class="error-section">
        <div class="error-header">错误信息</div>
        <div class="error-content">{{ currentRecord.errorMessage }}</div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="viewDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, markRaw } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TableInstance } from 'element-plus'
import {
  Search,
  RefreshLeft,
  Notebook,
  View,
  Delete,
  Download,
  VideoPlay,
  FolderOpened,
  Monitor,
  CopyDocument
} from '@element-plus/icons-vue'
import {
  getExecutionHistoryList,
  deleteExecutionHistory,
  batchDeleteExecutionHistory,
  exportExecutionHistory
} from '@/api/task'

// 表格引用
const tableRef = ref<TableInstance>()

// 加载状态
const loading = ref(false)

// 对话框状态
const viewDialogVisible = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 执行记录列表
const historyList = ref<any[]>([])

// 当前查看的记录
const currentRecord = ref<any>({})

// 选中的ID列表
const selectedIds = ref<number[]>([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  taskType: '',
  status: '',
  dateRange: null as string[] | null
})

// 解析执行结果
const parsedResults = computed(() => {
  if (!currentRecord.value.result) return []
  try {
    const results = JSON.parse(currentRecord.value.result)
    if (Array.isArray(results)) {
      return results
    }
    return []
  } catch {
    return []
  }
})

// 获取任务类型颜色
const getTaskTypeColor = (type: string) => {
  const colorMap: Record<string, string> = {
    manual: 'primary',
    script: 'success',
    file: 'warning',
    command: 'info'
  }
  return colorMap[type] || 'info'
}

// 获取任务类型标签
const getTaskTypeLabel = (type: string) => {
  const labelMap: Record<string, string> = {
    manual: '手动执行',
    script: '脚本执行',
    file: '文件分发',
    command: '系统命令'
  }
  return labelMap[type] || type || '-'
}

// 获取任务图标
const getTaskIcon = (type: string) => {
  const iconMap: Record<string, any> = {
    manual: markRaw(VideoPlay),
    script: markRaw(VideoPlay),
    file: markRaw(FolderOpened),
    command: markRaw(Monitor)
  }
  return iconMap[type] || markRaw(VideoPlay)
}

// 获取任务图标样式类
const getTaskIconClass = (type: string) => {
  const classMap: Record<string, string> = {
    manual: 'icon-manual',
    script: 'icon-script',
    file: 'icon-file',
    command: 'icon-command'
  }
  return classMap[type] || 'icon-manual'
}

// 获取状态类型
const getStatusType = (status: string) => {
  const typeMap: Record<string, string> = {
    pending: 'info',
    running: 'warning',
    success: 'success',
    failed: 'danger'
  }
  return typeMap[status] || 'info'
}

// 获取状态标签
const getStatusLabel = (status: string) => {
  const labelMap: Record<string, string> = {
    pending: '等待中',
    running: '执行中',
    success: '成功',
    failed: '失败'
  }
  return labelMap[status] || status || '-'
}

// 格式化日期时间
const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 搜索处理
const handleSearch = () => {
  pagination.page = 1
  loadHistory()
}

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.taskType = ''
  searchForm.status = ''
  searchForm.dateRange = null
  pagination.page = 1
  loadHistory()
}

// 分页大小改变
const handleSizeChange = () => {
  pagination.page = 1
  loadHistory()
}

// 选择变化
const handleSelectionChange = (rows: any[]) => {
  selectedIds.value = rows.map(row => row.id)
}

// 加载执行记录列表
const loadHistory = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      taskType: searchForm.taskType,
      status: searchForm.status
    }

    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.startDate = searchForm.dateRange[0]
      params.endDate = searchForm.dateRange[1]
    }

    const res = await getExecutionHistoryList(params)
    historyList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取执行记录列表失败')
  } finally {
    loading.value = false
  }
}

// 查看详情
const handleView = (row: any) => {
  currentRecord.value = { ...row }
  viewDialogVisible.value = true
}

// 删除单条记录
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除该执行记录吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteExecutionHistory(row.id)
    ElMessage.success('删除成功')
    loadHistory()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 批量删除
const handleBatchDelete = async () => {
  if (selectedIds.value.length === 0) {
    ElMessage.warning('请先选择要删除的记录')
    return
  }
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 条记录吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await batchDeleteExecutionHistory(selectedIds.value)
    ElMessage.success('删除成功')
    selectedIds.value = []
    loadHistory()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 导出
const handleExport = async () => {
  try {
    const ids = selectedIds.value.length > 0 ? selectedIds.value : undefined
    const data = await exportExecutionHistory(ids)

    // 转换为 CSV
    const headers = ['ID', '任务名称', '任务类型', '状态', '目标主机', '执行用户', '执行时间']
    const typeMap: Record<string, string> = {
      manual: '手动执行',
      script: '脚本执行',
      file: '文件分发',
      command: '系统命令'
    }
    const statusMap: Record<string, string> = {
      pending: '等待中',
      running: '执行中',
      success: '成功',
      failed: '失败'
    }

    const csvContent = [
      headers.join(','),
      ...data.map((item: any) => [
        item.id,
        `"${(item.name || '').replace(/"/g, '""')}"`,
        typeMap[item.taskType] || item.taskType,
        statusMap[item.status] || item.status,
        `"${(item.targetHostsDisplay || '').replace(/"/g, '""')}"`,
        `"${(item.createdByName || '未知用户').replace(/"/g, '""')}"`,
        item.createdAt
      ].join(','))
    ].join('\n')

    // 添加 BOM 以支持中文
    const BOM = '\uFEFF'
    const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `执行记录_${new Date().toISOString().slice(0, 10)}.csv`
    link.click()
    URL.revokeObjectURL(url)

    ElMessage.success('导出成功')
  } catch (error: any) {
    ElMessage.error(error.message || '导出失败')
  }
}

// 复制输出
const copyOutput = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text || '')
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.history-container {
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
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  background: #ecfeff;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #0891b2;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #a5f3fc;
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
  width: 200px;
}

.date-picker {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
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

/* 搜索框样式 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-icon {
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
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

/* 任务名称单元格 */
.task-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.task-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.task-icon.icon-manual {
  color: #409eff;
}

.task-icon.icon-script {
  color: #67c23a;
}

.task-icon.icon-file {
  color: #e6a23c;
}

.task-icon.icon-command {
  color: #409eff;
}

.task-name {
  font-weight: 500;
}

.description-text {
  color: #606266;
}

.hosts-ellipsis {
  display: inline-block;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: #000;
  color: #fff;
  font-size: 12px;
  flex-shrink: 0;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-view:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

/* 分页器 */
.pagination-container {
  padding: 12px 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.history-view-dialog) {
  border-radius: 12px;
}

:deep(.history-view-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.history-view-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.history-view-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 详情样式 */
.target-hosts-detail {
  max-height: 100px;
  overflow-y: auto;
  word-break: break-all;
}

/* 执行结果样式 */
.result-section {
  margin-top: 20px;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.result-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.result-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.host-result {
  background: #f8f9fa;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #e4e7ed;
}

.host-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f0f2f5;
  border-bottom: 1px solid #e4e7ed;
}

.host-header.success {
  background: #f0f9eb;
  border-bottom-color: #e1f3d8;
}

.host-header.failed {
  background: #fef0f0;
  border-bottom-color: #fde2e2;
}

.host-name {
  font-weight: 600;
  color: #303133;
}

.host-ip {
  color: #909399;
  font-size: 13px;
}

.host-error {
  padding: 12px 16px;
  background: #fef0f0;
  color: #f56c6c;
  font-size: 13px;
}

.error-label {
  font-weight: 600;
}

.host-output {
  padding: 12px 16px;
}

.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: #606266;
  font-size: 13px;
}

.output-content {
  margin: 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}

/* 错误信息样式 */
.error-section {
  margin-top: 20px;
}

.error-header {
  font-size: 16px;
  font-weight: 600;
  color: #f56c6c;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #fde2e2;
}

.error-content {
  padding: 12px;
  background: #fef0f0;
  color: #f56c6c;
  border-radius: 6px;
  font-size: 13px;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.el-input__wrapper),
:deep(.el-textarea__inner) {
  border-radius: 8px;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 1200px;
  min-width: 600px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }

  .search-inputs {
    flex-direction: column;
  }

  .search-input,
  .date-picker {
    width: 100%;
  }

  .header-actions {
    flex-direction: column;
  }
}
</style>
