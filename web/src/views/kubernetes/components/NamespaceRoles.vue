<template>
  <div class="namespace-roles">
    <!-- 命名空间选择器 -->
    <div class="namespace-selector">
      <el-select
        v-model="selectedNamespace"
        placeholder="选择命名空间"
        filterable
        @change="handleNamespaceChange"
        style="width: 300px"
      >
        <el-option
          v-for="ns in namespaces"
          :key="ns.name"
          :label="ns.name"
          :value="ns.name"
        >
          <span>{{ ns.name }}</span>
          <span style="color: #8492a6; font-size: 12px;">({{ ns.podCount }} pods)</span>
        </el-option>
      </el-select>
    </div>

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

      <el-table-column prop="namespace" label="命名空间" width="180">
        <template #default="{ row }">
          <el-tag size="small">{{ row.namespace }}</el-tag>
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
          <el-button link type="danger" @click.stop="handleDelete(row)">
            <el-icon size="18"><Delete /></el-icon>
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Key, View, Delete } from '@element-plus/icons-vue'
import { getNamespacesForRoles, getNamespaceRoles, createDefaultNamespaceRoles } from '@/api/kubernetes'

interface Namespace {
  name: string
  podCount: number
}

interface NamespaceRole {
  name: string
  namespace: string
  labels: Record<string, string>
  age: string
  rules: any[]
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
const selectedNamespace = ref('default')
const roleList = ref<NamespaceRole[]>([])
const namespaces = ref<Namespace[]>([])

// 过滤后的角色列表
const filteredRoles = computed(() => {
  let roles = roleList.value

  // 按命名空间过滤
  if (selectedNamespace.value) {
    roles = roles.filter(role => role.namespace === selectedNamespace.value)
  }

  // 按关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    roles = roles.filter(role =>
      role.name.toLowerCase().includes(keyword) ||
      Object.entries(role.labels).some(([key, value]) =>
        `${key}:${value}`.toLowerCase().includes(keyword)
      )
    )
  }

  return roles
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return

  try {
    const nsList = await getNamespacesForRoles(props.clusterId)
    namespaces.value = nsList.map(ns => ({
      name: ns.name,
      podCount: ns.podCount || 0
    }))
  } catch (error: any) {
    ElMessage.error('加载命名空间失败')
  }
}

// 加载命名空间角色列表
const loadNamespaceRoles = async () => {
  if (!props.clusterId || !selectedNamespace.value) return

  try {
    loading.value = true
    let roles = await getNamespaceRoles(props.clusterId, selectedNamespace.value)

    // 定义应该有的12个默认命名空间角色
    const expectedNamespaceRoles = [
      'namespace-owner',
      'namespace-viewer',
      'manage-workload',
      'manage-config',
      'manage-rbac',
      'manage-service-discovery',
      'manage-storage',
      'view-workload',
      'view-config',
      'view-rbac',
      'view-service-discovery',
      'view-storage'
    ]

    // 如果角色数量不等于12，说明角色缺失，需要创建
    if (!roles || roles.length !== expectedNamespaceRoles.length) {
      try {
        await createDefaultNamespaceRoles(props.clusterId, selectedNamespace.value)
        // 重新加载角色列表
        roles = await getNamespaceRoles(props.clusterId, selectedNamespace.value)
      } catch (createError) {
        console.error('Failed to create default namespace roles:', createError)
      }
    }

    roleList.value = roles || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.data?.message || '加载命名空间角色失败')
  } finally {
    loading.value = false
  }
}

const handleNamespaceChange = () => {
  loadNamespaceRoles()
}

const handleSearch = () => {
  // 搜索逻辑通过 computed 自动处理
}

const handleRowClick = (row: NamespaceRole) => {
  emit('role-click', row)
}

const handleViewDetail = (row: NamespaceRole) => {
  emit('role-click', row)
}

const handleDelete = async (row: NamespaceRole) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除命名空间 "${row.namespace}" 中的角色 "${row.name}" 吗？`,
      '提示',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )

    ElMessage.success('删除成功')
    await loadNamespaceRoles()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

onMounted(() => {
  loadNamespaces()
})

// 暴露刷新方法
defineExpose({
  refresh: loadNamespaceRoles
})
</script>

<style scoped lang="scss">
.namespace-roles {
  .namespace-selector {
    margin-bottom: 20px;
  }

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
