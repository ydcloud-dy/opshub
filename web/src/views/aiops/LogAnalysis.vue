<template>
  <div class="log-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DocumentChecked /></el-icon>
        </div>
        <div>
          <h2 class="page-title">日志分析</h2>
          <p class="page-subtitle">支持手动粘贴、主机日志和 Kubernetes 资源日志，AI 自动提炼异常摘要、证据链和处理建议</p>
        </div>
      </div>
      <div class="header-actions">
        <ProviderSelect v-model="providerId" placeholder="选择模型" />
        <el-button class="black-button" :loading="analyzing" :disabled="analyzing" @click="handleAnalyze">
          <el-icon><MagicStick /></el-icon>
          {{ analyzeButtonText }}
        </el-button>
      </div>
    </div>

    <div class="log-layout">
      <section class="input-card">
        <div class="section-title">日志输入</div>
        <el-radio-group v-model="sourceType" class="source-tabs" :disabled="analyzing" @change="handleSourceTypeChange">
          <el-radio-button label="manual">手动粘贴</el-radio-button>
          <el-radio-button label="host">主机日志</el-radio-button>
          <el-radio-button label="kubernetes">K8s 资源日志</el-radio-button>
        </el-radio-group>
        <div class="input-row">
          <el-input v-if="sourceType === 'manual'" v-model="source" :disabled="analyzing" placeholder="日志来源，例如 order-service / nginx / pod/app-xxx" />
          <template v-if="sourceType === 'host'">
            <el-input v-model="hostKeyword" clearable :disabled="analyzing" placeholder="搜索主机名称或 IP" @keyup.enter="loadHosts" />
            <el-button class="black-button" :loading="hostLoading" :disabled="analyzing" @click="loadHosts">查询主机</el-button>
          </template>
          <el-button v-if="sourceType === 'manual'" :disabled="analyzing" @click="fillExample">填充示例</el-button>
        </div>
        <template v-if="sourceType === 'host'">
          <div class="selector-grid">
            <el-select v-model="hostForm.hostId" filterable :disabled="analyzing" placeholder="选择主机" class="full-width">
              <el-option v-for="host in hosts" :key="host.id" :label="`${host.name} (${host.ip})`" :value="host.id" />
            </el-select>
            <el-input-number v-model="tailLines" :min="20" :max="5000" :step="100" :disabled="analyzing" class="full-width" />
          </div>
          <el-input v-model="hostForm.logPath" :disabled="analyzing" placeholder="日志路径，例如 /var/log/messages 或 /data/app/logs/error.log" />
        </template>
        <template v-else-if="sourceType === 'kubernetes'">
          <el-form label-position="top" class="k8s-log-form">
            <el-form-item label="资源类型">
              <el-segmented v-model="k8sForm.objectType" :options="k8sObjectTypeOptions" :disabled="analyzing" @change="handleK8sObjectTypeChange" />
            </el-form-item>
          </el-form>
          <div class="selector-grid">
            <el-select v-model="k8sForm.clusterId" filterable :disabled="analyzing" placeholder="选择集群" @change="handleClusterChange">
              <el-option v-for="cluster in clusters" :key="cluster.id" :label="cluster.alias || cluster.name" :value="cluster.id" />
            </el-select>
            <el-select v-model="k8sForm.namespace" filterable :disabled="analyzing" placeholder="选择命名空间" @change="loadPods">
              <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
            </el-select>
          </div>
          <div class="selector-grid">
            <el-select v-model="k8sForm.objectName" filterable :disabled="analyzing" :placeholder="`选择 ${k8sObjectTypeLabel}`" @change="handleK8sObjectChange">
              <el-option v-for="item in k8sObjects" :key="`${item.namespace}-${item.name}`" :label="item.name" :value="item.name" />
            </el-select>
            <el-select
              v-model="k8sForm.container"
              clearable
              filterable
              :loading="containerLoading"
              :disabled="analyzing || containerLoading"
              placeholder="容器，可选（默认第一个）"
              no-data-text="选择资源后自动加载容器"
            >
              <el-option v-for="item in containers" :key="item" :label="item" :value="item" />
            </el-select>
          </div>
          <el-input-number v-model="tailLines" :min="20" :max="5000" :step="100" :disabled="analyzing" />
        </template>
        <el-input
          v-if="sourceType === 'manual'"
          v-model="logs"
          type="textarea"
          :rows="22"
          resize="none"
          :disabled="analyzing"
          placeholder="请粘贴最近的错误日志、异常堆栈或访问日志..."
        />
        <div v-else class="source-note">
          点击“开始分析”后会从目标来源读取最近 {{ tailLines }} 行日志并分析，不会执行写入、删除或重启操作。K8s 工作负载会自动选择关联 Pod 读取样本日志。
        </div>
      </section>

      <section class="result-card">
        <div class="section-head">
          <div class="section-title">分析结果</div>
          <el-tag v-if="analyzing && logRun" type="warning" effect="plain">已耗时 {{ logRun.elapsed }}s</el-tag>
        </div>
        <div v-if="analyzing && logRun" class="analysis-progress-card">
          <div class="progress-main">
            <div>
              <div class="progress-title">正在分析 {{ logRun.title }}</div>
              <div class="progress-subtitle">{{ logRun.status }}</div>
            </div>
            <el-progress type="circle" :width="74" :percentage="progressPercent" :stroke-width="7" status="warning" />
          </div>
          <div class="progress-target">
            <span>已锁定来源</span>
            <strong>{{ logRun.target }}</strong>
          </div>
          <div class="progress-steps">
            <div v-for="step in logRun.steps" :key="step" class="progress-step">
              <span class="step-dot" />
              {{ step }}
            </div>
          </div>
          <div class="progress-hint">
            后端正在只读采集日志并调用模型分析，耗时取决于日志行数、目标主机/K8s 接口和模型响应速度。页面会在返回后自动展示结果。
          </div>
        </div>
        <el-alert v-else-if="analysisError" class="analysis-error" type="error" :title="analysisError" show-icon :closable="false" />
        <el-empty v-else-if="!result" description="选择日志来源后点击开始分析" />
        <template v-else>
          <div class="result-meta">
            <el-tag :type="result.fallback ? 'warning' : 'success'" effect="plain">
              {{ result.fallback ? '本地降级分析' : result.model }}
            </el-tag>
            <el-tag type="info" effect="plain">任务 #{{ result.taskId }}</el-tag>
          </div>
          <MarkdownView class="result-markdown" :content="result.conclusion" />
        </template>
      </section>
    </div>

    <section v-if="result" class="evidence-card">
      <div class="section-title">日志证据</div>
      <pre>{{ JSON.stringify(result.evidence, null, 2) }}</pre>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { DocumentChecked, MagicStick } from '@element-plus/icons-vue'
