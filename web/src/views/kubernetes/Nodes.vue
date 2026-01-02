<template>
  <div class="nodes-container">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-blue">
          <el-icon><Monitor /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">节点总数</div>
          <div class="stat-value">{{ nodeList.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-green">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">运行正常</div>
          <div class="stat-value">{{ readyNodeCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-orange">
          <el-icon><Odometer /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">Pod总数</div>
          <div class="stat-value">{{ totalPodCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-purple">
          <el-icon><CPU /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">总CPU核数</div>
          <div class="stat-value">{{ totalCPUCores }}</div>
        </div>
      </div>
    </div>

    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">节点管理</h2>
          <p class="page-subtitle">管理 Kubernetes 集群节点，监控节点状态和资源使用</p>
        </div>
      </div>
      <div class="header-actions">
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
        <el-button class="black-button" @click="loadNodes">
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
          placeholder="搜索节点名称..."
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
          v-model="searchStatus"
          placeholder="节点状态"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><CircleCheck /></el-icon>
          </template>
          <el-option label="正常" value="Ready" />
          <el-option label="异常" value="NotReady" />
        </el-select>

        <el-select
          v-model="searchRole"
          placeholder="节点角色"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><User /></el-icon>
          </template>
          <el-option label="Master" value="master" />
          <el-option label="Control Plane" value="control-plane" />
          <el-option label="Worker" value="worker" />
        </el-select>
      </div>
    </div>

    <!-- 节点列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedNodeList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
      <el-table-column label="节点名称" min-width="220" fixed="left">
        <template #header>
          <span class="header-with-icon">
            <el-icon class="header-icon header-icon-blue"><Monitor /></el-icon>
            节点名称
          </span>
        </template>
        <template #default="{ row }">
          <div class="node-name-cell">
            <div class="node-icon-wrapper">
              <el-icon class="node-icon" :size="18"><Platform /></el-icon>
            </div>
            <div class="node-name-content">
              <div class="node-name link-text" @click="goToNodeDetail(row)">{{ row.name }}</div>
              <div class="node-ip">{{ row.internalIP }}</div>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag
            :type="row.status === 'Ready' ? 'success' : 'danger'"
            effect="dark"
            size="large"
            class="status-tag"
          >
            {{ row.status === 'Ready' ? '正常' : '异常' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="角色" width="130" align="center">
        <template #default="{ row }">
          <div :class="['role-badge', 'role-' + (row.roles || 'worker')]">
            <el-icon :size="14"><User /></el-icon>
            <span>{{ getRoleText(row.roles) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="Kubelet版本" width="140">
        <template #default="{ row }">
          <div class="version-cell">
            <el-icon class="version-icon"><InfoFilled /></el-icon>
            <span>{{ row.version || '-' }}</span>
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

      <el-table-column label="运行时间" width="140">
        <template #default="{ row }">
          <div class="age-cell">
            <el-icon class="age-icon"><Clock /></el-icon>
            <span>{{ row.age || '-' }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="CPU" width="140">
        <template #default="{ row }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-cpu">
              <el-icon><Cpu /></el-icon>
            </div>
            <span class="resource-value">{{ formatCPU(row.cpuCapacity) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="内存" width="140">
        <template #default="{ row }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-memory">
              <el-icon><Coin /></el-icon>
            </div>
            <span class="resource-value">{{ formatMemory(row.memoryCapacity) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="Pod数量" width="110" align="center">
        <template #default="{ row }">
          <div class="pod-count-cell">
            <span class="pod-count">{{ row.podCount ?? 0 }}/{{ row.podCapacity ?? 0 }}</span>
            <span class="pod-label">Pods</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="调度状态" width="120" align="center">
        <template #default="{ row }">
          <el-tag
            :type="row.schedulable ? 'success' : 'warning'"
            effect="dark"
            size="large"
          >
            {{ row.schedulable ? '可调度' : '不可调度' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="污点" width="120" align="center">
        <template #default="{ row }">
          <div class="taint-cell" @click="showTaints(row)">
            <div class="taint-badge-wrapper">
              <el-icon class="taint-icon"><WarnTriangleFilled /></el-icon>
              <span class="taint-count">{{ row.taintCount ?? 0 }}</span>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="100" fixed="right" align="center">
        <template #default="{ row }">
          <el-button
            type="primary"
            link
            @click="handleViewDetails(row)"
            class="action-btn"
          >
            <el-icon><View /></el-icon>
            详情
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
          :total="filteredNodeList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="节点标签"
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

    <!-- 污点弹窗 -->
    <el-dialog
      v-model="taintDialogVisible"
      title="节点污点"
      width="700px"
      class="taint-dialog"
    >
      <div class="taint-dialog-content">
        <el-table :data="taintList" class="taint-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="200">
            <template #default="{ row }">
              <div class="taint-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="taint-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="200">
            <template #default="{ row }">
              <span class="taint-value">{{ row.value || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="effect" label="Effect" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="getEffectTagType(row.effect)" class="effect-tag">
                {{ row.effect }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="taintDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Search,
  PriceTag,
  Monitor,
  Platform,
  CircleCheck,
  User,
  InfoFilled,
  Clock,
  Cpu,
  Coin,
  View,
  Odometer,
  Refresh,
  CopyDocument,
  WarnTriangleFilled
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster, getNodes, type NodeInfo } from '@/api/kubernetes'

const loading = ref(false)
const router = useRouter()
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const nodeList = ref<NodeInfo[]>([])

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')
const searchRole = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('nodes_pagination')

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])

// 污点弹窗
const taintDialogVisible = ref(false)
const taintList = ref<{ key: string; value: string; effect: string }[]>([])

// 过滤后的节点列表
const filteredNodeList = computed(() => {
  let result = nodeList.value

  if (searchName.value) {
    result = result.filter(node =>
      node.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (searchStatus.value) {
    result = result.filter(node => node.status === searchStatus.value)
  }

  if (searchRole.value) {
    result = result.filter(node => node.roles === searchRole.value)
  }

  return result
})

// 分页后的节点列表
const paginatedNodeList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredNodeList.value.slice(start, end)
})

// 统计数据
const readyNodeCount = computed(() => {
  return nodeList.value.filter(node => node.status === 'Ready').length
})

const totalPodCount = computed(() => {
  const usedPods = nodeList.value.reduce((sum, node) => sum + (node.podCount || 0), 0)
  const totalPods = nodeList.value.reduce((sum, node) => sum + (node.podCapacity || 0), 0)
  return `${usedPods}/${totalPods}`
})

const totalCPUCores = computed(() => {
  let totalCores = 0
  nodeList.value.forEach(node => {
    if (node.cpuCapacity) {
      const cores = parseCPU(node.cpuCapacity)
      totalCores += cores
    }
  })
  return totalCores.toFixed(1)
})

// 解析 CPU 核数
const parseCPU = (cpu: string): number => {
  if (!cpu) return 0
  if (cpu.endsWith('m')) {
    return parseInt(cpu) / 1000
  }
  return parseFloat(cpu) || 0
}

// 格式化 CPU 显示
const formatCPU = (cpu: string) => {
  if (!cpu) return '-'
  if (cpu.endsWith('m')) {
    const millicores = parseInt(cpu)
    if (isNaN(millicores)) return cpu
    return (millicores / 1000).toFixed(2) + ' 核'
  }
  return cpu + ' 核'
}

// 格式化内存显示
const formatMemory = (memory: string) => {
  if (!memory) return '-'

  const match = memory.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return memory

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  if (!unit) {
    const bytes = value
    const tb = bytes / (1024 * 1024 * 1024 * 1024)
    if (tb >= 1) return Math.ceil(tb) + ' TB'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return Math.ceil(gb) + ' GB'
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return Math.ceil(mb) + ' MB'
    return memory
  }

  let bytes = 0
  switch (unit) {
    case 'KI':
      bytes = value * 1024
      break
    case 'MI':
      bytes = value * 1024 * 1024
      break
    case 'GI':
      bytes = value * 1024 * 1024 * 1024
      break
    case 'TI':
      bytes = value * 1024 * 1024 * 1024 * 1024
      break
  }

  const tb = bytes / (1024 * 1024 * 1024 * 1024)
  if (tb >= 1) return Math.ceil(tb) + ' TB'

  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return Math.ceil(gb) + ' GB'

  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return Math.ceil(mb) + ' MB'

  return memory
}

// 获取角色文本
const getRoleText = (role: string | undefined) => {
  if (!role) return 'Worker'
  if (role === 'master') return 'Master'
  if (role === 'control-plane') return 'Control Plane'
  if (role === 'worker') return 'Worker'
  return role
}

// 获取 Effect 标签类型
const getEffectTagType = (effect: string) => {
  switch (effect) {
    case 'NoSchedule':
      return 'warning'
    case 'NoExecute':
      return 'danger'
    case 'PreferNoSchedule':
      return 'info'
    default:
      return ''
  }
}

// 显示标签弹窗
const showLabels = (row: NodeInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 跳转到节点详情页
const goToNodeDetail = (row: NodeInfo) => {
  const cluster = clusterList.value.find(c => c.id === selectedClusterId.value)
  router.push({
    name: 'K8sNodeDetail',
    params: {
      clusterId: selectedClusterId.value,
      nodeName: row.name
    },
    query: {
      clusterName: cluster?.alias || cluster?.name
    }
  })
}

// 显示污点弹窗
const showTaints = (row: NodeInfo) => {
  const taints = row.taints || []
  taintList.value = taints.map(taint => ({
    key: taint.key,
    value: taint.value || '',
    effect: taint.effect
  }))
  taintDialogVisible.value = true
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

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
    console.error('保存分页状态失败:', error)
  }
}

// 从 localStorage 恢复分页状态
const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch (error) {
    console.error('恢复分页状态失败:', error)
    currentPage.value = 1
    pageSize.value = 10
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredNodeList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
  savePaginationState()
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('nodes_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNodes()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('nodes_selected_cluster_id', selectedClusterId.value.toString())
  }
  // 切换集群时重置分页
  currentPage.value = 1
  await loadNodes()
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const data = await getNodes(selectedClusterId.value)
    nodeList.value = data || []
    // 恢复分页状态
    restorePaginationState()
  } catch (error) {
    console.error(error)
    nodeList.value = []
    ElMessage.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  // 搜索时重置到第一页
  currentPage.value = 1
  savePaginationState()
}

// 查看详情
const handleViewDetails = (row: NodeInfo) => {
  console.log('查看节点详情:', row)
  ElMessage.info('详情功能开发中...')
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.nodes-container {
  padding: 0;
  background-color: transparent;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  flex-shrink: 0;
}

.stat-icon-blue {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-green {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-orange {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-purple {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #d4af37;
  line-height: 1;
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

.cluster-select {
  width: 280px;
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

.filter-select {
  width: 150px;
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

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 8px;
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

/* 节点名称单元格 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.node-icon-wrapper {
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

.node-icon {
  color: #d4af37;
}

.node-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.node-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.link-text {
  color: #d4af37;
  cursor: pointer;
  transition: all 0.3s;
}

.link-text:hover {
  color: #bfa13f;
  text-decoration: underline;
}

.node-ip {
  font-size: 12px;
  color: #909399;
}

/* 角色标签 */
.role-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 14px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
}

.role-master {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.role-control-plane {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.role-worker {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #ffffff;
  border: 1px solid #dcdfe6;
}

/* 版本单元格 */
.version-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.version-icon {
  color: #d4af37;
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

/* 资源单元格 */
.resource-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.resource-icon-cpu {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.resource-icon-memory {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.resource-value {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
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

/* 污点单元格 */
.taint-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.taint-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.taint-icon {
  color: #d4af37;
  font-size: 20px;
  transition: all 0.3s;
}

.taint-count {
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

.taint-cell:hover .taint-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.taint-cell:hover .taint-count {
  background-color: #bfa13f;
  border-color: #bfa13f;
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

/* 操作按钮 */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* 状态标签 */
.status-tag {
  border-radius: 8px;
  padding: 6px 14px;
  font-weight: 500;
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

/* 污点弹窗 */
.taint-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.taint-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.taint-dialog-content {
  padding: 8px 0;
}

.taint-table {
  width: 100%;
}

.taint-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.taint-key-wrapper {
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

.taint-key-wrapper:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%) !important;
  border-color: #bfa13f !important;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3) !important;
  transform: translateY(-1px);
}

.taint-key-wrapper:active {
  transform: translateY(0);
}

.taint-key-text {
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

.taint-key-wrapper:hover .copy-icon {
  opacity: 1;
}

.taint-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
  white-space: pre-wrap;
}

.effect-tag {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
  }

  .cluster-select {
    width: 100%;
  }
}
</style>
