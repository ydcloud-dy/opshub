<template>
  <div class="env-config-content">
    <div class="env-tabs-wrapper">
      <el-tabs v-model="activeEnvTab" class="env-type-tabs">
        <el-tab-pane label="普通变量" name="normal">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <el-icon><Key /></el-icon>
                普通环境变量
              </span>
              <el-button type="primary" @click="showAddEnvDialog('normal')" :icon="Plus" size="default">
                添加变量
              </el-button>
            </div>
            <div v-if="normalEnvs.length > 0" class="env-table-wrapper">
              <el-table :data="normalEnvs" class="env-table" size="default">
                <el-table-column label="名称" prop="name" min-width="180">
                  <template #default="{ row }">
                    <span class="env-name">{{ row.name }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="值" prop="value" min-width="250">
                  <template #default="{ row }">
                    <div class="env-value-cell">
                      <el-input
                        v-if="row.editing"
                        v-model="row.tempValue"
                        placeholder="请输入变量值"
                        size="small"
                      />
                      <span v-else class="env-value">{{ row.value || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="150" align="center">
                  <template #default="{ row, $index }">
                    <div class="action-buttons">
                      <template v-if="row.editing">
                        <el-button type="success" link size="small" @click="saveEnvEdit(row, $index)">
                          <el-icon><Select /></el-icon>
                          保存
                        </el-button>
                        <el-button type="info" link size="small" @click="cancelEnvEdit(row, $index)">
                          <el-icon><Close /></el-icon>
                          取消
                        </el-button>
                      </template>
                      <template v-else>
                        <el-button type="primary" link size="small" @click="editEnv(row, $index)">
                          <el-icon><Edit /></el-icon>
                          编辑
                        </el-button>
                        <el-button type="danger" link size="small" @click="removeEnv('normal', $index)">
                          <el-icon><Delete /></el-icon>
                          删除
                        </el-button>
                      </template>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无环境变量配置" :image-size="80" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="配置映射引用" name="configmap">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <el-icon><Document /></el-icon>
                ConfigMap 引用
              </span>
              <el-button type="primary" @click="showAddEnvDialog('configmap')" :icon="Plus" size="default">
                添加引用
              </el-button>
            </div>
            <div v-if="configmapEnvs.length > 0" class="env-table-wrapper">
              <el-table :data="configmapEnvs" class="env-table" size="default">
                <el-table-column label="变量名称" prop="name" min-width="180">
                  <template #default="{ row }">
                    <span class="env-name">{{ row.name }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="ConfigMap" prop="configmapName" min-width="150">
                  <template #default="{ row }">
                    <span class="env-resource">{{ row.configmapName }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="Key" prop="key" min-width="150">
                  <template #default="{ row }">
                    <span class="env-key">{{ row.key }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="150" align="center">
                  <template #default="{ row, $index }">
                    <div class="action-buttons">
                      <el-button type="primary" link size="small" @click="editConfigMapEnv(row, $index)">
                        <el-icon><Edit /></el-icon>
                        编辑
                      </el-button>
                      <el-button type="danger" link size="small" @click="removeEnv('configmap', $index)">
                        <el-icon><Delete /></el-icon>
                        删除
                      </el-button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无 ConfigMap 引用" :image-size="80" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="密钥引用" name="secret">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <el-icon><Lock /></el-icon>
                Secret 引用
              </span>
              <el-button type="primary" @click="showAddEnvDialog('secret')" :icon="Plus" size="default">
                添加引用
              </el-button>
            </div>
            <div v-if="secretEnvs.length > 0" class="env-table-wrapper">
              <el-table :data="secretEnvs" class="env-table" size="default">
                <el-table-column label="变量名称" prop="name" min-width="180">
                  <template #default="{ row }">
                    <span class="env-name">{{ row.name }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="Secret" prop="secretName" min-width="150">
                  <template #default="{ row }">
                    <span class="env-resource">{{ row.secretName }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="Key" prop="key" min-width="150">
                  <template #default="{ row }">
                    <span class="env-key">{{ row.key }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="150" align="center">
                  <template #default="{ row, $index }">
                    <div class="action-buttons">
                      <el-button type="primary" link size="small" @click="editSecretEnv(row, $index)">
                        <el-icon><Edit /></el-icon>
                        编辑
                      </el-button>
                      <el-button type="danger" link size="small" @click="removeEnv('secret', $index)">
                        <el-icon><Delete /></el-icon>
                        删除
                      </el-button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无 Secret 引用" :image-size="80" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 添加/编辑普通变量对话框 -->
    <el-dialog
      v-model="normalEnvDialogVisible"
      :title="editingEnvIndex >= 0 ? '编辑环境变量' : '添加环境变量'"
      width="600px"
    >
      <el-form :model="normalEnvForm" label-width="100px" label-position="left">
        <el-form-item label="变量名称" required>
          <el-input v-model="normalEnvForm.name" placeholder="例如: DATABASE_URL" clearable />
        </el-form-item>
        <el-form-item label="变量值" required>
          <el-input
            v-model="normalEnvForm.value"
            type="textarea"
            :rows="3"
            placeholder="请输入变量值"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="normalEnvDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveNormalEnv">确定</el-button>
      </template>
    </el-dialog>

    <!-- 添加/编辑 ConfigMap 引用对话框 -->
    <el-dialog
      v-model="configmapEnvDialogVisible"
      :title="editingConfigMapIndex >= 0 ? '编辑 ConfigMap 引用' : '添加 ConfigMap 引用'"
      width="600px"
    >
      <el-form :model="configmapEnvForm" label-width="120px" label-position="left">
        <el-form-item label="变量名称" required>
          <el-input v-model="configmapEnvForm.name" placeholder="环境变量名称" clearable />
        </el-form-item>
        <el-form-item label="ConfigMap" required>
          <el-select
            v-model="configmapEnvForm.configmapName"
            placeholder="选择 ConfigMap"
            style="width: 100%"
            filterable
          >
            <el-option
              v-for="cm in configmapList"
              :key="cm.name"
              :label="cm.name"
              :value="cm.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Key" required>
          <el-input v-model="configmapEnvForm.key" placeholder="ConfigMap 中的键名" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configmapEnvDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveConfigMapEnv">确定</el-button>
      </template>
    </el-dialog>

    <!-- 添加/编辑 Secret 引用对话框 -->
    <el-dialog
      v-model="secretEnvDialogVisible"
      :title="editingSecretIndex >= 0 ? '编辑 Secret 引用' : '添加 Secret 引用'"
      width="600px"
    >
      <el-form :model="secretEnvForm" label-width="120px" label-position="left">
        <el-form-item label="变量名称" required>
          <el-input v-model="secretEnvForm.name" placeholder="环境变量名称" clearable />
        </el-form-item>
        <el-form-item label="Secret" required>
          <el-select
            v-model="secretEnvForm.secretName"
            placeholder="选择 Secret"
            style="width: 100%"
            filterable
          >
            <el-option
              v-for="sec in secretList"
              :key="sec.name"
              :label="sec.name"
              :value="sec.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Key" required>
          <el-input v-model="secretEnvForm.key" placeholder="Secret 中的键名" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="secretEnvDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSecretEnv">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Edit, Delete, Select, Close, Key, Document, Lock } from '@element-plus/icons-vue'

// 环境变量接口定义
interface NormalEnv {
  name: string
  value: string
  valueFrom?: never
  editing?: boolean
  tempValue?: string
}

interface ConfigMapEnv {
  name: string
  configmapName: string
  key: string
  valueFrom: {
    type: 'configmap'
    configMapName: string
    key: string
  }
}

interface SecretEnv {
  name: string
  secretName: string
  key: string
  valueFrom: {
    type: 'secret'
    secretName: string
    key: string
  }
}

type EnvVar = NormalEnv | ConfigMapEnv | SecretEnv

const props = defineProps<{
  envs: EnvVar[]
  configmapList?: { name: string }[]
  secretList?: { name: string }[]
}>()

const emit = defineEmits<{
  update: [envs: EnvVar[]]
}>()

const activeEnvTab = ref('normal')
const normalEnvDialogVisible = ref(false)
const configmapEnvDialogVisible = ref(false)
const secretEnvDialogVisible = ref(false)
const editingEnvIndex = ref(-1)
const editingConfigMapIndex = ref(-1)
const editingSecretIndex = ref(-1)

// 普通环境变量表单
const normalEnvForm = ref({
  name: '',
  value: ''
})

// ConfigMap 环境变量表单
const configmapEnvForm = ref({
  name: '',
  configmapName: '',
  key: ''
})

// Secret 环境变量表单
const secretEnvForm = ref({
  name: '',
  secretName: '',
  key: ''
})

// 分离不同类型的环境变量
const normalEnvs = computed(() => {
  return props.envs.filter(env => !env.valueFrom) as NormalEnv[]
})

const configmapEnvs = computed(() => {
  return props.envs.filter(env => env.valueFrom?.type === 'configmap') as ConfigMapEnv[]
})

const secretEnvs = computed(() => {
  return props.envs.filter(env => env.valueFrom?.type === 'secret') as SecretEnv[]
})

// 显示添加对话框
const showAddEnvDialog = (type: string) => {
  if (type === 'normal') {
    normalEnvForm.value = { name: '', value: '' }
    editingEnvIndex.value = -1
    normalEnvDialogVisible.value = true
  } else if (type === 'configmap') {
    configmapEnvForm.value = { name: '', configmapName: '', key: '' }
    editingConfigMapIndex.value = -1
    configmapEnvDialogVisible.value = true
  } else if (type === 'secret') {
    secretEnvForm.value = { name: '', secretName: '', key: '' }
    editingSecretIndex.value = -1
    secretEnvDialogVisible.value = true
  }
}

// 编辑普通环境变量
const editEnv = (row: NormalEnv, index: number) => {
  row.editing = true
  row.tempValue = row.value
}

const saveEnvEdit = (row: NormalEnv, index: number) => {
  if (row.tempValue !== undefined) {
    row.value = row.tempValue
  }
  row.editing = false
  row.tempValue = undefined
  updateEnvs()
}

const cancelEnvEdit = (row: NormalEnv, index: number) => {
  row.editing = false
  row.tempValue = undefined
}

// 保存普通环境变量
const saveNormalEnv = () => {
  if (!normalEnvForm.value.name) {
    ElMessage.warning('请输入变量名称')
    return
  }
  if (!normalEnvForm.value.value) {
    ElMessage.warning('请输入变量值')
    return
  }

  const newEnv: NormalEnv = {
    name: normalEnvForm.value.name,
    value: normalEnvForm.value.value
  }

  if (editingEnvIndex.value >= 0) {
    const normalEnvList = normalEnvs.value
    normalEnvList[editingEnvIndex.value] = newEnv
  } else {
    normalEnvs.value.push(newEnv)
  }

  normalEnvDialogVisible.value = false
  updateEnvs()
}

// 编辑 ConfigMap 引用
const editConfigMapEnv = (row: ConfigMapEnv, index: number) => {
  configmapEnvForm.value = {
    name: row.name,
    configmapName: row.configmapName,
    key: row.key
  }
  editingConfigMapIndex.value = index
  configmapEnvDialogVisible.value = true
}

// 保存 ConfigMap 引用
const saveConfigMapEnv = () => {
  if (!configmapEnvForm.value.name) {
    ElMessage.warning('请输入变量名称')
    return
  }
  if (!configmapEnvForm.value.configmapName) {
    ElMessage.warning('请选择 ConfigMap')
    return
  }
  if (!configmapEnvForm.value.key) {
    ElMessage.warning('请输入 Key')
    return
  }

  const newEnv: ConfigMapEnv = {
    name: configmapEnvForm.value.name,
    configmapName: configmapEnvForm.value.configmapName,
    key: configmapEnvForm.value.key,
    valueFrom: {
      type: 'configmap',
      configMapName: configmapEnvForm.value.configmapName,
      key: configmapEnvForm.value.key
    }
  }

  if (editingConfigMapIndex.value >= 0) {
    const configmapEnvList = configmapEnvs.value
    configmapEnvList[editingConfigMapIndex.value] = newEnv
  } else {
    configmapEnvs.value.push(newEnv)
  }

  configmapEnvDialogVisible.value = false
  updateEnvs()
}

// 编辑 Secret 引用
const editSecretEnv = (row: SecretEnv, index: number) => {
  secretEnvForm.value = {
    name: row.name,
    secretName: row.secretName,
    key: row.key
  }
  editingSecretIndex.value = index
  secretEnvDialogVisible.value = true
}

// 保存 Secret 引用
const saveSecretEnv = () => {
  if (!secretEnvForm.value.name) {
    ElMessage.warning('请输入变量名称')
    return
  }
  if (!secretEnvForm.value.secretName) {
    ElMessage.warning('请选择 Secret')
    return
  }
  if (!secretEnvForm.value.key) {
    ElMessage.warning('请输入 Key')
    return
  }

  const newEnv: SecretEnv = {
    name: secretEnvForm.value.name,
    secretName: secretEnvForm.value.secretName,
    key: secretEnvForm.value.key,
    valueFrom: {
      type: 'secret',
      secretName: secretEnvForm.value.secretName,
      key: secretEnvForm.value.key
    }
  }

  if (editingSecretIndex.value >= 0) {
    const secretEnvList = secretEnvs.value
    secretEnvList[editingSecretIndex.value] = newEnv
  } else {
    secretEnvs.value.push(newEnv)
  }

  secretEnvDialogVisible.value = false
  updateEnvs()
}

// 删除环境变量
const removeEnv = (type: string, index: number) => {
  const updatedEnvs = [...props.envs]

  if (type === 'normal') {
    const normalEnvList = normalEnvs.value
    const targetEnv = normalEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  } else if (type === 'configmap') {
    const configmapEnvList = configmapEnvs.value
    const targetEnv = configmapEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  } else if (type === 'secret') {
    const secretEnvList = secretEnvs.value
    const targetEnv = secretEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  }

  emit('update', updatedEnvs)
}

// 更新环境变量列表
const updateEnvs = () => {
  const updatedEnvs: EnvVar[] = [
    ...normalEnvs.value,
    ...configmapEnvs.value,
    ...secretEnvs.value
  ]
  emit('update', updatedEnvs)
}
</script>

<style scoped>
.env-config-content {
  padding: 0;
  background: #fff;
  color: #111827;
}

.env-tabs-wrapper {
  width: 100%;
}

.env-type-tabs {
  width: 100%;
}

.env-type-tabs :deep(.el-tabs__header) {
  height: auto;
  min-height: 44px;
  margin: 0 0 14px;
  padding: 6px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #f9fafb;
}

.env-type-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.env-type-tabs :deep(.el-tabs__nav) {
  display: flex;
  gap: 6px;
  height: auto;
  border: none;
}

.env-type-tabs :deep(.el-tabs__item) {
  height: 32px;
  line-height: 32px;
  padding: 0 14px !important;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #6b7280;
  font-size: 13px;
  font-weight: 700;
}

.env-type-tabs :deep(.el-tabs__item:hover) {
  background: #ffffff;
  color: #111827;
}

.env-type-tabs :deep(.el-tabs__item.is-active) {
  border-color: #d1d5db;
  background: #ffffff;
  color: #111827;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.06);
}

.env-type-tabs :deep(.el-tabs__active-bar) {
  display: none;
}

.env-section {
  width: 100%;
}

.env-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px 12px 0 0;
  margin-bottom: 0;
}

.env-header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 800;
  color: #111827;
}

.env-header-title .el-icon {
  font-size: 18px;
  color: #111827;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  background: #f3f4f6;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: none;
}

.env-header .el-button {
  height: 32px;
  padding: 0 14px;
  background: #111827;
  border: 1px solid #111827;
  border-radius: 8px;
  color: #ffffff;
  font-weight: 700;
  box-shadow: none;
}

.env-header .el-button:hover {
  background: #374151;
  border-color: #374151;
  color: #ffffff;
  box-shadow: none;
}

.env-table-wrapper {
  border: 1px solid #e5e7eb;
  border-top: none;
  border-radius: 0 0 12px 12px;
  padding: 16px;
  background: #ffffff;
}

.env-table {
  background: #fff;
  border-radius: 8px;
}

.env-table :deep(.el-table__header-wrapper) {
  background: #f9fafb;
}

.env-table :deep(.el-table__header th) {
  background: #f9fafb;
  color: #374151;
  font-weight: 700;
  border-bottom: 1px solid #e5e7eb;
}

.env-table :deep(.el-table__body tr) {
  transition: all 0.3s ease;
}

.env-table :deep(.el-table__body tr:hover) {
  background: #f9fafb;
}

.env-table :deep(.el-table__body td) {
  border-bottom: 1px solid #f3f4f6;
}

.env-name {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  color: #111827;
  font-weight: 600;
}

.env-value {
  color: #4b5563;
  word-break: break-all;
}

.env-value-cell {
  display: flex;
  align-items: center;
}

.env-resource {
  color: #111827;
  font-weight: 600;
}

.env-key {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  color: #047857;
  font-weight: 600;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-buttons :deep(.el-button) {
  width: auto;
  min-width: 56px;
  height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #374151;
  box-shadow: none;
}

.action-buttons :deep(.el-button span) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.action-buttons :deep(.el-button--primary),
.action-buttons :deep(.el-button--primary.is-link) {
  border-color: #d1d5db;
  background: #f9fafb;
  color: #111827;
}

.action-buttons :deep(.el-button--primary:hover),
.action-buttons :deep(.el-button--primary.is-link:hover) {
  border-color: #9ca3af;
  background: #f3f4f6;
  color: #111827;
}

.action-buttons :deep(.el-button--danger),
.action-buttons :deep(.el-button--danger.is-link) {
  border-color: #fecdd3;
  background: #fff1f2;
  color: #dc2626;
}

.action-buttons :deep(.el-button--danger:hover),
.action-buttons :deep(.el-button--danger.is-link:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
}

.action-buttons :deep(.el-button--success),
.action-buttons :deep(.el-button--success.is-link) {
  border-color: #bbf7d0;
  background: #f0fdf4;
  color: #15803d;
}

.action-buttons :deep(.el-button--info),
.action-buttons :deep(.el-button--info.is-link) {
  border-color: #e5e7eb;
  background: #f9fafb;
  color: #4b5563;
}

:deep(.el-dialog__body) {
  padding: 20px;
}

:deep(.el-form-item__label) {
  font-weight: 600;
  color: #333;
}

:deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  transition: all 0.3s ease;
}

:deep(.el-input__wrapper:hover) {
  border-color: #9ca3af;
  box-shadow: none;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #111827;
  box-shadow: 0 0 0 2px rgba(17, 24, 39, 0.08);
}

:deep(.el-tabs__content) {
  padding-top: 16px;
}

:deep(.el-tabs__item) {
  font-size: 14px;
  font-weight: 500;
}

:deep(.el-tabs__item.is-active) {
  color: #111827;
}

:deep(.el-tabs__active-bar) {
  background: #111827;
}

:deep(.el-empty) {
  padding: 40px 0;
}

:deep(.el-empty__description) {
  color: #999;
}
</style>
