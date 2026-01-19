<template>
  <div class="alert-channels-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Bell /></el-icon>
        </div>
        <div>
          <h2 class="page-title">告警通道配置</h2>
          <p class="page-subtitle">配置邮件、企业微信、钉钉、飞书等告警通知通道</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增通道
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
        <el-table-column label="通道名称" prop="name" min-width="200" />

        <el-table-column label="通道类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="getChannelTypeColor(row.channelType)" effect="dark">
              {{ getChannelTypeName(row.channelType) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.enabled" type="success" effect="dark">启用</el-tag>
            <el-tag v-else type="info" effect="dark">禁用</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="配置预览" min-width="300">
          <template #default="{ row }">
            <div class="config-preview">{{ getConfigPreview(row) }}</div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="测试" placement="top">
                <el-button link class="action-btn action-check" @click="handleTest(row)">
                  <el-icon><Message /></el-icon>
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
      class="channel-edit-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px">
        <el-form-item label="通道名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入通道名称，如：企业微信通知" />
        </el-form-item>

        <el-form-item label="通道类型" prop="channelType">
          <el-select v-model="form.channelType" placeholder="请选择通道类型" style="width: 100%;">
            <el-option label="邮件通知" value="email" />
            <el-option label="Webhook" value="webhook" />
            <el-option label="企业微信" value="wechat" />
            <el-option label="钉钉" value="dingtalk" />
            <el-option label="飞书" value="feishu" />
          </el-select>
        </el-form-item>

        <el-form-item label="启用状态" prop="enabled">
          <el-switch v-model="form.enabled" />
        </el-form-item>

        <!-- 邮件配置 -->
        <template v-if="form.channelType === 'email'">
          <el-divider content-position="left">邮件服务器配置</el-divider>
          <el-form-item label="SMTP服务器" prop="config.smtpHost">
            <el-input v-model="form.config.smtpHost" placeholder="smtp.example.com" />
          </el-form-item>
          <el-form-item label="SMTP端口" prop="config.smtpPort">
            <el-input-number v-model="form.config.smtpPort" :min="1" :max="65535" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="发件人邮箱" prop="config.fromEmail">
            <el-input v-model="form.config.fromEmail" placeholder="noreply@example.com" />
          </el-form-item>
          <el-form-item label="发件人名称" prop="config.fromName">
            <el-input v-model="form.config.fromName" placeholder="OpsHub监控" />
          </el-form-item>
          <el-form-item label="SMTP用户名" prop="config.smtpUser">
            <el-input v-model="form.config.smtpUser" placeholder="user@example.com" />
          </el-form-item>
          <el-form-item label="SMTP密码" prop="config.smtpPassword">
            <el-input v-model="form.config.smtpPassword" type="password" placeholder="请输入SMTP密码" show-password />
          </el-form-item>
        </template>

        <!-- Webhook配置 -->
        <template v-if="form.channelType === 'webhook'">
          <el-divider content-position="left">Webhook配置</el-divider>
          <el-form-item label="Webhook URL" prop="config.webhookUrl">
            <el-input v-model="form.config.webhookUrl" placeholder="https://example.com/webhook" />
          </el-form-item>
        </template>

        <!-- 企业微信配置 -->
        <template v-if="form.channelType === 'wechat'">
          <el-divider content-position="left">企业微信配置</el-divider>
          <el-form-item label="Webhook URL" prop="config.wechatWebhook">
            <el-input v-model="form.config.wechatWebhook" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
          </el-form-item>
        </template>

        <!-- 钉钉配置 -->
        <template v-if="form.channelType === 'dingtalk'">
          <el-divider content-position="left">钉钉配置</el-divider>
          <el-form-item label="Webhook URL" prop="config.dingtalkWebhook">
            <el-input v-model="form.config.dingtalkWebhook" placeholder="https://oapi.dingtalk.com/robot/send?access_token=xxx" />
          </el-form-item>
          <el-form-item label="加签密钥" prop="config.dingtalkSecret">
            <el-input v-model="form.config.dingtalkSecret" type="password" placeholder="SEC开头的密钥（可选）" show-password />
          </el-form-item>
        </template>

        <!-- 飞书配置 -->
        <template v-if="form.channelType === 'feishu'">
          <el-divider content-position="left">飞书配置</el-divider>
          <el-form-item label="Webhook URL" prop="config.feishuWebhook">
            <el-input v-model="form.config.feishuWebhook" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" />
          </el-form-item>
        </template>
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import {
  Plus,
  Refresh,
  Edit,
  Delete,
  Message,
  Bell
} from '@element-plus/icons-vue'
import {
  getAlertChannels,
  createAlertChannel,
  updateAlertChannel,
  deleteAlertChannel,
  type AlertChannel
} from '@/api/alert-config'

const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const submitting = ref(false)
const formRef = ref<FormInstance>()

// 表格数据
const tableData = ref<any[]>([])

// 表单数据
const form = reactive({
  id: 0,
  name: '',
  channelType: 'email',
  enabled: true,
  config: {
    smtpHost: '',
    smtpPort: 465,
    fromEmail: '',
    fromName: '',
    smtpUser: '',
    smtpPassword: '',
    webhookUrl: '',
    wechatWebhook: '',
    dingtalkWebhook: '',
    dingtalkSecret: '',
    feishuWebhook: ''
  }
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入通道名称', trigger: 'blur' }],
  channelType: [{ required: true, message: '请选择通道类型', trigger: 'change' }]
}

// 获取通道类型颜色
const getChannelTypeColor = (type: string) => {
  const colorMap: Record<string, string> = {
    email: 'primary',
    webhook: 'success',
    wechat: 'warning',
    dingtalk: 'danger',
    feishu: 'info'
  }
  return colorMap[type] || ''
}

// 获取通道类型名称
const getChannelTypeName = (type: string) => {
  const nameMap: Record<string, string> = {
    email: '邮件',
    webhook: 'Webhook',
    wechat: '企业微信',
    dingtalk: '钉钉',
    feishu: '飞书'
  }
  return nameMap[type] || type
}

// 获取配置预览
const getConfigPreview = (row: any) => {
  try {
    const config = typeof row.config === 'string' ? JSON.parse(row.config) : row.config
    if (row.channelType === 'email') {
      return `SMTP: ${config.smtpHost}:${config.smtpPort} | 发件人: ${config.fromEmail}`
    } else if (row.channelType === 'webhook') {
      return `URL: ${config.webhookUrl || '-'}`
    } else if (row.channelType === 'wechat') {
      return `Webhook: ${config.wechatWebhook?.substring(0, 50) || '-'}...`
    } else if (row.channelType === 'dingtalk') {
      return `Webhook: ${config.dingtalkWebhook?.substring(0, 50) || '-'}...`
    } else if (row.channelType === 'feishu') {
      return `Webhook: ${config.feishuWebhook?.substring(0, 50) || '-'}...`
    }
    return '-'
  } catch {
    return row.config || '-'
  }
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return '-'
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const data = await getAlertChannels()
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
  dialogTitle.value = '新增告警通道'
  Object.assign(form, {
    id: 0,
    name: '',
    channelType: 'email',
    enabled: true,
    config: {
      smtpHost: '',
      smtpPort: 465,
      fromEmail: '',
      fromName: '',
      smtpUser: '',
      smtpPassword: '',
      webhookUrl: '',
      wechatWebhook: '',
      dingtalkWebhook: '',
      dingtalkSecret: '',
      feishuWebhook: ''
    }
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogTitle.value = '编辑告警通道'
  const config = typeof row.config === 'string' ? JSON.parse(row.config) : row.config
  Object.assign(form, {
    id: row.id,
    name: row.name,
    channelType: row.channelType,
    enabled: row.enabled,
    config: {
      smtpHost: config.smtpHost || '',
      smtpPort: config.smtpPort || 465,
      fromEmail: config.fromEmail || '',
      fromName: config.fromName || '',
      smtpUser: config.smtpUser || '',
      smtpPassword: config.smtpPassword || '',
      webhookUrl: config.webhookUrl || '',
      wechatWebhook: config.wechatWebhook || '',
      dingtalkWebhook: config.dingtalkWebhook || '',
      dingtalkSecret: config.dingtalkSecret || '',
      feishuWebhook: config.feishuWebhook || ''
    }
  })
  dialogVisible.value = true
}

// 测试
const handleTest = async (row: any) => {
  ElMessage.info('测试功能开发中...')
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该告警通道吗？', '提示', { type: 'warning' })
    loading.value = true
    await deleteAlertChannel(row.id)
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
      submitting.value = true
      try {
        const submitData = {
          name: form.name,
          channelType: form.channelType,
          enabled: form.enabled,
          config: JSON.stringify(form.config)
        }

        if (form.id) {
          await updateAlertChannel(form.id, submitData)
          ElMessage.success('更新成功')
        } else {
          await createAlertChannel(submitData)
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
.alert-channels-container {
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

.config-preview {
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.action-check:hover {
  background-color: #e8f5e9;
  color: #67c23a;
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

:deep(.channel-edit-dialog) {
  border-radius: 12px;
}

:deep(.channel-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.channel-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.channel-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}
</style>
