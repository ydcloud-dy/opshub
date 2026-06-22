<template>
  <div class="mount-config-content">
    <div class="mount-section">
      <div class="mount-header">
        <span class="mount-header-title">
          <el-icon><FolderOpened /></el-icon>
          卷挂载配置
        </span>
        <el-button type="primary" @click="addMount" :icon="Plus" size="default">
          添加挂载点
        </el-button>
      </div>
      <div v-if="localMounts.length > 0" class="mount-table-wrapper">
        <el-table :data="localMounts" class="mount-table" size="default">
          <el-table-column label="卷名" min-width="150">
            <template #default="{ row }">
              <el-select
                v-model="row.name"
                placeholder="选择数据卷"
                filterable
                size="small"
                @change="updateMounts"
              >
                <el-option
                  v-for="vol in volumes"
                  :key="vol.name"
                  :label="vol.name"
                  :value="vol.name"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="挂载路径" min-width="200">
            <template #default="{ row }">
              <el-input
                v-model="row.mountPath"
                placeholder="/container/path"
                size="small"
                @input="updateMounts"
              />
            </template>
          </el-table-column>
          <el-table-column label="子路径" min-width="180">
            <template #default="{ row }">
              <el-input
                v-model="row.subPath"
                placeholder="可选，挂载卷的子路径"
                size="small"
                @input="updateMounts"
              />
            </template>
          </el-table-column>
          <el-table-column label="读写模式" width="120" align="center">
            <template #default="{ row }">
              <el-switch
                v-model="row.readOnly"
                active-text="只读"
                inactive-text="读写"
                @change="updateMounts"
              />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row, $index }">
              <el-button type="danger" link @click="removeMount($index)" :icon="Delete" />
            </template>
          </el-table-column>
        </el-table>
      </div>
      <el-empty v-else description="暂无挂载点配置" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Plus, Delete, FolderOpened } from '@element-plus/icons-vue'

interface VolumeMount {
  name: string
  mountPath: string
  subPath?: string
  readOnly?: boolean
}

const props = defineProps<{
  volumeMounts: VolumeMount[]
  volumes: { name: string; type?: string }[]
}>()

const emit = defineEmits<{
  update: [mounts: VolumeMount[]]
}>()

const localMounts = ref<VolumeMount[]>([])

watch(() => props.volumeMounts, (newVal) => {
  localMounts.value = (newVal || []).map(m => ({
    name: m.name || '',
    mountPath: m.mountPath || '',
    subPath: m.subPath || '',
    readOnly: m.readOnly || false
  }))
}, { immediate: true, deep: true })

const addMount = () => {
  localMounts.value.push({
    name: '',
    mountPath: '',
    subPath: '',
    readOnly: false
  })
  updateMounts()
}

const removeMount = (index: number) => {
  localMounts.value.splice(index, 1)
  updateMounts()
}

const updateMounts = () => {
  emit('update', [...localMounts.value])
}
</script>

<style scoped>
.mount-config-content {
  padding: 0;
  background: transparent;
}

.mount-section {
  width: 100%;
}

.mount-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px 12px 0 0;
  margin-bottom: 0;
}

.mount-header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 800;
  color: #111827;
  letter-spacing: 0;
}

.mount-header-title .el-icon {
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  background: #f3f4f6;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  color: #111827;
  box-shadow: none;
}

.mount-header .el-button {
  height: 32px;
  padding: 0 14px;
  font-weight: 700;
  border-radius: 8px;
  background: #111827;
  border: 1px solid #111827;
  color: #ffffff;
  box-shadow: none;
}

.mount-header .el-button:hover {
  background: #374151;
  border-color: #374151;
  color: #ffffff;
  box-shadow: none;
}

.mount-table-wrapper {
  border: 1px solid #e5e7eb;
  border-top: none;
  border-radius: 0 0 12px 12px;
  padding: 16px;
  background: #ffffff;
}

.mount-table {
  background: #ffffff;
  border-radius: 8px;
  overflow: hidden;
}

.mount-table :deep(.el-table__header-wrapper) {
  background: #f9fafb;
}

.mount-table :deep(.el-table__header th) {
  background: #f9fafb;
  color: #374151;
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0;
  border-bottom: 1px solid #e5e7eb;
}

.mount-table :deep(.el-table__body) {
  font-size: 13px;
}

.mount-table :deep(.el-table__body tr) {
  transition: all 0.3s ease;
}

.mount-table :deep(.el-table__body tr:hover) {
  background: #f9fafb;
}

.mount-table :deep(.el-table__body td) {
  border-bottom: 1px solid #f3f4f6;
}

.mount-table :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.mount-table :deep(.el-input__wrapper:hover) {
  border-color: #9ca3af;
  box-shadow: none;
}

.mount-table :deep(.el-input__wrapper.is-focus) {
  border-color: #111827;
  box-shadow: 0 0 0 2px rgba(17, 24, 39, 0.08);
}

.mount-table :deep(.el-select .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
}

.mount-table :deep(.el-switch) {
  --el-switch-on-color: #111827;
}

.mount-table :deep(.el-button--danger),
.mount-table :deep(.el-button--danger.is-link) {
  width: 32px;
  min-width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid #fecdd3;
  border-radius: 8px;
  background: #fff1f2;
  color: #dc2626;
}

.mount-table :deep(.el-button--danger:hover),
.mount-table :deep(.el-button--danger.is-link:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
}

:deep(.el-empty) {
  padding: 60px 0;
}

:deep(.el-empty__description) {
  color: #999;
}
</style>