import { analyzeLogs, type AIDiagnosisResponse } from '@/api/aiops'
import { getHostList } from '@/api/host'
import {
  getClusterList,
  getNamespaces,
  getPodDetail,
  getPods,
  getWorkloadDetail,
  getWorkloadPods,
  getWorkloads,
  type Cluster,
  type NamespaceInfo,
  type PodInfo,
  type WorkloadInfo
} from '@/api/kubernetes'
import MarkdownView from './components/MarkdownView.vue'
import ProviderSelect from './components/ProviderSelect.vue'

type LogSourceType = 'manual' | 'host' | 'kubernetes'
type K8sLogObjectType = 'pod' | 'deployment' | 'statefulset' | 'daemonset' | 'job' | 'cronjob'
interface LogAnalysisRun {
  sourceType: LogSourceType
  title: string
  target: string
  startedAt: number
  elapsed: number
  status: string
  steps: string[]
}

const k8sObjectTypeOptions: Array<{ label: string; value: K8sLogObjectType }> = [
  { label: 'Pod', value: 'pod' },
  { label: 'Deployment', value: 'deployment' },
  { label: 'StatefulSet', value: 'statefulset' },
  { label: 'DaemonSet', value: 'daemonset' },
  { label: 'Job', value: 'job' },
  { label: 'CronJob', value: 'cronjob' }
]

