<template>
  <div class="notice-object-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><MessageBox /></el-icon>
        </div>
        <div>
          <h2>通知对象</h2>
          <p>将通知路由、通知模板和值班表组合成故障中心可引用的通知对象</p>
        </div>
      </div>
      <div class="head-actions">
        <el-input v-model="keyword" clearable placeholder="搜索通知对象" class="head-search" @keyup.enter="loadObjects">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button @click="loadAll">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          创建通知对象
        </el-button>
      </div>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Bell /></el-icon>
        </div>
        <div>
          <div class="stat-label">通知对象</div>
          <div class="stat-value">{{ objects.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div>
          <div class="stat-label">启用中</div>
          <div class="stat-value">{{ enabledCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><DocumentCopy /></el-icon>
        </div>
        <div>
          <div class="stat-label">通知模板</div>
          <div class="stat-value">{{ templates.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><Calendar /></el-icon>
        </div>
        <div>
          <div class="stat-label">值班表</div>
          <div class="stat-value">{{ dutyTables.length }}</div>
        </div>
      </div>
    </div>

    <div class="table-panel">
      <el-table :data="objects" v-loading="loading" class="dense-table" :header-cell-style="tableHeaderStyle">
        <el-table-column label="名称" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="name-cell">
              <strong>{{ row.name }}</strong>
              <span>{{ row.uuid || `ID ${row.id}` }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="值班表" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.dutyTableName || getDutyTableName(row.dutyTableId) }}</template>
        </el-table-column>
        <el-table-column label="今日值班" min-width="200">
          <template #default="{ row }">
            <div class="user-tags">
              <el-tag v-if="!row.currentDutyUsers?.length" effect="light">暂无</el-tag>
              <el-tag v-for="user in row.currentDutyUsers || []" :key="`${row.id}-${user.username}`" effect="light">
                {{ user.realName || user.username }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="通知路由" min-width="260">
          <template #default="{ row }">
            <div class="route-tags">
              <el-tag v-for="route in parseRoutes(row.routes)" :key="`${row.id}-${route.noticeType}-${route.noticeTemplateId}`" effect="light">
                {{ getNoticeTypeName(route.noticeType) }} / {{ getTemplateName(route.noticeTemplateId) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="96" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleEnabledChange(row)" />
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="148" fixed="right" align="center">
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
    </div>

    <el-drawer v-model="drawerVisible" size="1080px" class="monitor-drawer watch-notice-drawer" :close-on-click-modal="false" :show-close="false">
      <template #header="{ close, titleId, titleClass }">
        <div class="watch-drawer-title">
          <el-button text class="drawer-close-btn" @click="close">
            <el-icon><Close /></el-icon>
          </el-button>
          <h3 :id="titleId" :class="titleClass">{{ drawerTitle }}</h3>
        </div>
      </template>

      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="watch-notice-form">
        <section class="watch-form-block watch-basic-block">
          <div class="watch-block-title">
            <span>基础信息</span>
          </div>
          <div class="watch-basic-grid">
            <el-form-item label="通知对象名称" prop="name">
              <el-input v-model="form.name" placeholder="请输入通知对象名称" />
            </el-form-item>
            <el-form-item label="值班表">
              <el-select v-model="form.dutyTableId" clearable filterable placeholder="请选择值班表" style="width: 100%">
                <el-option v-for="table in dutyTables" :key="table.id" :label="table.name" :value="table.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="描述" class="full-row">
              <el-input v-model="form.description" :rows="3" type="textarea" placeholder="请输入描述" />
            </el-form-item>
            <el-form-item label="启用状态" class="enable-row">
              <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
            </el-form-item>
          </div>
        </section>

        <section class="watch-form-block watch-route-block">
          <div class="watch-block-title">
            <span>通知路由</span>
          </div>

          <div v-for="(route, index) in form.routes" :key="route.key" class="watch-route-card">
            <div class="watch-route-head">
              <div>
                <strong>通知策略 {{ index + 1 }}</strong>
                <span>{{ getNoticeTypeName(route.noticeType) }}</span>
              </div>
	              <div class="watch-route-actions">
	                <el-button class="watch-test-btn" :loading="testingRouteKey === route.key" @click="handleTestRoute(index)">
	                  通知测试
	                </el-button>
	                <el-switch v-model="route.enabled" active-text="启用" inactive-text="停用" />
	                <el-button text class="watch-delete-btn" :disabled="form.routes.length === 1" @click="removeRoute(index)">
	                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>

            <div class="watch-required-caption"><span>*</span> 通知类型</div>
            <div class="watch-type-grid">
              <button
                v-for="item in noticeTypeOptions"
                :key="item.value"
                type="button"
                class="watch-type-card"
                :class="{ active: route.noticeType === item.value }"
                @click="selectNoticeType(route, item.value)"
              >
                <span class="watch-type-logo" :class="`logo-${item.logo}`">
                  <svg v-if="item.value === 'FeiShu'" viewBox="0 0 48 48" aria-hidden="true">
                    <path fill="#2262ff" d="M10 29.4 22.6 36.6c2.9 1.7 6.6 1.2 9-1.2l8.9-8.9c1.1-1.1.1-3-1.4-2.7l-12.8 2.4c-1.5.3-3.1-.1-4.3-1.1L11.7 16.4c-1.2-1-3 .1-2.7 1.7l1 5.5c.4 2.4 1.8 4.5 4 5.8Z" />
                    <path fill="#0cc7a3" d="M17.7 11.1 29.5 25l9.4-1.8c1.8-.3 2.6-2.5 1.3-3.8L29.9 9.4c-2.8-2.7-7-3.2-10.3-1.2l-1.4.8c-.8.5-1 1.4-.5 2.1Z" />
                    <path fill="#7c5cff" d="M10 17.2 22 26.8c1.2 1 2.8 1.4 4.3 1.1l3.2-.6-12-14.1c-.6-.7-1.6-.9-2.4-.4L10.7 15c-.9.5-1 1.6-.7 2.2Z" />
                  </svg>
                  <svg v-else-if="item.value === 'Email'" viewBox="0 0 48 48" aria-hidden="true">
                    <rect x="7" y="11" width="34" height="26" rx="7" fill="#2f80ed" />
                    <path fill="#fff" d="M11.6 17.7c.5-.7 1.5-.9 2.2-.4L24 24.5l10.2-7.2c.7-.5 1.7-.3 2.2.4.5.7.3 1.7-.4 2.2l-11.1 7.8c-.5.4-1.2.4-1.8 0L12 19.9c-.7-.5-.9-1.5-.4-2.2Z" />
                  </svg>
                  <svg v-else-if="item.value === 'DingDing'" viewBox="0 0 48 48" aria-hidden="true">
                    <path fill="#38a3ff" d="M35.8 7.8c1.3.6 1.7 2.3.8 3.4l-16.8 22c-.9 1.1-2.7.8-3.1-.6l-2-6.8-6.4-2.5c-1.3-.5-1.4-2.3-.2-3L35.8 7.8Z" />
                    <path fill="#64c4ff" d="m19.9 33.3 3.6 7.4c.6 1.2 2.3 1.2 2.8.1l10.1-22.1-16.5 14.6Z" />
                  </svg>
                  <svg v-else-if="item.value === 'WeChat'" viewBox="0 0 48 48" aria-hidden="true">
                    <path fill="#12b76a" d="M22.3 13.2c-8 0-14.4 5.1-14.4 11.4 0 3.4 1.9 6.5 4.8 8.5l-1 4.2 5-2.3c1.7.6 3.6 1 5.6 1 8 0 14.4-5.1 14.4-11.4s-6.4-11.4-14.4-11.4Z" />
                    <path fill="#2f80ed" d="M31 20.9c5.8 0 10.5 3.8 10.5 8.4 0 2.5-1.4 4.8-3.6 6.3l.7 3.1-3.7-1.7c-1.2.5-2.6.7-4 .7-5.8 0-10.5-3.8-10.5-8.4s4.8-8.4 10.6-8.4Z" />
                    <circle cx="17.4" cy="23.3" r="1.8" fill="#fff" />
                    <circle cx="26" cy="23.3" r="1.8" fill="#fff" />
                  </svg>
                  <svg v-else viewBox="0 0 48 48" aria-hidden="true">
                    <path fill="#4bb8f3" d="M18.6 9.5h10.8l1.1 5.2c1 .4 1.9.9 2.8 1.6l5.1-1.7 5.4 9.4-4 3.5c.1.6.2 1.2.2 1.8s-.1 1.2-.2 1.8l4 3.5-5.4 9.4-5.1-1.7c-.9.7-1.8 1.2-2.8 1.6l-1.1 5.2H18.6l-1.1-5.2c-1-.4-1.9-.9-2.8-1.6l-5.1 1.7-5.4-9.4 4-3.5c-.1-.6-.2-1.2-.2-1.8s.1-1.2.2-1.8l-4-3.5 5.4-9.4 5.1 1.7c.9-.7 1.8-1.2 2.8-1.6l1.1-5.2Z" />
                    <circle cx="24" cy="29.3" r="7.1" fill="#fff" />
                    <circle cx="24" cy="29.3" r="3.3" fill="#4bb8f3" />
                  </svg>
                </span>
                <span class="watch-type-text">
                  <strong>{{ item.label }}</strong>
                </span>
              </button>
            </div>

            <div class="watch-config-grid">
	              <template v-if="!isEmailRoute(route)">
	                <el-form-item label="Hook 地址" class="full-row" required>
	                  <el-input
	                    v-model="route.hook"
	                    :placeholder="route.noticeType === 'WebHook' ? '请输入接收告警 JSON 的 HTTP 地址' : '请输入机器人 WebHook 地址'"
	                  />
	                </el-form-item>
	                <el-form-item v-if="route.noticeType === 'DingDing'" label="加签密钥（选填）" class="full-row">
	                  <el-input
	                    v-model="route.sign"
	                    placeholder="未开启加签请留空；开启加签时填写 SEC 开头的密钥"
	                  />
	                  <div class="watch-field-tip">
	                    未开启加签请留空；如果机器人启用了关键词安全，通知模板正文必须包含机器人配置的关键词。
	                  </div>
	                </el-form-item>
                <template v-if="route.noticeType === 'FeiShu'">
                  <el-form-item label="飞书 App ID">
                    <el-input v-model="route.feishuAppId" placeholder="选填，用于上传回调图表图片" />
                  </el-form-item>
                  <el-form-item label="飞书 App Secret">
                    <el-input v-model="route.feishuAppSecret" type="password" show-password placeholder="选填，用于上传回调图表图片" />
                  </el-form-item>
                </template>
              </template>

              <el-form-item label="通知模板" class="full-row" required>
                <el-select v-model="route.noticeTemplateId" filterable placeholder="请选择通知模板" style="width: 100%">
                  <el-option v-for="template in getTemplatesByType(route.noticeType)" :key="template.id" :label="template.name" :value="String(template.id)" />
                </el-select>
              </el-form-item>
              <el-form-item label="适用级别" class="full-row" required>
                <el-checkbox-group v-model="route.severitys" class="level-checks">
                  <el-checkbox-button label="P0">P0 紧急</el-checkbox-button>
                  <el-checkbox-button label="P1">P1 重要</el-checkbox-button>
                  <el-checkbox-button label="P2">P2 一般</el-checkbox-button>
                </el-checkbox-group>
              </el-form-item>

              <template v-if="isEmailRoute(route)">
                <el-form-item label="邮件主题" required>
                  <el-input v-model="route.subject" placeholder="请输入邮件主题" />
                </el-form-item>
                <el-form-item label="SMTP 服务器" required>
                  <el-input v-model="route.smtpHost" placeholder="smtp.example.com" />
                </el-form-item>
                <el-form-item label="SMTP 端口" required>
                  <el-input-number v-model="route.smtpPort" :min="1" :max="65535" controls-position="right" style="width: 100%" />
                </el-form-item>
                <el-form-item label="发件邮箱" required>
                  <el-input v-model="route.fromEmail" placeholder="notice@example.com" />
                </el-form-item>
                <el-form-item label="发件名称">
                  <el-input v-model="route.fromName" placeholder="OpsHub" />
                </el-form-item>
                <el-form-item label="SMTP 用户名">
                  <el-input v-model="route.smtpUser" placeholder="通常与发件邮箱一致" />
                </el-form-item>
                <el-form-item label="SMTP 密码">
                  <el-input v-model="route.smtpPassword" type="password" show-password placeholder="请输入 SMTP 授权码或密码" />
                </el-form-item>
                <el-form-item label="固定收件人">
                  <el-select v-model="route.to" multiple filterable allow-create default-first-option placeholder="选填，未填写时使用值班表当班人员邮箱" style="width: 100%">
                    <el-option v-for="user in userOptions" :key="`to-${user.username}`" :label="userLabel(user)" :value="user.email" :disabled="!user.email" />
                  </el-select>
                </el-form-item>
                <el-form-item label="抄送人" class="full-row">
                  <el-select v-model="route.cc" multiple filterable allow-create default-first-option placeholder="请输入或选择抄送人" style="width: 100%">
                    <el-option v-for="user in userOptions" :key="`cc-${user.username}`" :label="userLabel(user)" :value="user.email" :disabled="!user.email" />
                  </el-select>
                </el-form-item>
              </template>

              <el-form-item label="生效时间" class="full-row">
                <div class="watch-effective-time">
                  <el-select
                    v-model="route.week"
                    multiple
                    collapse-tags
                    collapse-tags-tooltip
                    clearable
                    placeholder="选择生效星期，不选代表每天"
                    class="week-select"
                  >
                    <el-option v-for="item in weekOptions" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                  <el-time-picker v-model="route.startTime" value-format="HH:mm" format="HH:mm" placeholder="开始时间" />
                  <span>至</span>
                  <el-time-picker v-model="route.endTime" value-format="HH:mm" format="HH:mm" placeholder="结束时间" />
                </div>
              </el-form-item>
            </div>

            <div v-if="!isEmailRoute(route)" class="watch-headers-editor">
              <div class="watch-mini-title">
                <span>自定义 Header</span>
                <el-button link type="primary" @click="addHeaderRow(route)">
                  <el-icon><Plus /></el-icon>
                  添加
                </el-button>
              </div>
              <div v-if="route.headers.length" class="watch-header-list">
                <div v-for="(header, headerIndex) in route.headers" :key="header.key" class="watch-header-row">
                  <el-input v-model="header.name" placeholder="Header 名称" />
                  <el-input v-model="header.value" placeholder="Header 值" />
                  <el-button text class="watch-delete-btn" @click="removeHeaderRow(route, headerIndex)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
              <div v-else class="watch-empty-line">默认不添加自定义 Header</div>
            </div>
          </div>

          <button type="button" class="watch-add-route" @click="addRoute">
            <el-icon><Plus /></el-icon>
            添加策略
          </button>
        </section>
      </el-form>
      <template #footer>
        <div class="watch-drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button class="watch-submit-btn" type="primary" :loading="submitting" @click="handleSubmit">提交</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Bell, Calendar, CircleCheck, Close, CopyDocument, Delete, DocumentCopy, Edit, MessageBox, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { getUserList } from '@/api/user'
import {
  createMonitorNoticeObject,
  deleteMonitorNoticeObject,
  getMonitorDutyTables,
  getMonitorNoticeObjects,
  getMonitorNoticeTemplates,
  testMonitorNoticeObject,
  updateMonitorNoticeObject,
  type MonitorDutyTable,
  type MonitorDutyUser,
  type MonitorNoticeObject,
  type MonitorNoticeTemplate
} from '@/api/monitor-datasource'

interface RouteForm {
  key: string
  noticeType: string
  noticeTemplateId: string
  severitys: string[]
  hook: string
  feishuAppId: string
  feishuAppSecret: string
  headers: HeaderRow[]
  sign: string
  subject: string
  smtpHost: string
  smtpPort: number
  smtpUser: string
  smtpPassword: string
  fromEmail: string
  fromName: string
  to: string[]
  cc: string[]
  week: string[]
  startTime: string
  endTime: string
  enabled: boolean
}

interface HeaderRow {
  key: string
  name: string
  value: string
}

interface ParsedRoute {
  noticeType?: string
  noticeTemplateId?: string | number
  severitys?: string[]
}

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
const loading = ref(false)
const submitting = ref(false)
const testingRouteKey = ref('')
const drawerVisible = ref(false)
const drawerTitle = ref('新建通知对象')
const keyword = ref('')
const formRef = ref<FormInstance>()
const objects = ref<MonitorNoticeObject[]>([])
const templates = ref<MonitorNoticeTemplate[]>([])
const dutyTables = ref<MonitorDutyTable[]>([])
const userOptions = ref<MonitorDutyUser[]>([])

const noticeTypeOptions = [
  { label: '飞书', value: 'FeiShu', logo: 'feishu' },
  { label: '邮件', value: 'Email', logo: 'email' },
  { label: '钉钉', value: 'DingDing', logo: 'dingtalk' },
  { label: '企业微信', value: 'WeChat', logo: 'wecom' },
  { label: 'WebHook', value: 'WebHook', logo: 'webhook' }
]

const weekOptions = [
  { label: '周一', value: 'Monday' },
  { label: '周二', value: 'Tuesday' },
  { label: '周三', value: 'Wednesday' },
  { label: '周四', value: 'Thursday' },
  { label: '周五', value: 'Friday' },
  { label: '周六', value: 'Saturday' },
  { label: '周日', value: 'Sunday' }
]

const form = reactive({
  id: undefined as number | undefined,
  name: '',
  description: '',
  dutyTableId: undefined as number | undefined,
  enabled: true,
  routes: [] as RouteForm[]
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入通知对象名称', trigger: 'blur' }]
}

const enabledCount = computed(() => objects.value.filter(item => item.enabled).length)

const loadObjects = async () => {
  loading.value = true
  try {
    objects.value = await getMonitorNoticeObjects({ keyword: keyword.value.trim() }) || []
  } finally {
    loading.value = false
  }
}

const loadMeta = async () => {
  const [templateData, dutyData, users] = await Promise.all([
    getMonitorNoticeTemplates(),
    getMonitorDutyTables(),
    getUserList({ page: 1, pageSize: 500 })
  ])
  templates.value = (templateData || []).filter(item => isAllowedNoticeType(item.noticeType))
  dutyTables.value = dutyData || []
  const list = Array.isArray(users?.list) ? users.list : Array.isArray(users) ? users : []
  userOptions.value = list.map((item: any) => ({
    id: item.id || item.ID,
    username: item.username,
    realName: item.realName,
    email: item.email,
    phone: item.phone,
    notifyUserId: item.notifyUserId || item.feishuOpenId || item.feishuUserId || item.dingtalkUserId || item.wecomUserId,
    feishuUserId: item.feishuUserId,
    feishuOpenId: item.feishuOpenId,
    dingtalkUserId: item.dingtalkUserId,
    wecomUserId: item.wecomUserId
  })).filter((item: MonitorDutyUser) => item.username)
}

const loadAll = async () => {
  await Promise.all([loadMeta(), loadObjects()])
}

const handleAdd = () => {
  resetForm()
  addRoute()
  drawerTitle.value = '创建通知对象'
  drawerVisible.value = true
}

const handleEdit = (row: MonitorNoticeObject) => {
  resetForm()
  form.id = row.id
  form.name = row.name
  form.description = row.description || ''
  form.dutyTableId = row.dutyTableId
  form.enabled = row.enabled ?? true
  form.routes = parseRouteForms(row.routes)
  if (!form.routes.length) addRoute()
  drawerTitle.value = '编辑通知对象'
  drawerVisible.value = true
}

const handleCopy = (row: MonitorNoticeObject) => {
  handleEdit(row)
  form.id = undefined
  form.name = `${row.name}-复制`
  form.enabled = true
  drawerTitle.value = '复制通知对象'
}

const handleEnabledChange = async (row: MonitorNoticeObject) => {
  if (!row.id) return
  try {
    await updateMonitorNoticeObject(row.id, { ...row })
    ElMessage.success('状态已更新')
  } catch {
    row.enabled = !row.enabled
  }
}

const handleDelete = async (row: MonitorNoticeObject) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除通知对象「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorNoticeObject(row.id)
  ElMessage.success('删除成功')
  await loadObjects()
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (!form.routes.length) {
    ElMessage.warning('请至少配置一条通知路由')
    return
  }
  if (!validateRoutesBeforeSubmit()) {
    return
  }
  const routes = form.routes.map(buildRoutePayload)
  submitting.value = true
  try {
    const payload: MonitorNoticeObject = {
      name: form.name,
      description: form.description,
      dutyTableId: form.dutyTableId || 0,
      enabled: form.enabled,
      lastStatus: 'ready',
      routes: JSON.stringify(routes)
    }
    if (form.id) {
      await updateMonitorNoticeObject(form.id, payload)
      ElMessage.success('更新成功')
    } else {
      await createMonitorNoticeObject(payload)
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    await loadAll()
  } finally {
    submitting.value = false
  }
}

const buildNoticeObjectPayload = (): MonitorNoticeObject => ({
  name: form.name || '通知对象测试',
  description: form.description,
  dutyTableId: form.dutyTableId || 0,
  enabled: form.enabled,
  lastStatus: 'ready',
  routes: JSON.stringify(form.routes.map(buildRoutePayload))
})

const handleTestRoute = async (index: number) => {
  const route = form.routes[index]
  if (!route) return
  if (!validateRoute(route, index)) {
    return
  }
  testingRouteKey.value = route.key
  try {
    const testSeverity = route.severitys[0] || 'P1'
    await testMonitorNoticeObject({
      noticeObject: buildNoticeObjectPayload(),
      routeIndex: index,
      severity: testSeverity,
      state: 'firing'
    })
    ElMessage.success(`通知测试已发送（本次使用 ${testSeverity}；真实告警还会校验等级和生效时间）`)
  } finally {
    testingRouteKey.value = ''
  }
}

const addRoute = () => {
  form.routes.push({
    key: `${Date.now()}-${Math.random()}`,
    noticeType: 'FeiShu',
    noticeTemplateId: getDefaultTemplateId('FeiShu'),
    severitys: ['P0'],
    hook: '',
    feishuAppId: '',
    feishuAppSecret: '',
    headers: [],
    sign: '',
    subject: '',
    smtpHost: '',
    smtpPort: 465,
    smtpUser: '',
    smtpPassword: '',
    fromEmail: '',
    fromName: 'OpsHub',
    to: [],
    cc: [],
    week: [],
    startTime: '',
    endTime: '',
    enabled: true
  })
}

const removeRoute = (index: number) => {
  if (form.routes.length <= 1) return
  form.routes.splice(index, 1)
}

const selectNoticeType = (route: RouteForm, type: string) => {
  route.noticeType = type
  route.noticeTemplateId = getDefaultTemplateId(type)
  route.hook = ''
  route.feishuAppId = ''
  route.feishuAppSecret = ''
  route.subject = ''
  route.smtpHost = ''
  route.smtpPort = 465
  route.smtpUser = ''
  route.smtpPassword = ''
  route.fromEmail = ''
  route.fromName = 'OpsHub'
  route.to = []
  route.cc = []
}

const getDefaultTemplateId = (type?: string) => {
  const template = getTemplatesByType(type)[0]
  return template?.id ? String(template.id) : ''
}

const validateRoutesBeforeSubmit = () => {
  for (const [index, route] of form.routes.entries()) {
    if (!validateRoute(route, index)) {
      return false
    }
  }
  return true
}

const validateRoute = (route: RouteForm, index: number) => {
  const label = `通知策略 ${index + 1}`
  if (!isAllowedNoticeType(route.noticeType)) {
    ElMessage.warning(`${label} 请选择通知类型`)
    return false
  }
  if (!String(route.noticeTemplateId || '').trim()) {
    ElMessage.warning(`${label} 请选择通知模板`)
    return false
  }
  if (!route.severitys.length) {
    ElMessage.warning(`${label} 请选择告警级别`)
    return false
  }
  if (isEmailRoute(route)) {
    if (!route.subject.trim()) {
      ElMessage.warning(`${label} 请填写邮件主题`)
      return false
    }
    if (!route.smtpHost.trim()) {
      ElMessage.warning(`${label} 请填写 SMTP 服务器`)
      return false
    }
    if (!Number(route.smtpPort)) {
      ElMessage.warning(`${label} 请填写 SMTP 端口`)
      return false
    }
    if (!route.fromEmail.trim()) {
      ElMessage.warning(`${label} 请填写发件邮箱`)
      return false
    }
    if (route.smtpUser.trim() && !route.smtpPassword.trim()) {
      ElMessage.warning(`${label} 请填写 SMTP 密码`)
      return false
    }
    if (!hasEmailRecipients(route.to) && !form.dutyTableId) {
      ElMessage.warning(`${label} 请填写固定收件人或选择值班表`)
      return false
    }
  } else if (!route.hook.trim()) {
    ElMessage.warning(`${label} 请填写 Hook 地址`)
    return false
  }
  if ((route.startTime && !route.endTime) || (!route.startTime && route.endTime)) {
    ElMessage.warning(`${label} 请同时填写生效开始时间和结束时间`)
    return false
  }
  return true
}

const buildRoutePayload = (route: RouteForm) => ({
  noticeType: route.noticeType,
  noticeTemplateId: route.noticeTemplateId,
  severitys: route.severitys.length ? route.severitys : ['P0'],
  hook: route.hook,
  feishuAppId: route.noticeType === 'FeiShu' ? route.feishuAppId.trim() : '',
  feishuAppSecret: route.noticeType === 'FeiShu' ? route.feishuAppSecret.trim() : '',
  headers: buildHeaders(route.headers),
  sign: route.sign,
  subject: route.subject,
  smtpHost: isEmailRoute(route) ? route.smtpHost.trim() : '',
  smtpPort: isEmailRoute(route) ? route.smtpPort : 0,
  smtpUser: isEmailRoute(route) ? route.smtpUser.trim() : '',
  smtpPassword: isEmailRoute(route) ? route.smtpPassword : '',
  fromEmail: isEmailRoute(route) ? route.fromEmail.trim() : '',
  fromName: isEmailRoute(route) ? route.fromName.trim() : '',
  to: isEmailRoute(route) ? route.to : [],
  cc: route.cc,
  effectiveTime: {
    week: route.week,
    startTime: route.startTime,
    endTime: route.endTime
  },
  enabled: route.enabled
})

const parseRouteForms = (raw?: string): RouteForm[] => {
  const routes = safeParse<any[]>(raw, [])
  if (!Array.isArray(routes)) return []
  return routes.map(route => ({
    key: `${Date.now()}-${Math.random()}`,
    noticeType: normalizeNoticeRouteType(route.noticeType),
    noticeTemplateId: String(route.noticeTemplateId || route.noticeTmplId || ''),
    severitys: Array.isArray(route.severitys) ? route.severitys : ['P0'],
    hook: route.hook || '',
    feishuAppId: route.feishuAppId || '',
    feishuAppSecret: route.feishuAppSecret || '',
    headers: parseHeaders(route.headers),
    sign: route.sign || '',
    subject: route.subject || '',
    smtpHost: route.smtpHost || '',
    smtpPort: Number(route.smtpPort || 465),
    smtpUser: route.smtpUser || '',
    smtpPassword: route.smtpPassword || '',
    fromEmail: route.fromEmail || '',
    fromName: route.fromName || 'OpsHub',
    to: Array.isArray(route.to) ? route.to : [],
    cc: Array.isArray(route.cc) ? route.cc : [],
    week: Array.isArray(route.effectiveTime?.week) ? route.effectiveTime.week : [],
    startTime: route.effectiveTime?.startTime || '',
    endTime: route.effectiveTime?.endTime || '',
    enabled: route.enabled ?? true
  }))
}

const parseRoutes = (raw?: string): ParsedRoute[] => {
  const routes = safeParse<ParsedRoute[]>(raw, [])
  return Array.isArray(routes) ? routes : []
}

const hasEmailRecipients = (values: string[]) => values.some(value => String(value || '').includes('@'))

const addHeaderRow = (route: RouteForm) => {
  route.headers.push({ key: `${Date.now()}-${Math.random()}`, name: '', value: '' })
}

const removeHeaderRow = (route: RouteForm, index: number) => {
  route.headers.splice(index, 1)
}

const buildHeaders = (headers: HeaderRow[]) => headers.reduce<Record<string, string>>((acc, item) => {
  const name = item.name.trim()
  if (!name) return acc
  acc[name] = item.value.trim()
  return acc
}, {})

const parseHeaders = (headers: unknown): HeaderRow[] => {
  const normalized = typeof headers === 'string'
    ? safeParse<Record<string, unknown>>(headers, {})
    : headers
  if (!normalized || typeof normalized !== 'object' || Array.isArray(normalized)) return []
  return Object.entries(normalized as Record<string, unknown>).map(([name, value]) => ({
    key: `${Date.now()}-${Math.random()}`,
    name,
    value: String(value ?? '')
  }))
}

const safeParse = <T,>(raw: string | undefined, fallback: T): T => {
  if (!raw) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

const normalizeNoticeRouteType = (type?: string) => noticeTypeOptions.some(item => item.value === type) ? String(type) : 'FeiShu'
const isAllowedNoticeType = (type?: string) => noticeTypeOptions.some(item => item.value === type)

const resetForm = () => {
  form.id = undefined
  form.name = ''
  form.description = ''
  form.dutyTableId = undefined
  form.enabled = true
  form.routes = []
  formRef.value?.clearValidate()
}

const getTemplatesByType = (type?: string) => templates.value.filter(item => item.noticeType === type && item.enabled !== false)
const getTemplateName = (id?: number | string) => templates.value.find(item => String(item.id) === String(id))?.name || '未选模板'
const getDutyTableName = (id?: number) => dutyTables.value.find(item => item.id === id)?.name || '-'
const getNoticeTypeName = (type?: string) => noticeTypeOptions.find(item => item.value === type)?.label || type || '-'
const userLabel = (user: MonitorDutyUser) => `${user.realName || user.username}${user.email ? ` / ${user.email}` : ' / 未配置邮箱'}`
const isEmailRoute = (route: RouteForm) => route.noticeType === 'Email'

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

onMounted(loadAll)
</script>

<style scoped>
.notice-object-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-head,
.overview-strip,
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

.overview-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.metric-item {
  padding: 14px 18px;
  border-right: 1px solid #edf1f7;
}

.metric-item:last-child {
  border-right: 0;
}

.metric-item span {
  display: block;
  color: #667085;
  font-size: 12px;
}

.metric-item strong {
  display: block;
  margin-top: 5px;
  color: #111827;
  font-size: 24px;
  font-weight: 760;
  line-height: 1;
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

.user-tags,
.route-tags,
.table-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.table-actions {
  justify-content: center;
  gap: 4px;
  flex-wrap: nowrap;
  white-space: nowrap;
}

.icon-btn {
  width: 30px;
  height: 30px;
  margin-left: 0;
  padding: 0;
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

.drawer-section.basic-section {
  padding-top: 0;
}

.drawer-section:first-child {
  padding-top: 0;
}

.section-head,
.route-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-head {
  margin-bottom: 12px;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.section-head.plain {
  align-items: flex-start;
}

.section-head > div > strong,
.section-head > div > span {
  display: block;
}

.section-head > div > span {
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  font-weight: 400;
  line-height: 1.5;
}

.route-card {
  margin-bottom: 12px;
  padding: 16px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.route-card-head {
  margin-bottom: 12px;
}

.route-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.route-card-head strong,
.route-card-head span {
  display: block;
}

.route-card-head strong {
  color: #111827;
  font-size: 14px;
}

.route-card-head span {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
}

.type-picker {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}

.type-pill {
  min-height: 62px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 1px solid #e5e9f2;
  border-radius: 7px;
  background: #fff;
  color: #344054;
  cursor: pointer;
}

.type-pill.active,
.type-pill:hover {
  border-color: #2563eb;
  background: #f8fbff;
  color: #1d4ed8;
}

.type-pill span {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #e5e9f2;
  border-radius: 7px;
  background: #f8fafc;
  font-weight: 760;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 14px;
}

.compact-grid {
  grid-template-columns: minmax(0, 1fr);
}

.time-range {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.time-range :deep(.el-date-editor) {
  flex: 1;
}

.headers-editor {
  padding: 12px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcfe;
}

.mini-head,
.header-row {
  display: flex;
  align-items: center;
}

.mini-head {
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
  color: #111827;
  font-size: 13px;
  font-weight: 750;
}

.header-row {
  gap: 8px;
}

.header-row + .header-row {
  margin-top: 8px;
}

.header-row .el-input:first-child {
  flex: 0 0 35%;
}

.header-row .el-input:nth-child(2) {
  min-width: 0;
  flex: 1;
}

.headers-editor :deep(.el-empty) {
  padding: 4px 0;
}

.watch-drawer-title {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.watch-drawer-title h3 {
  margin: 0;
  color: #111827;
  font-size: 17px;
  font-weight: 650;
  line-height: 1.4;
}

.drawer-close-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  color: #344054;
}

.watch-notice-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.watch-form-block {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background: #fff;
}

.watch-basic-block,
.watch-route-block {
  border: 0;
  border-radius: 0;
}

.watch-basic-block .watch-block-title {
  display: none;
}

.watch-basic-block .watch-basic-grid {
  padding: 6px 0 24px;
  border-bottom: 1px solid #ebeef5;
}

.watch-route-block .watch-block-title {
  min-height: 38px;
  padding: 0;
  border-bottom: 0;
  font-size: 16px;
}

.watch-block-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
  padding: 0 18px;
  border-bottom: 1px solid #ebeef5;
  color: #1f2937;
  font-size: 14px;
  font-weight: 650;
}

.watch-basic-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 20px;
  padding: 18px;
}

.watch-config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  padding: 22px;
}

.watch-basic-grid :deep(.el-form-item),
.watch-config-grid :deep(.el-form-item) {
  margin-bottom: 0;
}

.watch-notice-form :deep(.el-form-item__label) {
  margin-bottom: 7px;
  color: #606266;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.2;
}

.watch-notice-form :deep(.el-input__wrapper),
.watch-notice-form :deep(.el-textarea__inner),
.watch-notice-form :deep(.el-select__wrapper) {
  border-radius: 4px;
  box-shadow: 0 0 0 1px #dcdfe6 inset;
}

.watch-notice-form :deep(.el-input__wrapper:hover),
.watch-notice-form :deep(.el-select__wrapper:hover),
.watch-notice-form :deep(.el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset;
}

.watch-field-tip {
  margin-top: 7px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}

.full-row {
  grid-column: 1 / -1;
}

.enable-row {
  align-self: center;
}

.watch-route-card {
  margin: 18px 0 22px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.watch-route-card + .watch-route-card {
  margin-top: 0;
}

.watch-route-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 54px;
  padding: 0 22px;
  border-bottom: 1px solid #ebeef5;
}

.watch-route-head strong,
.watch-route-head span {
  display: block;
}

.watch-route-head strong {
  color: #1f2937;
  font-size: 14px;
  font-weight: 650;
}

.watch-route-head span {
  margin-top: 2px;
  color: #909399;
  font-size: 12px;
}

.watch-route-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.watch-test-btn {
  height: 30px;
  padding: 0 12px;
  border-color: #dcdfe6;
  border-radius: 4px;
  color: #1f2937;
  font-size: 13px;
  font-weight: 500;
}

.watch-test-btn:hover {
  border-color: #111827;
  color: #111827;
  background: #fafafa;
}

.watch-delete-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  color: #909399;
}

.watch-delete-btn:hover {
  color: #f56c6c;
  background: #fef0f0;
}

.watch-type-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
  padding: 10px 22px 0;
}

.watch-required-caption {
  padding: 20px 22px 0;
  color: #303133;
  font-size: 13px;
  font-weight: 600;
}

.watch-required-caption span {
  color: #f56c6c;
}

.watch-type-card {
  min-height: 112px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 16px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
  color: #303133;
  text-align: center;
  cursor: pointer;
  transition: border-color .15s ease, box-shadow .15s ease, background .15s ease, transform .15s ease;
}

.watch-type-card:hover,
.watch-type-card.active {
  border-color: #2f88ff;
  background: #fff;
  box-shadow: 0 0 0 1px rgba(47, 136, 255, .18);
}

.watch-type-card:hover {
  transform: translateY(-1px);
}

.watch-type-logo {
  width: 52px;
  height: 52px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 52px;
  border-radius: 12px;
  background: #f5f7fa;
}

.watch-type-logo svg {
  width: 42px;
  height: 42px;
  display: block;
}

.watch-type-logo.logo-email {
  background: #eef6ff;
}

.watch-type-logo.logo-dingtalk {
  background: #eef7ff;
}

.watch-type-logo.logo-wecom {
  background: #f0fbf7;
}

.watch-type-logo.logo-webhook {
  background: #edf8ff;
}

.watch-type-text {
  min-width: 0;
}

.watch-type-text strong,
.watch-type-text em {
  display: block;
  font-style: normal;
}

.watch-type-text strong {
  color: #1f2937;
  font-size: 15px;
  font-weight: 600;
}

.watch-type-text em {
  margin-top: 4px;
  color: #909399;
  font-size: 12px;
  line-height: 1.2;
}

.level-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.level-checks :deep(.el-checkbox-button__inner) {
  min-width: 42px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 8px 12px;
  box-shadow: none;
}

.level-checks :deep(.el-checkbox-button.is-checked .el-checkbox-button__inner) {
  border-color: #303133;
  background: #303133;
  color: #fff;
  box-shadow: none;
}

.watch-effective-time {
  display: grid;
  grid-template-columns: minmax(260px, 1.2fr) minmax(150px, .8fr) auto minmax(150px, .8fr);
  align-items: center;
  gap: 8px;
  width: 100%;
}

.watch-effective-time :deep(.el-date-editor.el-input),
.watch-effective-time .week-select {
  width: 100%;
}

.watch-effective-time span {
  color: #909399;
  font-size: 12px;
}

.watch-headers-editor {
  margin: 0 22px 22px;
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  background: #fafafa;
}

.watch-mini-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #606266;
  font-size: 13px;
  font-weight: 600;
}

.watch-header-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.watch-header-row {
  display: grid;
  grid-template-columns: minmax(120px, .7fr) minmax(0, 1fr) 32px;
  gap: 8px;
  align-items: center;
}

.watch-empty-line {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  color: #909399;
  font-size: 12px;
  background: #fff;
}

.watch-add-route {
  width: 100%;
  min-height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0 0 22px;
  border: 1px dashed #c0c4cc;
  border-radius: 4px;
  background: #fff;
  color: #303133;
  font-size: 13px;
  cursor: pointer;
}

.watch-add-route:hover {
  border-color: #303133;
  background: #fafafa;
}

.watch-drawer-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.watch-submit-btn {
  min-width: 88px;
  border-color: #111827;
  background: #111827;
}

.watch-submit-btn:hover,
.watch-submit-btn:focus {
  border-color: #303133;
  background: #303133;
}

:global(.monitor-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding: 14px 22px;
  border-bottom: 1px solid #edf1f7;
  background: #fff;
}

:global(.monitor-drawer .el-drawer__body) {
  padding: 18px 22px 22px;
  background: #f5f7fa;
}

:global(.watch-notice-drawer .el-drawer__body) {
  padding: 24px 28px 28px;
  background: #fff;
}

:global(.monitor-drawer .el-drawer__footer) {
  padding: 12px 22px;
  border-top: 1px solid #edf1f7;
  background: #fff;
}

@media (max-width: 980px) {
  .page-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-actions,
  .head-search {
    width: 100%;
  }

  .overview-strip,
  .form-grid,
  .type-picker,
  .watch-basic-grid,
  .watch-config-grid,
  .watch-type-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .overview-strip,
  .form-grid,
  .type-picker,
  .watch-basic-grid,
  .watch-config-grid,
  .watch-type-grid,
  .watch-header-row {
    grid-template-columns: 1fr;
  }

  .watch-route-head {
    align-items: flex-start;
    flex-direction: column;
    padding: 12px 16px;
  }

  .watch-effective-time {
    grid-template-columns: 1fr;
  }
}
</style>
