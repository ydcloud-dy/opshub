<template>
  <div class="configmap-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchName"
          placeholder="搜索 ConfigMap 名称..."
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
      </div>

      <div class="search-bar-right">
        <el-button type="primary" class="black-button" @click="handleCreateYAML">
          <el-icon style="margin-right: 4px;"><Document /></el-icon>
          YAML创建
        </el-button>

        <el-button type="primary" class="black-button" @click="handleCreateForm">
          <el-icon style="margin-right: 4px;"><Plus /></el-icon>
          表单创建
        </el-button>
      </div>
    </div>

    <!-- ConfigMap 列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedConfigMaps"
        v-loading="loading"
        class="modern-table"
        size="default"
      >
        <el-table-column label="名称" prop="name" min-width="200" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Key /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Key /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="数据项" prop="dataCount" width="100" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.dataCount }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" prop="age" width="140" />

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ row.createdAt || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleEditForm(row)">
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
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredConfigMaps.length"
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

    <!-- 表单创建弹窗 -->
    <el-dialog v-model="formDialogVisible" :title="formDialogTitle" width="min(1420px, 88vw)" class="form-dialog configmap-editor-dialog">
      <el-form :model="formData" label-width="100px" class="configmap-form">
        <div class="configmap-hero">
          <div>
            <div class="configmap-hero-kicker">Kubernetes ConfigMap</div>
            <div class="configmap-hero-title">{{ formData.name || '新的配置' }}</div>
            <p>用键值对管理应用配置，支持普通数据、二进制数据、标签和注解。</p>
          </div>
          <div class="configmap-hero-badges">
            <span>{{ formData.data.length }} Data</span>
            <span>{{ formData.binaryData.length }} BinaryData</span>
          </div>
        </div>

        <div class="form-row resource-fields">
          <el-form-item label="名称" required>
            <el-input v-model="formData.name" placeholder="请输入 ConfigMap 名称" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="命名空间" required>
            <el-select v-model="formData.namespace" placeholder="请选择命名空间" style="width: 100%;">
              <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
            </el-select>
          </el-form-item>
        </div>

        <!-- 标签页 -->
        <el-tabs v-model="activeTab" class="form-tabs">
          <!-- 数据标签页 -->
          <el-tab-pane label="数据" name="data">
            <div class="tab-content">
              <!-- Data 部分 -->
              <div class="data-section">
                <div class="section-header">
                  <span class="section-title">Data</span>
                  <el-button size="small" type="primary" @click="addDataRow">
                    <el-icon><Plus /></el-icon> 添加数据
                  </el-button>
                </div>
                <el-table :data="formData.data" class="form-table config-data-table">
                  <el-table-column label="Key" width="260">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" type="textarea" :rows="4" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeDataRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <!-- BinaryData 部分 -->
              <div class="binarydata-section">
                <div class="section-header">
                  <span class="section-title">BinaryData</span>
                  <el-button size="small" type="primary" @click="addBinaryDataRow">
                    <el-icon><Plus /></el-icon> 添加二进制数据
                  </el-button>
                </div>
                <el-table :data="formData.binaryData" class="form-table config-data-table">
                  <el-table-column label="Key" width="260">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" type="textarea" :rows="4" placeholder="请输入 Value (Base64编码)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeBinaryDataRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </el-tab-pane>

          <!-- 标签/注解标签页 -->
          <el-tab-pane label="标签/注解" name="metadata">
            <div class="tab-content">
              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">标签</span>
                  <el-button size="small" @click="addLabelRow">
                    <el-icon><Plus /></el-icon> 添加
                  </el-button>
                </div>
                <el-table :data="formData.labels" class="form-table metadata-table">
                  <el-table-column label="Key" width="260">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeLabelRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">注解</span>
                  <el-button size="small" @click="addAnnotationRow">
                    <el-icon><Plus /></el-icon> 添加
                  </el-button>
                </div>
                <el-table :data="formData.annotations" class="form-table metadata-table">
                  <el-table-column label="Key" width="260">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeAnnotationRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="formDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveForm" :loading="saving" class="black-button">{{ isEditMode ? '保存' : '创建' }}</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Key, Document, Delete, Plus, Edit } from '@element-plus/icons-vue'
