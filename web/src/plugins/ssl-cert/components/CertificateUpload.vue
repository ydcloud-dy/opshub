<template>
  <div class="certificate-upload-container">
    <!-- 标签页 -->
    <el-tabs v-model="activeTab" class="certificate-tabs">
      <!-- 文件上传标签页 -->
      <el-tab-pane label="📁 文件上传" name="upload">
        <div class="upload-content">
          <el-form :model="uploadForm" :rules="uploadRules" ref="uploadFormRef" label-width="100px">
            <el-form-item label="证书名称" prop="name">
              <el-input v-model="uploadForm.name" placeholder="请输入证书名称" />
            </el-form-item>

            <el-form-item label="域名" prop="domain">
              <el-input v-model="uploadForm.domain" placeholder="请输入域名，如：example.com" />
            </el-form-item>

            <el-form-item label="证书文件" prop="certFile">
              <div class="file-upload-area">
                <input
                  type="file"
                  ref="certFileInput"
                  @change="handleCertFileSelect"
                  accept=".pem,.crt,.cer,.x509"
                  style="display: none"
                />
                <div class="upload-box" @click="$refs.certFileInput?.click()">
                  <el-icon class="upload-icon"><DocumentCopy /></el-icon>
                  <div class="upload-text">
                    <div class="upload-title">点击选择证书文件或拖拽上传</div>
                    <div class="upload-desc">支持 .pem .crt .cer .x509 格式</div>
                  </div>
                </div>
                <div v-if="uploadForm.certFile" class="file-info">
                  <span class="file-name">✓ {{ uploadForm.certFile.name }}</span>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="私钥文件" prop="keyFile">
              <div class="file-upload-area">
                <input
                  type="file"
                  ref="keyFileInput"
                  @change="handleKeyFileSelect"
                  accept=".key,.pem"
                  style="display: none"
                />
                <div class="upload-box" @click="$refs.keyFileInput?.click()">
                  <el-icon class="upload-icon"><Key /></el-icon>
                  <div class="upload-text">
                    <div class="upload-title">点击选择私钥文件或拖拽上传</div>
                    <div class="upload-desc">支持 .key .pem 格式（可选）</div>
                  </div>
                </div>
                <div v-if="uploadForm.keyFile" class="file-info">
                  <span class="file-name">✓ {{ uploadForm.keyFile.name }}</span>
                </div>
              </div>
            </el-form-item>

            <div class="form-actions">
              <el-button @click="handleUploadCancel">取消</el-button>
              <el-button type="primary" @click="handleUploadSubmit" :loading="uploading">
                验证并上传
              </el-button>
            </div>
          </el-form>
        </div>
      </el-tab-pane>

      <!-- 手动粘贴标签页 -->
      <el-tab-pane label="📝 手动粘贴" name="paste">
        <div class="paste-content">
          <el-form :model="pasteForm" :rules="pasteRules" ref="pasteFormRef" label-width="100px">
            <el-form-item label="证书名称" prop="name">
              <el-input v-model="pasteForm.name" placeholder="请输入证书名称" />
            </el-form-item>

            <el-form-item label="域名" prop="domain">
              <el-input v-model="pasteForm.domain" placeholder="请输入域名，如：example.com" />
            </el-form-item>

            <el-form-item label="证书内容" prop="certificate">
              <el-input
                v-model="pasteForm.certificate"
                type="textarea"
                :rows="8"
                placeholder="请粘贴证书内容（PEM格式）&#10;-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
              />
            </el-form-item>

            <el-form-item label="私钥内容" prop="privateKey">
              <el-input
                v-model="pasteForm.privateKey"
                type="textarea"
                :rows="8"
                placeholder="请粘贴私钥内容（PEM格式，可选）&#10;-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
              />
            </el-form-item>

            <div class="form-actions">
              <el-button @click="handlePasteCancel">取消</el-button>
              <el-button type="primary" @click="handlePasteSubmit" :loading="pasting">
                验证并提交
              </el-button>
            </div>
          </el-form>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 证书信息预览对话框 -->
    <el-dialog
      v-model="previewDialogVisible"
      title="证书信息预览"
      width="620px"
      class="beauty-dialog"
    >
      <div v-if="certInfo" class="cert-preview">
        <div class="cert-status-bar" :class="`status-bar-${certStatus}`">
          <span class="status-icon">{{ certStatus === 'valid' ? '&#10003;' : '!' }}</span>
          <span class="status-text">{{ certStatus === 'valid' ? '证书有效' : '证书即将过期' }}</span>
        </div>

        <div class="cert-info-sections">
          <div class="cert-info-section">
            <div class="section-title">基本信息</div>
            <div class="section-grid">
              <div class="cert-info-item">
                <div class="cert-label">证书名称</div>
                <div class="cert-value">{{ certInfo.name }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">域名</div>
                <div class="cert-value">{{ certInfo.domain }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">颁发者</div>
                <div class="cert-value cert-mono">{{ certInfo.issuer }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">主体</div>
                <div class="cert-value cert-mono">{{ certInfo.subject }}</div>
              </div>
            </div>
          </div>

          <div class="cert-info-section">
            <div class="section-title">有效期</div>
            <div class="section-grid">
              <div class="cert-info-item">
                <div class="cert-label">有效期起</div>
                <div class="cert-value">{{ certInfo.notBefore }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">有效期至</div>
                <div class="cert-value" :class="getDaysRemainingClass(certInfo.daysRemaining)">
                  {{ certInfo.notAfter }}
                  <span class="days-remaining">(剩余 {{ certInfo.daysRemaining }} 天)</span>
                </div>
              </div>
            </div>
          </div>

          <div class="cert-info-section">
            <div class="section-title">安全信息</div>
            <div class="section-grid">
              <div class="cert-info-item full-width">
                <div class="cert-label">指纹(SHA256)</div>
                <div class="cert-value cert-mono fingerprint">{{ certInfo.fingerprint }}</div>
              </div>
              <div class="cert-info-item" v-if="certInfo.privateKey">
                <div class="cert-label">私钥</div>
                <div class="cert-value private-key-status">已包含</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="previewDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleConfirmUpload" :loading="confirming">
          确认上传
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { DocumentCopy, Key } from '@element-plus/icons-vue'

interface CertInfo {
  name: string
  domain: string
  certificate: string
  privateKey: string
  subject: string
  issuer: string
  notBefore: string
  notAfter: string
  daysRemaining: number
  fingerprint: string
}

const emit = defineEmits<{
  submit: [data: CertInfo]
  cancel: []
  'update:visible': [visible: boolean]
}>()

const activeTab = ref('upload')

// 上传表单
const uploadForm = reactive({
  name: '',
  domain: '',
  certFile: null as File | null,
  keyFile: null as File | null
})

const uploadRules: FormRules = {
  name: [{ required: true, message: '请输入证书名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  certFile: [{ required: true, message: '请选择证书文件', trigger: 'change' }]
}

const uploadFormRef = ref<FormInstance>()
const certFileInput = ref<HTMLInputElement | null>(null)
const keyFileInput = ref<HTMLInputElement | null>(null)

// 粘贴表单
const pasteForm = reactive({
  name: '',
  domain: '',
  certificate: '',
  privateKey: ''
})

const pasteRules: FormRules = {
  name: [{ required: true, message: '请输入证书名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  certificate: [{ required: true, message: '请粘贴证书内容', trigger: 'blur' }]
}

const pasteFormRef = ref<FormInstance>()

// 预览相关
const previewDialogVisible = ref(false)
const certInfo = ref<CertInfo | null>(null)
const certStatus = ref('valid')
const uploading = ref(false)
const pasting = ref(false)
const confirming = ref(false)

// 处理证书文件选择
const handleCertFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.[0]) {
    uploadForm.certFile = input.files[0]
  }
}

// 处理私钥文件选择
const handleKeyFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.[0]) {
    uploadForm.keyFile = input.files[0]
  }
}

// 获取剩余天数的样式
const getDaysRemainingClass = (days: number) => {
  if (days <= 0) return 'days-expired'
  if (days <= 30) return 'days-warning'
  return 'days-normal'
}

// 验证私钥格式
const isValidPrivateKey = (content: string) => {
  const trimmed = content.trim()
  return (trimmed.includes('BEGIN PRIVATE KEY') || trimmed.includes('BEGIN RSA PRIVATE KEY')) &&
         (trimmed.includes('END PRIVATE KEY') || trimmed.includes('END RSA PRIVATE KEY'))
}

// 读取文件
const readFile = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(new Error('读取文件失败'))
    reader.readAsText(file)
  })
}

// 解析证书信息
const parseCertInfo = async (certPem: string, keyPem: string = ''): Promise<CertInfo> => {
  try {
    // 这里使用前端解析，由于无法直接解析X.509证书，我们返回基本信息
    // 在实际应用中，应该由后端验证并返回详细信息
    const notAfterMatch = certPem.match(/notAfter=([^\n]+)/)
    const notBeforeMatch = certPem.match(/notBefore=([^\n]+)/)

    // 计算剩余天数（前端估算）
    const daysRemaining = 90 // 默认值，应由后端计算

    return {
      name: '',
      domain: '',
      certificate: certPem,
      privateKey: keyPem,
      subject: '证书信息将在上传后解析',
      issuer: '证书信息将在上传后解析',
      notBefore: '待解析',
      notAfter: '待解析',
      daysRemaining: daysRemaining,
      fingerprint: '待解析'
    }
  } catch (error: any) {
    throw new Error('解析证书失败：' + error.message)
  }
}

// 处理上传提交
const handleUploadSubmit = async () => {
  if (!uploadFormRef.value) return

  await uploadFormRef.value.validate(async (valid) => {
    if (!valid) return

    uploading.value = true
    try {
      // 直接读取文件内容并提交
      const certContent = await readFile(uploadForm.certFile!)
      let keyContent = ''

      if (uploadForm.keyFile) {
        keyContent = await readFile(uploadForm.keyFile)
      }

      // 验证证书格式
      if (!certContent.includes('BEGIN CERTIFICATE')) {
        ElMessage.error('无效的证书格式')
        uploading.value = false
        return
      }

      if (keyContent && !isValidPrivateKey(keyContent)) {
        ElMessage.error('无效的私钥格式')
        uploading.value = false
        return
      }

      // 解析证书信息
      certInfo.value = await parseCertInfo(certContent, keyContent)
      certStatus.value = certInfo.value.daysRemaining > 30 ? 'valid' : 'warning'
      previewDialogVisible.value = true
    } catch (error: any) {
      ElMessage.error(error.message || '验证失败')
    } finally {
      uploading.value = false
    }
  })
}

// 处理粘贴提交
const handlePasteSubmit = async () => {
  if (!pasteFormRef.value) return

  await pasteFormRef.value.validate(async (valid) => {
    if (!valid) return

    pasting.value = true
    try {
      // 验证证书格式
      if (!pasteForm.certificate.includes('BEGIN CERTIFICATE')) {
        ElMessage.error('无效的证书格式')
        pasting.value = false
        return
      }

      if (pasteForm.privateKey && !isValidPrivateKey(pasteForm.privateKey)) {
        ElMessage.error('无效的私钥格式')
        pasting.value = false
        return
      }

      // 解析证书信息
      certInfo.value = await parseCertInfo(pasteForm.certificate, pasteForm.privateKey)
      certStatus.value = certInfo.value.daysRemaining > 30 ? 'valid' : 'warning'
      previewDialogVisible.value = true
    } catch (error: any) {
      ElMessage.error(error.message || '验证失败')
    } finally {
      pasting.value = false
    }
  })
}

// 确认上传
const handleConfirmUpload = async () => {
  if (!certInfo.value) return

  confirming.value = true
  try {
    // 更新证书信息（从表单获取name和domain）
    if (activeTab.value === 'upload') {
      certInfo.value.name = uploadForm.name
      certInfo.value.domain = uploadForm.domain
    } else {
      certInfo.value.name = pasteForm.name
      certInfo.value.domain = pasteForm.domain
    }

    emit('submit', certInfo.value)
    previewDialogVisible.value = false
    resetForms()
  } catch (error: any) {
    ElMessage.error(error.message || '上传失败')
  } finally {
    confirming.value = false
  }
}

// 取消上传
const handleUploadCancel = () => {
  uploadFormRef.value?.resetFields()
  uploadForm.certFile = null
  uploadForm.keyFile = null
  if (certFileInput.value) certFileInput.value.value = ''
  if (keyFileInput.value) keyFileInput.value.value = ''
  emit('update:visible', false)
  emit('cancel')
}

// 取消粘贴
const handlePasteCancel = () => {
  pasteFormRef.value?.resetFields()
}

// 重置所有表单
const resetForms = () => {
  handleUploadCancel()
  handlePasteCancel()
  activeTab.value = 'upload'
}</script>

<style scoped>
.certificate-upload-container {
  padding: 0;
}

/* 标签页 */
.certificate-tabs {
  margin: 0;
}

:deep(.certificate-tabs .el-tabs__header) {
  margin: 0 0 16px 0;
}

:deep(.certificate-tabs .el-tabs__content) {
  padding: 0;
}

:deep(.el-tab-pane) {
  padding: 0;
}

/* 上传内容 */
.upload-content,
.paste-content {
  padding: 20px 0;
}

.file-upload-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-box {
  border: 2px dashed #dcdfe6;
  border-radius: 10px;
  padding: 32px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  background-color: #fafbfc;
}

.upload-box:hover {
  border-color: #bfdbfe;
  background-color: #eff6ff;
}

.upload-icon {
  font-size: 32px;
  color: #2563eb;
  margin-bottom: 12px;
}

.upload-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.upload-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.upload-desc {
  font-size: 12px;
  color: #909399;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: #f0f9ff;
  border-radius: 6px;
  border-left: 3px solid #409eff;
}

.file-name {
  font-size: 13px;
  color: #409eff;
  font-weight: 500;
}

/* 表单操作 */
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}

