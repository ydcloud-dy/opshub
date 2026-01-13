<template>
  <div class="service-accounts-tab">
    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="search-section">
        <el-input
          v-model="searchName"
          placeholder="搜索 ServiceAccount 名称..."
          clearable
          @input="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="action-buttons">
        <el-button class="black-button" @click="handleCreate">
          <el-icon><Plus /></el-icon>
          新增 ServiceAccount
        </el-button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" min-width="200" fixed="left">
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon"><User /></el-icon>
              </div>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="命名空间" prop="namespace" width="180">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.namespace }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" prop="age" width="140" />

        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link @click="handleEdit(row)" class="action-btn">
              <el-icon :size="18"><Edit /></el-icon>
            </el-button>
            <el-button link @click="handleDelete(row)" class="action-btn danger">
              <el-icon :size="18"><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="标签"
      width="600px"
    >
      <el-table :data="labelList" max-height="400">
        <el-table-column prop="key" label="Key" min-width="150" />
        <el-table-column prop="value" label="Value" min-width="150" />
      </el-table>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="yamlDialogTitle"
      width="80%"
      :close-on-click-modal="false"
      top="5vh"
    >
      <div class="yaml-editor-container">
        <el-input
          v-model="yamlContent"
          type="textarea"
          :rows="20"
          placeholder="请输入 YAML 配置..."
          class="yaml-editor"
        />
      </div>
      <template #footer>
        <el-button @click="yamlDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveYaml" :loading="yamlSaving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, User, PriceTag, Edit, Delete, Plus } from '@element-plus/icons-vue'
import { getServiceAccounts, type ServiceAccountInfo } from '@/api/kubernetes'
import axios from 'axios'

interface Props {
  clusterId: number
  namespace?: string
}

const props = defineProps<Props>()
const loading = ref(false)
const serviceAccounts = ref<ServiceAccountInfo[]>([])
const searchName = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const yamlDialogVisible = ref(false)
const yamlDialogTitle = ref('')
const yamlContent = ref('')
const yamlSaving = ref(false)
const editingItem = ref<ServiceAccountInfo | null>(null)

// 默认 ServiceAccount YAML 模板
const defaultServiceAccountYaml = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-serviceaccount
  namespace: default
`.trim()

// 过滤后的数据
const filteredData = computed(() => {
  let result = serviceAccounts.value
  if (searchName.value) {
    result = result.filter(item =>
      item.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }
  return result
})

// 分页后的数据
const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const data = await getServiceAccounts(props.clusterId, props.namespace)
    serviceAccounts.value = data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 ServiceAccount 列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 分页
const handlePageChange = () => {}
const handleSizeChange = () => {}

// 显示标签
const showLabels = (row: ServiceAccountInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 新增
const handleCreate = () => {
  yamlDialogTitle.value = '新增 ServiceAccount'
  yamlContent.value = defaultServiceAccountYaml
  editingItem.value = null
  yamlDialogVisible.value = true
}

// 编辑
const handleEdit = async (row: ServiceAccountInfo) => {
  try {
    const token = localStorage.getItem('token') || ''
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 将对象转换为 YAML 格式
    yamlContent.value = JSONToYAML(response.data.data)
    yamlDialogTitle.value = '编辑 ServiceAccount'
    editingItem.value = row
    yamlDialogVisible.value = true
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 ServiceAccount YAML 失败')
  }
}

// 删除
const handleDelete = async (row: ServiceAccountInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 ServiceAccount "${row.name}" 吗？此操作不可撤销。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token') || ''
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  }
}

// 保存 YAML
const handleSaveYaml = async () => {
  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token') || ''
    const namespace = props.namespace || 'default'

    if (editingItem.value) {
      // 编辑模式 - 需要实现更新 API
      ElMessage.info('编辑功能开发中...')
    } else {
      // 新增模式
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/serviceactions/${namespace}`,
        yamlContent.value,
        {
          params: { clusterId: props.clusterId },
          headers: {
            'Content-Type': 'application/yaml',
            Authorization: `Bearer ${token}`
          }
        }
      )
      ElMessage.success('创建成功')
      yamlDialogVisible.value = false
      await loadData()
    }
  } catch (error: any) {
    console.error(error)
    ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
  } finally {
    yamlSaving.value = false
  }
}

// 简单的 JSON 转 YAML (临时方案)
const JSONToYAML = (obj: any): string => {
  const yaml: string[] = []
  const convert = (o: any, indent = 0) => {
    const spaces = '  '.repeat(indent)
    if (Array.isArray(o)) {
      o.forEach(item => {
        if (typeof item === 'object' && item !== null) {
          yaml.push(spaces + '- ')
          const firstKey = Object.keys(item)[0]
          if (firstKey) {
            yaml.push(spaces + firstKey + ':')
            convert(item[firstKey], indent + 1)
          }
        } else {
          yaml.push(spaces + '- ' + item)
        }
      })
    } else if (typeof o === 'object' && o !== null) {
      Object.keys(o).forEach(key => {
        const value = o[key]
        if (value === null) {
          yaml.push(spaces + key + ': null')
        } else if (Array.isArray(value)) {
          yaml.push(spaces + key + ':')
          convert(value, indent + 1)
        } else if (typeof value === 'object') {
          yaml.push(spaces + key + ':')
          convert(value, indent + 1)
        } else {
          yaml.push(spaces + key + ': ' + value)
        }
      })
    }
  }
  convert(obj)
  return yaml.join('\n')
}

// 监听 props 变化
watch(() => [props.clusterId, props.namespace], () => {
  loadData()
}, { immediate: true })
</script>

<style scoped>
.service-accounts-tab {
  width: 100%;
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.search-section {
  flex: 1;
}

.search-input {
  width: 300px;
}

.search-icon {
  color: #d4af37;
}

.action-buttons {
  display: flex;
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

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon-wrapper {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #d4af37;
  flex-shrink: 0;
}

.name-icon {
  color: #d4af37;
  font-size: 14px;
}

.name-text {
  font-weight: 600;
  color: #d4af37;
}

.secrets-count {
  color: #606266;
  font-weight: 500;
}

.label-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.label-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.label-icon {
  color: #d4af37;
  font-size: 18px;
}

.label-count {
  position: absolute;
  top: -6px;
  right: -6px;
  background-color: #d4af37;
  color: #000;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #d4af37;
  z-index: 1;
}

.label-cell:hover .label-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.label-cell:hover .label-count {
  background-color: #bfa13f;
}

.action-btn {
  color: #d4af37;
  margin: 0 4px;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn:hover {
  transform: scale(1.1);
}

.yaml-editor-container {
  width: 100%;
}

.yaml-editor {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid #f0f0f0;
}
</style>
