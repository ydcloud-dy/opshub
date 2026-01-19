<template>
  <div class="alert-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">告警日志</h2>
          <p class="page-subtitle">查看域名监控告警的发送历史和状态</p>
        </div>
      </div>
      <div class="header-actions">
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
          <el-icon><List /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">总告警数</div>
          <div class="stat-value">{{ stats.totalAlerts }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">发送成功</div>
          <div class="stat-value">{{ stats.successSent }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">发送失败</div>
          <div class="stat-value">{{ stats.failedSent }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">今日告警</div>
          <div class="stat-value">{{ stats.todayAlerts }}</div>
        </div>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.domain"
          placeholder="搜索域名..."
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.alertType"
          placeholder="告警类型"
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <el-option label="域名无法访问" value="domain_down" />
          <el-option label="响应时间过高" value="high_response_time" />
          <el-option label="SSL证书即将过期" value="ssl_expiring" />
          <el-option label="SSL证书已过期" value="ssl_expired" />
          <el-option label="SSL证书无效" value="ssl_invalid" />
        </el-select>

        <el-select
          v-model="searchForm.status"
          placeholder="发送状态"
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
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
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="域名" prop="domain" min-width="200" />

        <el-table-column label="告警类型" width="140" align="center">
          <template #default="{ row }">
            <el-tag :type="getAlertTypeColor(row.alertType)" effect="dark">
              {{ getAlertTypeName(row.alertType) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="告警消息" prop="message" min-width="250" show-overflow-tooltip />

        <el-table-column label="发送通道" width="120" align="center">
          <template #default="{ row }">
            <span v-if="row.channelType">{{ getChannelTypeName(row.channelType) }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="发送状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'success'" type="success" effect="dark">成功</el-tag>
            <el-tag v-else-if="row.status === 'failed'" type="danger" effect="dark">失败</el-tag>
            <el-tag v-else type="warning" effect="dark">待发送</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="发送时间" prop="sentAt" width="180">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.sentAt) }}</span>
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
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="告警日志详情"
      width="700px"
      class="detail-dialog"
    >
      <div v-if="currentLog" class="detail-content">
        <div class="detail-info">
          <div class="info-item">
            <span class="info-label">域名:</span>
            <span class="info-value">{{ currentLog.domain }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">告警类型:</span>
            <el-tag :type="getAlertTypeColor(currentLog.alertType)" effect="dark">
              {{ getAlertTypeName(currentLog.alertType) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">发送状态:</span>
            <el-tag v-if="currentLog.status === 'success'" type="success" effect="dark">成功</el-tag>
            <el-tag v-else-if="currentLog.status === 'failed'" type="danger" effect="dark">失败</el-tag>
            <el-tag v-else type="warning" effect="dark">待发送</el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">发送通道:</span>
            <span class="info-value">{{ currentLog.channelType || '-' }}</span>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="section-title">告警消息</h4>
          <div class="message-content">{{ currentLog.message }}</div>
        </div>

        <div v-if="currentLog.status === 'failed' && currentLog.errorMsg" class="detail-section">
          <h4 class="section-title">错误信息</h4>
          <div class="error-content">{{ currentLog.errorMsg }}</div>
        </div>

        <div class="detail-info">
          <div class="info-item">
            <span class="info-label">发送时间:</span>
            <span class="info-value">{{ formatDateTime(currentLog.sentAt) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">创建时间:</span>
            <span class="info-value">{{ formatDateTime(currentLog.createdAt) }}</span>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh,
  Search,
  RefreshLeft,
  View,
  Document,
  List,
  CircleCheck,
  CircleClose,
  Warning
} from '@element-plus/icons-vue'
import {
  getAlertLogs,
  getAlertStats,
  type AlertLog
} from '@/api/alert-config'

const loading = ref(false)
const detailDialogVisible = ref(false)

// 搜索表单
const searchForm = reactive({
  domain: '',
  alertType: '',
  status: ''
})

// 统计数据
const stats = ref({
  totalAlerts: 0,
  successSent: 0,
  failedSent: 0,
  todayAlerts: 0
})

// 表格数据
const tableData = ref<AlertLog[]>([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 当前查看的日志
const currentLog = ref<AlertLog | null>(null)

// 获取告警类型颜色
const getAlertTypeColor = (type: string) => {
  const colorMap: Record<string, string> = {
    domain_down: 'danger',
    high_response_time: 'warning',
    ssl_expiring: 'warning',
    ssl_expired: 'danger',
    ssl_invalid: 'danger'
  }
  return colorMap[type] || ''
}

// 获取告警类型名称
const getAlertTypeName = (type: string) => {
  const nameMap: Record<string, string> = {
    domain_down: '域名无法访问',
    high_response_time: '响应时间过高',
    ssl_expiring: 'SSL即将过期',
    ssl_expired: 'SSL已过期',
    ssl_invalid: 'SSL无效'
  }
  return nameMap[type] || type
}

// 获取通道类型名称
const getChannelTypeName = (type: string) => {
  const nameMap: Record<string, string> = {
    email: '邮件',
    webhook: 'Webhook',
    wechat: '企业微信',
    dingtalk: '钉钉',
    feishu: '飞书'
  }
  return nameMap[type] || type
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return '-'
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize
    }
    if (searchForm.domain) params.domainMonitorId = searchForm.domain
    if (searchForm.alertType) params.alertType = searchForm.alertType
    if (searchForm.status) params.status = searchForm.status

    const result = await getAlertLogs(params)
    tableData.value = result?.data || []
    pagination.total = result?.total || 0
  } catch (error: any) {
    ElMessage.error('加载数据失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

// 加载统计数据
const loadStats = async () => {
  try {
    const data = await getAlertStats()
    stats.value = data || { totalAlerts: 0, successSent: 0, failedSent: 0, todayAlerts: 0 }
  } catch (error: any) {
    console.error(error)
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadData()
}

// 重置搜索
const handleReset = () => {
  searchForm.domain = ''
  searchForm.alertType = ''
  searchForm.status = ''
  pagination.page = 1
  loadData()
}

// 分页大小变化
const handleSizeChange = (size: number) => {
  pagination.pageSize = size
  pagination.page = 1
  loadData()
}

// 页码变化
const handlePageChange = (page: number) => {
  pagination.page = page
  loadData()
}

// 查看详情
const handleView = (row: AlertLog) => {
  currentLog.value = row
  detailDialogVisible.value = true
}

onMounted(() => {
  loadData()
  loadStats()
})
</script>

<style scoped>
.alert-logs-container {
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

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
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

/* 详情对话框 */
:deep(.detail-dialog) {
  border-radius: 12px;
}

:deep(.detail-dialog .el-dialog__header) {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

:deep(.detail-dialog .el-dialog__title) {
  color: #d4af37;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.info-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.info-label {
  color: #606266;
  font-weight: 600;
  font-size: 14px;
  min-width: 70px;
}

.info-value {
  color: #303133;
  font-size: 14px;
  font-weight: 500;
}

.detail-section {
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.section-title {
  margin: 0 0 12px 0;
  font-size: 15px;
  color: #303133;
  font-weight: 600;
}

.message-content {
  color: #303133;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.error-content {
  background: #fef0f0;
  color: #f56c6c;
  padding: 12px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
