<template>
  <div class="users-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h2 class="page-title">用户管理</h2>
          <p class="page-subtitle">统一维护平台账号、组织归属、岗位与角色权限</p>
        </div>
      </div>
      <el-button class="black-button" @click="handleAdd">
        <el-icon><Plus /></el-icon>
        新增用户
      </el-button>
    </div>

    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索用户名、姓名或邮箱..."
          clearable
          class="search-input"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
        <div v-if="selectedDepartment" class="active-filter">
          <span>组织范围</span>
          <strong>{{ selectedDepartmentPath }}</strong>
          <el-button link :icon="Close" aria-label="清除部门筛选" @click="clearDepartmentSelection" />
        </div>
      </div>
      <div class="search-actions">
        <el-button class="black-button query-button" @click="handleSearch">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
        <el-button class="reset-btn" @click="resetSearch">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <div class="users-workspace">
      <aside class="dept-filter-panel">
        <div class="panel-header">
          <div>
            <strong>部门组织</strong>
            <span>按组织快速筛选用户</span>
          </div>
          <el-icon><OfficeBuilding /></el-icon>
        </div>
        <button
          type="button"
          class="all-users-button"
          :class="{ active: !selectedDepartment }"
          @click="clearDepartmentSelection"
        >
          <span><el-icon><UserFilled /></el-icon>全部用户</span>
          <strong>{{ allUserTotal }}</strong>
        </button>
        <el-tree
          ref="treeRef"
          :data="departmentTree"
          :props="treeProps"
          :highlight-current="true"
          node-key="id"
          default-expand-all
          @node-click="handleNodeClick"
          class="dept-tree"
        >
          <template #default="{ node, data }">
            <span class="custom-tree-node">
              <span class="node-main">
                <el-icon class="node-type-icon">
                  <OfficeBuilding v-if="data.deptType === 1" />
                  <Location v-else-if="data.deptType === 2" />
                  <Folder v-else />
                </el-icon>
                <span class="node-label">{{ node.label }}</span>
              </span>
              <span class="node-count">{{ data.userCount || 0 }}</span>
            </span>
          </template>
        </el-tree>
      </aside>

      <section class="table-wrapper">
        <div class="table-toolbar">
          <div>
            <strong>{{ selectedDepartment ? selectedDepartmentPath : '全部用户' }}</strong>
            <span>{{ selectedDepartment ? '当前组织及下级组织账号' : '平台全部账号' }}</span>
          </div>
          <el-tag effect="plain" type="info">共 {{ pagination.total }} 人</el-tag>
        </div>

        <el-table
          :data="userList"
          v-loading="loading"
          class="modern-table"
          :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        >
          <el-table-column label="用户" min-width="230">
            <template #default="{ row }">
              <div class="user-identity">
                <el-avatar v-if="row.avatar" :src="row.avatar" :size="38" />
                <el-avatar v-else :size="38" class="user-avatar">{{ userInitial(row) }}</el-avatar>
                <div class="user-identity-text">
                  <div>
                    <strong>{{ row.realName || row.username }}</strong>
                    <el-tag v-if="row.source === 'ldap'" size="small" type="info" effect="plain">LDAP</el-tag>
                  </div>
                  <span>@{{ row.username }}</span>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="联系方式" min-width="220">
            <template #default="{ row }">
              <div class="contact-cell">
                <span><el-icon><Message /></el-icon>{{ row.email || '-' }}</span>
                <span><el-icon><Phone /></el-icon>{{ row.phone || '未填写手机号' }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="部门" min-width="150">
            <template #default="{ row }">
              <div class="department-cell">
                <el-icon><OfficeBuilding /></el-icon>
                <span>{{ row.department?.name || row.department?.deptName || '未分配部门' }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="角色与岗位" min-width="220">
            <template #default="{ row }">
              <div class="assignment-cell">
                <div v-if="roleNames(row).length > 0">
                  <span class="assignment-label">角色</span>
                  <el-tag v-for="name in roleNames(row).slice(0, 2)" :key="`role-${name}`" size="small" effect="plain">{{ name }}</el-tag>
                  <small v-if="roleNames(row).length > 2">+{{ roleNames(row).length - 2 }}</small>
                </div>
                <div v-if="positionNames(row).length > 0">
                  <span class="assignment-label">岗位</span>
                  <span>{{ positionNames(row).join('、') }}</span>
                </div>
                <span v-if="roleNames(row).length === 0 && positionNames(row).length === 0" class="empty-text">未分配</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.isLocked" type="warning" effect="dark">锁定中</el-tag>
              <el-tag v-else :type="row.status === 1 ? 'success' : 'danger'" effect="dark">
                {{ row.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="190" fixed="right" align="center">
            <template #default="{ row }">
              <div class="action-buttons">
                <el-tooltip v-if="row.isLocked" content="解锁" placement="top">
                  <el-button link class="action-btn action-unlock" @click="handleUnlock(row)"><el-icon><Unlock /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="编辑" placement="top">
                  <el-button link class="action-btn action-edit" @click="handleEdit(row)"><el-icon><Edit /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip :content="row.source === 'ldap' ? 'LDAP 用户无法重置密码' : '重置密码'" placement="top">
                  <el-button link class="action-btn action-key" :disabled="row.source === 'ldap'" @click="handleResetPassword(row)"><el-icon><Key /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button link class="action-btn action-delete" @click="handleDelete(row)"><el-icon><Delete /></el-icon></el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-container">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="loadUsers"
            @current-change="loadUsers"
          />
        </div>
      </section>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="55%"
      class="user-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="userForm" :rules="rules" ref="formRef" label-width="104px" class="user-form">
        <!-- 基本信息 -->
        <div class="form-section-title">
          <el-icon><User /></el-icon>
          <span>基本信息</span>
        </div>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="userForm.username" :disabled="isEdit" placeholder="请输入用户名">
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="真实姓名" prop="realName">
              <el-input v-model="userForm.realName" placeholder="请输入真实姓名">
                <template #prefix>
                  <el-icon><Postcard /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="userForm.email" placeholder="请输入邮箱">
                <template #prefix>
                  <el-icon><Message /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="手机号" prop="phone">
              <el-input v-model="userForm.phone" placeholder="请输入手机号">
                <template #prefix>
                  <el-icon><Phone /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16" v-if="!isEdit">
          <el-col :span="12">
            <el-form-item label="密码" prop="password">
              <el-input v-model="userForm.password" type="password" show-password placeholder="请输入密码（至少6位）">
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态" prop="status">
              <el-radio-group v-model="userForm.status">
                <el-radio :label="1">启用</el-radio>
                <el-radio :label="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16" v-if="isEdit">
          <el-col :span="12">
            <el-form-item label="状态" prop="status">
              <el-radio-group v-model="userForm.status">
                <el-radio :label="1">启用</el-radio>
                <el-radio :label="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <div class="form-section-title">
          <el-icon><Message /></el-icon>
          <span>通知标识</span>
        </div>

        <el-row :gutter="16">
          <el-col :span="24">
            <el-form-item label="用户标识">
              <el-input
                v-model="userForm.notifyUserId"
                placeholder="飞书填 open_id，如 ou_xxx；钉钉填手机号或 userId；企业微信填账号"
              >
                <template #prefix>
                  <el-icon><Message /></el-icon>
                </template>
              </el-input>
              <div class="form-tip">
                用于值班表通知时 @ 对应人员。飞书外部群通常填写 ou_ 开头的 open_id；钉钉未接通讯录时建议填写手机号，接入企业通讯录/内部机器人时可填写钉钉 userId；企业微信填写成员账号或手机号。无法被平台解析时会降级显示为 @姓名。
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 组织信息 -->
        <div class="form-section-title">
          <el-icon><OfficeBuilding /></el-icon>
          <span>组织信息</span>
        </div>

        <el-form-item label="部门" prop="departmentId">
          <el-tree-select
            v-model="userForm.departmentId"
            :data="departmentTreeData"
            :props="{ label: 'label', value: 'id', children: 'children' }"
            placeholder="请选择部门"
            clearable
            check-strictly
            :render-after-expand="false"
          >
            <template #default="{ data }">
              <span>{{ data.label }}</span>
              <span class="tree-node-count">({{ data.userCount || 0 }})</span>
            </template>
          </el-tree-select>
        </el-form-item>

        <el-form-item label="岗位" prop="positionIds">
          <el-select
            v-model="userForm.positionIds"
            multiple
            placeholder="请选择岗位"
            style="width: 100%"
          >
            <el-option
              v-for="pos in positionOptions"
              :key="'pos-' + (pos.ID || pos.id)"
              :label="pos.postName"
              :value="pos.ID || pos.id"
            />
          </el-select>
        </el-form-item>

        <!-- 权限信息 -->
        <div class="form-section-title">
          <el-icon><Key /></el-icon>
          <span>权限信息</span>
        </div>

        <el-form-item label="角色" prop="roleIds">
          <el-select
            v-model="userForm.roleIds"
            multiple
            placeholder="请选择角色"
            style="width: 100%"
          >
            <el-option
              v-for="role in roleOptions"
              :key="'role-' + role.ID"
              :label="role.name"
              :value="role.ID"
            >
              <span>{{ role.name }}</span>
              <span class="role-code">{{ role.code }}</span>
            </el-option>
          </el-select>
        </el-form-item>

        <!-- 其他信息 -->
        <div class="form-section-title">
          <el-icon><Document /></el-icon>
          <span>其他信息</span>
        </div>

        <el-form-item label="个人简介" prop="bio">
          <el-input
            v-model="userForm.bio"
            type="textarea"
            :rows="3"
            placeholder="请输入个人简介"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitLoading">
            <el-icon><Check /></el-icon>
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="resetPasswordVisible"
      title="重置密码"
      width="40%"
      class="user-password-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleResetPasswordClose"
    >
      <el-form :model="resetPasswordForm" :rules="resetPasswordRules" ref="resetPasswordFormRef" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="resetPasswordForm.username" disabled />
        </el-form-item>
        <el-form-item label="新密码" prop="password">
          <el-input v-model="resetPasswordForm.password" type="password" show-password placeholder="请输入新密码（至少6位）" />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="resetPasswordForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码" />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="resetPasswordVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleResetPasswordSubmit" :loading="resetPasswordLoading">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance } from 'element-plus'
import {
  User, UserFilled, Postcard, Message, Phone, Lock, Plus, Search, RefreshLeft, Close,
  OfficeBuilding, Location, Folder, Key, Document, Check, Edit, Delete, Unlock
} from '@element-plus/icons-vue'
import { getUserList, createUser, updateUser, deleteUser, resetUserPassword, assignUserRoles, assignUserPositions, unlockUser } from '@/api/user'
import { getDepartmentTree } from '@/api/department'
import { getAllRoles } from '@/api/role'
import { getPositionList } from '@/api/position'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const treeRef = ref()

// 部门树相关
const departmentTree = ref([])
const treeProps = {
  children: 'children',
  label: 'deptName'
}
const selectedDepartment = ref<any>(null)
const selectedDepartmentPath = ref('')

// 角色和岗位选项
const roleOptions = ref([])
const positionOptions = ref([])

// 处理部门树数据，转换为el-tree-select需要的格式
const departmentTreeData = computed(() => {
  const convertTree = (nodes: any[]): any[] => {
    return nodes.map(node => ({
      id: node.id,
      label: node.deptName || node.name,
      userCount: node.userCount || 0,
      children: node.children ? convertTree(node.children) : []
    }))
  }
  return convertTree(departmentTree.value)
})

// 重置密码相关
const resetPasswordVisible = ref(false)
const resetPasswordLoading = ref(false)
const resetPasswordFormRef = ref<FormInstance>()
const resetPasswordForm = reactive({
  userId: 0,
  username: '',
  password: '',
  confirmPassword: ''
})

const searchForm = reactive({
  keyword: '',
  departmentId: null as number | null
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const allUserTotal = ref(0)
const userList = ref([])

const userInitial = (row: any) => {
  const name = row.realName || row.username || '?'
  return name.substring(0, 1).toUpperCase()
}

const roleNames = (row: any): string[] => {
  return (row.roles || []).map((role: any) => role.name).filter(Boolean)
}

const positionNames = (row: any): string[] => {
  return (row.positions || []).map((position: any) => position.postName || position.name).filter(Boolean)
}

const userForm = reactive({
  id: 0,
  username: '',
  password: '',
  realName: '',
  email: '',
  phone: '',
  notifyUserId: '',
  feishuUserId: '',
  feishuOpenId: '',
  dingtalkUserId: '',
  wecomUserId: '',
  status: 1,
  departmentId: null as number | null,
  positionIds: [] as number[],
  roleIds: [] as number[],
  bio: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }]
}

const resetPasswordRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule: any, value: any, callback: any) => {
        if (value !== resetPasswordForm.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await getUserList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      departmentId: searchForm.departmentId
    })
    userList.value = res.list || []
    pagination.total = res.total || 0
    if (!searchForm.departmentId && !searchForm.keyword) {
      allUserTotal.value = pagination.total
    }
  } catch (error) {
  } finally {
    loading.value = false
  }
}

// 加载部门树
const loadDepartmentTree = async () => {
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res || []
  } catch (error) {
  }
}

