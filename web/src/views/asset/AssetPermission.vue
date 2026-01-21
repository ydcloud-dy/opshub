<template>
  <div class="permission-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">权限配置</h2>
          <p class="page-subtitle">配置角色对资产分组和主机的访问权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          添加权限
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.roleName"
          placeholder="搜索角色名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-input
          v-model="searchForm.groupName"
          placeholder="搜索资产分组..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格和分页容器 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredPermissions"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column label="角色" min-width="150">
          <template #default="{ row }">
            <el-tag type="primary">{{ row.roleName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资产分组" min-width="180">
          <template #default="{ row }">
            <el-tag type="success">{{ row.assetGroupName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="主机" min-width="200">
          <template #default="{ row }">
            <el-tag v-if="!row.hostId" type="info">全部主机</el-tag>
            <div v-else>
              <div>{{ row.hostName }}</div>
              <div class="host-ip">{{ row.hostIp }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作权限" min-width="200">
          <template #default="{ row }">
            <div class="permission-tags">
              <el-tag v-if="(row.permissions & 1) > 0" size="small" type="success">查看</el-tag>
              <el-tag v-if="(row.permissions & 2) > 0" size="small" type="primary">编辑</el-tag>
              <el-tag v-if="(row.permissions & 4) > 0" size="small" type="danger">删除</el-tag>
              <el-tag v-if="(row.permissions & 8) > 0" size="small" type="warning">终端</el-tag>
              <el-tag v-if="(row.permissions & 16) > 0" size="small" type="info">文件</el-tag>
              <el-tag v-if="(row.permissions & 32) > 0" size="small">采集</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="createdAt" label="创建时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button
                  link
                  class="action-btn action-edit"
                  @click="handleEditClick(row)"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  link
                  class="action-btn action-delete"
                  @click="handleDeleteClick(row)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- 添加权限对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="添加权限"
      width="50%"
      class="permission-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="100px"
      >
        <el-form-item label="角色" prop="roleId">
          <el-select
            v-model="formData.roleId"
            placeholder="请选择角色"
            style="width: 100%"
            clearable
            filterable
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="资产分组" prop="assetGroupId">
          <el-tree-select
            v-model="formData.assetGroupId"
            :data="groupTreeData"
            check-strictly
            :render-after-expand="false"
            placeholder="请选择资产分组"
            style="width: 100%"
            @change="handleGroupChange"
          />
        </el-form-item>

        <el-form-item label="主机">
          <el-radio-group v-model="hostSelectionType" @change="handleHostTypeChange">
            <el-radio value="all">全部主机</el-radio>
            <el-radio value="specific">指定主机</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="hostSelectionType === 'specific'" label="选择主机" prop="hostIds">
          <el-select
            v-model="formData.hostIds"
            multiple
            placeholder="请选择主机"
            style="width: 100%"
            :loading="loadingHosts"
          >
            <el-option
              v-for="host in hostList"
              :key="host.id"
              :label="`${host.name} (${host.ip})`"
              :value="host.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="操作权限">
          <el-checkbox-group v-model="selectedPermissions">
            <el-checkbox :value="1">查看 - 查看主机详情</el-checkbox>
            <el-checkbox :value="2">编辑 - 创建、修改主机配置</el-checkbox>
            <el-checkbox :value="4">删除 - 删除主机</el-checkbox>
            <el-checkbox :value="8">终端 - SSH连接主机</el-checkbox>
            <el-checkbox :value="16">文件 - 文件上传、下载、删除</el-checkbox>
            <el-checkbox :value="32">采集 - 采集主机系统信息</el-checkbox>
          </el-checkbox-group>
          <div class="permission-tip">默认仅授予查看权限，请根据需要勾选其他操作权限</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑权限对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑权限"
      width="50%"
      class="permission-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleEditDialogClose"
    >
      <el-form
        ref="editFormRef"
        :model="editFormData"
        :rules="formRules"
        label-width="100px"
      >
        <el-form-item label="角色" prop="roleId">
          <el-select
            v-model="editFormData.roleId"
            placeholder="请选择角色"
            style="width: 100%"
            clearable
            filterable
            disabled
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="资产分组" prop="assetGroupId">
          <el-tree-select
            v-model="editFormData.assetGroupId"
            :data="groupTreeData"
            check-strictly
            :render-after-expand="false"
            placeholder="请选择资产分组"
            style="width: 100%"
            disabled
          />
        </el-form-item>

        <el-form-item label="主机">
          <el-radio-group v-model="editHostSelectionType" @change="handleEditHostTypeChange">
            <el-radio value="all">全部主机</el-radio>
            <el-radio value="specific">指定主机</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="editHostSelectionType === 'specific'" label="选择主机" prop="hostIds">
          <el-select
            v-model="editFormData.hostIds"
            multiple
            placeholder="请选择主机"
            style="width: 100%"
            :loading="editLoadingHosts"
          >
            <el-option
              v-for="host in editHostList"
              :key="host.id"
              :label="`${host.name} (${host.ip})`"
              :value="host.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="操作权限">
          <el-checkbox-group v-model="editFormData.permissions">
            <el-checkbox :value="1">查看 - 查看主机详情</el-checkbox>
            <el-checkbox :value="2">编辑 - 创建、修改主机配置</el-checkbox>
            <el-checkbox :value="4">删除 - 删除主机</el-checkbox>
            <el-checkbox :value="8">终端 - SSH连接主机</el-checkbox>
            <el-checkbox :value="16">文件 - 文件上传、下载、删除</el-checkbox>
            <el-checkbox :value="32">采集 - 采集主机系统信息</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleEditSubmit" :loading="editSubmitting">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Search, RefreshLeft, Lock, Edit } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  getAssetPermissions,
  createAssetPermission,
  deleteAssetPermission,
  getAssetPermissionDetail,
  updateAssetPermission
} from '@/api/assetPermission'
import { getAllRoles } from '@/api/role'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'

const loading = ref(false)
const permissions = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const deletingId = ref(0)

// 对话框相关
const dialogVisible = ref(false)
const editDialogVisible = ref(false)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const hostSelectionType = ref('all')
const loadingHosts = ref(false)
const selectedPermissions = ref<number[]>([1]) // 默认仅查看权限

// 编辑表单数据
const editFormData = reactive({
  id: null as number | null,
  roleId: null as number | null,
  assetGroupId: null as number | null,
  hostIds: [] as number[],
  permissions: [] as number[]
})
const editSubmitting = ref(false)
const editFormRef = ref<FormInstance>()
const editHostSelectionType = ref('all')
const editLoadingHosts = ref(false)
const editHostList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  roleName: '',
  groupName: ''
})

// 使用普通 ref 存储列表数据
const roleList = ref<any[]>([])
const groupTreeData = ref<any[]>([])
const hostList = ref<any[]>([])

// 表单数据 - 使用 reactive 以便更好地支持 v-model 绑定
const formData = reactive({
  roleId: null as number | null,
  assetGroupId: null as number | null,
  hostIds: [] as number[]
})

// 过滤后的权限列表
const filteredPermissions = computed(() => {
  let result = permissions.value

  if (searchForm.roleName) {
    result = result.filter(item =>
      item.roleName?.includes(searchForm.roleName)
    )
  }

  if (searchForm.groupName) {
    result = result.filter(item =>
      item.assetGroupName?.includes(searchForm.groupName)
    )
  }

  return result
})

// 表单验证规则
const formRules: FormRules = {
  roleId: [{ required: true, message: '请选择角色', trigger: 'change' }],
  assetGroupId: [{ required: true, message: '请选择资产分组', trigger: 'change' }]
}

// 加载权限列表
const loadPermissions = async () => {
  loading.value = true
  try {
    const response = await getAssetPermissions({
      page: page.value,
      pageSize: pageSize.value
    })
    permissions.value = response.list || []
    total.value = response.total || 0
  } catch (error: any) {
    ElMessage.error('加载权限列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 加载角色列表
const loadRoles = async () => {
  try {
    const response = await getAllRoles()
    // API 返回的字段是 ID (大写)，不是 id
    roleList.value = (response || []).map((item: any) => ({
      id: item.ID,
      name: item.name,
      code: item.code
    }))
  } catch (error: any) {
    ElMessage.error('加载角色列表失败: ' + (error.message || '未知错误'))
  }
}

// 加载资产分组树
const loadAssetGroupTree = async () => {
  try {
    const data = await getGroupTree()
    groupTreeData.value = convertTreeData(data || [])
  } catch (error: any) {
    ElMessage.error('加载资产分组失败: ' + (error.message || '未知错误'))
  }
}

// 转换树形数据格式
const convertTreeData = (nodes: any[]): any[] => {
  return nodes.map((node: any) => ({
    value: node.id,
    label: node.name,
    children: node.children ? convertTreeData(node.children) : undefined
  }))
}

// 加载主机列表
const loadHosts = async (groupId?: number) => {
  if (!groupId) return

  loadingHosts.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000, groupId })
    hostList.value = (response.list || []).map((item: any) => ({
      id: item.id,
      name: item.name,
      ip: item.ip
    }))
  } catch (error: any) {
    ElMessage.error('加载主机列表失败: ' + (error.message || '未知错误'))
  } finally {
    loadingHosts.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  page.value = 1
  loadPermissions()
}

// 重置搜索
const handleReset = () => {
  searchForm.roleName = ''
  searchForm.groupName = ''
  page.value = 1
  loadPermissions()
}

// 处理资产分组变化
const handleGroupChange = (value: number) => {
  formData.hostIds = []
  hostList.value = []
  if (value && hostSelectionType.value === 'specific') {
    loadHosts(value)
  }
}

// 处理主机类型变化
const handleHostTypeChange = (value: string) => {
  if (value === 'specific' && formData.assetGroupId) {
    loadHosts(formData.assetGroupId)
  } else {
    formData.hostIds = []
  }
}

// 添加权限
const handleAdd = () => {
  resetForm()
  dialogVisible.value = true
}

// 删除权限
const handleDeleteClick = (row: any) => {
  ElMessageBox.confirm('确定删除此权限吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await handleDelete(row.id)
  }).catch(() => {})
}

const handleDelete = async (id: number) => {
  deletingId.value = id
  try {
    await deleteAssetPermission(id)
    ElMessage.success('删除成功')
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingId.value = 0
  }
}

// 编辑权限
const handleEditClick = async (row: any) => {
  try {
    const detail = await getAssetPermissionDetail(row.id)

    editFormData.id = detail.id
    editFormData.roleId = detail.roleId
    editFormData.assetGroupId = detail.assetGroupId
    editFormData.hostIds = detail.hostIds || []
    editFormData.permissions = []

    // 根据权限位掩码设置checkbox
    if ((detail.permissions & 1) > 0) editFormData.permissions.push(1)
    if ((detail.permissions & 2) > 0) editFormData.permissions.push(2)
    if ((detail.permissions & 4) > 0) editFormData.permissions.push(4)
    if ((detail.permissions & 8) > 0) editFormData.permissions.push(8)
    if ((detail.permissions & 16) > 0) editFormData.permissions.push(16)
    if ((detail.permissions & 32) > 0) editFormData.permissions.push(32)

    // 设置主机选择类型
    editHostSelectionType.value = detail.isAllHosts ? 'all' : 'specific'

    // 加载主机列表
    if (!detail.isAllHosts) {
      await loadEditHosts(detail.assetGroupId)
    }

    editDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('加载权限详情失败: ' + (error.message || '未知错误'))
  }
}

// 加载编辑时的主机列表
const loadEditHosts = async (groupId?: number) => {
  if (!groupId) return

  editLoadingHosts.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000, groupId })
    editHostList.value = (response.list || []).map((item: any) => ({
      id: item.id,
      name: item.name,
      ip: item.ip
    }))
  } catch (error: any) {
    ElMessage.error('加载主机列表失败: ' + (error.message || '未知错误'))
  } finally {
    editLoadingHosts.value = false
  }
}

