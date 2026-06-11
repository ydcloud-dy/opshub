<template>
  <div class="networkpolicy-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input v-model="searchName" placeholder="搜索 NetworkPolicy 名称..." clearable class="search-input" @input="handleSearch">
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </div>

      <div class="search-bar-right">
        <el-button class="black-button" @click="handleCreateYAML">
          <el-icon><Document /></el-icon> YAML创建
        </el-button>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table :data="paginatedPolicies" v-loading="loading" class="modern-table" size="default">
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Lock /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Lock /></el-icon>
              <div class="name-text">{{ row.name }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="命名空间" prop="namespace" width="140" />
        <el-table-column label="Pod 选择器" min-width="200">
          <template #default="{ row }">
            <div v-if="Object.keys(row.podSelector).length" class="selector-display">
              <span v-for="(value, key) in row.podSelector" :key="key" class="selector-item">
                {{ key }}: {{ value }}
              </span>
            </div>
            <div v-else>-</div>
          </template>
        </el-table-column>
        <el-table-column label="入站规则" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.ingress.length > 0 ? 'warning' : 'info'" size="small">
              {{ row.ingress.length }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="出站规则" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.egress.length > 0 ? 'danger' : 'info'" size="small">
              {{ row.egress.length }}
            </el-tag>
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

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="filteredPolicies.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`NetworkPolicy YAML - ${selectedPolicy?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 NetworkPolicy" width="900px" :lock-scroll="false" class="yaml-dialog">
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
import { Search, Lock, Document, Delete } from '@element-plus/icons-vue'
import { getNetworkPolicies, getNetworkPolicyYAML, updateNetworkPolicyYAML, createNetworkPolicyYAML, createNetworkPolicy, deleteNetworkPolicy, getNamespaces, type NetworkPolicyDetailInfo } from '@/api/kubernetes'
import { load, dump } from 'js-yaml'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['yaml', 'refresh', 'count-update'])

const loading = ref(false)
const saving = ref(false)
const policyList = ref<NetworkPolicyDetailInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPolicy = ref<NetworkPolicyDetailInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据

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

const filteredPolicies = computed(() => {
  let result = policyList.value
  if (searchName.value) {
    result = result.filter(p => p.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(p => p.namespace === filterNamespace.value)
  }
  return result
})

const paginatedPolicies = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredPolicies.value.slice(start, end)
})

const loadPolicies = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getNetworkPolicies(props.clusterId, props.namespace || undefined)
    policyList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    ElMessage.error('获取 NetworkPolicy 列表失败')
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
  }
}

