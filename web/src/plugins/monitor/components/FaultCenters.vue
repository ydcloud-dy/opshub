<template>
  <div class="fault-centers-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><FirstAidKit /></el-icon>
        </div>
        <div>
          <h2 class="page-title">故障中心</h2>
          <p class="page-subtitle">统一承接规则事件、通知路由、恢复等待和告警升级策略</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          新增故障中心
        </el-button>
        <el-button @click="loadAll">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="search-bar">
      <el-input v-model="keyword" placeholder="搜索故障中心名称或描述..." clearable class="search-input" @keyup.enter="loadCenters">
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button @click="handleReset">重置</el-button>
    </div>

    <el-empty v-if="!loading && centers.length === 0" description="暂无故障中心" />

    <div v-else class="center-grid" v-loading="loading">
      <div v-for="item in centers" :key="item.id" class="center-card" @click="openDetail(item)">
        <div class="card-top">
          <div class="center-title">
            <span class="status-dot" :class="{ active: (item.currentAlertNumber || 0) > 0 }"></span>
            <span>{{ item.name }}</span>
          </div>
          <span class="detail-hint">进入详情</span>
        </div>

        <p class="center-desc">{{ item.description || '未填写描述' }}</p>

        <div class="metric-row">
          <div class="metric-item">
            <div class="metric-value warning">{{ item.currentPreAlertNumber || 0 }}</div>
            <div class="metric-label">预告警</div>
          </div>
          <div class="metric-item">
            <div class="metric-value danger">{{ item.currentAlertNumber || 0 }}</div>
            <div class="metric-label">待处理</div>
          </div>
          <div class="metric-item">
            <div class="metric-value success">{{ item.currentRecoverNumber || 0 }}</div>
            <div class="metric-label">待恢复</div>
          </div>
        </div>

        <div class="card-footer">
          <el-tag effect="light" size="small">{{ item.aggregationType || 'Rule' }}</el-tag>
          <div class="card-footer-actions">
            <span>{{ formatDateTime(item.createdAt) }}</span>
            <el-button link class="card-action danger" @click.stop="handleDelete(item)">
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="880px" class="center-dialog" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="center-form">
        <section class="dialog-section">
          <div class="section-heading">
            <strong>基础信息</strong>
            <span>定义故障中心承接的业务范围和默认通知对象</span>
          </div>
          <div class="field-grid">
            <el-form-item label="名称" prop="name">
              <el-input v-model="form.name" placeholder="如：核心业务故障中心" />
            </el-form-item>
            <el-form-item label="默认通知对象">
              <el-select v-model="form.noticeObjectIds" multiple clearable filterable placeholder="规则未指定通知对象时使用这里的对象">
                <el-option
                  v-for="object in enabledNoticeObjects"
                  :key="object.id"
                  :label="object.name"
                  :value="object.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="描述" class="field-span-2">
              <el-input v-model="form.description" type="textarea" :rows="3" resize="none" placeholder="说明该故障中心承接的业务或团队" />
            </el-form-item>
          </div>
        </section>

        <section class="dialog-section">
          <div class="section-heading">
            <strong>通知策略</strong>
            <span>控制恢复通知、恢复等待和不同等级的重复通知间隔</span>
          </div>
          <div class="policy-grid">
            <div class="policy-card switch-card">
              <div>
                <span class="policy-title">恢复通知</span>
                <p>事件恢复后向通知对象发送恢复消息</p>
              </div>
              <el-switch v-model="form.recoverNotify" />
            </div>
            <div class="policy-card">
              <span class="policy-title">恢复等待</span>
              <div class="number-with-unit">
                <el-input-number v-model="form.recoverWaitSeconds" :min="1" :max="86400" controls-position="right" />
                <span>秒</span>
              </div>
            </div>
            <div class="policy-card">
              <span class="policy-title">P0 重复通知</span>
              <div class="number-with-unit">
                <el-input-number v-model="form.repeat.p0" :min="1" :max="10080" controls-position="right" />
                <span>分钟</span>
              </div>
            </div>
            <div class="policy-card">
              <span class="policy-title">P1 重复通知</span>
              <div class="number-with-unit">
                <el-input-number v-model="form.repeat.p1" :min="1" :max="10080" controls-position="right" />
                <span>分钟</span>
              </div>
            </div>
            <div class="policy-card">
              <span class="policy-title">P2 重复通知</span>
              <div class="number-with-unit">
                <el-input-number v-model="form.repeat.p2" :min="1" :max="10080" controls-position="right" />
                <span>分钟</span>
              </div>
            </div>
          </div>
        </section>

        <section class="dialog-section">
          <div class="section-heading">
            <strong>告警升级</strong>
            <span>开启后，未及时处理的事件会按升级策略再次通知</span>
          </div>
          <div class="upgrade-panel">
            <div class="switch-card upgrade-switch">
              <div>
                <span class="policy-title">启用告警升级</span>
                <p>只对下方选择的等级生效</p>
              </div>
              <el-switch v-model="form.upgradeEnabled" />
            </div>
            <div class="severity-picker" :class="{ disabled: !form.upgradeEnabled }">
              <span>可升级等级</span>
              <el-checkbox-group v-model="form.upgradableSeverities">
                <el-checkbox-button label="p0">P0</el-checkbox-button>
                <el-checkbox-button label="p1">P1</el-checkbox-button>
                <el-checkbox-button label="p2">P2</el-checkbox-button>
              </el-checkbox-group>
            </div>
          </div>
        </section>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Delete, FirstAidKit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import {
  createMonitorFaultCenter,
  deleteMonitorFaultCenter,
  getMonitorFaultCenters,
  getMonitorNoticeObjects,
  type MonitorFaultCenter,
  type MonitorNoticeObject
} from '@/api/monitor-datasource'

