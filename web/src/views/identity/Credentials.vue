<template>
  <div class="credentials-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">凭证管理</h2>
          <p class="page-subtitle">管理您在各应用中的账号密码，实现一键登录</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          添加凭证
        </el-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 提示信息 -->
      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="info-alert"
      >
        <template #title>
          凭证用于存储您在各应用中的账号密码，实现一键登录。凭证数据加密存储，仅您本人可见。
        </template>
      </el-alert>

      <!-- 凭证网格 -->
      <div class="credentials-grid" v-loading="loading">
        <div v-for="cred in credentialList" :key="cred.id" class="credential-card">
          <div class="credential-header">
            <div class="app-icon">
              <img v-if="cred.appIcon" :src="cred.appIcon" :alt="cred.appName" />
              <el-icon v-else :size="24"><Key /></el-icon>
            </div>
            <div class="app-info">
              <h4>{{ cred.appName || '未知应用' }}</h4>
              <span class="username">{{ cred.username }}</span>
            </div>
          </div>
          <div class="credential-body">
            <div class="info-item">
              <span class="label">创建时间</span>
              <span class="value">{{ cred.createdAt }}</span>
            </div>
            <div class="info-item">
              <span class="label">更新时间</span>
              <span class="value">{{ cred.updatedAt }}</span>
            </div>
          </div>
          <div class="credential-footer">
            <el-button class="black-button" size="small" @click="handleEdit(cred)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(cred)">删除</el-button>
          </div>
        </div>

        <!-- 添加新凭证卡片 -->
        <div class="credential-card add-card" @click="handleAdd">
          <el-icon :size="32"><Plus /></el-icon>
          <span>添加新凭证</span>
        </div>
      </div>

      <el-empty v-if="credentialList.length === 0 && !loading" description="暂无凭证" />
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="500px"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="应用" prop="appId">
          <el-select v-model="form.appId" placeholder="请选择应用" style="width: 100%" filterable>
            <el-option v-for="app in appList" :key="app.id" :label="app.name" :value="app.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入应用账号" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="额外数据" prop="extraData">
          <el-input v-model="form.extraData" type="textarea" :rows="2" placeholder="JSON格式的额外数据（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Lock, Plus, Key } from '@element-plus/icons-vue'
import {
  getUserCredentials,
  createUserCredential,
  updateUserCredential,
  deleteUserCredential,
  getSSOApplications,
  type UserCredential
} from '@/api/identity'

const credentialList = ref<UserCredential[]>([])
const appList = ref<any[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('添加凭证')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const isEdit = ref(false)

const form = reactive({
  id: 0,
  appId: undefined as number | undefined,
  username: '',
  password: '',
  extraData: ''
})

const rules: FormRules = {
  appId: [{ required: true, message: '请选择应用', trigger: 'change' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }]
}

const loadCredentials = async () => {
  loading.value = true
  try {
    const res = await getUserCredentials()
    if (res.data.code === 0) {
      credentialList.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载凭证失败:', error)
  } finally {
    loading.value = false
  }
}

const loadApps = async () => {
  try {
    const res = await getSSOApplications({ enabled: true, pageSize: 100 })
    if (res.data.code === 0) {
      appList.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('加载应用列表失败:', error)
  }
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '添加凭证'
  Object.assign(form, {
    id: 0,
    appId: undefined,
    username: '',
    password: '',
    extraData: ''
  })
  dialogVisible.value = true
}

const handleEdit = (cred: Credential) => {
  isEdit.value = true
  dialogTitle.value = '编辑凭证'
  Object.assign(form, {
    id: cred.id,
    appId: cred.appId,
    username: cred.username,
    password: '',
    extraData: cred.extraData || ''
  })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      if (isEdit.value) {
        await updateUserCredential(form.id, form)
        ElMessage.success('更新成功')
      } else {
        await createUserCredential(form)
        ElMessage.success('添加成功')
      }
      dialogVisible.value = false
      loadCredentials()
    } catch (error) {
      ElMessage.error('操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

const handleDelete = (cred: Credential) => {
  ElMessageBox.confirm(`确定要删除该凭证吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteUserCredential(cred.id)
      ElMessage.success('删除成功')
      loadCredentials()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}

onMounted(() => {
  loadCredentials()
  loadApps()
})
</script>

<style scoped>
.credentials-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
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
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  border: 1px solid #d4af37;
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

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-alert {
  border-radius: 8px;
}

.credentials-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.credential-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 20px;
  transition: all 0.3s;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.credential-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 16px rgba(212, 175, 55, 0.15);
}

.credential-card.add-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #909399;
  min-height: 180px;
  border-style: dashed;
}

.credential-card.add-card:hover {
  color: #d4af37;
  border-color: #d4af37;
}

.credential-card.add-card span {
  margin-top: 8px;
}

.credential-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #ebeef5;
}

.app-icon {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.app-icon .el-icon {
  color: #d4af37;
}

.app-info h4 {
  margin: 0 0 4px 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.app-info .username {
  font-size: 13px;
  color: #909399;
}

.credential-body {
  margin-bottom: 16px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.info-item .label {
  font-size: 13px;
  color: #909399;
}

.info-item .value {
  font-size: 13px;
  color: #606266;
}

.credential-footer {
  display: flex;
  gap: 8px;
}

.black-button {
  background-color: #000 !important;
  border-color: #000 !important;
  color: #d4af37 !important;
}

.black-button:hover {
  background-color: #1a1a1a !important;
  border-color: #1a1a1a !important;
}
</style>
