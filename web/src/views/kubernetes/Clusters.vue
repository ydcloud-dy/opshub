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
          <el-icon><Monitor /></el-icon>
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
      <div v-if="selectedClusters.length > 0" class="batch-actions-bar">
        <div class="batch-actions-left">
          <el-checkbox
            v-model="selectAllCurrentPage"
            :indeterminate="isIndeterminate"
            @change="handleSelectAllCurrentPage"
          >
            <span class="selected-count">已选择 {{ selectedClusters.length }} 个集群</span>
          </el-checkbox>
        </div>
        <div class="batch-actions-right">
          <el-button type="primary" :icon="Refresh" @click="handleBatchSync" plain>
            批量同步
          </el-button>
          <el-button type="danger" :icon="Delete" @click="handleBatchDelete" plain>
            批量删除
          </el-button>
          <el-button @click="clearSelection">取消选择</el-button>
        </div>
      </div>

      <el-table
        ref="clusterTableRef"
        :data="paginatedClusterList"
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
          <el-button link @click="handleViewDetail(row)" class="cluster-name-link">
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredClusterList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 注册/编辑集群对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑集群' : '注册集群'"
      width="1080px"
      class="cluster-edit-dialog"
      @close="handleDialogClose"
    >
      <div class="cluster-dialog-hero">
        <div class="cluster-dialog-hero-icon">
          <el-icon><Platform /></el-icon>
        </div>
        <div class="cluster-dialog-hero-content">
          <div class="cluster-dialog-hero-title">{{ isEdit ? '更新 Kubernetes 集群连接' : '接入 Kubernetes 集群' }}</div>
          <div class="cluster-dialog-hero-desc">
            支持 KubeConfig 或 Service Account Token，保存后即可统一管理资源、权限与状态同步。
          </div>
        </div>
      </div>

      <el-form :model="clusterForm" :rules="rules" ref="formRef" label-width="100px" class="cluster-form">
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="section-title">
            <span class="section-title-dot"></span>
            <span>基本信息</span>
          </div>
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
          <div class="section-title">
            <span class="section-title-dot"></span>
            <span>认证配置</span>
          </div>
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
              <div class="upload-row">
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
            <el-form-item label="TLS 验证">
              <el-switch v-model="skipTLSVerify" active-text="跳过验证" inactive-text="验证证书" />
              <span class="tls-tip">
                跳过 TLS 验证仅适用于测试环境，生产环境请提供 CA 证书
              </span>
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
          <div class="section-title">
            <span class="section-title-dot"></span>
            <span>集群信息</span>
          </div>
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
          <el-button type="primary" class="cluster-submit-button" @click="handleSubmit" :loading="submitLoading">
            {{ isEdit ? '保存' : '注册集群' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 查看集群凭证对话框 -->
    <el-dialog
      v-model="configDialogVisible"
      title="集群凭证"
      width="900px"
      class="cluster-config-dialog"
    >
      <div class="credential-overview">
        <div class="credential-overview-icon">
          <el-icon><Key /></el-icon>
        </div>
        <div class="credential-overview-content">
          <div class="credential-overview-title">{{ currentCluster?.alias || currentCluster?.name || '集群凭证' }}</div>
          <div class="credential-overview-desc">查看、复制或下载该集群的 KubeConfig 配置，请妥善保管访问凭证。</div>
        </div>
      </div>

      <div class="credential-description">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="集群名称">{{ currentCluster?.name }}</el-descriptions-item>
          <el-descriptions-item label="别名">{{ currentCluster?.alias || '-' }}</el-descriptions-item>
          <el-descriptions-item label="API Endpoint">{{ currentCluster?.apiEndpoint }}</el-descriptions-item>
          <el-descriptions-item label="版本">{{ currentCluster?.version }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="credential-toolbar">
        <div class="credential-toolbar-title">
          <span class="section-title-dot"></span>
          <span>KubeConfig 配置</span>
        </div>
        <div class="credential-toolbar-actions">
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
  Monitor,
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
  createDefaultNamespaceRoles,
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
const skipTLSVerify = ref(true)  // 默认跳过 TLS 验证，适用于自签名证书
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
const clusterTableRef = ref() // 表格引用
const selectAllCurrentPage = ref(false) // 全选当前页
const isIndeterminate = ref(false) // 半选状态

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('cluster_list_pagination')

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

// 分页后的集群列表
const paginatedClusterList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredClusterList.value.slice(start, end)
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
    // 恢复分页状态
    restorePaginationState()
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  } finally {
    loading.value = false
  }
}

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
    // 保存分页状态失败
  }
}

