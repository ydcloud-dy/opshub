<template>
  <div class="duty-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Calendar /></el-icon>
        </div>
        <div>
          <h2>值班表</h2>
          <p>维护值班表负责人，并按日期选择值班用户</p>
        </div>
      </div>
      <div class="head-actions">
        <el-input v-model="keyword" clearable placeholder="搜索值班表" class="head-search" @keyup.enter="loadDutyTables">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button @click="loadAll">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="handleAddTable">
          <el-icon><Plus /></el-icon>
          新建值班表
        </el-button>
      </div>
    </div>

    <div class="duty-layout">
      <aside class="table-list">
        <div
          v-for="table in dutyTables"
          :key="table.id"
          role="button"
          tabindex="0"
          class="duty-table-card"
          :class="{ active: selectedDutyTableId === table.id }"
          @click="selectDutyTable(table)"
          @keydown.enter="selectDutyTable(table)"
        >
          <div class="card-main">
            <strong>{{ table.name }}</strong>
            <span>{{ table.managerUsername || '未设置负责人' }}</span>
          </div>
          <div class="today-users">
            <el-tag v-if="!table.currentDutyUsers?.length" size="small" effect="light">今日暂无</el-tag>
            <el-tag v-for="user in table.currentDutyUsers || []" :key="`${table.id}-${user.username}`" size="small" effect="light">
              {{ user.realName || user.username }}
            </el-tag>
          </div>
          <div class="card-actions">
            <el-button link class="icon-btn" @click.stop="handleEditTable(table)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button link class="icon-btn danger" @click.stop="handleDeleteTable(table)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </aside>

      <main class="schedule-panel">
        <div class="schedule-toolbar">
          <div>
            <h3>{{ selectedDutyTable?.name || '请选择值班表' }}</h3>
            <p>{{ selectedDutyTable?.description || '按日期维护值班用户' }}</p>
          </div>
          <div class="schedule-actions">
            <el-date-picker v-model="selectedMonth" class="month-picker" type="month" value-format="YYYY-MM" format="YYYY-MM" :clearable="false" @change="loadSchedules" />
            <el-button :disabled="!selectedDutyTableId || !schedules.length" :loading="submittingReset" @click="handleResetSchedules">
              <el-icon><RefreshLeft /></el-icon>
              重置本月排班
            </el-button>
            <el-button type="primary" :disabled="!selectedDutyTableId" @click="openPublishDrawer">
              <el-icon><Calendar /></el-icon>
              发布日程
            </el-button>
          </div>
        </div>

        <div class="weekday-row">
          <span v-for="day in weekdays" :key="day">{{ day }}</span>
        </div>
        <div class="calendar-grid" v-loading="schedulesLoading">
          <button
            v-for="day in calendarDays"
            :key="day.key"
            type="button"
            class="day-cell"
            :class="{ muted: !day.inMonth, today: day.date === todayText }"
            :disabled="!selectedDutyTableId"
            @click="openSchedule(day.date)"
          >
            <span class="day-number">{{ day.label }}</span>
            <div class="day-users">
              <template v-if="getScheduleUsers(day.date).length">
                <em v-for="user in getScheduleUsers(day.date)" :key="`${day.date}-${user.username}`">{{ user.realName || user.username }}</em>
              </template>
              <small v-else>未排班</small>
            </div>
          </button>
        </div>
      </main>
    </div>

    <el-dialog v-model="tableDialogVisible" :title="tableDialogTitle" width="560px" class="monitor-dialog">
      <el-form ref="tableFormRef" :model="tableForm" :rules="tableRules" label-width="96px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="tableForm.name" placeholder="例如：SRE 一线值班" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-select v-model="tableForm.managerUsername" clearable filterable placeholder="选择负责人" style="width: 100%">
            <el-option v-for="user in userOptions" :key="user.username" :label="userLabel(user)" :value="user.username" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="tableForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="tableForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tableDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submittingTable" @click="submitTable">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="publishDrawerVisible"
      title="发布日程"
      size="760px"
      class="monitor-drawer duty-publish-drawer"
      :close-on-click-modal="false"
    >
      <div class="publish-panel">
        <div class="publish-summary">
          <div>
            <span>值班表</span>
            <strong>{{ selectedDutyTable?.name || '-' }}</strong>
          </div>
          <div>
            <span>开始日期</span>
            <strong>{{ publishForm.startDate }}</strong>
          </div>
          <div>
            <span>生成日程</span>
            <strong>{{ publishPreview.length }} 天</strong>
          </div>
          <div>
            <span>覆盖范围</span>
            <strong class="range-text">{{ publishDateRangeText }}</strong>
          </div>
        </div>

        <div class="publish-section">
          <div class="section-head">
            <div>
              <strong>基础信息</strong>
              <span>选择排班开始日期，日程会按值班组持续时间跨月生成</span>
            </div>
          </div>
          <el-date-picker
            v-model="publishForm.startDate"
            class="publish-month-picker"
            type="date"
            value-format="YYYY-MM-DD"
            format="YYYY-MM-DD"
            :clearable="false"
          />
        </div>

        <div class="publish-section">
          <div class="section-head">
            <div>
              <strong>值班组</strong>
              <span>按组顺序轮转，组内人员会同时显示在日历中</span>
            </div>
            <el-button link type="primary" @click="addPublishGroup">
              <el-icon><Plus /></el-icon>
              添加值班组
            </el-button>
          </div>

          <div v-for="(group, index) in publishGroups" :key="group.key" class="publish-group-card">
            <div class="group-head">
              <div class="group-title">
                <span class="group-index">{{ index + 1 }}</span>
                <div>
                  <strong>值班组 {{ index + 1 }}</strong>
                  <small>{{ getPublishGroupSummary(group) }}</small>
                </div>
              </div>
              <el-button link class="icon-btn danger" :disabled="publishGroups.length === 1" @click="removePublishGroup(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>

            <div class="group-controls">
              <el-form-item label="组内人员">
                <el-select
                  v-model="group.usernames"
                  multiple
                  filterable
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="选择值班人员"
                  style="width: 100%"
                >
                  <el-option v-for="user in userOptions" :key="user.username" :label="userLabel(user)" :value="user.username" />
                </el-select>
              </el-form-item>
              <el-form-item label="持续">
                <div class="duration-control">
                  <el-input-number v-model="group.duration" :min="1" :max="getDurationMax(group)" controls-position="right" />
                  <el-select v-model="group.unit" class="duration-unit">
                    <el-option label="天" value="day" />
                    <el-option label="周" value="week" />
                  </el-select>
                </div>
              </el-form-item>
            </div>

            <div class="selected-users">
              <button
                v-for="user in getPublishGroupUsers(group)"
                :key="`${group.key}-${user.username}`"
                type="button"
                class="selected-user"
                @click="removePublishUser(group, user.username)"
              >
                <span class="user-avatar">{{ getAvatarText(user) }}</span>
                <span>
                  <strong>{{ user.realName || user.username }}</strong>
                  <small>{{ user.username }}</small>
                </span>
                <el-icon><Close /></el-icon>
              </button>
              <div v-if="!getPublishGroupUsers(group).length" class="empty-users">
                <el-icon><UserFilled /></el-icon>
                请选择这个值班组的人员
              </div>
            </div>
          </div>
        </div>

        <div class="publish-section">
          <div class="section-head">
            <div>
              <strong>日程预览</strong>
              <span>提交后会覆盖 {{ publishDateRangeText }} 内已有排班</span>
            </div>
            <el-tag effect="light">{{ publishCycleText }}</el-tag>
          </div>
          <div v-if="visiblePublishPreview.length" class="preview-list">
            <div v-for="item in visiblePublishPreview" :key="item.dutyDate" class="preview-row">
              <span>{{ item.dutyDate }}</span>
              <div>
                <em v-for="user in item.users" :key="`${item.dutyDate}-${user.username}`">{{ user.realName || user.username }}</em>
              </div>
            </div>
            <div v-if="hiddenPublishPreviewCount > 0" class="preview-more">
              还有 {{ hiddenPublishPreviewCount }} 天排班会一并发布
            </div>
          </div>
          <el-empty v-else :image-size="72" description="选择人员后生成预览" />
        </div>
      </div>
      <template #footer>
        <div class="publish-footer">
          <el-button @click="resetPublishDraft">
            <el-icon><RefreshLeft /></el-icon>
            重置编排
          </el-button>
          <div>
            <el-button @click="publishDrawerVisible = false">取消</el-button>
            <el-button type="primary" :loading="submittingPublish" @click="submitPublishedSchedule">发布到日历</el-button>
          </div>
        </div>
      </template>
    </el-drawer>

    <el-drawer v-model="scheduleDrawerVisible" :title="scheduleTitle" size="520px" class="monitor-drawer">
      <el-form label-width="96px">
        <el-form-item label="值班日期">
          <el-input :model-value="scheduleForm.dutyDate" disabled />
        </el-form-item>
        <el-form-item label="值班用户">
          <el-select v-model="scheduleForm.usernames" multiple filterable placeholder="选择值班用户" style="width: 100%">
            <el-option v-for="user in userOptions" :key="user.username" :label="userLabel(user)" :value="user.username" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="scheduleForm.active" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="scheduleDrawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submittingSchedule" @click="submitSchedule">保存排班</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Calendar, Close, Delete, Edit, Plus, Refresh, RefreshLeft, Search, UserFilled } from '@element-plus/icons-vue'
