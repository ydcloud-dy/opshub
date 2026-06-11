<template>
  <div class="probe-tasks-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Odometer /></el-icon>
        </div>
        <div>
          <h2 class="page-title">拨测任务</h2>
          <p class="page-subtitle">按 HTTP、ICMP、TCP、SSL 协议持续探测端点，并写入 Prometheus 远程写入数据源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          创建
        </el-button>
        <el-button @click="loadAll">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索任务名称、端点..."
          clearable
          class="search-input"
          @keyup.enter="loadTasks"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="searchForm.protocol" placeholder="拨测类型" clearable class="search-input small" @change="loadTasks">
          <el-option v-for="item in protocolOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="searchForm.status" placeholder="任务状态" clearable class="search-input small" @change="loadTasks">
          <el-option label="正常" value="normal" />
          <el-option label="异常" value="abnormal" />
          <el-option label="未执行" value="unknown" />
        </el-select>
      </div>
      <el-button class="reset-btn" @click="handleReset">
        <el-icon><RefreshLeft /></el-icon>
        重置
      </el-button>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <div class="stat-label">任务总数</div>
          <div class="stat-value">{{ tableData.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div>
          <div class="stat-label">启用任务</div>
          <div class="stat-value">{{ enabledCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><Warning /></el-icon>
        </div>
        <div>
          <div class="stat-label">异常任务</div>
          <div class="stat-value">{{ abnormalCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <div class="stat-label">可写数据源</div>
          <div class="stat-value">{{ remoteWriteSources.length }}</div>
        </div>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table
        :data="pagedTasks"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="tableHeaderStyle"
        height="560"
        empty-text="暂无拨测任务"
      >
        <el-table-column label="任务名称" min-width="210" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="task-name-cell">
              <span class="protocol-dot" :class="row.protocol">{{ getProtocolName(row.protocol).slice(0, 1) }}</span>
              <div>
                <strong>{{ row.name }}</strong>
                <em>{{ row.description || '无描述' }}</em>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="拨测类型" width="116" align="center">
          <template #default="{ row }">
            <el-tag :type="getProtocolTag(row.protocol)" effect="light">{{ getProtocolName(row.protocol) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="端点" prop="endpoint" min-width="280" show-overflow-tooltip />
        <el-table-column label="执行频率" width="110" align="center">
          <template #default="{ row }">{{ row.frequencySeconds }}s</template>
        </el-table-column>
        <el-table-column label="写入配置" min-width="190" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="source-cell">
              <el-tag v-if="row.writeRuleEnabled" :type="row.dataSourceRemoteOk ? 'success' : 'warning'" effect="light">
                {{ row.dataSourceRemoteOk ? '可写入' : '未就绪' }}
              </el-tag>
              <el-tag v-else type="info" effect="light">未写入</el-tag>
              <span>{{ row.dataSourceName || getDataSourceName(row.dataSourceId) || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="190">
          <template #default="{ row }">
            <span class="time-text">{{ formatDateTime(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作人" width="110" show-overflow-tooltip>
          <template #default="{ row }">{{ row.operator || 'admin' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusTag(row.status)" effect="light">{{ getStatusName(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="90" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleEnabledChange(row)" />
          </template>
        </el-table-column>
        <el-table-column label="最近结果" width="170" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.lastProbeAt" class="last-result" :class="row.status">
              {{ row.lastDurationMs || 0 }}ms · {{ formatDateTime(row.lastProbeAt) }}
            </span>
            <span v-else class="muted-text">未执行</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="156" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="拨测一下" placement="top">
                <el-button link class="action-btn action-run" :loading="row.running" @click="handleRun(row)">
                  <el-icon><VideoPlay /></el-icon>
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
      <div class="table-pagination">
        <span>共 {{ filteredTasks.length }} 条拨测任务</span>
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="filteredTasks.length"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
        />
      </div>
    </div>

    <el-drawer
      v-model="drawerVisible"
      :title="drawerTitle"
      size="940px"
      class="probe-drawer"
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="112px" class="probe-form">
        <div class="drawer-section">
          <div class="drawer-section-title">基础配置</div>
          <div class="form-grid">
            <el-form-item label="任务名称" prop="name">
              <el-input v-model="form.name" placeholder="如：官网可用性拨测" />
            </el-form-item>
            <el-form-item label="拨测类型" prop="protocol">
              <el-segmented v-model="form.protocol" :options="protocolOptions" @change="handleProtocolChange" />
            </el-form-item>
          </div>
          <el-form-item label="任务描述">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="请输入任务描述" />
          </el-form-item>
        </div>

        <div class="drawer-section">
          <div class="drawer-section-title">端点配置</div>
          <el-form-item label="端点" prop="endpoint">
            <el-input
              v-model="form.endpoint"
              type="textarea"
              :rows="4"
              :placeholder="getEndpointPlaceholder(form.protocol)"
            />
          </el-form-item>
          <template v-if="form.protocol === 'http'">
            <div class="form-grid">
              <el-form-item label="请求方法">
                <el-select v-model="form.method" style="width: 100%">
                  <el-option label="GET" value="GET" />
                  <el-option label="POST" value="POST" />
                  <el-option label="PUT" value="PUT" />
                  <el-option label="HEAD" value="HEAD" />
                </el-select>
              </el-form-item>
              <el-form-item label="请求头">
                <el-input v-model="form.headers" placeholder='{"X-App":"opshub"}' />
              </el-form-item>
            </div>
            <el-form-item v-if="form.method !== 'GET' && form.method !== 'HEAD'" label="请求体">
              <el-input v-model="form.body" type="textarea" :rows="3" placeholder='{"ping":"ok"}' />
            </el-form-item>
          </template>
        </div>

        <div class="drawer-section">
          <div class="drawer-section-title">策略配置</div>
          <div class="form-grid three">
            <el-form-item label="执行频率">
              <div class="inline-number-field">
                <el-input-number v-model="form.frequencySeconds" :min="10" :max="86400" controls-position="right" />
                <span class="form-suffix">秒</span>
              </div>
            </el-form-item>
            <el-form-item label="超时时间">
              <div class="inline-number-field">
                <el-input-number v-model="form.timeoutSeconds" :min="1" :max="120" controls-position="right" />
                <span class="form-suffix">秒</span>
              </div>
            </el-form-item>
            <el-form-item label="启用状态">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </div>
          <div v-if="form.protocol === 'icmp'" class="form-grid">
            <el-form-item label="请求次数">
              <el-input-number v-model="form.icmpCount" :min="1" :max="10" controls-position="right" style="width: 100%" />
            </el-form-item>
            <el-form-item label="请求间隔">
              <div class="inline-number-field">
                <el-input-number v-model="form.icmpIntervalMs" :min="100" :max="10000" :step="100" controls-position="right" />
                <span class="form-suffix">ms</span>
              </div>
            </el-form-item>
          </div>
        </div>

        <div class="drawer-section">
          <div class="drawer-section-title">写入配置</div>
          <el-alert
            class="write-alert"
            :type="remoteWriteSources.length ? 'info' : 'warning'"
            :closable="false"
            show-icon
            :title="remoteWriteSources.length ? '仅可选择已开启远程写入的数据源。保存后定时任务会写入 opshub_probe_* 指标。' : '暂无开启远程写入的 Prometheus / VictoriaMetrics 数据源，请先在数据源管理中配置。'"
          />
          <div class="form-grid">
            <el-form-item label="启用写入">
              <el-switch v-model="form.writeRuleEnabled" />
            </el-form-item>
            <el-form-item label="写入数据源" prop="dataSourceId">
              <el-select
                v-model="form.dataSourceId"
                filterable
                clearable
                placeholder="请选择 Prometheus / VictoriaMetrics"
                style="width: 100%"
                :disabled="!form.writeRuleEnabled"
              >
                <el-option
                  v-for="item in writableSourceOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                  :disabled="!item.remoteWriteEnabled"
                >
                  <div class="source-option">
                    <span>{{ item.name }}</span>
                    <el-tag :type="item.remoteWriteEnabled ? 'success' : 'warning'" size="small" effect="light">
                      {{ item.remoteWriteEnabled ? 'remote write' : '未开启' }}
                    </el-tag>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </div>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="resultDialogVisible" title="拨测结果" width="820px" class="probe-result-dialog">
      <div v-if="runSummary" class="run-summary" :class="runSummary.status">
        <strong>{{ runSummary.success ? '拨测成功' : '拨测异常' }}</strong>
        <span>总耗时 {{ runSummary.durationMs }}ms</span>
        <span v-if="runSummary.remoteWriteErr">写入失败：{{ runSummary.remoteWriteErr }}</span>
        <span v-else-if="runSummary.remoteWriteOk">已写入远程数据源</span>
      </div>
      <el-table :data="runSummary?.results || []" class="modern-table" :header-cell-style="tableHeaderStyle">
        <el-table-column label="端点" prop="endpoint" min-width="260" show-overflow-tooltip />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" effect="light">{{ row.success ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="100" align="right">
          <template #default="{ row }">{{ row.durationMs }}ms</template>
        </el-table-column>
        <el-table-column label="状态码" width="100" align="center">
          <template #default="{ row }">{{ row.statusCode || '-' }}</template>
        </el-table-column>
        <el-table-column label="证书剩余" width="110" align="center">
          <template #default="{ row }">{{ row.sslExpireAt ? `${row.sslDaysLeft} 天` : '-' }}</template>
        </el-table-column>
        <el-table-column label="结果" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.message || row.error || '-' }}</template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="resultDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  CircleCheck,
  Connection,
  Delete,
  Document,
  Edit,
  Odometer,
  Plus,
  Refresh,
  RefreshLeft,
  Search,
  VideoPlay,
  Warning
} from '@element-plus/icons-vue'
import {
  createMonitorProbeTask,
  deleteMonitorProbeTask,
  getMonitorDataSources,
  getMonitorProbeTasks,
  runMonitorProbeTask,
  updateMonitorProbeTask,
  type MonitorDataSource,
  type MonitorProbeRunSummary,
  type MonitorProbeTask,
  type ProbeProtocol
} from '@/api/monitor-datasource'

type ProbeTaskRow = MonitorProbeTask & { running?: boolean }

interface ProbeTaskForm {
  id?: number
  name: string
  protocol: ProbeProtocol
  endpoint: string
  method: string
  headers: string
  body: string
  frequencySeconds: number
  timeoutSeconds: number
  icmpCount: number
  icmpIntervalMs: number
  dataSourceId?: number
  writeRuleEnabled: boolean
  enabled: boolean
  description: string
  operator: string
}

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
const loading = ref(false)
const submitting = ref(false)
const drawerVisible = ref(false)
const resultDialogVisible = ref(false)
const drawerTitle = ref('创建拨测任务')
const formRef = ref<FormInstance>()
const tableData = ref<ProbeTaskRow[]>([])
const dataSources = ref<MonitorDataSource[]>([])
const runSummary = ref<MonitorProbeRunSummary>()

const searchForm = reactive({
  keyword: '',
  protocol: '' as ProbeProtocol | '',
  status: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10
})

const form = reactive<ProbeTaskForm>({
  name: '',
  protocol: 'http',
  endpoint: '',
  method: 'GET',
  headers: '',
  body: '',
  frequencySeconds: 60,
  timeoutSeconds: 5,
  icmpCount: 3,
  icmpIntervalMs: 1000,
  dataSourceId: undefined,
  writeRuleEnabled: true,
  enabled: true,
  description: '',
  operator: 'admin'
})

const protocolOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'ICMP', value: 'icmp' },
  { label: 'TCP', value: 'tcp' },
  { label: 'SSL', value: 'ssl' }
]

const rules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择拨测类型', trigger: 'change' }],
  endpoint: [{ required: true, message: '请输入拨测端点', trigger: 'blur' }]
}

const enabledCount = computed(() => tableData.value.filter(item => item.enabled).length)
const abnormalCount = computed(() => tableData.value.filter(item => item.status === 'abnormal').length)
const remoteWriteSources = computed(() => dataSources.value.filter(item => isPromCompatible(item.type) && item.enabled && item.remoteWriteEnabled))
const writableSourceOptions = computed(() => dataSources.value.filter(item => isPromCompatible(item.type) && item.enabled))
const filteredTasks = computed(() => tableData.value)
const pagedTasks = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize
  return filteredTasks.value.slice(start, start + pagination.pageSize)
})

watch(() => [searchForm.keyword, searchForm.protocol, searchForm.status], () => {
  pagination.page = 1
})

const isPromCompatible = (type?: string) => type === 'prometheus' || type === 'victoriametrics'

const loadTasks = async () => {
  loading.value = true
  try {
    tableData.value = await getMonitorProbeTasks({
      keyword: searchForm.keyword || undefined,
      protocol: searchForm.protocol || undefined,
      status: searchForm.status || undefined
    }) || []
  } finally {
    loading.value = false
  }
}

const loadDataSources = async () => {
  dataSources.value = await getMonitorDataSources() || []
}

const loadAll = async () => {
  await Promise.all([loadTasks(), loadDataSources()])
}

const handleReset = () => {
  searchForm.keyword = ''
  searchForm.protocol = ''
  searchForm.status = ''
  loadTasks()
}

const resetForm = () => {
  Object.assign(form, {
    id: undefined,
    name: '',
    protocol: 'http',
    endpoint: '',
    method: 'GET',
    headers: '',
    body: '',
    frequencySeconds: 60,
    timeoutSeconds: 5,
    icmpCount: 3,
    icmpIntervalMs: 1000,
    dataSourceId: remoteWriteSources.value[0]?.id,
    writeRuleEnabled: true,
    enabled: true,
    description: '',
    operator: 'admin'
  })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  drawerTitle.value = '创建拨测任务'
  drawerVisible.value = true
}

const handleEdit = (row: MonitorProbeTask) => {
  resetForm()
  Object.assign(form, {
    id: row.id,
    name: row.name,
    protocol: row.protocol,
    endpoint: row.endpoint,
    method: row.method || 'GET',
    headers: row.headers || '',
    body: row.body || '',
    frequencySeconds: row.frequencySeconds || 60,
    timeoutSeconds: row.timeoutSeconds || 5,
    icmpCount: row.icmpCount || 3,
    icmpIntervalMs: row.icmpIntervalMs || 1000,
    dataSourceId: row.dataSourceId || undefined,
    writeRuleEnabled: row.writeRuleEnabled ?? true,
    enabled: row.enabled,
    description: row.description || '',
    operator: row.operator || 'admin'
  })
  drawerTitle.value = '编辑拨测任务'
  drawerVisible.value = true
}

const handleProtocolChange = () => {
  form.endpoint = ''
  form.headers = ''
  form.body = ''
  form.method = 'GET'
}

const buildPayload = (): MonitorProbeTask => ({
  id: form.id,
  name: form.name,
  protocol: form.protocol,
  endpoint: form.endpoint,
  method: form.method,
  headers: form.headers,
  body: form.body,
  frequencySeconds: form.frequencySeconds,
  timeoutSeconds: form.timeoutSeconds,
  icmpCount: form.icmpCount,
  icmpIntervalMs: form.icmpIntervalMs,
  dataSourceId: form.writeRuleEnabled ? form.dataSourceId : 0,
  writeRuleEnabled: form.writeRuleEnabled,
  enabled: form.enabled,
  description: form.description,
  operator: form.operator || 'admin'
})

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (form.writeRuleEnabled) {
    const source = dataSources.value.find(item => item.id === form.dataSourceId)
    if (!source || !source.remoteWriteEnabled || !isPromCompatible(source.type)) {
      ElMessage.warning('请选择已开启远程写入的 Prometheus / VictoriaMetrics 数据源')
      return
    }
  }
  submitting.value = true
  try {
    if (form.id) {
      await updateMonitorProbeTask(form.id, buildPayload())
      ElMessage.success('更新成功')
    } else {
      await createMonitorProbeTask(buildPayload())
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    await loadAll()
  } finally {
    submitting.value = false
  }
}

const handleEnabledChange = async (row: ProbeTaskRow) => {
  if (!row.id) return
  try {
    await updateMonitorProbeTask(row.id, { ...row })
    ElMessage.success('状态已更新')
  } catch {
    row.enabled = !row.enabled
  }
}

const handleRun = async (row: ProbeTaskRow) => {
  if (!row.id) return
  row.running = true
  try {
    runSummary.value = await runMonitorProbeTask(row.id)
    resultDialogVisible.value = true
    await loadAll()
  } finally {
    row.running = false
  }
}

const handleDelete = async (row: MonitorProbeTask) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除拨测任务「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorProbeTask(row.id)
  ElMessage.success('删除成功')
  await loadTasks()
}

const getProtocolName = (protocol?: string) => {
  const map: Record<string, string> = { http: 'HTTP', icmp: 'ICMP', tcp: 'TCP', ssl: 'SSL' }
  return protocol ? map[protocol] || protocol.toUpperCase() : '-'
}

const getProtocolTag = (protocol?: string) => {
  const map: Record<string, string> = { http: 'primary', icmp: 'success', tcp: 'warning', ssl: 'danger' }
  return protocol ? map[protocol] || 'info' : 'info'
}

const getStatusName = (status?: string) => {
  const map: Record<string, string> = { normal: '正常', abnormal: '异常', unknown: '未执行' }
  return status ? map[status] || status : '未执行'
}

const getStatusTag = (status?: string) => {
  const map: Record<string, string> = { normal: 'success', abnormal: 'danger', unknown: 'info' }
  return status ? map[status] || 'info' : 'info'
}

const getDataSourceName = (id?: number) => dataSources.value.find(item => item.id === id)?.name || ''

const getEndpointPlaceholder = (protocol: ProbeProtocol) => {
  const map: Record<ProbeProtocol, string> = {
    http: 'https://example.com/health\nhttps://api.example.com/ready',
    icmp: 'github.com, 8.8.8.8',
    tcp: '192.168.1.1:80, 10.0.0.1:443',
    ssl: 'github.com, example.com:443'
  }
  return map[protocol]
}

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

onMounted(loadAll)
</script>

<style scoped>
.probe-tasks-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header,
.search-bar,
.stat-card,
.table-wrapper {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
}

.page-title-group,
.header-actions,
.search-inputs,
.action-buttons,
.source-cell {
  display: flex;
  align-items: center;
}

.page-title-group {
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 22px;
}

.page-title {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.header-actions,
.search-inputs,
.action-buttons,
.source-cell {
  gap: 10px;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
}

.search-inputs {
  flex: 1;
  flex-wrap: wrap;
}

.search-input {
  width: 280px;
}

.search-input.small {
  width: 150px;
}

.reset-btn {
  background: #fff;
  border-color: #d8dee9;
  color: #344054;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 92px;
  padding: 16px;
}

.stat-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  font-size: 21px;
}

.stat-icon-primary {
  background: #eef4ff;
  border: 1px solid #dbe8ff;
  color: #2563eb;
}

.stat-icon-success {
  background: #ecfdf3;
  border: 1px solid #bbf7d0;
  color: #16a34a;
}

.stat-icon-warning {
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #d97706;
}

.stat-icon-danger {
  background: #fff1f2;
  border: 1px solid #fecdd3;
  color: #e11d48;
}

.stat-label {
  color: #667085;
  font-size: 13px;
}

.stat-value {
  margin-top: 4px;
  color: #111827;
  font-size: 28px;
  font-weight: 750;
  line-height: 1;
}

.table-wrapper {
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.task-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.task-name-cell > div {
  min-width: 0;
}

.task-name-cell strong,
.task-name-cell em {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-name-cell strong {
  color: #111827;
  font-size: 13px;
  font-weight: 700;
}

.task-name-cell em {
  margin-top: 3px;
  color: #98a2b3;
  font-size: 12px;
  font-style: normal;
}

.protocol-dot {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 7px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
}

.protocol-dot.icmp {
  background: #ecfdf3;
  color: #16a34a;
}

.protocol-dot.tcp {
  background: #fffbeb;
  color: #d97706;
}

.protocol-dot.ssl {
  background: #fff1f2;
  color: #e11d48;
}

.source-cell {
  min-width: 0;
}

.source-cell span:last-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.time-text,
.last-result {
  white-space: nowrap;
}

.last-result {
  color: #344054;
  font-size: 12px;
}

.last-result.abnormal {
  color: #dc2626;
}

.muted-text {
  color: #98a2b3;
}

.action-buttons {
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 7px;
  color: #667085;
}

.action-run:hover {
  background: #ecfdf3;
  color: #16a34a;
}

.action-edit:hover {
  background: #eff6ff;
  color: #2563eb;
}

.action-delete:hover {
  background: #fff1f2;
  color: #e11d48;
}

.table-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
}

.table-pagination > span {
  color: #667085;
  font-size: 12px;
}

.drawer-section {
  padding: 16px 0;
  border-bottom: 1px solid #edf1f7;
}

.drawer-section:first-child {
  padding-top: 0;
}

.drawer-section-title {
  margin-bottom: 14px;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 14px;
}

.form-grid.three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-suffix {
  flex: 0 0 auto;
  color: #667085;
  font-size: 13px;
  white-space: nowrap;
}

.inline-number-field {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.inline-number-field :deep(.el-input-number) {
  flex: 1 1 auto;
  width: auto;
  min-width: 0;
}

.write-alert {
  margin-bottom: 14px;
}

.source-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.run-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  padding: 12px 14px;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  background: #ecfdf3;
  color: #166534;
}

.run-summary.abnormal {
  border-color: #fecdd3;
  background: #fff1f2;
  color: #991b1b;
}

@media (max-width: 1000px) {
  .page-header,
  .search-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .stats-cards,
  .form-grid,
  .form-grid.three {
    grid-template-columns: 1fr;
  }
}
</style>
