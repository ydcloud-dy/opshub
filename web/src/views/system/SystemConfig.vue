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
          <p class="page-subtitle">管理系统基础配置、安全设置</p>
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
              <div class="logo-upload-container">
                <div class="logo-preview" v-if="config.systemLogo">
                  <img :src="config.systemLogo" alt="Logo预览" />
                  <div class="logo-actions">
                    <el-button type="danger" size="small" @click="removeLogo">
                      <el-icon><Delete /></el-icon>
                      删除
                    </el-button>
                  </div>
                </div>
                <el-upload
                  v-else
                  class="logo-uploader"
                  :show-file-list="false"
                  :before-upload="beforeLogoUpload"
                  :http-request="handleLogoUpload"
                  accept=".png,.jpg,.jpeg,.ico,.svg"
                >
                  <div class="upload-trigger">
                    <el-icon class="upload-icon"><Plus /></el-icon>
                    <span class="upload-text">上传Logo</span>
                  </div>
                </el-upload>
                <div class="upload-tip">支持 png/jpg/jpeg/ico/svg 格式，大小不超过 2MB</div>
              </div>
            </el-form-item>
            <el-form-item label="系统描述">
              <el-input
                v-model="config.systemDescription"
                type="textarea"
                :rows="3"
                placeholder="请输入系统描述"
              />
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

            <el-divider content-position="left">
              <el-icon class="divider-icon"><Key /></el-icon>
              MFA 多因素认证
            </el-divider>

            <el-form-item label="启用 MFA">
              <el-switch
                v-model="config.mfaEnabled"
                active-text="开启"
                inactive-text="关闭"
              />
              <span class="form-tip">启用后用户可绑定 MFA 设备</span>
            </el-form-item>
            <el-form-item label="强制 MFA" v-if="config.mfaEnabled">
              <el-switch
                v-model="config.mfaEnforced"
                active-text="开启"
                inactive-text="关闭"
              />
              <span class="form-tip">开启后所有用户必须绑定 MFA 才能登录</span>
            </el-form-item>
            <el-form-item label="MFA 类型" v-if="config.mfaEnabled">
              <el-radio-group v-model="config.mfaType">
                <el-radio value="totp">TOTP（验证器应用）</el-radio>
              </el-radio-group>
              <span class="form-tip">目前支持 Google/Microsoft Authenticator 等验证器应用</span>
            </el-form-item>
            <el-form-item label="记住设备时长" v-if="config.mfaEnabled">
              <el-select v-model="config.mfaSkipDuration" style="width: 200px;">
                <el-option :value="0" label="每次登录都需要验证" />
                <el-option :value="86400" label="1 天" />
                <el-option :value="604800" label="7 天" />
                <el-option :value="1209600" label="14 天" />
                <el-option :value="2592000" label="30 天" />
              </el-select>
              <span class="form-tip">同一设备在此时间内无需再次验证 MFA</span>
            </el-form-item>
          </el-form>
        </div>

        <!-- LDAP 配置 -->
        <div v-show="activeNav === 2" class="config-section">
          <div class="section-header">
            <el-icon class="section-icon"><Connection /></el-icon>
            <span>LDAP 配置</span>
          </div>

          <el-form :model="ldapConfig" label-width="160px" class="config-form">
            <!-- 启用开关 -->
            <el-form-item label="启用 LDAP 认证">
              <el-switch
                v-model="ldapConfig.enabled"
                active-text="启用"
                inactive-text="关闭"
              />
              <span class="form-tip">启用后支持使用 LDAP 账号登录系统</span>
            </el-form-item>

            <el-divider content-position="left">服务器设置</el-divider>

            <el-form-item label="服务器地址">
              <el-input v-model="ldapConfig.host" placeholder="如: ldap.example.com 或 192.168.1.100" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="ldapConfig.port" :min="1" :max="65535" />
              <span class="form-tip">LDAP 默认 389，LDAPS 默认 636</span>
            </el-form-item>
            <el-form-item label="连接方式">
              <el-radio-group v-model="ldapConnectionMode" @change="handleConnectionModeChange">
                <el-radio value="plain">LDAP（明文）</el-radio>
                <el-radio value="ldaps">LDAPS（SSL/TLS）</el-radio>
                <el-radio value="starttls">StartTLS</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="跳过证书验证" v-if="ldapConfig.useTls || ldapConfig.startTls">
              <el-switch v-model="ldapConfig.skipVerify" />
              <span class="form-tip">测试环境可开启，生产环境建议关闭</span>
            </el-form-item>
            <el-form-item>
              <el-button
                class="ldap-test-button"
                @click="handleTestLDAP"
                :loading="ldapTesting"
                type="primary"
              >
                测试连接
              </el-button>
              <div v-if="ldapTestResult" :class="['ldap-test-result', ldapTestResult.success ? 'success' : 'error']">
                <el-icon v-if="ldapTestResult.success"><CircleCheckFilled /></el-icon>
                <el-icon v-else><CircleCloseFilled /></el-icon>
                <span>{{ ldapTestResult.message }}</span>
                <span v-if="ldapTestResult.success && ldapTestResult.userCount !== undefined">
                  （发现 {{ ldapTestResult.userCount }} 个用户）
                </span>
              </div>
            </el-form-item>

            <el-divider content-position="left">认证设置</el-divider>

            <el-form-item label="Bind DN">
              <el-input v-model="ldapConfig.bindDn" placeholder="如: cn=admin,dc=example,dc=com" />
              <span class="form-tip">管理员 DN，用于搜索用户</span>
            </el-form-item>
            <el-form-item label="Bind 密码">
              <el-input v-model="ldapConfig.bindPassword" type="password" show-password placeholder="Bind DN 对应的密码" />
            </el-form-item>
            <el-form-item label="Base DN">
              <el-input v-model="ldapConfig.baseDn" placeholder="如: dc=example,dc=com" />
              <span class="form-tip">用户搜索的根节点</span>
            </el-form-item>
            <el-form-item label="用户搜索过滤器">
              <el-input v-model="ldapConfig.userFilter" placeholder="(uid=%s)" />
              <span class="form-tip">%s 会被替换为用户名。OpenLDAP 用 (uid=%s)，AD 用 (sAMAccountName=%s)</span>
            </el-form-item>

            <el-divider content-position="left">属性映射</el-divider>

            <el-form-item label="用户名属性">
              <el-input v-model="ldapConfig.attrUsername" placeholder="uid" />
              <span class="form-tip">OpenLDAP: uid | AD: sAMAccountName</span>
            </el-form-item>
            <el-form-item label="邮箱属性">
              <el-input v-model="ldapConfig.attrEmail" placeholder="mail" />
            </el-form-item>
            <el-form-item label="姓名属性">
              <el-input v-model="ldapConfig.attrRealName" placeholder="cn" />
              <span class="form-tip">OpenLDAP: cn | AD: displayName</span>
            </el-form-item>
            <el-form-item label="电话属性">
              <el-input v-model="ldapConfig.attrPhone" placeholder="telephoneNumber" />
            </el-form-item>

            <el-divider content-position="left">用户设置</el-divider>

            <el-form-item label="自动创建用户">
              <el-switch v-model="ldapConfig.autoCreateUser" />
              <span class="form-tip">LDAP 用户首次登录时自动在系统中创建本地账号</span>
            </el-form-item>
            <el-form-item label="默认角色">
              <el-select v-model="ldapConfig.defaultRoleId" placeholder="请选择" clearable>
                <el-option
                  v-for="role in roles"
                  :key="role.id"
                  :label="role.name"
                  :value="role.id"
                />
              </el-select>
              <span class="form-tip">LDAP 用户自动创建时分配的默认角色</span>
            </el-form-item>
            <el-form-item label="默认部门">
              <el-select v-model="ldapConfig.defaultDeptId" placeholder="请选择" clearable>
                <el-option
                  v-for="dept in departments"
                  :key="dept.id"
                  :label="dept.name"
                  :value="dept.id"
                />
              </el-select>
              <span class="form-tip">LDAP 用户自动创建时分配的默认部门</span>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Setting, Check, HomeFilled, Lock,
  Edit, Plus, Delete, Connection,
  CircleCheckFilled, CircleCloseFilled, Key
} from '@element-plus/icons-vue'
import request from '@/utils/request'
import {
  getAllConfig,
  saveBasicConfig,
  saveSecurityConfig,
  uploadLogo,
  getLDAPConfig,
  saveLDAPConfig,
  testLDAPConnection,
  type LDAPConfig
} from '@/api/system'
import { useSystemStore } from '@/stores/system'

