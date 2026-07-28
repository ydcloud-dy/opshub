<template>
  <section class="panel access-panel">
    <div class="panel-head">
      <div>
        <h3>日志访问控制</h3>
        <p>按用户或角色授权采集策略，并叠加主机分组和 Kubernetes 资产权限</p>
      </div>
      <el-button v-if="isAdmin" type="primary" class="primary-action" @click="openCreate">
        <el-icon><Plus /></el-icon>
        新增策略
      </el-button>
    </div>

    <el-alert
      v-if="!isAdmin"
      type="info"
      :closable="false"
      show-icon
      title="日志权限由管理员统一配置"
      description="未配置有效访问策略时，非管理员账号默认不能查询日志。"
    />

    <template v-else>
      <el-alert
        class="security-alert"
        type="warning"
        :closable="false"
        show-icon
        title="非管理员默认无日志访问权限"
        description="授权范围最终取采集策略范围与资产权限的交集。"
      />
      <el-table v-loading="loading" :data="items" empty-text="暂无访问策略，非管理员默认不能查询日志">
        <el-table-column label="策略" min-width="190">
          <template #default="{ row }">
            <div class="name-cell">
              <strong>{{ row.payload.name }}</strong>
              <small>{{ row.payload.description || '未填写说明' }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="授权对象" min-width="150">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.payload.subjectType === 'user' ? '用户' : '角色' }}</el-tag>
            {{ subjectName(row) }}
          </template>
        </el-table-column>
        <el-table-column label="采集策略范围" min-width="250">
          <template #default="{ row }">
            <el-tag v-if="row.payload.scopeMode === 'all'" size="small" type="warning">全部采集策略</el-tag>
            <div v-else-if="policyNames(row).length" class="scope-tags">
              <el-tooltip :content="policyNames(row).join('、')" placement="top" :show-after="400">
                <div class="scope-tags-inner">
                  <el-tag v-for="name in policyNames(row).slice(0, 2)" :key="name" size="small" type="success">{{ name }}</el-tag>
                  <span v-if="policyNames(row).length > 2" class="more-count">+{{ policyNames(row).length - 2 }}</span>
                </div>
              </el-tooltip>
            </div>
            <span v-else class="empty-scope">未选择</span>
          </template>
        </el-table-column>
        <el-table-column label="允许操作" min-width="180">
          <template #default="{ row }">
            <div class="tag-list">
              <el-tag v-for="action in row.payload.allowedActions" :key="action" size="small">{{ actionText(action) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="禁止查看字段" min-width="170">
          <template #default="{ row }">{{ row.payload.deniedFields.join(', ') || '-' }}</template>
        </el-table-column>
        <el-table-column label="脱敏显示字段" min-width="170">
          <template #default="{ row }">{{ row.payload.maskFields.join(', ') || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.payload.enabled ? 'success' : 'info'">{{ row.payload.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link @click="openEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除此访问策略？" @confirm="remove(row)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <el-dialog
      v-model="visible"
      :title="editingId ? '编辑访问策略' : '新增访问策略'"
      width="min(880px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form :model="form" label-position="top">
        <div class="form-grid">
          <el-form-item label="策略名称" required><el-input v-model="form.name" placeholder="例如：生产环境日志只读" /></el-form-item>
          <el-form-item label="适用存储">
            <el-select v-model="form.storageId" clearable placeholder="全部存储">
              <el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="授权对象类型">
            <el-radio-group v-model="form.subjectType" @change="form.subjectId = undefined">
              <el-radio-button value="user">用户</el-radio-button>
              <el-radio-button value="role">角色</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="授权对象" required>
            <el-select v-model="form.subjectId" filterable placeholder="请选择用户或角色">
              <el-option v-for="item in subjectOptions" :key="item.id" :label="item.label" :value="item.id" />
            </el-select>
          </el-form-item>

          <el-form-item label="日志范围" required class="full scope-form-item">
            <div class="scope-editor">
              <el-radio-group v-model="form.scopeMode" class="scope-mode">
                <el-radio-button value="collection_policy">指定采集策略</el-radio-button>
                <el-radio-button value="all">全部采集策略</el-radio-button>
              </el-radio-group>
              <el-select
                v-if="form.scopeMode === 'collection_policy'"
                v-model="form.collectionPolicyIds"
                class="policy-select"
                multiple
                filterable
                collapse-tags
                collapse-tags-tooltip
                :max-collapse-tags="3"
                placeholder="请选择允许查询的采集策略"
              >
                <el-option
                  v-for="item in options.collectionPolicies"
                  :key="item.id"
                  :label="collectionPolicyLabel(item)"
                  :value="item.id"
                >
                  <div class="policy-option">
                    <span class="policy-option-name">{{ item.name }}</span>
                    <span class="policy-option-meta">{{ sourceModeText(item.sourceMode) }}<template v-if="item.environment"> · {{ item.environment }}</template></span>
                    <span class="policy-option-status" :class="`is-${item.status}`">{{ collectionPolicyStatusText(item.status) }}</span>
                  </div>
                </el-option>
              </el-select>
              <div class="scope-note" :class="{ danger: form.scopeMode === 'all' }">
                {{ form.scopeMode === 'all' ? '该对象可访问全部采集策略产生的日志，仍受资产权限限制。' : '可同时选择多个采集策略，最终范围与资产权限取交集。' }}
              </div>
            </div>
          </el-form-item>

          <el-form-item label="允许操作" class="full">
            <el-checkbox-group v-model="form.allowedActions" class="action-options">
              <el-checkbox value="query">查询与上下文</el-checkbox>
              <el-checkbox value="tail">实时 Tail</el-checkbox>
              <el-checkbox value="export">异步导出</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="禁止查看字段" class="full">
            <el-select v-model="form.deniedFields" multiple filterable allow-create default-first-option placeholder="输入字段名后回车，例如 authorization">
              <el-option v-for="field in fieldSuggestions" :key="field" :label="field" :value="field" />
            </el-select>
            <div class="form-tip">选中字段不会出现在列表、日志详情、上下文及导出文件中。</div>
          </el-form-item>
          <el-form-item label="脱敏显示字段" class="full">
            <el-select v-model="form.maskFields" multiple filterable allow-create default-first-option placeholder="输入字段名后回车">
              <el-option v-for="field in fieldSuggestions" :key="field" :label="field" :value="field" />
            </el-select>
            <div class="form-tip">字段保留但内容统一显示为 ******；禁止查看的优先级更高。</div>
          </el-form-item>
          <el-form-item label="策略说明" class="full"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="启用策略"><el-switch v-model="form.enabled" /></el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" class="primary-action" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import {
  createLogAccessPolicy,
  deleteLogAccessPolicy,
  getLogAccessPolicies,
  getLogAccessPolicyOptions,
  updateLogAccessPolicy,
  type LogAccessPolicy,
  type LogStorageCluster,
} from '@/api/logcenter'

interface CollectionPolicyOption {
  id: number
  name: string
  sourceMode: string
  status: string
  environment?: string
}

defineProps<{ storages: LogStorageCluster[] }>()
const userStore = useUserStore()
const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref<number>()
const items = ref<LogAccessPolicy[]>([])
const options = reactive<{
  users: Array<{ id: number; username: string; realName?: string }>
  roles: Array<{ id: number; name: string; code: string }>
  collectionPolicies: CollectionPolicyOption[]
}>({ users: [], roles: [], collectionPolicies: [] })

const defaultForm = () => ({
  name: '',
  description: '',
  subjectType: 'role' as 'user' | 'role',
  subjectId: undefined as number | undefined,
  storageId: undefined as number | undefined,
  libraryItemPattern: '',
  scopeMode: 'collection_policy' as 'all' | 'collection_policy',
  collectionPolicyIds: [] as number[],
  allowedActions: ['query', 'tail', 'export'],
  deniedFields: [] as string[],
  maskFields: [] as string[],
  enabled: true,
})
const form = reactive(defaultForm())
const fieldSuggestions = ['body', 'authorization', 'cookie', 'token', 'password', 'secret', 'attributes', 'resourceAttributes', 'labels.namespace', 'labels.service', 'filePath', 'containerImage']

const subjectOptions = computed(() => form.subjectType === 'user'
  ? options.users.map(item => ({ id: item.id, label: `${item.realName || item.username} (${item.username})` }))
  : options.roles.map(item => ({ id: item.id, label: `${item.name} (${item.code})` })))

const load = async () => {
  if (!isAdmin.value) return
  loading.value = true
  try {
    const [policies, metadata] = await Promise.all([getLogAccessPolicies(), getLogAccessPolicyOptions()])
    items.value = policies || []
    Object.assign(options, metadata || {})
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editingId.value = undefined
  Object.assign(form, defaultForm())
  visible.value = true
}

const openEdit = (row: LogAccessPolicy) => {
  editingId.value = row.id
  Object.assign(form, defaultForm(), JSON.parse(JSON.stringify(row.payload)))
  form.collectionPolicyIds = [...(row.payload.collectionPolicyIds || [])]
  visible.value = true
}

const save = async () => {
  if (!form.name.trim() || !form.subjectId) return ElMessage.warning('请填写策略名称并选择授权对象')
  if (!form.allowedActions.length) return ElMessage.warning('至少允许一种日志操作')
  if (form.scopeMode === 'collection_policy' && !form.collectionPolicyIds.length) return ElMessage.warning('请至少选择一个采集策略')
  saving.value = true
  try {
    const payload = JSON.parse(JSON.stringify({ ...form, subjectId: form.subjectId }))
    editingId.value ? await updateLogAccessPolicy(editingId.value, payload) : await createLogAccessPolicy(payload)
    ElMessage.success('访问策略已保存')
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const remove = async (row: LogAccessPolicy) => {
  await deleteLogAccessPolicy(row.id)
  ElMessage.success('访问策略已删除')
  await load()
}

const subjectName = (row: LogAccessPolicy) => row.payload.subjectType === 'user'
  ? options.users.find(item => item.id === row.payload.subjectId)?.realName || options.users.find(item => item.id === row.payload.subjectId)?.username || `用户 #${row.payload.subjectId}`
  : options.roles.find(item => item.id === row.payload.subjectId)?.name || `角色 #${row.payload.subjectId}`

const policyNames = (row: LogAccessPolicy) => (row.payload.collectionPolicyIds || []).map((id) => {
  const policy = options.collectionPolicies.find(item => item.id === id)
  return policy?.name || `采集策略 #${id}（已删除）`
})

const collectionPolicyLabel = (item: CollectionPolicyOption) => [item.name, sourceModeText(item.sourceMode), item.environment].filter(Boolean).join(' · ')
const sourceModeText = (mode: string) => mode === 'kubernetes' ? 'Kubernetes' : mode === 'host' ? '主机' : mode
const collectionPolicyStatusText = (status: string) => ({ draft: '草稿', published: '已发布', disabled: '已停用', archived: '已归档' }[status] || status)
const actionText = (action: string) => ({ query: '查询', tail: 'Tail', export: '导出' }[action] || action)

onMounted(load)
</script>

<style scoped>
.access-panel{padding:20px}.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:18px}.panel-head h3{margin:0;color:#111827;font-size:15px}.panel-head p{margin:6px 0 0;color:#667085;font-size:13px}.security-alert{margin-bottom:16px}.name-cell strong,.name-cell small{display:block}.name-cell small{margin-top:4px;color:#98a2b3;font-size:12px}.tag-list,.scope-tags-inner{display:flex;align-items:center;flex-wrap:wrap;gap:5px}.scope-tags{min-width:0}.more-count{color:#667085;font-size:12px}.empty-scope{color:#d92d20;font-size:12px}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:0 20px}.form-grid :deep(.el-select){width:100%}.full{grid-column:1/-1}.scope-form-item{margin-top:4px}.scope-editor{width:100%;padding:16px;border:1px solid #e4e7ec;border-radius:6px;background:#f9fafb}.scope-mode{margin-bottom:14px}.policy-select{display:block}.scope-note{margin-top:9px;color:#667085;font-size:12px}.scope-note.danger{color:#b54708}.action-options{display:flex;flex-wrap:wrap;gap:8px 24px}.form-tip{margin-top:6px;color:#98a2b3;font-size:12px}.policy-option{display:grid;grid-template-columns:minmax(140px,1fr) auto auto;align-items:center;gap:14px;width:100%}.policy-option-name{overflow:hidden;color:#344054;text-overflow:ellipsis}.policy-option-meta{color:#667085;font-size:12px}.policy-option-status{color:#667085;font-size:12px}.policy-option-status.is-published{color:#067647}.policy-option-status.is-disabled,.policy-option-status.is-archived{color:#b42318}@media(max-width:700px){.panel-head{align-items:stretch;flex-direction:column}.form-grid{grid-template-columns:1fr}.full{grid-column:auto}.scope-editor{box-sizing:border-box;padding:12px}.policy-option{grid-template-columns:minmax(0,1fr) auto}.policy-option-meta{display:none}}
</style>
