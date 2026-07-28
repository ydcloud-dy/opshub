<template>
  <div class="log-center-page template-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><CollectionTag /></el-icon>
        </div>
        <div>
          <h2>查询模板</h2>
          <p>保存常用日志筛选条件，并从查询页直接执行</p>
        </div>
      </div>
      <el-button type="primary" class="primary-action" @click="openCreate">
        <el-icon><Plus /></el-icon>
        新增模板
      </el-button>
    </div>

    <section class="panel">
      <div class="filter-row">
        <el-input v-model="filters.keyword" placeholder="搜索模板名称、关键字、说明" clearable style="width: 300px" @keyup.enter="loadTemplates" />
        <el-input v-model="filters.category" placeholder="分类" clearable style="width: 160px" />
        <el-button @click="loadTemplates">查询</el-button>
      </div>

      <el-table v-loading="loading" :data="templates" height="calc(100vh - 300px)" empty-text="暂无查询模板">
        <el-table-column prop="name" label="模板名称" min-width="190" show-overflow-tooltip />
        <el-table-column prop="category" label="分类" width="130" />
        <el-table-column label="检索条件" min-width="360">
          <template #default="{ row }">
            <div class="template-summary">
              <strong>{{ templateKeyword(row) }}</strong>
              <div v-if="templateTags(row).length" class="summary-tags">
                <el-tag v-for="tag in templateTags(row)" :key="tag" size="small" type="info">{{ tag }}</el-tag>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="timeRange" label="时间范围" width="100" />
        <el-table-column prop="isPublic" label="共享" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.isPublic ? 'success' : 'info'">{{ row.isPublic ? '共享' : '私有' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新时间" width="170">
          <template #default="{ row }">{{ formatTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="runTemplate(row)">执行</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="primary" @click="handleClone(row)">克隆</el-button>
            <el-popconfirm title="确定删除这个查询模板？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next"
          @current-change="loadTemplates"
          @size-change="handlePageSizeChange"
        />
      </div>
    </section>

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑查询模板' : '新增查询模板'"
      width="min(1040px, calc(100vw - 32px))"
      destroy-on-close
      class="template-dialog"
    >
      <el-form :model="form" label-position="top" class="template-form">
        <section class="editor-section">
          <div class="editor-section-head">
            <span class="section-index">01</span>
            <div><strong>基本信息</strong><small>设置模板名称、分类和默认时间范围</small></div>
          </div>
          <div class="editor-section-body form-grid basic-grid">
            <el-form-item label="模板名称" class="required-item">
              <el-input v-model="form.name" maxlength="120" placeholder="例如：生产环境错误日志" />
            </el-form-item>
            <el-form-item label="分类">
              <el-input v-model="form.category" maxlength="60" placeholder="例如：Java、Nginx、Kubernetes" />
            </el-form-item>
            <el-form-item label="时间范围">
              <el-select v-model="form.timeRange" @change="refreshTemplateResources">
                <el-option v-for="option in timeRangeOptions" :key="option.value" :label="option.label" :value="option.value" />
              </el-select>
            </el-form-item>
          </div>
        </section>

        <section class="editor-section">
          <div class="editor-section-head with-action">
            <span class="section-index">02</span>
            <div><strong>查询规则</strong><small>每个模板必须明确选择主机或 Kubernetes 日志</small></div>
            <el-dropdown @command="applyPreset">
              <el-button>套用常用场景</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="all">主机全部日志</el-dropdown-item>
                  <el-dropdown-item command="error">主机错误日志</el-dropdown-item>
                  <el-dropdown-item command="warn">主机警告日志</el-dropdown-item>
                  <el-dropdown-item command="timeout">主机超时日志</el-dropdown-item>
                  <el-dropdown-item command="kubernetes-error">Kubernetes 错误日志</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="editor-section-body">
            <div class="source-picker">
              <span>日志来源</span>
              <el-segmented v-model="form.sourceMode" :options="sourceModeOptions" @change="handleSourceModeChange" />
              <small>查询时只会扫描所选类型的日志数据</small>
            </div>
            <el-form-item label="日志正文包含" class="keyword-item">
              <el-input v-model="form.keyword" clearable placeholder="例如：timeout、connection refused；留空表示该来源下的全部日志" />
            </el-form-item>
            <div class="form-grid query-grid">
              <el-form-item label="日志级别">
                <el-select v-model="form.level" clearable placeholder="全部级别">
                  <el-option v-for="level in levelOptions" :key="level" :label="level" :value="level" />
                </el-select>
              </el-form-item>
              <el-form-item label="排序方式">
                <el-segmented v-model="form.sort" :options="sortOptions" />
              </el-form-item>
              <el-form-item label="每页显示">
                <el-select v-model="form.pageSize">
                  <el-option v-for="size in pageSizeOptions" :key="size" :label="`${size} 条`" :value="size" />
                </el-select>
              </el-form-item>
            </div>
          </div>
        </section>

        <section class="editor-section">
          <div class="editor-section-head">
            <span class="section-index">03</span>
            <div><strong>资源范围</strong><small>从内置日志库和资产中加载，Kubernetes 资源按层级联动</small></div>
          </div>
          <div v-loading="resourceLoading" class="editor-section-body form-grid scope-grid">
            <div class="resource-context full-row">
              <span>{{ resourceContextText }}</span>
              <el-button link type="primary" :loading="resourceLoading" @click="refreshTemplateResources">刷新资源</el-button>
            </div>
            <el-alert v-if="resourceError" class="full-row" type="warning" :closable="false" show-icon :title="resourceError" />
            <el-form-item v-if="form.sourceMode === 'host'" label="主机" class="full-row">
              <el-select v-model="form.hostIds" multiple filterable collapse-tags :max-collapse-tags="4" clearable placeholder="不限主机">
                <el-option v-for="host in availableHosts" :key="host.id" :label="host.label" :value="host.id" />
              </el-select>
            </el-form-item>
            <el-form-item v-else label="Kubernetes 集群" class="full-row">
              <el-select v-model="form.clusterIds" multiple filterable collapse-tags :max-collapse-tags="4" clearable placeholder="不限集群" @change="handleClusterChange">
                <el-option v-for="cluster in availableClusters" :key="cluster.id" :label="cluster.label" :value="cluster.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="环境">
              <el-select v-model="form.environments" multiple filterable allow-create default-first-option collapse-tags clearable placeholder="不限环境">
                <el-option v-for="value in resourceOptions.environments" :key="value" :label="value" :value="value" />
              </el-select>
            </el-form-item>
            <el-form-item label="服务">
              <el-select v-model="form.services" multiple filterable allow-create default-first-option collapse-tags clearable placeholder="不限服务">
                <el-option v-for="value in resourceOptions.services" :key="value" :label="value" :value="value" />
              </el-select>
            </el-form-item>
            <template v-if="form.sourceMode === 'kubernetes'">
              <el-form-item label="Namespace">
                <el-select v-model="form.namespaces" multiple filterable collapse-tags clearable placeholder="不限 Namespace" @change="handleNamespaceChange">
                  <el-option v-for="option in kubernetesNamespaceOptions" :key="option.value" :label="option.label" :value="option.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="工作负载">
                <el-select v-model="form.workloads" multiple filterable collapse-tags clearable placeholder="不限工作负载" @change="handleWorkloadChange">
                  <el-option v-for="option in kubernetesWorkloadOptions" :key="option.value" :label="option.label" :value="option.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="Pod">
                <el-select v-model="form.pods" multiple filterable collapse-tags clearable placeholder="不限 Pod" @change="handlePodChange">
                  <el-option v-for="option in kubernetesPodOptions" :key="option.value" :label="option.label" :value="option.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="容器">
                <el-select v-model="form.containers" multiple filterable collapse-tags clearable placeholder="不限容器">
                  <el-option v-for="option in kubernetesContainerOptions" :key="option.value" :label="option.label" :value="option.value" />
                </el-select>
              </el-form-item>
            </template>
          </div>
        </section>

        <section class="editor-section">
          <div class="editor-section-head">
            <span class="section-index">04</span>
            <div><strong>字段条件</strong><small>用 AND / OR 组合多个结构化日志条件</small></div>
          </div>
          <div class="editor-section-body condition-builder">
          <div class="condition-head">
            <div>
              <strong>其他字段条件</strong>
              <span>范围条件始终使用 AND，本组可选择全部满足或任一满足</span>
            </div>
            <div class="condition-actions">
              <el-segmented v-if="form.conditions.length" v-model="form.filterLogic" :options="filterLogicOptions" size="small" />
              <el-button link @click="addCondition"><el-icon><Plus /></el-icon>添加条件</el-button>
            </div>
          </div>
          <div v-if="form.conditions.length" class="condition-list">
            <div v-for="(condition, index) in form.conditions" :key="condition.id" class="condition-row">
              <span class="condition-connector">{{ index === 0 ? '当' : form.filterLogic === 'and' ? '且' : '或' }}</span>
              <el-select v-model="condition.field" filterable placeholder="字段">
                <el-option v-for="field in templateFieldOptions" :key="field.value" :label="field.label" :value="field.value" />
              </el-select>
              <el-select v-model="condition.operator" placeholder="操作符">
                <el-option v-for="operator in operatorOptions" :key="operator.value" :label="operator.label" :value="operator.value" />
              </el-select>
              <el-input v-model="condition.value" :placeholder="condition.operator === 'in' ? '多个值使用逗号分隔' : '条件值'" />
              <el-button circle text title="删除条件" @click="removeCondition(condition.id)"><el-icon><Delete /></el-icon></el-button>
            </div>
          </div>
          <div v-else class="condition-empty">未添加字段条件，将按上述范围直接查询</div>
          </div>
        </section>

        <section class="editor-section last-section">
          <div class="editor-section-head">
            <span class="section-index">05</span>
            <div><strong>说明与共享</strong><small>说明模板用途，并决定是否对其他用户可见</small></div>
          </div>
          <div class="editor-section-body description-grid">
            <el-form-item label="模板说明">
              <el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="说明使用场景和适用范围" />
            </el-form-item>
            <div class="share-setting">
              <div><strong>共享模板</strong><small>开启后，其他有日志查询权限的用户可以使用该模板</small></div>
              <el-switch v-model="form.isPublic" />
            </div>
          </div>
        </section>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="primary-action" :loading="saving" @click="saveTemplate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { CollectionTag, Delete, Plus } from '@element-plus/icons-vue'
