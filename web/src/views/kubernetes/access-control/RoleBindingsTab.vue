<template>
  <div class="role-bindings-tab">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 RoleBinding 名称..." clearable @input="handleSearch" class="search-input">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <div class="table-wrapper">
      <el-table :data="paginatedList" v-loading="loading" class="modern-table" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }">
        <el-table-column label="名称" min-width="200" fixed="left">
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper"><el-icon class="name-icon"><Link /></el-icon></div>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="命名空间" prop="namespace" width="180">
          <template #default="{ row }"><el-tag size="small" type="info">{{ row.namespace }}</el-tag></template>
        </el-table-column>
        <el-table-column label="Role" width="180">
          <template #default="{ row }">
            <span class="role-name">{{ row.roleName }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Users" width="150">
          <template #default="{ row }">
            <span class="subject-text">{{ getSubjectsByKind(row.subjects, 'User') }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Groups" width="150">
          <template #default="{ row }">
            <span class="subject-text">{{ getSubjectsByKind(row.subjects, 'Group') }}</span>
          </template>
        </el-table-column>
        <el-table-column label="ServiceAccounts" width="200">
          <template #default="{ row }">
            <span class="subject-text">{{ getSubjectsByKind(row.subjects, 'ServiceAccount') }}</span>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="140" />
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link @click="handleEdit(row)" class="action-btn">
              <el-icon :size="18"><Edit /></el-icon>
            </el-button>
            <el-button link @click="handleDelete(row)" class="action-btn danger">
              <el-icon :size="18"><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrapper">
        <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[10, 20, 50, 100]" :total="filteredData.length" layout="total, sizes, prev, pager, next, jumper" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Link, Edit, Delete } from '@element-plus/icons-vue'
import { getRoleBindings, type RoleBindingInfo } from '@/api/kubernetes'

interface Props {
  clusterId: number
  namespace: string
}

const props = defineProps<Props>()
const loading = ref(false)
const roleBindings = ref<RoleBindingInfo[]>([])
const searchName = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const filteredData = computed(() => {
  let result = roleBindings.value
  if (searchName.value) {
    result = result.filter(item => item.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredData.value.slice(start, start + pageSize.value)
})

const loadData = async () => {
  loading.value = true
  try {
    const data = await getRoleBindings(props.clusterId, props.namespace)
    roleBindings.value = data || []
  } catch (error) {
    ElMessage.error('获取 RoleBinding 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { currentPage.value = 1 }

// 根据 kind 获取 subjects 名称列表
const getSubjectsByKind = (subjects: any[] | undefined, kind: string) => {
  if (!subjects) return '-'
  const filtered = subjects.filter(s => s.kind === kind)
  if (filtered.length === 0) return '-'
  return filtered.map(s => s.name).join(', ')
}

const handleEdit = (row: RoleBindingInfo) => {
  ElMessage.info('编辑功能开发中...')
}

const handleDelete = (row: RoleBindingInfo) => {
  ElMessage.info('删除功能开发中...')
}

watch(() => [props.clusterId, props.namespace], loadData, { immediate: true })
</script>

<style scoped>
.role-bindings-tab { width: 100%; }
.search-bar { margin-bottom: 16px; }
.search-input { width: 300px; }
.search-icon { color: #d4af37; }
.table-wrapper { background: #fff; border-radius: 8px; overflow: hidden; }
.name-cell { display: flex; align-items: center; gap: 10px; }
.name-icon-wrapper { width: 32px; height: 32px; border-radius: 6px; background: linear-gradient(135deg, #000 0%, #1a1a1a 100%); display: flex; align-items: center; justify-content: center; border: 1px solid #d4af37; }
.name-icon { color: #d4af37; }
.name-text { font-weight: 600; color: #d4af37; }
.role-name { font-family: monospace; color: #606266; }
.subject-text { font-size: 13px; color: #606266; }
.action-btn { color: #d4af37; margin: 0 4px; }
.action-btn.danger { color: #f56c6c; }
.action-btn:hover { transform: scale(1.1); }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid #f0f0f0; }
</style>