// 加载角色选项
const loadRoleOptions = async () => {
  try {
    roleOptions.value = []  // 先清空
    const res = await getAllRoles()
    roleOptions.value = res || []
  } catch (error) {
  }
}

// 加载岗位选项
const loadPositionOptions = async () => {
  try {
    positionOptions.value = []  // 先清空
    const res = await getPositionList({ page: 1, pageSize: 1000 })
    const list = res.list || []
    if (list.length > 0) {
    }
    positionOptions.value = list
  } catch (error) {
  }
}

// 构建部门路径
const buildDepartmentPath = (node: any, path: string[] = []): string => {
  path.unshift(node.deptName || node.name)
  if (node.parentId && departmentTree.value) {
    const findParent = (nodes: any[], id: number): any => {
      for (const n of nodes) {
        if (n.id === id) return n
        if (n.children) {
          const found = findParent(n.children, id)
          if (found) return found
        }
      }
      return null
    }
    const parent = findParent(departmentTree.value, node.parentId)
    if (parent) {
      return buildDepartmentPath(parent, path)
    }
  }
  return path.join(' / ')
}

// 处理部门节点点击
const handleNodeClick = (data: any) => {
  selectedDepartment.value = data
  selectedDepartmentPath.value = buildDepartmentPath(data)
  searchForm.departmentId = data.id
  pagination.page = 1
  loadUsers()
}