import {
  cloneLogTemplate,
  createLogTemplate,
  deleteLogTemplate,
  getInternalLogAssets,
  getLogStorageClusters,
  getLogTemplates,
  queryInternalLogResourceOptions,
  updateLogTemplate,
  type InternalKubernetesResourceOption,
  type InternalLogQueryRequest,
  type InternalLogResourceOptions,
  type LogQueryTemplate,
  type LogPolicyTargetCluster,
  type LogPolicyTargetHost,
  type LogStorageCluster,
} from '@/api/logcenter'

type SourceMode = 'host' | 'kubernetes'
type SortMode = 'asc' | 'desc'
type TemplateCondition = { id: number; field: string; operator: string; value: string }
type AssetOption = { id: number; label: string }
type ResourceSelectOption = { value: string; label: string }
type TemplateDefinition = {
  version: number
  sourceMode: SourceMode
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
  sort: SortMode
  pageSize: number
}
type TemplateEditor = {
  name: string
  category: string
  keyword: string
  timeRange: string
  sourceMode: SourceMode
  level: string
  hostIds: number[]
  clusterIds: number[]
  environments: string[]
  services: string[]
  namespaces: string[]
  workloads: string[]
  pods: string[]
  containers: string[]
  nodes: string[]
  conditions: TemplateCondition[]
  filterLogic: 'and' | 'or'
  sort: SortMode
  pageSize: number
  description: string
  isPublic: boolean
}