import { getUserList } from '@/api/user'
import {
  createMonitorDutyTable,
  deleteMonitorDutyTable,
  deleteMonitorDutySchedule,
  getMonitorDutySchedules,
  getMonitorDutyTables,
  updateMonitorDutyTable,
  upsertMonitorDutySchedule,
  type MonitorDutySchedule,
  type MonitorDutyTable,
  type MonitorDutyUser
} from '@/api/monitor-datasource'

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
void tableHeaderStyle

const loading = ref(false)
const schedulesLoading = ref(false)
const submittingTable = ref(false)
const submittingSchedule = ref(false)
const submittingPublish = ref(false)
const submittingReset = ref(false)
const tableDialogVisible = ref(false)
const scheduleDrawerVisible = ref(false)
const publishDrawerVisible = ref(false)
const tableDialogTitle = ref('新建值班表')
const keyword = ref('')
const selectedMonth = ref(formatMonth(new Date()))
const selectedDutyTableId = ref<number>()
const dutyTables = ref<MonitorDutyTable[]>([])
const schedules = ref<MonitorDutySchedule[]>([])
const userOptions = ref<MonitorDutyUser[]>([])
const tableFormRef = ref<FormInstance>()

const tableForm = reactive<MonitorDutyTable>({
  name: '',
  description: '',
  managerUsername: '',
  enabled: true
})

