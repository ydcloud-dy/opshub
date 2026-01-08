<template>
  <div class="resource-section">
    <div class="resource-group">
      <div class="group-header">
        <span class="group-title">CPU 限制</span>
      </div>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="请求 (Request)">
            <el-input v-model="localContainer.cpuRequest" placeholder="例如: 100m" @input="update" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="限制 (Limit)">
            <el-input v-model="localContainer.cpuLimit" placeholder="例如: 500m 或 1" @input="update" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <div class="resource-group">
      <div class="group-header">
        <span class="group-title">内存限制</span>
      </div>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="请求 (Request)">
            <el-input v-model="localContainer.memoryRequest" placeholder="例如: 128Mi" @input="update" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="限制 (Limit)">
            <el-input v-model="localContainer.memoryLimit" placeholder="例如: 512Mi" @input="update" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface Container {
  cpuRequest?: string
  cpuLimit?: string
  memoryRequest?: string
  memoryLimit?: string
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive<Container>({
  cpuRequest: '',
  cpuLimit: '',
  memoryRequest: '',
  memoryLimit: ''
})

watch(() => props.container, (newVal) => {
  localContainer.cpuRequest = newVal.cpuRequest || ''
  localContainer.cpuLimit = newVal.cpuLimit || ''
  localContainer.memoryRequest = newVal.memoryRequest || ''
  localContainer.memoryLimit = newVal.memoryLimit || ''
}, { deep: true, immediate: true })

const update = () => {
  emit('update', { ...localContainer })
}
</script>

<style scoped>
.resource-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.resource-group {
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.group-header {
  margin-bottom: 16px;
}

.group-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
