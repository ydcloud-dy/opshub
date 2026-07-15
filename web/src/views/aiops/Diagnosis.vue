<template>
  <div class="diagnosis-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><FirstAidKit /></el-icon>
        </div>
        <div>
          <h2 class="page-title">智能诊断</h2>
          <p class="page-subtitle">支持 Kubernetes 对象和主机资产诊断，自动整理证据链并生成排障建议</p>
        </div>
      </div>
      <div class="header-actions">
        <ProviderSelect v-model="providerId" placeholder="选择模型" />
        <el-button class="black-button" :loading="diagnosing" @click="handleDiagnose">
          <el-icon><MagicStick /></el-icon>
          {{ diagnosing && activeDiagnosis ? `诊断中：${activeDiagnosis.title}` : '开始诊断' }}
        </el-button>
      </div>
    </div>

    <div class="mode-card">
      <button class="mode-item" :class="{ active: diagnoseMode === 'kubernetes' }" @click="diagnoseMode = 'kubernetes'">
        <span class="mode-icon"><el-icon><Connection /></el-icon></span>
        <span>
          <strong>Kubernetes 诊断</strong>
          <small>Pod / 工作负载 / Service / Ingress / Node 状态、事件、日志</small>
        </span>
      </button>
      <button class="mode-item" :class="{ active: diagnoseMode === 'host' }" @click="diagnoseMode = 'host'">
        <span class="mode-icon"><el-icon><Monitor /></el-icon></span>
        <span>
          <strong>主机诊断</strong>
          <small>资源水位、Agent、在线状态、采集时间</small>
        </span>
      </button>
    </div>

    <div class="diagnosis-grid">
      <section class="form-card">
        <template v-if="diagnoseMode === 'kubernetes'">
          <div class="section-title">Kubernetes 对象</div>
          <el-form label-position="top">
            <el-form-item label="对象类型">
              <div class="object-type-grid">
                <button
                  v-for="item in objectTypeOptions"
                  :key="item.value"
                  type="button"
                  class="object-type-chip"
                  :class="{ active: k8sForm.objectType === item.value }"
                  :title="item.label"
                  @click="selectObjectType(item.value)"
                >
                  <span class="chip-dot"></span>
                  <span class="chip-text">{{ item.label }}</span>
                </button>
              </div>
            </el-form-item>
            <el-form-item label="集群">
              <el-select v-model="k8sForm.clusterId" placeholder="选择集群" filterable class="full-width" @change="handleClusterChange">
                <el-option v-for="cluster in clusters" :key="cluster.id" :label="cluster.alias || cluster.name" :value="cluster.id" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="needsNamespace" label="命名空间">
              <el-select v-model="k8sForm.namespace" placeholder="选择命名空间" filterable class="full-width" @change="reloadObjects">
                <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
              </el-select>
            </el-form-item>
            <el-form-item :label="objectTypeLabel">
              <el-select v-if="canSelectObject" v-model="k8sForm.name" :placeholder="`选择 ${objectTypeLabel}`" filterable class="full-width" @change="handleObjectChange">
                <el-option
                  v-for="item in objects"
                  :key="`${item.namespace || 'cluster'}-${item.name}`"
                  :label="item.namespace ? `${item.namespace}/${item.name}` : item.name"
                  :value="item.name"
                />
              </el-select>
              <el-input v-else v-model="k8sForm.name" :placeholder="`请输入 ${objectTypeLabel} 名称`" />
            </el-form-item>
            <el-form-item v-if="needsContainer" label="容器">
              <el-select v-model="k8sForm.container" placeholder="可选，默认第一个容器" clearable class="full-width">
                <el-option v-for="item in containers" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志行数">
              <el-input-number v-model="k8sForm.tailLines" :min="20" :max="1000" :step="20" />
            </el-form-item>
          </el-form>
        </template>

        <template v-else>
          <div class="section-title">主机资产</div>
          <div class="host-search">
            <el-input v-model="hostKeyword" clearable placeholder="搜索主机名称或 IP" @keyup.enter="loadHosts" />
            <el-button class="black-button" :loading="hostLoading" @click="loadHosts">查询</el-button>
          </div>
          <el-scrollbar class="host-list">
            <button
              v-for="host in hosts"
              :key="host.id"
              class="host-option"
              :class="{ active: hostForm.hostId === host.id, diagnosing: isDiagnosingHost(host.id) }"
              @click="selectHost(host)"
            >
              <span class="host-avatar" :class="host.status === 1 ? 'online' : 'offline'">{{ host.name?.slice(0, 1) || 'H' }}</span>
              <span class="host-main">
                <strong>{{ host.name }}</strong>
                <small>{{ host.ip }} · {{ host.os || '未知系统' }}</small>
              </span>
              <span class="host-badges">
                <el-tag v-if="isDiagnosingHost(host.id)" size="small" type="info" effect="dark">诊断中</el-tag>
                <el-tag size="small" :type="host.status === 1 ? 'success' : 'info'" effect="plain">
                  {{ host.statusText || (host.status === 1 ? '在线' : '未知') }}
                </el-tag>
              </span>
            </button>
          </el-scrollbar>
          <div v-if="selectedHost" class="host-summary">
            <div>
              <span>CPU</span>
              <strong>{{ formatPercent(selectedHost.cpuUsage) }}</strong>
            </div>
            <div>
              <span>内存</span>
              <strong>{{ formatPercent(selectedHost.memoryUsage) }}</strong>
            </div>
            <div>
              <span>磁盘</span>
              <strong>{{ formatPercent(selectedHost.diskUsage) }}</strong>
            </div>
          </div>
          <el-form label-position="top" class="host-focus-form">
            <el-form-item label="诊断关注点">
              <el-input
                v-model="hostForm.focus"
                type="textarea"
                :rows="4"
                resize="none"
                placeholder="例如：磁盘使用率太高、Agent 离线、应用响应慢、主机采集数据异常"
              />
            </el-form-item>
          </el-form>
        </template>
      </section>

      <section class="result-card">
        <div class="section-title">诊断结果</div>
        <div v-if="activeDiagnosis" class="diagnosis-running-card" :class="activeDiagnosis.status">
          <div class="running-head">
            <div class="running-icon">
              <el-icon><Loading /></el-icon>
            </div>
            <div class="running-main">
              <div class="running-label">
                {{ activeDiagnosis.status === 'running' ? '正在诊断' : activeDiagnosis.status === 'success' ? '本次诊断目标' : '诊断失败' }}
              </div>
              <h3>{{ activeDiagnosis.title }}</h3>
              <p>{{ activeDiagnosis.subtitle }}</p>
            </div>
            <el-tag :type="activeDiagnosis.status === 'running' ? 'info' : activeDiagnosis.status === 'success' ? 'success' : 'danger'" effect="plain">
              {{ activeDiagnosis.status === 'running' ? '进行中' : activeDiagnosis.status === 'success' ? '已完成' : '失败' }}
            </el-tag>
          </div>
          <div class="running-details">
            <span v-for="item in activeDiagnosis.details" :key="item">{{ item }}</span>
          </div>
          <div v-if="activeDiagnosis.status === 'running'" class="running-notice">
            本次任务已固定为上方目标，左侧切换主机或对象不会影响当前正在执行的诊断。
          </div>
          <div v-if="activeDiagnosis.status === 'running'" class="diagnosis-steps">
            <div class="diagnosis-step active">
              <span></span>
              已锁定诊断目标，不受左侧切换影响
            </div>
            <div class="diagnosis-step active">
              <span></span>
              正在采集对象状态、资源水位、事件和上下文证据
            </div>
            <div class="diagnosis-step">
              <span></span>
              调用 AI 模型生成结论和排障建议
            </div>
          </div>
          <div class="running-time">
            开始时间：{{ activeDiagnosis.startedAt }}
            <span v-if="activeDiagnosis.status === 'running'">已运行：{{ formatElapsed(activeDiagnosis.startedAtMs) }}</span>
          </div>
        </div>
        <el-empty v-if="!activeDiagnosis && !displayResult" description="选择对象后点击开始诊断" />
        <div v-if="displayResult">
          <div class="result-meta">
            <el-tag :type="displayResult.fallback ? 'warning' : 'success'" effect="plain">
              {{ displayResult.fallback ? '本地降级分析' : displayResult.model }}
            </el-tag>
            <el-tag type="info" effect="plain">任务 #{{ displayResult.taskId }}</el-tag>
          </div>
          <MarkdownView class="result-markdown" :content="displayResult.conclusion" />
        </div>
      </section>
    </div>

    <section v-if="displayResult" class="evidence-card">
      <div class="section-title">证据链</div>
      <pre>{{ JSON.stringify(displayResult.evidence, null, 2) }}</pre>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, FirstAidKit, Loading, MagicStick, Monitor } from '@element-plus/icons-vue'
