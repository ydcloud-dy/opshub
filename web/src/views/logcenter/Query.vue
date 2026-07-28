<template>
  <div class="log-center-page internal-query-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon"><el-icon><Search /></el-icon></div>
        <div>
          <h2>日志查询</h2>
          <p>检索由 OpsHub Log Agent 采集并写入 ClickHouse 的主机与 Kubernetes 日志</p>
        </div>
      </div>
      <div class="head-actions">
        <el-button class="template-trigger" :loading="templateLoading" :disabled="tailing" @click="openTemplatePicker">
          <el-icon><CollectionTag /></el-icon>
          使用模板
        </el-button>
        <el-button :type="tailing ? 'danger' : 'default'" :disabled="!form.storageId" @click="toggleTail">
          <el-icon><SwitchButton v-if="tailing" /><VideoPlay v-else /></el-icon>
          {{ tailing ? '停止 Tail' : '实时 Tail' }}
        </el-button>
        <el-button :disabled="!form.storageId" @click="openExportDialog">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <el-button :loading="loading" :disabled="tailing" @click="runQuery">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <section class="panel query-panel">
      <div class="query-context-row">
        <div class="context-control storage-context">
          <span class="control-label">日志存储</span>
          <el-select v-model="form.storageId" class="storage-select" placeholder="选择 ClickHouse 存储" @change="loadFields">
            <el-option
              v-for="item in storages"
              :key="item.id"
              :label="`${item.name}${item.isPrimary ? '（主存储）' : ''}`"
              :value="item.id"
              :disabled="!item.enabled || !item.initializedAt"
            />
          </el-select>
          <el-tag size="small" :type="activeStorage?.status === 'healthy' ? 'success' : 'warning'">
            {{ activeStorage?.status === 'healthy' ? '连接正常' : '待检测' }}
          </el-tag>
          <el-tag v-if="routeHostId" size="small" type="info" closable @close="clearHostScope">{{ hostLabel(routeHostId) }}</el-tag>
          <el-tag v-if="routeClusterId" size="small" type="info" closable @close="clearClusterScope">{{ clusterLabel(routeClusterId) }}</el-tag>
        </div>
        <div class="context-control time-context">
          <span class="control-label">时间范围</span>
          <el-select v-model="quickRange" class="quick-range-select" :disabled="tailing" @change="applyQuickRange">
            <el-option label="自定义时间" :value="0" />
            <el-option label="最近 15 分钟" :value="15" />
            <el-option label="最近 1 小时" :value="60" />
            <el-option label="最近 6 小时" :value="360" />
            <el-option label="最近 24 小时" :value="1440" />
            <el-option label="最近 7 天" :value="10080" />
          </el-select>
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            :clearable="false"
            :disabled="tailing"
            class="time-picker"
            @change="handleTimeRangeChange"
          />
        </div>
      </div>

      <div class="query-search-row">
        <el-input v-model="form.query" class="query-text-input" clearable :disabled="tailing" placeholder="搜索日志正文，输入 * 查询全部日志" @keyup.enter="runQuery">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button class="condition-shortcut" :disabled="tailing" @click="openConditionBuilder">
          <el-icon><Plus /></el-icon>
          组合条件
          <span v-if="conditions.length">{{ conditions.length }}</span>
        </el-button>
        <el-select v-model="form.limit" class="page-size-select" :disabled="tailing" @change="runQuery">
          <el-option v-for="size in pageSizeOptions" :key="size" :label="`${size} 条/页`" :value="size" />
        </el-select>
        <el-button type="primary" class="primary-action" :loading="loading" :disabled="tailing" @click="runQuery">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
      </div>

      <div v-loading="resourceLoading" class="filter-section">
        <div class="filter-section-head">
          <div class="filter-title">
            <el-icon><Filter /></el-icon>
            <strong>范围筛选</strong>
            <el-tag v-if="activeFilterCount" size="small" type="info">{{ activeFilterCount }} 项已选</el-tag>
          </div>
          <div class="filter-head-actions">
            <el-button v-if="activeFilterCount" link :disabled="tailing" @click="clearScopeFilters">清空筛选</el-button>
            <el-button link class="advanced-toggle" @click="advancedOpen = !advancedOpen">
              {{ advancedOpen ? '收起组合条件' : '组合字段条件' }}
              <el-icon :class="{ rotated: advancedOpen }"><ArrowDown /></el-icon>
            </el-button>
          </div>
        </div>

        <div v-if="activeFilterChips.length" class="active-filter-summary">
          <span>当前筛选</span>
          <el-tag v-for="chip in activeFilterChips" :key="chip.key" size="small" effect="plain">{{ chip.label }}：{{ chip.value }}</el-tag>
        </div>

        <div class="primary-filter-grid">
          <div class="filter-control source-filter-control">
            <span>日志来源</span>
            <el-segmented v-model="sourceMode" :options="sourceModeOptions" :disabled="tailing" @change="handleSourceModeChange" />
          </div>
          <div v-if="sourceMode === 'host'" class="filter-control">
            <span>主机</span>
            <el-select v-model="scope.hostIds" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部主机">
              <el-option v-for="host in availableHosts" :key="host.id" :label="host.label" :value="host.id" />
            </el-select>
          </div>
          <div class="filter-control">
            <span>环境</span>
            <el-select v-model="scope.environments" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部环境">
              <el-option v-for="value in resourceOptions.environments" :key="value" :label="value" :value="value" />
            </el-select>
          </div>
          <div class="filter-control">
            <span>服务</span>
            <el-select v-model="scope.services" multiple filterable allow-create collapse-tags clearable :disabled="tailing" placeholder="全部服务">
              <el-option v-for="value in resourceOptions.services" :key="value" :label="value" :value="value" />
            </el-select>
          </div>
          <div class="filter-control level-filter-control">
            <span>日志级别</span>
            <el-select v-model="scope.level" clearable :disabled="tailing" placeholder="全部级别">
              <el-option v-for="level in ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']" :key="level" :label="level" :value="level" />
            </el-select>
          </div>
        </div>

        <div v-if="sourceMode === 'kubernetes'" class="kubernetes-filter-band">
          <div class="kubernetes-band-head">
            <div>
              <strong>Kubernetes 资源范围</strong>
              <span>按集群到容器逐级筛选，下级选项会自动匹配上级资源</span>
            </div>
            <span v-if="resourceOptions.kubernetesResources.length">已识别 {{ resourceOptions.kubernetesResources.length }} 条资源路径</span>
          </div>
          <div class="kubernetes-filter-grid">
            <div class="filter-control hierarchy-control">
              <span><i>1</i>Kubernetes 集群</span>
              <el-select v-model="scope.clusterIds" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部集群" @change="handleClusterScopeChange">
                <el-option v-for="cluster in availableClusters" :key="cluster.id" :label="cluster.label" :value="cluster.id" />
              </el-select>
            </div>
            <div class="filter-control hierarchy-control">
              <span><i>2</i>Namespace</span>
              <el-select v-model="scope.namespaces" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部 Namespace" @change="handleNamespaceScopeChange">
                <el-option v-for="option in kubernetesNamespaceOptions" :key="option.value" :label="option.label" :value="option.value" />
              </el-select>
            </div>
            <div class="filter-control hierarchy-control">
              <span><i>3</i>工作负载</span>
              <el-select v-model="scope.workloads" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部工作负载" @change="handleWorkloadScopeChange">
                <el-option v-for="option in kubernetesWorkloadOptions" :key="option.value" :label="option.label" :value="option.value" />
              </el-select>
            </div>
            <div class="filter-control hierarchy-control">
              <span><i>4</i>Pod</span>
              <el-select v-model="scope.pods" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部 Pod" @change="handlePodScopeChange">
                <el-option v-for="option in kubernetesPodOptions" :key="option.value" :label="option.label" :value="option.value" />
              </el-select>
            </div>
            <div class="filter-control hierarchy-control">
              <span><i>5</i>容器</span>
              <el-select v-model="scope.containers" multiple filterable collapse-tags clearable :disabled="tailing" placeholder="全部容器">
                <el-option v-for="option in kubernetesContainerOptions" :key="option.value" :label="option.label" :value="option.value" />
              </el-select>
            </div>
          </div>
        </div>

        <el-collapse-transition>
          <div v-show="advancedOpen" class="advanced-filter-area">
            <div class="condition-builder">
              <div class="condition-head">
                <div class="condition-title">
                  <strong>组合字段条件</strong>
                  <span>范围筛选始终与本条件组同时满足</span>
                </div>
                <div class="condition-actions">
                  <el-segmented v-if="conditions.length" v-model="filterLogic" :options="filterLogicOptions" :disabled="tailing" size="small" />
                  <el-button link :disabled="tailing" @click="addCondition"><el-icon><Plus /></el-icon>添加条件</el-button>
                </div>
              </div>
              <div v-if="conditions.length" class="condition-list">
                <div v-for="(condition, index) in conditions" :key="condition.id" class="condition-row">
                  <span class="condition-connector">{{ index === 0 ? '当' : filterLogic === 'and' ? '且' : '或' }}</span>
                  <el-select v-model="condition.field" class="condition-field-select" filterable :disabled="tailing" placeholder="字段">
                    <el-option v-for="field in conditionFields" :key="field.name" :label="fieldDisplayName(field.name)" :value="field.name" />
                  </el-select>
                  <el-select v-model="condition.operator" class="condition-operator-select" :disabled="tailing" placeholder="操作符" @change="handleConditionOperatorChange(condition)">
                    <el-option v-for="operator in conditionOperators" :key="operator.value" :label="operator.label" :value="operator.value" />
                  </el-select>
                  <el-select
                    v-if="condition.operator === 'in'"
                    v-model="condition.values"
                    class="condition-value-input"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    collapse-tags
                    :max-collapse-tags="3"
                    :disabled="tailing"
                    placeholder="输入一个值后按 Enter，可继续添加"
                    @paste="handleConditionValuesPaste($event, condition)"
                  >
                    <el-option v-for="option in conditionValueOptions(condition.field)" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                  <el-input v-else v-model="condition.value" class="condition-value-input" :disabled="tailing" placeholder="条件值" @keyup.enter="runQuery" />
                  <el-button class="condition-delete" circle text :disabled="tailing" title="删除条件" @click="removeCondition(condition.id)"><el-icon><Delete /></el-icon></el-button>
                </div>
              </div>
              <div v-else class="empty-condition">未添加组合条件；点击“添加条件”可按字段精确检索</div>
            </div>
          </div>
        </el-collapse-transition>
      </div>
    </section>

    <section class="panel histogram-panel">
      <div class="section-head">
        <div>
          <strong>日志趋势</strong>
          <span v-if="histogramError" class="trend-error">{{ histogramError }} <button type="button" @click="runQuery">重试</button></span>
          <span v-else>{{ totalCount.toLocaleString() }} 条趋势统计 · 查询耗时 {{ durationMs }} ms</span>
        </div>
        <span>{{ formatRangeLabel }}</span>
      </div>
      <div ref="chartRef" class="histogram-chart"></div>
    </section>

    <section class="query-result-grid">
      <aside class="panel field-panel">
        <div class="section-head compact">
          <strong>显示字段</strong>
          <span>{{ selectedFields.length }} / {{ fieldOptions.length }}</span>
        </div>
        <el-input v-model="fieldKeyword" size="small" clearable placeholder="搜索字段" />
        <div class="field-list">
          <button
            v-for="field in filteredFields"
            :key="field.name"
            type="button"
            :class="{ active: selectedFields.includes(field.name) }"
            @click="toggleField(field.name)"
          >
            <span>{{ fieldDisplayName(field.name) }}</span>
            <em>{{ field.type }}</em>
          </button>
        </div>
      </aside>

      <div class="panel log-panel">
        <div class="section-head log-head">
          <div>
            <strong>日志明细</strong>
            <span v-if="tailing" class="tail-status"><i :class="{ connected: tailConnected }"></i>{{ tailStatusText }} · 新增 {{ tailReceived.toLocaleString() }} 条 · 当前 {{ items.length.toLocaleString() }} 条</span>
            <span v-else-if="loading">正在查询 ClickHouse...</span>
            <span v-else>第 {{ currentPage }} 页 · 本页 {{ items.length.toLocaleString() }} 条</span>
          </div>
          <div class="log-view-actions">
            <label class="wrap-control">
              <span>自动换行</span>
              <el-switch v-model="wrapLines" size="small" />
            </label>
            <el-segmented v-model="form.sort" :options="sortOptions" :disabled="tailing" size="small" />
          </div>
        </div>

        <el-alert v-if="errorMessage" type="error" :closable="false" show-icon>
          <template #title>
            <div class="query-error-title">
              <span>{{ errorMessage }}</span>
              <el-button link type="danger" :loading="loading" @click="runQuery">重新查询</el-button>
            </div>
          </template>
        </el-alert>
        <div v-loading="(loading && !items.length) || loadingMore" ref="logStreamRef" class="log-stream">
          <div class="log-table" :class="{ 'wrap-lines': wrapLines }" :style="logTableStyle">
            <div class="log-columns-head">
              <span class="log-column-expander"></span>
              <span v-if="showTimestamp">时间</span>
              <span v-if="showLevel">级别</span>
              <span v-for="field in selectedColumnFields" :key="`head-${field.name}`">{{ field.label }}</span>
              <span v-if="showMessage">日志正文</span>
              <span>操作</span>
            </div>
            <article
              v-for="(item, index) in renderedItems"
              :key="logKey(item, index)"
              class="log-row"
              :class="{ expanded: expandedRows.has(logKey(item, index)) }"
            >
              <div class="log-line">
                <button class="expand-button" type="button" title="展开日志详情" @click="toggleExpanded(item, index)">
                  <el-icon><ArrowRight /></el-icon>
                </button>
                <time v-if="showTimestamp">{{ formatTimestamp(item.timestamp) }}</time>
                <span v-if="showLevel" class="level-badge" :class="levelClass(item.level)">{{ levelText(item.level) }}</span>
                <span v-for="field in selectedColumnFields" :key="field.name" class="log-field-cell" :title="displayFieldTitle(item, field.name)">{{ displayFieldValue(item, field.name) }}</span>
                <div v-if="showMessage" class="log-message" :title="wrapLines ? undefined : item.message">
                  <p v-html="highlightMessage(item.message)"></p>
                </div>
                <div class="row-actions">
                  <el-button link size="small" @click="openContext(item)">上下文</el-button>
                  <el-button link size="small" @click="copyLog(item)">复制</el-button>
                </div>
              </div>
              <div v-if="expandedRows.has(logKey(item, index))" class="log-detail">
                <div class="detail-section">
                  <strong>索引字段</strong>
                  <dl>
                    <template v-for="entry in detailEntries(item)" :key="entry.key">
                      <dt>{{ fieldDisplayName(entry.key) }}</dt><dd :title="displayFieldTitle(item, entry.key)">{{ displayFieldValue(item, entry.key) }}</dd>
                    </template>
                  </dl>
                </div>
                <div class="detail-section message-section">
                  <strong>日志正文</strong>
                  <pre>{{ item.message }}</pre>
                </div>
              </div>
            </article>
          </div>

          <el-empty v-if="!loading && !items.length && !errorMessage" description="当前条件下没有日志" :image-size="88" />
        </div>

        <div v-if="items.length" class="result-footer">
          <span v-if="tailing">内存保留最近 10,000 条，页面实时渲染最近 500 条</span>
          <template v-else>
            <span>共 {{ totalCount.toLocaleString() }} 条 · 当前 {{ pageStart.toLocaleString() }}-{{ pageEnd.toLocaleString() }} 条</span>
            <div class="page-actions">
              <el-button :disabled="currentPage <= 1 || loadingMore" @click="goPreviousPage">上一页</el-button>
              <span>第 {{ currentPage }} / {{ totalPages }} 页</span>
              <el-button :disabled="!hasMore || loadingMore" @click="goNextPage">下一页</el-button>
            </div>
          </template>
        </div>
      </div>
    </section>

    <el-dialog v-model="templatePickerVisible" width="760px" class="template-picker-dialog" destroy-on-close append-to-body>
      <template #header>
        <div class="template-dialog-title">
          <div class="template-dialog-icon"><el-icon><CollectionTag /></el-icon></div>
          <div>
            <strong>使用查询模板</strong>
            <span>选择模板后将替换当前查询条件并立即执行</span>
          </div>
        </div>
      </template>
      <div class="template-search-bar">
        <el-input v-model="templateKeyword" clearable placeholder="搜索模板名称、日志关键字或说明" @keyup.enter="searchTemplates">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button :loading="templateListLoading" @click="searchTemplates">搜索</el-button>
      </div>
      <div v-loading="templateListLoading" class="template-result-list">
        <div v-for="item in queryTemplates" :key="item.id" class="template-result-item" :class="{ active: selectedTemplateId === item.id }">
          <div class="template-result-main">
            <div class="template-result-name">
              <strong>{{ item.name }}</strong>
              <el-tag size="small" :type="templateSourceMode(item) === 'kubernetes' ? 'primary' : 'info'">{{ templateSourceLabel(item) }}</el-tag>
              <el-tag v-if="item.category" size="small" effect="plain">{{ item.category }}</el-tag>
            </div>
            <p>{{ item.description || templateQuerySummary(item) }}</p>
            <div class="template-result-meta">
              <span>时间：{{ templateRangeLabel(item.timeRange) }}</span>
              <span>{{ templateScopeSummary(item) }}</span>
            </div>
          </div>
          <el-button type="primary" class="primary-action" :loading="templateExecutingId === item.id" @click="handleTemplateSelection(item.id)">使用并查询</el-button>
        </div>
        <el-empty v-if="!templateListLoading && !queryTemplates.length" description="没有匹配的查询模板" :image-size="72" />
      </div>
      <div class="template-pagination">
        <span>共 {{ templatePagination.total }} 个模板</span>
        <el-pagination
          v-model:current-page="templatePagination.page"
          :page-size="templatePagination.pageSize"
          :total="templatePagination.total"
          layout="prev, pager, next"
          background
          small
          @current-change="loadQueryTemplates"
        />
      </div>
    </el-dialog>

    <el-drawer v-model="contextVisible" title="日志上下文" size="70%" destroy-on-close @opened="centerContextLog">
      <div class="context-toolbar">
        <span>当前日志前后 5 分钟</span>
        <el-tag size="small" :type="levelTagType(activeLog?.level)">{{ levelText(activeLog?.level) }}</el-tag>
      </div>
      <div v-loading="contextLoading" ref="contextListRef" class="context-list">
        <div
          v-for="(item, index) in contextItems"
          :key="`context-${index}-${logKey(item, index)}`"
          :data-active="isActiveContext(item) ? 'true' : 'false'"
          class="context-row"
          :class="{ active: isActiveContext(item) }"
        >
          <time>{{ formatTimestamp(item.timestamp) }}</time>
          <span class="level-badge" :class="levelClass(item.level)">{{ levelText(item.level) }}</span>
          <pre>{{ item.message }}</pre>
        </div>
        <el-empty v-if="!contextLoading && !contextItems.length" description="未查询到上下文日志" />
      </div>
    </el-drawer>

    <el-dialog v-model="exportVisible" title="异步导出日志" width="560px" destroy-on-close>
      <div v-if="!exportTask" class="export-form">
        <el-form label-position="top">
          <el-form-item label="文件格式">
            <el-segmented v-model="exportForm.format" :options="exportFormatOptions" />
          </el-form-item>
          <el-form-item label="最大导出条数">
            <el-input-number v-model="exportForm.maxRows" :min="1000" :max="1000000" :step="10000" controls-position="right" />
          </el-form-item>
        </el-form>
      </div>
      <div v-else class="export-progress">
        <div class="export-status-line">
          <el-tag :type="exportStatusType(exportTask.status)">{{ exportStatusText(exportTask.status) }}</el-tag>
          <span>{{ exportTask.exportedRows.toLocaleString() }} / {{ exportTask.maxRows.toLocaleString() }} 条</span>
        </div>
        <el-progress :percentage="exportTask.progress" :status="exportTask.status === 'failed' ? 'exception' : exportTask.status === 'completed' ? 'success' : undefined" />
        <dl>
		  <dt>执行次数</dt><dd>{{ exportTask.attemptCount }} / {{ exportTask.maxAttempts }}</dd>
          <dt>文件</dt><dd>{{ exportTask.fileName || '生成中' }}</dd>
          <dt>大小</dt><dd>{{ formatBytes(exportTask.fileSize) }}</dd>
          <dt>过期时间</dt><dd>{{ exportTask.expiresAt ? new Date(exportTask.expiresAt).toLocaleString() : '-' }}</dd>
        </dl>
		<el-alert v-if="exportTask.errorMessage" :type="exportTask.status === 'failed' ? 'error' : 'warning'" :closable="false" :title="exportTask.errorMessage" />
      </div>
      <template #footer>
        <el-button @click="exportVisible = false">关闭</el-button>
        <el-button v-if="!exportTask" type="primary" class="primary-action" :loading="exportCreating" @click="createExport">创建导出任务</el-button>
        <el-button v-else-if="exportTask.status === 'completed'" type="primary" class="primary-action" @click="downloadExport">下载文件</el-button>
        <el-button v-else-if="exportTask.status === 'failed' || exportTask.status === 'expired'" @click="resetExport">重新创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowRight, CollectionTag, Delete, Download, Filter, Plus, Refresh, Search, SwitchButton, VideoPlay } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import {
  createInternalLogExport,
  downloadInternalLogExport,
  getInternalLogAssets,
  getInternalLogExport,
  getInternalLogFields,
  getLogTemplate,
  getLogTemplates,
  getLogStorageClusters,
  queryInternalLogContext,
  queryInternalLogHistogram,
  queryInternalLogResourceOptions,
  queryInternalLogs,
  streamInternalLogs,
  type InternalLogQueryRequest,
  type InternalLogResourceOptions,
  type InternalKubernetesResourceOption,
  type LogExportTask,
  type LogItem,
  type LogPolicyTargetCluster,
  type LogPolicyTargetHost,
  type LogQueryTemplate,
  type LogQueryResponse,
  type LogStorageCluster,
} from '@/api/logcenter'