const router = useRouter()
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editingId = ref<number | undefined>()
const templates = ref<LogQueryTemplate[]>([])
const pagination = ref({ page: 1, pageSize: 20, total: 0 })
const resourceLoading = ref(false)
const resourceError = ref('')
const storages = ref<LogStorageCluster[]>([])
const activeStorageId = ref<number>()
const assetHosts = ref<LogPolicyTargetHost[]>([])
const assetClusters = ref<LogPolicyTargetCluster[]>([])
const resourceOptions = ref<InternalLogResourceOptions>({
  hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [], kubernetesResources: [],
})
let conditionSequence = 0
let resourceSequence = 0
let resourceController: AbortController | undefined
let assetsLoaded = false

const filters = ref({ keyword: '', datasourceType: '', category: '' })
const sourceModeOptions = [{ label: '主机', value: 'host' }, { label: 'Kubernetes', value: 'kubernetes' }]
const sortOptions = [{ label: '最新优先', value: 'desc' }, { label: '最早优先', value: 'asc' }]
const levelOptions = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']
const pageSizeOptions = [50, 100, 200, 500, 1000, 2000]
const timeRangeOptions = [
  { label: '最近 15 分钟', value: '15m' },
  { label: '最近 1 小时', value: '1h' },
  { label: '最近 6 小时', value: '6h' },
  { label: '最近 24 小时', value: '24h' },
  { label: '最近 7 天', value: '7d' },
]
const operatorOptions = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
  { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' },
  { label: '属于', value: 'in' },
]
const filterLogicOptions = [{ label: '全部满足 AND', value: 'and' }, { label: '任一满足 OR', value: 'or' }]
const templateFieldOptions = [
  { label: '来源类型', value: 'sourceType' },
  { label: '资产类型', value: 'assetType' },
  { label: '文件路径', value: 'filePath' },
  { label: '工作负载', value: 'workloadName' },
  { label: 'Pod', value: 'podName' },
  { label: '容器', value: 'containerName' },
  { label: '节点', value: 'nodeName' },
  { label: 'Trace ID', value: 'traceId' },
]

const emptyForm = (): TemplateEditor => ({
  name: '',
  category: '常用查询',
  keyword: '',
  timeRange: '15m',
  sourceMode: 'host',
  level: '',
  hostIds: [],
  clusterIds: [],
  environments: [],
  services: [],
  namespaces: [],
  workloads: [],
  pods: [],
  containers: [],
  nodes: [],
  conditions: [],
  filterLogic: 'and',
  sort: 'desc',
  pageSize: 200,
  description: '',
  isPublic: false,
})

const form = ref<TemplateEditor>(emptyForm())