// 从 localStorage 恢复分页状态
const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch (error) {
    currentPage.value = 1
    pageSize.value = 10
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  // 当每页数量变化时，可能需要调整当前页码
  const maxPage = Math.ceil(filteredClusterList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
  savePaginationState()
}

// 搜索
const handleSearch = () => {
  // 搜索时重置到第一页
  currentPage.value = 1
  savePaginationState()
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
  updateSelectAllStatus()
}

// 更新全选状态
const updateSelectAllStatus = () => {
  const currentPageCount = paginatedClusterList.value.length
  const selectedCount = selectedClusters.value.length

  if (selectedCount === 0) {
    selectAllCurrentPage.value = false
    isIndeterminate.value = false
  } else if (selectedCount === currentPageCount) {
    selectAllCurrentPage.value = true
    isIndeterminate.value = false
  } else {
    selectAllCurrentPage.value = false
    isIndeterminate.value = true
  }
}

// 处理当前页全选
const handleSelectAllCurrentPage = (checked: boolean) => {
  if (checked) {
    // 添加当前页所有集群到已选择列表（去重）
    const currentPageIds = new Set(selectedClusters.value.map(c => c.id))
    paginatedClusterList.value.forEach(cluster => {
      if (!currentPageIds.has(cluster.id)) {
        selectedClusters.value.push(cluster)
      }
    })
  } else {
    // 移除当前页的集群
    const currentPageIds = new Set(paginatedClusterList.value.map(c => c.id))
    selectedClusters.value = selectedClusters.value.filter(c => !currentPageIds.has(c.id))
  }
  updateSelectAllStatus()
  // 同步表格选择状态
  syncTableSelection()
}

// 同步表格选择状态
const syncTableSelection = () => {
  if (clusterTableRef.value) {
    const selectedIds = new Set(selectedClusters.value.map(c => c.id))
    paginatedClusterList.value.forEach(row => {
      const isSelected = selectedIds.has(row.id)
      clusterTableRef.value.toggleRowSelection(row, isSelected)
    })
  }
}

// 清除选择
const clearSelection = () => {
  selectedClusters.value = []
  selectAllCurrentPage.value = false
  isIndeterminate.value = false
  if (clusterTableRef.value) {
    clusterTableRef.value.clearSelection()
  }
}

const isConfirmCancel = (error: unknown) => error === 'cancel' || error === 'close'

const getErrorMessage = (error: any, fallback: string) => {
  return error?.response?.data?.message || error?.message || fallback
}

const escapeHtml = (value: string) => {
  return value.replace(/[&<>"']/g, char => {
    const entities: Record<string, string> = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    }
    return entities[char] ?? char
  })
}

const deleteClusterWithLoading = async (cluster: Cluster, force = false) => {
  const loadingMsg = ElMessage.info({
    message: force ? '正在强制删除 OpsHub 本地记录，请稍候...' : '正在删除集群，请稍候...',
    duration: 0,
    type: 'info'
  })

  try {
    await deleteCluster(cluster.id, force)
  } finally {
    loadingMsg.close()
  }
}

const confirmForceDeleteCluster = async (cluster: Cluster, reason: string) => {
  await ElMessageBox.confirm(
    `<div style="line-height: 1.8;">
      <p style="margin: 0 0 12px 0; font-weight: 600; color: #e6a23c;">
        集群 <strong>"${escapeHtml(cluster.name)}"</strong> 删除时无法完成集群内资源清理。
      </p>
      <div style="padding: 12px; background: #fff7e6; border-left: 3px solid #e6a23c; margin-bottom: 10px; border-radius: 4px;">
        <p style="margin: 0 0 8px 0; color: #606266; font-size: 14px;"><strong>失败原因：</strong></p>
        <p style="margin: 0; color: #909399; font-size: 13px; word-break: break-all;">${escapeHtml(reason)}</p>
      </div>
      <p style="margin: 0; color: #606266; font-size: 13px;">
        如果这个集群已经在云上删除或注销，可以强制删除 OpsHub 本地的集群记录、访问凭据、角色绑定和缓存。
      </p>
      <p style="margin: 8px 0 0 0; color: #f56c6c; font-size: 13px;">
        强制删除不会再连接目标集群，也不会清理真实集群内残留资源。
      </p>
    </div>`,
    '强制删除本地记录',
    {
      type: 'warning',
      confirmButtonText: '强制删除本地记录',
      cancelButtonText: '取消',
      dangerouslyUseHTMLString: true,
      customClass: 'delete-cluster-confirm'
    }
  )

  await deleteClusterWithLoading(cluster, true)
}

