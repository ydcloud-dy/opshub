<template>
  <div class="instance-page">
    <section class="section-head panel">
      <div>
        <h3>采集实例</h3>
        <p>实例按采集策略归组，展开策略即可查看对应主机 Agent 或 Kubernetes 节点采集器</p>
      </div>
      <div class="section-actions">
        <el-input v-model="keyword" clearable placeholder="搜索策略、实例或目标">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="modeFilter" clearable placeholder="全部类型">
          <el-option label="主机 Agent" value="host" />
          <el-option label="Kubernetes 采集器" value="kubernetes" />
        </el-select>
        <el-select v-model="statusFilter" clearable placeholder="全部状态">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="空闲" value="idle" />
          <el-option label="已卸载历史" value="retired" />
        </el-select>
        <el-button :loading="loading" @click="load()"><el-icon><Refresh /></el-icon>刷新</el-button>
      </div>
    </section>

    <section class="instance-summary panel">
      <div><span>主机实例</span><strong>{{ summary.host }}</strong><small>Host Agent</small></div>
      <div><span>Kubernetes 实例</span><strong>{{ summary.kubernetes }}</strong><small>节点 DaemonSet</small></div>
      <div><span>在线运行</span><strong class="success">{{ summary.online }}</strong><small>最近心跳正常</small></div>
      <div><span>离线或异常</span><strong :class="{ danger: summary.abnormal > 0 }">{{ summary.abnormal }}</strong><small>需检查心跳或错误</small></div>
    </section>

    <section class="panel groups-panel" v-loading="loading && !items.length">
      <div class="groups-toolbar">
        <div>
          <strong>策略实例组</strong>
          <span>{{ filteredGroups.length }} 个分组 · {{ visibleInstanceTotal }} 个可见实例</span>
        </div>
        <span>一个实例命中多个策略时会出现在多个策略组中</span>
      </div>

      <el-empty v-if="!filteredGroups.length && !loading" description="没有匹配的采集策略或实例" :image-size="72" />
      <el-collapse v-else v-model="expandedGroups" class="policy-groups">
        <el-collapse-item v-for="group in filteredGroups" :key="group.key" :name="group.key">
          <template #title>
            <div class="group-title">
              <span class="group-source-icon" :class="group.sourceMode">
                <el-icon v-if="group.sourceMode === 'kubernetes'"><Grid /></el-icon>
                <el-icon v-else><Monitor /></el-icon>
              </span>
              <div class="group-identity">
                <div>
                  <strong>{{ group.name }}</strong>
                  <el-tag size="small" :type="group.sourceMode === 'kubernetes' ? 'warning' : 'info'">
                    {{ group.sourceMode === 'kubernetes' ? 'Kubernetes' : '主机' }}
                  </el-tag>
                  <el-tag v-if="group.policy" size="small" :type="policyStatusType(group.policy.status)">{{ policyStatusText(group.policy.status) }}</el-tag>
                </div>
                <small>{{ groupDescription(group) }}</small>
              </div>
              <div class="group-metrics">
                <span><small>目标</small><strong>{{ group.targetCount }}</strong></span>
                <span><small>在线</small><strong class="success">{{ group.online }}</strong><em>/ {{ group.total }}</em></span>
                <span v-if="group.policy"><small>已应用</small><strong>{{ group.applied }}</strong><em>/ {{ group.total }}</em></span>
                <span v-if="group.pending"><small>待确认</small><strong class="warning">{{ group.pending }}</strong></span>
                <span v-if="group.failed"><small>失败</small><strong class="danger">{{ group.failed }}</strong></span>
              </div>
            </div>
          </template>

          <div class="group-content">
            <div class="group-context">
              <span>{{ targetSummary(group) }}</span>
              <span v-if="group.policy">策略版本 v{{ group.policy.version || 0 }}</span>
              <span v-if="group.policy?.updatedAt">更新于 {{ formatTime(group.policy.updatedAt) }}</span>
            </div>
            <el-table :data="group.instances" size="small" empty-text="该策略暂未匹配到采集实例">
              <el-table-column label="实例" min-width="250">
                <template #default="{ row }">
                  <div class="instance-name">
                    <span class="status-dot" :class="row.status"></span>
                    <div>
                      <strong>{{ instanceTitle(row) }}</strong>
                      <small :title="row.instance.instanceId">{{ instanceSubtitle(row) }}</small>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="145">
                <template #default="{ row }">
                  <div class="status-tags">
                    <el-tag size="small" :type="row.status === 'online' ? 'success' : 'danger'">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag>
                    <el-tag v-if="row.lifecycleStatus !== 'active'" size="small" :type="lifecycleType(row.lifecycleStatus)">{{ lifecycleText(row.lifecycleStatus) }}</el-tag>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="目标资产" min-width="190">
                <template #default="{ row }">
                  <div class="target-cell"><strong>{{ targetTitle(row) }}</strong><small>{{ targetSubtext(row) }}</small></div>
                </template>
              </el-table-column>
              <el-table-column label="版本" width="125">
                <template #default="{ row }"><span>{{ row.instance.version || '-' }}</span><small class="muted">配置 v{{ row.instance.configVersion || 0 }}</small></template>
              </el-table-column>
              <el-table-column label="策略应用" min-width="150">
                <template #default="{ row }">
                  <template v-if="group.policy">
                    <el-tag v-if="policyAssignment(row, group.policy.id)" size="small" :type="assignmentType(policyAssignment(row, group.policy.id)?.applyStatus || '')">
                      v{{ policyAssignment(row, group.policy.id)?.policyVersion || 0 }} · {{ assignmentText(policyAssignment(row, group.policy.id)?.applyStatus || '') }}
                    </el-tag>
                    <span v-else class="muted">等待下发</span>
                  </template>
                  <span v-else class="muted">未关联策略</span>
                </template>
              </el-table-column>
              <el-table-column label="输入 / 输出" width="145">
                <template #default="{ row }"><strong>{{ number(row.instance.inputEps) }}</strong> / {{ number(row.instance.outputEps) }} EPS</template>
              </el-table-column>
              <el-table-column label="WAL" width="105"><template #default="{ row }">{{ bytes(row.instance.walBytes) }}</template></el-table-column>
              <el-table-column label="最后心跳" width="172"><template #default="{ row }">{{ formatTime(row.instance.lastHeartbeatAt) }}</template></el-table-column>
              <el-table-column label="异常" min-width="190" show-overflow-tooltip>
                <template #default="{ row }"><span :class="{ error: row.instance.lastError }">{{ row.instance.lastError || lifecycleHint(row) }}</span></template>
              </el-table-column>
              <el-table-column v-if="isAdmin" label="操作" fixed="right" width="132">
                <template #default="{ row }">
                  <el-button link type="primary" :disabled="!canRestart(row)" @click="restart(row)">重载</el-button>
                  <el-button link type="danger" :disabled="!canCleanup(row)" @click="cleanup(row)">清理</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-collapse-item>
      </el-collapse>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Grid, Monitor, Refresh, Search } from '@element-plus/icons-vue'