type FieldOption = { name: string; type: string; displayName?: string }
type DetailEntry = { key: string; value: any }
type ConditionRow = { id: number; field: string; operator: string; value: string; values: string[] }
type AssetOption = { id: number; label: string }
type ResourceSelectOption = { value: string; label: string }
type ConditionValueOption = { value: string; label: string }
type QueryTemplateDefinition = {
  version: number
  sourceMode: 'host' | 'kubernetes'
  level: string
  scope: {
    hostIds: number[]
    clusterIds: number[]
    environments: string[]
    services: string[]
    namespaces: string[]
    workloads: string[]
    pods: string[]
    containers: string[]
    nodes: string[]
  }
  filters: Array<{ field: string; operator: string; value: string | string[] }>
  filterLogic: 'and' | 'or'
  sort: 'asc' | 'desc'
  pageSize: number
}

const maxPageSize = 2000
const pageSizeOptions = [50, 100, 200, 500, 1000, maxPageSize]
const route = useRoute()
const routeNumberList = (plural: string, singular: string) => String(route.query[plural] || route.query[singular] || '').split(',').map(Number).filter(Boolean)
const routeStringList = (name: string) => String(route.query[name] || '').split(',').map(item => item.trim()).filter(Boolean)
const routeHostIds = ref(routeNumberList('hostIds', 'hostId'))
const routeClusterIds = ref(routeNumberList('clusterIds', 'clusterId'))
const routeHostId = ref(routeHostIds.value[0] || 0)
const routeClusterId = ref(routeClusterIds.value[0] || 0)
const routeStorageId = Number(route.query.storageId || 0)
const routePageSize = Math.min(Number(route.query.pageSize || 0), maxPageSize)
const routeSort = String(route.query.sort || '').toLowerCase() === 'asc' ? 'asc' : 'desc'
const routeFilterLogic = String(route.query.filterLogic || '').toLowerCase() === 'or' ? 'or' : 'and'
const routeSourceMode = ['host', 'kubernetes'].includes(String(route.query.sourceMode)) ? String(route.query.sourceMode) as 'host' | 'kubernetes' : undefined
const routeStart = route.query.start ? new Date(String(route.query.start)) : undefined
const routeEnd = route.query.end ? new Date(String(route.query.end)) : undefined
const chartRef = ref<HTMLElement>()
const contextListRef = ref<HTMLElement>()
const logStreamRef = ref<HTMLElement>()
const storages = ref<LogStorageCluster[]>([])
const assetHosts = ref<LogPolicyTargetHost[]>([])
const assetClusters = ref<LogPolicyTargetCluster[]>([])
const fieldOptions = ref<FieldOption[]>([])
const resourceOptions = ref<InternalLogResourceOptions>({
  hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [], kubernetesResources: [],
})
const queryTemplates = ref<LogQueryTemplate[]>([])
const selectedTemplateId = ref<number>()
const templateLoading = ref(false)
const templateListLoading = ref(false)
const templatePickerVisible = ref(false)
const templateKeyword = ref('')
const templateExecutingId = ref<number>()
const templatePagination = reactive({ page: 1, pageSize: 10, total: 0 })
const items = ref<LogItem[]>([])
const histogram = ref<Array<{ time: string; count: number }>>([])
const loading = ref(false)
const loadingMore = ref(false)
const resourceLoading = ref(false)
const contextLoading = ref(false)
const errorMessage = ref('')
const histogramError = ref('')
const nextCursor = ref('')
const hasMore = ref(false)
const currentPage = ref(1)
const pageCursors = ref<string[]>([''])
const durationMs = ref(0)
const quickRange = ref(15)
const timeRange = ref<[Date, Date]>(routeStart && routeEnd && !Number.isNaN(routeStart.getTime()) && !Number.isNaN(routeEnd.getTime()) ? [routeStart, routeEnd] : [new Date(Date.now() - 15 * 60 * 1000), new Date()])
const fieldKeyword = ref('')
const selectedFields = ref<string[]>(['timestamp', 'level', 'service', 'message'])
const wrapLines = ref(window.localStorage.getItem('opshub:log-query:wrap-lines') !== 'false')
const expandedRows = ref(new Set<string>())
const contextVisible = ref(false)
const contextItems = ref<LogItem[]>([])
const activeLog = ref<LogItem>()
const sourceMode = ref<'host' | 'kubernetes'>(routeSourceMode || (routeClusterId.value ? 'kubernetes' : 'host'))
const conditions = ref<ConditionRow[]>([])
const filterLogic = ref<'and' | 'or'>(routeFilterLogic)
const tailing = ref(false)
const tailConnected = ref(false)
const tailReceived = ref(0)
const tailReconnectAttempt = ref(0)
const tailBufferLimit = 10000
const tailRenderLimit = 500
const queryRenderLimit = 2000
const tailStatusText = computed(() => {
  if (tailConnected.value) return '实时接收中'
  if (tailReconnectAttempt.value > 0) return `连接中断，正在重连（第 ${tailReconnectAttempt.value} 次）`
  return '正在连接'
})
const exportVisible = ref(false)
const exportCreating = ref(false)
const exportTask = ref<LogExportTask>()
const exportForm = reactive({ format: 'ndjson' as 'ndjson' | 'csv', maxRows: 100000 })
let chart: echarts.ECharts | undefined
let queryController: AbortController | undefined
let histogramController: AbortController | undefined
let resourceController: AbortController | undefined
let tailController: AbortController | undefined
let tailKnownIdentities = new Set<string>()
let exportTimer: number | undefined
let conditionSequence = 0
let querySequence = 0
let resourceSequence = 0
let timeRangeQueryTimer: ReturnType<typeof setTimeout> | undefined
let initialDataLoaded = false
let lastTemplateRouteSignature = ''
let applyingTemplate = false

