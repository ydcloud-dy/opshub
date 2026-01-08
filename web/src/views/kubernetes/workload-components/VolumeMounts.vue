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
      <div v-if="mounts.length > 0" class="mount-table-wrapper">
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
                  v-for="vol in volumeList"
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
                v-model="row.path"
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
  path: string
  subPath?: string
  readOnly?: boolean
}

const props = defineProps<{
  mounts: VolumeMount[]
  volumeList?: { name: string }[]
}>()

const emit = defineEmits<{
  update: [mounts: VolumeMount[]]
}>()

const localMounts = ref<VolumeMount[]>([])

watch(() => props.mounts, (newVal) => {
  localMounts.value = (newVal || []).map(m => ({
    name: m.name,
    path: m.path,
    subPath: m.subPath || '',
    readOnly: m.readOnly || false
  }))
}, { immediate: true, deep: true })

const addMount = () => {
  localMounts.value.push({
    name: '',
    path: '',
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
  background: #fff;
}

.mount-section {
  width: 100%;
}

.mount-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: linear-gradient(to right, #f8f9fa, #ffffff);
  border: 1px solid #e4e7ed;
  border-radius: 8px 8px 0 0;
  margin-bottom: 0;
}

.mount-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.mount-header-title .el-icon {
  font-size: 18px;
  color: #409eff;
}

.mount-table-wrapper {
  border: 1px solid #e4e7ed;
  border-top: none;
  border-radius: 0 0 8px 8px;
  padding: 16px;
  background: #fafbfc;
}

.mount-table {
  background: #fff;
  border-radius: 6px;
}

.mount-table :deep(.el-table__header-wrapper) {
  background: #f5f7fa;
}

.mount-table :deep(.el-table__header th) {
  background: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

.mount-table :deep(.el-table__body) {
  font-size: 13px;
}

.mount-table :deep(.el-input__wrapper) {
  box-shadow: none !important;
  background: transparent;
}

.mount-table :deep(.el-input__wrapper:hover) {
  box-shadow: none !important;
}

.mount-table :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset !important;
}

:deep(.el-empty) {
  padding: 40px 0;
}
</style>