import {
  deleteLogCollectorInstance,
  getLogCollectionPolicies,
  getLogCollectorInstances,
  restartLogCollectorInstance,
  type LogCollectionPolicy,
  type LogCollectorAssignment,
  type LogCollectorInstanceView,
} from '@/api/logcenter'
import { useUserStore } from '@/stores/user'

type SourceMode = 'host' | 'kubernetes'
type PolicyGroup = {
  key: string
  name: string
  sourceMode: SourceMode
  policy?: LogCollectionPolicy
  instances: LogCollectorInstanceView[]
  targetCount: number
  total: number
  online: number
  applied: number
  pending: number
  failed: number
}

const props = defineProps<{ presetPolicyId?: number }>()
const userStore = useUserStore()
const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))
const loading = ref(false)
const items = ref<LogCollectorInstanceView[]>([])
const policies = ref<LogCollectionPolicy[]>([])
const keyword = ref('')
const modeFilter = ref('')
const statusFilter = ref('')
const expandedGroups = ref<string[]>([])
let timer: ReturnType<typeof setInterval> | undefined
let expansionInitialized = false

const normalizeInstance = (row: LogCollectorInstanceView): LogCollectorInstanceView => ({
  ...row,
  status: row.status || row.runtimeStatus || row.instance.status || 'offline',
  lifecycleStatus: row.lifecycleStatus || 'active',
  assignments: Array.isArray(row.assignments) ? row.assignments : [],
})
const instanceSourceMode = (row: LogCollectorInstanceView): SourceMode => isKubernetes(row) ? 'kubernetes' : 'host'
const hostNames = computed(() => {
  const result = new Map<number, { name: string; ip: string }>()
  policies.value.forEach(policy => (policy.targetHosts || []).forEach(host => result.set(Number(host.id), { name: host.name, ip: host.ip })))
  return result
})
const clusterNames = computed(() => {
  const result = new Map<number, { name: string; alias?: string }>()
  policies.value.forEach(policy => (policy.targetClusters || []).forEach(cluster => result.set(Number(cluster.id), { name: cluster.name, alias: cluster.alias })))
  return result
})
const policyIds = computed(() => new Set(policies.value.map(policy => Number(policy.id))))
const baseGroups = computed<PolicyGroup[]>(() => {
  const groups: PolicyGroup[] = policies.value.map(policy => {
    const sourceMode: SourceMode = policy.payload.sourceMode === 'kubernetes' ? 'kubernetes' : 'host'
    const instances = items.value.filter(row => row.assignments.some(item => Number(item.policyId) === Number(policy.id)))
    return {
      key: `policy-${policy.id}`,
      name: policy.payload.name || `策略 #${policy.id}`,
      sourceMode,
      policy,
      instances,
      targetCount: Number(policy.targetExpected || policy.targetCount || 0),
      total: Number(policy.instanceTotal || instances.length),
      online: Number(policy.instanceOnline || instances.filter(row => row.status === 'online').length),
      applied: Number(policy.instanceApplied || 0),
      pending: Number(policy.instancePending || 0),
      failed: Number(policy.errorInstances || 0),
    }
  })
  const orphaned = items.value.filter(row => !row.assignments.some(item => policyIds.value.has(Number(item.policyId))))
  for (const sourceMode of ['host', 'kubernetes'] as SourceMode[]) {
    const instances = orphaned.filter(row => instanceSourceMode(row) === sourceMode)
    if (!instances.length) continue
    groups.push({
      key: `unassigned-${sourceMode}`,
      name: sourceMode === 'kubernetes' ? '未关联策略的 Kubernetes 实例' : '未关联策略的主机实例',
      sourceMode,
      instances,
      targetCount: instances.length,
      total: instances.length,
      online: instances.filter(row => row.status === 'online').length,
      applied: 0,
      pending: 0,
      failed: instances.filter(row => Boolean(row.instance.lastError)).length,
    })
  }
  return groups
})
const matchesStatus = (row: LogCollectorInstanceView) => {
  if (statusFilter.value === 'retired') return row.lifecycleStatus === 'retired'
  if (statusFilter.value === 'idle') return row.lifecycleStatus === 'idle'
  if (statusFilter.value && row.status !== statusFilter.value) return false
  return true
}
const instanceSearchText = (row: LogCollectorInstanceView) => [
  row.instance.instanceId, row.instance.hostname, row.instance.nodeName, row.instance.podName,
  row.instance.namespace, row.instance.agentId, targetTitle(row), targetSubtext(row), row.instance.lastError,
].filter(Boolean).join(' ').toLowerCase()
const filteredGroups = computed<PolicyGroup[]>(() => {
  const query = keyword.value.trim().toLowerCase()
  return baseGroups.value.flatMap(group => {
    if (modeFilter.value && group.sourceMode !== modeFilter.value) return []
    const groupMatches = !query || [group.name, group.policy?.payload.environment, group.policy?.payload.service]
      .filter(Boolean).join(' ').toLowerCase().includes(query)
    const instances = group.instances.filter(row => matchesStatus(row) && (groupMatches || instanceSearchText(row).includes(query)))
    const keepEmptyPolicy = Boolean(group.policy && !query && !statusFilter.value)
    if (!instances.length && !keepEmptyPolicy) return []
    return [{ ...group, instances }]
  })
})
const visibleInstanceTotal = computed(() => filteredGroups.value.reduce((total, group) => total + group.instances.length, 0))
const summary = computed(() => ({
  host: items.value.filter(row => !isKubernetes(row)).length,
  kubernetes: items.value.filter(isKubernetes).length,
  online: items.value.filter(row => row.status === 'online').length,
  abnormal: items.value.filter(row => row.status !== 'online' || Boolean(row.instance.lastError)).length,
}))

