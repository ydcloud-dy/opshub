<template>
  <div class="system-config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Setting /></el-icon>
        </div>
        <div>
          <h2 class="page-title">系统配置</h2>
          <p class="page-subtitle">管理系统基础配置、安全设置与通知服务</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleSave" :loading="saving">
          <el-icon style="margin-right: 6px;"><Check /></el-icon>
          保存配置
        </el-button>
      </div>
    </div>

    <!-- 配置内容 -->
    <div class="config-content">
      <!-- 左侧导航 -->
      <div class="config-nav">
        <div class="nav-header">配置分类</div>
        <div class="nav-list">
          <div
            v-for="(item, index) in navItems"
            :key="index"
            :class="['nav-item', { active: activeNav === index }]"
            @click="activeNav = index"
          >
            <el-icon class="nav-icon"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </div>
        </div>
      </div>

      <!-- 右侧配置表单 -->
      <div class="config-form-wrapper">
        <!-- 基础配置 -->
        <div v-show="activeNav === 0" class="config-section">
          <div class="section-header">
            <el-icon class="section-icon"><HomeFilled /></el-icon>
            <span>基础配置</span>
          </div>
          <el-form :model="config" label-width="140px" class="config-form">
            <el-form-item label="系统名称">
              <el-input v-model="config.systemName" placeholder="请输入系统名称">
                <template #prefix>
                  <el-icon><Edit /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="系统Logo">
              <el-input v-model="config.systemLogo" placeholder="请输入系统Logo地址">
                <template #prefix>
                  <el-icon><Picture /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="系统描述">
              <el-input
                v-model="config.systemDescription"
                type="textarea"
                :rows="3"
                placeholder="请输入系统描述"
              />
            </el-form-item>
            <el-form-item label="版权信息">
              <el-input v-model="config.copyright" placeholder="请输入版权信息">
                <template #prefix>
                  <el-icon><Document /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-form>
        </div>

        <!-- 安全配置 -->
        <div v-show="activeNav === 1" class="config-section">
          <div class="section-header">
            <el-icon class="section-icon"><Lock /></el-icon>
            <span>安全配置</span>
          </div>
          <el-form :model="config" label-width="140px" class="config-form">
            <el-form-item label="密码最小长度">
              <el-input-number v-model="config.passwordMinLength" :min="6" :max="20" />
              <span class="form-tip">建议设置 8 位以上</span>
            </el-form-item>
            <el-form-item label="Session超时">
              <el-input-number v-model="config.sessionTimeout" :min="300" :step="300" />
              <span class="form-tip">单位：秒</span>
            </el-form-item>
            <el-form-item label="开启验证码">
              <el-switch
                v-model="config.enableCaptcha"
                active-text="开启"
                inactive-text="关闭"
              />
            </el-form-item>
            <el-form-item label="最大登录失败">
              <el-input-number v-model="config.maxLoginAttempts" :min="3" :max="10" />
              <span class="form-tip">超过次数将锁定账户</span>
            </el-form-item>
            <el-form-item label="账户锁定时间">
              <el-input-number v-model="config.lockoutDuration" :min="60" :step="60" />
              <span class="form-tip">单位：秒</span>
            </el-form-item>
          </el-form>
        </div>

        <!-- 邮件配置 -->
        <div v-show="activeNav === 2" class="config-section">
          <div class="section-header">
            <el-icon class="section-icon"><Message /></el-icon>
            <span>邮件配置</span>
          </div>
          <el-form :model="config" label-width="140px" class="config-form">
            <el-form-item label="启用邮件通知">
              <el-switch
                v-model="config.enableEmailNotification"
                active-text="开启"
                inactive-text="关闭"
              />
            </el-form-item>
            <el-form-item label="SMTP服务器">
              <el-input v-model="config.smtpHost" placeholder="smtp.example.com">
                <template #prefix>
                  <el-icon><Link /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="SMTP端口">
              <el-input-number v-model="config.smtpPort" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="SMTP用户名">
              <el-input v-model="config.smtpUsername" placeholder="请输入SMTP用户名">
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="SMTP密码">
              <el-input v-model="config.smtpPassword" type="password" show-password placeholder="请输入SMTP密码">
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="发件人邮箱">
              <el-input v-model="config.smtpFrom" placeholder="noreply@example.com">
                <template #prefix>
                  <el-icon><Message /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="发件人名称">
              <el-input v-model="config.smtpFromName" placeholder="OpsHub System">
                <template #prefix>
                  <el-icon><UserFilled /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item>
              <el-button @click="handleTestEmail" :loading="testingEmail">
                <el-icon style="margin-right: 4px;"><Promotion /></el-icon>
                测试邮件
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 其他配置 -->
        <div v-show="activeNav === 3" class="config-section">
          <div class="section-header">
            <el-icon class="section-icon"><Tools /></el-icon>
            <span>其他配置</span>
          </div>
          <el-form :model="config" label-width="140px" class="config-form">
            <el-form-item label="开启用户注册">
              <el-switch
                v-model="config.enableRegister"
                active-text="开启"
                inactive-text="关闭"
              />
            </el-form-item>
            <el-form-item label="默认用户角色">
              <el-select v-model="config.defaultUserRole" placeholder="请选择默认角色" style="width: 200px;">
                <el-option label="普通用户" value="user" />
                <el-option label="访客" value="guest" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志保留天数">
              <el-input-number v-model="config.logRetentionDays" :min="7" :max="365" />
              <span class="form-tip">超过天数的日志将被自动清理</span>
            </el-form-item>
            <el-form-item label="文件上传限制">
              <el-input-number v-model="config.maxUploadSize" :min="1" :max="100" />
              <span class="form-tip">单位：MB</span>
            </el-form-item>
            <el-form-item label="允许上传类型">
              <el-input v-model="config.allowedFileTypes" placeholder=".jpg,.png,.pdf,.doc,.docx">
                <template #prefix>
                  <el-icon><Files /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Setting, Check, HomeFilled, Lock, Message, Tools,
  Edit, Picture, Document, Link, User, UserFilled,
  Promotion, Files
} from '@element-plus/icons-vue'

