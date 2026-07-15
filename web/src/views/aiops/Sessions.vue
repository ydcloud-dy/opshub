<template>
  <div class="sessions-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Tickets /></el-icon>
        </div>
        <div>
          <h2 class="page-title">AI会话记录</h2>
          <p class="page-subtitle">查看 AI 问答、日志分析、智能诊断和工具调用审计记录</p>
        </div>
      </div>
      <el-button @click="loadData">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-tabs v-model="activeTab" class="record-tabs" @tab-change="loadData">
      <el-tab-pane label="会话记录" name="sessions">
        <div class="filter-bar">
          <el-select v-model="filters.type" placeholder="会话类型" clearable @change="loadSessions">
            <el-option label="AI助手" value="chat" />
            <el-option label="智能诊断" value="diagnosis" />
            <el-option label="日志分析" value="log" />
          </el-select>
        </div>
        <div class="table-wrapper">
          <el-table v-loading="loading" :data="sessions" class="modern-table">
            <el-table-column prop="id" label="ID" width="80" align="center" />
            <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
            <el-table-column label="类型" width="110">
              <template #default="{ row }">
                <el-tag effect="plain">{{ typeLabel(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="messageCount" label="消息数" width="100" align="center" />
            <el-table-column prop="summary" label="摘要" min-width="260" show-overflow-tooltip />
            <el-table-column label="更新时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link class="action-btn action-view" @click="openSession(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-wrap">
            <el-pagination
              v-model:current-page="pagination.page"
              v-model:page-size="pagination.pageSize"
              :total="pagination.total"
              layout="total, sizes, prev, pager, next"
              @current-change="loadSessions"
              @size-change="loadSessions"
            />
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="诊断任务" name="tasks">
        <div class="table-wrapper">
          <el-table v-loading="taskLoading" :data="tasks" class="modern-table">
            <el-table-column prop="id" label="ID" width="80" align="center" />
            <el-table-column prop="objectType" label="对象类型" width="120" />
            <el-table-column label="对象" min-width="240">
              <template #default="{ row }">
                {{ row.namespace ? `${row.namespace}/` : '' }}{{ row.objectName || '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="clusterId" label="集群ID" width="100" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" effect="plain">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="conclusion" label="结论" min-width="280" show-overflow-tooltip />
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link class="action-btn action-view" @click="openTask(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-wrap">
            <el-pagination
              v-model:current-page="taskPagination.page"
              v-model:page-size="taskPagination.pageSize"
              :total="taskPagination.total"
              layout="total, sizes, prev, pager, next"
              @current-change="loadTasks"
              @size-change="loadTasks"
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="detailVisible" title="会话详情" width="min(1280px, 92vw)" top="4vh" class="session-dialog">
      <div v-if="detail" class="detail-layout">
        <div class="message-list">
          <div v-for="msg in detail.messages" :key="msg.id" class="detail-message" :class="msg.role">
            <div class="detail-role">{{ roleLabel(msg.role) }}</div>
            <MarkdownView v-if="msg.role === 'assistant'" :content="msg.content" />
            <pre v-else>{{ msg.content }}</pre>
          </div>
        </div>
        <div class="tool-list">
          <div class="tool-title">工具调用</div>
          <el-empty v-if="!detail.tools?.length" description="暂无工具调用" :image-size="80" />
          <div v-for="tool in detail.tools" :key="tool.id" class="tool-card">
            <strong>{{ tool.toolName }}</strong>
            <span>{{ tool.status }} · {{ formatDateTime(tool.createdAt) }}</span>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="taskDetailVisible" title="诊断任务详情" width="min(1120px, 92vw)" top="6vh" class="session-dialog">
      <div v-if="currentTask" class="task-detail">
        <div class="task-meta">
          <el-tag effect="plain">{{ currentTask.objectType }}</el-tag>
          <el-tag :type="currentTask.status === 'success' ? 'success' : 'danger'" effect="plain">{{ currentTask.status }}</el-tag>
        </div>
        <MarkdownView :content="currentTask.conclusion || currentTask.error" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh, Tickets, View } from '@element-plus/icons-vue'
import { getAIDiagnosisTasks, getAISessionDetail, getAISessions } from '@/api/aiops'
import MarkdownView from './components/MarkdownView.vue'

const activeTab = ref('sessions')
const loading = ref(false)
const taskLoading = ref(false)
const sessions = ref<any[]>([])
const tasks = ref<any[]>([])
const detailVisible = ref(false)
const detail = ref<any>(null)
const taskDetailVisible = ref(false)
const currentTask = ref<any>(null)

const filters = reactive({ type: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const taskPagination = reactive({ page: 1, pageSize: 20, total: 0 })

const typeLabel = (type: string) => {
  const map: Record<string, string> = {
    chat: 'AI助手',
    diagnosis: '智能诊断',
    log: '日志分析'
  }
  return map[type] || type
}

const roleLabel = (role: string) => {
  const map: Record<string, string> = {
    user: '用户',
    assistant: 'AI助手',
    system: '系统'
  }
  return map[role] || role
}

const formatDateTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const normalizePage = (res: any) => {
  return {
    data: res?.data || [],
    total: res?.total || 0,
    page: res?.page || 1,
    pageSize: res?.page_size || res?.pageSize || 20
  }
}

const loadSessions = async () => {
  loading.value = true
  try {
    const res = normalizePage(await getAISessions({
      page: pagination.page,
      pageSize: pagination.pageSize,
      type: filters.type || undefined
    }))
    sessions.value = res.data
    pagination.total = res.total
  } finally {
    loading.value = false
  }
}

const loadTasks = async () => {
  taskLoading.value = true
  try {
    const res = normalizePage(await getAIDiagnosisTasks({
      page: taskPagination.page,
      pageSize: taskPagination.pageSize
    }))
    tasks.value = res.data
    taskPagination.total = res.total
  } finally {
    taskLoading.value = false
  }
}

const loadData = () => {
  if (activeTab.value === 'sessions') {
    loadSessions()
  } else {
    loadTasks()
  }
}

const openSession = async (row: any) => {
  detail.value = await getAISessionDetail(row.id)
  detailVisible.value = true
}

const openTask = (row: any) => {
  currentTask.value = row
  taskDetailVisible.value = true
}

onMounted(() => {
  loadSessions()
})
</script>

<style scoped>
.sessions-page {
  min-height: 100%;
}

.page-header,
.table-wrapper {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 18px;
  box-shadow: 0 12px 32px rgba(31, 45, 61, 0.07);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px;
  margin-bottom: 18px;
}

.page-title-group {
  display: flex;
  gap: 14px;
  align-items: center;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #111827;
  color: #fff;
  font-size: 20px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  color: #111827;
}

.page-subtitle {
  color: #64748b;
  margin-top: 4px;
  font-size: 13px;
}

.record-tabs {
  background: #fff;
  border-radius: 18px;
  padding: 12px 18px 18px;
  border: 1px solid #e5e7eb;
}

.filter-bar {
  margin: 8px 0 14px;
}

.table-wrapper {
  padding: 14px;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.action-view {
  color: #2563eb;
}

.detail-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 18px;
  min-height: 640px;
}

.message-list {
  max-height: calc(92vh - 160px);
  overflow: auto;
  padding-right: 6px;
}

.detail-message {
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  padding: 16px;
  margin-bottom: 14px;
  background: #f8fafc;
  min-width: 0;
}

.detail-message.assistant {
  background: #f9fafb;
}

.detail-role {
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
  margin-bottom: 8px;
}

.detail-message pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.7;
  font-family: inherit;
}

.detail-message :deep(.markdown-view) {
  max-width: 100%;
  overflow-x: auto;
}

.detail-message :deep(table) {
  display: block;
  width: max-content;
  max-width: 100%;
  overflow-x: auto;
}

.tool-list {
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  background: #f8fafc;
  padding: 14px;
  max-height: calc(92vh - 160px);
  overflow: auto;
}

.tool-title {
  font-weight: 800;
  margin-bottom: 12px;
  color: #111827;
}

.tool-card {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 10px;
  margin-bottom: 10px;
  background: #fff;
  word-break: break-word;
}

.tool-card span {
  display: block;
  color: #64748b;
  font-size: 12px;
  margin-top: 4px;
}

.task-detail {
  display: grid;
  gap: 14px;
}

.task-meta {
  display: flex;
  gap: 10px;
  align-items: center;
}

.task-detail .markdown-view {
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #f8fafc;
  padding: 14px;
  max-height: calc(88vh - 190px);
  overflow: auto;
}

:deep(.modern-table .el-table__header th) {
  background: #f8fafc;
  color: #475569;
  font-weight: 700;
}

:deep(.session-dialog .el-dialog__body) {
  padding-top: 12px;
}

@media (max-width: 980px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .detail-layout {
    grid-template-columns: 1fr;
    min-height: 0;
  }

  .message-list,
  .tool-list {
    max-height: none;
  }
}
</style>
