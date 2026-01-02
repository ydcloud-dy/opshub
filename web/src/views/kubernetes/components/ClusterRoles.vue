<template>
  <div class="cluster-roles">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <el-form :inline="true">
        <el-form-item>
          <el-input
            v-model="searchKeyword"
            placeholder="搜索角色名称"
            clearable
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            style="width: 240px"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon style="margin-right: 4px;"><Search /></el-icon>
            搜索
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 角色列表 -->
    <el-table
      :data="filteredRoles"
      border
      stripe
      v-loading="loading"
      style="width: 100%"
      @row-click="handleRowClick"
    >
      <el-table-column prop="name" label="角色名称" min-width="250">
        <template #default="{ row }">
          <span class="role-name-link">
            <el-icon color="#409EFF" :size="18"><Key /></el-icon>
            {{ row.name }}
          </span>
        </template>
      </el-table-column>

      <el-table-column prop="labels" label="标签" min-width="200">
        <template #default="{ row }">
          <el-tag
            v-for="(value, key) in row.labels"
            :key="key"
            size="small"
            style="margin-right: 4px; margin-bottom: 4px;"
          >
            {{ key }}: {{ value }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="age" label="创建时间" width="180">
        <template #default="{ row }">
          {{ row.age }}
        </template>
      </el-table-column>

      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="handleViewDetail(row)">
            <el-icon size="18"><View /></el-icon>
          </el-button>
          <el-button link type="danger" @click.stop="handleDelete(row)" v-if="row.isCustom">
            <el-icon size="18"><Delete /></el-icon>
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Key, View, Delete } from '@element-plus/icons-vue'
import { getClusterRoles, deleteRole } from '@/api/kubernetes'

interface ClusterRole {
  name: string
  labels: Record<string, string>
  age: string
  rules: any[]
  isCustom: boolean
}

const props = defineProps({
  clusterId: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['role-click'])

const loading = ref(false)
const searchKeyword = ref('')
const roleList = ref<ClusterRole[]>([])

// 过滤后的角色列表
const filteredRoles = computed(() => {
  if (!searchKeyword.value) {
    return roleList.value
  }
  const keyword = searchKeyword.value.toLowerCase()
  return roleList.value.filter(role =>
    role.name.toLowerCase().includes(keyword) ||
    Object.entries(role.labels).some(([key, value]) =>
      `${key}:${value}`.toLowerCase().includes(keyword)
    )
  )
})

// 加载集群角色列表
const loadClusterRoles = async () => {
  if (!props.clusterId) return

  try {
    loading.value = true
    const roles = await getClusterRoles(props.clusterId)
    roleList.value = roles || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '加载集群角色失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  // 搜索逻辑通过 computed 自动处理
}

const handleRowClick = (row: ClusterRole) => {
  emit('role-click', row)
}

const handleViewDetail = (row: ClusterRole) => {
  emit('role-click', row)
}

const handleDelete = async (row: ClusterRole) => {
  try {
    await ElMessageBox.confirm(`确定要删除角色 "${row.name}" 吗？`, '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    // 集群角色的 namespace 为空字符串
    await deleteRole(props.clusterId, '', row.name)
    ElMessage.success('删除成功')
    await loadClusterRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

onMounted(() => {
  loadClusterRoles()
})

// 暴露刷新方法供父组件调用
defineExpose({
  refresh: loadClusterRoles
})
</script>

<style scoped lang="scss">
.cluster-roles {
  .search-bar {
    margin-bottom: 20px;
    padding: 16px;
    background: #f5f5f5;
    border-radius: 8px;
  }

  .role-name-link {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: #409EFF;
    font-weight: 500;

    &:hover {
      text-decoration: underline;
    }
  }

  :deep(.el-table) {
    cursor: pointer;

    .el-table__row:hover {
      background-color: #f5f7fa;
    }
  }
}
</style>