import { diagnoseHost, diagnoseKubernetes, type AIDiagnosisResponse, type AIKubernetesDiagnosisObjectType } from '@/api/aiops'
import { getHostList } from '@/api/host'
import MarkdownView from './components/MarkdownView.vue'
import ProviderSelect from './components/ProviderSelect.vue'
import {
  getClusterList,
  getDeployments,
  getIngresses,
  getNamespaces,
  getNodes,
  getConfigMaps,
  getSecrets,
  getPersistentVolumeClaims,
  getPersistentVolumes,
  getStorageClasses,
  getEndpoints,
  getNetworkPolicies,
  getPods,
  getServices,
  getWorkloads,
  type Cluster,
  type DeploymentInfo,
  type NamespaceInfo,
  type NodeInfo,
  type PodInfo,
  type ServiceInfo,
  type IngressInfo,
  type WorkloadInfo,
  type ConfigMapInfo,
  type SecretInfo,
  type PVCInfo,
  type PVInfo,
  type StorageClassInfo,
  type EndpointsInfo,
  type NetworkPolicyDetailInfo
} from '@/api/kubernetes'

const objectTypeOptions: Array<{ label: string; value: AIKubernetesDiagnosisObjectType }> = [
  { label: 'Pod', value: 'pod' },
  { label: 'Deployment', value: 'deployment' },
  { label: 'StatefulSet', value: 'statefulset' },
  { label: 'DaemonSet', value: 'daemonset' },
  { label: 'Job', value: 'job' },
  { label: 'CronJob', value: 'cronjob' },
  { label: 'Service', value: 'service' },
  { label: 'Ingress', value: 'ingress' },
  { label: 'Node', value: 'node' },
  { label: 'Namespace', value: 'namespace' },
  { label: 'ConfigMap', value: 'configmap' },
  { label: 'Secret', value: 'secret' },
  { label: 'PVC', value: 'persistentvolumeclaim' },
  { label: 'PV', value: 'persistentvolume' },
  { label: 'StorageClass', value: 'storageclass' },
  { label: 'Endpoints', value: 'endpoints' },
  { label: 'NetworkPolicy', value: 'networkpolicy' }
]