interface CenterForm {
  name: string
  description: string
  noticeObjectIds: number[]
  recoverNotify: boolean
  recoverWaitSeconds: number
  repeat: {
    p0: number
    p1: number
    p2: number
  }
  upgradeEnabled: boolean
  upgradableSeverities: string[]
}

const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('新增故障中心')
const keyword = ref('')
const formRef = ref<FormInstance>()
const centers = ref<MonitorFaultCenter[]>([])
const noticeObjects = ref<MonitorNoticeObject[]>([])

const form = reactive<CenterForm>({
  name: '',
  description: '',
  noticeObjectIds: [],
  recoverNotify: true,
  recoverWaitSeconds: 30,
  repeat: {
    p0: 30,
    p1: 60,
    p2: 120
  },
  upgradeEnabled: false,
  upgradableSeverities: ['p0', 'p1']
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入故障中心名称', trigger: 'blur' }]
}

const enabledNoticeObjects = computed(() => noticeObjects.value.filter(item => item.enabled !== false && item.id))

const loadCenters = async () => {
  loading.value = true
  try {
    centers.value = await getMonitorFaultCenters({ keyword: keyword.value.trim() }) || []
  } finally {
    loading.value = false
  }
}

const loadNoticeObjects = async () => {
  noticeObjects.value = await getMonitorNoticeObjects() || []
}

const loadAll = async () => {
  await Promise.all([loadCenters(), loadNoticeObjects()])
}

const handleAdd = () => {
  resetForm()
  dialogTitle.value = '新增故障中心'
  dialogVisible.value = true
}

const handleReset = () => {
  keyword.value = ''
  loadCenters()
}

const handleDelete = async (row: MonitorFaultCenter) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除故障中心「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorFaultCenter(row.id)
  ElMessage.success('删除成功')
  await loadCenters()
}

const openDetail = (row: MonitorFaultCenter) => {
  if (row.id) router.push(`/monitor/fault-centers/${row.id}`)
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload = buildPayload()
    await createMonitorFaultCenter(payload)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    await loadCenters()
  } finally {
    submitting.value = false
  }
}

const buildPayload = (): MonitorFaultCenter => ({
  name: form.name,
  description: form.description,
  noticeObjectIds: JSON.stringify(form.noticeObjectIds),
  noticeChannelIds: '[]',
  noticeRoutes: '[]',
  repeatNoticeInterval: JSON.stringify({
    p0: form.repeat.p0,
    p1: form.repeat.p1,
    p2: form.repeat.p2
  }),
  recoverNotify: form.recoverNotify,
  aggregationType: 'Rule',
  recoverWaitSeconds: form.recoverWaitSeconds,
  upgradeEnabled: form.upgradeEnabled,
  upgradableSeverities: JSON.stringify(form.upgradableSeverities),
  upgradeStrategy: JSON.stringify({ enabled: form.upgradeEnabled, timeout: 30, repeatInterval: 60, noticeObjectIds: form.noticeObjectIds })
})

const resetForm = () => {
  form.name = ''
  form.description = ''
  form.noticeObjectIds = []
  form.recoverNotify = true
  form.recoverWaitSeconds = 30
  form.repeat = { p0: 30, p1: 60, p2: 120 }
  form.upgradeEnabled = false
  form.upgradableSeverities = ['p0', 'p1']
  formRef.value?.clearValidate()
}

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

onMounted(loadAll)
</script>

<style scoped>
.fault-centers-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header,
.search-bar,
.center-card {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
}

.page-title-group,
.header-actions,
.search-bar,
.card-top,
.center-title,
.card-footer {
  display: flex;
  align-items: center;
}

.page-title-group {
  gap: 14px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
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
}

.page-subtitle {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.header-actions {
  gap: 10px;
}

.search-bar {
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
}

.search-input {
  width: 360px;
}

.center-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(240px, 1fr));
  gap: 12px;
  min-height: 180px;
}

.center-card {
  min-height: 176px;
  padding: 16px;
  cursor: pointer;
  transition: border-color .16s ease, box-shadow .16s ease;
}

.center-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 8px 24px rgba(15, 23, 42, .06);
}

.card-top,
.card-footer {
  justify-content: space-between;
  gap: 10px;
}

.center-title {
  min-width: 0;
  gap: 8px;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.center-title span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-hint {
  flex: 0 0 auto;
  padding: 3px 8px;
  border: 1px solid #e0e7ff;
  border-radius: 999px;
  background: #f5f7ff;
  color: #3151a3;
  font-size: 12px;
  font-weight: 650;
}

.status-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 50%;
  background: #16a34a;
}

