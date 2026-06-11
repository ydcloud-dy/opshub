<template>
  <div class="credentials-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">凭证管理</h2>
          <p class="page-subtitle">管理SSH认证凭证，支持密码和密钥两种认证方式</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新建凭证
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选栏 -->
    <div class="filter-bar">
      <div class="filter-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索凭证名称..."
          clearable
          class="filter-input"
          style="width: 300px;"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.type"
          placeholder="认证方式"
          clearable
          class="filter-input"
          @change="handleSearch"
        >
          <el-option label="密码认证" value="password" />
          <el-option label="密钥认证" value="key" />
        </el-select>
      </div>

      <div class="filter-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
        <el-button @click="loadCredentialList">
          <el-icon style="margin-right: 4px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 凭证列表 -->
    <div class="table-wrapper">
      <el-table
        :data="credentialList"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="凭证名称" prop="name" min-width="180">
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="credential-icon" :color="row.type === 'password' ? '#e6a23c' : '#67c23a'">
                <Key v-if="row.type === 'key'" />
                <Lock v-else />
              </el-icon>
              <span class="name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="认证方式" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.type === 'password' ? 'warning' : 'success'" size="small">
              {{ row.typeText }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="用户名" prop="username" min-width="120">
          <template #default="{ row }">
            <span v-if="row.username">{{ row.username }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="使用情况" min-width="150">
          <template #default="{ row }">
            <div class="usage-cell">
              <span class="usage-count">{{ row.hostCount || 0 }} 台主机</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.description">{{ row.description }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createTime" width="180" />

        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)" :disabled="row.hostCount > 0">
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
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadCredentialList"
          @current-change="loadCredentialList"
        />
      </div>
    </div>

    <!-- 新增/编辑凭证对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑凭证' : '新建凭证'"
      width="50%"
      class="credential-dialog responsive-dialog"
      @close="handleDialogClose"
    >
      <el-alert v-if="isEdit" type="info" :closable="false" style="margin-bottom: 20px;">
        <template #title>
          <span style="font-size: 13px;">出于安全考虑，私钥和密码不会在编辑时显示。如需修改，请重新填写。留空则保持原值不变。</span>
        </template>
      </el-alert>

      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="凭证名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入凭证名称，如：生产环境root凭证" />
        </el-form-item>

        <el-form-item label="认证方式" prop="type">
          <el-radio-group v-model="form.type" @change="handleAuthTypeChange">
            <el-radio label="password">密码认证</el-radio>
            <el-radio label="key">密钥认证</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="form.type === 'password'" label="用户名">
          <el-input v-model="form.username" placeholder="如：root" />
        </el-form-item>

        <el-form-item v-if="form.type === 'password'" label="密码" prop="password">
          <el-input v-model="form.password" type="password" :placeholder="isEdit ? '如需修改密码请在此填写，留空则保持不变' : '请输入密码'" show-password />
        </el-form-item>

        <el-form-item v-if="form.type === 'key'" label="用户名">
          <el-input v-model="form.username" placeholder="如：root（可选）" />
        </el-form-item>

        <el-form-item v-if="form.type === 'key'" label="私钥" prop="privateKey">
          <el-input
            v-model="form.privateKey"
            type="textarea"
            :rows="8"
            :placeholder="isEdit ? '如需修改私钥请在此填写，留空则保持不变' : '请粘贴PEM格式的私钥内容'"
          />
        </el-form-item>

        <el-form-item v-if="form.type === 'key'" label="私钥密码">
          <el-input v-model="form.passphrase" type="password" placeholder="如果私钥有密码请输入（可选）" show-password />
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="请输入备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">{{ isEdit ? '保存' : '确定' }}</el-button>
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
  Edit,
  Delete,
  Search,
  Refresh,
  RefreshLeft,
  Lock,
  Key
} from '@element-plus/icons-vue'
import {
  getCredentialList,
  getCredential,
  createCredential,
  updateCredential,
  deleteCredential
} from '@/api/host'

// 加载状态
const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const isEdit = ref(false)

// 表单引用
const formRef = ref<FormInstance>()

// 搜索表单
const searchForm = reactive({
  keyword: '',
  type: undefined as string | undefined
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 凭证列表
const credentialList = ref([])

// 表单
const form = reactive({
  id: 0,
  name: '',
  type: 'password',
  username: '',
  password: '',
  privateKey: '',
  passphrase: '',
  description: ''
})

// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入凭证名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  privateKey: [{ required: true, message: '请输入私钥', trigger: 'blur' }]
}

// 加载凭证列表
const loadCredentialList = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined
    }
    if (searchForm.type !== undefined) {
      params.type = searchForm.type
    }

    const res = await getCredentialList(params)
    credentialList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取凭证列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadCredentialList()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.type = undefined
  loadCredentialList()
}

// 新增凭证
const handleAdd = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    type: 'password',
    username: '',
    password: '',
    privateKey: '',
    passphrase: '',
    description: ''
  })
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑凭证
const handleEdit = async (row: any) => {

  // 获取完整的凭证信息（包括解密后的私钥）
  try {
    const credential = await getCredential(row.id)

    Object.assign(form, {
      id: credential.id,
      name: credential.name,
      type: credential.type,
      username: credential.username || '',
      password: credential.password || '',
      privateKey: credential.privateKey || '',
      passphrase: credential.passphrase || '',
      description: credential.description || ''
    })
  } catch (error) {
    ElMessage.error('获取凭证详情失败')
    return
  }

  isEdit.value = true
  dialogVisible.value = true
}