// 处理编辑时的主机类型变化
const handleEditHostTypeChange = (value: string) => {
  if (value === 'specific' && editFormData.assetGroupId) {
    loadEditHosts(editFormData.assetGroupId)
  } else {
    editFormData.hostIds = []
  }
}

// 关闭编辑对话框
const handleEditDialogClose = () => {
  editFormData.id = null
  editFormData.roleId = null
  editFormData.assetGroupId = null
  editFormData.hostIds = []
  editFormData.permissions = []
  editHostSelectionType.value = 'all'
  editHostList.value = []
  editFormRef.value?.clearValidate()
}

// 提交编辑
const handleEditSubmit = async () => {
  if (editFormData.id === null) return

  editSubmitting.value = true
  try {
    // 计算权限位掩码
    const permissions = editFormData.permissions.reduce((acc, val) => acc | val, 0)

    await updateAssetPermission(editFormData.id, {
      roleId: editFormData.roleId!,
      assetGroupId: editFormData.assetGroupId!,
      hostIds: editHostSelectionType.value === 'all' ? [] : editFormData.hostIds,
      permissions: permissions
    })
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('更新失败: ' + (error.message || '未知错误'))
  } finally {
    editSubmitting.value = false
  }
}

// 分页变化
const handleSizeChange = () => {
  page.value = 1
  loadPermissions()
}