const normalizeStrings = (value: unknown) => Array.isArray(value) ? value.map(String).map(item => item.trim()).filter(Boolean) : []
const normalizeNumbers = (value: unknown) => Array.isArray(value) ? value.map(Number).filter(Boolean) : []

const mergeAssetOptions = (items: AssetOption[], ids: string[], prefix: string) => {
  const result = new Map<number, string>()
  items.forEach(item => result.set(item.id, item.label))
  ids.forEach(value => {
    const id = Number(value)
    if (id && !result.has(id)) result.set(id, `${prefix} #${id}`)
  })
  return Array.from(result, ([id, label]) => ({ id, label })).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
}
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
const simpleResourceOptions = (values: string[]): ResourceSelectOption[] => Array.from(new Set(values.filter(Boolean)))
  .sort((left, right) => left.localeCompare(right, 'zh-CN'))
  .map(value => ({ value, label: value }))
const pathMatchesClusters = (item: InternalKubernetesResourceOption) => !form.value.clusterIds.length || form.value.clusterIds.some(id => String(id) === String(item.clusterId))
const pathMatchesNamespaces = (item: InternalKubernetesResourceOption) => !form.value.namespaces.length || form.value.namespaces.includes(item.namespace)
const pathMatchesWorkloads = (item: InternalKubernetesResourceOption) => !form.value.workloads.length || form.value.workloads.includes(item.workloadName)
const pathMatchesPods = (item: InternalKubernetesResourceOption) => !form.value.pods.length || form.value.pods.includes(item.podName)
const kubernetesNamespaceOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(pathMatchesClusters)
  return resourceOptions.value.kubernetesResources.length ? simpleResourceOptions(paths.map(item => item.namespace)) : simpleResourceOptions(resourceOptions.value.namespaces)
})
const kubernetesWorkloadOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item))
  if (!resourceOptions.value.kubernetesResources.length) return simpleResourceOptions(resourceOptions.value.workloads)
  const values = new Map<string, { kinds: Set<string>; namespaces: Set<string> }>()
  paths.forEach(item => {
    if (!item.workloadName) return
    const metadata = values.get(item.workloadName) || { kinds: new Set<string>(), namespaces: new Set<string>() }
    if (item.workloadKind) metadata.kinds.add(item.workloadKind)
    if (item.namespace) metadata.namespaces.add(item.namespace)
    values.set(item.workloadName, metadata)
  })
  return Array.from(values, ([value, metadata]) => {
    const kind = Array.from(metadata.kinds).sort().join('/')
    const namespaceHint = !form.value.namespaces.length && metadata.namespaces.size === 1 ? ` · ${Array.from(metadata.namespaces)[0]}` : metadata.namespaces.size > 1 ? ` · ${metadata.namespaces.size} 个 Namespace` : ''
    return { value, label: `${kind ? `${kind} / ` : ''}${value}${namespaceHint}` }
  }).sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
})
const kubernetesPodOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item) && pathMatchesWorkloads(item))
  if (!resourceOptions.value.kubernetesResources.length) return simpleResourceOptions(resourceOptions.value.pods)
  return simpleResourceOptions(paths.map(item => item.podName))
})
const kubernetesContainerOptions = computed<ResourceSelectOption[]>(() => {
  const paths = resourceOptions.value.kubernetesResources.filter(item => pathMatchesClusters(item) && pathMatchesNamespaces(item) && pathMatchesWorkloads(item) && pathMatchesPods(item))
  return resourceOptions.value.kubernetesResources.length ? simpleResourceOptions(paths.map(item => item.containerName)) : simpleResourceOptions(resourceOptions.value.containers)
})
const activeStorageName = computed(() => storages.value.find(item => item.id === activeStorageId.value)?.name || '')
const resourceContextText = computed(() => {
  if (resourceLoading.value) return '正在从内置日志库加载资源...'
  if (!activeStorageId.value) return '暂无已初始化的内置日志库'
  const count = form.value.sourceMode === 'host' ? availableHosts.value.length : availableClusters.value.length
  return `数据来源：${activeStorageName.value || '内置日志库'} · 已识别 ${count} 个${form.value.sourceMode === 'host' ? '主机' : '集群'}资源`
})
const retainAvailableValues = (values: string[], options: ResourceSelectOption[]) => {
  const available = new Set(options.map(item => item.value))
  return values.filter(value => available.has(value))
}
const handleClusterChange = () => {
  form.value.namespaces = retainAvailableValues(form.value.namespaces, kubernetesNamespaceOptions.value)
  handleNamespaceChange()
}
const handleNamespaceChange = () => {
  form.value.workloads = retainAvailableValues(form.value.workloads, kubernetesWorkloadOptions.value)
  handleWorkloadChange()
}
const handleWorkloadChange = () => {
  form.value.pods = retainAvailableValues(form.value.pods, kubernetesPodOptions.value)
  handlePodChange()
}
const handlePodChange = () => {
  form.value.containers = retainAvailableValues(form.value.containers, kubernetesContainerOptions.value)
}

