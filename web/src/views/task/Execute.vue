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
            <div class="log-header-line">
              <span class="log-time">{{ log.time }}</span>
              <span class="log-status-badge" :class="log.status">
                {{ log.status === 'success' ? '成功' : log.status === 'error' ? '失败' : '信息' }}
              </span>
              <span class="log-host">{{ log.host || '' }}</span>
            </div>
            <div class="log-output">
              <pre>{{ log.message }}</pre>
            </div>
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
        :data="filteredTemplates"
        @row-click="selectTemplate"
        highlight-current-row
        v-loading="templatesLoading"
        style="cursor: pointer;"
      >
        <el-table-column label="名称" prop="name" width="180" />
        <el-table-column label="类型" prop="category" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.category }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容" prop="content" show-overflow-tooltip />
        <el-table-column label="备注" prop="description" show-overflow-tooltip />
      </el-table>
      <template #footer>
        <el-button @click="showTemplateDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmTemplateSelection">确定</el-button>
      </template>
    </el-dialog>

    <!-- 参数填写对话框 -->
    <el-dialog
      v-model="showParamDialog"
      title="填写模板参数"
      width="600px"
      destroy-on-close
    >
      <div class="param-dialog-content">
        <el-alert
          type="info"
          :closable="false"
          style="margin-bottom: 20px;"
        >
          <template #title>
            模板: <strong>{{ currentTemplate?.name }}</strong>
          </template>
        </el-alert>
        <el-form label-width="120px">
          <el-form-item
            v-for="(param, index) in templateParams"
            :key="index"
            :label="param.name"
            :required="param.required"
          >
            <el-input
              v-if="param.type === 'text'"
              v-model="paramValues[param.varName]"
              :placeholder="param.helpText || `请输入${param.name}`"
            />
            <el-input
              v-else-if="param.type === 'password'"
              v-model="paramValues[param.varName]"
              type="password"
              show-password
              :placeholder="param.helpText || `请输入${param.name}`"
            />
            <el-select
              v-else-if="param.type === 'select'"
              v-model="paramValues[param.varName]"
              :placeholder="param.helpText || `请选择${param.name}`"
              style="width: 100%;"
            >
              <el-option
                v-for="opt in (param.options || [])"
                :key="opt"
                :label="opt"
                :value="opt"
              />
            </el-select>
            <el-input
              v-else
              v-model="paramValues[param.varName]"
              :placeholder="param.helpText || `请输入${param.name}`"
            />
            <div v-if="param.helpText" class="param-help">{{ param.helpText }}</div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="showParamDialog = false">取消</el-button>
        <el-button type="primary" @click="applyTemplateWithParams">应用模板</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
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
import { executeTask, getAllJobTemplates } from '@/api/task'

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
const selectedGroupId = ref<number | null>(null)

