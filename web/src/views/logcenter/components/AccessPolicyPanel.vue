<template>
  <section class="panel access-panel">
    <div class="panel-head"><div><h3>日志访问控制</h3><p>自动继承主机分组和 Kubernetes 授权，并可进一步限制查询、Tail、导出及敏感字段</p></div><el-button v-if="isAdmin" type="primary" class="primary-action" @click="openCreate"><el-icon><Plus /></el-icon>新增策略</el-button></div>
    <el-alert v-if="!isAdmin" type="info" :closable="false" show-icon title="当前账号使用资产权限自动过滤日志" description="只有管理员可以配置字段隐藏、掩码和操作权限。" />
    <template v-else>
      <el-table v-loading="loading" :data="items" empty-text="暂无附加访问策略，用户默认按资产权限查询">
        <el-table-column label="策略" min-width="200"><template #default="{ row }"><div class="name-cell"><strong>{{ row.payload.name }}</strong><small>{{ row.payload.description || '未填写说明' }}</small></div></template></el-table-column>
        <el-table-column label="授权对象" min-width="150"><template #default="{ row }"><el-tag size="small" type="info">{{ row.payload.subjectType === 'user' ? '用户' : '角色' }}</el-tag> {{ subjectName(row) }}</template></el-table-column>
        <el-table-column label="允许操作" min-width="190"><template #default="{ row }"><div class="tag-list"><el-tag v-for="action in row.payload.allowedActions" :key="action" size="small">{{ actionText(action) }}</el-tag></div></template></el-table-column>
        <el-table-column label="隐藏字段" min-width="170"><template #default="{ row }">{{ row.payload.deniedFields.join(', ') || '-' }}</template></el-table-column>
        <el-table-column label="掩码字段" min-width="170"><template #default="{ row }">{{ row.payload.maskFields.join(', ') || '-' }}</template></el-table-column>
        <el-table-column label="状态" width="90"><template #default="{ row }"><el-tag size="small" :type="row.payload.enabled ? 'success' : 'info'">{{ row.payload.enabled ? '启用' : '停用' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="140" fixed="right"><template #default="{ row }"><el-button link @click="openEdit(row)">编辑</el-button><el-popconfirm title="确定删除此访问策略？" @confirm="remove(row)"><template #reference><el-button link type="danger">删除</el-button></template></el-popconfirm></template></el-table-column>
      </el-table>
    </template>

    <el-dialog v-model="visible" :title="editingId ? '编辑访问策略' : '新增访问策略'" width="min(760px, calc(100vw - 32px))" destroy-on-close>
      <el-form :model="form" label-position="top"><div class="form-grid">
        <el-form-item label="策略名称" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="适用存储"><el-select v-model="form.storageId" clearable placeholder="全部存储"><el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item>
        <el-form-item label="授权对象类型"><el-radio-group v-model="form.subjectType"><el-radio-button value="user">用户</el-radio-button><el-radio-button value="role">角色</el-radio-button></el-radio-group></el-form-item>
        <el-form-item label="授权对象" required><el-select v-model="form.subjectId" filterable><el-option v-for="item in subjectOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item label="允许操作" class="full"><el-checkbox-group v-model="form.allowedActions"><el-checkbox value="query">查询与上下文</el-checkbox><el-checkbox value="tail">实时 Tail</el-checkbox><el-checkbox value="export">异步导出</el-checkbox></el-checkbox-group></el-form-item>
        <el-form-item label="隐藏字段" class="full"><el-select v-model="form.deniedFields" multiple filterable allow-create default-first-option placeholder="输入字段名后回车，例如 authorization"><el-option v-for="field in fieldSuggestions" :key="field" :label="field" :value="field" /></el-select><div class="form-tip">字段将从列表、展开详情、上下文和导出文件中完全移除。</div></el-form-item>
        <el-form-item label="掩码字段" class="full"><el-select v-model="form.maskFields" multiple filterable allow-create default-first-option placeholder="输入字段名后回车"><el-option v-for="field in fieldSuggestions" :key="field" :label="field" :value="field" /></el-select><div class="form-tip">字段保留但统一显示为 ******；隐藏字段优先级更高。</div></el-form-item>
        <el-form-item label="策略说明" class="full"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="启用策略"><el-switch v-model="form.enabled" /></el-form-item>
      </div></el-form>
      <template #footer><el-button @click="visible = false">取消</el-button><el-button type="primary" class="primary-action" :loading="saving" @click="save">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { createLogAccessPolicy, deleteLogAccessPolicy, getLogAccessPolicies, getLogAccessPolicyOptions, updateLogAccessPolicy, type LogAccessPolicy, type LogStorageCluster } from '@/api/logcenter'
