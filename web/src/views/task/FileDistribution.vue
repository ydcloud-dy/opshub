<template>
  <div class="distribution-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><FolderOpened /></el-icon>
        </div>
        <div>
          <h2 class="page-title">文件分发</h2>
          <p class="page-subtitle">分发文件到多台主机，支持批量操作和断点续传</p>
        </div>
      </div>
    </div>

    <div class="distribution-main">
      <!-- 数据源 -->
      <div class="section-card">
        <div class="section-header">
          <span class="section-title">数据源</span>
        </div>
        <div class="section-content">
          <div class="source-tabs">
            <el-button
              :type="sourceType === 'upload' ? 'primary' : ''"
              @click="sourceType = 'upload'"
            >
              <el-icon style="margin-right: 6px;"><Upload /></el-icon>
              上传本地文件
            </el-button>
            <el-button
              :type="sourceType === 'host' ? 'primary' : ''"
              @click="sourceType = 'host'"
            >
              <el-icon style="margin-right: 6px;"><Monitor /></el-icon>
              添加主机文件
            </el-button>
          </div>

          <div v-if="sourceType === 'upload'" class="source-content">
            <el-upload
              ref="uploadRef"
              class="upload-area"
              drag
              :auto-upload="false"
              :on-change="handleFileChange"
              :file-list="fileList"
            >
              <div class="upload-content">
                <el-icon class="upload-icon"><UploadFilled /></el-icon>
                <div class="upload-text">
                  <p>拖拽文件到此处或点击上传</p>
                  <p class="upload-hint">支持任意格式文件</p>
                </div>
              </div>
            </el-upload>

            <div v-if="fileList.length > 0" class="file-list">
              <div v-for="(file, index) in fileList" :key="index" class="file-item">
                <el-icon class="file-icon"><Document /></el-icon>
                <span class="file-name">{{ file.name }}</span>
                <span class="file-size">{{ formatFileSize(file.size || 0) }}</span>
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="removeFile(index)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </div>

            <div v-else class="empty-state">
              暂无数据
            </div>
          </div>

          <div v-if="sourceType === 'host'" class="source-content">
            <el-empty description="暂未实现主机文件选择功能" :image-size="100" />
          </div>
        </div>
      </div>

      <!-- 分发目标 -->
      <div class="section-card">
        <div class="section-header">
          <span class="section-title">分发目标</span>
          <el-icon class="section-tip"><InfoFilled /></el-icon>
        </div>
        <div class="section-content">
          <div class="form-item">
            <label class="form-label">
              <span class="required">*</span>
              目标路径:
            </label>
            <el-input
              v-model="targetPath"
              placeholder="请输入目标路径"
              clearable
            />
          </div>

          <div class="form-item">
            <label class="form-label">
              <span class="required">*</span>
              目标主机:
            </label>
            <el-button @click="showHostDialog = true">
              <el-icon style="margin-right: 6px;"><Plus /></el-icon>
              添加目标主机
            </el-button>
            <div v-if="selectedHosts.length > 0" class="selected-hosts">
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
      </div>

      <!-- 开始执行按钮 -->
      <div class="execute-actions">
        <el-button
          type="primary"
          size="large"
          :loading="distributing"
          @click="handleDistribute"
          class="execute-button"
        >
          <el-icon style="margin-right: 6px;"><VideoPlay /></el-icon>
          {{ distributing ? '分发中...' : '开始执行' }}
        </el-button>
      </div>
    </div>

    <!-- 分发记录 -->
    <div class="distribution-log">
      <div class="log-header">
        <span>分发记录</span>
        <el-icon><InfoFilled /></el-icon>
      </div>
      <div class="log-content">
        <div v-if="distributionLogs.length === 0" class="empty-log">
          暂无分发记录
        </div>
        <div v-else class="log-list">
          <div
            v-for="log in distributionLogs"
            :key="log.id"
            class="log-item"
            :class="log.status"
          >
            <div class="log-time">{{ log.time }}</div>
            <div class="log-message">{{ log.message }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 选择主机对话框 -->
    <el-dialog
      v-model="showHostDialog"
      title="选择主机"
      width="800px"
      destroy-on-close
    >
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
      <template #footer>
        <el-button @click="showHostDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmHostSelection">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Upload,
  UploadFilled,
  Monitor,
  Document,
  Close,
  InfoFilled,
  Plus,
  VideoPlay,
  Search,
  FolderOpened
} from '@element-plus/icons-vue'
import type { UploadUserFile } from 'element-plus'
import { getHostList } from '@/api/host'

// 数据源类型
const sourceType = ref('upload')

// 文件列表
const fileList = ref<UploadUserFile[]>([])
const uploadRef = ref()

// 目标路径
const targetPath = ref('')

// 选中的主机
const selectedHosts = ref<any[]>([])

// 分发状态
const distributing = ref(false)

// 分发日志
const distributionLogs = ref<any[]>([])

// 主机对话框
const showHostDialog = ref(false)
const hostSearchKeyword = ref('')
const tempSelectedHosts = ref<any[]>([])
const allHosts = ref<any[]>([])
const hostsLoading = ref(false)

