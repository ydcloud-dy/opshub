<template>
  <div class="inventory-page">
    <PageHeader :icon="Share" title="应用依赖拓扑" description="以应用为中心查看环境、入口、部署资源、数据组件和出向调用关系。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ topology.nodes.length }}</strong><span>拓扑节点</span></div>
          <div class="inventory-header-metric"><strong>{{ topology.edges.length }}</strong><span>动态链路</span></div>
          <div class="inventory-header-metric"><strong>{{ topology.stats.application || 0 }}</strong><span>应用节点</span></div>
          <div class="inventory-header-metric"><strong>{{ selectedAppName || '全局' }}</strong><span>当前范围</span></div>
        </div>
      </template>
      <el-button :icon="Refresh" :loading="loading" @click="loadTopology">刷新</el-button>
      <el-button :icon="FullScreen" @click="resetView">适应画布</el-button>
    </PageHeader>

    <div class="inventory-toolbar topology-toolbar">
      <el-select v-model="selectedAppId" clearable filterable placeholder="全局应用关系" style="width:280px" @change="changeApplication">
        <el-option v-for="app in applications" :key="app.id" :label="`${app.name} (${app.code})`" :value="app.id" />
      </el-select>
      <el-segmented v-model="layoutMode" :options="layoutOptions" @change="applyLayout" />
      <span class="inventory-muted">{{ selectedAppName ? `正在追踪“${selectedAppName}”的完整资产链路` : '正在查看全部应用之间的调用网络' }}</span>
    </div>

    <section class="inventory-panel inventory-topology-panel">
      <div class="inventory-topology-panel__head">
        <div class="inventory-topology-panel__title">
          <h3>{{ selectedAppName || '全局应用调用网络' }}</h3>
          <p>{{ selectedAppName ? '入口、环境、部署、数据组件与外部调用按真实关系展开。' : '双击应用节点进入详情，拖动节点可以临时整理视图。' }}</p>
        </div>
        <div class="inventory-topology-legend">
          <span v-for="item in legend" :key="item.type" class="inventory-topology-legend__item" :style="{ '--legend-color': item.color }">
            <i class="inventory-topology-legend__dot" />{{ item.label }} <strong>{{ topology.stats[item.type] || 0 }}</strong>
          </span>
        </div>
      </div>

      <div class="inventory-topology inventory-topology--live" v-loading="loading">
        <VueFlow
          id="app-inventory-topology"
          v-model:nodes="flowNodes"
          v-model:edges="flowEdges"
          class="inventory-topology__canvas inventory-topology-flow"
          :min-zoom="0.25"
          :max-zoom="2.2"
          :nodes-connectable="false"
          :edges-focusable="false"
          :elevate-nodes-on-select="true"
          fit-view-on-init
          @init="onFlowInit"
          @node-double-click="openNode"
        >
          <Background :variant="BackgroundVariant.Dots" :gap="22" :size="1.1" pattern-color="rgba(167, 184, 205, 0.18)" />
          <template #node-inventory="nodeProps"><TopologyNodeCard v-bind="nodeProps" /></template>
          <template #edge-telemetry="edgeProps"><TopologyFlowEdge v-bind="edgeProps" /></template>
          <MiniMap v-if="flowNodes.length > 4" pannable zoomable :node-color="miniMapNodeColor" mask-color="rgba(10, 13, 17, 0.72)" />
          <Controls position="bottom-right" :show-interactive="false" />
        </VueFlow>
        <div v-if="!loading && !topology.nodes.length" class="inventory-topology-empty inventory-topology-empty--dark">
          <el-empty description="暂无拓扑数据" />
          <p>先登记应用环境、资源或调用依赖，拓扑会在这里展开。</p>
        </div>
        <div v-else-if="!loading && topology.nodes.length && !topology.edges.length" class="inventory-topology-zero-edge">尚未登记应用调用关系</div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { FullScreen, Refresh, Share } from '@element-plus/icons-vue'
