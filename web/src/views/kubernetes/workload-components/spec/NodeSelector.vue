<template>
  <div class="node-selector-wrapper">
    <!-- 调度类型选择 -->
    <div class="scheduling-type-content">
      <el-radio-group v-model="localSchedulingType" class="scheduling-type-radio" @change="handleSchedulingTypeChange">
        <el-radio value="any" class="scheduling-radio-item" @click="selectSchedulingType('any')">
          <span class="radio-label">任意可用节点</span>
        </el-radio>
        <el-radio value="specified" class="scheduling-radio-item" @click="selectSchedulingType('specified')">
          <span class="radio-label">指定节点</span>
        </el-radio>
        <el-radio value="match" class="scheduling-radio-item" @click="selectSchedulingType('match')">
          <span class="radio-label">调度规则匹配</span>
        </el-radio>
      </el-radio-group>
      <div class="scheduling-type-tip">
        当前选择: {{ getSchedulingTypeLabel(localSchedulingType) }} | 规则数量: {{ formData.matchRules?.length || 0 }}
      </div>
    </div>

    <!-- 指定节点 -->
    <div v-if="localSchedulingType === 'specified'" class="node-config-section">
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">节点名称</label>
          <el-select
            v-model="localSpecifiedNode"
            placeholder="请选择节点"
            class="grid-input"
            filterable
            @change="handleSpecifiedNodeChange"
          >
            <el-option
              v-for="node in nodeList"
              :key="node.name"
              :label="node.name"
              :value="node.name"
            />
          </el-select>
        </div>
      </div>
    </div>

    <!-- 调度规则匹配 -->
    <div v-if="localSchedulingType === 'match'" class="match-rules-section">
      <div class="match-rules-header">
        <span>根据节点标签匹配调度规则</span>
        <el-button type="primary" :icon="Plus" size="small" @click="emit('addMatchRule')">添加规则</el-button>
      </div>
      <div class="match-rules-list">
        <div v-for="(rule, index) in formData.matchRules" :key="'rule-'+index" class="match-rule-item">
          <div class="rule-row">
            <div class="form-grid-item">
              <label class="form-grid-label">键</label>
              <el-select
                v-model="rule.key"
                placeholder="选择或输入键"
                class="grid-input"
                filterable
                allow-create
              >
                <el-option
                  v-for="label in commonNodeLabels"
                  :key="label.key"
                  :label="label.key"
                  :value="label.key"
                >
                  <span class="label-option">
                    <span class="label-key">{{ label.key }}</span>
                    <span class="label-separator">:</span>
                    <span class="label-value">{{ label.value }}</span>
                  </span>
                </el-option>
              </el-select>
            </div>
            <div class="form-grid-item">
              <label class="form-grid-label">操作符</label>
              <el-select v-model="rule.operator" placeholder="选择操作符" class="grid-input">
                <el-option label="等于" value="In" />
                <el-option label="不等于" value="NotIn" />
                <el-option label="存在" value="Exists" />
                <el-option label="不存在" value="DoesNotExist" />
                <el-option label="大于" value="Gt" />
                <el-option label="小于" value="Lt" />
              </el-select>
            </div>
          </div>
          <div class="rule-row" v-if="rule.operator !== 'Exists' && rule.operator !== 'DoesNotExist'">
            <div class="form-grid-item">
              <label class="form-grid-label">值</label>
              <el-input
                v-model="rule.value"
                placeholder="请输入值"
                class="grid-input"
              />
            </div>
          </div>
          <div class="rule-actions">
            <el-button type="danger" :icon="Delete" size="small" @click="emit('removeMatchRule', index)">删除</el-button>
          </div>
        </div>
        <el-empty v-if="formData.matchRules.length === 0" description="暂无匹配规则" :image-size="80" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'
import { ref, watch, nextTick } from 'vue'

interface MatchRule {
  key: string
  operator: string
  value: string
}

interface FormData {
  schedulingType: string
  specifiedNode: string
  matchRules: MatchRule[]
}

const props = defineProps<{
  formData: FormData
  nodeList: { name: string }[]
  commonNodeLabels: { key: string; value: string }[]
}>()

const emit = defineEmits<{
  addMatchRule: []
  removeMatchRule: [index: number]
  update: [data: { schedulingType: string; specifiedNode: string }]
}>()

// 本地状态
const localSchedulingType = ref(props.formData.schedulingType || 'any')
const localSpecifiedNode = ref(props.formData.specifiedNode || '')

const selectSchedulingType = (type: string) => {
  if (localSchedulingType.value === type) {
    handleSchedulingTypeChange(type)
    return
  }
  localSchedulingType.value = type
  handleSchedulingTypeChange(type)
}

// 监听 props 变化
watch(() => props.formData, (newData) => {
  localSchedulingType.value = newData.schedulingType || 'any'
  localSpecifiedNode.value = newData.specifiedNode || ''
}, { immediate: true, deep: true })

// 处理调度类型变化
const handleSchedulingTypeChange = async (newType: string) => {
  localSchedulingType.value = newType

  // 如果切换到非"指定节点"类型，清空指定节点
  if (newType !== 'specified') {
    localSpecifiedNode.value = ''
  }

  // 等待 DOM 更新后再 emit
  await nextTick()

  // 通知父组件更新
  emit('update', {
    schedulingType: newType,
    specifiedNode: localSpecifiedNode.value
  })
}

// 处理指定节点变化
const handleSpecifiedNodeChange = (node: string) => {

  // 通知父组件更新
  emit('update', {
    schedulingType: localSchedulingType.value,
    specifiedNode: node
  })
}

// 获取调度类型标签
const getSchedulingTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    'any': '任意可用节点',
    'specified': '指定节点',
    'match': '调度规则匹配'
  }
  return labels[type] || type
}

</script>

<style scoped>
.node-selector-wrapper {
  padding: 0;
  background: transparent;
}

.scheduling-type-content {
  margin-bottom: 20px;
}

.scheduling-type-radio {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.scheduling-radio-item {
  margin: 0 !important;
  padding: 12px 24px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fff;
  transition: all 0.3s;
  display: flex;
  align-items: center;
}

.scheduling-radio-item:hover {
  border-color: #409eff;
  background: #ecf5ff;
}

.scheduling-radio-item.is-checked {
  border-color: #409eff;
  background: #ecf5ff;
}

.radio-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.scheduling-type-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}

.node-config-section {
  margin-top: 24px;
}

.match-rules-section {
  margin-top: 24px;
}

.match-rules-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.match-rules-header span {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.match-rules-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.match-rule-item {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 20px;
  transition: all 0.3s;
}

.match-rule-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}

.rule-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 16px;
}

.rule-row:last-of-type {
  margin-bottom: 0;
}

.form-grid-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-grid-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.grid-input {
  width: 100%;
}

.rule-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
}

.label-option {
  display: flex;
  align-items: center;
  gap: 4px;
}

.label-key {
  color: #303133;
  font-weight: 500;
}

.label-separator {
  color: #909399;
}

.label-value {
  color: #606266;
  font-size: 12px;
}
</style>
