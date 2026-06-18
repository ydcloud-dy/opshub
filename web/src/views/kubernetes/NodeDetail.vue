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
          <el-tag v-if="nodeInfo.status === 'Ready'" type="success" effect="light" size="large" class="status-tag">正常</el-tag>
          <el-tag v-else type="danger" effect="light" size="large" class="status-tag">异常</el-tag>
        </div>
        <div class="node-meta">
          <span class="meta-item">
            <el-icon><Platform /></el-icon>
            {{ clusterName || '所属集群' }}
          </span>
          <span class="meta-item">
            <el-icon><Connection /></el-icon>
            {{ nodeInfo.internalIP || '-' }}
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
          <div class="stat-detail">{{ formatResource(nodeInfo.cpuCapacity || '') }}</div>
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
          <div class="stat-detail">{{ formatMemory(nodeInfo.memoryCapacity || '') }}</div>
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

    <!-- 节点监控 -->
    <div class="node-monitor-section">
      <div class="monitor-hero-card">
        <div class="monitor-hero-left">
          <div class="monitor-hero-icon">
            <el-icon><DataAnalysis /></el-icon>
          </div>
          <div>
            <div class="monitor-hero-title">节点监控</div>
            <div class="monitor-hero-desc">
              {{ nodeMetrics?.metricsAvailable ? `最后采集：${nodeMetrics.collectedAt}` : (nodeMetrics?.metricsMessage || '正在等待节点指标数据') }}
            </div>
          </div>
        </div>
        <div class="monitor-health-score">
          <span class="score-label">健康分</span>
          <strong>{{ nodeMetrics?.healthScore ?? '--' }}</strong>
        </div>
      </div>

      <el-alert
        v-if="nodeMetrics && !nodeMetrics.metricsAvailable"
        class="metrics-alert"
        type="warning"
        show-icon
        :closable="false"
        title="Metrics Server 未返回实时资源指标"
        :description="nodeMetrics.metricsMessage || '当前集群暂时无法获取 CPU/内存实时数据，可先查看节点状态、Pod 分布和事件信息。'"
      />

      <div class="monitor-summary-grid">
        <div class="monitor-summary-card cpu">
          <div class="summary-card-top">
            <span>CPU 使用</span>
            <strong>{{ percentText(nodeMetrics?.cpuUsage) }}</strong>
          </div>
          <el-progress :percentage="ratioToPercent(nodeMetrics?.cpuUsage)" :stroke-width="9" :show-text="false" />
          <div class="summary-card-foot">{{ formatMilliCPU(nodeMetrics?.cpuUsed || 0) }} / {{ formatMilliCPU(nodeMetrics?.cpuAllocatable || 0) }}</div>
        </div>
        <div class="monitor-summary-card memory">
          <div class="summary-card-top">
            <span>内存使用</span>
            <strong>{{ percentText(nodeMetrics?.memoryUsage) }}</strong>
          </div>
          <el-progress :percentage="ratioToPercent(nodeMetrics?.memoryUsage)" :stroke-width="9" :show-text="false" />
          <div class="summary-card-foot">{{ formatBytes(nodeMetrics?.memoryUsed || 0) }} / {{ formatBytes(nodeMetrics?.memoryAllocatable || 0) }}</div>
        </div>
        <div class="monitor-summary-card pods">
          <div class="summary-card-top">
            <span>Pod 使用</span>
            <strong>{{ nodeMetrics?.podCount ?? 0 }}/{{ nodeMetrics?.podCapacity ?? 0 }}</strong>
          </div>
          <el-progress :percentage="ratioToPercent(nodeMetrics?.podUsage)" :stroke-width="9" :show-text="false" />
          <div class="summary-card-foot">运行 {{ nodeMetrics?.podRunning ?? 0 }}，异常 {{ (nodeMetrics?.podPending || 0) + (nodeMetrics?.podFailed || 0) }}</div>
        </div>
        <div class="monitor-summary-card restarts">
          <div class="summary-card-top">
            <span>重启次数</span>
            <strong>{{ nodeMetrics?.totalRestarts ?? 0 }}</strong>
          </div>
          <div class="condition-pills">
            <span :class="['condition-pill', nodeMetrics?.conditionSummary?.ready ? 'ok' : 'bad']">Ready</span>
            <span :class="['condition-pill', nodeMetrics?.conditionSummary?.diskPressure ? 'bad' : 'ok']">Disk</span>
            <span :class="['condition-pill', nodeMetrics?.conditionSummary?.memoryPressure ? 'bad' : 'ok']">Memory</span>
          </div>
        </div>
      </div>

      <div class="monitor-chart-grid">
        <div class="monitor-card">
          <div class="monitor-card-header">
            <span>资源趋势</span>
            <small>最近 {{ monitorHistory.length }} 次刷新采样</small>
          </div>
          <div ref="resourceChartRef" class="monitor-chart"></div>
        </div>
        <div class="monitor-card">
          <div class="monitor-card-header">
            <span>Pod 分布</span>
            <small>当前节点工作负载状态</small>
          </div>
          <div ref="podChartRef" class="monitor-chart"></div>
        </div>
      </div>

      <div class="monitor-detail-grid">
        <div class="monitor-card">
          <div class="monitor-card-header">
            <span>Top Pods</span>
            <small>按 CPU 使用排序</small>
          </div>
          <el-table :data="nodeMetrics?.topPods || []" class="monitor-table" max-height="300">
            <el-table-column prop="name" label="Pod" min-width="180" show-overflow-tooltip />
            <el-table-column prop="namespace" label="命名空间" width="120" />
            <el-table-column prop="cpuUsedText" label="CPU" width="90" />
            <el-table-column prop="memoryText" label="内存" width="100" />
            <el-table-column prop="restarts" label="重启" width="70" align="center" />
          </el-table>
        </div>
        <div class="monitor-card">
          <div class="monitor-card-header">
            <span>最近事件</span>
            <small>节点相关事件</small>
          </div>
          <div class="monitor-event-list">
            <div v-for="(event, index) in nodeMetrics?.events || []" :key="index" class="monitor-event-item">
              <el-tag :type="event.type === 'Warning' ? 'warning' : 'success'" size="small">{{ event.type }}</el-tag>
              <div class="event-main">
                <div class="event-title">{{ event.reason || '-' }}</div>
                <div class="event-message">{{ event.message }}</div>
              </div>
              <span class="event-time">{{ event.lastTimestamp || '-' }}</span>
            </div>
            <div v-if="!nodeMetrics?.events?.length" class="monitor-empty">暂无节点事件</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 信息卡片 -->
    <div class="info-grid">
      <!-- 基本信息 -->
      <div class="info-card">
        <div class="card-header">
          <div class="header-left">
            <el-icon class="header-icon"><InfoFilled /></el-icon>
            <h3>基本信息</h3>
          </div>
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
            <span class="info-value">{{ getRoleText(nodeInfo.roles || '') }}</span>
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
          <div class="header-left">
            <el-icon class="header-icon"><Monitor /></el-icon>
            <h3>系统信息</h3>
          </div>
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
          <div class="header-left">
            <el-icon class="header-icon"><Connection /></el-icon>
            <h3>网络信息</h3>
          </div>
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
          <div class="header-left">
            <el-icon class="header-icon"><Document /></el-icon>
            <h3>注解 ({{ Object.keys(nodeInfo.annotations || {}).length }})</h3>
          </div>
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
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts'
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
  WarnTriangleFilled,
  DataAnalysis
} from '@element-plus/icons-vue'
import { getNodeMetrics, getNodes, type NodeInfo, type NodeMetricsInfo } from '@/api/kubernetes'
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
const nodeMetrics = ref<NodeMetricsInfo | null>(null)
const monitorHistory = ref<{ time: string; cpu: number; memory: number; pods: number }[]>([])
const resourceChartRef = ref<HTMLElement>()
const podChartRef = ref<HTMLElement>()

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
    ElMessage.error('获取节点信息失败')
  } finally {
    loading.value = false
  }
}

