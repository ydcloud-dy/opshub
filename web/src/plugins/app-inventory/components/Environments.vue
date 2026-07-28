<template>
  <div class="inventory-page">
    <PageHeader :icon="SetUp" title="环境管理" description="统一维护共享运行环境；一个环境可以承载多个应用，每个应用只关联一个环境。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ environments.length }}</strong><span>环境总数</span></div>
          <div class="inventory-header-metric"><strong>{{ activeCount }}</strong><span>可用环境</span></div>
          <div class="inventory-header-metric"><strong>{{ productionCount }}</strong><span>生产环境</span></div>
          <div class="inventory-header-metric"><strong>{{ coveredApplicationCount }}</strong><span>关联应用</span></div>
        </div>
      </template>
      <el-button :icon="Refresh" :loading="loading" @click="loadData">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增环境</el-button>
    </PageHeader>

    <div class="inventory-toolbar">
      <el-select v-model="filters.applicationId" clearable filterable placeholder="按关联应用筛选">
        <el-option v-for="app in applications" :key="app.id" :label="app.name" :value="app.id" />
      </el-select>
      <el-select v-model="filters.kind" clearable placeholder="环境类型">
        <el-option v-for="item in environmentKinds" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-input v-model="filters.keyword" clearable placeholder="搜索环境名称、编码或区域" :prefix-icon="Search" />
      <el-select v-model="filters.status" clearable placeholder="状态"><el-option label="可用" value="active" /><el-option label="已停用" value="disabled" /></el-select>
      <el-button text @click="resetFilters">重置</el-button>
    </div>

    <section class="inventory-panel">
      <el-table v-loading="loading" :data="filteredEnvironments" stripe>
        <el-table-column label="环境" min-width="220"><template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ row.code }}</div></template></el-table-column>
        <el-table-column label="类型" width="120"><template #default="{ row }"><el-tag size="small" effect="plain">{{ kindLabel(row.kind) }}</el-tag></template></el-table-column>
        <el-table-column prop="region" label="区域" min-width="150"><template #default="{ row }">{{ row.region || '未设置' }}</template></el-table-column>
        <el-table-column label="关联应用" min-width="180">
          <template #default="{ row }">
            <el-popover v-if="environmentApplicationCount(row)" placement="bottom-start" :width="260" trigger="click">
              <template #reference><el-button link type="primary">{{ environmentApplicationCount(row) }} 个应用</el-button></template>
              <div class="inventory-related-list">
                <button v-for="app in applicationsForEnvironment(row.id)" :key="app.id" type="button" @click="router.push(`/app-inventory/apps/${app.id}`)"><span>{{ app.name }}</span><small>{{ app.code }}</small></button>
              </div>
            </el-popover>
            <span v-else class="inventory-muted">暂未使用</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="更新时间" width="170"><template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template></el-table-column>
        <el-table-column label="操作" width="150" fixed="right"><template #default="{ row }"><el-button link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button><el-button link type="danger" :icon="Delete" @click="remove(row)">删除</el-button></template></el-table-column>
      </el-table>
      <div v-if="!loading && !filteredEnvironments.length" class="inventory-empty">当前筛选条件下没有环境记录。</div>
    </section>

    <el-dialog v-model="dialog.visible" width="760px" class="inventory-editor-dialog" destroy-on-close @closed="formRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="SetUp" eyebrow="SHARED RUNTIME" :title="dialog.id ? '编辑共享环境' : '新增共享环境'" description="环境是平台级运行边界，应用登记时从这里选择，不再在环境上绑定单个应用。" /></template>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section">
          <div class="inventory-form-section__heading"><span>01</span><div><h4>环境标识</h4><p>编码全局唯一，可供多个应用共同选择。</p></div></div>
          <div class="inventory-form-grid">
            <el-form-item label="环境名称" prop="name"><el-input v-model="form.name" maxlength="80" placeholder="例如：华南生产环境" /></el-form-item>
            <el-form-item label="环境编码" prop="code"><el-input v-model="form.code" maxlength="40" placeholder="例如：prod-south" /></el-form-item>
            <el-form-item label="环境类型" prop="kind"><el-select v-model="form.kind"><el-option v-for="item in environmentKinds" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
            <el-form-item label="区域"><el-input v-model="form.region" maxlength="100" placeholder="例如：华南 / IDC-A / cn-shenzhen" /></el-form-item>
          </div>
        </section>
        <section class="inventory-form-section">
          <div class="inventory-form-section__heading"><span>02</span><div><h4>治理状态</h4><p>正在被应用使用的环境不能停用或删除。</p></div></div>
          <div class="inventory-form-grid">
            <el-form-item label="状态" prop="status"><el-select v-model="form.status"><el-option label="可用" value="active" /><el-option label="已停用" value="disabled" :disabled="Boolean(form.applicationCount)" /></el-select></el-form-item>
            <el-form-item label="当前关联"><div class="inventory-readonly-field"><span>{{ dialog.id ? `${form.applicationCount || 0} 个应用` : '新环境暂未关联应用' }}</span><small v-if="form.applicationCount">使用中不可停用</small></div></el-form-item>
            <el-form-item label="说明" class="el-form-item--full"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="补充网络边界、可用区或维护约束" /></el-form-item>
          </div>
        </section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项</span><div><el-button @click="dialog.visible = false">取消</el-button><el-button type="primary" :loading="dialog.saving" @click="submit">保存环境</el-button></div></div></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Delete, Edit, Plus, Refresh, Search, SetUp } from '@element-plus/icons-vue'