// 批量同步集群
const handleBatchSync = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要同步选中的 ${selectedClusters.value.length} 个集群吗？`,
      '批量同步确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      }
    )

    const loadingMsg = ElMessage.info({
      message: `正在同步 ${selectedClusters.value.length} 个集群，请稍候...`,
      duration: 0,
      type: 'info'
    })

    // 并发同步所有选中的集群
    const syncPromises = selectedClusters.value.map(cluster => syncClusterStatus(cluster.id))
    await Promise.all(syncPromises)

    loadingMsg.close()
    clearSelection()
    await loadClusters()
    ElMessage.success('同步成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '同步失败')
    }
  }
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

    const clustersToDelete = [...selectedClusters.value]

    // 显示正在删除的提示
    const loadingMsg = ElMessage.info({
      message: `正在删除 ${clustersToDelete.length} 个集群，请稍候...`,
      duration: 0,
      type: 'info'
    })

    let results: PromiseSettledResult<unknown>[] = []
    try {
      // 并发删除所有选中的集群
      results = await Promise.allSettled(clustersToDelete.map(cluster => deleteCluster(cluster.id)))
    } finally {
      loadingMsg.close()
    }

    const failedClusters = results
      .map((result, index) => ({ result, cluster: clustersToDelete[index]! }))
      .filter((item): item is { result: PromiseRejectedResult; cluster: Cluster } => item.result.status === 'rejected')

    if (failedClusters.length > 0) {
      const firstReason = getErrorMessage(failedClusters[0]!.result.reason, '删除失败')
      await ElMessageBox.confirm(
        `有 ${failedClusters.length} 个集群无法完成集群内资源清理。若这些集群已经不存在，是否强制删除它们在 OpsHub 的本地记录？\n\n失败原因：${firstReason}`,
        '批量强制删除确认',
        {
          confirmButtonText: '强制删除本地记录',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      const forceLoadingMsg = ElMessage.info({
        message: `正在强制删除 ${failedClusters.length} 个集群的本地记录，请稍候...`,
        duration: 0,
        type: 'info'
      })

      let forceResults: PromiseSettledResult<unknown>[] = []
      try {
        forceResults = await Promise.allSettled(
          failedClusters.map(item => deleteCluster(item.cluster.id, true))
        )
      } finally {
        forceLoadingMsg.close()
      }

      const forceFailedCount = forceResults.filter(result => result.status === 'rejected').length
      if (forceFailedCount > 0) {
        ElMessage.error(`仍有 ${forceFailedCount} 个集群强制删除失败，请稍后重试`)
      } else {
        ElMessage.success('已强制删除不可达集群的本地记录')
      }
    } else {
      ElMessage.success('删除成功')
    }

    clearSelection()
    await loadClusters()
  } catch (error: any) {
    if (!isConfirmCancel(error)) {
      ElMessage.error(getErrorMessage(error, '删除失败'))
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

          // 注册成功后立即创建默认集群角色和常用命名空间角色
          const roleLoadingMsg = ElMessage.info({
            message: '正在初始化默认角色，请稍候...',
            duration: 0,
            showClose: false
          })

          try {
            // 并行创建集群角色和命名空间角色（ClusterRole）
            const [clusterRolesResult, namespaceRolesResult] = await Promise.all([
              createDefaultClusterRoles(newCluster.id),
              createDefaultNamespaceRoles(newCluster.id).catch(() => {
                // 命名空间角色创建失败不影响整体流程
                return { created: [] }
              })
            ])

            roleLoadingMsg.close()

            const clusterCount = clusterRolesResult?.created?.length || 0
            const namespaceCount = namespaceRolesResult?.created?.length || 0
            ElMessage.success(`默认角色初始化完成（集群角色：${clusterCount}个，命名空间角色：${namespaceCount}个）`)
          } catch (roleError) {
            roleLoadingMsg.close()
            ElMessage.warning('集群注册成功，但创建默认角色失败，请稍后在角色管理页面手动创建')
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
  // 根据 skipTLSVerify 决定是否跳过 TLS 验证
  const tlsConfig = skipTLSVerify.value
    ? '    insecure-skip-tls-verify: true'
    : '    certificate-authority-data: ""'

  return `apiVersion: v1
