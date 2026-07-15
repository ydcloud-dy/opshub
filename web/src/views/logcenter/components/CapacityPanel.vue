<template>
  <section class="panel capacity-panel">
    <div class="panel-head">
      <div><h3>容量预测</h3><p>基于 ClickHouse 当前压缩率、最近 24 小时日志量和磁盘余量估算</p></div>
<<<<<<< HEAD
      <div class="capacity-controls"><el-select v-model="storageId" placeholder="选择存储"><el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" /></el-select><el-input-number v-model="retentionDays" :min="1" :max="3650" controls-position="right" /><el-button :loading="loading" @click="load">重新计算</el-button></div>
=======
      <div class="capacity-action-area">
        <div class="capacity-controls">
          <label class="capacity-control storage-control">
            <span>日志存储</span>
            <el-select v-model="storageId" placeholder="选择存储" @change="handleStorageChange">
              <el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </label>
          <label class="capacity-control retention-control">
            <span>预测保留周期</span>
            <div class="retention-input"><el-input-number v-model="retentionDays" :min="1" :max="3650" controls-position="right" /><em>天</em></div>
          </label>
          <el-button :loading="loading" @click="load">重新计算</el-button>
        </div>
        <small>保留周期默认继承所选存储配置，可按规划周期调整后重新计算</small>
      </div>
>>>>>>> feat: update log
    </div>
    <el-empty v-if="!result && !loading" description="选择已初始化的存储后计算容量" />
    <div v-else v-loading="loading" class="capacity-content">
      <div class="metric-grid">
        <div><span>当前日志行数</span><strong>{{ number(result?.currentRows) }}</strong></div>
        <div><span>当前压缩存储</span><strong>{{ bytes(result?.currentCompressedBytes) }}</strong></div>
        <div><span>最近 24h 日志</span><strong>{{ number(result?.logsLast24Hours) }}</strong></div>
        <div><span>平均单条大小</span><strong>{{ bytes(result?.averageRecordBytes) }}</strong></div>
        <div><span>实测压缩比</span><strong>{{ decimal(result?.compressionRatio) }} : 1</strong></div>
        <div><span>预计每日存储</span><strong>{{ bytes(result?.dailyStoredBytes) }}</strong></div>
      </div>
      <div class="forecast-band">
        <div><span>{{ result?.retentionDays || retentionDays }} 天预计占用</span><strong>{{ bytes(result?.projectedStoredBytes) }}</strong><small>建议预留 {{ bytes(result?.recommendedBytes) }}</small></div>
        <div><span>磁盘可用</span><strong>{{ bytes(result?.diskFreeBytes) }}</strong><small>总容量 {{ bytes(result?.diskTotalBytes) }}</small></div>
        <div><span>按当前速度可写</span><strong>{{ result?.daysUntilFull ? `${decimal(result.daysUntilFull)} 天` : '-' }}</strong><small>不含副本扩容和突发流量</small></div>
      </div>
      <div class="usage-row"><div><strong>预测容量占比</strong><span>{{ decimal(result?.projectedUsagePercent) }}%</span></div><el-progress :percentage="Math.min(100, Number(result?.projectedUsagePercent || 0))" :status="usageStatus" :stroke-width="10" /></div>
      <el-alert v-if="Number(result?.projectedUsagePercent || 0) >= 80" type="warning" :closable="false" show-icon title="预测容量偏高" description="建议降低保留周期、扩容 ClickHouse 磁盘，或为高价值日志单独设置较长保留时间。" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { getLogCapacityEstimate, type LogCapacityEstimate, type LogStorageCluster } from '@/api/logcenter'
const props = defineProps<{ storages: LogStorageCluster[] }>()
const storageId = ref<number>()
const retentionDays = ref(30)
const loading = ref(false)
const result = ref<LogCapacityEstimate>()
const usageStatus = computed(() => Number(result.value?.projectedUsagePercent || 0) >= 90 ? 'exception' : Number(result.value?.projectedUsagePercent || 0) >= 75 ? 'warning' : 'success')
const load = async () => { if (!storageId.value) return; loading.value = true; try { result.value = await getLogCapacityEstimate({ storageId: storageId.value, retentionDays: retentionDays.value }) as any } finally { loading.value = false } }
<<<<<<< HEAD
=======
const handleStorageChange = () => { const storage = props.storages.find(item => item.id === storageId.value); retentionDays.value = storage?.defaultRetentionDays || 30; void load() }
>>>>>>> feat: update log
const chooseDefault = () => { const storage = props.storages.find(item => item.isPrimary && item.initializedAt) || props.storages.find(item => item.initializedAt); if (storage?.id && !storageId.value) { storageId.value = storage.id; retentionDays.value = storage.defaultRetentionDays || 30; void load() } }
watch(() => props.storages, chooseDefault, { deep: true })
const number = (value?: number) => Number(value || 0).toLocaleString()
const decimal = (value?: number) => Number(value || 0).toFixed(2)
const bytes = (value?: number) => { const size = Number(value || 0); if (size >= 1099511627776) return `${(size / 1099511627776).toFixed(2)} TiB`; if (size >= 1073741824) return `${(size / 1073741824).toFixed(2)} GiB`; if (size >= 1048576) return `${(size / 1048576).toFixed(2)} MiB`; if (size >= 1024) return `${(size / 1024).toFixed(1)} KiB`; return `${Math.round(size)} B` }
onMounted(chooseDefault)
</script>

