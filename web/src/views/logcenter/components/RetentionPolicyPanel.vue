<template>
  <section class="panel retention-panel">
    <div class="panel-head">
      <div>
        <h3>差异化保留策略</h3>
        <p>采集策略显式绑定保留策略，发布时由后端生成配置快照并下发到 Agent</p>
      </div>
      <el-button type="primary" class="primary-action" @click="openCreate"><el-icon><Plus /></el-icon>新增策略</el-button>
    </div>
    <el-table v-loading="loading" :data="items" empty-text="暂无保留策略">
      <el-table-column label="策略名称" min-width="210">
        <template #default="{ row }"><div class="name-cell"><strong>{{ row.payload.name }}</strong><small>{{ row.payload.description || '未填写说明' }}</small></div></template>
      </el-table-column>
      <el-table-column label="默认保留" width="110"><template #default="{ row }">{{ row.payload.defaultDays }} 天</template></el-table-column>
      <el-table-column label="级别覆盖" min-width="270">
        <template #default="{ row }"><div class="level-tags"><el-tag v-for="(days, level) in row.payload.levelDays" :key="level" size="small" :type="levelType(String(level))">{{ level }} {{ days }} 天</el-tag><span v-if="!Object.keys(row.payload.levelDays || {}).length">跟随默认</span></div></template>
      </el-table-column>
      <el-table-column label="绑定采集策略" width="125"><template #default="{ row }"><el-tag size="small" :type="row.boundPolicyCount ? 'primary' : 'info'">{{ row.boundPolicyCount || 0 }} 个</el-tag></template></el-table-column>
      <el-table-column label="状态" width="95"><template #default="{ row }"><el-tag size="small" :type="row.payload.enabled ? 'success' : 'info'">{{ row.payload.enabled ? '启用' : '停用' }}</el-tag></template></el-table-column>
      <el-table-column label="操作" width="140" fixed="right"><template #default="{ row }"><el-button link @click="openEdit(row)">编辑</el-button><el-tooltip :disabled="!row.boundPolicyCount" :content="`仍被 ${row.boundPolicyCount} 个采集策略引用`" placement="top"><span><el-popconfirm title="确定删除此保留策略？" :disabled="Boolean(row.boundPolicyCount)" @confirm="remove(row)"><template #reference><el-button link type="danger" :disabled="Boolean(row.boundPolicyCount)">删除</el-button></template></el-popconfirm></span></el-tooltip></template></el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="editingId ? '编辑保留策略' : '新增保留策略'" width="min(860px, calc(100vw - 32px))" class="retention-dialog" destroy-on-close>
      <el-form :model="form" label-position="top" class="retention-form">
        <section class="form-section">
          <div class="form-section-head"><strong>基本信息</strong><span>策略需要在采集策略中显式绑定后才会生效</span></div>
          <div class="basic-grid">
            <el-form-item label="策略名称" required><el-input v-model="form.name" maxlength="120" placeholder="例如：生产关键日志" /></el-form-item>
            <el-form-item label="策略状态"><div class="status-control"><el-switch v-model="form.enabled" /><span>{{ form.enabled ? '可被采集策略绑定并发布' : '禁止新发布，已发布快照不受影响' }}</span></div></el-form-item>
          </div>
        </section>

        <section class="form-section retention-settings">
          <div class="form-section-head"><strong>默认规则</strong><span>未匹配日志级别时使用默认保留周期</span></div>
          <div class="settings-grid single-setting">
            <el-form-item label="默认保留周期"><div class="days-input"><el-input-number v-model="form.defaultDays" :min="1" :max="3650" controls-position="right" /><span>天</span></div></el-form-item>
          </div>
          <el-alert v-if="editingId && currentBoundCount" class="binding-alert" type="warning" :closable="false" show-icon :title="`保存后会为 ${currentBoundCount} 个绑定采集策略生成待发布变更，发布后仅影响新采集日志`" />
        </section>

        <section class="form-section level-section">
          <div class="form-section-head"><strong>按日志级别保留</strong><span>不同级别可设置独立周期，减少低价值日志占用并保留关键事件</span></div>
          <div class="level-grid">
            <label v-for="level in levels" :key="level.value" class="level-card">
              <span class="level-badge" :class="`level-${level.value.toLowerCase()}`">{{ level.value }}</span>
              <span class="level-description">{{ level.label }}</span>
              <div class="days-input"><el-input-number v-model="form.levelDays[level.value]" :min="1" :max="3650" controls-position="right" /><span>天</span></div>
            </label>
          </div>
        </section>

        <section class="form-section description-section">
          <el-form-item label="策略说明"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="说明该策略适用的业务、环境或合规要求" /></el-form-item>
        </section>
      </el-form>
      <template #footer><el-button @click="visible = false">取消</el-button><el-button type="primary" class="primary-action" :loading="saving" @click="save">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { createLogRetentionPolicy, deleteLogRetentionPolicy, getLogRetentionPolicies, updateLogRetentionPolicy, type LogRetentionPolicy } from '@/api/logcenter'

