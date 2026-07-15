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

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑查询模板' : '新增查询模板'" width="900px" destroy-on-close>
      <el-form :model="form" label-position="top" class="template-form">
        <div class="form-grid basic-grid">
          <el-form-item label="模板名称">
            <el-input v-model="form.name" placeholder="例如：生产环境错误日志" />
          </el-form-item>
          <el-form-item label="分类">
            <el-input v-model="form.category" placeholder="例如：Java、Nginx、Kubernetes" />
          </el-form-item>
          <el-form-item label="时间范围">
            <el-select v-model="form.timeRange">
              <el-option v-for="option in timeRangeOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
          </el-form-item>
        </div>

        <div class="builder-head">
          <strong>日志筛选</strong>
          <el-dropdown @command="applyPreset">
            <el-button>套用常用场景</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="all">全部日志</el-dropdown-item>
                <el-dropdown-item command="error">错误日志</el-dropdown-item>
                <el-dropdown-item command="warn">警告日志</el-dropdown-item>
                <el-dropdown-item command="timeout">超时日志</el-dropdown-item>
                <el-dropdown-item command="kubernetes-error">Kubernetes 错误日志</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <el-form-item label="日志正文包含">
          <el-input v-model="form.keyword" clearable placeholder="例如：timeout、connection refused；留空表示查询全部日志" />
        </el-form-item>

        <div class="form-grid query-grid">
          <el-form-item label="日志来源">
            <el-segmented v-model="form.sourceMode" :options="sourceModeOptions" />
          </el-form-item>
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

        <div class="form-grid scope-grid">
          <el-form-item label="环境">
            <el-select v-model="form.environments" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
          <el-form-item label="服务">
            <el-select v-model="form.services" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
          <el-form-item v-if="form.sourceMode !== 'host'" label="Namespace">
            <el-select v-model="form.namespaces" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
          <el-form-item v-if="form.sourceMode !== 'host'" label="工作负载">
            <el-select v-model="form.workloads" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
          <el-form-item label="容器">
            <el-select v-model="form.containers" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
          <el-form-item label="节点">
            <el-select v-model="form.nodes" multiple filterable allow-create default-first-option placeholder="输入后回车添加" />
          </el-form-item>
        </div>

        <div class="condition-builder">
          <div class="condition-head">
            <strong>其他字段条件</strong>
            <el-button link @click="addCondition"><el-icon><Plus /></el-icon>添加条件</el-button>
          </div>
          <div v-if="form.conditions.length" class="condition-list">
            <div v-for="condition in form.conditions" :key="condition.id" class="condition-row">
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
          <el-empty v-else description="未添加其他字段条件" :image-size="54" />
        </div>

        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="共享">
          <el-switch v-model="form.isPublic" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="primary-action" :loading="saving" @click="saveTemplate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { CollectionTag, Delete, Plus } from '@element-plus/icons-vue'
import {
  cloneLogTemplate,
  createLogTemplate,
  deleteLogTemplate,
  getLogTemplates,
  updateLogTemplate,
  type LogQueryTemplate,
} from '@/api/logcenter'

type SourceMode = 'all' | 'host' | 'kubernetes'
type SortMode = 'asc' | 'desc'
type TemplateCondition = { id: number; field: string; operator: string; value: string }
type TemplateDefinition = {
  version: number
  sourceMode: SourceMode
  level: string
  scope: {
    environments: string[]
    services: string[]
    namespaces: string[]
    workloads: string[]
    containers: string[]
    nodes: string[]
  }
  filters: Array<{ field: string; operator: string; value: string | string[] }>
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
  environments: string[]
  services: string[]
  namespaces: string[]
  workloads: string[]
  containers: string[]
  nodes: string[]
  conditions: TemplateCondition[]
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
let conditionSequence = 0

const filters = ref({ keyword: '', datasourceType: '', category: '' })
const sourceModeOptions = [{ label: '全部来源', value: 'all' }, { label: '主机', value: 'host' }, { label: 'Kubernetes', value: 'kubernetes' }]
const sortOptions = [{ label: '最新优先', value: 'desc' }, { label: '最早优先', value: 'asc' }]
const levelOptions = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']
const pageSizeOptions = [50, 100, 200, 500]
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
  sourceMode: 'all',
  level: '',
  environments: [],
  services: [],
  namespaces: [],
  workloads: [],
  containers: [],
  nodes: [],
  conditions: [],
  sort: 'desc',
  pageSize: 200,
  description: '',
  isPublic: false,
})

const form = ref<TemplateEditor>(emptyForm())

const normalizeStrings = (value: unknown) => Array.isArray(value) ? value.map(String).map(item => item.trim()).filter(Boolean) : []

const parseDefinition = (row: LogQueryTemplate): TemplateDefinition => {
  const fallback: TemplateDefinition = {
    version: 1,
    sourceMode: 'all',
    level: '',
    scope: { environments: [], services: [], namespaces: [], workloads: [], containers: [], nodes: [] },
    filters: [],
    sort: 'desc',
    pageSize: 200,
  }
  try {
    const parsed = JSON.parse(row.variables || '') as Partial<TemplateDefinition>
    if (!parsed || typeof parsed !== 'object' || Number(parsed.version) !== 1) return fallback
    const scope = parsed.scope || fallback.scope
    return {
      version: 1,
      sourceMode: ['host', 'kubernetes'].includes(String(parsed.sourceMode)) ? parsed.sourceMode as SourceMode : 'all',
      level: String(parsed.level || ''),
      scope: {
        environments: normalizeStrings(scope.environments),
        services: normalizeStrings(scope.services),
        namespaces: normalizeStrings(scope.namespaces),
        workloads: normalizeStrings(scope.workloads),
        containers: normalizeStrings(scope.containers),
        nodes: normalizeStrings(scope.nodes),
      },
      filters: Array.isArray(parsed.filters) ? parsed.filters.filter(item => item?.field && item?.operator).map(item => ({ field: String(item.field), operator: String(item.operator), value: Array.isArray(item.value) ? item.value.map(String) : String(item.value ?? '') })) : [],
      sort: parsed.sort === 'asc' ? 'asc' : 'desc',
      pageSize: pageSizeOptions.includes(Number(parsed.pageSize)) ? Number(parsed.pageSize) : 200,
    }
  } catch {
    return fallback
  }
}

