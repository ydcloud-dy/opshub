<template>
  <div class="container-config">
    <!-- 标准容器 -->
    <div class="container-section">
      <div class="section-header">
        <span class="section-title">标准容器 (Containers)</span>
        <el-button class="section-add-btn" type="primary" :icon="Plus" size="small" @click="addContainer('containers')">添加容器</el-button>
      </div>
      <div class="container-list">
        <el-collapse v-model="activeContainers" accordion>
          <el-collapse-item v-for="(container, index) in containers" :key="'container-'+index" :name="index">
            <template #title>
              <div class="container-title">
                <span class="container-icon-badge">
                  <el-icon><Box /></el-icon>
                </span>
                <span class="container-name">{{ container.name || '未命名容器' }}</span>
                <el-tag size="small" type="success">{{ container.image || '无镜像' }}</el-tag>
                <el-button
                  type="danger"
                  plain
                  :icon="Delete"
                  size="small"
                  @click.stop="removeContainer('containers', index)"
                  class="remove-btn"
                  circle
                  title="删除容器"
                  aria-label="删除容器"
                />
              </div>
            </template>
            <div class="container-detail">
              <el-tabs :model-value="getContainerActiveTab('containers', index)" @tab-change="(tab) => setContainerActiveTab('containers', index, tab as string)">
                <el-tab-pane label="基础配置" name="basic">
                  <ContainerBasicInfo :container="container" @update="updateContainer('containers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="运行命令" name="command">
                  <ContainerCommand :container="container" @update="updateContainer('containers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="环境变量" name="env">
                  <EnvConfig
                    :envs="container.env || []"
                    :configmapList="configMaps"
                    :secretList="secrets"
                    @update="updateContainerEnv('containers', index, $event)"
                  />
                </el-tab-pane>
                <el-tab-pane label="健康检测" name="health">
                  <HealthCheck
                    :livenessProbe="container.livenessProbe"
                    :readinessProbe="container.readinessProbe"
                    :startupProbe="container.startupProbe"
                    @updateLiveness="updateContainerProbe('containers', index, 'livenessProbe', $event)"
                    @updateReadiness="updateContainerProbe('containers', index, 'readinessProbe', $event)"
                    @updateStartup="updateContainerProbe('containers', index, 'startupProbe', $event)"
                  />
                </el-tab-pane>
                <el-tab-pane label="资源配置" name="resources">
                  <ResourceConfig :resources="container.resources || {}" @update="updateContainerResources('containers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="端口配置" name="ports">
                  <PortConfig :ports="container.ports || []" @update="updateContainerPorts('containers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="存储挂载" name="volumes">
                  <VolumeMounts :volumeMounts="container.volumeMounts || []" :volumes="volumes" @update="updateContainerVolumeMounts('containers', index, $event)" />
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-collapse-item>
        </el-collapse>
        <el-empty v-if="containers.length === 0" description="暂无标准容器" :image-size="60" />
      </div>
    </div>

    <!-- 初始化容器 -->
    <div class="container-section">
      <div class="section-header">
        <span class="section-title">初始化容器 (Init Containers)</span>
        <el-button class="section-add-btn" type="primary" :icon="Plus" size="small" @click="addContainer('initContainers')">添加初始化容器</el-button>
      </div>
      <div class="container-list">
        <el-collapse v-model="activeInitContainers" accordion>
          <el-collapse-item v-for="(container, index) in initContainers" :key="'init-container-'+index" :name="index">
            <template #title>
              <div class="container-title">
                <span class="container-icon-badge init">
                  <el-icon><Box /></el-icon>
                </span>
                <span class="container-name">{{ container.name || '未命名容器' }}</span>
                <el-tag size="small" type="warning">{{ container.image || '无镜像' }}</el-tag>
                <el-button
                  type="danger"
                  plain
                  :icon="Delete"
                  size="small"
                  @click.stop="removeContainer('initContainers', index)"
                  class="remove-btn"
                  circle
                  title="删除初始化容器"
                  aria-label="删除初始化容器"
                />
              </div>
            </template>
            <div class="container-detail">
              <el-tabs :model-value="getContainerActiveTab('initContainers', index)" @tab-change="(tab) => setContainerActiveTab('initContainers', index, tab as string)">
                <el-tab-pane label="基础配置" name="basic">
                  <ContainerBasicInfo :container="container" @update="updateContainer('initContainers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="运行命令" name="command">
                  <ContainerCommand :container="container" @update="updateContainer('initContainers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="环境变量" name="env">
                  <EnvConfig
                    :envs="container.env || []"
                    :configmapList="configMaps"
                    :secretList="secrets"
                    @update="updateContainerEnv('initContainers', index, $event)"
                  />
                </el-tab-pane>
                <el-tab-pane label="健康检测" name="health">
                  <HealthCheck
                    :livenessProbe="container.livenessProbe"
                    :readinessProbe="container.readinessProbe"
                    :startupProbe="container.startupProbe"
                    @updateLiveness="updateContainerProbe('initContainers', index, 'livenessProbe', $event)"
                    @updateReadiness="updateContainerProbe('initContainers', index, 'readinessProbe', $event)"
                    @updateStartup="updateContainerProbe('initContainers', index, 'startupProbe', $event)"
                  />
                </el-tab-pane>
                <el-tab-pane label="资源配置" name="resources">
                  <ResourceConfig :resources="container.resources || {}" @update="updateContainerResources('initContainers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="端口配置" name="ports">
                  <PortConfig :ports="container.ports || []" @update="updateContainerPorts('initContainers', index, $event)" />
                </el-tab-pane>
                <el-tab-pane label="存储挂载" name="volumes">
                  <VolumeMounts :volumeMounts="container.volumeMounts || []" :volumes="volumes" @update="updateContainerVolumeMounts('initContainers', index, $event)" />
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-collapse-item>
        </el-collapse>
        <el-empty v-if="initContainers.length === 0" description="暂无初始化容器" :image-size="60" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Delete, Box } from '@element-plus/icons-vue'