const openPresetGroup = (force = false) => {
  if (expansionInitialized && !force) return
  const key = props.presetPolicyId ? `policy-${props.presetPolicyId}` : ''
  if (key && baseGroups.value.some(group => group.key === key)) expandedGroups.value = [key]
  else if (!expandedGroups.value.length && baseGroups.value.length) expandedGroups.value = [baseGroups.value[0].key]
  expansionInitialized = true
}
const load = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const [instanceRows, policyRows] = await Promise.all([getLogCollectorInstances(), getLogCollectionPolicies()]) as any[]
    items.value = (Array.isArray(instanceRows) ? instanceRows : []).map(normalizeInstance)
    policies.value = (Array.isArray(policyRows) ? policyRows : []).map((policy: LogCollectionPolicy) => ({
      ...policy,
      payload: policy.payload || ({} as LogCollectionPolicy['payload']),
      targetHosts: Array.isArray(policy.targetHosts) ? policy.targetHosts : [],
      targetClusters: Array.isArray(policy.targetClusters) ? policy.targetClusters : [],
    }))
    openPresetGroup()
  } finally {
    if (!silent) loading.value = false
  }
}

const restart = async (row: LogCollectorInstanceView) => {
  await ElMessageBox.confirm(`确认让 ${instanceTitle(row)} 重新拉取并应用配置？`, '重载采集实例', { type: 'warning' })
  await restartLogCollectorInstance(row.instance.instanceId)
  ElMessage.success('已发送重载请求')
  await load(true)
}
const cleanup = async (row: LogCollectorInstanceView) => {
  await ElMessageBox.confirm(
    `确认清理 ${instanceTitle(row)} 的历史实例记录？只会删除离线且没有活动策略的实例，不会删除日志数据。`,
    '清理采集实例',
    { type: 'warning', confirmButtonText: '确认清理', cancelButtonText: '取消' },
  )
  await deleteLogCollectorInstance(row.instance.instanceId)
  ElMessage.success('历史实例已清理')
  await load(true)
}