// 清除部门选择
const clearDepartmentSelection = () => {
  selectedDepartment.value = null
  selectedDepartmentPath.value = ''
  searchForm.departmentId = null
  treeRef.value?.setCurrentKey(null)
  pagination.page = 1
  loadUsers()
}

const handleSearch = () => {
  pagination.page = 1
  loadUsers()
}

const resetSearch = () => {
  searchForm.keyword = ''
  clearDepartmentSelection()
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新增用户'
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  dialogTitle.value = '编辑用户'

  // 正确处理ID字段，兼容大小写
  userForm.id = Number(row.ID || row.id)
  userForm.username = row.username
  userForm.realName = row.realName || ''
  userForm.email = row.email || ''
  userForm.phone = row.phone || ''
  userForm.notifyUserId = row.notifyUserId || row.feishuOpenId || row.feishuUserId || row.dingtalkUserId || row.wecomUserId || ''
  userForm.feishuUserId = row.feishuUserId || ''
  userForm.feishuOpenId = row.feishuOpenId || ''
  userForm.dingtalkUserId = row.dingtalkUserId || ''
  userForm.wecomUserId = row.wecomUserId || ''
  userForm.status = row.status ?? 1
  userForm.departmentId = row.departmentId ? Number(row.departmentId) : null

  // 处理岗位ID
  if (row.positionIds && Array.isArray(row.positionIds)) {
    userForm.positionIds = row.positionIds.map((id: any) => Number(id))
  } else if (row.positions && Array.isArray(row.positions) && row.positions.length > 0) {
    userForm.positionIds = row.positions.map((p: any) => Number(p.ID || p.id))
  } else {
    userForm.positionIds = []
  }

  // 处理角色ID
  if (row.roleIds && Array.isArray(row.roleIds)) {
    userForm.roleIds = row.roleIds
  } else if (row.roles && Array.isArray(row.roles) && row.roles.length > 0) {
    userForm.roleIds = row.roles.map((r: any) => r.ID || r.id)
  } else {
    userForm.roleIds = []
  }


  userForm.bio = row.bio || ''
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该用户吗？', '提示', {
      type: 'warning'
    })
    await deleteUser(row.ID || row.id)
    ElMessage.success('删除成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
    }
  }
}

