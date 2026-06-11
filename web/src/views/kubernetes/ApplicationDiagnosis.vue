<template>
  <div class="app-diagnosis-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Cpu /></el-icon>
        </div>
        <div>
          <h2 class="page-title">应用诊断</h2>
          <p class="page-subtitle">基于 Arthas 的 Java 应用诊断工具</p>
        </div>
      </div>
    </div>

    <!-- 选择器栏 -->
    <div class="selector-bar">
      <div class="selector-inputs">
        <div class="selector-item">
          <span class="selector-label">集群</span>
          <el-select
            v-model="selectedCluster"
            placeholder="请选择集群"
            @change="handleClusterChange"
            class="selector-select"
          >
            <el-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :label="cluster.name"
              :value="cluster.id"
            />
          </el-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">命名空间</span>
          <el-select
            v-model="selectedNamespace"
            placeholder="请选择命名空间"
            @change="handleNamespaceChange"
            :disabled="!selectedCluster"
            class="selector-select"
          >
            <el-option
              v-for="ns in namespaces"
              :key="ns.name"
              :label="ns.name"
              :value="ns.name"
            />
          </el-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">Pod</span>
          <el-select
            v-model="selectedPod"
            placeholder="请选择Pod"
            @change="handlePodChange"
            :disabled="!selectedNamespace"
            class="selector-select"
          >
            <el-option
              v-for="pod in pods"
              :key="pod.name"
              :label="formatPodOptionLabel(pod)"
              :value="pod.name"
              :disabled="!isPodConnectable(pod)"
            />
          </el-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">容器</span>
          <el-select
            v-model="selectedContainer"
            placeholder="请选择容器"
            @change="handleContainerChange"
            :disabled="!selectedPod"
            class="selector-select"
          >
            <el-option
              v-for="container in containers"
              :key="container"
              :label="container"
              :value="container"
            />
          </el-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">进程</span>
          <el-select
            v-model="selectedProcess"
            placeholder="请选择进程"
            :disabled="!selectedContainer"
            :loading="loadingProcesses"
            class="selector-select process-select"
            @change="handleProcessChange"
          >
            <el-option
              v-for="proc in processes"
              :key="proc.pid"
              :label="formatProcessLabel(proc)"
              :value="proc.pid"
            />
          </el-select>
        </div>
      </div>

      <div class="selector-actions">
        <el-button
          type="primary"
          :icon="isCurrentTargetAttached ? Link : Download"
          @click="handleAttach"
          :disabled="!selectedProcess || isCurrentTargetAttached"
          :loading="attaching"
          class="attach-btn"
        >
          {{ isCurrentTargetAttached ? '已连接' : '连接' }}
        </el-button>
      </div>
    </div>

    <!-- Tab 内容区 -->
    <div class="content-wrapper">
      <el-tabs v-model="activeTab" class="diagnosis-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="控制面板" name="dashboard">
          <DashboardPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
            @connection-error="handleDiagnosisConnectionError"
          />
        </el-tab-pane>

        <el-tab-pane label="线程清单" name="threads">
          <ThreadListPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="JVM信息" name="jvm">
          <JvmInfoPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="系统信息" name="sysinfo">
          <SystemInfoPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="线程堆栈" name="stack">
          <ThreadStackPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="火焰图" name="flame">
          <FlameGraphPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="方法追踪" name="trace">
          <MethodTracePanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="方法监测" name="watch">
          <MethodWatchPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>

        <el-tab-pane label="方法监控" name="monitor">
          <MethodMonitorPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="isCurrentTargetAttached"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Download, Link, Cpu } from '@element-plus/icons-vue'
import { getClusterList, getNamespaces, getPods, getPodDetail, type Cluster, type NamespaceInfo, type PodInfo } from '@/api/kubernetes'
import { listJavaProcesses, checkArthasInstalled, installArthas, executeArthasCommand, type ArthasCheckResult, type JavaProcess } from '@/api/arthas'

