<template>
  <div class="dns-provider-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">DNS验证配置</h2>
          <p class="page-subtitle">配置DNS服务商API凭证，用于ACME证书申请时的域名所有权验证（DNS-01验证）</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增配置
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
        <el-table-column label="名称" prop="name" min-width="120" />

        <el-table-column label="服务商" width="130" align="center">
          <template #default="{ row }">
            <el-tag>{{ getProviderName(row.provider) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="邮箱" prop="email" min-width="150" />

        <el-table-column label="电话" prop="phone" width="130" />

        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.enabled" type="success">启用</el-tag>
            <el-tag v-else type="info">禁用</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="连接测试" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.last_test_ok" type="success" size="small">正常</el-tag>
            <el-tag v-else-if="row.last_test_at" type="danger" size="small">失败</el-tag>
            <el-tag v-else type="info" size="small">未测试</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="上次测试" width="170">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.last_test_at) || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="测试连接" placement="top">
                <el-button link class="action-btn action-test" @click="handleTest(row)" :loading="row.testing">
                  <el-icon><Connection /></el-icon>
                </el-button>
              </el-tooltip>
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="620px"
      :close-on-click-modal="false"
      class="beauty-dialog"
      destroy-on-close
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="130px" class="beauty-form">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入配置名称" />
        </el-form-item>

        <el-form-item label="DNS服务商" prop="provider">
          <el-select v-model="form.provider" placeholder="请选择DNS服务商" style="width: 100%" :disabled="!!form.id">
            <el-option label="阿里云DNS" value="aliyun" />
          </el-select>
        </el-form-item>

        <!-- 阿里云配置 -->
        <template v-if="form.provider === 'aliyun'">
          <el-form-item label="AccessKey ID" prop="config.access_key_id">
            <el-input v-model="form.config.access_key_id" placeholder="请输入AccessKey ID" />
          </el-form-item>
          <el-form-item label="AccessKey Secret" prop="config.access_key_secret">
            <el-input v-model="form.config.access_key_secret" type="password" placeholder="请输入AccessKey Secret" show-password />
          </el-form-item>
        </template>

        <el-divider />

        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱地址" />
        </el-form-item>

        <el-form-item label="电话" prop="phone">
          <el-input v-model="form.phone" placeholder="请输入电话号码" />
        </el-form-item>

        <el-form-item label="状态" prop="enabled">
          <el-select v-model="form.enabled" style="width: 100%">
            <el-option label="启用" :value="true" />
            <el-option label="禁用" :value="false" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import { Plus, Refresh, Edit, Delete, Connection } from '@element-plus/icons-vue'
import {
  getDNSProviders,
  getDNSProviderDetail,
  createDNSProvider,
  updateDNSProvider,
  deleteDNSProvider,
  testDNSProvider
} from '../api/ssl-cert'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const formRef = ref<FormInstance>()

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 表单数据
const form = reactive({
  id: 0,
  name: '',
  provider: '',
  config: {} as Record<string, string>,
  email: '',
  phone: '',
  enabled: true
})

// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择DNS服务商', trigger: 'change' }],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  phone: [{ required: true, message: '请输入电话号码', trigger: 'blur' }]
}

// 获取服务商名称
const getProviderName = (provider: string) => {
  const names: Record<string, string> = {
    aliyun: '阿里云'
  }
  return names[provider] || provider
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getDNSProviders({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增DNS配置'
  Object.assign(form, {
    id: 0,
    name: '',
    provider: '',
    config: {},
    email: '',
    phone: '',
    enabled: true
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = async (row: any) => {
  dialogTitle.value = '编辑DNS配置'
  try {
    // 获取完整详情(包含配置)
    const detail = await getDNSProviderDetail(row.id)
    Object.assign(form, {
      id: detail.id,
      name: detail.name,
      provider: detail.provider,
      config: detail.config || {},
      email: detail.email || '',
      phone: detail.phone || '',
      enabled: detail.enabled
    })
    dialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

// 提交
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        if (form.id) {
          await updateDNSProvider(form.id, {
            name: form.name,
            config: form.config,
            email: form.email,
            phone: form.phone,
            enabled: form.enabled
          })
          ElMessage.success('保存成功')
          dialogVisible.value = false
          loadData()
        } else {
          await createDNSProvider({
            name: form.name,
            provider: form.provider,
            config: form.config,
            email: form.email,
            phone: form.phone,
            enabled: form.enabled
          })
          ElMessage.success('创建成功')
          dialogVisible.value = false
          loadData()
        }
      } catch (error: any) {
        // 错误已由 request 拦截器处理并显示
      } finally {
        submitting.value = false
      }
    }
  })
}

// 测试连接
const handleTest = async (row: any) => {
  try {
    row.testing = true
    await testDNSProvider(row.id)
    ElMessage.success('连接测试成功')
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    row.testing = false
  }
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该DNS配置吗？', '提示', { type: 'warning' })
    loading.value = true
    await deleteDNSProvider(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      // 错误已由 request 拦截器处理
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.dns-provider-container {
  padding: 0;
  background-color: transparent;
}

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
  background: #f8fafc;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #111827;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

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
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-test:hover {
  background-color: #e8f5e9;
  color: #67c23a;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.pagination-wrapper {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 6px;
  line-height: 1.5;
}

/* 弹窗美化 */
:deep(.beauty-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.beauty-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  margin-right: 0;
  border-bottom: 1px solid #f0f0f0;
  background: #fafbfc;
}

:deep(.beauty-dialog .el-dialog__title) {
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.beauty-dialog .el-dialog__headerbtn) {
  top: 20px;
  right: 20px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-dialog .el-dialog__headerbtn:hover) {
  background: #f0f0f0;
}

:deep(.beauty-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 65vh;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

:deep(.beauty-dialog .el-dialog__body::-webkit-scrollbar) {
  display: none;
}

:deep(.beauty-dialog .el-dialog__footer) {
  padding: 16px 24px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafbfc;
}

/* 表单美化 */
:deep(.beauty-form .el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.beauty-form .el-input__wrapper),
:deep(.beauty-form .el-textarea__inner) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-form .el-input__wrapper:hover),
:deep(.beauty-form .el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset;
}

:deep(.beauty-form .el-input__wrapper.is-focus),
:deep(.beauty-form .el-textarea__inner:focus) {
  box-shadow: 0 0 0 1px #000 inset;
}

:deep(.beauty-form .el-select .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #000 inset !important;
}

:deep(.beauty-form .el-divider__text) {
  font-size: 13px;
  font-weight: 600;
  color: #909399;
  background: #fff;
}
</style>
