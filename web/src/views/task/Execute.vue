<template>
  <div class="execute-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><VideoPlay /></el-icon>
        </div>
        <div>
          <h2 class="page-title">执行任务</h2>
          <p class="page-subtitle">执行脚本任务，实时查看执行结果和日志</p>
        </div>
      </div>
    </div>

    <!-- 主要内容区 -->
    <div class="main-card">
      <!-- 目标主机 -->
      <div class="form-section">
        <div class="section-label">
          <span class="required">*</span>
          <span>目标主机</span>
        </div>
        <div class="section-content">
          <el-button @click="showHostDialog = true" class="add-btn">
            <el-icon style="margin-right: 6px;"><Plus /></el-icon>
            添加目标主机
          </el-button>
          <div v-if="selectedHosts.length > 0" class="selected-items">
            <el-tag
              v-for="host in selectedHosts"
              :key="host.id"
              closable
              @close="removeHost(host.id)"
              style="margin: 8px 8px 0 0;"
            >
              {{ host.name }} ({{ host.ip }})
            </el-tag>
          </div>
        </div>
      </div>

      <!-- 执行命令 -->
      <div class="form-section">
        <div class="section-label">
          <span class="required">*</span>
          <span>执行命令</span>
        </div>
        <div class="section-content">
          <div class="command-toolbar">
            <el-radio-group v-model="scriptType" class="script-type-group">
              <el-radio-button label="Shell">Shell</el-radio-button>
              <el-radio-button label="Python">Python</el-radio-button>
            </el-radio-group>
            <div class="toolbar-right">
              <el-link type="primary" :icon="QuestionFilled" underline="hover">使用全局变量?</el-link>
              <el-button size="small" @click="showTemplateDialog = true">
                <el-icon style="margin-right: 4px;"><Plus /></el-icon>
                从执行模板中选择
              </el-button>
            </div>
          </div>
          <div class="code-editor-wrapper">
            <el-input
              v-model="scriptContent"
              type="textarea"
              :rows="15"
              placeholder="请输入脚本内容..."
              class="code-editor"
            />
          </div>
        </div>
      </div>

      <!-- 开始执行按钮 -->
      <div class="execute-button-section">
        <el-button
          type="primary"
          size="large"
          :loading="executing"
          @click="handleExecute"
          class="execute-button"
        >
          <el-icon style="margin-right: 6px;"><VideoPlay /></el-icon>
          {{ executing ? '执行中...' : '开始执行' }}
        </el-button>
      </div>
    </div>

    <!-- 执行记录 -->
    <div class="log-card" v-if="true">
      <div class="log-header">
        <span class="log-title">执行记录</span>
        <el-icon><InfoFilled /></el-icon>
      </div>
      <div class="log-content">
        <div v-if="executionLogs.length === 0" class="empty-log">
          暂无执行记录
        </div>
        <div v-else class="log-list">
          <div
            v-for="log in executionLogs"
            :key="log.id"
            class="log-item"
            :class="log.status"
          >
            <span class="log-time">{{ log.time }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 选择主机对话框 -->
    <el-dialog
      v-model="showHostDialog"
      title="主机列表"
      width="1000px"
      destroy-on-close
    >
      <div class="host-dialog-content">
        <div class="host-groups">
          <div class="group-title">分组列表</div>
          <el-tree
            :data="hostGroups"
            :props="{ label: 'name', children: 'children' }"
            node-key="id"
            default-expand-all
            @node-click="handleGroupClick"
          >
            <template #default="{ node, data }">
              <span class="tree-node">
                <el-icon><Folder /></el-icon>
                <span>{{ node.label }}</span>
                <span class="group-count">{{ data.hostCount || 0 }}</span>
              </span>
            </template>
          </el-tree>
        </div>
        <div class="host-list">
          <el-input
            v-model="hostSearchKeyword"
            placeholder="输入名称/IP搜索"
            clearable
            style="margin-bottom: 16px;"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-table
            :data="filteredHosts"
            @selection-change="handleHostSelectionChange"
            height="400px"
            v-loading="hostsLoading"
          >
            <el-table-column type="selection" width="55" />
            <el-table-column label="主机名称" prop="name" />
            <el-table-column label="IP地址" prop="ip">
              <template #default="{ row }">
                <el-tag size="small">{{ row.ip }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="备注信息" prop="description" />
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="showHostDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmHostSelection">确定</el-button>
      </template>
    </el-dialog>

    <!-- 选择执行模板对话框 -->
    <el-dialog
      v-model="showTemplateDialog"
      title="选择执行模板"
      width="1200px"
      destroy-on-close
    >
      <div class="template-filter">
        <el-select
          v-model="templateFilter.type"
          placeholder="请选择"
          clearable
          style="width: 200px; margin-right: 12px;"
        >
          <el-option label="系统信息" value="system" />
          <el-option label="部署" value="deploy" />
          <el-option label="监控" value="monitor" />
          <el-option label="备份" value="backup" />
        </el-select>
        <el-input
          v-model="templateFilter.name"
          placeholder="请输入"
          clearable
          style="width: 300px; margin-right: 12px;"
        />
        <el-button type="primary" @click="refreshTemplates">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <el-table
        :data="templates"
        @row-click="selectTemplate"
        highlight-current-row
      >
        <el-table-column label="名称" prop="name" width="180" />
        <el-table-column label="类型" prop="type" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容" prop="content" show-overflow-tooltip />
        <el-table-column label="备注" prop="remark" show-overflow-tooltip />
      </el-table>
      <template #footer>
        <el-button @click="showTemplateDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmTemplateSelection">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Plus,
  VideoPlay,
  QuestionFilled,
  InfoFilled,
  Folder,
  Search,
  Refresh
} from '@element-plus/icons-vue'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'

// 脚本类型
const scriptType = ref('Shell')
const scriptContent = ref('')

// 选中的主机
const selectedHosts = ref<any[]>([])

// 执行状态
const executing = ref(false)

// 执行日志
const executionLogs = ref<any[]>([])

// 主机对话框
const showHostDialog = ref(false)
const hostSearchKeyword = ref('')
const tempSelectedHosts = ref<any[]>([])
const hostGroups = ref<any[]>([])
const allHosts = ref<any[]>([])
const hostsLoading = ref(false)

// 过滤后的主机列表
const filteredHosts = computed(() => {
  let hosts = allHosts.value

  if (hostSearchKeyword.value) {
    const keyword = hostSearchKeyword.value.toLowerCase()
    hosts = hosts.filter(
      (host) =>
        host.name.toLowerCase().includes(keyword) ||
        host.ip.includes(keyword)
    )
  }

  return hosts
})

// 模板对话框
const showTemplateDialog = ref(false)
const templateFilter = ref({
  type: '',
  name: '',
})

// 模板列表
const templates = ref([
  {
    id: 1,
    name: '获取内存使用情况',
    type: '系统信息',
    content: 'free -m',
    remark: '',
  },
])

const selectedTemplate = ref<any>(null)

// 加载主机分组
const loadHostGroups = async () => {
  try {
    const data = await getGroupTree()
    hostGroups.value = data || []
  } catch (error) {
    console.error('加载主机分组失败:', error)
  }
}

// 加载主机列表
const loadHostList = async () => {
  hostsLoading.value = true
  try {
    const params = {
      page: 1,
      pageSize: 1000,
    }
    const response = await getHostList(params)
    if (Array.isArray(response)) {
      allHosts.value = response
    } else if (response.data && Array.isArray(response.data)) {
      allHosts.value = response.data
    } else {
      allHosts.value = []
    }
  } catch (error) {
    console.error('加载主机列表失败:', error)
    allHosts.value = []
  } finally {
    hostsLoading.value = false
  }
}

// 分组点击
const handleGroupClick = (data: any) => {
  console.log('选中分组:', data)
}

// 主机选择变化
const handleHostSelectionChange = (selection: any[]) => {
  tempSelectedHosts.value = selection
}

// 确认主机选择
const confirmHostSelection = () => {
  selectedHosts.value = [...tempSelectedHosts.value]
  showHostDialog.value = false
  ElMessage.success(`已选择 ${selectedHosts.value.length} 台主机`)
}

// 移除主机
const removeHost = (id: number) => {
  const index = selectedHosts.value.findIndex((h) => h.id === id)
  if (index !== -1) {
    selectedHosts.value.splice(index, 1)
  }
}

// 选择模板
const selectTemplate = (row: any) => {
  selectedTemplate.value = row
}

// 刷新模板列表
const refreshTemplates = () => {
  ElMessage.success('刷新成功')
}

// 确认模板选择
const confirmTemplateSelection = () => {
  if (selectedTemplate.value) {
    scriptContent.value = selectedTemplate.value.content
    showTemplateDialog.value = false
    ElMessage.success('已应用模板')
  }
}

// 添加日志
const addLog = (message: string, status: string = 'info') => {
  const now = new Date()
  const time = `${now.getHours().toString().padStart(2, '0')}:${now
    .getMinutes()
    .toString()
    .padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
  executionLogs.value.unshift({
    id: Date.now(),
    time,
    message,
    status,
  })
}

// 执行任务
const handleExecute = async () => {
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请先选择目标主机')
    return
  }
  if (!scriptContent.value.trim()) {
    ElMessage.warning('请输入执行命令')
    return
  }

  executing.value = true
  addLog(`开始执行任务，目标主机: ${selectedHosts.value.length} 台`, 'info')

  try {
    await new Promise((resolve) => setTimeout(resolve, 2000))
    addLog('任务执行成功', 'success')
    ElMessage.success('任务执行成功')
  } catch (error: any) {
    addLog('任务执行失败: ' + error.message, 'error')
    ElMessage.error('任务执行失败')
  } finally {
    executing.value = false
  }
}

onMounted(() => {
  loadHostGroups()
  loadHostList()
})
</script>

<style scoped lang="scss">
.execute-container {
  padding: 0;
  background-color: transparent;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header {
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
  border-radius: 10px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 28px;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: #909399;
  line-height: 20px;
}

.main-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 24px;
}

.form-section {
  margin-bottom: 24px;

  &:last-of-type {
    margin-bottom: 0;
  }
}

.section-label {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;

  .required {
    color: #f56c6c;
    margin-right: 4px;
  }
}

.section-content {
  .add-btn {
    margin-bottom: 12px;
  }
}

.selected-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.command-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;

  .script-type-group {
    :deep(.el-radio-button__inner) {
      padding: 8px 20px;
    }
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.code-editor-wrapper {
  :deep(.el-textarea__inner) {
    font-family: 'Courier New', monospace;
    font-size: 14px;
  }
}

.execute-button-section {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}

.execute-button {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  padding: 12px 40px;
  font-size: 16px;

  &:hover {
    background: linear-gradient(135deg, #7e8ef5 0%, #8d5cb8 100%);
  }
}

.log-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.log-header {
  padding: 16px 20px;
  background: #fafafa;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-radius: 8px 8px 0 0;

  .log-title {
    font-size: 16px;
    font-weight: 600;
    color: #303133;
  }
}

.log-content {
  max-height: 400px;
  padding: 16px 20px;
  overflow-y: auto;

  .empty-log {
    text-align: center;
    color: #909399;
    padding: 40px 0;
  }
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-item {
  padding: 10px 12px;
  border-radius: 4px;
  border-left: 3px solid transparent;
  display: flex;
  gap: 12px;
  font-size: 13px;

  .log-time {
    color: #909399;
    flex-shrink: 0;
    font-family: 'Courier New', monospace;
    min-width: 70px;
  }

  .log-message {
    color: #606266;
    flex: 1;
  }

  &.info {
    background: #ecf5ff;
    border-left-color: #409eff;
  }

  &.success {
    background: #f0f9ff;
    border-left-color: #67c23a;

    .log-message {
      color: #67c23a;
      font-weight: 500;
    }
  }

  &.error {
    background: #fef0f0;
    border-left-color: #f56c6c;

    .log-message {
      color: #f56c6c;
      font-weight: 500;
    }
  }
}

.host-dialog-content {
  display: flex;
  gap: 20px;
  height: 500px;
}

.host-groups {
  width: 250px;
  border-right: 1px solid #e4e7ed;
  padding-right: 20px;

  .group-title {
    font-weight: 600;
    margin-bottom: 12px;
    color: #303133;
  }

  .tree-node {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;

    .group-count {
      margin-left: auto;
      color: #909399;
      font-size: 12px;
    }
  }
}

.host-list {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.template-filter {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  gap: 12px;
}
</style>