kind: Config
clusters:
- cluster:
${tlsConfig}
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
    await ElMessageBox.confirm(
      `<div style="line-height: 1.8;">
        <p style="margin-bottom: 12px; font-weight: 600; color: #f56c6c;">
          <i class="el-icon-warning" style="margin-right: 4px;"></i>
          确定要删除集群 <strong>"${escapeHtml(row.name)}"</strong> 吗？
        </p>
        <div style="padding: 12px; background: #fef0f0; border-left: 3px solid #f56c6c; margin-bottom: 8px; border-radius: 4px;">
          <p style="margin: 0 0 8px 0; color: #606266; font-size: 14px;"><strong>删除集群将同时清理以下资源：</strong></p>
          <ul style="margin: 0; padding-left: 20px; color: #909399; font-size: 13px;">
            <li>所有用户的集群访问凭据（ServiceAccount）</li>
            <li>所有用户的角色绑定（ClusterRoleBinding 和 RoleBinding）</li>
            <li>所有默认集群角色（ClusterRole）</li>
            <li>所有命名空间中的 OpsHub 管理的 RoleBinding</li>
            <li>数据库中的集群记录、访问凭据和角色绑定数据</li>
          </ul>
        </div>
        <p style="margin: 8px 0 0 0; color: #e6a23c; font-size: 13px;">
          <i class="el-icon-warning" style="margin-right: 4px;"></i>
          此操作不可恢复，请谨慎操作！
        </p>
      </div>`,
      '删除集群',
      {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        dangerouslyUseHTMLString: true,
        customClass: 'delete-cluster-confirm'
      }
    )
  } catch (error: any) {
    if (!isConfirmCancel(error)) {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
    return
  }

  try {
    await deleteClusterWithLoading(row)
  } catch (error: any) {
    if (isConfirmCancel(error)) {
      return
    }

    const reason = getErrorMessage(error, '删除失败')
    try {
      await confirmForceDeleteCluster(row, reason)
    } catch (forceError: any) {
      if (!isConfirmCancel(forceError)) {
        ElMessage.error(getErrorMessage(forceError, '强制删除失败'))
      }
      return
    }

    ElMessage.success('已强制删除 OpsHub 本地集群记录')
    await loadClusters()
    return
  }

  ElMessage.success('集群已删除，所有相关资源已清理')
  await loadClusters()
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
      // 其他错误，不清空凭据
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
  skipTLSVerify.value = true  // 重置 TLS 验证选项
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
      // 获取用户信息失败
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

// 监听分页列表变化，保持选中状态
watch(paginatedClusterList, () => {
  if (selectedClusters.value.length > 0) {
    nextTick(() => {
      syncTableSelection()
    })
  }
}, { flush: 'post' })
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
.batch-actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  margin-bottom: 16px;
  background: #fff;
  border: 2px solid rgba(212, 175, 55, 0.3);
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.15);
  animation: slideIn 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.batch-actions-left {
  display: flex;
  align-items: center;
}

.batch-actions-left :deep(.el-checkbox__label) {
  font-size: 15px;
  color: #1a1a1a;
  font-weight: 600;
}

/* 批量操作栏内的复选框样式 */
.batch-actions-left :deep(.el-checkbox) {
  display: flex;
  align-items: center;
  height: auto;
}

.batch-actions-left :deep(.el-checkbox__input) {
  display: inline-flex;
  align-items: center;
  position: relative;
  white-space: nowrap;
  cursor: pointer;
  outline: none;
  line-height: 1;
}

.batch-actions-left :deep(.el-checkbox__inner) {
  border: 2px solid #d4af37;
  background: #ffffff;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  box-sizing: border-box;
}

.batch-actions-left :deep(.el-checkbox__inner:hover) {
  border-color: #c9a227;
  box-shadow: 0 0 0 3px rgba(212, 175, 55, 0.15);
}

.batch-actions-left :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.batch-actions-left :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.batch-actions-left :deep(.el-checkbox__input.is-checked .el-checkbox__inner::after) {
  border-color: #ffffff;
  border-width: 2px;
  height: 11px;
  left: 6px;
  width: 5px;
}

.batch-actions-left :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner::before) {
  background-color: #ffffff;
  height: 2px;
  top: 10px;
  width: 10px;
  left: 6px;
}

.batch-actions-left :deep(.el-checkbox__label) {
  font-size: 15px;
  color: #1a1a1a;
  font-weight: 600;
  line-height: 22px;
  padding-left: 8px;
}

