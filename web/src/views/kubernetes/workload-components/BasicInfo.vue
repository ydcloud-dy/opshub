<template>
  <div class="info-panel basic-panel">
    <div class="panel-header">
      <span class="panel-icon">📋</span>
      <span class="panel-title">基础信息</span>
    </div>
    <div class="panel-content">
      <div class="form-row">
        <label>名称</label>
        <el-input v-model="formData.name" size="small" disabled />
      </div>
      <div class="form-row">
        <label>命名空间</label>
        <el-input v-model="formData.namespace" size="small" disabled />
      </div>
      <div class="form-row" v-if="formData.type === 'Deployment' || formData.type === 'StatefulSet'">
        <label>副本数</label>
        <el-input-number v-model="formData.replicas" :min="0" :max="100" size="small" />
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
  background: #fff;
}

.basic-panel {
  border-right: 1px solid #e9ecef;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #e9ecef;
  background: #fff;
  position: sticky;
  top: 0;
  z-index: 10;
}

.panel-icon {
  font-size: 18px;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #212529;
  flex: 1;
}

.panel-content {
  padding: 12px 16px;
}

.form-row {
  margin-bottom: 12px;
}

.form-row label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #495057;
  margin-bottom: 6px;
}

.form-row .el-input,
.form-row .el-input-number {
  width: 100%;
}

.form-section {
  margin-bottom: 16px;
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.form-section-header label {
  font-size: 13px;
  font-weight: 500;
  color: #495057;
}

.key-value-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.key-value-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.key-value-row .el-input {
  flex: 1;
}

.separator {
  color: #adb5bd;
  font-weight: 500;
  font-size: 14px;
}

.empty-tip {
  text-align: center;
  padding: 20px;
  color: #adb5bd;
  font-size: 13px;
}
</style>
