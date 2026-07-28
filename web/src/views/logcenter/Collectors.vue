<template>
  <div class="log-center-page ingest-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2>采集接入</h2>
          <p>管理日志采集链路、采集策略、运行实例与配置发布版本</p>
        </div>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="collector-tabs">
      <el-tab-pane label="采集链路" name="pipeline">
    <section class="panel pipeline-panel" v-loading="loading && !status">
      <div class="panel-heading">
        <div>
          <h3>实时采集链路</h3>
          <p>{{ pipelineDescription }}</p>
        </div>
        <div class="pipeline-heading-actions">
          <span class="checked-at">更新于 {{ formatTime(status?.checkedAt) }}</span>
          <div class="head-actions">
            <el-tag effect="plain" :type="pipelineHealthy ? 'success' : 'danger'">
              {{ pipelineHealthy ? '链路正常' : '链路异常' }}
            </el-tag>
            <el-button :loading="loading" @click="loadStatus">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button type="primary" class="primary-action" :loading="testing" @click="runTest">
              <el-icon><Promotion /></el-icon>
              写入测试日志
            </el-button>
          </div>
        </div>
      </div>

      <div class="pipeline-flow" :class="{ 'with-queue': queueEnabled }">
        <article class="pipeline-node" :class="nodeClass(status?.gateway)">
          <div class="node-top">
            <span class="node-icon gateway"><el-icon><Connection /></el-icon></span>
            <span class="status-dot"></span>
          </div>
          <strong>Log Gateway</strong>
          <p>Agent 鉴权、限流与批次确认</p>
          <dl>
            <div><dt>接收日志</dt><dd>{{ number(status?.gateway.acceptedRecords) }}</dd></div>
            <div><dt>拒绝批次</dt><dd>{{ number(status?.gateway.rejectedBatches) }}</dd></div>
          </dl>
          <small>{{ status?.gateway.reachable ? '服务可访问' : shortError(status?.gateway.lastError) }}</small>
        </article>

        <div class="flow-line" :class="{ active: status?.gateway.reachable && (queueEnabled ? status?.queue?.reachable : status?.writer.reachable) }">
          <span></span><el-icon><ArrowRight /></el-icon>
        </div>

        <template v-if="queueEnabled">
          <article class="pipeline-node" :class="queueNodeClass">
            <div class="node-top">
              <span class="node-icon queue"><el-icon><Box /></el-icon></span>
              <span class="status-dot"></span>
            </div>
            <strong>Redpanda / Kafka</strong>
            <p>持久化缓冲、削峰与故障重放</p>
            <dl>
              <div><dt>消费积压</dt><dd>{{ number(status?.queue?.lag) }}</dd></div>
              <div><dt>Broker</dt><dd>{{ number(status?.queue?.brokerCount) }}</dd></div>
            </dl>
            <small>{{ status?.queue?.reachable ? status?.queue?.topic || '队列连接正常' : shortError(status?.queue?.lastError) }}</small>
          </article>

          <div class="flow-line" :class="{ active: status?.queue?.reachable && status?.writer.reachable }">
            <span></span><el-icon><ArrowRight /></el-icon>
          </div>
        </template>

        <article class="pipeline-node" :class="nodeClass(status?.writer)">
          <div class="node-top">
            <span class="node-icon writer"><el-icon><DataLine /></el-icon></span>
            <span class="status-dot"></span>
          </div>
          <strong>Log Writer</strong>
          <p>有界队列、重试、去重与落库</p>
          <dl>
            <div><dt>写入日志</dt><dd>{{ number(status?.writer.acceptedRecords) }}</dd></div>
            <div><dt>队列</dt><dd>{{ status?.writer.queueDepth || 0 }}/{{ status?.writer.queueCapacity || 0 }}</dd></div>
          </dl>
          <small>{{ status?.writer.reachable ? '服务可访问' : shortError(status?.writer.lastError) }}</small>
        </article>

        <div class="flow-line" :class="{ active: status?.writer.reachable && status?.storage.reachable }">
          <span></span><el-icon><ArrowRight /></el-icon>
        </div>

        <article class="pipeline-node" :class="storageNodeClass">
          <div class="node-top">
            <span class="node-icon storage"><el-icon><Coin /></el-icon></span>
            <span class="status-dot"></span>
          </div>
          <strong>ClickHouse</strong>
          <p>原始日志、聚合指标与 TTL</p>
          <dl>
            <div><dt>存储名称</dt><dd class="ellipsis">{{ status?.storage.name || '未配置' }}</dd></div>
            <div><dt>初始化</dt><dd>{{ status?.storage.initializedAt ? '已完成' : '未完成' }}</dd></div>
          </dl>
          <small>{{ storageStatusText }}</small>
        </article>
      </div>
    </section>

    <div class="detail-grid">
      <section class="panel metric-panel">
        <div class="panel-heading compact"><h3>运行指标</h3></div>
        <div class="metric-grid">
          <div><span>接收批次</span><strong>{{ number(status?.gateway.acceptedBatches) }}</strong></div>
          <div><span>接收日志</span><strong>{{ number(status?.gateway.acceptedRecords) }}</strong></div>
          <div><span>重复批次</span><strong>{{ number(status?.writer.duplicateBatches) }}</strong></div>
          <div><span>失败批次</span><strong :class="{ danger: (status?.writer.failedBatches || 0) > 0 }">{{ number(status?.writer.failedBatches) }}</strong></div>
          <div><span>队列积压</span><strong :class="{ danger: (status?.queue?.lag || 0) > 0 }">{{ number(status?.queue?.lag) }}</strong></div>
          <div><span>死信批次</span><strong :class="{ danger: (status?.queue?.deadletterBatches || 0) > 0 }">{{ number(status?.queue?.deadletterBatches) }}</strong></div>
        </div>
      </section>

      <section class="panel runtime-panel">
        <div class="panel-heading compact"><h3>运行信息</h3></div>
        <div class="runtime-list">
          <div><span>传输模式</span><el-tag size="small" type="info">{{ queueModeText }}</el-tag></div>
          <div><span>Agent 接入地址</span><strong :title="status?.publicGatewayUrl">{{ status?.publicGatewayUrl || '-' }}</strong></div>
          <div><span>Gateway 实例</span><strong>{{ status?.gateway.instanceId || '-' }}</strong></div>
          <div><span>Writer 实例</span><strong>{{ status?.writer.instanceId || '-' }}</strong></div>
          <div v-if="queueEnabled"><span>Consumer Group</span><strong>{{ status?.queue?.consumerGroup || '-' }}</strong></div>
          <div v-if="queueEnabled"><span>发布延迟</span><strong>{{ latency(status?.gateway.publishLatencyMs) }}</strong></div>
          <div><span>写入延迟</span><strong>{{ latency(status?.writer.writeLatencyMs) }}</strong></div>
          <div><span>Gateway 运行时长</span><strong>{{ duration(status?.gateway.uptimeSeconds) }}</strong></div>
          <div><span>Writer 运行时长</span><strong>{{ duration(status?.writer.uptimeSeconds) }}</strong></div>
          <div><span>最近成功写入</span><strong>{{ formatTime(status?.writer.lastSuccessAt) }}</strong></div>
        </div>
      </section>

	  <section class="panel agent-panel">
		<div class="panel-heading compact agent-heading">
		  <div>
			<h3>OpsHub Log Agent</h3>
			<p>当前内置采集内核版本 v0.3.0，支持主机与 Kubernetes 节点模式</p>
		  </div>
		  <el-button @click="configVisible = true">
			<el-icon><DocumentCopy /></el-icon>
			配置模板
		  </el-button>
		</div>
		<div class="capability-list">
		  <div><span class="capability-icon file"><el-icon><Files /></el-icon></span><p><strong>文件跟踪</strong><small>Glob、inode 与 offset checkpoint</small></p><el-tag size="small" type="success">已就绪</el-tag></div>
		  <div><span class="capability-icon rotate"><el-icon><RefreshRight /></el-icon></span><p><strong>日志轮转</strong><small>rename、copytruncate 与内容指纹</small></p><el-tag size="small" type="success">已就绪</el-tag></div>
		  <div><span class="capability-icon parser"><el-icon><Operation /></el-icon></span><p><strong>解析合并</strong><small>Raw、JSON、Regex 与多行堆栈</small></p><el-tag size="small" type="success">已就绪</el-tag></div>
		  <div><span class="capability-icon wal"><el-icon><Lock /></el-icon></span><p><strong>可靠传输</strong><small>有界 WAL、ACK 清理与断线重放</small></p><el-tag size="small" type="success">已就绪</el-tag></div>
		</div>
	  </section>
    </div>

    <section class="panel readiness-panel">
      <div class="panel-heading compact readiness-heading">
        <div>
          <h3>生产就绪检查</h3>
          <p>每次刷新都会实时检查数据面、存储、Agent 接入地址和关键生产配置</p>
        </div>
        <div class="readiness-summary">
          <el-tag size="small" type="success">{{ status?.readinessSummary?.passed || 0 }} 项通过</el-tag>
          <el-tag v-if="status?.readinessSummary?.warnings" size="small" type="warning">{{ status.readinessSummary.warnings }} 项建议优化</el-tag>
          <el-tag v-if="status?.readinessSummary?.failed" size="small" type="danger">{{ status.readinessSummary.failed }} 项未通过</el-tag>
        </div>
      </div>
      <div class="readiness-list">
        <div v-for="item in status?.readiness || []" :key="item.id" class="readiness-item" :class="`is-${item.status}`">
          <span class="readiness-icon">
            <el-icon v-if="item.status === 'passed'"><CircleCheck /></el-icon>
            <el-icon v-else-if="item.status === 'warning'"><WarningFilled /></el-icon>
            <el-icon v-else><CircleClose /></el-icon>
          </span>
          <div class="readiness-content">
            <div><strong>{{ item.title }}</strong><el-tag size="small" :type="readinessType(item.status)">{{ readinessText(item.status) }}</el-tag></div>
            <p :title="item.description">{{ item.description }}</p>
            <small v-if="item.recommendation">建议：{{ item.recommendation }}</small>
          </div>
        </div>
      </div>
    </section>

    <el-alert
      v-if="lastError"
      class="error-alert"
      type="error"
      :closable="false"
      show-icon
      title="采集链路存在异常"
      :description="lastError"
    />

	<el-dialog v-model="configVisible" title="主机 Agent 日志采集配置" width="min(760px, calc(100vw - 32px))" class="agent-config-dialog">
	  <div class="config-toolbar">
		<span>保存到 <code>/etc/opshub-agent/agent.json</code></span>
		<div>
		  <el-button @click="copyConfig"><el-icon><DocumentCopy /></el-icon>复制</el-button>
		  <el-button type="primary" class="primary-action" @click="downloadConfig"><el-icon><Download /></el-icon>下载</el-button>
		</div>
	  </div>
	  <pre class="config-preview"><code>{{ agentConfigTemplate }}</code></pre>
	</el-dialog>
      </el-tab-pane>
      <el-tab-pane label="采集策略" name="policies" lazy>
        <CollectionPolicyPanel :preset-host-id="Number(route.query.hostId || 0)" :preset-cluster-id="Number(route.query.clusterId || 0)" @open-instances="openInstances" />
      </el-tab-pane>
      <el-tab-pane label="采集实例" name="instances" lazy>
        <CollectorInstancePanel :preset-policy-id="selectedInstancePolicyId" />
      </el-tab-pane>
      <el-tab-pane label="发布记录" name="revisions" lazy>
        <PolicyRevisionPanel />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight, Box, CircleCheck, CircleClose, Coin, Connection, DataLine, DocumentCopy, Download, Files, Lock, Operation, Promotion, Refresh, RefreshRight, WarningFilled } from '@element-plus/icons-vue'
