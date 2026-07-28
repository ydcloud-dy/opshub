<template>
  <div class="log-center-page log-library-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon"><el-icon><FolderOpened /></el-icon></div>
        <div>
          <h2>日志库</h2>
          <p>管理 OpsHub 自带日志中心使用的 ClickHouse 存储、表结构和保留周期</p>
        </div>
      </div>
      <el-button v-if="activeTab === 'storage'" type="primary" class="primary-action" @click="openCreate">
        <el-icon><Plus /></el-icon>
        新增存储
      </el-button>
    </div>

    <el-tabs v-model="activeTab" class="library-tabs">
      <el-tab-pane label="存储管理" name="storage">
    <section class="panel storage-panel">
      <div class="panel-head">
        <div>
          <h3>内置日志存储</h3>
          <p>连接测试通过后，OpsHub 自动创建原始日志表、分钟聚合表和 TTL</p>
        </div>
        <div class="storage-summary">
          <span><i class="summary-dot healthy"></i>{{ healthyCount }} 个健康</span>
          <span><i class="summary-dot"></i>{{ items.length }} 个存储</span>
        </div>
      </div>

      <el-alert
        v-if="!items.length"
        type="info"
        :closable="false"
        show-icon
        title="尚未配置内置日志存储"
        description="Docker Compose 默认使用 http://clickhouse:8123；本地后端使用 http://127.0.0.1:18123。"
        class="storage-empty-tip"
      />

      <el-table v-loading="loading" :data="items" empty-text="暂无 ClickHouse 存储配置">
        <el-table-column prop="name" label="存储名称" min-width="190">
          <template #default="{ row }">
            <div class="storage-name">
              <strong>{{ row.name }}</strong>
              <span v-if="row.isPrimary">主存储</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="endpoints" label="HTTP 地址" min-width="260" show-overflow-tooltip />
        <el-table-column prop="databaseName" label="数据库" width="150" />
        <el-table-column prop="queueMode" label="写入模式" width="110">
          <template #default>直接写入</template>
        </el-table-column>
        <el-table-column prop="defaultRetentionDays" label="默认保留" width="110">
          <template #default="{ row }">{{ row.defaultRetentionDays }} 天</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="statusType(row)">{{ statusText(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastTestAt" label="最近检测" width="170">
          <template #default="{ row }">{{ formatTime(row.lastTestAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="290" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :loading="actionId === row.id && action === 'test'" @click="testStorage(row)">测试</el-button>
            <el-button link type="primary" :loading="actionId === row.id && action === 'initialize'" @click="initializeStorage(row)">
              {{ row.initializedAt ? '更新表结构' : '初始化' }}
            </el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm title="只删除 OpsHub 中的连接配置，不会删除 ClickHouse 数据，确定继续？" @confirm="deleteStorage(row)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </section>
      </el-tab-pane>
      <el-tab-pane label="保留策略" name="retention" lazy><RetentionPolicyPanel /></el-tab-pane>
      <el-tab-pane label="容量预测" name="capacity" lazy><CapacityPanel :storages="items" /></el-tab-pane>
      <el-tab-pane label="访问控制" name="access" lazy><AccessPolicyPanel :storages="items" /></el-tab-pane>
    </el-tabs>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑 ClickHouse 存储' : '新增 ClickHouse 存储'" width="720px" destroy-on-close>
      <el-form :model="form" label-width="112px" class="storage-form">
        <div class="storage-form-grid">
          <el-form-item label="存储名称" required>
            <el-input v-model="form.name" placeholder="例如：生产日志存储" />
          </el-form-item>
          <el-form-item label="存储类型">
            <el-input model-value="ClickHouse" disabled />
          </el-form-item>
        </div>
        <el-form-item label="HTTP 地址" required>
          <el-input v-model="form.endpoints" placeholder="http://clickhouse:8123" />
          <div class="form-tip">支持 HTTP/HTTPS；多个地址可用逗号分隔，当前直写模式使用第一个地址。</div>
        </el-form-item>
        <div class="storage-form-grid">
          <el-form-item label="数据库">
            <el-input v-model="form.databaseName" placeholder="opshub_logs" />
          </el-form-item>
          <el-form-item label="保留天数">
            <el-input-number v-model="form.defaultRetentionDays" :min="1" :max="3650" controls-position="right" class="full" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="opshub" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" show-password :placeholder="form.passwordConfigured ? '留空保持原密码' : '请输入密码'" />
          </el-form-item>
          <el-form-item label="请求超时">
            <el-input-number v-model="form.timeout" :min="5" :max="1800" controls-position="right" class="full" />
            <div class="form-tip">日志大范围查询建议 300 秒以上；导出任务可继续后台执行。</div>
          </el-form-item>
          <el-form-item label="启用存储">
            <el-switch v-model="form.enabled" />
          </el-form-item>
        </div>
        <el-form-item label="TLS 校验">
          <el-switch v-model="form.skipTlsVerify" active-text="跳过证书校验" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="primary-action" :loading="saving" @click="saveStorage">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { FolderOpened, Plus } from '@element-plus/icons-vue'
