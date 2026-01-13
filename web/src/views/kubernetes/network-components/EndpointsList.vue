<template>
  <div class="endpoints-list">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 Endpoints 名称..." clearable class="search-input" @input="handleSearch">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
      </el-select>

      <el-button class="black-button" @click="handleCreateYAML">
        <el-icon><Document /></el-icon> YAML创建
      </el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredEndpoints" v-loading="loading" class="modern-table" size="default">
        <el-table-column label="名称" prop="name" min-width="200" fixed>
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
        <el-table-column label="端点" min-width="300">
          <template #default="{ row }">
            <div v-if="row.subsets.length > 0">
              <div v-for="(subset, idx) in row.subsets" :key="idx" class="subset-item">
                <el-tag size="small" type="success" class="endpoint-tag">
                  {{ subset.addresses.length }} 就绪
                </el-tag>
                <el-tag v-if="subset.notReadyAddresses.length > 0" size="small" type="warning" class="endpoint-tag">
                  {{ subset.notReadyAddresses.length }} 未就绪
                </el-tag>
                <div class="ports-display">
                  {{ subset.ports.map(p => `${p.port}/${p.protocol}`).join(', ') }}
                </div>
              </div>
            </div>
            <el-tag v-else type="info" size="small">无端点</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="120" />
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
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`Endpoints YAML - ${selectedEndpoint?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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

    <el-dialog v-model="detailDialogVisible" :title="`Endpoints 详情 - ${selectedEndpoint?.name}`" width="800px">
      <div v-if="selectedEndpoint">
        <div v-for="(subset, idx) in selectedEndpoint.subsets" :key="idx" class="detail-subset">
          <h4>Subset {{ idx + 1 }}</h4>
          <div><strong>就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.addresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.addresses.length">无</div>

          <div style="margin-top: 10px;"><strong>未就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.notReadyAddresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.notReadyAddresses.length">无</div>

          <div style="margin-top: 10px;"><strong>端口:</strong></div>
          <div>{{ subset.ports.map(p => `${p.name || '-'}: ${p.port}/${p.protocol}`).join(', ') || '-' }}</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- YAML 创建弹窗 -->
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 Endpoints" width="900px" :lock-scroll="false" class="yaml-dialog">
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
import { Search, Document, Connection, Delete } from '@element-plus/icons-vue'
import { getEndpoints, getEndpointsDetail, createEndpointYAML, getEndpointYAML, updateEndpointYAML, deleteEndpoint, getNamespaces, type EndpointsInfo } from '@/api/kubernetes'
import { load } from 'js-yaml'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['refresh'])

const loading = ref(false)
const endpointsList = ref<EndpointsInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const detailDialogVisible = ref(false)
const selectedEndpoint = ref<EndpointsInfo | null>(null)

// YAML 编辑相关
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null)
const saving = ref(false)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    console.error('YAML 解析错误:', error)
    throw error
  }
}

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredEndpoints = computed(() => {
  let result = endpointsList.value
  if (searchName.value) {
    result = result.filter(e => e.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(e => e.namespace === filterNamespace.value)
  }
  return result
})

const loadEndpoints = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getEndpoints(props.clusterId, props.namespace || undefined)
    endpointsList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Endpoints 列表失败')
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
  // 本地过滤
}

const handleDetail = (endpoint: EndpointsInfo) => {
  selectedEndpoint.value = endpoint
  detailDialogVisible.value = true
}

const handleEditYAML = async (endpoint: EndpointsInfo) => {
  if (!props.clusterId) return
  selectedEndpoint.value = endpoint
  try {
    const response = await getEndpointYAML(props.clusterId, endpoint.namespace, endpoint.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response
    // 转换为 YAML 格式
    const yaml = jsonToYaml(originalJsonData.value)
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 YAML 失败')
  }
}

const handleDelete = async (endpoint: EndpointsInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 Endpoint ${endpoint.name} 吗？`, '删除确认', { type: 'error' })
    await deleteEndpoint(props.clusterId, endpoint.namespace, endpoint.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadEndpoints()
  } catch (error) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('删除失败')
    }
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

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedEndpoint.value) return

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
      if (!jsonData.metadata.name && selectedEndpoint.value) {
        jsonData.metadata.name = selectedEndpoint.value.name
      }
      if (!jsonData.metadata.namespace && selectedEndpoint.value) {
        jsonData.metadata.namespace = selectedEndpoint.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Endpoints'
      }
    } catch (e) {
      console.warn('YAML 解析失败，使用原始 JSON:', e)
      jsonData = originalJsonData.value
    }

    await updateEndpointYAML(
      props.clusterId,
      selectedEndpoint.value.namespace,
      selectedEndpoint.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadEndpoints()
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

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: v1
kind: Endpoints
metadata:
  name: my-endpoints
  namespace: ${defaultNamespace}
subsets:
  - addresses:
      - ip: 192.168.1.1
    ports:
      - port: 80
        protocol: TCP
        name: http
`
  createYamlDialogVisible.value = true
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
      jsonData.kind = 'Endpoints'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createEndpointYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadEndpoints()
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
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

watch(() => props.clusterId, () => {
  loadEndpoints()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadEndpoints()
})

onMounted(() => {
  loadEndpoints()
  loadNamespaces()
})

// 暴露方法给父组件
defineExpose({
  loadData: () => loadEndpoints(true)
})
</script>

<style scoped>
.endpoints-list {
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

.subset-item {
  margin-bottom: 8px;
}

.endpoint-tag {
  margin-right: 4px;
}

.ports-display {
  font-size: 12px;
  color: #606266;
  margin-top: 4px;
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

.detail-subset {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #eee;
}

.detail-subset:last-child {
  border-bottom: none;
}

.address-item {
  font-size: 13px;
  color: #606266;
  padding: 4px 0;
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
