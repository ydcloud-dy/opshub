<template>
  <div class="nodes-container">
    <!-- 页面标题和操作 -->
    <div class="page-header">
      <h2 class="page-title">节点管理</h2>
      <el-select
        v-model="selectedClusterId"
        placeholder="选择集群"
        style="width: 300px"
        @change="handleClusterChange"
      >
        <el-option
          v-for="cluster in clusterList"
          :key="cluster.id"
          :label="cluster.alias || cluster.name"
          :value="cluster.id"
        />
      </el-select>
    </div>

    <!-- 搜索区域 -->
    <div class="search-section">
      <el-input
        v-model="searchName"
        placeholder="根据名称搜索"
        clearable
        style="width: 200px"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-select
        v-model="searchStatus"
        placeholder="节点状态"
        clearable
        style="width: 150px"
        @change="handleSearch"
      >
        <el-option label="Ready" value="Ready" />
        <el-option label="NotReady" value="NotReady" />
      </el-select>
      <el-select
        v-model="searchRole"
        placeholder="节点角色"
        clearable
        style="width: 150px"
        @change="handleSearch"
      >
        <el-option label="master" value="master" />
        <el-option label="control-plane" value="control-plane" />
        <el-option label="worker" value="worker" />
      </el-select>
    </div>

    <!-- 节点列表 -->
    <el-table :data="filteredNodeList" border stripe v-loading="loading" style="width: 100%">
      <el-table-column label="节点名称" min-width="200" fixed="left">
        <template #default="{ row }">
          <div class="node-name-cell">
            <img src="/k8s.png" class="k8s-icon" alt="k8s" />
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'Ready' ? 'success' : 'danger'">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="150">
        <template #default="{ row }">
          <span :class="['role-badge', 'role-' + row.roles]">{{ row.roles }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="version" label="kubelet版本" width="150" />
      <el-table-column label="标签" width="100" align="center">
        <template #default="{ row }">
          <el-badge :value="Object.keys(row.labels || {}).length" :max="99" class="label-badge">
            <el-icon class="label-icon" @click="showLabels(row)" :size="20">
              <PriceTag />
            </el-icon>
          </el-badge>
        </template>
      </el-table-column>
      <el-table-column prop="age" label="运行时间" width="120" />
      <el-table-column label="CPU/内存" width="180">
        <template #default="{ row }">
          <div class="resource-cell">
            <div class="resource-item">
              <span class="resource-label">CPU:</span>
              <span class="resource-value">{{ formatCPU(row.cpuCapacity) }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">内存:</span>
              <span class="resource-value">{{ formatMemory(row.memoryCapacity) }}</span>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="podCount" label="Pod数量" width="100" align="center">
        <template #default="{ row }">
          {{ row.podCount ?? 0 }}
        </template>
      </el-table-column>
      <el-table-column label="调度状态" width="120" align="center">
        <template #default="{ row }">
          <el-tag :type="row.schedulable ? 'success' : 'warning'">
            {{ row.schedulable ? '可调度' : '不可调度' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="taintCount" label="污点数量" width="100" align="center">
        <template #default="{ row }">
          {{ row.taintCount ?? 0 }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewDetails(row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="节点标签"
      width="600px"
    >
      <el-table :data="labelList" border>
        <el-table-column prop="key" label="Key" min-width="200" />
        <el-table-column prop="value" label="Value" min-width="300" />
      </el-table>
      <template #footer>
        <el-button @click="labelDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, PriceTag } from '@element-plus/icons-vue'
import { getClusterList, type Cluster, getNodes, type NodeInfo } from '@/api/kubernetes'

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const nodeList = ref<NodeInfo[]>([])

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')
const searchRole = ref('')

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])

// 过滤后的节点列表
const filteredNodeList = computed(() => {
  let result = nodeList.value

  if (searchName.value) {
    result = result.filter(node =>
      node.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (searchStatus.value) {
    result = result.filter(node => node.status === searchStatus.value)
  }

  if (searchRole.value) {
    result = result.filter(node => node.roles === searchRole.value)
  }

  return result
})

// 格式化 CPU 显示
const formatCPU = (cpu: string) => {
  if (!cpu) return '-'
  // CPU 格式如 "4" 或 "800m" (millicores)
  if (cpu.endsWith('m')) {
    // 毫核，转换为核
    const millicores = parseInt(cpu)
    if (isNaN(millicores)) return cpu
    return (millicores / 1000).toFixed(2) + '核'
  }
  return cpu + '核'
}

// 格式化内存显示
const formatMemory = (memory: string) => {
  if (!memory) return '-'

  // 匹配数字和单位，如: 16082156Ki, 16Gi, 1Ti, 512Mi
  const match = memory.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return memory

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  if (!unit) {
    // 纯数字，假设是字节数
    const bytes = value
    const tb = bytes / (1024 * 1024 * 1024 * 1024)
    if (tb >= 1) return Math.ceil(tb) + 'T'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return Math.ceil(gb) + 'G'
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return Math.ceil(mb) + 'M'
    return memory
  }

  // 转换为字节数后再转换到合适单位
  let bytes = 0
  switch (unit) {
    case 'KI':
      bytes = value * 1024
      break
    case 'MI':
      bytes = value * 1024 * 1024
      break
    case 'GI':
      bytes = value * 1024 * 1024 * 1024
      break
    case 'TI':
      bytes = value * 1024 * 1024 * 1024 * 1024
      break
  }

  // 转换为更合适的单位（向上取整）
  const tb = bytes / (1024 * 1024 * 1024 * 1024)
  if (tb >= 1) return Math.ceil(tb) + 'T'

  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return Math.ceil(gb) + 'G'

  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return Math.ceil(mb) + 'M'

  return memory
}

// 显示标签弹窗
const showLabels = (row: NodeInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      // 从 localStorage 读取上次选择的集群ID
      const savedClusterId = localStorage.getItem('nodes_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        // 检查保存的集群ID是否还存在
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNodes()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  // 保存到 localStorage
  if (selectedClusterId.value) {
    localStorage.setItem('nodes_selected_cluster_id', selectedClusterId.value.toString())
  }
  await loadNodes()
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const data = await getNodes(selectedClusterId.value)
    console.log('节点数据:', data)
    if (data && data.length > 0) {
      console.log('第一个节点的 memoryCapacity:', data[0].memoryCapacity, '类型:', typeof data[0].memoryCapacity)
      console.log('第一个节点的 cpuCapacity:', data[0].cpuCapacity, '类型:', typeof data[0].cpuCapacity)
    }
    nodeList.value = data || []
  } catch (error) {
    console.error(error)
    nodeList.value = []
    ElMessage.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  // 搜索逻辑由 computed 自动处理
}

// 查看详情
const handleViewDetails = (row: NodeInfo) => {
  console.log('查看节点详情:', row)
  ElMessage.info('详情功能开发中...')
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.nodes-container {
  padding: 20px;
  background-color: #fff;
  min-height: 100%;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e6e6e6;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: #303133;
}

.search-section {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

/* K8s 图标 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.k8s-icon {
  width: 18px;
  height: 18px;
  color: #326ce5;
}

/* 角色标签样式 */
.role-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.role-master {
  background-color: #e6f7ff;
  color: #1890ff;
  border: 1px solid #91d5ff;
}

.role-control-plane {
  background-color: #f6ffed;
  color: #52c41a;
  border: 1px solid #b7eb8f;
}

.role-worker {
  background-color: #f5f5f5;
  color: #595959;
  border: 1px solid #d9d9d9;
}

/* 标签图标 */
.label-badge {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.label-badge :deep(.el-badge__content) {
  transform: translateY(-50%) translateX(50%);
  right: 0;
  top: 0;
}

.label-icon {
  color: #409eff;
  cursor: pointer;
  transition: color 0.3s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.label-icon:hover {
  color: #66b1ff;
}

/* 资源显示 */
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-item {
  display: flex;
  align-items: center;
}

.resource-label {
  color: #909399;
  margin-right: 8px;
  font-size: 12px;
}

.resource-value {
  color: #303133;
  font-size: 13px;
}
</style>