import { getNamespaces } from '@/api/kubernetes'
import axios from 'axios'
import * as yaml from 'js-yaml'

interface ConfigMapInfo {
  name: string
  namespace: string
  dataCount: number
  age: string
  createdAt?: string
}

interface KeyValueRow {
  key: string
  value: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const configMapList = ref<ConfigMapInfo[]>([])
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
const selectedConfigMap = ref<ConfigMapInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)
const isEditMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 ConfigMap (YAML)'
  }
  return `ConfigMap YAML - ${selectedConfigMap.value?.name || ''}`
})

// 表单对话框标题
const formDialogTitle = computed(() => {
  if (isEditMode.value) {
    return `编辑 ConfigMap - ${formData.value.name}`
  }
  return '新增 ConfigMap'
})

// 表单创建
const formDialogVisible = ref(false)
const activeTab = ref('data')
const formData = ref({
  name: '',
  namespace: '',
  data: [] as KeyValueRow[],
  binaryData: [] as KeyValueRow[],
  labels: [] as KeyValueRow[],
  annotations: [] as KeyValueRow[]
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 过滤后的列表
const filteredConfigMaps = computed(() => {
  let result = configMapList.value

  if (searchName.value) {
    result = result.filter(cm =>
      cm.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (filterNamespace.value) {
    result = result.filter(cm => cm.namespace === filterNamespace.value)
  }

  return result
})

// 分页后的列表
const paginatedConfigMaps = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredConfigMaps.value.slice(start, end)
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

// 加载 ConfigMap 列表
const loadConfigMaps = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/configmaps`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    configMapList.value = response.data.data || []
  } catch (error) {
    configMapList.value = []
    // 不显示错误提示，避免频繁弹出
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// YAML 创建
const handleCreateYAML = () => {
  isCreateMode.value = true
  selectedConfigMap.value = null
  // 默认 ConfigMap YAML 模板
  yamlContent.value = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
  namespace: default
data:
  key1: value1
  key2: value2
`
  yamlDialogVisible.value = true
}

// 表单创建
const handleCreateForm = () => {
  isEditMode.value = false
  formData.value = {
    name: '',
    namespace: namespaces.value[0]?.name || '',
    data: [],
    binaryData: [],
    labels: [],
    annotations: []
  }
  formDialogVisible.value = true
}

// 编辑表单
const handleEditForm = async (row: ConfigMapInfo) => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )


    // 获取ConfigMap对象，可能是items或yaml字段
    let configMap: any = response.data?.data?.items || response.data?.data?.yaml

    // 如果是yaml字符串，需要解析
    if (typeof configMap === 'string') {
      configMap = yaml.load(configMap)
    }


    if (!configMap || !configMap.metadata) {
      ElMessage.error('获取ConfigMap数据失败')
      return
    }

    // 填充表单数据
    formData.value = {
      name: configMap.metadata?.name || '',
      namespace: configMap.metadata?.namespace || '',
      data: configMap.data ? Object.entries(configMap.data).map(([key, value]) => ({ key, value: String(value) })) : [],
      binaryData: configMap.binaryData ? Object.entries(configMap.binaryData).map(([key, value]) => ({ key, value: String(value) })) : [],
      labels: configMap.metadata?.labels ? Object.entries(configMap.metadata.labels).map(([key, value]) => ({ key, value })) : [],
      annotations: configMap.metadata?.annotations ? Object.entries(configMap.metadata.annotations).map(([key, value]) => ({ key, value })) : []
    }


    isEditMode.value = true
    formDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取详情失败: ${error.response?.data?.message || error.message}`)
  }
}

// 编辑 YAML
const handleEditYAML = async (row: ConfigMapInfo) => {
  selectedConfigMap.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 后端返回的是 {data: {items: ConfigMap对象}}
    const jsonData = response.data.data?.items
    if (jsonData) {
      yamlContent.value = yaml.dump(jsonData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })
      yamlDialogVisible.value = true
    }
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  saving.value = true
  try {
    const token = localStorage.getItem('token')

    // 从 YAML 中解析对象
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      ElMessage.error('YAML 中缺少必要的 metadata.name 字段')
      return
    }
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || 'default'

    if (isCreateMode.value) {
      // 创建模式 - 发送 JSON 对象
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('创建成功')
    } else {
      // 编辑模式 - 发送 JSON 对象
      if (!selectedConfigMap.value) return
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${selectedConfigMap.value.namespace}/${selectedConfigMap.value.name}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('保存成功')
    }

    yamlDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 保存表单
const handleSaveForm = async () => {
  if (!formData.value.name) {
    ElMessage.error('请输入名称')
    return
  }
  if (!formData.value.namespace) {
    ElMessage.error('请选择命名空间')
    return
  }

  saving.value = true
  try {
    const token = localStorage.getItem('token')


    // 构建 ConfigMap 对象
    const configMapObj: any = {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: formData.value.name,
        namespace: formData.value.namespace
      }
    }

    // 添加 Data
    if (formData.value.data.length > 0) {
      configMapObj.data = {}
      formData.value.data.forEach(row => {
        if (row.key) {
          configMapObj.data[row.key] = row.value
        }
      })
    }

    // 添加 BinaryData
    if (formData.value.binaryData.length > 0) {
      configMapObj.binaryData = {}
      formData.value.binaryData.forEach(row => {
        if (row.key) {
          configMapObj.binaryData[row.key] = row.value
        }
      })
    }

    // 添加标签
    if (formData.value.labels.length > 0) {
      configMapObj.metadata.labels = {}
      formData.value.labels.forEach(row => {
        if (row.key) {
          configMapObj.metadata.labels[row.key] = row.value
        }
      })
    }

    // 添加注解
    if (formData.value.annotations.length > 0) {
      configMapObj.metadata.annotations = {}
      formData.value.annotations.forEach(row => {
        if (row.key) {
          configMapObj.metadata.annotations[row.key] = row.value
        }
      })
    }

    if (isEditMode.value) {
      // 编辑模式：使用 PUT 请求
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/${formData.value.name}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('更新成功')
    } else {
      // 创建模式：使用 POST 请求
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('创建成功')
    }

    formDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 删除 ConfigMap
const handleDelete = async (row: ConfigMapInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 ConfigMap ${row.name} 吗？此操作不可恢复！`,
      '删除 ConfigMap 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 数据行操作
const addDataRow = () => {
  formData.value.data.push({ key: '', value: '' })
}

const removeDataRow = (index: number) => {
  formData.value.data.splice(index, 1)
}

// BinaryData 行操作
const addBinaryDataRow = () => {
  formData.value.binaryData.push({ key: '', value: '' })
}

const removeBinaryDataRow = (index: number) => {
  formData.value.binaryData.splice(index, 1)
}

// 标签行操作
const addLabelRow = () => {
  formData.value.labels.push({ key: '', value: '' })
}

const removeLabelRow = (index: number) => {
  formData.value.labels.splice(index, 1)
}

// 注解行操作
const addAnnotationRow = () => {
  formData.value.annotations.push({ key: '', value: '' })
}

const removeAnnotationRow = (index: number) => {
  formData.value.annotations.splice(index, 1)
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
    loadConfigMaps()
  }
})

