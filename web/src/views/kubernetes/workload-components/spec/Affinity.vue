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
  padding: 24px 32px;
  background: #fff;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.affinity-action-buttons {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.affinity-config-container {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
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
  padding: 16px 20px;
  background: linear-gradient(to right, #f8f9fa, #ffffff);
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.config-type-badge {
  display: flex;
  align-items: center;
}

.config-header-actions {
  display: flex;
  gap: 12px;
}

.config-container-body {
  padding: 24px;
  flex: 1;
  overflow-y: auto;
}

.config-form-section {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e4e7ed;
}

.config-form-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 12px;
}

.affinity-type-radio {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.affinity-radio-item {
  margin: 0 !important;
  padding: 10px 20px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fff;
  transition: all 0.3s;
}

.affinity-radio-item:hover {
  border-color: #409eff;
  background: #ecf5ff;
}

.affinity-radio-item.is-checked {
  border-color: #409eff;
  background: #ecf5ff;
}

.full-width-input {
  width: 100%;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.expressions-list,
.labels-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.expression-config-row {
  background: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  transition: all 0.3s;
}

.expression-config-row:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
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
  font-weight: 500;
  color: #606266;
}

.expression-config-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid #e4e7ed;
}

.label-config-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
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
  border-bottom: 1px solid #e4e7ed;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.affinity-rule-card {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s;
}

.affinity-rule-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.15);
}

.affinity-rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(to right, #f8f9fa, #ffffff);
  border-bottom: 1px solid #e4e7ed;
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
  font-weight: 500;
  color: #606266;
  min-width: 100px;
}

.detail-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.rule-expressions-section,
.rule-labels-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
}

.expressions-title,
.labels-title {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
  margin-bottom: 12px;
}

.rule-expression-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #fff;
  border-radius: 6px;
  margin-bottom: 8px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
}

.exp-key {
  color: #303133;
  font-weight: 500;
}

.exp-operator {
  color: #409eff;
  font-weight: 500;
}

.exp-values {
  color: #606266;
}

.rule-labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.rule-label-item {
  padding: 4px 12px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', monospace;
}
</style>
