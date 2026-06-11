<template>
  <div class="groups-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Collection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">业务分组</h2>
          <p class="page-subtitle">管理主机业务分组，支持多级层级结构</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增分组
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
          v-model="searchForm.name"
          placeholder="搜索分组名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="分组状态"
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
        :data="filteredGroupTree"
        v-loading="loading"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        :default-expand-all="isExpandAll"
        :indent="30"
        class="modern-table group-tree-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="分组名称" prop="name" min-width="300">
          <template #default="{ row }">
            <span style="display: inline-flex; align-items: center;">
              <el-icon v-if="!row.parentId || row.parentId === 0" style="color: #67c23a; margin-right: 8px;"><Collection /></el-icon>
              <el-icon v-else style="color: #409eff; margin-right: 8px;"><Folder /></el-icon>
              {{ row.name }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="code" min-width="150">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-gold"><Key /></el-icon>
              分组编码
            </span>
          </template>
        </el-table-column>

        <el-table-column label="描述" prop="description" min-width="300" show-overflow-tooltip />

        <el-table-column label="主机数量" width="120" align="center">
          <template #default="{ row }">
            <el-tag type="primary">{{ row.hostCount || 0 }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="dark">
              {{ row.status === 1 ? '正常' : '停用' }}
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
      class="group-edit-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form :model="groupForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="上级分组" prop="parentId" v-if="showParentSelect && !isRootGroup">
          <el-cascader
            v-model="parentPath"
            :options="parentOptions"
            :props="cascaderProps"
            clearable
            placeholder="请选择上级分组"
            style="width: 100%"
            @change="handleParentChange"
          />
          <div class="form-tip">不选择则为顶级分组</div>
        </el-form-item>

        <el-form-item label="分组名称" prop="name">
          <el-input v-model="groupForm.name" placeholder="请输入分组名称" />
        </el-form-item>

        <el-form-item label="分组编码" prop="code">
          <el-input v-model="groupForm.code" placeholder="请输入分组编码" />
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input v-model="groupForm.description" type="textarea" :rows="3" placeholder="请输入分组描述" />
        </el-form-item>

        <el-form-item label="显示顺序" prop="sort">
          <el-input-number v-model="groupForm.sort" :min="0" />
        </el-form-item>

        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="groupForm.status">
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
  Collection,
  Folder,
  Search,
  RefreshLeft,
  Sort,
  Key
} from '@element-plus/icons-vue'
import {
  getGroupTree,
  getParentOptions,
  createGroup,
  updateGroup,
  deleteGroup
} from '@/api/assetGroup'

// 加载状态
const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const isRootGroup = ref(false)

// 表单引用
const formRef = ref<FormInstance>()
const tableRef = ref<InstanceType<typeof ElTable>>()

// 分组树数据
const groupTree = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  name: '',
  status: undefined as number | undefined
})

// 展开/折叠状态
const isExpandAll = ref(true)

// 过滤后的分组树
const filteredGroupTree = computed(() => {
  if (!searchForm.name && searchForm.status === undefined) {
    return groupTree.value
  }
  return filterTree(groupTree.value)
})

