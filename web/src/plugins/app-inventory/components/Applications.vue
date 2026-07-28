<template>
  <div class="inventory-page">
    <PageHeader :icon="Grid" title="应用台账" description="应用是资产关联的唯一入口，环境、入口、资源和依赖都会从这里展开。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ overviewCounts.applications || 0 }}</strong><span>应用总数</span></div>
          <div class="inventory-header-metric"><strong>{{ activeCount }}</strong><span>运行中</span></div>
          <div class="inventory-header-metric"><strong>{{ environmentCount }}</strong><span>覆盖环境</span></div>
          <div class="inventory-header-metric"><strong>{{ unhealthyCount }}</strong><span>异常应用</span></div>
        </div>
      </template>
      <el-button :icon="OfficeBuilding" @click="router.push('/app-inventory/environments')">环境管理</el-button>
      <el-button :icon="Refresh" :loading="loading" @click="loadData">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">登记应用</el-button>
    </PageHeader>

    <div class="inventory-toolbar">
      <el-input v-model="filters.keyword" clearable placeholder="搜索应用名称、编码、负责人或部门" :prefix-icon="Search" @keyup.enter="loadData" />
      <el-select v-model="filters.status" clearable placeholder="治理状态" @change="loadData">
        <el-option label="运行中" value="active" />
        <el-option label="已停用" value="disabled" />
        <el-option label="规划中" value="planned" />
      </el-select>
      <el-button type="primary" :icon="Search" @click="loadData">查询</el-button>
      <el-button text @click="resetFilters">重置</el-button>
    </div>

    <section class="inventory-panel">
      <el-table v-loading="loading" :data="applications" stripe>
        <el-table-column label="应用" min-width="230">
          <template #default="{ row }">
            <div class="inventory-link" @click="openDetail(row)">{{ row.name }}</div>
            <div class="inventory-table-cell__sub">{{ row.code }} · {{ row.language || '技术栈未登记' }}</div>
            <div v-if="parseTagList(row.tags).length" class="inventory-tag-list inventory-tag-list--compact">
              <el-tag v-for="tag in parseTagList(row.tags).slice(0, 3)" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="负责人 / 部门" min-width="180">
          <template #default="{ row }">
            {{ row.ownerName || row.ownerUsername || '未设置' }}
            <div class="inventory-table-cell__sub">{{ row.departmentName || row.team || '未关联部门' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="运行环境" min-width="150">
          <template #default="{ row }">
            <span>{{ row.environmentName || '未关联环境' }}</span>
            <div class="inventory-table-cell__sub">{{ lifecycleLabel(row.lifecycle) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="重要级别" width="100">
          <template #default="{ row }"><el-tag size="small" :type="criticalityType(row.criticality)" effect="plain">{{ criticalityLabel(row.criticality) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="健康" width="110">
          <template #default="{ row }">
            <StatusTag :status="row.healthStatus" />
            <div class="inventory-table-cell__sub">{{ formatDateTime(row.healthCheckedAt) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="资产规模" min-width="220">
          <template #default="{ row }">
            <div class="inventory-tag-list">
              <el-tag size="small" effect="plain">域名 {{ row.domainCount || 0 }}</el-tag>
              <el-tag size="small" effect="plain">资源 {{ row.resourceCount || 0 }}</el-tag>
              <el-tag size="small" effect="plain">组件 {{ row.componentCount || 0 }}</el-tag>
              <el-tag size="small" effect="plain">依赖 {{ row.dependencyCount || 0 }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">查看详情</el-button>
            <el-button link type="primary" :loading="probingId === row.id" @click="probe(row)">立即检测</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && !applications.length" class="inventory-empty">暂无应用台账记录。</div>
      <div class="inventory-pagination">
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize" :total="pagination.total" layout="total, sizes, prev, pager, next" :page-sizes="[10, 20, 50]" @current-change="loadData" @size-change="loadData" />
      </div>
    </section>

    <el-dialog v-model="dialog.visible" width="900px" class="inventory-editor-dialog" destroy-on-close @closed="formRef?.resetFields()">
      <template #header>
        <EditorDialogHeader :icon="Grid" eyebrow="APPLICATION CATALOG" :title="dialog.editing ? '编辑应用资料' : '登记应用'" description="负责人、部门和运行环境均来自平台主数据，健康状态由资产探测自动计算。" />
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section">
          <div class="inventory-form-section__heading"><span>01</span><div><h4>身份与归属</h4><p>应用负责人和部门使用平台用户、组织架构数据。</p></div></div>
          <div class="inventory-form-grid">
            <el-form-item label="应用名称" prop="name"><el-input v-model="form.name" maxlength="120" placeholder="例如：订单中心" /></el-form-item>
            <el-form-item label="应用编码" prop="code"><el-input v-model="form.code" maxlength="80" placeholder="例如：order-center" :disabled="dialog.editing" /></el-form-item>
            <el-form-item label="负责人" prop="ownerUserId">
              <el-select v-model="form.ownerUserId" filterable placeholder="选择平台用户" @change="handleOwnerChange">
                <el-option v-for="item in references.users || []" :key="item.id" :label="`${item.name} (${item.code})${item.departmentId ? '' : ' · 未分配部门'}`" :value="item.id" :disabled="!item.departmentId" />
              </el-select>
            </el-form-item>
            <el-form-item label="所属部门（自动关联）" prop="departmentId">
              <div class="inventory-readonly-field"><span>{{ selectedOwnerDepartment || '请选择负责人' }}</span><small>来自平台组织架构</small></div>
            </el-form-item>
            <el-form-item label="运行环境" prop="environmentId" class="el-form-item--full">
              <el-select v-model="form.environmentId" filterable placeholder="选择共享运行环境">
                <el-option v-for="item in environments" :key="item.id" :label="`${item.name} · ${lifecycleLabel(item.kind)}${item.status === 'active' ? '' : ' · 已停用'}`" :value="item.id" :disabled="item.status !== 'active'" />
              </el-select>
              <div class="inventory-form-hint">一个环境可以关联多个应用；当前应用只选择一个运行环境。</div>
            </el-form-item>
          </div>
        </section>

        <section class="inventory-form-section">
          <div class="inventory-form-section__heading"><span>02</span><div><h4>治理信息</h4><p>健康状态由域名、资源和组件探测结果聚合生成。</p></div></div>
          <div class="inventory-form-grid inventory-form-grid--four">
            <el-form-item label="重要级别" prop="criticality"><el-select v-model="form.criticality"><el-option label="核心" value="critical" /><el-option label="高" value="high" /><el-option label="中" value="medium" /><el-option label="低" value="low" /></el-select></el-form-item>
            <el-form-item label="治理状态" prop="status"><el-select v-model="form.status"><el-option label="运行中" value="active" /><el-option label="已停用" value="disabled" /><el-option label="规划中" value="planned" /></el-select><div class="inventory-form-hint">仅控制应用是否纳入运行态治理，不代表健康结果。</div></el-form-item>
            <el-form-item label="健康状态"><div class="inventory-readonly-field"><StatusTag :status="form.healthStatus || 'unknown'" /><span>系统自动检测</span></div></el-form-item>
            <el-form-item label="最近检测"><div class="inventory-readonly-field"><span>{{ formatDateTime(form.healthCheckedAt) }}</span></div></el-form-item>
          </div>
        </section>

        <section class="inventory-form-section">
          <div class="inventory-form-section__heading"><span>03</span><div><h4>研发资料</h4><p>标签使用多选控件保存，不需要填写数组或 JSON。</p></div></div>
          <div class="inventory-form-grid">
            <el-form-item label="主要技术栈"><el-select v-model="form.language" filterable allow-create default-first-option clearable placeholder="选择或输入技术栈"><el-option v-for="item in languageOptions" :key="item" :label="item" :value="item" /></el-select></el-form-item>
            <el-form-item label="代码仓库" prop="repositoryUrl"><el-input v-model="form.repositoryUrl" maxlength="500" placeholder="https://git.example.com/team/repo" /></el-form-item>
            <el-form-item label="文档地址" prop="documentationUrl"><el-input v-model="form.documentationUrl" maxlength="500" placeholder="https://wiki.example.com/apps/order" /></el-form-item>
            <el-form-item label="业务标签"><el-select v-model="form.tags" multiple filterable allow-create default-first-option :multiple-limit="12" placeholder="选择或输入标签"><el-option v-for="item in tagOptions" :key="item" :label="item" :value="item" /></el-select></el-form-item>
            <el-form-item label="应用说明" class="el-form-item--full"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="1000" show-word-limit placeholder="说明应用职责、核心链路或维护注意事项" /></el-form-item>
          </div>
        </section>
      </el-form>
      <template #footer>
        <div class="inventory-dialog-footer"><span><i>*</i> 为必填项；健康状态和生命周期不由用户手动维护</span><div><el-button @click="dialog.visible = false">取消</el-button><el-button type="primary" :loading="dialog.saving" @click="submit">保存应用</el-button></div></div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Grid, OfficeBuilding, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { createApplication, deleteApplication, getOverview, getReferences, listApplications, listEnvironments, probeApplication, updateApplication, type Application, type Environment } from '../api'
import { normalizeTagList, parseTagList, serializeTagList, validateOptionalURL } from '../form-utils'
import EditorDialogHeader from './EditorDialogHeader.vue'
import PageHeader from './PageHeader.vue'
import StatusTag from './StatusTag.vue'

const router = useRouter()
const route = useRoute()
const formRef = ref<FormInstance>()
const loading = ref(false)
const applications = ref<Application[]>([])
const environments = ref<Environment[]>([])
const references = ref<Record<string, any[]>>({})
const probingId = ref(0)
const filters = reactive({ keyword: '', status: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const overviewCounts = ref<Record<string, number>>({})
const dialog = reactive({ visible: false, editing: false, saving: false, id: 0 })
const form = reactive<any>(defaultForm())
const languageOptions = ['Go', 'Java', 'Kotlin', 'Node.js', 'TypeScript', 'Python', 'PHP', 'C#', 'Rust']
const tagOptions = ['核心链路', '面向客户', '内部平台', '数据服务', '支付', '订单', '生产关键']
const rules: FormRules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }, { min: 2, max: 120, message: '应用名称长度为 2-120 个字符', trigger: 'blur' }],
  code: [{ required: true, message: '请输入应用编码', trigger: 'blur' }, { pattern: /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/, message: '编码只能包含字母、数字、点、下划线和短横线', trigger: 'blur' }],
  ownerUserId: [{ required: true, message: '请选择负责人', trigger: 'change' }],
  departmentId: [{ required: true, message: '负责人没有关联有效部门', trigger: 'change' }],
  environmentId: [{ required: true, message: '请选择运行环境', trigger: 'change' }],
  criticality: [{ required: true, message: '请选择重要级别', trigger: 'change' }],
  status: [{ required: true, message: '请选择应用状态', trigger: 'change' }],
  repositoryUrl: [{ validator: validateOptionalURL, trigger: 'blur' }],
  documentationUrl: [{ validator: validateOptionalURL, trigger: 'blur' }],
}

const activeCount = computed(() => overviewCounts.value.activeApplications || 0)
const environmentCount = computed(() => overviewCounts.value.applicationEnvironments || 0)
const unhealthyCount = computed(() => overviewCounts.value.unhealthyApplications || 0)
const selectedOwnerDepartment = computed(() => {
  const owner = (references.value.users || []).find(item => item.id === form.ownerUserId)
  return (references.value.departments || []).find(item => item.id === owner?.departmentId)?.name || ''
})

function defaultForm() {
  return { code: '', name: '', description: '', ownerUserId: undefined as number | undefined, departmentId: 0, environmentId: undefined as number | undefined, criticality: 'medium', status: 'active', healthStatus: 'unknown', healthCheckedAt: '', repositoryUrl: '', documentationUrl: '', language: '', tags: [] as string[] }
}
const lifecycleLabel = (value?: string) => ({ production: '生产', staging: '预发布', test: '测试', development: '开发' } as Record<string, string>)[value || ''] || value || '未设置'
const criticalityLabel = (value?: string) => ({ critical: '核心', high: '高', medium: '中', low: '低' } as Record<string, string>)[value || ''] || value || '中'
const criticalityType = (value?: string) => value === 'critical' || value === 'high' ? 'danger' : value === 'low' ? 'info' : 'warning'
const formatDateTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '尚未检测'

const loadOptions = async () => {
  const [refs, envs] = await Promise.all([getReferences(), listEnvironments()])
  references.value = refs || {}
  environments.value = envs || []
}
const loadData = async () => {
  loading.value = true
  try {
    const [data, overview] = await Promise.all([
      listApplications({ ...filters, page: pagination.page, page_size: pagination.pageSize }),
      getOverview(),
    ])
    applications.value = data?.list || []
    pagination.total = data?.total || 0
    overviewCounts.value = overview?.counts || {}
  } catch {
    ElMessage.error('加载应用台账失败')
  } finally { loading.value = false }
}
const resetFilters = () => { filters.keyword = ''; filters.status = ''; pagination.page = 1; loadData() }
const fillForm = (row?: Application) => {
  Object.assign(form, row ? { ...defaultForm(), ...row, tags: parseTagList(row.tags) } : defaultForm())
  if (!form.ownerUserId) form.ownerUserId = undefined
  if (!form.environmentId) form.environmentId = undefined
  if (form.ownerUserId) {
    const owner = (references.value.users || []).find(item => item.id === form.ownerUserId)
    form.departmentId = owner?.departmentId || 0
  }
}
const openCreate = () => { dialog.editing = false; dialog.id = 0; fillForm(); dialog.visible = true }
const openEdit = (row: Application) => { dialog.editing = true; dialog.id = row.id; fillForm(row); dialog.visible = true }
const openDetail = (row: Application) => router.push(`/app-inventory/apps/${row.id}`)
const handleOwnerChange = (id: number) => {
  const owner = (references.value.users || []).find(item => item.id === id)
  form.departmentId = owner?.departmentId || 0
  formRef.value?.validateField('departmentId')
}
const submit = async () => {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  dialog.saving = true
  try {
    const payload = { code: form.code, name: form.name, description: form.description, ownerUserId: form.ownerUserId, environmentId: form.environmentId, criticality: form.criticality, status: form.status, repositoryUrl: form.repositoryUrl, documentationUrl: form.documentationUrl, language: form.language, tags: serializeTagList(normalizeTagList(form.tags)) }
    if (dialog.editing) await updateApplication(dialog.id, payload)
    else await createApplication(payload)
    ElMessage.success(dialog.editing ? '应用更新成功' : '应用登记成功')
    dialog.visible = false
    await loadData()
  } finally { dialog.saving = false }
}
const probe = async (row: Application) => {
  probingId.value = row.id
  try { await probeApplication(row.id); ElMessage.success(`已完成“${row.name}”健康检测`); await loadData() } finally { probingId.value = 0 }
}
const remove = async (row: Application) => {
  await ElMessageBox.confirm(`删除应用“${row.name}”会移除其域名、资源、组件、依赖和发现记录，是否继续？`, '确认删除', { type: 'warning' })
  await deleteApplication(row.id)
  ElMessage.success('应用已删除')
  await loadData()
}

onMounted(async () => {
  await loadOptions()
  await loadData()
  if (route.query.create === '1') openCreate()
})
</script>

<style scoped>
.el-table { min-height: 280px; }
.inventory-pagination { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