const handleSearch = () => {
  currentPage.value = 1
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-networkpolicy
  namespace: ${defaultNamespace}
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: frontend
      ports:
        - protocol: TCP
          port: 80
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (policy: NetworkPolicyDetailInfo) => {
  if (!props.clusterId) return
  selectedPolicy.value = policy
  try {
    const response = await getNetworkPolicyYAML(props.clusterId, policy.namespace, policy.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response.items || response
    // 转换为 YAML 格式
    const yaml = dump(originalJsonData.value, { indent: 2, lineWidth: -1 })
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取 YAML 失败')
  }
}


// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedPolicy.value) return

  saving.value = true
  try {
    // 尝试将 YAML 转回 JSON
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      // 确保基本的元数据存在
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedPolicy.value) {
        jsonData.metadata.name = selectedPolicy.value.name
      }
      if (!jsonData.metadata.namespace && selectedPolicy.value) {
        jsonData.metadata.namespace = selectedPolicy.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'networking.k8s.io/v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'NetworkPolicy'
      }
    } catch (e) {
      ElMessage.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateNetworkPolicyYAML(
      props.clusterId,
      selectedPolicy.value.namespace,
      selectedPolicy.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadPolicies()
  } catch (error) {
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

const handleDelete = async (policy: NetworkPolicyDetailInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 NetworkPolicy ${policy.name} 吗？`, '删除确认', { type: 'error' })
    await deleteNetworkPolicy(props.clusterId, policy.namespace, policy.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadPolicies()
  } catch (error) {
    if (error !== 'cancel') {
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
      jsonData.apiVersion = 'networking.k8s.io/v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'NetworkPolicy'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间和名称
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    const name = jsonData.metadata.name

    if (!name) {
      ElMessage.error('YAML 中缺少 metadata.name 字段')
      return
    }

    // 从 spec 中提取数据
    const spec = jsonData.spec || {}

    // 转换 podSelector
    const podSelector = spec.podSelector?.matchLabels || {}

    // 转换 policyTypes
    const policyTypes = (spec.policyTypes || []).map((pt: string) => pt)

    // 转换 ingress 规则
    const ingress = (spec.ingress || []).map((rule: any) => {
      const ingressRule: any = {}

      // 转换 ports
      if (rule.ports) {
        ingressRule.ports = rule.ports.map((p: any) => {
          const portInfo: any = {}
          if (p.protocol) portInfo.protocol = p.protocol
          if (p.port) portInfo.port = p.port
          if (p.endPort) portInfo.endPort = p.endPort
          return portInfo
        })
      }

      // 转换 from
      if (rule.from) {
        ingressRule.from = rule.from.map((f: any) => {
          const peerInfo: any = {}
          if (f.podSelector?.matchLabels) peerInfo.podSelector = f.podSelector.matchLabels
          if (f.namespaceSelector?.matchLabels) peerInfo.namespaceSelector = f.namespaceSelector.matchLabels
          if (f.ipBlock) peerInfo.ipBlock = { cidr: f.ipBlock.cidr, except: f.ipBlock.except }
          return peerInfo
        })
      }

      return ingressRule
    })

    // 转换 egress 规则
    const egress = (spec.egress || []).map((rule: any) => {
      const egressRule: any = {}

      // 转换 ports
      if (rule.ports) {
        egressRule.ports = rule.ports.map((p: any) => {
          const portInfo: any = {}
          if (p.protocol) portInfo.protocol = p.protocol
          if (p.port) portInfo.port = p.port
          if (p.endPort) portInfo.endPort = p.endPort
          return portInfo
        })
      }

      // 转换 to
      if (rule.to) {
        egressRule.to = rule.to.map((t: any) => {
          const peerInfo: any = {}
          if (t.podSelector?.matchLabels) peerInfo.podSelector = t.podSelector.matchLabels
          if (t.namespaceSelector?.matchLabels) peerInfo.namespaceSelector = t.namespaceSelector.matchLabels
          if (t.ipBlock) peerInfo.ipBlock = { cidr: t.ipBlock.cidr, except: t.ipBlock.except }
          return peerInfo
        })
      }

      return egressRule
    })

    // 构建创建请求数据
    const createData = {
      name,
      podSelector,
      policyTypes,
      ingress,
      egress
    }

    await createNetworkPolicy(props.clusterId, namespace, createData)
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadPolicies()
  } catch (error) {
    ElMessage.error('创建失败: ' + (error as any).message)
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
  loadPolicies()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadPolicies()
})

// 监听筛选后的数据变化，更新计数
watch(filteredPolicies, (newData) => {
  emit('count-update', newData.length)
  const totalPages = Math.max(1, Math.ceil(newData.length / pageSize.value))
  if (currentPage.value > totalPages) {
    currentPage.value = totalPages
  }
})

watch(pageSize, () => {
  currentPage.value = 1
})

onMounted(() => {
  loadPolicies()
  loadNamespaces()
})

// 暴露方法给父组件
defineExpose({
  loadData: () => loadPolicies(true)
})
</script>

<style scoped>
.networkpolicy-list {
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
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-bar-left {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-bar-right {
  display: flex;
  gap: 12px;
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
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 18px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.name-text {
  font-weight: 500;
  color: #303133;
}

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-blue {
  color: #d4af37;
}

.selector-display {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.selector-item {
  font-size: 12px;
  padding: 2px 6px;
  background: #f0f0f0;
  border-radius: 3px;
  color: #606266;
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
