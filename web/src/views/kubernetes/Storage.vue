<template>
  <div class="storage-page">
    <div class="page-header">
      <h2 class="page-title">存储管理</h2>
      <div class="header-controls">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          @change="handleClusterChange"
          class="cluster-selector"
        >
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-button @click="loadCurrentResources" class="black-button">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </div>

    <div class="namespace-selector-wrapper">
      <el-select
        v-model="selectedNamespace"
        placeholder="选择命名空间"
        @change="handleNamespaceChange"
        clearable
        class="namespace-selector"
      >
        <el-option label="全部命名空间" value="" />
        <el-option
          v-for="ns in namespaceList"
          :key="ns.name"
          :label="ns.name"
          :value="ns.name"
        />
      </el-select>
    </div>

    <div class="storage-content">
      <el-tabs v-model="activeTab" class="storage-tabs">
        <el-tab-pane label="PersistentVolumeClaims" name="pvcs">
          <PVCList
            ref="pvcListRef"
            :clusterId="selectedClusterId"
            :namespace="selectedNamespace"
            @refresh="loadCurrentResources"
          />
        </el-tab-pane>
        <el-tab-pane label="PersistentVolumes" name="pvs">
          <PVList
            ref="pvListRef"
            :clusterId="selectedClusterId"
            @refresh="loadCurrentResources"
          />
        </el-tab-pane>
        <el-tab-pane label="StorageClasses" name="storageclasses">
          <StorageClassList
            ref="storageClassListRef"
            :clusterId="selectedClusterId"
            @refresh="loadCurrentResources"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getClusterList, getNamespaces } from '@/api/kubernetes'
import PVCList from './storage-components/PVCList.vue'
import PVList from './storage-components/PVList.vue'
import StorageClassList from './storage-components/StorageClassList.vue'

const clusterList = ref<any[]>([])
const namespaceList = ref<any[]>([])
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>('')
const activeTab = ref('pvcs')

// 子组件引用
const pvcListRef = ref()
const pvListRef = ref()
const storageClassListRef = ref()

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []

    // 从本地存储恢复上次选择的集群
    const savedClusterId = localStorage.getItem('storage_cluster_id')
    if (savedClusterId) {
      const clusterExists = clusterList.value.some((c: any) => c.id === Number(savedClusterId))
      if (clusterExists) {
        selectedClusterId.value = Number(savedClusterId)
        loadNamespaces()
      } else if (clusterList.value.length > 0) {
        selectedClusterId.value = clusterList.value[0].id
        loadNamespaces()
      }
    } else if (clusterList.value.length > 0) {
      selectedClusterId.value = clusterList.value[0].id
      loadNamespaces()
    }
  } catch (error) {
    console.error('获取集群列表失败:', error)
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return
  try {
    const data = await getNamespaces(selectedClusterId.value)
    namespaceList.value = data || []
  } catch (error) {
    console.error('获取命名空间列表失败:', error)
  }
}

// 集群变化
const handleClusterChange = () => {
  localStorage.setItem('storage_cluster_id', String(selectedClusterId.value))
  selectedNamespace.value = ''
  loadNamespaces()
  loadCurrentResources()
}

// 命名空间变化
const handleNamespaceChange = () => {
  localStorage.setItem('storage_namespace', String(selectedNamespace.value))
  loadCurrentResources()
}

// 加载当前标签页的资源
const loadCurrentResources = () => {
  switch (activeTab.value) {
    case 'pvcs':
      pvcListRef.value?.loadData()
      break
    case 'pvs':
      pvListRef.value?.loadData()
      break
    case 'storageclasses':
      storageClassListRef.value?.loadData()
      break
  }
}

onMounted(() => {
  loadClusters()

  // 恢复上次选择的命名空间
  const savedNamespace = localStorage.getItem('storage_namespace')
  if (savedNamespace) {
    selectedNamespace.value = savedNamespace
  }
})
</script>

<style scoped>
.storage-page {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.header-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.cluster-selector {
  width: 200px;
}

.namespace-selector-wrapper {
  margin-bottom: 16px;
}

.namespace-selector {
  width: 200px;
}

.storage-content {
  flex: 1;
  overflow: hidden;
}

.storage-tabs {
  height: 100%;
}

.storage-tabs :deep(.el-tabs__content) {
  height: calc(100% - 55px);
  overflow: auto;
}

.storage-tabs :deep(.el-tab-pane) {
  height: 100%;
}

/* 黑色按钮样式 */
.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}
</style>
