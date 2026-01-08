<template>
  <div class="scaling-strategy-wrapper">
    <div class="strategy-section">
      <el-radio-group v-model="formData.strategyType" class="strategy-radio">
        <el-radio value="RollingUpdate" class="strategy-radio-item">
          <span class="radio-label">滚动升级</span>
        </el-radio>
        <el-radio value="Recreate" class="strategy-radio-item">
          <span class="radio-label">重新创建</span>
        </el-radio>
      </el-radio-group>
    </div>

    <div class="strategy-form-grid">
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">最大 Pod 数量</label>
          <el-input
            v-model="formData.maxSurge"
            placeholder="例如: 3 或 25%"
            class="grid-input"
          />
        </div>
        <div class="form-grid-item">
          <label class="form-grid-label">最大不可用数量</label>
          <el-input
            v-model="formData.maxUnavailable"
            placeholder="例如: 1 或 25%"
            class="grid-input"
          />
        </div>
      </div>
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">最小就绪时间(秒)</label>
          <el-input
            v-model="formData.minReadySeconds"
            placeholder="最小就绪时间"
            class="grid-input"
          />
        </div>
        <div class="form-grid-item">
          <label class="form-grid-label">进程截止时间(秒)</label>
          <el-input
            v-model="formData.progressDeadlineSeconds"
            placeholder="进程截止时间"
            class="grid-input"
          />
        </div>
      </div>
      <div class="form-grid-row">
        <div class="form-grid-item full-width">
          <label class="form-grid-label">修订历史记录限制</label>
          <el-input
            v-model="formData.revisionHistoryLimit"
            placeholder="修订历史记录限制"
            class="grid-input"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface FormData {
  strategyType: string
  maxSurge: string
  maxUnavailable: string
  minReadySeconds: string
  progressDeadlineSeconds: string
  revisionHistoryLimit: string
}

const props = defineProps<{
  formData: FormData
}>()
</script>

<style scoped>
.scaling-strategy-wrapper {
  padding: 24px 32px;
  background: #fff;
}

.strategy-section {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e4e7ed;
}

.strategy-radio {
  display: flex;
  gap: 16px;
}

.strategy-radio :deep(.el-radio-group) {
  display: flex;
  gap: 16px;
}

.strategy-radio-item {
  margin: 0 !important;
  padding: 12px 24px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fff;
  transition: all 0.3s;
  display: flex;
  align-items: center;
}

.strategy-radio-item:hover {
  border-color: #409eff;
  background: #ecf5ff;
}

.strategy-radio-item.is-checked {
  border-color: #409eff;
  background: #ecf5ff;
}

.strategy-radio-item :deep(.el-radio__label) {
  padding-left: 8px;
}

.strategy-radio-item :deep(.el-radio__input) {
  transform: scale(1.15);
}

.strategy-radio-item :deep(.el-radio__input.is-checked .el-radio__inner) {
  background: #409eff;
  border-color: #409eff;
}

.strategy-radio-item :deep(.el-radio__inner) {
  border-color: #dcdfe6;
  transition: all 0.3s;
}

.radio-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.strategy-form-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-grid-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.form-grid-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-grid-item.full-width {
  grid-column: 1 / -1;
}

.form-grid-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  line-height: 1.5;
}

.grid-input {
  width: 100%;
}

.grid-input :deep(.el-input__wrapper) {
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s;
  background-color: #fff;
  padding: 8px 15px;
}

.grid-input :deep(.el-input__wrapper:hover) {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
}

.grid-input :deep(.el-input__wrapper.is-focus) {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.3);
}

.grid-input :deep(.el-input__inner) {
  font-size: 14px;
  color: #303133;
}
</style>
