<template>
  <div class="sso-apps-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <h2 class="page-title">应用管理</h2>
      <div class="header-actions">
        <el-button @click="showTemplates = true">选择模板</el-button>
        <el-button class="black-button" @click="handleAdd">新增应用</el-button>
      </div>
    </div>

    <!-- 搜索表单 -->
    <el-form :inline="true" :model="searchForm" class="search-form">
      <el-form-item label="关键词">
        <el-input v-model="searchForm.keyword" placeholder="应用名称/编码" clearable />
      </el-form-item>
      <el-form-item label="分类">
        <el-select v-model="searchForm.category" placeholder="请选择" clearable>
          <el-option v-for="cat in categories" :key="cat.value" :label="cat.label" :value="cat.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="searchForm.enabled" placeholder="请选择" clearable>
          <el-option label="启用" :value="true" />
          <el-option label="禁用" :value="false" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button class="black-button" @click="loadApps">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 表格 -->
    <el-table :data="appList" border stripe v-loading="loading" style="width: 100%">
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
          <el-tag size="small" class="category-tag">{{ getCategoryLabel(row.category) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="url" label="访问地址" min-width="200" show-overflow-tooltip />
      <el-table-column label="SSO类型" width="100">
        <template #default="{ row }">
          {{ row.ssoType || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
            {{ row.enabled ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="排序" width="80" prop="sort" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
          <el-button type="primary" link @click="handlePermission(row)">权限</el-button>
          <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <el-pagination
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="loadApps"
      @current-change="loadApps"
      style="margin-top: 20px; justify-content: center"
    />

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="650px"
      @close="handleDialogClose"
    >
      <el-form :model="formData" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="应用名称" prop="name">
              <el-input v-model="formData.name" placeholder="请输入应用名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="应用编码" prop="code">
              <el-input v-model="formData.code" placeholder="请输入应用编码" :disabled="isEdit" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="分类" prop="category">
              <el-select v-model="formData.category" placeholder="请选择分类" style="width: 100%">
                <el-option v-for="cat in categories" :key="cat.value" :label="cat.label" :value="cat.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="SSO类型">
              <el-select v-model="formData.ssoType" placeholder="请选择" style="width: 100%">
                <el-option label="无" value="" />
                <el-option label="OAuth2" value="oauth2" />
                <el-option label="OIDC" value="oidc" />
                <el-option label="SAML" value="saml" />
                <el-option label="表单代填" value="form" />
                <el-option label="Token" value="token" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="访问地址" prop="url">
          <el-input v-model="formData.url" placeholder="请输入访问地址" />
        </el-form-item>
        <el-form-item label="图标URL">
          <el-input v-model="formData.icon" placeholder="请输入图标URL" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="请输入描述" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="启用状态">
              <el-switch v-model="formData.enabled" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="排序">
              <el-input-number v-model="formData.sort" :min="0" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 模板选择对话框 -->
    <el-dialog v-model="showTemplates" title="选择应用模板" width="800px">
      <div class="template-grid">
        <div
          v-for="template in templates"
          :key="template.code"
          class="template-card"
          @click="handleSelectTemplate(template)"
        >
          <div class="template-icon">
            <img v-if="template.icon" :src="template.icon" :alt="template.name" />
            <el-icon v-else :size="32"><Grid /></el-icon>
          </div>
          <div class="template-info">
            <h4>{{ template.name }}</h4>
            <p>{{ template.description }}</p>
            <el-tag size="small">{{ template.ssoType }}</el-tag>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Grid } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import {
  getSSOApplications,
  getAppTemplates,
  getAppCategories,
  createSSOApplication,
  updateSSOApplication,
  deleteSSOApplication,
  type SSOApplication,
  type AppTemplate
} from '@/api/identity'

const router = useRouter()
const loading = ref(false)
const appList = ref<SSOApplication[]>([])
const templates = ref<AppTemplate[]>([])
const categories = ref<{ value: string; label: string }[]>([])
const dialogVisible = ref(false)
const showTemplates = ref(false)
const isEdit = ref(false)
const formRef = ref()

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

const formData = reactive({
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

const rules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入应用编码', trigger: 'blur' }],
  url: [{ required: true, message: '请输入访问地址', trigger: 'blur' }],
  category: [{ required: true, message: '请选择分类', trigger: 'change' }]
}

const dialogTitle = computed(() => isEdit.value ? '编辑应用' : '新增应用')

const getCategoryLabel = (category: string) => {
  const cat = categories.value.find(c => c.value === category)
  return cat?.label || category
}

// 加载应用列表
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
    console.error('加载应用列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载模板和分类
const loadTemplatesAndCategories = async () => {
  try {
    const [templatesRes, categoriesRes] = await Promise.all([
      getAppTemplates(),
      getAppCategories()
    ])
    if (templatesRes.data.code === 0) {
      templates.value = templatesRes.data.data || []
    }
    if (categoriesRes.data.code === 0) {
      categories.value = categoriesRes.data.data || []
    }
  } catch (error) {
    console.error('加载模板和分类失败:', error)
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
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (app: SSOApplication) => {
  isEdit.value = true
  Object.assign(formData, {
    id: app.id,
    name: app.name,
    code: app.code,
    icon: app.icon,
    description: app.description,
    category: app.category,
    url: app.url,
    ssoType: app.ssoType,
    ssoConfig: app.ssoConfig,
    enabled: app.enabled,
    sort: app.sort
  })
  dialogVisible.value = true
}

const handlePermission = (app: SSOApplication) => {
  router.push({ path: '/identity/permissions', query: { appId: app.id } })
}

const handleDelete = async (app: SSOApplication) => {
  try {
    await ElMessageBox.confirm(`确定要删除应用 "${app.name}" 吗？`, '提示', {
      type: 'warning'
    })
    await deleteSSOApplication(app.id)
    ElMessage.success('删除成功')
    loadApps()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSelectTemplate = (template: AppTemplate) => {
  Object.assign(formData, {
    name: template.name,
    code: template.code,
    icon: template.icon,
    description: template.description,
    category: template.category,
    ssoType: template.ssoType,
    url: template.urlTemplate
  })
  showTemplates.value = false
  isEdit.value = false
  dialogVisible.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    if (isEdit.value) {
      await updateSSOApplication(formData.id, formData)
      ElMessage.success('更新成功')
    } else {
      await createSSOApplication(formData)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadApps()
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
  formRef.value?.clearValidate()
}

onMounted(() => {
  loadApps()
  loadTemplatesAndCategories()
})
</script>

<style scoped>
.sso-apps-container {
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

.header-actions {
  display: flex;
  gap: 12px;
}

.search-form {
  margin-bottom: 20px;
}

.app-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-icon-small {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon-small img {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.app-icon-small .el-icon {
  color: #d4af37;
  font-size: 16px;
}

.app-text .app-name {
  font-weight: 500;
  color: #303133;
}

.app-text .app-code {
  font-size: 12px;
  color: #909399;
}

.category-tag {
  background: rgba(212, 175, 55, 0.1);
  color: #d4af37;
  border-color: rgba(212, 175, 55, 0.3);
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.template-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: center;
}

.template-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 16px rgba(212, 175, 55, 0.15);
}

.template-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 12px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 12px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.template-icon img {
  width: 36px;
  height: 36px;
  object-fit: contain;
}

.template-icon .el-icon {
  color: #d4af37;
}

.template-info h4 {
  margin: 0 0 8px 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.template-info p {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: #909399;
  height: 36px;
  overflow: hidden;
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
