<template>
  <div class="dashboard">
    <section class="dashboard-command">
      <div class="command-copy">
        <div class="command-eyebrow">
          <span class="page-mark">
            <el-icon><DataAnalysis /></el-icon>
          </span>
          <span>OpsHub Overview</span>
        </div>
        <h1>运维总览</h1>
        <p>聚合资产、集群、操作审计与告警事件，快速识别当前平台运行状态。</p>
      </div>
      <div class="command-actions">
        <span class="sync-chip"><i></i>实时概览</span>
        <el-button class="refresh-btn" plain @click="refreshDashboard">
          <el-icon><Refresh /></el-icon>
          刷新数据
        </el-button>
      </div>
    </section>

    <section class="stats-grid">
      <article
        v-for="(stat, index) in topStats"
        :key="index"
        class="stat-card"
        :style="{ '--stat-color': stat.color, '--stat-bg': stat.bg }"
      >
        <div class="stat-card-top">
          <div class="stat-info">
            <span class="stat-label">{{ stat.label }}</span>
            <strong class="stat-value">{{ stat.value }}</strong>
          </div>
          <div class="stat-icon">
            <el-icon :size="25">
              <component :is="stat.icon" />
            </el-icon>
          </div>
        </div>
        <div class="stat-card-bottom">
          <span>{{ stat.caption }}</span>
          <em>{{ stat.badge }}</em>
        </div>
      </article>
    </section>

    <section class="dashboard-grid">
      <article class="dashboard-panel host-panel">
        <header class="panel-header">
          <div>
            <h2>主机状态分布</h2>
            <p>在线与离线资产占比</p>
          </div>
          <button class="panel-link" @click="$router.push('/asset/hosts')">
            查看全部
            <el-icon><ArrowRight /></el-icon>
          </button>
        </header>
        <div class="panel-body host-panel-body">
          <div ref="hostStatusChart" class="chart-container chart-container-donut"></div>
          <div class="status-breakdown">
            <div class="status-rate">
              <span>在线率</span>
              <strong>{{ hostOnlineRate }}%</strong>
            </div>
            <div class="status-row">
              <span><i class="dot online"></i>在线主机</span>
              <strong>{{ hostOnlineCount }}</strong>
            </div>
            <div class="status-row">
              <span><i class="dot offline"></i>离线主机</span>
              <strong>{{ hostOfflineCount }}</strong>
            </div>
            <div class="status-row muted">
              <span>资产总量</span>
              <strong>{{ hostTotalCount }}</strong>
            </div>
          </div>
        </div>
      </article>

      <article class="dashboard-panel">
        <header class="panel-header">
          <div>
            <h2>K8s 集群资源概览</h2>
            <p>节点与 Pod 规模对比</p>
          </div>
          <button class="panel-link" @click="$router.push('/kubernetes/clusters')">
            查看全部
            <el-icon><ArrowRight /></el-icon>
          </button>
        </header>
        <div class="panel-metrics">
          <div>
            <strong>{{ totalNodeCount }}</strong>
            <span>节点总数</span>
          </div>
          <div>
            <strong>{{ totalPodCount }}</strong>
            <span>Pod 总数</span>
          </div>
          <div>
            <strong>{{ clusters.length }}</strong>
            <span>集群数量</span>
          </div>
        </div>
        <div ref="k8sResourceChart" class="chart-container"></div>
      </article>
    </section>

    <section class="dashboard-grid dashboard-grid-lower">
      <article class="dashboard-panel trend-panel">
        <header class="panel-header">
          <div>
            <h2>操作趋势</h2>
            <p>最近 7 天操作记录</p>
          </div>
          <button class="panel-link" @click="$router.push('/audit/operation-logs')">
            查看全部
            <el-icon><ArrowRight /></el-icon>
          </button>
        </header>
        <div ref="operationTrendChart" class="chart-container"></div>
      </article>

      <article class="dashboard-panel alert-panel">
        <header class="panel-header">
          <div>
            <h2>告警统计</h2>
            <p>按告警类型聚合</p>
          </div>
          <button class="panel-link" @click="$router.push('/monitor/fault-centers')">
            查看全部
            <el-icon><ArrowRight /></el-icon>
          </button>
        </header>
        <div class="alert-summary">
          <div>
            <span>今日事件</span>
            <strong>{{ alertTodayCount }}</strong>
          </div>
          <div>
            <span>预告警</span>
            <strong>{{ alertPendingCount }}</strong>
          </div>
        </div>
        <div ref="alertStatsChart" class="chart-container chart-container-donut"></div>
      </article>
    </section>

    <section class="quick-access-panel">
      <header class="panel-header">
        <div>
          <h2>快速入口</h2>
          <p>常用运维入口集中访问</p>
        </div>
      </header>
      <div class="quick-access-grid">
        <button
          v-for="item in quickAccess"
          :key="item.path"
          class="quick-item"
          :style="{ '--quick-color': item.color, '--quick-bg': item.bg }"
          @click="$router.push(item.path)"
        >
          <span class="quick-icon">
            <el-icon><component :is="item.icon" /></el-icon>
          </span>
          <span class="quick-copy">
            <strong>{{ item.label }}</strong>
            <em>{{ item.desc }}</em>
          </span>
          <el-icon class="quick-arrow"><ArrowRight /></el-icon>
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, markRaw, computed } from 'vue'
import {
  OfficeBuilding,
  Connection,
  Document,
  Warning,
  Key,
  Cloudy,
  Refresh,
  DataAnalysis,
  ArrowRight
} from '@element-plus/icons-vue'
import { getHostList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'
import { getOperationLogList } from '@/api/audit'
import {
  getMonitorAlertEvents,
  getMonitorAlertEventStats,
  type MonitorAlertEvent,
  type MonitorAlertEventStats
} from '@/api/monitor-datasource'
import * as echarts from 'echarts'

const topStats = ref([
  {
    label: '主机总数',
    value: '0',
    icon: markRaw(OfficeBuilding),
    color: '#2563eb',
    bg: '#eff6ff',
    caption: '纳管资产',
    badge: '在线状态'
  },
  {
    label: 'K8s集群',
    value: '0',
    icon: markRaw(Connection),
    color: '#16a34a',
    bg: '#ecfdf3',
    caption: '集群资源',
    badge: '节点 / Pod'
  },
  {
    label: '今日操作',
    value: '0',
    icon: markRaw(Document),
    color: '#d97706',
    bg: '#fffbeb',
    caption: '审计记录',
    badge: '今日'
  },
  {
    label: '活跃告警',
    value: '0',
    icon: markRaw(Warning),
    color: '#dc2626',
    bg: '#fef2f2',
    caption: '告警事件',
    badge: '待处理'
  }
])

const quickAccess = [
  {
    label: '主机管理',
    desc: '资产连接与分组',
    path: '/asset/hosts',
    icon: markRaw(OfficeBuilding),
    color: '#2563eb',
    bg: '#eff6ff'
  },
  {
    label: 'K8s 集群',
    desc: '集群资源与状态',
    path: '/kubernetes/clusters',
    icon: markRaw(Connection),
    color: '#16a34a',
    bg: '#ecfdf3'
  },
  {
    label: '操作日志',
    desc: '审计追踪',
    path: '/audit/operation-logs',
    icon: markRaw(Document),
    color: '#d97706',
    bg: '#fffbeb'
  },
  {
    label: '故障中心',
    desc: '告警事件处理',
    path: '/monitor/fault-centers',
    icon: markRaw(Warning),
    color: '#dc2626',
    bg: '#fef2f2'
  },
  {
    label: '凭据管理',
    desc: '账号与密钥',
    path: '/asset/credentials',
    icon: markRaw(Key),
    color: '#7c3aed',
    bg: '#f5f3ff'
  },
  {
    label: '云账号',
    desc: '多云接入',
    path: '/asset/cloud-accounts',
    icon: markRaw(Cloudy),
    color: '#0891b2',
    bg: '#ecfeff'
  }
]

const hostStatusChart = ref<HTMLElement>()
const k8sResourceChart = ref<HTMLElement>()
const operationTrendChart = ref<HTMLElement>()
const alertStatsChart = ref<HTMLElement>()

const hosts = ref<any[]>([])
const clusters = ref<any[]>([])
const operationTrend = ref<{ date: string; label: string; count: number }[]>([])
const alertEvents = ref<MonitorAlertEvent[]>([])
const alertEventStats = ref<MonitorAlertEventStats>({
  totalRules: 0,
  enabledRules: 0,
  firingRules: 0,
  pendingRules: 0,
  todayEvents: 0,
  unresolvedEvents: 0
})

const isHostOnline = (host: any) => {
  const status = String(host?.status ?? '').toLowerCase()
  return host?.status === 1 || ['online', 'running', 'active', 'success'].includes(status)
}

const hostOnlineCount = computed(() => hosts.value.filter(isHostOnline).length)
const hostTotalCount = computed(() => hosts.value.length)
const hostOfflineCount = computed(() => Math.max(hostTotalCount.value - hostOnlineCount.value, 0))
const hostOnlineRate = computed(() => {
  if (!hostTotalCount.value) return 0
  return Math.round((hostOnlineCount.value / hostTotalCount.value) * 100)
})
const pickNumber = (source: Record<string, any>, keys: string[]) => {
  for (const key of keys) {
    const raw = source?.[key]
    if (Array.isArray(raw)) return raw.length
    if (raw === undefined || raw === null || raw === '') continue

    const value = Number(raw)
    if (Number.isFinite(value)) return value
  }
  return 0
}
const getNodeCount = (cluster: any) => pickNumber(cluster, ['nodeCount', 'node_count', 'totalNodes', 'nodes'])
const getPodCount = (cluster: any) => pickNumber(cluster, ['podCount', 'pod_count', 'totalPods', 'pods'])
const totalNodeCount = computed(() => clusters.value.reduce((sum, item: any) => sum + getNodeCount(item), 0))
const totalPodCount = computed(() => clusters.value.reduce((sum, item: any) => sum + getPodCount(item), 0))
const alertTodayCount = computed(() => alertEventStats.value.todayEvents || 0)
const alertPendingCount = computed(() => alertEventStats.value.pendingRules || 0)

const formatDate = (date: Date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const getChart = (target?: HTMLElement) => {
  if (!target) return null
  return echarts.getInstanceByDom(target) || echarts.init(target)
}

const resizeCharts = () => {
  ;[hostStatusChart.value, k8sResourceChart.value, operationTrendChart.value, alertStatsChart.value].forEach((target) => {
    if (!target) return
    echarts.getInstanceByDom(target)?.resize()
  })
}

const disposeCharts = () => {
  ;[hostStatusChart.value, k8sResourceChart.value, operationTrendChart.value, alertStatsChart.value].forEach((target) => {
    if (!target) return
    echarts.getInstanceByDom(target)?.dispose()
  })
}

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

const fetchOperationLogs = async () => {
  try {
    const today = new Date()
    const days = Array.from({ length: 7 }, (_, index) => {
      const date = new Date(today)
      date.setDate(today.getDate() - (6 - index))
      return {
        date: formatDate(date),
        label: `${date.getMonth() + 1}/${date.getDate()}`
      }
    })

    const results = await Promise.all(
      days.map(day =>
        getOperationLogList({
          page: 1,
          pageSize: 1,
          startTime: day.date,
          endTime: day.date
        }).catch(() => ({ total: 0 }))
      )
    )

    operationTrend.value = days.map((day, index) => ({
      ...day,
      count: Number((results[index] as any)?.total || 0)
    }))
    topStats.value[2].value = String(operationTrend.value[operationTrend.value.length - 1]?.count || 0)
    await nextTick()
    renderOperationTrendChart()
  } catch (error) {
    topStats.value[2].value = '0'
  }
}

const fetchAlertEvents = async () => {
  try {
    const [stats, eventsRes]: any[] = await Promise.all([
      getMonitorAlertEventStats(),
      getMonitorAlertEvents({ page: 1, pageSize: 100, scope: 'active' })
    ])
    if (stats) {
      alertEventStats.value = {
        totalRules: Number(stats.totalRules || 0),
        enabledRules: Number(stats.enabledRules || 0),
        firingRules: Number(stats.firingRules || 0),
        pendingRules: Number(stats.pendingRules || 0),
        todayEvents: Number(stats.todayEvents || 0),
        unresolvedEvents: Number(stats.unresolvedEvents || 0)
      }
      topStats.value[3].value = String(alertEventStats.value.unresolvedEvents || 0)
    }
    if (eventsRes?.list && Array.isArray(eventsRes.list)) {
      alertEvents.value = eventsRes.list
    } else if (Array.isArray(eventsRes)) {
      alertEvents.value = eventsRes
    } else {
      alertEvents.value = []
    }
    await nextTick()
    renderAlertStatsChart()
  } catch (error) {
    topStats.value[3].value = '0'
  }
}

const renderHostStatusChart = () => {
  const chart = getChart(hostStatusChart.value)
  if (!chart) return

  const option = {
    color: ['#16a34a', '#cbd5e1'],
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    legend: {
      bottom: 0,
      left: 'center',
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 16,
      textStyle: {
        color: '#475467',
        fontSize: 12
      }
    },
    series: [
      {
        name: '主机状态',
        type: 'pie',
        radius: ['54%', '76%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 4
        },
        label: {
          show: true,
          position: 'center',
          formatter: `${hostOnlineRate.value}%\n在线率`,
          lineHeight: 24,
          color: '#111827',
          fontSize: 22,
          fontWeight: 760
        },
        labelLine: {
          show: false
        },
        data: [
          { value: hostOnlineCount.value, name: '在线', itemStyle: { color: '#16a34a' } },
          { value: hostOfflineCount.value, name: '离线', itemStyle: { color: '#cbd5e1' } }
        ]
      }
    ]
  }

  chart.setOption(option)
}

const renderK8sResourceChart = () => {
  const chart = getChart(k8sResourceChart.value)
  if (!chart) return

  const clusterNames = clusters.value.map(c => c.name || '未命名')
  const nodeCounts = clusters.value.map(getNodeCount)
  const podCounts = clusters.value.map(getPodCount)

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
      top: 0,
      right: 6,
      textStyle: {
        color: '#475467',
        fontSize: 12
      }
    },
    grid: {
      left: 6,
      right: 12,
      bottom: 4,
      top: 42,
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: clusterNames.length > 0 ? clusterNames : ['暂无数据'],
      axisLabel: {
        interval: 0,
        rotate: clusterNames.length > 3 ? 30 : 0,
        color: '#667085',
        hideOverlap: true
      },
      axisLine: {
        lineStyle: { color: '#e2e8f0' }
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
        lineStyle: { color: '#edf2f7', type: 'dashed' }
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
        barMaxWidth: 34,
        barGap: '28%'
      },
      {
        name: 'Pod数',
        type: 'bar',
        data: podCounts.length > 0 ? podCounts : [0],
        itemStyle: {
          color: '#16a34a',
          borderRadius: [6, 6, 0, 0]
        },
        barMaxWidth: 34,
        barGap: '28%'
      }
    ]
  }

  chart.setOption(option)
}