import { getLogIngestStatus, testLogIngest, type LogIngestComponentStatus, type LogIngestStatus } from '@/api/logcenter'
import CollectionPolicyPanel from './components/CollectionPolicyPanel.vue'
import CollectorInstancePanel from './components/CollectorInstancePanel.vue'
import PolicyRevisionPanel from './components/PolicyRevisionPanel.vue'

const route = useRoute()
const requestedTab = String(route.query.tab || 'pipeline')
const activeTab = ref(['pipeline', 'policies', 'instances', 'revisions'].includes(requestedTab) ? requestedTab : 'pipeline')
const loading = ref(false)
const testing = ref(false)
const configVisible = ref(false)
const selectedInstancePolicyId = ref<number>()
const status = ref<LogIngestStatus>()
let timer: ReturnType<typeof setInterval> | undefined

const openInstances = (policyId?: number) => {
  selectedInstancePolicyId.value = policyId
  activeTab.value = 'instances'
}

const agentConfigTemplate = JSON.stringify({
  serverUrl: 'https://opshub.example.com',
  agentId: '平台注册后自动生成',
  agentToken: '平台注册后自动生成',
  hostId: 1,
  interval: 30,
  logMetricsAddress: '127.0.0.1:19877',
  logCollection: {
    enabled: true,
    gatewayUrl: 'http://opshub.example.com:19880',
    gatewayToken: '请填写 OPSHUB_LOG_INGEST_TOKEN',
    stateDir: '/var/lib/opshub-agent/logs',
    scanIntervalSeconds: 1,
    batchSize: 500,
    flushIntervalSeconds: 2,
    maxWalBytes: 2147483648,
    sources: [{
      id: 'application-log',
      paths: ['/data/apps/*/logs/*.log'],
      excludePaths: ['*.gz', '*.tmp'],
      readFrom: 'latest',
      environment: 'production',
      service: 'application',
      maxLineBytes: 262144,
      parser: { type: 'raw' },
      multiline: { enabled: true, preset: 'java', maxLines: 500, maxBytes: 1048576, flushSeconds: 2 },
    }],
  },
}, null, 2)