// 过滤后的主机列表
const filteredHosts = computed(() => {
  let hosts = allHosts.value

  // 根据选中的分组过滤
  if (selectedGroupId.value !== null) {
    hosts = hosts.filter((host) => host.groupId === selectedGroupId.value)
  }

  // 根据搜索关键词过滤
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

// 过滤后的模板列表
const filteredTemplates = computed(() => {
  let result = allTemplates.value

  // 根据类型过滤
  if (templateFilter.value.type) {
    result = result.filter((template) => template.category === templateFilter.value.type)
  }

  // 根据名称过滤
  if (templateFilter.value.name) {
    const keyword = templateFilter.value.name.toLowerCase()
    result = result.filter((template) =>
      template.name.toLowerCase().includes(keyword)
    )
  }

  return result
})

// 模板对话框
const showTemplateDialog = ref(false)
const templateFilter = ref({
  type: '',
  name: '',
})

// 模板列表
const templates = ref<any[]>([])
const allTemplates = ref<any[]>([])
const templatesLoading = ref(false)

const selectedTemplate = ref<any>(null)

// 参数对话框
const showParamDialog = ref(false)
const currentTemplate = ref<any>(null)
const templateParams = ref<any[]>([])
const paramValues = ref<Record<string, string>>({})

// 加载主机分组
const loadHostGroups = async () => {
  try {
    const data = await getGroupTree()
    hostGroups.value = data || []
  } catch (error) {
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
    } else if (response.list && Array.isArray(response.list)) {
      allHosts.value = response.list
    } else if (response.data && Array.isArray(response.data)) {
      allHosts.value = response.data
    } else {
      allHosts.value = []
    }
  } catch (error) {
    allHosts.value = []
  } finally {
    hostsLoading.value = false
  }
}

// 加载模板列表
const loadTemplates = async () => {
  templatesLoading.value = true
  try {
    const response = await getAllJobTemplates()
    if (Array.isArray(response)) {
      allTemplates.value = response
    } else if (response.list && Array.isArray(response.list)) {
      allTemplates.value = response.list
    } else if (response.data && Array.isArray(response.data)) {
      allTemplates.value = response.data
    } else {
      allTemplates.value = []
    }
  } catch (error) {
    allTemplates.value = []
    ElMessage.error('加载模板列表失败')
  } finally {
    templatesLoading.value = false
  }
}

// 分组点击
const handleGroupClick = (data: any) => {
  selectedGroupId.value = data.id
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
  showTemplateDialog.value = false

  // 调试：打印模板数据

  // 解析模板参数
  let params: any[] = []
  if (row.variables && row.variables !== '[]') {
    try {
      params = JSON.parse(row.variables)
    } catch (e) {
      params = []
    }
  }

  // 如果模板有参数，弹出参数填写对话框
  if (params && params.length > 0) {
    currentTemplate.value = row
    templateParams.value = params
    // 初始化参数值（使用默认值）
    const values: Record<string, string> = {}
    params.forEach((param: any) => {
      values[param.varName] = param.defaultValue || ''
    })
    paramValues.value = values
    showParamDialog.value = true
  } else {
    // 没有参数，直接应用模板
    scriptContent.value = row.content
    ElMessage.success('已应用模板: ' + row.name)
  }
}

// 应用带参数的模板
const applyTemplateWithParams = () => {
  // 检查必填参数
  for (const param of templateParams.value) {
    if (param.required && !paramValues.value[param.varName]) {
      ElMessage.warning(`请填写参数: ${param.name}`)
      return
    }
  }

  // 替换模板中的变量占位符
  let content = currentTemplate.value.content
  for (const [varName, value] of Object.entries(paramValues.value)) {
    // 替换 {{变量名}} 格式的占位符
    const regex = new RegExp(`\\{\\{${varName}\\}\\}`, 'g')
    content = content.replace(regex, value)
  }

  scriptContent.value = content
  showParamDialog.value = false
  ElMessage.success('已应用模板: ' + currentTemplate.value.name)
}

// 刷新模板列表
const refreshTemplates = async () => {
  await loadTemplates()
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
const addLog = (message: string, status: string = 'info', host: string = '') => {
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
    host,
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
  const hostIds = selectedHosts.value.map((h) => h.id)
  addLog(`开始执行任务，目标主机: ${selectedHosts.value.length} 台`, 'info')

  try {
    const response = await executeTask({
      hostIds,
      scriptType: scriptType.value,
      content: scriptContent.value,
    })

    // 处理执行结果
    response.results.forEach((result) => {
      const hostInfo = `${result.hostName} (${result.hostIp})`
      if (result.status === 'success') {
        addLog(result.output || '执行完成，无输出', 'success', hostInfo)
      } else {
        addLog(`错误: ${result.error}\n${result.output || ''}`, 'error', hostInfo)
      }
    })

    // 检查是否全部成功
    const allSuccess = response.results.every((r) => r.status === 'success')
    if (allSuccess) {
      ElMessage.success('任务执行成功')
    } else {
      ElMessage.warning('部分任务执行失败，请查看执行记录')
    }
  } catch (error: any) {
    addLog('任务执行失败: ' + (error.message || error), 'error')
    ElMessage.error('任务执行失败: ' + (error.message || error))
  } finally {
    executing.value = false
  }
}

onMounted(() => {
  loadHostGroups()
  loadHostList()
})

// 监听模板对话框打开，自动加载模板列表
watch(showTemplateDialog, (newValue) => {
  if (newValue) {
    loadTemplates()
  }
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
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  background: #ecfeff;
  border: 1px solid #a5f3fc;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #0891b2;
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
  background: #000;
  color: #fff;
  border: none;
  padding: 12px 40px;
  font-size: 16px;

  &:hover {
    background: #333;
  }

  &:active {
    background: #1a1a1a;
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
  max-height: 500px;
  padding: 16px 20px;
  overflow-y: auto;
  background: #1e1e1e;
  border-radius: 0 0 8px 8px;

  .empty-log {
    text-align: center;
    color: #909399;
    padding: 40px 0;
  }
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-item {
  background: #2d2d2d;
  border-radius: 6px;
  overflow: hidden;
  border-left: 3px solid #909399;

  &.success {
    border-left-color: #67c23a;
  }

  &.error {
    border-left-color: #f56c6c;
  }

  &.info {
    border-left-color: #409eff;
  }

  .log-header-line {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: #252525;
    border-bottom: 1px solid #3a3a3a;
  }

  .log-time {
    color: #888;
    font-family: 'Courier New', Consolas, monospace;
    font-size: 12px;
  }

  .log-status-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;

    &.success {
      background: rgba(103, 194, 58, 0.2);
      color: #67c23a;
    }

    &.error {
      background: rgba(245, 108, 108, 0.2);
      color: #f56c6c;
    }

    &.info {
      background: rgba(64, 158, 255, 0.2);
      color: #409eff;
    }
  }

  .log-host {
    color: #e6a23c;
    font-family: 'Courier New', Consolas, monospace;
    font-size: 13px;
  }

  .log-output {
    padding: 12px 14px;

    pre {
      margin: 0;
      padding: 0;
      font-family: 'Courier New', Consolas, 'Liberation Mono', monospace;
      font-size: 13px;
      line-height: 1.6;
      color: #d4d4d4;
      white-space: pre-wrap;
      word-wrap: break-word;
      background: transparent;
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

// 模板表格行可点击样式
:deep(.el-table__row) {
  cursor: pointer;

  &:hover {
    background-color: #f5f7fa !important;
  }
}

// 参数对话框样式
.param-dialog-content {
  .param-help {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
  }
}
</style>
