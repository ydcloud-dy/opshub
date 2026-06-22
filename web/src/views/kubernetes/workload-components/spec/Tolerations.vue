<template>
  <div class="spec-content-wrapper">
    <div class="spec-content-header">
      <h3>容忍度</h3>
      <p>配置 Pod 对节点污点的容忍度</p>
    </div>
    <div class="spec-content">
      <div class="tolerations-table-wrapper">
        <el-table :data="tolerations" border stripe>
          <el-table-column label="键" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.key" placeholder="键名，如: key" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="运算符" min-width="120">
            <template #default="{ row }">
              <el-select v-model="row.operator" placeholder="运算符" size="small">
                <el-option label="Equal" value="Equal" />
                <el-option label="Exists" value="Exists" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="值" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.value" placeholder="值" size="small" :disabled="row.operator === 'Exists'" />
            </template>
          </el-table-column>
          <el-table-column label="影响" min-width="150">
            <template #default="{ row }">
              <el-select v-model="row.effect" placeholder="影响类型" size="small">
                <el-option label="NoExecute" value="NoExecute" />
                <el-option label="NoSchedule" value="NoSchedule" />
                <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="容忍时间(秒)" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.tolerationSeconds" placeholder="仅 NoExecute 生效" size="small" :disabled="row.effect !== 'NoExecute'" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ $index }">
              <el-button
                type="danger"
                link
                @click="emit('removeToleration', $index)"
                :icon="Delete"
                size="small"
                title="删除容忍"
                aria-label="删除容忍"
              />
            </template>
          </el-table-column>
        </el-table>
        <div class="tolerations-actions">
          <el-button type="primary" @click="emit('addToleration')" :icon="Plus">添加容忍</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

interface Toleration {
  key: string
  operator: 'Equal' | 'Exists'
  value: string
  effect: 'NoExecute' | 'NoSchedule' | 'PreferNoSchedule'
  tolerationSeconds?: string
}

const props = defineProps<{
  tolerations: Toleration[]
}>()

const emit = defineEmits<{
  addToleration: []
  removeToleration: [index: number]
}>()
</script>

<style scoped>
.spec-content-wrapper {
  padding: 0;
  background: transparent;
}

.spec-content-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
}

.spec-content-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.spec-content-header p {
  margin: 0;
  font-size: 13px;
  color: #909399;
}

.spec-content {
  background: #fff;
}

.tolerations-table-wrapper {
  width: 100%;
}

.tolerations-table-wrapper .el-table {
  width: 100%;
  margin-bottom: 16px;
}

.tolerations-actions {
  display: flex;
  justify-content: flex-start;
  padding-top: 8px;
}

.tolerations-actions :deep(.el-button--primary) {
  height: 32px;
  padding: 0 14px;
  border: 1px solid #111827;
  border-radius: 8px;
  background: #111827;
  color: #ffffff;
  font-weight: 700;
  box-shadow: none;
}

.tolerations-actions :deep(.el-button--primary:hover) {
  border-color: #374151;
  background: #374151;
  color: #ffffff;
  box-shadow: none;
}

.tolerations-table-wrapper :deep(.el-button--danger),
.tolerations-table-wrapper :deep(.el-button--danger.is-link) {
  width: 32px;
  min-width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid #fecdd3;
  border-radius: 8px;
  background: #fff1f2;
  color: #dc2626;
  box-shadow: none;
}

.tolerations-table-wrapper :deep(.el-button--danger:hover),
.tolerations-table-wrapper :deep(.el-button--danger.is-link:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
}
</style>