const saving = ref(false)
const testingEmail = ref(false)
const activeNav = ref(0)

const navItems = [
  { label: '基础配置', icon: 'HomeFilled' },
  { label: '安全配置', icon: 'Lock' },
  { label: '邮件配置', icon: 'Message' },
  { label: '其他配置', icon: 'Tools' }
]

const config = reactive({
  // 基础配置
  systemName: 'OpsHub',
  systemLogo: '',
  systemDescription: '运维管理平台',
  copyright: '© 2025 OpsHub. All rights reserved.',

  // 安全配置
  passwordMinLength: 8,
  sessionTimeout: 3600,
  enableCaptcha: true,
  maxLoginAttempts: 5,
  lockoutDuration: 300,

  // 邮件配置
  enableEmailNotification: false,
  smtpHost: '',
  smtpPort: 587,
  smtpUsername: '',
  smtpPassword: '',
  smtpFrom: '',
  smtpFromName: 'OpsHub System',

  // 其他配置
  enableRegister: false,
  defaultUserRole: 'user',
  logRetentionDays: 30,
  maxUploadSize: 10,
  allowedFileTypes: '.jpg,.png,.gif,.pdf,.doc,.docx,.xls,.xlsx'
})

const loadConfig = async () => {
  try {
    // TODO: 从后端加载配置
    const savedConfig = localStorage.getItem('system_config')
    if (savedConfig) {
      Object.assign(config, JSON.parse(savedConfig))
    }
  } catch (error) {
    console.error('加载配置失败', error)
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    // TODO: 调用后端API保存配置
    localStorage.setItem('system_config', JSON.stringify(config))
    await new Promise(resolve => setTimeout(resolve, 500))
    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleTestEmail = async () => {
  if (!config.smtpHost || !config.smtpFrom) {
    ElMessage.warning('请先配置SMTP服务器和发件人邮箱')
    return
  }

  testingEmail.value = true
  try {
    // TODO: 调用后端API发送测试邮件
    await new Promise(resolve => setTimeout(resolve, 1500))
    ElMessage.success('测试邮件发送成功，请检查收件箱')
  } catch (error) {
    ElMessage.error('测试邮件发送失败')
  } finally {
    testingEmail.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.system-config-container {
  padding: 0;
  background-color: transparent;
  min-height: 100%;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 20px 24px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 52px;
  height: 52px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 24px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 6px 0 0 0;
  font-size: 14px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 配置内容区域 */
.config-content {
  display: flex;
  gap: 16px;
  min-height: calc(100vh - 220px);
}

/* 左侧导航 */
.config-nav {
  width: 200px;
  min-width: 200px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.nav-header {
  padding: 16px 20px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-bottom: 1px solid #ebeef5;
}

.nav-list {
  padding: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 4px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #606266;
  font-size: 14px;
}

.nav-item:hover {
  background: #f5f7fa;
  color: #303133;
}

.nav-item.active {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  color: #d4af37;
  font-weight: 500;
}

.nav-item.active .nav-icon {
  color: #d4af37;
}

.nav-icon {
  font-size: 18px;
  color: #909399;
  transition: color 0.2s ease;
}

.nav-item:hover .nav-icon {
  color: #606266;
}

/* 右侧配置表单 */
.config-form-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.config-section {
  padding: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.section-icon {
  font-size: 22px;
  color: #d4af37;
}

.config-form {
  max-width: 600px;
}

.config-form :deep(.el-form-item) {
  margin-bottom: 24px;
}

.config-form :deep(.el-form-item__label) {
  color: #606266;
  font-weight: 500;
}

.config-form :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px #dcdfe6 inset;
  transition: all 0.2s ease;
}

.config-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #d4af37 inset;
}

.config-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #d4af37 inset, 0 0 8px rgba(212, 175, 55, 0.2);
}

.config-form :deep(.el-textarea__inner) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

.config-form :deep(.el-textarea__inner:hover) {
  border-color: #d4af37;
}

.config-form :deep(.el-textarea__inner:focus) {
  border-color: #d4af37;
  box-shadow: 0 0 8px rgba(212, 175, 55, 0.2);
}

.config-form :deep(.el-input__prefix) {
  color: #d4af37;
}

.config-form :deep(.el-input-number) {
  width: 160px;
}

.config-form :deep(.el-switch.is-checked .el-switch__core) {
  background-color: #000;
  border-color: #000;
}

.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #909399;
}

/* 黑色按钮样式 */
.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 12px 24px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.black-button:hover {
  background-color: #1a1a1a !important;
  border-color: #1a1a1a !important;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.black-button:active {
  transform: translateY(0);
}

/* 响应式布局 */
@media (max-width: 768px) {
  .config-content {
    flex-direction: column;
  }

  .config-nav {
    width: 100%;
    min-width: auto;
  }

  .nav-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .nav-item {
    flex: 1;
    min-width: 120px;
    justify-content: center;
    margin-bottom: 0;
  }

  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .black-button {
    width: 100%;
  }
}
</style>
