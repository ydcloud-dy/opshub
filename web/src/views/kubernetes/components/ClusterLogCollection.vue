<template>
  <section class="cluster-log-panel" v-loading="loading">
    <header class="panel-head">
      <div class="title-group">
        <span class="title-icon"><el-icon><Histogram /></el-icon></span>
        <div><h3>日志采集</h3><p>Kubernetes DaemonSet · CRI / Docker · 节点 WAL</p></div>
      </div>
      <div class="actions">
        <el-button @click="loadStatus"><el-icon><Refresh /></el-icon>刷新</el-button>
        <el-button @click="downloadManifest"><el-icon><Download /></el-icon>下载 YAML</el-button>
        <el-button class="dark-button" :loading="installing" @click="installCollector"><el-icon><SetUp /></el-icon>{{ status?.installed ? '升级采集器' : '安装采集器' }}</el-button>
        <el-button v-if="status?.installed" type="danger" plain :loading="uninstalling" @click="uninstallCollector"><el-icon><Delete /></el-icon>卸载</el-button>
      </div>
    </header>

    <div class="status-grid">
      <div><span>部署状态</span><strong><i :class="status?.installed ? 'healthy' : 'idle'"></i>{{ status?.installed ? '已安装' : '未安装' }}</strong></div>
      <div><span>DaemonSet 节点</span><strong>{{ status?.readyNodes || 0 }} / {{ status?.desiredNodes || 0 }}</strong></div>
      <div><span>在线实例</span><strong>{{ status?.instanceOnline || 0 }} / {{ status?.instanceTotal || 0 }}</strong></div>
      <div><span>已发布策略</span><strong>{{ status?.policyCount || 0 }}</strong></div>
      <div><span>采集 Token</span><strong>{{ status?.credentialConfigured ? `${status.tokenHint || ''}••••` : '未生成' }}</strong></div>
    </div>

    <el-alert v-if="status?.lastError" type="error" :closable="false" show-icon :title="status.lastError" />

    <div class="instance-head">
      <h4>节点采集实例</h4>
      <div><el-button link @click="openQuery"><el-icon><Search /></el-icon>查询集群日志</el-button><el-button link @click="createPolicy"><el-icon><Plus /></el-icon>新建采集策略</el-button></div>
    </div>
    <el-table :data="status?.instances || []" size="small" empty-text="安装采集器并发布策略后，节点实例会显示在这里">
      <el-table-column prop="nodeName" label="节点" min-width="170" show-overflow-tooltip />
      <el-table-column prop="podName" label="采集 Pod" min-width="190" show-overflow-tooltip />
      <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag size="small" :type="row.status === 'online' ? 'success' : 'info'">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag></template></el-table-column>
      <el-table-column prop="version" label="版本" width="90" />
      <el-table-column label="吞吐" width="150"><template #default="{ row }">{{ number(row.outputEps) }} EPS</template></el-table-column>
      <el-table-column label="WAL" width="110"><template #default="{ row }">{{ formatBytes(row.walBytes) }}</template></el-table-column>
      <el-table-column label="最近写入" width="180"><template #default="{ row }">{{ formatTime(row.lastIngestAt) }}</template></el-table-column>
      <el-table-column prop="lastError" label="最近错误" min-width="190" show-overflow-tooltip><template #default="{ row }">{{ row.lastError || '-' }}</template></el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Download, Histogram, Plus, Refresh, Search, SetUp } from '@element-plus/icons-vue'
import {
  generateLogKubernetesCollectorManifest, getLogKubernetesCollectorStatus,
  installLogKubernetesCollector, uninstallLogKubernetesCollector,
  type LogKubernetesCollectorStatus,
} from '@/api/logcenter'

const props = defineProps<{ clusterId: number }>()
const router = useRouter()
const status = ref<LogKubernetesCollectorStatus>()
const loading = ref(false)
const installing = ref(false)
const uninstalling = ref(false)

const loadStatus = async () => {
  if (!props.clusterId) return
  loading.value = true
  try { status.value = await getLogKubernetesCollectorStatus(props.clusterId) as any } finally { loading.value = false }
}