// 监听筛选后的数据变化，更新计数
watch(filteredConfigMaps, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadConfigMaps()
  }
})

// 暴露方法给父组件
defineExpose({
  loadConfigMaps
})
</script>

<style scoped>
.configmap-list {
  padding: 0;
}

/* 搜索栏 */
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
  font-weight: 600;
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

/* 表单弹窗 */
.form-dialog :deep(.el-dialog__body) {
  padding: 20px 24px;
  max-height: 600px;
  overflow-y: auto;
}

.configmap-form {
  max-width: 100%;
}

.form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.form-row .el-form-item {
  flex: 1;
  margin-bottom: 0;
}

.form-tabs {
  margin-top: 16px;
}

.tab-content {
  padding: 16px 0;
}

.table-actions-wrapper {
  margin-bottom: 12px;
}

.data-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.binarydata-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.data-title {
  font-weight: 600;
  color: #333;
}

.metadata-section {
  margin-bottom: 24px;
}

.metadata-section:last-child {
  margin-bottom: 0;
}

.metadata-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.metadata-title {
  font-weight: 600;
  color: #333;
}

.table-header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.table-title {
  font-weight: 600;
  color: #d4af37;
}

.form-table {
  width: 100%;
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

/* ConfigMap 创建/编辑弹窗美化 */
.configmap-editor-dialog {
  --cfg-primary: #2563eb;
  --cfg-border: #e6ebf2;
  --cfg-muted: #667085;
  --cfg-ink: #111827;
}

.configmap-list {
  --cfg-primary: #2563eb;
  --cfg-border: #e6ebf2;
  --cfg-muted: #667085;
  --cfg-ink: #111827;
}

.configmap-list .search-bar,
.configmap-list .table-wrapper {
  border: 1px solid var(--cfg-border);
  border-radius: 18px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.05);
}