// 递归过滤树节点
const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []

  for (const node of nodes) {
    const matchName = !searchForm.name || node.name?.includes(searchForm.name)
    const matchStatus = searchForm.status === undefined || node.status === searchForm.status

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

// 级联选择器配置
const cascaderProps = {
  value: 'id',
  label: 'label',
  children: 'children',
  checkStrictly: true,
  emitPath: true
}

// 分组表单
const groupForm = reactive({
  id: 0,
  parentId: 0,
  name: '',
  code: '',
  description: '',
  sort: 0,
  status: 1
})

// 分组类型是否禁用
const showParentSelect = computed(() => {
  // 编辑模式且不是顶级分组时显示上级选择
  return !isEdit.value || (isEdit.value && !isRootGroup.value)
})

// 表单验证规则
const rules: FormRules = {
  name: [
    { required: true, message: '请输入分组名称', trigger: 'blur' },
    { min: 2, max: 100, message: '分组名称长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入分组编码', trigger: 'blur' },
    { min: 2, max: 50, message: '分组编码长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

// 监听搜索条件变化
import { watch } from 'vue'
watch([() => searchForm.name, () => searchForm.status], () => {
  if (searchForm.name || searchForm.status !== undefined) {
    isExpandAll.value = true
    nextTick(() => {
      toggleExpandAllRows(true)
    })
  }
})

// 重置搜索
const handleReset = () => {
  searchForm.name = ''
  searchForm.status = undefined
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

  toggleRows(filteredGroupTree.value)
}

// 加载分组树
const loadGroupTree = async () => {
  loading.value = true
  try {
    const data = await getGroupTree()
    // 后端已经返回树形结构，直接使用
    groupTree.value = data || []
  } catch (error) {
    ElMessage.error('获取分组列表失败')
  } finally {
    loading.value = false
  }
}

// 加载父级选项
const loadParentOptions = async () => {
  try {
    const data = await getParentOptions()
    // 后端已经返回树形结构，直接使用
    parentOptions.value = data || []
  } catch (error) {
  }
}

// 重置表单
const resetForm = () => {
  groupForm.id = 0
  groupForm.parentId = 0
  groupForm.name = ''
  groupForm.code = ''
  groupForm.description = ''
  groupForm.sort = 0
  groupForm.status = 1
  parentPath.value = []
  isRootGroup.value = false
  formRef.value?.clearValidate()
}

// 新增顶级分组
const handleAdd = () => {
  resetForm()
  loadParentOptions()
  dialogTitle.value = '新增分组'
  isEdit.value = false
  isRootGroup.value = false
  dialogVisible.value = true
}

// 编辑分组
const handleEdit = (row: any) => {
  Object.assign(groupForm, {
    id: row.id,
    parentId: row.parentId || 0,
    name: row.name,
    code: row.code || '',
    description: row.description || '',
    sort: row.sort || 0,
    status: row.status
  })
  dialogTitle.value = '编辑分组'
  isEdit.value = true
  isRootGroup.value = !row.parentId || row.parentId === 0
  if (row.parentId && row.parentId !== 0) {
    parentPath.value = [row.parentId]
  } else {
    parentPath.value = []
  }
  loadParentOptions()
  dialogVisible.value = true
}

// 删除分组
const handleDelete = (row: any) => {
  const hasChildren = row.children && row.children.length > 0
  const confirmMsg = hasChildren
    ? `该分组下有 ${row.children.length} 个子分组，确定要删除分组"${row.name}"及其所有子分组吗？`
    : `确定要删除分组"${row.name}"吗？`

  ElMessageBox.confirm(confirmMsg, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteGroup(row.id)
      ElMessage.success('删除成功')
      loadGroupTree()
      loadParentOptions()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

const handleParentChange = (value: number[]) => {
  if (value && value.length > 0) {
    const parentId = value[value.length - 1]
    groupForm.parentId = parentId
  } else {
    groupForm.parentId = 0
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const data = { ...groupForm }

      if (isEdit.value) {
        await updateGroup(data.id, data)
        ElMessage.success('更新成功')
      } else {
        await createGroup(data)
        ElMessage.success('创建成功')
      }

      dialogVisible.value = false
      loadGroupTree()
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
  loadGroupTree()
  loadParentOptions()
})
</script>

<style scoped>
.groups-container {
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

/* 树形表格特定样式 */
.group-tree-table :deep(.el-table__expand-icon) {
  color: #606266 !important;
  font-size: 16px !important;
  padding: 0 !important;
}

.group-tree-table :deep(.el-table__expand-icon:hover) {
  color: #d4af37 !important;
}

.group-tree-table :deep(.el-table__expand-icon--expanded) {
  transform: rotate(90deg);
}

/* 缩进元素 */
.group-tree-table :deep(.el-table__indent) {
  display: inline-block !important;
  width: 30px !important;
}

/* 展开图标容器 */
.group-tree-table :deep(.el-table__cell .el-table__expand-icon) {
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

:deep(.group-edit-dialog) {
  border-radius: 12px;
}

:deep(.group-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.group-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.group-edit-dialog .el-dialog__footer) {
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

/* 资产页统一视觉优化 */
.groups-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header,
.search-bar,
.table-wrapper {
  border: 1px solid #e5e9f2;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.page-header {
  align-items: center;
  gap: 16px;
  margin-bottom: 0;
  padding: 18px 20px;
  border-radius: 10px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 9px;
  color: #111827;
  background: rgba(255, 175, 53, 0.18);
}

.page-title {
  color: #111827;
  font-size: 22px;
  font-weight: 750;
}

.page-subtitle {
  color: #667085;
}

.header-actions,
.search-actions,
.search-inputs {
  flex-wrap: wrap;
}

.search-bar {
  margin-bottom: 0;
  padding: 14px 16px;
  border-radius: 10px;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.04);
}

.search-input {
  width: 240px;
}

.search-icon,
.header-icon-gold {
  color: #d97706;
}

.search-bar :deep(.el-input__wrapper),
.search-bar :deep(.el-select__wrapper) {
  border: 0;
  border-radius: 7px;
  box-shadow: 0 0 0 1px #e5e9f2 inset;
}

.search-bar :deep(.el-input__wrapper:hover),
.search-bar :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px #d8dee9 inset;
}

.search-bar :deep(.el-input__wrapper.is-focus),
.search-bar :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px #111827 inset, 0 0 0 3px rgba(255, 175, 53, 0.16);
}

.table-wrapper {
  border-radius: 10px;
}

.modern-table :deep(.el-table__header th) {
  background: #f8fafc !important;
  color: #475467 !important;
}

.modern-table :deep(.el-table__body td) {
  border-bottom-color: #edf1f7;
}

.group-tree-table :deep(.el-table__row:hover > td) {
  background: #f8fafc !important;
}

.black-button {
  height: 36px;
  padding: 0 14px;
  border-radius: 7px;
  box-shadow: 0 8px 18px rgba(17, 24, 39, 0.12);
}

.reset-btn {
  background: #fff;
}

.action-btn {
  width: 30px;
  height: 30px;
  border-radius: 7px;
}

:deep(.group-edit-dialog) {
  border-radius: 10px;
}

:deep(.group-edit-dialog .el-dialog__header) {
  background: #fbfcfe;
  border-bottom-color: #edf1f7;
}

@media (max-width: 900px) {
  .page-header,
  .search-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .search-input,
  .search-inputs,
  .search-actions {
    width: 100%;
  }
}
</style>
