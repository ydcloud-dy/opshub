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
          <el-button class="section-add-btn" type="primary" @click="emit('addLabel')" :icon="Plus" size="small">添加</el-button>
        </div>
        <div class="key-value-list">
          <div v-for="(label, index) in formData.labels" :key="'label-'+index" class="key-value-row">
            <el-input v-model="label.key" placeholder="key" size="small" />
            <span class="separator">=</span>
            <el-input v-model="label.value" placeholder="value" size="small" />
            <el-button class="kv-delete-btn" link type="danger" @click="emit('removeLabel', index)" :icon="Delete" size="small" />
          </div>
          <div v-if="formData.labels.length === 0" class="empty-tip">暂无标签</div>
        </div>
      </div>
      <div class="form-section">
        <div class="form-section-header">
          <label>注解</label>
          <el-button class="section-add-btn" type="primary" @click="emit('addAnnotation')" :icon="Plus" size="small">添加</el-button>
        </div>
        <div class="key-value-list">
          <div v-for="(anno, index) in formData.annotations" :key="'anno-'+index" class="key-value-row">
            <el-input v-model="anno.key" placeholder="key" size="small" />
            <span class="separator">=</span>
            <el-input v-model="anno.value" placeholder="value" size="small" />
            <el-button class="kv-delete-btn" link type="danger" @click="emit('removeAnnotation', index)" :icon="Delete" size="small" />
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
.basic-panel {
  --editor-primary: #1d4ed8;
  --editor-primary-dark: #1e40af;
  --editor-primary-soft: #dbeafe;
  --editor-ink: #101828;
  --editor-muted: #475569;
  --editor-border: #cbd5e1;
  --editor-danger: #be123c;
  --editor-danger-soft: #fff1f2;
}

.info-panel {
  background: #f8fafc;
  border-radius: 18px;
  border: 1px solid var(--editor-border);
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.1);
  overflow: hidden;
}

.basic-panel {
  border-right: 0;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--editor-border);
  background: linear-gradient(135deg, #eaf2ff 0%, #dbeafe 100%);
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
  background: var(--editor-primary-soft);
  border: 1px solid #bfdbfe;
  border-radius: 12px;
  color: var(--editor-primary);
  box-shadow: none;
}

.panel-title {
  font-size: 16px;
  font-weight: 800;
  color: var(--editor-ink);
  flex: 1;
  letter-spacing: -0.01em;
}

.panel-content {
  padding: 18px;
  background: #eef4fb;
}

.form-row {
  margin-bottom: 18px;
}

.form-row label {
  display: block;
  font-size: 13px;
  font-weight: 800;
  color: #344054;
  margin-bottom: 8px;
  letter-spacing: 0;
}

.form-row .el-input :deep(.el-input__wrapper) {
  background: #ffffff;
  border: 1px solid #d8dee8;
  border-radius: 10px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  transition: all 0.2s ease;
}

.form-row .el-input :deep(.el-input__wrapper:hover) {
  border-color: #bfdbfe;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.08);
}

.form-row .el-input :deep(.el-input__wrapper.is-focus) {
  border-color: var(--editor-primary);
  box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.12);
}

.form-row .el-input-number {
  width: 100%;
}

.form-row .el-input-number :deep(.el-input__wrapper) {
  background: #ffffff;
  border: 1px solid #d8dee8;
  border-radius: 10px;
}

.form-tip {
  font-size: 12px;
  color: var(--editor-muted);
  margin-top: 6px;
  line-height: 1.5;
}

.form-section {
  margin-bottom: 18px;
  padding: 14px;
  background: #eaf2ff;
  border-radius: 16px;
  border: 1px solid var(--editor-border);
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.form-section-header label {
  font-size: 14px;
  font-weight: 800;
  color: var(--editor-ink);
  letter-spacing: 0;
}

.form-section-header .section-add-btn {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--editor-primary);
  border-radius: 999px;
  background: var(--editor-primary);
  color: #ffffff;
  font-weight: 700;
  letter-spacing: 0;
  box-shadow: 0 10px 22px rgba(29, 78, 216, 0.22);
  transition: all 0.2s ease;
}

.form-section-header .section-add-btn:hover {
  border-color: var(--editor-primary-dark);
  background: var(--editor-primary-dark);
  color: #ffffff;
  box-shadow: 0 14px 28px rgba(29, 78, 216, 0.28);
  transform: translateY(-1px);
}

.form-section-header .section-add-btn:active {
  transform: translateY(0);
}

.form-section-header .section-add-btn :deep(.el-icon) {
  font-size: 13px;
  font-weight: 700;
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
  border-radius: 14px;
  border: 1px solid #c7d2e4;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
  transition: all 0.2s ease;
}

.key-value-row:hover {
  border-color: #93c5fd;
  box-shadow: 0 14px 28px rgba(37, 99, 235, 0.14);
}

.key-value-row .el-input {
  flex: 1;
}

.key-value-row .el-input :deep(.el-input__wrapper) {
  border: none;
  box-shadow: none;
  background: transparent;
}

.kv-delete-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 1px solid #fecdd3;
  background: var(--editor-danger-soft);
  color: var(--editor-danger);
  transition: all 0.2s ease;
}

.kv-delete-btn:hover {
  background: #ffe4e6;
  color: #be123c;
  box-shadow: 0 10px 20px rgba(225, 29, 72, 0.12);
  transform: translateY(-1px);
}

.separator {
  color: var(--editor-primary);
  font-weight: 800;
  font-size: 16px;
}

.empty-tip {
  text-align: center;
  padding: 24px;
  color: var(--editor-muted);
  font-size: 13px;
  background: #ffffff;
  border-radius: 14px;
  border: 1px dashed #cbd5e1;
}
</style>