const renderOperationTrendChart = () => {
  const chart = getChart(operationTrendChart.value)
  if (!chart) return

  const dates = operationTrend.value.map(item => item.label)
  const counts = operationTrend.value.map(item => item.count)

  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#fff' }
    },
    grid: {
      left: 6,
      right: 12,
      bottom: 4,
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
        lineStyle: { color: '#e2e8f0' }
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
        lineStyle: { color: '#edf2f7', type: 'dashed' }
      }
    },
    series: [
      {
        name: '操作次数',
        type: 'line',
        smooth: true,
        data: counts,
        areaStyle: {
          color: 'rgba(217, 119, 6, 0.10)'
        },
        symbol: 'circle',
        symbolSize: 6,
        itemStyle: { color: '#d97706' },
        lineStyle: { width: 3, color: '#d97706' }
      }
    ]
  }

  chart.setOption(option)
}

const renderAlertStatsChart = () => {
  const chart = getChart(alertStatsChart.value)
  if (!chart) return

  const typeMap = new Map<string, number>()
  alertEvents.value.forEach((event: MonitorAlertEvent) => {
    const type = event.dataSourceType || event.severity || '未知'
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
      bottom: 0,
      left: 'center',
      data: typeData.map(d => d.name),
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 14,
      textStyle: {
        color: '#475467',
        fontSize: 12
      }
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: ['50%', '72%'],
        center: ['50%', '43%'],
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 4
        },
        data: typeData.length > 0 ? typeData : [{ name: '暂无数据', value: 1 }],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.18)'
          }
        }
      }
    ]
  }

  chart.setOption(option)
}

