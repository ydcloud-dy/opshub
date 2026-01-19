<template>
  <div class="alert-receivers-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h2 class="page-title">告警接收人</h2>
          <p class="page-subtitle">管理域名监控告警的接收人及其联系方式</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增接收人
        </el-button>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="tableData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="姓名" prop="name" min-width="150" />

        <el-table-column label="邮箱" prop="email" min-width="200">
          <template #default="{ row }">
            <span v-if="row.email">{{ row.email }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="手机号" prop="phone" width="130">
          <template #default="{ row }">
            <span v-if="row.phone">{{ row.phone }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="企业微信ID" prop="wechatId" width="130">
          <template #default="{ row }">
            <span v-if="row.wechatId">{{ row.wechatId }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="钉钉ID" prop="dingtalkId" width="130">
          <template #default="{ row }">
            <span v-if="row.dingtalkId">{{ row.dingtalkId }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="飞书ID" prop="feishuId" width="130">
          <template #default="{ row }">
            <span v-if="row.feishuId">{{ row.feishuId }}</span>
            <span v-else style="color: #ccc;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="通知方式" width="150" align="center">
          <template #default="{ row }">
            <div class="notification-methods">
              <el-tooltip content="邮件" placement="top">
                <el-tag v-if="row.enableEmail" size="small" type="success">邮件</el-tag>
              </el-tooltip>
              <el-tooltip content="企业微信" placement="top">
                <el-tag v-if="row.enableWeChat" size="small" type="warning">微信</el-tag>
              </el-tooltip>
              <el-tooltip content="钉钉" placement="top">
                <el-tag v-if="row.enableDingTalk" size="small" type="danger">钉钉</el-tag>
              </el-tooltip>
              <el-tooltip content="飞书" placement="top">
                <el-tag v-if="row.enableFeishu" size="small" type="info">飞书</el-tag>
              </el-tooltip>
              <el-tooltip content="系统消息" placement="top">
                <el-tag v-if="row.enableSystemMsg" size="small">系统</el-tag>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="700px"
      class="receiver-edit-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-divider content-position="left">基本信息</el-divider>

        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入接收人姓名" />
        </el-form-item>

        <el-divider content-position="left">联系方式</el-divider>

        <el-form-item label="邮箱地址">
          <el-input v-model="form.email" placeholder="example@email.com" />
        </el-form-item>

        <el-form-item label="手机号码">
          <el-input v-model="form.phone" placeholder="13800138000" />
        </el-form-item>

        <el-form-item label="企业微信ID">
          <el-input v-model="form.wechatId" placeholder="企业微信User ID" />
        </el-form-item>

        <el-form-item label="钉钉ID">
          <el-input v-model="form.dingtalkId" placeholder="钉钉User ID" />
        </el-form-item>

        <el-form-item label="飞书ID">
          <el-input v-model="form.feishuId" placeholder="飞书User ID" />
        </el-form-item>

        <el-divider content-position="left">通知方式</el-divider>

        <el-form-item label="启用的通知">
          <el-checkbox-group v-model="selectedNotifications">
            <el-checkbox label="enableEmail">邮件通知</el-checkbox>
            <el-checkbox label="enableWebhook">Webhook通知</el-checkbox>
            <el-checkbox label="enableWeChat">企业微信</el-checkbox>
            <el-checkbox label="enableDingTalk">钉钉通知</el-checkbox>
            <el-checkbox label="enableFeishu">飞书通知</el-checkbox>
            <el-checkbox label="enableSystemMsg">系统消息</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-alert
          title="提示"
          type="info"
          :closable="false"
          show-icon
          style="margin-top: 16px;"
        >
          至少需要选择一种通知方式，并确保已配置相应的联系方式。
        </el-alert>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import {
  Plus,
  Refresh,
  Edit,
  Delete,
  User
} from '@element-plus/icons-vue'
import {
  getAlertReceivers,
  createAlertReceiver,
  updateAlertReceiver,
  deleteAlertReceiver,
  type AlertReceiver
} from '@/api/alert-config'

const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const submitting = ref(false)
const formRef = ref<FormInstance>()

// 表格数据
const tableData = ref<any[]>([])

// 选中的通知方式
const selectedNotifications = ref<string[]>(['enableEmail', enableWeChat'])

// 表单数据
const form = reactive<AlertReceiver>({
  name: '',
  email: '',
  phone: '',
  wechatId: '',
  dingtalkId: '',
  feishuId: '',
  userId: undefined,
  enableEmail: true,
  enableWebhook: false,
  enableWeChat: true,
  enableDingTalk: false,
  enableFeishu: false,
  enableSystemMsg: true
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入接收人姓名', trigger: 'blur' }]
}

// 根据选中的通知方式更新表单
const updateFormFromSelection = () => {
  form.enableEmail = selectedNotifications.value.includes('enableEmail')
  form.enableWebhook = selectedNotifications.value.includes('enableWebhook')
  form.enableWeChat = selectedNotifications.value.includes('enableWeChat')
  form.enableDingTalk = selectedNotifications.value.includes('enableDingTalk')
  form.enableFeishu = selectedNotifications.value.includes('enableFeishu')
  form.enableSystemMsg = selectedNotifications.value.includes('enableSystemMsg')
}

// 根据表单数据更新选中状态
const updateSelectionFromForm = () => {
  selectedNotifications.value = []
  if (form.enableEmail) selectedNotifications.value.push('enableEmail')
  if (form.enableWebhook) selectedNotifications.value.push('enableWebhook')
  if (form.enableWeChat) selectedNotifications.value.push('enableWeChat')
  if (form.enableDingTalk) selectedNotifications.value.push('enableDingTalk')
  if (form.enableFeishu) selectedNotifications.value.push('enableFeishu')
  if (form.enableSystemMsg) selectedNotifications.value.push('enableSystemMsg')
}

// 监听选中状态变化
import { watch } from 'vue'
watch(selectedNotifications, () => {
  updateFormFromSelection()
})

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return '-'
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const data = await getAlertReceivers()
    tableData.value = data || []
  } catch (error: any) {
    ElMessage.error('加载数据失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增告警接收人'
  Object.assign(form, {
    id: undefined,
    name: '',
    email: '',
    phone: '',
    wechatId: '',
    dingtalkId: '',
    feishuId: '',
    userId: undefined,
    enableEmail: true,
    enableWebhook: false,
    enableWeChat: true,
    enableDingTalk: false,
    enableFeishu: false,
    enableSystemMsg: true
  })
  selectedNotifications.value = ['enableEmail', 'enableWeChat', 'enableSystemMsg']
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogTitle.value = '编辑告警接收人'
  Object.assign(form, {
    id: row.id,
    name: row.name,
    email: row.email || '',
    phone: row.phone || '',
    wechatId: row.wechatId || '',
    dingtalkId: row.dingtalkId || '',
    feishuId: row.feishuId || '',
    userId: row.userId,
    enableEmail: row.enableEmail,
    enableWebhook: row.enableWebhook,
    enableWeChat: row.enableWeChat,
    enableDingTalk: row.enableDingTalk,
    enableFeishu: row.enableFeishu,
    enableSystemMsg: row.enableSystemMsg
  })
  updateSelectionFromForm()
  dialogVisible.value = true
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该告警接收人吗？', '提示', { type: 'warning' })
    loading.value = true
    await deleteAlertReceiver(row.id)
    ElMessage.success('删除成功')
    await loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
      console.error(error)
    }
  } finally {
    loading.value = false
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      // 确保至少选择一种通知方式
      updateFormFromSelection()
      if (selectedNotifications.value.length === 0) {
        ElMessage.warning('请至少选择一种通知方式')
        return
      }

      submitting.value = true
      try {
        if (form.id) {
          await updateAlertReceiver(form.id, form)
          ElMessage.success('更新成功')
        } else {
          await createAlertReceiver(form)
          ElMessage.success('创建成功')
        }
        dialogVisible.value = false
        await loadData()
      } catch (error: any) {
        ElMessage.error(error.response?.data?.message || '操作失败')
        console.error(error)
      } finally {
        submitting.value = false
      }
    }
  })
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.alert-receivers-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.notification-methods {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: center;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.receiver-edit-dialog) {
  border-radius: 12px;
}

:deep(.receiver-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.receiver-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.receiver-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}
</style>
