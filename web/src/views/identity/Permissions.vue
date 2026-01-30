<template>
  <div class="permissions-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <h2 class="page-title">访问策略</h2>
      <el-button class="black-button" @click="handleAdd">添加权限</el-button>
    </div>

    <!-- 搜索表单 -->
    <el-form :inline="true" :model="searchForm" class="search-form">
      <el-form-item label="应用">
        <el-select v-model="searchForm.appId" placeholder="请选择应用" clearable filterable>
          <el-option v-for="app in appList" :key="app.id" :label="app.name" :value="app.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="主体类型">
        <el-select v-model="searchForm.subjectType" placeholder="请选择" clearable>
          <el-option label="用户" value="user" />
          <el-option label="角色" value="role" />
          <el-option label="部门" value="dept" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button class="black-button" @click="loadPermissions">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 表格 -->
    <el-table :data="permissionList" border stripe v-loading="loading" style="width: 100%">
      <el-table-column label="应用" min-width="150">
        <template #default="{ row }">
          {{ getAppName(row.appId) }}
        </template>
      </el-table-column>
      <el-table-column label="主体类型" width="100">
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
          {{ row.permission || 'access' }}
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180" prop="createdAt" />
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
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
      @size-change="loadPermissions"
      @current-change="loadPermissions"
      style="margin-top: 20px; justify-content: center"
    />

    <!-- 新增对话框 -->
    <el-dialog v-model="dialogVisible" title="添加权限" width="500px" @close="handleDialogClose">
      <el-form :model="formData" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="选择应用" prop="appId">
          <el-select v-model="formData.appId" placeholder="请选择应用" style="width: 100%" filterable>
            <el-option v-for="app in appList" :key="app.id" :label="app.name" :value="app.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="主体类型" prop="subjectType">
          <el-select v-model="formData.subjectType" placeholder="请选择" style="width: 100%">
            <el-option label="用户" value="user" />
            <el-option label="角色" value="role" />
            <el-option label="部门" value="dept" />
          </el-select>
        </el-form-item>
        <el-form-item label="选择用户" prop="subjectId" v-if="formData.subjectType === 'user'">
          <el-select v-model="formData.subjectId" placeholder="请选择用户" style="width: 100%" filterable>
            <el-option v-for="user in userList" :key="user.id" :label="user.username" :value="user.id">
              <span>{{ user.realName || user.username }}</span>
              <span style="color: #909399; margin-left: 8px;">{{ user.username }}</span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="选择角色" prop="subjectId" v-if="formData.subjectType === 'role'">
          <el-select v-model="formData.subjectId" placeholder="请选择角色" style="width: 100%" filterable>
            <el-option v-for="role in roleList" :key="role.id" :label="role.name" :value="role.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="选择部门" prop="subjectId" v-if="formData.subjectType === 'dept'">
          <el-select v-model="formData.subjectId" placeholder="请选择部门" style="width: 100%" filterable>
            <el-option v-for="dept in deptList" :key="dept.id" :label="dept.deptName" :value="dept.id" />
          </el-select>
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
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getAppPermissions,
  createAppPermission,
  deleteAppPermission,
  getSSOApplications,
  type AppPermission,
  type SSOApplication
} from '@/api/identity'
import { getUserList } from '@/api/user'
import { getRoleList } from '@/api/role'
import { getDepartmentTree } from '@/api/department'

const route = useRoute()
const loading = ref(false)
const permissionList = ref<AppPermission[]>([])
const appList = ref<SSOApplication[]>([])
const userList = ref<any[]>([])
const roleList = ref<any[]>([])
const deptList = ref<any[]>([])
const dialogVisible = ref(false)
const formRef = ref()

