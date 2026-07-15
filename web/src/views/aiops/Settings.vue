<template>
  <div class="aiops-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><SetUp /></el-icon>
        </div>
        <div>
          <h2 class="page-title">AI配置</h2>
          <p class="page-subtitle">配置 OpenAI-compatible 模型，支持 DeepSeek、通义千问、OpenAI、Ollama 等接口</p>
        </div>
      </div>
      <el-button class="black-button" @click="handleCreate">
        <el-icon><Plus /></el-icon>
        新增模型配置
      </el-button>
    </div>

    <div class="tips-card">
      <div class="tip-title">配置说明</div>
      <div class="tip-text">
        Base URL 支持填写到服务根路径或 `/v1`，系统会自动拼接 `/chat/completions`。推理强度留空时不会向模型传递该参数。
      </div>
    </div>

    <div class="table-wrapper">
      <el-table v-loading="loading" :data="providers" class="modern-table">
        <el-table-column label="配置名称" min-width="180">
          <template #default="{ row }">
            <div class="provider-cell">
              <div class="provider-icon">
                <el-icon><DataAnalysis /></el-icon>
              </div>
              <div>
                <div class="provider-name">{{ row.name }}</div>
                <div class="provider-meta">{{ row.provider || 'openai-compatible' }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" min-width="150" />
        <el-table-column label="推理" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.reasoningEffort" effect="plain">{{ reasoningEffortText(row.reasoningEffort) }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="baseUrl" label="Base URL" min-width="260" show-overflow-tooltip />
        <el-table-column prop="apiKey" label="API Key" min-width="150" />
        <el-table-column label="状态" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="plain">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="默认" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="warning" effect="plain">默认</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="lastTestMsg" label="最近测试" min-width="180" show-overflow-tooltip />
        <el-table-column label="操作" width="250" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button link class="action-btn action-view" :loading="testingId === row.id" @click="handleTest(row)">
                <el-icon><Connection /></el-icon>
              </el-button>
              <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑模型配置' : '新增模型配置'" width="760px" class="aiops-dialog">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="form.name" placeholder="例如：DeepSeek Chat" />
        </el-form-item>
        <el-form-item label="供应商类型">
          <el-select v-model="form.provider" class="full-width">
            <el-option label="OpenAI Compatible" value="openai-compatible" />
          </el-select>
        </el-form-item>
        <el-form-item label="Base URL" prop="baseUrl">
          <el-input v-model="form.baseUrl" placeholder="例如：https://api.deepseek.com/v1" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="form.apiKey" type="password" show-password placeholder="编辑时不填则保留原 API Key" />
        </el-form-item>
        <el-form-item label="模型名称" prop="model">
          <el-input v-model="form.model" placeholder="例如：deepseek-chat、qwen-plus、gpt-4.1-mini" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="温度">
            <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" controls-position="right" />
          </el-form-item>
          <el-form-item label="最大Token">
            <el-input-number v-model="form.maxTokens" :min="256" :max="32000" :step="256" controls-position="right" />
          </el-form-item>
          <el-form-item label="超时秒数">
            <el-input-number v-model="form.timeout" :min="5" :max="600" controls-position="right" />
          </el-form-item>
          <el-form-item label="推理强度">
            <el-select v-model="form.reasoningEffort" class="full-width" placeholder="默认，不传递">
              <el-option label="默认（不传递）" value="" />
              <el-option label="Minimal" value="minimal" />
              <el-option label="Low" value="low" />
              <el-option label="Medium" value="medium" />
              <el-option label="High" value="high" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="form.isDefault" active-text="默认模型" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" placeholder="可填写模型用途或部署说明" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button class="black-button" :loading="saving" @click="handleSave">保存配置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Connection, DataAnalysis, Delete, Edit, Plus, SetUp } from '@element-plus/icons-vue'
import {
  createAIProvider,
  deleteAIProvider,
  getAIProviders,
  testAIProvider,
  updateAIProvider,
  type AIProvider,
  type AIProviderPayload
} from '@/api/aiops'

const loading = ref(false)
const saving = ref(false)
const testingId = ref<number | null>(null)
const providers = ref<AIProvider[]>([])
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<AIProviderPayload & { id?: number }>({
  name: '',
  provider: 'openai-compatible',
  baseUrl: '',
  apiKey: '',
  model: '',
  temperature: 0.2,
  maxTokens: 8192,
  timeout: 180,
  reasoningEffort: '',
  enabled: true,
  isDefault: true,
  remark: ''
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  baseUrl: [{ required: true, message: '请输入 Base URL', trigger: 'blur' }],
  model: [{ required: true, message: '请输入模型名称', trigger: 'blur' }]
}

const loadProviders = async () => {
  loading.value = true
  try {
    providers.value = await getAIProviders()
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(form, {
    id: undefined,
    name: '',
    provider: 'openai-compatible',
    baseUrl: '',
    apiKey: '',
    model: '',
    temperature: 0.2,
    maxTokens: 8192,
    timeout: 180,
    reasoningEffort: '',
    enabled: true,
    isDefault: providers.value.length === 0,
    remark: ''
  })
}

const handleCreate = () => {
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (row: AIProvider) => {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    provider: row.provider || 'openai-compatible',
    baseUrl: row.baseUrl,
    apiKey: row.apiKey || '',
    model: row.model,
    temperature: row.temperature ?? 0.2,
    maxTokens: row.maxTokens || 8192,
    timeout: row.timeout || 180,
    reasoningEffort: row.reasoningEffort || '',
    enabled: row.enabled,
    isDefault: row.isDefault,
    remark: row.remark || ''
  })
  dialogVisible.value = true
}

const handleSave = async () => {
  await formRef.value?.validate()
  saving.value = true
  try {
    const payload: AIProviderPayload = { ...form }
    if (form.id) {
      await updateAIProvider(form.id, payload)
    } else {
      await createAIProvider(payload)
    }
    ElMessage.success('AI配置已保存')
    dialogVisible.value = false
    await loadProviders()
  } finally {
    saving.value = false
  }
}

const handleTest = async (row: AIProvider) => {
  testingId.value = row.id
  try {
    await testAIProvider(row.id)
    ElMessage.success('模型连接正常')
    await loadProviders()
  } finally {
    testingId.value = null
  }
}

const handleDelete = async (row: AIProvider) => {
  await ElMessageBox.confirm(`确认删除模型配置「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonClass: 'black-confirm-button'
  })
  await deleteAIProvider(row.id)
  ElMessage.success('已删除')
  await loadProviders()
}

const reasoningEffortText = (value?: string) => {
  const map: Record<string, string> = {
    minimal: 'Minimal',
    low: 'Low',
    medium: 'Medium',
    high: 'High'
  }
  return map[String(value || '').toLowerCase()] || value || '-'
}

onMounted(loadProviders)
</script>

<style scoped>
.aiops-page {
  min-height: 100%;
}

.page-header,
.tips-card,
.table-wrapper {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 16px;
  box-shadow: 0 10px 30px rgba(31, 45, 61, 0.06);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px;
  margin-bottom: 16px;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 14px;
}

.page-title-icon,
.provider-icon {
  width: 44px;
  height: 44px;
  border-radius: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #111827;
  color: #fff;
  font-size: 20px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  color: #1f2937;
}

.page-subtitle,
.provider-meta,
.tip-text {
  color: #6b7280;
  font-size: 13px;
  margin-top: 4px;
}

.tips-card {
  padding: 16px 18px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, #fff, #f8fafc);
}

.tip-title {
  font-weight: 700;
  color: #111827;
  margin-bottom: 4px;
}

.table-wrapper {
  padding: 16px;
}

.provider-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.provider-icon {
  width: 36px;
  height: 36px;
  font-size: 16px;
  background: #eef2ff;
  color: #3730a3;
}

.provider-name {
  color: #111827;
  font-weight: 700;
}

.black-button {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 10px;
}

.action-btn {
  width: 30px;
  height: 30px;
  border-radius: 9px;
}

.action-view {
  color: #2563eb;
}

.action-edit {
  color: #111827;
}

.action-delete {
  color: #dc2626;
}

.full-width {
  width: 100%;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 18px;
  row-gap: 2px;
}

.form-grid :deep(.el-form-item) {
  min-width: 0;
}

.form-grid :deep(.el-input-number) {
  width: 100%;
}

.form-grid :deep(.el-input-number .el-input__inner) {
  text-align: left;
}

:deep(.modern-table .el-table__header th) {
  background: #f8fafc;
  color: #475569;
  font-weight: 700;
}

:deep(.black-confirm-button) {
  background: #111827 !important;
  border-color: #111827 !important;
  color: #fff !important;
}
</style>