const props = defineProps<{ storages: LogStorageCluster[] }>()
const userStore = useUserStore()
const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))
const loading = ref(false), saving = ref(false), visible = ref(false)
const editingId = ref<number>()
const items = ref<LogAccessPolicy[]>([])
const options = reactive<{ users: Array<{ id: number; username: string; realName?: string }>; roles: Array<{ id: number; name: string; code: string }> }>({ users: [], roles: [] })
const defaultForm = () => ({ name: '', description: '', subjectType: 'role' as 'user' | 'role', subjectId: undefined as number | undefined, storageId: undefined as number | undefined, libraryItemPattern: '', allowedActions: ['query', 'tail', 'export'], deniedFields: [] as string[], maskFields: [] as string[], enabled: true })
const form = reactive(defaultForm())
const fieldSuggestions = ['body', 'authorization', 'cookie', 'token', 'password', 'secret', 'attributes', 'resourceAttributes', 'labels.namespace', 'labels.service', 'filePath', 'containerImage']
const subjectOptions = computed(() => form.subjectType === 'user' ? options.users.map(item => ({ id: item.id, label: `${item.realName || item.username} (${item.username})` })) : options.roles.map(item => ({ id: item.id, label: `${item.name} (${item.code})` })))
watch(() => form.subjectType, () => { form.subjectId = undefined })
const load = async () => { if (!isAdmin.value) return; loading.value = true; try { const [policies, metadata] = await Promise.all([getLogAccessPolicies(), getLogAccessPolicyOptions()]); items.value = policies as any || []; Object.assign(options, metadata || {}) } finally { loading.value = false } }
const openCreate = () => { editingId.value = undefined; Object.assign(form, defaultForm()); visible.value = true }
const openEdit = (row: LogAccessPolicy) => { editingId.value = row.id; Object.assign(form, defaultForm(), JSON.parse(JSON.stringify(row.payload))); visible.value = true }
const save = async () => { if (!form.name.trim() || !form.subjectId) return ElMessage.warning('请填写策略名称并选择授权对象'); if (!form.allowedActions.length) return ElMessage.warning('至少允许一种日志操作'); saving.value = true; try { const payload = { ...JSON.parse(JSON.stringify(form)), subjectId: form.subjectId }; editingId.value ? await updateLogAccessPolicy(editingId.value, payload) : await createLogAccessPolicy(payload); ElMessage.success('访问策略已保存'); visible.value = false; await load() } finally { saving.value = false } }
const remove = async (row: LogAccessPolicy) => { await deleteLogAccessPolicy(row.id); ElMessage.success('访问策略已删除'); await load() }
const subjectName = (row: LogAccessPolicy) => row.payload.subjectType === 'user' ? options.users.find(item => item.id === row.payload.subjectId)?.realName || options.users.find(item => item.id === row.payload.subjectId)?.username || `用户 #${row.payload.subjectId}` : options.roles.find(item => item.id === row.payload.subjectId)?.name || `角色 #${row.payload.subjectId}`
const actionText = (action: string) => ({ query: '查询', tail: 'Tail', export: '导出' }[action] || action)
onMounted(load)
</script>

<style scoped>
.access-panel{padding:20px}.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:18px}.panel-head h3{margin:0;color:#111827;font-size:15px}.panel-head p{margin:6px 0 0;color:#667085;font-size:13px}.name-cell strong,.name-cell small{display:block}.name-cell small{margin-top:4px;color:#98a2b3;font-size:12px}.tag-list{display:flex;flex-wrap:wrap;gap:5px}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:0 18px}.form-grid :deep(.el-select){width:100%}.full{grid-column:1/-1}.form-tip{margin-top:6px;color:#98a2b3;font-size:12px}@media(max-width:700px){.form-grid{grid-template-columns:1fr}.full{grid-column:auto}}
</style>
