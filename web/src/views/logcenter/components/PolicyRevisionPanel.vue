<template>
  <div class="revision-page">
    <div class="section-head"><div><h2>发布记录</h2><p>每次发布都会保留不可变版本，可快速回滚到已验证配置</p></div><el-button :loading="loading" @click="load"><el-icon><Refresh /></el-icon>刷新</el-button></div>
    <section class="panel revision-table" v-loading="loading">
      <el-table :data="items" empty-text="暂无发布记录">
        <el-table-column prop="policyName" label="策略" min-width="210"><template #default="{ row }"><strong>{{ row.policyName || `策略 #${row.policyId}` }}</strong></template></el-table-column>
        <el-table-column label="版本" width="90"><template #default="{ row }"><el-tag size="small">v{{ row.version }}</el-tag></template></el-table-column>
        <el-table-column prop="changeSummary" label="变更说明" min-width="260" show-overflow-tooltip />
        <el-table-column label="校验值" min-width="180"><template #default="{ row }"><code>{{ row.checksum?.slice(0, 16) }}…</code></template></el-table-column>
        <el-table-column label="发布时间" width="180"><template #default="{ row }">{{ formatTime(row.createdAt) }}</template></el-table-column>
        <el-table-column v-if="isAdmin" label="操作" fixed="right" width="100"><template #default="{ row }"><el-button link type="primary" @click="rollback(row)">回滚到此版</el-button></template></el-table-column>
      </el-table>
    </section>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getLogPolicyRevisions, rollbackLogCollectionPolicy, type LogPolicyRevision } from '@/api/logcenter'
import { useUserStore } from '@/stores/user'
const userStore=useUserStore();const isAdmin=computed(()=>(userStore.userInfo?.roles||[]).some((role:any)=>role.code==='admin'))
const loading=ref(false); const items=ref<LogPolicyRevision[]>([])
const load=async()=>{loading.value=true;try{items.value=await getLogPolicyRevisions() as any||[]}finally{loading.value=false}}
const rollback=async(row:LogPolicyRevision)=>{await ElMessageBox.confirm(`确认将“${row.policyName || `策略 #${row.policyId}`}”回滚到 v${row.version}？回滚会生成一个新的发布版本。`,'回滚策略',{type:'warning'});await rollbackLogCollectionPolicy(row.policyId,row.version);ElMessage.success('策略已回滚并重新发布');await load()}
const formatTime=(value?:string)=>value?new Date(value).toLocaleString():'-';onMounted(load)
</script>
<style scoped>
.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:16px}.section-head h2{margin:0;color:#101828;font-size:20px}.section-head p{margin:6px 0 0;color:#667085;font-size:13px}.revision-table{overflow:hidden}code{color:#475467;font:12px ui-monospace,SFMono-Regular,Menlo,monospace}
</style>