// 过滤后的主机列表
const filteredHosts = computed(() => {
  let hosts = allHosts.value

  // 按关键词过滤
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

// 文件变化
const handleFileChange = (file: any, files: UploadUserFile[]) => {
  fileList.value = files
}

// 移除文件
const removeFile = (index: number) => {
  fileList.value.splice(index, 1)
  if (uploadRef.value) {
    uploadRef.value.clearFiles()
  }
}

// 格式化文件大小
const formatFileSize = (size: number) => {
  if (size < 1024) {
    return size + ' B'
  } else if (size < 1024 * 1024) {
    return (size / 1024).toFixed(2) + ' KB'
  } else {
    return (size / (1024 * 1024)).toFixed(2) + ' MB'
  }
}

// 移除主机
const removeHost = (id: number) => {
  const index = selectedHosts.value.findIndex((h) => h.id === id)
  if (index !== -1) {
    selectedHosts.value.splice(index, 1)
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
    // 根据API返回格式调整
    if (Array.isArray(response)) {
      allHosts.value = response
    } else if (response.data && Array.isArray(response.data)) {
      allHosts.value = response.data
    } else {
      allHosts.value = []
    }
  } catch (error) {
    ElMessage.error('加载主机列表失败')
    console.error('加载主机列表失败:', error)
    allHosts.value = []
  } finally {
    hostsLoading.value = false
  }
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

// 添加日志
const addLog = (message: string, status: string = 'info') => {
  const now = new Date()
  const time = `${now.getHours().toString().padStart(2, '0')}:${now
    .getMinutes()
    .toString()
    .padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
  distributionLogs.value.unshift({
    id: Date.now(),
    time,
    message,
    status,
  })
}

// 执行分发
const handleDistribute = async () => {
  if (fileList.value.length === 0) {
    ElMessage.warning('请先上传文件')
    return
  }
  if (!targetPath.value.trim()) {
    ElMessage.warning('请输入目标路径')
    return
  }
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请选择目标主机')
    return
  }

  distributing.value = true
  addLog(
    `开始分发文件: ${fileList.value.map((f) => f.name).join(', ')}，目标主机: ${selectedHosts.value.length} 台`,
    'info'
  )

  try {
    // TODO: 调用文件分发API
    await new Promise((resolve) => setTimeout(resolve, 2000))
    addLog('文件分发成功', 'success')
    ElMessage.success('文件分发成功')
  } catch (error: any) {
    addLog('文件分发失败: ' + error.message, 'error')
    ElMessage.error('文件分发失败')
  } finally {
    distributing.value = false
  }
}

onMounted(() => {
  loadHostList()
})
</script>

<style scoped lang="scss">
.distribution-container {
  padding: 0;
  background-color: transparent;
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.page-header {
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.distribution-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}

.distribution-log {
  width: 350px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
}

.section-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;

  .section-title {
    margin-right: 8px;
  }

  .section-tip {
    color: #909399;
    cursor: help;
  }
}

.source-tabs {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.source-content {
  .upload-area {
    :deep(.el-upload) {
      width: 100%;
    }

    :deep(.el-upload-dragger) {
      width: 100%;
      padding: 60px 40px;
      border: 2px dashed #dcdfe6;
      border-radius: 8px;
      background: #fafafa;
      transition: all 0.3s;

      &:hover {
        border-color: #409eff;
        background: #f0f9ff;
      }
    }

    .upload-content {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 16px;

      .upload-icon {
        font-size: 64px;
        color: #c0c4cc;
      }

      .upload-text {
        text-align: center;

        p {
          margin: 0 0 8px 0;
          font-size: 16px;
          font-weight: 500;
          color: #303133;
        }

        .upload-hint {
          font-size: 13px;
          color: #909399;
        }
      }
    }
  }

  .file-list {
    margin-top: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;

    .file-icon {
      font-size: 20px;
      color: #409eff;
    }

    .file-name {
      flex: 1;
      font-size: 14px;
      color: #303133;
    }

    .file-size {
      font-size: 12px;
      color: #909399;
    }
  }

  .empty-state {
    text-align: center;
    color: #909399;
    padding: 40px 0;
  }
}

.form-item {
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }

  .form-label {
    display: block;
    margin-bottom: 8px;
    font-size: 14px;
    color: #606266;

    .required {
      color: #f56c6c;
      margin-right: 4px;
    }
  }

  .selected-hosts {
    margin-top: 12px;
  }
}

.execute-actions {
  display: flex;
  justify-content: center;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
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

.log-header {
  padding: 16px 20px;
  background: #fafafa;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.log-content {
  flex: 1;
  padding: 16px;
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
  padding: 12px;
  border-radius: 4px;
  border-left: 3px solid transparent;

  .log-time {
    font-size: 12px;
    color: #909399;
    margin-bottom: 4px;
    font-family: 'Courier New', monospace;
  }

  .log-message {
    font-size: 13px;
    color: #606266;
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
</style>