<style scoped>
<<<<<<< HEAD
.capacity-panel{padding:20px}.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:22px}.panel-head h3{margin:0;color:#111827;font-size:15px}.panel-head p{margin:6px 0 0;color:#667085;font-size:13px}.capacity-controls{display:flex;gap:8px}.capacity-controls .el-select{width:220px}.capacity-controls .el-input-number{width:130px}.capacity-content{min-height:260px}.metric-grid{display:grid;grid-template-columns:repeat(3,1fr);border:1px solid #eaecf0}.metric-grid>div{padding:16px 18px;border-right:1px solid #eaecf0;border-bottom:1px solid #eaecf0}.metric-grid>div:nth-child(3n){border-right:0}.metric-grid>div:nth-child(n+4){border-bottom:0}.metric-grid span,.forecast-band span,.forecast-band small{display:block;color:#667085;font-size:12px}.metric-grid strong{display:block;margin-top:7px;color:#101828;font-size:20px}.forecast-band{display:grid;grid-template-columns:repeat(3,1fr);gap:24px;margin:22px 0;padding:18px 20px;background:#f8fafc}.forecast-band strong{display:block;margin:7px 0;color:#111827;font-size:22px}.usage-row{margin-bottom:18px}.usage-row>div{display:flex;justify-content:space-between;margin-bottom:8px;color:#344054;font-size:13px}@media(max-width:900px){.panel-head{flex-direction:column}.capacity-controls{flex-wrap:wrap}.metric-grid,.forecast-band{grid-template-columns:1fr}.metric-grid>div{border-right:0;border-bottom:1px solid #eaecf0}.metric-grid>div:nth-child(n+4){border-bottom:1px solid #eaecf0}.metric-grid>div:last-child{border-bottom:0}}
=======
.capacity-panel{padding:20px}.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:22px}.panel-head h3{margin:0;color:#111827;font-size:15px}.panel-head p{margin:6px 0 0;color:#667085;font-size:13px}.capacity-action-area{display:flex;align-items:flex-end;flex-direction:column;gap:6px}.capacity-action-area>small{color:#98a2b3;font-size:11px}.capacity-controls{display:flex;align-items:flex-end;gap:10px}.capacity-control{display:flex;flex-direction:column;gap:6px}.capacity-control>span{color:#667085;font-size:11px;font-weight:600}.storage-control .el-select{width:220px}.retention-input{display:flex;align-items:center;gap:8px}.retention-input .el-input-number{width:130px}.retention-input em{color:#667085;font-size:12px;font-style:normal}.capacity-content{min-height:260px}.metric-grid{display:grid;grid-template-columns:repeat(3,1fr);border:1px solid #eaecf0}.metric-grid>div{padding:16px 18px;border-right:1px solid #eaecf0;border-bottom:1px solid #eaecf0}.metric-grid>div:nth-child(3n){border-right:0}.metric-grid>div:nth-child(n+4){border-bottom:0}.metric-grid span,.forecast-band span,.forecast-band small{display:block;color:#667085;font-size:12px}.metric-grid strong{display:block;margin-top:7px;color:#101828;font-size:20px}.forecast-band{display:grid;grid-template-columns:repeat(3,1fr);gap:24px;margin:22px 0;padding:18px 20px;background:#f8fafc}.forecast-band strong{display:block;margin:7px 0;color:#111827;font-size:22px}.usage-row{margin-bottom:18px}.usage-row>div{display:flex;justify-content:space-between;margin-bottom:8px;color:#344054;font-size:13px}@media(max-width:900px){.panel-head{flex-direction:column}.capacity-action-area{align-items:flex-start;width:100%}.capacity-controls{flex-wrap:wrap}.metric-grid,.forecast-band{grid-template-columns:1fr}.metric-grid>div{border-right:0;border-bottom:1px solid #eaecf0}.metric-grid>div:nth-child(n+4){border-bottom:1px solid #eaecf0}.metric-grid>div:last-child{border-bottom:0}}@media(max-width:560px){.capacity-controls,.capacity-control,.storage-control .el-select,.retention-input{width:100%}.retention-input .el-input-number{flex:1;width:auto}}
>>>>>>> feat: update log
</style>