const systemStore = useSystemStore()
const saving = ref(false)
const activeNav = ref(0)

const navItems = [
  { label: '基础配置', icon: 'HomeFilled' },
  { label: '安全配置', icon: 'Lock' },
  { label: 'LDAP 配置', icon: 'Connection' }
]

const config = reactive({
  // 基础配置
  systemName: 'OpsHub',
  systemLogo: '',
  systemDescription: '运维管理平台',

  // 安全配置
  passwordMinLength: 8,
  sessionTimeout: 3600,
  enableCaptcha: true,
  maxLoginAttempts: 5,
  lockoutDuration: 300,

  // MFA配置
  mfaEnabled: false,
  mfaEnforced: false,
  mfaType: 'totp',
  mfaSkipDuration: 2592000
})

// LDAP 配置
const ldapConfig = reactive<LDAPConfig>({
  enabled: false,
  host: '',
  port: 389,
  useTls: false,
  startTls: false,
  skipVerify: false,
  bindDn: '',
  bindPassword: '',
  baseDn: '',
  userFilter: '(uid=%s)',
  attrUsername: 'uid',
  attrEmail: 'mail',
  attrRealName: 'cn',
  attrPhone: 'telephoneNumber',
  defaultRoleId: 0,
  defaultDeptId: 0,
  autoCreateUser: true
})

