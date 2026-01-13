<template>
  <div class="access-control-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">访问控制</h2>
          <p class="page-subtitle">管理 Kubernetes 集群的访问控制和权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-select
          v-model="selectedNamespace"
          placeholder="选择命名空间"
          class="namespace-select"
          @change="handleNamespaceChange"
          :disabled="!selectedClusterId"
        >
          <template #prefix>
            <el-icon class="search-icon"><FolderOpened /></el-icon>
          </template>
          <el-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </el-select>
        <el-button class="black-button" @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 访问控制类型标签 -->
    <div class="access-types-bar">
      <div
        v-for="type in accessTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
        <el-icon class="type-icon">
          <component :is="type.icon" />
        </el-icon>
        <span class="type-label">{{ type.label }}</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- ServiceAccounts -->
      <ServiceAccountsTab
        v-if="activeTab === 'serviceaccounts' && selectedClusterId"
        :cluster-id="selectedClusterId"
        :namespace="selectedNamespace"
      />

      <!-- Roles -->
      <template v-if="activeTab === 'roles'">
        <RolesTab
          v-if="selectedClusterId && selectedNamespace"
          :cluster-id="selectedClusterId"
          :namespace="selectedNamespace"
        />
        <el-empty v-else description="请选择命名空间" />
      </template>

      <!-- RoleBindings -->
      <template v-if="activeTab === 'rolebindings'">
        <RoleBindingsTab
          v-if="selectedClusterId && selectedNamespace"
          :cluster-id="selectedClusterId"
          :namespace="selectedNamespace"
        />
        <el-empty v-else description="请选择命名空间" />
      </template>

      <!-- ClusterRoles -->
      <ClusterRolesTab
        v-if="activeTab === 'clusterroles' && selectedClusterId"
        :cluster-id="selectedClusterId"
      />

      <!-- ClusterRoleBindings -->
      <ClusterRoleBindingsTab
        v-if="activeTab === 'clusterrolebindings' && selectedClusterId"
        :cluster-id="selectedClusterId"
      />

      <!-- PodSecurityPolicies -->
      <PodSecurityPoliciesTab
        v-if="activeTab === 'podsecuritypolicies' && selectedClusterId"
        :cluster-id="selectedClusterId"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Lock,
  Platform,
  FolderOpened,
  Refresh,
  User,
  Key,
  Link,
  Connection
} from '@element-plus/icons-vue'
import { getClusterList, getNamespaces, type Cluster, type NamespaceInfo } from '@/api/kubernetes'
import ServiceAccountsTab from './access-control/ServiceAccountsTab.vue'
import RolesTab from './access-control/RolesTab.vue'
import RoleBindingsTab from './access-control/RoleBindingsTab.vue'
import ClusterRolesTab from './access-control/ClusterRolesTab.vue'
import ClusterRoleBindingsTab from './access-control/ClusterRoleBindingsTab.vue'
import PodSecurityPoliciesTab from './access-control/PodSecurityPoliciesTab.vue'

// 访问控制类型定义
interface AccessType {
  label: string
  value: string
  icon: any
}

const accessTypes: AccessType[] = [
  { label: 'ServiceAccounts', value: 'serviceaccounts', icon: User },
  { label: 'Roles', value: 'roles', icon: Key },
  { label: 'RoleBindings', value: 'rolebindings', icon: Link },
  { label: 'ClusterRoles', value: 'clusterroles', icon: Key },
  { label: 'ClusterRoleBindings', value: 'clusterrolebindings', icon: Connection },
  { label: 'PodSecurityPolicies', value: 'podsecuritypolicies', icon: Lock },
]

const activeTab = ref('serviceaccounts')
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>()
const clusterList = ref<Cluster[]>([])
const namespaceList = ref<NamespaceInfo[]>([])

// 加载集群列表
const loadClusters = async () => {
  try {
    const list = await getClusterList()
    clusterList.value = list || []

    // 恢复上次选择的集群
    const savedClusterId = localStorage.getItem('access_control_cluster_id')
    if (savedClusterId) {
      const id = parseInt(savedClusterId)
      if (clusterList.value.some(c => c.id === id)) {
        selectedClusterId.value = id
      }
    }

    // 如果有集群但没保存的选择，默认选第一个
    if (!selectedClusterId.value && clusterList.value.length > 0) {
      selectedClusterId.value = clusterList.value[0].id
    }

    if (selectedClusterId.value) {
      await loadNamespaces()
    }
  } catch (error) {
    console.error('加载集群列表失败:', error)
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  try {
    const list = await getNamespaces(selectedClusterId.value)
    namespaceList.value = list || []

    // 恢复上次选择的命名空间
    const savedNamespace = localStorage.getItem('access_control_namespace')
    if (savedNamespace && namespaceList.value.some(ns => ns.name === savedNamespace)) {
      selectedNamespace.value = savedNamespace
    } else if (namespaceList.value.length > 0) {
      selectedNamespace.value = namespaceList.value[0].name
    }
  } catch (error) {
    console.error('加载命名空间列表失败:', error)
  }
}

// 处理集群切换
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('access_control_cluster_id', selectedClusterId.value.toString())
    await loadNamespaces()
  }
}

// 处理命名空间切换
const handleNamespaceChange = () => {
  if (selectedNamespace.value) {
    localStorage.setItem('access_control_namespace', selectedNamespace.value)
  }
}

// 处理标签切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
}

// 加载数据
const loadData = () => {
  loadClusters()
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.access-control-container {
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

.cluster-select {
  width: 280px;
}

.namespace-select {
  width: 240px;
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

.search-icon {
  color: #d4af37;
}

/* 访问控制类型标签栏 */
.access-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow-x: auto;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #1a1a1a;
  border: 1px solid #1a1a1a;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  white-space: nowrap;
  color: #e0e0e0;
  font-size: 14px;
  user-select: none;
}

.type-tab:hover {
  background: #333;
  border-color: #333;
  transform: translateY(-1px);
}

.type-tab.active {
  background: #d4af37;
  color: #1a1a1a;
  border-color: #d4af37;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
  font-weight: 600;
}

.type-tab.active .type-icon {
  color: #1a1a1a;
}

.type-icon {
  font-size: 18px;
  color: #d4af37;
  transition: color 0.3s ease;
}

.type-tab:not(.active) .type-icon {
  color: #d4af37;
}

.type-label {
  font-size: 14px;
}

/* 内容区域 */
.content-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 16px;
  min-height: 400px;
}
</style>
