<template>
  <div class="ports-config-wrapper">
    <!-- 端口列表 -->
    <div v-if="localPorts.length > 0" class="ports-list">
      <div v-for="(port, index) in localPorts" :key="'port-'+index" class="port-item-card">
        <div class="port-card-header">
          <div class="port-title">
            <span class="port-number">端口 {{ index + 1 }}</span>
            <span v-if="port.name" class="port-name">{{ port.name }}</span>
          </div>
          <el-button
            type="danger"
            link
            @click="removePort(index)"
            :icon="Delete"
            size="small"
            title="删除端口"
            aria-label="删除端口"
          />
        </div>
        <div class="port-card-body">
          <div class="port-field-row">
            <div class="port-field">
              <label>容器端口</label>
              <el-input-number v-model="port.containerPort" :min="1" :max="65535" placeholder="端口号" size="small" style="width: 100%;" @change="updatePorts" />
            </div>
            <div class="port-field">
              <label>协议</label>
              <el-select v-model="port.protocol" placeholder="选择协议" size="small" style="width: 100%;" @change="updatePorts">
                <el-option label="TCP" value="TCP" />
                <el-option label="UDP" value="UDP" />
                <el-option label="SCTP" value="SCTP" />
              </el-select>
            </div>
          </div>
          <div class="port-field-row">
            <div class="port-field">
              <label>端口名称</label>
              <el-input v-model="port.name" placeholder="端口名称，如: http" size="small" @input="updatePorts" />
            </div>
            <div class="port-field">
              <label>主机端口</label>
              <el-input-number v-model="port.hostPort" :min="1" :max="65535" placeholder="可选" size="small" style="width: 100%;" @change="updatePorts" />
            </div>
          </div>
          <div class="port-field-row">
            <div class="port-field full-width">
              <label>主机 IP</label>
              <el-input v-model="port.hostIP" placeholder="可选，绑定到特定主机的 IP 地址" size="small" @input="updatePorts" />
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="empty-ports">
      <el-empty description="暂未配置端口" :image-size="80">
        <el-button type="primary" @click="addPort" :icon="Plus">添加端口</el-button>
      </el-empty>
    </div>

    <!-- 添加端口按钮 -->
    <div v-if="localPorts.length > 0" class="add-port-section">
      <el-button type="primary" @click="addPort" :icon="Plus" style="width: 100%;">添加端口</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'

interface ContainerPort {
  containerPort: number
  name: string
  protocol: 'TCP' | 'UDP' | 'SCTP'
  hostPort?: number
  hostIP?: string
}

const props = defineProps<{
  ports: ContainerPort[]
}>()

const emit = defineEmits<{
  update: [ports: ContainerPort[]]
}>()

const localPorts = ref<ContainerPort[]>([])

watch(() => props.ports, (newVal) => {
  localPorts.value = (newVal || []).map(p => ({
    containerPort: p.containerPort || 0,
    name: p.name || '',
    protocol: p.protocol || 'TCP',
    hostPort: p.hostPort,
    hostIP: p.hostIP || ''
  }))
}, { immediate: true, deep: true })

const addPort = () => {
  localPorts.value.push({
    containerPort: 0,
    name: '',
    protocol: 'TCP',
    hostPort: undefined,
    hostIP: ''
  })
  updatePorts()
}

const removePort = (index: number) => {
  localPorts.value.splice(index, 1)
  updatePorts()
}

const updatePorts = () => {
  emit('update', [...localPorts.value])
}
</script>

<style scoped>
.ports-config-wrapper {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.ports-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.port-item-card {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  background: #ffffff;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.port-item-card:hover {
  border-color: #d1d5db;
  box-shadow: none;
}

.port-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #ffffff;
  border-bottom: 1px solid #e5e7eb;
}

.port-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.port-number {
  font-weight: 600;
  color: #1a1a1a;
  font-size: 14px;
}

.port-name {
  padding: 4px 12px;
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  box-shadow: none;
}

.port-card-body {
  padding: 20px;
  background: #ffffff;
}

.port-field-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.port-field-row:last-child {
  margin-bottom: 0;
}

.port-field {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.port-field.full-width {
  flex: 100%;
}

.port-field label {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.3px;
}

.port-card-body :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.port-card-body :deep(.el-input__wrapper:hover) {
  border-color: #9ca3af;
  box-shadow: none;
}

.port-card-body :deep(.el-input__wrapper.is-focus) {
  border-color: #111827;
  box-shadow: 0 0 0 2px rgba(17, 24, 39, 0.08);
}

.port-card-body :deep(.el-input-number) {
  width: 100%;
}

.port-card-body :deep(.el-input-number .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
}

.port-card-body :deep(.el-select .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
}

.empty-ports {
  padding: 60px 20px;
  text-align: center;
  background: #ffffff;
  border-radius: 12px;
  border: 1px dashed #e0e0e0;
}

.empty-ports :deep(.el-empty) {
  --el-empty-padding: 40px 0;
}

.empty-ports :deep(.el-empty__description) {
  color: #999;
}

.add-port-section {
  margin-top: 8px;
}

.add-port-section .el-button {
  border-radius: 8px;
  font-weight: 700;
  background: #111827;
  border: 1px solid #111827;
  color: #ffffff;
  box-shadow: none;
}

.add-port-section .el-button:hover {
  background: #374151;
  border-color: #374151;
  color: #ffffff;
  box-shadow: none;
}

.port-card-header :deep(.el-button--danger),
.port-card-header :deep(.el-button--danger.is-link) {
  width: 32px;
  min-width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid #fecdd3;
  border-radius: 8px;
  background: #fff1f2;
  color: #dc2626;
}

.port-card-header :deep(.el-button--danger:hover),
.port-card-header :deep(.el-button--danger.is-link:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
}
</style>