const diagnosisCacheKey = 'opshub-aiops-diagnosis-state-v2'

interface ActiveDiagnosis {
  mode: 'kubernetes' | 'host'
  status: 'running' | 'success' | 'error'
  targetId?: number | string
  title: string
  subtitle: string
  details: string[]
  startedAt: string
  startedAtMs: number
}

const diagnoseMode = ref<'kubernetes' | 'host'>('kubernetes')
const clusters = ref<Cluster[]>([])
const namespaces = ref<NamespaceInfo[]>([])
const pods = ref<PodInfo[]>([])
const deployments = ref<DeploymentInfo[]>([])
const objects = ref<Array<{ name: string; namespace?: string; containers?: { name: string }[] }>>([])
const containers = ref<string[]>([])
const k8sResult = ref<AIDiagnosisResponse | null>(null)
const hostResult = ref<AIDiagnosisResponse | null>(null)
const diagnosing = ref(false)
const activeDiagnosis = ref<ActiveDiagnosis | null>(null)
const providerId = ref<number | undefined>()

const hostKeyword = ref('')
const hostLoading = ref(false)
const hosts = ref<any[]>([])
const now = ref(Date.now())
let diagnosisTicker: number | undefined

const k8sForm = reactive({
  objectType: 'pod' as AIKubernetesDiagnosisObjectType,
  clusterId: undefined as number | undefined,
  namespace: '',
  name: '',
  container: '',
  tailLines: 120
})