const queueEnabled = computed(() => ['kafka', 'redpanda'].includes(String(status.value?.mode || '').toLowerCase()))
const pipelineHealthy = computed(() => Boolean(
  status.value?.gateway.reachable
  && (!queueEnabled.value || status.value?.queue?.reachable)
  && status.value?.writer.reachable
  && status.value?.storage.reachable
  && status.value?.storage.initializedAt,
))
const queueModeText = computed(() => queueEnabled.value ? 'Redpanda / Kafka' : 'ClickHouse 直写')
const pipelineDescription = computed(() => queueEnabled.value
  ? '日志经过鉴权与限流后进入持久化队列，由可水平扩展的 Writer 消费并批量写入 ClickHouse'
  : '当前使用直写模式，日志经过鉴权、限流与批量写入后进入内置日志库')
const queueNodeClass = computed(() => status.value?.queue?.reachable ? 'healthy' : 'unhealthy')
const storageNodeClass = computed(() => status.value?.storage.reachable && status.value?.storage.initializedAt ? 'healthy' : 'unhealthy')
const storageStatusText = computed(() => {
  if (!status.value?.storage.id) return '尚未配置内置日志存储'
  if (status.value.storage.reachable) return '存储连接正常'
  return shortError(status.value.storage.lastError || status.value.storage.status)
})
const lastError = computed(() => (
  status.value?.gateway.lastError || status.value?.queue?.lastError || status.value?.writer.lastError || status.value?.storage.lastError || ''
))