const form = reactive({
  storageId: undefined as number | undefined,
  query: '*',
  limit: pageSizeOptions.includes(routePageSize) ? routePageSize : 200,
  sort: routeSort as 'asc' | 'desc',
})
const renderedItems = computed(() => {
  const limit = tailing.value ? tailRenderLimit : queryRenderLimit
  if (items.value.length <= limit) return items.value
  return form.sort === 'desc' ? items.value.slice(0, limit) : items.value.slice(-limit)
})
const scope = reactive({
  hostIds: [...routeHostIds.value] as number[],
  clusterIds: [...routeClusterIds.value] as number[],
  environments: routeStringList('environments'), services: routeStringList('services'), namespaces: routeStringList('namespaces'), workloads: routeStringList('workloads'),
  pods: routeStringList('pods'), containers: routeStringList('containers'), nodes: [], level: '',
})
const advancedOpen = ref(Boolean(scope.namespaces.length || scope.workloads.length || scope.pods.length || scope.containers.length))
const sortOptions = [{ label: '最新优先', value: 'desc' }, { label: '最早优先', value: 'asc' }]
const sourceModeOptions = [{ label: '主机', value: 'host' }, { label: 'Kubernetes', value: 'kubernetes' }]
const conditionOperators = [
  { label: '等于', value: 'eq' }, { label: '不等于', value: 'neq' }, { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' }, { label: '属于任意值（多值）', value: 'in' },
]
const filterLogicOptions = [{ label: '全部满足 AND', value: 'and' }, { label: '任一满足 OR', value: 'or' }]
const normalizeTemplateStrings = (value: unknown) => Array.isArray(value) ? value.map(String).map(item => item.trim()).filter(Boolean) : []
const normalizeTemplateNumbers = (value: unknown) => Array.isArray(value) ? value.map(Number).filter(Boolean) : []
const emptyTemplateDefinition = (): QueryTemplateDefinition => ({
  version: 1,
  sourceMode: 'host',
  level: '',
  scope: { hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [] },
  filters: [],
  filterLogic: 'and',
  sort: 'desc',
  pageSize: 200,
})
const parseTemplateDefinition = (template: LogQueryTemplate): QueryTemplateDefinition => {
  const fallback = emptyTemplateDefinition()
  try {
    const parsed = JSON.parse(template.variables || '') as Partial<QueryTemplateDefinition>
    if (!parsed || typeof parsed !== 'object' || Number(parsed.version) !== 1) return fallback
    const templateScope = parsed.scope || fallback.scope
    return {
      version: 1,
      sourceMode: parsed.sourceMode === 'kubernetes' ? 'kubernetes' : 'host',
      level: String(parsed.level || ''),
      scope: {
        hostIds: normalizeTemplateNumbers(templateScope.hostIds),
        clusterIds: normalizeTemplateNumbers(templateScope.clusterIds),
        environments: normalizeTemplateStrings(templateScope.environments),
        services: normalizeTemplateStrings(templateScope.services),
        namespaces: normalizeTemplateStrings(templateScope.namespaces),
        workloads: normalizeTemplateStrings(templateScope.workloads),
        pods: normalizeTemplateStrings(templateScope.pods),
        containers: normalizeTemplateStrings(templateScope.containers),
        nodes: normalizeTemplateStrings(templateScope.nodes),
      },
      filters: Array.isArray(parsed.filters) ? parsed.filters.filter(item => item?.field && item?.operator).map(item => ({
        field: String(item.field),
        operator: String(item.operator),
        value: Array.isArray(item.value) ? item.value.map(String) : String(item.value ?? ''),
      })) : [],
      filterLogic: parsed.filterLogic === 'or' ? 'or' : 'and',
      sort: parsed.sort === 'asc' ? 'asc' : 'desc',
      pageSize: pageSizeOptions.includes(Number(parsed.pageSize)) ? Number(parsed.pageSize) : 200,
    }
  } catch {
    return fallback
  }
}
const templateRangeMinutes = (value?: string) => {
  const matched = String(value || '15m').trim().match(/^(\d+)(m|h|d)$/i)
  if (!matched) return 15
  const amount = Number(matched[1])
  const unit = String(matched[2] || 'm').toLowerCase()
  return amount * (unit === 'd' ? 1440 : unit === 'h' ? 60 : 1)
}
const templateSourceMode = (template: LogQueryTemplate) => parseTemplateDefinition(template).sourceMode
const templateSourceLabel = (template: LogQueryTemplate) => templateSourceMode(template) === 'kubernetes' ? 'Kubernetes' : '主机'
const templateRangeLabel = (value?: string) => {
  const minutes = templateRangeMinutes(value)
  if (minutes % 1440 === 0) return `最近 ${minutes / 1440} 天`
  if (minutes % 60 === 0) return `最近 ${minutes / 60} 小时`
  return `最近 ${minutes} 分钟`
}
const templateQuerySummary = (template: LogQueryTemplate) => template.query && template.query !== '*' ? `日志正文包含：${template.query}` : '查询全部日志正文'
const templateScopeSummary = (template: LogQueryTemplate) => {
  const definition = parseTemplateDefinition(template)
  const parts: string[] = []
  if (definition.scope.hostIds.length) parts.push(`${definition.scope.hostIds.length} 台主机`)
  if (definition.scope.clusterIds.length) parts.push(`${definition.scope.clusterIds.length} 个集群`)
  if (definition.scope.namespaces.length) parts.push(`Namespace ${definition.scope.namespaces.join('、')}`)
  if (definition.scope.workloads.length) parts.push(`工作负载 ${definition.scope.workloads.join('、')}`)
  if (definition.scope.pods.length) parts.push(`${definition.scope.pods.length} 个 Pod`)
  if (definition.scope.containers.length) parts.push(`${definition.scope.containers.length} 个容器`)
  if (definition.scope.environments.length) parts.push(`环境 ${definition.scope.environments.join('、')}`)
  if (definition.scope.services.length) parts.push(`服务 ${definition.scope.services.join('、')}`)
  return parts.join(' · ') || '未限制资源范围'
}
const loadQueryTemplates = async (_page?: number) => {
  templateListLoading.value = true
  try {
    const result = await getLogTemplates({
      page: templatePagination.page,
      pageSize: templatePagination.pageSize,
      keyword: templateKeyword.value.trim(),
    }) as any
    queryTemplates.value = Array.isArray(result?.data) ? result.data : []
    templatePagination.total = Number(result?.total || 0)
  } finally {
    templateListLoading.value = false
  }
}
const searchTemplates = () => {
  templatePagination.page = 1
  void loadQueryTemplates()
}
const openTemplatePicker = () => {
  templatePickerVisible.value = true
  searchTemplates()
}
const templateByID = async (id: number) => {
  const cached = queryTemplates.value.find(item => Number(item.id) === id)
  return cached || await getLogTemplate(id) as any as LogQueryTemplate
}
const applyQueryTemplate = async (template: LogQueryTemplate, useRouteRange = false) => {
  if (!template.id) return
  if (tailing.value) stopTail(true)
  applyingTemplate = true
  resourceSequence += 1
  resourceController?.abort()
  resourceLoading.value = false
  try {
    const definition = parseTemplateDefinition(template)
    const preferredStorage = storages.value.find(item => item.id === Number(template.datasourceId) && item.enabled && item.initializedAt)
    const availableStorage = preferredStorage
      || storages.value.find(item => item.id === form.storageId && item.enabled && item.initializedAt)
      || storages.value.find(item => item.isPrimary && item.enabled && item.initializedAt)
      || storages.value.find(item => item.enabled && item.initializedAt)
    form.storageId = availableStorage?.id
    form.query = template.query?.trim() || '*'
    form.sort = definition.sort
    form.limit = definition.pageSize
    sourceMode.value = definition.sourceMode
    scope.hostIds = [...definition.scope.hostIds]
    scope.clusterIds = [...definition.scope.clusterIds]
    scope.environments = [...definition.scope.environments]
    scope.services = [...definition.scope.services]
    scope.namespaces = [...definition.scope.namespaces]
    scope.workloads = [...definition.scope.workloads]
    scope.pods = [...definition.scope.pods]
    scope.containers = [...definition.scope.containers]
    scope.nodes = []
    scope.level = definition.level
    filterLogic.value = definition.filterLogic
    const templateFilters = [...definition.filters]
    if (definition.scope.nodes.length) templateFilters.unshift({ field: 'nodeName', operator: 'in', value: [...definition.scope.nodes] })
    conditions.value = templateFilters.map(item => {
      const operator = String(item.operator || 'eq')
      const values = Array.isArray(item.value) ? item.value.map(String) : splitConditionValues(String(item.value || ''))
      return {
        id: ++conditionSequence,
        field: item.field,
        operator,
        value: operator === 'in' ? '' : String(Array.isArray(item.value) ? item.value[0] || '' : item.value || ''),
        values: operator === 'in' ? values : [],
      }
    })
    routeHostIds.value = []
    routeClusterIds.value = []
    routeHostId.value = 0
    routeClusterId.value = 0
    selectedTemplateId.value = Number(template.id)
    const minutes = templateRangeMinutes(template.timeRange)
    const routeStartValue = useRouteRange && route.query.start ? new Date(String(route.query.start)) : undefined
    const routeEndValue = useRouteRange && route.query.end ? new Date(String(route.query.end)) : undefined
    const hasValidRouteRange = routeStartValue && routeEndValue && !Number.isNaN(routeStartValue.getTime()) && !Number.isNaN(routeEndValue.getTime())
    const end = hasValidRouteRange ? routeEndValue! : new Date()
    const start = hasValidRouteRange ? routeStartValue! : new Date(end.getTime() - minutes * 60 * 1000)
    quickRange.value = hasValidRouteRange ? 0 : minutes
    timeRange.value = [start, end]
    advancedOpen.value = Boolean(conditions.value.length || scope.namespaces.length || scope.workloads.length || scope.pods.length || scope.containers.length)
    await loadFields()
    await runQuery({ preserveResourceScope: true })
    ElMessage.success(`已执行查询模板“${template.name}”`)
  } finally {
    applyingTemplate = false
  }
}
const handleTemplateSelection = async (value?: string | number) => {
  const id = Number(value || 0)
  if (!id) return
  templateLoading.value = true
  templateExecutingId.value = id
  try {
    await applyQueryTemplate(await templateByID(id))
    templatePickerVisible.value = false
  } finally {
    templateLoading.value = false
    templateExecutingId.value = undefined
  }
}
const applyRouteTemplateIfNeeded = async (force = false) => {
  const id = Number(route.query.templateId || 0)
  if (!id || (!initialDataLoaded && !force)) return false
  const signature = route.fullPath
  if (!force && signature === lastTemplateRouteSignature) return true
  lastTemplateRouteSignature = signature
  templateLoading.value = true
  try {
    await applyQueryTemplate(await templateByID(id), true)
    return true
  } catch (error) {
    if (lastTemplateRouteSignature === signature) lastTemplateRouteSignature = ''
    throw error
  } finally {
    templateLoading.value = false
  }
}
const exportFormatOptions = [{ label: 'NDJSON', value: 'ndjson' }, { label: 'CSV', value: 'csv' }]
const splitConditionValues = (value: string) => value.split(/[,，;；\n\r\t]+/).map(item => item.trim()).filter(Boolean)
const normalizedConditionValues = (condition: ConditionRow) => {
  if (condition.operator !== 'in') return condition.value.trim() ? [condition.value.trim()] : []
  return Array.from(new Set(condition.values.flatMap(splitConditionValues)))
}
const conditionHasValue = (condition: ConditionRow) => normalizedConditionValues(condition).length > 0
const conditionDisplayValue = (condition: ConditionRow) => normalizedConditionValues(condition).join('、')
const activeStorage = computed(() => storages.value.find(item => item.id === form.storageId))
const totalCount = computed(() => histogram.value.reduce((sum, item) => sum + Number(item.count || 0), 0))
const totalPages = computed(() => Math.max(1, totalCount.value ? Math.ceil(totalCount.value / form.limit) : currentPage.value + (hasMore.value ? 1 : 0)))
const pageStart = computed(() => items.value.length ? (currentPage.value - 1) * form.limit + 1 : 0)
const pageEnd = computed(() => items.value.length ? pageStart.value + items.value.length - 1 : 0)
const conditionFields = computed(() => fieldOptions.value.filter(item => item.name !== 'timestamp'))
const activeFilterCount = computed(() => (
  1
  + scope.hostIds.length + scope.clusterIds.length + scope.environments.length + scope.services.length
  + scope.namespaces.length + scope.workloads.length + scope.pods.length + scope.containers.length
  + (scope.level ? 1 : 0)
  + conditions.value.filter(item => item.field && conditionHasValue(item)).length
))
const filteredFields = computed(() => {
  const keyword = fieldKeyword.value.trim().toLowerCase()
  return keyword ? fieldOptions.value.filter(item => `${item.name} ${item.displayName || ''}`.toLowerCase().includes(keyword)) : fieldOptions.value
})
const formatRangeLabel = computed(() => `${timeRange.value[0].toLocaleString()} - ${timeRange.value[1].toLocaleString()}`)
const activeFilterChips = computed(() => {
  const chips: Array<{ key: string; label: string; value: string }> = []
  chips.push({ key: 'source', label: '来源', value: sourceMode.value === 'host' ? '主机' : 'Kubernetes' })
  for (const id of scope.hostIds) chips.push({ key: `host-${id}`, label: '主机', value: hostLabel(id) })
  for (const id of scope.clusterIds) chips.push({ key: `cluster-${id}`, label: '集群', value: clusterLabel(id) })
  for (const value of scope.environments) chips.push({ key: `env-${value}`, label: '环境', value })
  for (const value of scope.services) chips.push({ key: `service-${value}`, label: '服务', value })
  for (const value of scope.namespaces) chips.push({ key: `namespace-${value}`, label: 'Namespace', value })
  for (const value of scope.workloads) chips.push({ key: `workload-${value}`, label: '工作负载', value })
  for (const value of scope.pods) chips.push({ key: `pod-${value}`, label: 'Pod', value })
  for (const value of scope.containers) chips.push({ key: `container-${value}`, label: '容器', value })
  if (scope.level) chips.push({ key: `level-${scope.level}`, label: '级别', value: scope.level })
  for (const condition of conditions.value) {
    if (!condition.field || !conditionHasValue(condition)) continue
    chips.push({ key: `condition-${condition.id}`, label: fieldDisplayName(condition.field), value: `${operatorDisplayName(condition.operator)} ${conditionDisplayValue(condition)}` })
  }
  return chips
})
const availableHosts = computed<AssetOption[]>(() => mergeAssetOptions(
  assetHosts.value.map(item => ({ id: Number(item.id), label: `${item.name || item.ip || `主机 #${item.id}`}${item.ip ? ` · ${item.ip}` : ''}` })),
  resourceOptions.value.hostIds,
  '主机',
))
const availableClusters = computed<AssetOption[]>(() => mergeAssetOptions(
  assetClusters.value.map(item => ({ id: Number(item.id), label: item.alias || item.name || `集群 #${item.id}` })),
  resourceOptions.value.clusterIds,
  '集群',
))
const conditionValueOptions = (field: string): ConditionValueOption[] => {
  const fromValues = (values: string[]) => Array.from(new Set(values.filter(Boolean)))
    .sort((left, right) => left.localeCompare(right, 'zh-CN'))
    .map(value => ({ value, label: value }))
  switch (field) {
    case 'hostId':
      return availableHosts.value.map(item => ({ value: String(item.id), label: item.label }))
    case 'clusterId':
      return availableClusters.value.map(item => ({ value: String(item.id), label: item.label }))
    case 'environment':
      return fromValues(resourceOptions.value.environments)
    case 'service':
      return fromValues(resourceOptions.value.services)
    case 'namespace':
      return fromValues(resourceOptions.value.namespaces)
    case 'workloadKind':
      return fromValues(resourceOptions.value.kubernetesResources.map(item => item.workloadKind))
    case 'workloadName':
      return fromValues(resourceOptions.value.workloads)
    case 'podName':
      return fromValues(resourceOptions.value.pods)
    case 'containerName':
      return fromValues(resourceOptions.value.containers)
    case 'nodeName':
      return fromValues(resourceOptions.value.nodes)
    case 'level':
      return fromValues(['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'])
    case 'assetType':
      return [{ value: 'host', label: '主机' }, { value: 'kubernetes', label: 'Kubernetes' }]
    default:
      return []
  }
}
const pathMatchesClusters = (item: InternalKubernetesResourceOption) => !scope.clusterIds.length || scope.clusterIds.some(id => String(id) === item.clusterId)
const pathMatchesNamespaces = (item: InternalKubernetesResourceOption) => !scope.namespaces.length || scope.namespaces.includes(item.namespace)
const pathMatchesWorkloads = (item: InternalKubernetesResourceOption) => !scope.workloads.length || scope.workloads.includes(item.workloadName)
const pathMatchesPods = (item: InternalKubernetesResourceOption) => !scope.pods.length || scope.pods.includes(item.podName)
const simpleResourceOptions = (values: string[]): ResourceSelectOption[] => Array.from(new Set(values.filter(Boolean)))
  .sort((left, right) => left.localeCompare(right, 'zh-CN'))
  .map(value => ({ value, label: value }))
const kubernetesNamespaceOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(pathMatchesClusters)
  return resourceOptions.value.kubernetesResources.length ? simpleResourceOptions(paths.map(item => item.namespace)) : simpleResourceOptions(resourceOptions.value.namespaces)
})
const kubernetesWorkloadOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item))
  if (!resourceOptions.value.kubernetesResources.length) return simpleResourceOptions(resourceOptions.value.workloads)
  const values = new Map<string, { kinds: Set<string>; namespaces: Set<string> }>()
  for (const item of paths) {
    if (!item.workloadName) continue
    const current = values.get(item.workloadName) || { kinds: new Set<string>(), namespaces: new Set<string>() }
    if (item.workloadKind) current.kinds.add(item.workloadKind)
    if (item.namespace) current.namespaces.add(item.namespace)
    values.set(item.workloadName, current)
  }
  return Array.from(values, ([value, metadata]) => {
    const kind = Array.from(metadata.kinds).sort().join('/')
    const namespaceHint = !scope.namespaces.length && metadata.namespaces.size === 1 ? ` · ${Array.from(metadata.namespaces)[0]}` : metadata.namespaces.size > 1 ? ` · ${metadata.namespaces.size} 个 Namespace` : ''
    return { value, label: `${kind ? `${kind} / ` : ''}${value}${namespaceHint}` }
  }).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
})
const kubernetesPodOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item) && pathMatchesWorkloads(item))
  if (!resourceOptions.value.kubernetesResources.length) return simpleResourceOptions(resourceOptions.value.pods)
  const values = new Map<string, Set<string>>()
  for (const item of paths) {
    if (!item.podName) continue
    const namespaces = values.get(item.podName) || new Set<string>()
    if (item.namespace) namespaces.add(item.namespace)
    values.set(item.podName, namespaces)
  }
  return Array.from(values, ([value, namespaces]) => ({
    value,
    label: !scope.namespaces.length && namespaces.size === 1 ? `${value} · ${Array.from(namespaces)[0]}` : namespaces.size > 1 ? `${value} · ${namespaces.size} 个 Namespace` : value,
  })).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
})
const kubernetesContainerOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item) && pathMatchesWorkloads(item) && pathMatchesPods(item))
  return resourceOptions.value.kubernetesResources.length ? simpleResourceOptions(paths.map(item => item.containerName)) : simpleResourceOptions(resourceOptions.value.containers)
})

