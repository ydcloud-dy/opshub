<template>
  <div class="dashboard">
    <section class="dashboard-command">
      <div class="command-copy">
        <div class="command-title-row">
          <h1>仪表盘</h1>
        </div>
        <p>聚合资产、集群、操作审计与告警数据，快速判断当前运维状态。</p>
      </div>
      <div class="command-actions">
        <el-button plain @click="refreshDashboard">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </section>

    <el-row :gutter="14" class="stats-row">
      <el-col
        v-for="(stat, index) in topStats"
        :key="index"
        :xs="24"
        :sm="12"
        :lg="6"
      >
        <el-card class="stat-card" shadow="never" :style="{ '--stat-color': stat.color }">
          <div class="stat-content">
            <div class="stat-icon">
              <el-icon :size="26">
                <component :is="stat.icon" />
              </el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ stat.label }}</div>
              <div class="stat-value">{{ stat.value }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="14" class="chart-row">
      <el-col :xs="24" :lg="12">
        <el-card class="chart-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <span class="card-title">主机状态分布</span>
                <span class="card-desc">在线与离线资产占比</span>
              </div>
              <el-button type="primary" link size="small" @click="$router.push('/asset/hosts')">查看全部</el-button>
            </div>
          </template>
          <div ref="hostStatusChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="12">
        <el-card class="chart-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <span class="card-title">K8s 集群资源概览</span>
                <span class="card-desc">节点与 Pod 规模对比</span>
              </div>
              <el-button type="primary" link size="small" @click="$router.push('/kubernetes/clusters')">查看全部</el-button>
            </div>
          </template>
          <div ref="k8sResourceChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="14" class="chart-row">
      <el-col :xs="24" :lg="12">
        <el-card class="chart-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <span class="card-title">操作趋势</span>
                <span class="card-desc">最近 7 天操作记录</span>
              </div>
              <el-button type="primary" link size="small" @click="$router.push('/audit/operation-logs')">查看全部</el-button>
            </div>
          </template>
          <div ref="operationTrendChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="12">
        <el-card class="chart-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <span class="card-title">告警统计</span>
                <span class="card-desc">按告警类型聚合</span>
              </div>
              <el-button type="primary" link size="small" @click="$router.push('/monitor/fault-centers')">查看全部</el-button>
            </div>
          </template>
          <div ref="alertStatsChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="14" class="quick-access-row">
      <el-col :span="24">
        <el-card class="quick-access-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span class="card-title">快速入口</span>
            </div>
          </template>
          <div class="quick-access-grid">
            <div class="quick-item" @click="$router.push('/asset/hosts')">
              <el-icon><OfficeBuilding /></el-icon>
              <span>主机管理</span>
            </div>
            <div class="quick-item" @click="$router.push('/kubernetes/clusters')">
              <el-icon><Connection /></el-icon>
              <span>K8s 集群</span>
            </div>
            <div class="quick-item" @click="$router.push('/audit/operation-logs')">
              <el-icon><Document /></el-icon>
              <span>操作日志</span>
            </div>
            <div class="quick-item" @click="$router.push('/monitor/fault-centers')">
              <el-icon><Warning /></el-icon>
              <span>故障中心</span>
            </div>
            <div class="quick-item" @click="$router.push('/asset/credentials')">
              <el-icon><Key /></el-icon>
              <span>凭据管理</span>
            </div>
            <div class="quick-item" @click="$router.push('/asset/cloud-accounts')">
              <el-icon><Cloudy /></el-icon>
              <span>云账号</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, markRaw } from 'vue'