// 获取节点指标
const loadNodeMetrics = async () => {
  try {
    const metrics = await getNodeMetrics(clusterId, nodeName)
    nodeMetrics.value = metrics
    cpuUsage.value = ratioToPercent(metrics.cpuUsage)
    memoryUsage.value = ratioToPercent(metrics.memoryUsage)
    appendMonitorHistory(metrics)
    await nextTick()
    renderMonitorCharts()
  } catch (error: any) {
    nodeMetrics.value = {
      collectedAt: '',
      metricsAvailable: false,
      metricsMessage: error?.message || '节点监控数据暂不可用',
      healthScore: 0,
      cpuUsage: 0,
      memoryUsage: 0,
      podUsage: 0,
      cpuUsed: 0,
      memoryUsed: 0,
      cpuCapacity: 0,
      cpuAllocatable: 0,
      memoryCapacity: 0,
      memoryAllocatable: 0,
      podCount: nodeInfo.value.podCount || 0,
      podCapacity: nodeInfo.value.podCapacity || 0,
      podRunning: 0,
      podPending: 0,
      podFailed: 0,
      podSucceeded: 0,
      totalRestarts: 0,
      conditions: nodeInfo.value.conditions || [],
      conditionSummary: {
        ready: nodeInfo.value.status === 'Ready',
        memoryPressure: false,
        diskPressure: false,
        pidPressure: false,
        networkUnavailable: false
      },
      topPods: [],
      events: [],
      addresses: {},
      podCIDR: nodeInfo.value.podCIDR || '',
      podCIDRs: [],
      providerID: nodeInfo.value.providerID || '',
      unschedulable: !nodeInfo.value.schedulable
    }
    await nextTick()
    renderMonitorCharts()
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
  }
}

