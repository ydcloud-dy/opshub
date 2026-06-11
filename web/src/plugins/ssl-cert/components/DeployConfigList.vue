<template>
  <div class="deploy-config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Upload /></el-icon>
        </div>
        <div>
          <h2 class="page-title">部署配置</h2>
          <p class="page-subtitle">配置证书自动部署到Nginx服务器或K8s集群</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增配置
        </el-button>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select
          v-model="searchForm.deploy_type"
          placeholder="部署类型"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="Nginx SSH" value="nginx_ssh" />
          <el-option label="K8s Secret" value="k8s_secret" />
        </el-select>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="tableData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="配置名称" prop="name" width="120" show-overflow-tooltip />

        <el-table-column label="关联证书" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.certificate">{{ row.certificate.name }} ({{ row.certificate.domain }})</span>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="部署类型" width="130" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.deploy_type === 'nginx_ssh'" type="primary">Nginx SSH</el-tag>
            <el-tag v-else type="success">K8s Secret</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="自动部署" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.auto_deploy" @change="handleAutoDeployChange(row)" />
          </template>
        </el-table-column>

        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.enabled" type="success">启用</el-tag>
            <el-tag v-else type="info">禁用</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="上次部署" min-width="180">
          <template #default="{ row }">
            <div v-if="row.last_deploy_at">
              <span>{{ formatDateTime(row.last_deploy_at) }}</span>
              <el-tag v-if="row.last_deploy_ok" type="success" size="small" style="margin-left: 8px;">成功</el-tag>
              <el-tag v-else type="danger" size="small" style="margin-left: 8px;">失败</el-tag>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="立即部署" placement="top">
                <el-button link class="action-btn action-deploy" @click="handleDeploy(row)" :loading="row.deploying">
                  <el-icon><Upload /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="测试配置" placement="top">
                <el-button link class="action-btn action-test" @click="handleTest(row)" :loading="row.testing">
                  <el-icon><Connection /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
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
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="680px"
      :close-on-click-modal="false"
      class="beauty-dialog"
      destroy-on-close
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px" class="beauty-form">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入配置名称" />
        </el-form-item>

        <el-form-item label="关联证书" prop="certificate_id">
          <el-select v-model="form.certificate_id" placeholder="请选择证书" style="width: 100%" filterable>
            <el-option
              v-for="cert in certificates"
              :key="cert.id"
              :label="`${cert.name} (${cert.domain})`"
              :value="cert.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="部署类型" prop="deploy_type">
          <el-select v-model="form.deploy_type" placeholder="请选择部署类型" style="width: 100%" :disabled="!!form.id">
            <el-option label="Nginx SSH" value="nginx_ssh" />
            <el-option label="K8s Secret" value="k8s_secret" />
          </el-select>
        </el-form-item>

        <!-- Nginx SSH配置 -->
        <template v-if="form.deploy_type === 'nginx_ssh'">
          <el-divider content-position="left">Nginx配置</el-divider>
          <el-form-item label="资产分组">
            <el-tree-select
              v-model="selectedGroupId"
              :data="assetGroups"
              :props="{ label: 'name', value: 'id', children: 'children' }"
              placeholder="请选择资产分组"
              style="width: 100%"
              clearable
              check-strictly
              :render-after-expand="false"
              @change="onGroupChange"
            />
          </el-form-item>
          <el-form-item label="目标主机" prop="target_config.host_id">
            <el-select
              v-model="form.target_config.host_id"
              placeholder="请先选择资产分组"
              style="width: 100%"
              filterable
              :loading="hostsLoading"
              :disabled="!selectedGroupId"
              clearable
            >
              <el-option
                v-for="host in hosts"
                :key="host.id"
                :label="`${host.name} (${host.ip})`"
                :value="host.id"
              >
                <div class="host-option">
                  <span class="host-name">{{ host.name }}</span>
                  <span class="host-ip">{{ host.ip }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="证书路径" prop="target_config.cert_path">
            <el-input v-model="form.target_config.cert_path" placeholder="/etc/nginx/ssl/cert.pem" />
          </el-form-item>
          <el-form-item label="私钥路径" prop="target_config.key_path">
            <el-input v-model="form.target_config.key_path" placeholder="/etc/nginx/ssl/key.pem" />
          </el-form-item>
          <el-form-item label="备份旧证书">
            <el-switch v-model="form.target_config.backup_enabled" />
            <span style="margin-left: 12px; color: #909399; font-size: 13px;">部署前备份原有证书文件</span>
          </el-form-item>
          <el-form-item label="备份路径" v-if="form.target_config.backup_enabled">
            <el-input v-model="form.target_config.backup_path" placeholder="留空则默认备份到证书目录下的backup文件夹" />
          </el-form-item>
        </template>

        <!-- K8s Secret配置 -->
        <template v-if="form.deploy_type === 'k8s_secret'">
          <el-divider content-position="left">K8s配置</el-divider>

          <!-- 容器插件不可用提示 -->
          <el-alert
            v-if="!k8sPluginAvailable"
            title="容器管理插件未启用"
            type="warning"
            :closable="false"
            show-icon
            style="margin-bottom: 16px;"
          >
            <template #default>
              该部署类型需要启用容器管理插件。请先在插件管理中启用 Kubernetes 插件，并添加集群配置。
            </template>
          </el-alert>

          <el-form-item label="K8s集群" prop="target_config.cluster_id">
            <el-select
              v-model="form.target_config.cluster_id"
              placeholder="请选择K8s集群"
              style="width: 100%"
              filterable
              :loading="k8sClustersLoading"
              :disabled="!k8sPluginAvailable"
              @change="onClusterChange"
            >
              <el-option
                v-for="cluster in k8sClusters"
                :key="cluster.id"
                :label="`${cluster.name}${cluster.alias ? ' (' + cluster.alias + ')' : ''}`"
                :value="cluster.id"
              >
                <div class="cluster-option">
                  <span class="cluster-name">{{ cluster.name }}</span>
                  <span class="cluster-alias">{{ cluster.alias || cluster.apiEndpoint }}</span>
                </div>
              </el-option>
            </el-select>
            <div class="form-tip">从容器管理中选择已配置的K8s集群</div>
          </el-form-item>

          <el-form-item label="命名空间" prop="target_config.namespace">
            <el-select
              v-model="form.target_config.namespace"
              placeholder="请先选择集群"
              style="width: 100%"
              filterable
              :loading="k8sNamespacesLoading"
              :disabled="!form.target_config.cluster_id"
              @change="onNamespaceChange"
            >
              <el-option
                v-for="ns in k8sNamespaces"
                :key="ns.name"
                :label="ns.name"
                :value="ns.name"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="Secret名称" prop="target_config.secret_name">
            <el-select
              v-model="form.target_config.secret_name"
              placeholder="请先选择命名空间"
              style="width: 100%"
              filterable
              allow-create
              default-first-option
              :loading="k8sSecretsLoading"
              :disabled="!form.target_config.namespace"
            >
              <el-option
                v-for="secret in k8sSecrets"
                :key="secret.name"
                :label="secret.name"
                :value="secret.name"
              >
                <div class="secret-option">
                  <span class="secret-name">{{ secret.name }}</span>
                  <span class="secret-type">{{ secret.type }}</span>
                </div>
              </el-option>
            </el-select>
            <div class="form-tip">可选择已有Secret或输入新名称创建，将以 kubernetes.io/tls 类型创建/更新</div>
          </el-form-item>

          <el-form-item label="触发滚动更新">
            <el-switch v-model="form.target_config.trigger_rollout" />
            <div class="form-tip">部署后触发关联Deployment滚动更新</div>
          </el-form-item>

          <el-form-item label="Deployment" v-if="form.target_config.trigger_rollout">
            <el-select
              v-model="form.target_config.deployments"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="选择或输入Deployment名称"
              style="width: 100%"
              :loading="k8sDeploymentsLoading"
            >
              <el-option
                v-for="deploy in k8sDeployments"
                :key="deploy.name"
                :label="deploy.name"
                :value="deploy.name"
              >
                <div class="deploy-option">
                  <span class="deploy-name">{{ deploy.name }}</span>
                  <span class="deploy-ready">{{ deploy.ready }}</span>
                </div>
              </el-option>
            </el-select>
            <div class="form-tip">可从列表选择或手动输入Deployment名称</div>
          </el-form-item>
        </template>

        <el-form-item label="自动部署">
          <el-switch v-model="form.auto_deploy" />
          <div class="form-tip">证书续期后自动部署</div>
        </el-form-item>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus, Refresh, Edit, Delete, Upload, Connection, Warning } from '@element-plus/icons-vue'
