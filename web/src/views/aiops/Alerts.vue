<template>
  <div class="aiops-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Bell /></el-icon>
        </div>
        <div>
          <h2 class="page-title">告警分析</h2>
          <p class="page-subtitle">关联监控告警、规则、最近事件和操作审计，生成根因分析与排查建议</p>
        </div>
      </div>
      <div class="header-actions">
        <ProviderSelect v-model="providerId" placeholder="选择模型" />
        <el-button class="black-button" :loading="analyzing" :disabled="!selectedEvent || analyzing" @click="analyzeSelected">
          <el-icon><MagicStick /></el-icon>
          {{ analyzeButtonText }}
        </el-button>
      </div>
    </div>

    <div class="alert-workspace">
      <section class="panel-card alert-list-card">
        <div class="section-head">
          <div class="section-title">告警事件</div>
          <el-button text @click="loadEvents">刷新</el-button>
        </div>
        <div class="filter-row">
          <el-select v-model="filters.state" clearable placeholder="状态" @change="loadEvents">
            <el-option label="Firing" value="firing" />
            <el-option label="Pending" value="pending" />
            <el-option label="Recovered" value="recovered" />
          </el-select>
          <el-select v-model="filters.severity" clearable placeholder="级别" @change="loadEvents">
            <el-option label="P0" value="p0" />
            <el-option label="P1" value="p1" />
            <el-option label="P2" value="p2" />
          </el-select>
        </div>
        <el-table
          v-loading="loading"
          :data="events"
          class="modern-table"
          highlight-current-row
          empty-text="暂无告警事件"
          :row-class-name="alertRowClassName"
          @current-change="handleCurrentChange"
        >
          <el-table-column label="告警" min-width="260">
            <template #default="{ row }">
              <div class="event-title">{{ row.ruleName }}</div>
              <div class="event-meta">{{ row.dataSourceName || '-' }} · {{ row.dataSourceType || '-' }}</div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="stateTag(row.state)" effect="plain">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="级别" width="110">
            <template #default="{ row }">
              <el-tag :type="severityTag(row.severity)" effect="plain">{{ severityLabel(row.severity) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="当前值" width="110" />
          <el-table-column prop="lastEvalAt" label="最近评估" width="180" />
        </el-table>
        <div class="pagination-wrap">
          <el-pagination
            v-model:current-page="eventPagination.page"
            v-model:page-size="eventPagination.pageSize"
            :total="eventPagination.total"
            layout="total, sizes, prev, pager, next"
            @current-change="loadEvents"
            @size-change="loadEvents"
          />
        </div>
      </section>

      <section class="panel-card analysis-card">
        <el-tabs v-model="analysisTab" class="analysis-tabs">
          <el-tab-pane label="根因分析" name="current">
            <div class="analysis-head">
              <div>
                <div class="section-title">根因分析</div>
                <p>先在左侧选择一条告警，再点击分析；分析期间会锁定当时选中的告警。</p>
              </div>
              <el-tag v-if="analyzing && alertRun" type="warning" effect="plain">已耗时 {{ alertRun.elapsed }}s</el-tag>
            </div>
            <el-input
              v-model="query"
              type="textarea"
              :rows="4"
              resize="none"
              :disabled="analyzing"
              placeholder="可补充现象，例如：刚发布后开始告警、只影响某个命名空间"
            />
            <div class="selected-card" v-if="activeEvent">
              <div class="selected-card-head">
                <strong>{{ activeEvent.ruleName }}</strong>
                <el-tag :type="severityTag(activeEvent.severity)" effect="plain">{{ severityLabel(activeEvent.severity) }}</el-tag>
              </div>
              <span>{{ activeEvent.state }} · {{ activeEvent.dataSourceName || '-' }} · {{ activeEvent.dataSourceType || '-' }} · {{ activeEvent.lastEvalAt }}</span>
              <p>{{ activeEvent.message || activeEvent.labels || '暂无告警消息，分析时会结合规则、数据源和事件上下文。' }}</p>
            </div>
            <div v-if="analyzing && alertRun" class="analysis-progress-card">
              <div class="progress-main">
                <div>
                  <div class="progress-title">正在分析：{{ alertRun.event.ruleName }}</div>
                  <div class="progress-subtitle">{{ alertRun.status }}</div>
                </div>
                <el-progress type="circle" :width="76" :percentage="progressPercent" :stroke-width="7" status="warning" />
              </div>
              <div class="progress-steps">
                <div v-for="step in alertRun.steps" :key="step" class="progress-step">
                  <span class="step-dot" />
                  {{ step }}
                </div>
              </div>
              <div class="progress-hint">
                已固定分析对象为告警事件 #{{ alertRun.event.id }}。即使你切换表格行，这次请求仍然分析这条告警，返回后会自动展示根因报告。
              </div>
            </div>
            <el-alert v-else-if="analysisError" class="analysis-error" type="error" :title="analysisError" show-icon :closable="false" />
            <el-empty v-else-if="!analysis" description="选择告警后点击分析" />
            <template v-else>
              <div class="analysis-meta">
                <el-tag :type="analysis.fallback ? 'warning' : 'success'" effect="plain">
                  {{ analysis.fallback ? '本地分析' : analysis.model }}
                </el-tag>
                <el-tag :type="severityTag(analysis.severity)" effect="plain">{{ severityLabel(analysis.severity) }}</el-tag>
              </div>
              <MarkdownView class="analysis-output" :content="analysis.rootCause" />
            </template>
          </el-tab-pane>

          <el-tab-pane label="分析记录" name="history">
            <div class="section-head compact-head">
              <div>
                <div class="section-title">分析记录</div>
                <p>历史记录在这里查看，不再和当前分析结果上下挤在一起。</p>
              </div>
              <el-button text @click="loadAnalyses">刷新</el-button>
            </div>
            <el-table :data="analyses" class="modern-table history-table" empty-text="暂无分析记录">
              <el-table-column label="规则" min-width="190">
                <template #default="{ row }">
                  <div class="event-title">{{ row.ruleName || '-' }}</div>
                  <div class="event-meta">告警事件 #{{ row.alertEventId || '-' }}</div>
                </template>
              </el-table-column>
              <el-table-column label="级别" width="86">
                <template #default="{ row }">
                  <el-tag :type="severityTag(row.severity)" effect="plain">{{ severityLabel(row.severity) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="summary" label="摘要" min-width="220" show-overflow-tooltip />
              <el-table-column label="操作" width="86" fixed="right" align="center">
                <template #default="{ row }">
                  <el-button link class="action-link" @click="showHistoryAnalysis(row)">查看</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="pagination-wrap">
              <el-pagination
                v-model:current-page="analysisPagination.page"
                v-model:page-size="analysisPagination.pageSize"
                :total="analysisPagination.total"
                layout="total, prev, pager, next"
                @current-change="loadAnalyses"
                @size-change="loadAnalyses"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Bell, MagicStick } from '@element-plus/icons-vue'
import {
  analyzeAIAlert,
  getAIAlertEvents,
  getAIRootCauseAnalyses,
  type AIAlertEvent,
  type AIRootCauseAnalysis
} from '@/api/aiops'
import MarkdownView from './components/MarkdownView.vue'
import ProviderSelect from './components/ProviderSelect.vue'

interface AlertAnalysisRun {
  event: AIAlertEvent
  startedAt: number
  elapsed: number
  status: string
  steps: string[]
}

const loading = ref(false)
const analyzing = ref(false)
const events = ref<AIAlertEvent[]>([])
const analyses = ref<AIRootCauseAnalysis[]>([])
const selectedEvent = ref<AIAlertEvent | null>(null)
const analyzingEvent = ref<AIAlertEvent | null>(null)
const analysis = ref<AIRootCauseAnalysis | null>(null)
const analysisError = ref('')
const alertRun = ref<AlertAnalysisRun | null>(null)
const query = ref('')
const analysisTab = ref('current')
const filters = reactive({ state: '', severity: '' })
const eventPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const analysisPagination = reactive({ page: 1, pageSize: 10, total: 0 })
const providerId = ref<number | undefined>()
let alertTimer: number | undefined

const activeEvent = computed(() => analyzingEvent.value || selectedEvent.value)
const analyzeButtonText = computed(() => (analyzing.value && alertRun.value ? `分析中 ${alertRun.value.elapsed}s` : '分析选中告警'))
const progressPercent = computed(() => {
  if (!alertRun.value) return 0
  return Math.min(92, Math.max(8, Math.floor(alertRun.value.elapsed * 2.4)))
})

const normalizePage = (res: any) => ({
  data: res?.data || [],
  total: res?.total || 0,
  page: res?.page || 1,
  pageSize: res?.page_size || res?.pageSize || 20
})

const errorMessage = (error: unknown, fallback: string) => {
  const responseMessage = (error as any)?.response?.data?.message || (error as any)?.response?.data?.msg
  return responseMessage || (error as Error)?.message || fallback
}

const alertRunStatus = (elapsed: number) => {
  if (elapsed < 3) return '正在提交根因分析请求并锁定告警上下文...'
  if (elapsed < 10) return '正在读取告警事件、规则配置、数据源和最近评估信息...'
  if (elapsed < 25) return '正在关联最近事件、操作审计和可能的影响范围...'
  if (elapsed < 90) return '模型正在生成根因排序、证据链和排查建议...'
  return '仍在等待模型或后端返回，请勿重复点击；返回后会自动展示报告。'
}

const startAlertRun = (event: AIAlertEvent) => {
  stopAlertRun(false)
  alertRun.value = {
    event,
    startedAt: Date.now(),
    elapsed: 0,
    status: alertRunStatus(0),
    steps: ['锁定选中告警', '读取规则和事件上下文', '关联最近操作与评估记录', '生成根因分析报告']
  }
  alertTimer = window.setInterval(() => {
    if (!alertRun.value) return
    const elapsed = Math.floor((Date.now() - alertRun.value.startedAt) / 1000)
    alertRun.value.elapsed = elapsed
    alertRun.value.status = alertRunStatus(elapsed)
  }, 1000)
}

const stopAlertRun = (clearRun = true) => {
  if (alertTimer) {
    window.clearInterval(alertTimer)
    alertTimer = undefined
  }
  if (clearRun) {
    alertRun.value = null
  }
}

const loadEvents = async () => {
  loading.value = true
  try {
    const res = normalizePage(await getAIAlertEvents({
      page: eventPagination.page,
      pageSize: eventPagination.pageSize,
      state: filters.state || undefined,
      severity: filters.severity || undefined
    }))
    events.value = res.data
    eventPagination.total = res.total
  } finally {
    loading.value = false
  }
}

const handleCurrentChange = (event: AIAlertEvent | null) => {
  selectedEvent.value = event
  analysisError.value = ''
}

const loadAnalyses = async () => {
  const res = normalizePage(await getAIRootCauseAnalyses({
    page: analysisPagination.page,
    pageSize: analysisPagination.pageSize
  }))
  analyses.value = res.data
  analysisPagination.total = res.total
}

const showHistoryAnalysis = (row: AIRootCauseAnalysis) => {
  analysis.value = row
  analysisError.value = ''
  analysisTab.value = 'current'
}

const analyzeSelected = async () => {
  if (!selectedEvent.value) return
  const event = selectedEvent.value
  analyzingEvent.value = event
  analysis.value = null
  analysisError.value = ''
  startAlertRun(event)
  analyzing.value = true
  analysisTab.value = 'current'
  try {
    analysis.value = await analyzeAIAlert({
      alertEventId: event.id,
      query: query.value || undefined,
      providerId: providerId.value
    })
    await loadAnalyses()
  } catch (error) {
    analysisError.value = errorMessage(error, '告警根因分析失败，请检查模型配置或告警上下文')
    ElMessage.error(analysisError.value)
  } finally {
    analyzing.value = false
    analyzingEvent.value = null
    stopAlertRun()
  }
}

const stateTag = (state?: string) => {
  if (state === 'firing') return 'danger'
  if (state === 'pending') return 'warning'
  if (state === 'recovered') return 'success'
  return 'info'
}

const severityTag = (level?: string) => {
  const normalized = String(level || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'p0') return 'danger'
  if (normalized === 'warning' || normalized === 'p1' || normalized === 'high') return 'warning'
  return 'info'
}

const severityLabel = (level?: string) => {
  const normalized = String(level || '').toLowerCase()
  if (!normalized) return '-'
  if (normalized === 'critical' || normalized === 'p0') return 'P0'
  if (normalized === 'warning' || normalized === 'p1' || normalized === 'high') return 'P1'
  if (normalized === 'info' || normalized === 'p2' || normalized === 'medium' || normalized === 'low') return 'P2'
  return level || '-'
}

const alertRowClassName = ({ row }: { row: AIAlertEvent }) => {
  if (analyzingEvent.value?.id === row.id) return 'analyzing-alert-row'
  return ''
}

onMounted(async () => {
  await Promise.all([loadEvents(), loadAnalyses()])
})

onUnmounted(() => {
  stopAlertRun()
})
</script>

<style scoped>
.aiops-page {
  min-height: 100%;
}

.page-header,
.panel-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 18px;
  box-shadow: 0 12px 32px rgba(31, 45, 61, 0.07);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 22px 24px;
  margin-bottom: 18px;
}

.header-actions {
  display: flex;
  align-items: end;
  justify-content: flex-end;
  gap: 12px;
  flex: 0 0 auto;
  min-width: 430px;
}

.page-title-group {
  display: flex;
  gap: 14px;
  align-items: center;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #111827;
  color: #fff;
  font-size: 20px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  color: #111827;
}

.page-subtitle,
.event-meta,
.selected-card span {
  color: #64748b;
  font-size: 13px;
}

.alert-workspace {
  display: grid;
  grid-template-columns: minmax(560px, 1.12fr) minmax(420px, 0.88fr);
  gap: 18px;
  align-items: start;
}

.panel-card {
  padding: 20px;
  min-width: 0;
}

.section-head,
.filter-row,
.analysis-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.section-title {
  font-weight: 800;
  color: #111827;
  margin-bottom: 16px;
}

.compact-head {
  margin-bottom: 14px;
}

.compact-head .section-title {
  margin-bottom: 4px;
}

.compact-head p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.analysis-tabs :deep(.el-tabs__header) {
  margin-bottom: 18px;
}

.analysis-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background: #e5e7eb;
}

.analysis-tabs :deep(.el-tabs__item) {
  color: #64748b;
  font-weight: 700;
}

.analysis-tabs :deep(.el-tabs__item.is-active) {
  color: #111827;
}

.analysis-tabs :deep(.el-tabs__active-bar) {
  background-color: #111827;
}

.analysis-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.analysis-head .section-title {
  margin-bottom: 4px;
}

.analysis-head p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
  line-height: 1.6;
}