const parseDefinition = (row: LogQueryTemplate): TemplateDefinition => {
  const fallback: TemplateDefinition = {
    version: 1,
    sourceMode: 'host',
    level: '',
    scope: { hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [] },
    filters: [],
    filterLogic: 'and',
    sort: 'desc',
    pageSize: 200,
  }
  try {
    const parsed = JSON.parse(row.variables || '') as Partial<TemplateDefinition>
    if (!parsed || typeof parsed !== 'object' || Number(parsed.version) !== 1) return fallback
    const scope = parsed.scope || fallback.scope
    return {
      version: 1,
      sourceMode: parsed.sourceMode === 'kubernetes' ? 'kubernetes' : 'host',
      level: String(parsed.level || ''),
      scope: {
        hostIds: normalizeNumbers(scope.hostIds),
        clusterIds: normalizeNumbers(scope.clusterIds),
        environments: normalizeStrings(scope.environments),
        services: normalizeStrings(scope.services),
        namespaces: normalizeStrings(scope.namespaces),
        workloads: normalizeStrings(scope.workloads),
        pods: normalizeStrings(scope.pods),
        containers: normalizeStrings(scope.containers),
        nodes: normalizeStrings(scope.nodes),
      },
      filters: Array.isArray(parsed.filters) ? parsed.filters.filter(item => item?.field && item?.operator).map(item => ({ field: String(item.field), operator: String(item.operator), value: Array.isArray(item.value) ? item.value.map(String) : String(item.value ?? '') })) : [],
      filterLogic: parsed.filterLogic === 'or' ? 'or' : 'and',
      sort: parsed.sort === 'asc' ? 'asc' : 'desc',
      pageSize: pageSizeOptions.includes(Number(parsed.pageSize)) ? Number(parsed.pageSize) : 200,
    }
  } catch {
    return fallback
  }
}

const editorFromTemplate = (row: LogQueryTemplate): TemplateEditor => {
  const definition = parseDefinition(row)
  const legacyNodeConditions: TemplateCondition[] = definition.scope.nodes.length
    ? [{ id: ++conditionSequence, field: 'nodeName', operator: 'in', value: definition.scope.nodes.join(',') }]
    : []
  return {
    name: row.name,
    category: row.category || '常用查询',
    keyword: row.query === '*' ? '' : row.query,
    timeRange: row.timeRange || '15m',
    sourceMode: definition.sourceMode,
    level: definition.level,
    hostIds: [...definition.scope.hostIds],
    clusterIds: [...definition.scope.clusterIds],
    environments: [...definition.scope.environments],
    services: [...definition.scope.services],
    namespaces: [...definition.scope.namespaces],
    workloads: [...definition.scope.workloads],
    pods: [...definition.scope.pods],
    containers: [...definition.scope.containers],
    nodes: [],
    conditions: [...legacyNodeConditions, ...definition.filters.map(item => ({
      id: ++conditionSequence,
      field: item.field,
      operator: item.operator,
      value: Array.isArray(item.value) ? item.value.join(',') : String(item.value),
    }))],
    filterLogic: definition.filterLogic,
    sort: definition.sort,
    pageSize: definition.pageSize,
    description: row.description || '',
    isPublic: Boolean(row.isPublic),
  }
}

const serializeDefinition = (): TemplateDefinition => ({
  version: 1,
  sourceMode: form.value.sourceMode,
  level: form.value.level,
  scope: {
    hostIds: form.value.sourceMode === 'host' ? [...form.value.hostIds] : [],
    clusterIds: form.value.sourceMode === 'host' ? [] : [...form.value.clusterIds],
    environments: [...form.value.environments],
    services: [...form.value.services],
    namespaces: form.value.sourceMode === 'host' ? [] : [...form.value.namespaces],
    workloads: form.value.sourceMode === 'host' ? [] : [...form.value.workloads],
    pods: form.value.sourceMode === 'host' ? [] : [...form.value.pods],
    containers: form.value.sourceMode === 'host' ? [] : [...form.value.containers],
    nodes: [],
  },
  filters: form.value.conditions
    .filter(item => item.field && item.value.trim())
    .map(item => ({
      field: item.field,
      operator: item.operator,
      value: item.operator === 'in' ? item.value.split(/[,，]/).map(value => value.trim()).filter(Boolean) : item.value.trim(),
    })),
  filterLogic: form.value.filterLogic,
  sort: form.value.sort,
  pageSize: form.value.pageSize,
})