// 刷新数据
const refreshData = () => {
  loadNodeDetail()
  loadPods()
  loadEvents()
}

const ratioToPercent = (value?: number) => {
  if (!Number.isFinite(value || 0)) return 0
  return Math.max(0, Math.min(100, Number(((value || 0) * 100).toFixed(1))))
}

const percentText = (value?: number) => `${ratioToPercent(value)}%`

const formatMilliCPU = (value: number) => {
  if (!value) return '0m'
  if (value >= 1000) return `${(value / 1000).toFixed(value % 1000 === 0 ? 0 : 1)} 核`
  return `${value}m`
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }
  return `${value.toFixed(index <= 1 ? 0 : 1)} ${units[index]}`
}

const appendMonitorHistory = (metrics: NodeMetricsInfo) => {
  const time = metrics.collectedAt ? metrics.collectedAt.slice(11, 19) : new Date().toLocaleTimeString()
  monitorHistory.value.push({
    time,
    cpu: ratioToPercent(metrics.cpuUsage),
    memory: ratioToPercent(metrics.memoryUsage),
    pods: ratioToPercent(metrics.podUsage)
  })
  if (monitorHistory.value.length > 20) {
    monitorHistory.value = monitorHistory.value.slice(-20)
  }
}

const getChart = (target?: HTMLElement) => {
  if (!target) return null
  return echarts.getInstanceByDom(target) || echarts.init(target)
}