const installCollector = async () => {
  await ElMessageBox.confirm(
    status.value?.installed ? '升级会轮换集群采集 Token，并滚动更新所有节点采集 Pod。' : '将在目标集群创建只读 RBAC、ConfigMap、Secret 和 DaemonSet。',
    status.value?.installed ? '升级日志采集器' : '安装日志采集器', { type: 'warning', confirmButtonText: '确认执行' },
  )
  installing.value = true
  try {
    await installLogKubernetesCollector(props.clusterId)
    ElMessage.success(status.value?.installed ? '采集器升级已提交' : '采集器安装已提交')
    await loadStatus()
  } finally { installing.value = false }
}

const downloadManifest = async () => {
  await ElMessageBox.confirm('生成 YAML 会轮换采集 Token，下载后需要立即在目标集群应用。', '下载安装 YAML', { type: 'warning', confirmButtonText: '生成并下载' })
  const result = await generateLogKubernetesCollectorManifest(props.clusterId) as any
  const blob = new Blob([result.yaml], { type: 'application/yaml;charset=utf-8' })
  const href = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = href; anchor.download = `opshub-log-agent-cluster-${props.clusterId}.yaml`; anchor.click()
  URL.revokeObjectURL(href)
  ElMessage.success('安装 YAML 已下载')
  await loadStatus()
}

const uninstallCollector = async () => {
  await ElMessageBox.confirm('卸载后集群节点将停止采集，节点 WAL 数据不会自动删除。', '卸载日志采集器', { type: 'warning', confirmButtonText: '确认卸载' })
  uninstalling.value = true
  try { await uninstallLogKubernetesCollector(props.clusterId); ElMessage.success('日志采集器已卸载'); await loadStatus() } finally { uninstalling.value = false }
}

const createPolicy = () => router.push({ path: '/logs/collectors', query: { tab: 'policies', clusterId: props.clusterId } })
const openQuery = () => router.push({ path: '/logs/query', query: { clusterId: props.clusterId } })
const number = (value?: number) => Number(value || 0).toFixed(Number(value || 0) % 1 ? 1 : 0)
const formatBytes = (value?: number) => !value ? '0 B' : value >= 1073741824 ? `${(value / 1073741824).toFixed(1)} GiB` : value >= 1048576 ? `${(value / 1048576).toFixed(1)} MiB` : `${Math.round(value / 1024)} KiB`
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'

onMounted(loadStatus)
</script>

<style scoped>
.cluster-log-panel { margin-bottom:20px; padding:22px 24px; border:1px solid #e4e9f2; border-radius:8px; background:#fff; box-shadow:0 2px 10px rgba(15,23,42,.05); }
.panel-head,.title-group,.actions,.instance-head,.instance-head>div { display:flex; align-items:center; }
.panel-head,.instance-head { justify-content:space-between; gap:18px; }.title-group { gap:12px; }.title-icon { display:grid;place-items:center;width:40px;height:40px;border:1px solid #dfe5ee;border-radius:7px;background:#f8fafc;color:#111827;font-size:20px; }
h3,h4,p { margin:0; }h3 { font-size:17px;color:#101828; }h4 { font-size:14px;color:#344054; }.title-group p { margin-top:4px;color:#7b8495;font-size:12px; }.actions { flex-wrap:wrap;justify-content:flex-end;gap:8px; }.dark-button { border-color:#111827!important;background:#111827!important;color:#fff!important; }
.status-grid { display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px;margin:20px 0; }.status-grid>div { padding:14px 16px;border:1px solid #e8ecf2;background:#fafbfc; }.status-grid span,.status-grid strong { display:block; }.status-grid span { color:#8a94a6;font-size:12px; }.status-grid strong { margin-top:7px;color:#182230;font-size:16px; }.status-grid i { display:inline-block;width:8px;height:8px;margin-right:7px;border-radius:50%; }.status-grid i.healthy { background:#22c55e; }.status-grid i.idle { background:#94a3b8; }
.instance-head { margin:20px 0 10px; }.instance-head>div { gap:10px; }
@media (max-width:1100px) { .panel-head { align-items:flex-start;flex-direction:column; }.actions { justify-content:flex-start; }.status-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
@media (max-width:640px) { .cluster-log-panel { padding:16px; }.status-grid { grid-template-columns:1fr; }.instance-head { align-items:flex-start;flex-direction:column; } }
</style>