const refreshDashboard = () => {
  fetchHosts()
  fetchClusters()
  fetchOperationLogs()
  fetchAlertEvents()
}

onMounted(() => {
  refreshDashboard()
  window.addEventListener('resize', resizeCharts)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeCharts)
  disposeCharts()
})
</script>

<style scoped>
.dashboard {
  display: grid;
  gap: 16px;
  color: #111827;
}

.dashboard-command {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 118px;
  padding: 22px 26px;
  background: #ffffff;
  border: 1px solid #e1e7ef;
  border-radius: 10px;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.command-copy {
  min-width: 0;
}

.command-eyebrow {
  display: flex;
  align-items: center;
  gap: 9px;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.page-mark {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #111827;
  background: #ffaf35;
  border-radius: 8px;
}

.command-copy h1 {
  margin: 10px 0 0;
  color: #111827;
  font-size: 25px;
  font-weight: 760;
  line-height: 1.2;
  letter-spacing: 0;
}

.command-copy p {
  margin: 7px 0 0;
  color: #667085;
  font-size: 14px;
  line-height: 1.5;
}

.command-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.sync-chip {
  height: 34px;
  padding: 0 12px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #344054;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
}

.sync-chip i {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: #16a34a;
  box-shadow: 0 0 0 4px rgba(22, 163, 74, 0.13);
}

.refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  border-radius: 8px;
  font-weight: 650;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.stat-card {
  position: relative;
  min-height: 116px;
  padding: 18px;
  border: 1px solid #e1e7ef;
  border-radius: 10px;
  background: #ffffff;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
  overflow: hidden;
}

.stat-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 12px 26px rgba(15, 23, 42, 0.07);
  transform: translateY(-1px);
}

.stat-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.stat-icon {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--stat-color);
  background: var(--stat-bg);
  border: 1px solid rgba(148, 163, 184, 0.22);
}

