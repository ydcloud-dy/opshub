<template>
  <div class="ports-config-wrapper">
    <!-- 端口列表 -->
    <div v-if="ports && ports.length > 0" class="ports-list">
      <div v-for="(port, index) in ports" :key="'port-'+index" class="port-item-card">
        <div class="port-card-header">
          <div class="port-title">
            <span class="port-number">端口 {{ index + 1 }}</span>
            <span v-if="port.name" class="port-name">{{ port.name }}</span>
          </div>
          <el-button type="danger" link @click="$emit('remove', index)" :icon="Delete" size="small">删除</el-button>
        </div>
        <div class="port-card-body">
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="容器端口" label-width="80px">
                <el-input-number v-model="port.containerPort" :min="1" :max="65535" placeholder="端口号" size="small" style="width: 100%;" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="协议" label-width="50px">
                <el-select v-model="port.protocol" placeholder="选择协议" size="small" style="width: 100%;">
                  <el-option label="TCP" value="TCP" />
                  <el-option label="UDP" value="UDP" />
                  <el-option label="SCTP" value="SCTP" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="端口名称" label-width="80px">
                <el-input v-model="port.name" placeholder="端口名称，如: http" size="small" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="主机端口" label-width="80px">
                <el-input-number v-model="port.hostPort" :min="1" :max="65535" placeholder="可选" size="small" style="width: 100%;" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item label="主机 IP" label-width="80px">
                <el-input v-model="port.hostIP" placeholder="可选，绑定到特定主机的 IP 地址" size="small" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>
      </div>
    </div>
    <div v-else class="empty-ports">
      <el-empty description="暂未配置端口" :image-size="80">
        <el-button type="primary" @click="$emit('add')" :icon="Plus">添加端口</el-button>
      </el-empty>
    </div>

    <!-- 添加端口按钮 -->
    <div v-if="ports && ports.length > 0" class="add-port-section">
      <el-button type="primary" @click="$emit('add')" :icon="Plus" style="width: 100%;">添加端口</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'

interface ContainerPort {
  containerPort: number
  name: string
  protocol: 'TCP' | 'UDP' | 'SCTP'
  hostPort?: number
  hostIP?: string
}

defineProps<{
  ports: ContainerPort[]
}>()

defineEmits<{
  add: []
  remove: [index: number]
}>()
</script>

<style scoped>
.ports-config-wrapper {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ports-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.port-item-card {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.port-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);
}

.port-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.port-number {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.port-name {
  padding: 2px 8px;
  background: var(--el-color-primary);
  color: white;
  border-radius: 4px;
  font-size: 12px;
}

.port-card-body {
  padding: 16px;
}

.empty-ports {
  padding: 20px 0;
}

.add-port-section {
  margin-top: 8px;
}
</style>
