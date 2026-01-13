<template>
  <el-dialog
    v-model="dialogVisible"
    :title="`Pod 详情: ${podData?.metadata?.name || ''}`"
    width="1200px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-if="loading" class="loading-container">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <div v-else-if="podData" class="pod-detail-container">
      <!-- 左侧内容 -->
      <div class="content-area">
        <!-- 基本信息 -->
        <div class="info-section">
          <div class="section-header">
            <span class="section-icon">📋</span>
            <span class="section-title">基本信息</span>
          </div>
          <div class="info-grid">
            <div class="info-item">
              <label>名称</label>
              <span>{{ podData.metadata?.name }}</span>
            </div>
            <div class="info-item">
              <label>命名空间</label>
              <span>{{ podData.metadata?.namespace }}</span>
            </div>
            <div class="info-item">
              <label>状态</label>
              <el-tag :type="getStatusType(podData.status?.phase)" size="small">
                {{ podData.status?.phase }}
              </el-tag>
            </div>
            <div class="info-item">
              <label>创建时间</label>
              <span>{{ formatAge(podData.metadata?.creationTimestamp) }}</span>
            </div>
            <div class="info-item">
              <label>节点</label>
              <span>{{ podData.spec?.nodeName }}</span>
            </div>
            <div class="info-item">
              <label>Pod IP</label>
              <span>{{ podData.status?.podIP || '-' }}</span>
            </div>
            <div class="info-item">
              <label>重启策略</label>
              <span>{{ podData.spec?.restartPolicy }}</span>
            </div>
            <div class="info-item">
              <label>QoS 类</label>
              <span>{{ podData.status?.qosClass || '-' }}</span>
            </div>
          </div>

          <!-- 标签 -->
          <div class="tags-section" v-if="hasLabels">
            <label>标签</label>
            <div class="tags-container">
              <el-tag
                v-for="(value, key) in podData.metadata?.labels"
                :key="key"
                size="small"
                class="tag-item"
              >
                {{ key }}: {{ value }}
              </el-tag>
            </div>
          </div>

          <!-- 注解 -->
          <div class="annotations-section" v-if="hasAnnotations">
            <label>注解</label>
            <div class="annotations-container">
              <div
                v-for="(value, key) in podData.metadata?.annotations"
                :key="key"
                class="annotation-item"
              >
                <span class="annotation-key">{{ key }}:</span>
                <el-tooltip :content="value" placement="top" effect="light" :show-after="500">
                  <span class="annotation-value truncated">{{ value }}</span>
                </el-tooltip>
              </div>
            </div>
          </div>
        </div>

        <!-- 条件 -->
        <div class="conditions-section" v-if="podData.status?.conditions && podData.status.conditions.length > 0">
          <div class="section-header">
            <span class="section-icon">🔍</span>
            <span class="section-title">条件</span>
          </div>
          <el-table :data="podData.status.conditions" size="small" class="conditions-table">
            <el-table-column prop="type" label="类型" width="140" />
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'True' ? 'success' : 'danger'" size="small">
                  {{ row.status || 'Unknown' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="lastTransitionTime" label="更新时间" width="140">
              <template #default="{ row }">
                {{ formatAge(row.lastTransitionTime) }}
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="内容" width="130" />
            <el-table-column prop="message" label="消息" show-overflow-tooltip />
          </el-table>
        </div>

        <!-- Tab 内容 -->
        <el-tabs v-model="activeTab" class="detail-tabs">
          <!-- 容器 -->
          <el-tab-pane label="容器" name="containers">
            <div v-if="podData.spec?.containers && podData.spec.containers.length > 0" class="containers-list">
              <div
                v-for="(container, index) in podData.spec.containers"
                :key="index"
                class="container-item"
              >
                <div class="container-header">
                  <div class="container-name">{{ container.name }}</div>
                  <el-tag size="small" type="info" effect="plain">{{ container.image }}</el-tag>
                </div>

                <div class="container-grid">
                  <div class="container-section">
                    <div class="section-label">基本配置</div>
                    <div class="info-row">
                      <label>镜像拉取策略:</label>
                      <span>{{ container.imagePullPolicy }}</span>
                    </div>
                    <div class="info-row" v-if="container.command && container.command.length > 0">
                      <label>命令:</label>
                      <code>{{ container.command.join(' ') }}</code>
                    </div>
                    <div class="info-row" v-if="container.args && container.args.length > 0">
                      <label>参数:</label>
                      <code>{{ container.args.join(' ') }}</code>
                    </div>
                    <div class="info-row" v-if="container.ports && container.ports.length > 0">
                      <label>端口:</label>
                      <div class="ports-list">
                        <span v-for="(port, i) in container.ports" :key="i" class="port-tag">
                          {{ port.containerPort }}/{{ port.protocol || 'TCP' }}
                        </span>
                      </div>
                    </div>
                  </div>

                  <div class="container-section">
                    <div class="section-label">资源限制</div>
                    <div class="info-row">
                      <label>CPU:</label>
                      <span>{{ container.resources?.limits?.cpu || '-' }}</span>
                    </div>
                    <div class="info-row">
                      <label>内存:</label>
                      <span>{{ container.resources?.limits?.memory || '-' }}</span>
                    </div>
                  </div>

                  <div class="container-section">
                    <div class="section-label">资源请求</div>
                    <div class="info-row">
                      <label>CPU:</label>
                      <span>{{ container.resources?.requests?.cpu || '-' }}</span>
                    </div>
                    <div class="info-row">
                      <label>内存:</label>
                      <span>{{ container.resources?.requests?.memory || '-' }}</span>
                    </div>
                  </div>

                  <div class="container-section" v-if="podData.status?.containerStatuses?.[index]">
                    <div class="section-label">运行状态</div>
                    <div class="info-row">
                      <label>状态:</label>
                      <el-tag
                        :type="getContainerStateType(podData.status.containerStatuses[index].state)"
                        size="small"
                      >
                        {{ getContainerState(podData.status.containerStatuses[index]) }}
                      </el-tag>
                    </div>
                    <div class="info-row">
                      <label>重启次数:</label>
                      <span>{{ podData.status.containerStatuses[index].restartCount }}</span>
                    </div>
                    <div class="info-row" v-if="podData.status.containerStatuses[index].state?.running">
                      <label>启动时间:</label>
                      <span>{{ formatAge(podData.status.containerStatuses[index].state.running.startedAt) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无容器" />
          </el-tab-pane>

          <!-- 初始化容器 -->
          <el-tab-pane label="初始化容器" name="initContainers">
            <div v-if="podData.spec?.initContainers && podData.spec.initContainers.length > 0" class="containers-list">
              <div
                v-for="(container, index) in podData.spec.initContainers"
                :key="index"
                class="container-item"
              >
                <div class="container-header">
                  <div class="container-name">{{ container.name }}</div>
                  <el-tag size="small" type="info" effect="plain">{{ container.image }}</el-tag>
                </div>

                <div class="container-grid">
                  <div class="container-section">
                    <div class="section-label">基本配置</div>
                    <div class="info-row">
                      <label>镜像拉取策略:</label>
                      <span>{{ container.imagePullPolicy }}</span>
                    </div>
                    <div class="info-row" v-if="container.command && container.command.length > 0">
                      <label>命令:</label>
                      <code>{{ container.command.join(' ') }}</code>
                    </div>
                    <div class="info-row" v-if="container.args && container.args.length > 0">
                      <label>参数:</label>
                      <code>{{ container.args.join(' ') }}</code>
                    </div>
                  </div>

                  <div class="container-section" v-if="podData.status?.initContainerStatuses?.[index]">
                    <div class="section-label">运行状态</div>
                    <div class="info-row">
                      <label>状态:</label>
                      <el-tag
                        :type="getContainerStateType(podData.status.initContainerStatuses[index].state)"
                        size="small"
                      >
                        {{ getContainerState(podData.status.initContainerStatuses[index]) }}
                      </el-tag>
                    </div>
                    <div class="info-row">
                      <label>重启次数:</label>
                      <span>{{ podData.status.initContainerStatuses[index].restartCount }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无初始化容器" />
          </el-tab-pane>

          <!-- 事件 -->
          <el-tab-pane label="事件" name="events">
            <div v-if="events.length > 0">
              <el-timeline>
                <el-timeline-item
                  v-for="(event, index) in events"
                  :key="index"
                  :timestamp="formatAge(event.lastTimestamp)"
                  placement="top"
                >
                  <div class="event-item">
                    <div class="event-header">
                      <el-tag :type="getEventType(event.type)" size="small">
                        {{ event.type }}
                      </el-tag>
                      <span class="event-reason">{{ event.reason }}</span>
                    </div>
                    <div class="event-message">{{ event.message }}</div>
                    <div class="event-meta">
                      <span>来源: {{ event.source?.component || '-' }}</span>
                      <span v-if="event.count && event.count > 0">次数: {{ event.count }}</span>
                    </div>
                  </div>
                </el-timeline-item>
              </el-timeline>
            </div>
            <el-empty v-else description="暂无事件" />
          </el-tab-pane>

          <!-- 存储 -->
          <el-tab-pane label="存储" name="volumes">
            <div v-if="podData.spec?.volumes && podData.spec.volumes.length > 0">
              <el-table :data="podData.spec.volumes" size="small" class="volumes-table">
                <el-table-column prop="name" label="名称" width="140" />
                <el-table-column label="类型" width="100">
                  <template #default="{ row }">
                    <el-tag size="small" effect="plain">
                      {{ getVolumeType(row) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="配置" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ getVolumeConfig(row) }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无存储卷" />
          </el-tab-pane>

          <!-- 创建者 -->
          <el-tab-pane label="创建者" name="owner">
            <div v-if="podData.metadata?.ownerReferences && podData.metadata.ownerReferences.length > 0" class="owner-list">
              <div
                v-for="(owner, index) in podData.metadata.ownerReferences"
                :key="index"
                class="owner-item"
              >
                <div class="info-row">
                  <label>类型:</label>
                  <el-tag size="small" type="info">{{ owner.kind }}</el-tag>
                </div>
                <div class="info-row">
                  <label>名称:</label>
                  <span>{{ owner.name }}</span>
                </div>
                <div class="info-row">
                  <label>控制器:</label>
                  <el-tag :type="owner.controller ? 'success' : 'info'" size="small">
                    {{ owner.controller ? '是' : '否' }}
                  </el-tag>
                </div>
              </div>
            </div>
            <el-empty v-else description="无创建者信息" />
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <el-empty v-else description="暂无数据" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import axios from 'axios'
import { formatAge } from '@/utils/format'

interface PodData {
  metadata?: {
    name?: string
    namespace?: string
    creationTimestamp?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
    ownerReferences?: Array<{
      kind?: string
      name?: string
      controller?: boolean
    }>
  }
  spec?: {
    nodeName?: string
    restartPolicy?: string
    containers?: Array<{
      name?: string
      image?: string
      imagePullPolicy?: string
      command?: string[]
      args?: string[]
      ports?: Array<{ containerPort?: number; protocol?: string }>
      resources?: {
        limits?: { cpu?: string; memory?: string }
        requests?: { cpu?: string; memory?: string }
      }
    }>
    initContainers?: Array<{
      name?: string
      image?: string
      imagePullPolicy?: string
      command?: string[]
      args?: string[]
    }>
    volumes?: Array<any>
  }
  status?: {
    phase?: string
    podIP?: string
    qosClass?: string
    conditions?: Array<{
      type?: string
      status?: string
      lastTransitionTime?: string
      reason?: string
      message?: string
    }>
    containerStatuses?: Array<any>
    initContainerStatuses?: Array<any>
  }
}

interface Event {
  type?: string
  reason?: string
  message?: string
  lastTimestamp?: string
  source?: { component?: string }
  count?: number
}

const props = defineProps<{
  visible: boolean
  clusterId: number | string
  namespace: string
  podName: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const loading = ref(false)
const podData = ref<PodData | null>(null)
const events = ref<Event[]>([])
const activeTab = ref('containers')

const hasLabels = computed(() => {
  return podData.value?.metadata?.labels && Object.keys(podData.value.metadata.labels).length > 0
})

const hasAnnotations = computed(() => {
  return podData.value?.metadata?.annotations && Object.keys(podData.value.metadata.annotations).length > 0
})

const getStatusType = (phase?: string) => {
  const typeMap: Record<string, any> = {
    Running: 'success',
    Succeeded: 'success',
    Pending: 'warning',
    Failed: 'danger',
    Unknown: 'info'
  }
  return typeMap[phase || ''] || 'info'
}

const getContainerState = (status: any) => {
  if (status.state?.running) return 'Running'
  if (status.state?.waiting) return `Waiting: ${status.state.waiting.reason || ''}`
  if (status.state?.terminated) return `Terminated: ${status.state.terminated.reason || ''}`
  return 'Unknown'
}

const getContainerStateType = (state: any) => {
  if (state.running) return 'success'
  if (state.terminated?.exitCode === 0) return 'success'
  if (state.waiting) return 'warning'
  if (state.terminated) return 'danger'
  return 'info'
}

const getEventType = (type?: string) => {
  return type === 'Warning' ? 'danger' : 'success'
}

const getVolumeType = (volume: any) => {
  if (volume.persistentVolumeClaim) return 'PVC'
  if (volume.configMap) return 'ConfigMap'
  if (volume.secret) return 'Secret'
  if (volume.emptyDir) return 'EmptyDir'
  if (volume.hostPath) return 'HostPath'
  if (volume.nfs) return 'NFS'
  return 'Unknown'
}

const getVolumeConfig = (volume: any) => {
  if (volume.persistentVolumeClaim) {
    return `ClaimName: ${volume.persistentVolumeClaim.claimName}, ReadOnly: ${volume.persistentVolumeClaim.readOnly || false}`
  }
  if (volume.configMap) {
    return `Name: ${volume.configMap.name}`
  }
  if (volume.secret) {
    return `SecretName: ${volume.secret.secretName}`
  }
  if (volume.emptyDir) {
    return `Medium: ${volume.emptyDir.medium || 'default'}, SizeLimit: ${volume.emptyDir.sizeLimit || 'unlimited'}`
  }
  if (volume.hostPath) {
    return `Path: ${volume.hostPath.path}, Type: ${volume.hostPath.type || ''}`
  }
  if (volume.nfs) {
    return `Server: ${volume.nfs.server}, Path: ${volume.nfs.path}`
  }
  return JSON.stringify(volume)
}

const loadPodDetail = async () => {
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/pods/${props.namespace}/${props.podName}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    podData.value = response.data
  } catch (error: any) {
    console.error('获取 Pod 详情失败:', error)
    ElMessage.error(error.response?.data?.message || '获取 Pod 详情失败')
  } finally {
    loading.value = false
  }
}

const loadPodEvents = async () => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/pods/${props.namespace}/${props.podName}/events`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    events.value = response.data.events || []
  } catch (error: any) {
    console.error('获取 Pod 事件失败:', error)
  }
}

const handleClose = () => {
  dialogVisible.value = false
  podData.value = null
  events.value = []
  activeTab.value = 'containers'
}

// 监听 visible 变化，加载数据
watch(() => props.visible, (newVal) => {
  if (newVal) {
    loadPodDetail()
    loadPodEvents()
  }
})
</script>

<style scoped>
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  gap: 16px;
  font-size: 14px;
  color: #999;
}

.loading-container .el-icon {
  font-size: 32px;
  color: #d4af37;
}

.pod-detail-container {
  display: flex;
  flex-direction: column;
  height: 70vh;
}

.content-area {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 0;
  border-bottom: 2px solid #d4af37;
  margin-bottom: 20px;
}

.section-icon {
  font-size: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.info-section {
  margin-bottom: 24px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px 20px;
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.info-item label {
  font-size: 12px;
  color: #999;
  font-weight: 500;
}

.info-item span {
  font-size: 14px;
  color: #333;
}

.tags-section,
.annotations-section {
  margin-top: 16px;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
}

.tags-section label,
.annotations-section label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #333;
  margin-bottom: 10px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.annotations-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.annotation-item {
  display: flex;
  gap: 8px;
  font-size: 12px;
  padding: 6px 10px;
  background: #ffffff;
  border-radius: 4px;
  align-items: center;
}

.annotation-key {
  font-weight: 600;
  color: #666;
  min-width: 150px;
  flex-shrink: 0;
}

.annotation-value {
  color: #333;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.annotation-value.truncated {
  max-width: 300px;
  cursor: help;
}

.conditions-section {
  margin-bottom: 24px;
}

.conditions-table {
  width: 100%;
  border-radius: 8px;
  overflow: hidden;
}

.conditions-table :deep(.el-table__header) {
  background: #fafafa;
}

.conditions-table :deep(.el-table__body tr:hover) {
  background: #f9f9f9;
}

.conditions-table :deep(.el-table__body td) {
  padding: 8px 0;
}

.detail-tabs {
  margin-top: 20px;
}

.detail-tabs :deep(.el-tabs__header) {
  background: #fafafa;
  border-radius: 8px 8px 0 0;
  padding: 0 16px;
}

.detail-tabs :deep(.el-tabs__content) {
  padding: 20px 0;
}

.containers-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.container-item {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 20px;
  background: #ffffff;
}

.container-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.container-name {
  font-size: 15px;
  font-weight: 600;
  color: #333;
}

.container-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.container-section {
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
}

.section-label {
  font-size: 13px;
  font-weight: 600;
  color: #666;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed #e0e0e0;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
  font-size: 13px;
}

.info-row:last-child {
  margin-bottom: 0;
}

.info-row label {
  font-weight: 600;
  color: #666;
  min-width: 80px;
  flex-shrink: 0;
}

.info-row code {
  padding: 3px 8px;
  background: #f5f5f5;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #d4af37;
  word-break: break-all;
}

.ports-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.port-tag {
  display: inline-block;
  padding: 2px 8px;
  background: #e8f4fd;
  border: 1px solid #b3d8ff;
  border-radius: 4px;
  font-size: 12px;
  color: #409eff;
  font-family: 'Courier New', monospace;
}

.event-item {
  padding: 12px;
  background: #ffffff;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.event-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.event-reason {
  font-weight: 600;
  color: #333;
}

.event-message {
  font-size: 13px;
  color: #666;
  margin-bottom: 8px;
}

.event-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #999;
}

.volumes-table {
  width: 100%;
  border-radius: 8px;
  overflow: hidden;
}

.volumes-table :deep(.el-table__header) {
  background: #fafafa;
}

.owner-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.owner-item {
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}
</style>
