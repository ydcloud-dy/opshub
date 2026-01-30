<template>
  <div class="sources-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h2 class="page-title">身份源管理</h2>
          <p class="page-subtitle">配置第三方登录方式，支持微信、钉钉、飞书、GitHub等</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增身份源
        </el-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 搜索栏 -->
      <div class="filter-bar">
        <div class="filter-inputs">
          <el-input
            v-model="searchForm.keyword"
            placeholder="搜索名称..."
            clearable
            class="filter-input"
            @keyup.enter="loadSources"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-select v-model="searchForm.enabled" placeholder="状态" clearable class="filter-input" @change="loadSources">
            <el-option label="启用" :value="true" />
            <el-option label="禁用" :value="false" />
          </el-select>
        </div>
        <div class="filter-actions">
          <el-button class="black-button" @click="loadSources">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
      </div>

      <!-- 表格 -->
      <div class="table-wrapper">
        <el-table :data="sourceList" v-loading="loading" border stripe>
          <el-table-column prop="name" label="名称" min-width="150">
            <template #default="{ row }">
              <div class="source-name">
                <div class="source-icon">
                  <img v-if="row.icon" :src="row.icon" :alt="row.name" />
                  <el-icon v-else><Link /></el-icon>
                </div>
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" width="120">
            <template #default="{ row }">
              <el-tag>{{ getSourceTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="autoCreateUser" label="自动创建用户" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="row.autoCreateUser ? 'success' : 'info'" size="small">
                {{ row.autoCreateUser ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="enabled" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-switch
                v-model="row.enabled"
                @change="handleToggleEnabled(row)"
              />
            </template>
          </el-table-column>
          <el-table-column prop="sort" label="排序" width="80" align="center" />
          <el-table-column prop="createdAt" label="创建时间" width="170" />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button class="black-button" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
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
            @size-change="loadSources"
            @current-change="loadSources"
          />
        </div>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入身份源名称" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
            <el-option label="微信" value="wechat" />
            <el-option label="钉钉" value="dingtalk" />
            <el-option label="飞书" value="feishu" />
            <el-option label="QQ" value="qq" />
            <el-option label="GitHub" value="github" />
            <el-option label="支付宝" value="alipay" />
            <el-option label="百度" value="baidu" />
            <el-option label="LDAP" value="ldap" />
            <el-option label="OIDC" value="oidc" />
            <el-option label="SAML" value="saml" />
          </el-select>
        </el-form-item>
        <el-form-item label="图标URL" prop="icon">
          <el-input v-model="form.icon" placeholder="请输入图标URL" />
        </el-form-item>
        <el-form-item label="配置" prop="config">
          <el-input
            v-model="form.config"
            type="textarea"
            :rows="4"
            placeholder="请输入JSON格式配置"
          />
        </el-form-item>
        <el-form-item label="自动创建用户" prop="autoCreateUser">
          <el-switch v-model="form.autoCreateUser" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="启用" prop="enabled">
          <el-switch v-model="form.enabled" />
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
import { User, Plus, Search, Link } from '@element-plus/icons-vue'
import {
  getIdentitySources,
  createIdentitySource,
  updateIdentitySource,
  deleteIdentitySource,
  type IdentitySource
} from '@/api/identity'

const sourceList = ref<IdentitySource[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('新增身份源')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const isEdit = ref(false)

const searchForm = reactive({
  keyword: '',
  enabled: undefined as boolean | undefined
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const form = reactive({
  id: 0,
  name: '',
  type: '',
  icon: '',
  config: '',
  autoCreateUser: false,
  defaultRoleId: 0,
  enabled: true,
  sort: 0
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

const sourceTypeMap: Record<string, string> = {
  wechat: '微信',
  dingtalk: '钉钉',
  feishu: '飞书',
  qq: 'QQ',
  github: 'GitHub',
  alipay: '支付宝',
  baidu: '百度',
  ldap: 'LDAP',
  oidc: 'OIDC',
  saml: 'SAML'
}

const getSourceTypeLabel = (type: string) => sourceTypeMap[type] || type

const loadSources = async () => {
  loading.value = true
  try {
    const res = await getIdentitySources({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      enabled: searchForm.enabled
    })
    if (res.data.code === 0) {
      sourceList.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    console.error('加载身份源失败:', error)
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.keyword = ''
  searchForm.enabled = undefined
  pagination.page = 1
  loadSources()
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新增身份源'
  Object.assign(form, {
    id: 0,
    name: '',
    type: '',
    icon: '',
    config: '',
    autoCreateUser: false,
    defaultRoleId: 0,
    enabled: true,
    sort: 0
  })
  dialogVisible.value = true
}

const handleEdit = (row: IdentitySource) => {
  isEdit.value = true
  dialogTitle.value = '编辑身份源'
  Object.assign(form, { ...row })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      if (isEdit.value) {
        await updateIdentitySource(form.id, form)
        ElMessage.success('更新成功')
      } else {
        await createIdentitySource(form)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadSources()
    } catch (error) {
      ElMessage.error('操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

const handleDelete = (row: IdentitySource) => {
  ElMessageBox.confirm(`确定要删除身份源"${row.name}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteIdentitySource(row.id)
      ElMessage.success('删除成功')
      loadSources()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

const handleToggleEnabled = async (row: IdentitySource) => {
  try {
    await updateIdentitySource(row.id, { enabled: row.enabled })
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch (error) {
    row.enabled = !row.enabled
    ElMessage.error('操作失败')
  }
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}

onMounted(() => {
  loadSources()
})
</script>

<style scoped>
.sources-container {
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

.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.filter-inputs {
  display: flex;
  gap: 12px;
}

.filter-input {
  width: 200px;
}

.filter-actions {
  display: flex;
  gap: 8px;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.source-name {
  display: flex;
  align-items: center;
  gap: 10px;
}

.source-icon {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 6px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.source-icon img {
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.source-icon .el-icon {
  color: #d4af37;
  font-size: 16px;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
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
