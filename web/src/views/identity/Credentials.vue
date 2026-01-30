<template>
  <div class="credentials-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <h2 class="page-title">凭证管理</h2>
      <el-button class="black-button" @click="handleAdd">添加凭证</el-button>
    </div>

    <!-- 说明 -->
    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 20px"
    >
      <template #title>
        凭证用于存储您在各应用中的账号密码，实现一键登录。凭证数据加密存储，仅您本人可见。
      </template>
    </el-alert>

    <!-- 凭证列表 -->
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
            <span class="label">创建时间：</span>
            <span class="value">{{ cred.createdAt }}</span>
          </div>
          <div class="info-item">
            <span class="label">更新时间：</span>
            <span class="value">{{ cred.updatedAt }}</span>
          </div>
        </div>
        <div class="credential-footer">
          <el-button type="primary" link @click="handleEdit(cred)">编辑</el-button>
          <el-button type="danger" link @click="handleDelete(cred)">删除</el-button>
        </div>
      </div>

      <!-- 添加新凭证卡片 -->
      <div class="credential-card add-card" @click="handleAdd">
        <el-icon :size="32"><Plus /></el-icon>
        <span>添加凭证</span>
      </div>
    </div>

    <el-empty v-if="!loading && credentialList.length === 0" description="暂无凭证" />

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="500px"
      @close="handleDialogClose"
    >
      <el-form :model="formData" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="选择应用" prop="appId">
          <el-select v-model="formData.appId" placeholder="请选择应用" style="width: 100%" filterable>
            <el-option
              v-for="app in appList"
              :key="app.id"
              :label="app.name"
              :value="app.id"
            >
              <div style="display: flex; align-items: center; gap: 8px;">
                <img
                  v-if="app.icon"
                  :src="app.icon"
                  style="width: 20px; height: 20px; object-fit: contain;"
                />
                <span>{{ app.name }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="账号" prop="username">
          <el-input v-model="formData.username" placeholder="请输入账号" />
        </el-form-item>
        <el-form-item label="密码" :prop="isEdit ? '' : 'password'">
          <el-input
            v-model="formData.password"
            type="password"
            :placeholder="isEdit ? '不修改请留空' : '请输入密码'"
            show-password
          />
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="formData.extraData"
            type="textarea"
            :rows="2"
            placeholder="可选，备注信息"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Key, Plus } from '@element-plus/icons-vue'
import {
  getUserCredentials,
  createUserCredential,
  updateUserCredential,
  deleteUserCredential,
  getSSOApplications,
  type UserCredential,
  type SSOApplication
} from '@/api/identity'

const loading = ref(false)
const credentialList = ref<UserCredential[]>([])
const appList = ref<SSOApplication[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()

const formData = reactive({
  id: 0,
  appId: 0,
  username: '',
  password: '',
  extraData: ''
})

const rules = {
  appId: [{ required: true, message: '请选择应用', trigger: 'change' }],
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const dialogTitle = computed(() => isEdit.value ? '编辑凭证' : '添加凭证')

// 加载凭证列表
const loadCredentials = async () => {
  loading.value = true
  try {
    const res = await getUserCredentials()
    if (res.data.code === 0) {
      credentialList.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载凭证列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载应用列表
const loadApps = async () => {
  try {
    const res = await getSSOApplications({ page: 1, pageSize: 100, enabled: true })
    if (res.data.code === 0) {
      appList.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('加载应用列表失败:', error)
  }
}

const handleAdd = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (cred: UserCredential) => {
  isEdit.value = true
  Object.assign(formData, {
    id: cred.id,
    appId: cred.appId,
    username: cred.username,
    password: '',
    extraData: cred.extraData || ''
  })
  dialogVisible.value = true
}

const handleDelete = async (cred: UserCredential) => {
  try {
    await ElMessageBox.confirm(`确定要删除 "${cred.appName}" 的凭证吗？`, '提示', {
      type: 'warning'
    })
    await deleteUserCredential(cred.id)
    ElMessage.success('删除成功')
    loadCredentials()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    const data = {
      appId: formData.appId,
      username: formData.username,
      password: formData.password || undefined,
      extraData: formData.extraData || undefined
    }
    if (isEdit.value) {
      await updateUserCredential(formData.id, data)
      ElMessage.success('更新成功')
    } else {
      await createUserCredential(data)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    loadCredentials()
  } catch (error) {
    console.error('提交失败:', error)
  }
}

const handleDialogClose = () => {
  resetForm()
}

const resetForm = () => {
  Object.assign(formData, {
    id: 0,
    appId: 0,
    username: '',
    password: '',
    extraData: ''
  })
  formRef.value?.clearValidate()
}

onMounted(() => {
  loadCredentials()
  loadApps()
})
</script>

<style scoped>
.credentials-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.credentials-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.credential-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.3s ease;
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
  min-height: 160px;
  cursor: pointer;
  border-style: dashed;
  color: #909399;
}

.credential-card.add-card:hover {
  color: #d4af37;
  border-color: #d4af37;
}

.credential-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.app-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.app-icon .el-icon {
  color: #d4af37;
}

.app-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
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
  margin-bottom: 8px;
  font-size: 13px;
}

.info-item .label {
  color: #909399;
}

.info-item .value {
  color: #606266;
}

.credential-footer {
  display: flex;
  gap: 8px;
  border-top: 1px solid #ebeef5;
  padding-top: 12px;
}

.black-button {
  background-color: #000 !important;
  border-color: #000 !important;
  color: #d4af37 !important;
}

.black-button:hover {
  background-color: #1a1a1a !important;
}
</style>
