<template>
  <div class="storageclass-list">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 StorageClass 名称..." clearable class="search-input" @input="handleSearch">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-button class="black-button" @click="handleCreateYAML">
        <el-icon><Document /></el-icon> YAML创建
      </el-button>

      <el-button class="black-button" @click="loadStorageClasses">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredStorageClasses" v-loading="loading" class="modern-table">
        <el-table-column label="名称" prop="name" min-width="200" fixed>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Box /></el-icon>
              <div class="name-text">{{ row.name }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Provisioner" prop="provisioner" min-width="250" />
        <el-table-column label="回收策略" prop="reclaimPolicy" width="120">
          <template #default="{ row }">
            {{ formatReclaimPolicy(row.reclaimPolicy) }}
          </template>
        </el-table-column>
        <el-table-column label="绑定模式" prop="volumeBindingMode" width="140">
          <template #default="{ row }">
            {{ formatVolumeBindingMode(row.volumeBindingMode) }}
          </template>
        </el-table-column>
        <el-table-column label="允许卷扩展" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.allowVolumeExpansion ? 'success' : 'info'" size="small">
              {{ row.allowVolumeExpansion ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="120" />
        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn danger" @click="handleDelete(row)">
                  <el-icon :size="18"><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`StorageClass YAML - ${selectedStorageClass?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleYamlInput"
          @scroll="handleYamlScroll"
          ref="yamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 创建弹窗 -->
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 StorageClass" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in createYamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="createYamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleCreateYamlInput"
          @scroll="handleCreateYamlScroll"
          ref="createYamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createYamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveCreateYAML" :loading="creating">创建</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Document, Delete, Refresh, Box } from '@element-plus/icons-vue'
import { load } from 'js-yaml'
import {
  getStorageClasses,
  getStorageClassYAML,
  updateStorageClassYAML,
  createStorageClassYAML,
  deleteStorageClass,
  type StorageClassInfo
} from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['refresh'])

const loading = ref(false)
const saving = ref(false)
const storageClassList = ref<StorageClassInfo[]>([])
const searchName = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedStorageClass = ref<StorageClassInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredStorageClasses = computed(() => {
  let result = storageClassList.value
  if (searchName.value) {
    result = result.filter(s => s.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const loadStorageClasses = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getStorageClasses(props.clusterId)
    storageClassList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 StorageClass 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  // 本地过滤
}

const formatReclaimPolicy = (policy: string) => {
  const policyMap: Record<string, string> = {
    'Retain': '保留',
    'Delete': '删除'
  }
  return policyMap[policy] || policy
}

const formatVolumeBindingMode = (mode: string) => {
  const modeMap: Record<string, string> = {
    'Immediate': '立即绑定',
    'WaitForFirstConsumer': '等待消费者'
  }
  return modeMap[mode] || mode
}

const handleCreateYAML = () => {
  createYamlContent.value = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (sc: StorageClassInfo) => {
  if (!props.clusterId) return
  selectedStorageClass.value = sc
  try {
    const response = await getStorageClassYAML(props.clusterId, sc.name)
    originalJsonData.value = response
    const yaml = jsonToYaml(originalJsonData.value)
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 YAML 失败')
  }
}

const jsonToYaml = (obj: any, indent = 0): string => {
  const spaces = '  '.repeat(indent)
  let result = ''

  if (Array.isArray(obj)) {
    for (const item of obj) {
      result += `${spaces}- ${jsonToYaml(item, indent).trim()}\n`
    }
  } else if (typeof obj === 'object' && obj !== null) {
    for (const [key, value] of Object.entries(obj)) {
      if (value === null || value === undefined) {
        result += `${spaces}${key}: null\n`
      } else if (typeof value === 'object') {
        result += `${spaces}${key}:\n${jsonToYaml(value, indent + 1)}`
      } else {
        result += `${spaces}${key}: ${value}\n`
      }
    }
  } else {
    result = `${obj}\n`
  }

  return result
}

const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    console.error('YAML 解析错误:', error)
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedStorageClass.value) return

  saving.value = true
  try {
    let jsonData = originalJsonData.value
    try {
      jsonData = yamlToJson(yamlContent.value)
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedStorageClass.value) {
        jsonData.metadata.name = selectedStorageClass.value.name
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'storage.k8s.io/v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'StorageClass'
      }
    } catch (e) {
      console.warn('YAML 解析失败，使用原始 JSON:', e)
      jsonData = originalJsonData.value
    }

    await updateStorageClassYAML(
      props.clusterId,
      selectedStorageClass.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadStorageClasses()
  } catch (error) {
    console.error(error)
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleYamlInput = () => {
  // 处理输入
}

const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'storage.k8s.io/v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'StorageClass'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    await createStorageClassYAML(
      props.clusterId,
      jsonData
    )
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadStorageClasses()
  } catch (error) {
    console.error(error)
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

const handleCreateYamlInput = () => {
  // 处理输入
}

const handleCreateYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.create-yaml .yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

const handleDelete = async (sc: StorageClassInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 StorageClass ${sc.name} 吗？`, '删除确认', { type: 'error' })
    await deleteStorageClass(props.clusterId, sc.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadStorageClasses()
  } catch (error) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('删除失败')
    }
  }
}

// 修复页面偏移
watch(yamlDialogVisible, (val) => {
  if (val) {
    const scrollBarWidth = window.innerWidth - document.documentElement.clientWidth
    if (scrollBarWidth > 0) {
      document.body.style.paddingRight = `${scrollBarWidth}px`
    }
  } else {
    document.body.style.paddingRight = ''
  }
})

watch(() => props.clusterId, () => {
  loadStorageClasses()
})

onMounted(() => {
  loadStorageClasses()
})

defineExpose({
  loadData: () => loadStorageClasses(true)
})
</script>

<style scoped>
.storageclass-list {
  width: 100%;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.search-input {
  width: 280px;
}

.search-icon {
  color: #d4af37;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon {
  color: #d4af37;
  font-size: 18px;
}

.name-text {
  font-weight: 500;
  color: #d4af37;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #d4af37;
  transition: all 0.3s;
}

.action-btn:hover {
  color: #bfa13f;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
}

.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #d4af37;
  border-radius: 6px;
  overflow: hidden;
  background-color: #000000;
}

.yaml-line-numbers {
  background-color: #0d0d0d;
  color: #666;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #333;
}

.line-number {
  height: 20.8px;
  line-height: 1.6;
}

.yaml-textarea {
  flex: 1;
  background-color: #000000;
  color: #d4af37;
  border: none;
  outline: none;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  min-height: 400px;
}

.yaml-textarea::placeholder {
  color: #555;
}

.yaml-textarea:focus {
  outline: none;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 0;
  background-color: #1a1a1a;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