const scheduleForm = reactive({
  dutyDate: '',
  usernames: [] as string[],
  active: true
})

type RotationUnit = 'day' | 'week'

interface PublishGroup {
  key: number
  usernames: string[]
  duration: number
  unit: RotationUnit
}

interface PublishPreviewItem {
  dutyDate: string
  users: MonitorDutyUser[]
  groupIndex: number
}

let publishGroupSeed = 1

const publishForm = reactive({
  startDate: getMonthStartDate(selectedMonth.value)
})
const publishGroups = ref<PublishGroup[]>([])

const tableRules: FormRules = {
  name: [{ required: true, message: '请输入值班表名称', trigger: 'blur' }]
}

const weekdays = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
const todayText = formatDate(new Date())
const selectedDutyTable = computed(() => dutyTables.value.find(item => item.id === selectedDutyTableId.value))
const scheduleTitle = computed(() => `${scheduleForm.dutyDate} 排班`)

const calendarDays = computed(() => buildCalendarDays(selectedMonth.value))
const publishPreview = computed(() => buildPublishPreview(publishForm.startDate, publishGroups.value))
const visiblePublishPreview = computed(() => publishPreview.value.slice(0, 24))
const hiddenPublishPreviewCount = computed(() => Math.max(0, publishPreview.value.length - visiblePublishPreview.value.length))
const publishDateRangeText = computed(() => {
  const records = publishPreview.value
  if (!records.length) return '-'
  return `${records[0].dutyDate} 至 ${records[records.length - 1].dutyDate}`
})
const publishCycleText = computed(() => {
  const days = publishPreview.value.length
  if (!days) return '未生成'
  if (days % 7 === 0) return `${days} 天 / ${days / 7} 周`
  return `${days} 天 / ${Math.floor(days / 7)} 周 ${days % 7} 天`
})

