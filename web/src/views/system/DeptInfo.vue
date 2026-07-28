<template>
  <div class="dept-info-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><OfficeBuilding /></el-icon>
        </div>
        <div>
          <h2 class="page-title">部门信息</h2>
          <p class="page-subtitle">管理组织架构，支持公司、中心、部门三级层级结构</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增部门
        </el-button>
        <el-button @click="toggleExpandAll">
          <el-icon style="margin-right: 6px;"><Sort /></el-icon>
          {{ isExpandAll ? '折叠全部' : '展开全部' }}
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.deptName"
          placeholder="搜索部门名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.deptStatus"
          placeholder="部门状态"
          clearable
          class="search-input"
        >
          <el-option label="正常" :value="1" />
          <el-option label="停用" :value="0" />
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
        ref="tableRef"
        :data="filteredDeptTree"
        v-loading="loading"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        :default-expand-all="isExpandAll"
        :indent="30"
        class="modern-table dept-tree-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="部门名称" prop="deptName" min-width="350">
          <template #default="{ row }">
            <span style="display: inline-flex; align-items: center;">
              <el-icon v-if="row.deptType === 1" style="color: #409eff; margin-right: 8px;"><OfficeBuilding /></el-icon>
              <el-icon v-else-if="row.deptType === 2" style="color: #67c23a; margin-right: 8px;"><Location /></el-icon>
              <el-icon v-else style="color: #e6a23c; margin-right: 8px;"><Folder /></el-icon>
              {{ row.deptName }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="code" min-width="120">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-gold"><Key /></el-icon>
              部门编码
            </span>
          </template>
        </el-table-column>

        <el-table-column label="部门类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.deptType === 1" class="dept-type-tag company-tag">公司</el-tag>
            <el-tag v-else-if="row.deptType === 2" class="dept-type-tag center-tag">中心</el-tag>
            <el-tag v-else class="dept-type-tag dept-tag">部门</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.deptStatus === 1 ? 'success' : 'danger'" effect="dark">
              {{ row.deptStatus === 1 ? '正常' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createTime" min-width="180" />

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
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
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="50%"
      class="dept-edit-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="deptForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="上级部门" prop="parentId" v-if="showParentSelect">
          <el-cascader
            v-model="parentPath"
            :options="filteredParentOptions"
            :props="cascaderProps"
            clearable
            placeholder="请选择上级部门"
            style="width: 100%"
            @change="handleParentChange"
          />
          <div class="form-tip" v-if="deptForm.deptType === 2">中心的上级只能是公司</div>
          <div class="form-tip" v-else-if="deptForm.deptType === 3">部门的上级可以是公司或中心</div>
        </el-form-item>

        <el-form-item label="部门类型" prop="deptType">
          <el-radio-group v-model="deptForm.deptType" :disabled="deptTypeDisabled" @change="handleDeptTypeChange">
            <el-radio :label="1">公司</el-radio>
            <el-radio :label="2">中心</el-radio>
            <el-radio :label="3">部门</el-radio>
          </el-radio-group>
          <div class="form-tip">部门类型决定层级关系：公司 > 中心 > 部门</div>
        </el-form-item>

        <el-form-item label="部门名称" prop="deptName">
          <el-input v-model="deptForm.deptName" placeholder="请输入部门名称" />
        </el-form-item>

        <el-form-item label="部门编码" prop="code">
          <el-input v-model="deptForm.code" placeholder="请输入部门编码" />
        </el-form-item>

        <el-form-item label="显示顺序" prop="sort">
          <el-input-number v-model="deptForm.sort" :min="0" />
        </el-form-item>

        <el-form-item label="状态" prop="deptStatus">
          <el-radio-group v-model="deptForm.deptStatus">
            <el-radio :label="1">正常</el-radio>
            <el-radio :label="0">停用</el-radio>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules, ElTable } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  OfficeBuilding,
  Location,
  Folder,
  Search,
  RefreshLeft,
  Sort,
  Key
} from '@element-plus/icons-vue'
import {
  getDepartmentTree,
  getParentOptions,
  createDepartment,
  updateDepartment,
  deleteDepartment
} from '@/api/department'

// 加载状态
const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)

// 表单引用
const formRef = ref<FormInstance>()
const tableRef = ref<InstanceType<typeof ElTable>>()

// 部门树数据
const deptTree = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  deptName: '',
  deptStatus: undefined as number | undefined
})

// 展开/折叠状态
const isExpandAll = ref(true)

// 过滤后的部门树
const filteredDeptTree = computed(() => {
  if (!searchForm.deptName && searchForm.deptStatus === undefined) {
    return deptTree.value
  }
  return filterTree(deptTree.value)
})

// 递归过滤树节点
const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []

  for (const node of nodes) {
    const matchName = !searchForm.deptName || node.deptName?.includes(searchForm.deptName)
    const matchStatus = searchForm.deptStatus === undefined || node.deptStatus === searchForm.deptStatus

    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = filterTree(node.children)
    }

    // 如果当前节点匹配或有匹配的子节点，则保留
    if ((matchName && matchStatus) || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : undefined
      })
    }
  }

  return result
}