import ContainerBasicInfo from './ContainerBasicInfo.vue'
import ContainerCommand from './ContainerCommand.vue'
import EnvConfig from './EnvConfig.vue'
import HealthCheck from './HealthCheck.vue'
import ResourceConfig from './ResourceConfig.vue'
import PortConfig from './PortConfig.vue'
import VolumeMounts from './VolumeMounts.vue'

interface Container {
  name: string
  image: string
  imagePullPolicy?: string
  workingDir?: string
  command?: string[]
  args?: string[]
  env?: any[]
  resources?: any
  ports?: any[]
  volumeMounts?: any[]
  stdin?: boolean
  tty?: boolean
  activeTab?: string
}

const props = defineProps<{
  containers: Container[]
  initContainers: Container[]
  volumes: any[]
  configMaps?: { name: string }[]
  secrets?: { name: string }[]
}>()

const emit = defineEmits<{
  updateContainers: [containers: Container[]]
  updateInitContainers: [initContainers: Container[]]
}>()

const activeContainers = ref<number[]>([])
const activeInitContainers = ref<number[]>([])

// 获取容器的活动标签页
const getContainerActiveTab = (type: 'containers' | 'initContainers', index: number) => {
  const containerList = type === 'containers' ? props.containers : props.initContainers
  return containerList[index]?.activeTab || 'basic'
}

// 设置容器的活动标签页
const setContainerActiveTab = (type: 'containers' | 'initContainers', index: number, tabName: string) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], activeTab: tabName }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], activeTab: tabName }
    emit('updateInitContainers', updated)
  }
}

const addContainer = (type: 'containers' | 'initContainers') => {
  const newContainer: Container = {
    name: '',
    image: '',
    imagePullPolicy: 'IfNotPresent',
    command: [],
    args: [],
    env: [],
    ports: [],
    volumeMounts: [],
    activeTab: 'basic'
  }

  if (type === 'containers') {
    const updated = [...props.containers, newContainer]
    emit('updateContainers', updated)
    activeContainers.value = [updated.length - 1]
  } else {
    const updated = [...props.initContainers, newContainer]
    emit('updateInitContainers', updated)
    activeInitContainers.value = [updated.length - 1]
  }
}

const removeContainer = (type: 'containers' | 'initContainers', index: number) => {
  if (type === 'containers') {
    const updated = props.containers.filter((_, i) => i !== index)
    emit('updateContainers', updated)
  } else {
    const updated = props.initContainers.filter((_, i) => i !== index)
    emit('updateInitContainers', updated)
  }
}

const updateContainer = (type: 'containers' | 'initContainers', index: number, data: Partial<Container>) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], ...data }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], ...data }
    emit('updateInitContainers', updated)
  }
}

const updateContainerEnv = (type: 'containers' | 'initContainers', index: number, envs: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], env: envs }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], env: envs }
    emit('updateInitContainers', updated)
  }
}

const updateContainerResources = (type: 'containers' | 'initContainers', index: number, resources: any) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], resources }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], resources }
    emit('updateInitContainers', updated)
  }
}

const updateContainerPorts = (type: 'containers' | 'initContainers', index: number, ports: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], ports }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], ports }
    emit('updateInitContainers', updated)
  }
}

const updateContainerVolumeMounts = (type: 'containers' | 'initContainers', index: number, volumeMounts: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], volumeMounts }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], volumeMounts }
    emit('updateInitContainers', updated)
  }
}

const updateContainerProbe = (type: 'containers' | 'initContainers', index: number, probeType: string, probe: any) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], [probeType]: probe }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], [probeType]: probe }
    emit('updateInitContainers', updated)
  }
}
</script>

<style scoped>
.container-config {
  --editor-primary: #111827;
  --editor-primary-dark: #374151;
  --editor-primary-soft: #f3f4f6;
  --editor-primary-softer: #f9fafb;
  --editor-ink: #101828;
  --editor-muted: #475569;
  --editor-border: #e5e7eb;
  --editor-danger: #dc2626;
  --editor-danger-soft: #fff1f2;
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 0;
}

.container-section {
  background: #ffffff;
  border: 1px solid var(--editor-border);
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.05);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  background: #ffffff;
  border-bottom: 1px solid var(--editor-border);
}