const loadStatus = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    status.value = await getLogIngestStatus() as any
  } finally {
    loading.value = false
  }
}

const runTest = async () => {
  testing.value = true
  try {
    await testLogIngest({ message: `OpsHub 采集链路测试 ${new Date().toLocaleString()}`, level: 'INFO' })
    ElMessage.success('测试日志已写入 ClickHouse，可前往日志查询查看')
    await loadStatus(true)
  } finally {
    testing.value = false
  }
}

const copyConfig = async () => {
  await navigator.clipboard.writeText(agentConfigTemplate)
  ElMessage.success('Agent 配置已复制')
}

const downloadConfig = () => {
  const blob = new Blob([agentConfigTemplate], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'opshub-agent.json'
  link.click()
  URL.revokeObjectURL(url)
}

const nodeClass = (item?: LogIngestComponentStatus) => item?.reachable && item.status === 'healthy' ? 'healthy' : 'unhealthy'
const number = (value?: number) => Number(value || 0).toLocaleString()
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const shortError = (value?: string) => {
  if (!value) return '服务不可访问'
  return value.length > 76 ? `${value.slice(0, 76)}...` : value
}
const duration = (seconds?: number) => {
  const value = Number(seconds || 0)
  if (value < 60) return `${value} 秒`
  if (value < 3600) return `${Math.floor(value / 60)} 分钟`
  if (value < 86400) return `${Math.floor(value / 3600)} 小时`
  return `${Math.floor(value / 86400)} 天 ${Math.floor((value % 86400) / 3600)} 小时`
}
const latency = (milliseconds?: number) => `${Number(milliseconds || 0).toFixed(1)} ms`
const readinessText = (value: string) => ({ passed: '通过', warning: '建议优化', failed: '未通过' }[value] || value)
const readinessType = (value: string) => value === 'passed' ? 'success' : value === 'warning' ? 'warning' : 'danger'

onMounted(async () => {
  await loadStatus()
  timer = setInterval(() => loadStatus(true), 5000)
})
onBeforeUnmount(() => timer && clearInterval(timer))
</script>

<style scoped>
.collector-tabs :deep(.el-tabs__header) { margin: 0 0 16px; padding: 0 4px; }
.collector-tabs :deep(.el-tabs__item) { height: 42px; color: #667085; font-size: 14px; }
.collector-tabs :deep(.el-tabs__item.is-active) { color: #111827; font-weight: 650; }
.collector-tabs :deep(.el-tabs__active-bar) { height: 2px; background: #111827; }
.collector-tabs :deep(.el-tabs__content) { overflow: visible; }
.head-actions { display: flex; align-items: center; gap: 10px; }
.pipeline-panel { padding: 20px; }
.panel-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 24px; }
.panel-heading h3 { margin: 0; color: #111827; font-size: 15px; font-weight: 650; }
.panel-heading p { margin: 6px 0 0; color: #6b7280; font-size: 13px; }
.panel-heading.compact { margin-bottom: 16px; }
.pipeline-heading-actions { display: flex; align-items: flex-end; flex-direction: column; gap: 8px; }
.checked-at { color: #9ca3af; font-size: 12px; white-space: nowrap; }
.pipeline-flow { display: grid; grid-template-columns: minmax(210px, 1fr) 76px minmax(210px, 1fr) 76px minmax(210px, 1fr); align-items: center; }
.pipeline-flow.with-queue { grid-template-columns: minmax(180px, 1fr) 48px minmax(180px, 1fr) 48px minmax(180px, 1fr) 48px minmax(180px, 1fr); }
.pipeline-node { min-width: 0; padding: 18px; border: 1px solid #e5e7eb; border-radius: 8px; background: #fff; transition: border-color .2s, box-shadow .2s; }
.pipeline-node.healthy { border-color: #bbf7d0; box-shadow: inset 0 3px 0 #22c55e; }
.pipeline-node.unhealthy { border-color: #fecaca; box-shadow: inset 0 3px 0 #ef4444; }
.node-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
.node-icon { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 7px; font-size: 18px; }
.node-icon.gateway { color: #2563eb; background: #eff6ff; }
.node-icon.writer { color: #7c3aed; background: #f5f3ff; }
.node-icon.queue { color: #0369a1; background: #f0f9ff; }
.node-icon.storage { color: #c2410c; background: #fff7ed; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: #ef4444; box-shadow: 0 0 0 4px #fef2f2; }
.healthy .status-dot { background: #22c55e; box-shadow: 0 0 0 4px #f0fdf4; }
.pipeline-node > strong { color: #111827; font-size: 15px; }
.pipeline-node > p { min-height: 20px; margin: 5px 0 16px; color: #6b7280; font-size: 12px; }
.pipeline-node dl { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin: 0 0 14px; }
.pipeline-node dl div { min-width: 0; padding: 9px 10px; background: #f8fafc; }
.pipeline-node dt { color: #9ca3af; font-size: 11px; }
.pipeline-node dd { margin: 4px 0 0; color: #1f2937; font-size: 13px; font-weight: 650; }
.pipeline-node small { display: block; overflow: hidden; color: #6b7280; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.ellipsis { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.flow-line { position: relative; display: flex; align-items: center; color: #cbd5e1; }
.flow-line span { width: 100%; border-top: 1px dashed #cbd5e1; }
.flow-line .el-icon { margin-left: -3px; }
.flow-line.active { color: #22c55e; }
.flow-line.active span { border-color: #22c55e; animation: flow 1.2s linear infinite; }
.detail-grid { display: grid; grid-template-columns: 1.3fr 1fr; gap: 16px; margin-top: 16px; }
.metric-panel, .runtime-panel { padding: 20px; }
.agent-panel { grid-column: 1 / -1; padding: 20px; }
.agent-heading { align-items: center; }
.capability-list { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.capability-list > div { display: grid; grid-template-columns: 36px minmax(0, 1fr) auto; align-items: center; gap: 10px; min-width: 0; padding: 13px 14px; border: 1px solid #eef0f3; border-radius: 7px; background: #fafafa; }
.capability-list p { min-width: 0; margin: 0; }
.capability-list strong, .capability-list small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.capability-list strong { color: #1f2937; font-size: 13px; }
.capability-list small { margin-top: 4px; color: #8b95a5; font-size: 11px; }
.capability-icon { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 7px; font-size: 17px; }
.capability-icon.file { color: #2563eb; background: #eff6ff; }
.capability-icon.rotate { color: #7c3aed; background: #f5f3ff; }
.capability-icon.parser { color: #047857; background: #ecfdf5; }
.capability-icon.wal { color: #c2410c; background: #fff7ed; }
.config-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 12px; color: #6b7280; font-size: 13px; }
.config-toolbar > div { display: flex; gap: 8px; }
.config-toolbar code { color: #111827; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
.config-preview { max-height: 520px; margin: 0; overflow: auto; padding: 16px; border: 1px solid #e5e7eb; border-radius: 7px; background: #f8fafc; color: #1f2937; font: 12px/1.65 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; white-space: pre; }
.metric-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.metric-grid div { padding: 14px; border: 1px solid #eef0f3; border-radius: 7px; background: #fafafa; }
.metric-grid span { display: block; color: #6b7280; font-size: 12px; }
.metric-grid strong { display: block; margin-top: 8px; color: #111827; font-size: 22px; font-weight: 650; }
.metric-grid strong.danger { color: #dc2626; }
.runtime-list { display: grid; gap: 13px; }
.runtime-list > div { display: flex; align-items: center; justify-content: space-between; gap: 16px; color: #6b7280; font-size: 13px; }
.runtime-list strong { overflow: hidden; color: #1f2937; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.readiness-panel { margin-top: 16px; padding: 20px; }
.readiness-heading { align-items: center; }
.readiness-summary { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
.readiness-list { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; border: 1px solid #eaecf0; background: #eaecf0; }
.readiness-item { display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 12px; min-width: 0; padding: 15px 16px; background: #fff; }
.readiness-icon { display: grid; width: 32px; height: 32px; place-items: center; background: #f0fdf4; color: #16a34a; font-size: 18px; }
.readiness-item.is-warning .readiness-icon { background: #fffbeb; color: #d97706; }
.readiness-item.is-failed .readiness-icon { background: #fef2f2; color: #dc2626; }
.readiness-content { min-width: 0; }
.readiness-content > div { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.readiness-content strong { color: #1f2937; font-size: 13px; }
.readiness-content p { overflow: hidden; margin: 5px 0 0; color: #667085; font-size: 12px; line-height: 1.5; text-overflow: ellipsis; white-space: nowrap; }
.readiness-content small { display: block; margin-top: 4px; color: #98a2b3; font-size: 11px; line-height: 1.5; }
.error-alert { margin-top: 16px; }
@keyframes flow { to { border-color: #16a34a; } }
@media (max-width: 1100px) {
  .pipeline-flow { grid-template-columns: 1fr; gap: 12px; }
  .flow-line { width: 1px; height: 24px; margin: 0 auto; }
  .flow-line span { height: 100%; border-top: 0; border-left: 1px dashed #cbd5e1; }
  .flow-line .el-icon { display: none; }
	.capability-list { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 820px) {
  .panel-heading { align-items: flex-start; flex-direction: column; }
  .pipeline-heading-actions { align-items: flex-start; width: 100%; }
  .head-actions { flex-wrap: wrap; justify-content: flex-start; }
  .detail-grid { grid-template-columns: 1fr; }
  .metric-grid { grid-template-columns: repeat(2, 1fr); }
	.capability-list { grid-template-columns: 1fr; }
	.config-toolbar { align-items: flex-start; flex-direction: column; }
  .readiness-heading { align-items: flex-start; flex-direction: column; }
  .readiness-summary { justify-content: flex-start; }
  .readiness-list { grid-template-columns: 1fr; }
}
</style>