import {
  getDeployConfigs,
  createDeployConfig,
  updateDeployConfig,
  deleteDeployConfig,
  executeDeploy,
  testDeployConfig,
  getCertificates
} from '../api/ssl-cert'
import { getHostList, getHost } from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'
import { getClusterList, getNamespaces, getDeployments, getSecrets } from '@/api/kubernetes'
import type { Cluster, NamespaceInfo, DeploymentInfo, SecretInfo } from '@/api/kubernetes'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const formRef = ref<FormInstance>()
const hostsLoading = ref(false)

// 搜索
const searchForm = reactive({
  deploy_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 证书列表
const certificates = ref<any[]>([])

// 资产分组列表
const assetGroups = ref<any[]>([])

// 选中的分组ID
const selectedGroupId = ref<number | null>(null)

// 主机列表
const hosts = ref<any[]>([])

// K8s 相关
const k8sClusters = ref<Cluster[]>([])
const k8sNamespaces = ref<NamespaceInfo[]>([])
const k8sSecrets = ref<SecretInfo[]>([])
const k8sDeployments = ref<DeploymentInfo[]>([])
const k8sClustersLoading = ref(false)
const k8sNamespacesLoading = ref(false)
const k8sSecretsLoading = ref(false)
const k8sDeploymentsLoading = ref(false)
const k8sPluginAvailable = ref(true)

// 表单数据
const form = reactive({
  id: 0,
  name: '',
  certificate_id: null as number | null,
  deploy_type: '',
  target_config: {} as Record<string, any>,
  auto_deploy: true,
  enabled: true
})

// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  certificate_id: [{ required: true, message: '请选择证书', trigger: 'change' }],
  deploy_type: [{ required: true, message: '请选择部署类型', trigger: 'change' }],
  'target_config.host_id': [{ required: true, message: '请选择目标主机', trigger: 'change' }],
  'target_config.cert_path': [{ required: true, message: '请输入证书路径', trigger: 'blur' }],
  'target_config.key_path': [{ required: true, message: '请输入私钥路径', trigger: 'blur' }],
  'target_config.cluster_id': [{ required: true, message: '请选择K8s集群', trigger: 'change' }],
  'target_config.namespace': [{ required: true, message: '请选择命名空间', trigger: 'change' }],
  'target_config.secret_name': [{ required: true, message: '请选择或输入Secret名称', trigger: 'change' }]
}

// 监听部署类型变化，初始化 target_config 默认值
watch(() => form.deploy_type, (newType) => {
  // 只在新增模式下初始化默认值
  if (form.id === 0) {
    if (newType === 'nginx_ssh') {
      form.target_config = {
        host_id: null,
        cert_path: '',
        key_path: '',
        backup_enabled: false,
        backup_path: ''
      }
      selectedGroupId.value = null
      hosts.value = []
    } else if (newType === 'k8s_secret') {
      form.target_config = {
        cluster_id: null,
        namespace: '',
        secret_name: '',
        trigger_rollout: false,
        deployments: []
      }
      // 加载 K8s 集群列表
      loadK8sClusters()
    }
  }
})

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getDeployConfigs({
      page: pagination.page,
      page_size: pagination.pageSize,
      deploy_type: searchForm.deploy_type || undefined
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 加载证书列表
const loadCertificates = async () => {
  try {
    const res = await getCertificates({ page: 1, page_size: 1000 })
    certificates.value = res.list || []
  } catch (error) {
    // ignore
  }
}

// 加载资产分组（保持树形结构）
const loadAssetGroups = async () => {
  try {
    const res = await getGroupTree()
    assetGroups.value = res.data || res || []
  } catch (error) {
    // ignore
  }
}

// 当分组改变时加载该分组下的主机
const onGroupChange = async (groupId: number | null, resetHostId: boolean = true) => {
  hosts.value = []
  if (resetHostId) {
    form.target_config.host_id = null
  }
  if (!groupId) return

  hostsLoading.value = true
  try {
    const res = await getHostList({ page: 1, page_size: 500, groupId })
    hosts.value = res.data?.list || res.list || []
  } catch (error) {
    // ignore
  } finally {
    hostsLoading.value = false
  }
}

// 加载单个主机信息（编辑时回显用）
const loadHostById = async (hostId: number) => {
  if (!hostId) return
  try {
    const res = await getHost(hostId)
    const hostData = res.data || res
    if (hostData && hostData.id) {
      // 设置选中的分组
      if (hostData.groupId) {
        selectedGroupId.value = hostData.groupId
        // 加载该分组下的主机，不重置 host_id
        await onGroupChange(hostData.groupId, false)
      } else {
        // 如果主机没有分组，直接添加到列表中
        hosts.value = [hostData]
      }
    }
  } catch (error) {
    // ignore
  }
}

// 加载 K8s 集群列表
const loadK8sClusters = async () => {
  k8sClustersLoading.value = true
  k8sPluginAvailable.value = true
  try {
    const res = await getClusterList()
    // 处理可能的 AxiosResponse 包装
    k8sClusters.value = (res as any)?.data || res || []
  } catch (error: any) {
    k8sClusters.value = []
    // 如果是 404 或其他错误，说明容器插件可能未启用
    if (error?.response?.status === 404 || error?.message?.includes('404')) {
      k8sPluginAvailable.value = false
    }
  } finally {
    k8sClustersLoading.value = false
  }
}

// 当选择集群时加载命名空间
const onClusterChange = async (clusterId: number | null) => {
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  form.target_config.namespace = ''
  form.target_config.secret_name = ''
  form.target_config.deployments = []

  if (!clusterId) return

  k8sNamespacesLoading.value = true
  try {
    const res = await getNamespaces(clusterId)
    k8sNamespaces.value = (res as any)?.data || res || []
  } catch (error) {
    k8sNamespaces.value = []
  } finally {
    k8sNamespacesLoading.value = false
  }
}

// 当选择命名空间时加载 Secrets 和 Deployments
const onNamespaceChange = async (namespace: string) => {
  k8sSecrets.value = []
  k8sDeployments.value = []
  form.target_config.secret_name = ''
  form.target_config.deployments = []

  if (!namespace || !form.target_config.cluster_id) return

  // 并行加载 Secrets 和 Deployments
  k8sSecretsLoading.value = true
  k8sDeploymentsLoading.value = true

  try {
    const [secretsRes, deploymentsRes] = await Promise.all([
      getSecrets(form.target_config.cluster_id, namespace),
      getDeployments(form.target_config.cluster_id, namespace)
    ])
    k8sSecrets.value = (secretsRes as any)?.data || secretsRes || []
    k8sDeployments.value = (deploymentsRes as any)?.data || deploymentsRes || []
  } catch (error) {
    k8sSecrets.value = []
    k8sDeployments.value = []
  } finally {
    k8sSecretsLoading.value = false
    k8sDeploymentsLoading.value = false
  }
}

// 编辑时加载 K8s 相关数据
const loadK8sDataForEdit = async (clusterId: number, namespace: string) => {
  // 先加载集群列表
  await loadK8sClusters()

  // 再加载命名空间
  if (clusterId) {
    k8sNamespacesLoading.value = true
    try {
      const res = await getNamespaces(clusterId)
      k8sNamespaces.value = (res as any)?.data || res || []
    } catch (error) {
      k8sNamespaces.value = []
    } finally {
      k8sNamespacesLoading.value = false
    }
  }

  // 加载 Secrets 和 Deployments
  if (clusterId && namespace) {
    k8sSecretsLoading.value = true
    k8sDeploymentsLoading.value = true
    try {
      const [secretsRes, deploymentsRes] = await Promise.all([
        getSecrets(clusterId, namespace),
        getDeployments(clusterId, namespace)
      ])
      k8sSecrets.value = (secretsRes as any)?.data || secretsRes || []
      k8sDeployments.value = (deploymentsRes as any)?.data || deploymentsRes || []
    } catch (error) {
      k8sSecrets.value = []
      k8sDeployments.value = []
    } finally {
      k8sSecretsLoading.value = false
      k8sDeploymentsLoading.value = false
    }
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增部署配置'
  hosts.value = []
  selectedGroupId.value = null
  k8sClusters.value = []
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  k8sPluginAvailable.value = true
  Object.assign(form, {
    id: 0,
    name: '',
    certificate_id: null,
    deploy_type: '',
    target_config: {},
    auto_deploy: true,
    enabled: true
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogTitle.value = '编辑部署配置'
  hosts.value = []
  selectedGroupId.value = null
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  const targetConfig = row.target_config ? JSON.parse(row.target_config) : {}
  Object.assign(form, {
    id: row.id,
    name: row.name,
    certificate_id: row.certificate_id,
    deploy_type: row.deploy_type,
    target_config: targetConfig,
    auto_deploy: row.auto_deploy,
    enabled: row.enabled
  })
  // 如果是 nginx_ssh 类型且有 host_id，加载主机信息以便回显
  if (row.deploy_type === 'nginx_ssh' && targetConfig.host_id) {
    loadHostById(targetConfig.host_id)
  }
  // 如果是 k8s_secret 类型，加载 K8s 相关数据以便回显
  if (row.deploy_type === 'k8s_secret') {
    loadK8sDataForEdit(targetConfig.cluster_id, targetConfig.namespace)
  }
  dialogVisible.value = true
}

// 提交
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        if (form.id) {
          await updateDeployConfig(form.id, {
            name: form.name,
            target_config: form.target_config,
            auto_deploy: form.auto_deploy,
            enabled: form.enabled
          })
          ElMessage.success('保存成功')
          dialogVisible.value = false
          loadData()
        } else {
          await createDeployConfig({
            name: form.name,
            certificate_id: form.certificate_id!,
            deploy_type: form.deploy_type,
            target_config: form.target_config,
            auto_deploy: form.auto_deploy,
            enabled: form.enabled
          })
          ElMessage.success('创建成功')
          dialogVisible.value = false
          loadData()
        }
      } catch (error: any) {
        // 错误已由 request 拦截器处理
      } finally {
        submitting.value = false
      }
    }
  })
}

// 自动部署切换
const handleAutoDeployChange = async (row: any) => {
  try {
    await updateDeployConfig(row.id, { auto_deploy: row.auto_deploy })
    ElMessage.success('更新成功')
  } catch (error) {
    row.auto_deploy = !row.auto_deploy
  }
}

// 立即部署
const handleDeploy = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要立即部署证书吗？', '提示', { type: 'warning' })
    row.deploying = true
    await executeDeploy(row.id)
    ElMessage.success('部署成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      // 错误已由 request 拦截器处理
    }
  } finally {
    row.deploying = false
  }
}