.filter-row {
  justify-content: flex-start;
  margin-bottom: 14px;
}

.event-title {
  color: #111827;
  font-weight: 800;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.analysis-card {
  align-self: stretch;
  position: sticky;
  top: 16px;
}

.selected-card {
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #f8fafc;
  padding: 12px;
  margin: 12px 0;
}

.selected-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}

.selected-card strong {
  color: #111827;
  font-size: 16px;
}

.selected-card span {
  display: block;
}

.selected-card p {
  color: #334155;
  line-height: 1.6;
  margin: 8px 0 0;
}

.analysis-meta {
  justify-content: flex-start;
  margin: 14px 0 10px;
}

.analysis-progress-card {
  border: 1px solid #fde68a;
  border-radius: 18px;
  background:
    radial-gradient(circle at 90% 10%, rgba(251, 191, 36, 0.2), transparent 28%),
    linear-gradient(135deg, #fffdf5 0%, #ffffff 52%, #f8fafc 100%);
  margin-top: 14px;
  padding: 18px;
}

.progress-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.progress-title {
  color: #111827;
  font-size: 18px;
  font-weight: 900;
}

.progress-subtitle {
  color: #64748b;
  line-height: 1.7;
  margin-top: 6px;
}

.progress-steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 18px;
}

.progress-step {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #fff;
  color: #334155;
  line-height: 1.55;
  padding: 12px;
}

