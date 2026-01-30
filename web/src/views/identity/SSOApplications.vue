<template>
  <div class="apps-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Grid /></el-icon>
        </div>
        <div>
          <h2 class="page-title">应用管理</h2>
          <p class="page-subtitle">配置SSO应用，支持Jenkins、GitLab、Harbor、Grafana等</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="showTemplates = true">
          <el-icon style="margin-right: 6px;"><Files /></el-icon>
          选择模板
        </el-button>
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增应用
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
            placeholder="搜索应用名称..."
            clearable
            class="filter-input"
            @keyup.enter="loadApps"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-select v-model="searchForm.category" placeholder="分类" clearable class="filter-input" @change="loadApps">
            <el-option v-for="cat in categories" :key="cat.value" :label="cat.label" :value="cat.value" />
          </el-select>
          <el-select v-model="searchForm.enabled" placeholder="状态" clearable class="filter-input" @change="loadApps">
            <el-option label="启用" :value="true" />
            <el-option label="禁用" :value="false" />
          </el-select>
        </div>
        <div class="filter-actions">
          <el-button class="black-button" @click="loadApps">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
      </div>

      <!-- 表格 -->
      <div class="table-wrapper">
        <el-table :data="appList" v-loading="loading" border stripe>
          <el-table-column label="应用" min-width="200">
            <template #default="{ row }">
              <div class="app-cell">
                <div class="app-icon-small">
                  <img v-if="row.icon" :src="row.icon" :alt="row.name" />
                  <el-icon v-else><Grid /></el-icon>
                </div>
                <div class="app-text">
                  <div class="app-name">{{ row.name }}</div>
                  <div class="app-code">{{ row.code }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="分类" width="120">
            <template #default="{ row }">
              <el-tag size="small">{{ getCategoryLabel(row.category) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="url" label="URL" min-width="200" show-overflow-tooltip />
          <el-table-column label="SSO类型" width="100">
            <template #default="{ row }">
              <span>{{ row.ssoType || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="handleToggleEnabled(row)" />
            </template>
          </el-table-column>
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
            @size-change="loadApps"
            @current-change="loadApps"
          />
        </div>
      </div>
    </div>

    <!-- 模板选择对话框 -->
    <el-dialog v-model="showTemplates" title="选择应用模板" width="800px">
      <div class="template-grid">
        <div
          v-for="tpl in templates"
          :key="tpl.code"
          class="template-card"
          @click="handleSelectTemplate(tpl)"
        >
          <div class="template-icon">
            <img v-if="tpl.icon" :src="tpl.icon" :alt="tpl.name" />
            <el-icon v-else :size="28"><Grid /></el-icon>
          </div>
          <div class="template-info">
            <div class="template-name">{{ tpl.name }}</div>
            <div class="template-desc">{{ tpl.description }}</div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="650px"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="应用名称" prop="name">
              <el-input v-model="form.name" placeholder="请输入应用名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="应用编码" prop="code">
              <el-input v-model="form.code" placeholder="请输入应用编码" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="分类" prop="category">
              <el-select v-model="form.category" placeholder="请选择分类" style="width: 100%">
                <el-option v-for="cat in categories" :key="cat.value" :label="cat.label" :value="cat.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="SSO类型" prop="ssoType">
              <el-select v-model="form.ssoType" placeholder="请选择" style="width: 100%">
                <el-option label="OAuth2" value="oauth2" />
                <el-option label="SAML" value="saml" />
                <el-option label="表单代填" value="form" />
                <el-option label="Token" value="token" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="应用URL" prop="url">
          <el-input v-model="form.url" placeholder="请输入应用URL" />
        </el-form-item>
        <el-form-item label="图标URL" prop="icon">
          <el-input v-model="form.icon" placeholder="请输入图标URL" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="SSO配置" prop="ssoConfig">
          <el-input v-model="form.ssoConfig" type="textarea" :rows="3" placeholder="请输入JSON格式的SSO配置" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="排序" prop="sort">
              <el-input-number v-model="form.sort" :min="0" :max="999" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="启用" prop="enabled">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </el-col>
        </el-row>
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
import { Grid, Plus, Search, Files } from '@element-plus/icons-vue'
import {
  getSSOApplications,
  createSSOApplication,
  updateSSOApplication,
  deleteSSOApplication,
  getAppTemplates,
  type SSOApplication
} from '@/api/identity'

const appList = ref<SSOApplication[]>([])
const templates = ref<any[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const showTemplates = ref(false)
const dialogTitle = ref('新增应用')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const isEdit = ref(false)

const categories = [
  { value: 'cicd', label: 'CI/CD' },
  { value: 'code', label: '代码管理' },
  { value: 'monitor', label: '监控告警' },
  { value: 'registry', label: '镜像仓库' },
  { value: 'other', label: '其他' }
]

const searchForm = reactive({
  keyword: '',
  category: '',
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
  code: '',
  icon: '',
  description: '',
  category: '',
  url: '',
  ssoType: '',
  ssoConfig: '',
  enabled: true,
  sort: 0
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  url: [{ required: true, message: '请输入应用URL', trigger: 'blur' }]
}

const getCategoryLabel = (cat: string) => {
  const found = categories.find(c => c.value === cat)
  return found?.label || cat || '未分类'
}

const loadApps = async () => {
  loading.value = true
  try {
    const res = await getSSOApplications({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      category: searchForm.category,
      enabled: searchForm.enabled
    })
    if (res.data.code === 0) {
      appList.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    console.error('加载应用失败:', error)
  } finally {
    loading.value = false
  }
}

const loadTemplates = async () => {
  try {
    const res = await getAppTemplates()
    if (res.data.code === 0) {
      templates.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载模板失败:', error)
  }
}

const resetSearch = () => {
  searchForm.keyword = ''
  searchForm.category = ''
  searchForm.enabled = undefined
  pagination.page = 1
  loadApps()
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新增应用'
  Object.assign(form, {
    id: 0,
    name: '',
    code: '',
    icon: '',
    description: '',
    category: '',
    url: '',
    ssoType: '',
    ssoConfig: '',
    enabled: true,
    sort: 0
  })
  dialogVisible.value = true
}

const handleSelectTemplate = (tpl: any) => {
  Object.assign(form, {
    id: 0,
    name: tpl.name,
    code: tpl.code,
    icon: tpl.icon || '',
    description: tpl.description || '',
    category: tpl.category || '',
    url: '',
    ssoType: tpl.ssoType || '',
    ssoConfig: tpl.ssoConfig || '',
    enabled: true,
    sort: 0
  })
  showTemplates.value = false
  dialogTitle.value = '新增应用'
  isEdit.value = false
  dialogVisible.value = true
}

const handleEdit = (row: SSOApplication) => {
  isEdit.value = true
  dialogTitle.value = '编辑应用'
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
        await updateSSOApplication(form.id, form)
        ElMessage.success('更新成功')
      } else {
        await createSSOApplication(form)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadApps()
    } catch (error) {
      ElMessage.error('操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

const handleDelete = (row: SSOApplication) => {
  ElMessageBox.confirm(`确定要删除应用"${row.name}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteSSOApplication(row.id)
      ElMessage.success('删除成功')
      loadApps()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

const handleToggleEnabled = async (row: SSOApplication) => {
  try {
    await updateSSOApplication(row.id, { enabled: row.enabled })
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
  loadApps()
  loadTemplates()
})
</script>

<style scoped>
.apps-container {
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

.header-actions {
  display: flex;
  gap: 12px;
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
  width: 180px;
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

.app-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.app-icon-small {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 6px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon-small img {
  width: 22px;
  height: 22px;
  object-fit: contain;
}

.app-icon-small .el-icon {
  color: #d4af37;
  font-size: 18px;
}

.app-text {
  display: flex;
  flex-direction: column;
}

.app-name {
  font-weight: 500;
  color: #303133;
}

.app-code {
  font-size: 12px;
  color: #909399;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

/* 模板网格 */
.template-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.template-card {
  background: #fafafa;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 12px;
}

.template-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.15);
}

.template-icon {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.template-icon img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.template-icon .el-icon {
  color: #d4af37;
}

.template-info {
  flex: 1;
  min-width: 0;
}

.template-name {
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.template-desc {
  font-size: 12px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