// 导入子组件
import DashboardPanel from './diagnosis-components/DashboardPanel.vue'
import ThreadListPanel from './diagnosis-components/ThreadListPanel.vue'
import JvmInfoPanel from './diagnosis-components/JvmInfoPanel.vue'
import SystemInfoPanel from './diagnosis-components/SystemInfoPanel.vue'
import ThreadStackPanel from './diagnosis-components/ThreadStackPanel.vue'
import FlameGraphPanel from './diagnosis-components/FlameGraphPanel.vue'
import MethodTracePanel from './diagnosis-components/MethodTracePanel.vue'
import MethodWatchPanel from './diagnosis-components/MethodWatchPanel.vue'
import MethodMonitorPanel from './diagnosis-components/MethodMonitorPanel.vue'

const route = useRoute()

// 状态存储的 key
const STORAGE_KEY = 'arthas_diagnosis_state'

interface DiagnosisState {
  clusterId?: number | string | null
  namespace?: string
  pod?: string
  container?: string
  processId?: string
  activeTab?: string
  attached?: boolean
}

// 选择器数据
const clusters = ref<Cluster[]>([])
const namespaces = ref<NamespaceInfo[]>([])
const pods = ref<PodInfo[]>([])
const containers = ref<string[]>([])
const processes = ref<JavaProcess[]>([])

// 选中的值
const selectedCluster = ref<number | null>(null)
const selectedNamespace = ref<string>('')
const selectedPod = ref<string>('')
const selectedContainer = ref<string>('')
const selectedProcess = ref<string>('')

// 状态
const activeTab = ref('dashboard')
const attaching = ref(false)
const attached = ref(false)
const attachedTargetKey = ref('')
const loadingProcesses = ref(false)

const unwrapData = <T>(res: unknown): T => {
  return ((res as any)?.data ?? res) as T
}

const unwrapArray = <T>(res: unknown): T[] => {
  const data = unwrapData<unknown>(res)
  return Array.isArray(data) ? data as T[] : []
}

const currentTargetKey = computed(() => {
  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value || !selectedContainer.value || !selectedProcess.value) {
    return ''
  }
  return [
    selectedCluster.value,
    selectedNamespace.value,
    selectedPod.value,
    selectedContainer.value,
    selectedProcess.value
  ].join('|')
})

const isCurrentTargetAttached = computed(() => {
  return attached.value && attachedTargetKey.value === currentTargetKey.value
})

const resetAttachment = () => {
  attached.value = false
  attachedTargetKey.value = ''
}

