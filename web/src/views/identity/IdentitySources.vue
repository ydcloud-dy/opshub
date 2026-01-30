<template>
  <div class="identity-sources-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <h2 class="page-title">身份源管理</h2>
      <el-button class="black-button" @click="handleAdd">新增身份源</el-button>
    </div>

    <!-- 搜索表单 -->
    <el-form :inline="true" :model="searchForm" class="search-form">
      <el-form-item label="关键词">
        <el-input v-model="searchForm.keyword" placeholder="身份源名称" clearable />
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="searchForm.enabled" placeholder="请选择" clearable>
          <el-option label="启用" :value="true" />
          <el-option label="禁用" :value="false" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button class="black-button" @click="loadSources">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 身份源卡片列表 -->
    <div class="source-grid" v-loading="loading">
      <div
        v-for="source in sourceList"
        :key="source.id"
        class="source-card"
        :class="{ disabled: !source.enabled }"
      >
        <div class="source-header">
          <div class="source-icon">
            <img v-if="source.icon" :src="source.icon" :alt="source.name" />
            <el-icon v-else :size="24"><Key /></el-icon>
          </div>
          <div class="source-title">
            <h4>{{ source.name }}</h4>
            <el-tag size="small" :type="source.enabled ? 'success' : 'info'">
              {{ source.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </div>
        </div>
        <div class="source-body">
          <div class="source-type">
            <span class="label">类型：</span>
            <span class="value">{{ getSourceTypeLabel(source.type) }}</span>
          </div>
          <div class="source-config">
            <span class="label">自动创建用户：</span>
            <span class="value">{{ source.autoCreateUser ? '是' : '否' }}</span>
          </div>
        </div>
        <div class="source-footer">
          <el-button type="primary" link @click="handleEdit(source)">编辑</el-button>
          <el-button type="primary" link @click="handleToggleEnable(source)">
            {{ source.enabled ? '禁用' : '启用' }}
          </el-button>
          <el-button type="danger" link @click="handleDelete(source)">删除</el-button>
        </div>
      </div>

      <!-- 添加新身份源卡片 -->
      <div class="source-card add-card" @click="handleAdd">
        <el-icon :size="32"><Plus /></el-icon>
        <span>添加身份源</span>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      @close="handleDialogClose"
    >
      <el-form :model="formData" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="身份源名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入身份源名称" />
        </el-form-item>
        <el-form-item label="身份源类型" prop="type">
          <el-select v-model="formData.type" placeholder="请选择类型" style="width: 100%">
            <el-option label="GitHub" value="github" />
            <el-option label="GitLab" value="gitlab" />
            <el-option label="微信" value="wechat" />
            <el-option label="企业微信" value="wechat_work" />
            <el-option label="钉钉" value="dingtalk" />
            <el-option label="飞书" value="feishu" />
            <el-option label="QQ" value="qq" />
            <el-option label="支付宝" value="alipay" />
            <el-option label="百度" value="baidu" />
            <el-option label="LDAP" value="ldap" />
            <el-option label="OIDC" value="oidc" />
            <el-option label="SAML" value="saml" />
          </el-select>
        </el-form-item>
        <el-form-item label="图标URL">
          <el-input v-model="formData.icon" placeholder="请输入图标URL" />
        </el-form-item>

        <el-divider content-position="left">OAuth配置</el-divider>
        <el-form-item label="Client ID" prop="clientId">
          <el-input v-model="configData.clientId" placeholder="请输入Client ID" />
        </el-form-item>
        <el-form-item label="Client Secret" prop="clientSecret">
          <el-input v-model="configData.clientSecret" placeholder="请输入Client Secret" show-password />
        </el-form-item>
        <el-form-item label="回调地址">
          <el-input v-model="configData.redirectUri" placeholder="请输入回调地址" />
        </el-form-item>
        <el-form-item label="权限范围">
          <el-input v-model="configData.scopes" placeholder="如: user:email" />
        </el-form-item>

        <el-divider content-position="left">用户配置</el-divider>
        <el-form-item label="自动创建用户">
          <el-switch v-model="formData.autoCreateUser" />
        </el-form-item>
        <el-form-item label="默认角色" v-if="formData.autoCreateUser">
          <el-select v-model="formData.defaultRoleId" placeholder="请选择默认角色" style="width: 100%">
            <el-option v-for="role in roleList" :key="role.id" :label="role.name" :value="role.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="formData.enabled" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="formData.sort" :min="0" />
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
  getIdentitySources,
  createIdentitySource,
  updateIdentitySource,
  deleteIdentitySource,
  type IdentitySource
} from '@/api/identity'
import { getRoleList } from '@/api/role'

const loading = ref(false)
const sourceList = ref<IdentitySource[]>([])
const roleList = ref<any[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()

const searchForm = reactive({
  keyword: '',
  enabled: undefined as boolean | undefined
})

const formData = reactive({
  id: 0,
  name: '',
  type: '',
  icon: '',
  autoCreateUser: false,
  defaultRoleId: 0,
  enabled: true,
  sort: 0
})

const configData = reactive({
  clientId: '',
  clientSecret: '',
  redirectUri: '',
  scopes: ''
})

const rules = {
  name: [{ required: true, message: '请输入身份源名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择身份源类型', trigger: 'change' }]
}

const dialogTitle = computed(() => isEdit.value ? '编辑身份源' : '新增身份源')

// 类型标签映射
const sourceTypeMap: Record<string, string> = {
  github: 'GitHub',
  gitlab: 'GitLab',
  wechat: '微信',
  wechat_work: '企业微信',
  dingtalk: '钉钉',
  feishu: '飞书',
  qq: 'QQ',
  alipay: '支付宝',
  baidu: '百度',
  ldap: 'LDAP',
  oidc: 'OIDC',
  saml: 'SAML'
}

const getSourceTypeLabel = (type: string) => {
  return sourceTypeMap[type] || type
}

// 加载身份源列表
const loadSources = async () => {
  loading.value = true
  try {
    const res = await getIdentitySources({
      page: 1,
      pageSize: 100,
      keyword: searchForm.keyword,
      enabled: searchForm.enabled
    })
    if (res.data.code === 0) {
      sourceList.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('加载身份源列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载角色列表
const loadRoles = async () => {
  try {
    const res = await getRoleList({ page: 1, pageSize: 100 })
    if (res.data.code === 0) {
      roleList.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('加载角色列表失败:', error)
  }
}

const resetSearch = () => {
  searchForm.keyword = ''
  searchForm.enabled = undefined
  loadSources()
}

const handleAdd = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (source: IdentitySource) => {
  isEdit.value = true
  Object.assign(formData, {
    id: source.id,
    name: source.name,
    type: source.type,
    icon: source.icon,
    autoCreateUser: source.autoCreateUser,
    defaultRoleId: source.defaultRoleId,
    enabled: source.enabled,
    sort: source.sort
  })
  // 解析配置
  try {
    const config = JSON.parse(source.config || '{}')
    Object.assign(configData, config)
  } catch {
    // ignore
  }
  dialogVisible.value = true
}

const handleToggleEnable = async (source: IdentitySource) => {
  try {
    await updateIdentitySource(source.id, { enabled: !source.enabled })
    ElMessage.success(source.enabled ? '已禁用' : '已启用')
    loadSources()
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

const handleDelete = async (source: IdentitySource) => {
  try {
    await ElMessageBox.confirm(`确定要删除身份源 "${source.name}" 吗？`, '提示', {
      type: 'warning'
    })
    await deleteIdentitySource(source.id)
    ElMessage.success('删除成功')
    loadSources()
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
      ...formData,
      config: JSON.stringify(configData)
    }
    if (isEdit.value) {
      await updateIdentitySource(formData.id, data)
      ElMessage.success('更新成功')
    } else {
      await createIdentitySource(data)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadSources()
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
    name: '',
    type: '',
    icon: '',
    autoCreateUser: false,
    defaultRoleId: 0,
    enabled: true,
    sort: 0
  })
  Object.assign(configData, {
    clientId: '',
    clientSecret: '',
    redirectUri: '',
    scopes: ''
  })
  formRef.value?.clearValidate()
}

onMounted(() => {
  loadSources()
  loadRoles()
})
</script>

<style scoped>
.identity-sources-container {
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

.search-form {
  margin-bottom: 20px;
}

.source-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.source-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.3s ease;
}

.source-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 16px rgba(212, 175, 55, 0.15);
}

.source-card.disabled {
  opacity: 0.6;
}

.source-card.add-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 180px;
  cursor: pointer;
  border-style: dashed;
  color: #909399;
}

.source-card.add-card:hover {
  color: #d4af37;
  border-color: #d4af37;
}

.source-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.source-icon {
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

.source-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.source-icon .el-icon {
  color: #d4af37;
}

.source-title h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.source-body {
  margin-bottom: 16px;
}

.source-body > div {
  margin-bottom: 8px;
  font-size: 13px;
}

.source-body .label {
  color: #909399;
}

.source-body .value {
  color: #606266;
}

.source-footer {
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