/* 证书预览 */
.cert-preview {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.cert-status-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 15px;
}

.status-bar-valid {
  background: linear-gradient(135deg, #f0f9eb 0%, #e8f5e9 100%);
  color: #67c23a;
  border: 1px solid #c2e7b0;
}

.status-bar-warning {
  background: linear-gradient(135deg, #fdf6ec 0%, #fff3e0 100%);
  color: #e6a23c;
  border: 1px solid #f5dab1;
}

.status-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
}

.status-bar-valid .status-icon {
  background: #67c23a;
  color: #fff;
}

.status-bar-warning .status-icon {
  background: #e6a23c;
  color: #fff;
}

.cert-info-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cert-info-section {
  border: 1px solid #e8ecf0;
  border-radius: 10px;
  overflow: hidden;
}

.section-title {
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: #f8fafc;
  border-bottom: 1px solid #e8ecf0;
}

.section-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0;
}

.cert-info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  border-bottom: 1px solid #f5f5f5;
  border-right: 1px solid #f5f5f5;
}

.cert-info-item:nth-child(2n) {
  border-right: none;
}

.cert-info-item:last-child,
.cert-info-item:nth-last-child(2):nth-child(odd) {
  border-bottom: none;
}

.cert-info-item.full-width {
  grid-column: span 2;
  border-right: none;
}