const buildPayload = (cursor = '', skipHistory = false): InternalLogQueryRequest => {
  const queryFilters: InternalLogQueryRequest['filters'] = []
  for (const condition of conditions.value) {
    const values = normalizedConditionValues(condition)
    if (!condition.field || !values.length) continue
    queryFilters.push({
      field: condition.field,
      operator: condition.operator,
      value: condition.operator === 'in' ? values : values[0],
    })
  }
  return {
    storageId: form.storageId,
    start: timeRange.value[0].toISOString(),
    end: timeRange.value[1].toISOString(),
    query: form.query.trim() || '*',
    scope: {
      assetTypes: [sourceMode.value],
      hostIds: sourceMode.value === 'kubernetes' ? [] : [...scope.hostIds],
      clusterIds: sourceMode.value === 'host' ? [] : [...scope.clusterIds],
      environments: [...scope.environments], services: [...scope.services], namespaces: sourceMode.value === 'host' ? [] : [...scope.namespaces],
      workloads: sourceMode.value === 'host' ? [] : [...scope.workloads], pods: sourceMode.value === 'host' ? [] : [...scope.pods],
      containers: sourceMode.value === 'host' ? [] : [...scope.containers], nodes: [],
      levels: scope.level ? [scope.level] : [],
    },
    filters: queryFilters,
    filterLogic: filterLogic.value,
    sort: form.sort,
    limit: form.limit,
    cursor,
    skipHistory,
  }
}

