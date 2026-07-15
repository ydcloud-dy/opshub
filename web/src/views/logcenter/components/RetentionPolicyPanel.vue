<template>
  <section class="panel retention-panel">
    <div class="panel-head">
      <div>
        <h3>差异化保留策略</h3>
        <p>按日志级别配置保留周期，采集策略发布时会将配置快照下发到 Agent</p>
      </div>
      <el-button type="primary" class="primary-action" @click="openCreate"><el-icon><Plus /></el-icon>新增策略</el-button>
    </div>
    <el-table v-loading="loading" :data="items" empty-text="暂无保留策略">
      <el-table-column label="策略名称" min-width="210">
        <template #default="{ row }"><div class="name-cell"><strong>{{ row.payload.name }}</strong><small>{{ row.payload.description || '未填写说明' }}</small></div></template>
      </el-table-column>
      <el-table-column label="适用存储" min-width="150"><template #default="{ row }">{{ storageName(row.payload.storageId) }}</template></el-table-column>
      <el-table-column label="默认保留" width="110"><template #default="{ row }">{{ row.payload.defaultDays }} 天</template></el-table-column>
      <el-table-column label="级别覆盖" min-width="270">
        <template #default="{ row }"><div class="level-tags"><el-tag v-for="(days, level) in row.payload.levelDays" :key="level" size="small" :type="levelType(String(level))">{{ level }} {{ days }} 天</el-tag><span v-if="!Object.keys(row.payload.levelDays || {}).length">跟随默认</span></div></template>
      </el-table-column>
      <el-table-column label="状态" width="95"><template #default="{ row }"><el-tag size="small" :type="row.payload.enabled ? 'success' : 'info'">{{ row.payload.enabled ? '启用' : '停用' }}</el-tag></template></el-table-column>
      <el-table-column label="操作" width="140" fixed="right"><template #default="{ row }"><el-button link @click="openEdit(row)">编辑</el-button><el-popconfirm title="确定删除此保留策略？" @confirm="remove(row)"><template #reference><el-button link type="danger">删除</el-button></template></el-popconfirm></template></el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="editingId ? '编辑保留策略' : '新增保留策略'" width="min(720px, calc(100vw - 32px))" destroy-on-close>
      <el-form :model="form" label-position="top">
        <div class="form-grid">
          <el-form-item label="策略名称" required><el-input v-model="form.name" placeholder="例如：生产关键日志" /></el-form-item>
          <el-form-item label="适用存储"><el-select v-model="form.storageId" clearable placeholder="全部存储"><el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item>
          <el-form-item label="默认保留天数"><el-input-number v-model="form.defaultDays" :min="1" :max="3650" controls-position="right" /></el-form-item>
          <el-form-item label="优先级"><el-input-number v-model="form.priority" :min="1" :max="9999" controls-position="right" /></el-form-item>
          <el-form-item v-for="level in levels" :key="level" :label="`${level} 保留天数`"><el-input-number v-model="form.levelDays[level]" :min="1" :max="3650" controls-position="right" /></el-form-item>
          <el-form-item label="启用策略"><el-switch v-model="form.enabled" /></el-form-item>
          <el-form-item label="策略说明" class="full"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
        </div>
      </el-form>
      <template #footer><el-button @click="visible = false">取消</el-button><el-button type="primary" class="primary-action" :loading="saving" @click="save">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { createLogRetentionPolicy, deleteLogRetentionPolicy, getLogRetentionPolicies, updateLogRetentionPolicy, type LogRetentionPolicy, type LogStorageCluster } from '@/api/logcenter'

const props = defineProps<{ storages: LogStorageCluster[] }>()
const levels = ['TRACE', 'DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref<number>()
const items = ref<LogRetentionPolicy[]>([])
const defaultForm = () => ({ name: '', description: '', storageId: undefined as number | undefined, defaultDays: 30, levelDays: { TRACE: 7, DEBUG: 7, INFO: 30, WARN: 60, ERROR: 90, FATAL: 180 } as Record<string, number>, priority: 100, enabled: true })
const form = reactive(defaultForm())

const load = async () => { loading.value = true; try { items.value = await getLogRetentionPolicies() as any || [] } finally { loading.value = false } }
const openCreate = () => { editingId.value = undefined; Object.assign(form, defaultForm()); visible.value = true }
const openEdit = (row: LogRetentionPolicy) => { editingId.value = row.id; Object.assign(form, defaultForm(), JSON.parse(JSON.stringify(row.payload))); visible.value = true }
const save = async () => {
  if (!form.name.trim()) return ElMessage.warning('请输入策略名称')
  saving.value = true
  try {
    const payload = JSON.parse(JSON.stringify(form))
    editingId.value ? await updateLogRetentionPolicy(editingId.value, payload) : await createLogRetentionPolicy(payload)
    ElMessage.success('保留策略已保存'); visible.value = false; await load()
  } finally { saving.value = false }
}
const remove = async (row: LogRetentionPolicy) => { await deleteLogRetentionPolicy(row.id); ElMessage.success('保留策略已删除'); await load() }
const storageName = (id?: number) => !id ? '全部存储' : props.storages.find(item => item.id === id)?.name || `存储 #${id}`
const levelType = (level: string) => ['ERROR', 'FATAL'].includes(level) ? 'danger' : level === 'WARN' ? 'warning' : level === 'INFO' ? 'success' : 'info'
onMounted(load)
</script>

<style scoped>
.retention-panel { padding: 20px; }.panel-head { display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:18px; }.panel-head h3 { margin:0;color:#111827;font-size:15px; }.panel-head p { margin:6px 0 0;color:#667085;font-size:13px; }.name-cell strong,.name-cell small { display:block; }.name-cell small { margin-top:4px;color:#98a2b3;font-size:12px; }.level-tags { display:flex;flex-wrap:wrap;gap:5px; }.level-tags > span { color:#98a2b3;font-size:12px; }.form-grid { display:grid;grid-template-columns:1fr 1fr;gap:0 18px; }.form-grid :deep(.el-select),.form-grid :deep(.el-input-number) { width:100%; }.full { grid-column:1/-1; }@media(max-width:700px){.form-grid{grid-template-columns:1fr}.full{grid-column:auto}}
</style>