.step-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 999px;
  background: #111827;
  margin-top: 7px;
  box-shadow: 0 0 0 5px rgba(17, 24, 39, 0.08);
}

.progress-hint {
  margin-top: 14px;
  color: #64748b;
  line-height: 1.7;
}

.analysis-error {
  margin-top: 14px;
}

.analysis-output {
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  border-radius: 14px;
  padding: 16px;
  max-height: calc(100vh - 430px);
  min-height: 260px;
  overflow: auto;
}

.history-table {
  width: 100%;
}

.alert-list-card :deep(.el-table__body-wrapper),
.history-table :deep(.el-table__body-wrapper) {
  scrollbar-width: thin;
}

.black-button {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.action-link {
  color: #2563eb;
}

:deep(.modern-table .el-table__header th) {
  background: #f8fafc;
  color: #475569;
  font-weight: 700;
}

:deep(.modern-table .analyzing-alert-row td) {
  background: #fff7ed !important;
}

@media (max-width: 1280px) {
  .alert-workspace {
    grid-template-columns: 1fr;
  }

  .analysis-card {
    position: static;
  }

  .analysis-output {
    max-height: none;
  }
}

@media (max-width: 980px) {

  .page-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .header-actions {
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .analysis-head,
  .progress-main {
    align-items: flex-start;
    flex-direction: column;
  }

  .progress-steps {
    grid-template-columns: 1fr;
  }
}
</style>
