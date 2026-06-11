<template>
  <div class="plugin-install-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Upload /></el-icon>
        </div>
        <div>
          <h2 class="page-title">插件安装</h2>
          <p class="page-subtitle">上传插件压缩包进行安装，系统将自动解压并部署前后端插件</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="handleBackToList">
          <el-icon style="margin-right: 6px;"><Back /></el-icon>
          返回列表
        </el-button>
      </div>
    </div>

    <!-- 安装说明 -->
    <div class="info-panel">
      <div class="panel-header">
        <el-icon class="header-icon"><InfoFilled /></el-icon>
        <span class="header-title">安装说明</span>
      </div>
      <div class="info-content">
        <div class="info-item">
          <div class="info-icon">
            <el-icon color="#409eff"><Check /></el-icon>
          </div>
          <div class="info-text">
            <span class="info-title">插件包格式</span>
            <span class="info-desc">上传的插件必须是 .zip 格式的压缩包</span>
          </div>
        </div>
        <div class="info-item">
          <div class="info-icon">
            <el-icon color="#409eff"><Check /></el-icon>
          </div>
          <div class="info-text">
            <span class="info-title">目录结构</span>
            <span class="info-desc">压缩包内应包含 web/ 目录（前端插件）和 backend/ 目录（后端插件）</span>
          </div>
        </div>
        <div class="info-item">
          <div class="info-icon">
            <el-icon color="#409eff"><Check /></el-icon>
          </div>
          <div class="info-text">
            <span class="info-title">自动部署</span>
            <span class="info-desc">上传后系统将自动解压并将前后端插件放置到对应目录</span>
          </div>
        </div>
        <div class="info-item">
          <div class="info-icon">
            <el-icon color="#f56c6c"><Warning /></el-icon>
          </div>
          <div class="info-text">
            <span class="info-title">重要提示</span>
            <span class="info-desc">安装插件后需要重启服务并手动注册才能生效，请谨慎操作</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 上传区域 -->
    <div class="upload-panel">
      <div class="panel-header">
        <el-icon class="header-icon"><FolderOpened /></el-icon>
        <span class="header-title">上传插件包</span>
      </div>
      <div class="upload-wrapper">
        <el-upload
          ref="uploadRef"
          class="plugin-upload"
          drag
          :auto-upload="false"
          :limit="1"
          accept=".zip"
          :on-change="handleFileChange"
          :on-exceed="handleExceed"
          :file-list="fileList"
        >
          <div class="upload-content">
            <el-icon class="upload-icon"><UploadFilled /></el-icon>
            <div class="upload-text">
              <p class="upload-title">拖拽文件到此处或点击上传</p>
              <p class="upload-hint">仅支持 .zip 格式的插件压缩包，文件大小不超过 50MB</p>
            </div>
          </div>
        </el-upload>

        <!-- 文件信息显示 -->
        <div v-if="selectedFile" class="file-info-card">
          <div class="file-info-header">
            <el-icon class="file-icon" color="#409eff"><Document /></el-icon>
            <div class="file-details">
              <span class="file-name">{{ selectedFile.name }}</span>
              <span class="file-size">{{ formatFileSize(selectedFile.size) }}</span>
            </div>
            <el-tag size="small" type="success">已选择</el-tag>
          </div>
          <el-progress
            v-if="uploading"
            :percentage="uploadProgress"
            :status="uploadStatus"
            :stroke-width="8"
          />
        </div>

        <!-- 操作按钮 -->
        <div class="upload-actions">
          <el-button
            type="primary"
            size="large"
            :disabled="!selectedFile || uploading"
            :loading="uploading"
            @click="handleUpload"
            class="black-button"
          >
            <el-icon style="margin-right: 6px;"><Upload /></el-icon>
            {{ uploading ? '安装中...' : '开始安装' }}
          </el-button>
          <el-button
            size="large"
            :disabled="uploading"
            @click="handleClear"
          >
            清空
          </el-button>
        </div>
      </div>
    </div>

    <!-- 安装日志 -->
    <div v-if="installLog.length > 0" class="log-panel">
      <div class="panel-header">
        <el-icon class="header-icon"><List /></el-icon>
        <span class="header-title">安装日志</span>
      </div>
      <div class="log-content">
        <div v-for="(log, index) in installLog" :key="index" class="log-item" :class="log.type">
          <span class="log-time">{{ log.time }}</span>
          <span class="log-text">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UploadInstance, UploadUserFile, UploadFile } from 'element-plus'
import {
  Upload,
  Back,
  InfoFilled,
  Check,
  Warning,
  FolderOpened,
  UploadFilled,
  Document,
  List
} from '@element-plus/icons-vue'
import { uploadPlugin } from '@/api/plugin'

const router = useRouter()
const uploadRef = ref<UploadInstance>()
const fileList = ref<UploadUserFile[]>([])
const selectedFile = ref<File | null>(null)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadStatus = ref<'success' | 'exception' | 'warning' | ''>('')
const installLog = ref<Array<{ time: string; message: string; type: string }>>([])

// 文件选择变化
const handleFileChange = (file: UploadFile) => {
  if (!file.name.endsWith('.zip')) {
    ElMessage.error('只能上传 .zip 格式的文件')
    fileList.value = []
    selectedFile.value = null
    return
  }
  selectedFile.value = file.raw as File
  addLog('文件已选择: ' + file.name, 'info')
}

