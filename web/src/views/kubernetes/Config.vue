<template>
  <div class="config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Key /></el-icon>
        </div>
        <div>
          <h2 class="page-title">配置管理</h2>
          <p class="page-subtitle">管理 Kubernetes ConfigMaps 和 Secrets</p>
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
        <el-button class="black-button" @click="loadCurrentResources">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="config-tabs">
      <el-tab-pane label="ConfigMaps" name="configmaps">
        <ConfigMapList
          v-if="activeTab === 'configmaps'"
          :clusterId="selectedClusterId"
          @edit="handleEditConfigMap"
          @yaml="handleEditConfigMapYAML"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>

      <el-tab-pane label="Secrets" name="secrets">
        <SecretList
          v-if="activeTab === 'secrets'"
          :clusterId="selectedClusterId"
          @edit="handleEditSecret"
          @yaml="handleEditSecretYAML"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>

      <el-tab-pane label="ResourceQuotas" name="resourcequotas">
        <ResourceQuotaList
          v-if="activeTab === 'resourcequotas'"
          :clusterId="selectedClusterId"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>

      <el-tab-pane label="LimitRanges" name="limitranges">
        <LimitRangeList
          v-if="activeTab === 'limitranges'"
          :clusterId="selectedClusterId"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>

      <el-tab-pane label="HPA" name="hpa">
        <HPAList
          v-if="activeTab === 'hpa'"
          :clusterId="selectedClusterId"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>

      <el-tab-pane label="PodDisruptionBudgets" name="pdb">
        <PodDisruptionBudgetList
          v-if="activeTab === 'pdb'"
          :clusterId="selectedClusterId"
          @refresh="loadCurrentResources"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Platform,
  Refresh,
  Key
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import ConfigMapList from './config-components/ConfigMapList.vue'
import SecretList from './config-components/SecretList.vue'
import ResourceQuotaList from './config-components/ResourceQuotaList.vue'
import LimitRangeList from './config-components/LimitRangeList.vue'
import HPAList from './config-components/HPAList.vue'
import PodDisruptionBudgetList from './config-components/PodDisruptionBudgetList.vue'

const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('configmaps')

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('config_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('config_selected_cluster_id', selectedClusterId.value.toString())
  }
}

// Tab 切换
const handleTabChange = () => {
  localStorage.setItem('config_active_tab', activeTab.value)
}

// 加载当前资源
const loadCurrentResources = () => {
  // 由子组件处理
}

// ConfigMap 操作
const handleEditConfigMap = (configMap: any) => {
  ElMessage.info('编辑 ConfigMap 功能开发中...')
}

const handleEditConfigMapYAML = (configMap: any) => {
  ElMessage.info('编辑 ConfigMap YAML 功能开发中...')
}

// Secret 操作
const handleEditSecret = (secret: any) => {
  ElMessage.info('编辑 Secret 功能开发中...')
}

const handleEditSecretYAML = (secret: any) => {
  ElMessage.info('编辑 Secret YAML 功能开发中...')
}

onMounted(() => {
  loadClusters()
  const savedTab = localStorage.getItem('config_active_tab')
  if (savedTab) {
    activeTab.value = savedTab
  }
})
</script>

<style scoped>
.config-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
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

.config-tabs {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

.config-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

.config-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: #d4af37;
}

.config-tabs :deep(.el-tabs__item) {
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}

.config-tabs :deep(.el-tabs__item.is-active) {
  color: #d4af37;
}

.config-tabs :deep(.el-tabs__active-bar) {
  background-color: #d4af37;
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 8px;
}
</style>