const hostForm = reactive({
  hostId: undefined as number | undefined,
  focus: ''
})

const selectedHost = computed(() => hosts.value.find(item => item.id === hostForm.hostId))
const result = computed(() => diagnoseMode.value === 'kubernetes' ? k8sResult.value : hostResult.value)
const displayResult = computed(() => {
  if (!activeDiagnosis.value) return result.value
  return activeDiagnosis.value.mode === 'kubernetes' ? k8sResult.value : hostResult.value
})
const isDiagnosingHost = (hostId?: number | string) => {
  if (hostId === undefined || hostId === null || !activeDiagnosis.value) return false
  return activeDiagnosis.value.mode === 'host'
    && activeDiagnosis.value.status === 'running'
    && String(activeDiagnosis.value.targetId) === String(hostId)
}
const objectTypeLabel = computed(() => objectTypeOptions.find(item => item.value === k8sForm.objectType)?.label || '对象')
const canSelectObject = computed(() => objects.value.length > 0)
const clusterScopeTypes: AIKubernetesDiagnosisObjectType[] = ['node', 'namespace', 'persistentvolume', 'storageclass']
const needsNamespace = computed(() => !clusterScopeTypes.includes(k8sForm.objectType))
const needsContainer = computed(() => ['pod', 'deployment', 'statefulset', 'daemonset', 'job'].includes(k8sForm.objectType))
const workloadTypeMap: Partial<Record<AIKubernetesDiagnosisObjectType, WorkloadInfo['type']>> = {
  statefulset: 'StatefulSet',
  daemonset: 'DaemonSet',
  job: 'Job',
  cronjob: 'CronJob'
}

const normalizeObjects = <T extends { name: string; namespace?: string; containers?: { name: string }[] }>(items: T[]) => {
  return (items || []).map(item => ({
    name: item.name,
    namespace: item.namespace,
    containers: item.containers
  }))
}

const selectObjectType = async (value: AIKubernetesDiagnosisObjectType) => {
  if (k8sForm.objectType === value) return
  k8sForm.objectType = value
  await reloadObjects()
}

const restoreDiagnosisState = () => {
  try {
    const raw = localStorage.getItem(diagnosisCacheKey)
    if (!raw) return
    const cached = JSON.parse(raw)
    if (cached?.diagnoseMode === 'kubernetes' || cached?.diagnoseMode === 'host') {
      diagnoseMode.value = cached.diagnoseMode
    }
    if (cached?.k8sForm) {
      Object.assign(k8sForm, cached.k8sForm)
    }
    if (cached?.hostForm) {
      Object.assign(hostForm, cached.hostForm)
    }
    k8sResult.value = cached?.k8sResult || null
    hostResult.value = cached?.hostResult || null
  } catch {
    localStorage.removeItem(diagnosisCacheKey)
  }
}

const cacheDiagnosisState = () => {
  localStorage.setItem(diagnosisCacheKey, JSON.stringify({
      diagnoseMode: diagnoseMode.value,
      k8sForm,
      hostForm,
      k8sResult: k8sResult.value,
      hostResult: hostResult.value
  }))
}

watch(
  () => ({
    diagnoseMode: diagnoseMode.value,
    k8sForm: { ...k8sForm },
    hostForm: { ...hostForm },
    k8sResult: k8sResult.value,
    hostResult: hostResult.value
  }),
  cacheDiagnosisState,
  { deep: true }
)

const loadClusters = async () => {
  clusters.value = await (getClusterList() as unknown as Promise<Cluster[]>)
  if (!k8sForm.clusterId && clusters.value.length > 0) {
    k8sForm.clusterId = clusters.value[0]?.id
  }
  if (k8sForm.clusterId) {
    await loadNamespacesForCluster()
    await reloadObjects({ preserveName: true })
  }
}

