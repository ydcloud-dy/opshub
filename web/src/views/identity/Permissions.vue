<template>
  <div class="permissions-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Key /></el-icon>
        </div>
        <div>
          <h2 class="page-title">访问策略</h2>
          <p class="page-subtitle">配置用户、角色或部门对应用的访问权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          添加权限
        </el-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 搜索栏 -->
      <div class="filter-bar">
        <div class="filter-inputs">
          <el-select v-model="searchForm.appId" placeholder="选择应用" clearable filterable class="filter-input" @change="loadPermissions">
            <el-option v-for="app in appList" :key="app.id" :label="app.name" :value="app.id" />
          </el-select>
          <el-select v-model="searchForm.subjectType" placeholder="主体类型" clearable class="filter-input" @change="loadPermissions">
            <el-option label="用户" value="user" />
            <el-option label="角色" value="role" />
            <el-option label="部门" value="dept" />
          </el-select>
        </div>
        <div class="filter-actions">
          <el-button class="black-button" @click="loadPermissions">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
      </div>

      <!-- 表格 -->
      <div class="table-wrapper">
        <el-table :data="permissionList" v-loading="loading" border stripe>
          <el-table-column label="应用" min-width="150">
            <template #default="{ row }">
              <div class="app-cell">
                <div class="app-icon-small">
                  <el-icon><Grid /></el-icon>
                </div>
                <span>{{ getAppName(row.appId) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="主体类型" width="100" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="getSubjectTypeTag(row.subjectType)">
                {{ getSubjectTypeLabel(row.subjectType) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="主体" min-width="150">
            <template #default="{ row }">
              {{ getSubjectName(row.subjectType, row.subjectId) }}
            </template>
          </el-table-column>
          <el-table-column label="权限" width="100">
            <template #default="{ row }">
              <el-tag size="small" type="success">{{ row.permission || 'access' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="170" />
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
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
            @size-change="loadPermissions"
            @current-change="loadPermissions"
          />
        </div>
      </div>
    </div>

    <!-- 新增对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="添加权限"
      width="500px"
      @close="handleDialogClose"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="应用" prop="appId">
          <el-select v-model="form.appId" placeholder="请选择应用" style="width: 100%" filterable>
            <el-option v-for="app in appList" :key="app.id" :label="app.name" :value="app.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="主体类型" prop="subjectType">
          <el-select v-model="form.subjectType" placeholder="请选择" style="width: 100%" @change="handleSubjectTypeChange">
            <el-option label="用户" value="user" />
            <el-option label="角色" value="role" />
            <el-option label="部门" value="dept" />
          </el-select>
        </el-form-item>
        <el-form-item label="主体" prop="subjectId">
          <el-select v-model="form.subjectId" placeholder="请选择" style="width: 100%" filterable>
            <el-option v-for="item in subjectOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="权限" prop="permission">
          <el-select v-model="form.permission" placeholder="请选择" style="width: 100%">
            <el-option label="访问" value="access" />
            <el-option label="管理" value="admin" />
          </el-select>
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
import { Key, Plus, Grid } from '@element-plus/icons-vue'
import {
  getPermissions,
  createPermission,
  deletePermission,
  getSSOApplications
} from '@/api/identity'

const permissionList = ref<any[]>([])
const appList = ref<any[]>([])
const userList = ref<any[]>([])
const roleList = ref<any[]>([])
const deptList = ref<any[]>([])
const subjectOptions = ref<any[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()

const searchForm = reactive({
  appId: undefined as number | undefined,
  subjectType: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const form = reactive({
  appId: undefined as number | undefined,
  subjectType: '',
  subjectId: undefined as number | undefined,
  permission: 'access'
})

const rules: FormRules = {
  appId: [{ required: true, message: '请选择应用', trigger: 'change' }],
  subjectType: [{ required: true, message: '请选择主体类型', trigger: 'change' }],
  subjectId: [{ required: true, message: '请选择主体', trigger: 'change' }]
}

const getSubjectTypeLabel = (type: string) => {
  const map: Record<string, string> = { user: '用户', role: '角色', dept: '部门' }
  return map[type] || type
}

const getSubjectTypeTag = (type: string) => {
  const map: Record<string, string> = { user: '', role: 'success', dept: 'warning' }
  return map[type] || ''
}

const getAppName = (appId: number) => {
  const app = appList.value.find(a => a.id === appId)
  return app?.name || `应用${appId}`
}

const getSubjectName = (type: string, id: number) => {
  let list: any[] = []
  if (type === 'user') list = userList.value
  else if (type === 'role') list = roleList.value
  else if (type === 'dept') list = deptList.value
  const item = list.find(i => i.id === id)
  return item?.name || item?.username || `${type}:${id}`
}

const loadPermissions = async () => {
  loading.value = true
  try {
    const res = await getPermissions({
      page: pagination.page,
      pageSize: pagination.pageSize,
      appId: searchForm.appId,
      subjectType: searchForm.subjectType
    })
    if (res.data.code === 0) {
      permissionList.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    console.error('加载权限失败:', error)
  } finally {
    loading.value = false
  }
}

const loadApps = async () => {
  try {
    const res = await getSSOApplications({ pageSize: 100 })
    if (res.data.code === 0) {
      appList.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('加载应用失败:', error)
  }
}

const resetSearch = () => {
  searchForm.appId = undefined
  searchForm.subjectType = ''
  pagination.page = 1
  loadPermissions()
}

const handleAdd = () => {
  Object.assign(form, {
    appId: undefined,
    subjectType: '',
    subjectId: undefined,
    permission: 'access'
  })
  subjectOptions.value = []
  dialogVisible.value = true
}

const handleSubjectTypeChange = () => {
  form.subjectId = undefined
  if (form.subjectType === 'user') {
    subjectOptions.value = userList.value.map(u => ({ id: u.id, name: u.username || u.realName }))
  } else if (form.subjectType === 'role') {
    subjectOptions.value = roleList.value.map(r => ({ id: r.id, name: r.name }))
  } else if (form.subjectType === 'dept') {
    subjectOptions.value = deptList.value.map(d => ({ id: d.id, name: d.name }))
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      await createPermission(form)
      ElMessage.success('添加成功')
      dialogVisible.value = false
      loadPermissions()
    } catch (error) {
      ElMessage.error('操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定要删除该权限配置吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deletePermission(row.id)
      ElMessage.success('删除成功')
      loadPermissions()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}

onMounted(() => {
  loadPermissions()
  loadApps()
})
</script>

<style scoped>
.permissions-container {
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

.app-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.app-icon-small {
  width: 28px;
  height: 28px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 4px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-icon-small .el-icon {
  color: #d4af37;
  font-size: 14px;
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
