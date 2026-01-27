<template>
  <div class="domain-monitor-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">域名监控</h2>
          <p class="page-subtitle">实时监控域名的可用性、响应时间和SSL证书状态</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增监控
        </el-button>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
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
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="监控状态"
          clearable
          class="search-input"
        >
          <el-option label="正常" value="normal" />
          <el-option label="异常" value="abnormal" />
          <el-option label="暂停" value="paused" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
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
          <div class="stat-label">监控总数</div>
          <div class="stat-value">{{ stats.total }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">正常</div>
          <div class="stat-value">{{ stats.normal }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">异常</div>
          <div class="stat-value">{{ stats.abnormal }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">暂停</div>
          <div class="stat-value">{{ stats.paused }}</div>
        </div>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="域名" prop="domain" min-width="200">
          <template #default="{ row }">
            <div class="domain-cell">
              <el-link :href="`http://${row.domain}`" target="_blank" type="primary">
                {{ row.domain }}
              </el-link>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'normal'" type="success" effect="dark">正常</el-tag>
            <el-tag v-else-if="row.status === 'abnormal'" type="danger" effect="dark">异常</el-tag>
            <el-tag v-else type="warning" effect="dark">暂停</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="响应时间" width="120" align="center">
          <template #default="{ row }">
            <span :class="getResponseTimeClass(row.responseTime)">
              {{ row.responseTime }}ms
            </span>
          </template>
        </el-table-column>

        <el-table-column label="SSL证书" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.sslValid" type="success" size="small">有效</el-tag>
            <el-tag v-else type="danger" size="small">无效</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="SSL到期时间" prop="sslExpiry" width="180" />

        <el-table-column label="检查间隔" width="120" align="center">
          <template #default="{ row }">
            {{ Math.round(row.checkInterval / 60) }}分钟
          </template>
        </el-table-column>

        <el-table-column label="最后检查" prop="lastCheck" width="180">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.lastCheck) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="查看详情" placement="top">
                <el-button link class="action-btn action-view" @click="handleView(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="立即检查" placement="top">
                <el-button link class="action-btn action-check" @click="handleCheck(row)" :loading="row.checking">
                  <el-icon><Refresh /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
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
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      class="monitor-edit-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="请输入域名，如：example.com" />
        </el-form-item>
        <el-form-item label="检查间隔" prop="checkInterval">
          <el-input-number v-model="form.checkInterval" :min="1" :max="1440" style="width: 100%;" />
          <div class="form-tip">监控检查间隔（分钟），范围：1-1440</div>
        </el-form-item>
        <el-form-item label="启用SSL检查" prop="enableSSL">
          <el-switch v-model="form.enableSSL" />
          <div class="form-tip">启用后将检查SSL证书有效性</div>
        </el-form-item>
        <el-form-item label="告警通知" prop="enableAlert">
          <el-switch v-model="form.enableAlert" />
          <div class="form-tip">启用后当域名异常时将发送告警通知</div>
        </el-form-item>

        <!-- 告警配置 -->
        <template v-if="form.enableAlert">
          <el-divider content-position="left">告警触发条件</el-divider>

          <el-form-item label="响应时间阈值" prop="responseThreshold">
            <el-input-number v-model="form.responseThreshold" :min="100" :max="30000" :step="100" style="width: 100%;" />
            <div class="form-tip">响应时间超过此值时触发告警（毫秒），默认1000ms</div>
          </el-form-item>

          <el-form-item label="SSL过期告警" prop="sslExpiryDays">
            <el-input-number v-model="form.sslExpiryDays" :min="1" :max="365" style="width: 100%;" />
            <div class="form-tip">SSL证书过期前多少天开始告警，默认30天</div>
          </el-form-item>

          <el-alert
            title="告警通道配置提示"
            type="info"
            :closable="false"
            show-icon
            style="margin-bottom: 16px;"
          >
            <template #default>
              <div>告警通知支持邮件、企业微信、钉钉、飞书、Webhook等方式。</div>
              <div style="margin-top: 4px; color: #909399; font-size: 12px;">请在"告警配置"页面配置通知通道和接收人。</div>
            </template>
          </el-alert>
        </template>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="域名监控详情"
      width="700px"
      class="detail-dialog"
    >
      <div v-if="currentDomain" class="detail-content">
        <div class="detail-info">
          <div class="info-item">
            <span class="info-label">域名:</span>
            <span class="info-value">
              <el-link :href="`http://${currentDomain.domain}`" target="_blank" type="primary">
                {{ currentDomain.domain }}
              </el-link>
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">状态:</span>
            <el-tag v-if="currentDomain.status === 'normal'" type="success" effect="dark">正常</el-tag>
            <el-tag v-else-if="currentDomain.status === 'abnormal'" type="danger" effect="dark">异常</el-tag>
            <el-tag v-else type="warning" effect="dark">暂停</el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">响应时间:</span>
            <span :class="['info-value', getResponseTimeClass(currentDomain.responseTime)]">
              {{ currentDomain.responseTime }}ms
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">SSL证书:</span>
            <el-tag v-if="currentDomain.sslValid" type="success" size="small">有效</el-tag>
            <el-tag v-else type="danger" size="small">无效</el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">SSL到期:</span>
            <span class="info-value">{{ formatDateTime(currentDomain.sslExpiry) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">检查间隔:</span>
            <span class="info-value">{{ Math.round(currentDomain.checkInterval / 60) }}分钟</span>
          </div>
          <div class="info-item">
            <span class="info-label">最后检查:</span>
            <span class="info-value">{{ formatDateTime(currentDomain.lastCheck) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">创建时间:</span>
            <span class="info-value">{{ formatDateTime(currentDomain.createdAt) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">更新时间:</span>
            <span class="info-value">{{ formatDateTime(currentDomain.updatedAt) }}</span>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="section-title">检查历史</h4>
          <el-empty v-if="!checkHistory.length" description="暂无检查历史" :image-size="80" />
          <el-timeline v-else>
            <el-timeline-item
              v-for="item in checkHistory"
              :key="item.id"
              :timestamp="item.time"
              placement="top"
            >
              <div class="history-item">
                <span class="history-status" :class="`history-${item.status}`">
                  {{ item.status === 'success' ? '正常' : '异常' }}
                </span>
                <span class="history-time">{{ item.responseTime }}ms</span>
                <span v-if="item.message" class="history-message">{{ item.message }}</span>
              </div>
            </el-timeline-item>
          </el-timeline>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import {
  Plus,
  Search,
  RefreshLeft,
  Refresh,
  Edit,
  Delete,
  View,
  Monitor,
  List,
  CircleCheck,
  CircleClose,
  Warning
} from '@element-plus/icons-vue'
import {
  getDomainMonitors,
  getDomainMonitor,
  createDomainMonitor,
  updateDomainMonitor,
  deleteDomainMonitor,
  checkDomain,
  getDomainStats,
  getDomainCheckHistory
} from '@/api/domain-monitor'

const loading = ref(false)
const dialogVisible = ref(false)
const detailDialogVisible = ref(false)
const dialogTitle = ref('')
const submitting = ref(false)
const formRef = ref<FormInstance>()

// 搜索表单
const searchForm = reactive({
  domain: '',
  status: ''
})

// 统计数据
const stats = ref({
  total: 0,
  normal: 0,
  abnormal: 0,
  paused: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 当前查看的域名
const currentDomain = ref<any>(null)

// 检查历史（模拟数据）
const checkHistory = ref<any[]>([])

// 表单数据
const form = reactive({
  id: 0,
  domain: '',
  checkInterval: 5,
  enableSSL: true,
  enableAlert: false,
  responseThreshold: 1000,
  sslExpiryDays: 30
})

const rules: FormRules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  checkInterval: [{ required: true, message: '请输入检查间隔', trigger: 'blur' }]
}

// 过滤后的数据
const filteredData = computed(() => {
  if (!searchForm.domain && !searchForm.status) {
    return tableData.value
  }
  return tableData.value.filter(item => {
    const matchDomain = !searchForm.domain || item.domain.toLowerCase().includes(searchForm.domain.toLowerCase())
    const matchStatus = !searchForm.status || item.status === searchForm.status
    return matchDomain && matchStatus
  })
})

// 获取响应时间样式类
const getResponseTimeClass = (time: number) => {
  if (time === 0) return 'response-time-error'
  if (time < 200) return 'response-time-good'
  if (time < 500) return 'response-time-warning'
  return 'response-time-error'
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined | Date) => {
  if (!dateTime) return '-'
  // 如果是 Date 对象，转换为字符串
  let dateStr = ''
  if (dateTime instanceof Date) {
    const year = dateTime.getFullYear()
    const month = String(dateTime.getMonth() + 1).padStart(2, '0')
    const day = String(dateTime.getDate()).padStart(2, '0')
    const hours = String(dateTime.getHours()).padStart(2, '0')
    const minutes = String(dateTime.getMinutes()).padStart(2, '0')
    const seconds = String(dateTime.getSeconds()).padStart(2, '0')
    dateStr = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  } else {
    // 如果是字符串，移除T和时区部分
    dateStr = String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
  }
  return dateStr
}

// 重置搜索
const handleReset = () => {
  searchForm.domain = ''
  searchForm.status = ''
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const [monitors, statsData] = await Promise.all([
      getDomainMonitors(),
      getDomainStats()
    ])
    tableData.value = monitors || []
    stats.value = statsData || { total: 0, normal: 0, abnormal: 0, paused: 0 }
  } catch (error: any) {
    ElMessage.error('加载数据失败')
  } finally {
    loading.value = false
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增域名监控'
  Object.assign(form, {
    id: 0,
    domain: '',
    checkInterval: 5,
    enableSSL: true,
    enableAlert: false,
    responseThreshold: 1000,
    sslExpiryDays: 30
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogTitle.value = '编辑域名监控'
  Object.assign(form, {
    id: row.id,
    domain: row.domain,
    checkInterval: Math.round(row.checkInterval / 60), // 秒转分钟
    enableSSL: row.enableSSL ?? true,
    enableAlert: row.enableAlert ?? false,
    responseThreshold: row.responseThreshold ?? 1000,
    sslExpiryDays: row.sslExpiryDays ?? 30
  })
  dialogVisible.value = true
}

// 查看详情
const handleView = async (row: any) => {
  try {
    loading.value = true
    const [detail, historyData] = await Promise.all([
      getDomainMonitor(row.id),
      getDomainCheckHistory(row.id, 1, 20)
    ])
    console.log('Domain detail:', detail)
    console.log('History data:', historyData)
    currentDomain.value = detail
    // 转换检查历史数据格式
    if (historyData && historyData.list) {
      console.log('History list:', historyData.list)
      checkHistory.value = historyData.list.map((item: any) => ({
        id: item.id,
        time: formatDateTime(item.checkedAt),
        status: item.status === 'normal' ? 'success' : 'error',
        responseTime: item.responseTime,
        message: item.errorMessage || ''
      }))
    } else {
      console.log('No history list found in:', historyData)
      checkHistory.value = []
    }
    detailDialogVisible.value = true
  } catch (error: any) {
    console.error('Error fetching details:', error)
    ElMessage.error('获取详情失败')
  } finally {
    loading.value = false
  }
}

// 立即检查
const handleCheck = async (row: any) => {
  try {
    row.checking = true
    await checkDomain(row.id)
    ElMessage.success('检查完成')
    await loadData()
  } catch (error: any) {
    ElMessage.error('检查失败')
  } finally {
    row.checking = false
  }
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该域名监控吗？', '提示', { type: 'warning' })
    loading.value = true
    await deleteDomainMonitor(row.id)
    ElMessage.success('删除成功')
    await loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  } finally {
    loading.value = false
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        // 转换检查间隔：前端是分钟，后端是秒
        const submitData = {
          ...form,
          checkInterval: form.checkInterval * 60
        }

        if (form.id) {
          await updateDomainMonitor(form.id, submitData)
          ElMessage.success('更新成功')
        } else {
          const result = await createDomainMonitor(submitData)
          ElMessage.success('创建成功')
          // 创建成功后自动执行一次检查
          try {
            await checkDomain(result.id)
          } catch (checkError) {
            // First check failed, but monitor was created
          }
        }
        dialogVisible.value = false
        await loadData()
      } catch (error: any) {
        ElMessage.error(error.response?.data?.message || '操作失败')
      } finally {
        submitting.value = false
      }
    }
  })
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

onMounted(() => {
  loadData()
  // 每30秒自动刷新数据
  setInterval(() => {
    loadData()
  }, 30000)
})
</script>

<style scoped>
.domain-monitor-container {
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

.domain-cell {
  display: flex;
  align-items: center;
}

.response-time-good {
  color: #67c23a;
  font-weight: 500;
}

.response-time-warning {
  color: #e6a23c;
  font-weight: 500;
}

.response-time-error {
  color: #f56c6c;
  font-weight: 500;
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

.action-check:hover {
  background-color: #e8f5e9;
  color: #67c23a;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
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

/* 表单提示 */
.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
  line-height: 1.5;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.monitor-edit-dialog) {
  border-radius: 12px;
}

:deep(.monitor-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.monitor-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.monitor-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
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
  margin: 0 0 16px 0;
  font-size: 15px;
  color: #303133;
  font-weight: 600;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.history-status {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.history-success {
  background: #f0f9ff;
  color: #67c23a;
}

.history-error {
  background: #fef0f0;
  color: #f56c6c;
}

.history-time {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #909399;
}

.history-message {
  color: #606266;
  font-size: 13px;
}
</style>
