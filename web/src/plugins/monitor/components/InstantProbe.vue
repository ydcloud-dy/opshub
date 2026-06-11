<template>
  <div class="instant-probe-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><VideoPlay /></el-icon>
        </div>
        <div>
          <h2 class="page-title">即时拨测</h2>
          <p class="page-subtitle">不创建任务，直接对 HTTP、ICMP、TCP、SSL 端点执行一次拨测</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="resetForm">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <div class="probe-console">
      <div class="protocol-tabs">
        <button
          v-for="item in protocolOptions"
          :key="item.value"
          type="button"
          class="protocol-tab"
          :class="{ active: form.protocol === item.value }"
          @click="selectProtocol(item.value)"
        >
          <span>{{ item.label }}</span>
          <em>{{ item.desc }}</em>
        </button>
      </div>

      <div class="probe-form-panel">
        <div class="endpoint-row">
          <el-input
            v-model="form.endpoint"
            :placeholder="getEndpointPlaceholder(form.protocol)"
            clearable
            size="large"
            @keyup.enter="handleRun"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" size="large" :loading="running" @click="handleRun">
            <el-icon><VideoPlay /></el-icon>
            拨测一下
          </el-button>
        </div>

        <div class="advanced-panel">
          <div class="advanced-title">
            <el-icon><Setting /></el-icon>
            <span>高级选项</span>
          </div>
          <div class="form-grid" :class="{ three: form.protocol === 'icmp' }">
            <label>
              <span>超时时间</span>
              <div class="suffix-input">
                <el-input-number v-model="form.timeoutSeconds" :min="1" :max="120" controls-position="right" />
                <em>秒</em>
              </div>
            </label>
            <label v-if="form.protocol === 'http'">
              <span>请求方法</span>
              <el-select v-model="form.method">
                <el-option label="GET" value="GET" />
                <el-option label="POST" value="POST" />
                <el-option label="PUT" value="PUT" />
                <el-option label="HEAD" value="HEAD" />
              </el-select>
            </label>
            <label v-if="form.protocol === 'icmp'">
              <span>请求次数</span>
              <el-input-number v-model="form.icmpCount" :min="1" :max="10" controls-position="right" />
            </label>
            <label v-if="form.protocol === 'icmp'">
              <span>请求间隔</span>
              <div class="suffix-input">
                <el-input-number v-model="form.icmpIntervalMs" :min="100" :max="10000" :step="100" controls-position="right" />
                <em>ms</em>
              </div>
            </label>
          </div>
          <div v-if="form.protocol === 'http'" class="http-extra">
            <el-input v-model="form.headers" placeholder='请求头 JSON，如 {"X-App":"opshub"}' />
            <el-input
              v-if="form.method !== 'GET' && form.method !== 'HEAD'"
              v-model="form.body"
              type="textarea"
              :rows="3"
              placeholder='请求体 JSON，如 {"ping":"ok"}'
            />
          </div>
        </div>
      </div>
    </div>

    <div class="result-section">
      <div class="section-header">
        <div>
          <h3 class="section-title">拨测结果</h3>
          <p class="section-subtitle">{{ resultSummaryText }}</p>
        </div>
        <el-tag v-if="summary" :type="summary.success ? 'success' : 'danger'" effect="light">
          {{ summary.success ? '成功' : '异常' }}
        </el-tag>
      </div>

      <div v-if="summary" class="result-cards">
        <div class="result-card">
          <span>总耗时</span>
          <strong>{{ summary.durationMs }}ms</strong>
        </div>
        <div class="result-card">
          <span>成功端点</span>
          <strong>{{ successCount }}</strong>
        </div>
        <div class="result-card">
          <span>失败端点</span>
          <strong>{{ failureCount }}</strong>
        </div>
      </div>

      <el-table
        :data="summary?.results || []"
        v-loading="running"
        class="modern-table"
        :header-cell-style="tableHeaderStyle"
        empty-text="输入端点后点击拨测一下"
      >
        <el-table-column label="端点" prop="endpoint" min-width="280" show-overflow-tooltip />
        <el-table-column label="协议" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="getProtocolTag(row.protocol)" effect="light">{{ getProtocolName(row.protocol) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" effect="light">{{ row.success ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="110" align="right">
          <template #default="{ row }">{{ row.durationMs }}ms</template>
        </el-table-column>
        <el-table-column label="状态码" width="100" align="center">
          <template #default="{ row }">{{ row.statusCode || '-' }}</template>
        </el-table-column>
        <el-table-column label="证书到期时间" width="270" min-width="270">
          <template #default="{ row }">
            <span class="certificate-expiry">
              {{ row.sslExpireAt ? `${formatDateTime(row.sslExpireAt)}（剩余 ${row.sslDaysLeft} 天）` : '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="结果" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">{{ row.message || row.error || '-' }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  RefreshLeft,
  Search,
  Setting,
  VideoPlay
} from '@element-plus/icons-vue'
import {
  runMonitorInstantProbe,
  type MonitorProbeRunSummary,
  type ProbeProtocol
} from '@/api/monitor-datasource'

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
const running = ref(false)
const summary = ref<MonitorProbeRunSummary>()

const form = reactive({
  protocol: 'http' as ProbeProtocol,
  endpoint: '',
  method: 'GET',
  headers: '',
  body: '',
  timeoutSeconds: 5,
  icmpCount: 3,
  icmpIntervalMs: 1000
})

const protocolOptions: Array<{ label: string; value: ProbeProtocol; desc: string }> = [
  { label: 'HTTP', value: 'http', desc: 'URL 可用性' },
  { label: 'ICMP', value: 'icmp', desc: '主机连通性' },
  { label: 'TCP', value: 'tcp', desc: '端口连通性' },
  { label: 'SSL', value: 'ssl', desc: '证书有效性' }
]

const successCount = computed(() => summary.value?.results.filter(item => item.success).length || 0)
const failureCount = computed(() => summary.value?.results.filter(item => !item.success).length || 0)
const resultSummaryText = computed(() => {
  if (!summary.value) return '最近一次即时拨测结果会显示在这里'
  return `${formatDateTime(new Date().toISOString())} · ${summary.value.results.length} 个端点`
})

const selectProtocol = (protocol: ProbeProtocol) => {
  form.protocol = protocol
  form.endpoint = ''
  form.method = 'GET'
  form.headers = ''
  form.body = ''
  summary.value = undefined
}

const resetForm = () => {
  Object.assign(form, {
    protocol: 'http',
    endpoint: '',
    method: 'GET',
    headers: '',
    body: '',
    timeoutSeconds: 5,
    icmpCount: 3,
    icmpIntervalMs: 1000
  })
  summary.value = undefined
}

const handleRun = async () => {
  if (!form.endpoint.trim()) {
    ElMessage.warning('请输入拨测端点')
    return
  }
  running.value = true
  try {
    summary.value = await runMonitorInstantProbe({ ...form })
  } finally {
    running.value = false
  }
}

const getEndpointPlaceholder = (protocol: ProbeProtocol) => {
  const map: Record<ProbeProtocol, string> = {
    http: 'https://example.com/health',
    icmp: 'github.com 或 8.8.8.8',
    tcp: '127.0.0.1:80',
    ssl: 'example.com 或 example.com:443'
  }
  return map[protocol]
}

const getProtocolName = (protocol?: string) => {
  const map: Record<string, string> = { http: 'HTTP', icmp: 'ICMP', tcp: 'TCP', ssl: 'SSL' }
  return protocol ? map[protocol] || protocol.toUpperCase() : '-'
}

const getProtocolTag = (protocol?: string) => {
  const map: Record<string, string> = { http: 'primary', icmp: 'success', tcp: 'warning', ssl: 'danger' }
  return protocol ? map[protocol] || 'info' : 'info'
}

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
</script>

<style scoped>
.instant-probe-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header,
.probe-console,
.result-section {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
}

.page-title-group,
.header-actions,
.endpoint-row,
.advanced-title,
.section-header {
  display: flex;
  align-items: center;
}

.page-title-group {
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 22px;
}

.page-title {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.3;
}

.page-subtitle,
.section-subtitle {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.probe-console {
  overflow: hidden;
}

.protocol-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-bottom: 1px solid #edf1f7;
}

.protocol-tab {
  min-height: 76px;
  padding: 14px 18px;
  border: 0;
  border-right: 1px solid #edf1f7;
  background: #fbfcfe;
  color: #344054;
  text-align: left;
  cursor: pointer;
}

.protocol-tab:last-child {
  border-right: 0;
}

.protocol-tab span,
.protocol-tab em {
  display: block;
}

.protocol-tab span {
  color: #111827;
  font-size: 15px;
  font-weight: 800;
}

.protocol-tab em {
  margin-top: 6px;
  color: #667085;
  font-size: 12px;
  font-style: normal;
}

.protocol-tab.active {
  background: #eef4ff;
  box-shadow: inset 0 -3px 0 #2563eb;
}

.protocol-tab.active span {
  color: #1d4ed8;
}

.probe-form-panel {
  padding: 18px;
}

.endpoint-row {
  gap: 12px;
}

.endpoint-row .el-input {
  flex: 1;
}

.advanced-panel {
  margin-top: 16px;
  padding: 14px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.advanced-title {
  gap: 8px;
  margin-bottom: 14px;
  color: #111827;
  font-size: 14px;
  font-weight: 750;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-grid.three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.form-grid label > span {
  color: #344054;
  font-size: 13px;
  font-weight: 650;
}

.suffix-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.suffix-input .el-input-number {
  flex: 1;
}

.suffix-input em {
  color: #667085;
  font-style: normal;
  font-size: 13px;
}

.http-extra {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
  margin-top: 14px;
}

.result-section {
  overflow: hidden;
}

.section-header {
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid #edf1f7;
}

.section-title {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
}

.result-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #edf1f7;
}

.result-card {
  min-height: 74px;
  padding: 14px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.result-card span {
  color: #667085;
  font-size: 13px;
}

.result-card strong {
  display: block;
  margin-top: 6px;
  color: #111827;
  font-size: 24px;
  font-weight: 800;
}

.modern-table {
  width: 100%;
}

.certificate-expiry {
  display: inline-block;
  min-width: 245px;
  white-space: nowrap;
  color: #344054;
}

@media (max-width: 900px) {
  .page-header,
  .endpoint-row {
    flex-direction: column;
    align-items: stretch;
  }

  .protocol-tabs,
  .form-grid,
  .form-grid.three,
  .result-cards {
    grid-template-columns: 1fr;
  }
}
</style>