.section-title {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 800;
  color: var(--editor-ink);
  letter-spacing: -0.01em;
}

.section-title::before {
  content: '';
  width: 4px;
  height: 18px;
  border-radius: 4px;
  background: var(--editor-primary);
  box-shadow: none;
}

.section-header .section-add-btn {
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--editor-primary);
  border-radius: 8px;
  background: var(--editor-primary);
  color: #ffffff;
  font-weight: 700;
  letter-spacing: 0;
  box-shadow: none;
  transition: all 0.2s ease;
}

.section-header .section-add-btn:hover {
  background: var(--editor-primary-dark);
  border-color: var(--editor-primary-dark);
  color: #ffffff;
  box-shadow: none;
  transform: none;
}

.section-header .section-add-btn:active {
  transform: translateY(0);
}

.section-header .section-add-btn :deep(.el-icon) {
  font-size: 13px;
  font-weight: 700;
}

.container-list {
  padding: 16px;
  background: #f8fafc;
}

.container-list :deep(.el-collapse) {
  border: none;
}

.container-list :deep(.el-collapse-item) {
  margin-bottom: 12px;
  border: 1px solid var(--editor-border);
  border-radius: 12px;
  overflow: hidden;
  background: #ffffff;
  box-shadow: none;
}

.container-list :deep(.el-collapse-item:last-child) {
  margin-bottom: 0;
}

.container-list :deep(.el-collapse-item__header) {
  min-height: 56px;
  padding: 0 16px;
  border: none;
  border-bottom: 1px solid var(--editor-border);
  background: #ffffff;
  color: var(--editor-ink);
  transition: all 0.2s ease;
}

.container-list :deep(.el-collapse-item__header:hover) {
  background: #f9fafb;
}

.container-list :deep(.el-collapse-item__wrap) {
  border: none;
  background: #ffffff;
}

.container-list :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}

.container-title {
  display: grid;
  grid-template-columns: 34px minmax(150px, 260px) minmax(0, 1fr) 32px;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-width: 0;
}

.container-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border: 1px solid #d1d5db;
  border-radius: 10px;
  background: var(--editor-primary-soft);
  color: var(--editor-primary);
  box-shadow: none;
}

.container-icon-badge.init {
  border-color: #d1d5db;
  color: #374151;
  background: #f9fafb;
}

.container-icon-badge .el-icon {
  font-size: 18px;
}

.container-name {
  min-width: 0;
  color: var(--editor-ink);
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: normal;
}

.container-title :deep(.el-tag) {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  justify-self: start;
  border: 1px solid #d1d5db;
  border-radius: 999px;
  background: #f9fafb;
  color: #2563eb;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 700;
}

.remove-btn {
  width: 32px !important;
  min-width: 32px !important;
  height: 32px !important;
  margin-left: 0 !important;
  padding: 0 !important;
  border-color: #fecdd3 !important;
  border-radius: 9px !important;
  background: var(--editor-danger-soft) !important;
  color: var(--editor-danger) !important;
  font-weight: 700 !important;
  box-shadow: none !important;
  transition: all 0.2s ease;
}

.remove-btn:hover {
  border-color: #fca5a5 !important;
  background: #fee2e2 !important;
  color: #b91c1c !important;
  box-shadow: none !important;
  transform: none;
}

.remove-btn:active {
  transform: translateY(0);
}

.remove-btn :deep(.el-icon) {
  font-size: 13px;
}

.container-detail {
  padding: 16px;
  background: #ffffff;
}

.container-detail :deep(.el-tabs__header) {
  background: #ffffff;
  border-radius: 0;
  margin-bottom: 18px;
  border: 1px solid var(--editor-border);
  border-width: 0 0 1px 0;
  padding: 0;
}

.container-detail :deep(.el-tabs__nav) {
  border: none;
  gap: 24px;
}

.container-detail :deep(.el-tabs__item) {
  color: var(--editor-muted);
  font-weight: 700;
  border: none;
  border-radius: 0;
  padding: 0 !important;
  height: 42px;
  line-height: 42px;
  transition: all 0.2s ease;
}

.container-detail :deep(.el-tabs__item:hover) {
  color: #111827;
  background: transparent;
}

.container-detail :deep(.el-tabs__item.is-active) {
  color: #111827;
  background: transparent;
  box-shadow: none;
}

.container-detail :deep(.el-tabs__active-bar) {
  display: block;
  height: 3px;
  border-radius: 999px;
  background: #111827;
}

.container-detail :deep(.el-collapse) {
  border: none;
}

.container-detail :deep(.el-collapse-item__header) {
  background: #ffffff;
  border-radius: 8px;
  margin-bottom: 12px;
  padding: 16px 20px;
  border: 1px solid #e8e8e8;
  font-weight: 600;
  color: #333;
  transition: all 0.3s ease;
}

.container-detail :deep(.el-collapse-item__header:hover) {
  border-color: #bfdbfe;
  background: var(--editor-primary-softer);
}

.container-detail :deep(.el-collapse-item__wrap) {
  background: transparent;
  border: none;
}

.container-detail :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
</style>