// 保存状态到 localStorage
const saveState = () => {
  const state = {
    clusterId: selectedCluster.value,
    namespace: selectedNamespace.value,
    pod: selectedPod.value,
    container: selectedContainer.value,
    processId: selectedProcess.value,
    activeTab: activeTab.value,
    attached: false
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

const getQueryString = (value: unknown) => {
  if (Array.isArray(value)) {
    return value[0] ? String(value[0]) : ''
  }
  return value ? String(value) : ''
}

const getRouteState = (): DiagnosisState => {
  const query = route.query
  const clusterId = getQueryString(query.clusterId || query.cluster)
  const namespace = getQueryString(query.namespace || query.ns)
  const pod = getQueryString(query.pod || query.podName)
  const container = getQueryString(query.container)
  const processId = getQueryString(query.processId || query.pid)
  const activeTab = getQueryString(query.tab || query.activeTab)

  return {
    ...(clusterId ? { clusterId } : {}),
    ...(namespace ? { namespace } : {}),
    ...(pod ? { pod } : {}),
    ...(container ? { container } : {}),
    ...(processId ? { processId } : {}),
    ...(activeTab ? { activeTab } : {})
  }
}

const getInitialState = (): DiagnosisState => {
  let savedState: DiagnosisState = {}
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved) {
    try {
      savedState = JSON.parse(saved)
    } catch (e) {
      savedState = {}
    }
  }

  return {
    ...savedState,
    ...getRouteState()
  }
}

const normalizeClusterId = (clusterId: DiagnosisState['clusterId']) => {
  if (clusterId === null || clusterId === undefined || clusterId === '') {
    return null
  }
  const id = Number(clusterId)
  return Number.isFinite(id) ? id : null
}

const normalizeContainerNames = (items: unknown) => {
  if (!Array.isArray(items)) {
    return []
  }

  return items
    .map((item: any) => typeof item === 'string' ? item : item?.name)
    .filter((name): name is string => Boolean(name))
}

const getSelectedPodInfo = () => {
  return pods.value.find(pod => pod.name === selectedPod.value)
}

const isPodConnectable = (pod: PodInfo) => {
  return !pod.status || pod.status === 'Running'
}

const formatPodOptionLabel = (pod: PodInfo) => {
  return isPodConnectable(pod) ? pod.name : `${pod.name} (${pod.status})`
}

const formatProcessLabel = (proc: JavaProcess) => {
  return `${proc.pid} - ${proc.mainClass || proc.commandLine || 'Java进程'}`
}

const getArthasOutput = (res: unknown) => {
  const data = unwrapData<unknown>(res)
  return typeof data === 'string' ? data : JSON.stringify(data || '')
}

const getArthasConnectError = (output: string) => {
  if (!output) {
    return ''
  }
  const patterns = [
    { regex: /Connection refused/i, message: 'Arthas telnet 连接被拒绝，请稍后重试' },
    { regex: /Connect.*error/i, message: 'Arthas telnet 连接失败，请稍后重试' },
    { regex: /Failed to execute Arthas command/i, message: 'Arthas 命令执行失败' },
    { regex: /Unable to attach|attach not supported|not support attach/i, message: '当前 JVM 不支持 Arthas attach' },
    { regex: /No process|Can not find|process not found/i, message: '目标 Java 进程不存在，请重新选择进程' }
  ]
  return patterns.find(item => item.regex.test(output))?.message || ''
}

// 从 localStorage 恢复状态
const restoreState = async () => {
  const state = getInitialState()

  try {
    // 恢复 Tab 状态
    if (state.activeTab) {
      activeTab.value = state.activeTab
    }

    // 恢复选择状态（需要按顺序恢复，因为有依赖关系）
    const clusterId = normalizeClusterId(state.clusterId)
    if (clusterId && clusters.value.some(cluster => Number(cluster.id) === clusterId)) {
      selectedCluster.value = clusterId
      await loadNamespaces()

      if (state.namespace && namespaces.value.some(ns => ns.name === state.namespace)) {
        selectedNamespace.value = state.namespace
        await loadPods()

        if (state.pod && pods.value.some(pod => pod.name === state.pod && isPodConnectable(pod))) {
          selectedPod.value = state.pod
          await loadContainers()

          if (state.container && containers.value.includes(state.container)) {
            selectedContainer.value = state.container
            await loadProcesses()

            if (state.processId && processes.value.some(proc => proc.pid === state.processId)) {
              selectedProcess.value = state.processId
            }
          }
        }
      }
    }
    resetAttachment()
    saveState()
  } catch (e) {
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const res = await getClusterList()
    clusters.value = unwrapArray<Cluster>(res)
  } catch (error) {
  }
}

// 加载命名空间
const loadNamespaces = async () => {
  if (!selectedCluster.value) return
  try {
    const res = await getNamespaces(selectedCluster.value)
    namespaces.value = unwrapArray<NamespaceInfo>(res)
  } catch (error) {
  }
}

// 加载 Pods
const loadPods = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) return
  try {
    const res = await getPods(selectedCluster.value, selectedNamespace.value)
    pods.value = unwrapArray<PodInfo>(res)
  } catch (error) {
    pods.value = []
  }
}

// 加载容器
const loadContainers = async () => {
  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value) return
  containers.value = []

  const selectedPodInfo = getSelectedPodInfo()
  const summaryContainers = normalizeContainerNames(selectedPodInfo?.containers)
  if (summaryContainers.length > 0) {
    containers.value = summaryContainers
    return
  }

  try {
    const res = await getPodDetail(selectedCluster.value, selectedNamespace.value, selectedPod.value)
    const podDetail = unwrapData<any>(res)
    const containerList = normalizeContainerNames(podDetail?.spec?.containers)
    const initContainers = normalizeContainerNames(podDetail?.spec?.initContainers)
    const ephemeralContainers = normalizeContainerNames(podDetail?.spec?.ephemeralContainers)
    containers.value = [...containerList, ...initContainers, ...ephemeralContainers]
    if (containers.value.length === 0) {
      ElMessage.warning('该 Pod 未获取到可诊断容器')
    }
  } catch (error) {
    containers.value = []
    ElMessage.error('获取 Pod 容器失败')
  }
}

