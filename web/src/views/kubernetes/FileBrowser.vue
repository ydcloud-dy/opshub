<template>
  <el-dialog
    v-model="dialogVisible"
    :title="`文件浏览 - Pod: ${podName} | 容器: ${containerName}`"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-if="loading" class="loading-container">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <div v-else class="file-browser">
      <!-- 路径导航 -->
      <div class="breadcrumb-container">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item @click="navigateToRoot">
            <el-icon><House /></el-icon>
            根目录
          </el-breadcrumb-item>
          <el-breadcrumb-item
            v-for="(segment, index) in pathSegments"
            :key="index"
            @click="navigateToSegment(index)"
          >
            {{ segment }}
          </el-breadcrumb-item>
        </el-breadcrumb>
        <div class="current-path">{{ currentPathDisplay }}</div>
      </div>

      <!-- 工具栏 -->
      <div class="toolbar">
        <el-button size="small" @click="refreshFiles" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button size="small" @click="navigateUp" :disabled="currentPath === '/'">
          <el-icon><Back /></el-icon>
          返回上级
        </el-button>
        <el-upload
          :action="uploadUrl"
          :headers="uploadHeaders"
          :data="uploadData"
          :show-file-list="false"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :before-upload="beforeUpload"
        >
          <el-button size="small" type="primary" :loading="uploading">
            <el-icon><Upload /></el-icon>
            上传文件
          </el-button>
        </el-upload>
      </div>

      <!-- 文件列表 -->
      <div class="file-list-container">
        <el-table :data="files" size="small" class="file-table" v-loading="loading">
          <el-table-column width="50">
            <template #default="{ row }">
              <el-icon class="file-icon" :class="getFileIconClass(row)">
                <Folder v-if="row.isDir" />
                <Document v-else />
              </el-icon>
            </template>
          </el-table-column>
          <el-table-column label="名称" min-width="200">
            <template #default="{ row }">
              <span
                class="file-name"
                :class="{ 'directory': row.isDir }"
                @click="handleFileClick(row)"
              >
                {{ row.name }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="120" align="right">
            <template #default="{ row }">
              {{ row.isDir ? '-' : formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column label="权限" width="100" align="center">
            <template #default="{ row }">
              <code class="permission-code">{{ row.mode || '-' }}</code>
            </template>
          </el-table-column>
          <el-table-column label="修改时间" width="160">
            <template #default="{ row }">
              {{ formatDate(row.modTime) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center">
            <template #default="{ row }">
              <el-button
                v-if="!row.isDir"
                type="primary"
                link
                size="small"
                @click="downloadFile(row)"
                :loading="downloadingFiles[row.name]"
              >
                <el-icon><Download /></el-icon>
                下载
              </el-button>
              <span v-else class="not-applicable">-</span>
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-if="!loading && files.length === 0" description="目录为空" />
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, House, Refresh, Back, Folder, Document, Download, Upload } from '@element-plus/icons-vue'
import axios from 'axios'

interface FileInfo {
  name: string
  size: string
  mode: string
  isDir: boolean
  modTime: string
  user: string
  group: string
  link: string
  path: string
}

const props = defineProps<{
  visible: boolean
  clusterId: number | string
  namespace: string
  podName: string
  containerName: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const loading = ref(false)
const uploading = ref(false)
const downloadingFiles = ref<Record<string, boolean>>({})
const files = ref<FileInfo[]>([])
const currentPath = ref('/')

const pathSegments = computed(() => {
  const path = currentPath.value.startsWith('/') ? currentPath.value.slice(1) : currentPath.value
  return path ? path.split('/').filter(p => p) : []
})

const currentPathDisplay = computed(() => {
  return currentPath.value || '/'
})

// 上传相关的计算属性
const uploadUrl = computed(() => {
  return '/api/v1/plugins/kubernetes/pods/files/upload'
})

const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token')
  return {
    Authorization: `Bearer ${token}`
  }
})

const uploadData = computed(() => {
  return {
    cluster_id: props.clusterId,
    namespace: props.namespace,
    podName: props.podName,
    containerName: props.containerName,
    path: currentPath.value
  }
})

const getFileIconClass = (file: FileInfo) => {
  if (file.isDir) {
    return 'icon-directory'
  }
  return 'icon-file'
}

const formatSize = (size: string): string => {
  if (!size || size === '-') return '-'

  // 后端返回的是字节数字符串，需要格式化
  const bytes = parseInt(size)
  if (isNaN(bytes)) return size

  if (bytes === 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return (bytes / Math.pow(k, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const loadFiles = async () => {
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    console.log('📂 Loading files:', {
      clusterId: props.clusterId,
      namespace: props.namespace,
      podName: props.podName,
      containerName: props.containerName,
      path: currentPath.value
    })

    const response = await axios.get('/api/v1/plugins/kubernetes/pods/files', {
      params: {
        clusterId: props.clusterId,
        namespace: props.namespace,
        podName: props.podName,
        containerName: props.containerName,
        path: currentPath.value
      },
      headers: { Authorization: `Bearer ${token}` },
      timeout: 60000 // 60秒超时
    })

    console.log('✅ API Response:', response.data)

    // 适配新的响应格式 {code: 0, data: {files: [...]}, msg: "获取成功"}
    if (response.data.code === 0 && response.data.data) {
      files.value = response.data.data.files || []
      console.log('✅ Files loaded:', files.value.length)
    } else {
      // 兼容旧格式
      files.value = response.data.files || []
      console.log('✅ Files loaded (legacy):', files.value.length)
    }
  } catch (error: any) {
    console.error('❌ 获取文件列表失败:', error)
    console.error('❌ Error code:', error.code)
    console.error('❌ Error message:', error.message)
    console.error('❌ Error response:', error.response?.data)

    let errorMsg = '获取文件列表失败'
    if (error.code === 'ECONNABORTED') {
      errorMsg = '请求超时，请检查Pod是否正常运行'
    } else if (error.response?.data?.msg) {
      errorMsg = error.response.data.msg
    } else if (error.response?.data?.message) {
      errorMsg = error.response.data.message
    } else if (error.message) {
      errorMsg = error.message
    }

    ElMessage.error('文件列表加载失败: ' + errorMsg)
    files.value = []
  } finally {
    loading.value = false
  }
}

const refreshFiles = () => {
  loadFiles()
}

const navigateToRoot = () => {
  currentPath.value = '/'
  loadFiles()
}

const navigateUp = () => {
  if (currentPath.value === '/') return
  const pathParts = currentPath.value.split('/').filter(p => p)
  pathParts.pop()
  currentPath.value = '/' + pathParts.join('/')
  loadFiles()
}

const navigateToSegment = (index: number) => {
  const segments = pathSegments.value.slice(0, index + 1)
  currentPath.value = '/' + segments.join('/')
  loadFiles()
}

const handleFileClick = (file: FileInfo) => {
  if (file.isDir) {
    // 进入目录
    const newPath = currentPath.value === '/'
      ? '/' + file.name
      : currentPath.value + '/' + file.name
    currentPath.value = newPath
    loadFiles()
  } else {
    // 点击文件，可以预览或下载
    ElMessage.info(`文件: ${file.name}`)
  }
}

const downloadFile = async (file: FileInfo) => {
  try {
    const token = localStorage.getItem('token')
    const filePath = currentPath.value === '/'
      ? '/' + file.name
      : currentPath.value + '/' + file.name

    // 标记该文件正在下载
    downloadingFiles.value[file.name] = true

    const response = await axios.get('/api/v1/plugins/kubernetes/pods/files/download', {
      params: {
        cluster_id: props.clusterId,
        namespace: props.namespace,
        podName: props.podName,
        containerName: props.containerName,
        path: filePath
      },
      headers: { Authorization: `Bearer ${token}` },
      responseType: 'blob',
      timeout: 120000 // 120秒超时
    })

    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)

    ElMessage.success(`文件 ${file.name} 下载成功`)
  } catch (error: any) {
    console.error('下载文件失败:', error)
    const errorMsg = error.response?.data?.msg || error.response?.data?.message || '下载文件失败'
    ElMessage.error(errorMsg)
  } finally {
    // 清除下载状态
    downloadingFiles.value[file.name] = false
  }
}

// 上传前处理
const beforeUpload = (file: File) => {
  uploading.value = true
  console.log('📤 Uploading file:', file.name, 'size:', file.size)
  return true
}

// 上传成功处理
const handleUploadSuccess = (response: any) => {
  uploading.value = false
  console.log('✅ Upload response:', response)

  if (response.code === 0) {
    ElMessage.success(response.msg || '文件上传成功')
    // 刷新文件列表
    loadFiles()
  } else {
    ElMessage.error(response.msg || '文件上传失败')
  }
}

// 上传失败处理
const handleUploadError = (error: any) => {
  uploading.value = false
  console.error('❌ Upload error:', error)
  const errorMsg = error.response?.data?.msg || error.response?.data?.message || '文件上传失败'
  ElMessage.error(errorMsg)
}

const handleClose = () => {
  dialogVisible.value = false
  currentPath.value = '/'
  files.value = []
}

// 监听 visible 变化，加载数据
watch(() => props.visible, (newVal) => {
  console.log('🔍 FileBrowser visibility changed:', newVal)
  console.log('🔍 FileBrowser props:', {
    clusterId: props.clusterId,
    namespace: props.namespace,
    podName: props.podName,
    containerName: props.containerName
  })
  if (newVal) {
    if (!props.clusterId) {
      ElMessage.error('集群ID未设置')
      return
    }
    currentPath.value = '/'
    loadFiles()
  }
})
</script>

<style scoped>
/* 对话框样式 - 简洁黑白风格 */
:deep(.el-dialog) {
  background: #fff;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

:deep(.el-dialog__header) {
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  padding: 16px 20px;
  border-radius: 8px 8px 0 0;
}

:deep(.el-dialog__title) {
  color: #000;
  font-size: 16px;
  font-weight: 600;
}

:deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #909399;
  font-size: 18px;
}

:deep(.el-dialog__headerbtn .el-dialog__close:hover) {
  color: #000;
}

:deep(.el-dialog__body) {
  background: #fff;
  padding: 20px;
  color: #000;
}

/* 加载容器 */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  gap: 16px;
  font-size: 14px;
  color: #606266;
  background: #f5f5f5;
  border-radius: 8px;
}

.loading-container .el-icon {
  font-size: 32px;
  color: #409eff;
}

/* 文件浏览器容器 */
.file-browser {
  padding: 0;
}

/* 面包屑导航 */
.breadcrumb-container {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #f5f5f5;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
}

.breadcrumb-container :deep(.el-breadcrumb) {
  margin-bottom: 8px;
}

.breadcrumb-container :deep(.el-breadcrumb__item) {
  cursor: pointer;
  color: #606266;
  transition: color 0.2s;
}

.breadcrumb-container :deep(.el-breadcrumb__item:hover) {
  color: #000;
}

.breadcrumb-container :deep(.el-breadcrumb__item__inner) {
  cursor: pointer;
  font-weight: 500;
}

.breadcrumb-container :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
  color: #000;
  font-weight: 600;
}

.current-path {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  color: #000;
  padding: 6px 12px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  display: inline-block;
}

/* 工具栏 */
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  align-items: center;
}

.toolbar :deep(.el-button) {
  background: #fff;
  border: 1px solid #dcdfe6;
  color: #606266;
  transition: all 0.2s;
}

.toolbar :deep(.el-button:hover) {
  border-color: #000;
  color: #000;
}

.toolbar :deep(.el-button:disabled) {
  background: #f5f5f5;
  border-color: #e0e0e0;
  color: #c0c4cc;
}

.toolbar :deep(.el-button--primary) {
  background: #000;
  border-color: #000;
  color: #fff;
}

.toolbar :deep(.el-button--primary:hover) {
  background: #333;
  border-color: #333;
}

/* 上传组件 */
.toolbar :deep(.el-upload) {
  display: inline-block;
}

/* 文件列表容器 */
.file-list-container {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.file-table {
  width: 100%;
}

.file-table :deep(.el-table__header-wrapper) {
  background: #f5f5f5;
}

.file-table :deep(.el-table__header th) {
  background: transparent;
  color: #000;
  font-weight: 600;
  border-bottom: 1px solid #e0e0e0;
  padding: 12px 12px;
}

.file-table :deep(.el-table__body tr) {
  background: #fff;
  transition: background-color 0.2s;
}

.file-table :deep(.el-table__body tr:hover) {
  background: #f5f5f5;
}

.file-table :deep(.el-table__body td) {
  border-bottom: 1px solid #f0f0f0;
  padding: 12px 12px;
  color: #000;
}

.file-table :deep(.el-table__empty-block) {
  background: transparent;
  color: #909399;
}

/* 文件图标 */
.file-icon {
  font-size: 18px;
}

.icon-directory {
  color: #000;
}

.icon-file {
  color: #909399;
}

/* 文件名样式 */
.file-name {
  cursor: pointer;
  word-break: break-all;
  transition: color 0.2s;
}

.file-name.directory {
  color: #000;
  font-weight: 600;
}

.file-name.directory:hover {
  color: #409eff;
  text-decoration: underline;
}

/* 权限代码 */
.permission-code {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  padding: 3px 8px;
  background: #f5f5f5;
  border: 1px solid #e0e0e0;
  border-radius: 3px;
  color: #606266;
}

/* 下载按钮 */
.file-table :deep(.el-button--primary.is-link) {
  color: #000;
}

.file-table :deep(.el-button--primary.is-link:hover) {
  color: #409eff;
}

/* 空状态 */
.not-applicable {
  color: #c0c4cc;
}

/* Empty 组件样式 */
.file-list-container :deep(.el-empty) {
  padding: 40px 20px;
}

.file-list-container :deep(.el-empty__description p) {
  color: #909399;
}

/* Dialog 关闭按钮 */
:deep(.el-dialog__footer) {
  border-top: 1px solid #e0e0e0;
  background: #f5f5f5;
}
</style>