const handleResetPassword = (row: any) => {
  resetPasswordForm.userId = row.ID || row.id
  resetPasswordForm.username = row.username
  resetPasswordForm.password = ''
  resetPasswordForm.confirmPassword = ''
  resetPasswordVisible.value = true
}

const handleUnlock = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要解锁用户 "${row.username}" 吗？`, '提示', {
      type: 'warning'
    })
    await unlockUser(row.ID || row.id)
    ElMessage.success('用户已解锁')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
    }
  }
}

const handleResetPasswordSubmit = async () => {
  if (!resetPasswordFormRef.value) return

  await resetPasswordFormRef.value.validate(async (valid) => {
    if (valid) {
      resetPasswordLoading.value = true
      try {
        await resetUserPassword(resetPasswordForm.userId, resetPasswordForm.password)
        ElMessage.success('密码重置成功')
        resetPasswordVisible.value = false
      } catch (error) {
      } finally {
        resetPasswordLoading.value = false
      }
    }
  })
}

const handleResetPasswordClose = () => {
  resetPasswordFormRef.value?.resetFields()
  Object.assign(resetPasswordForm, {
    userId: 0,
    username: '',
    password: '',
    confirmPassword: ''
  })
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        // 保存角色ID和岗位ID，过滤掉null值
        const roleIds = (userForm.roleIds || []).filter((id: any) => id != null)
        const positionIds = (userForm.positionIds || []).filter((id: any) => id != null)

        if (isEdit.value) {
          // 清理userForm中的null值，避免发送到后端
          const userData = {
            ...userForm,
            feishuUserId: userForm.notifyUserId,
            feishuOpenId: userForm.notifyUserId,
            dingtalkUserId: userForm.notifyUserId,
            wecomUserId: userForm.notifyUserId,
            positionIds: positionIds,
            roleIds: roleIds
          }

          // 更新用户基本信息
          await updateUser(userForm.id, userData)

          // 分配角色（传空数组表示清空角色）
          await assignUserRoles(userForm.id, roleIds)

          // 分配岗位（传空数组表示清空岗位）
          await assignUserPositions(userForm.id, positionIds)

          ElMessage.success('更新成功')
        } else {
          // 创建新用户
          await createUser({
            ...userForm,
            feishuUserId: userForm.notifyUserId,
            feishuOpenId: userForm.notifyUserId,
            dingtalkUserId: userForm.notifyUserId,
            wecomUserId: userForm.notifyUserId
          })

          // 分配角色
          if (roleIds.length > 0) {
            // 获取刚创建的用户ID，这里需要从响应中获取
            // 暂时先跳过，需要后端返回创建的用户ID
          }

          ElMessage.success('创建成功')
        }

        dialogVisible.value = false
        loadUsers()
      } catch (error) {
        ElMessage.error('操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
  Object.assign(userForm, {
    id: 0,
    username: '',
    password: '',
    realName: '',
    email: '',
    phone: '',
    notifyUserId: '',
    feishuUserId: '',
    feishuOpenId: '',
    dingtalkUserId: '',
    wecomUserId: '',
    status: 1,
    departmentId: null,
    positionIds: [],
    roleIds: [],
    bio: ''
  })
}

onMounted(() => {
  loadDepartmentTree()
  loadRoleOptions()
  loadPositionOptions()
  loadUsers()
})
</script>

<style scoped>
.users-container {
  padding: 0;
  background: transparent;
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
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #d4af37;
  font-size: 22px;
  background: #111;
  border: 1px solid #d4af37;
  border-radius: 8px;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0;
  color: #909399;
  font-size: 13px;
  line-height: 1.4;
}

.black-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 18px;
  color: #fff !important;
  font-weight: 500;
  background: #000 !important;
  border-color: #000 !important;
  border-radius: 6px;
}

.black-button:hover,
.black-button:focus {
  color: #fff !important;
  background: #333 !important;
  border-color: #333 !important;
}

.search-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-inputs {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.search-input {
  width: 360px;
  max-width: 100%;
}

.search-icon {
  color: #d4af37;
}

.active-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  height: 32px;
  padding: 0 6px 0 12px;
  color: #606266;
  font-size: 12px;
  background: #fff9e8;
  border: 1px solid #ead58b;
  border-radius: 6px;
}

.active-filter strong {
  max-width: 260px;
  overflow: hidden;
  color: #303133;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.query-button {
  padding: 8px 16px;
}

.reset-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: #606266;
  background: #f5f7fa;
  border-color: #dcdfe6;
}

.users-workspace {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  align-items: stretch;
  gap: 12px;
}

.dept-filter-panel {
  min-height: 520px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.panel-header {
  min-height: 66px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.panel-header > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.panel-header strong {
  color: #303133;
  font-size: 15px;
}

.panel-header span {
  color: #909399;
  font-size: 12px;
}

.panel-header > .el-icon {
  color: #d4af37;
  font-size: 20px;
}

.all-users-button {
  width: calc(100% - 20px);
  min-height: 40px;
  margin: 10px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #606266;
  font: inherit;
  background: transparent;
  border: 0;
  border-radius: 6px;
  cursor: pointer;
}

.all-users-button span {
  display: flex;
  align-items: center;
  gap: 8px;
}

.all-users-button strong {
  min-width: 24px;
  color: #909399;
  font-size: 12px;
  text-align: right;
}

.all-users-button:hover {
  background: #f5f7fa;
}

.all-users-button.active {
  color: #1f1f1f;
  font-weight: 600;
  background: #fff7df;
}

.dept-tree {
  flex: 1;
  padding: 0 10px 12px;
  overflow-y: auto;
  background: transparent;
  font-size: 14px;
}

.dept-tree :deep(.el-tree-node__content) {
  height: 38px;
  margin-bottom: 2px;
  border-radius: 6px;
}

.dept-tree :deep(.el-tree-node__content:hover) {
  background: #f5f7fa;
}

.dept-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  color: #1f1f1f;
  font-weight: 600;
  background: #fff7df;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-width: 0;
  padding-right: 6px;
}

.node-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
}

.node-type-icon {
  flex-shrink: 0;
  color: #7d8795;
}

.node-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-count {
  min-width: 24px;
  color: #909399;
  font-size: 12px;
  text-align: right;
}

.table-wrapper {
  min-width: 0;
  overflow: hidden;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.table-toolbar {
  min-height: 66px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.table-toolbar > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.table-toolbar strong {
  color: #303133;
  font-size: 15px;
}

.table-toolbar span {
  color: #909399;
  font-size: 12px;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: #fafbfc !important;
}

.user-identity {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-avatar {
  color: #d4af37;
  font-weight: 600;
  background: #171717;
}

.user-identity-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.user-identity-text > div {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
}

.user-identity-text strong {
  overflow: hidden;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-identity-text > span {
  color: #909399;
  font-size: 12px;
}

.contact-cell,
.assignment-cell {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.contact-cell span {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  color: #606266;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contact-cell .el-icon,
.department-cell .el-icon {
  flex-shrink: 0;
  color: #9aa3af;
}

.department-cell {
  display: flex;
  align-items: center;
  gap: 7px;
  color: #606266;
}

.assignment-cell > div {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  color: #606266;
  font-size: 12px;
}

.assignment-label {
  width: 30px;
  flex-shrink: 0;
  color: #909399;
}

.assignment-cell small {
  color: #909399;
}

.empty-text {
  color: #b2b8c2;
  font-size: 13px;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.action-btn {
  width: 30px;
  height: 30px;
  margin: 0 !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #7d8795;
  border-radius: 6px;
  transition: color 0.2s ease, background-color 0.2s ease, transform 0.2s ease;
}

.action-btn:not(.is-disabled):hover {
  transform: translateY(-1px);
}

.action-edit:hover {
  color: #409eff;
  background: #ecf5ff;
}

.action-key:hover,
.action-unlock:hover {
  color: #b88218;
  background: #fdf6ec;
}

.action-delete:hover {
  color: #f56c6c;
  background: #fef0f0;
}

.pagination-container {
  min-height: 58px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

:deep(.user-dialog),
:deep(.user-password-dialog) {
  border-radius: 8px;
}

:deep(.user-dialog .el-dialog__header),
:deep(.user-password-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.user-dialog .el-dialog__body) {
  max-height: 68vh;
  padding: 20px 24px 24px;
  overflow-y: auto;
}

:deep(.user-password-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.user-dialog .el-dialog__footer),
:deep(.user-password-dialog .el-dialog__footer) {
  padding: 14px 24px 18px;
  border-top: 1px solid #f0f0f0;
}

.user-form {
  width: 100%;
}

.form-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 24px 0 18px;
  padding-bottom: 10px;
  border-bottom: 1px solid #ebeef5;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.form-section-title:first-child {
  margin-top: 0;
}

.form-section-title .el-icon {
  color: #d4af37;
  font-size: 17px;
}

.form-tip {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}

.tree-node-count {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}

.role-code {
  margin-left: 12px;
  color: #909399;
  font-size: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer .el-button {
  display: flex;
  align-items: center;
  gap: 6px;
}

.user-form :deep(.el-input__prefix) {
  color: #a8abb2;
}

:deep(.el-input__wrapper),
:deep(.el-textarea__inner),
:deep(.el-select__wrapper) {
  border-radius: 6px;
}

:deep(.el-tag) {
  border-radius: 5px;
  font-weight: 500;
}

:deep(.responsive-dialog) {
  max-width: 960px;
  min-width: 620px;
}

:deep(.user-password-dialog) {
  max-width: 520px;
}

@media (max-width: 1200px) {
  .users-workspace {
    grid-template-columns: 1fr;
  }

  .dept-filter-panel {
    min-height: 0;
  }

  .dept-tree {
    max-height: 240px;
  }
}

@media (max-width: 768px) {
  .page-header,
  .search-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .page-header .black-button {
    align-self: flex-end;
  }

  .search-inputs {
    align-items: stretch;
    flex-direction: column;
  }

  .search-input,
  .active-filter {
    width: 100%;
  }

  .active-filter strong {
    max-width: none;
    flex: 1;
  }

  .search-actions {
    justify-content: flex-end;
  }

  .pagination-container {
    overflow-x: auto;
    justify-content: flex-start;
  }

  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }

  :deep(.responsive-dialog) {
    width: 95% !important;
    min-width: auto;
  }
}
</style>
