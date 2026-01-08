<template>
  <div class="workloads-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Tools /></el-icon>
        </div>
        <div>
          <h2 class="page-title">工作负载</h2>
          <p class="page-subtitle">管理 Kubernetes 工作负载资源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="loadWorkloads">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchName"
          placeholder="搜索工作负载名称..."
          clearable
          @clear="handleSearch"
          @keyup.enter="handleSearch"
          @input="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>

        <el-select
          v-model="selectedNamespace"
          placeholder="命名空间"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><FolderOpened /></el-icon>
          </template>
          <el-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </el-select>

        <el-select
          v-model="selectedType"
          placeholder="工作负载类型"
          @change="handleTypeChange"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><Grid /></el-icon>
          </template>
          <el-option label="所有" value="" />
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
        </el-select>
      </div>
    </div>

    <!-- 工作负载列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedWorkloadList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
        <el-table-column label="名称" min-width="220" fixed="left">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Tools /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="workload-name-cell">
              <div class="workload-name-content">
                <div class="workload-name golden-text">{{ row.name }}</div>
                <div class="workload-namespace">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="标签" width="120" align="center">
          <template #default="{ row }">
            <div class="label-cell" @click="showLabels(row)">
              <div class="label-badge-wrapper">
                <span class="label-count">{{ Object.keys(row.labels || {}).length }}</span>
                <el-icon class="label-icon"><PriceTag /></el-icon>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="容器组" width="150" align="center">
          <template #default="{ row }">
            <div class="pod-count-cell">
              <span class="pod-count">{{ row.readyPods || 0 }}/{{ row.desiredPods || 0 }}</span>
              <span class="pod-label">Pods</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Requests/Limits" min-width="200">
          <template #default="{ row }">
            <div class="resource-cell">
              <div v-if="row.requests?.cpu || row.limits?.cpu" class="resource-item">
                <span class="resource-label">CPU:</span>
                <span v-if="row.requests?.cpu" class="resource-value requests-value">{{ row.requests.cpu }}</span>
                <span v-if="row.requests?.cpu && row.limits?.cpu" class="resource-separator">/</span>
                <span v-if="row.limits?.cpu" class="resource-value limits-value">{{ row.limits.cpu }}</span>
              </div>
              <div v-if="row.requests?.memory || row.limits?.memory" class="resource-item">
                <span class="resource-label">Mem:</span>
                <span v-if="row.requests?.memory" class="resource-value requests-value">{{ row.requests.memory }}</span>
                <span v-if="row.requests?.memory && row.limits?.memory" class="resource-separator">/</span>
                <span v-if="row.limits?.memory" class="resource-value limits-value">{{ row.limits.memory }}</span>
              </div>
              <div v-if="!row.requests?.cpu && !row.requests?.memory && !row.limits?.cpu && !row.limits?.memory" class="resource-empty">-</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="镜像" min-width="300">
          <template #default="{ row }">
            <div class="image-cell">
              <el-tooltip
                v-if="row.images && row.images.length > 0"
                :content="row.images.join('\n')"
                placement="top"
              >
                <div class="image-list">
                  <span v-for="(image, index) in getDisplayImages(row.images)" :key="index" class="image-item">
                    {{ image }}
                  </span>
                  <span v-if="row.images.length > 2" class="image-more">
                    +{{ row.images.length - 2 }}
                  </span>
                </div>
              </el-tooltip>
              <span v-else class="image-empty">-</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" width="150">
          <template #default="{ row }">
            <div class="age-cell">
              <el-icon class="age-icon"><Clock /></el-icon>
              <span>{{ formatAge(row.createdAt) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-dropdown trigger="click" @command="(command: string) => handleActionCommand(command, row)">
              <el-button link class="action-btn">
                <el-icon :size="18"><Edit /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu class="action-dropdown-menu">
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    <span>编辑</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="yaml">
                    <el-icon><Document /></el-icon>
                    <span>YAML</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="pods">
                    <el-icon><Monitor /></el-icon>
                    <span>Pods</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="restart" divided>
                    <el-icon><RefreshRight /></el-icon>
                    <span>重启</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="scale">
                    <el-icon><Rank /></el-icon>
                    <span>扩缩容</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided class="danger-item">
                    <el-icon><Delete /></el-icon>
                    <span>删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredWorkloadList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="工作负载标签"
      width="700px"
      class="label-dialog"
    >
      <div class="label-dialog-content">
        <el-table :data="labelList" class="label-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="280">
            <template #default="{ row }">
              <div class="label-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="label-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="350">
            <template #default="{ row }">
              <span class="label-value">{{ row.value }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="labelDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="`工作负载 YAML - ${selectedWorkload?.name || ''}`"
      width="900px"
      class="yaml-dialog"
    >
      <div class="yaml-dialog-content">
        <div class="yaml-editor-wrapper">
          <div class="yaml-line-numbers">
            <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
          </div>
          <textarea
            v-model="yamlContent"
            class="yaml-textarea"
            placeholder="YAML 内容"
            spellcheck="false"
            @input="handleYamlInput"
            @scroll="handleYamlScroll"
            ref="yamlTextarea"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="handleSaveYAML" :loading="yamlSaving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import {
  Search,
  Tools,
  Grid,
  Platform,
  FolderOpened,
  PriceTag,
  Clock,
  Refresh,
  Edit,
  View,
  Document,
  Monitor,
  RefreshRight,
  Rank,
  Delete,
  CopyDocument
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'

// 工作负载接口定义
interface Workload {
  name: string
  namespace: string
  type: string
  labels?: Record<string, string>
  readyPods?: number
  desiredPods?: number
  requests?: { cpu: string; memory: string }
  limits?: { cpu: string; memory: string }
  images?: string[]
  createdAt?: string
  updatedAt?: string
}

interface Namespace {
  name: string
}

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const namespaceList = ref<Namespace[]>([])
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>('')
const selectedType = ref<string>('')
const workloadList = ref<Workload[]>([])

// 搜索条件
const searchName = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlSaving = ref(false)
const selectedWorkload = ref<Workload | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 过滤后的工作负载列表
const filteredWorkloadList = computed(() => {
  let result = workloadList.value

  if (searchName.value) {
    result = result.filter(workload =>
      workload.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  return result
})

// 分页后的工作负载列表
const paginatedWorkloadList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredWorkloadList.value.slice(start, end)
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 获取类型图标
const getTypeIcon = (type: string) => {
  return Tools
}

// 格式化资源显示
const formatResource = (resource: { cpu: string; memory: string }) => {
  const parts: string[] = []
  if (resource.cpu) parts.push(`cpu: ${resource.cpu}`)
  if (resource.memory) parts.push(`mem: ${resource.memory}`)
  return parts.join(' | ')
}

// 格式化存活时间
const formatAge = (createdAt: string | undefined): string => {
  if (!createdAt) return '-'

  const created = new Date(createdAt)
  const now = new Date()
  const diffMs = now.getTime() - created.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays < 1) {
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    if (diffHours < 1) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60))
      return diffMinutes < 1 ? '刚刚' : `${diffMinutes}分钟前`
    }
    return `${diffHours}小时前`
  }

  if (diffDays < 7) {
    return `${diffDays}天前`
  }

  const diffWeeks = Math.floor(diffDays / 7)
  if (diffWeeks < 4) {
    return `${diffWeeks}周前`
  }

  const diffMonths = Math.floor(diffDays / 30)
  if (diffMonths < 12) {
    return `${diffMonths}个月前`
  }

  const diffYears = Math.floor(diffDays / 365)
  return `${diffYears}年前`
}

// 获取显示的镜像（最多显示2个）
const getDisplayImages = (images?: string[]) => {
  if (!images || images.length === 0) return []
  return images.slice(0, 2).map(img => {
    // 只保留镜像名和tag，去掉registry部分
    const parts = img.split('/')
    const nameAndTag = parts[parts.length - 1]
    // 如果tag太长，截断显示
    if (nameAndTag.length > 50) {
      return nameAndTag.substring(0, 50) + '...'
    }
    return nameAndTag
  })
}

// 显示标签弹窗
const showLabels = (row: Workload) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 复制到剪贴板
const copyToClipboard = async (text: string, type: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${type} 已复制到剪贴板`)
  } catch (error) {
    // 降级方案：使用传统方法
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      ElMessage.success(`${type} 已复制到剪贴板`)
    } catch (err) {
      ElMessage.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredWorkloadList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('workloads_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNamespaces()
      await loadWorkloads()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    namespaceList.value = response.data.data || []
  } catch (error) {
    console.error(error)
    namespaceList.value = []
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('workloads_selected_cluster_id', selectedClusterId.value.toString())
  }
  selectedNamespace.value = ''
  currentPage.value = 1
  await loadNamespaces()
  await loadWorkloads()
}

// 切换工作负载类型
const handleTypeChange = () => {
  currentPage.value = 1
  loadWorkloads()
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 加载工作负载列表
const loadWorkloads = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const params: any = { clusterId: selectedClusterId.value }
    if (selectedType.value) params.type = selectedType.value
    if (selectedNamespace.value) params.namespace = selectedNamespace.value

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads`,
      {
        params,
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    workloadList.value = response.data.data || []
  } catch (error) {
    console.error(error)
    workloadList.value = []
    ElMessage.error('获取工作负载列表失败')
  } finally {
    loading.value = false
  }
}

// 处理下拉菜单命令
const handleActionCommand = (command: string, row: Workload) => {
  selectedWorkload.value = row

  switch (command) {
    case 'edit':
      handleShowEditDialog()
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'pods':
      ElMessage.info('Pods 列表功能开发中...')
      break
    case 'restart':
      handleRestart()
      break
    case 'scale':
      handleScale()
      break
    case 'delete':
      handleDelete()
      break
  }
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const workloadType = selectedWorkload.value.type.toLowerCase()
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/${workloadType}s/${namespace}/${name}/yaml`,
      {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    console.error('获取 YAML 失败:', error)
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  } finally {
    yamlSaving.value = false
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const workloadType = selectedWorkload.value.type.toLowerCase()
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    await axios.put(
      `/api/v1/plugins/kubernetes/resources/${workloadType}s/${namespace}/${name}/yaml`,
      {
        clusterId,
        yaml: yamlContent.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    await loadWorkloads()
  } catch (error) {
    console.error('保存 YAML 失败:', error)
    ElMessage.error('保存 YAML 失败')
  } finally {
    yamlSaving.value = false
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 输入时自动调整滚动
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 重启工作负载
const handleRestart = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要重启工作负载 ${selectedWorkload.value.name} 吗？`,
      '重启确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/restart`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('工作负载重启成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('重启失败:', error)
      ElMessage.error(`重启失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 扩缩容工作负载
const handleScale = async () => {
  if (!selectedWorkload.value) return

  try {
    const { value } = await ElMessageBox.prompt(
      `请输入 ${selectedWorkload.value.name} 的副本数：`,
      '扩缩容',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: selectedWorkload.value.desiredPods?.toString() || '1',
        confirmButtonClass: 'black-button'
      }
    )

    const replicas = parseInt(value)
    if (isNaN(replicas) || replicas < 0) {
      ElMessage.error('请输入有效的副本数')
      return
    }

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/scale`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type,
        replicas
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('扩缩容成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('扩缩容失败:', error)
      ElMessage.error(`扩缩容失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 显示编辑对话框
const handleShowEditDialog = async () => {
  if (!selectedWorkload.value) return

  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const workloadType = selectedWorkload.value.type
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        params: { clusterId, type: workloadType },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    // 获取返回的 JSON 数据
    const workloadData = response.data.data?.items
    if (workloadData) {
      console.log('获取到工作负载数据:', workloadData)
      ElMessage.success('工作负载详情获取成功！')
      // TODO: 打开编辑对话框并填充数据
    } else {
      ElMessage.warning('未获取到工作负载数据')
    }
  } catch (error: any) {
    console.error('获取工作负载详情失败:', error)
    ElMessage.error(`获取工作负载详情失败: ${error.response?.data?.message || error.message}`)
  }
}

// 删除工作负载
const handleDelete = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除工作负载 ${selectedWorkload.value.name} 吗？此操作不可恢复！`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}`,
      {
        params: {
          clusterId: selectedClusterId.value,
          type: selectedWorkload.value.type
        },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.workloads-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
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

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.filter-select,
.cluster-select {
  width: 180px;
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

/* 搜索框样式优化 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-bar :deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
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

/* 现代表格 */
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

.modern-table :deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 工作负载名称单元格 */
.workload-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workload-icon-wrapper {
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

.workload-icon {
  color: #d4af37;
  font-size: 18px;
}

.workload-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.workload-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.golden-text {
  color: #d4af37 !important;
}

.workload-namespace {
  font-size: 12px;
  color: #909399;
}

/* 标签单元格 */
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
  gap: 8px;
}

.label-icon {
  color: #d4af37;
  font-size: 20px;
  transition: all 0.3s;
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
  border-color: #bfa13f;
}

/* Pod 数量 */
.pod-count-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.pod-count {
  font-size: 18px;
  font-weight: 600;
  color: #d4af37;
}

.pod-label {
  font-size: 11px;
  color: #909399;
}

/* 资源单元格 */
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.resource-label {
  color: #909399;
  font-weight: 500;
  min-width: 45px;
}

.resource-value {
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
}

.requests-value {
  color: #67c23a;
}

.limits-value {
  color: #e6a23c;
}

.resource-separator {
  color: #dcdfe6;
  margin: 0 4px;
}

.resource-empty {
  color: #909399;
}

/* 镜像单元格 */
.image-cell {
  display: flex;
  align-items: center;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.image-item {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-more {
  font-size: 11px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.image-empty {
  color: #909399;
  font-size: 13px;
}

/* 时间单元格 */
.age-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.age-icon {
  color: #d4af37;
}

/* 操作按钮 */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #d4af37;
}

.action-btn:hover {
  color: #bfa13f;
}

/* 下拉菜单样式 */
.action-dropdown-menu {
  min-width: 140px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item .el-icon) {
  color: #d4af37;
  font-size: 16px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item .el-icon) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover) {
  background-color: #f5f5f5;
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover) {
  background-color: #fef0f0;
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover .el-icon) {
  color: #f56c6c;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* 标签弹窗 */
.label-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.label-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.label-dialog-content {
  padding: 8px 0;
}

.label-table {
  width: 100%;
}

.label-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.label-key-wrapper {
  display: inline-flex !important;
  align-items: center !important;
  gap: 6px !important;
  padding: 5px 12px !important;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%) !important;
  color: #d4af37 !important;
  border: 1px solid #d4af37 !important;
  border-radius: 6px !important;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 12px !important;
  font-weight: 500 !important;
  cursor: pointer !important;
  transition: all 0.3s !important;
  user-select: none;
}

.label-key-wrapper:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%) !important;
  border-color: #bfa13f !important;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3) !important;
  transform: translateY(-1px);
}

.label-key-wrapper:active {
  transform: translateY(0);
}

.label-key-text {
  flex: 1;
  word-break: break-all;
  line-height: 1.4;
  white-space: pre-wrap;
}

.copy-icon {
  font-size: 14px;
  flex-shrink: 0;
  opacity: 0.7;
  transition: opacity 0.3s;
}

.label-key-wrapper:hover .copy-icon {
  opacity: 1;
}

.label-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
  white-space: pre-wrap;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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

.yaml-dialog-content {
  padding: 0;
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

/* 响应式设计 */
@media (max-width: 1400px) {
  .search-inputs {
    flex-wrap: wrap;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
  }

  .cluster-select,
  .filter-select {
    width: 100%;
  }
}
</style>
