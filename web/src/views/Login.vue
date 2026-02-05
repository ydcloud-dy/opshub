<template>
  <div class="login-container">
    <!-- 左侧：品牌展示区 -->
    <div class="brand-section">
      <div class="curved-divider"></div>
      <div class="brand-content">
        <h1 class="brand-title">{{ systemStore.systemName }} {{ systemStore.systemDescription }}</h1>
        <div class="brand-slogan">
          <span>高效</span>
          <span>安全</span>
          <span>便捷</span>
        </div>
        <p class="brand-subtitle">一键通达所有应用</p>
        <div class="brand-illustration">
          <svg viewBox="0 0 400 300" class="illustration-svg">
            <!-- 平台基座 -->
            <rect x="100" y="200" width="200" height="20" rx="4" fill="url(#goldGradient)" opacity="0.4"/>
            <rect x="120" y="180" width="160" height="20" rx="4" fill="url(#goldGradient)" opacity="0.6"/>
            <rect x="140" y="160" width="120" height="20" rx="4" fill="url(#goldGradient)" opacity="0.8"/>

            <!-- 服务器/中心节点 -->
            <rect x="180" y="100" width="40" height="60" rx="4" fill="url(#goldGradient)" opacity="0.9"/>
            <circle cx="200" cy="90" r="15" fill="url(#goldGradient)" opacity="0.8"/>

            <!-- 连接线 -->
            <line x1="200" y1="75" x2="150" y2="50" stroke="url(#goldGradient)" stroke-width="2" opacity="0.7"/>
            <line x1="200" y1="75" x2="250" y2="50" stroke="url(#goldGradient)" stroke-width="2" opacity="0.7"/>
            <line x1="200" y1="75" x2="200" y2="40" stroke="url(#goldGradient)" stroke-width="2" opacity="0.7"/>

            <!-- 小图标节点 -->
            <circle cx="150" cy="50" r="8" fill="url(#goldGradient)" opacity="0.9"/>
            <circle cx="250" cy="50" r="8" fill="url(#goldGradient)" opacity="0.9"/>
            <circle cx="200" cy="40" r="8" fill="url(#goldGradient)" opacity="0.9"/>

            <!-- 装饰元素 -->
            <rect x="80" y="220" width="30" height="30" rx="4" fill="url(#goldGradient)" opacity="0.3"/>
            <rect x="290" y="220" width="30" height="30" rx="4" fill="url(#goldGradient)" opacity="0.3"/>

            <!-- 定义渐变 -->
            <defs>
              <linearGradient id="goldGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#D4AF37;stop-opacity:1" />
                <stop offset="50%" style="stop-color:#FFD700;stop-opacity:1" />
                <stop offset="100%" style="stop-color:#FFA500;stop-opacity:1" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>
    </div>

    <!-- 右侧：登录表单区 -->
    <div class="login-section">
      <div class="login-wrapper">
        <div class="login-header">
          <h2>用户登录</h2>
          <div class="header-line"></div>
        </div>

        <el-form :model="loginForm" :rules="rules" ref="formRef" class="login-form" size="large">
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              show-password
              :prefix-icon="Lock"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-form-item prop="captchaCode" v-if="captchaEnabled">
            <div class="captcha-wrapper">
              <el-input
                v-model="loginForm.captchaCode"
                placeholder="请输入验证码"
                :prefix-icon="Key"
                @keyup.enter="handleLogin"
              />
              <div class="captcha-image" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                <span v-else class="captcha-loading">加载中...</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item>
            <div class="form-options">
              <el-checkbox v-model="loginForm.remember">记住登录名</el-checkbox>
            </div>
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              @click="handleLogin"
              :loading="loading"
              class="login-button"
            >
              登录
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance } from 'element-plus'
import { User, Lock, Key } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { useSystemStore } from '@/stores/system'
import request from '@/utils/request'
import { getPublicConfig } from '@/api/system'

const router = useRouter()
const userStore = useUserStore()
const systemStore = useSystemStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const captchaImage = ref('')
const captchaId = ref('')
const captchaEnabled = ref(true) // 默认开启验证码

const loginForm = reactive({
  username: '',
  password: '',
  captchaCode: '',
  captchaId: '',
  remember: false
})

// 动态验证规则
const rules = computed(() => ({
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  captchaCode: captchaEnabled.value
    ? [{ required: true, message: '请输入验证码', trigger: 'blur' }]
    : []
}))

