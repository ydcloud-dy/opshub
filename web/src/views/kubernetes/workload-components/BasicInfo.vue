<template>
  <div class="info-panel basic-panel">
    <div class="panel-header">
      <span class="panel-icon">📋</span>
      <span class="panel-title">基础信息</span>
    </div>
    <div class="panel-content">
      <div class="form-row">
        <label>名称</label>
        <el-input v-model="formData.name" size="small" :disabled="!isCreateMode" placeholder="请输入工作负载名称" />
      </div>
      <div class="form-row">
        <label>命名空间</label>
        <el-select v-if="isCreateMode" v-model="formData.namespace" size="small" filterable placeholder="选择命名空间" style="width: 100%">
          <el-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </el-select>
        <el-input v-else v-model="formData.namespace" size="small" disabled />
      </div>
      <div class="form-row" v-if="formData.type === 'Deployment' || formData.type === 'StatefulSet'">
        <label>副本数</label>
        <el-input-number v-model="formData.replicas" :min="0" :max="100" size="small" />
        <div class="form-tip" v-if="formData.type === 'Deployment'">Deployment 会维护指定数量的 Pod 副本</div>
        <div class="form-tip" v-else-if="formData.type === 'StatefulSet'">StatefulSet 会维护指定数量的有序 Pod 副本</div>
      </div>
      <div class="form-row" v-if="formData.type === 'DaemonSet'">
        <label>副本数</label>
        <el-input value="每个节点一个 Pod" disabled size="small" />
        <div class="form-tip">DaemonSet 会在每个符合条件的节点上运行一个 Pod</div>
      </div>
      <div class="form-row" v-if="formData.type === 'Pod'">
        <label>副本数</label>
        <el-input value="单个 Pod（无副本）" disabled size="small" />
        <div class="form-tip">Pod 是独立的单元，不涉及副本管理</div>
      </div>
      <div class="form-row" v-if="formData.type === 'Job'">
        <label>副本数</label>
        <el-input value="请使用「扩容配置」中的 Job 任务配置" disabled size="small" />
        <div class="form-tip">Job 使用完成次数和并行度来控制 Pod 数量，而非传统副本数</div>
      </div>
      <div class="form-row" v-if="formData.type === 'CronJob'">
        <label>副本数</label>
        <el-input value="请使用「扩容配置」中的 CronJob 配置" disabled size="small" />
        <div class="form-tip">CronJob 通过调度规则和 Job 配置来管理 Pod，而非传统副本数</div>
      </div>
      <div class="form-section">
        <div class="form-section-header">
          <label>标签</label>
          <el-button link type="primary" @click="emit('addLabel')" :icon="Plus" size="small">添加</el-button>
        </div>
        <div class="key-value-list">
          <div v-for="(label, index) in formData.labels" :key="'label-'+index" class="key-value-row">
            <el-input v-model="label.key" placeholder="key" size="small" />
            <span class="separator">=</span>
            <el-input v-model="label.value" placeholder="value" size="small" />
            <el-button link type="danger" @click="emit('removeLabel', index)" :icon="Delete" size="small" />
          </div>
          <div v-if="formData.labels.length === 0" class="empty-tip">暂无标签</div>
        </div>
      </div>
      <div class="form-section">
        <div class="form-section-header">
          <label>注解</label>
          <el-button link type="primary" @click="emit('addAnnotation')" :icon="Plus" size="small">添加</el-button>
        </div>
        <div class="key-value-list">
          <div v-for="(anno, index) in formData.annotations" :key="'anno-'+index" class="key-value-row">
            <el-input v-model="anno.key" placeholder="key" size="small" />
            <span class="separator">=</span>
            <el-input v-model="anno.value" placeholder="value" size="small" />
            <el-button link type="danger" @click="emit('removeAnnotation', index)" :icon="Delete" size="small" />
          </div>
          <div v-if="formData.annotations.length === 0" class="empty-tip">暂无注解</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

interface FormData {
  name: string
  namespace: string
  type: string
  replicas: number
  labels: { key: string; value: string }[]
  annotations: { key: string; value: string }[]
}

const props = defineProps<{
  formData: FormData
  isCreateMode?: boolean
  namespaceList?: { name: string }[]
}>()

const emit = defineEmits<{
  addLabel: []
  removeLabel: [index: number]
  addAnnotation: []
  removeAnnotation: [index: number]
}>()
</script>

<style scoped>
.info-panel {
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.basic-panel {
  border-right: 1px solid #f0f0f0;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 2px solid #d4af37;
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  position: sticky;
  top: 0;
  z-index: 10;
}

.panel-icon {
  font-size: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: #d4af37;
  border-radius: 8px;
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  flex: 1;
  letter-spacing: 0.3px;
}

.panel-content {
  padding: 20px;
  background: #ffffff;
}

.form-row {
  margin-bottom: 20px;
}

.form-row label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
  letter-spacing: 0.3px;
}

.form-row .el-input :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.form-row .el-input :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.form-row .el-input :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.form-row .el-input-number {
  width: 100%;
}

.form-row .el-input-number :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 6px;
  line-height: 1.5;
}

.form-section {
  margin-bottom: 24px;
  padding: 16px;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-radius: 10px;
  border: 1px solid #e8e8e8;
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.form-section-header label {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.3px;
}

.form-section-header .el-button {
  font-weight: 500;
  background: #d4af37;
  border: none;
  color: #1a1a1a;
}

.form-section-header .el-button:hover {
  background: #c9a227;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
}

.key-value-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.key-value-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: #ffffff;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
  transition: all 0.3s ease;
}

.key-value-row:hover {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.key-value-row .el-input {
  flex: 1;
}

.key-value-row .el-input :deep(.el-input__wrapper) {
  border: none;
  box-shadow: none;
  background: transparent;
}

.separator {
  color: #d4af37;
  font-weight: 600;
  font-size: 16px;
}

.empty-tip {
  text-align: center;
  padding: 24px;
  color: #999;
  font-size: 13px;
  background: #ffffff;
  border-radius: 8px;
  border: 1px dashed #e0e0e0;
}
</style>
