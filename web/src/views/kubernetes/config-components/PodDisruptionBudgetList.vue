<template>
  <div class="pdb-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <el-input
        v-model="searchName"
        placeholder="搜索 PodDisruptionBudget 名称..."
        clearable
        class="search-input"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
      </el-select>

      <el-button type="primary" class="black-button create-btn" @click="handleCreate">
        <el-icon style="margin-right: 4px;"><Plus /></el-icon>
        新增 PodDisruptionBudget
      </el-button>
    </div>

    <!-- PodDisruptionBudget 列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedPDBs"
        v-loading="loading"
        class="modern-table"
        size="default"
      >
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Lock /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Min Available" prop="minAvailable" width="140" align="center">
          <template #default="{ row }">
            <span class="resource-value">{{ row.minAvailable || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Max Unavailable" prop="maxUnavailable" width="150" align="center">
          <template #default="{ row }">
            <span class="resource-value">{{ row.maxUnavailable || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Allowed Disruptions" prop="allowedDisruptions" width="170" align="center">
          <template #default="{ row }">
            <el-tag type="success" size="small">{{ row.allowedDisruptions ?? '-' }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Current Healthy" prop="currentHealthy" width="150" align="center">
          <template #default="{ row }">
            <el-tag :type="getHealthyTagType(row.currentHealthy, row.desiredHealthy)" size="small">
              {{ row.currentHealthy ?? '-' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Desired Healthy" prop="desiredHealthy" width="150" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.desiredHealthy ?? '-' }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ row.createdAt || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" fixed="right" align="center">
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredPDBs.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="yamlDialogTitle" width="900px" class="yaml-dialog">
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
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving" class="black-button">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Lock, Document, Delete, Plus } from '@element-plus/icons-vue'
import { getNamespaces } from '@/api/kubernetes'
import axios from 'axios'

interface PDBInfo {
  name: string
  namespace: string
  minAvailable?: string
  maxUnavailable?: string
  allowedDisruptions?: number
  currentHealthy?: number
  desiredHealthy?: number
  age: string
  createdAt?: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh'])

const loading = ref(false)
const pdbList = ref<PDBInfo[]>([])
const namespaces = ref<{ name: string }[]>([])

// 搜索和筛选
const searchName = ref('')
const filterNamespace = ref('')

// 分页
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPDB = ref<PDBInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 PodDisruptionBudget'
  }
  return `PodDisruptionBudget YAML - ${selectedPDB.value?.name || ''}`
})

// 默认 PodDisruptionBudget YAML 模板
const getDefaultPDBYAML = () => `apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: example-pdb
  namespace: default
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: example
`

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 获取健康状态标签类型
const getHealthyTagType = (current: number | undefined, desired: number | undefined) => {
  if (current === undefined || desired === undefined) return 'info'
  if (current < desired) return 'danger'
  if (current === desired) return 'success'
  return 'info'
}

// 过滤后的列表
const filteredPDBs = computed(() => {
  let result = pdbList.value

  if (searchName.value) {
    result = result.filter(p =>
      p.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (filterNamespace.value) {
    result = result.filter(p => p.namespace === filterNamespace.value)
  }

  return result
})

// 分页后的列表
const paginatedPDBs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredPDBs.value.slice(start, end)
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
    console.error('获取命名空间列表失败:', error)
  }
}

// 加载 PodDisruptionBudget 列表
const loadPDBs = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/poddisruptionbudgets`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    pdbList.value = response.data.data || []
  } catch (error) {
    console.error('获取 PodDisruptionBudget 列表失败:', error)
    pdbList.value = []
    ElMessage.error('获取 PodDisruptionBudget 列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 编辑 YAML
const handleEditYAML = async (row: PDBInfo) => {
  selectedPDB.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 响应拦截器已经返回了 res.data，所以直接用 response.yaml
    yamlContent.value = response.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    console.error('获取 YAML 失败:', error)
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 新增 PodDisruptionBudget
const handleCreate = () => {
  isCreateMode.value = true
  selectedPDB.value = null
  yamlContent.value = getDefaultPDBYAML()
  yamlDialogVisible.value = true
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (isCreateMode.value) {
    // 创建模式
    const nameMatch = yamlContent.value.match(/name:\s*(.+)/)
    const nsMatch = yamlContent.value.match(/namespace:\s*(.+)/)
    if (!nameMatch || !nsMatch) {
      ElMessage.error('YAML中缺少name或namespace字段')
      return
    }
    const namespace = nsMatch[1].trim()

    saving.value = true
    try {
      const token = localStorage.getItem('token')
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${namespace}/yaml`,
        {
          clusterId: props.clusterId,
          yaml: yamlContent.value
        },
        {
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('创建成功')
      yamlDialogVisible.value = false
      await loadPDBs()
      emit('refresh')
    } catch (error: any) {
      console.error('创建失败:', error)
      ElMessage.error(`创建失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  } else {
    // 编辑模式
    if (!selectedPDB.value) return

    saving.value = true
    try {
      const token = localStorage.getItem('token')
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${selectedPDB.value.namespace}/${selectedPDB.value.name}/yaml`,
        {
          clusterId: props.clusterId,
          yaml: yamlContent.value
        },
        {
          headers: { Authorization: `Bearer ${token}` }
        }
      )

      ElMessage.success('保存成功')
      yamlDialogVisible.value = false
      await loadPDBs()
      emit('refresh')
    } catch (error: any) {
      console.error('保存失败:', error)
      ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  }
}

// 删除 PodDisruptionBudget
const handleDelete = async (row: PDBInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 PodDisruptionBudget ${row.name} 吗？此操作不可恢复！`,
      '删除 PodDisruptionBudget 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadPDBs()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 可以添加输入验证
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 监听 clusterId 变化
watch(() => props.clusterId, (newVal) => {
  if (newVal) {
    currentPage.value = 1
    loadNamespaces()
    loadPDBs()
  }
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadPDBs()
  }
})
</script>

<style scoped>
.pdb-list {
  padding: 0;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 16px;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 200px;
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.resource-value {
  font-size: 13px;
  color: #606266;
}

/* 名称单元格 */
.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #d4af37;
  flex-shrink: 0;
}

.name-icon {
  color: #d4af37;
}

.name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name-text {
  font-size: 14px;
  font-weight: 600;
  color: #d4af37;
}

.namespace-text {
  font-size: 12px;
  color: #909399;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: #d4af37;
  padding: 4px;
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

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #1a1a1a;
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

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}
</style>