const buildResourceOptionsPayload = (): InternalLogQueryRequest => {
  const payload = buildPayload('', true)
  payload.scope = { ...payload.scope, hostIds: [], clusterIds: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [] }
  return payload
}

const mergeAssetOptions = (items: AssetOption[], ids: string[], prefix: string) => {
  const result = new Map<number, string>()
  for (const item of items) result.set(item.id, item.label)
  for (const value of ids) {
    const id = Number(value)
    if (id && !result.has(id)) result.set(id, `${prefix} #${id}`)
  }
  return Array.from(result, ([id, label]) => ({ id, label })).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
}
const hostLabel = (id: number) => availableHosts.value.find(item => item.id === id)?.label || `主机 #${id}`
const clusterLabel = (id: number) => availableClusters.value.find(item => item.id === id)?.label || `集群 #${id}`
const clearHostScope = () => { scope.hostIds = scope.hostIds.filter(id => id !== routeHostId.value); routeHostId.value = 0 }
const clearClusterScope = () => { scope.clusterIds = scope.clusterIds.filter(id => id !== routeClusterId.value); routeClusterId.value = 0 }
const retainAvailableValues = (values: string[], options: ResourceSelectOption[]) => {
  const available = new Set(options.map(item => item.value))
  return values.filter(value => available.has(value))
}
const handleClusterScopeChange = () => {
  scope.namespaces = retainAvailableValues(scope.namespaces, kubernetesNamespaceOptions.value)
  handleNamespaceScopeChange()
}
const handleNamespaceScopeChange = () => {
  scope.workloads = retainAvailableValues(scope.workloads, kubernetesWorkloadOptions.value)
  handleWorkloadScopeChange()
}
const handleWorkloadScopeChange = () => {
  scope.pods = retainAvailableValues(scope.pods, kubernetesPodOptions.value)
  handlePodScopeChange()
}
const handlePodScopeChange = () => {
  scope.containers = retainAvailableValues(scope.containers, kubernetesContainerOptions.value)
}
const handleSourceModeChange = () => {
  if (sourceMode.value === 'host') {
    scope.clusterIds = []; scope.namespaces = []; scope.workloads = []; scope.pods = []; scope.containers = []; scope.nodes = []
  } else {
    scope.hostIds = []
  }
  void loadResourceOptions()
}
const clearScopeFilters = () => {
  scope.hostIds = []; scope.clusterIds = []; scope.environments = []; scope.services = []; scope.namespaces = []
  scope.workloads = []; scope.pods = []; scope.containers = []; scope.nodes = []; scope.level = ''
  conditions.value = []; filterLogic.value = 'and'
}
const openConditionBuilder = () => {
  advancedOpen.value = true
  if (!conditions.value.length) addCondition()
}

const loadInitialData = async () => {
  try {
    const [storageResult, assetResult] = await Promise.all([
      getLogStorageClusters({ enabled: true }) as any,
      getInternalLogAssets() as any,
    ])
    storages.value = storageResult || []
    assetHosts.value = assetResult?.hosts || []
    assetClusters.value = assetResult?.clusters || []
    if (await applyRouteTemplateIfNeeded(true)) return
    const available = storages.value.find(item => item.isPrimary && item.initializedAt) || storages.value.find(item => item.initializedAt)
    form.storageId = storages.value.find(item => item.id === routeStorageId)?.id || available?.id
    if (route.query.q) form.query = String(route.query.q)
    await loadFields()
    if (route.query.filters) {
      try {
        const filters = JSON.parse(String(route.query.filters)) as Array<{ field?: string; operator?: string; value?: any }>
        conditions.value = filters.filter(item => item.field && String(item.field) !== 'level').map(item => {
          const operator = String(item.operator || 'eq')
          const rawValues = Array.isArray(item.value) ? item.value.map(String) : splitConditionValues(String(item.value ?? ''))
          return {
            id: ++conditionSequence,
            field: String(item.field),
            operator,
            value: operator === 'in' ? '' : String(Array.isArray(item.value) ? item.value[0] ?? '' : item.value ?? ''),
            values: operator === 'in' ? rawValues : [],
          }
        })
        const levelFilter = filters.find(item => item.field === 'level')
        if (levelFilter) scope.level = String(levelFilter.value || '')
        if (conditions.value.length) advancedOpen.value = true
      } catch {}
    }
    if (form.storageId) await runQuery()
  } finally {
    initialDataLoaded = true
  }
}

const loadFields = async () => {
  if (!form.storageId) { fieldOptions.value = []; return }
  fieldOptions.value = (await getInternalLogFields({ storageId: form.storageId }) as any) || []
}

const isCanceledRequest = (error: any) => error?.code === 'ERR_CANCELED'
  || ['CanceledError', 'AbortError'].includes(String(error?.name || ''))
  || ['canceled', 'cancelled'].includes(String(error?.message || '').toLowerCase())

const requestErrorText = (error: any, fallback: string) => {
  const data = error?.response?.data
  const serverMessage = typeof data === 'object' ? data?.message || data?.msg || data?.error || data?.detail : ''
  if (serverMessage) return String(serverMessage)
  const message = String(error?.message || '')
  if (error?.code === 'ECONNABORTED' || /timeout/i.test(message)) return `${fallback}：请求超时，请缩小时间范围后重试`
  if (!error?.response || /network error/i.test(message)) return `${fallback}：日志服务连接暂时中断，当前结果已保留，请重试`
  return message || fallback
}

const loadResourceOptions = async (preserveScope = false) => {
  if (!form.storageId) return
  const sequence = ++resourceSequence
  resourceController?.abort()
  const controller = new AbortController()
  resourceController = controller
  resourceLoading.value = true
  try {
    const result = await queryInternalLogResourceOptions(buildResourceOptionsPayload(), controller.signal) as any
    if (sequence !== resourceSequence) return
    resourceOptions.value = {
      hostIds: result?.hostIds || [], clusterIds: result?.clusterIds || [], environments: result?.environments || [],
      services: result?.services || [], namespaces: result?.namespaces || [], workloads: result?.workloads || [],
      pods: result?.pods || [], containers: result?.containers || [], nodes: result?.nodes || [],
      kubernetesResources: result?.kubernetesResources || [],
    }
    if (!preserveScope) handleClusterScopeChange()
  } catch (error: any) {
    if (!isCanceledRequest(error)) console.warn('读取日志资源选项失败', error)
  } finally {
    if (sequence === resourceSequence) resourceLoading.value = false
  }
}

const runQuery = async (options?: { preserveResourceScope?: boolean }) => {
  if (!form.storageId) { ElMessage.warning('请先在日志库中配置并初始化 ClickHouse'); return }
  if (tailing.value) return
  const sequence = ++querySequence
  queryController?.abort(); histogramController?.abort()
  const currentQueryController = new AbortController()
  const currentHistogramController = new AbortController()
  queryController = currentQueryController; histogramController = currentHistogramController
  loadingMore.value = false; loading.value = true; errorMessage.value = ''; histogramError.value = ''
  try {
    const [queryState, histogramState] = await Promise.allSettled([
      queryInternalLogs(buildPayload(), currentQueryController.signal) as any,
      queryInternalLogHistogram(buildPayload('', true), currentHistogramController.signal) as any,
    ])
    if (sequence !== querySequence) return
    let querySucceeded = false
    if (queryState.status === 'fulfilled') {
      const result = queryState.value as LogQueryResponse
      pageCursors.value = ['']
      applyPageResult(result, 1)
      durationMs.value = Number(result.durationMs || 0)
      querySucceeded = true
    } else if (!isCanceledRequest(queryState.reason)) {
      errorMessage.value = requestErrorText(queryState.reason, '日志查询失败')
    }
    if (histogramState.status === 'fulfilled') {
      histogram.value = histogramState.value?.histogram || []
      renderChart()
    } else if (!isCanceledRequest(histogramState.reason)) {
      histogramError.value = requestErrorText(histogramState.reason, '趋势统计暂时不可用')
    }
    if (querySucceeded) void loadResourceOptions(Boolean(options?.preserveResourceScope))
  } finally {
    if (sequence === querySequence) loading.value = false
  }
}

const applyPageResult = (result: LogQueryResponse, page: number) => {
  items.value = result.items || []
  currentPage.value = page
  nextCursor.value = result.nextCursor || ''
  hasMore.value = Boolean(result.hasMore)
  expandedRows.value = new Set()
  if (result.nextCursor) pageCursors.value[page] = result.nextCursor
  else pageCursors.value.splice(page)
  nextTick(() => { if (logStreamRef.value) logStreamRef.value.scrollTop = 0 })
}