const handleClusterChange = async () => {
  k8sForm.namespace = ''
  k8sForm.name = ''
  k8sForm.container = ''
  if (!k8sForm.clusterId) return
  await loadNamespacesForCluster()
  await reloadObjects()
}

const loadNamespacesForCluster = async () => {
  if (!k8sForm.clusterId) return
  namespaces.value = await (getNamespaces(k8sForm.clusterId) as unknown as Promise<NamespaceInfo[]>)
  if (needsNamespace.value && namespaces.value.length > 0) {
    const exists = namespaces.value.some(item => item.name === k8sForm.namespace)
    if (!exists) {
      k8sForm.namespace = namespaces.value[0]?.name || ''
    }
  }
}

const reloadObjects = async (options?: { preserveName?: boolean }) => {
  const previousName = k8sForm.name
  const previousContainer = k8sForm.container
  if (!options?.preserveName) {
    k8sForm.name = ''
    k8sForm.container = ''
  }
  containers.value = []
  objects.value = []
  if (!k8sForm.clusterId) return
  if (needsNamespace.value && !k8sForm.namespace) return
  if (k8sForm.objectType === 'pod') {
    pods.value = await (getPods(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<PodInfo[]>)
    objects.value = normalizeObjects(pods.value)
  } else if (k8sForm.objectType === 'deployment') {
    deployments.value = await (getDeployments(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<DeploymentInfo[]>)
    objects.value = normalizeObjects(deployments.value)
  } else if (workloadTypeMap[k8sForm.objectType]) {
    objects.value = normalizeObjects(await (getWorkloads(k8sForm.clusterId, k8sForm.namespace, workloadTypeMap[k8sForm.objectType]) as unknown as Promise<WorkloadInfo[]>))
  } else if (k8sForm.objectType === 'service') {
    objects.value = normalizeObjects(await (getServices(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<ServiceInfo[]>))
  } else if (k8sForm.objectType === 'ingress') {
    objects.value = normalizeObjects(await (getIngresses(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<IngressInfo[]>))
  } else if (k8sForm.objectType === 'node') {
    objects.value = normalizeObjects(await (getNodes(k8sForm.clusterId) as unknown as Promise<NodeInfo[]>))
  } else if (k8sForm.objectType === 'namespace') {
    objects.value = namespaces.value.map(item => ({ name: item.name }))
  } else if (k8sForm.objectType === 'configmap') {
    objects.value = normalizeObjects(await (getConfigMaps(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<ConfigMapInfo[]>))
  } else if (k8sForm.objectType === 'secret') {
    objects.value = normalizeObjects(await (getSecrets(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<SecretInfo[]>))
  } else if (k8sForm.objectType === 'persistentvolumeclaim') {
    objects.value = normalizeObjects(await (getPersistentVolumeClaims(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<PVCInfo[]>))
  } else if (k8sForm.objectType === 'persistentvolume') {
    objects.value = normalizeObjects(await (getPersistentVolumes(k8sForm.clusterId) as unknown as Promise<PVInfo[]>))
  } else if (k8sForm.objectType === 'storageclass') {
    objects.value = normalizeObjects(await (getStorageClasses(k8sForm.clusterId) as unknown as Promise<StorageClassInfo[]>))
  } else if (k8sForm.objectType === 'endpoints') {
    objects.value = normalizeObjects(await (getEndpoints(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<EndpointsInfo[]>))
  } else if (k8sForm.objectType === 'networkpolicy') {
    objects.value = normalizeObjects(await (getNetworkPolicies(k8sForm.clusterId, k8sForm.namespace) as unknown as Promise<NetworkPolicyDetailInfo[]>))
  }
  if (options?.preserveName && previousName && objects.value.some(item => item.name === previousName)) {
    k8sForm.name = previousName
    k8sForm.container = previousContainer
    handleObjectChange()
  } else if (objects.value.length === 1) {
    k8sForm.name = objects.value[0].name
    handleObjectChange()
  }
}

const handleObjectChange = () => {
  containers.value = []
  k8sForm.container = ''
  if (k8sForm.objectType !== 'pod') return
  const pod = pods.value.find(item => item.name === k8sForm.name)
  containers.value = pod?.containers?.map(item => item.name) || []
  if (containers.value.length > 0) {
    k8sForm.container = containers.value[0] || ''
  }
}

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

const selectHost = (host: any) => {
  hostForm.hostId = host.id
}

const handleDiagnose = async () => {
  if (diagnoseMode.value === 'kubernetes') {
    if (!k8sForm.clusterId || !k8sForm.name || (needsNamespace.value && !k8sForm.namespace)) {
      ElMessage.warning('请先选择完整的 Kubernetes 诊断对象')
      return
    }
  } else if (!hostForm.hostId) {
    ElMessage.warning('请先选择要诊断的主机')
    return
  }

  const namespaceRequired = needsNamespace.value
  const k8sPayload = diagnoseMode.value === 'kubernetes'
    ? {
        objectType: k8sForm.objectType,
        clusterId: k8sForm.clusterId as number,
        namespace: namespaceRequired ? k8sForm.namespace : '',
        name: k8sForm.name,
        container: k8sForm.container,
        tailLines: k8sForm.tailLines,
        providerId: providerId.value
      }
    : null
  const hostPayload = diagnoseMode.value === 'host'
    ? {
        hostId: hostForm.hostId as number,
        focus: hostForm.focus.trim(),
        providerId: providerId.value,
        host: selectedHost.value ? { ...selectedHost.value } : null
      }
    : null
  const targetSnapshot = k8sPayload
    ? buildK8sDiagnosisSnapshot(k8sPayload, namespaceRequired)
    : buildHostDiagnosisSnapshot(hostPayload?.host, hostPayload?.hostId, hostPayload?.focus)
  activeDiagnosis.value = targetSnapshot
  diagnosing.value = true
  try {
    if (targetSnapshot.mode === 'kubernetes') {
      k8sResult.value = null
      k8sResult.value = await diagnoseKubernetes(k8sPayload!)
      activeDiagnosis.value = { ...targetSnapshot, status: 'success' }
      return
    }
    hostResult.value = null
    hostResult.value = await diagnoseHost(hostPayload!.hostId, { focus: hostPayload!.focus, providerId: hostPayload!.providerId })
    activeDiagnosis.value = { ...targetSnapshot, status: 'success' }
  } catch (error) {
    activeDiagnosis.value = { ...targetSnapshot, status: 'error' }
    ElMessage.error('诊断失败，请检查目标对象、凭据或 AI 模型配置')
  } finally {
    diagnosing.value = false
  }
}

const formatPercent = (value?: number) => {
  if (value === undefined || value === null || Number.isNaN(Number(value))) return '-'
  return `${Number(value).toFixed(1)}%`
}

const formatDiagnosisTime = () => {
  return new Date().toLocaleString()
}

const formatElapsed = (startedAtMs: number) => {
  const seconds = Math.max(0, Math.floor((now.value - startedAtMs) / 1000))
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  return `${minutes} 分 ${restSeconds} 秒`
}

const buildK8sDiagnosisSnapshot = (
  payload: {
    objectType: AIKubernetesDiagnosisObjectType
    clusterId: number
    namespace?: string
    name: string
    container?: string
    tailLines?: number
  },
  namespaceRequired: boolean
): ActiveDiagnosis => {
  const cluster = clusters.value.find(item => item.id === payload.clusterId)
  const typeLabel = objectTypeOptions.find(item => item.value === payload.objectType)?.label || '对象'
  const scope = namespaceRequired ? `${payload.namespace}/` : ''
  const details = [
    `集群：${cluster?.alias || cluster?.name || `#${payload.clusterId}`}`,
    namespaceRequired ? `命名空间：${payload.namespace}` : '范围：集群级资源',
    `对象类型：${typeLabel}`,
    `对象名称：${payload.name}`
  ]
  if (payload.container) {
    details.push(`容器：${payload.container}`)
  }
  return {
    mode: 'kubernetes',
    status: 'running',
    targetId: `${payload.clusterId}:${payload.objectType}:${payload.namespace || '_cluster'}:${payload.name}`,
    title: `${typeLabel} ${scope}${payload.name}`,
    subtitle: `正在诊断 Kubernetes 对象，集群为 ${cluster?.alias || cluster?.name || `#${payload.clusterId}`}`,
    details,
    startedAt: formatDiagnosisTime(),
    startedAtMs: Date.now()
  }
}

const buildHostDiagnosisSnapshot = (host: any, hostId?: number, focus = ''): ActiveDiagnosis => {
  const details = [
    `主机：${host?.name || `#${hostId}`}`,
    `IP：${host?.ip || '-'}`,
    `系统：${host?.os || '未知系统'}`,
    `状态：${host?.statusText || (host?.status === 1 ? '在线' : '未知')}`
  ]
  if (focus) {
    details.push(`关注点：${focus}`)
  }
  return {
    mode: 'host',
    status: 'running',
    targetId: hostId,
    title: host?.name || `主机 #${hostId}`,
    subtitle: `正在诊断主机资产 ${host?.ip || ''}`,
    details,
    startedAt: formatDiagnosisTime(),
    startedAtMs: Date.now()
  }
}

onMounted(async () => {
  diagnosisTicker = window.setInterval(() => {
    if (activeDiagnosis.value?.status === 'running') {
      now.value = Date.now()
    }
  }, 1000)
  restoreDiagnosisState()
  await Promise.all([loadClusters(), loadHosts()])
})

onUnmounted(() => {
  if (diagnosisTicker) {
    window.clearInterval(diagnosisTicker)
  }
})
</script>

<style scoped>
.diagnosis-page {
  min-height: 100%;
}

.page-header,
.mode-card,
.form-card,
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

.mode-card {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 14px;
  margin-bottom: 18px;
}

.mode-item {
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  border-radius: 16px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  text-align: left;
  cursor: pointer;
  color: #334155;
}

.mode-item.active {
  border-color: #111827;
  background: #111827;
  color: #fff;
}

.mode-item small {
  display: block;
  margin-top: 4px;
  color: inherit;
  opacity: 0.72;
}

.mode-icon {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.86);
  color: #111827;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.diagnosis-grid {
  display: grid;
  grid-template-columns: 420px 1fr;
  gap: 18px;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.form-card,
.result-card,
.evidence-card {
  padding: 20px;
  min-width: 0;
  overflow: hidden;
}

.form-card :deep(.el-form),
.form-card :deep(.el-form-item),
.form-card :deep(.el-form-item__content) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.form-card :deep(.el-form-item__content) {
  display: block;
  overflow: hidden;
}

.section-title {
  font-weight: 800;
  color: #111827;
  margin-bottom: 16px;
}

.full-width {
  width: 100%;
}

.object-type-grid {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(116px, 1fr));
  gap: 8px;
  padding: 8px;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  max-height: 154px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
}

.object-type-chip {
  width: 100%;
  height: 38px;
  min-width: 0;
  box-sizing: border-box;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #fff;
  color: #334155;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.16s ease;
  overflow: hidden;
  padding: 0 9px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 7px;
  text-align: left;
}

.chip-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  border-radius: 999px;
  background: #cbd5e1;
}

.chip-text {
  display: block;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.object-type-chip:hover {
  background: #fff;
  border-color: #94a3b8;
  color: #111827;
  transform: translateY(-1px);
}

.object-type-chip.active {
  background: #111827;
  border-color: #111827;
  color: #fff;
  box-shadow: 0 8px 16px rgba(17, 24, 39, 0.14);
}

.object-type-chip.active .chip-dot {
  background: #fff;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.18);
}

.host-search {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
}

.host-list {
  height: clamp(460px, calc(100vh - 380px), 700px);
  padding-right: 4px;
}

.host-option {
  width: 100%;
  border: 1px solid #e5e7eb;
  background: #fff;
  border-radius: 14px;
  padding: 12px;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  text-align: left;
}

.host-option.active {
  border-color: #111827;
  box-shadow: 0 10px 22px rgba(17, 24, 39, 0.1);
}

.host-option.diagnosing {
  border-color: #111827;
  background: linear-gradient(180deg, #f8fafc 0%, #ffffff 100%);
  box-shadow: 0 12px 26px rgba(17, 24, 39, 0.12);
}

.host-option.diagnosing .host-avatar {
  background: #111827;
  color: #fff;
}

.host-avatar {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #e2e8f0;
  color: #334155;
  font-weight: 800;
}

.host-avatar.online {
  background: #dcfce7;
  color: #15803d;
}

.host-avatar.offline {
  background: #f1f5f9;
  color: #64748b;
}

.host-main {
  flex: 1;
  min-width: 0;
}

.host-main strong,
.host-main small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-main strong {
  color: #111827;
}

.host-main small {
  margin-top: 3px;
  color: #64748b;
}

.host-badges {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
}

.host-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-top: 14px;
  margin-bottom: 16px;
}

.host-summary div {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  padding: 12px;
}

.host-summary span,
.host-summary strong {
  display: block;
}

.host-summary span {
  color: #64748b;
  font-size: 12px;
}

.host-summary strong {
  margin-top: 4px;
  color: #111827;
  font-size: 18px;
}

.host-focus-form {
  margin-top: 4px;
}

.diagnosis-running-card {
  border: 1px solid #e5e7eb;
  background: linear-gradient(180deg, #f8fafc 0%, #fff 100%);
  border-radius: 16px;
  padding: 16px;
  margin-bottom: 14px;
}

.diagnosis-running-card.running {
  border-color: #111827;
  background: linear-gradient(180deg, #f8fafc 0%, #fff 100%);
  box-shadow: 0 14px 30px rgba(17, 24, 39, 0.1);
}

.diagnosis-running-card.success {
  border-color: #bbf7d0;
}

.diagnosis-running-card.error {
  border-color: #fecaca;
  background: linear-gradient(180deg, #fef2f2 0%, #fff 100%);
}

.running-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.running-icon {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: #111827;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
}

.diagnosis-running-card.running .running-icon .el-icon {
  animation: spin 1s linear infinite;
}

.running-main {
  flex: 1;
  min-width: 0;
}

.running-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
  margin-bottom: 4px;
}

.running-main h3 {
  margin: 0;
  color: #111827;
  font-size: 18px;
}

.running-main p {
  margin: 5px 0 0;
  color: #64748b;
  font-size: 13px;
}

.running-details {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.running-details span {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  border: 1px solid #e2e8f0;
  background: #fff;
  border-radius: 999px;
  padding: 5px 9px;
  color: #334155;
  font-size: 12px;
  font-weight: 700;
}

.running-notice {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid #dbe3ee;
  background: #f8fafc;
  color: #334155;
  font-size: 13px;
  line-height: 1.6;
}

.diagnosis-steps {
  display: grid;
  gap: 8px;
  margin-top: 14px;
}

.diagnosis-step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
  font-size: 13px;
}

.diagnosis-step span {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #cbd5e1;
}

.diagnosis-step.active {
  color: #111827;
  font-weight: 700;
}

.diagnosis-step.active span {
  background: #111827;
}

.running-time {
  margin-top: 12px;
  color: #94a3b8;
  font-size: 12px;
}

.result-meta {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.result-markdown {
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  border-radius: 14px;
  padding: 16px;
  min-width: 0;
  max-height: 560px;
  overflow: auto;
  overflow-wrap: anywhere;
}

.evidence-card {
  margin-top: 18px;
}

.evidence-card pre {
  white-space: pre-wrap;
  line-height: 1.75;
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

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .mode-card,
  .diagnosis-grid {
    grid-template-columns: 1fr;
  }

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

  .host-list {
    height: 420px;
  }
}

@media (max-width: 520px) {
  .object-type-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