// 超出文件数量限制
const handleExceed = () => {
  ElMessage.warning('只能上传一个文件')
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

// 添加日志
const addLog = (message: string, type: string = 'info') => {
  const now = new Date()
  const time = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
  installLog.value.push({ time, message, type })
}

// 上传插件
const handleUpload = async () => {
  if (!selectedFile.value) {
    ElMessage.warning('请先选择要上传的插件包')
    return
  }

  try {
    await ElMessageBox.confirm(
      '确定要安装此插件吗？安装后需要重启服务并手动注册才能生效。',
      '确认安装',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    uploading.value = true
    uploadProgress.value = 0
    uploadStatus.value = ''
    addLog('开始上传插件包...', 'info')

    // 模拟上传进度
    const progressInterval = setInterval(() => {
      if (uploadProgress.value < 90) {
        uploadProgress.value += 10
      }
    }, 200)

    try {
      await uploadPlugin(selectedFile.value)
      clearInterval(progressInterval)
      uploadProgress.value = 100
      uploadStatus.value = 'success'
      addLog('插件上传成功', 'success')
      addLog('插件解压完成', 'success')
      addLog('插件文件已复制到对应目录', 'success')
      addLog('请按照文档手动注册插件并重启服务', 'warning')

      ElMessage.success('插件安装成功，请按照文档手动注册插件')

      // 3秒后跳转到插件列表
      setTimeout(() => {
        router.push('/plugin/list')
      }, 3000)
    } catch (error: any) {
      clearInterval(progressInterval)
      uploadStatus.value = 'exception'
      addLog('插件安装失败: ' + (error.message || '未知错误'), 'error')
      ElMessage.error(error.message || '插件安装失败')
    } finally {
      uploading.value = false
    }
  } catch {
    // 用户取消
  }
}

// 清空选择
const handleClear = () => {
  fileList.value = []
  selectedFile.value = null
  uploadRef.value?.clearFiles()
  addLog('已清空选择', 'info')
}

// 返回列表
const handleBackToList = () => {
  router.push('/plugin/list')
}
</script>

<style scoped lang="scss">
.plugin-install-container {
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background-color: transparent;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
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
  background: #f5f3ff;
  border: 1px solid #ddd6fe;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #7c3aed;
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

.info-panel,
.upload-panel,
.log-panel {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;

  .panel-header {
    padding: 16px 20px;
    background: #fafafa;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    gap: 8px;

    .header-icon {
      font-size: 18px;
      color: #409eff;
    }

    .header-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }
  }
}

.info-content {
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;

  .info-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;

    .info-icon {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      background: #f0f9ff;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
      font-size: 18px;
    }

    .info-text {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 4px;

      .info-title {
        font-size: 14px;
        font-weight: 600;
        color: #303133;
      }

      .info-desc {
        font-size: 13px;
        color: #606266;
        line-height: 1.5;
      }
    }
  }
}

.upload-wrapper {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;

  .plugin-upload {
    :deep(.el-upload) {
      width: 100%;
    }

    :deep(.el-upload-dragger) {
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

        .upload-title {
          margin: 0 0 8px 0;
          font-size: 16px;
          font-weight: 500;
          color: #303133;
        }

        .upload-hint {
          margin: 0;
          font-size: 13px;
          color: #909399;
        }
      }
    }
  }

  .file-info-card {
    padding: 16px;
    background: #f0f9ff;
    border-radius: 8px;
    border: 1px solid #b3d8ff;

    .file-info-header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 12px;

      .file-icon {
        font-size: 24px;
      }

      .file-details {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;

        .file-name {
          font-size: 14px;
          font-weight: 500;
          color: #303133;
          word-break: break-all;
        }

        .file-size {
          font-size: 12px;
          color: #909399;
        }
      }
    }
  }

  .upload-actions {
    display: flex;
    justify-content: center;
    gap: 12px;
  }
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;

  &:hover {
    background-color: #1a1a1a !important;
  }
}

.log-content {
  max-height: 300px;
  overflow-y: auto;
  padding: 16px 20px;

  .log-item {
    padding: 10px 12px;
    margin-bottom: 8px;
    border-radius: 4px;
    font-size: 13px;
    display: flex;
    gap: 12px;
    border-left: 3px solid transparent;

    &:last-child {
      margin-bottom: 0;
    }

    .log-time {
      color: #909399;
      flex-shrink: 0;
      font-family: 'Courier New', monospace;
    }

    .log-text {
      flex: 1;
      color: #606266;
    }

    &.info {
      background: #ecf5ff;
      border-left-color: #409eff;
    }

    &.success {
      background: #f0f9ff;
      border-left-color: #67c23a;

      .log-text {
        color: #67c23a;
        font-weight: 500;
      }
    }

    &.warning {
      background: #fdf6ec;
      border-left-color: #e6a23c;

      .log-text {
        color: #e6a23c;
        font-weight: 500;
      }
    }

    &.error {
      background: #fef0f0;
      border-left-color: #f56c6c;

      .log-text {
        color: #f56c6c;
        font-weight: 500;
      }
    }
  }
}
</style>