.configmap-list .search-icon,
.configmap-list .header-icon-blue {
  color: var(--cfg-primary);
}

.configmap-list .name-icon-wrapper {
  border: 0;
  background: #eff6ff;
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.16);
}

.configmap-list .name-icon {
  color: var(--cfg-primary);
}

.configmap-list .name-text {
  color: var(--cfg-ink);
  font-weight: 800;
}

.configmap-list .modern-table :deep(.el-table__header th) {
  background: #f8fbff !important;
  color: #475467;
  font-weight: 900;
  border-bottom: 1px solid var(--cfg-border);
}

.configmap-list .modern-table :deep(.el-table__row td) {
  border-bottom: 1px solid #edf2f7;
}

.configmap-list .modern-table :deep(.el-table__row:hover) {
  background: #f8fbff !important;
}

.configmap-editor-dialog :deep(.el-dialog) {
  border-radius: 24px;
  overflow: hidden;
  background: #f7faff;
  box-shadow: 0 28px 80px rgba(15, 23, 42, 0.22);
}

.configmap-editor-dialog :deep(.el-dialog__header) {
  padding: 20px 26px;
  margin: 0;
  background: #ffffff;
  border-bottom: 1px solid var(--cfg-border);
}

.configmap-editor-dialog :deep(.el-dialog__title) {
  color: var(--cfg-ink);
  font-size: 20px;
  font-weight: 900;
}

.configmap-editor-dialog :deep(.el-dialog__body) {
  padding: 22px;
  max-height: min(74vh, 820px);
  background:
    radial-gradient(circle at top left, rgba(37, 99, 235, 0.08), transparent 28%),
    #f7faff;
}

.configmap-editor-dialog :deep(.el-dialog__footer) {
  background: #ffffff;
  border-top: 1px solid var(--cfg-border);
}

.configmap-hero {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 18px;
  padding: 20px 22px;
  border: 1px solid var(--cfg-border);
  border-radius: 22px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(248, 251, 255, 0.92)),
    radial-gradient(circle at 90% 10%, rgba(37, 99, 235, 0.13), transparent 32%);
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.06);
}

.configmap-hero-kicker {
  width: fit-content;
  margin-bottom: 8px;
  padding: 5px 10px;
  border-radius: 999px;
  background: #eff6ff;
  color: var(--cfg-primary);
  border: 1px solid #bfdbfe;
  font-size: 12px;
  font-weight: 900;
}

.configmap-hero-title {
  color: var(--cfg-ink);
  font-size: 26px;
  font-weight: 900;
  letter-spacing: -0.04em;
}

