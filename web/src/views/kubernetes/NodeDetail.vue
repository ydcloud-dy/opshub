<template>
  <div class="node-detail-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-top">
          <el-button class="back-btn" @click="goBack" :icon="ArrowLeft">返回列表</el-button>
          <el-button class="black-button" @click="refreshData">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
        <div class="node-name-section">
          <h1 class="node-title">
            <el-icon class="title-icon" :size="28"><Monitor /></el-icon>
            {{ nodeName }}
          </h1>
          <el-tag v-if="nodeInfo.status === 'Ready'" type="success" effect="dark" size="large" class="status-tag">正常</el-tag>
          <el-tag v-else type="danger" effect="dark" size="large" class="status-tag">异常</el-tag>
        </div>
        <div class="node-meta">
          <span class="meta-item">
            <el-icon><Platform /></el-icon>
            {{ clusterName }}
          </span>
          <span class="meta-item">
            <el-icon><Connection /></el-icon>
            {{ nodeInfo.internalIP }}
          </span>
          <span class="meta-item" v-if="nodeInfo.version">
            <el-icon><InfoFilled /></el-icon>
            {{ nodeInfo.version }}
          </span>
        </div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-cpu">
          <el-icon><Cpu /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">CPU 使用率</div>
          <div class="stat-value">{{ cpuUsage }}%</div>
          <div class="stat-detail">{{ formatResource(nodeInfo.cpuCapacity) }}</div>
        </div>
        <div class="stat-progress">
          <div class="progress-bar" :style="{ width: cpuUsage + '%' }"></div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-memory">
          <el-icon><Coin /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">内存使用率</div>
          <div class="stat-value">{{ memoryUsage }}%</div>
          <div class="stat-detail">{{ formatMemory(nodeInfo.memoryCapacity) }}</div>
        </div>
        <div class="stat-progress">
          <div class="progress-bar" :style="{ width: memoryUsage + '%' }"></div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-pod">
          <el-icon><Odometer /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">运行 Pod</div>
          <div class="stat-value">{{ nodeInfo.podCount || 0 }}</div>
          <div class="stat-detail">/{{ nodeInfo.podCapacity || 110 }} Pods</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-uptime">
          <el-icon><Clock /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">运行时间</div>
          <div class="stat-value">{{ nodeInfo.age || '-' }}</div>
        </div>
      </div>
    </div>

    <!-- 信息卡片 -->
    <div class="info-grid">
      <!-- 基本信息 -->
      <div class="info-card">
        <div class="card-header">
          <el-icon class="header-icon"><InfoFilled /></el-icon>
          <h3>基本信息</h3>
        </div>
        <div class="card-body">
          <div class="info-row">
            <span class="info-label">节点名称</span>
            <span class="info-value">{{ nodeInfo.name }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">内部IP</span>
            <span class="info-value">{{ nodeInfo.internalIP }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">外部IP</span>
            <span class="info-value">{{ nodeInfo.externalIP || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">角色</span>
            <span class="info-value">{{ getRoleText(nodeInfo.roles) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">调度状态</span>
            <el-tag :type="nodeInfo.schedulable ? 'success' : 'warning'" size="small">
              {{ nodeInfo.schedulable ? '可调度' : '不可调度' }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- 系统信息 -->
      <div class="info-card">
        <div class="card-header">
          <el-icon class="header-icon"><Monitor /></el-icon>
          <h3>系统信息</h3>
        </div>
        <div class="card-body">
          <div class="info-row">
            <span class="info-label">操作系统</span>
            <span class="info-value">{{ nodeInfo.osImage }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">内核版本</span>
            <span class="info-value">{{ nodeInfo.kernelVersion }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">容器运行时</span>
            <span class="info-value">{{ nodeInfo.containerRuntime }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Kubelet 版本</span>
            <span class="info-value">{{ nodeInfo.version }}</span>
          </div>
        </div>
      </div>

      <!-- 网络信息 -->
      <div class="info-card">
        <div class="card-header">
          <el-icon class="header-icon"><Connection /></el-icon>
          <h3>网络信息</h3>
        </div>
        <div class="card-body">
          <div class="info-row">
            <span class="info-label">Pod CIDR</span>
            <span class="info-value">{{ nodeInfo.podCIDR || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Provider ID</span>
            <span class="info-value">{{ formatProviderID(nodeInfo.providerID) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 标签、注解和污点 -->
    <div class="labels-annotations-grid">
      <div class="info-card">
        <div class="card-header">
          <div class="header-left">
            <el-icon class="header-icon"><PriceTag /></el-icon>
            <h3>标签 ({{ Object.keys(nodeInfo.labels || {}).length }})</h3>
          </div>
        </div>
        <div class="card-body">
          <div class="tags-container">
            <div
              v-for="(value, key) in nodeInfo.labels"
              :key="key"
              class="tag-item"
            >
              <span class="tag-key">{{ key }}:</span>
              <span class="tag-value">{{ value !== undefined && value !== null && value !== '' ? value : '(空)' }}</span>
            </div>
            <div v-if="!nodeInfo.labels || Object.keys(nodeInfo.labels).length === 0" class="empty-tip">
              暂无标签
            </div>
          </div>
        </div>
      </div>

      <div class="info-card">
        <div class="card-header">
          <div class="header-left">
            <el-icon class="header-icon"><WarnTriangleFilled /></el-icon>
            <h3>污点 ({{ nodeInfo.taintCount || 0 }})</h3>
          </div>
        </div>
        <div class="card-body">
          <div class="tags-container">
            <div
              v-for="(taint, index) in nodeInfo.taints"
              :key="index"
              class="tag-item"
            >
              <span class="taint-key">{{ taint.key }}</span>
              <span v-if="taint.value" class="taint-separator">=</span>
              <span v-if="taint.value" class="taint-value">{{ taint.value }}</span>
              <span class="taint-separator">:</span>
              <span class="taint-effect" :class="getTaintEffectClass(taint.effect)">{{ taint.effect }}</span>
            </div>
            <div v-if="!nodeInfo.taints || nodeInfo.taints.length === 0" class="empty-tip">
              暂无污点
            </div>
          </div>
        </div>
      </div>

      <div class="info-card">
        <div class="card-header">
          <el-icon class="header-icon"><Document /></el-icon>
          <h3>注解 ({{ Object.keys(nodeInfo.annotations || {}).length }})</h3>
        </div>
        <div class="card-body">
          <div class="tags-container">
            <div
              v-for="(value, key) in nodeInfo.annotations"
              :key="key"
              class="tag-item"
            >
              <span class="tag-key">{{ key }}:</span>
              <span class="tag-value">{{ value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pod 列表 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-title">
          <el-icon class="title-icon"><Odometer /></el-icon>
          <h3>运行的 Pod ({{ pods.length }})</h3>
        </div>
        <div class="search-wrapper">
          <el-input
            v-model="podSearchKeyword"
            placeholder="搜索Pod名称或命名空间"
            prefix-icon="Search"
            clearable
            style="width: 280px"
            class="search-input"
          />
        </div>
      </div>
      <div class="table-wrapper">
        <el-table :data="paginatedPods" class="modern-table" v-loading="podsLoading">
          <el-table-column prop="name" label="Pod 名称" min-width="200">
            <template #default="{ row }">
              <div class="pod-name-cell">
                <span class="pod-name">{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="namespace" label="命名空间" width="150" />
          <el-table-column prop="ready" label="就绪" width="100" align="center" />
          <el-table-column prop="status" label="状态" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="ip" label="IP" width="140" />
          <el-table-column prop="age" label="运行时间" width="120" />
          <el-table-column prop="restarts" label="重启次数" width="100" align="center" />
        </el-table>
        <div class="pagination-wrapper">
          <el-pagination
            v-model:current-page="podCurrentPage"
            v-model:page-size="podPageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="filteredPods.length"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handlePodPageSizeChange"
            @current-change="handlePodPageChange"
          />
        </div>
      </div>
    </div>

    <!-- 节点状态信息 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-title">
          <el-icon class="title-icon"><InfoFilled /></el-icon>
          <h3>节点状态</h3>
        </div>
      </div>
      <div class="conditions-content">
        <el-table :data="nodeInfo.conditions || []" class="conditions-table">
          <el-table-column prop="type" label="Type" width="180" />
          <el-table-column prop="status" label="Status" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'True' ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="lastHeartbeatTime" label="LastHeartbeatTime" width="180" />
          <el-table-column prop="lastTransitionTime" label="LastTransitionTime" width="180" />
          <el-table-column prop="reason" label="Reason" width="180" />
          <el-table-column prop="message" label="Message" min-width="300" />
        </el-table>
      </div>
    </div>

    <!-- 事件列表 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-title">
          <el-icon class="title-icon"><Bell /></el-icon>
          <h3>事件 ({{ events.length }})</h3>
        </div>
      </div>
      <div class="table-wrapper">
        <el-table :data="recentEvents" class="modern-table" v-loading="eventsLoading">
          <el-table-column prop="type" label="类型" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.type === 'Warning' ? 'warning' : 'success'" size="small">
                {{ row.type }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" width="150" />
          <el-table-column prop="message" label="消息" min-width="300" />
          <el-table-column prop="source" label="来源" width="150" />
          <el-table-column prop="count" label="次数" width="80" align="center" />
          <el-table-column prop="lastTimestamp" label="最后时间" width="180" />
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Monitor,
  Refresh,
  Cpu,
  Coin,
  Odometer,
  Clock,
  InfoFilled,
  PriceTag,
  Connection,
  Document,
  Bell,
  Platform,
  Delete,
  WarnTriangleFilled
} from '@element-plus/icons-vue'
import { getNodes, type NodeInfo } from '@/api/kubernetes'
import axios from 'axios'

const route = useRoute()
const router = useRouter()

const clusterId = Number(route.params.clusterId)
const nodeName = route.params.nodeName as string
const clusterName = ref(route.query.clusterName as string || '')

const loading = ref(false)
const podsLoading = ref(false)
const eventsLoading = ref(false)

const nodeInfo = ref<Partial<NodeInfo>>({})
const networkInfo = ref<any>({})
const clusterInfo = ref<any>({})
const pods = ref<any[]>([])
const events = ref<any[]>([])
const cpuUsage = ref(0)
const memoryUsage = ref(0)
const podSearchKeyword = ref('')

// Pod 分页
const podCurrentPage = ref(1)
const podPageSize = ref(10)
const podPaginationStorageKey = ref(`node_detail_${nodeName}_pod_pagination`)

// 过滤后的Pod列表
const filteredPods = computed(() => {
  if (!podSearchKeyword.value) {
    return pods.value
  }
  const keyword = podSearchKeyword.value.toLowerCase()
  return pods.value.filter(pod =>
    pod.name.toLowerCase().includes(keyword) ||
    pod.namespace.toLowerCase().includes(keyword)
  )
})

// 分页后的 Pod 列表
const paginatedPods = computed(() => {
  const start = (podCurrentPage.value - 1) * podPageSize.value
  const end = start + podPageSize.value
  return filteredPods.value.slice(start, end)
})

// 限制显示最近20条事件
const recentEvents = computed(() => {
  return events.value.slice(0, 20)
})

// 获取节点详情
const loadNodeDetail = async () => {
  loading.value = true
  try {
    const nodes = await getNodes(clusterId)
    const node = nodes.find(n => n.name === nodeName)
    if (node) {
      nodeInfo.value = node
      // 获取节点指标
      await loadNodeMetrics()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取节点信息失败')
  } finally {
    loading.value = false
  }
}

// 获取节点指标
const loadNodeMetrics = async () => {
  try {
    // 调用后端 API 获取节点指标
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/metrics?clusterId=${clusterId}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    const metrics = response.data.data
    cpuUsage.value = metrics.cpuUsage ? (metrics.cpuUsage * 100).toFixed(1) : 0
    memoryUsage.value = metrics.memoryUsage ? (metrics.memoryUsage * 100).toFixed(1) : 0
  } catch (error) {
    console.error('获取指标失败:', error)
  }
}

// 获取 Pod 列表
const loadPods = async () => {
  podsLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/pods?clusterId=${clusterId}&nodeName=${nodeName}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    pods.value = response.data.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Pod 列表失败')
  } finally {
    podsLoading.value = false
  }
}

// 获取事件列表
const loadEvents = async () => {
  eventsLoading.value = true
  try {
    const token = localStorage.getItem('token')
    // 先尝试获取所有事件，然后在前端过滤节点相关的事件
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/events?clusterId=${clusterId}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    // 过滤出与该节点相关的事件
    const allEvents = response.data.data || []
    events.value = allEvents.filter((event: any) =>
      event.involvedObject &&
      event.involvedObject.name === nodeName &&
      event.involvedObject.kind === 'Node'
    )
  } catch (error) {
    console.error(error)
    ElMessage.error('获取事件列表失败')
  } finally {
    eventsLoading.value = false
  }
}

// Pod 分页处理
const handlePodPageChange = (page: number) => {
  podCurrentPage.value = page
  savePodPaginationState()
}

const handlePodPageSizeChange = (size: number) => {
  podPageSize.value = size
  podCurrentPage.value = 1
  savePodPaginationState()
}

// 保存 Pod 分页状态到 localStorage
const savePodPaginationState = () => {
  try {
    localStorage.setItem(podPaginationStorageKey.value, JSON.stringify({
      currentPage: podCurrentPage.value,
      pageSize: podPageSize.value
    }))
  } catch (error) {
    console.error('保存Pod分页状态失败:', error)
  }
}

// 从 localStorage 恢复 Pod 分页状态
const loadPodPaginationState = () => {
  try {
    const saved = localStorage.getItem(podPaginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      podCurrentPage.value = state.currentPage || 1
      podPageSize.value = state.pageSize || 10
    }
  } catch (error) {
    console.error('恢复Pod分页状态失败:', error)
  }
}

// 刷新数据
const refreshData = () => {
  loadNodeDetail()
  loadPods()
  loadEvents()
}

// 返回
const goBack = () => {
  router.back()
}

// 格式化资源
const formatResource = (cpu: string) => {
  if (!cpu) return '-'
  if (cpu.endsWith('m')) {
    const millicores = parseInt(cpu)
    return (millicores / 1000).toFixed(2) + ' 核'
  }
  return cpu + ' 核'
}

// 格式化内存
const formatMemory = (memory: string) => {
  if (!memory) return '-'
  const match = memory.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return memory

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

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
    default:
      bytes = value
  }

  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return Math.ceil(gb) + ' GB'
  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return Math.ceil(mb) + ' MB'
  return memory
}

// 格式化 Provider ID
const formatProviderID = (providerID: string | undefined) => {
  if (!providerID) return '-'
  // Provider ID 通常格式为: provider://zone/instance
  // 简化显示，只取最后部分
  const parts = providerID.split('/')
  if (parts.length > 0) {
    return parts[parts.length - 1]
  }
  return providerID
}

// 获取角色文本
const getRoleText = (role: string) => {
  if (!role) return 'Worker'
  if (role === 'master') return 'Master'
  if (role === 'control-plane') return 'Control Plane'
  if (role === 'worker') return 'Worker'
  return role
}

// 获取状态类型
const getStatusType = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': 'success',
    'Pending': 'warning',
    'Failed': 'danger',
    'Succeeded': 'info',
    'Unknown': 'info'
  }
  return statusMap[status] || 'info'
}

// 获取污点 effect 的样式类
const getTaintEffectClass = (effect: string) => {
  const effectClassMap: Record<string, string> = {
    'NoSchedule': 'effect-no-schedule',
    'PreferNoSchedule': 'effect-prefer-no-schedule',
    'NoExecute': 'effect-no-execute'
  }
  return effectClassMap[effect] || ''
}

onMounted(() => {
  loadPodPaginationState()
  loadNodeDetail()
  loadPods()
  loadEvents()
})
</script>

<style scoped>
.node-detail-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  margin-bottom: 24px;

  .header-content {
    background: #fff;
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    margin-bottom: 20px;
  }

  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;

    .back-btn {
      background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
      color: #D4AF37;
      border: 1px solid rgba(212, 175, 55, 0.3);
      font-weight: 500;
      padding: 12px 24px;
      transition: all 0.3s ease;
    }

    .back-btn:hover {
      background: linear-gradient(135deg, #000000 0%, #2c3e50 100%);
      border-color: #D4AF37;
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
    }
  }

  .node-name-section {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;

    .node-title {
      display: flex;
      align-items: center;
      gap: 12px;
      margin: 0;
      font-size: 28px;
      font-weight: 600;
      color: #303133;

      .title-icon {
        color: #D4AF37;
      }
    }

    .status-tag {
      font-weight: 500;
    }
  }

  .node-meta {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-wrap: wrap;

    .meta-item {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 14px;
      color: #606266;

      .el-icon {
        color: #909399;
      }
    }
  }
}

.header-right {
  display: flex;
  gap: 12px;
}

.black-button {
  background-color: #000000 !important;
  color: #d4af37 !important;
  border-color: #d4af37 !important;
  border-radius: 6px;
  padding: 8px 16px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #bfa13f !important;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.stat-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  border: 1px solid #e0e0e0;
  overflow: hidden;
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(212, 175, 55, 0.2);
  border-color: #d4af37;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-icon-cpu {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-memory {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-pod {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-uptime {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 6px;
}

.stat-value {
  font-size: 26px;
  font-weight: 700;
  color: #d4af37;
  line-height: 1.2;
}

.stat-detail {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.stat-progress {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: #f0f0f0;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #d4af37 0%, #bfa13f 100%);
  transition: width 0.5s ease;
}

/* 信息卡片网格 */
.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.info-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  border: 1px solid #e0e0e0;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 16px 20px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-bottom: 1px solid #d4af37;
}

.card-header .header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-header .header-icon {
  font-size: 20px;
  color: #d4af37;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #d4af37;
}

.card-body {
  padding: 16px 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  font-family: 'Monaco', 'Menlo', monospace;
}

/* 标签、污点和注解 */
.labels-annotations-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.tags-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
}

.tag-item {
  padding: 8px 12px;
  background: #fafafa;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'Monaco', 'Menlo', monospace;
  line-height: 1.6;
  word-break: break-all;
}

.tag-key {
  color: #d4af37;
  font-weight: 600;
  margin-right: 6px;
}

.tag-value {
  color: #606266;
}

.empty-tip {
  padding: 20px;
  text-align: center;
  color: #909399;
  font-size: 14px;
}

/* 污点样式 */
.taint-key {
  color: #d4af37;
  font-weight: 600;
  margin-right: 6px;
}

.taint-separator {
  color: #909399;
  margin: 0 6px;
}

.taint-value {
  color: #606266;
}

.taint-effect {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.taint-effect.effect-no-schedule {
  background: #fef0f0;
  color: #f56c6c;
}

.taint-effect.effect-prefer-no-schedule {
  background: #fdf6ec;
  color: #e6a23c;
}

.taint-effect.effect-no-execute {
  background: #f0f9ff;
  color: #409eff;
}

/* 污点编辑弹窗 */
.taint-edit-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.taint-edit-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.taint-edit-content {
  padding: 8px 0;
}

.taint-list {
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 16px;
}

.taint-edit-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px;
  background: #f8fafc;
  border-radius: 6px;
  transition: all 0.3s;
}

.taint-edit-row:hover {
  background: #f1f5f9;
}

.taint-key-input,
.taint-value-input {
  flex: 1;
  min-width: 120px;
}

.taint-effect-select {
  width: 140px;
  flex-shrink: 0;
}

.taint-separator {
  color: #909399;
  font-weight: 600;
  font-size: 14px;
}

.empty-taints {
  padding: 40px;
  text-align: center;
  color: #909399;
  font-size: 14px;
}

.add-taint-btn {
  width: 100%;
}

/* 区块卡片 */
.section-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  border: 1px solid #e0e0e0;
  margin-bottom: 20px;
  overflow: hidden;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-bottom: 1px solid #d4af37;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-title .title-icon {
  font-size: 20px;
  color: #d4af37;
}

.section-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #d4af37;
}

.search-wrapper {
  display: flex;
  align-items: center;
}

.search-input :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.1);
  border-color: #d4af37;
  border-radius: 6px;
}

.search-input :deep(.el-input__wrapper:hover) {
  border-color: #bfa13f;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 0 0 2px rgba(212, 175, 55, 0.1);
}

.search-input :deep(.el-input__inner) {
  color: #d4af37;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: rgba(212, 175, 55, 0.6);
}

.search-input :deep(.el-input__prefix) {
  color: #d4af37;
}

/* 节点状态信息 */
.conditions-content {
  padding: 0;
}

.conditions-table {
  width: 100%;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.conditions-table :deep(.el-table__header) {
  background: #fafbfc;
}

.conditions-table :deep(.el-table__header th) {
  background: #f5f7fa;
  color: #606266;
  font-weight: 600;
  border-bottom: 1px solid #e0e0e0;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.conditions-table :deep(.el-table__body) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.conditions-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.conditions-table :deep(.el-table__row:hover) {
  background-color: #f5f7fa;
}

.conditions-table :deep(.el-table__row td) {
  border-bottom: 1px solid #f0f0f0;
  padding: 12px 0;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.conditions-table :deep(.el-table__cell) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.table-wrapper {
  padding: 0;
}

.modern-table {
  width: 100%;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.modern-table :deep(.el-table__header) {
  background: #fafbfc;
}

.modern-table :deep(.el-table__header th) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.modern-table :deep(.el-table__body) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.modern-table :deep(.el-table__row td) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.modern-table :deep(.el-table__cell) {
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 13px !important;
}

.pod-name-cell {
  display: flex;
  align-items: center;
}

.pod-name {
  font-weight: 500;
  color: #303133;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

.pagination-wrapper :deep(.el-pagination) {
  display: flex;
  gap: 8px;
}

.pagination-wrapper :deep(.el-pagination__total) {
  color: #606266;
}

.pagination-wrapper :deep(.el-pagination__sizes) {
  color: #606266;
}

.pagination-wrapper :deep(.el-pager li) {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}

.pagination-wrapper :deep(.el-pager li.is-active) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-color: #d4af37;
}

.pagination-wrapper :deep(.el-pager li.is-active .number) {
  color: #d4af37;
}

.pagination-wrapper :deep(.btn-prev),
.pagination-wrapper :deep(.btn-next) {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}

.pagination-wrapper :deep(.btn-prev:hover),
.pagination-wrapper :deep(.btn-next:hover) {
  border-color: #d4af37;
  color: #d4af37;
}

/* 响应式 */
@media (max-width: 1400px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .labels-annotations-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
  .info-grid {
    grid-template-columns: 1fr;
  }
  .labels-annotations-grid {
    grid-template-columns: 1fr;
  }
}
</style>
