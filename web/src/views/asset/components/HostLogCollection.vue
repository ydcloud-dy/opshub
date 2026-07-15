<template>
  <section class="host-log-section">
    <div class="host-log-head">
      <div class="host-log-title">
        <span class="host-log-icon"><el-icon><Document /></el-icon></span>
        <div><strong>日志采集</strong><small>OpsHub Agent 自动拉取并应用采集策略</small></div>
      </div>
      <div class="host-log-actions">
        <el-button @click="openQuery"><el-icon><Search /></el-icon>查询该主机日志</el-button>
        <el-button v-if="isAdmin" type="primary" class="black-button" @click="configure"><el-icon><Setting /></el-icon>配置采集策略</el-button>
      </div>
    </div>

    <div v-loading="loading" class="host-log-body">
      <div class="collector-summary">
        <div><span>Agent 状态</span><el-tag size="small" :type="host.agentId && host.agentStatus === 'online' ? 'success' : 'info'">{{ host.agentId ? (host.agentStatusText || '已安装') : '未安装' }}</el-tag></div>
        <div><span>采集策略</span><strong>{{ assignedPolicies.length }}</strong></div>
        <div><span>配置版本</span><strong>{{ instance?.instance.configVersion || 0 }}</strong></div>
        <div><span>最近写入</span><strong>{{ formatTime(instance?.instance.lastIngestAt) }}</strong></div>
        <div><span>WAL 积压</span><strong>{{ formatBytes(instance?.instance.walBytes || 0) }}</strong></div>
      </div>

      <el-alert v-if="!host.agentId" type="warning" :closable="false" show-icon title="当前主机尚未安装 Agent，策略可以提前配置，安装后会自动生效。" />
      <el-table v-else :data="assignedPolicies" size="small" empty-text="当前主机尚未绑定日志采集策略">
        <el-table-column label="策略" min-width="220"><template #default="{ row }"><strong>{{ row.payload.name }}</strong><small class="policy-path">{{ row.payload.paths.join(', ') }}</small></template></el-table-column>
        <el-table-column label="版本" width="80"><template #default="{ row }">v{{ row.version }}</template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag size="small" :type="row.status === 'published' ? 'success' : 'info'">{{ row.status === 'published' ? '已发布' : row.status === 'disabled' ? '已停用' : '草稿' }}</el-tag></template></el-table-column>
        <el-table-column label="应用状态" width="110"><template #default="{ row }"><el-tag size="small" :type="assignmentType(assignmentFor(row.id)?.applyStatus)">{{ assignmentText(assignmentFor(row.id)?.applyStatus) }}</el-tag></template></el-table-column>
        <el-table-column label="服务 / 环境" min-width="150"><template #default="{ row }">{{ row.payload.service || '-' }} / {{ row.payload.environment || '-' }}</template></el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Document, Search, Setting } from '@element-plus/icons-vue'
import { getLogCollectionPolicies, getLogCollectorInstances, type LogCollectionPolicy, type LogCollectorInstanceView } from '@/api/logcenter'
import { useUserStore } from '@/stores/user'

const props = defineProps<{ host: any }>()
const router = useRouter()
const userStore = useUserStore()
const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))
const loading = ref(false)
const assignedPolicies = ref<LogCollectionPolicy[]>([])
const instance = ref<LogCollectorInstanceView>()

const load = async () => {
  if (!props.host?.id) return
  loading.value = true
  try {
    const [policies, instances] = await Promise.all([getLogCollectionPolicies(), getLogCollectorInstances()]) as any
    assignedPolicies.value = (policies || []).filter((policy: LogCollectionPolicy) => policy.targetHosts.some(item => item.id === props.host.id))
    instance.value = (instances || []).find((item: LogCollectorInstanceView) => item.instance.hostId === props.host.id)
  } finally { loading.value = false }
}

const configure = () => router.push({ path: '/logs/collectors', query: { tab: 'policies', hostId: props.host.id } })
const openQuery = () => router.push({ path: '/logs/query', query: { hostId: props.host.id } })
const assignmentFor = (policyId: number) => instance.value?.assignments.find(item => item.policyId === policyId)
const assignmentText = (status?: string) => ({ applied: '已应用', pending: '待应用', failed: '失败', disabled: '已停用' }[status || ''] || '未下发')
const assignmentType = (status?: string) => status === 'applied' ? 'success' : status === 'failed' ? 'danger' : status === 'pending' ? 'warning' : 'info'
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const formatBytes = (value: number) => value >= 1073741824 ? `${(value / 1073741824).toFixed(1)} GiB` : value >= 1048576 ? `${(value / 1048576).toFixed(1)} MiB` : `${Math.round(value / 1024)} KiB`

watch(() => props.host?.id, load)
onMounted(load)
</script>

<style scoped>
.host-log-section{margin-top:16px;border:1px solid #e7eaf0;border-radius:8px;background:#fff;overflow:hidden}.host-log-head{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:16px 18px;border-bottom:1px solid #eef0f3;background:#fafbfc}.host-log-title{display:flex;align-items:center;gap:11px}.host-log-title strong,.host-log-title small{display:block}.host-log-title strong{color:#20242c;font-size:14px}.host-log-title small{margin-top:4px;color:#8b95a5;font-size:12px}.host-log-icon{display:grid;width:34px;height:34px;place-items:center;border-radius:7px;color:#047857;background:#ecfdf5}.host-log-actions{display:flex;gap:8px}.host-log-body{padding:16px 18px}.collector-summary{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:10px;margin-bottom:14px}.collector-summary>div{min-width:0;padding:11px 12px;border:1px solid #eef0f3;background:#fafafa}.collector-summary span{display:block;margin-bottom:6px;color:#8b95a5;font-size:11px}.collector-summary strong{display:block;overflow:hidden;color:#20242c;font-size:13px;text-overflow:ellipsis;white-space:nowrap}.policy-path{display:block;margin-top:4px;overflow:hidden;color:#8b95a5;font:11px ui-monospace,SFMono-Regular,Menlo,monospace;text-overflow:ellipsis;white-space:nowrap}.black-button{border-color:#111827;background:#111827;color:#fff}.black-button:hover{border-color:#000;background:#000;color:#fff}@media(max-width:900px){.host-log-head{align-items:flex-start;flex-direction:column}.collector-summary{grid-template-columns:repeat(2,minmax(0,1fr))}}
</style>