.configmap-hero p {
  margin: 8px 0 0;
  color: var(--cfg-muted);
}

.configmap-hero-badges {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.configmap-hero-badges span {
  padding: 8px 10px;
  border: 1px solid #dbeafe;
  border-radius: 999px;
  background: #ffffff;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 900;
}

.resource-fields {
  padding: 18px;
  border: 1px solid var(--cfg-border);
  border-radius: 18px;
  background: #ffffff;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.configmap-form :deep(.el-input__wrapper),
.configmap-form :deep(.el-select .el-input__wrapper),
.configmap-form :deep(.el-textarea__inner) {
  border: 1px solid #d7e2f2;
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.configmap-form :deep(.el-input__wrapper:hover),
.configmap-form :deep(.el-input__wrapper.is-focus),
.configmap-form :deep(.el-select .el-input__wrapper:hover),
.configmap-form :deep(.el-select .el-input__wrapper.is-focus),
.configmap-form :deep(.el-textarea__inner:hover),
.configmap-form :deep(.el-textarea__inner:focus) {
  border-color: var(--cfg-primary);
  box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.1);
}

.form-tabs {
  overflow: hidden;
  border: 1px solid var(--cfg-border);
  border-radius: 18px;
  background: #ffffff;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.05);
}

.form-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 18px;
  background: #ffffff;
  border-bottom: 1px solid var(--cfg-border);
}

.form-tabs :deep(.el-tabs__item) {
  height: 54px;
  line-height: 54px;
  color: #475467;
  font-weight: 900;
}

.form-tabs :deep(.el-tabs__item.is-active),
.form-tabs :deep(.el-tabs__item:hover) {
  color: var(--cfg-primary);
}

.form-tabs :deep(.el-tabs__active-bar) {
  background: var(--cfg-primary);
}

.tab-content {
  padding: 18px;
}

.data-section,
.binarydata-section,
.metadata-section {
  margin-bottom: 18px;
  padding: 18px;
  border: 1px solid var(--cfg-border);
  border-radius: 18px;
  background: #f8fbff;
}

.section-title,
.metadata-title {
  color: var(--cfg-ink);
  font-weight: 900;
}

.section-header .el-button,
.metadata-header .el-button {
  border-radius: 999px;
  background: var(--cfg-primary);
  border-color: var(--cfg-primary);
  color: #ffffff;
  font-weight: 800;
}

.form-table {
  overflow: hidden;
  border: 1px solid var(--cfg-border);
  border-radius: 14px;
}

.form-table :deep(.el-table__header th) {
  background: #ffffff !important;
  color: #475467;
  font-weight: 900;
  border-bottom: 1px solid var(--cfg-border);
}

.form-table :deep(.el-table__row td) {
  background: #ffffff;
  border-bottom: 1px solid #edf2f7;
  vertical-align: top;
}

.config-data-table :deep(.el-textarea__inner) {
  min-height: 118px !important;
  font-family: 'SFMono-Regular', 'Cascadia Code', 'Menlo', monospace;
  line-height: 1.65;
}

.black-button {
  background: #1f2937 !important;
  border-color: #1f2937 !important;
  color: #ffffff !important;
  border-radius: 12px;
}

.black-button:hover {
  background: #111827 !important;
  border-color: #111827 !important;
}

.yaml-dialog :deep(.el-dialog__header) {
  background: #ffffff;
  border-bottom: 1px solid #e6ebf2;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #111827;
}

.yaml-dialog :deep(.el-dialog__body) {
  background: #f7faff;
}

.yaml-editor-wrapper {
  border-color: #d7e2f2;
  border-radius: 16px;
  background: #ffffff;
}

.yaml-line-numbers {
  background: #f8fbff;
  color: #98a2b3;
  border-right-color: #e6ebf2;
}

.yaml-textarea {
  background: #ffffff;
  color: #1f2937;
}

@media (max-width: 960px) {
  .configmap-hero,
  .form-row {
    flex-direction: column;
  }
}
</style>