.selected-count {
  font-size: 15px;
  color: #1a1a1a;
  font-weight: 600;
}

.batch-actions-right {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.batch-actions-right .el-button {
  border-radius: 8px;
  font-weight: 600;
  font-size: 14px;
  padding: 10px 18px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border: 2px solid;
  background: #1a1a1a;
  color: #ffffff;
  border-color: #1a1a1a;
}

.batch-actions-right .el-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
  background: #333333;
  border-color: #333333;
  color: #ffffff;
}

/* 移除特定按钮类型的颜色样式，统一使用黑底白字 */
.batch-actions-right .el-button--primary,
.batch-actions-right .el-button--warning,
.batch-actions-right .el-button--info,
.batch-actions-right .el-button--danger,
.batch-actions-right .el-button--success {
  background: #1a1a1a;
  border-color: #1a1a1a;
  color: #ffffff;
}

.batch-actions-right .el-button--primary:hover,
.batch-actions-right .el-button--warning:hover,
.batch-actions-right .el-button--info:hover,
.batch-actions-right .el-button--danger:hover,
.batch-actions-right .el-button--success:hover {
  background: #333333;
  border-color: #333333;
  color: #ffffff;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
}

/* 表格复选框金色样式 */
.modern-table :deep(.el-checkbox__inner) {
  border: 2px solid #d4af37;
  background: #ffffff;
  border-radius: 4px;
}

.modern-table :deep(.el-checkbox__inner:hover) {
  border-color: #c9a227;
}

.modern-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.modern-table :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.modern-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner::after) {
  border-color: #ffffff;
  border-width: 2px;
}

.modern-table :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner::before) {
  background-color: #ffffff;
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

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
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

.cluster-name-link {
  color: #303133 !important;
  font-size: 14px;
  font-weight: 500;
}

.cluster-name-link:hover {
  color: #409eff !important;
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
  background: #111827 !important;
  color: #ffffff !important;
  border-color: #111827 !important;
  border-radius: 10px;
  padding: 10px 18px;
  font-weight: 600;
  box-shadow: 0 8px 20px rgba(17, 24, 39, 0.12);
}

.black-button:hover {
  background: #1f2937 !important;
  border-color: #1f2937 !important;
  transform: translateY(-1px);
}

.form-section {
  margin-bottom: 16px;
  padding: 18px 20px 4px;
  border: 1px solid #e5e9f2;
  border-radius: 14px;
  background: #ffffff;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.04);
}

.form-section:last-of-type {
  margin-bottom: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  letter-spacing: 0.2px;
}

.section-title-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #2563eb;
  box-shadow: 0 0 0 5px rgba(37, 99, 235, 0.1);
  flex-shrink: 0;
}

.code-editor-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid #d7deea;
  border-radius: 12px;
  overflow: hidden;
  background-color: #1f2937;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 12px 8px;
  background-color: #111827;
  border-right: 1px solid #374151;
  user-select: none;
  min-width: 40px;
  text-align: right;
}

.line-number {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #6b7280;
  min-height: 20.8px;
}

.code-textarea {
  flex: 1;
  min-height: 200px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d1d5db;
  background-color: #1f2937;
  border: none;
  outline: none;
  resize: vertical;
  font-feature-settings: "liga" 0;
}

.code-textarea::placeholder {
  color: #6b7280;
}

.code-textarea:focus {
  background-color: #1f2937;
  color: #e5e7eb;
}

.code-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 10px 12px;
  background-color: #f8fafc;
  border: 1px solid #e5e9f2;
  border-radius: 10px;
  font-size: 12px;
  color: #667085;
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

.cluster-dialog-hero,
.credential-overview {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  margin-bottom: 16px;
  border: 1px solid #e5e9f2;
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.07), rgba(255, 255, 255, 0.95)),
    #ffffff;
}

.cluster-dialog-hero-icon,
.credential-overview-icon {
  width: 46px;
  height: 46px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2563eb;
  background: #eff6ff;
  border: 1px solid #dbeafe;
  font-size: 22px;
  flex-shrink: 0;
}

.cluster-dialog-hero-title,
.credential-overview-title {
  font-size: 17px;
  line-height: 1.4;
  font-weight: 700;
  color: #111827;
}

.cluster-dialog-hero-desc,
.credential-overview-desc {
  margin-top: 4px;
  font-size: 13px;
  color: #667085;
}