.status-dot.active {
  background: #e11d48;
}

.center-desc {
  height: 38px;
  margin: 10px 0 14px;
  overflow: hidden;
  color: #667085;
  font-size: 13px;
  line-height: 19px;
}

.metric-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.metric-item {
  padding: 10px 8px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
  text-align: center;
}

.metric-value {
  font-size: 22px;
  font-weight: 780;
  line-height: 1;
}

.metric-value.warning {
  color: #d97706;
}

.metric-value.danger {
  color: #e11d48;
}

.metric-value.success {
  color: #16a34a;
}

.metric-label {
  margin-top: 5px;
  color: #667085;
  font-size: 12px;
}

.card-footer {
  flex-wrap: nowrap;
  margin-top: 14px;
  color: #98a2b3;
  font-size: 12px;
}

.card-footer-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

.card-footer-actions > span {
  flex: 0 0 auto;
  white-space: nowrap;
}

.card-action {
  min-height: 24px;
  padding: 0 4px;
  color: #667085;
  font-size: 12px;
}

.card-action:hover {
  color: #2563eb;
}

.card-action.danger:hover {
  color: #e11d48;
}

:deep(.center-dialog .el-dialog__header) {
  border-bottom: 1px solid #edf1f7;
  background: #fbfcfe;
  padding: 22px 28px 18px;
  margin-right: 0;
}

:deep(.center-dialog .el-dialog__title) {
  color: #111827;
  font-size: 20px;
  font-weight: 750;
}

:deep(.center-dialog .el-dialog__headerbtn) {
  top: 18px;
  right: 22px;
}

:deep(.center-dialog .el-dialog__body) {
  max-height: min(680px, calc(100vh - 210px));
  overflow-y: auto;
  padding: 18px 28px 20px;
}

:deep(.center-dialog .el-dialog__footer) {
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
  padding: 16px 28px 18px;
}

.center-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dialog-section {
  padding: 16px;
  border: 1px solid #e6ebf2;
  border-radius: 8px;
  background: #fff;
}

.section-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}

.section-heading strong {
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.section-heading span {
  color: #8a94a6;
  font-size: 12px;
  line-height: 18px;
  text-align: right;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.field-span-2 {
  grid-column: 1 / -1;
}

:deep(.center-form .el-form-item) {
  margin-bottom: 0;
}

:deep(.center-form .el-form-item__label) {
  padding-bottom: 7px;
  color: #3f4654;
  font-size: 13px;
  font-weight: 650;
  line-height: 18px;
}

:deep(.center-form .el-select) {
  width: 100%;
}

.policy-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.policy-card,
.upgrade-panel {
  border: 1px solid #e6ebf2;
  border-radius: 8px;
  background: #fbfcfe;
}

.policy-card {
  min-height: 92px;
  padding: 14px;
}

.switch-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.policy-title {
  display: block;
  color: #111827;
  font-size: 14px;
  font-weight: 750;
  line-height: 20px;
}

.policy-card p,
.switch-card p {
  margin: 5px 0 0;
  color: #8a94a6;
  font-size: 12px;
  line-height: 18px;
}

.number-with-unit {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
}

.number-with-unit > span {
  flex: 0 0 34px;
  color: #667085;
  font-size: 13px;
  white-space: nowrap;
}

:deep(.number-with-unit .el-input-number) {
  width: 150px;
}

:deep(.number-with-unit .el-input__wrapper) {
  box-shadow: 0 0 0 1px #dbe2ec inset;
}

.upgrade-panel {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(260px, 1fr);
  gap: 12px;
  padding: 14px;
}

.upgrade-switch {
  padding: 0;
}

.severity-picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 56px;
  padding: 0 2px 0 14px;
  border-left: 1px solid #e6ebf2;
}

.severity-picker.disabled {
  opacity: .55;
}

.severity-picker > span {
  color: #3f4654;
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
}

:deep(.severity-picker .el-checkbox-button__inner) {
  min-width: 48px;
  padding: 9px 14px;
  border-color: #dbe2ec;
  color: #3f4654;
  font-weight: 700;
}

:deep(.severity-picker .el-checkbox-button.is-checked .el-checkbox-button__inner) {
  border-color: #111827;
  background: #111827;
  color: #fff;
  box-shadow: none;
}

@media (max-width: 1200px) {
  .center-grid {
    grid-template-columns: repeat(2, minmax(240px, 1fr));
  }
}

@media (max-width: 720px) {
  .page-header,
  .search-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .center-grid,
  .field-grid,
  .policy-grid,
  .upgrade-panel {
    grid-template-columns: 1fr;
  }

  .search-input {
    width: 100%;
  }

  .section-heading,
  .severity-picker {
    align-items: flex-start;
    flex-direction: column;
  }

  .section-heading span {
    text-align: left;
  }

  .severity-picker {
    border-left: 0;
    border-top: 1px solid #e6ebf2;
    padding: 12px 0 0;
  }
}
</style>
