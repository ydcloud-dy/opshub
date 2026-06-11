<template>
  <div class="notice-template-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DocumentCopy /></el-icon>
        </div>
        <div>
          <h2>通知模板</h2>
          <p>按通知类型维护触发模板和恢复模板</p>
        </div>
      </div>
      <div class="head-actions">
        <el-input v-model="keyword" placeholder="搜索模板名称或描述" clearable class="head-search" @keyup.enter="loadTemplates">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="typeFilter" clearable placeholder="通知类型" class="type-filter">
          <el-option v-for="item in noticeTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-button @click="loadTemplates">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          新建模板
        </el-button>
      </div>
    </div>

    <div class="template-layout">
      <aside class="type-rail">
        <button
          v-for="item in typeStats"
          :key="item.value"
          type="button"
          class="type-item"
          :class="{ active: typeFilter === item.value }"
          @click="toggleTypeFilter(item.value)"
        >
          <PlatformLogo :type="item.value" :logo="item.logo" />
          <span>
            <strong>{{ item.label }}</strong>
            <em>{{ item.count }} 个模板</em>
          </span>
        </button>
      </aside>

      <main class="table-panel">
        <el-table :data="filteredTemplates" v-loading="loading" class="dense-table" :header-cell-style="tableHeaderStyle">
          <el-table-column label="模板名称" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="name-cell">
                <strong>{{ row.name }}</strong>
                <span>ID {{ row.id }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="132">
            <template #default="{ row }">
              <span class="type-chip">
                <PlatformLogo :type="row.noticeType" :logo="getNoticeTypeLogo(row.noticeType)" mini />
                <span>{{ getNoticeTypeName(row.noticeType) }}</span>
              </span>
            </template>
          </el-table-column>
          <el-table-column label="描述" prop="description" min-width="180" show-overflow-tooltip />
          <el-table-column label="触发模板" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">{{ row.templateFiring || '-' }}</template>
          </el-table-column>
          <el-table-column label="恢复模板" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">{{ row.templateRecover || '-' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="92" align="center">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="handleEnabledChange(row)" />
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="118" fixed="right" align="center">
            <template #default="{ row }">
              <div class="table-actions">
                <el-tooltip content="复制" placement="top">
                  <el-button link class="icon-btn" @click="handleCopy(row)">
                    <el-icon><CopyDocument /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="编辑" placement="top">
                  <el-button link class="icon-btn" @click="handleEdit(row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button link class="icon-btn danger" @click="handleDelete(row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </main>
    </div>

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="960px" class="monitor-drawer" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="104px">
        <div class="drawer-section">
          <div class="form-grid">
            <el-form-item label="模板名称" prop="name">
              <el-input v-model="form.name" placeholder="例如：飞书生产告警模板" />
            </el-form-item>
            <el-form-item label="通知类型" prop="noticeType">
              <el-select v-model="form.noticeType" style="width: 100%" :disabled="Boolean(form.id)" @change="handleNoticeTypeChange">
                <el-option v-for="item in noticeTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </div>
          <el-form-item label="描述">
            <el-input v-model="form.description" placeholder="模板用途、适用团队或渠道" />
          </el-form-item>
          <div class="form-grid">
            <el-form-item label="启用">
              <el-switch v-model="form.enabled" />
            </el-form-item>
            <el-form-item v-if="form.noticeType === 'FeiShu'" label="飞书卡片">
              <el-switch v-model="form.enableFeiShuJsonCard" />
            </el-form-item>
          </div>
        </div>

        <div class="drawer-section">
          <div class="section-title">
            <span>模板内容</span>
            <em>支持 ${rule_name}、${severity}、${labels.instance}、${matched_logs}、${event_url} 等变量</em>
          </div>
          <div class="template-editor-grid">
            <el-form-item label="触发模板" prop="templateFiring" class="template-editor-card">
              <el-input v-model="form.templateFiring" type="textarea" :rows="18" placeholder="告警触发时使用" resize="none" />
            </el-form-item>
            <el-form-item label="恢复模板" prop="templateRecover" class="template-editor-card">
              <el-input v-model="form.templateRecover" type="textarea" :rows="18" placeholder="告警恢复时使用" resize="none" />
            </el-form-item>
          </div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { CopyDocument, Delete, DocumentCopy, Edit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import {
  createMonitorNoticeTemplate,
  deleteMonitorNoticeTemplate,
  getMonitorNoticeTemplates,
  updateMonitorNoticeTemplate,
  type MonitorNoticeTemplate
} from '@/api/monitor-datasource'

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
const loading = ref(false)
const submitting = ref(false)
const drawerVisible = ref(false)
const drawerTitle = ref('新建通知模板')
const keyword = ref('')
const typeFilter = ref('')
const templates = ref<MonitorNoticeTemplate[]>([])
const formRef = ref<FormInstance>()

const noticeTypeOptions = [
  { label: '飞书', value: 'FeiShu', logo: 'feishu' },
  { label: '邮件', value: 'Email', logo: 'email' },
  { label: '钉钉', value: 'DingDing', logo: 'dingtalk' },
  { label: '企业微信', value: 'WeChat', logo: 'wecom' },
  { label: 'Webhook', value: 'WebHook', logo: 'webhook' }
]

const PlatformLogo = defineComponent({
  name: 'PlatformLogo',
  props: {
    type: { type: String, required: true },
    logo: { type: String, default: '' },
    mini: { type: Boolean, default: false }
  },
  setup(props) {
    const svgAttrs = { viewBox: '0 0 48 48', 'aria-hidden': 'true' }
    return () => {
      const type = props.type
      const logo = props.logo || getNoticeTypeLogo(type)
      const children = type === 'FeiShu'
        ? [
            h('path', { fill: '#2262ff', d: 'M10 29.4 22.6 36.6c2.9 1.7 6.6 1.2 9-1.2l8.9-8.9c1.1-1.1.1-3-1.4-2.7l-12.8 2.4c-1.5.3-3.1-.1-4.3-1.1L11.7 16.4c-1.2-1-3 .1-2.7 1.7l1 5.5c.4 2.4 1.8 4.5 4 5.8Z' }),
            h('path', { fill: '#0cc7a3', d: 'M17.7 11.1 29.5 25l9.4-1.8c1.8-.3 2.6-2.5 1.3-3.8L29.9 9.4c-2.8-2.7-7-3.2-10.3-1.2l-1.4.8c-.8.5-1 1.4-.5 2.1Z' }),
            h('path', { fill: '#7c5cff', d: 'M10 17.2 22 26.8c1.2 1 2.8 1.4 4.3 1.1l3.2-.6-12-14.1c-.6-.7-1.6-.9-2.4-.4L10.7 15c-.9.5-1 1.6-.7 2.2Z' })
          ]
        : type === 'Email'
          ? [
              h('rect', { x: '7', y: '11', width: '34', height: '26', rx: '7', fill: '#2f80ed' }),
              h('path', { fill: '#fff', d: 'M11.6 17.7c.5-.7 1.5-.9 2.2-.4L24 24.5l10.2-7.2c.7-.5 1.7-.3 2.2.4.5.7.3 1.7-.4 2.2l-11.1 7.8c-.5.4-1.2.4-1.8 0L12 19.9c-.7-.5-.9-1.5-.4-2.2Z' })
            ]
          : type === 'DingDing'
            ? [
                h('path', { fill: '#38a3ff', d: 'M35.8 7.8c1.3.6 1.7 2.3.8 3.4l-16.8 22c-.9 1.1-2.7.8-3.1-.6l-2-6.8-6.4-2.5c-1.3-.5-1.4-2.3-.2-3L35.8 7.8Z' }),
                h('path', { fill: '#64c4ff', d: 'm19.9 33.3 3.6 7.4c.6 1.2 2.3 1.2 2.8.1l10.1-22.1-16.5 14.6Z' })
              ]
            : type === 'WeChat'
              ? [
                  h('path', { fill: '#12b76a', d: 'M22.3 13.2c-8 0-14.4 5.1-14.4 11.4 0 3.4 1.9 6.5 4.8 8.5l-1 4.2 5-2.3c1.7.6 3.6 1 5.6 1 8 0 14.4-5.1 14.4-11.4s-6.4-11.4-14.4-11.4Z' }),
                  h('path', { fill: '#2f80ed', d: 'M31 20.9c5.8 0 10.5 3.8 10.5 8.4 0 2.5-1.4 4.8-3.6 6.3l.7 3.1-3.7-1.7c-1.2.5-2.6.7-4 .7-5.8 0-10.5-3.8-10.5-8.4s4.8-8.4 10.6-8.4Z' }),
                  h('circle', { cx: '17.4', cy: '23.3', r: '1.8', fill: '#fff' }),
                  h('circle', { cx: '26', cy: '23.3', r: '1.8', fill: '#fff' })
                ]
              : [
                  h('path', { fill: '#4bb8f3', d: 'M18.6 9.5h10.8l1.1 5.2c1 .4 1.9.9 2.8 1.6l5.1-1.7 5.4 9.4-4 3.5c.1.6.2 1.2.2 1.8s-.1 1.2-.2 1.8l4 3.5-5.4 9.4-5.1-1.7c-.9.7-1.8 1.2-2.8 1.6l-1.1 5.2H18.6l-1.1-5.2c-1-.4-1.9-.9-2.8-1.6l-5.1 1.7-5.4-9.4 4-3.5c-.1-.6-.2-1.2-.2-1.8s.1-1.2.2-1.8l-4-3.5 5.4-9.4 5.1 1.7c.9-.7 1.8-1.2 2.8-1.6l1.1-5.2Z' }),
                  h('circle', { cx: '24', cy: '29.3', r: '7.1', fill: '#fff' }),
                  h('circle', { cx: '24', cy: '29.3', r: '3.3', fill: '#4bb8f3' })
                ]
      return h('span', { class: ['platform-logo', `logo-${logo}`, { mini: props.mini }] }, [
        h('svg', svgAttrs, children)
      ])
    }
  }
})

type NoticeTemplateDefault = { firing: string; recover: string; enableFeiShuJsonCard?: boolean }

const defaultTemplateMap = {
  FeiShu: {
    enableFeiShuJsonCard: true,
    firing: `{
  "schema": "2.0",
  "config": {
    "width_mode": "fill",
    "enable_forward": true
  },
  "header": {
    "template": "red",
    "title": {
      "tag": "plain_text",
      "content": "【告警中】- OpsHub 业务系统 🔥"
    }
  },
  "body": {
    "elements": [
	      { "tag": "markdown", "content": "**🤖 告警类型:** \${rule_name}\\n**🫧 告警指纹:** \${fingerprint}\\n**📌 告警等级:** \${severity}\\n**🖥 告警主机:** \${labels.instance}\\n**🕘 开始时间:** {{ .FirstTriggerTime | formatTime }}\\n**👤 值班人员:** \${duty_user_feishu_at}\\n**📝 告警事件:** \${annotations}\\n\${matched_logs_block}\\n[查看事件](\${event_url})" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队" }
    ]
  }
}`,
    recover: `{
  "schema": "2.0",
  "config": {
    "width_mode": "fill",
    "enable_forward": true
  },
  "header": {
    "template": "green",
    "title": {
      "tag": "plain_text",
      "content": "【已恢复】- OpsHub 业务系统 ✨"
    }
  },
  "body": {
    "elements": [
	      { "tag": "markdown", "content": "**🤖 告警类型:** \${rule_name}\\n**🫧 告警指纹:** \${fingerprint}\\n**📌 告警等级:** \${severity}\\n**🖥 告警主机:** \${labels.instance}\\n**🕘 开始时间:** {{ .FirstTriggerTime | formatTime }}\\n**🕘 恢复时间:** {{ .RecoverTime | formatTime }}\\n**👤 值班人员:** \${duty_user_feishu_at}\\n**📝 告警事件:** \${annotations}\\n\${matched_logs_block}\\n[查看事件](\${event_url})" },
      { "tag": "hr" },
      { "tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队" }
    ]
  }
}`
  },
  DingDing: {
    firing: `### 🔥 OpsHub 告警中

> **\${rule_name}**
> 等级：**\${severity}**  ｜ 当前值：**\${value}**
> 实例：\${labels.instance}

**基础信息**

- 开始时间：{{ .FirstTriggerTime | formatTime }}
- 值班人员：\${duty_user}
- 通知对象：\${duty_user_dingtalk_at}
- 事件指纹：\${fingerprint}

**告警详情**

> \${annotations}

\${matched_logs_block}

[🔎 查看事件](\${event_url})

---
OpsHub 监控中心`,
    recover: `### ✅ OpsHub 已恢复

> **\${rule_name}**
> 等级：**\${severity}**
> 实例：\${labels.instance}

**恢复信息**

- 开始时间：{{ .FirstTriggerTime | formatTime }}
- 恢复时间：{{ .RecoverTime | formatTime }}
- 值班人员：\${duty_user}
- 通知对象：\${duty_user_dingtalk_at}
- 事件指纹：\${fingerprint}

**恢复详情**

> \${annotations}

\${matched_logs_block}

[🔎 查看事件](\${event_url})

---
OpsHub 监控中心`
  },
  WeChat: {
    firing: `## 🔥 OpsHub 告警中
> 规则：\${rule_name}
> 指纹：\${fingerprint}
> 等级：<font color="warning">\${severity}</font>
> 实例：\${labels.instance}
> 当前值：{{value}}
> 开始时间：{{ .FirstTriggerTime | formatTime }}
> 值班人员：\${duty_user}
> 事件说明：\${annotations}

\${matched_logs_block}

[查看事件](\${event_url})`,
    recover: `## ✅ OpsHub 已恢复
> 规则：\${rule_name}
> 指纹：\${fingerprint}
> 等级：<font color="info">\${severity}</font>
> 实例：\${labels.instance}
> 开始时间：{{ .FirstTriggerTime | formatTime }}
> 恢复时间：{{ .RecoverTime | formatTime }}
> 值班人员：\${duty_user}
> 事件说明：\${annotations}

\${matched_logs_block}

[查看事件](\${event_url})`
  },
  Email: {
    firing: `<div style="font-family:Arial,'PingFang SC',sans-serif;color:#111827;line-height:1.7">
  <h2 style="margin:0 0 12px;color:#dc2626">OpsHub 告警中</h2>
  <p><b>规则：</b>\${rule_name}</p>
  <p><b>指纹：</b>\${fingerprint}</p>
  <p><b>等级：</b>\${severity}</p>
  <p><b>实例：</b>\${labels.instance}</p>
  <p><b>当前值：</b>{{value}}</p>
  <p><b>开始时间：</b>{{ .FirstTriggerTime | formatTime }}</p>
  <p><b>值班人员：</b>\${duty_user}</p>
  <p><b>事件说明：</b>\${annotations}</p>
  <p><b>命中日志：</b></p>
  <pre style="padding:12px;background:#f8fafc;border:1px solid #e5e7eb;border-radius:6px;white-space:pre-wrap">\${matched_logs}</pre>
  <p><a href="\${event_url}" style="color:#2563eb">查看事件</a></p>
  <hr/>
  <p style="color:#667085">OpsHub 运维平台</p>
</div>`,
    recover: `<div style="font-family:Arial,'PingFang SC',sans-serif;color:#111827;line-height:1.7">
  <h2 style="margin:0 0 12px;color:#16a34a">OpsHub 已恢复</h2>
  <p><b>规则：</b>\${rule_name}</p>
  <p><b>指纹：</b>\${fingerprint}</p>
  <p><b>等级：</b>\${severity}</p>
  <p><b>实例：</b>\${labels.instance}</p>
  <p><b>开始时间：</b>{{ .FirstTriggerTime | formatTime }}</p>
  <p><b>恢复时间：</b>{{ .RecoverTime | formatTime }}</p>
  <p><b>值班人员：</b>\${duty_user}</p>
  <p><b>事件说明：</b>\${annotations}</p>
  <p><b>命中日志：</b></p>
  <pre style="padding:12px;background:#f8fafc;border:1px solid #e5e7eb;border-radius:6px;white-space:pre-wrap">\${matched_logs}</pre>
  <p><a href="\${event_url}" style="color:#2563eb">查看事件</a></p>
  <hr/>
  <p style="color:#667085">OpsHub 运维平台</p>
</div>`
  },
  WebHook: {
    firing: `{"status":"firing","platform":"OpsHub","ruleName":"\${rule_name}","fingerprint":"\${fingerprint}","severity":"\${severity}","instance":"\${labels.instance}","value":"{{value}}","startedAt":"{{ .FirstTriggerTime | formatTime }}","dutyUser":"\${duty_user}","annotations":"\${annotations}","matchedLogs":"\${matched_logs}","matchedLogCount":"\${matched_log_count}","eventUrl":"\${event_url}"}`,
    recover: `{"status":"recovered","platform":"OpsHub","ruleName":"\${rule_name}","fingerprint":"\${fingerprint}","severity":"\${severity}","instance":"\${labels.instance}","startedAt":"{{ .FirstTriggerTime | formatTime }}","recoveredAt":"{{ .RecoverTime | formatTime }}","dutyUser":"\${duty_user}","annotations":"\${annotations}","matchedLogs":"\${matched_logs}","matchedLogCount":"\${matched_log_count}","eventUrl":"\${event_url}"}`
  }
} satisfies Record<string, NoticeTemplateDefault>

type NoticeTemplateType = keyof typeof defaultTemplateMap

const getTemplateDefaults = (type?: string): NoticeTemplateDefault => {
  if (type && Object.prototype.hasOwnProperty.call(defaultTemplateMap, type)) {
    return defaultTemplateMap[type as NoticeTemplateType]
  }
  return defaultTemplateMap.FeiShu
}

const form = reactive<MonitorNoticeTemplate>({
  name: '',
  noticeType: 'FeiShu',
  description: '',
  template: '',
  templateFiring: '',
  templateRecover: '',
  enableFeiShuJsonCard: false,
  enabled: true
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  noticeType: [{ required: true, message: '请选择通知类型', trigger: 'change' }],
  templateFiring: [{ required: true, message: '请输入触发模板', trigger: 'blur' }],
  templateRecover: [{ required: true, message: '请输入恢复模板', trigger: 'blur' }]
}

const typeStats = computed(() => noticeTypeOptions.map(item => ({
  ...item,
  count: templates.value.filter(template => template.noticeType === item.value).length
})))

const filteredTemplates = computed(() => {
  const keywordText = keyword.value.trim().toLowerCase()
  return templates.value.filter(template => {
    const matchesAllowedType = isAllowedNoticeType(template.noticeType)
    const matchesType = !typeFilter.value || template.noticeType === typeFilter.value
    const matchesKeyword = !keywordText ||
      template.name.toLowerCase().includes(keywordText) ||
      (template.description || '').toLowerCase().includes(keywordText)
    return matchesAllowedType && matchesType && matchesKeyword
  })
})

const isAllowedNoticeType = (type?: string) => noticeTypeOptions.some(item => item.value === type)

const loadTemplates = async () => {
  loading.value = true
  try {
    templates.value = await getMonitorNoticeTemplates() || []
  } finally {
    loading.value = false
  }
}

const toggleTypeFilter = (type: string) => {
  typeFilter.value = typeFilter.value === type ? '' : type
}

const handleAdd = () => {
  resetForm()
  fillTemplateDefaults(form.noticeType)
  drawerTitle.value = '新建通知模板'
  drawerVisible.value = true
}

const handleEdit = (row: MonitorNoticeTemplate) => {
  resetForm()
  const defaults = getTemplateDefaults(row.noticeType)
  Object.assign(form, {
    ...row,
    template: '',
    templateFiring: row.templateFiring || row.template || defaults.firing,
    templateRecover: row.templateRecover || row.template || defaults.recover
  })
  form.enabled = row.enabled ?? true
  form.enableFeiShuJsonCard = row.enableFeiShuJsonCard ?? defaults.enableFeiShuJsonCard ?? false
  drawerTitle.value = '编辑通知模板'
  drawerVisible.value = true
}

const handleCopy = (row: MonitorNoticeTemplate) => {
  resetForm()
  const defaults = getTemplateDefaults(row.noticeType)
  Object.assign(form, {
    ...row,
    id: undefined,
    name: `${row.name}-复制`,
    template: '',
    templateFiring: row.templateFiring || row.template || defaults.firing,
    templateRecover: row.templateRecover || row.template || defaults.recover,
    enabled: true
  })
  drawerTitle.value = '复制通知模板'
  drawerVisible.value = true
}

const handleEnabledChange = async (row: MonitorNoticeTemplate) => {
  if (!row.id) return
  try {
    await updateMonitorNoticeTemplate(row.id, buildTemplatePayload(row))
    ElMessage.success('状态已更新')
  } catch {
    row.enabled = !row.enabled
  }
}

const handleDelete = async (row: MonitorNoticeTemplate) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除通知模板「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorNoticeTemplate(row.id)
  ElMessage.success('删除成功')
  await loadTemplates()
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (!String(form.templateFiring || '').trim() || !String(form.templateRecover || '').trim()) {
    ElMessage.warning('请填写触发模板和恢复模板')
    return
  }
  const payload = buildTemplatePayload(form)
  submitting.value = true
  try {
    if (form.id) {
      await updateMonitorNoticeTemplate(form.id, payload)
      ElMessage.success('更新成功')
    } else {
      await createMonitorNoticeTemplate(payload)
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    await loadTemplates()
  } finally {
    submitting.value = false
  }
}

const resetForm = () => {
  Object.assign(form, {
    id: undefined,
    name: '',
    noticeType: 'FeiShu',
    description: '',
    template: '',
    templateFiring: '',
    templateRecover: '',
    enableFeiShuJsonCard: true,
    enabled: true
  })
  formRef.value?.clearValidate()
}

const handleNoticeTypeChange = (type: string) => {
  fillTemplateDefaults(type)
}

const fillTemplateDefaults = (type: string) => {
  const defaults = getTemplateDefaults(type)
  form.template = ''
  form.templateFiring = defaults.firing
  form.templateRecover = defaults.recover
  form.enableFeiShuJsonCard = defaults.enableFeiShuJsonCard ?? false
}

const buildTemplatePayload = (source: MonitorNoticeTemplate): MonitorNoticeTemplate => {
  const defaults = getTemplateDefaults(source.noticeType)
  return {
    ...source,
    template: '',
    templateFiring: String(source.templateFiring || source.template || defaults.firing).trim(),
    templateRecover: String(source.templateRecover || source.template || defaults.recover).trim(),
    enableFeiShuJsonCard: source.noticeType === 'FeiShu'
      ? (source.enableFeiShuJsonCard ?? defaults.enableFeiShuJsonCard ?? true)
      : false
  }
}

const getNoticeTypeName = (type?: string) => noticeTypeOptions.find(item => item.value === type)?.label || type || '-'
const getNoticeTypeLogo = (type?: string) => noticeTypeOptions.find(item => item.value === type)?.logo || 'webhook'

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

onMounted(loadTemplates)
</script>

<style scoped>
.notice-template-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-head,
.type-rail,
.table-panel {
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
}

.page-head h2 {
  margin: 0;
  color: #111827;
  font-size: 21px;
  font-weight: 750;
}

.page-head p {
  margin: 5px 0 0;
  color: #667085;
  font-size: 13px;
}

.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.head-search {
  width: 260px;
}

.type-filter {
  width: 140px;
}

.template-layout {
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr);
  gap: 12px;
}

.type-rail {
  padding: 10px;
}

.type-item {
  width: 100%;
  min-height: 58px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: #344054;
  cursor: pointer;
  text-align: left;
}

.type-item:hover,
.type-item.active {
  border-color: #bfdbfe;
  background: #f8fbff;
}

.platform-logo {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  background: #f5f7fa;
}

.platform-logo.mini {
  width: 24px;
  height: 24px;
  border-radius: 6px;
}

.platform-logo svg {
  width: 27px;
  height: 27px;
  display: block;
}

.platform-logo.mini svg {
  width: 19px;
  height: 19px;
}

.platform-logo.logo-email {
  background: #eef6ff;
}

.platform-logo.logo-dingtalk {
  background: #eef7ff;
}

.platform-logo.logo-wecom {
  background: #f0fbf7;
}

.platform-logo.logo-webhook {
  background: #edf8ff;
}

.type-item strong,
.type-item em {
  display: block;
}

.type-item strong {
  color: #111827;
  font-size: 13px;
}

.type-item em {
  margin-top: 3px;
  color: #667085;
  font-style: normal;
  font-size: 12px;
}

.table-panel {
  overflow: hidden;
}

.dense-table {
  width: 100%;
}

.name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.name-cell strong {
  color: #111827;
  font-size: 13px;
}

.name-cell span {
  color: #98a2b3;
  font-size: 12px;
}

.table-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.type-chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  max-width: 100%;
  padding: 2px 8px 2px 4px;
  border: 1px solid #e5e9f2;
  border-radius: 7px;
  background: #f8fafc;
  color: #111827;
  font-size: 13px;
  font-weight: 650;
}

.icon-btn {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  color: #667085;
}

.icon-btn:hover {
  background: #eff6ff;
  color: #2563eb;
}

.icon-btn.danger:hover {
  background: #fff1f2;
  color: #e11d48;
}

.drawer-section {
  padding: 16px 0;
  border-bottom: 1px solid #edf1f7;
}

.drawer-section:first-child {
  padding-top: 0;
}

.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.section-title em {
  color: #667085;
  font-size: 12px;
  font-style: normal;
  font-weight: 500;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 14px;
}

.template-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.template-editor-card {
  display: block;
  margin-bottom: 0;
  padding: 12px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcfe;
}

.template-editor-card :deep(.el-form-item__label) {
  margin-bottom: 8px;
  color: #111827;
  font-weight: 700;
}

.template-editor-card :deep(.el-textarea__inner) {
  font-family: Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.55;
  border-radius: 7px;
  background: #fff;
}

:global(.monitor-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding: 16px 22px 14px;
  border-bottom: 1px solid #edf1f7;
  background: #fbfcfe;
}

:global(.monitor-drawer .el-drawer__body) {
  padding: 18px 22px;
}

:global(.monitor-drawer .el-drawer__footer) {
  padding: 12px 22px;
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
}

@media (max-width: 1120px) {
  .page-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-actions {
    justify-content: flex-start;
    width: 100%;
  }

  .template-layout {
    grid-template-columns: 1fr;
  }

  .type-rail {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
}

@media (max-width: 720px) {
  .head-search,
  .type-filter {
    width: 100%;
  }

  .type-rail,
  .form-grid,
  .template-editor-grid {
    grid-template-columns: 1fr;
  }

  .section-title {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