const loadPage = async (page: number, cursor: string) => {
  if (loadingMore.value || page < 1) return
  const sequence = ++querySequence
  queryController?.abort()
  const controller = new AbortController()
  queryController = controller
  loading.value = false; loadingMore.value = true
  errorMessage.value = ''
  try {
    const result = await queryInternalLogs(buildPayload(cursor, true), controller.signal) as any as LogQueryResponse
    if (sequence !== querySequence) return
    applyPageResult(result, page)
    durationMs.value = Number(result.durationMs || 0)
  } catch (error: any) {
    if (sequence === querySequence && !isCanceledRequest(error)) errorMessage.value = requestErrorText(error, '日志分页查询失败')
  } finally {
    if (sequence === querySequence) loadingMore.value = false
  }
}

const goPreviousPage = () => {
  if (currentPage.value <= 1) return
  const targetPage = currentPage.value - 1
  void loadPage(targetPage, pageCursors.value[targetPage - 1] || '')
}

const goNextPage = () => {
  if (!hasMore.value || !nextCursor.value) return
  const targetPage = currentPage.value + 1
  pageCursors.value[targetPage - 1] = nextCursor.value
  void loadPage(targetPage, nextCursor.value)
}

const scheduleTimeRangeQuery = () => {
  if (!form.storageId || tailing.value) return
  if (timeRangeQueryTimer) clearTimeout(timeRangeQueryTimer)
  timeRangeQueryTimer = setTimeout(() => {
    timeRangeQueryTimer = undefined
    void runQuery()
  }, 180)
}
const applyQuickRange = () => {
  if (quickRange.value <= 0) return
  const end = new Date()
  timeRange.value = [new Date(end.getTime() - quickRange.value * 60 * 1000), end]
  scheduleTimeRangeQuery()
}
const handleTimeRangeChange = () => {
  quickRange.value = 0
  scheduleTimeRangeQuery()
}

const addCondition = () => {
  conditionSequence += 1
  conditions.value.push({ id: conditionSequence, field: '', operator: 'eq', value: '', values: [] })
}
const removeCondition = (id: number) => { conditions.value = conditions.value.filter(item => item.id !== id) }
const handleConditionOperatorChange = (condition: ConditionRow) => {
  if (condition.operator === 'in') {
    condition.values = Array.from(new Set([...condition.values, ...splitConditionValues(condition.value)]))
    condition.value = ''
  } else if (!condition.value && condition.values.length) {
    condition.value = condition.values[0] || ''
  }
}
const handleConditionValuesPaste = (event: ClipboardEvent, condition: ConditionRow) => {
  const values = splitConditionValues(event.clipboardData?.getData('text') || '')
  if (values.length <= 1) return
  event.preventDefault()
  condition.values = Array.from(new Set([...condition.values, ...values]))
}

const renderChart = () => {
  nextTick(() => {
    if (!chartRef.value) return
    chart ||= echarts.init(chartRef.value)
    chart.setOption({
      animation: false,
      grid: { left: 46, right: 18, top: 18, bottom: 30 },
      tooltip: { trigger: 'axis', formatter: (params: any) => `${params?.[0]?.axisValueLabel || ''}<br/>日志数：${params?.[0]?.value || 0}` },
      xAxis: { type: 'category', boundaryGap: false, data: histogram.value.map(item => formatChartTime(item.time)), axisLine: { lineStyle: { color: '#d8dee8' } }, axisLabel: { color: '#7b8494', hideOverlap: true } },
      yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: '#edf0f5' } }, axisLabel: { color: '#7b8494' } },
      series: [{ type: 'line', data: histogram.value.map(item => item.count), showSymbol: false, smooth: 0.25, lineStyle: { color: '#2563eb', width: 2 }, areaStyle: { color: 'rgba(37,99,235,.10)' } }],
    }, true)
  })
}

const toggleField = (name: string) => {
  selectedFields.value = selectedFields.value.includes(name) ? selectedFields.value.filter(item => item !== name) : [...selectedFields.value, name]
}
const toggleExpanded = (item: LogItem, index: number) => {
  const key = logKey(item, index)
  const next = new Set(expandedRows.value)
  next.has(key) ? next.delete(key) : next.add(key)
  expandedRows.value = next
}
const detailEntries = (item: LogItem): DetailEntry[] => {
  const all = { ...item.labels, ...item.fields }
  const entries = Object.entries(all).map(([key, value]) => ({ key, value }))
  if (!selectedFields.value.length) return entries
  return entries.sort((left, right) => Number(selectedFields.value.includes(right.key)) - Number(selectedFields.value.includes(left.key)))
}
const openContext = async (item: LogItem) => {
  activeLog.value = item; contextVisible.value = true; contextLoading.value = true; contextItems.value = []
  try {
    const result = await queryInternalLogContext({
      storageId: form.storageId, timestamp: item.timestamp, message: item.message, level: item.level,
      labels: item.labels, fields: item.fields, beforeSeconds: 300, afterSeconds: 300, limit: 201,
    }) as any as LogQueryResponse
    contextItems.value = result.items || []
  } finally { contextLoading.value = false; await nextTick(); centerContextLog() }
}
const centerContextLog = () => {
  nextTick(() => contextListRef.value?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'center' }))
}
const isActiveContext = (item: LogItem) => item.contextSelected === true
const copyLog = async (item: LogItem) => { await navigator.clipboard.writeText(`${item.timestamp} ${levelText(item.level)} ${item.message}`); ElMessage.success('日志已复制') }

const toggleTail = () => tailing.value ? stopTail() : startTail()
const trimTailBuffer = () => {
  if (items.value.length <= tailBufferLimit) return
  const removed = form.sort === 'desc'
    ? items.value.splice(tailBufferLimit)
    : items.value.splice(0, items.value.length - tailBufferLimit)
  for (const item of removed) tailKnownIdentities.delete(logIdentity(item))
}
const startTail = () => {
  if (!form.storageId) { ElMessage.warning('请先选择日志存储'); return }
  stopTail(true)
  querySequence += 1
  queryController?.abort()
  histogramController?.abort()
  loading.value = false
  const controller = new AbortController()
  tailController = controller
  tailing.value = true; tailConnected.value = false; tailReconnectAttempt.value = 0; tailReceived.value = 0; errorMessage.value = ''
  tailKnownIdentities = new Set(items.value.map(item => logIdentity(item)))

  const payload = buildPayload('', true)
  const waitForReconnect = (delayMs: number) => new Promise<void>(resolve => {
    const timer = window.setTimeout(resolve, delayMs)
    controller.signal.addEventListener('abort', () => { window.clearTimeout(timer); resolve() }, { once: true })
  })

  const runTailConnection = async () => {
    let attempt = 0
    while (!controller.signal.aborted && tailController === controller) {
      let terminalStatus = 0
      try {
        await streamInternalLogs(payload, {
          signal: controller.signal,
          onReady: () => {
            if (tailController !== controller) return
            attempt = 0
            tailReconnectAttempt.value = 0
            tailConnected.value = true
            errorMessage.value = ''
          },
          onLogs: received => {
            if (tailController !== controller) return
            const receivedItems = received.items || []
            const lastItem = receivedItems[receivedItems.length - 1]
            if (received.cursor) payload.cursor = received.cursor
            if (lastItem?.timestamp) payload.start = lastItem.timestamp

            const incoming: LogItem[] = []
            for (const item of receivedItems) {
              const identity = logIdentity(item)
              if (tailKnownIdentities.has(identity)) continue
              tailKnownIdentities.add(identity)
              incoming.push(item)
            }
            if (!incoming.length) return
            tailConnected.value = true
            tailReconnectAttempt.value = 0
            tailReceived.value += incoming.length
            if (form.sort === 'desc') {
              items.value.unshift(...incoming.slice().reverse())
            } else {
              items.value.push(...incoming)
            }
            trimTailBuffer()
          },
          onError: payloadError => {
            if (tailController !== controller) return
            tailConnected.value = false
            tailReconnectAttempt.value = Math.max(1, tailReconnectAttempt.value)
            console.warn('实时 Tail 查询暂时失败，准备重连', payloadError.message || '')
          },
          onEnd: () => {
            if (tailController === controller) tailConnected.value = false
          },
        })
      } catch (error: any) {
        if (controller.signal.aborted || tailController !== controller) return
        terminalStatus = Number(error?.status || 0)
        if (![401, 403].includes(terminalStatus)) {
          console.warn('实时 Tail 连接中断，准备重连', error)
        }
      }

      if (controller.signal.aborted || tailController !== controller) return
      if ([401, 403].includes(terminalStatus)) {
        errorMessage.value = terminalStatus === 401 ? '登录已过期，请重新登录后再使用 Tail' : '没有实时 Tail 权限'
        tailConnected.value = false
        tailing.value = false
        return
      }
      attempt += 1
      tailReconnectAttempt.value = attempt
      tailConnected.value = false
      await waitForReconnect(Math.min(30000, 1000 * (2 ** Math.min(attempt - 1, 5))))
    }
  }

  void runTailConnection().catch((error: any) => {
    if (controller.signal.aborted || tailController !== controller) return
    errorMessage.value = error?.message || '实时 Tail 连接失败'
    tailConnected.value = false
    tailing.value = false
  })
}
const stopTail = (silent = false) => {
  tailController?.abort(); tailController = undefined; tailing.value = false; tailConnected.value = false; tailReconnectAttempt.value = 0
  tailKnownIdentities.clear()
  if (!silent) ElMessage.success('实时 Tail 已停止')
}