import {
  createLogStorageCluster,
  deleteLogStorageCluster,
  getLogStorageClusters,
  initializeLogStorageCluster,
  testLogStorageCluster,
  updateLogStorageCluster,
  type LogStorageCluster,
} from '@/api/logcenter'
import RetentionPolicyPanel from './components/RetentionPolicyPanel.vue'
import CapacityPanel from './components/CapacityPanel.vue'
import AccessPolicyPanel from './components/AccessPolicyPanel.vue'

const route = useRoute()
const requestedTab = String(route.query.tab || 'storage')
const activeTab = ref(['storage', 'retention', 'capacity', 'access'].includes(requestedTab) ? requestedTab : 'storage')
const items = ref<LogStorageCluster[]>([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const actionId = ref<number>()
const action = ref<'test' | 'initialize' | ''>('')
const form = ref<Partial<LogStorageCluster>>({})
const healthyCount = computed(() => items.value.filter(item => item.enabled && item.status === 'healthy').length)

const defaultForm = (): Partial<LogStorageCluster> => ({
  name: 'OpsHub 内置日志存储', storageType: 'clickhouse', endpoints: '', databaseName: 'opshub_logs',
  username: 'opshub', password: '', timeout: 300, queueMode: 'direct', defaultRetentionDays: 30,
  skipTlsVerify: false, enabled: true,
})

const loadData = async () => {
  loading.value = true
  try { items.value = await getLogStorageClusters() as any } finally { loading.value = false }
}
const openCreate = () => { form.value = defaultForm(); dialogVisible.value = true }
const openEdit = (row: LogStorageCluster) => { form.value = { ...row, password: '' }; dialogVisible.value = true }

const saveStorage = async () => {
  if (!form.value.name?.trim() || !form.value.endpoints?.trim()) {
    ElMessage.warning('请填写存储名称和 ClickHouse HTTP 地址')
    return
  }
  saving.value = true
  try {
    if (form.value.id) await updateLogStorageCluster(form.value.id, form.value)
    else await createLogStorageCluster(form.value)
    ElMessage.success('日志存储配置已保存')
    dialogVisible.value = false
    await loadData()
  } finally { saving.value = false }
}

const testStorage = async (row: LogStorageCluster) => {
  if (!row.id) return
  actionId.value = row.id; action.value = 'test'
  try { await testLogStorageCluster(row.id); ElMessage.success('ClickHouse 连接正常'); await loadData() }
  finally { actionId.value = undefined; action.value = '' }
}
const initializeStorage = async (row: LogStorageCluster) => {
  if (!row.id) return
  actionId.value = row.id; action.value = 'initialize'
  try { await initializeLogStorageCluster(row.id); ElMessage.success('ClickHouse 日志表初始化完成'); await loadData() }
  finally { actionId.value = undefined; action.value = '' }
}
const deleteStorage = async (row: LogStorageCluster) => {
  if (!row.id) return
  await deleteLogStorageCluster(row.id)
  ElMessage.success('存储配置已删除')
  await loadData()
}
const statusText = (row: LogStorageCluster) => {
  if (!row.enabled || row.status === 'disabled') return '已停用'
  if (row.status === 'healthy' && row.initializedAt) return '运行正常'
  if (row.status === 'healthy') return '连接正常'
  if (row.status === 'error') return '连接异常'
  return '待检测'
}
const statusType = (row: LogStorageCluster) => {
  if (!row.enabled || row.status === 'disabled') return 'info'
  if (row.status === 'healthy') return 'success'
  if (row.status === 'error') return 'danger'
  return 'warning'
}
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
onMounted(loadData)
</script>

<style scoped>
.storage-panel { padding: 20px; }
.library-tabs :deep(.el-tabs__header) { margin: 0 0 16px; padding: 0 4px; }
.library-tabs :deep(.el-tabs__item) { height: 42px; color: #667085; font-size: 14px; }
.library-tabs :deep(.el-tabs__item.is-active) { color: #111827; font-weight: 650; }
.library-tabs :deep(.el-tabs__active-bar) { height: 2px; background: #111827; }
.panel-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
.panel-head h3 { margin: 0; color: #111827; font-size: 15px; }
.panel-head p { margin: 6px 0 0; color: #667085; font-size: 13px; }
.storage-summary { display: flex; align-items: center; gap: 16px; color: #667085; font-size: 12px; }
.storage-summary span { display: inline-flex; align-items: center; gap: 6px; }
.summary-dot { width: 7px; height: 7px; border-radius: 50%; background: #94a3b8; }
.summary-dot.healthy { background: #22c55e; }
.storage-empty-tip { margin-bottom: 16px; }
.storage-name { display: flex; align-items: center; gap: 8px; }
.storage-name strong { color: #111827; }
.storage-name span { padding: 2px 6px; border-radius: 4px; background: #eef2ff; color: #4338ca; font-size: 11px; }
.storage-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 18px; }
.form-tip { margin-top: 6px; color: #98a2b3; font-size: 12px; line-height: 1.5; }
.full { width: 100%; }
@media (max-width: 760px) { .storage-form-grid { grid-template-columns: 1fr; } }
</style>