const loadTemplateResourceOptions = async (preferredStorageId?: number) => {
  const sequence = ++resourceSequence
  resourceController?.abort()
  const controller = new AbortController()
  resourceController = controller
  resourceLoading.value = true
  resourceError.value = ''
  try {
    if (!assetsLoaded) {
      const [storageRows, assets] = await Promise.all([
        getLogStorageClusters({ enabled: true }) as any,
        getInternalLogAssets() as any,
      ])
      if (sequence !== resourceSequence) return
      storages.value = Array.isArray(storageRows) ? storageRows : []
      assetHosts.value = Array.isArray(assets?.hosts) ? assets.hosts : []
      assetClusters.value = Array.isArray(assets?.clusters) ? assets.clusters : []
      assetsLoaded = true
    }
    const preferred = storages.value.find(item => item.id === preferredStorageId && item.initializedAt)
    const available = preferred
      || storages.value.find(item => item.isPrimary && item.initializedAt)
      || storages.value.find(item => item.initializedAt)
    activeStorageId.value = available?.id
    if (!activeStorageId.value) {
      resourceOptions.value = { hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [], kubernetesResources: [] }
      resourceError.value = '请先在“日志库”中初始化并启用 ClickHouse 存储'
      return
    }
    const end = new Date()
    const result = await queryInternalLogResourceOptions({
      storageId: activeStorageId.value,
      start: new Date(end.getTime() - rangeMilliseconds(form.value.timeRange)).toISOString(),
      end: end.toISOString(),
      query: '*',
      scope: {
        assetTypes: [form.value.sourceMode],
        hostIds: [], clusterIds: [], environments: [], services: [], namespaces: [], workloads: [], pods: [], containers: [], nodes: [], levels: [],
      },
      filters: [],
      filterLogic: 'and',
      sort: 'desc',
      limit: 1,
      skipHistory: true,
    } as InternalLogQueryRequest, controller.signal) as any
    if (sequence !== resourceSequence) return
    resourceOptions.value = {
      hostIds: result?.hostIds || [], clusterIds: result?.clusterIds || [], environments: result?.environments || [],
      services: result?.services || [], namespaces: result?.namespaces || [], workloads: result?.workloads || [],
      pods: result?.pods || [], containers: result?.containers || [], nodes: result?.nodes || [],
      kubernetesResources: result?.kubernetesResources || [],
    }
    handleClusterChange()
  } catch (error: any) {
    if (!['ERR_CANCELED', 'CanceledError', 'AbortError'].includes(String(error?.code || error?.name || ''))) {
      resourceError.value = error?.response?.data?.message || error?.message || '加载日志资源失败'
    }
  } finally {
    if (sequence === resourceSequence) resourceLoading.value = false
  }
}
const refreshTemplateResources = () => { void loadTemplateResourceOptions(activeStorageId.value) }

const loadTemplates = async () => {
  loading.value = true
  try {
    const data = await getLogTemplates({ ...filters.value, page: pagination.value.page, pageSize: pagination.value.pageSize }) as any
    templates.value = data?.data || []
    pagination.value.total = data?.total || 0
  } finally {
    loading.value = false
  }
}

const handlePageSizeChange = () => {
  pagination.value.page = 1
  void loadTemplates()
}

const openCreate = () => {
  editingId.value = undefined
  form.value = emptyForm()
  dialogVisible.value = true
  void loadTemplateResourceOptions()
}

const openEdit = (row: LogQueryTemplate) => {
  editingId.value = row.id
  form.value = editorFromTemplate(row)
  dialogVisible.value = true
  void loadTemplateResourceOptions(row.datasourceId)
}

const addCondition = () => {
  form.value.conditions.push({ id: ++conditionSequence, field: 'filePath', operator: 'contains', value: '' })
}

const removeCondition = (id: number) => {
  form.value.conditions = form.value.conditions.filter(item => item.id !== id)
}

const handleSourceModeChange = (value: SourceMode) => {
  if (value === 'host') {
    form.value.clusterIds = []
    form.value.namespaces = []
    form.value.workloads = []
    form.value.pods = []
    form.value.containers = []
    form.value.nodes = []
  } else {
    form.value.hostIds = []
  }
  void loadTemplateResourceOptions(activeStorageId.value)
}

const applyPreset = (command: string) => {
  form.value.keyword = ''
  form.value.level = ''
  form.value.sourceMode = 'host'
  if (command === 'error') form.value.level = 'ERROR'
  if (command === 'warn') form.value.level = 'WARN'
  if (command === 'timeout') form.value.keyword = 'timeout'
  if (command === 'kubernetes-error') {
    form.value.sourceMode = 'kubernetes'
    form.value.level = 'ERROR'
  }
  handleSourceModeChange(form.value.sourceMode)
}

const saveTemplate = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('模板名称不能为空')
    return
  }
  const payload: LogQueryTemplate = {
    name: form.value.name.trim(),
    category: form.value.category.trim() || '常用查询',
    datasourceId: activeStorageId.value,
    datasourceType: 'internal_clickhouse',
    queryLanguage: 'structured',
    query: form.value.keyword.trim() || '*',
    index: '',
    timeRange: form.value.timeRange,
    variables: JSON.stringify(serializeDefinition()),
    description: form.value.description.trim(),
    isPublic: form.value.isPublic,
    sort: 0,
  }
  saving.value = true
  try {
    if (editingId.value) await updateLogTemplate(editingId.value, payload)
    else await createLogTemplate(payload)
    ElMessage.success('查询模板已保存')
    dialogVisible.value = false
    await loadTemplates()
  } finally {
    saving.value = false
  }
}