const renderMonitorCharts = () => {
  const history = monitorHistory.value
  const resourceChart = getChart(resourceChartRef.value)
  if (resourceChart) {
    resourceChart.setOption({
      color: ['#2563eb', '#16a34a', '#f97316'],
      tooltip: { trigger: 'axis', valueFormatter: (value: number) => `${value}%` },
      legend: { top: 0, right: 8, textStyle: { color: '#667085' } },
      grid: { left: 36, right: 18, top: 42, bottom: 28 },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: history.map(item => item.time),
        axisLine: { lineStyle: { color: '#e5e9f2' } },
        axisLabel: { color: '#98a2b3' }
      },
      yAxis: {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { color: '#98a2b3', formatter: '{value}%' },
        splitLine: { lineStyle: { color: '#eef2f7' } }
      },
      series: [
        { name: 'CPU', type: 'line', smooth: true, areaStyle: { opacity: 0.08 }, data: history.map(item => item.cpu) },
        { name: '内存', type: 'line', smooth: true, areaStyle: { opacity: 0.08 }, data: history.map(item => item.memory) },
        { name: 'Pod', type: 'line', smooth: true, areaStyle: { opacity: 0.06 }, data: history.map(item => item.pods) }
      ]
    })
  }

  const podChart = getChart(podChartRef.value)
  const metrics = nodeMetrics.value
  if (podChart) {
    podChart.setOption({
      color: ['#16a34a', '#f59e0b', '#ef4444', '#94a3b8'],
      tooltip: { trigger: 'item' },
      legend: { bottom: 0, textStyle: { color: '#667085' } },
      series: [
        {
          name: 'Pod 状态',
          type: 'pie',
          radius: ['48%', '70%'],
          center: ['50%', '44%'],
          avoidLabelOverlap: true,
          label: { formatter: '{b}\n{c}', color: '#344054' },
          data: [
            { name: 'Running', value: metrics?.podRunning || 0 },
            { name: 'Pending', value: metrics?.podPending || 0 },
            { name: 'Failed', value: metrics?.podFailed || 0 },
            { name: 'Succeeded', value: metrics?.podSucceeded || 0 }
          ].filter(item => item.value > 0)
        }
      ]
    })
  }
}

const resizeMonitorCharts = () => {
  if (resourceChartRef.value) echarts.getInstanceByDom(resourceChartRef.value)?.resize()
  if (podChartRef.value) echarts.getInstanceByDom(podChartRef.value)?.resize()
}

const disposeMonitorCharts = () => {
  if (resourceChartRef.value) echarts.getInstanceByDom(resourceChartRef.value)?.dispose()
  if (podChartRef.value) echarts.getInstanceByDom(podChartRef.value)?.dispose()
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

  const value = parseFloat(match[1] || '0')
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
  window.addEventListener('resize', resizeMonitorCharts)
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeMonitorCharts)
  disposeMonitorCharts()
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
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
}

.card-header .header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-header .header-icon {
  font-size: 20px;
  color: #606266;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
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
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-title .title-icon {
  font-size: 20px;
  color: #606266;
}

.section-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.search-wrapper {
  display: flex;
  align-items: center;
}

.search-input :deep(.el-input__wrapper) {
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  border-radius: 6px;
}

.search-input :deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
}

.search-input :deep(.el-input__inner) {
  color: #303133;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: #a8abb2;
}

.search-input :deep(.el-input__prefix) {
  color: #909399;
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
  background: #409eff;
  border-color: #409eff;
}

.pagination-wrapper :deep(.el-pager li.is-active .number) {
  color: #fff;
}

.pagination-wrapper :deep(.btn-prev),
.pagination-wrapper :deep(.btn-next) {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}

.pagination-wrapper :deep(.btn-prev:hover),
.pagination-wrapper :deep(.btn-next:hover) {
  border-color: #409eff;
  color: #409eff;
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

/* 节点详情与集群详情统一：单层卡片铺满，内容收在左侧 */
.node-detail-container {
  --detail-primary: #2563eb;
  --detail-primary-soft: #eff6ff;
  --detail-success: #16a34a;
  --detail-warning: #f97316;
  --detail-danger: #dc2626;
  --detail-ink: #303133;
  --detail-muted: #909399;
  --detail-border: #e5e9f2;
  min-height: 100vh;
  padding: 0;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.page-header .header-content {
  position: relative;
  overflow: hidden;
  display: block;
  width: 100%;
  min-height: 170px;
  box-sizing: border-box;
  padding: 22px 48px;
  border: 1px solid #dfe6f1;
  border-radius: 14px;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.98), rgba(255, 255, 255, 0.9)),
    radial-gradient(circle at 84% 16%, rgba(37, 99, 235, 0.14), transparent 32%);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.page-header .header-content::after {
  content: '';
  position: absolute;
  right: 5%;
  top: -58px;
  width: 190px;
  height: 190px;
  border-radius: 50%;
  background: rgba(37, 99, 235, 0.08);
  pointer-events: none;
}

.page-header .header-top,
.page-header .node-name-section,
.page-header .node-meta {
  position: relative;
  z-index: 1;
  max-width: 560px;
}

.page-header .header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: none;
  margin-bottom: 14px;
}

