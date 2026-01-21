<template>
  <div class="roles-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h2 class="page-title">角色管理</h2>
          <p class="page-subtitle">管理系统角色权限，支持角色创建、编辑与权限分配</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增角色
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.name"
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
          v-model="searchForm.code"
          placeholder="搜索角色编码..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Key /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="角色状态"
          clearable
          class="search-input"
          @change="handleSearch"
        >
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredRoleList"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column prop="ID" label="ID" width="80" align="center" />

        <el-table-column label="角色名称" prop="name" min-width="150">
          <template #default="{ row }">
            <div class="role-name-cell">
              <el-icon class="role-icon"><UserFilled /></el-icon>
              <span class="role-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="code" min-width="150">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-gold"><Key /></el-icon>
              角色编码
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="description" label="描述" min-width="200">
          <template #default="{ row }">
            <span class="description-text">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="dark">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="createTime" label="创建时间" min-width="180" />

        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="授权" placement="top">
                <el-button link class="action-btn action-permission" @click="handlePermission(row)">
                  <el-icon><Setting /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
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
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          layout="total, prev, pager, next, jumper"
          @current-change="loadRoles"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="50%"
      class="role-edit-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="roleForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="roleForm.name" placeholder="请输入角色名称" />
        </el-form-item>

        <el-form-item label="角色编码" prop="code">
          <el-input v-model="roleForm.code" placeholder="请输入角色编码" />
        </el-form-item>

        <el-form-item label="显示顺序" prop="sort">
          <el-input-number v-model="roleForm.sort" :min="0" />
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="roleForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入角色描述"
          />
        </el-form-item>

        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="roleForm.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 授权对话框 -->
    <el-dialog
      v-model="permissionDialogVisible"
      title="角色授权"
      width="50%"
      class="permission-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handlePermissionDialogClose"
    >
      <el-alert
        title="提示"
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 20px;"
      >
        <template #default>
          为角色 <strong>{{ currentRole.name }}</strong> 分配菜单权限。勾选的菜单及其下属接口将被授权给该角色。
        </template>
      </el-alert>

      <div v-loading="menuLoading" style="min-height: 300px;">
        <el-tree
          ref="menuTreeRef"
          :data="menuTree"
          :props="{ label: 'name', children: 'children' }"
          show-checkbox
          node-key="ID"
          :default-expanded-keys="expandedKeys"
          :check-strictly="false"
          class="permission-tree"
        >
          <template #default="{ node, data }">
            <div class="tree-node">
              <el-icon v-if="data.type === 1" class="tree-node-icon folder-icon">
                <Folder />
              </el-icon>
              <el-icon v-else-if="data.type === 2" class="tree-node-icon menu-icon">
                <Menu />
              </el-icon>
              <el-icon v-else class="tree-node-icon button-icon">
                <Operation />
              </el-icon>
              <span class="tree-node-label">{{ data.name }}</span>
              <el-tag v-if="data.type === 1" size="small" type="info">目录</el-tag>
              <el-tag v-else-if="data.type === 2" size="small" type="success">菜单</el-tag>
              <el-tag v-else size="small" type="warning">按钮</el-tag>
            </div>
          </template>
        </el-tree>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="permissionDialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handlePermissionSubmit" :loading="submitting">保存授权</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  User,
  UserFilled,
  Search,
  RefreshLeft,
  Key,
  Setting,
  Folder,
  Menu,
  Operation
} from '@element-plus/icons-vue'
import { getRoleList, createRole, updateRole, deleteRole, getRoleMenus, assignRoleMenus } from '@/api/role'
import { getMenuTree } from '@/api/menu'

// 加载状态
const loading = ref(false)
const submitting = ref(false)
const menuLoading = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)

// 授权对话框状态
const permissionDialogVisible = ref(false)
const currentRole = ref<any>({})
const menuTree = ref<any[]>([])
const selectedMenuIds = ref<number[]>([])
const expandedKeys = ref<number[]>([])
const menuTreeRef = ref()

// 表单引用
const formRef = ref<FormInstance>()

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 角色列表
const roleList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  name: '',
  code: '',
  status: undefined as number | undefined
})

// 过滤后的角色列表
const filteredRoleList = computed(() => {
  let result = [...roleList.value]

  if (searchForm.name) {
    result = result.filter(item => item.name?.includes(searchForm.name))
  }

  if (searchForm.code) {
    result = result.filter(item => item.code?.includes(searchForm.code))
  }

  if (searchForm.status !== undefined) {
    result = result.filter(item => item.status === searchForm.status)
  }

  return result
})

// 角色表单
const roleForm = reactive({
  id: 0,
  name: '',
  code: '',
  description: '',
  status: 1,
  sort: 0
})