// 加载公开配置
const loadPublicConfig = async () => {
  try {
    const res = await getPublicConfig()
    if (res) {
      captchaEnabled.value = res.enableCaptcha !== false
      // 更新系统配置store（用于显示系统名称、Logo、更新页面标题和favicon）
      systemStore.updateConfig({
        systemName: res.systemName,
        systemLogo: res.systemLogo,
        systemDescription: res.systemDescription
      })
    }
  } catch (error) {
    console.error('加载公开配置失败', error)
    // 默认开启验证码
    captchaEnabled.value = true
  }
}

// 获取验证码
const refreshCaptcha = async () => {
  if (!captchaEnabled.value) return

  try {
    captchaImage.value = ''
    const res: any = await request.get('/api/v1/captcha')
    captchaImage.value = res.image
    captchaId.value = res.captchaId
    loginForm.captchaId = res.captchaId
  } catch (error) {
    ElMessage.error('获取验证码失败')
  }
}

const handleLogin = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        await userStore.login({
          username: loginForm.username,
          password: loginForm.password,
          captchaId: loginForm.captchaId,
          captchaCode: loginForm.captchaCode
        })

        // 如果选择了记住登录名，保存到本地
        if (loginForm.remember) {
          localStorage.setItem('rememberedUsername', loginForm.username)
        } else {
          localStorage.removeItem('rememberedUsername')
        }

        ElMessage.success('登录成功')
        await router.push('/')
      } catch (error: any) {

        // 提取错误消息 - 支持多种错误对象格式
        let errorMessage = '登录失败'
        if (error) {
          // 优先使用message字段
          if (error.message && typeof error.message === 'string' && error.message !== '400') {
            errorMessage = error.message
          }
          // 其次尝试response.data.message
          else if (error.response && error.response.data && error.response.data.message) {
            errorMessage = error.response.data.message
          }
          // 如果message字段是"400"等状态码字符串，尝试其他字段
          else if (error.response && error.response.data && error.response.data.data) {
            errorMessage = error.response.data.data
          }
        }

        ElMessage.error(errorMessage)
        // 登录失败后刷新验证码
        refreshCaptcha()
        loginForm.captchaCode = ''
      } finally {
        loading.value = false
      }
    }
  })
}

onMounted(async () => {
  // 加载记住的用户名
  const rememberedUsername = localStorage.getItem('rememberedUsername')
  if (rememberedUsername) {
    loginForm.username = rememberedUsername
    loginForm.remember = true
  }

  // 加载公开配置
  await loadPublicConfig()

  // 如果开启了验证码，加载验证码
  if (captchaEnabled.value) {
    refreshCaptcha()
  }
})
</script>

<style scoped>
.login-container {
  display: flex;
  min-height: 100vh;
  background: #ffffff;
  position: relative;
  overflow: hidden;
}

/* 左侧品牌区 - 黑白风格 */
.brand-section {
  flex: 0 0 62%;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 50%, #1a1a1a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

/* 微妙的灰色渐变背景 */
.brand-section::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(circle at 20% 30%, rgba(255, 255, 255, 0.03) 0%, transparent 50%),
    radial-gradient(circle at 80% 70%, rgba(255, 255, 255, 0.02) 0%, transparent 50%),
    radial-gradient(circle at 50% 50%, rgba(255, 255, 255, 0.015) 0%, transparent 60%);
  animation: shimmer 15s ease-in-out infinite;
}

@keyframes shimmer {
  0%, 100% { opacity: 0.8; }
  50% { opacity: 1; }
}

/* 曲线分割效果 - 金色保留 */
.curved-divider {
  position: absolute;
  right: -15%;
  top: 0;
  width: 30%;
  height: 100%;
  background: linear-gradient(135deg, rgba(212, 175, 55, 0.25) 0%, rgba(255, 215, 0, 0.15) 50%, rgba(255, 165, 0, 0.2) 100%);
  clip-path: polygon(
    30% 0%,
    70% 0%,
    100% 5%,
    100% 95%,
    70% 100%,
    30% 100%,
    0% 95%,
    0% 5%
  );
  box-shadow: -10px 0 30px rgba(212, 175, 55, 0.35);
}

.brand-section::after {
  content: '';
  position: absolute;
  top: -50%;
  right: -20%;
  width: 80%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.02) 0%, transparent 70%);
  animation: float 20s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-20px) rotate(5deg); }
}

.brand-content {
  text-align: center;
  color: #ffffff;
  z-index: 1;
  padding: 60px;
}

.brand-title {
  font-size: 52px;
  font-weight: 700;
  margin-bottom: 40px;
  letter-spacing: 3px;
  color: #ffffff;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
}