const loadUsers = async () => {
  const users = await getUserList({ page: 1, pageSize: 500 })
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

const loadDutyTables = async () => {
  loading.value = true
  try {
    dutyTables.value = await getMonitorDutyTables({ keyword: keyword.value.trim() }) || []
    if (!selectedDutyTableId.value && dutyTables.value[0]?.id) {
      selectedDutyTableId.value = dutyTables.value[0].id
    }
  } finally {
    loading.value = false
  }
}

const loadSchedules = async () => {
  if (!selectedDutyTableId.value) {
    schedules.value = []
    return
  }
  schedulesLoading.value = true
  try {
    schedules.value = await getMonitorDutySchedules({
      dutyTableId: selectedDutyTableId.value,
      month: selectedMonth.value
    }) || []
  } finally {
    schedulesLoading.value = false
  }
}

const loadAll = async () => {
  await Promise.all([loadUsers(), loadDutyTables()])
  await loadSchedules()
}

const selectDutyTable = async (table: MonitorDutyTable) => {
  selectedDutyTableId.value = table.id
  await loadSchedules()
}

const handleAddTable = () => {
  resetTableForm()
  tableDialogTitle.value = '新建值班表'
  tableDialogVisible.value = true
}

const handleEditTable = (table: MonitorDutyTable) => {
  resetTableForm()
  Object.assign(tableForm, { ...table })
  tableForm.enabled = table.enabled ?? true
  tableDialogTitle.value = '编辑值班表'
  tableDialogVisible.value = true
}

const handleDeleteTable = async (table: MonitorDutyTable) => {
  if (!table.id) return
  await ElMessageBox.confirm(`确定删除值班表「${table.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorDutyTable(table.id)
  ElMessage.success('删除成功')
  if (selectedDutyTableId.value === table.id) selectedDutyTableId.value = undefined
  await loadAll()
}

const submitTable = async () => {
  await tableFormRef.value?.validate()
  submittingTable.value = true
  try {
    const manager = userOptions.value.find(user => user.username === tableForm.managerUsername)
    const payload = {
      ...tableForm,
      managerUserId: manager?.id || 0
    }
    if (tableForm.id) {
      await updateMonitorDutyTable(tableForm.id, payload)
      ElMessage.success('更新成功')
    } else {
      await createMonitorDutyTable(payload)
      ElMessage.success('创建成功')
    }
    tableDialogVisible.value = false
    await loadAll()
  } finally {
    submittingTable.value = false
  }
}

const openSchedule = (date: string) => {
  if (!selectedDutyTableId.value) return
  const users = getScheduleUsers(date)
  scheduleForm.dutyDate = date
  scheduleForm.usernames = users.map(user => user.username)
  scheduleForm.active = getSchedule(date)?.status !== 'disabled'
  scheduleDrawerVisible.value = true
}

const submitSchedule = async () => {
  if (!selectedDutyTableId.value) return
  submittingSchedule.value = true
  try {
    const users = buildDutyUsers(scheduleForm.usernames)
    await upsertMonitorDutySchedule({
      dutyTableId: selectedDutyTableId.value,
      dutyDate: scheduleForm.dutyDate,
      users: JSON.stringify(users),
      status: scheduleForm.active ? 'active' : 'disabled'
    })
    ElMessage.success('排班已保存')
    scheduleDrawerVisible.value = false
    await Promise.all([loadDutyTables(), loadSchedules()])
  } finally {
    submittingSchedule.value = false
  }
}

const openPublishDrawer = () => {
  if (!selectedDutyTableId.value) {
    ElMessage.warning('请先选择值班表')
    return
  }
  resetPublishDraft()
  publishDrawerVisible.value = true
}

const resetPublishDraft = () => {
  publishForm.startDate = getMonthStartDate(selectedMonth.value)
  publishGroups.value = [createPublishGroupFromCurrentUsers()]
}

const createPublishGroupFromCurrentUsers = (): PublishGroup => {
  const currentUsers = selectedDutyTable.value?.currentDutyUsers?.map(user => user.username).filter(Boolean) || []
  return {
    key: publishGroupSeed++,
    usernames: currentUsers,
    duration: 1,
    unit: 'day'
  }
}

const addPublishGroup = () => {
  publishGroups.value.push({
    key: publishGroupSeed++,
    usernames: [],
    duration: 1,
    unit: 'day'
  })
}

const removePublishGroup = (index: number) => {
  if (publishGroups.value.length <= 1) return
  publishGroups.value.splice(index, 1)
}

const removePublishUser = (group: PublishGroup, username: string) => {
  group.usernames = group.usernames.filter(item => item !== username)
}

const handleResetSchedules = async () => {
  if (!selectedDutyTableId.value) return
  const resetItems = schedules.value.filter(item => item.id)
  if (!resetItems.length) {
    ElMessage.info('当前月份暂无可重置排班')
    return
  }
  await ElMessageBox.confirm(`确定清空「${selectedDutyTable.value?.name || '当前值班表'}」${selectedMonth.value} 的 ${resetItems.length} 条排班吗？`, '重置排班', {
    type: 'warning',
    confirmButtonText: '确认重置',
    cancelButtonText: '取消'
  })
  submittingReset.value = true
  try {
    await Promise.all(resetItems.map(item => deleteMonitorDutySchedule(item.id!)))
    ElMessage.success(`已重置 ${selectedMonth.value} 排班`)
    await Promise.all([loadDutyTables(), loadSchedules()])
  } finally {
    submittingReset.value = false
  }
}

const submitPublishedSchedule = async () => {
  if (!selectedDutyTableId.value) return
  const records = publishPreview.value
  if (!records.length) {
    ElMessage.warning('请至少添加一个包含人员的值班组')
    return
  }
  await ElMessageBox.confirm(`发布后将覆盖 ${publishDateRangeText.value} 已有排班，确定继续吗？`, '发布日程', {
    type: 'warning',
    confirmButtonText: '确认发布',
    cancelButtonText: '取消'
  })
  submittingPublish.value = true
  try {
    await Promise.all(records.map(item => upsertMonitorDutySchedule({
      dutyTableId: selectedDutyTableId.value!,
      dutyDate: item.dutyDate,
      users: JSON.stringify(item.users),
      status: 'active'
    })))
    selectedMonth.value = getMonthFromDate(records[0].dutyDate)
    ElMessage.success(`已发布 ${records.length} 天排班：${publishDateRangeText.value}`)
    publishDrawerVisible.value = false
    await Promise.all([loadDutyTables(), loadSchedules()])
  } finally {
    submittingPublish.value = false
  }
}

const getSchedule = (date: string) => schedules.value.find(item => item.dutyDate === date)
const getScheduleUsers = (date: string) => {
  const schedule = getSchedule(date)
  if (!schedule || schedule.status === 'disabled') return []
  return safeParse<MonitorDutyUser[]>(schedule.users, [])
}

const resetTableForm = () => {
  Object.assign(tableForm, {
    id: undefined,
    name: '',
    description: '',
    managerUserId: 0,
    managerUsername: '',
    enabled: true
  })
  tableFormRef.value?.clearValidate()
}

const safeParse = <T,>(raw: string | undefined, fallback: T): T => {
  if (!raw) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

const getPublishGroupUsers = (group: PublishGroup) => buildDutyUsers(group.usernames)
const getPublishGroupSummary = (group: PublishGroup) => {
  const names = getPublishGroupUsers(group).map(user => user.realName || user.username)
  const durationText = group.unit === 'week' ? `${group.duration || 1} 周` : `${group.duration || 1} 天`
  return names.length ? `${names.join('、')} · 持续 ${durationText}` : `未选择人员 · 持续 ${durationText}`
}
const getAvatarText = (user: MonitorDutyUser) => (user.realName || user.username || '?').slice(0, 1).toUpperCase()
const getDurationMax = (group: PublishGroup) => group.unit === 'week' ? 104 : 730

const buildDutyUsers = (usernames: string[]) => usernames.map(username => {
  const user = userOptions.value.find(item => item.username === username)
  return {
    id: user?.id,
    username,
    realName: user?.realName,
    email: user?.email,
    phone: user?.phone,
    notifyUserId: user?.notifyUserId,
    feishuUserId: user?.feishuUserId,
    feishuOpenId: user?.feishuOpenId,
    dingtalkUserId: user?.dingtalkUserId,
    wecomUserId: user?.wecomUserId
  }
}).filter((user): user is MonitorDutyUser => Boolean(user.username))

const userLabel = (user: MonitorDutyUser) => `${user.realName || user.username}${user.email ? ` / ${user.email}` : ''}`

function buildPublishPreview(startDate: string, groups: PublishGroup[]): PublishPreviewItem[] {
  const activeGroups = groups
    .map((group, index) => ({
      ...group,
      index,
      users: buildDutyUsers(group.usernames),
      durationDays: getGroupDurationDays(group)
    }))
    .filter(group => group.users.length && group.durationDays > 0)
  if (!activeGroups.length) return []

  const totalDays = activeGroups.reduce((sum, group) => sum + group.durationDays, 0)
  const dates = getDatesFromStartDate(startDate, totalDays)
  const preview: PublishPreviewItem[] = []
  let groupCursor = 0
  let remainDays = activeGroups[groupCursor].durationDays

  dates.forEach(dutyDate => {
    const group = activeGroups[groupCursor]
    preview.push({
      dutyDate,
      users: group.users,
      groupIndex: group.index
    })
    remainDays -= 1
    if (remainDays <= 0) {
      groupCursor = (groupCursor + 1) % activeGroups.length
      remainDays = activeGroups[groupCursor].durationDays
    }
  })

  return preview
}

function getGroupDurationDays(group: PublishGroup) {
  const value = Number(group.duration) || 1
  const boundedValue = Math.min(getDurationMax(group), Math.max(1, value))
  return boundedValue * (group.unit === 'week' ? 7 : 1)
}

function buildCalendarDays(month: string) {
  const [year, monthText] = month.split('-').map(Number)
  const first = new Date(year, monthText - 1, 1)
  const start = new Date(first)
  const day = first.getDay() || 7
  start.setDate(first.getDate() - day + 1)
  return Array.from({ length: 42 }).map((_, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    return {
      key: `${month}-${index}`,
      date: formatDate(date),
      label: String(date.getDate()),
      inMonth: date.getMonth() === monthText - 1
    }
  })
}

function getDatesFromStartDate(startDate: string, totalDays: number) {
  const [year, monthText, dayText] = startDate.split('-').map(Number)
  if (!year || !monthText || !dayText || totalDays <= 0) return []
  const start = new Date(year, monthText - 1, dayText)
  return Array.from({ length: totalDays }).map((_, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    return formatDate(date)
  })
}

function formatMonth(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}`
}

function getMonthStartDate(month: string) {
  return `${month}-01`
}

function getMonthFromDate(date: string) {
  return /^\d{4}-\d{2}-\d{2}$/.test(date) ? date.slice(0, 7) : selectedMonth.value
}

function formatDate(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

onMounted(loadAll)
</script>

<style scoped>
.duty-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-head,
.table-list,
.schedule-panel {
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

.page-head h2,
.schedule-toolbar h3 {
  margin: 0;
  color: #111827;
  font-weight: 750;
}

.page-head h2 {
  font-size: 21px;
}

.schedule-toolbar h3 {
  font-size: 17px;
}

.page-head p,
.schedule-toolbar p {
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

.duty-layout {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 12px;
}

.table-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
}

.duty-table-card {
  position: relative;
  display: block;
  width: 100%;
  min-height: 112px;
  padding: 12px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.duty-table-card:hover,
.duty-table-card.active {
  border-color: #bfdbfe;
  background: #f8fbff;
}

.card-main strong,
.card-main span {
  display: block;
}

.card-main strong {
  color: #111827;
  font-size: 14px;
}

.card-main span {
  margin-top: 5px;
  color: #667085;
  font-size: 12px;
}

.today-users {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 12px;
  padding-right: 72px;
}

.card-actions {
  position: absolute;
  right: 8px;
  bottom: 8px;
  display: flex;
  gap: 2px;
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

.schedule-panel {
  min-width: 0;
  padding: 14px;
}

.schedule-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}

.schedule-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.month-picker {
  width: 160px;
}

.weekday-row,
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
}

.weekday-row {
  border: 1px solid #e5e9f2;
  border-bottom: 0;
  border-radius: 8px 8px 0 0;
  overflow: hidden;
}

.weekday-row span {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid #edf1f7;
  background: #fbfcfe;
  color: #667085;
  font-size: 12px;
  font-weight: 650;
}

.weekday-row span:last-child {
  border-right: 0;
}

.calendar-grid {
  border-left: 1px solid #e5e9f2;
  border-top: 1px solid #e5e9f2;
}

.day-cell {
  min-height: 118px;
  padding: 10px;
  border: 0;
  border-right: 1px solid #e5e9f2;
  border-bottom: 1px solid #e5e9f2;
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.day-cell:hover {
  background: #f8fbff;
}

.day-cell.muted {
  background: #fbfcfe;
}

.day-cell.today {
  box-shadow: inset 0 0 0 2px #2563eb;
}

.day-number {
  color: #111827;
  font-size: 13px;
  font-weight: 750;
}

.day-cell.muted .day-number {
  color: #98a2b3;
}

.day-users {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-top: 10px;
}

.day-users em,
.day-users small {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-style: normal;
  font-size: 12px;
}

.day-users em {
  width: fit-content;
  max-width: 100%;
  padding: 2px 7px;
  border: 1px solid #dbe8ff;
  border-radius: 999px;
  background: #eef4ff;
  color: #1d4ed8;
}

.day-users small {
  color: #98a2b3;
}

.publish-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.publish-summary {
  display: grid;
  grid-template-columns: minmax(130px, .9fr) minmax(130px, .8fr) minmax(110px, .65fr) minmax(220px, 1.5fr);
  gap: 10px;
}

.publish-summary > div {
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.publish-summary span,
.section-head span,
.group-title small,
.selected-user small {
  display: block;
  color: #667085;
  font-size: 12px;
}

.publish-summary strong {
  display: block;
  min-width: 0;
  margin-top: 5px;
  overflow: hidden;
  color: #111827;
  font-size: 15px;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.publish-summary .range-text {
  overflow: visible;
  line-height: 1.35;
  text-overflow: clip;
  white-space: normal;
  word-break: keep-all;
}

.publish-section {
  padding: 14px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.section-head strong {
  color: #111827;
  font-size: 14px;
  font-weight: 760;
}

.publish-month-picker {
  width: 220px;
}

.publish-group-card {
  padding: 14px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcfe;
}

.publish-group-card + .publish-group-card {
  margin-top: 12px;
}

.group-head,
.group-title,
.duration-control,
.selected-user,
.preview-row {
  display: flex;
  align-items: center;
}

.group-head {
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.group-title {
  min-width: 0;
  gap: 10px;
}

.group-title > div {
  min-width: 0;
}

.group-title strong,
.selected-user strong {
  display: block;
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  font-weight: 720;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-title small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-index {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid #c7d7fe;
  border-radius: 8px;
  background: #eef4ff;
  color: #1d4ed8;
  font-weight: 760;
}

.group-controls {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 230px;
  gap: 12px;
}

.duration-control {
  width: 100%;
  gap: 8px;
}

.duration-control :deep(.el-input-number) {
  width: 118px;
}

.duration-unit {
  width: 86px;
}

.selected-users {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.selected-user {
  min-width: 0;
  gap: 9px;
  padding: 8px 10px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  text-align: left;
  cursor: pointer;
}

.selected-user:hover {
  border-color: #bfdbfe;
  background: #f8fbff;
}

.selected-user > span:nth-child(2) {
  min-width: 0;
  flex: 1;
}

.selected-user .el-icon {
  flex: 0 0 auto;
  color: #98a2b3;
}

.user-avatar {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: #eef4ff;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 760;
}

.empty-users {
  display: flex;
  align-items: center;
  gap: 8px;
  grid-column: 1 / -1;
  min-height: 44px;
  padding: 0 12px;
  border: 1px dashed #d6dce8;
  border-radius: 8px;
  background: #fff;
  color: #98a2b3;
  font-size: 13px;
}

.preview-list {
  display: grid;
  gap: 8px;
  max-height: 430px;
  overflow: auto;
  padding-right: 4px;
}

.preview-row {
  justify-content: space-between;
  gap: 12px;
  min-height: 42px;
  padding: 8px 10px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.preview-row > span {
  flex: 0 0 auto;
  color: #667085;
  font-size: 13px;
  font-weight: 650;
}

.preview-row > div {
  display: flex;
  min-width: 0;
  justify-content: flex-end;
  gap: 6px;
  flex-wrap: wrap;
}

.preview-row em {
  max-width: 120px;
  padding: 2px 7px;
  overflow: hidden;
  border: 1px solid #dbe8ff;
  border-radius: 999px;
  background: #eef4ff;
  color: #1d4ed8;
  font-style: normal;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-more {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 38px;
  border: 1px dashed #d6dce8;
  border-radius: 8px;
  background: #fbfcfe;
  color: #667085;
  font-size: 13px;
}

.publish-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

.publish-footer > div {
  display: flex;
  align-items: center;
  gap: 10px;
}

:global(.monitor-dialog .el-dialog__header),
:global(.monitor-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding-bottom: 14px;
  border-bottom: 1px solid #edf1f7;
  background: #fbfcfe;
}

:global(.monitor-drawer .el-drawer__body) {
  padding: 18px 22px;
}

:global(.monitor-drawer .el-drawer__footer),
:global(.monitor-dialog .el-dialog__footer) {
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
}

@media (max-width: 1120px) {
  .page-head,
  .schedule-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-search,
  .head-actions,
  .schedule-actions {
    width: 100%;
  }

  .schedule-actions {
    justify-content: flex-start;
  }

  .duty-layout {
    grid-template-columns: 1fr;
  }

  .table-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .table-list,
  .weekday-row,
  .calendar-grid {
    grid-template-columns: 1fr;
  }

  .day-cell {
    min-height: 92px;
  }

  .publish-summary,
  .group-controls,
  .selected-users {
    grid-template-columns: 1fr;
  }

  .publish-month-picker,
  .month-picker {
    width: 100%;
  }
}
</style>