const sourceType = ref<LogSourceType>('manual')
const source = ref('')
const logs = ref('')
const analyzing = ref(false)
const analysisError = ref('')
const logRun = ref<LogAnalysisRun | null>(null)
const sourceResults = reactive<Record<LogSourceType, AIDiagnosisResponse | null>>({
  manual: null,
  host: null,
  kubernetes: null
})
const tailLines = ref(200)
const providerId = ref<number | undefined>()
const hostLoading = ref(false)
const containerLoading = ref(false)
const hostKeyword = ref('')
const hosts = ref<any[]>([])
const clusters = ref<Cluster[]>([])
const namespaces = ref<NamespaceInfo[]>([])
const pods = ref<PodInfo[]>([])
const workloads = ref<WorkloadInfo[]>([])
const containers = ref<string[]>([])

const hostForm = reactive({
  hostId: undefined as number | undefined,
  logPath: '/var/log/messages'
})

const k8sForm = reactive({
  objectType: 'pod' as K8sLogObjectType,
  clusterId: undefined as number | undefined,
  namespace: '',
  objectName: '',
  container: ''
})

const result = computed(() => sourceResults[sourceType.value])
const analyzeButtonText = computed(() => (analyzing.value && logRun.value ? `分析中 ${logRun.value.elapsed}s` : '开始分析'))
const progressPercent = computed(() => {
  if (!logRun.value) return 0
  return Math.min(92, Math.max(8, Math.floor(logRun.value.elapsed * 2.2)))
})
const workloadTypeMap: Record<Exclude<K8sLogObjectType, 'pod'>, WorkloadInfo['type']> = {
  deployment: 'Deployment',
  statefulset: 'StatefulSet',
  daemonset: 'DaemonSet',
  job: 'Job',
  cronjob: 'CronJob'
}
const k8sObjectTypeLabel = computed(() => k8sObjectTypeOptions.find(item => item.value === k8sForm.objectType)?.label || '资源')
const k8sObjects = computed<Array<{ name: string; namespace?: string; containers?: { name: string }[] }>>(() => (
  k8sForm.objectType === 'pod' ? pods.value : workloads.value
))

let logTimer: number | undefined

const errorMessage = (error: unknown, fallback: string) => {
  const err = error as any
  const data = err?.response?.data
  if (typeof data === 'string' && data.trim()) return data
  if (data && typeof data === 'object') {
    const message = data.message || data.msg || data.error || data.detail
    const detail = data.detail || data.error
    if (message && detail && detail !== message) return `${message}：${detail}`
    if (message) return message
    if (typeof data.data === 'string' && data.data.trim()) return data.data
  }
  return err?.message || fallback
}

const currentLogTarget = (type: LogSourceType) => {
  if (type === 'manual') {
    return source.value || '手动粘贴日志'
  }
  if (type === 'host') {
    const host = hosts.value.find(item => item.id === hostForm.hostId)
    const hostName = host ? `${host.name || host.hostname || '未命名主机'}(${host.ip || '-'})` : `主机 #${hostForm.hostId || '-'}`
    return `${hostName} · ${hostForm.logPath || '-'} · 最近 ${tailLines.value} 行`
  }
  const cluster = clusters.value.find(item => item.id === k8sForm.clusterId)
  const clusterName = cluster?.alias || cluster?.name || `集群 #${k8sForm.clusterId || '-'}`
  const container = k8sForm.container ? ` · 容器 ${k8sForm.container}` : ''
  return `${clusterName} · ${k8sForm.namespace || '-'} · ${k8sObjectTypeLabel.value}/${k8sForm.objectName || '-'}${container} · 最近 ${tailLines.value} 行`
}

