<template>
  <div class="cluster-detail-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-top">
          <el-button class="back-btn" @click="handleBack" :icon="ArrowLeft">返回列表</el-button>
        </div>
        <div class="cluster-name-section">
          <h1 class="cluster-title">
            <el-icon class="title-icon" :size="28"><Platform /></el-icon>
            {{ clusterInfo?.alias || clusterInfo?.name }}
          </h1>
          <el-tag :type="getStatusType(clusterInfo?.status || 1)" size="large" class="status-tag">
            {{ getStatusText(clusterInfo?.status || 1) }}
          </el-tag>
        </div>
        <div class="cluster-meta">
          <span class="meta-item">
            <el-icon><Connection /></el-icon>
            {{ clusterInfo?.apiEndpoint }}
          </span>
          <span class="meta-item">
            <el-icon><InfoFilled /></el-icon>
            {{ clusterInfo?.version }}
          </span>
          <span class="meta-item" v-if="clusterInfo?.provider">
            <el-icon><Shop /></el-icon>
            {{ getProviderText(clusterInfo.provider) }}
          </span>
        </div>
      </div>
    </div>

    <!-- 快速统计卡片 -->
    <div class="quick-stats">
      <div class="stat-card" v-for="(stat, index) in quickStats" :key="index" :style="{ '--delay': index * 0.1 + 's' }">
        <div class="stat-icon-wrapper" :style="{ background: stat.color }">
          <el-icon :size="32" :color="stat.iconColor">
            <component :is="stat.icon" />
          </el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
        </div>
        <div class="stat-trend" v-if="stat.trend">
          <el-icon :size="16"><TrendCharts /></el-icon>
        </div>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <!-- 左侧列 -->
      <div class="left-column">
        <!-- 资源使用率 -->
        <el-card shadow="hover" class="modern-card">
          <template #header>
            <div class="card-title-section">
              <el-icon class="card-icon" :size="20" color="#D4AF37"><DataAnalysis /></el-icon>
              <span class="card-title">资源使用率</span>
            </div>
          </template>
          <div class="resource-usage">
            <div class="usage-item">
              <div class="usage-header">
                <div class="usage-label">
                  <el-icon color="#D4AF37" :size="18"><Cpu /></el-icon>
                  <span>CPU 使用率</span>
                </div>
                <span class="usage-value">{{ Math.round(clusterStats.cpuUsage) }}%</span>
              </div>
              <div class="progress-wrapper">
                <el-progress
                  :percentage="Math.round(clusterStats.cpuUsage)"
                  :color="getProgressColor(clusterStats.cpuUsage)"
                  :show-text="false"
                  :stroke-width="8"
                />
              </div>
              <div class="usage-detail">
                <span class="detail-text">已使用: {{ clusterStats.cpuUsed.toFixed(2) }} 核</span>
                <span class="detail-text">可分配: {{ clusterStats.cpuAllocatable.toFixed(2) }} 核</span>
              </div>
            </div>

            <div class="usage-item">
              <div class="usage-header">
                <div class="usage-label">
                  <el-icon color="#D4AF37" :size="18"><Coin /></el-icon>
                  <span>内存使用率</span>
                </div>
                <span class="usage-value">{{ Math.round(clusterStats.memoryUsage) }}%</span>
              </div>
              <div class="progress-wrapper">
                <el-progress
                  :percentage="Math.round(clusterStats.memoryUsage)"
                  :color="getProgressColor(clusterStats.memoryUsage)"
                  :show-text="false"
                  :stroke-width="8"
                />
              </div>
              <div class="usage-detail">
                <span class="detail-text">已使用: {{ formatBytes(clusterStats.memoryUsed) }}</span>
                <span class="detail-text">可分配: {{ formatBytes(clusterStats.memoryAllocatable) }}</span>
              </div>
            </div>
          </div>
        </el-card>

        <!-- 网络配置 -->
        <el-card shadow="hover" class="modern-card">
          <template #header>
            <div class="card-title-section">
              <el-icon class="card-icon" :size="20" color="#D4AF37"><Connection /></el-icon>
              <span class="card-title">网络配置</span>
            </div>
          </template>
          <div class="network-config">
            <div class="config-grid">
              <div class="config-item">
                <div class="config-label">API Server</div>
                <el-tag type="primary" size="large">{{ networkInfo.apiServerAddress || '-' }}</el-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Service CIDR</div>
                <el-tag type="info" size="large">{{ networkInfo.serviceCIDR || '-' }}</el-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Pod CIDR</div>
                <el-tag type="success" size="large">{{ networkInfo.podCIDR || '-' }}</el-tag>
              </div>
              <div class="config-item">
                <div class="config-label">网络插件</div>
                <el-tag type="warning" size="large">{{ networkInfo.networkPlugin || '-' }}</el-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Proxy 模式</div>
                <el-tag type="primary" size="large">{{ networkInfo.proxyMode || '-' }}</el-tag>
              </div>
              <div class="config-item">
                <div class="config-label">DNS 服务</div>
                <el-tag type="success" size="large">{{ networkInfo.dnsService || '-' }}</el-tag>
              </div>
            </div>
          </div>
        </el-card>

        <!-- 集群信息 -->
        <el-card shadow="hover" class="modern-card">
          <template #header>
            <div class="card-title-section">
              <el-icon class="card-icon" :size="20" color="#D4AF37"><InfoFilled /></el-icon>
              <span class="card-title">集群信息</span>
            </div>
          </template>
          <div class="cluster-info-grid">
            <div class="info-row">
              <span class="info-label">集群名称</span>
              <span class="info-value">{{ clusterInfo?.name }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">别名</span>
              <span class="info-value">{{ clusterInfo?.alias || '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">服务商</span>
              <span class="info-value">{{ clusterInfo?.provider ? getProviderText(clusterInfo.provider) : '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">区域</span>
              <span class="info-value">{{ clusterInfo?.region || '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">创建时间</span>
              <span class="info-value">{{ clusterInfo?.createdAt }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">更新时间</span>
              <span class="info-value">{{ clusterInfo?.updatedAt }}</span>
            </div>
            <div class="info-row full-width">
              <span class="info-label">备注</span>
              <span class="info-value">{{ clusterInfo?.description || '-' }}</span>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 右侧列 -->
      <div class="right-column">
        <!-- 组件信息 -->
        <el-card shadow="hover" class="modern-card">
          <template #header>
            <div class="card-title-section">
              <el-icon class="card-icon" :size="20" color="#D4AF37"><Files /></el-icon>
              <span class="card-title">组件信息</span>
            </div>
          </template>

          <!-- 运行时环境 -->
          <div class="component-section">
            <div class="section-header">
              <el-icon :size="16"><Monitor /></el-icon>
              <span>运行时环境</span>
            </div>
            <div class="runtime-cards">
              <div class="runtime-card">
                <div class="runtime-label">容器运行时</div>
                <div class="runtime-value">{{ componentInfo.runtime.containerRuntime || '-' }}</div>
              </div>
              <div class="runtime-card">
                <div class="runtime-label">Kubelet 版本</div>
                <div class="runtime-value">{{ componentInfo.runtime.version || '-' }}</div>
              </div>
            </div>
          </div>

          <!-- 控制平面组件 -->
          <div class="component-section" v-if="componentInfo.components.length > 0">
            <div class="section-header">
              <el-icon :size="16"><Setting /></el-icon>
              <span>控制平面组件</span>
            </div>
            <div class="component-list">
              <div
                v-for="component in componentInfo.components"
                :key="component.name"
                class="component-item"
              >
                <div class="component-main">
                  <el-icon class="component-icon" :size="20" color="#D4AF37"><CircleCheck /></el-icon>
                  <div class="component-info">
                    <div class="component-name">{{ component.name }}</div>
                    <el-tag size="small" type="info">{{ component.version }}</el-tag>
                  </div>
                </div>
                <el-tag
                  :type="component.status === 'Running' ? 'success' : 'danger'"
                  size="small"
                >
                  {{ component.status }}
                </el-tag>
              </div>
            </div>
          </div>

          <!-- 存储类 -->
          <div class="component-section" v-if="componentInfo.storage.length > 0">
            <div class="section-header">
              <el-icon :size="16"><Folder /></el-icon>
              <span>存储类</span>
            </div>
            <div class="storage-list">
              <div
                v-for="storage in componentInfo.storage"
                :key="storage.name"
                class="storage-item"
              >
                <div class="storage-main">
                  <el-icon class="storage-icon" :size="18" color="#D4AF37"><Folder /></el-icon>
                  <div class="storage-info">
                    <div class="storage-name">{{ storage.name }}</div>
                    <div class="storage-provisioner">{{ storage.provisioner }}</div>
                  </div>
                </div>
                <el-tag
                  :type="storage.reclaimPolicy === 'Delete' ? 'danger' : 'warning'"
                  size="small"
                >
                  {{ storage.reclaimPolicy }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- 节点信息 -->
    <el-card shadow="hover" class="modern-card full-width-card">
      <template #header>
        <div class="card-title-section">
          <el-icon class="card-icon" :size="20" color="#D4AF37"><Monitor /></el-icon>
          <span class="card-title">节点信息</span>
          <span class="node-count">{{ nodeList.length }}个节点</span>
        </div>
      </template>
      <el-table
        :data="nodeList"
        stripe
        style="width: 100%"
        v-loading="nodesLoading"
      >
        <el-table-column prop="name" label="节点名称" min-width="150" />
        <el-table-column label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="getNodeRoleType(row.roles)" size="small">
              {{ row.roles || 'Worker' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Ready' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="internalIP" label="内部IP" width="150" />
        <el-table-column prop="externalIP" label="外部IP" width="150">
          <template #default="{ row }">
            {{ row.externalIP || '无' }}
          </template>
        </el-table-column>
        <el-table-column prop="version" label="K8s版本" width="120" />
        <el-table-column prop="osImage" label="操作系统" min-width="180" show-overflow-tooltip />
        <el-table-column prop="age" label="创建时间" width="180" />
      </el-table>
    </el-card>

    <!-- 最近事件 -->
    <el-card shadow="hover" class="modern-card full-width-card">
      <template #header>
        <div class="card-title-section">
          <el-icon class="card-icon" :size="20" color="#D4AF37"><Document /></el-icon>
          <span class="card-title">最近事件</span>
          <span class="event-count">最近50条</span>
        </div>
      </template>
      <el-table
        :data="eventList"
        stripe
        style="width: 100%"
        v-loading="eventsLoading"
        :empty-text="'No Data'"
      >
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.type === 'Normal' ? 'success' : 'warning'" size="small">
              {{ row.type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" width="150" show-overflow-tooltip />
        <el-table-column prop="message" label="消息" min-width="300" show-overflow-tooltip />
        <el-table-column prop="source" label="来源" width="200" show-overflow-tooltip />
        <el-table-column prop="count" label="次数" width="80" align="center" />
        <el-table-column prop="lastTimestamp" label="最后发生时间" width="180" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowLeft,
  Platform,
  Connection,
  InfoFilled,
  Monitor,
  Files,
  Box,
  Cpu,
  TrendCharts,
  Shop,
  DataAnalysis,
  Coin,
  Setting,
  Folder,
  CircleCheck,
  Refresh,
  Document
} from '@element-plus/icons-vue'
import {
  getClusterDetail,
  getClusterStats,
  getClusterNetworkInfo,
  getClusterComponentInfo,
  getNodes,
  getClusterEvents,
  type Cluster,
  type ClusterStats,
  type ClusterNetworkInfo,
  type ClusterComponentInfo,
  type NodeInfo,
  type EventInfo
} from '@/api/kubernetes'

const route = useRoute()
const router = useRouter()

const clusterId = ref<number>(parseInt(route.params.id as string))
const clusterInfo = ref<Cluster>()
let errorMessageShown = false // 跟踪是否已显示错误消息
const clusterStats = ref<ClusterStats>({
  nodeCount: 0,
  workloadCount: 0,
  podCount: 0,
  cpuUsage: 0,
  memoryUsage: 0,
  cpuCapacity: 0,
  memoryCapacity: 0,
  cpuAllocatable: 0,
  memoryAllocatable: 0,
  cpuUsed: 0,
  memoryUsed: 0
})

const networkInfo = ref<ClusterNetworkInfo>({
  serviceCIDR: '',
  podCIDR: '',
  apiServerAddress: '',
  networkPlugin: '',
  proxyMode: '',
  dnsService: ''
})

const componentInfo = ref<ClusterComponentInfo>({
  components: [],
  runtime: {
    containerRuntime: '',
    version: ''
  },
  storage: []
})

const nodeList = ref<NodeInfo[]>([])
const eventList = ref<EventInfo[]>([])
const nodesLoading = ref(false)
const eventsLoading = ref(false)

// 快速统计卡片数据
const quickStats = computed(() => [
  {
    label: '节点数量',
    value: clusterStats.value.nodeCount,
    icon: Monitor,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: '工作负载',
    value: clusterStats.value.workloadCount,
    icon: Box,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: 'Pod 总数',
    value: clusterStats.value.podCount,
    icon: Files,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: 'CPU 使用率',
    value: Math.round(clusterStats.value.cpuUsage) + '%',
    icon: Cpu,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: false
  }
])

// 加载集群详情
const loadClusterDetail = async () => {
  try {
    const data = await getClusterDetail(clusterId.value)
    clusterInfo.value = data
    // 并行加载所有数据
    await Promise.all([
      loadClusterStats(),
      loadNetworkInfo(),
      loadComponentInfo(),
      loadNodes(),
      loadEvents()
    ])
  } catch (error: any) {
    // 只显示一次错误消息
    if (!errorMessageShown) {
      errorMessageShown = true
      if (error.response?.status === 403 || error.response?.status === 401) {
        ElMessage.error({
          message: '您没有权限访问该集群，请联系管理员授权',
          duration: 5000,
          showClose: true
        })
      } else {
        ElMessage.error(error.response?.data?.message || '获取集群信息失败')
      }
    }
  }
}

// 加载集群统计信息
const loadClusterStats = async () => {
  try {
    const data = await getClusterStats(clusterId.value)
    clusterStats.value = data
  } catch (error: any) {
    console.error('加载统计信息失败:', error)
    throw error // 抛出错误，让 Promise.all 捕获
  }
}

// 加载网络信息
const loadNetworkInfo = async () => {
  try {
    const data = await getClusterNetworkInfo(clusterId.value)
    networkInfo.value = data
  } catch (error: any) {
    console.error('加载网络信息失败:', error)
    throw error
  }
}

// 加载组件信息
const loadComponentInfo = async () => {
  try {
    const data = await getClusterComponentInfo(clusterId.value)
    console.log('[ClusterDetail] 组件信息响应:', data)
    console.log('[ClusterDetail] components 数量:', data?.components?.length || 0)
    console.log('[ClusterDetail] components 详情:', data?.components)

    // 手动触发响应式更新
    componentInfo.value = {
      components: data?.components || [],
      runtime: data?.runtime || { containerRuntime: '', version: '' },
      storage: data?.storage || []
    }

    console.log('[ClusterDetail] componentInfo.value:', componentInfo.value)
    console.log('[ClusterDetail] componentInfo.value.components.length:', componentInfo.value.components.length)

    // 强制触发重新渲染
    await nextTick()
    console.log('[ClusterDetail] nextTick 后 components.length:', componentInfo.value.components.length)
  } catch (error: any) {
    console.error('加载组件信息失败:', error)
    throw error
  }
}

// 加载节点列表
const loadNodes = async () => {
  nodesLoading.value = true
  try {
    const data = await getNodes(clusterId.value)
    nodeList.value = data || []
  } catch (error: any) {
    console.error('加载节点信息失败:', error)
    throw error
  } finally {
    nodesLoading.value = false
  }
}

// 加载事件列表
const loadEvents = async () => {
  eventsLoading.value = true
  try {
    const data = await getClusterEvents(clusterId.value)
    eventList.value = data || []
  } catch (error: any) {
    console.error('加载事件信息失败:', error)
    throw error
  } finally {
    eventsLoading.value = false
  }
}

// 格式化字节数
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 获取进度条颜色
const getProgressColor = (percentage: number) => {
  if (percentage < 50) return '#67C23A'
  if (percentage < 80) return '#E6A23C'
  return '#F56C6C'
}

// 返回列表
const handleBack = () => {
  router.push('/kubernetes/clusters')
}

// 获取状态类型
const getStatusType = (status: number) => {
  const statusMap: Record<number, string> = {
    1: 'success',
    2: 'danger',
    3: 'info'
  }
  return statusMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (status: number) => {
  const statusMap: Record<number, string> = {
    1: '正常',
    2: '连接失败',
    3: '不可用'
  }
  return statusMap[status] || '未知'
}

// 获取服务商文本
const getProviderText = (provider: string) => {
  const providerMap: Record<string, string> = {
    native: '自建集群',
    aliyun: '阿里云 ACK',
    tencent: '腾讯云 TKE',
    aws: 'AWS EKS'
  }
  return providerMap[provider] || provider
}

// 获取节点角色类型
const getNodeRoleType = (roles: string) => {
  if (!roles) return 'info'
  if (roles.toLowerCase().includes('master') || roles.toLowerCase().includes('control-plane')) {
    return 'danger'
  }
  return 'info'
}

onMounted(() => {
  loadClusterDetail()
})
</script>

<style scoped lang="scss">
.cluster-detail-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  padding: 24px;
}

/* 页面头部 */
.page-header {
  margin-bottom: 24px;

  .header-content {
    background: #fff;
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  }

  .header-top {
    margin-bottom: 20px;

    .back-btn {
      background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
      color: #D4AF37;
      border: 1px solid rgba(212, 175, 55, 0.3);
      font-weight: 500;
      padding: 12px 24px;
      font-size: 14px;
      border-radius: 8px;
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      display: inline-flex;
      align-items: center;
      gap: 6px;
      letter-spacing: 0.5px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(212, 175, 55, 0.4);
        border-color: rgba(212, 175, 55, 0.5);
        background: linear-gradient(135deg, #34495e 0%, #1a1a1a 100%);
      }

      &:active {
        transform: translateY(0);
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
      }

      :deep(.el-icon) {
        font-size: 16px;
        transition: transform 0.3s;
      }

      &:hover :deep(.el-icon) {
        transform: translateX(-3px);
      }
    }
  }

  .cluster-name-section {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;

    .cluster-title {
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
      font-size: 14px;
      padding: 8px 16px;
      border-radius: 20px;
    }
  }

  .cluster-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 24px;

    .meta-item {
      display: flex;
      align-items: center;
      gap: 6px;
      color: #606266;
      font-size: 14px;

      .el-icon {
        color: #D4AF37;
      }
    }
  }
}

/* 快速统计卡片 */
.quick-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;

  .stat-card {
    position: relative;
    background: #fff;
    border-radius: 12px;
    padding: 24px;
    display: flex;
    align-items: center;
    gap: 20px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    overflow: hidden;
    transition: all 0.3s ease;
    animation: slideInUp 0.5s ease-out var(--delay) backwards;

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: #D4AF37;
    }

    &:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(212, 175, 55, 0.3);
    }

    .stat-icon-wrapper {
      width: 64px;
      height: 64px;
      border-radius: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .stat-info {
      flex: 1;

      .stat-value {
        font-size: 32px;
        font-weight: 600;
        color: #303133;
        line-height: 1.2;
        margin-bottom: 4px;
      }

      .stat-label {
        font-size: 14px;
        color: #909399;
      }
    }

    .stat-trend {
      color: #D4AF37;
      font-size: 20px;
    }
  }
}

/* 主内容区 */
.main-content {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 20px;
}

/* 卡片通用样式 */
.modern-card {
  border-radius: 12px;
  border: none;
  margin-bottom: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  transition: all 0.3s;

  &:hover {
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  }

  :deep(.el-card__header) {
    padding: 20px 24px;
    border-bottom: 1px solid #f0f0f0;
  }

  :deep(.el-card__body) {
    padding: 24px;
  }

  .card-title-section {
    display: flex;
    align-items: center;
    gap: 10px;

    .card-icon {
      flex-shrink: 0;
    }

    .card-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }

    .node-count,
    .event-count {
      margin-left: auto;
      font-size: 13px;
      color: #909399;
      background: #f5f7fa;
      padding: 4px 12px;
      border-radius: 12px;
    }
  }
}

/* 全宽卡片 */
.full-width-card {
  grid-column: 1 / -1;
  margin-bottom: 20px;
}

/* 资源使用率 */
.resource-usage {
  .usage-item {
    margin-bottom: 24px;

    &:last-child {
      margin-bottom: 0;
    }

    .usage-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      .usage-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 15px;
        font-weight: 500;
        color: #303133;
      }

      .usage-value {
        font-size: 24px;
        font-weight: 600;
        color: #D4AF37;
      }
    }

    .progress-wrapper {
      margin-bottom: 8px;
    }

    .usage-detail {
      display: flex;
      justify-content: space-between;
      font-size: 13px;
      color: #909399;

      .detail-text {
        padding: 4px 12px;
        background: #f5f7fa;
        border-radius: 4px;
      }
    }
  }
}

/* 网络配置 */
.network-config {
  .config-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;

    .config-item {
      .config-label {
        font-size: 13px;
        color: #909399;
        margin-bottom: 8px;
      }

      .config-value {
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        word-break: break-all;

        &.primary {
          color: #D4AF37;
        }
      }

      .el-tag {
        width: 100%;
        justify-content: center;
      }
    }
  }
}

/* 集群信息 */
.cluster-info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;

  .info-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 8px;

    &.full-width {
      grid-column: 1 / -1;
    }

    .info-label {
      font-size: 13px;
      color: #909399;
    }

    .info-value {
      font-size: 14px;
      font-weight: 500;
      color: #303133;
      word-break: break-all;
    }
  }
}

/* 组件信息 */
.component-section {
  margin-bottom: 24px;

  &:last-child {
    margin-bottom: 0;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 15px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 16px;
    padding-bottom: 12px;
    border-bottom: 2px solid #f0f0f0;
  }

  .runtime-cards {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;

    .runtime-card {
      background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
      color: #D4AF37;
      padding: 16px;
      border-radius: 8px;
      transition: all 0.3s;
      border: 1px solid rgba(212, 175, 55, 0.2);

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
        border-color: rgba(212, 175, 55, 0.4);
      }

      .runtime-label {
        font-size: 12px;
        opacity: 0.9;
        margin-bottom: 6px;
      }

      .runtime-value {
        font-size: 15px;
        font-weight: 600;
      }
    }
  }

  .component-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .component-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 16px;
      background: #fafafa;
      border-radius: 8px;
      border-left: 3px solid #D4AF37;
      transition: all 0.3s;

      &:hover {
        background: #fefcf5;
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
      }

      .component-main {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;

        .component-icon {
          flex-shrink: 0;
        }

        .component-info {
          display: flex;
          align-items: center;
          gap: 12px;

          .component-name {
            font-size: 14px;
            font-weight: 500;
            color: #303133;
          }
        }
      }
    }
  }

  .storage-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .storage-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 16px;
      background: #fafafa;
      border-radius: 8px;
      border-left: 3px solid #D4AF37;
      transition: all 0.3s;

      &:hover {
        background: #fefcf5;
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
      }

      .storage-main {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;

        .storage-icon {
          flex-shrink: 0;
        }

        .storage-info {
          .storage-name {
            font-size: 14px;
            font-weight: 500;
            color: #303133;
            margin-bottom: 4px;
          }

          .storage-provisioner {
            font-size: 12px;
            color: #909399;
          }
        }
      }
    }
  }
}

/* 动画 */
@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .quick-stats {
    grid-template-columns: repeat(2, 1fr);
  }

  .main-content {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .cluster-detail-container {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;

    .header-content {
      width: 100%;
    }

    .cluster-meta {
      flex-direction: column;
      gap: 12px;
    }
  }

  .quick-stats {
    grid-template-columns: 1fr;
  }
}
</style>