// LDAP 连接方式
const ldapConnectionMode = computed(() => {
  if (ldapConfig.useTls) return 'ldaps'
  if (ldapConfig.startTls) return 'starttls'
  return 'plain'
})

const handleConnectionModeChange = (mode: string) => {
  ldapConfig.useTls = mode === 'ldaps'
  ldapConfig.startTls = mode === 'starttls'
  // 自动调整端口
  if (mode === 'ldaps' && ldapConfig.port === 389) {
    ldapConfig.port = 636
  } else if (mode !== 'ldaps' && ldapConfig.port === 636) {
    ldapConfig.port = 389
  }
}

const ldapTesting = ref(false)
const ldapTestResult = ref<{ success: boolean; message: string; userCount?: number } | null>(null)
const roles = ref<{ id: number; name: string }[]>([])
const departments = ref<{ id: number; name: string }[]>([])

// 加载角色和部门列表（用于LDAP默认分配）
const loadRolesAndDepts = async () => {
  try {
    const [rolesRes, deptsRes]: any[] = await Promise.all([
      request.get('/api/v1/roles/all'),
      request.get('/api/v1/departments/tree')
    ])
    if (Array.isArray(rolesRes)) {
      roles.value = rolesRes.map((r: any) => ({ id: r.ID || r.id, name: r.name || r.Name }))
    }
    if (Array.isArray(deptsRes)) {
      // 扁平化部门树（部门VO字段: id, deptName, children）
      const flatDepts: { id: number; name: string }[] = []
      const flatten = (items: any[], prefix = '') => {
        for (const item of items) {
          const name = item.deptName || item.name || ''
          const id = item.id || item.ID
          if (id) {
            const displayName = prefix ? prefix + ' / ' + name : name
            flatDepts.push({ id, name: displayName || `部门${id}` })
          }
          if (item.children && item.children.length > 0) {
            flatten(item.children, prefix ? prefix + ' / ' + name : name)
          }
        }
      }
      flatten(deptsRes)
      departments.value = flatDepts
    }
  } catch (error) {
    // 静默处理
  }
}

// 加载LDAP配置
const loadLDAPConfig = async () => {
  try {
    const res: any = await getLDAPConfig()
    if (res) {
      Object.assign(ldapConfig, res)
    }
  } catch (error) {
    console.error('加载LDAP配置失败', error)
  }
}

// 测试LDAP连接
const handleTestLDAP = async () => {
  if (!ldapConfig.host || !ldapConfig.bindDn || !ldapConfig.baseDn) {
    ElMessage.warning('请先填写服务器地址、Bind DN 和 Base DN')
    return
  }

  ldapTesting.value = true
  ldapTestResult.value = null
  try {
    const res: any = await testLDAPConnection({ ...ldapConfig })
    ldapTestResult.value = {
      success: true,
      message: res?.message || '连接成功',
      userCount: res?.userCount
    }
    ElMessage.success('LDAP连接测试成功')
  } catch (error: any) {
    ldapTestResult.value = {
      success: false,
      message: error?.message || '连接失败'
    }
  } finally {
    ldapTesting.value = false
  }
}

// 保存LDAP配置
const handleSaveLDAP = async () => {
  saving.value = true
  try {
    await saveLDAPConfig({ ...ldapConfig })
    ElMessage.success('LDAP配置保存成功')
  } catch (error) {
    ElMessage.error('LDAP配置保存失败')
  } finally {
    saving.value = false
  }
}