const isKubernetes = (row: LogCollectorInstanceView) => row.instance.mode === 'kubernetes-node' || Boolean(row.instance.clusterId)
const canRestart = (row: LogCollectorInstanceView) => row.status === 'online' && row.lifecycleStatus !== 'retired'
const canCleanup = (row: LogCollectorInstanceView) => row.status !== 'online' && !row.assignments.some(item => item.desiredState === 'active')
const instanceTitle = (row: LogCollectorInstanceView) => {
  if (isKubernetes(row)) return row.instance.nodeName || row.instance.hostname || row.instance.instanceId
  return hostNames.value.get(Number(row.instance.hostId))?.name || row.instance.hostname || row.instance.instanceId
}
const instanceSubtitle = (row: LogCollectorInstanceView) => {
  if (!isKubernetes(row)) return row.instance.instanceId
  const pod = [row.instance.namespace, row.instance.podName].filter(Boolean).join('/')
  return pod ? `${pod} · ${row.instance.instanceId}` : row.instance.instanceId
}
const targetTitle = (row: LogCollectorInstanceView) => {
  if (isKubernetes(row)) {
    const cluster = clusterNames.value.get(Number(row.instance.clusterId))
    return cluster?.alias || cluster?.name || `集群 #${row.instance.clusterId || '-'}`
  }
  return hostNames.value.get(Number(row.instance.hostId))?.name || row.instance.hostname || `主机 #${row.instance.hostId || '-'}`
}
const targetSubtext = (row: LogCollectorInstanceView) => {
  if (isKubernetes(row)) return row.instance.nodeName ? `节点 ${row.instance.nodeName}` : `集群 ID ${row.instance.clusterId || '-'}`
  return hostNames.value.get(Number(row.instance.hostId))?.ip || row.instance.agentId || '未绑定 Agent'
}
const policyAssignment = (row: LogCollectorInstanceView, policyId: number): LogCollectorAssignment | undefined => row.assignments.find(item => Number(item.policyId) === Number(policyId))
const groupDescription = (group: PolicyGroup) => {
  if (!group.policy) return '实例存在，但尚未匹配当前采集策略'
  return [group.policy.payload.environment || '未配置环境', group.policy.payload.service || '未配置服务', `v${group.policy.version || 0}`].join(' · ')
}
const targetSummary = (group: PolicyGroup) => {
  if (!group.policy) return '请创建或发布采集策略以纳管这些实例'
  if (group.sourceMode === 'kubernetes') {
    const names = (group.policy.targetClusters || []).map(item => item.alias || item.name).filter(Boolean)
    return names.length ? `目标集群：${names.join('、')}` : '尚未选择目标集群'
  }
  const names = (group.policy.targetHosts || []).map(item => item.name).filter(Boolean)
  return names.length ? `目标主机：${names.slice(0, 6).join('、')}${names.length > 6 ? ` 等 ${names.length} 台` : ''}` : '尚未选择目标主机'
}
const lifecycleText = (status?: string) => ({ idle: '空闲', retired: '已卸载历史', active: '运行中' }[status || 'active'] || status)
const lifecycleType = (status?: string) => status === 'retired' ? 'info' : status === 'idle' ? 'warning' : 'success'
const lifecycleHint = (row: LogCollectorInstanceView) => row.lifecycleStatus === 'retired' ? '采集器已卸载，仅保留历史心跳记录' : '-'
const assignmentText = (status: string) => ({ applied: '已应用', pending: '待应用', failed: '失败', disabled: '已停用' }[status] || status || '待应用')
const assignmentType = (status: string) => status === 'applied' ? 'success' : status === 'failed' ? 'danger' : status === 'disabled' ? 'info' : 'warning'
const policyStatusText = (status: string) => ({ draft: '草稿', published: '已发布', disabled: '已停用' }[status] || status)
const policyStatusType = (status: string) => status === 'published' ? 'success' : status === 'disabled' ? 'info' : 'warning'
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const number = (value: number) => Number(value || 0).toFixed(value && value < 10 ? 2 : 0)
const bytes = (value: number) => value >= 1073741824 ? `${(value / 1073741824).toFixed(1)} GiB` : value >= 1048576 ? `${(value / 1048576).toFixed(1)} MiB` : `${Math.round((value || 0) / 1024)} KiB`

