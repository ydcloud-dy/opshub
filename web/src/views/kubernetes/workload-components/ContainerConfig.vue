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
                <el-button type="danger" plain :icon="Delete" size="small" @click.stop="removeContainer('containers', index)" class="remove-btn">删除</el-button>
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
                  <EnvConfig :envs="container.env || []" @update="updateContainerEnv('containers', index, $event)" />
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
                <el-button type="danger" plain :icon="Delete" size="small" @click.stop="removeContainer('initContainers', index)" class="remove-btn">删除</el-button>
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
                  <EnvConfig :envs="container.env || []" @update="updateContainerEnv('initContainers', index, $event)" />
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
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.container-section {
  background: #ffffff;
  border-radius: 4px;
  overflow: hidden;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #d4af37;
  border-bottom: 1px solid #d4af37;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  letter-spacing: 0.3px;
}

.section-header .section-add-btn {
  height: 32px;
  padding: 0 14px;
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: #7a5200;
  font-weight: 700;
  letter-spacing: 0.2px;
  box-shadow:
    0 8px 18px rgba(88, 60, 0, 0.14),
    inset 0 1px 0 rgba(255, 255, 255, 0.75);
  transition: all 0.2s ease;
}

.section-header .section-add-btn:hover {
  background: #ffffff;
  border-color: rgba(255, 255, 255, 0.9);
  color: #5f4200;
  box-shadow:
    0 10px 22px rgba(88, 60, 0, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.85);
  transform: translateY(-1px);
}

.section-header .section-add-btn:active {
  transform: translateY(0);
}

.section-header .section-add-btn :deep(.el-icon) {
  font-size: 13px;
  font-weight: 700;
}

.container-list {
  padding: 20px;
  background: #ffffff;
}

.container-title {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.container-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border: 1px solid rgba(212, 175, 55, 0.36);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(255, 248, 222, 0.96), rgba(255, 255, 255, 0.92)),
    radial-gradient(circle at 30% 20%, rgba(212, 175, 55, 0.26), transparent 55%);
  color: #b78b11;
  box-shadow: 0 8px 16px rgba(212, 175, 55, 0.14);
}

.container-icon-badge.init {
  color: #b7791f;
  background:
    linear-gradient(135deg, rgba(255, 246, 232, 0.98), rgba(255, 255, 255, 0.92)),
    radial-gradient(circle at 30% 20%, rgba(250, 173, 20, 0.24), transparent 55%);
}

.container-icon-badge .el-icon {
  font-size: 18px;
}

.container-name {
  min-width: 120px;
  color: #2f2f2f;
  font-weight: 700;
  word-break: break-word;
}

.remove-btn {
  margin-left: auto;
  height: 30px;
  padding: 0 12px;
  border-color: #ffd2d2;
  border-radius: 999px;
  background: #fff7f7;
  color: #f05252;
  font-weight: 700;
  box-shadow: none;
  transition: all 0.2s ease;
}

.remove-btn:hover {
  border-color: #ff8a8a;
  background: #fff0f0;
  color: #d92d20;
  box-shadow: 0 8px 18px rgba(240, 82, 82, 0.16);
  transform: translateY(-1px);
}

.remove-btn:active {
  transform: translateY(0);
}

.remove-btn :deep(.el-icon) {
  font-size: 13px;
}

.container-detail {
  padding: 24px;
  background: #fafafa;
}

.container-detail :deep(.el-tabs__header) {
  background: #ffffff;
  border-radius: 8px;
  margin-bottom: 16px;
  border: 1px solid #e8e8e8;
}

.container-detail :deep(.el-tabs__nav) {
  border: none;
}

.container-detail :deep(.el-tabs__item) {
  color: #666;
  font-weight: 500;
  border: none;
  padding: 0 20px;
  height: 44px;
  line-height: 44px;
  transition: all 0.3s ease;
}

.container-detail :deep(.el-tabs__item:hover) {
  color: #d4af37;
}

.container-detail :deep(.el-tabs__item.is-active) {
  color: #d4af37;
  background: transparent;
}

.container-detail :deep(.el-tabs__active-bar) {
  height: 2px;
  background: #d4af37;
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
  border-color: #d4af37;
  background: #fafafa;
}

.container-detail :deep(.el-collapse-item__wrap) {
  background: transparent;
  border: none;
}

.container-detail :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
</style>
