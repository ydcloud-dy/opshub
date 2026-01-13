<template>
  <div class="service-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <el-input
        v-model="searchName"
        placeholder="搜索服务名称..."
        clearable
        class="search-input"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="filterType" placeholder="服务类型" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option label="ClusterIP" value="ClusterIP" />
        <el-option label="NodePort" value="NodePort" />
        <el-option label="LoadBalancer" value="LoadBalancer" />
      </el-select>

      <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
      </el-select>

      <el-button class="black-button" @click="handleCreate">创建服务</el-button>
      <el-button class="black-button" @click="handleCreateYAML">
        <el-icon><Document /></el-icon> YAML创建
      </el-button>
    </div>

    <!-- 服务列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedServices"
        v-loading="loading"
        class="modern-table"
        size="default"
      >
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Connection /></el-icon>
              <div>
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" prop="type" width="130">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)" size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Cluster IP" prop="clusterIP" width="140" />

        <el-table-column label="外部 IP" prop="externalIP" width="140">
          <template #default="{ row }">
            {{ row.externalIP || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="端口" min-width="200">
          <template #default="{ row }">
            <div v-for="port in row.ports" :key="port.port" class="port-item">
              {{ port.protocol }}: {{ port.port }}
              <span v-if="port.targetPort">→ {{ port.targetPort }}</span>
              <span v-if="port.nodePort"> ({{ port.nodePort }})</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="端点" prop="endpoints" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.endpoints > 0" type="success" size="small">{{ row.endpoints }}</el-tag>
            <el-tag v-else type="info" size="small">0</el-tag>
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
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleEdit(row)">
                  <el-icon :size="18"><Edit /></el-icon>
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
          :page-sizes="[10, 20, 50]"
          :total="filteredServices.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="`Service YAML - ${selectedService?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 Service" width="900px" :lock-scroll="false" class="yaml-dialog">
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

    <!-- 编辑对话框 -->
    <ServiceEditDialog
      ref="editDialogRef"
      :clusterId="clusterId"
      @success="handleEditSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Connection, Document, Edit, Delete } from '@element-plus/icons-vue'
import { load } from 'js-yaml'
import { getServices, getServiceYAML, updateServiceYAML, createServiceYAML, deleteService, getNamespaces, type ServiceInfo } from '@/api/kubernetes'
import ServiceEditDialog from './ServiceEditDialog.vue'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh'])

const loading = ref(false)
const saving = ref(false)
const serviceList = ref<ServiceInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterType = ref('')
const filterNamespace = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedService = ref<ServiceInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据
const editDialogRef = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredServices = computed(() => {
  let result = serviceList.value
  if (searchName.value) {
    result = result.filter(s => s.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterType.value) {
    result = result.filter(s => s.type === filterType.value)
  }
  if (filterNamespace.value) {
    result = result.filter(s => s.namespace === filterNamespace.value)
  }
  return result
})

const paginatedServices = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredServices.value.slice(start, end)
})

const loadServices = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getServices(props.clusterId, props.namespace || undefined)
    serviceList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取服务列表失败')
  } finally {
    loading.value = false
  }
}

const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
    console.error(error)
  }
}

const handleSearch = () => {
  currentPage.value = 1
}

const getTypeTagType = (type: string) => {
  const map: Record<string, string> = {
    ClusterIP: 'success',
    NodePort: 'warning',
    LoadBalancer: 'danger'
  }
  return map[type] || 'info'
}

const handleCreate = () => {
  editDialogRef.value?.openCreate(namespaces.value)
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: ${defaultNamespace}
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
`
  createYamlDialogVisible.value = true
}

const handleEdit = (service: ServiceInfo) => {
  editDialogRef.value?.openEdit(service, namespaces.value)
}

const handleEditSuccess = () => {
  emit('refresh')
  loadServices()
}

const handleEditYAML = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  selectedService.value = service
  try {
    const response = await getServiceYAML(props.clusterId, service.namespace, service.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response.items || response
    // 转换为 YAML 格式
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

// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    console.error('YAML 解析错误:', error)
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedService.value) return

  saving.value = true
  try {
    // 尝试将 YAML 转回 JSON，如果失败则使用原始 JSON
    let jsonData = originalJsonData.value
    try {
      jsonData = yamlToJson(yamlContent.value)
      // 确保基本的元数据存在
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedService.value) {
        jsonData.metadata.name = selectedService.value.name
      }
      if (!jsonData.metadata.namespace && selectedService.value) {
        jsonData.metadata.namespace = selectedService.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Service'
      }
    } catch (e) {
      console.warn('YAML 解析失败，使用原始 JSON:', e)
      jsonData = originalJsonData.value
    }

    await updateServiceYAML(
      props.clusterId,
      selectedService.value.namespace,
      selectedService.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
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

const handleDelete = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除服务 ${service.name} 吗？`, '删除确认', { type: 'error' })
    await deleteService(props.clusterId, service.namespace, service.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadServices()
  } catch (error) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('删除失败')
    }
  }
}

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    // 确保基本的元数据存在
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'Service'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createServiceYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
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

watch(() => props.clusterId, () => {
  loadServices()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadServices()
})

onMounted(() => {
  loadServices()
  loadNamespaces()
})

// 暴露方法给父组件
defineExpose({
  loadData: () => loadServices(true)
})
</script>

<style scoped>
.service-list {
  width: 100%;
}

/* 黑色按钮样式 */
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

.filter-select {
  width: 180px;
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

.namespace-text {
  font-size: 12px;
  color: #909399;
}

.port-item {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}

/* 操作按钮 */
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

/* YAML 编辑弹窗 */
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