// 父级选项
const parentOptions = ref([])
const parentPath = ref<number[]>([])

const findParentPath = (nodes: any[], targetId: number, path: number[] = []): number[] => {
  for (const node of nodes) {
    const currentPath = [...path, Number(node.id)]
    if (Number(node.id) === targetId) {
      return currentPath
    }
    if (node.children) {
      const childPath = findParentPath(node.children, targetId, currentPath)
      if (childPath.length > 0) {
        return childPath
      }
    }
  }
  return []
}

// 根据部门类型过滤上级选项
const filteredParentOptions = computed(() => {
  const currentType = deptForm.deptType

  // 创建一个ID到部门类型的映射
  const deptTypeMap = new Map<number, number>()
  const buildMap = (nodes: any[]) => {
    for (const node of nodes) {
      deptTypeMap.set(node.id, node.deptType)
      if (node.children) {
        buildMap(node.children)
      }
    }
  }
  buildMap(deptTree.value)

  const allowedTypes = currentType === 2 ? new Set([1]) : new Set([1, 2])

  // 递归过滤并保留合法父级。历史数据中的顶级中心也需要展示。
  const filterNodes = (nodes: any[]): any[] => {
    const result: any[] = []
    for (const node of nodes) {
      if (isEdit.value && Number(node.id) === deptForm.id) {
        continue
      }

      const nodeType = Number(node.deptType ?? deptTypeMap.get(Number(node.id)))
      const filteredChildren = node.children ? filterNodes(node.children) : []

      if (allowedTypes.has(nodeType)) {
        result.push({
          ...node,
          deptType: nodeType,
          children: filteredChildren.length > 0 ? filteredChildren : undefined
        })
      } else if (filteredChildren.length > 0) {
        result.push(...filteredChildren)
      }
    }
    return result
  }

  return currentType === 1 ? [] : filterNodes(parentOptions.value)
})

// 级联选择器配置
const cascaderProps = {
  value: 'id',
  label: 'label',
  children: 'children',
  checkStrictly: true,
  emitPath: true
}

// 部门表单
const deptForm = reactive({
  id: 0,
  parentId: 0,
  deptType: 3,
  deptName: '',
  code: '',
  sort: 0,
  deptStatus: 1
})

// 部门类型是否禁用
const deptTypeDisabled = computed(() => {
  return isEdit.value
})

// 是否显示上级部门选择
const showParentSelect = computed(() => {
  return deptForm.deptType !== 1 // 公司不显示上级部门
})

