<template>
  <div class="roles-tab">
    <div class="search-bar">
      <el-input
        v-model="searchName"
        placeholder="搜索 Role 名称..."
        clearable
        @input="handleSearch"
        class="search-input"
      >
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <div class="table-wrapper">
      <el-table
        :data="paginatedList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" min-width="200" fixed="left">
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon"><Key /></el-icon>
              </div>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="命名空间" prop="namespace" width="180">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.namespace }}</el-tag>
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
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next, jumper"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Key, Edit, Delete } from '@element-plus/icons-vue'
import { getRoles, type RoleInfo } from '@/api/kubernetes'

interface Props {
  clusterId: number
  namespace: string
}

const props = defineProps<Props>()
const loading = ref(false)
const roles = ref<RoleInfo[]>([])
const searchName = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const filteredData = computed(() => {
  let result = roles.value
  if (searchName.value) {
    result = result.filter(item =>
      item.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }
  return result
})

const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

const loadData = async () => {
  loading.value = true
  try {
    const data = await getRoles(props.clusterId, props.namespace)
    roles.value = data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Role 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
}

const handleEdit = (row: RoleInfo) => {
  ElMessage.info('编辑功能开发中...')
}

const handleDelete = (row: RoleInfo) => {
  ElMessage.info('删除功能开发中...')
}

watch(() => [props.clusterId, props.namespace], () => {
  loadData()
}, { immediate: true })
</script>

<style scoped>
.roles-tab { width: 100%; }
.search-bar { margin-bottom: 16px; }
.search-input { width: 300px; }
.search-icon { color: #d4af37; }
.table-wrapper { background: #fff; border-radius: 8px; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06); overflow: hidden; }
.modern-table { width: 100%; }
.modern-table :deep(.el-table__row:hover) { background-color: #f8fafc !important; }
.name-cell { display: flex; align-items: center; gap: 10px; }
.name-icon-wrapper { width: 32px; height: 32px; border-radius: 6px; background: linear-gradient(135deg, #000 0%, #1a1a1a 100%); display: flex; align-items: center; justify-content: center; border: 1px solid #d4af37; flex-shrink: 0; }
.name-icon { color: #d4af37; font-size: 14px; }
.name-text { font-weight: 600; color: #d4af37; }
.action-btn { color: #d4af37; margin: 0 4px; }
.action-btn.danger { color: #f56c6c; }
.action-btn:hover { transform: scale(1.1); }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid #f0f0f0; }
</style>