const levels = [
  { value: 'TRACE', label: '链路跟踪' },
  { value: 'DEBUG', label: '调试信息' },
  { value: 'INFO', label: '运行信息' },
  { value: 'WARN', label: '风险警告' },
  { value: 'ERROR', label: '错误日志' },
  { value: 'FATAL', label: '致命错误' },
]
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref<number>()
const items = ref<LogRetentionPolicy[]>([])
const defaultForm = () => ({ name: '', description: '', storageId: undefined as number | undefined, defaultDays: 30, levelDays: { TRACE: 7, DEBUG: 7, INFO: 30, WARN: 60, ERROR: 90, FATAL: 180 } as Record<string, number>, priority: 100, enabled: true })
const form = reactive(defaultForm())
const currentBoundCount = ref(0)

const load = async () => { loading.value = true; try { items.value = await getLogRetentionPolicies() as any || [] } finally { loading.value = false } }
const openCreate = () => { editingId.value = undefined; currentBoundCount.value = 0; Object.assign(form, defaultForm()); visible.value = true }
const openEdit = (row: LogRetentionPolicy) => { editingId.value = row.id; currentBoundCount.value = Number(row.boundPolicyCount || 0); Object.assign(form, defaultForm(), JSON.parse(JSON.stringify(row.payload))); visible.value = true }
const save = async () => {
  if (!form.name.trim()) return ElMessage.warning('请输入策略名称')
  saving.value = true
  try {
    const payload = JSON.parse(JSON.stringify(form))
    const result = editingId.value ? await updateLogRetentionPolicy(editingId.value, payload) : await createLogRetentionPolicy(payload)
    const updated = Number((result as LogRetentionPolicy)?.updatedPolicyCount || 0)
    ElMessage.success(updated ? `保留策略已保存，${updated} 个采集策略已生成待发布变更` : '保留策略已保存')
    visible.value = false; await load()
  } finally { saving.value = false }
}
const remove = async (row: LogRetentionPolicy) => { await deleteLogRetentionPolicy(row.id); ElMessage.success('保留策略已删除'); await load() }
const levelType = (level: string) => ['ERROR', 'FATAL'].includes(level) ? 'danger' : level === 'WARN' ? 'warning' : level === 'INFO' ? 'success' : 'info'
onMounted(load)
</script>

<style scoped>
.retention-panel { padding: 20px; }.panel-head { display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:18px; }.panel-head h3 { margin:0;color:#111827;font-size:15px; }.panel-head p { margin:6px 0 0;color:#667085;font-size:13px; }.name-cell strong,.name-cell small { display:block; }.name-cell small { margin-top:4px;color:#98a2b3;font-size:12px; }.level-tags { display:flex;flex-wrap:wrap;gap:5px; }.level-tags > span { color:#98a2b3;font-size:12px; }.retention-form{display:flex;flex-direction:column;gap:16px}.form-section{padding:16px 18px;border:1px solid #e7ebf1;border-radius:7px;background:#fff}.form-section-head{display:flex;align-items:baseline;gap:10px;margin-bottom:14px}.form-section-head strong{color:#111827;font-size:14px}.form-section-head span{color:#98a2b3;font-size:12px}.basic-grid,.settings-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 18px}.settings-grid.single-setting{grid-template-columns:minmax(220px,320px)}.retention-form :deep(.el-form-item){margin-bottom:0}.retention-form :deep(.el-select),.retention-form :deep(.el-input-number){width:100%}.days-input{display:flex;align-items:center;gap:9px;width:100%}.days-input span{flex:0 0 auto;color:#667085;font-size:12px}.status-control{display:flex;align-items:center;gap:10px;min-height:32px;color:#667085;font-size:12px}.binding-alert{margin-top:14px}.level-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}.level-card{display:grid;grid-template-columns:auto minmax(0,1fr);align-items:center;gap:7px 9px;min-width:0;padding:12px;border:1px solid #edf0f4;border-radius:7px;background:#fafbfc}.level-card .days-input{grid-column:1/-1}.level-badge{display:inline-flex;align-items:center;justify-content:center;min-width:52px;padding:3px 7px;border-radius:4px;background:#f2f4f7;color:#475467;font-size:10px;font-weight:750}.level-description{overflow:hidden;color:#667085;font-size:12px;text-overflow:ellipsis;white-space:nowrap}.level-trace,.level-debug{background:#f5f3ff;color:#7c3aed}.level-info{background:#eff6ff;color:#2563eb}.level-warn{background:#fffbeb;color:#d97706}.level-error,.level-fatal{background:#fef2f2;color:#dc2626}.description-section :deep(.el-form-item){width:100%}@media(max-width:760px){.form-section-head{align-items:flex-start;flex-direction:column;gap:4px}.basic-grid,.settings-grid,.level-grid{grid-template-columns:1fr}.settings-grid{gap:14px}.level-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:520px){.level-grid{grid-template-columns:1fr}}
</style>