// 表单验证规则
const rules: FormRules = {
  parentId: [{
    validator: (_rule: any, value: number, callback: (error?: Error) => void) => {
      if (deptForm.deptType !== 1 && !value) {
        callback(new Error(deptForm.deptType === 2 ? '请选择上级公司' : '请选择上级公司或中心'))
        return
      }
      callback()
    },
    trigger: 'change'
  }],
  deptType: [{ required: true, message: '请选择部门类型', trigger: 'change' }],
  deptName: [
    { required: true, message: '请输入部门名称', trigger: 'blur' },
    { min: 2, max: 50, message: '部门名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入部门编码', trigger: 'blur' },
    { min: 2, max: 50, message: '部门编码长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  deptStatus: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

// 搜索后自动展开
const handleSearch = () => {
  if (searchForm.deptName || searchForm.deptStatus !== undefined) {
    isExpandAll.value = true
    nextTick(() => {
      toggleExpandAllRows(true)
    })
  }
}

// 监听搜索条件变化
import { watch } from 'vue'
watch([() => searchForm.deptName, () => searchForm.deptStatus], () => {
  handleSearch()
})

// 重置搜索
const handleReset = () => {
  searchForm.deptName = ''
  searchForm.deptStatus = undefined
  isExpandAll.value = false
  nextTick(() => {
    toggleExpandAllRows(false)
  })
}

// 切换全部展开/折叠
const toggleExpandAll = () => {
  isExpandAll.value = !isExpandAll.value
  toggleExpandAllRows(isExpandAll.value)
}

// 展开/折叠所有行
const toggleExpandAllRows = (expand: boolean) => {
  const table = tableRef.value
  if (!table) return

  const toggleRows = (rows: any[]) => {
    rows.forEach(row => {
      if (row.children && row.children.length > 0) {
        table.toggleRowExpansion(row, expand)
        toggleRows(row.children)
      }
    })
  }

  toggleRows(filteredDeptTree.value)
}

// 加载部门树
const loadDeptTree = async () => {
  loading.value = true
  try {
    const data = await getDepartmentTree()
    // 后端已经返回树形结构，直接使用
    deptTree.value = data || []
  } catch (error) {
    ElMessage.error('获取部门列表失败')
  } finally {
    loading.value = false
  }
}

// 构建树形结构
const buildTree = (data: any[]) => {
  const map = new Map()
  data.forEach(item => {
    map.set(item.id, { ...item, children: [] })
  })

  const tree: any[] = []
  data.forEach(item => {
    const node = map.get(item.id)
    if (item.parentId === 0 || item.parentId === null) {
      tree.push(node)
    } else {
      const parent = map.get(item.parentId)
      if (parent) {
        parent.children.push(node)
      }
    }
  })

  // 清理空children
  const cleanEmptyChildren = (nodes: any[]) => {
    nodes.forEach(node => {
      if (node.children.length === 0) {
        delete node.children
      } else {
        cleanEmptyChildren(node.children)
      }
    })
  }
  cleanEmptyChildren(tree)

  return tree
}

// 加载父级选项
const loadParentOptions = async () => {
  try {
    const data = await getParentOptions()
    // 后端已经返回树形结构，直接使用
    parentOptions.value = data || []
    if (isEdit.value && deptForm.parentId) {
      parentPath.value = findParentPath(parentOptions.value, deptForm.parentId)
    }
  } catch (error) {
  }
}

// 重置表单
const resetForm = () => {
  deptForm.id = 0
  deptForm.parentId = 0
  deptForm.deptType = 3
  deptForm.deptName = ''
  deptForm.code = ''
  deptForm.sort = 0
  deptForm.deptStatus = 1
  parentPath.value = []
  formRef.value?.clearValidate()
}

// 新增顶级部门
const handleAdd = () => {
  resetForm()
  loadParentOptions()
  dialogTitle.value = '新增部门'
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑部门
const handleEdit = (row: any) => {
  Object.assign(deptForm, {
    id: row.id,
    parentId: row.parentId || 0,
    deptType: row.deptType,
    deptName: row.deptName,
    code: row.code || '',
    sort: row.sort || 0,
    deptStatus: row.deptStatus
  })
  dialogTitle.value = '编辑部门'
  isEdit.value = true
  if (row.parentId && row.parentId !== 0) {
    parentPath.value = findParentPath(parentOptions.value, row.parentId)
  } else {
    parentPath.value = []
  }
  dialogVisible.value = true
}

// 删除部门
const handleDelete = (row: any) => {
  const hasChildren = row.children && row.children.length > 0
  const confirmMsg = hasChildren
    ? `该部门下有 ${row.children.length} 个子部门，确定要删除部门"${row.deptName}"及其所有子部门吗？`
    : `确定要删除部门"${row.deptName}"吗？`

  ElMessageBox.confirm(confirmMsg, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteDepartment(row.id)
      ElMessage.success('删除成功')
      loadDeptTree()
      loadParentOptions()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

// 父级改变
const handleDeptTypeChange = () => {
  // 部门类型改变时，重置上级部门选择
  parentPath.value = []
  deptForm.parentId = 0
  formRef.value?.clearValidate('parentId')
}

const handleParentChange = (value: number[]) => {
  if (value && value.length > 0) {
    const parentId = value[value.length - 1]
    deptForm.parentId = parentId
  } else {
    deptForm.parentId = 0
  }
  formRef.value?.validateField('parentId')
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const data = { ...deptForm }

      if (isEdit.value) {
        await updateDepartment(data.id, data)
        ElMessage.success('更新成功')
      } else {
        await createDepartment(data)
        ElMessage.success('创建成功')
      }

      dialogVisible.value = false
      loadDeptTree()
      loadParentOptions()
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

onMounted(() => {
  loadDeptTree()
  loadParentOptions()
})
</script>

<style scoped>
.dept-info-container {
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

/* 树形表格特定样式 */
.dept-tree-table :deep(.el-table__expand-icon) {
  color: #606266 !important;
  font-size: 16px !important;
  padding: 0 !important;
}

.dept-tree-table :deep(.el-table__expand-icon:hover) {
  color: #d4af37 !important;
}

.dept-tree-table :deep(.el-table__expand-icon--expanded) {
  transform: rotate(90deg);
}

/* 缩进元素 */
.dept-tree-table :deep(.el-table__indent) {
  display: inline-block !important;
  width: 30px !important;
}

/* 展开图标容器 */
.dept-tree-table :deep(.el-table__cell .el-table__expand-icon) {
  display: inline-block !important;
  margin-right: 4px !important;
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

/* 部门名称单元格 */
.dept-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dept-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.company-icon {
  color: #409eff;
}

.center-icon {
  color: #67c23a;
}

.folder-icon {
  color: #e6a23c;
}

.dept-name {
  flex: 1;
  font-weight: 500;
}

/* 部门类型标签 */
.dept-type-tag {
  border-radius: 6px;
  font-size: 12px;
  padding: 4px 10px;
  font-weight: 500;
}

.company-tag {
  background-color: #ecf5ff;
  color: #409eff;
  border-color: #b3d8ff;
}

.center-tag {
  background-color: #e8f5e9;
  color: #4caf50;
  border-color: #a5d6a7;
}

.dept-tag {
  background-color: #fdf6ec;
  color: #e6a23c;
  border-color: #f5dab1;
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
  background-color: #e8f4ff;
  color: #409eff;
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

/* 表单提示 */
.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.dept-edit-dialog) {
  border-radius: 12px;
}

:deep(.dept-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.dept-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.dept-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 展开/折叠图标 */
:deep(.el-table__expand-icon) {
  color: #606266;
  font-size: 14px;
}

:deep(.el-table__expand-icon--expanded) {
  transform: rotate(90deg);
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
</style>