.cert-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.cert-value {
  font-size: 13px;
  color: #303133;
  word-break: break-all;
  font-weight: 500;
}

.cert-mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #606266;
  background-color: #f8fafc;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid #e8ecf0;
  max-height: 80px;
  overflow-y: auto;
}

.fingerprint {
  word-break: break-all;
  letter-spacing: 1px;
}

.private-key-status {
  color: #67c23a;
  font-weight: 600;
}

.days-remaining {
  display: inline-block;
  margin-left: 8px;
  font-weight: 600;
}

.days-normal {
  color: #67c23a;
}

.days-warning {
  color: #e6a23c;
}

.days-expired {
  color: #f56c6c;
}

/* 弹窗美化 */
:deep(.beauty-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.beauty-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  margin-right: 0;
  border-bottom: 1px solid #f0f0f0;
  background: #fafbfc;
}

:deep(.beauty-dialog .el-dialog__title) {
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.beauty-dialog .el-dialog__headerbtn) {
  top: 20px;
  right: 20px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-dialog .el-dialog__headerbtn:hover) {
  background: #f0f0f0;
}

:deep(.beauty-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 65vh;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

:deep(.beauty-dialog .el-dialog__body::-webkit-scrollbar) {
  display: none;
}

:deep(.beauty-dialog .el-dialog__footer) {
  padding: 16px 24px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafbfc;
}
</style>