.page-header .header-top .back-btn,
.page-header .header-top .black-button {
  min-height: 40px;
  padding: 10px 18px;
  border: 1px solid #d7deea !important;
  border-radius: 10px;
  background: #ffffff !important;
  color: #344054 !important;
  font-weight: 600;
  letter-spacing: 0.2px;
  box-shadow: 0 6px 14px rgba(15, 23, 42, 0.06);
}

.page-header .header-top .back-btn:hover,
.page-header .header-top .black-button:hover {
  border-color: #bfdbfe !important;
  background: var(--detail-primary-soft) !important;
  color: var(--detail-primary) !important;
  transform: translateY(-2px);
  box-shadow: 0 10px 24px rgba(37, 99, 235, 0.12);
}

.page-header .node-name-section {
  gap: 18px;
  margin-bottom: 12px;
}

.page-header .node-name-section .node-title {
  color: var(--detail-ink);
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.page-header .node-name-section .node-title .title-icon {
  width: auto;
  height: auto;
  border-radius: 0;
  background: transparent;
  color: var(--detail-primary);
  box-shadow: none;
}

.page-header .node-name-section .status-tag {
  padding: 8px 18px;
  border-radius: 20px;
  font-weight: 600;
}

.page-header .node-meta {
  gap: 16px;
}

.page-header .node-meta .meta-item {
  gap: 8px;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  color: #606266;
}

.page-header .node-meta .meta-item .el-icon {
  color: var(--detail-primary);
}

.stats-grid {
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  position: relative;
  gap: 20px;
  padding: 24px;
  border: none;
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.stat-card::before {
  content: '';
  position: absolute;
  inset: 0 0 auto 0;
  height: 4px;
  background: linear-gradient(90deg, var(--detail-primary), #60a5fa);
}

.stat-card:hover {
  border-color: transparent;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.1);
  transform: translateY(-4px);
}

.stat-icon,
.stat-icon-cpu,
.stat-icon-memory,
.stat-icon-pod,
.stat-icon-uptime {
  width: 64px;
  height: 64px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 12px;
  background: var(--detail-primary-soft);
  color: var(--detail-primary);
}

.stat-icon-memory {
  background: #ecfdf3;
  color: var(--detail-success);
}

.stat-icon-pod {
  background: #fff7ed;
  color: var(--detail-warning);
}

.stat-icon-uptime {
  background: #eef2ff;
  color: #4f46e5;
}

.stat-label {
  color: var(--detail-muted);
  font-size: 14px;
}

.stat-value {
  color: var(--detail-ink);
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.stat-detail {
  color: var(--detail-muted);
  font-size: 13px;
}

.progress-bar {
  background: linear-gradient(90deg, var(--detail-primary) 0%, #60a5fa 100%);
}

.info-card,
.section-card {
  border: none;
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.card-header,
.section-header {
  padding: 20px 24px;
  background: #ffffff;
  border-bottom: 1px solid #f0f0f0;
}

.card-header .header-icon,
.section-title .title-icon {
  color: var(--detail-primary);
}

.card-header h3,
.section-title h3 {
  color: var(--detail-ink);
  font-weight: 700;
}

.tag-item {
  border: 1px solid #e5e9f2;
  border-left: 3px solid var(--detail-primary);
  border-radius: 10px;
  background: #f8fbff;
}

.tag-key,
.taint-key {
  color: var(--detail-primary);
}

.modern-table :deep(.el-table__header th),
.conditions-table :deep(.el-table__header th) {
  background: #f5f7fa !important;
  color: #606266;
  border-bottom: 1px solid #e0e0e0;
}

.modern-table :deep(.el-table__row:hover),
.conditions-table :deep(.el-table__row:hover) {
  background-color: #f8fbff !important;
}

.node-monitor-section {
  margin-bottom: 20px;
}

.monitor-hero-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  padding: 20px 24px;
  margin-bottom: 14px;
  border: 1px solid #dfe6f1;
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(255, 255, 255, 0.94)),
    #ffffff;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.monitor-hero-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.monitor-hero-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  color: var(--detail-primary);
  background: #eff6ff;
  border: 1px solid #dbeafe;
  font-size: 24px;
}

.monitor-hero-title {
  font-size: 18px;
  font-weight: 800;
  color: #111827;
}

.monitor-hero-desc {
  margin-top: 4px;
  font-size: 13px;
  color: #667085;
}

.monitor-health-score {
  min-width: 118px;
  padding: 10px 16px;
  border-radius: 14px;
  background: #ffffff;
  border: 1px solid #e5e9f2;
  text-align: center;
}

.score-label {
  display: block;
  color: #667085;
  font-size: 12px;
}

.monitor-health-score strong {
  display: block;
  margin-top: 2px;
  color: var(--detail-primary);
  font-size: 28px;
  line-height: 1.1;
}

.metrics-alert {
  margin-bottom: 14px;
  border-radius: 12px;
}

.monitor-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.monitor-summary-card {
  padding: 16px;
  border: 1px solid #e5e9f2;
  border-radius: 14px;
  background: #ffffff;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.04);
}

.summary-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.summary-card-top span {
  color: #667085;
  font-size: 13px;
  font-weight: 600;
}

.summary-card-top strong {
  color: #111827;
  font-size: 22px;
  line-height: 1;
}

.summary-card-foot {
  margin-top: 10px;
  color: #98a2b3;
  font-size: 12px;
}

.condition-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.condition-pill {
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.condition-pill.ok {
  color: #15803d;
  background: #ecfdf3;
}

.condition-pill.bad {
  color: #b42318;
  background: #fef3f2;
}

.monitor-chart-grid,
.monitor-detail-grid {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 16px;
  margin-bottom: 16px;
}

.monitor-detail-grid {
  grid-template-columns: 1fr 1fr;
}

.monitor-card {
  min-width: 0;
  border: 1px solid #e5e9f2;
  border-radius: 16px;
  background: #ffffff;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.monitor-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-bottom: 1px solid #eef2f7;
}

.monitor-card-header span {
  color: #111827;
  font-size: 15px;
  font-weight: 800;
}

.monitor-card-header small {
  color: #98a2b3;
  font-size: 12px;
}

.monitor-chart {
  height: 300px;
}

.monitor-table {
  width: 100%;
}

.monitor-event-list {
  max-height: 300px;
  overflow-y: auto;
}

.monitor-event-item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  align-items: flex-start;
  padding: 12px 16px;
  border-bottom: 1px solid #f2f4f7;
}

.event-main {
  min-width: 0;
}

.event-title {
  color: #111827;
  font-weight: 700;
  font-size: 13px;
}

.event-message {
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 1.5;
  word-break: break-all;
}

.event-time {
  color: #98a2b3;
  font-size: 12px;
  white-space: nowrap;
}

.monitor-empty {
  padding: 42px 20px;
  color: #98a2b3;
  text-align: center;
  font-size: 13px;
}

@media (max-width: 1400px) {
  .page-header .header-content {
    display: block;
  }

  .monitor-summary-grid,
  .monitor-chart-grid,
  .monitor-detail-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .page-header .header-content {
    padding: 22px;
    min-height: 0;
  }

  .monitor-hero-card,
  .monitor-hero-left,
  .monitor-card-header,
  .monitor-event-item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .monitor-summary-grid,
  .monitor-chart-grid,
  .monitor-detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
