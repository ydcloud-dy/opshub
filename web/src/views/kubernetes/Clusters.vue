<template>
  <div class="clusters-container">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-blue">
          <el-icon><Platform /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">集群总数</div>
          <div class="stat-value">{{ clusterList.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-green">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">运行正常</div>
          <div class="stat-value">{{ clusterList.filter(c => c.status === 1).length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-orange">
          <el-icon><Odometer /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">总节点数</div>
          <div class="stat-value">{{ totalNodeCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-purple">
          <el-icon><Connection /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">自建集群</div>
          <div class="stat-value">{{ clusterList.filter(c => c.provider === 'native').length }}</div>
        </div>
      </div>
    </div>

    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Platform /></el-icon>
        </div>
        <div>
          <h2 class="page-title">集群管理</h2>
          <p class="page-subtitle">管理您的 Kubernetes 集群，支持多云平台统一管理</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="sync-button" @click="handleSyncAll" :loading="syncing">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          同步状态
        </el-button>
        <el-button v-if="isAdmin" class="black-button" @click="handleRegister">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          注册集群
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索集群名称或别名..."
          clearable
          @clear="handleSearch"
          @keyup.enter="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="集群状态"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><CircleCheck /></el-icon>
          </template>
          <el-option label="正常" :value="1" />
          <el-option label="连接失败" :value="2" />
          <el-option label="不可用" :value="3" />
        </el-select>

        <el-input
          v-model="searchForm.version"
          placeholder="集群版本..."
          clearable
          @clear="handleSearch"
          @keyup.enter="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><InfoFilled /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
        <el-button class="search-btn" type="primary" @click="handleSearch">
          <el-icon style="margin-right: 4px;"><Search /></el-icon>
          搜索
        </el-button>
      </div>
    </div>

    <!-- 集群列表 -->
    <div class="table-wrapper">
      <!-- 批量操作栏 -->
      <div v-if="selectedClusters.length > 0" class="batch-actions">
        <span class="selected-info">已选择 {{ selectedClusters.length }} 项</span>
        <el-button type="danger" @click="handleBatchDelete">
          <el-icon style="margin-right: 4px;"><Delete /></el-icon>
          批量删除
        </el-button>
        <el-button @click="selectedClusters = []">取消选择</el-button>
      </div>

      <el-table
        :data="filteredClusterList"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        @selection-change="handleSelectionChange"
      >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="name" min-width="180">
        <template #header>
          <span class="header-with-icon">
            <el-icon class="header-icon header-icon-blue"><Platform /></el-icon>
            集群名称
          </span>
        </template>
        <template #default="{ row }">
          <el-button link type="primary" @click="handleViewDetail(row)" style="font-size: 14px;">
            {{ row.name }}
          </el-button>
        </template>
      </el-table-column>
      <el-table-column prop="alias" label="别名" min-width="120">
        <template #default="{ row }">
          {{ row.alias || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)" effect="dark">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="version" width="120">
        <template #header>
          <span class="header-with-icon">
            <el-icon class="header-icon header-icon-purple"><InfoFilled /></el-icon>
            版本
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="nodeCount" label="节点数" width="100" />
      <el-table-column prop="provider" label="服务商" width="120">
        <template #default="{ row }">
          {{ getProviderText(row.provider) }}
        </template>
      </el-table-column>
      <el-table-column prop="region" label="区域" width="120">
        <template #default="{ row }">
          {{ row.region || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="description" label="备注" min-width="150" show-overflow-tooltip />
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <div class="action-buttons">
            <el-tooltip content="凭证" placement="top">
              <el-button v-if="isAdmin" link class="action-btn" @click="handleViewConfig(row)">
                <el-icon><Key /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="授权" placement="top">
              <el-button link class="action-btn action-auth" @click="handleAuthorize(row)">
                <el-icon><Lock /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="同步" placement="top">
              <el-button link class="action-btn action-sync" @click="handleSync(row)">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="编辑" placement="top">
              <el-button v-if="isAdmin" link class="action-btn action-edit" @click="handleEdit(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <el-button v-if="isAdmin" link class="action-btn action-delete" @click="handleDelete(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
    </el-table>
    </div>

    <!-- 注册/编辑集群对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑集群' : '注册集群'"
      width="700px"
      @close="handleDialogClose"
    >
      <el-form :model="clusterForm" :rules="rules" ref="formRef" label-width="100px">
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="集群名称" prop="name">
                <el-input v-model="clusterForm.name" placeholder="请输入集群名称"  />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="集群别名">
                <el-input v-model="clusterForm.alias" placeholder="可选" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <!-- 认证配置 -->
        <div class="form-section">
          <div class="section-title">认证配置</div>
          <el-form-item label="认证方式">
            <el-radio-group v-model="authType" @change="handleAuthTypeChange">
              <el-radio-button label="config">KubeConfig 文件</el-radio-button>
              <el-radio-button label="token">Service Account Token</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <!-- KubeConfig 方式 -->
          <template v-if="authType === 'config'">
            <el-alert
              v-if="isEdit"
              title="配置信息"
              type="info"
              :closable="false"
              style="margin-bottom: 12px"
            >
              <template #default>
                <div style="font-size: 12px;">
                  <p style="margin: 0 0 8px 0;">
                    <strong>当前集群配置信息：</strong>
                  </p>
                  <ul style="margin: 0; padding-left: 20px;">
                    <li>API Endpoint: {{ clusterForm.apiEndpoint || '未配置' }}</li>
                    <li>服务商: {{ clusterForm.provider ? getProviderText(clusterForm.provider) : '未配置' }}</li>
                    <li>区域: {{ clusterForm.region || '未配置' }}</li>
                  </ul>
                  <p style="margin: 8px 0 0 0; color: #409eff;">
                    💡 下方显示的是当前的 KubeConfig 配置，您可以直接编辑或上传新文件替换
                  </p>
                </div>
              </template>
            </el-alert>
            <el-form-item label="配置内容" prop="kubeConfig">
              <div style="margin-bottom: 8px;">
                <el-button size="small" @click="handleUploadKubeConfig">
                  <el-icon><Upload /></el-icon>
                  上传 KubeConfig 文件
                </el-button>
                <input
                  ref="fileInputRef"
                  type="file"
                  style="display: none"
                  @change="handleFileChange"
                />
              </div>
              <div class="code-editor-wrapper">
                <div class="line-numbers">
                  <div v-for="n in lineCount" :key="n" class="line-number">{{ n }}</div>
                </div>
                <textarea
                  v-model="clusterForm.kubeConfig"
                  class="code-textarea"
                  :placeholder="isEdit ? '' : '请粘贴 KubeConfig 文件内容或点击上方按钮上传'"
                  spellcheck="false"
                  @input="updateLineCount"

                ></textarea>
              </div>
              <div class="code-tip" v-if="!isEdit">
                <el-icon><InfoFilled /></el-icon>
                <span>如何获取 KubeConfig？通常位于 ~/.kube/config 文件中</span>
              </div>
            </el-form-item>
          </template>

          <!-- Token 方式 -->
          <template v-if="authType === 'token'">
            <el-form-item label="API 地址" prop="apiEndpoint">
              <el-input
                v-model="clusterForm.apiEndpoint"
                placeholder="https://k8s-api.example.com:6443"

              >
                <template #prepend>
                  <el-icon><Connection /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="Token" prop="token">
              <div class="code-editor-wrapper">
                <div class="line-numbers">
                  <div v-for="n in tokenLineCount" :key="n" class="line-number">{{ n }}</div>
                </div>
                <textarea
                  v-model="clusterForm.token"
                  class="code-textarea"
                  placeholder="请输入 Service Account Token"
                  spellcheck="false"
                  @input="updateTokenLineCount"

                ></textarea>
              </div>
              <div class="code-tip">
                <el-icon><InfoFilled /></el-icon>
                <span>如何获取 Token？使用 kubectl create token 命令创建</span>
              </div>
            </el-form-item>
          </template>
        </div>

        <!-- 集群信息 -->
        <div class="form-section">
          <div class="section-title">集群信息</div>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="服务商">
                <el-select v-model="clusterForm.provider" placeholder="请选择" style="width: 100%">
                  <el-option label="自建集群" value="native" />
                  <el-option label="阿里云 ACK" value="aliyun" />
                  <el-option label="腾讯云 TKE" value="tencent" />
                  <el-option label="AWS EKS" value="aws" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="区域">
                <el-input v-model="clusterForm.region" placeholder="例如: cn-beijing" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="备注">
            <el-input
              v-model="clusterForm.description"
              type="textarea"
              :rows="2"
              placeholder="请输入集群备注（可选）"
            />
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitLoading">
            {{ isEdit ? '保存' : '注册集群' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 查看集群凭证对话框 -->
    <el-dialog
      v-model="configDialogVisible"
      title="集群凭证"
      width="700px"
    >
      <div style="margin-bottom: 16px;">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="集群名称">{{ currentCluster?.name }}</el-descriptions-item>
          <el-descriptions-item label="别名">{{ currentCluster?.alias || '-' }}</el-descriptions-item>
          <el-descriptions-item label="API Endpoint">{{ currentCluster?.apiEndpoint }}</el-descriptions-item>
          <el-descriptions-item label="版本">{{ currentCluster?.version }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center;">
        <span style="font-weight: 500;">KubeConfig 配置</span>
        <div>
          <el-button size="small" @click="handleCopyConfig">
            <el-icon><DocumentCopy /></el-icon>
            复制
          </el-button>
          <el-button size="small" @click="handleDownloadConfig">
            <el-icon><Download /></el-icon>
            下载
          </el-button>
        </div>
      </div>

      <div class="code-editor-wrapper">
        <div class="line-numbers">
          <div v-for="n in configLineCount" :key="n" class="line-number">{{ n }}</div>
        </div>
        <textarea
          v-model="currentConfig"
          class="code-textarea"
          readonly
          spellcheck="false"
        ></textarea>
      </div>

      <div class="code-tip">
        <el-icon><Warning /></el-icon>
        <span>请妥善保管集群凭证，不要泄露给他人</span>
      </div>
    </el-dialog>

    <!-- 授权对话框 -->
    <el-dialog
      v-model="authorizeDialogVisible"
      title="集群授权"
      width="900px"
    >
      <el-tabs v-model="activeAuthTab" type="border-card">
        <!-- 连接信息 -->
        <el-tab-pane label="连接信息" name="connection">
          <div class="connection-info">
            <div class="info-section">
              <div class="section-title">
                <el-icon><Connection /></el-icon>
                <span>集群连接信息</span>
              </div>
              <el-descriptions :column="2" border style="margin-top: 16px;">
                <el-descriptions-item label="集群名称">{{ currentCluster?.name }}</el-descriptions-item>
                <el-descriptions-item label="别名">{{ currentCluster?.alias || '-' }}</el-descriptions-item>
                <el-descriptions-item label="API Endpoint">{{ currentCluster?.apiEndpoint }}</el-descriptions-item>
                <el-descriptions-item label="版本">{{ currentCluster?.version }}</el-descriptions-item>
              </el-descriptions>
            </div>

            <div class="credential-section">
              <div class="section-header">
                <div class="section-title">
                  <el-icon><Key /></el-icon>
                  <span>凭据管理</span>
                </div>
                <div v-if="!generatedKubeConfig">
                  <el-button
                    type="primary"
                    :icon="Download"
                    @click="handleApplyCredential"
                    :loading="credentialLoading"
                  >
                    凭据申请
                  </el-button>
                </div>
                <div v-else>
                  <el-button
                    type="danger"
                    :icon="Delete"
                    @click="handleRevokeCredential"
                    :loading="revokeLoading"
                  >
                    吊销凭据
                  </el-button>
                </div>
              </div>

              <div v-if="generatedKubeConfig" class="kubeconfig-display">
                <div class="kubeconfig-header">
                  <span style="font-weight: 500;">生成的 KubeConfig 凭据</span>
                  <el-button
                    type="primary"
                    :icon="DocumentCopy"
                    @click="handleCopyKubeConfig"
                    size="small"
                  >
                    复制
                  </el-button>
                </div>
                <el-input
                  v-model="generatedKubeConfig"
                  type="textarea"
                  :rows="10"
                  readonly
                  class="kubeconfig-textarea"
                />
                <div class="code-tip">
                  <el-icon><Warning /></el-icon>
                  <span>此凭据文件包含您的集群访问权限，请妥善保管，不要泄露给他人</span>
                </div>
              </div>

              <div v-else class="no-credential-tip">
                <el-empty description="暂无凭据，请点击上方按钮申请">
                  <template #image>
                    <el-icon :size="60" color="#909399"><Key /></el-icon>
                  </template>
                </el-empty>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 用户 -->
        <el-tab-pane v-if="isAdmin" name="users">
          <template #label>
            <span class="tab-label">
              <el-icon class="tab-icon"><User /></el-icon>
              用户
            </span>
          </template>
          <div class="tab-content">
            <ClusterAuthDialog
              v-if="currentCluster"
              :cluster="currentCluster"
              :model-value="true"
              :credential-users="clusterCredentialUsers"
              @refresh="loadClusterCredentials"
            />
            <el-empty v-else description="请先选择集群" />
          </div>
        </el-tab-pane>

        <!-- 角色 -->
        <el-tab-pane v-if="isAdmin" name="roles">
          <template #label>
            <span class="tab-label">
              <el-icon class="tab-icon"><Key /></el-icon>
              角色
            </span>
          </template>
          <div class="tab-content">
            <UserRoleBinding
              v-if="currentCluster"
              :cluster="currentCluster"
            />
            <el-empty v-else description="请先选择集群" />
          </div>
        </el-tab-pane>
      </el-tabs>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="authorizeDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance } from 'element-plus'
import {
  Search,
  InfoFilled,
  Connection,
  Upload,
  Platform,
  Key,
  Refresh,
  RefreshLeft,
  Plus,
  Edit,
  Delete,
  Lock,
  DocumentCopy,
  Download,
  Warning,
  Odometer,
  CircleCheck,
  User
} from '@element-plus/icons-vue'
import {
  getClusterList,
  createCluster,
  updateCluster,
  deleteCluster,
  testClusterConnection,
  getClusterDetail,
  getClusterConfig,
  generateKubeConfig,
  revokeKubeConfig,
  getClusterCredentialUsers,
  getExistingKubeConfig,
  syncClusterStatus,
  syncAllClustersStatus,
  createDefaultClusterRoles,
  type Cluster,
  type CredentialUser
} from '@/api/kubernetes'
import ClusterAuthDialog from './components/ClusterAuthDialog.vue'
import UserRoleBinding from './components/UserRoleBinding.vue'
import { useUserStore } from '@/stores/user'

// 用户权限
const userStore = useUserStore()
const isAdmin = computed(() => {
  if (!userStore.userInfo) {
    return false
  }

  // 确保 roles 是数组，如果不是则返回 false
  if (!Array.isArray(userStore.userInfo.roles)) {
    return false
  }

  // 检查是否有 admin 角色
  return userStore.userInfo.roles.some((role: any) => role.code === 'admin')
})

const loading = ref(false)
const dialogVisible = ref(false)
const configDialogVisible = ref(false)
const authorizeDialogVisible = ref(false)
const showRoleBindingDialog = ref(false)
const activeAuthTab = ref('connection')
const credentialLoading = ref(false)
const revokeLoading = ref(false)
const generatedKubeConfig = ref('')
const currentCredentialUsername = ref('')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const fileInputRef = ref<HTMLInputElement>()
const authType = ref('config')
const lineCount = ref(1)
const tokenLineCount = ref(1)
const isEdit = ref(false)
const editClusterId = ref<number>()
const kubeConfigEditable = ref(false)
const currentCluster = ref<Cluster>()
const currentConfig = ref('')
const configLineCount = ref(1)
const router = useRouter()
const syncing = ref(false) // 同步状态

const clusterList = ref<Cluster[]>([])
const clusterCredentialUsers = ref<CredentialUser[]>([])
const selectedClusters = ref<Cluster[]>([]) // 选中的集群

// 搜索表单
const searchForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
  version: ''
})

const clusterForm = reactive({
  name: '',
  alias: '',
  apiEndpoint: '',
  kubeConfig: '',
  token: '',
  provider: 'native',
  region: '',
  description: ''
})

const rules = {
  name: [{ required: true, message: '请输入集群名称', trigger: 'blur' }],
  kubeConfig: [
    {
      required: true,
      message: '请输入 KubeConfig',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'config' && !value) {
          callback(new Error('请输入 KubeConfig'))
        } else {
          callback()
        }
      }
    }
  ],
  apiEndpoint: [
    {
      required: true,
      message: '请输入 API Endpoint',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'token' && !value) {
          callback(new Error('请输入 API Endpoint'))
        } else {
          callback()
        }
      }
    }
  ],
  token: [
    {
      required: true,
      message: '请输入 Token',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'token' && !value) {
          callback(new Error('请输入 Token'))
        } else {
          callback()
        }
      }
    }
  ]
}

// 过滤后的集群列表
const filteredClusterList = computed(() => {
  let result = clusterList.value

  // 按关键词搜索（集群名称或别名）
  if (searchForm.keyword) {
    const keyword = searchForm.keyword.toLowerCase()
    result = result.filter(cluster =>
      cluster.name.toLowerCase().includes(keyword) ||
      (cluster.alias || '').toLowerCase().includes(keyword)
    )
  }

  // 按状态筛选
  if (searchForm.status !== undefined) {
    result = result.filter(cluster => cluster.status === searchForm.status)
  }

  // 按版本筛选
  if (searchForm.version) {
    result = result.filter(cluster =>
      cluster.version && cluster.version.toLowerCase().includes(searchForm.version.toLowerCase())
    )
  }

  return result
})

// 总节点数
const totalNodeCount = computed(() => {
  return clusterList.value.reduce((sum, cluster) => sum + (cluster.nodeCount || 0), 0)
})

// 加载集群列表
const loadClusters = async () => {
  loading.value = true
  try {
    const data = await getClusterList()
    // 强制刷新：使用新数组替换旧数组
    clusterList.value = [...(data || [])]
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  // filteredClusterList 会自动更新
}

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  searchForm.version = ''
}

// 注册集群
const handleRegister = () => {
  isEdit.value = false
  kubeConfigEditable.value = true
  dialogVisible.value = true
}

// 查看集群详情
const handleViewDetail = (row: Cluster) => {
  router.push(`/kubernetes/clusters/${row.id}`)
}

// 编辑集群
const handleEdit = async (row: Cluster) => {
  isEdit.value = true
  editClusterId.value = row.id
  kubeConfigEditable.value = true

  try {
    // 获取现有的 kubeconfig 内容
    const config = await getClusterConfig(row.id)

    // 填充表单数据
    Object.assign(clusterForm, {
      name: row.name,
      alias: row.alias,
      apiEndpoint: row.apiEndpoint,
      kubeConfig: config, // 显示现有的 KubeConfig
      token: "",
      provider: row.provider,
      region: row.region,
      description: row.description
    })

    // 更新行号
    updateLineCount()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '获取集群配置失败')
    // 即使失败也打开对话框，但不显示配置
    Object.assign(clusterForm, {
      name: row.name,
      alias: row.alias,
      apiEndpoint: row.apiEndpoint,
      kubeConfig: "",
      token: "",
      provider: row.provider,
      region: row.region,
      description: row.description
    })
  }

  dialogVisible.value = true
}

// 同步集群信息
const handleSync = async (row: Cluster) => {
  const loadingMsg = ElMessage.info({
    message: '正在同步集群信息...',
    duration: 0,
    type: 'info'
  })

  try {
    // 调用新的同步状态 API
    await syncClusterStatus(row.id)

    // 等待一小段时间让同步完成
    await new Promise(resolve => setTimeout(resolve, 2000))

    loadingMsg.close()

    // 重新加载列表
    await loadClusters()
    ElMessage.success('同步成功')
  } catch (error: any) {
    loadingMsg.close()
    ElMessage.error(error.response?.data?.message || '同步失败')
  }
}

// 同步所有集群状态
const handleSyncAll = async () => {
  syncing.value = true
  try {
    await syncAllClustersStatus()

    // 等待一小段时间让同步完成
    await new Promise(resolve => setTimeout(resolve, 3000))

    // 重新加载列表
    await loadClusters()
    ElMessage.success('批量同步任务已启动，请稍后刷新查看')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '同步失败')
  } finally {
    syncing.value = false
  }
}

// 处理表格选择变化
const handleSelectionChange = (selection: Cluster[]) => {
  selectedClusters.value = selection
}

// 批量删除集群
const handleBatchDelete = async () => {
  if (selectedClusters.value.length === 0) {
    ElMessage.warning('请选择要删除的集群')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedClusters.value.length} 个集群吗？此操作不可恢复！`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 并发删除所有选中的集群
    const deletePromises = selectedClusters.value.map(cluster => deleteCluster(cluster.id))
    await Promise.all(deletePromises)

    selectedClusters.value = []
    await loadClusters()
    ElMessage.success('删除成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 认证方式切换
const handleAuthTypeChange = () => {
  formRef.value?.clearValidate()
  setTimeout(() => {
    formRef.value?.validate()
  }, 50)
}

// 更新行号
const updateLineCount = () => {
  const lines = clusterForm.kubeConfig.split('\n').length
  lineCount.value = lines || 1
}

// 更新 Token 行号
const updateTokenLineCount = () => {
  const lines = clusterForm.token.split('\n').length
  tokenLineCount.value = lines || 1
}

// 上传 KubeConfig 文件
const handleUploadKubeConfig = () => {
  fileInputRef.value?.click()
}

// 处理文件选择
const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]

  if (!file) return

  const reader = new FileReader()
  reader.onload = (e) => {
    const content = e.target?.result as string
    clusterForm.kubeConfig = content
    updateLineCount()
    ElMessage.success('文件读取成功')
  }
  reader.onerror = () => {
    ElMessage.error('文件读取失败')
  }
  reader.readAsText(file)

  // 清空 input value，允许重复上传同一文件
  target.value = ''
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        let kubeConfig = clusterForm.kubeConfig
        if (authType.value === 'token') {
          kubeConfig = buildKubeConfigFromToken(
            clusterForm.apiEndpoint,
            clusterForm.token
          )
        }

        if (isEdit.value && editClusterId.value) {
          // 编辑模式 - 可以更新名称、备注、服务商等信息
          // 如果需要更新 KubeConfig，在编辑模式下重新输入即可
          const updateData: any = {
            name: clusterForm.name,
            alias: clusterForm.alias,
            region: clusterForm.region,
            provider: clusterForm.provider,
            description: clusterForm.description
          }

          // 如果重新输入了 KubeConfig，则更新它
          if (clusterForm.kubeConfig && authType.value === 'config') {
            updateData.kubeConfig = clusterForm.kubeConfig
          } else if (clusterForm.token && authType.value === 'token') {
            updateData.kubeConfig = buildKubeConfigFromToken(
              clusterForm.apiEndpoint,
              clusterForm.token
            )
            updateData.apiEndpoint = clusterForm.apiEndpoint
          }

          await updateCluster(editClusterId.value, updateData)
          ElMessage.success('更新成功')
        } else {
          // 新增模式
          const requestData: any = {
            name: clusterForm.name,
            kubeConfig: kubeConfig
          }

          if (authType.value === 'token') {
            requestData.apiEndpoint = clusterForm.apiEndpoint
          }

          if (clusterForm.alias) requestData.alias = clusterForm.alias
          if (clusterForm.provider) requestData.provider = clusterForm.provider
          if (clusterForm.region) requestData.region = clusterForm.region
          if (clusterForm.description) requestData.description = clusterForm.description

          const newCluster = await createCluster(requestData)
          ElMessage.success('集群注册成功')

          // 注册成功后立即创建默认集群角色，显示加载提示
          const roleLoadingMsg = ElMessage.info({
            message: '正在创建默认集群角色，请稍候...',
            duration: 0,
            showClose: false
          })

          try {
            await createDefaultClusterRoles(newCluster.id)
            roleLoadingMsg.close()
            ElMessage.success('默认集群角色创建成功')
            console.log('默认集群角色创建成功')
          } catch (roleError) {
            roleLoadingMsg.close()
            console.error('创建默认集群角色失败:', roleError)
            ElMessage.warning('集群注册成功，但创建默认角色失败，请稍后在角色管理页面手动创建')
            // 角色创建失败不影响集群注册，只记录错误
          }
        }

        dialogVisible.value = false
        loadClusters()
      } catch (error: any) {
        ElMessage.error(error.response?.data?.message || '操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

// 从 Token 构建 KubeConfig
const buildKubeConfigFromToken = (apiEndpoint: string, token: string) => {
  return `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ""
    server: ${apiEndpoint}
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: default-user
  name: default-context
current-context: default-context
users:
- name: default-user
  user:
    token: ${token}
`
}

// 测试连接
const handleTestConnection = async (row: Cluster) => {
  const loadingMsg = ElMessage.info({
    message: '正在测试连接...',
    duration: 0,
    type: 'info'
  })

  try {
    const result = await testClusterConnection(row.id)
    loadingMsg.close()

    // 重新加载列表以更新节点数
    await loadClusters()

    ElMessage.success(`连接成功！版本: ${result.version}`)
  } catch (error: any) {
    loadingMsg.close()
    ElMessage.error(error.response?.data?.message || '连接失败')
  }
}

// 删除集群
const handleDelete = async (row: Cluster) => {
  try {
    await ElMessageBox.confirm('确定要删除该集群吗？此操作不可恢复！', '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    await deleteCluster(row.id)
    ElMessage.success('删除成功')
    loadClusters()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 查看集群凭证
const handleViewConfig = async (row: Cluster) => {
  try {
    const cluster = await getClusterDetail(row.id)
    currentCluster.value = cluster

    // 获取解密后的 KubeConfig
    const config = await getClusterConfig(row.id)
    currentConfig.value = config

    configDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '获取集群凭证失败')
  }
}

// 监听 config 内容变化，更新行号
watch(currentConfig, () => {
  const lines = currentConfig.value.split('\n').length
  configLineCount.value = lines || 1
})

// 复制配置
const handleCopyConfig = async () => {
  try {
    await navigator.clipboard.writeText(currentConfig.value)
    ElMessage.success('复制成功')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}

// 下载配置
const handleDownloadConfig = () => {
  const blob = new Blob([currentConfig.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  const filename = `kubeconfig-${currentCluster.value?.name || 'cluster'}.conf`
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('下载成功')
}

// 加载集群凭据用户列表
const loadClusterCredentials = async () => {
  if (!currentCluster.value) return

  try {
    const users = await getClusterCredentialUsers(currentCluster.value.id)
    clusterCredentialUsers.value = users
    // 不再自动刷新当前用户凭据，避免误清空
  } catch (error: any) {
    console.error('加载凭据用户失败:', error)
    ElMessage.error(error.response?.data?.message || '加载凭据用户失败')
  }
}

// 刷新当前用户的凭据
const refreshCurrentUserCredential = async () => {
  if (!currentCluster.value) return

  try {
    const result = await getExistingKubeConfig(currentCluster.value.id)
    generatedKubeConfig.value = result.kubeconfig
    currentCredentialUsername.value = result.username

    // 保存到localStorage
    const username = getCurrentUsername()
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.setItem(storageKey, result.kubeconfig)
    localStorage.setItem(usernameKey, result.username)
  } catch (error: any) {
    // 只有明确的 404 错误（用户尚未申请凭据）才清空显示
    // 其他错误（如网络错误、后端查找失败）不清空，保持现有状态
    if (error.response?.status === 404) {
      generatedKubeConfig.value = ''
      currentCredentialUsername.value = ''
      // 同时清除 localStorage
      const username = getCurrentUsername()
      const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
      const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
      localStorage.removeItem(storageKey)
      localStorage.removeItem(usernameKey)
    } else {
      // 其他错误，记录日志但不清空凭据
      console.error('刷新当前用户凭据失败:', error)
    }
  }
}

// 打开授权对话框
const handleAuthorize = async (row: Cluster) => {
  try {
    const cluster = await getClusterDetail(row.id)
    currentCluster.value = cluster

    authorizeDialogVisible.value = true
    activeAuthTab.value = 'connection'

    // 先尝试从后端API获取用户现有的kubeconfig
    try {
      const result = await getExistingKubeConfig(cluster.id)
      generatedKubeConfig.value = result.kubeconfig
      currentCredentialUsername.value = result.username

      // 保存到localStorage
      const username = getCurrentUsername()
      const storageKey = `kubeconfig_${cluster.id}_${username}`
      const usernameKey = `kubeconfig_username_${cluster.id}_${username}`
      localStorage.setItem(storageKey, result.kubeconfig)
      localStorage.setItem(usernameKey, result.username)
    } catch (error: any) {
      // 如果是404错误（用户尚未申请凭据），清空显示
      if (error.response?.status === 404) {
        generatedKubeConfig.value = ''
        currentCredentialUsername.value = ''
      } else {
        // 其他错误，也清空显示
        console.error('获取现有kubeconfig失败:', error)
        generatedKubeConfig.value = ''
        currentCredentialUsername.value = ''
      }
    }

    // 加载凭据用户列表（从后端API获取）
    await loadClusterCredentials()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '获取集群信息失败')
  }
}

// 申请凭据
const handleApplyCredential = async () => {
  if (!currentCluster.value) return

  try {
    credentialLoading.value = true

    // 获取当前用户名
    const username = getCurrentUsername()

    // 调用后端API生成kubeconfig
    const result = await generateKubeConfig(currentCluster.value.id, username)
    generatedKubeConfig.value = result.kubeconfig
    currentCredentialUsername.value = result.username

    // 保存到 localStorage
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.setItem(storageKey, result.kubeconfig)
    localStorage.setItem(usernameKey, result.username)

    ElMessage.success('凭据申请成功')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '凭据申请失败')
  } finally {
    credentialLoading.value = false
  }
}

// 吊销凭据
const handleRevokeCredential = async () => {
  if (!currentCluster.value || !currentCredentialUsername.value) return

  try {
    await ElMessageBox.confirm('确定要吊销该凭据吗？吊销后将无法使用该 KubeConfig 访问集群。', '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    revokeLoading.value = true

    // 调用后端API撤销kubeconfig
    await revokeKubeConfig(currentCluster.value.id, currentCredentialUsername.value)

    // 清空凭据
    generatedKubeConfig.value = ''
    currentCredentialUsername.value = ''

    // 清除 localStorage 中的凭据
    const username = getCurrentUsername()
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.removeItem(storageKey)
    localStorage.removeItem(usernameKey)

    ElMessage.success('凭据吊销成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '凭据吊销失败')
    }
  } finally {
    revokeLoading.value = false
  }
}

// 获取当前用户名
const getCurrentUsername = () => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      return user.username || 'opshub-user'
    } catch {
      return 'opshub-user'
    }
  }
  return 'opshub-user'
}

// 复制生成的kubeconfig
const handleCopyKubeConfig = async () => {
  try {
    await navigator.clipboard.writeText(generatedKubeConfig.value)
    ElMessage.success('复制成功')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}

// 关闭对话框
const handleDialogClose = () => {
  formRef.value?.resetFields()
  Object.assign(clusterForm, {
    name: '',
    alias: '',
    apiEndpoint: '',
    kubeConfig: '',
    token: '',
    provider: 'native',
    region: '',
    description: ''
  })
  authType.value = 'config'
  isEdit.value = false
  editClusterId.value = undefined
  kubeConfigEditable.value = true
}

// 获取状态类型
const getStatusType = (status: number) => {
  const statusMap: Record<number, string> = {
    1: 'success',
    2: 'danger',
    3: 'info'
  }
  return statusMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (status: number) => {
  const statusMap: Record<number, string> = {
    1: '正常',
    2: '连接失败',
    3: '不可用'
  }
  return statusMap[status] || '未知'
}

// 获取服务商文本
const getProviderText = (provider: string) => {
  const providerMap: Record<string, string> = {
    native: '自建集群',
    aliyun: '阿里云 ACK',
    tencent: '腾讯云 TKE',
    aws: 'AWS EKS'
  }
  return providerMap[provider] || provider || '未配置'
}

onMounted(async () => {
  // 确保用户信息已加载
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (error) {
      console.error('获取用户信息失败:', error)
    }
  }

  loadClusters()
})

// 监听标签页切换，当切换到用户标签时加载凭据用户列表，切换到连接信息标签时刷新当前用户凭据
watch(activeAuthTab, async (newTab) => {
  if (!currentCluster.value) return

  if (newTab === 'users') {
    // 切换到用户标签，加载凭据用户列表
    await loadClusterCredentials()
  } else if (newTab === 'connection') {
    // 切换到连接信息标签，刷新当前用户的凭据
    await refreshCurrentUserCredential()
  }
})
</script>

<style scoped>
.clusters-container {
  padding: 0;
  background-color: transparent;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  flex-shrink: 0;
}

.stat-icon-blue {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-green {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-orange {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-purple {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #d4af37;
  line-height: 1;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
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
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.sync-button {
  background: linear-gradient(135deg, #67C23A 0%, #85CE61 100%);
  color: #fff;
  border: none;
  font-weight: 500;
  padding: 10px 20px;
  font-size: 14px;
  border-radius: 8px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.2);

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(103, 194, 58, 0.4);
    background: linear-gradient(135deg, #85CE61 0%, #67C23A 100%);
  }

  &:active {
    transform: translateY(0);
    box-shadow: 0 2px 8px rgba(103, 194, 58, 0.3);
  }
}

/* 批量操作栏 */
.batch-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 12px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  border-left: 4px solid #409EFF;
}

.selected-info {
  flex: 1;
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 150px;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

.search-btn {
  background: #000;
  border-color: #000;
}

.search-btn:hover {
  background: #333;
  border-color: #333;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.modern-table :deep(.el-button--link) {
  transition: all 0.2s ease;
}

.modern-table :deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 搜索框样式优化 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-bar :deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.search-icon {
  color: #d4af37;
}

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-blue {
  color: #d4af37;
}

.header-icon-purple {
  color: #d4af37;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  background-color: #f5f7fa;
  transform: scale(1.1);
}

.action-auth:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-sync:hover {
  background-color: #e8f8f0;
  color: #67c23a;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

.form-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px dashed #dcdfe6;
}

.form-section:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-left: 8px;
  border-left: 3px solid #000000;
}

.code-editor-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
  background-color: #282c34;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 12px 8px;
  background-color: #21252b;
  border-right: 1px solid #3e4451;
  user-select: none;
  min-width: 40px;
  text-align: right;
}

.line-number {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #5c6370;
  min-height: 20.8px;
}

.code-textarea {
  flex: 1;
  min-height: 200px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #abb2bf;
  background-color: #282c34;
  border: none;
  outline: none;
  resize: vertical;
  font-feature-settings: "liga" 0;
}

.code-textarea::placeholder {
  color: #5c6370;
}

.code-textarea:focus {
  background-color: #282c34;
  color: #abb2bf;
}

.code-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 12px;
  background-color: #f4f4f5;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
}

.code-tip .el-icon {
  color: #409eff;
  font-size: 14px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 授权对话框样式 */
.connection-info {
  padding: 20px;
}

.info-section {
  margin-bottom: 24px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  font-size: 15px;
  color: #303133;
  margin-bottom: 12px;
}

.credential-section {
  margin-top: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.kubeconfig-display {
  margin-top: 16px;
}

.kubeconfig-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.kubeconfig-textarea :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.5;
  background-color: #f5f7fa;
}

.no-credential-tip {
  padding: 40px 0;
  text-align: center;
}

.tab-content {
  padding: 20px;
  text-align: center;
}

/* 授权对话框标签页样式 */
.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-icon {
  font-size: 16px;
  color: #d4af37;
}

:deep(.el-tabs__item) {
  &.is-active {
    .tab-icon {
      color: #d4af37;
    }
  }
}
</style>