const handleClone = async (row: LogQueryTemplate) => {
  if (!row.id) return
  await cloneLogTemplate(row.id)
  ElMessage.success('模板已克隆')
  await loadTemplates()
}

const handleDelete = async (row: LogQueryTemplate) => {
  if (!row.id) return
  await deleteLogTemplate(row.id)
  ElMessage.success('模板已删除')
  await loadTemplates()
}

const rangeMilliseconds = (value?: string) => {
  const matched = String(value || '15m').trim().match(/^(\d+)(m|h|d)$/i)
  if (!matched) return 15 * 60 * 1000
  const amount = Number(matched[1])
  const unit = (matched[2] || 'm').toLowerCase()
  return amount * (unit === 'd' ? 86400000 : unit === 'h' ? 3600000 : 60000)
}

const runTemplate = (row: LogQueryTemplate) => {
  const definition = parseDefinition(row)
  const end = new Date()
  const start = new Date(end.getTime() - rangeMilliseconds(row.timeRange))
  const query: Record<string, string | number | undefined> = {
    templateId: row.id,
    storageId: row.datasourceId || undefined,
    q: row.query || '*',
    sourceMode: definition.sourceMode,
    start: start.toISOString(),
    end: end.toISOString(),
    sort: definition.sort,
    pageSize: definition.pageSize,
    filterLogic: definition.filterLogic,
    templateRun: Date.now(),
  }
  for (const [key, values] of Object.entries(definition.scope)) {
    if (values.length) query[key] = values.join(',')
  }
  const structuredFilters = [...definition.filters]
  if (definition.level) structuredFilters.unshift({ field: 'level', operator: 'eq', value: definition.level })
  if (structuredFilters.length) query.filters = JSON.stringify(structuredFilters)
  router.push({ path: '/logs/query', query })
}

const templateKeyword = (row: LogQueryTemplate) => row.query && row.query !== '*' ? `正文包含：${row.query}` : '全部日志正文'

const templateTags = (row: LogQueryTemplate) => {
  const definition = parseDefinition(row)
  const tags: string[] = []
  if (definition.sourceMode === 'host') tags.push('主机')
  if (definition.sourceMode === 'kubernetes') tags.push('Kubernetes')
  if (definition.scope.hostIds.length) tags.push(`${definition.scope.hostIds.length} 台主机`)
  if (definition.scope.clusterIds.length) tags.push(`${definition.scope.clusterIds.length} 个集群`)
  if (definition.level) tags.push(definition.level)
  if (definition.scope.environments.length) tags.push(`环境 ${definition.scope.environments.join('/')}`)
  if (definition.scope.services.length) tags.push(`服务 ${definition.scope.services.join('/')}`)
  if (definition.scope.namespaces.length) tags.push(`Namespace ${definition.scope.namespaces.join('/')}`)
  if (definition.filters.length) tags.push(`${definition.filters.length} 个字段条件 · ${definition.filterLogic.toUpperCase()}`)
  return tags.slice(0, 4)
}

const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'

watch(filters, () => {
  pagination.value.page = 1
  void loadTemplates()
}, { deep: true })

onMounted(loadTemplates)
onBeforeUnmount(() => resourceController?.abort())
</script>