watch(() => props.presetPolicyId, () => openPresetGroup(true))
onMounted(async () => {
  await load()
  timer = setInterval(() => {
    if (!document.hidden) void load(true)
  }, 15000)
})
onBeforeUnmount(() => timer && clearInterval(timer))
</script>

<style scoped>
.instance-page { display: flex; flex-direction: column; gap: 12px; }
.section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin: 0; padding: 18px 20px; }
.section-head h3 { margin: 0; color: #111827; font-size: 15px; font-weight: 650; }
.section-head p { margin: 6px 0 0; color: #667085; font-size: 13px; }
.section-actions { display: flex; align-items: center; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
.section-actions .el-input { width: 220px; }
.section-actions .el-select { width: 145px; }
.instance-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 1px; overflow: hidden; padding: 0; background: #eef0f3; }
.instance-summary > div { min-width: 0; padding: 14px 18px; background: #fff; }
.instance-summary span, .instance-summary strong, .instance-summary small { display: block; }
.instance-summary span { color: #667085; font-size: 12px; }
.instance-summary strong { margin-top: 5px; color: #111827; font-size: 22px; font-weight: 750; }
.instance-summary small { margin-top: 3px; color: #98a2b3; font-size: 11px; }
.instance-summary strong.success { color: #15803d; }
.instance-summary strong.danger { color: #dc2626; }
.groups-panel { overflow: hidden; padding: 0; }
.groups-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 14px 18px; border-bottom: 1px solid #eaecf0; }
.groups-toolbar > div { display: flex; align-items: baseline; gap: 10px; }
.groups-toolbar strong { color: #101828; font-size: 13px; }
.groups-toolbar span { color: #98a2b3; font-size: 11px; }
.policy-groups { border: 0; }
.policy-groups :deep(.el-collapse-item__header) { height: auto; min-height: 78px; padding: 0 18px; border-bottom-color: #eaecf0; line-height: normal; }
.policy-groups :deep(.el-collapse-item__arrow) { margin-left: 14px; color: #667085; font-size: 16px; }
.policy-groups :deep(.el-collapse-item__wrap) { border-bottom-color: #eaecf0; }
.policy-groups :deep(.el-collapse-item__content) { padding: 0; }
.group-title { display: grid; grid-template-columns: 38px minmax(220px, 1fr) auto; align-items: center; gap: 12px; width: 100%; min-width: 0; padding: 13px 0; }
.group-source-icon { display: grid; width: 36px; height: 36px; place-items: center; border: 1px solid #dbe3ec; border-radius: 6px; background: #f8fafc; color: #475467; font-size: 17px; }
.group-source-icon.kubernetes { border-color: #fed7aa; background: #fff7ed; color: #c2410c; }
.group-identity { min-width: 0; }
.group-identity > div { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; }
.group-identity strong { overflow: hidden; color: #101828; font-size: 13px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.group-identity > small { display: block; margin-top: 6px; overflow: hidden; color: #98a2b3; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.group-metrics { display: flex; align-items: center; justify-content: flex-end; gap: 20px; }
.group-metrics > span { display: grid; grid-template-columns: auto auto; align-items: baseline; column-gap: 4px; min-width: 50px; }
.group-metrics small { grid-column: 1 / 3; margin-bottom: 3px; color: #98a2b3; font-size: 10px; }
.group-metrics strong { color: #344054; font-size: 14px; }
.group-metrics strong.success { color: #15803d; }.group-metrics strong.warning { color: #b45309; }.group-metrics strong.danger { color: #dc2626; }
.group-metrics em { color: #98a2b3; font-size: 10px; font-style: normal; }
.group-content { padding: 0 18px 18px; background: #fafbfc; }
.group-context { display: flex; align-items: center; flex-wrap: wrap; gap: 18px; min-height: 38px; color: #667085; font-size: 11px; }
.group-content :deep(.el-table) { border: 1px solid #eaecf0; }
.instance-name { display: flex; align-items: center; min-width: 0; gap: 10px; }
.instance-name strong, .instance-name small { display: block; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.instance-name strong { color: #101828; }
.instance-name small { margin-top: 4px; color: #98a2b3; font-size: 11px; }
.status-dot { width: 9px; height: 9px; flex: 0 0 auto; border-radius: 50%; background: #ef4444; box-shadow: 0 0 0 4px #fef2f2; }
.status-dot.online { background: #22c55e; box-shadow: 0 0 0 4px #f0fdf4; }
.status-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.target-cell strong, .target-cell small, .muted { display: block; }
.target-cell strong { color: #344054; font-size: 13px; }
.target-cell small, .muted { margin-top: 4px; color: #98a2b3; font-size: 11px; }
.error { color: #d92d20; }
@media (max-width: 1100px) {
  .group-title { grid-template-columns: 38px minmax(180px, 1fr); }
  .group-metrics { grid-column: 2; justify-content: flex-start; }
}
@media (max-width: 900px) {
  .section-head { flex-direction: column; }
  .section-actions { justify-content: flex-start; width: 100%; }
  .instance-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .groups-toolbar { align-items: flex-start; flex-direction: column; gap: 5px; }
}
@media (max-width: 640px) {
  .section-actions .el-input, .section-actions .el-select, .section-actions .el-button { width: 100%; }
  .instance-summary { grid-template-columns: 1fr; }
  .policy-groups :deep(.el-collapse-item__header) { padding: 0 12px; }
  .group-title { grid-template-columns: 38px minmax(0, 1fr); }
  .group-metrics { grid-column: 1 / 3; flex-wrap: wrap; gap: 12px; }
  .group-content { padding: 0 12px 12px; }
}
</style>