const searchForm = reactive({
  appId: undefined as number | undefined,
  subjectType: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const formData = reactive({
  appId: 0,
  subjectType: '',
  subjectId: 0
})

const rules = {
  appId: [{ required: true, message: '请选择应用', trigger: 'change' }],
  subjectType: [{ required: true, message: '请选择主体类型', trigger: 'change' }],
  subjectId: [{ required: true, message: '请选择主体', trigger: 'change' }]
}

const subjectTypeMap: Record<string, { label: string; tag: string }> = {
  user: { label: '用户', tag: '' },
  role: { label: '角色', tag: 'success' },
  dept: { label: '部门', tag: 'warning' }
}

const getSubjectTypeLabel = (type: string) => subjectTypeMap[type]?.label || type
const getSubjectTypeTag = (type: string) => subjectTypeMap[type]?.tag || ''

const getAppName = (appId: number) => {
  const app = appList.value.find(a => a.id === appId)
  return app?.name || '-'
}

const getSubjectName = (type: string, id: number) => {
  if (type === 'user') {
    const user = userList.value.find(u => u.id === id)
    return user?.realName || user?.username || id
  }
  if (type === 'role') {
    const role = roleList.value.find(r => r.id === id)
    return role?.name || id
  }
  if (type === 'dept') {
    const dept = deptList.value.find(d => d.id === id)
    return dept?.deptName || id
  }
  return id
}

// 加载权限列表
const loadPermissions = async () => {
  loading.value = true
  try {
    const res = await getAppPermissions({
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
    console.error('加载权限列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载应用、用户、角色、部门列表
const loadBaseData = async () => {
  try {
    const [appsRes, usersRes, rolesRes, deptsRes] = await Promise.all([
      getSSOApplications({ page: 1, pageSize: 100 }),
      getUserList({ page: 1, pageSize: 1000 }),
      getRoleList({ page: 1, pageSize: 100 }),
      getDepartmentTree()
    ])
    if (appsRes.data.code === 0) appList.value = appsRes.data.data?.list || []
    if (usersRes.data.code === 0) userList.value = usersRes.data.data?.list || []
    if (rolesRes.data.code === 0) roleList.value = rolesRes.data.data?.list || []
    if (deptsRes.data.code === 0) deptList.value = flattenDepts(deptsRes.data.data || [])
  } catch (error) {
    console.error('加载基础数据失败:', error)
  }
}

// 扁平化部门树
const flattenDepts = (depts: any[], result: any[] = []): any[] => {
  for (const dept of depts) {
    result.push(dept)
    if (dept.children && dept.children.length > 0) {
      flattenDepts(dept.children, result)
    }
  }
  return result
}

const resetSearch = () => {
  searchForm.appId = undefined
  searchForm.subjectType = ''
  pagination.page = 1
  loadPermissions()
}

const handleAdd = () => {
  resetForm()
  dialogVisible.value = true
}

const handleDelete = async (row: AppPermission) => {
  try {
    await ElMessageBox.confirm('确定要删除该权限吗？', '提示', { type: 'warning' })
    await deleteAppPermission(row.id)
    ElMessage.success('删除成功')
    loadPermissions()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    await createAppPermission({
      appId: formData.appId,
      subjectType: formData.subjectType,
      subjectId: formData.subjectId
    })
    ElMessage.success('添加成功')
    dialogVisible.value = false
    loadPermissions()
  } catch (error) {
    console.error('提交失败:', error)
  }
}

const handleDialogClose = () => {
  resetForm()
}

const resetForm = () => {
  Object.assign(formData, {
    appId: searchForm.appId || 0,
    subjectType: '',
    subjectId: 0
  })
  formRef.value?.clearValidate()
}

onMounted(() => {
  // 从URL参数获取appId
  if (route.query.appId) {
    searchForm.appId = Number(route.query.appId)
  }
  loadBaseData()
  loadPermissions()
})
</script>

<style scoped>
.permissions-container {
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

.black-button {
  background-color: #000 !important;
  border-color: #000 !important;
  color: #d4af37 !important;
}

.black-button:hover {
  background-color: #1a1a1a !important;
}
</style>