import {
  OfficeBuilding,
  Connection,
  Document,
  Warning,
  Key,
  Cloudy,
  Refresh
} from '@element-plus/icons-vue'
import { getHostList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'
import { getOperationLogList } from '@/api/audit'
import { getAlertLogs } from '@/api/alert-config'
import * as echarts from 'echarts'

// 顶部统计数据
const topStats = ref([
  {
    label: '主机总数',
    value: '0',
    icon: markRaw(OfficeBuilding),
    color: '#409EFF'
  },
  {
    label: 'K8s集群',
    value: '0',
    icon: markRaw(Connection),
    color: '#67C23A'
  },
  {
    label: '今日操作',
    value: '0',
    icon: markRaw(Document),
    color: '#E6A23C'
  },
  {
    label: '活跃告警',
    value: '0',
    icon: markRaw(Warning),
    color: '#F56C6C'
  }
])

// 图表DOM引用
const hostStatusChart = ref<HTMLElement>()
const k8sResourceChart = ref<HTMLElement>()
const operationTrendChart = ref<HTMLElement>()
const alertStatsChart = ref<HTMLElement>()

// 数据存储
const hosts = ref<any[]>([])
const clusters = ref<any[]>([])
const operationLogs = ref<any[]>([])
const alertLogs = ref<any[]>([])

// 获取主机列表
const fetchHosts = async () => {
  try {
    const res: any = await getHostList({ page: 1, pageSize: 100 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        hosts.value = res.list
        topStats.value[0].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        hosts.value = res
        topStats.value[0].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderHostStatusChart()
  } catch (error) {
    topStats.value[0].value = '0'
  }
}

// 获取K8s集群列表
const fetchClusters = async () => {
  try {
    const res: any = await getClusterList()
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        clusters.value = res.list
        topStats.value[1].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        clusters.value = res
        topStats.value[1].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderK8sResourceChart()
  } catch (error) {
    topStats.value[1].value = '0'
  }
}

// 获取操作日志列表
const fetchOperationLogs = async () => {
  try {
    const today = new Date()
    today.setHours(0, 0, 0, 0)

    const res: any = await getOperationLogList({ page: 1, pageSize: 500 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        operationLogs.value = res.list
        const todayCount = res.list.filter((log: any) => {
          const logDate = new Date(log.createdAt)
          return logDate >= today
        }).length
        topStats.value[2].value = String(todayCount)
      } else if (Array.isArray(res)) {
        operationLogs.value = res
        const todayCount = res.filter((log: any) => {
          const logDate = new Date(log.createdAt)
          return logDate >= today
        }).length
        topStats.value[2].value = String(todayCount)
      }
    }
    await nextTick()
    renderOperationTrendChart()
  } catch (error) {
    topStats.value[2].value = '0'
  }
}

// 获取告警日志列表
const fetchAlertLogs = async () => {
  try {
    const res: any = await getAlertLogs({ page: 1, pageSize: 100 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        alertLogs.value = res.list
        const activeCount = res.list.filter((log: any) => log.status === 'failed').length
        topStats.value[3].value = String(activeCount)
      } else if (Array.isArray(res)) {
        alertLogs.value = res
        const activeCount = res.filter((log: any) => log.status === 'failed').length
        topStats.value[3].value = String(activeCount)
      }
    }
    await nextTick()
    renderAlertStatsChart()
  } catch (error) {
    topStats.value[3].value = '0'
  }
}

// 渲染主机状态图表
const renderHostStatusChart = () => {
  if (!hostStatusChart.value) return

  const chart = echarts.init(hostStatusChart.value)

  const onlineCount = hosts.value.filter(h => h.status === 1).length
  const offlineCount = hosts.value.filter(h => h.status !== 1).length

  const option = {
    color: ['#16a34a', '#98a2b3'],
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: {
        color: '#667085',
        fontSize: 12
      }
    },
    series: [
      {
        name: '主机状态',
        type: 'pie',
        radius: ['48%', '72%'],
        center: ['40%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: '#fff',
          borderWidth: 3
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: onlineCount, name: '在线', itemStyle: { color: '#16a34a' } },
          { value: offlineCount, name: '离线', itemStyle: { color: '#98a2b3' } }
        ]
      }
    ]
  }

  chart.setOption(option)

  // 响应式
  window.addEventListener('resize', () => chart.resize())
}

// 渲染K8s资源图表
const renderK8sResourceChart = () => {
  if (!k8sResourceChart.value) return

  const chart = echarts.init(k8sResourceChart.value)

  const clusterNames = clusters.value.map(c => c.name || '未命名')
  const nodeCounts = clusters.value.map(c => c.nodeCount || 0)
  const podCounts = clusters.value.map(c => c.podCount || 0)

  const option = {
    color: ['#2563eb', '#16a34a'],
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    legend: {
      data: ['节点数', 'Pod数'],
      top: 4,
      textStyle: {
        color: '#667085',
        fontSize: 12
      }
    },
    grid: {
      left: 8,
      right: 12,
      bottom: 8,
      top: 44,
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: clusterNames.length > 0 ? clusterNames : ['暂无数据'],
      axisLabel: {
        interval: 0,
        rotate: clusterNames.length > 3 ? 30 : 0,
        color: '#667085'
      },
      axisLine: {
        lineStyle: { color: '#e5e9f2' }
      },
      axisTick: {
        show: false
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: '#667085'
      },
      splitLine: {
        lineStyle: { color: '#edf1f7', type: 'dashed' }
      }
    },
    series: [
      {
        name: '节点数',
        type: 'bar',
        data: nodeCounts.length > 0 ? nodeCounts : [0],
        itemStyle: {
          color: '#2563eb',
          borderRadius: [6, 6, 0, 0]
        },
        barMaxWidth: 40
      },
      {
        name: 'Pod数',
        type: 'bar',
        data: podCounts.length > 0 ? podCounts : [0],
        itemStyle: {
          color: '#16a34a',
          borderRadius: [6, 6, 0, 0]
        },
        barMaxWidth: 40
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

// 渲染操作趋势图表
const renderOperationTrendChart = () => {
  if (!operationTrendChart.value) return

  const chart = echarts.init(operationTrendChart.value)

  // 统计最近7天的操作数
  const today = new Date()
  const dates: string[] = []
  const counts: number[] = []

  for (let i = 6; i >= 0; i--) {
    const date = new Date(today)
    date.setDate(date.getDate() - i)
    const dateStr = `${date.getMonth() + 1}/${date.getDate()}`
    dates.push(dateStr)

    const count = operationLogs.value.filter((log: any) => {
      const logDate = new Date(log.createdAt)
      return logDate.toDateString() === date.toDateString()
    }).length

    counts.push(count)
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    grid: {
      left: 8,
      right: 12,
      bottom: 8,
      top: 18,
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates,
      axisLabel: {
        color: '#667085'
      },
      axisLine: {
        lineStyle: { color: '#e5e9f2' }
      },
      axisTick: {
        show: false
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: '#667085'
      },
      splitLine: {
        lineStyle: { color: '#edf1f7', type: 'dashed' }
      }
    },
    series: [
      {
        name: '操作次数',
        type: 'line',
        smooth: true,
        data: counts,
        areaStyle: {
          color: 'rgba(217, 119, 6, 0.08)'
        },
        symbol: 'circle',
        symbolSize: 7,
        itemStyle: { color: '#d97706' },
        lineStyle: { width: 3, color: '#d97706' }
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

// 渲染告警统计图表
const renderAlertStatsChart = () => {
  if (!alertStatsChart.value) return

  const chart = echarts.init(alertStatsChart.value)

  const successCount = alertLogs.value.filter((log: any) => log.status === 'success').length
  const failedCount = alertLogs.value.filter((log: any) => log.status === 'failed').length

  // 按告警类型统计
  const typeMap = new Map<string, number>()
  alertLogs.value.forEach((log: any) => {
    const type = log.alertType || '未知'
    typeMap.set(type, (typeMap.get(type) || 0) + 1)
  })

  const typeData = Array.from(typeMap.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 5)

  const option = {
    color: ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed'],
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: typeData.map(d => d.name),
      itemWidth: 10,
      itemHeight: 10,
      textStyle: {
        color: '#667085',
        fontSize: 12
      }
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: ['48%', '72%'],
        center: ['40%', '50%'],
        itemStyle: {
          borderRadius: 8,
          borderColor: '#fff',
          borderWidth: 3
        },
        data: typeData.length > 0 ? typeData : [{ name: '暂无数据', value: 1 }],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

const refreshDashboard = () => {
  fetchHosts()
  fetchClusters()
  fetchOperationLogs()
  fetchAlertLogs()
}

// 页面加载时获取数据
onMounted(() => {
  refreshDashboard()
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.dashboard-command {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
  background: #ffffff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
}

.command-copy {
  min-width: 0;
}

.command-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.command-title-row h1 {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.2;
  letter-spacing: 0;
}

.command-copy p {
  margin: 5px 0 0;
  color: #667085;
  font-size: 14px;
  line-height: 1.5;
}

.command-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.command-actions :deep(.el-button) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  border-radius: 7px;
}

.stats-row,
.chart-row,
.quick-access-row {
  row-gap: 16px;
}

.stat-card {
  height: 100%;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  overflow: hidden;
  background: #ffffff;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.stat-card:hover {
  border-color: #d8dee9;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.06);
  transform: translateY(-1px);
}

.stat-card :deep(.el-card__body) {
  padding: 16px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 14px;
}

.stat-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--stat-color);
  background: #f8fafc;
  border: 1px solid #edf1f7;
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-value {
  margin-top: 4px;
  color: #111827;
  font-size: 28px;
  font-weight: 760;
  line-height: 1;
  letter-spacing: 0;
}

.stat-label {
  color: #667085;
  font-size: 13px;
  font-weight: 600;
}

.chart-card {
  height: 100%;
  border: 1px solid #e5e9f2;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.045);
}

.chart-card :deep(.el-card__header) {
  padding: 14px 16px;
  border-bottom: 1px solid #eef2f7;
  background: #fbfcfe;
}

.chart-card :deep(.el-card__body) {
  padding: 10px 14px 14px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.card-title {
  display: block;
  color: #111827;
  font-size: 16px;
  font-weight: 680;
}

.card-desc {
  display: block;
  margin-top: 3px;
  color: #98a2b3;
  font-size: 12px;
}

.chart-container {
  width: 100%;
  height: 260px;
}

.quick-access-card {
  border: 1px solid #e5e9f2;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.045);
}

.quick-access-card :deep(.el-card__header) {
  padding: 14px 16px;
  border-bottom: 1px solid #eef2f7;
  background: #fbfcfe;
}

.quick-access-card :deep(.el-card__body) {
  padding: 16px;
}

.quick-access-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(120px, 1fr));
  gap: 12px;
}

.quick-item {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  min-height: 58px;
  padding: 11px 12px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #edf1f7;
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, transform 0.2s ease;
}

.quick-item:hover {
  background: #ffffff;
  border-color: #ffaf35;
  transform: translateY(-1px);
}

.quick-item .el-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #111827;
  background: rgba(255, 175, 53, 0.18);
  font-size: 20px;
}

.quick-item span {
  color: #344054;
  font-size: 14px;
  font-weight: 650;
  white-space: nowrap;
}

@media (max-width: 1200px) {
  .stat-value {
    font-size: 26px;
  }

  .chart-container {
    height: 240px;
  }

  .quick-access-grid {
    grid-template-columns: repeat(3, minmax(140px, 1fr));
  }
}

@media (max-width: 768px) {
  .dashboard-command {
    align-items: stretch;
    flex-direction: column;
  }

  .command-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .quick-access-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .quick-item {
    min-height: 64px;
    padding: 12px;
  }
}
</style>
