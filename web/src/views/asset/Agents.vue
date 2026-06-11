<template>
  <div class="agent-page-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon agent-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Agent管理</h2>
          <p class="page-subtitle">统一管理主机采集Agent，支持一键安装、批量安装、状态追踪和绑定清理</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" :disabled="selectedInstallableHosts.length === 0" :loading="batchInstalling" @click="handleBatchInstall">
          <el-icon><Lightning /></el-icon>
          批量一键安装
        </el-button>
        <el-button @click="loadAgents">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="agent-stats">
      <div class="agent-stat-card">
        <span class="stat-label">当前列表</span>
        <strong>{{ pagination.total }}</strong>
      </div>
      <div class="agent-stat-card online">
        <span class="stat-label">Agent在线</span>
        <strong>{{ pageStats.online }}</strong>
      </div>
      <div class="agent-stat-card pending">
        <span class="stat-label">待安装</span>
        <strong>{{ pageStats.pending }}</strong>
      </div>
      <div class="agent-stat-card muted">
        <span class="stat-label">未安装</span>
        <strong>{{ pageStats.uninstalled }}</strong>
      </div>
      <div class="agent-stat-card ssh-only">
        <span class="stat-label">仅SSH采集</span>
        <strong>{{ pageStats.sshOnly }}</strong>
      </div>
    </div>

    <div class="install-settings" :class="{ warning: installServerUrlWarning }">
      <div class="setting-copy">
        <div class="setting-title">Agent访问地址</div>
        <div class="setting-desc">
          这个地址会写入远端 Agent 配置，必须是目标主机能访问到的 OpsHub 或 Agent Gateway 地址，不能用 localhost。
        </div>
      </div>
      <el-input
        v-model="installServerUrl"
        class="server-url-input"
        placeholder="例如：https://agent-gateway.example.com 或 http://10.122.24.10:9876"
        clearable
        @blur="persistInstallServerUrl"
      />
    </div>
    <el-alert
      v-if="installServerUrlWarning"
      class="server-url-alert"
      type="warning"
      :closable="false"
      show-icon
      :title="installServerUrlWarning"
    />

    <div class="filter-bar">
      <div class="filter-inputs">
        <el-input
          v-model="filters.keyword"
          placeholder="搜索主机名称 / IP"
          clearable
          class="filter-input"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select v-model="filters.collectMode" placeholder="采集方式" clearable class="filter-input" @change="handleSearch">
          <el-option label="Agent采集" value="agent" />
          <el-option label="待安装" value="agent_pending" />
          <el-option label="SSH采集" value="ssh" />
        </el-select>
        <el-select v-model="filters.agentStatus" placeholder="Agent状态" clearable class="filter-input" @change="handleSearch">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="待安装" value="pending" />
          <el-option label="未安装" value="uninstalled" />
        </el-select>
      </div>
      <div class="filter-actions">
        <el-button @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
        <el-button type="primary" @click="handleSearch">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table
        :data="agentHosts"
        v-loading="loading"
        class="modern-table agent-table"
        @selection-change="selectedHosts = $event"
      >
        <el-table-column type="selection" width="48" :selectable="isAgentInstallable" />
        <el-table-column label="主机" min-width="230">
          <template #default="{ row }">
            <div class="host-cell">
              <div class="host-avatar" :class="{ online: row.agentStatus === 'online' }">
                <el-icon><Monitor /></el-icon>
              </div>
              <div>
                <div class="host-name">{{ row.name }}</div>
                <div class="host-meta">{{ row.ip }}:{{ row.port }}</div>
                <el-tag v-if="!isAgentSupported(row)" class="ssh-only-tag" size="small" type="info">
                  云主机仅SSH采集
                </el-tag>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="采集方式" width="130" align="center">
          <template #default="{ row }">
            <el-tag :type="getCollectModeType(row.collectMode)" size="small">
              {{ row.collectModeText || 'SSH采集' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Agent状态" width="140" align="center">
          <template #default="{ row }">
            <div class="agent-status-cell">
              <span class="status-dot" :class="row.agentStatus || 'uninstalled'"></span>
              <span>{{ row.agentStatusText || '未安装' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="版本" width="110" align="center">
          <template #default="{ row }">
            <span>{{ row.agentVersion || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后心跳" min-width="165">
          <template #default="{ row }">
            <span>{{ row.agentLastSeen || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后采集" min-width="165">
          <template #default="{ row }">
            <span>{{ row.agentLastCollectAt || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="资源概览" min-width="210">
          <template #default="{ row }">
            <div class="resource-mini">
              <span>CPU {{ formatPercent(row.cpuUsage) }}</span>
              <span>内存 {{ formatPercent(row.memoryUsage) }}</span>
              <span>磁盘 {{ formatPercent(row.diskUsage) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip :content="getAgentDisabledReason(row)" :disabled="isAgentInstallable(row)" placement="top">
                <span>
                  <el-button
                    class="install-action-btn"
                    size="small"
                    type="primary"
                    :disabled="!isAgentInstallable(row)"
                    :loading="isInstalling(row.id)"
                    @click="handleInstall(row)"
                  >
                    一键安装
                  </el-button>
                </span>
              </el-tooltip>
              <el-tooltip :content="getAgentDisabledReason(row)" :disabled="isAgentInstallable(row)" placement="top">
                <span>
                  <el-button
                    class="manual-action-btn"
                    size="small"
                    :disabled="!isAgentInstallable(row)"
                    @click="handleManualCommand(row)"
                  >
                    安装命令
                  </el-button>
                </span>
              </el-tooltip>
              <el-button
                v-if="isAgentSupported(row) && (row.agentId || row.collectMode === 'agent_pending')"
                class="revoke-action-btn"
                size="small"
                type="danger"
                plain
                @click="handleRevoke(row)"
              >
                解除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadAgents"
          @current-change="loadAgents"
        />
      </div>
    </div>

    <el-dialog v-model="commandDialogVisible" title="Agent安装命令" width="720px" class="agent-dialog">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="如果一键安装失败，可以复制命令到目标主机手动执行；脚本会优先使用 systemd，老系统会自动降级为后台进程模式。"
      />
      <el-input v-model="manualCommand" type="textarea" :rows="5" readonly class="command-input" />
      <template #footer>
        <el-button @click="commandDialogVisible = false">关闭</el-button>
        <el-button type="primary" :disabled="!manualCommand" @click="copyCommand">复制命令</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="resultDrawerVisible" title="Agent安装结果" size="520px">
      <div v-if="installResults.length === 0" class="empty-result">暂无安装结果</div>
      <div v-else class="result-list">
        <div v-for="item in installResults" :key="item.hostId" class="result-card" :class="{ success: item.success }">
          <div class="result-header">
            <strong>{{ item.hostName }}</strong>
            <div class="result-actions">
              <el-button
                v-if="!item.success && item.command?.command"
                size="small"
                plain
                @click="copyResultCommand(item)"
              >
                复制本次命令
              </el-button>
              <el-tag :type="item.success ? 'success' : 'danger'" size="small">
                {{ item.success ? '成功' : '失败' }}
              </el-tag>
            </div>
          </div>
          <div class="result-message">{{ item.message }}</div>
          <div v-if="getFailureSuggestion(item)" class="result-tip">
            {{ getFailureSuggestion(item) }}
          </div>
          <pre v-if="item.output" class="result-output">{{ item.output }}</pre>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Connection,
  Lightning,
  Monitor,
  Refresh,
  RefreshLeft,
  Search
} from '@element-plus/icons-vue'
import {
  getAgentInstallCommand,
  getHostList,
  installHostAgent,
  revokeHostAgent
} from '@/api/host'

const loading = ref(false)
const batchInstalling = ref(false)
const agentHosts = ref<any[]>([])
const selectedHosts = ref<any[]>([])
const installingIds = ref<Set<number>>(new Set())
const commandDialogVisible = ref(false)
const resultDrawerVisible = ref(false)
const manualCommand = ref('')
const installResults = ref<any[]>([])
const installServerUrl = ref('')

const INSTALL_SERVER_URL_KEY = 'opshub_agent_install_server_url'

const filters = reactive({
  keyword: '',
  collectMode: '',
  agentStatus: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const pageStats = computed(() => {
  return agentHosts.value.reduce(
    (stats, host) => {
      if (!isAgentSupported(host)) {
        stats.sshOnly += 1
        return stats
      }
      if (host.agentStatus === 'online') stats.online += 1
      if (host.collectMode === 'agent_pending') stats.pending += 1
      if (!host.agentId && host.collectMode !== 'agent_pending') stats.uninstalled += 1
      return stats
    },
    { online: 0, pending: 0, uninstalled: 0, sshOnly: 0 }
  )
})

const selectedInstallableHosts = computed(() => selectedHosts.value.filter(isAgentInstallable))

const isCloudHost = (host: any) => {
  return host?.type === 'cloud' || !!host?.cloudProvider || !!host?.cloudInstanceId || !!host?.cloudAccountId
}

const isAgentSupported = (host: any) => {
  if (!host) return false
  if (host.agentSupported === false) return false
  return !isCloudHost(host)
}

const isAgentInstallable = (host: any) => isAgentSupported(host)

const getAgentDisabledReason = (host: any) => {
  if (isAgentInstallable(host)) return ''
  return host?.agentDisabledReason || '云主机仅支持SSH采集，请使用SSH采集主机信息'
}

const installServerUrlWarning = computed(() => {
  if (!installServerUrl.value.trim()) return ''
  if (isLocalInstallServer(installServerUrl.value)) {
    return '当前地址是 localhost/127.0.0.1，远端主机会连接它自己，Agent 安装会失败。请改成目标主机可访问的 OpsHub 或 Agent Gateway 地址。'
  }
  return ''
})

const loadAgents = async () => {
  loading.value = true
  try {
    const data = await getHostList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword,
      collectMode: filters.collectMode,
      agentStatus: filters.agentStatus
    }) as any
    agentHosts.value = data.list || []
    pagination.total = data.total || 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载Agent列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  loadAgents()
}

const handleReset = () => {
  filters.keyword = ''
  filters.collectMode = ''
  filters.agentStatus = ''
  handleSearch()
}

const getCollectModeType = (mode?: string) => {
  if (mode === 'agent') return 'success'
  if (mode === 'agent_pending') return 'warning'
  return 'info'
}

const formatPercent = (value?: number) => {
  if (!value) return '-'
  return `${Number(value).toFixed(1)}%`
}

const initInstallServerUrl = () => {
  const saved = localStorage.getItem(INSTALL_SERVER_URL_KEY)
  installServerUrl.value = saved || window.location.origin
}

const normalizeInstallServerUrl = (value: string) => value.trim().replace(/\/+$/, '')

const parseInstallServerUrl = (value: string) => {
  try {
    const url = new URL(normalizeInstallServerUrl(value))
    if (!['http:', 'https:'].includes(url.protocol)) return null
    return url
  } catch {
    return null
  }
}

const isLocalInstallServer = (value: string) => {
  const url = parseInstallServerUrl(value)
  if (!url) return false
  return ['localhost', '127.0.0.1', '::1', '0.0.0.0'].includes(url.hostname)
}

const persistInstallServerUrl = () => {
  const normalized = normalizeInstallServerUrl(installServerUrl.value)
  installServerUrl.value = normalized
  if (normalized) {
    localStorage.setItem(INSTALL_SERVER_URL_KEY, normalized)
  }
}

const ensureInstallServerUrl = async () => {
  const normalized = normalizeInstallServerUrl(installServerUrl.value)
  const parsed = parseInstallServerUrl(normalized)
  if (!parsed) {
    await ElMessageBox.alert(
      '请先填写正确的 Agent 访问地址，必须以 http:// 或 https:// 开头。',
      'Agent访问地址无效',
      { type: 'warning', confirmButtonText: '知道了' }
    )
    return ''
  }
  if (isLocalInstallServer(normalized)) {
    await ElMessageBox.alert(
      '当前地址是 localhost/127.0.0.1。这个地址会在远端主机上执行，远端会连接它自己，所以会出现 curl: (7) Failed to connect to localhost。请改成目标主机能访问到的 OpsHub 或 Agent Gateway 地址，例如 https://agent-gateway.example.com。',
      'Agent访问地址不可用',
      { type: 'warning', confirmButtonText: '去修改' }
    )
    return ''
  }
  installServerUrl.value = normalized
  localStorage.setItem(INSTALL_SERVER_URL_KEY, normalized)
  return normalized
}

const getFailureSuggestion = (item: any) => {
  if (!item || item.success) return ''
  const text = `${item.message || ''}\n${item.output || ''}`.toLowerCase()
  if (text.includes('localhost') || text.includes('127.0.0.1')) {
    return '处理建议：把上方“Agent访问地址”改成目标主机能访问的 OpsHub 或 Agent Gateway 地址后重新一键安装。'
  }
  if (text.includes('sudo') || text.includes('permission')) {
    return '处理建议：一键安装需要目标用户为 root，或具备免密 sudo 权限。'
  }
  if (text.includes('agent二进制') || text.includes('binary') || text.includes('404')) {
    return '处理建议：检查上方“Agent访问地址”是否是目标主机可访问的 OpsHub 或 Agent Gateway 地址，并确认该地址可以下载 /api/v1/public/agents/binaries/opshub-agent-linux-amd64。'
  }
  if (text.includes('ssh')) {
    return '处理建议：检查主机 SSH 凭据、端口、防火墙和网络连通性。'
  }
  return '处理建议：可以查看下方输出定位原因，或复制“安装命令”到目标主机手动执行。'
}

const isInstalling = (hostId: number) => installingIds.value.has(hostId)

const setInstalling = (hostId: number, installing: boolean) => {
  const next = new Set(installingIds.value)
  if (installing) {
    next.add(hostId)
  } else {
    next.delete(hostId)
  }
  installingIds.value = next
}

const installOne = async (row: any, serverUrl: string) => {
  setInstalling(row.id, true)
  try {
    const result = await installHostAgent(row.id, { server: serverUrl }) as any
    return result
  } finally {
    setInstalling(row.id, false)
  }
}

const handleInstall = async (row: any) => {
  try {
    if (!isAgentInstallable(row)) {
      ElMessage.warning(getAgentDisabledReason(row))
      return
    }
    const serverUrl = await ensureInstallServerUrl()
    if (!serverUrl) return
    await ElMessageBox.confirm(
      `确定通过SSH在主机"${row.name}"上一键安装Agent吗？\n\nAgent将连接：${serverUrl}\n目标用户需要为root或具备免密sudo。`,
      '一键安装Agent',
      {
        confirmButtonText: '开始安装',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    const result = await installOne(row, serverUrl)
    installResults.value = [result]
    resultDrawerVisible.value = true
    if (result.success) {
      ElMessage.success(result.message || '安装命令已执行')
    } else {
      ElMessage.error(getFailureSuggestion(result) || result.message || '安装失败')
    }
    await loadAgents()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '一键安装失败')
    }
  }
}

const handleBatchInstall = async () => {
  const hosts = selectedInstallableHosts.value
  if (hosts.length === 0) {
    ElMessage.warning('请先选择可部署Agent的内网主机')
    return
  }
  try {
    const serverUrl = await ensureInstallServerUrl()
    if (!serverUrl) return
    await ElMessageBox.confirm(
      `确定为选中的 ${hosts.length} 台内网主机批量一键安装Agent吗？\n\nAgent将连接：${serverUrl}`,
      '批量一键安装',
      {
        confirmButtonText: '开始安装',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    batchInstalling.value = true
    const results: any[] = []
    for (const host of hosts) {
      const result = await installOne(host, serverUrl)
      results.push(result)
    }
    installResults.value = results
    resultDrawerVisible.value = true
    const successCount = results.filter(item => item.success).length
    ElMessage.success(`批量安装完成，成功 ${successCount} 台，失败 ${results.length - successCount} 台`)
    await loadAgents()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '批量安装失败')
    }
  } finally {
    batchInstalling.value = false
  }
}

const handleManualCommand = async (row: any) => {
  try {
    if (!isAgentInstallable(row)) {
      ElMessage.warning(getAgentDisabledReason(row))
      return
    }
    const serverUrl = await ensureInstallServerUrl()
    if (!serverUrl) return
    const data = await getAgentInstallCommand(row.id, { server: serverUrl }) as any
    manualCommand.value = data.command || ''
    commandDialogVisible.value = true
    await loadAgents()
  } catch (error: any) {
    ElMessage.error(error.message || '生成安装命令失败')
  }
}

const copyCommand = async () => {
  if (!manualCommand.value) return
  try {
    await navigator.clipboard.writeText(manualCommand.value)
    ElMessage.success('安装命令已复制')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}

const copyResultCommand = async (item: any) => {
  const command = item?.command?.command
  if (!command) return
  try {
    await navigator.clipboard.writeText(command)
    ElMessage.success('本次安装命令已复制')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制安装命令')
  }
}

const handleRevoke = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确定解除主机"${row.name}"的Agent绑定吗？`,
      '解除Agent绑定',
      {
        confirmButtonText: '解除绑定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await revokeHostAgent(row.id)
    ElMessage.success('Agent绑定已解除')
    await loadAgents()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '解除Agent绑定失败')
    }
  }
}

onMounted(() => {
  initInstallServerUrl()
  loadAgents()
})
</script>

<style scoped>
.agent-page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
  padding: 0;
  background-color: transparent;
}

.page-header,
.filter-bar,
.table-wrapper,
.agent-stat-card {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 12px;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
}

.page-title-group,
.header-actions,
.filter-bar,
.filter-inputs,
.filter-actions,
.host-cell,
.action-buttons,
.agent-status-cell,
.result-header {
  display: flex;
  align-items: center;
}

.page-title-group {
  gap: 14px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: #0f766e;
  background: linear-gradient(135deg, #ccfbf1 0%, #ecfeff 100%);
  font-size: 22px;
}

.page-title {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
}

.page-subtitle {
  margin: 6px 0 0;
  color: #667085;
  font-size: 13px;
}

.header-actions {
  gap: 10px;
}

.black-button {
  color: #fff;
  border-color: #111827;
  background: #111827;
}

.install-settings {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(360px, 520px);
  gap: 16px;
  align-items: center;
  padding: 14px 16px;
  background: linear-gradient(135deg, #f8fbff 0%, #ffffff 100%);
  border: 1px solid #e5e9f2;
  border-radius: 12px;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.install-settings.warning {
  border-color: #fed7aa;
  background: #fffaf3;
}

.setting-title {
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.setting-desc {
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 1.55;
}

.server-url-input {
  width: 100%;
}

.server-url-alert {
  margin-top: -6px;
}

.agent-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.agent-stat-card {
  padding: 16px;
}

.agent-stat-card strong {
  display: block;
  margin-top: 8px;
  color: #111827;
  font-size: 26px;
}

.stat-label {
  color: #667085;
  font-size: 13px;
}

.agent-stat-card.online strong {
  color: #059669;
}

.agent-stat-card.pending strong {
  color: #d97706;
}

.agent-stat-card.muted strong {
  color: #64748b;
}

.agent-stat-card.ssh-only strong {
  color: #0f766e;
}

.filter-bar {
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
}

.filter-inputs,
.filter-actions {
  gap: 10px;
}

.filter-input {
  width: 220px;
}

.table-wrapper {
  overflow: hidden;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.modern-table {
  flex: 1;
}

.modern-table :deep(.el-table__header th) {
  background: #f8fafc !important;
  color: #475467 !important;
  font-weight: 650;
}

.host-cell {
  gap: 10px;
}

.host-avatar {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  color: #64748b;
  background: #f1f5f9;
}

.host-avatar.online {
  color: #047857;
  background: #d1fae5;
}

.host-name {
  color: #111827;
  font-weight: 650;
}

.host-meta {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
}

.ssh-only-tag {
  margin-top: 6px;
  border-color: #ccfbf1;
  color: #0f766e;
  background: #f0fdfa;
}

.agent-status-cell {
  justify-content: center;
  gap: 7px;
  color: #344054;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #94a3b8;
}

.status-dot.online {
  background: #10b981;
}

.status-dot.offline {
  background: #ef4444;
}

.status-dot.pending {
  background: #f59e0b;
}

.resource-mini {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.resource-mini span {
  padding: 3px 7px;
  color: #475467;
  background: #f8fafc;
  border-radius: 999px;
  font-size: 12px;
}

.action-buttons {
  justify-content: center;
  gap: 6px;
  flex-wrap: nowrap;
  width: 100%;
}

.action-buttons > span {
  display: inline-flex;
}

.agent-table :deep(td.el-table-fixed-column--right .cell),
.agent-table :deep(.el-table__fixed-right .cell) {
  display: flex !important;
  justify-content: center !important;
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.agent-table :deep(.action-buttons .el-button) {
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  flex: 0 0 auto !important;
  width: auto !important;
  min-width: 70px !important;
  height: 32px !important;
  min-height: 32px !important;
  margin: 0 !important;
  padding: 0 12px !important;
  border-radius: 8px !important;
  font-size: 13px !important;
  font-weight: 650 !important;
  line-height: 1 !important;
  white-space: nowrap !important;
  opacity: 1 !important;
}

.agent-table :deep(.action-buttons .el-button.is-disabled) {
  color: #98a2b3 !important;
  border-color: #e5e7eb !important;
  background: #f8fafc !important;
}

.agent-table :deep(.install-action-btn.el-button) {
  color: #fff !important;
  border-color: #111827 !important;
  background: #111827 !important;
}

.agent-table :deep(.install-action-btn.el-button:hover) {
  color: #fff !important;
  border-color: #263244 !important;
  background: #263244 !important;
}

.agent-table :deep(.manual-action-btn.el-button) {
  color: #344054 !important;
  border-color: #d9e0eb !important;
  background: #fff !important;
}

.agent-table :deep(.manual-action-btn.el-button:hover) {
  color: #111827 !important;
  border-color: #b8c2d3 !important;
  background: #f8fafc !important;
}

.agent-table :deep(.revoke-action-btn.el-button) {
  color: #ef4444 !important;
  border-color: #fecaca !important;
  background: #fff !important;
}

.agent-table :deep(.revoke-action-btn.el-button:hover) {
  color: #dc2626 !important;
  border-color: #fca5a5 !important;
  background: #fff5f5 !important;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 14px 16px;
  border-top: 1px solid #edf1f7;
}

.command-input {
  margin-top: 14px;
}

.command-input :deep(.el-textarea__inner),
.result-output {
  font-family: Menlo, Monaco, Consolas, monospace;
}

.result-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.result-card {
  padding: 14px;
  border: 1px solid #fecaca;
  border-radius: 10px;
  background: #fff7f7;
}

.result-card.success {
  border-color: #bbf7d0;
  background: #f0fdf4;
}

.result-header {
  justify-content: space-between;
  gap: 10px;
}

.result-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.result-message {
  margin-top: 8px;
  color: #475467;
  font-size: 13px;
}

.result-tip {
  margin-top: 8px;
  padding: 8px 10px;
  color: #9a3412;
  background: #fff7ed;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
}

.result-output {
  max-height: 220px;
  margin: 10px 0 0;
  padding: 10px;
  overflow: auto;
  color: #1f2937;
  background: #f8fafc;
  border-radius: 8px;
  font-size: 12px;
  white-space: pre-wrap;
}

.empty-result {
  color: #98a2b3;
  text-align: center;
}

@media (max-width: 1100px) {
  .page-header,
  .filter-bar,
  .install-settings {
    align-items: flex-start;
    flex-direction: column;
  }

  .install-settings {
    display: flex;
  }

  .agent-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .agent-page-container {
    padding: 0;
  }

  .filter-input,
  .filter-inputs,
  .filter-actions,
  .server-url-input,
  .header-actions {
    width: 100%;
  }

  .filter-inputs,
  .filter-actions,
  .header-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .agent-stats {
    grid-template-columns: 1fr;
  }
}
</style>