const openExportDialog = () => {
  clearExportTimer(); exportTask.value = undefined; exportVisible.value = true
}
const createExport = async () => {
  if (!form.storageId) return
  if (!Number.isInteger(exportForm.maxRows) || exportForm.maxRows < 1000 || exportForm.maxRows > 1000000) {
    ElMessage.warning('最大导出条数必须在 1,000 到 1,000,000 之间')
    return
  }
  exportCreating.value = true
  try {
    exportTask.value = await createInternalLogExport({ ...buildPayload('', true), format: exportForm.format, maxRows: exportForm.maxRows }) as any
    startExportPolling()
  } finally { exportCreating.value = false }
}
const startExportPolling = () => {
  clearExportTimer()
  exportTimer = window.setInterval(async () => {
    if (!exportTask.value) return
    const task = await getInternalLogExport(exportTask.value.id) as any as LogExportTask
    exportTask.value = task
    if (['completed', 'failed', 'expired'].includes(task.status)) clearExportTimer()
  }, 1500)
}
const clearExportTimer = () => { if (exportTimer) window.clearInterval(exportTimer); exportTimer = undefined }
const resetExport = () => { clearExportTimer(); exportTask.value = undefined }
const downloadExport = async () => {
  if (!exportTask.value) return
  const blob = await downloadInternalLogExport(exportTask.value.id) as any as Blob
  const href = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = href
  anchor.download = exportTask.value.fileName || `opshub-logs-${exportTask.value.id}.${exportTask.value.format}`
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(href), 1000)
}
const exportStatusText = (status: string) => ({ pending: '等待中', running: '导出中', completed: '已完成', failed: '失败', expired: '已过期' }[status] || status)
const exportStatusType = (status: string) => status === 'completed' ? 'success' : status === 'failed' ? 'danger' : status === 'expired' ? 'info' : 'warning'
const formatBytes = (value?: number) => !value ? '0 B' : value >= 1073741824 ? `${(value / 1073741824).toFixed(2)} GiB` : value >= 1048576 ? `${(value / 1048576).toFixed(2)} MiB` : value >= 1024 ? `${(value / 1024).toFixed(1)} KiB` : `${value} B`
const highlightMessage = (message: string) => {
  const escaped = escapeHTML(message)
  const keyword = form.query.trim()
  if (!keyword || keyword === '*') return escaped
  return escaped.replace(new RegExp(escapeRegExp(escapeHTML(keyword)), 'gi'), value => `<mark>${value}</mark>`)
}
const escapeHTML = (value: string) => value.replace(/[&<>"]/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[char] || char))
const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const logIdentity = (item: LogItem) => `${item.timestamp}-${item.fields?.fingerprint || ''}-${item.fields?.sequence || ''}-${item.message}`
const logKey = (item: LogItem, index: number) => `${item.timestamp}-${item.fields?.fingerprint || index}-${item.fields?.sequence || ''}`
const resourceFieldDisplayNames: Record<string, string> = { assetId: '资产', hostId: '主机', clusterId: '集群' }
const fieldDisplayName = (name: string) => resourceFieldDisplayNames[name] || fieldOptions.value.find(item => item.name === name)?.displayName || name
const showTimestamp = computed(() => selectedFields.value.includes('timestamp'))
const showLevel = computed(() => selectedFields.value.includes('level'))
const showMessage = computed(() => selectedFields.value.includes('message'))
const selectedColumnFields = computed(() => selectedFields.value
  .filter(name => !['timestamp', 'message', 'level'].includes(name))
  .map(name => ({ name, label: fieldDisplayName(name) })))
const logTableStyle = computed(() => {
  const columns = ['24px']
  if (showTimestamp.value) columns.push('160px')
  if (showLevel.value) columns.push('58px')
  selectedColumnFields.value.forEach(() => columns.push('minmax(125px, 155px)'))
  if (showMessage.value) columns.push('minmax(360px, 1fr)')
  columns.push('76px')
  const fixedWidth = 100
    + (showTimestamp.value ? 160 : 0)
    + (showLevel.value ? 58 : 0)
    + selectedColumnFields.value.length * 155
    + (showMessage.value ? 360 : 0)
    + Math.max(0, columns.length - 1) * 8
  return {
    '--log-table-columns': columns.join(' '),
    '--log-table-min-width': `${Math.max(480, fixedWidth)}px`,
  }
})
const rawFieldValue = (item: LogItem, name: string) => ({ ...item.labels, ...item.fields } as Record<string, any>)[name]
const findAssetLabel = (items: AssetOption[], value: any) => {
  const id = String(value ?? '').trim()
  if (!id || id === '0') return ''
  return items.find(item => String(item.id) === id)?.label || ''
}
const displayResourceFieldValue = (item: LogItem, name: string, value: any) => {
  const id = String(value ?? '').trim()
  if (!id || id === '0') return '-'
  if (name === 'hostId') return findAssetLabel(availableHosts.value, id) || `主机 #${id}`
  if (name === 'clusterId') return findAssetLabel(availableClusters.value, id) || `集群 #${id}`
  const assetType = String(rawFieldValue(item, 'assetType') || '').trim().toLowerCase()
  if (assetType === 'host') return findAssetLabel(availableHosts.value, id) || `主机 #${id}`
  if (assetType === 'kubernetes' || assetType === 'k8s') return findAssetLabel(availableClusters.value, id) || `集群 #${id}`
  const host = findAssetLabel(availableHosts.value, id)
  const cluster = findAssetLabel(availableClusters.value, id)
  return host || cluster || `资产 #${id}`
}
const displayFieldValue = (item: LogItem, name: string) => {
  const value = rawFieldValue(item, name)
  if (['assetId', 'hostId', 'clusterId'].includes(name)) return displayResourceFieldValue(item, name, value)
  return value === undefined || value === null || value === '' ? '-' : formatValue(value)
}
const displayFieldTitle = (item: LogItem, name: string) => {
  const display = displayFieldValue(item, name)
  if (!['assetId', 'hostId', 'clusterId'].includes(name)) return display
  const raw = String(rawFieldValue(item, name) ?? '').trim()
  return raw && raw !== '0' ? `${display}（ID: ${raw}）` : display
}
const operatorDisplayName = (operator: string) => conditionOperators.find(item => item.value === operator)?.label || operator || '等于'
const levelText = (value?: string) => (value || 'INFO').toUpperCase()
const levelClass = (value?: string) => `level-${levelText(value).toLowerCase()}`
const levelTagType = (value?: string) => levelText(value) === 'ERROR' ? 'danger' : levelText(value) === 'WARN' ? 'warning' : 'info'
const formatTimestamp = (value: string) => value ? new Date(value).toLocaleString() : '-'
const formatChartTime = (value: string) => new Date(value).toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
const formatValue = (value: any) => typeof value === 'object' ? JSON.stringify(value) : String(value ?? '')

watch(() => form.sort, () => items.value.length && !tailing.value && !applyingTemplate && runQuery())
watch(wrapLines, value => window.localStorage.setItem('opshub:log-query:wrap-lines', String(value)))
watch(
  () => route.fullPath,
  () => {
    if (initialDataLoaded) void applyRouteTemplateIfNeeded()
  },
)
onMounted(() => { loadInitialData(); window.addEventListener('resize', resizeChart) })
onActivated(() => {
  nextTick(resizeChart)
  if (initialDataLoaded) void applyRouteTemplateIfNeeded()
})
onBeforeUnmount(() => {
  querySequence += 1; resourceSequence += 1
  if (timeRangeQueryTimer) clearTimeout(timeRangeQueryTimer)
  queryController?.abort(); histogramController?.abort(); resourceController?.abort(); stopTail(true); clearExportTimer(); chart?.dispose(); window.removeEventListener('resize', resizeChart)
})
const resizeChart = () => chart?.resize()
</script>

<style scoped>
.internal-query-page { display: flex; flex-direction: column; gap: 12px; }
.head-actions, .query-context-row, .context-control, .filter-section-head, .filter-title, .filter-head-actions, .kubernetes-band-head, .section-head, .section-head > div { display: flex; align-items: center; }
.head-actions, .context-control { gap: 10px; }
.template-trigger { min-width: 108px; }
.query-panel { padding: 0; overflow: hidden; }
.query-context-row { justify-content: space-between; gap: 20px; min-height: 64px; padding: 13px 18px; border-bottom: 1px solid #edf0f4; background: #fbfcfe; }
.control-label { flex: 0 0 auto; color: #667085; font-size: 12px; font-weight: 600; }
.storage-context { flex: 1 1 360px; min-width: 0; flex-wrap: wrap; }.storage-select { width: 250px; }.time-context { flex: 0 1 auto; justify-content: flex-end; flex-wrap: wrap; }.quick-range-select { width: 148px; }.time-context :deep(.time-picker) { width: 380px; }
.query-search-row { display: grid; grid-template-columns: minmax(300px, 1fr) 118px 126px auto; gap: 10px; align-items: center; padding: 16px 18px; }
.query-text-input :deep(.el-input__wrapper) { min-height: 38px; padding-left: 12px; }
.query-text-input :deep(.el-input__prefix) { color: #667085; font-size: 16px; }
.page-size-select { width: 126px; }
.condition-shortcut { height: 38px; }.condition-shortcut span { display: inline-grid; min-width: 20px; height: 20px; margin-left: 2px; padding: 0 5px; place-items: center; border-radius: 4px; background: #eef2f6; color: #344054; font-size: 11px; }
.query-search-row .primary-action { min-width: 92px; height: 38px; }
.filter-section { margin: 0 18px 16px; padding: 14px; border: 1px solid #e4e9f0; border-radius: 6px; background: linear-gradient(180deg, #ffffff 0%, #fbfcfe 100%); box-shadow: inset 0 1px 0 rgba(255,255,255,.75); }
.filter-section-head { justify-content: space-between; min-height: 34px; padding: 0 0 12px; border-bottom: 1px solid #edf0f4; }
.filter-title { gap: 8px; color: #344054; }.filter-title > .el-icon { color: #667085; }.filter-title strong { font-size: 13px; }
.filter-head-actions { gap: 6px; }
.advanced-toggle { color: #475467; }.advanced-toggle .el-icon { margin-left: 4px; transition: transform .2s; }.advanced-toggle .el-icon.rotated { transform: rotate(180deg); }
.active-filter-summary { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; margin: 12px 0; padding: 9px 10px; border: 1px dashed #d8dee8; border-radius: 6px; background: #f8fafc; color: #667085; font-size: 12px; }
.active-filter-summary > span { font-weight: 650; color: #344054; }
.primary-filter-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 12px; }
.filter-control { min-width: 0; padding: 10px; border: 1px solid #eef1f5; border-radius: 6px; background: #fff; }
.filter-control > span { display: block; margin-bottom: 7px; color: #667085; font-size: 11px; font-weight: 650; }
.filter-control :deep(.el-select), .filter-control :deep(.el-segmented) { width: 100%; }
.source-filter-control { grid-column: span 2; }.source-filter-control :deep(.el-segmented) { min-height: 32px; }
.kubernetes-filter-band { margin-top: 14px; padding: 14px; border: 1px solid #dfe5ee; border-radius: 6px; background: #f8fafc; }
.kubernetes-band-head { justify-content: space-between; gap: 16px; margin-bottom: 12px; color: #667085; font-size: 11px; }
.kubernetes-band-head > div { display: flex; align-items: baseline; gap: 10px; min-width: 0; }.kubernetes-band-head strong { color: #344054; font-size: 12px; }.kubernetes-band-head > span { flex: 0 0 auto; }
.kubernetes-filter-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px; }
.hierarchy-control > span { display: flex; align-items: center; gap: 6px; }.hierarchy-control > span i { display: inline-grid; width: 18px; height: 18px; place-items: center; border-radius: 4px; background: #111827; color: #fff; font-size: 10px; font-style: normal; }
.advanced-filter-area { margin-top: 14px; padding-top: 14px; border-top: 1px solid #dfe5ee; }
.condition-builder { margin-top: 12px; padding: 12px 14px; border: 1px solid #e4e9f0; border-radius: 6px; background: #fafbfc; }
.condition-head { display: flex; align-items: center; justify-content: space-between; min-height: 28px; }
.condition-title { display: flex; align-items: baseline; gap: 10px; min-width: 0; }.condition-title strong { color: #344054; font-size: 12px; }.condition-title span { color: #98a2b3; font-size: 11px; }
.condition-actions { display: flex; align-items: center; gap: 10px; }.condition-actions :deep(.el-segmented) { min-width: 250px; }
.condition-list { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
.condition-row { display: grid; grid-template-columns: 30px minmax(160px, .8fr) 130px minmax(220px, 1.5fr) 34px; gap: 8px; align-items: center; }
.condition-field-select, .condition-operator-select, .condition-value-input { min-width: 0; width: 100%; }
.condition-connector { display: grid; width: 28px; height: 28px; place-items: center; border: 1px solid #d8dee8; border-radius: 4px; background: #fff; color: #475467; font-size: 11px; font-weight: 700; }
.empty-condition { padding: 6px 0 2px; color: #98a2b3; font-size: 12px; }
.histogram-panel { padding: 14px 18px 8px; }
.section-head { justify-content: space-between; gap: 12px; margin-bottom: 10px; color: #667085; font-size: 12px; }
.section-head > div { gap: 10px; }
.section-head strong { color: #1f2937; font-size: 14px; }
.section-head.compact { margin-bottom: 12px; }
.histogram-chart { height: 190px; }
.trend-error { color: #b42318; }.trend-error button { padding: 0; border: 0; background: transparent; color: #b42318; cursor: pointer; font: inherit; text-decoration: underline; }
.query-result-grid { display: grid; grid-template-columns: 230px minmax(0, 1fr); gap: 12px; min-height: 480px; }
.field-panel { padding: 16px; overflow: hidden; }
.field-list { display: flex; flex-direction: column; gap: 2px; max-height: calc(100vh - 470px); min-height: 380px; margin-top: 10px; overflow-y: auto; }
.field-list button { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; padding: 7px 8px; border: 0; border-radius: 5px; background: transparent; color: #475467; cursor: pointer; text-align: left; }
.field-list button:hover, .field-list button.active { background: #eff6ff; color: #1d4ed8; }
.field-list span { overflow: hidden; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.field-list em { color: #98a2b3; font-size: 10px; font-style: normal; }
.log-panel { min-width: 0; overflow: hidden; }
.log-head { min-height: 50px; margin: 0; padding: 0 16px; border-bottom: 1px solid #edf0f4; }
.log-view-actions, .wrap-control { display: flex; align-items: center; }
.log-view-actions { gap: 14px; }
.wrap-control { gap: 7px; color: #667085; font-size: 12px; cursor: pointer; white-space: nowrap; }
.query-error-title { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; }
.tail-status { display: inline-flex; align-items: center; gap: 6px; }
.tail-status i { width: 7px; height: 7px; border-radius: 50%; background: #f59e0b; }
.tail-status i.connected { background: #16a34a; box-shadow: 0 0 0 3px rgba(22, 163, 74, .12); }
.log-stream { min-height: 390px; max-height: calc(100vh - 440px); overflow: auto; }
.log-table { width: 100%; min-width: var(--log-table-min-width, 760px); }
.log-columns-head, .log-line { display: grid; grid-template-columns: var(--log-table-columns, 24px 160px 58px minmax(360px, 1fr) 76px); align-items: center; gap: 8px; }
.log-columns-head { position: sticky; top: 0; z-index: 1; min-height: 34px; padding: 0 12px; border-bottom: 1px solid #e5eaf1; background: #f8fafc; color: #667085; font-size: 11px; font-weight: 700; }
.log-columns-head > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-column-expander { display: block; }
.log-row { border-bottom: 1px solid #f0f2f5; background: #fff; }
.log-row:hover { background: #fafcff; }
.log-line { align-items: start; min-height: 40px; padding: 9px 12px; }
.expand-button { display: grid; width: 24px; height: 24px; place-items: center; border: 0; background: transparent; color: #98a2b3; cursor: pointer; transition: transform .15s; }
.log-row.expanded .expand-button { transform: rotate(90deg); }
.log-line time { padding-top: 3px; color: #667085; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; white-space: nowrap; }
.level-badge { align-self: start; padding: 3px 6px; border-radius: 4px; background: #f2f4f7; color: #475467; font-size: 10px; font-weight: 700; text-align: center; }
.level-error, .level-fatal { background: #fef2f2; color: #dc2626; }
.level-warn, .level-warning { background: #fffbeb; color: #d97706; }
.level-info { background: #eff6ff; color: #2563eb; }
.level-debug, .level-trace { background: #f5f3ff; color: #7c3aed; }
.log-message { min-width: 0; }
.log-message p { margin: 1px 0 0; overflow: hidden; color: #1f2937; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.55; text-overflow: ellipsis; white-space: nowrap; }
.log-table.wrap-lines .log-message p { overflow: visible; overflow-wrap: anywhere; text-overflow: clip; white-space: pre-wrap; }
.log-field-cell { min-width: 0; overflow: hidden; color: #475467; font-size: 11px; line-height: 1.45; text-overflow: ellipsis; white-space: nowrap; }
.log-line :deep(mark) { padding: 0 2px; background: #fde68a; color: inherit; }
.row-actions { display: flex; opacity: 0; transition: opacity .15s; }
.log-row:hover .row-actions { opacity: 1; }
.log-detail { display: grid; grid-template-columns: minmax(300px, .9fr) minmax(360px, 1.1fr); gap: 16px; padding: 14px 44px 18px; border-top: 1px dashed #e5e7eb; background: #f9fafb; }
.detail-section > strong { display: block; margin-bottom: 10px; color: #344054; font-size: 12px; }
.detail-section dl { display: grid; grid-template-columns: 145px minmax(0, 1fr); margin: 0; font-size: 11px; }
.detail-section dt, .detail-section dd { margin: 0; padding: 5px 7px; border-bottom: 1px solid #edf0f4; }
.detail-section dt { color: #667085; }
.detail-section dd { color: #1f2937; overflow-wrap: anywhere; }
.message-section pre, .context-row pre { margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; overflow-wrap: anywhere; }
.message-section pre { max-height: 260px; padding: 12px; overflow: auto; border: 1px solid #e5e7eb; border-radius: 6px; background: #fff; color: #111827; font-size: 12px; line-height: 1.55; }
.result-footer { display: flex; align-items: center; justify-content: space-between; gap: 14px; min-height: 54px; padding: 0 16px; border-top: 1px solid #edf0f4; color: #667085; font-size: 12px; }
.page-actions { display: flex; align-items: center; gap: 10px; }
.context-toolbar { display: flex; justify-content: space-between; margin-bottom: 12px; color: #667085; font-size: 12px; }
.context-list { height: calc(100vh - 150px); overflow: auto; border: 1px solid #e5e7eb; }
.context-row { display: grid; grid-template-columns: 165px 58px minmax(0, 1fr); gap: 10px; padding: 9px 12px; border-bottom: 1px solid #f0f2f5; }
.context-row.active { position: relative; background: #eff6ff; box-shadow: inset 3px 0 #2563eb; }
.context-row time { color: #667085; font-size: 11px; }
.context-row pre { color: #1f2937; font-size: 12px; line-height: 1.5; }
.template-dialog-title { display: flex; align-items: center; gap: 12px; }
.template-dialog-icon { display: grid; width: 38px; height: 38px; flex: 0 0 auto; place-items: center; border: 1px solid #dfe5ee; border-radius: 6px; background: #f8fafc; color: #344054; font-size: 18px; }
.template-dialog-title strong, .template-dialog-title span { display: block; }
.template-dialog-title strong { color: #1f2937; font-size: 16px; }
.template-dialog-title span { margin-top: 3px; color: #98a2b3; font-size: 12px; }
.template-search-bar { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 10px; padding-bottom: 14px; border-bottom: 1px solid #edf0f4; }
.template-result-list { min-height: 300px; padding: 10px 0; }
.template-result-item { display: flex; align-items: center; justify-content: space-between; gap: 18px; min-height: 92px; padding: 14px 12px; border-bottom: 1px solid #edf0f4; }
.template-result-item:hover, .template-result-item.active { background: #f8fafc; }
.template-result-item.active { box-shadow: inset 3px 0 #111827; }
.template-result-main { min-width: 0; }
.template-result-name, .template-result-meta { display: flex; align-items: center; flex-wrap: wrap; }
.template-result-name { gap: 7px; }
.template-result-name strong { color: #1f2937; font-size: 14px; }
.template-result-main p { margin: 7px 0; overflow: hidden; color: #667085; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.template-result-meta { gap: 12px; color: #98a2b3; font-size: 11px; }
.template-result-meta span:last-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.template-result-item > .el-button { flex: 0 0 auto; }
.template-pagination { display: flex; align-items: center; justify-content: space-between; min-height: 44px; padding-top: 10px; border-top: 1px solid #edf0f4; color: #667085; font-size: 12px; }
.export-form :deep(.el-input-number) { width: 100%; }
.export-status-line { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; color: #667085; font-size: 12px; }
.export-progress dl { display: grid; grid-template-columns: 90px minmax(0, 1fr); margin: 18px 0; border-top: 1px solid #edf0f4; font-size: 12px; }
.export-progress dt, .export-progress dd { margin: 0; padding: 9px 6px; border-bottom: 1px solid #edf0f4; }
.export-progress dt { color: #667085; }.export-progress dd { color: #1f2937; overflow-wrap: anywhere; }
@media (max-width: 1200px) {
  .primary-filter-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .kubernetes-filter-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 900px) {
  .query-context-row { align-items: flex-start; flex-direction: column; }
  .storage-context, .time-context { flex: 0 0 auto; width: 100%; }
  .time-context { justify-content: flex-start; }
  .query-search-row { grid-template-columns: minmax(0, 1fr) 126px; }
  .query-text-input { grid-column: 1 / -1; }
  .query-search-row .primary-action { grid-column: 1 / -1; }
  .primary-filter-grid, .kubernetes-filter-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .condition-head { align-items: flex-start; flex-direction: column; gap: 10px; }
  .condition-actions { justify-content: space-between; width: 100%; }
  .condition-row { grid-template-columns: 30px minmax(0, 1fr) 120px 34px; }
  .condition-value-input { grid-column: 2 / 4; grid-row: 2; }
  .condition-delete { grid-column: 4; grid-row: 1; }
  .query-result-grid { grid-template-columns: 1fr; }
  .field-panel { display: none; }
  .log-detail { grid-template-columns: 1fr; padding-left: 20px; }
}
@media (max-width: 640px) {
  .head-actions { flex-wrap: wrap; }
  .template-trigger { width: 100%; }
  .template-result-item { align-items: stretch; flex-direction: column; }
  .template-result-item > .el-button { width: 100%; }
  .template-pagination { align-items: flex-start; flex-direction: column; gap: 8px; }
  .context-control { align-items: flex-start; flex-wrap: wrap; width: 100%; }
  .storage-context :deep(.storage-select), .time-context :deep(.quick-range-select), .time-context :deep(.time-picker) { width: 100%; }
  .time-context :deep(.time-picker) { display: grid; grid-template-columns: 20px minmax(0, 1fr); grid-template-rows: repeat(2, 24px); height: auto; row-gap: 4px; padding-top: 6px; padding-bottom: 6px; }
  .time-context :deep(.time-picker .el-range__icon) { grid-column: 1; grid-row: 1; margin: 0; }
  .time-context :deep(.time-picker .el-range-separator) { grid-column: 1; grid-row: 2; width: auto; padding: 0; line-height: 24px; text-align: center; }
  .time-context :deep(.time-picker .el-range-input) { grid-column: 2; width: 100%; min-width: 0; text-align: left; }
  .time-context :deep(.time-picker .el-range-input:first-of-type) { grid-row: 1; }
  .time-context :deep(.time-picker .el-range-input:last-of-type) { grid-row: 2; }
  .time-context :deep(.time-picker .el-range__close-icon) { display: none; }
  .query-search-row { grid-template-columns: 1fr; }
  .query-search-row .primary-action { grid-column: auto; }
  .page-size-select { width: 100%; }
  .source-filter-control { grid-column: span 1; }
  .primary-filter-grid, .kubernetes-filter-grid { grid-template-columns: 1fr; }
  .kubernetes-band-head { align-items: flex-start; flex-direction: column; gap: 4px; }
  .condition-title { align-items: flex-start; flex-direction: column; gap: 3px; }
  .condition-actions { align-items: stretch; flex-direction: column; }.condition-actions :deep(.el-segmented) { min-width: 0; width: 100%; }
  .condition-row { grid-template-columns: 30px minmax(0, 1fr) 34px; }
  .condition-field-select { grid-column: 2; grid-row: 1; }
  .condition-operator-select { grid-column: 2; grid-row: 2; }
  .condition-value-input { grid-column: 2; grid-row: 3; }
  .condition-delete { grid-column: 3; grid-row: 1; }
  .log-table { min-width: 1050px; }
  .log-message, .log-field-cell, .row-actions { grid-column: auto; }
}
</style>