import { createEnvironment, deleteEnvironment, listApplications, listEnvironments, updateEnvironment, type Application, type Environment } from '../api'
import EditorDialogHeader from './EditorDialogHeader.vue'
import PageHeader from './PageHeader.vue'
import StatusTag from './StatusTag.vue'

const router = useRouter()
const route = useRoute()
const formRef = ref<FormInstance>()
const loading = ref(false)
const applications = ref<Application[]>([])
const environments = ref<Environment[]>([])
const filters = reactive({ applicationId: route.query.app_id ? Number(route.query.app_id) : undefined as number | undefined, kind: '', keyword: '', status: '' })
const dialog = reactive({ visible: false, id: 0, saving: false })
const form = reactive<any>(defaultForm())
const environmentKinds = [{ label: '生产', value: 'production' }, { label: '预发布', value: 'staging' }, { label: '测试', value: 'test' }, { label: '开发', value: 'development' }]
const rules: FormRules = {
  name: [{ required: true, message: '请输入环境名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入环境编码', trigger: 'blur' }, { pattern: /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/, message: '编码只能包含字母、数字、点、下划线和短横线', trigger: 'blur' }],
  kind: [{ required: true, message: '请选择环境类型', trigger: 'change' }],
  status: [{ required: true, message: '请选择环境状态', trigger: 'change' }],
}

const applicationsForEnvironment = (id: number) => applications.value.filter(item => item.environmentId === id)
const environmentApplicationCount = (environment: Environment) => environment.applicationCount ?? applicationsForEnvironment(environment.id).length
const filteredEnvironments = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  const selectedEnvironmentId = filters.applicationId ? applications.value.find(item => item.id === filters.applicationId)?.environmentId : 0
  return environments.value.filter(item => {
    if (selectedEnvironmentId && item.id !== selectedEnvironmentId) return false
    if (filters.kind && item.kind !== filters.kind) return false
    if (filters.status && item.status !== filters.status) return false
    return !keyword || [item.name, item.code, item.region].some(value => String(value || '').toLowerCase().includes(keyword))
  })
})
const activeCount = computed(() => environments.value.filter(item => item.status === 'active').length)
const productionCount = computed(() => environments.value.filter(item => item.kind === 'production').length)
const coveredApplicationCount = computed(() => environments.value.reduce((total, item) => total + environmentApplicationCount(item), 0))

function defaultForm() { return { name: '', code: '', kind: 'production', region: '', status: 'active', description: '' } }
const kindLabel = (value?: string) => environmentKinds.find(item => item.value === value)?.label || value || '未设置'
const formatDateTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
const loadData = async () => {
  loading.value = true
  try {
    const [appData, environmentData] = await Promise.all([listApplications({ page: 1, page_size: 200 }), listEnvironments()])
    applications.value = appData?.list || []
    environments.value = environmentData || []
  } catch { ElMessage.error('加载环境列表失败') } finally { loading.value = false }
}
const resetFilters = () => { filters.applicationId = undefined; filters.kind = ''; filters.keyword = ''; filters.status = '' }
const openCreate = () => { Object.assign(form, defaultForm()); dialog.id = 0; dialog.visible = true }
const openEdit = (row: Environment) => { Object.assign(form, { ...defaultForm(), ...row }); dialog.id = row.id; dialog.visible = true }
const submit = async () => {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  dialog.saving = true
  try {
    if (dialog.id) await updateEnvironment(dialog.id, { ...form })
    else await createEnvironment({ ...form })
    ElMessage.success(dialog.id ? '环境更新成功' : '环境创建成功')
    dialog.visible = false
    await loadData()
  } finally { dialog.saving = false }
}
const remove = async (row: Environment) => {
  await ElMessageBox.confirm(`确认删除环境“${row.name}”？只有未被任何应用使用的环境可以删除。`, '确认删除', { type: 'warning' })
  await deleteEnvironment(row.id)
  ElMessage.success('环境已删除')
  await loadData()
}

onMounted(loadData)
</script>