<style scoped>
.template-summary { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.template-summary strong { overflow: hidden; color: #344054; font-size: 12px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.summary-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.template-dialog :deep(.el-dialog__body) { max-height: calc(100vh - 156px); overflow-y: auto; padding: 0; background: #f7f8fa; }
.template-dialog :deep(.el-dialog__header) { margin-right: 0; padding: 18px 22px; border-bottom: 1px solid #eaecf0; }
.template-dialog :deep(.el-dialog__footer) { padding: 14px 22px; border-top: 1px solid #eaecf0; }
.template-form { display: flex; flex-direction: column; }
.template-form :deep(.el-form-item) { min-width: 0; margin-bottom: 0; }
.template-form :deep(.el-form-item__label) { padding-bottom: 7px; color: #475467; font-size: 12px; font-weight: 600; }
.editor-section { display: grid; grid-template-columns: 210px minmax(0, 1fr); border-bottom: 1px solid #e8ebef; background: #fff; }
.editor-section-head { display: grid; grid-template-columns: 34px minmax(0, 1fr); align-content: start; gap: 10px; padding: 22px 18px 22px 22px; border-right: 1px solid #edf0f3; background: #fafbfc; }
.editor-section-head.with-action { grid-template-columns: 34px minmax(0, 1fr); }
.editor-section-head.with-action > .el-dropdown { grid-column: 2; margin-top: 12px; justify-self: start; }
.editor-section-head strong, .editor-section-head small { display: block; }
.editor-section-head strong { color: #101828; font-size: 14px; font-weight: 650; }
.editor-section-head small { margin-top: 5px; color: #98a2b3; font-size: 11px; line-height: 1.55; }
.section-index { display: grid; width: 30px; height: 26px; place-items: center; border: 1px solid #d0d5dd; border-radius: 4px; background: #fff; color: #667085; font-size: 10px; font-weight: 700; }
.editor-section-body { min-width: 0; padding: 22px; }
.last-section { border-bottom: 0; }
.form-grid { display: grid; gap: 12px; }
.basic-grid { grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr) 180px; }
.query-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.scope-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.scope-grid .full-row { grid-column: 1 / -1; }
.resource-context { display: flex; align-items: center; justify-content: space-between; gap: 14px; min-height: 34px; padding: 7px 10px; border: 1px solid #e4e7ec; border-radius: 5px; background: #f9fafb; color: #667085; font-size: 11px; }
.form-grid :deep(.el-select), .form-grid :deep(.el-segmented) { width: 100%; }
.required-item :deep(.el-form-item__label)::after { margin-left: 4px; color: #d92d20; content: '*'; }
.source-picker { display: grid; grid-template-columns: 88px 260px minmax(0, 1fr); align-items: center; gap: 12px; margin-bottom: 18px; padding: 12px 14px; border: 1px solid #e4e7ec; border-radius: 6px; background: #f9fafb; }
.source-picker > span { color: #344054; font-size: 12px; font-weight: 650; }
.source-picker > small { color: #98a2b3; font-size: 11px; }
.source-picker :deep(.el-segmented) { width: 100%; }
.keyword-item { margin-bottom: 16px !important; }
.condition-head, .condition-actions { display: flex; align-items: center; justify-content: space-between; }
.condition-head strong { color: #344054; font-size: 13px; }
.condition-head > div:first-child { display: flex; align-items: baseline; gap: 10px; }.condition-head > div:first-child span { color: #98a2b3; font-size: 11px; }
.condition-actions { gap: 10px; }.condition-actions :deep(.el-segmented) { min-width: 250px; }
.condition-builder { padding: 22px; }
.condition-list { display: flex; flex-direction: column; gap: 8px; margin-top: 10px; }
.condition-row { display: grid; grid-template-columns: 30px minmax(140px, .8fr) 122px minmax(180px, 1.5fr) 34px; gap: 8px; align-items: center; }
.condition-connector { display: grid; width: 28px; height: 28px; place-items: center; border: 1px solid #d8dee8; border-radius: 4px; color: #475467; font-size: 11px; font-weight: 700; }
.condition-empty { margin-top: 12px; padding: 16px; border: 1px dashed #d0d5dd; border-radius: 5px; background: #fcfcfd; color: #98a2b3; font-size: 12px; text-align: center; }
.description-grid { display: grid; grid-template-columns: minmax(0, 1fr) 260px; gap: 18px; }
.share-setting { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; padding: 13px 14px; border: 1px solid #e4e7ec; border-radius: 6px; background: #f9fafb; }
.share-setting strong, .share-setting small { display: block; }
.share-setting strong { color: #344054; font-size: 12px; }
.share-setting small { margin-top: 5px; color: #98a2b3; font-size: 11px; line-height: 1.55; }
@media (max-width: 960px) {
  .editor-section { grid-template-columns: 1fr; }
  .editor-section-head { border-right: 0; border-bottom: 1px solid #edf0f3; }
  .editor-section-head.with-action { grid-template-columns: 34px minmax(0, 1fr) auto; }
  .editor-section-head.with-action > .el-dropdown { grid-column: 3; grid-row: 1; margin-top: 0; }
  .basic-grid, .query-grid, .scope-grid { grid-template-columns: 1fr 1fr; }
  .source-picker { grid-template-columns: 80px minmax(220px, 1fr); }
  .source-picker > small { grid-column: 2; }
  .description-grid { grid-template-columns: 1fr; }
  .condition-head { align-items: flex-start; flex-direction: column; gap: 10px; }
  .condition-actions { width: 100%; }
  .condition-row { grid-template-columns: 30px 1fr 120px 34px; }
  .condition-row :deep(.el-input) { grid-column: 2 / 4; }
}
@media (max-width: 640px) {
  .editor-section-head.with-action { grid-template-columns: 34px minmax(0, 1fr); }
  .editor-section-head.with-action > .el-dropdown { grid-column: 2; grid-row: auto; }
  .basic-grid, .query-grid, .scope-grid { grid-template-columns: 1fr; }
  .source-picker { grid-template-columns: 1fr; }
  .source-picker > small { grid-column: auto; }
  .condition-actions { align-items: flex-start; flex-direction: column; }
  .condition-actions :deep(.el-segmented) { min-width: 0; width: 100%; }
}
</style>