// 加载进程
const loadProcesses = async () => {
  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value || !selectedContainer.value) return

  loadingProcesses.value = true
  try {
    const res = await listJavaProcesses({
      clusterId: selectedCluster.value,
      namespace: selectedNamespace.value,
      pod: selectedPod.value,
      container: selectedContainer.value
    })
    processes.value = unwrapArray<JavaProcess>(res)
    if (!selectedProcess.value && processes.value.length === 1) {
      const onlyProcess = processes.value[0]
      if (onlyProcess) {
        selectedProcess.value = onlyProcess.pid
      }
    }
    if (processes.value.length === 0) {
      ElMessage.warning('未检测到Java进程，请确保容器中有运行的Java应用')
    }
  } catch (error: any) {
    processes.value = []
    if (error.message && !error.message.includes('exit code')) {
      ElMessage.error('获取Java进程失败: ' + (error.message || '未知错误'))
    } else {
      ElMessage.warning('该容器未检测到Java环境')
    }
  } finally {
    loadingProcesses.value = false
  }
}

// 集群变更
const handleClusterChange = async () => {
  selectedNamespace.value = ''
  selectedPod.value = ''
  selectedContainer.value = ''
  selectedProcess.value = ''
  namespaces.value = []
  pods.value = []
  containers.value = []
  processes.value = []
  resetAttachment()

  await loadNamespaces()
  saveState()
}

// 命名空间变更
const handleNamespaceChange = async () => {
  selectedPod.value = ''
  selectedContainer.value = ''
  selectedProcess.value = ''
  pods.value = []
  containers.value = []
  processes.value = []
  resetAttachment()

  await loadPods()
  saveState()
}

// Pod变更
const handlePodChange = async () => {
  selectedContainer.value = ''
  selectedProcess.value = ''
  containers.value = []
  processes.value = []
  resetAttachment()

  const podInfo = getSelectedPodInfo()
  if (podInfo && !isPodConnectable(podInfo)) {
    selectedPod.value = ''
    ElMessage.warning('只能连接 Running 状态的 Pod')
    saveState()
    return
  }

  await loadContainers()
  saveState()
}

// 容器变更
const handleContainerChange = async () => {
  selectedProcess.value = ''
  processes.value = []
  resetAttachment()

  await loadProcesses()
  saveState()
}

// 进程变更
const handleProcessChange = () => {
  resetAttachment()
  saveState()
}

const handleDiagnosisConnectionError = (message?: string) => {
  resetAttachment()
  saveState()
  if (message) {
    ElMessage.error(message)
  }
}

// Tab 变更
const handleTabChange = () => {
  saveState()
}