.cluster-form :deep(.el-form-item__label) {
  color: #344054;
  font-weight: 600;
}

.cluster-form :deep(.el-input__wrapper),
.cluster-form :deep(.el-textarea__inner),
.cluster-form :deep(.el-select__wrapper) {
  border-radius: 10px;
  box-shadow: 0 0 0 1px #d7deea inset;
}

.cluster-form :deep(.el-input__wrapper:hover),
.cluster-form :deep(.el-textarea__inner:hover),
.cluster-form :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px #a9b9d5 inset;
}

.cluster-form :deep(.el-input__wrapper.is-focus),
.cluster-form :deep(.el-textarea__inner:focus),
.cluster-form :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px #2563eb inset, 0 0 0 3px rgba(37, 99, 235, 0.12);
}

.cluster-form :deep(.el-radio-button__inner) {
  border-color: #d7deea;
  color: #475467;
  font-weight: 600;
}

.cluster-form :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-radius: 10px 0 0 10px;
}

.cluster-form :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: 0 10px 10px 0;
}

.cluster-form :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background: #2563eb;
  border-color: #2563eb;
  color: #ffffff;
  box-shadow: -1px 0 0 0 #2563eb;
}

.upload-row {
  margin-bottom: 10px;
}

.upload-row .el-button,
.credential-toolbar-actions .el-button {
  border-radius: 9px;
  font-weight: 600;
}

.tls-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #667085;
  padding: 5px 10px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid #e5e9f2;
}

.cluster-submit-button {
  border-radius: 10px;
  font-weight: 700;
  padding: 10px 20px;
  background: #2563eb;
  border-color: #2563eb;
  box-shadow: 0 10px 20px rgba(37, 99, 235, 0.18);
}

.cluster-submit-button:hover {
  background: #1d4ed8;
  border-color: #1d4ed8;
  transform: translateY(-1px);
}

.credential-description {
  margin-bottom: 16px;
  border: 1px solid #e5e9f2;
  border-radius: 14px;
  overflow: hidden;
}

.credential-description :deep(.el-descriptions__body) {
  background: #ffffff;
}

.credential-description :deep(.el-descriptions__label) {
  width: 130px;
  background: #f8fafc !important;
  color: #475467;
  font-weight: 700;
}

.credential-description :deep(.el-descriptions__content) {
  color: #111827;
  font-weight: 500;
}

.credential-toolbar {
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.credential-toolbar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #111827;
}

.credential-toolbar-actions {
  display: flex;
  gap: 8px;
}

/* 授权对话框样式 */
.connection-info {
  padding: 20px;
}

.info-section {
  margin-bottom: 24px;
}

.connection-info .section-title {
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

/* 删除集群确认对话框样式 */
:deep(.delete-cluster-confirm) {
  .el-message-box__message {
    p {
      line-height: 1.8;
    }
  }

  .el-message-box__btns {
    .el-button--primary {
      background: #f56c6c;
      border-color: #f56c6c;

      &:hover {
        background: #f78989;
        border-color: #f78989;
      }

      &:active {
        background: #dd6161;
        border-color: #dd6161;
      }
    }
  }
}

/* 集群编辑对话框样式 */
:deep(.cluster-edit-dialog) {
  width: min(1080px, 92vw) !important;
}

:deep(.cluster-config-dialog) {
  width: min(900px, 92vw) !important;
}

:deep(.cluster-edit-dialog),
:deep(.cluster-config-dialog) {
  border-radius: 18px;
  overflow: hidden;
}

:deep(.cluster-edit-dialog .el-dialog__header),
:deep(.cluster-config-dialog .el-dialog__header) {
  padding: 22px 28px 16px;
  margin-right: 0;
  border-bottom: 1px solid #eef2f7;
}

:deep(.cluster-edit-dialog .el-dialog__title),
:deep(.cluster-config-dialog .el-dialog__title) {
  color: #111827;
  font-size: 20px;
  font-weight: 800;
}

:deep(.cluster-edit-dialog .el-dialog__body),
:deep(.cluster-config-dialog .el-dialog__body) {
  max-height: 70vh;
  overflow-y: auto;
  padding: 20px 28px 18px;
  background: #fbfcfe;
}

:deep(.cluster-edit-dialog .el-dialog__footer),
:deep(.cluster-config-dialog .el-dialog__footer) {
  padding: 16px 28px 20px;
  border-top: 1px solid #eef2f7;
  background: #ffffff;
}
</style>