.stat-info {
  min-width: 0;
}

.stat-value {
  display: block;
  margin-top: 8px;
  color: #111827;
  font-size: 30px;
  font-weight: 760;
  line-height: 1;
  letter-spacing: 0;
}

.stat-label {
  color: #667085;
  font-size: 13px;
  font-weight: 700;
}

.stat-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 18px;
  color: #667085;
  font-size: 12px;
  font-weight: 650;
}

.stat-card-bottom em {
  padding: 3px 8px;
  color: var(--stat-color);
  background: var(--stat-bg);
  border-radius: 999px;
  font-style: normal;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.dashboard-grid-lower {
  grid-template-columns: minmax(0, 1.2fr) minmax(360px, 0.8fr);
}

.dashboard-panel,
.quick-access-panel {
  min-width: 0;
  background: #ffffff;
  border: 1px solid #e1e7ef;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 72px;
  padding: 16px 20px;
  border-bottom: 1px solid #edf2f7;
  background: #fbfcfe;
}

.panel-header h2 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 760;
  line-height: 1.35;
  letter-spacing: 0;
}

.panel-header p {
  margin: 3px 0 0;
  color: #8a94a6;
  font-size: 13px;
  line-height: 1.4;
}

.panel-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 9px;
  color: #344054;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: color 0.2s ease, background-color 0.2s ease, border-color 0.2s ease;
}