const handlePageChange = () => {
  loadPermissions()
}

// 重置表单
const resetForm = () => {
  formData.roleId = null
  formData.assetGroupId = null
  formData.hostIds = []
  hostSelectionType.value = 'all'
  hostList.value = []
  selectedPermissions.value = [1] // 重置为仅查看权限
  formRef.value?.clearValidate()
}

// 关闭对话框
const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      // 计算权限位掩码
      const permissions = selectedPermissions.value.reduce((acc, val) => acc | val, 0)

      await createAssetPermission({
        roleId: formData.roleId!,
        assetGroupId: formData.assetGroupId!,
        hostIds: hostSelectionType.value === 'all' ? [] : formData.hostIds,
        permissions: permissions
      })
      ElMessage.success('添加成功')
      dialogVisible.value = false
      loadPermissions()
    } catch (error: any) {
      ElMessage.error('添加失败: ' + (error.message || '未知错误'))
    } finally {
      submitting.value = false
    }
  })
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadPermissions()
  loadRoles()
  loadAssetGroupTree()
})
</script>

<style scoped>
.permission-container {
  padding: 0;
  background-color: transparent;
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

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
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

.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e6f7ff;
  color: #1890ff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
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

/* 分页 */
.pagination-container {
  padding: 12px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
  border-radius: 0 0 12px 12px;
  display: flex;
  justify-content: flex-end;
}

/* 主机IP样式 */
.host-ip {
  font-size: 12px;
  color: #909399;
  font-family: 'Consolas', 'Monaco', monospace;
  margin-top: 4px;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.permission-dialog) {
  border-radius: 12px;
}

:deep(.permission-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.permission-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.permission-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 900px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }

  .search-input {
    width: auto;
    flex: 1;
    min-width: 200px;
  }

  .search-inputs {
    flex-direction: column;
  }
}

/* 权限标签样式 */
.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 权限表单提示 */
.permission-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}
</style>