const runTitle = (type: LogSourceType) => {
  if (type === 'manual') return source.value ? `${source.value} 日志分析` : '手动日志分析'
  if (type === 'host') return '主机日志分析'
  return 'Kubernetes 资源日志分析'
}

const runSteps = (type: LogSourceType) => {
  if (type === 'manual') {
    return ['锁定手动粘贴日志', '提取异常片段和错误关键词', '调用模型生成根因与处理建议', '返回 Markdown 分析报告']
  }
  if (type === 'host') {
    return ['锁定目标主机和日志路径', '通过只读 SSH 读取日志样本', '提取异常片段和时间线', '调用模型生成根因与处理建议']
  }
  return ['锁定集群、命名空间和资源对象', '通过 Kubernetes API 读取 Pod 日志样本', '提取异常片段和事件线索', '调用模型生成根因与处理建议']
}

const runStatus = (type: LogSourceType, elapsed: number) => {
  if (elapsed < 3) return '正在提交分析请求并初始化任务...'
  if (type === 'host' && elapsed < 12) return '正在通过只读 SSH 读取目标主机日志样本...'
  if (type === 'kubernetes' && elapsed < 12) return '正在通过 Kubernetes API 读取关联 Pod 日志样本...'
  if (elapsed < 30) return '正在整理异常片段、错误关键词和上下文证据...'
  if (elapsed < 90) return '模型正在生成根因分析，日志内容较多时会稍慢...'
  return '仍在等待模型或后端返回，请勿重复点击；返回后会自动更新结果。'
}

const startLogRun = (type: LogSourceType) => {
  stopLogRun(false)
  logRun.value = {
    sourceType: type,
    title: runTitle(type),
    target: currentLogTarget(type),
    startedAt: Date.now(),
    elapsed: 0,
    status: runStatus(type, 0),
    steps: runSteps(type)
  }
  logTimer = window.setInterval(() => {
    if (!logRun.value) return
    const elapsed = Math.floor((Date.now() - logRun.value.startedAt) / 1000)
    logRun.value.elapsed = elapsed
    logRun.value.status = runStatus(logRun.value.sourceType, elapsed)
  }, 1000)
}

const stopLogRun = (clearRun = true) => {
  if (logTimer) {
    window.clearInterval(logTimer)
    logTimer = undefined
  }
  if (clearRun) {
    logRun.value = null
  }
}

const handleAnalyze = async () => {
  const runType = sourceType.value
  if (runType === 'manual' && !logs.value.trim()) {
    ElMessage.warning('请先粘贴需要分析的日志')
    return
  }
  if (runType === 'host' && (!hostForm.hostId || !hostForm.logPath.trim())) {
    ElMessage.warning('请选择主机并填写日志路径')
    return
  }
  if (runType === 'kubernetes' && (!k8sForm.clusterId || !k8sForm.namespace || !k8sForm.objectName)) {
    ElMessage.warning('请选择集群、命名空间和 Kubernetes 资源')
    return
  }
  const payload = {
    logs: runType === 'manual' ? logs.value : undefined,
    source: source.value || undefined,
    sourceType: runType,
    providerId: providerId.value,
    title: source.value ? `${source.value} 日志分析` : runTitle(runType),
    hostId: hostForm.hostId,
    logPath: hostForm.logPath,
    clusterId: k8sForm.clusterId,
    namespace: k8sForm.namespace,
    k8sObjectType: k8sForm.objectType,
    podName: k8sForm.objectName,
    container: k8sForm.container,
    tailLines: tailLines.value
  }
  analysisError.value = ''
  sourceResults[runType] = null
  startLogRun(runType)
  analyzing.value = true
  try {
    sourceResults[runType] = await analyzeLogs(payload)
  } catch (error) {
    analysisError.value = errorMessage(error, '日志分析失败，请检查日志来源、凭证或模型配置')
    ElMessage.error(analysisError.value)
  } finally {
    analyzing.value = false
    stopLogRun()
  }
}

const handleSourceTypeChange = () => {}