.panel-link:hover {
  color: #111827;
  background: #ffffff;
  border-color: #d8dee9;
}

.panel-body {
  padding: 18px 20px 20px;
}

.chart-container {
  width: 100%;
  height: 278px;
}

.dashboard-panel > .chart-container {
  width: calc(100% - 40px);
  margin: 12px 20px 18px;
}

.chart-container-donut {
  height: 260px;
}

.host-panel-body {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(160px, 190px);
  align-items: center;
  gap: 12px;
}

.status-breakdown {
  display: grid;
  gap: 10px;
}

.status-rate {
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}

.status-rate span,
.status-row span,
.alert-summary span,
.panel-metrics span {
  display: block;
  color: #667085;
  font-size: 12px;
  font-weight: 650;
}

.status-rate strong {
  display: block;
  margin-top: 8px;
  color: #111827;
  font-size: 32px;
  line-height: 1;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 42px;
  padding: 0 13px;
  background: #ffffff;
  border: 1px solid #edf2f7;
  border-radius: 9px;
}

.status-row strong {
  color: #111827;
  font-size: 16px;
}

.status-row.muted {
  background: #fbfcfe;
}

.dot {
  width: 8px;
  height: 8px;
  display: inline-block;
  margin-right: 7px;
  border-radius: 999px;
}

