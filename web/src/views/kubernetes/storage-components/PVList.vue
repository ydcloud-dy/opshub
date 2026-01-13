<template>
  <div class="pv-list">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 PV 名称..." clearable class="search-input" @input="handleSearch">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-button class="black-button" @click="handleCreateYAML">
        <el-icon><Document /></el-icon> YAML创建
      </el-button>

      <el-button class="black-button" @click="loadPVs">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredPVs" v-loading="loading" class="modern-table">
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Folder /></el-icon>
              <div class="name-text">{{ row.name }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" prop="status" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="容量" prop="capacity" width="120" />
        <el-table-column label="访问模式" min-width="180">
          <template #default="{ row }">
            <div v-for="mode in row.accessModes" :key="mode" class="access-mode-item">
              {{ formatAccessMode(mode) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="回收策略" prop="reclaimPolicy" width="140">
          <template #default="{ row }">
            {{ formatReclaimPolicy(row.reclaimPolicy) }}
          </template>
        </el-table-column>
        <el-table-column label="声明" prop="claim" min-width="180">
          <template #default="{ row }">
            {{ row.claim || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="存储类" prop="storageClass" width="150">
          <template #default="{ row }">
            {{ row.storageClass || '-' }}
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

    <el-dialog v-model="yamlDialogVisible" :title="`PV YAML - ${selectedPV?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 PV" width="900px" :lock-scroll="false" class="yaml-dialog">
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
import { Search, Document, Delete, Refresh, Folder } from '@element-plus/icons-vue'
import { load } from 'js-yaml'
import {
  getPersistentVolumes,
  getPersistentVolumeYAML,
  updatePersistentVolumeYAML,
  createPersistentVolumeYAML,
  deletePersistentVolume,
  type PVInfo
} from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['refresh'])

const loading = ref(false)
const saving = ref(false)
const pvList = ref<PVInfo[]>([])
const searchName = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPV = ref<PVInfo | null>(null)
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

const filteredPVs = computed(() => {
  let result = pvList.value
  if (searchName.value) {
    result = result.filter(p => p.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const loadPVs = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getPersistentVolumes(props.clusterId)
    pvList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 PV 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  // 本地过滤
}

const formatAccessMode = (mode: string) => {
  const modeMap: Record<string, string> = {
    'ReadWriteOnce': 'RWO',
    'ReadOnlyMany': 'ROX',
    'ReadWriteMany': 'RWX',
    'ReadWriteOncePod': 'RWOP'
  }
  return modeMap[mode] || mode
}

const formatReclaimPolicy = (policy: string) => {
  const policyMap: Record<string, string> = {
    'Retain': '保留',
    'Delete': '删除',
    'Recycle': '回收'
  }
  return policyMap[policy] || policy
}

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    'Available': 'success',
    'Bound': 'warning',
    'Released': 'info',
    'Failed': 'danger'
  }
  return map[status] || 'info'
}

const handleCreateYAML = () => {
  createYamlContent.value = `apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: local-storage
  local:
    path: /mnt/data
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - node-1
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (pv: PVInfo) => {
  if (!props.clusterId) return
  selectedPV.value = pv
  try {
    const response = await getPersistentVolumeYAML(props.clusterId, pv.name)
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
  if (!props.clusterId || !selectedPV.value) return

  saving.value = true
  try {
    let jsonData = originalJsonData.value
    try {
      jsonData = yamlToJson(yamlContent.value)
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedPV.value) {
        jsonData.metadata.name = selectedPV.value.name
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'PersistentVolume'
      }
    } catch (e) {
      console.warn('YAML 解析失败，使用原始 JSON:', e)
      jsonData = originalJsonData.value
    }

    await updatePersistentVolumeYAML(
      props.clusterId,
      selectedPV.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadPVs()
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
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'PersistentVolume'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    await createPersistentVolumeYAML(
      props.clusterId,
      jsonData
    )
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadPVs()
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

const handleDelete = async (pv: PVInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 PV ${pv.name} 吗？`, '删除确认', { type: 'error' })
    await deletePersistentVolume(props.clusterId, pv.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadPVs()
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
  loadPVs()
})

onMounted(() => {
  loadPVs()
})

defineExpose({
  loadData: () => loadPVs(true)
})
</script>

<style scoped>
.pv-list {
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

.access-mode-item {
  font-size: 12px;
  padding: 2px 6px;
  background: #f0f0f0;
  border-radius: 3px;
  color: #606266;
  margin-bottom: 4px;
  display: inline-block;
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
