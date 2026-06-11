<template>
  <div class="dashboard-panel">
    <div v-if="!attached" class="not-attached">
      <el-empty description="请先选择Pod并连接到进程">
        <template #image>
          <el-icon :size="60" color="#909399"><Connection /></el-icon>
        </template>
      </el-empty>
    </div>

    <div v-else class="dashboard-content" v-loading="loading">
      <div class="toolbar">
        <el-button type="primary" :icon="Refresh" @click="loadDashboard" :loading="loading">刷新</el-button>
      </div>

      <!-- 错误提示 -->
      <el-alert
        v-if="hasConnectionError && isDataEmpty"
        title="Arthas 连接问题"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 16px"
      >
        <template #default>
          <div>{{ errorMessage }}</div>
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            <div>可能的原因：</div>
            <ul style="margin: 4px 0 0 16px; padding: 0;">
              <li v-if="errorMessage.includes('Connection refused')">Arthas telnet 服务尚未启动，请稍后重试</li>
              <li v-if="errorMessage.includes('attach')">JVM 可能不支持 Arthas attach</li>
              <li v-else>请点击「刷新」按钮重试，或检查原始输出了解详情</li>
            </ul>
          </div>
        </template>
      </el-alert>

      <!-- 线程 TOP-10 -->
      <div class="section">
        <div class="section-header">
          <h3>线程 TOP-10</h3>
        </div>
        <div class="section-body">
          <el-table :data="dashboardData.threads" stripe size="small" :header-cell-style="{ background: '#f5f7fa', color: '#606266' }" style="width: 100%">
            <el-table-column prop="id" label="ID" width="50" align="center" />
            <el-table-column prop="name" label="名称" show-overflow-tooltip />
            <el-table-column prop="group" label="Group" width="70" align="center" />
            <el-table-column prop="priority" label="优先级" width="70" align="center" />
            <el-table-column prop="state" label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tooltip :content="row.state" placement="top" :disabled="row.state?.length < 10">
                  <el-tag :type="getStateType(row.state)" size="small" effect="light" class="state-tag">{{ formatState(row.state) }}</el-tag>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column prop="cpu" label="CPU" width="60" align="right" />
            <el-table-column prop="time" label="时长" width="80" align="right" />
            <el-table-column prop="interrupted" label="中断" width="60" align="center">
              <template #default="{ row }">
                <span :class="row.interrupted ? 'text-danger' : 'text-muted'">{{ row.interrupted ? '是' : '否' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="daemon" label="守护" width="60" align="center">
              <template #default="{ row }">
                <span :class="row.daemon ? 'text-primary' : 'text-muted'">{{ row.daemon ? '是' : '否' }}</span>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="dashboardData.threads.length === 0 && !loading" description="暂无线程数据" :image-size="60" />
        </div>
      </div>

      <!-- JVM 内存 -->
      <div class="section">
        <div class="section-header">
          <h3>JVM 内存</h3>
        </div>
        <div class="section-body">
          <div class="memory-gc-container">
            <!-- 内存信息 -->
            <div class="memory-info" v-if="dashboardData.memory.length > 0">
              <div class="memory-card" v-for="mem in dashboardData.memory" :key="mem.type">
                <div class="memory-card-header">
                  <span class="memory-type">{{ mem.type }}</span>
                  <span class="memory-usage" :style="{ color: getUsageColor(calcMemoryUsage(mem)) }">{{ formatMemoryUsage(mem) }}</span>
                </div>
                <el-progress
                  :percentage="calcMemoryUsage(mem)"
                  :stroke-width="8"
                  :color="getUsageColor(calcMemoryUsage(mem))"
                  :show-text="false"
                />
                <div class="memory-detail">
                  <span>已用: <strong>{{ mem.used }}</strong></span>
                  <span>总量: <strong>{{ mem.total }}</strong></span>
                  <span v-if="mem.max && mem.max !== '-1'">最大: <strong>{{ mem.max }}</strong></span>
                </div>
              </div>
            </div>

            <!-- GC 信息 -->
            <div class="gc-info" v-if="dashboardData.gc.length > 0">
              <h4>GC 统计</h4>
              <div class="gc-cards">
                <div class="gc-card" v-for="gc in dashboardData.gc" :key="gc.name">
                  <div class="gc-name">{{ gc.name }}</div>
                  <div class="gc-stats">
                    <div class="gc-stat">
                      <span class="gc-label">次数</span>
                      <span class="gc-value">{{ gc.collectionCount }}</span>
                    </div>
                    <div class="gc-stat">
                      <span class="gc-label">耗时</span>
                      <span class="gc-value">{{ gc.collectionTime }} ms</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-if="dashboardData.memory.length === 0 && dashboardData.gc.length === 0 && !loading" description="暂无内存数据" :image-size="60" />
        </div>
      </div>

      <!-- 运行时 -->
      <div class="section">
        <div class="section-header">
          <h3>运行时</h3>
        </div>
        <div class="section-body">
          <div class="runtime-grid" v-if="dashboardData.runtime.length > 0">
            <div class="runtime-item" v-for="item in dashboardData.runtime" :key="item.name">
              <span class="runtime-name">{{ item.name }}</span>
              <span class="runtime-value">{{ item.value }}</span>
            </div>
          </div>
          <el-empty v-if="dashboardData.runtime.length === 0 && !loading" description="暂无运行时数据" :image-size="60" />
        </div>
      </div>

      <!-- 原始输出（可折叠） -->
      <el-collapse v-if="dashboardData.rawOutput" class="raw-output-collapse">
        <el-collapse-item title="原始输出" name="raw">
          <div class="output-content">
            <pre>{{ dashboardData.rawOutput }}</pre>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, reactive, computed } from 'vue'
import { Connection, Refresh } from '@element-plus/icons-vue'
import { getDashboard } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

const emit = defineEmits(['switch-tab', 'connection-error'])

const loading = ref(false)

interface ThreadInfo {
  id: string
  name: string
  group: string
  priority: string
  state: string
  cpu: string
  deltaTime: string
  time: string
  interrupted: boolean
  daemon: boolean
}

interface MemoryInfo {
  type: string
  used: string
  total: string
  max: string
  usage: string
}

interface GCInfo {
  name: string
  collectionCount: number
  collectionTime: number
}

interface RuntimeInfo {
  name: string
  value: string
}

interface DashboardData {
  threads: ThreadInfo[]
  memory: MemoryInfo[]
  gc: GCInfo[]
  runtime: RuntimeInfo[]
  rawOutput: string
}

const dashboardData = reactive<DashboardData>({
  threads: [],
  memory: [],
  gc: [],
  runtime: [],
  rawOutput: ''
})

// 检测是否有连接错误
const hasConnectionError = computed(() => {
  const output = dashboardData.rawOutput || ''
  return output.includes('Connection refused') ||
         output.includes('Connect.*error') ||
         output.includes('[ERROR]') ||
         output.includes('Unable to attach') ||
         output.includes('Failed to execute')
})

// 检测数据是否为空
const isDataEmpty = computed(() => {
  return dashboardData.threads.length === 0 &&
         dashboardData.memory.length === 0 &&
         dashboardData.gc.length === 0 &&
         dashboardData.runtime.length === 0
})

// 提取错误信息
const errorMessage = computed(() => {
  const output = dashboardData.rawOutput || ''
  // 尝试提取 [ERROR] 行
  const errorMatch = output.match(/\[ERROR\][^\n]+/)
  if (errorMatch) {
    return errorMatch[0]
  }
  // 尝试提取 Connection refused
  if (output.includes('Connection refused')) {
    return 'Arthas telnet 连接被拒绝 (Connection refused)'
  }
  // 其他错误
  if (output.includes('Unable to attach')) {
    return 'JVM 不支持 Arthas attach'
  }
  return '获取数据失败，请查看原始输出了解详情'
})

const getStateType = (state: string) => {
  const types: Record<string, string> = {
    'RUNNABLE': 'success',
    'BLOCKED': 'danger',
    'WAITING': 'warning',
    'TIMED_WAITING': 'info',
    'NEW': 'info',
    'TERMINATED': ''
  }
  return types[state] || 'info'
}

// 格式化状态显示（缩短过长的状态名）
const formatState = (state: string): string => {
  if (!state) return '-'
  const stateMap: Record<string, string> = {
    'TIMED_WAITING': 'T_WAIT',
    'WAITING': 'WAITING',
    'RUNNABLE': 'RUNNING',
    'BLOCKED': 'BLOCKED',
    'TERMINATED': 'TERM',
    'NEW': 'NEW'
  }
  return stateMap[state] || state
}

// 解析内存大小字符串为数字（单位：MB）
const parseMemorySize = (sizeStr: string): number => {
  if (!sizeStr || sizeStr === '-1') return 0
  const match = sizeStr.match(/([\d.]+)\s*([KMGT]?)/i)
  if (!match) return 0

  const value = parseFloat(match[1])
  const unit = (match[2] || '').toUpperCase()

  switch (unit) {
    case 'K': return value / 1024
    case 'G': return value * 1024
    case 'T': return value * 1024 * 1024
    default: return value // 默认 MB
  }
}

// 计算内存使用率
const calcMemoryUsage = (mem: MemoryInfo): number => {
  // 优先使用 usage 字段
  if (mem.usage && mem.usage !== '-') {
    const match = mem.usage.match(/[\d.]+/)
    if (match) {
      return Math.min(parseFloat(match[0]), 100)
    }
  }

  // 如果 usage 为空，根据 used 和 total/max 计算
  const used = parseMemorySize(mem.used)
  // 优先使用 max（如果有效），否则使用 total
  const total = mem.max && mem.max !== '-1' ? parseMemorySize(mem.max) : parseMemorySize(mem.total)

  if (total > 0) {
    return Math.min(Math.round((used / total) * 100 * 10) / 10, 100)
  }
  return 0
}

// 格式化内存使用率显示
const formatMemoryUsage = (mem: MemoryInfo): string => {
  const usage = calcMemoryUsage(mem)
  return usage > 0 ? `${usage}%` : '-'
}

const parseUsage = (usage: string): number => {
  if (!usage) return 0
  const match = usage.match(/[\d.]+/)
  return match ? Math.min(parseFloat(match[0]), 100) : 0
}

const getUsageColor = (percentage: number) => {
  if (percentage < 50) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const loadDashboard = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loading.value = true
  try {
    const res = await getDashboard({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    if (typeof res === 'object' && res !== null) {
      dashboardData.threads = res.threads || []
      dashboardData.memory = res.memory || []
      dashboardData.gc = res.gc || []
      dashboardData.runtime = res.runtime || []
      dashboardData.rawOutput = res.rawOutput || ''
    } else if (typeof res === 'string') {
      dashboardData.rawOutput = res
    }

    if (hasConnectionError.value && isDataEmpty.value) {
      emit('connection-error', errorMessage.value)
    }
  } catch (error: any) {
    const message = '加载Dashboard失败: ' + (error.message || '未知错误')
    emit('connection-error', message)
  } finally {
    loading.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadDashboard()
  } else {
    dashboardData.threads = []
    dashboardData.memory = []
    dashboardData.gc = []
    dashboardData.runtime = []
    dashboardData.rawOutput = ''
  }
})
</script>

<style scoped>
.dashboard-panel {
  min-height: 400px;
}

.not-attached {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.toolbar {
  margin-bottom: 16px;
}

.dashboard-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  overflow: hidden;
}

.section-header {
  padding: 12px 16px;
  background: #fafafa;
  border-bottom: 1px solid #ebeef5;
}

.section-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.section-body {
  padding: 16px;
}

/* 线程表格样式 */
.text-danger { color: #f56c6c; }
.text-primary { color: #409eff; }
.text-muted { color: #c0c4cc; }

.state-tag {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 内存和GC容器 */
.memory-gc-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 内存卡片网格 */
.memory-info {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.memory-card {
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.memory-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.memory-type {
  font-size: 13px;
  font-weight: 600;
  color: #495057;
}

.memory-usage {
  font-size: 14px;
  font-weight: 700;
}

.memory-detail {
  display: flex;
  gap: 16px;
  margin-top: 8px;
  font-size: 12px;
  color: #6c757d;
}

.memory-detail strong {
  color: #495057;
}

/* GC 信息 */
.gc-info h4 {
  margin: 0 0 12px 0;
  font-size: 13px;
  font-weight: 600;
  color: #606266;
}

.gc-cards {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.gc-card {
  flex: 1;
  min-width: 180px;
  max-width: 250px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 6px;
  color: #fff;
}

.gc-name {
  font-size: 12px;
  font-weight: 500;
  opacity: 0.9;
  margin-bottom: 8px;
}

.gc-stats {
  display: flex;
  justify-content: space-between;
}

.gc-stat {
  display: flex;
  flex-direction: column;
}

.gc-label {
  font-size: 11px;
  opacity: 0.8;
}

.gc-value {
  font-size: 16px;
  font-weight: 700;
}

/* 运行时网格 */
.runtime-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 8px;
}

.runtime-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
  font-size: 13px;
}

.runtime-name {
  color: #6c757d;
  font-weight: 500;
}

.runtime-value {
  color: #212529;
  font-weight: 600;
}

/* 原始输出 */
.raw-output-collapse {
  margin-top: 8px;
}

.output-content {
  padding: 12px;
  background: #1e1e1e;
  max-height: 300px;
  overflow: auto;
  border-radius: 4px;
}

.output-content pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 11px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