// 删除凭证
const handleDelete = (row: any) => {
  if (row.hostCount > 0) {
    ElMessage.warning('该凭证正在被使用，无法删除')
    return
  }

  ElMessageBox.confirm(`确定要删除凭证"${row.name}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteCredential(row.id)
      ElMessage.success('删除成功')
      loadCredentialList()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

// 认证方式变化
const handleAuthTypeChange = (type: string) => {
  // 清空对应字段
  if (type === 'password') {
    form.privateKey = ''
    form.passphrase = ''
  } else {
    form.password = ''
  }
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      if (isEdit.value) {
        // 编辑凭证时，如果不修改密码或私钥，需要从请求对象中删除这些字段
        const updateData: any = {
          id: form.id,
          name: form.name,
          type: form.type,
          username: form.username,
          description: form.description
        }
        // 只有当用户填写了密码或私钥时，才包含这些字段
        if (form.password) {
          updateData.password = form.password
        }
        if (form.privateKey) {
          updateData.privateKey = form.privateKey
        }
        if (form.passphrase) {
          updateData.passphrase = form.passphrase
        }
        await updateCredential(form.id, updateData)
        ElMessage.success('更新成功')
      } else {
        await createCredential(form)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadCredentialList()
    } catch (error: any) {
      ElMessage.error(error.message || '操作失败')
    } finally {
      submitting.value = false
    }
  })
}

onMounted(() => {
  loadCredentialList()
})
</script>

<style scoped>
.credentials-page-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
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
  flex-shrink: 0;
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

/* 筛选栏 */
.filter-bar {
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.filter-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.filter-input {
  width: 220px;
}

.filter-actions {
  display: flex;
  gap: 10px;
}

.filter-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  transition: all 0.3s ease;
}

.filter-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
}

.filter-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
}

.search-icon {
  color: #d4af37;
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

/* 表格 */
.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modern-table {
  flex: 1;
}

.modern-table :deep(.el-table__body-wrapper) {
  overflow-y: auto;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.credential-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.name {
  font-weight: 500;
  color: #303133;
}

.usage-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.usage-count {
  font-size: 13px;
  color: #606266;
}

.text-muted {
  color: #c0c4cc;
}

.pagination-wrapper {
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  justify-content: flex-end;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 14px;
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

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-btn:disabled:hover {
  transform: none;
  background-color: transparent;
  color: inherit;
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

:deep(.credential-dialog) {
  border-radius: 12px;
}

:deep(.credential-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.credential-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.credential-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

:deep(.responsive-dialog) {
  max-width: 1200px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }
}

/* 资产页统一视觉优化 */
.credentials-page-container {
  gap: 16px;
}

.page-header,
.filter-bar,
.table-wrapper {
  border: 1px solid #e5e9f2;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.page-header {
  align-items: center;
  gap: 16px;
  margin-bottom: 0;
  padding: 18px 20px;
  border-radius: 10px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 9px;
  color: #111827;
  background: rgba(255, 175, 53, 0.18);
}

.page-title {
  color: #111827;
  font-size: 22px;
  font-weight: 750;
}

.page-subtitle {
  color: #667085;
}

.header-actions,
.filter-actions,
.filter-inputs {
  flex-wrap: wrap;
}

.filter-bar {
  margin-bottom: 0;
  padding: 14px 16px;
  border-radius: 10px;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.04);
}

.filter-input {
  width: 240px;
}

.search-icon {
  color: #d97706;
}

.filter-bar :deep(.el-input__wrapper),
.filter-bar :deep(.el-select__wrapper) {
  border: 0;
  border-radius: 7px;
  box-shadow: 0 0 0 1px #e5e9f2 inset;
}

.filter-bar :deep(.el-input__wrapper:hover),
.filter-bar :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px #d8dee9 inset;
}

.filter-bar :deep(.el-input__wrapper.is-focus),
.filter-bar :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px #111827 inset, 0 0 0 3px rgba(255, 175, 53, 0.16);
}

.table-wrapper {
  border-radius: 10px;
}

.modern-table :deep(.el-table__header th) {
  background: #f8fafc !important;
  color: #475467 !important;
}

.modern-table :deep(.el-table__body td) {
  border-bottom-color: #edf1f7;
}

.name {
  color: #111827;
  font-weight: 650;
}

.usage-count,
.text-muted {
  color: #667085;
}

.black-button {
  height: 36px;
  padding: 0 14px;
  border-radius: 7px;
  box-shadow: 0 8px 18px rgba(17, 24, 39, 0.12);
}

.reset-btn {
  background: #fff;
}

.action-btn {
  width: 30px;
  height: 30px;
  border-radius: 7px;
}

.pagination-wrapper {
  min-height: 54px;
  border-top-color: #edf1f7;
}

:deep(.credential-dialog) {
  border-radius: 10px;
}

:deep(.credential-dialog .el-dialog__header) {
  background: #fbfcfe;
  border-bottom-color: #edf1f7;
}

@media (max-width: 900px) {
  .page-header,
  .filter-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .filter-input,
  .filter-inputs,
  .filter-actions {
    width: 100%;
  }
}
</style>