const editorFromTemplate = (row: LogQueryTemplate): TemplateEditor => {
  const definition = parseDefinition(row)
  return {
    name: row.name,
    category: row.category || '常用查询',
    keyword: row.query === '*' ? '' : row.query,
    timeRange: row.timeRange || '15m',
    sourceMode: definition.sourceMode,
    level: definition.level,
    environments: [...definition.scope.environments],
    services: [...definition.scope.services],
    namespaces: [...definition.scope.namespaces],
    workloads: [...definition.scope.workloads],
    containers: [...definition.scope.containers],
    nodes: [...definition.scope.nodes],
    conditions: definition.filters.map(item => ({
      id: ++conditionSequence,
      field: item.field,
      operator: item.operator,
      value: Array.isArray(item.value) ? item.value.join(',') : String(item.value),
    })),
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
    environments: [...form.value.environments],
    services: [...form.value.services],
    namespaces: form.value.sourceMode === 'host' ? [] : [...form.value.namespaces],
    workloads: form.value.sourceMode === 'host' ? [] : [...form.value.workloads],
    containers: [...form.value.containers],
    nodes: [...form.value.nodes],
  },
  filters: form.value.conditions
    .filter(item => item.field && item.value.trim())
    .map(item => ({
      field: item.field,
      operator: item.operator,
      value: item.operator === 'in' ? item.value.split(/[,，]/).map(value => value.trim()).filter(Boolean) : item.value.trim(),
    })),
  sort: form.value.sort,
  pageSize: form.value.pageSize,
})

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
}

const openEdit = (row: LogQueryTemplate) => {
  editingId.value = row.id
  form.value = editorFromTemplate(row)
  dialogVisible.value = true
}

const addCondition = () => {
  form.value.conditions.push({ id: ++conditionSequence, field: 'filePath', operator: 'contains', value: '' })
}

const removeCondition = (id: number) => {
  form.value.conditions = form.value.conditions.filter(item => item.id !== id)
}

const applyPreset = (command: string) => {
  form.value.keyword = ''
  form.value.level = ''
  form.value.sourceMode = 'all'
  if (command === 'error') form.value.level = 'ERROR'
  if (command === 'warn') form.value.level = 'WARN'
  if (command === 'timeout') form.value.keyword = 'timeout'
  if (command === 'kubernetes-error') {
    form.value.sourceMode = 'kubernetes'
    form.value.level = 'ERROR'
  }
}

const saveTemplate = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('模板名称不能为空')
    return
  }
  const payload: LogQueryTemplate = {
    name: form.value.name.trim(),
    category: form.value.category.trim() || '常用查询',
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
  if (definition.level) tags.push(definition.level)
  if (definition.scope.environments.length) tags.push(`环境 ${definition.scope.environments.join('/')}`)
  if (definition.scope.services.length) tags.push(`服务 ${definition.scope.services.join('/')}`)
  if (definition.scope.namespaces.length) tags.push(`Namespace ${definition.scope.namespaces.join('/')}`)
  if (definition.filters.length) tags.push(`${definition.filters.length} 个字段条件`)
  return tags.slice(0, 4)
}

const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'

watch(filters, () => {
  pagination.value.page = 1
  void loadTemplates()
}, { deep: true })

onMounted(loadTemplates)
</script>

<style scoped>
.template-summary { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.template-summary strong { overflow: hidden; color: #344054; font-size: 12px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.summary-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.template-form :deep(.el-form-item) { margin-bottom: 16px; }
.form-grid { display: grid; gap: 12px; }
.basic-grid { grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) 180px; }
.query-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.scope-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.form-grid :deep(.el-select), .form-grid :deep(.el-segmented) { width: 100%; }
.builder-head, .condition-head { display: flex; align-items: center; justify-content: space-between; }
.builder-head { min-height: 42px; margin: 4px 0 12px; padding-top: 14px; border-top: 1px solid #edf0f4; }
.builder-head strong, .condition-head strong { color: #344054; font-size: 13px; }
.condition-builder { margin: 2px 0 16px; padding-top: 12px; border-top: 1px solid #edf0f4; }
.condition-list { display: flex; flex-direction: column; gap: 8px; margin-top: 10px; }
.condition-row { display: grid; grid-template-columns: minmax(150px, .8fr) 130px minmax(220px, 1.5fr) 34px; gap: 8px; align-items: center; }
.condition-builder :deep(.el-empty) { padding: 12px 0 4px; }
@media (max-width: 900px) {
  .basic-grid, .query-grid, .scope-grid { grid-template-columns: 1fr 1fr; }
  .condition-row { grid-template-columns: 1fr 120px; }
  .condition-row :deep(.el-input) { grid-column: 1 / -1; }
}
</style>