const loadHosts = async () => {
  hostLoading.value = true
  try {
    const data: any = await getHostList({ page: 1, pageSize: 50, keyword: hostKeyword.value })
    hosts.value = data.list || []
    if (!hostForm.hostId && hosts.value.length > 0) {
      hostForm.hostId = hosts.value[0].id
    }
  } finally {
    hostLoading.value = false
  }
}

const loadClusters = async () => {
  clusters.value = await (getClusterList() as unknown as Promise<Cluster[]>)
  if (!k8sForm.clusterId && clusters.value.length > 0) {
    k8sForm.clusterId = clusters.value[0]?.id
    await handleClusterChange()
  }
}

const handleClusterChange = async () => {
  k8sForm.namespace = ''
  k8sForm.objectName = ''
  k8sForm.container = ''
  if (!k8sForm.clusterId) return
  namespaces.value = await (getNamespaces(k8sForm.clusterId) as unknown as Promise<NamespaceInfo[]>)
  if (namespaces.value.length > 0) {
    k8sForm.namespace = namespaces.value[0]?.name || ''
    await loadPods()
  }
}

const loadPods = async () => {
  k8sForm.objectName = ''
  k8sForm.container = ''
  containers.value = []
  workloads.value = []
  pods.value = []
  if (!k8sForm.clusterId || !k8sForm.namespace) return
  if (k8sForm.objectType === 'pod') {
    pods.value = await (getPods(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<PodInfo[]>)
    return
  }
  workloads.value = await (getWorkloads(k8sForm.clusterId, k8sForm.namespace, workloadTypeMap[k8sForm.objectType]) as unknown as Promise<WorkloadInfo[]>)
}

const handleK8sObjectTypeChange = () => {
  loadPods()
}

const normalizeItems = (value: any) => {
  if (Array.isArray(value)) return value
  if (Array.isArray(value?.items)) return value.items
  if (Array.isArray(value?.data?.items)) return value.data.items
  return []
}

const extractContainerNames = (value: any) => {
  const candidates = [
    value?.containers,
    value?.spec?.containers,
    value?.spec?.template?.spec?.containers,
    value?.spec?.jobTemplate?.spec?.template?.spec?.containers
  ]
  const names = candidates
    .flatMap(item => Array.isArray(item) ? item : [])
    .map((item: any) => typeof item === 'string' ? item : item?.name)
    .filter((item: unknown): item is string => typeof item === 'string' && item.trim().length > 0)
  return Array.from(new Set(names))
}

const handleK8sObjectChange = async () => {
  containers.value = []
  k8sForm.container = ''
  if (!k8sForm.clusterId || !k8sForm.namespace || !k8sForm.objectName) {
    return
  }

  containerLoading.value = true
  try {
    let names: string[] = []
    if (k8sForm.objectType === 'pod') {
      const pod = pods.value.find(item => item.name === k8sForm.objectName)
      names = extractContainerNames(pod)
      if (names.length === 0) {
        const detail = await getPodDetail(k8sForm.clusterId, k8sForm.namespace, k8sForm.objectName)
        names = extractContainerNames(detail)
      }
    } else {
      const workloadType = workloadTypeMap[k8sForm.objectType]
      const detail = await getWorkloadDetail(k8sForm.clusterId, k8sForm.namespace, k8sForm.objectName, workloadType)
      const workload = normalizeItems(detail)[0] || detail
      names = extractContainerNames(workload)

      if (names.length === 0) {
        const relatedPods = await getWorkloadPods(k8sForm.clusterId, k8sForm.namespace, k8sForm.objectName, workloadType)
        const podWithContainers = normalizeItems(relatedPods).find((item: any) => extractContainerNames(item).length > 0)
        names = extractContainerNames(podWithContainers)
      }
    }

    containers.value = names
    k8sForm.container = names[0] || ''
  } catch (error) {
    ElMessage.warning(errorMessage(error, '获取容器列表失败，开始分析时会由后端自动选择默认容器'))
  } finally {
    containerLoading.value = false
  }
}

const fillExample = () => {
  source.value = 'example-java-service'
  logs.value = `2026-06-23 09:31:02 ERROR [http-nio-8080-exec-7] com.example.OrderController - request failed
java.net.ConnectException: Connection refused
  at java.base/sun.nio.ch.Net.pollConnect(Native Method)
  at java.base/sun.nio.ch.Net.pollConnectNow(Net.java:672)
Caused by: redis.clients.jedis.exceptions.JedisConnectionException: Failed connecting to host redis.default.svc.cluster.local:6379
2026-06-23 09:31:03 WARN  HealthCheck - liveness probe failed: timeout after 3s
2026-06-23 09:31:05 ERROR Application - startup failed, dependency redis unavailable`
}

onMounted(async () => {
  await Promise.all([loadHosts(), loadClusters()])
})

onUnmounted(() => {
  stopLogRun()
})
</script>

<style scoped>
.log-page {
  min-height: 100%;
}

.page-header,
.input-card,
.result-card,
.evidence-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 18px;
  box-shadow: 0 12px 32px rgba(31, 45, 61, 0.07);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px;
  margin-bottom: 18px;
}