// 表单验证规则
const rules: FormRules = {
  name: [
    { required: true, message: '请输入角色名称', trigger: 'blur' },
    { min: 2, max: 50, message: '角色名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入角色编码', trigger: 'blur' },
    { min: 2, max: 50, message: '角色编码长度在 2 到 50 个字符', trigger: 'blur' }
  ]
}

// 搜索处理
const handleSearch = () => {
  // 搜索时自动回到第一页
  pagination.page = 1
}

// 重置搜索
const handleReset = () => {
  searchForm.name = ''
  searchForm.code = ''
  searchForm.status = undefined
}

// 加载角色列表
const loadRoles = async () => {
  loading.value = true
  try {
    const res = await getRoleList({
      page: pagination.page,
      pageSize: pagination.pageSize
    })
    roleList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    console.error('获取角色列表失败:', error)
    ElMessage.error('获取角色列表失败')
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  roleForm.id = 0
  roleForm.name = ''
  roleForm.code = ''
  roleForm.description = ''
  roleForm.status = 1
  roleForm.sort = 0
  formRef.value?.clearValidate()
}

// 新增角色
const handleAdd = () => {
  resetForm()
  dialogTitle.value = '新增角色'
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑角色
const handleEdit = (row: any) => {
  Object.assign(roleForm, {
    id: row.ID || row.id,
    name: row.name,
    code: row.code,
    description: row.description || '',
    status: row.status,
    sort: row.sort || 0
  })
  dialogTitle.value = '编辑角色'
  isEdit.value = true
  dialogVisible.value = true
}

// 删除角色
const handleDelete = async (row: any) => {
  ElMessageBox.confirm(`确定要删除角色"${row.name}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const roleId = row.ID || row.id
      await deleteRole(roleId)
      ElMessage.success('删除成功')
      loadRoles()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const data = { ...roleForm }

      if (isEdit.value) {
        await updateRole(data.id, data)
        ElMessage.success('更新成功')
      } else {
        await createRole(data)
        ElMessage.success('创建成功')
      }

      dialogVisible.value = false
      loadRoles()
    } catch (error: any) {
      ElMessage.error(error.message || (isEdit.value ? '更新失败' : '创建失败'))
    } finally {
      submitting.value = false
    }
  })
}

// 对话框关闭事件
const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

// 打开授权对话框
const handlePermission = async (row: any) => {
  currentRole.value = row
  permissionDialogVisible.value = true

  // 加载菜单树
  await loadMenuTree()

  // 加载角色已有权限（注意：后端返回的是大写ID）
  const roleId = row.ID || row.id
  await loadRoleMenus(roleId)
}

// 加载菜单树
const loadMenuTree = async () => {
  menuLoading.value = true
  try {
    const res: any = await getMenuTree()
    menuTree.value = res || []

    // 自动展开所有一级节点（使用大写ID）
    expandedKeys.value = menuTree.value.map((item: any) => item.ID || item.id).filter(id => id)
  } catch (error) {
    console.error('获取菜单树失败:', error)
    ElMessage.error('获取菜单树失败')
  } finally {
    menuLoading.value = false
  }
}

// 加载角色已分配的菜单
const loadRoleMenus = async (roleId: number) => {
  menuLoading.value = true
  try {
    const res: any = await getRoleMenus(roleId)
    console.log('角色菜单数据:', res)

    // 提取已分配的菜单ID
    const menus = res.menus || []
    console.log('已分配的菜单:', menus)

    // 获取所有菜单ID
    const allMenuIds = menus.map((m: any) => m.ID || m.id).filter((id: number) => id && id > 0)
    console.log('所有菜单IDs:', allMenuIds)

    selectedMenuIds.value = allMenuIds

    // 只设置叶子节点为选中状态，避免父节点被自动选中所有子节点
    // 从menuTree中获取所有菜单，判断哪些是叶子节点
    const leafMenuIds = getLeafMenuIdsFromTree(menuTree.value, allMenuIds)
    console.log('叶子节点IDs:', leafMenuIds)

    // 使用 nextTick 确保树形控件已渲染完成后再设置选中状态
    await nextTick()
    if (menuTreeRef.value) {
      // 只设置叶子节点为选中状态，el-tree会自动处理父节点的半选状态
      menuTreeRef.value.setCheckedKeys(leafMenuIds, false)
    }
  } catch (error) {
    console.error('获取角色菜单失败:', error)
    ElMessage.error('获取角色菜单失败')
  } finally {
    menuLoading.value = false
  }
}

// 从菜单树中获取叶子节点ID
const getLeafMenuIdsFromTree = (tree: any[], authorizedIds: number[]): number[] => {
  const leafIds: number[] = []

  const traverse = (nodes: any[]) => {
    nodes.forEach(node => {
      const nodeId = node.ID || node.id

      // 如果当前节点在授权列表中
      if (authorizedIds.includes(nodeId)) {
        // 如果没有子节点，或者子节点为空数组，则是叶子节点
        if (!node.children || node.children.length === 0) {
          leafIds.push(nodeId)
        } else {
          // 有子节点，递归处理子节点
          traverse(node.children)
        }
      }
    })
  }

  traverse(tree)
  return leafIds
}

// 获取叶子节点ID（避免父节点被自动选中）
const getLeafMenuIds = (menus: any[]): number[] => {
  if (!menus || menus.length === 0) return []

  const menuIds = menus.map((m: any) => m.ID || m.id).filter(id => id)
  const leafIds: number[] = []

  // 从菜单树中找出所有节点
  const allMenusFlat = flattenMenuTree(menuTree.value)

  // 判断每个已分配的菜单是否为叶子节点
  menuIds.forEach((menuId: number) => {
    // 检查这个菜单是否有子节点
    const hasChildren = allMenusFlat.some((m: any) => {
      const parentId = m.ParentID || m.parentId || m.parent_id
      return parentId === menuId
    })

    // 如果没有子节点，就是叶子节点
    if (!hasChildren) {
      leafIds.push(menuId)
    }
  })

  return leafIds
}

// 扁平化菜单树
const flattenMenuTree = (tree: any[]): any[] => {
  const result: any[] = []
  const traverse = (nodes: any[]) => {
    nodes.forEach((node: any) => {
      result.push(node)
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    })
  }
  traverse(tree)
  return result
}

// 提交授权
const handlePermissionSubmit = async () => {
  if (!menuTreeRef.value) return

  submitting.value = true
  try {
    // 获取选中的节点（包括半选中的父节点）
    const checkedKeys = menuTreeRef.value.getCheckedKeys()
    const halfCheckedKeys = menuTreeRef.value.getHalfCheckedKeys()
    console.log('完全选中的节点:', checkedKeys)
    console.log('半选中的节点:', halfCheckedKeys)

    const allKeys = [...checkedKeys, ...halfCheckedKeys]
    console.log('合并后的节点:', allKeys)

    // 过滤掉无效的ID（0或undefined）并去重
    const validKeys = [...new Set(allKeys.filter((key: number) => key && key > 0))]
    console.log('有效的节点:', validKeys)

    // 注意：后端返回的是大写ID
    const roleId = currentRole.value.ID || currentRole.value.id
    console.log('提交授权 - 角色ID:', roleId, '菜单IDs:', validKeys)

    await assignRoleMenus(roleId, validKeys)
    ElMessage.success('授权成功')
    permissionDialogVisible.value = false
  } catch (error: any) {
    console.error('授权失败:', error)
    ElMessage.error(error.message || '授权失败')
  } finally {
    submitting.value = false
  }
}

// 授权对话框关闭事件
const handlePermissionDialogClose = () => {
  currentRole.value = {}
  menuTree.value = []
  selectedMenuIds.value = []
  expandedKeys.value = []

  // 清空树形控件的选中状态
  if (menuTreeRef.value) {
    menuTreeRef.value.setCheckedKeys([])
  }
}

onMounted(() => {
  loadRoles()
})
</script>

<style scoped>
.roles-container {
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

/* 搜索框样式 */
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

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-gold {
  color: #d4af37;
}

/* 角色名称单元格 */
.role-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-icon {
  font-size: 18px;
  color: #409eff;
  flex-shrink: 0;
}

.role-name {
  font-weight: 500;
}

.description-text {
  color: #606266;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
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
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.action-permission:hover {
  background-color: #f0f9ff;
  color: #67C23A;
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

/* 分页器 */
.pagination-container {
  padding: 12px 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.role-edit-dialog) {
  border-radius: 12px;
}

:deep(.role-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.role-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.role-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.el-input__wrapper),
:deep(.el-textarea__inner) {
  border-radius: 8px;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
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
}

/* 授权对话框样式 */
.permission-dialog :deep(.el-dialog__body) {
  padding: 24px;
}

.permission-tree {
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  padding: 12px;
  max-height: 500px;
  overflow-y: auto;
}

.permission-tree :deep(.el-tree-node__content) {
  height: 36px;
  margin-bottom: 4px;
  border-radius: 6px;
}

.permission-tree :deep(.el-tree-node__content:hover) {
  background-color: #f5f7fa;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.tree-node-icon {
  font-size: 16px;
}

.folder-icon {
  color: #E6A23C;
}

.menu-icon {
  color: #409EFF;
}

.button-icon {
  color: #67C23A;
}

.tree-node-label {
  flex: 1;
  font-size: 14px;
  color: #303133;
}
</style>