.dot.online {
  background: #16a34a;
}

.dot.offline {
  background: #cbd5e1;
}

.panel-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 16px 20px 0;
}

.panel-metrics div,
.alert-summary div {
  padding: 12px 14px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 9px;
}

.panel-metrics strong,
.alert-summary strong {
  display: block;
  margin-top: 6px;
  color: #111827;
  font-size: 22px;
  line-height: 1;
}

.alert-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 16px 20px 0;
}

.quick-access-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
  padding: 16px 20px 20px;
}

.quick-item {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 72px;
  padding: 12px;
  border-radius: 10px;
  color: inherit;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.2s ease, background-color 0.2s ease, transform 0.2s ease;
}

.quick-item:hover {
  background: #fbfcfe;
  border-color: var(--quick-color);
  transform: translateY(-1px);
}

.quick-icon {
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 9px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--quick-color);
  background: var(--quick-bg);
  border: 1px solid rgba(148, 163, 184, 0.2);
  font-size: 19px;
}

.quick-copy {
  min-width: 0;
  flex: 1;
}

.quick-copy strong {
  display: block;
  color: #344054;
  font-size: 14px;
  font-weight: 760;
  white-space: nowrap;
}

.quick-copy em {
  display: block;
  margin-top: 3px;
  color: #8a94a6;
  font-size: 12px;
  font-style: normal;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.quick-arrow {
  color: #98a2b3;
  font-size: 14px;
}

@media (max-width: 1200px) {
  .stats-grid,
  .dashboard-grid,
  .dashboard-grid-lower {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-grid-lower {
    grid-template-columns: minmax(0, 1fr);
  }

  .stat-value {
    font-size: 26px;
  }

  .chart-container {
    height: 252px;
  }

  .quick-access-grid {
    grid-template-columns: repeat(3, minmax(140px, 1fr));
  }
}

@media (max-width: 768px) {
  .dashboard-command {
    align-items: stretch;
    flex-direction: column;
    min-height: unset;
    padding: 18px;
  }

  .command-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .stats-grid,
  .dashboard-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .host-panel-body {
    grid-template-columns: minmax(0, 1fr);
  }

  .panel-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .quick-access-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .quick-item {
    min-height: 64px;
    padding: 12px;
  }
}

@media (max-width: 560px) {
  .quick-access-grid,
  .panel-metrics,
  .alert-summary {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