// 测试配置
const handleTest = async (row: any) => {
  try {
    row.testing = true
    await testDeployConfig(row.id)
    ElMessage.success('配置测试成功')
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    row.testing = false
  }
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该部署配置吗？', '提示', { type: 'warning' })
    loading.value = true
    await deleteDeployConfig(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      // 错误已由 request 拦截器处理
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
  loadCertificates()
  loadAssetGroups()
})
</script>

<style scoped>
.deploy-config-container {
  padding: 0;
  background-color: transparent;
}

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
  background: #f8fafc;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #111827;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-inputs {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 200px;
}

.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-deploy:hover {
  background-color: #e8f5e9;
  color: #67c23a;
}

.action-test:hover {
  background-color: #fff3e0;
  color: #e6a23c;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.pagination-wrapper {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 6px;
  line-height: 1.5;
}

.host-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.host-name {
  color: #303133;
}

.host-ip {
  color: #909399;
  font-size: 12px;
}

.cluster-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.cluster-name {
  color: #303133;
}

.cluster-alias {
  color: #909399;
  font-size: 12px;
}

.secret-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.secret-name {
  color: #303133;
}

.secret-type {
  color: #909399;
  font-size: 11px;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.deploy-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.deploy-name {
  color: #303133;
}

.deploy-ready {
  color: #67c23a;
  font-size: 12px;
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

/* 表单美化 */
:deep(.beauty-form .el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.beauty-form .el-input__wrapper),
:deep(.beauty-form .el-textarea__inner) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-form .el-input__wrapper:hover),
:deep(.beauty-form .el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset;
}

:deep(.beauty-form .el-input__wrapper.is-focus),
:deep(.beauty-form .el-textarea__inner:focus) {
  box-shadow: 0 0 0 1px #000 inset;
}

:deep(.beauty-form .el-select .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #000 inset !important;
}

:deep(.beauty-form .el-divider__text) {
  font-size: 13px;
  font-weight: 600;
  color: #909399;
  background: #fff;
}
</style>
