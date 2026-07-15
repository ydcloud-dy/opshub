<template>
  <div class="provider-select">
    <div class="provider-shell">
      <div class="provider-icon" aria-hidden="true">
        <el-icon><DataAnalysis /></el-icon>
      </div>
      <div class="provider-main">
        <div class="provider-select-head">
          <span class="provider-label">模型</span>
          <span v-if="currentOption?.isDefault" class="provider-badge">默认</span>
        </div>
        <el-select
          v-model="selectedId"
          class="provider-field"
          filterable
          :loading="loading"
          :placeholder="placeholder"
          popper-class="ai-provider-select-dropdown"
        >
          <el-option
            v-for="item in options"
            :key="item.id"
            :label="formatLabel(item)"
            :value="item.id"
          >
            <div class="provider-option">
              <div class="provider-option-main">
                <strong>{{ item.name }}</strong>
                <span>{{ item.provider }} / {{ item.model }}</span>
              </div>
              <span v-if="item.isDefault" class="provider-option-badge">默认</span>
            </div>
          </el-option>
        </el-select>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { DataAnalysis } from '@element-plus/icons-vue'
import { getAIProviderOptions, type AIProviderOption } from '@/api/aiops'

const props = defineProps<{
  modelValue?: number
  placeholder?: string
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value?: number): void
}>()

const loading = ref(false)
const options = ref<AIProviderOption[]>([])
const selectedId = ref<number | undefined>(props.modelValue)

const loadOptions = async () => {
  loading.value = true
  try {
    options.value = await getAIProviderOptions()
    if (selectedId.value && options.value.some(item => item.id === selectedId.value)) {
      return
    }
    const defaultOption = options.value.find(item => item.isDefault) || options.value[0]
    selectedId.value = defaultOption?.id
  } finally {
    loading.value = false
  }
}

const currentOption = computed(() => options.value.find(item => item.id === selectedId.value))

const formatLabel = (item: AIProviderOption) => {
  return `${item.name} · ${item.model}`
}

watch(
  () => props.modelValue,
  value => {
    if (value !== selectedId.value) {
      selectedId.value = value
    }
  }
)

watch(selectedId, value => {
  if (value !== props.modelValue) {
    emit('update:modelValue', value)
  }
})

onMounted(loadOptions)
</script>

<style scoped>
.provider-select {
  width: 232px;
  max-width: 100%;
}

.provider-shell {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 48px;
  padding: 7px 10px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.05);
  transition: border-color 0.16s ease, box-shadow 0.16s ease, background 0.16s ease;
}

.provider-shell:hover {
  border-color: #cbd5e1;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.08);
}

.provider-icon {
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #0f172a;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 14px;
}

.provider-main {
  min-width: 0;
  flex: 1;
}

.provider-select-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 14px;
  margin-bottom: 1px;
}

.provider-label {
  color: #94a3b8;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
}

.provider-badge {
  font-size: 10px;
  color: #111827;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  padding: 1px 6px;
  line-height: 1.2;
}

.provider-field {
  width: 100%;
}

.provider-field :deep(.el-select__wrapper) {
  min-height: 22px;
  padding: 0;
  border: 0;
  box-shadow: none;
  background: transparent;
  line-height: 22px;
}

.provider-field :deep(.el-select__selected-item) {
  color: #0f172a;
  font-size: 12px;
  font-weight: 800;
  line-height: 22px;
}

.provider-field :deep(.el-select__placeholder) {
  color: #94a3b8;
  font-size: 12px;
  font-weight: 700;
}

.provider-field :deep(.el-select__suffix) {
  color: #64748b;
}

.provider-field :deep(.el-select__selected-item span) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
}

.provider-option-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.25;
}

.provider-option-main strong {
  color: #111827;
  font-size: 13px;
}

.provider-option-main span {
  color: #64748b;
  font-size: 12px;
}

.provider-option-badge {
  flex: 0 0 auto;
  color: #92400e;
  background: #fef3c7;
  border: 1px solid #fde68a;
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 11px;
  line-height: 1.3;
}

:global(.ai-provider-select-dropdown .el-select-dropdown__item) {
  height: auto;
  min-height: 50px;
  padding: 8px 12px;
}

:global(.ai-provider-select-dropdown .el-select-dropdown__item.is-selected .provider-option-main strong) {
  color: #2563eb;
}

:global(.ai-provider-select-dropdown .el-select-dropdown__item.is-hovering) {
  background: #f8fafc;
}

:global(.assistant-hero) .provider-shell {
  border-color: rgba(148, 163, 184, 0.28);
  background: rgba(255, 255, 255, 0.10);
  box-shadow: none;
}

:global(.assistant-hero) .provider-shell:hover {
  border-color: rgba(251, 191, 36, 0.72);
  background: rgba(255, 255, 255, 0.12);
}

:global(.assistant-hero) .provider-icon {
  color: #f8fafc;
  background: rgba(255, 255, 255, 0.10);
  border-color: rgba(255, 255, 255, 0.18);
}

:global(.assistant-hero) .provider-label,
:global(.assistant-hero) .provider-field :deep(.el-select__selected-item),
:global(.assistant-hero) .provider-field :deep(.el-select__placeholder),
:global(.assistant-hero) .provider-field :deep(.el-select__suffix) {
  color: #e2e8f0;
}

@media (max-width: 720px) {
  .provider-select {
    width: 100%;
  }
}
</style>
