<template>
  <div class="instance-page">
<<<<<<< HEAD
    <div class="section-head"><div><h2>采集实例</h2><p>查看每台主机 Agent 的在线状态、配置版本、吞吐和 WAL 积压</p></div><el-button :loading="loading" @click="load"><el-icon><Refresh /></el-icon>刷新</el-button></div>
=======
    <section class="section-head panel"><div><h3>采集实例</h3><p>查看每台主机 Agent 的在线状态、配置版本、吞吐和 WAL 积压</p></div><el-button :loading="loading" @click="load"><el-icon><Refresh /></el-icon>刷新</el-button></section>
>>>>>>> feat: update log
    <section class="panel instance-table" v-loading="loading">
      <el-table :data="items" empty-text="暂无采集实例">
        <el-table-column label="实例" min-width="210"><template #default="{ row }"><div class="instance-name"><span class="status-dot" :class="row.status"></span><div><strong>{{ row.instance.hostname || row.instance.instanceId }}</strong><small>{{ row.instance.instanceId }}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag size="small" :type="row.status === 'online' ? 'success' : 'danger'">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag></template></el-table-column>
        <el-table-column label="Agent 版本" width="110"><template #default="{ row }">{{ row.instance.version || '-' }}</template></el-table-column>
        <el-table-column label="配置版本" width="100"><template #default="{ row }">{{ row.instance.configVersion || 0 }}</template></el-table-column>
        <el-table-column label="下发策略" min-width="190"><template #default="{ row }"><div class="assignment-list"><el-tag v-for="item in row.assignments" :key="item.id" size="small" :type="assignmentType(item.applyStatus)">#{{ item.policyId }} v{{ item.policyVersion }} · {{ assignmentText(item.applyStatus) }}</el-tag><span v-if="!row.assignments.length">-</span></div></template></el-table-column>
        <el-table-column label="输入 / 输出" width="135"><template #default="{ row }"><strong>{{ number(row.instance.inputEps) }}</strong> / {{ number(row.instance.outputEps) }} EPS</template></el-table-column>
        <el-table-column label="WAL" width="105"><template #default="{ row }">{{ bytes(row.instance.walBytes) }}</template></el-table-column>
        <el-table-column label="最后心跳" width="172"><template #default="{ row }">{{ formatTime(row.instance.lastHeartbeatAt) }}</template></el-table-column>
        <el-table-column label="异常" min-width="180" show-overflow-tooltip><template #default="{ row }"><span :class="{ error: row.instance.lastError }">{{ row.instance.lastError || '-' }}</span></template></el-table-column>
        <el-table-column v-if="isAdmin" label="操作" fixed="right" width="92"><template #default="{ row }"><el-button link type="primary" @click="restart(row)">重载</el-button></template></el-table-column>
      </el-table>
    </section>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getLogCollectorInstances, restartLogCollectorInstance, type LogCollectorInstanceView } from '@/api/logcenter'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore(); const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role:any) => role.code === 'admin'))
const loading = ref(false); const items = ref<LogCollectorInstanceView[]>([]); let timer: ReturnType<typeof setInterval> | undefined
const load = async () => { loading.value = true; try { items.value = await getLogCollectorInstances() as any || [] } finally { loading.value = false } }
const restart = async (row: LogCollectorInstanceView) => { await ElMessageBox.confirm(`确认让 ${row.instance.hostname || row.instance.instanceId} 重新拉取并应用配置？`, '重载采集实例', { type:'warning' }); await restartLogCollectorInstance(row.instance.instanceId); ElMessage.success('已发送重载请求'); await load() }
const assignmentText = (status:string) => ({ applied:'已应用', pending:'待应用', failed:'失败', disabled:'已停用' }[status] || status)
const assignmentType = (status:string) => status === 'applied' ? 'success' : status === 'failed' ? 'danger' : status === 'disabled' ? 'info' : 'warning'
const formatTime = (value?:string) => value ? new Date(value).toLocaleString() : '-'; const number = (value:number) => Number(value || 0).toFixed(value && value < 10 ? 2 : 0); const bytes = (value:number) => value >= 1073741824 ? `${(value/1073741824).toFixed(1)} GiB` : value >= 1048576 ? `${(value/1048576).toFixed(1)} MiB` : `${Math.round((value||0)/1024)} KiB`
onMounted(async()=>{ await load(); timer=setInterval(load,15000) }); onBeforeUnmount(()=>timer&&clearInterval(timer))
</script>
<style scoped>
<<<<<<< HEAD
.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:16px}.section-head h2{margin:0;color:#101828;font-size:20px}.section-head p{margin:6px 0 0;color:#667085;font-size:13px}.instance-table{overflow:hidden}.instance-name{display:flex;align-items:center;gap:10px}.instance-name strong,.instance-name small{display:block}.instance-name small{margin-top:4px;color:#98a2b3;font-size:11px}.status-dot{width:9px;height:9px;border-radius:50%;background:#ef4444;box-shadow:0 0 0 4px #fef2f2}.status-dot.online{background:#22c55e;box-shadow:0 0 0 4px #f0fdf4}.assignment-list{display:flex;flex-wrap:wrap;gap:5px}.error{color:#d92d20}
=======
.instance-page{display:flex;flex-direction:column;gap:12px}.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin:0;padding:18px 20px}.section-head h3{margin:0;color:#111827;font-size:15px;font-weight:650}.section-head p{margin:6px 0 0;color:#667085;font-size:13px}.instance-table{overflow:hidden}.instance-name{display:flex;align-items:center;gap:10px}.instance-name strong,.instance-name small{display:block}.instance-name small{margin-top:4px;color:#98a2b3;font-size:11px}.status-dot{width:9px;height:9px;border-radius:50%;background:#ef4444;box-shadow:0 0 0 4px #fef2f2}.status-dot.online{background:#22c55e;box-shadow:0 0 0 4px #f0fdf4}.assignment-list{display:flex;flex-wrap:wrap;gap:5px}.error{color:#d92d20}@media(max-width:760px){.section-head{align-items:flex-start;flex-direction:column}}
>>>>>>> feat: update log
</style>
