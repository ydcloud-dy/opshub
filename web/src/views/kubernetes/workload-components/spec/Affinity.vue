<template>
  <div class="affinity-wrapper">
    <div class="affinity-action-buttons">
      <el-button type="primary" :icon="Plus" @click="emit('startAddAffinity', 'pod')">添加 Pod 亲和性</el-button>
      <el-button type="primary" :icon="Plus" @click="emit('startAddAffinity', 'node')">添加 Node 亲和性</el-button>
    </div>

    <!-- 配置表单 -->
    <div v-if="editingAffinityRule !== null" class="affinity-config-container">
      <div class="config-container-header">
        <div class="config-type-badge">
          <el-tag v-if="editingAffinityRule.type === 'podAffinity'" type="success">Pod 亲和性</el-tag>
          <el-tag v-else-if="editingAffinityRule.type === 'podAntiAffinity'" type="danger">Pod 反亲和性</el-tag>
          <el-tag v-else-if="editingAffinityRule.type === 'nodeAffinity'" type="success">Node 亲和性</el-tag>
          <el-tag v-else type="danger">Node 反亲和性</el-tag>
        </div>
        <div class="config-header-actions">
          <el-button @click="emit('cancelAffinityEdit')">取消</el-button>
          <el-button type="primary" @click="emit('saveAffinityRule')">添加</el-button>
        </div>
      </div>

      <div class="config-container-body">
        <!-- 类型 -->
        <div class="config-form-section">
          <label class="form-label">类型</label>
          <el-radio-group v-model="editingAffinityRule.type" class="affinity-type-radio">
            <template v-if="editingAffinityRule.type.includes('pod')">
              <el-radio value="podAffinity" class="affinity-radio-item">Pod 亲和性</el-radio>
              <el-radio value="podAntiAffinity" class="affinity-radio-item">Pod 反亲和性</el-radio>
            </template>
            <template v-else>
              <el-radio value="nodeAffinity" class="affinity-radio-item">Node 亲和性</el-radio>
              <el-radio value="nodeAntiAffinity" class="affinity-radio-item">Node 反亲和性</el-radio>
            </template>
          </el-radio-group>
        </div>

        <!-- Namespaces（仅Pod亲和性） -->
        <div v-if="editingAffinityRule.type.includes('pod')" class="config-form-section">
          <label class="form-label">Namespaces</label>
          <el-select
            v-model="editingAffinityRule.namespaces"
            multiple
            placeholder="选择命名空间"
            class="full-width-input"
          >
            <el-option
              v-for="ns in namespaceList"
              :key="ns.name"
              :label="ns.name"
              :value="ns.name"
            />
          </el-select>
        </div>

        <!-- 拓扑键（仅Pod亲和性） -->
        <div v-if="editingAffinityRule.type.includes('pod')" class="config-form-section">
          <label class="form-label">拓扑键 (Topology Key)</label>
          <el-input
            v-model="editingAffinityRule.topologyKey"
            placeholder="例如: kubernetes.io/hostname, topology.kubernetes.io/zone"
            class="full-width-input"
          />
          <div class="form-tip">
            常用拓扑键：kubernetes.io/hostname（节点）、topology.kubernetes.io/zone（可用区）、topology.kubernetes.io/region（区域）
          </div>
        </div>

        <!-- 优先级 -->
        <div class="config-form-section">
          <label class="form-label">优先级</label>
          <el-select v-model="editingAffinityRule.priority" class="full-width-input">
            <el-option label="Required (必须)" value="Required" />
            <el-option label="Preferred (首选)" value="Preferred" />
          </el-select>
        </div>

        <!-- 权重 -->
        <div v-if="editingAffinityRule.priority === 'Preferred'" class="config-form-section">
          <label class="form-label">权重</label>
          <el-input-number v-model="editingAffinityRule.weight" :min="1" :max="100" class="full-width-input" />
        </div>

        <!-- Match Expressions -->
        <div class="config-form-section">
          <div class="section-header">
            <label class="form-label">Match Expressions</label>
            <el-button type="primary" :icon="Plus" size="small" @click="emit('addMatchExpression')">添加</el-button>
          </div>
          <div class="expressions-list">
            <div v-for="(exp, index) in editingAffinityRule.matchExpressions" :key="'expr-'+index" class="expression-config-row">
              <div class="expression-config-grid">
                <div class="config-grid-item">
                  <label class="config-grid-label">Key</label>
                  <el-input v-model="exp.key" placeholder="例如: app" />
                </div>
                <div class="config-grid-item">
                  <label class="config-grid-label">Operator</label>
                  <el-select v-model="exp.operator" placeholder="选择操作符">
                    <el-option label="In" value="In" />
                    <el-option label="NotIn" value="NotIn" />
                    <el-option label="Exists" value="Exists" />
                    <el-option label="DoesNotExist" value="DoesNotExist" />
                    <el-option label="Gt" value="Gt" />
                    <el-option label="Lt" value="Lt" />
                  </el-select>
                </div>
                <div class="config-grid-item" v-if="exp.operator !== 'Exists' && exp.operator !== 'DoesNotExist'">
                  <label class="config-grid-label">Values</label>
                  <el-input v-model="exp.valueStr" placeholder="多个值用逗号分隔" />
                </div>
              </div>
              <div class="expression-config-actions">
                <el-button type="danger" :icon="Delete" size="small" @click="emit('removeMatchExpression', index)">删除</el-button>
              </div>
            </div>
            <el-empty v-if="editingAffinityRule.matchExpressions.length === 0" description="暂无匹配表达式" :image-size="60" />
          </div>
        </div>

        <!-- Match Labels -->
        <div class="config-form-section">
          <div class="section-header">
            <label class="form-label">Match Labels</label>
            <el-button type="primary" :icon="Plus" size="small" @click="emit('addMatchLabel')">添加</el-button>
          </div>
          <div class="labels-list">
            <div v-for="(label, index) in editingAffinityRule.matchLabels" :key="'label-'+index" class="label-config-row">
              <div class="label-config-grid">
                <el-input v-model="label.key" placeholder="Key" style="flex: 1" />
                <span class="label-separator">=</span>
                <el-input v-model="label.value" placeholder="Value" style="flex: 1" />
              </div>
              <el-button type="danger" :icon="Delete" size="small" @click="emit('removeMatchLabel', index)">删除</el-button>
            </div>
            <el-empty v-if="editingAffinityRule.matchLabels.length === 0" description="暂无标签" :image-size="60" />
          </div>
        </div>
      </div>
    </div>

    <!-- 已配置规则列表 -->
    <div v-if="affinityRules.length > 0" class="affinity-rules-list">
      <div class="affinity-rules-header">
        <span class="header-title">亲和性规则</span>
      </div>
      <div v-for="(rule, rIndex) in affinityRules" :key="'aff-rule-'+rIndex" class="affinity-rule-card">
        <div class="affinity-rule-header">
          <div class="rule-type-badge">
            <el-tag v-if="rule.type === 'podAffinity'" type="success">Pod 亲和性</el-tag>
            <el-tag v-else-if="rule.type === 'podAntiAffinity'" type="danger">Pod 反亲和性</el-tag>
            <el-tag v-else-if="rule.type === 'nodeAffinity'" type="success">Node 亲和性</el-tag>
            <el-tag v-else type="danger">Node 反亲和性</el-tag>
          </div>
          <el-button type="danger" :icon="Delete" size="small" @click="emit('removeAffinityRule', rIndex)">删除</el-button>
        </div>
        <div class="affinity-rule-body">
          <div class="rule-detail-row" v-if="rule.namespaces && rule.namespaces.length > 0">
            <span class="detail-label">Namespaces:</span>
            <span class="detail-value">{{ rule.namespaces.join(', ') }}</span>
          </div>
          <div class="rule-detail-row" v-if="rule.topologyKey && rule.type.includes('pod')">
            <span class="detail-label">拓扑键:</span>
            <span class="detail-value">{{ rule.topologyKey }}</span>
          </div>
          <div class="rule-detail-row">
            <span class="detail-label">优先级:</span>
            <span class="detail-value">{{ rule.priority }}</span>
            <span v-if="rule.priority === 'Preferred'" class="detail-label" style="margin-left: 20px;">权重:</span>
            <span v-if="rule.priority === 'Preferred'" class="detail-value">{{ rule.weight }}</span>
          </div>
          <div class="rule-expressions-section">
            <div class="expressions-title">Match Expressions:</div>
            <div v-for="(exp, eIndex) in rule.matchExpressions" :key="'aff-exp-'+rIndex+'-'+eIndex" class="rule-expression-item">
              <span class="exp-key">{{ exp.key }}</span>
              <span class="exp-operator">{{ exp.operator }}</span>
              <span class="exp-values">{{ exp.valueStr }}</span>
            </div>
          </div>
          <div class="rule-labels-section" v-if="rule.matchLabels && rule.matchLabels.length > 0">
            <div class="labels-title">Match Labels:</div>
            <div class="rule-labels-list">
              <span v-for="(label, lIndex) in rule.matchLabels" :key="'aff-label-'+rIndex+'-'+lIndex" class="rule-label-item">
                {{ label.key }}={{ label.value }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

interface AffinityRule {
  type: 'podAffinity' | 'podAntiAffinity' | 'nodeAffinity' | 'nodeAntiAffinity'
  namespaces?: string[]
  topologyKey?: string
  priority: 'Required' | 'Preferred'
  weight?: number
  matchExpressions: { key: string; operator: string; valueStr: string }[]
  matchLabels: { key: string; value: string }[]
}

const props = defineProps<{
  affinityRules: AffinityRule[]
  editingAffinityRule: AffinityRule | null
  namespaceList: { name: string }[]
}>()

const emit = defineEmits<{
  startAddAffinity: [type: 'pod' | 'node']
  cancelAffinityEdit: []
  saveAffinityRule: []
  addMatchExpression: []
  removeMatchExpression: [index: number]
  addMatchLabel: []
  removeMatchLabel: [index: number]
  removeAffinityRule: [index: number]
}>()
</script>

<style scoped>
.affinity-wrapper {
  padding: 0;
  background: transparent;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  color: #111827;
}

.affinity-action-buttons {
  display: flex;
  gap: 12px;
  margin-bottom: 18px;
  padding-bottom: 18px;
  border-bottom: 1px solid #e5e7eb;
  flex-shrink: 0;
}

.affinity-action-buttons :deep(.el-button) {
  height: 36px;
  padding: 0 16px;
  border: 1px solid #111827;
  border-radius: 8px;
  background: #111827;
  color: #ffffff;
  font-weight: 700;
  box-shadow: none;
}

.affinity-action-buttons :deep(.el-button:hover) {
  border-color: #374151;
  background: #374151;
  color: #ffffff;
  box-shadow: none;
}

.affinity-config-container {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 20px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.config-container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  background: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  flex-shrink: 0;
}

.config-type-badge {
  display: flex;
  align-items: center;
}

.config-header-actions {
  display: flex;
  gap: 8px;
}

.config-header-actions :deep(.el-button--primary) {
  border-color: #111827;
  background: #111827;
  color: #ffffff;
  box-shadow: none;
}

.config-header-actions :deep(.el-button--primary:hover) {
  border-color: #374151;
  background: #374151;
  color: #ffffff;
}

.config-header-actions :deep(.el-button--default) {
  border-color: #d1d5db;
  background: #ffffff;
  color: #374151;
}

.config-container-body {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.config-form-section {
  margin-bottom: 18px;
  padding: 18px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #ffffff;
}

.config-form-section:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 12px;
}

.affinity-type-radio {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.affinity-radio-item {
  margin: 0 !important;
  padding: 10px 18px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: #fff;
  transition: all 0.2s;
}

.affinity-radio-item:hover {
  border-color: #111827;
  background: #f9fafb;
}

.affinity-radio-item.is-checked {
  border-color: #111827;
  background: #f9fafb;
}

.full-width-input {
  width: 100%;
}

.form-tip {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f9fafb;
  border-left: 3px solid #111827;
  border-radius: 4px;
  font-size: 12px;
  color: #4b5563;
  line-height: 1.5;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header :deep(.el-button--primary) {
  height: 32px;
  padding: 0 14px;
  border-color: #111827;
  border-radius: 8px;
  background: #111827;
  color: #ffffff;
  font-weight: 700;
  box-shadow: none;
}

.section-header :deep(.el-button--primary:hover) {
  border-color: #374151;
  background: #374151;
  color: #ffffff;
}

.expressions-list,
.labels-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.expression-config-row {
  background: #f9fafb;
  padding: 16px;
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  transition: all 0.2s;
}

.expression-config-row:hover {
  border-color: #d1d5db;
  box-shadow: none;
}

.expression-config-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 12px;
}

.config-grid-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-grid-label {
  font-size: 13px;
  font-weight: 700;
  color: #4b5563;
}

.expression-config-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.label-config-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f9fafb;
  border-radius: 10px;
  border: 1px solid #e5e7eb;
}

.label-config-grid {
  display: flex;
  align-items: center;
  flex: 1;
  gap: 12px;
}

.label-separator {
  color: #909399;
  font-weight: 500;
}

.affinity-rules-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
  flex: 1;
}

