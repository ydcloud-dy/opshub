<template>
  <div :class="embedded ? 'inventory-embedded-page' : 'inventory-page'">
    <PageHeader v-if="!embedded" :icon="Lock" title="凭据中心" description="账号元数据和密文分离保存；列表永不返回密码，明文查看需要授权并记录审计。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ pagination.total }}</strong><span>凭据条目</span></div>
          <div class="inventory-header-metric"><strong>AES-256</strong><span>加密存储</span></div>
          <div class="inventory-header-metric"><strong>RBAC</strong><span>权限控制</span></div>
          <div class="inventory-header-metric"><strong>审计</strong><span>明文留痕</span></div>
        </div>
      </template>
      <el-button v-if="isAdmin" :icon="Document" @click="openAudits">查看审计</el-button>
      <el-button :icon="Refresh" :loading="loading" @click="loadData">刷新</el-button>
      <el-button v-if="isAdmin" type="primary" :icon="Plus" @click="openCreate">新增凭据</el-button>
    </PageHeader>

    <el-alert title="安全边界" type="info" :closable="false" show-icon style="margin-bottom:14px">
      <template #default>数据库只保存 AES-256-GCM 密文，主密钥来自服务器环境变量。所有查看明文操作都会记录用户、来源 IP、理由和结果。</template>
    </el-alert>

    <div class="inventory-toolbar">
      <el-input v-model="filters.keyword" clearable placeholder="名称、类型或用户名" :prefix-icon="Search" @keyup.enter="loadData" />
      <el-select v-model="filters.kind" clearable placeholder="凭据类型" @change="loadData"><el-option v-for="kind in credentialKinds" :key="kind.value" :label="kind.label" :value="kind.value" /></el-select>
      <el-select v-model="filters.status" clearable placeholder="状态" @change="loadData"><el-option label="正常" value="active" /><el-option label="即将过期" value="expiring" /><el-option label="已停用" value="disabled" /></el-select>
      <el-button type="primary" :icon="Search" @click="loadData">查询</el-button>
      <span v-if="embedded" class="inventory-toolbar__spacer" />
      <el-button v-if="embedded && isAdmin" :icon="Document" @click="openAudits">查看审计</el-button>
      <el-button v-if="embedded" :icon="Refresh" :loading="loading" @click="loadData">刷新</el-button>
      <el-button v-if="embedded && isAdmin" type="primary" :icon="Plus" @click="openCreate">新增凭据</el-button>
    </div>

    <section class="inventory-panel">
      <el-table v-loading="loading" :data="credentials" stripe>
        <el-table-column label="凭据" min-width="210">
          <template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ kindLabel(row.kind) }} · {{ scopeLabel(row.scope) }}</div></template>
        </el-table-column>
        <el-table-column prop="username" label="用户名 / 标识" min-width="160"><template #default="{ row }">{{ row.username || '-' }}</template></el-table-column>
        <el-table-column label="密文" width="120"><template #default="{ row }"><span style="font-family:monospace">{{ row.secretMask }}</span><el-icon v-if="row.hasSecret" color="#16803c" style="margin-left:6px"><CircleCheck /></el-icon></template></el-table-column>
        <el-table-column label="授权" width="100"><template #default="{ row }">{{ row.grantCount || 0 }} 条</template></el-table-column>
        <el-table-column label="到期时间" width="150"><template #default="{ row }"><span :style="expiryStyle(row.expiresAt)">{{ row.expiresAt ? formatDate(row.expiresAt) : '长期有效' }}</span></template></el-table-column>
        <el-table-column label="状态" width="90"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="!row.canReveal" @click="openReveal(row)">查看明文</el-button>
            <el-button v-if="row.canManage" link type="primary" @click="openGrants(row)">授权</el-button>
            <el-dropdown v-if="isAdmin" trigger="click" @command="(command:string) => handleCommand(command, row)">
              <el-button link type="primary">更多<el-icon><ArrowDown /></el-icon></el-button>
              <template #dropdown><el-dropdown-menu><el-dropdown-item command="edit">编辑</el-dropdown-item><el-dropdown-item command="delete" divided>删除</el-dropdown-item></el-dropdown-menu></template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && !credentials.length" class="inventory-empty">暂无凭据。账号密码不会直接保存在应用资产表中。</div>
      <div style="display:flex;justify-content:flex-end;margin-top:16px"><el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize" :total="pagination.total" layout="total, prev, pager, next" @current-change="loadData" /></div>
    </section>

    <el-dialog v-model="editor.visible" width="820px" class="inventory-editor-dialog" destroy-on-close>
      <template #header><EditorDialogHeader :icon="Lock" eyebrow="APPLICATION SECRET" :title="editor.id ? '编辑应用凭据' : '新增应用凭据'" description="密文与业务资产分离保存，使用、授权和查看明文均受权限控制。" /></template>
      <el-form ref="credentialFormRef" :model="form" :rules="credentialRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>凭据元数据</h4><p>列表仅展示这些信息，不返回任何密文内容。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="凭据名称" prop="name"><el-input v-model="form.name" maxlength="120" placeholder="例如：订单库生产账号" /></el-form-item>
          <el-form-item label="凭据类型" prop="kind"><el-select v-model="form.kind"><el-option v-for="kind in credentialKinds" :key="kind.value" :label="kind.label" :value="kind.value" /></el-select></el-form-item>
          <el-form-item label="用户名 / 标识"><el-input v-model="form.username" maxlength="255" autocomplete="off" placeholder="例如：order_app" /></el-form-item>
          <el-form-item label="共享范围" prop="scope"><el-select v-model="form.scope"><el-option label="私有" value="private" /><el-option label="应用内共享" value="application" /><el-option label="跨应用共享" value="shared" /></el-select></el-form-item>
          <el-form-item label="到期时间"><el-date-picker v-model="form.expiresAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable /></el-form-item>
          <el-form-item label="状态" prop="status"><el-select v-model="form.status"><el-option label="正常" value="active" /><el-option label="已停用" value="disabled" /></el-select></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="form.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>密文内容</h4><p>{{ editor.id ? '不填写则保留原密文；填写后会整体轮换并更新轮换时间。' : '根据凭据类型填写对应密文，至少填写一项。' }}</p></div></div>
          <el-alert v-if="editor.id" title="编辑页面不会回显历史密文。" type="warning" :closable="false" style="margin-bottom:14px" />
          <el-form-item prop="secret" class="el-form-item--full inventory-secret-group"><div class="inventory-form-grid">
            <el-form-item v-if="['password','database','middleware','other'].includes(form.kind)" label="密码"><el-input v-model="form.secret.password" type="password" show-password autocomplete="new-password" /></el-form-item>
            <el-form-item v-if="['token','other'].includes(form.kind)" label="访问令牌"><el-input v-model="form.secret.token" type="password" show-password autocomplete="off" /></el-form-item>
            <el-form-item v-if="form.kind === 'access-key'" label="AccessKey"><el-input v-model="form.secret.accessKey" type="password" show-password autocomplete="off" /></el-form-item>
            <el-form-item v-if="form.kind === 'access-key'" label="SecretKey"><el-input v-model="form.secret.secretKey" type="password" show-password autocomplete="off" /></el-form-item>
            <el-form-item v-if="form.kind === 'ssh-key'" label="私钥" class="el-form-item--full"><el-input v-model="form.secret.privateKey" type="textarea" :rows="5" autocomplete="off" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" /></el-form-item>
            <el-form-item v-if="form.kind === 'ssh-key'" label="私钥口令"><el-input v-model="form.secret.passphrase" type="password" show-password autocomplete="off" /></el-form-item>
          </div></el-form-item>
        </section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项，密文只提交到服务端加密</span><div><el-button @click="editor.visible=false">取消</el-button><el-button type="primary" :loading="editor.saving" @click="saveCredential">加密保存</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="reveal.visible" title="查看凭据明文" width="680px" destroy-on-close @closed="clearReveal">
      <template v-if="!reveal.data">
        <el-alert title="明文只在本次响应中返回，页面关闭后立即清空。请填写查看理由。" type="warning" :closable="false" style="margin-bottom:16px" />
        <el-form label-width="90px"><el-form-item label="查看理由"><el-input v-model="reveal.reason" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="例如：生产故障排查 / 凭据轮换核对" /></el-form-item></el-form>
      </template>
      <template v-else>
        <el-alert :title="`明文将在 ${reveal.secondsLeft} 秒后从页面清除`" type="info" :closable="false" style="margin-bottom:14px" />
        <el-descriptions :column="1" border>
          <el-descriptions-item label="用户名"><SecretValue :value="reveal.data.username" /></el-descriptions-item>
          <el-descriptions-item v-for="item in revealFields" :key="item.key" :label="item.label"><SecretValue :value="item.value" /></el-descriptions-item>
        </el-descriptions>
      </template>
      <template #footer><el-button @click="reveal.visible=false">关闭</el-button><el-button v-if="!reveal.data" type="primary" :loading="reveal.loading" :disabled="!reveal.reason.trim()" @click="doReveal">确认查看</el-button></template>
    </el-dialog>

    <el-dialog v-model="grantDialog.visible" :title="`凭据授权 - ${grantDialog.credential?.name || ''}`" width="760px">
      <el-table :data="grants" size="small" style="margin-bottom:18px">
        <el-table-column prop="subjectName" label="授权对象" min-width="150" />
        <el-table-column label="类型" width="90"><template #default="{ row }">{{ row.subjectType === 'role' ? '角色' : '用户' }}</template></el-table-column>
        <el-table-column label="权限" min-width="240"><template #default="{ row }"><div class="inventory-tag-list"><el-tag v-for="label in permissionLabels(row.permissions)" :key="label" size="small" effect="plain">{{ label }}</el-tag></div></template></el-table-column>
        <el-table-column width="65"><template #default="{ row }"><el-button link type="danger" @click="removeGrant(row)">删除</el-button></template></el-table-column>
      </el-table>
      <el-form :model="grantForm" label-width="90px" class="inventory-form-grid">
        <el-form-item label="主体类型"><el-segmented v-model="grantForm.subjectType" :options="[{label:'用户',value:'user'},{label:'角色',value:'role'}]" /></el-form-item>
        <el-form-item label="授权对象"><el-select v-model="grantForm.subjectId" filterable style="width:100%"><el-option v-for="item in grantSubjects" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item>
        <el-form-item label="权限范围" class="el-form-item--full"><el-checkbox-group v-model="grantForm.permissions"><el-checkbox :value="1">查看元数据</el-checkbox><el-checkbox :value="2">供系统使用</el-checkbox><el-checkbox :value="4">查看明文</el-checkbox><el-checkbox :value="8">管理授权</el-checkbox></el-checkbox-group></el-form-item>
      </el-form>
      <template #footer><el-button @click="grantDialog.visible=false">关闭</el-button><el-button type="primary" @click="saveGrant">保存授权</el-button></template>
    </el-dialog>

    <el-drawer v-model="auditDrawer" title="凭据明文查看审计" size="720px">
      <el-table :data="audits" size="small" stripe>
        <el-table-column prop="username" label="用户" width="100" />
        <el-table-column label="结果" width="75"><template #default="{ row }"><el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? '成功' : '拒绝' }}</el-tag></template></el-table-column>
        <el-table-column prop="reason" label="理由" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="来源 IP" width="130" />
        <el-table-column label="时间" width="170"><template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template></el-table-column>
      </el-table>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElButton, ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { ArrowDown, CircleCheck, Document, Lock, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import PageHeader from './PageHeader.vue'
import EditorDialogHeader from './EditorDialogHeader.vue'
import {
  createCredential, deleteCredential, deleteCredentialGrant, getReferences, listCredentialGrants,
  listCredentials, listSecretAudits, revealCredential, updateCredential, upsertCredentialGrant,
  type Credential,
} from '../api'
import StatusTag from './StatusTag.vue'

withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const userStore = useUserStore()
const loading = ref(false)
const credentials = ref<Credential[]>([])
const filters = reactive({ keyword: '', kind: '', status: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const references = ref<Record<string, any[]>>({})
const editor = reactive({ visible: false, id: 0, saving: false })
const credentialFormRef = ref<FormInstance>()
const form = reactive<any>(defaultForm())
const reveal = reactive<any>({ visible: false, credential: null, reason: '', data: null, loading: false, secondsLeft: 60, timer: 0 })
const grantDialog = reactive<any>({ visible: false, credential: null })
const grants = ref<any[]>([])
const grantForm = reactive<any>({ subjectType: 'user', subjectId: undefined as number | undefined, permissions: [1, 2] })
const auditDrawer = ref(false)
const audits = ref<any[]>([])
const credentialKinds = [{ label: '用户名密码', value: 'password' }, { label: '访问令牌', value: 'token' }, { label: '云 AccessKey', value: 'access-key' }, { label: 'SSH 私钥', value: 'ssh-key' }, { label: '数据库账号', value: 'database' }, { label: '中间件账号', value: 'middleware' }, { label: '其他', value: 'other' }]
const credentialRules: FormRules = {
  name: [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
  kind: [{ required: true, message: '请选择凭据类型', trigger: 'change' }],
  scope: [{ required: true, message: '请选择共享范围', trigger: 'change' }],
  status: [{ required: true, message: '请选择凭据状态', trigger: 'change' }],
  secret: [{ validator: (_rule, value, callback) => editor.id || Object.values(value || {}).some(item => String(item || '').trim()) ? callback() : callback(new Error('请填写当前凭据类型对应的密文内容')), trigger: 'change' }],
}
const isAdmin = computed(() => userStore.userInfo?.username === 'admin' || (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))
const grantSubjects = computed(() => grantForm.subjectType === 'role' ? references.value.roles || [] : references.value.users || [])
const revealFields = computed(() => {
  const secret = reveal.data?.secret || {}
  const labels: Record<string, string> = { password: '密码', token: '访问令牌', accessKey: 'AccessKey', secretKey: 'SecretKey', privateKey: '私钥', passphrase: '私钥口令' }
  const fields = Object.entries(labels).filter(([key]) => secret[key]).map(([key, label]) => ({ key, label, value: secret[key] }))
  Object.entries(secret.extra || {}).forEach(([key, value]) => fields.push({ key: `extra-${key}`, label: key, value }))
  return fields
})

function defaultForm() { return { name: '', kind: 'password', username: '', scope: 'private', status: 'active', description: '', expiresAt: null, secret: { password: '', token: '', accessKey: '', secretKey: '', privateKey: '', passphrase: '' } } }
const loadData = async () => { loading.value = true; try { const data = await listCredentials({ ...filters, page: pagination.page, page_size: pagination.pageSize }); credentials.value = data?.list || []; pagination.total = data?.total || 0 } finally { loading.value = false } }
const openCreate = () => { editor.id = 0; Object.assign(form, defaultForm()); editor.visible = true }
const openEdit = (row: Credential) => { editor.id = row.id; Object.assign(form, { ...defaultForm(), ...row, secret: { ...defaultForm().secret } }); editor.visible = true }
const saveCredential = async () => { if (!credentialFormRef.value || !await credentialFormRef.value.validate().catch(() => false)) return; editor.saving = true; try { const payload = { ...form, secret: { ...form.secret } }; editor.id ? await updateCredential(editor.id, payload) : await createCredential(payload); ElMessage.success('凭据已加密保存'); editor.visible = false; await loadData() } finally { editor.saving = false } }
const handleCommand = (command: string, row: Credential) => command === 'edit' ? openEdit(row) : removeCredential(row)
const removeCredential = async (row: Credential) => { await ElMessageBox.confirm(`确认删除凭据“${row.name}”？关联的资源和组件会保留，但将解除凭据关联。`, '删除确认', { type: 'warning' }); await deleteCredential(row.id); ElMessage.success('凭据已删除'); await loadData() }

const openReveal = (row: Credential) => { clearReveal(); reveal.credential = row; reveal.visible = true }
const doReveal = async () => { reveal.loading = true; try { reveal.data = await revealCredential(reveal.credential.id, reveal.reason); reveal.secondsLeft = 60; reveal.timer = window.setInterval(() => { reveal.secondsLeft--; if (reveal.secondsLeft <= 0) { clearReveal(); reveal.visible = false } }, 1000) } finally { reveal.loading = false } }
const clearReveal = () => { if (reveal.timer) window.clearInterval(reveal.timer); reveal.timer = 0; reveal.data = null; reveal.reason = ''; reveal.secondsLeft = 60 }

const openGrants = async (row: Credential) => { grantDialog.credential = row; grantDialog.visible = true; Object.assign(grantForm, { subjectType: 'user', subjectId: undefined, permissions: [1, 2] }); grants.value = await listCredentialGrants(row.id) }
const permissionMask = () => grantForm.permissions.reduce((mask: number, value: number) => mask | value, 0)
const permissionLabels = (mask: number) => [{ value: 1, label: '查看' }, { value: 2, label: '使用' }, { value: 4, label: '明文' }, { value: 8, label: '管理' }].filter(item => mask & item.value).map(item => item.label)
const saveGrant = async () => { if (!grantForm.subjectId || !grantForm.permissions.length) { ElMessage.warning('请选择授权对象和权限'); return } await upsertCredentialGrant(grantDialog.credential.id, { subjectType: grantForm.subjectType, subjectId: grantForm.subjectId, permissions: permissionMask() }); grants.value = await listCredentialGrants(grantDialog.credential.id); ElMessage.success('授权已保存'); await loadData() }
const removeGrant = async (row: any) => { await deleteCredentialGrant(row.id); grants.value = await listCredentialGrants(grantDialog.credential.id); await loadData() }
const openAudits = async () => { audits.value = await listSecretAudits(); auditDrawer.value = true }

const kindLabel = (kind: string) => credentialKinds.find(item => item.value === kind)?.label || kind
const scopeLabel = (scope: string) => ({ private: '私有', application: '应用内共享', shared: '跨应用共享' }[scope] || scope)
const formatDate = (value: string) => new Date(value).toLocaleDateString('zh-CN')
const formatDateTime = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const expiryStyle = (value?: string) => value && new Date(value).getTime() < Date.now() + 30 * 86400000 ? { color: '#d97706', fontWeight: 600 } : {}

const SecretValue = (_props: any, context: any) => {
  const value = context.attrs.value || '-'
  const copy = async () => { await navigator.clipboard.writeText(String(value)); ElMessage.success('已复制') }
  return h('div', { style: 'display:flex;align-items:center;justify-content:space-between;gap:12px;min-width:0' }, [h('code', { style: 'white-space:pre-wrap;word-break:break-all' }, String(value)), h(ElButton, { link: true, type: 'primary', onClick: copy }, () => '复制')])
}

onMounted(async () => { references.value = await getReferences(); await loadData() })
onBeforeUnmount(clearReveal)
</script>