import { Background, BackgroundVariant } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { MarkerType, VueFlow, type Edge, type Node, type VueFlowStore } from '@vue-flow/core'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import { getTopology, listApplications, type Application, type TopologyEdge, type TopologyNode } from '../api'
import PageHeader from './PageHeader.vue'
import TopologyFlowEdge from './TopologyFlowEdge.vue'
import TopologyNodeCard from './TopologyNodeCard.vue'

type LayoutMode = 'layered' | 'panorama'
type Palette = { label: string; color: string; glow: string }

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const applications = ref<Application[]>([])
const selectedAppId = ref<number | undefined>(route.query.app_id ? Number(route.query.app_id) : undefined)
const layoutMode = ref<LayoutMode>(selectedAppId.value ? 'layered' : 'panorama')
const flowNodes = ref<Node[]>([])
const flowEdges = ref<Edge[]>([])
const flowInstance = shallowRef<VueFlowStore>()
const topology = ref<{ nodes: TopologyNode[]; edges: TopologyEdge[]; stats: Record<string, number> }>({ nodes: [], edges: [], stats: {} })
const layoutOptions = [{ label: '链路分层', value: 'layered' }, { label: '全景关系', value: 'panorama' }]
const palette: Record<string, Palette> = {
  application: { label: '应用', color: '#ffc857', glow: 'rgba(255, 200, 87, .34)' },
  environment: { label: '环境', color: '#48d6c4', glow: 'rgba(72, 214, 196, .3)' },
  domain: { label: '入口', color: '#b58cff', glow: 'rgba(181, 140, 255, .3)' },
  resource: { label: '部署资源', color: '#53c7ff', glow: 'rgba(83, 199, 255, .3)' },
  component: { label: '数据组件', color: '#ff7d5c', glow: 'rgba(255, 125, 92, .3)' },
  external: { label: '外部依赖', color: '#aab6c5', glow: 'rgba(170, 182, 197, .24)' },
}
const fallbackPalette: Palette = { label: '外部依赖', color: '#aab6c5', glow: 'rgba(170, 182, 197, .24)' }
const getPalette = (type: string): Palette => palette[type] ?? fallbackPalette
const legend = Object.entries(palette).map(([type, item]) => ({ type, label: item.label, color: item.color }))
const selectedAppName = computed(() => applications.value.find(item => item.id === selectedAppId.value)?.name || '')

const loadApplications = async () => {
  const data = await listApplications({ page: 1, page_size: 200 })
  applications.value = data?.list || []
}

const loadTopology = async () => {
  loading.value = true
  try {
    topology.value = await getTopology(selectedAppId.value)
    applyLayout()
    await nextTick()
    window.requestAnimationFrame(resetView)
  } catch {
    ElMessage.error('加载依赖拓扑失败')
  } finally {
    loading.value = false
  }
}

const nodeDetail = (node: TopologyNode) => {
  const metadata = node.metadata || {}
  if (node.type === 'application') return [metadata.code, metadata.language || metadata.team].filter(Boolean).join(' · ')
  if (node.type === 'environment') return [metadata.kind, metadata.region].filter(Boolean).join(' · ')
  if (node.type === 'domain') return [String(metadata.protocol || '').toUpperCase(), metadata.port].filter(Boolean).join(' · ')
  if (node.type === 'resource') return [metadata.kind, metadata.namespace || metadata.address].filter(Boolean).join(' · ')
  if (node.type === 'component') return [metadata.type || metadata.category, metadata.address].filter(Boolean).join(' · ')
  return metadata.endpoint || '外部系统'
}