// 安装/连接 Arthas
const handleAttach = async () => {
  if (!selectedProcess.value) {
    ElMessage.warning('请先选择要诊断的进程')
    return
  }

  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value || !selectedContainer.value) {
    ElMessage.warning('请先选择集群、命名空间、Pod和容器')
    return
  }

  if (isCurrentTargetAttached.value) {
    ElMessage.info('当前进程已连接')
    return
  }

  attaching.value = true
  try {
    // 先检查Arthas是否已安装
    const checkRes = await checkArthasInstalled({
      clusterId: selectedCluster.value,
      namespace: selectedNamespace.value,
      pod: selectedPod.value,
      container: selectedContainer.value
    })

    const checkData = unwrapData<ArthasCheckResult>(checkRes)

    if (!checkData?.hasJava) {
      ElMessage.error('容器中未检测到Java环境，无法使用Arthas诊断')
      return
    }

    if (!checkData?.hasArthas) {
      ElMessage.info('正在安装Arthas...')
      await installArthas({
        clusterId: selectedCluster.value,
        namespace: selectedNamespace.value,
        pod: selectedPod.value,
        container: selectedContainer.value
      })
    }

    ElMessage.info('正在验证 Arthas 连接...')
    const verifyRes = await executeArthasCommand({
      clusterId: selectedCluster.value,
      namespace: selectedNamespace.value,
      pod: selectedPod.value,
      container: selectedContainer.value,
      processId: selectedProcess.value,
      command: 'thread -n 1'
    })
    const verifyOutput = getArthasOutput(verifyRes)
    const connectError = getArthasConnectError(verifyOutput)
    if (connectError) {
      throw new Error(connectError)
    }

    attached.value = true
    attachedTargetKey.value = currentTargetKey.value
    saveState()
    ElMessage.success('连接成功')
  } catch (error: any) {
    resetAttachment()
    saveState()
    ElMessage.error('连接失败: ' + (error.message || '未知错误'))
  } finally {
    attaching.value = false
  }
}

onMounted(async () => {
  await loadClusters()
  await restoreState()
})
</script>

<style scoped>
.app-diagnosis-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

/* 选择器栏 */
.selector-bar {
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.selector-inputs {
  display: flex;
  gap: 16px;
  flex: 1;
  flex-wrap: wrap;
  align-items: center;
}

.selector-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.selector-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
  font-weight: 500;
}

.selector-select {
  width: 150px;
}

.process-select {
  width: 200px;
}

.selector-actions {
  display: flex;
  gap: 10px;
}

.attach-btn {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-color: #d4af37;
  color: #d4af37;
}

.attach-btn:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #333 100%);
  border-color: #e5c158;
  color: #e5c158;
}

.attach-btn:disabled {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #c0c4cc;
}

/* 选择器样式 */
.selector-bar :deep(.el-select .el-input__wrapper) {
  border-radius: 6px;
  border: 1px solid #dcdfe6;
  box-shadow: none;
  transition: all 0.3s ease;
  background-color: #fff;
}

.selector-bar :deep(.el-select .el-input__wrapper:hover) {
  border-color: #d4af37;
}

.selector-bar :deep(.el-select .el-input.is-focus .el-input__wrapper) {
  border-color: #d4af37;
  box-shadow: 0 0 0 2px rgba(212, 175, 55, 0.15);
}

/* 内容区域 */
.content-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

/* Tab 样式 */
.diagnosis-tabs {
  padding: 0;
}

:deep(.diagnosis-tabs .el-tabs__header) {
  margin: 0;
  padding: 0 20px;
  background: #fafbfc;
  border-bottom: 1px solid #e4e7eb;
}

:deep(.diagnosis-tabs .el-tabs__nav-wrap::after) {
  display: none;
}

:deep(.diagnosis-tabs .el-tabs__item) {
  padding: 0 20px;
  height: 48px;
  line-height: 48px;
  font-size: 13px;
  color: #606266;
  font-weight: 500;
  transition: all 0.2s ease;
}

:deep(.diagnosis-tabs .el-tabs__item:hover) {
  color: #d4af37;
}

:deep(.diagnosis-tabs .el-tabs__item.is-active) {
  color: #d4af37;
  font-weight: 600;
}

:deep(.diagnosis-tabs .el-tabs__active-bar) {
  background-color: #d4af37;
  height: 3px;
}

:deep(.diagnosis-tabs .el-tabs__content) {
  padding: 20px;
}

/* 响应式布局 */
@media (max-width: 1200px) {
  .selector-inputs {
    gap: 12px;
  }

  .selector-select {
    width: 130px;
  }

  .process-select {
    width: 180px;
  }
}

@media (max-width: 992px) {
  .selector-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .selector-inputs {
    width: 100%;
  }

  .selector-actions {
    width: 100%;
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .selector-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    width: calc(50% - 8px);
  }

  .selector-select,
  .process-select {
    width: 100%;
  }

  :deep(.diagnosis-tabs .el-tabs__item) {
    padding: 0 12px;
    font-size: 12px;
  }
}
</style>