.header-actions {
  display: flex;
  align-items: end;
  justify-content: flex-end;
  gap: 12px;
  flex: 0 0 auto;
  min-width: 430px;
}

.page-title-group {
  display: flex;
  gap: 14px;
  align-items: center;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #111827;
  color: #fff;
  font-size: 20px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  color: #111827;
}

.page-subtitle {
  color: #64748b;
  margin-top: 4px;
  font-size: 13px;
}

.log-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 18px;
}

.input-card,
.result-card,
.evidence-card {
  padding: 20px;
}

.input-card {
  max-width: none;
}

.section-title {
  font-weight: 800;
  color: #111827;
  margin-bottom: 16px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.section-head .section-title {
  margin-bottom: 0;
}

.input-row {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.source-tabs {
  margin-bottom: 14px;
}

.selector-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 12px;
}

.full-width {
  width: 100%;
}

.source-note {
  margin-top: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #f8fafc;
  color: #475569;
  line-height: 1.7;
  padding: 16px;
}

.result-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.analysis-progress-card {
  border: 1px solid #fde68a;
  border-radius: 18px;
  background:
    radial-gradient(circle at 88% 18%, rgba(251, 191, 36, 0.18), transparent 28%),
    linear-gradient(135deg, #fffdf5 0%, #ffffff 48%, #f8fafc 100%);
  padding: 18px;
  min-height: 260px;
}

.progress-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.progress-title {
  color: #111827;
  font-size: 18px;
  font-weight: 900;
}

.progress-subtitle {
  color: #64748b;
  line-height: 1.7;
  margin-top: 6px;
}

.progress-target {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 18px 0;
  padding: 12px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.78);
  color: #64748b;
}

.progress-target strong {
  color: #111827;
  word-break: break-all;
}

.progress-steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.progress-step {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #fff;
  color: #334155;
  line-height: 1.55;
  padding: 12px;
}

.step-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 999px;
  background: #111827;
  margin-top: 7px;
  box-shadow: 0 0 0 5px rgba(17, 24, 39, 0.08);
}

.progress-hint {
  margin-top: 14px;
  color: #64748b;
  line-height: 1.7;
}

.analysis-error {
  margin-bottom: 12px;
}

.evidence-card pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.7;
  font-family: inherit;
  color: #111827;
}

.result-markdown {
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #fff;
  padding: 18px;
  max-height: none;
  min-height: 260px;
  overflow: auto;
}

.evidence-card {
  margin-top: 18px;
}

.evidence-card pre {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  background: #0f172a;
  color: #f8d675;
  border-radius: 14px;
  padding: 16px;
  max-height: 420px;
  overflow: auto;
}

.black-button {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

@media (max-width: 980px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .header-actions {
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .progress-steps {
    grid-template-columns: 1fr;
  }
}

</style>