const layeredPositions = (nodes: TopologyNode[]) => {
  const order = selectedAppId.value ? ['domain', 'application', 'environment', 'resource', 'component', 'external'] : ['application', 'external']
  const groups = new Map<string, TopologyNode[]>()
  nodes.forEach(node => groups.set(node.type, [...(groups.get(node.type) || []), node]))
  const columns = [...order.filter(type => groups.has(type)), ...Array.from(groups.keys()).filter(type => !order.includes(type))]
  const positions = new Map<string, { x: number; y: number }>()
  const maxRows = Math.max(1, ...columns.map(type => groups.get(type)?.length || 0))
  columns.forEach((type, columnIndex) => {
    const items = groups.get(type) || []
    const startY = 60 + (maxRows - items.length) * 54
    items.forEach((node, rowIndex) => positions.set(node.id, { x: 70 + columnIndex * 280, y: startY + rowIndex * 108 }))
  })
  return positions
}

const panoramaPositions = (nodes: TopologyNode[]) => {
  const positions = new Map<string, { x: number; y: number }>()
  const selectedNodeId = selectedAppId.value ? `app:${selectedAppId.value}` : ''
  const centerNode = nodes.find(node => node.id === selectedNodeId)
  const orbitNodes = centerNode ? nodes.filter(node => node.id !== selectedNodeId) : nodes
  const center = { x: 560, y: 360 }
  if (centerNode) positions.set(centerNode.id, center)
  const count = Math.max(orbitNodes.length, 1)
  orbitNodes.forEach((node, index) => {
    const angle = -Math.PI / 2 + index * (Math.PI * 2 / count)
    const ring = node.type === 'application' ? 260 : node.type === 'external' ? 430 : 350
    positions.set(node.id, { x: center.x + Math.cos(angle) * ring, y: center.y + Math.sin(angle) * ring * 0.72 })
  })
  return positions
}

const applyLayout = () => {
  const positions = layoutMode.value === 'layered' ? layeredPositions(topology.value.nodes) : panoramaPositions(topology.value.nodes)
  flowNodes.value = topology.value.nodes.map(node => {
    const style = getPalette(node.type)
    return {
      id: node.id,
      type: 'inventory',
      position: positions.get(node.id) || { x: 0, y: 0 },
      data: { ...node, typeLabel: style.label, detail: nodeDetail(node), color: style.color, glow: style.glow },
      draggable: true,
      selectable: true,
    }
  })
  flowEdges.value = topology.value.edges.map((edge, index) => {
    const targetType = topology.value.nodes.find(node => node.id === edge.target)?.type || 'external'
    const isDependency = edge.kind === 'dependency'
    const color = isDependency ? '#ffc857' : getPalette(targetType).color
    return {
      id: edge.id || `edge-${index}`,
      source: edge.source,
      target: edge.target,
      type: 'telemetry',
      animated: true,
      selectable: false,
      markerEnd: { type: MarkerType.ArrowClosed, color, width: 18, height: 18 },
      data: { ...edge, color, baseColor: isDependency ? 'rgba(255, 200, 87, .38)' : 'rgba(151, 171, 195, .35)', layout: layoutMode.value },
    }
  })
  nextTick(() => window.requestAnimationFrame(resetView))
}

const onFlowInit = (instance: VueFlowStore) => {
  flowInstance.value = instance
  resetView()
}

const resetView = () => flowInstance.value?.fitView({ padding: 0.22, duration: 480, maxZoom: 1.25 })
const openNode = ({ node }: { node: Node }) => {
  if (node.data?.type === 'application' && node.data?.appId) router.push(`/app-inventory/apps/${node.data.appId}`)
}
const miniMapNodeColor = (node: Node) => getPalette(String(node.data?.type || 'external')).color

const changeApplication = async () => {
  layoutMode.value = selectedAppId.value ? 'layered' : 'panorama'
  await router.replace({ path: '/app-inventory/topology', query: selectedAppId.value ? { app_id: selectedAppId.value } : {} })
  await loadTopology()
}

onMounted(async () => {
  await Promise.all([loadApplications(), loadTopology()])
})
</script>