.brand-slogan {
  display: flex;
  justify-content: center;
  gap: 40px;
  margin-bottom: 30px;
  font-size: 32px;
  font-weight: 600;
}

.brand-slogan span {
  padding: 12px 28px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #ffffff;
}

.brand-subtitle {
  font-size: 20px;
  opacity: 0.9;
  margin-bottom: 80px;
  letter-spacing: 2px;
  font-weight: 300;
  color: #cccccc;
}

.brand-illustration {
  max-width: 450px;
  margin: 0 auto;
}

.illustration-svg {
  width: 100%;
  height: auto;
  filter: drop-shadow(0 15px 30px rgba(0, 0, 0, 0.3));
}

/* 右侧登录区 - 白色背景 */
.login-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  background: #ffffff;
  padding: 60px 80px;
  position: relative;
}

/* 金色边框装饰保留 */
.login-section::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10%;
  width: 2px;
  height: 80%;
  background: linear-gradient(180deg, transparent 0%, rgba(212, 175, 55, 0.6) 50%, transparent 100%);
}

.login-wrapper {
  width: 100%;
  max-width: 450px;
}

.login-header {
  margin-bottom: 50px;
}

.login-header h2 {
  font-size: 32px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 20px;
}

.header-line {
  width: 70px;
  height: 4px;
  background: linear-gradient(90deg, #D4AF37, #FFD700, #FFA500);
  border-radius: 2px;
  box-shadow: 0 0 10px rgba(212, 175, 55, 0.4);
}

.login-form {
  margin-top: 40px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 32px;
}

/* 黑白风格输入框 - 金色图标保留 */
.login-form :deep(.el-input__wrapper) {
  padding: 14px 18px;
  border-radius: 10px;
  background: #ffffff;
  box-shadow: 0 0 0 1px #e0e0e0 inset;
  transition: all 0.3s;
}

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #cccccc inset;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #D4AF37 inset, 0 0 15px rgba(212, 175, 55, 0.2);
  background: #fafafa;
}

.login-form :deep(.el-input__inner) {
  font-size: 16px;
  color: #1a1a1a;
}

.login-form :deep(.el-input__inner)::placeholder {
  color: #999999;
}

.login-form :deep(.el-input__prefix-inner) {
  color: #D4AF37;
}

/* 验证码样式 */
.captcha-wrapper {
  display: flex;
  gap: 14px;
  width: 100%;
}

.captcha-wrapper :deep(.el-input) {
  flex: 1;
}

.captcha-image {
  flex-shrink: 0;
  width: 140px;
  height: 48px;
  border: 1px solid #e0e0e0;
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  transition: all 0.3s;
}

.captcha-image:hover {
  border-color: #D4AF37;
  background: #ffffff;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.25);
}

.captcha-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-loading {
  font-size: 13px;
  color: #999999;
}

/* 表单选项 */
.form-options {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
}

.form-options :deep(.el-checkbox__label) {
  font-size: 15px;
  color: #666666;
}

.form-options :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #D4AF37;
  border-color: #D4AF37;
}

.form-options :deep(.el-checkbox__inner) {
  border-color: #d0d0d0;
}

/* 登录按钮 - 黑白风格，金色装饰 */
.login-button {
  width: 100%;
  height: 52px;
  font-size: 17px;
  font-weight: 500;
  border-radius: 10px;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border: 2px solid #D4AF37;
  color: #D4AF37;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 0 0 0 rgba(212, 175, 55, 0);
  transition: all 0.3s;
}

.login-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15), 0 0 20px rgba(212, 175, 55, 0.3);
  background: linear-gradient(135deg, #D4AF37 0%, #FFD700 100%);
  color: #1a1a1a;
  border-color: #FFD700;
}

.login-button:active {
  transform: translateY(0);
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .brand-section {
    flex: 0 0 55%;
  }

  .brand-title {
    font-size: 44px;
  }

  .brand-slogan {
    font-size: 28px;
    gap: 30px;
  }
}

@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
  }

  .brand-section {
    flex: none;
    min-height: 45vh;
  }

  .curved-divider {
    display: none;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-slogan {
    font-size: 20px;
    gap: 20px;
  }

  .brand-slogan span {
    padding: 8px 16px;
  }

  .brand-illustration {
    max-width: 280px;
  }

  .login-section {
    padding: 30px 40px;
    align-items: center;
    justify-content: center;
  }

  .login-wrapper {
    padding: 20px;
  }
}
</style>
