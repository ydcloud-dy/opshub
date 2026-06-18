<template>
  <div class="task-list-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><List /></el-icon>
        </div>
        <div>
          <h2 class="page-title">任务记录</h2>
          <p class="page-subtitle">查看证书签发、续期、部署等任务执行记录</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select
          v-model="searchForm.task_type"
          placeholder="任务类型"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="签发证书" value="issue" />
          <el-option label="续期证书" value="renew" />
          <el-option label="部署证书" value="deploy" />
        </el-select>

        <el-select
          v-model="searchForm.status"
          placeholder="任务状态"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="待执行" value="pending" />
          <el-option label="执行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
        </el-select>

        <el-select
          v-model="searchForm.trigger_type"
          placeholder="触发方式"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="自动" value="auto" />
          <el-option label="手动" value="manual" />
        </el-select>
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
        :data="tableData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#f8fafc', color: '#475467', fontWeight: '700' }"
      >
        <el-table-column label="ID" prop="id" width="80" align="center">
          <template #default="{ row }">
            <span class="task-id">#{{ row.id }}</span>
          </template>
        </el-table-column>

        <el-table-column label="关联证书" width="300">
          <template #default="{ row }">
            <div v-if="row.certificate" class="cert-cell">
              <el-tooltip :content="row.certificate.name" placement="top" :disabled="!row.certificate.name">
                <span class="cert-name">{{ row.certificate.name }}</span>
              </el-tooltip>
              <el-tooltip :content="row.certificate.domain" placement="top" :disabled="!row.certificate.domain">
                <span class="cert-domain">{{ row.certificate.domain }}</span>
              </el-tooltip>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="任务类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.task_type === 'issue'" type="primary">签发证书</el-tag>
            <el-tag v-else-if="row.task_type === 'renew'" type="warning">续期证书</el-tag>
            <el-tag v-else type="success">部署证书</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'pending'" type="info">待执行</el-tag>
            <el-tag v-else-if="row.status === 'running'" type="warning" class="running-tag">
              <el-icon class="is-loading" style="margin-right: 4px;"><Loading /></el-icon>
              <span>执行中</span>
            </el-tag>
            <el-tag v-else-if="row.status === 'success'" type="success">成功</el-tag>
            <el-tag v-else type="danger">失败</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="触发方式" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.trigger_type === 'auto'" size="small">自动</el-tag>
            <el-tag v-else size="small" type="info">手动</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="开始时间" width="190" class-name="time-column">
          <template #default="{ row }">
            <span class="task-time-text">{{ formatDateTime(row.started_at) || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="结束时间" width="190" class-name="time-column">
          <template #default="{ row }">
            <span class="task-time-text">{{ formatDateTime(row.finished_at) || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="错误信息" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.error_message" class="error-text">{{ row.error_message }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-tooltip content="查看详情" placement="top">
              <el-button link class="action-btn action-view" @click="handleView(row)">
                <el-icon><View /></el-icon>
              </el-button>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="任务详情"
      width="640px"
      class="beauty-dialog"
    >
      <div v-if="currentTask" class="detail-content">
        <div class="detail-status-bar" :class="{
          'status-success': currentTask.status === 'success',
          'status-failed': currentTask.status === 'failed',
          'status-running': currentTask.status === 'running',
          'status-pending': currentTask.status === 'pending'
        }">
          <el-tag v-if="currentTask.status === 'pending'" type="info" effect="dark" size="large">待执行</el-tag>
          <el-tag v-else-if="currentTask.status === 'running'" type="warning" effect="dark" size="large">执行中</el-tag>
          <el-tag v-else-if="currentTask.status === 'success'" type="success" effect="dark" size="large">成功</el-tag>
          <el-tag v-else type="danger" effect="dark" size="large">失败</el-tag>
          <span class="detail-task-type">{{ getTaskTypeName(currentTask.task_type) }}</span>
          <span class="detail-task-id">#{{ currentTask.id }}</span>
        </div>
        <div class="detail-info">
          <div class="detail-info-section">
            <div class="detail-section-title">任务信息</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">任务类型</span>
                <span class="info-value">{{ getTaskTypeName(currentTask.task_type) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">触发方式</span>
                <span class="info-value">{{ currentTask.trigger_type === 'auto' ? '自动' : '手动' }}</span>
              </div>
              <div class="info-item" v-if="currentTask.certificate">
                <span class="info-label">关联证书</span>
                <span class="info-value">{{ currentTask.certificate.name }} ({{ currentTask.certificate.domain }})</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">时间</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.created_at) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">开始时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.started_at) || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">结束时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.finished_at) || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section" v-if="currentTask.error_message">
            <div class="detail-section-title error-section-title">错误信息</div>
            <div class="error-block">{{ currentTask.error_message }}</div>
          </div>
          <div class="detail-info-section" v-if="currentTask.result">
            <div class="detail-section-title">执行结果</div>
            <div class="result-block">
              <pre class="result-json">{{ formatJSON(currentTask.result) }}</pre>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, RefreshLeft, List, View, Loading } from '@element-plus/icons-vue'
import { getTasks, getTask } from '../api/ssl-cert'

const loading = ref(false)
const detailDialogVisible = ref(false)

// 搜索
const searchForm = reactive({
  task_type: '',
  status: '',
  trigger_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 当前任务
const currentTask = ref<any>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null

// 获取任务类型名称
const getTaskTypeName = (type: string) => {
  const names: Record<string, string> = {
    issue: '签发证书',
    renew: '续期证书',
    deploy: '部署证书'
  }
  return names[type] || type
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 格式化JSON
const formatJSON = (jsonStr: string) => {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
}

// 重置搜索
const handleReset = () => {
  searchForm.task_type = ''
  searchForm.status = ''
  searchForm.trigger_type = ''
  pagination.page = 1
  loadData()
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getTasks({
      page: pagination.page,
      page_size: pagination.pageSize,
      task_type: searchForm.task_type || undefined,
      status: searchForm.status || undefined,
      trigger_type: searchForm.trigger_type || undefined
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 查看详情
const handleView = async (row: any) => {
  try {
    const res = await getTask(row.id)
    currentTask.value = res
    detailDialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

onMounted(() => {
  loadData()
  // 每30秒自动刷新
  refreshTimer = setInterval(() => {
    loadData()
  }, 30000)
})

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<style scoped>
.task-list-container {
  padding: 0;
  background-color: transparent;
  color: #344054;
  font-family: inherit;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 18px 22px;
  background: linear-gradient(135deg, #ffffff 0%, #fbfdff 100%);
  border: 1px solid #e5e9f2;
  border-radius: 14px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.06);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 46px;
  height: 46px;
  background: linear-gradient(135deg, #eff6ff 0%, #f8fafc 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2563eb;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #dbeafe;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 750;
  color: #101828;
  letter-spacing: -0.01em;
}

.page-subtitle {
  margin: 6px 0 0 0;
  font-size: 13px;
  color: #667085;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.header-actions :deep(.el-button) {
  height: 36px;
  border-radius: 9px;
  font-weight: 600;
}

.search-bar {
  margin-bottom: 16px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
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
  width: 160px;
}

.search-bar :deep(.el-input__wrapper),
.search-bar :deep(.el-select__wrapper) {
  min-height: 36px;
  border-radius: 9px;
  box-shadow: 0 0 0 1px #d9e1ec inset;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  height: 36px;
  background: #f8fafc;
  border-color: #d9e1ec;
  border-radius: 9px;
  color: #475467;
  font-weight: 600;
}

.table-wrapper {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 14px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

.modern-table {
  width: 100%;
  color: #344054;
  font-size: 13px;
}

.modern-table :deep(.el-table__header th) {
  background: #f8fafc !important;
  color: #475467 !important;
  font-weight: 700 !important;
}

.modern-table :deep(.el-table__cell) {
  padding: 12px 0;
}

.modern-table :deep(.cell) {
  line-height: 1.45;
}

.modern-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: #f8fafc;
}

.task-id {
  color: #475467;
  font-weight: 700;
  letter-spacing: 0;
}

.cert-cell {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cert-name,
.cert-domain {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cert-name {
  color: #344054;
  font-weight: 700;
}

.cert-domain {
  color: #667085;
  font-size: 12px;
  font-family: inherit;
  font-weight: 500;
}

.modern-table :deep(.time-column .cell) {
  overflow: visible;
  text-overflow: clip;
  white-space: nowrap;
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
  display: inline-block;
}

.error-text {
  color: #f56c6c;
}

.task-time-text {
  display: inline-flex;
  width: 100%;
  align-items: center;
  white-space: nowrap;
  color: #344054;
  font-family: inherit;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.action-btn {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  transition: all 0.2s ease;
  color: #667085;
}

.action-view:hover {
  background-color: #e8f4ff;
  color: #409eff;
  transform: translateY(-1px);
}

.pagination-wrapper {
  padding: 14px 16px;
  border-top: 1px solid #eef2f7;
  display: flex;
  justify-content: flex-end;
}

.running-tag {
  display: inline-flex;
  align-items: center;
}

/* 弹窗美化 */
:deep(.beauty-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.beauty-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  margin-right: 0;
  border-bottom: 1px solid #f0f0f0;
  background: #fafbfc;
}

:deep(.beauty-dialog .el-dialog__title) {
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.beauty-dialog .el-dialog__headerbtn) {
  top: 20px;
  right: 20px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-dialog .el-dialog__headerbtn:hover) {
  background: #f0f0f0;
}

:deep(.beauty-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 65vh;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

:deep(.beauty-dialog .el-dialog__body::-webkit-scrollbar) {
  display: none;
}

:deep(.beauty-dialog .el-dialog__footer) {
  padding: 16px 24px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafbfc;
}

/* 详情弹窗 */
.detail-content {
  padding: 0;
}

.detail-status-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding: 16px 18px;
  background: linear-gradient(135deg, #f8fafc 0%, #ffffff 100%);
  border-radius: 12px;
  border: 1px solid #e5e9f2;
}

.detail-task-type {
  font-size: 18px;
  font-weight: 750;
  color: #101828;
}

.detail-task-id {
  font-size: 14px;
  color: #667085;
  font-family: inherit;
  font-weight: 700;
  margin-left: auto;
}

.detail-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-info-section {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 12px;
  overflow: hidden;
}

.detail-section-title {
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 700;
  color: #475467;
  letter-spacing: 0;
  background: #f8fafc;
  border-bottom: 1px solid #e5e9f2;
}

.error-section-title {
  color: #f56c6c;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0;
}

.detail-grid .info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  border-bottom: 1px solid #f5f5f5;
  border-right: 1px solid #f5f5f5;
}

.detail-grid .info-item:nth-child(2n) {
  border-right: none;
}

.detail-grid .info-item:last-child,
.detail-grid .info-item:nth-last-child(2):nth-child(odd) {
  border-bottom: none;
}

.info-label {
  color: #667085;
  font-size: 12px;
  font-weight: 500;
}

.info-value {
  color: #344054;
  font-size: 14px;
  word-break: break-all;
  font-weight: 500;
}

.error-block {
  padding: 14px 16px;
  font-size: 13px;
  color: #f56c6c;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.result-block {
  padding: 0;
}

.result-json {
  background: #f8fafc;
  padding: 14px 16px;
  font-size: 12px;
  overflow-x: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', monospace;
  line-height: 1.6;
  color: #475467;
}
</style>