.affinity-rules-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e5e7eb;
}

.header-title {
  font-size: 16px;
  font-weight: 800;
  color: #111827;
}

.affinity-rule-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}

.affinity-rule-card:hover {
  border-color: #d1d5db;
  box-shadow: none;
}

.affinity-rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  background: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
}

.rule-type-badge {
  display: flex;
  align-items: center;
}

.affinity-rule-body {
  padding: 20px;
}

.rule-detail-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.rule-detail-row:last-child {
  margin-bottom: 0;
}

.detail-label {
  font-size: 14px;
  font-weight: 700;
  color: #4b5563;
  min-width: 100px;
}

.detail-value {
  font-size: 14px;
  color: #111827;
  font-weight: 600;
}

.rule-expressions-section,
.rule-labels-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.expressions-title,
.labels-title {
  font-size: 13px;
  font-weight: 700;
  color: #4b5563;
  margin-bottom: 12px;
}

.rule-expression-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  margin-bottom: 8px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
}

.exp-key {
  color: #111827;
  font-weight: 700;
}

.exp-operator {
  color: #2563eb;
  font-weight: 700;
}

.exp-values {
  color: #4b5563;
}

.rule-labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.rule-label-item {
  padding: 4px 12px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  font-size: 12px;
  color: #4b5563;
  font-family: 'Monaco', 'Menlo', monospace;
}

.affinity-wrapper :deep(.el-button--danger) {
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

.affinity-wrapper :deep(.el-button--danger span) {
  display: none;
}

.affinity-wrapper :deep(.el-button--danger:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
  box-shadow: none;
}

.affinity-wrapper :deep(.el-input__wrapper),
.affinity-wrapper :deep(.el-select .el-input__wrapper) {
  border-color: #d1d5db;
  box-shadow: none;
}

.affinity-wrapper :deep(.el-input__wrapper:hover),
.affinity-wrapper :deep(.el-select .el-input__wrapper:hover) {
  border-color: #9ca3af;
}

.affinity-wrapper :deep(.el-input__wrapper.is-focus),
.affinity-wrapper :deep(.el-select .el-input__wrapper.is-focus) {
  border-color: #111827;
  box-shadow: 0 0 0 2px rgba(17, 24, 39, 0.08);
}
</style>