const loadConfig = async () => {
  try {
    const res = await getAllConfig()
    if (res) {
      // 基础配置
      if (res.basic) {
        config.systemName = res.basic.systemName || 'OpsHub'
        config.systemLogo = res.basic.systemLogo || ''
        config.systemDescription = res.basic.systemDescription || '运维管理平台'
      }
      // 安全配置
      if (res.security) {
        config.passwordMinLength = res.security.passwordMinLength || 8
        config.sessionTimeout = res.security.sessionTimeout || 3600
        config.enableCaptcha = res.security.enableCaptcha !== false
        config.maxLoginAttempts = res.security.maxLoginAttempts || 5
        config.lockoutDuration = res.security.lockoutDuration || 300
        // MFA配置
        config.mfaEnabled = res.security.mfaEnabled || false
        config.mfaEnforced = res.security.mfaEnforced || false
        config.mfaType = res.security.mfaType || 'totp'
        config.mfaSkipDuration = res.security.mfaSkipDuration || 2592000
      }
    }
  } catch (error) {
    console.error('加载配置失败', error)
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    if (activeNav.value === 0) {
      // 保存基础配置
      await saveBasicConfig({
        systemName: config.systemName,
        systemLogo: config.systemLogo,
        systemDescription: config.systemDescription
      })
      // 更新全局系统配置（更新侧边栏Logo、网页标题、favicon）
      systemStore.updateConfig({
        systemName: config.systemName,
        systemLogo: config.systemLogo,
        systemDescription: config.systemDescription
      })
    } else if (activeNav.value === 1) {
      // 保存安全配置
      await saveSecurityConfig({
        passwordMinLength: config.passwordMinLength,
        sessionTimeout: config.sessionTimeout,
        enableCaptcha: config.enableCaptcha,
        maxLoginAttempts: config.maxLoginAttempts,
        lockoutDuration: config.lockoutDuration,
        // MFA配置
        mfaEnabled: config.mfaEnabled,
        mfaEnforced: config.mfaEnforced,
        mfaType: config.mfaType,
        mfaSkipDuration: config.mfaSkipDuration
      })
    } else if (activeNav.value === 2) {
      // 保存LDAP配置
      await saveLDAPConfig({ ...ldapConfig })
    }

    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const beforeLogoUpload = (file: File) => {
  const validTypes = ['image/png', 'image/jpeg', 'image/jpg', 'image/x-icon', 'image/svg+xml']
  const isValidType = validTypes.includes(file.type) || file.name.endsWith('.ico') || file.name.endsWith('.svg')
  const isLt2M = file.size / 1024 / 1024 < 2

  if (!isValidType) {
    ElMessage.error('只能上传 png/jpg/jpeg/ico/svg 格式的图片!')
    return false
  }
  if (!isLt2M) {
    ElMessage.error('图片大小不能超过 2MB!')
    return false
  }
  return true
}

const handleLogoUpload = async (options: any) => {
  try {
    const res = await uploadLogo(options.file)
    if (res && res.url) {
      config.systemLogo = res.url
      ElMessage.success('Logo上传成功')
    }
  } catch (error) {
    ElMessage.error('Logo上传失败')
  }
}

const removeLogo = () => {
  config.systemLogo = ''
}

onMounted(() => {
  loadConfig()
  loadLDAPConfig()
  loadRolesAndDepts()
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

/* Logo上传样式 */
.logo-upload-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.logo-preview {
  position: relative;
  width: 120px;
  height: 120px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f7fa;
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.logo-preview .logo-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 8px;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
}

.logo-preview:hover .logo-actions {
  opacity: 1;
}

.logo-uploader {
  width: 120px;
  height: 120px;
}

.logo-uploader :deep(.el-upload) {
  width: 100%;
  height: 100%;
}

.upload-trigger {
  width: 120px;
  height: 120px;
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  background: #fafafa;
}

.upload-trigger:hover {
  border-color: #d4af37;
  background: #fff;
}

.upload-icon {
  font-size: 32px;
  color: #909399;
  margin-bottom: 8px;
}

.upload-text {
  font-size: 12px;
  color: #909399;
}

.upload-tip {
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

/* LDAP 测试结果 */
.ldap-test-button {
  min-width: 112px;
  height: 38px;
  border-radius: 8px;
  border-color: #111827 !important;
  background: #111827 !important;
  color: #ffffff !important;
  font-weight: 600;
  box-shadow: 0 8px 16px rgba(17, 24, 39, 0.12);
}

.ldap-test-button:hover,
.ldap-test-button:focus {
  border-color: #1f2937 !important;
  background: #1f2937 !important;
  color: #ffffff !important;
}

.ldap-test-button.is-loading,
.ldap-test-button.is-disabled {
  border-color: #374151 !important;
  background: #374151 !important;
  color: rgba(255, 255, 255, 0.82) !important;
}

.ldap-test-button :deep(.el-icon) {
  color: currentColor;
}

.ldap-test-result {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 12px;
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 13px;
}

.ldap-test-result.success {
  color: #67c23a;
  background: #f0f9eb;
}

.ldap-test-result.error {
  color: #f56c6c;
  background: #fef0f0;
}

/* LDAP 分割线 */
:deep(.el-divider__text) {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 6px;
}

.divider-icon {
  color: #d4af37;
  font-size: 16px;
}
</style>
