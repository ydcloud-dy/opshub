<template>
  <div class="role-management">
    <!-- 左侧侧边栏 -->
    <div class="sidebar">
      <div class="sidebar-header">
        <div class="title">角色管理</div>
      </div>

      <div class="sidebar-menu">
        <div
          class="menu-item"
          :class="{ active: activeTab === 'cluster' }"
          @click="activeTab = 'cluster'"
        >
          <el-icon><Key /></el-icon>
          <span>集群角色</span>
        </div>

        <div
          class="menu-item"
          :class="{ active: activeTab === 'namespace' }"
          @click="activeTab = 'namespace'"
        >
          <el-icon><FolderOpened /></el-icon>
          <span>命名空间角色</span>
        </div>
      </div>
    </div>

    <!-- 右侧内容区 -->
    <div class="content">
      <div class="content-header">
        <div class="header-info">
          <div class="header-title">
            <el-icon v-if="activeTab === 'cluster'"><Key /></el-icon>
            <el-icon v-else><FolderOpened /></el-icon>
            <span v-if="activeTab === 'cluster'">集群角色</span>
            <span v-else>命名空间角色</span>
          </div>
          <el-tag v-if="activeTab === 'cluster'" type="danger" size="small">
            {{ allClusterRoles.length }} / 14
          </el-tag>
          <el-tag v-else type="primary" size="small">
            {{ allNamespaceRoles.length }} / 12
          </el-tag>
        </div>
        <div class="header-actions">
          <el-button type="primary" @click="handleCreateRole" :loading="loading">
            <el-icon><Plus /></el-icon>
            添加角色
          </el-button>
          <el-button @click="loadRoles" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <!-- 集群角色列表 -->
      <div v-if="activeTab === 'cluster'" class="role-list" v-loading="loading">
        <div v-if="allClusterRoles.length > 0" class="role-grid">
          <div
            v-for="role in allClusterRoles"
            :key="role.name"
            class="role-card"
            @click="handleViewRoleDetail('', role.name)"
          >
            <div class="role-header">
              <div class="role-name">{{ role.name }}</div>
              <div class="role-actions">
                <el-button
                  link
                  type="primary"
                  size="small"
                  @click.stop="handleViewRoleDetail('', role.name)"
                >
                  <el-icon><View /></el-icon>
                  查看
                </el-button>
                <el-button
                  v-if="role.isCustom"
                  link
                  type="danger"
                  size="small"
                  @click.stop="handleDeleteRole('', role.name)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
            <div class="role-meta">
              <div class="meta-item">
                <span class="label">创建时间:</span>
                <span class="value">{{ role.age || '-' }}</span>
              </div>
              <div class="meta-item">
                <span class="label">规则数:</span>
                <span class="value">{{ role.rules?.length || 0 }}</span>
              </div>
            </div>
          </div>
        </div>
        <el-empty v-else description="暂无集群角色，点击左下角按钮创建" :image-size="100" />
      </div>

      <!-- 命名空间角色列表（显示所有命名空间的角色定义） -->
      <div v-else class="role-list" v-loading="loading">
        <div v-if="allNamespaceRoles.length > 0" class="role-grid">
          <div
            v-for="role in allNamespaceRoles"
            :key="role.name"
            class="role-card"
            @click="handleViewRoleDetail(sampleNamespace, role.name)"
          >
            <div class="role-header">
              <div class="role-name">{{ role.name }}</div>
              <div class="role-actions">
                <el-button
                  link
                  type="primary"
                  size="small"
                  @click.stop="handleViewRoleDetail(sampleNamespace, role.name)"
                >
                  <el-icon><View /></el-icon>
                  查看
                </el-button>
                <el-button
                  v-if="role.isCustom"
                  link
                  type="danger"
                  size="small"
                  @click.stop="handleDeleteNamespaceRole(role.name)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
            <div class="role-meta">
              <div class="meta-item">
                <span class="label">创建时间:</span>
                <span class="value">{{ role.age || '-' }}</span>
              </div>
              <div class="meta-item">
                <span class="label">规则数:</span>
                <span class="value">{{ role.rules?.length || 0 }}</span>
              </div>
            </div>
          </div>
        </div>
        <el-empty v-else description="暂无命名空间角色，点击左下角按钮创建" :image-size="100" />
      </div>
    </div>

    <!-- 创建角色对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      :title="activeTab === 'cluster' ? '创建集群角色' : '创建命名空间角色'"
      width="700px"
      @close="handleCloseCreateDialog"
      class="create-role-dialog"
    >
      <div class="create-role-content">
        <!-- 角色名称 -->
        <div class="form-section">
          <div class="section-title">
            <el-icon><Edit /></el-icon>
            <span>基本信息</span>
          </div>
          <div class="form-item">
            <label>角色名称</label>
            <el-input
              v-model="createForm.name"
              placeholder="请输入角色名称，如: pod-reader"
              clearable
            >
              <template #prefix>
                <el-icon><Key /></el-icon>
              </template>
            </el-input>
          </div>
        </div>

        <el-divider style="margin: 20px 0;" />

        <!-- API 组选择 -->
        <div class="form-section">
          <div class="section-title">
            <el-icon><Connection /></el-icon>
            <span>API 组</span>
            <el-tag size="small" type="info">步骤 1</el-tag>
          </div>
          <div class="form-item">
            <label>选择 API 组</label>
            <el-select
              v-model="createForm.apiGroups"
              multiple
              filterable
              allow-create
              placeholder="请先选择 API 组"
              style="width: 100%"
              @change="handleApiGroupsChange"
            >
              <el-option
                v-for="group in availableAPIGroups"
                :key="group"
                :label="group || 'core (空字符串)'"
                :value="group"
              />
            </el-select>
          </div>
        </div>

        <!-- 资源选择 -->
        <div class="form-section" :class="{ disabled: createForm.apiGroups.length === 0 }">
          <div class="section-title">
            <el-icon><Folder /></el-icon>
            <span>资源</span>
            <el-tag size="small" :type="createForm.apiGroups.length > 0 ? 'info' : 'info'">步骤 2</el-tag>
          </div>
          <div class="form-item">
            <label>选择资源</label>
            <el-select
              v-model="createForm.resources"
              multiple
              filterable
              allow-create
              :disabled="createForm.apiGroups.length === 0"
              :placeholder="createForm.apiGroups.length === 0 ? '请先选择 API 组' : '请选择资源'"
              style="width: 100%"
              @change="handleResourcesChange"
            >
              <el-option
                v-for="res in availableResources"
                :key="res"
                :label="res"
                :value="res"
              />
            </el-select>
            <div v-if="availableResources.length > 0" class="form-tip">
              已加载 {{ availableResources.length }} 个资源类型
            </div>
          </div>
        </div>

        <!-- 操作选择 -->
        <div class="form-section" :class="{ disabled: createForm.resources.length === 0 }">
          <div class="section-title">
            <el-icon><Setting /></el-icon>
            <span>操作权限</span>
            <el-tag size="small" :type="createForm.resources.length > 0 ? 'info' : 'info'">步骤 3</el-tag>
          </div>
          <div class="form-item">
            <label>选择操作</label>
            <el-select
              v-model="createForm.verbs"
              multiple
              :disabled="createForm.resources.length === 0"
              :placeholder="createForm.resources.length === 0 ? '请先选择资源' : '请选择操作'"
              style="width: 100%"
            >
              <el-option label="get - 查看单个资源" value="get" />
              <el-option label="list - 列出资源" value="list" />
              <el-option label="watch - 监听资源" value="watch" />
              <el-option label="create - 创建资源" value="create" />
              <el-option label="update - 更新资源" value="update" />
              <el-option label="patch - 部分更新" value="patch" />
              <el-option label="delete - 删除资源" value="delete" />
              <el-option label="deletecollection - 批量删除" value="deletecollection" />
              <el-option label="* - 所有权限" value="*" />
            </el-select>
          </div>
        </div>

        <!-- 预览规则 -->
        <div v-if="createForm.resources.length > 0 && createForm.verbs.length > 0" class="rule-preview">
          <div class="preview-title">
            <el-icon><View /></el-icon>
            <span>权限规则预览</span>
          </div>
          <div class="preview-content">
            <div class="preview-item">
              <span class="preview-label">API 组:</span>
              <el-tag v-for="api in createForm.apiGroups" :key="api" size="small" type="primary">
                {{ api || 'core' }}
              </el-tag>
            </div>
            <div class="preview-item">
              <span class="preview-label">资源:</span>
              <el-tag v-for="res in createForm.resources" :key="res" size="small" type="success">
                {{ res }}
              </el-tag>
            </div>
            <div class="preview-item">
              <span class="preview-label">操作:</span>
              <el-tag v-for="verb in createForm.verbs" :key="verb" size="small" :type="getVerbType(verb)">
                {{ verb }}
              </el-tag>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialogVisible = false" size="large">取消</el-button>
          <el-button
            type="primary"
            @click="handleSaveRole"
            :loading="loading"
            :disabled="!canSubmit"
            size="large"
          >
            <el-icon><Check /></el-icon>
            创建角色
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 角色详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`角色详情: ${currentRoleDetail?.name || ''}`"
      width="900px"
      @close="detailDialogVisible = false"
    >
      <div v-if="currentRoleDetail" class="role-detail">
        <!-- 基本信息 -->
        <div class="detail-header">
          <div class="detail-info">
            <el-icon :class="activeTab === 'cluster' ? 'icon-cluster' : 'icon-namespace'"><Key /></el-icon>
            <span class="role-type">{{ activeTab === 'cluster' ? 'ClusterRole' : 'Role' }}</span>
          </div>
          <div class="detail-info">
            <el-icon><Clock /></el-icon>
            <span class="value">{{ currentRoleDetail.age || '-' }}</span>
          </div>
          <div class="detail-info">
            <el-icon><Document /></el-icon>
            <span class="value">{{ currentRoleDetail.rules?.length || 0 }} 条规则</span>
          </div>
        </div>

        <el-divider style="margin: 16px 0;" />

        <!-- 权限规则 -->
        <div class="rules-section">
          <div class="rules-title">
            <el-icon><List /></el-icon>
            权限规则
          </div>

          <div class="rules-list">
            <div
              v-for="(rule, index) in currentRoleDetail.rules"
              :key="index"
              class="rule-item"
            >
              <div class="rule-main">
                <div class="rule-api">
                  <span class="api-label">API:</span>
                  <span class="api-value">{{ formatApiGroups(rule.apiGroups) }}</span>
                </div>

                <div class="rule-resources">
                  <div class="resource-group">
                    <span class="res-label">资源:</span>
                    <div class="res-tags">
                      <el-tag
                        v-for="res in rule.resources"
                        :key="res"
                        size="small"
                        type="success"
                        effect="plain"
                      >
                        {{ res }}
                      </el-tag>
                    </div>
                  </div>
                  <div class="resource-group">
                    <span class="res-label">名称:</span>
                    <div class="res-tags">
                      <el-tag
                        v-for="name in rule.resourceNames"
                        :key="name"
                        size="small"
                        type="info"
                        effect="plain"
                      >
                        {{ name }}
                      </el-tag>
                      <span v-if="!rule.resourceNames || rule.resourceNames.length === 0" class="empty-text">全部</span>
                    </div>
                  </div>
                </div>

                <div class="rule-verbs">
                  <span class="verb-label">操作:</span>
                  <div class="verb-tags">
                    <el-tag
                      v-for="verb in rule.verbs"
                      :key="verb"
                      size="small"
                      :type="getVerbType(verb)"
                    >
                      {{ formatVerb(verb) }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  Plus,
  Key,
  FolderOpened,
  Delete,
  View,
  Clock,
  Document,
  List,
  Edit,
  Connection,
  Folder,
  Setting,
  Check
} from '@element-plus/icons-vue'
import {
  getClusterRoles,
  getNamespacesForRoles,
  getNamespaceRoles,
  getRoleDetail,
  createDefaultClusterRoles,
  createDefaultNamespaceRoles,
  deleteRole,
  createRole,
  getAPIGroups,
  getResourcesByAPIGroup,
  type Cluster,
  type Role
} from '@/api/kubernetes'

interface Props {
  cluster: Cluster | null
}

const props = defineProps<Props>()

const loading = ref(false)
const activeTab = ref<'cluster' | 'namespace'>('cluster')
const detailDialogVisible = ref(false)
const createDialogVisible = ref(false)
const currentRoleDetail = ref<Role | null>(null)

// 所有角色数据
const allClusterRoles = ref<Role[]>([])
const allNamespaceRoles = ref<Role[]>([])
const sampleNamespace = ref('default')
const availableAPIGroups = ref<string[]>([])
const availableResources = ref<string[]>([])

// 创建角色表单
const createForm = ref({
  name: '',
  apiGroups: [] as string[],
  resources: [] as string[],
  resourceNames: [] as string[],
  verbs: [] as string[]
})

// 是否可以提交
const canSubmit = computed(() => {
  return createForm.value.name &&
         createForm.value.apiGroups.length > 0 &&
         createForm.value.resources.length > 0 &&
         createForm.value.verbs.length > 0
})

// API组变化处理
const handleApiGroupsChange = async () => {
  // 清空后续选项
  createForm.value.resources = []
  createForm.value.verbs = []
  availableResources.value = []

  if (!props.cluster || createForm.value.apiGroups.length === 0) {
    return
  }

  try {
    // 获取所有选择的API组的资源
    const resources = await getResourcesByAPIGroup(props.cluster.id, createForm.value.apiGroups)
    availableResources.value = resources || []
  } catch (error) {
    console.error('Failed to load resources:', error)
  }
}

// 资源变化处理
const handleResourcesChange = () => {
  // 清空操作选项
  createForm.value.verbs = []
}

// 加载所有角色
const loadRoles = async () => {
  if (!props.cluster) return
  loading.value = true
  try {
    // 加载所有集群角色
    let allClusterRolesList = await getClusterRoles(props.cluster.id)

    // 分离集群角色和命名空间角色（都是 ClusterRole，通过标签区分）
    allClusterRoles.value = (allClusterRolesList || []).filter(role =>
      !role.labels || role.labels['opshub.ydcloud-dy.com/namespace-role'] !== 'true'
    )

    // 命名空间角色：带有 namespace-role=true 标签的 ClusterRole
    allNamespaceRoles.value = (allClusterRolesList || []).filter(role =>
      role.labels && role.labels['opshub.ydcloud-dy.com/namespace-role'] === 'true'
    )

    // 加载可用的API组
    const apiGroups = await getAPIGroups(props.cluster.id)
    availableAPIGroups.value = apiGroups || []
  } catch (error) {
    console.error('Failed to load roles:', error)
    ElMessage.error('加载角色失败')
  } finally {
    loading.value = false
  }
}

// 查看角色详情
const handleViewRoleDetail = async (namespace: string, roleName: string) => {
  if (!props.cluster) return

  try {
    loading.value = true
    const detail = await getRoleDetail(props.cluster.id, namespace, roleName)
    currentRoleDetail.value = detail
    detailDialogVisible.value = true
  } catch (error) {
    console.error('Failed to load role detail:', error)
    ElMessage.error('加载角色详情失败')
  } finally {
    loading.value = false
  }
}

// 创建默认角色
const handleCreateRoles = async () => {
  if (!props.cluster) return

  try {
    await ElMessageBox.confirm(
      '确定要创建所有默认角色吗？这将为集群和所有命名空间创建标准角色。',
      '创建默认角色',
      { type: 'info' }
    )

    loading.value = true

    // 创建默认集群角色
    await createDefaultClusterRoles(props.cluster.id)
    ElMessage.success('集群角色创建成功')

    // 创建默认命名空间角色
    const nsList = await getNamespacesForRoles(props.cluster.id)
    for (const ns of nsList) {
      await createDefaultNamespaceRoles(props.cluster.id, ns.name)
    }
    ElMessage.success(`已为 ${nsList.length} 个命名空间创建角色`)

    // 重新加载
    await loadRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '创建失败')
    }
  } finally {
    loading.value = false
  }
}

// 打开创建角色对话框
const handleCreateRole = () => {
  if (!props.cluster) return
  createDialogVisible.value = true
}

// 关闭创建角色对话框
const handleCloseCreateDialog = () => {
  createForm.value = {
    name: '',
    apiGroups: [],
    resources: [],
    resourceNames: [],
    verbs: []
  }
}

// 保存角色
const handleSaveRole = async () => {
  if (!props.cluster) return

  // 验证表单
  if (!createForm.value.name) {
    ElMessage.warning('请输入角色名称')
    return
  }
  if (createForm.value.apiGroups.length === 0) {
    ElMessage.warning('请选择 API 组')
    return
  }
  if (createForm.value.resources.length === 0) {
    ElMessage.warning('请选择资源')
    return
  }
  if (createForm.value.verbs.length === 0) {
    ElMessage.warning('请选择操作')
    return
  }

  loading.value = true
  try {
    const namespace = activeTab.value === 'cluster' ? '' : sampleNamespace.value

    // 调用创建角色 API
    await createRole(props.cluster.id, {
      namespace: namespace,
      name: createForm.value.name,
      rules: [{
        apiGroups: createForm.value.apiGroups,
        resources: createForm.value.resources,
        resourceNames: [],
        verbs: createForm.value.verbs
      }]
    })

    ElMessage.success('创建成功')
    createDialogVisible.value = false
    handleCloseCreateDialog()
    await loadRoles()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '创建失败')
  } finally {
    loading.value = false
  }
}

// 删除集群角色
const handleDeleteRole = async (namespace: string, roleName: string) => {
  if (!props.cluster) return

  try {
    await ElMessageBox.confirm(
      `确定要删除集群角色 "${roleName}" 吗？此操作不可恢复！`,
      '删除角色',
      { type: 'warning' }
    )

    loading.value = true
    await deleteRole(props.cluster.id, namespace, roleName)
    ElMessage.success('删除成功')
    await loadRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  } finally {
    loading.value = false
  }
}

// 删除命名空间角色（需要从所有命名空间中删除）
const handleDeleteNamespaceRole = async (roleName: string) => {
  if (!props.cluster) return

  try {
    await ElMessageBox.confirm(
      `确定要从所有命名空间中删除角色 "${roleName}" 吗？此操作不可恢复！`,
      '删除角色',
      { type: 'warning' }
    )

    loading.value = true

    // 获取所有命名空间
    const nsList = await getNamespacesForRoles(props.cluster.id)

    // 从所有命名空间中删除该角色
    for (const ns of nsList) {
      try {
        await deleteRole(props.cluster.id, ns.name, roleName)
      } catch (error) {
        // 忽略单个命名空间的删除失败
        console.log(`Failed to delete role from ${ns.name}:`, error)
      }
    }

    ElMessage.success('删除成功')
    await loadRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  } finally {
    loading.value = false
  }
}

// 格式化 API 组
const formatApiGroups = (groups: string[]) => {
  if (!groups || groups.length === 0) return 'core'
  return groups.map(g => g || 'core').join(', ')
}

// 格式化操作动词
const formatVerb = (verb: string) => {
  return verb
}

// 获取操作标签类型
const getVerbType = (verb: string) => {
  const typeMap: Record<string, string> = {
    '*': 'danger',
    'get': '',
    'list': '',
    'watch': '',
    'create': 'success',
    'update': 'warning',
    'patch': 'warning',
    'delete': 'danger',
    'deletecollection': 'danger'
  }
  return typeMap[verb] || 'info'
}

// 初始化
watch(() => props.cluster, (newCluster) => {
  if (newCluster) {
    loadRoles()
  }
}, { immediate: true })
</script>

<style scoped lang="scss">
.role-management {
  display: flex;
  height: 600px;
  gap: 0;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #e4e7ed;
}

/* 左侧侧边栏 */
.sidebar {
  width: 240px;
  background: #fafbfc;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;

  .title {
    font-size: 16px;
    font-weight: 600;
    color: #303133;
  }
}

.sidebar-menu {
  flex: 1;
  padding: 12px 0;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #606266;
  font-size: 14px;

  .el-icon {
    font-size: 16px;
    color: #909399;
  }

  &:hover {
    background: rgba(212, 175, 55, 0.1);
    color: #d4af37;

    .el-icon {
      color: #d4af37;
    }
  }

  &.active {
    background: linear-gradient(90deg, rgba(212, 175, 55, 0.15) 0%, rgba(212, 175, 55, 0.05) 100%);
    color: #000;
    border-left: 3px solid #d4af37;
    padding-left: 17px;

    .el-icon {
      color: #d4af37;
    }
  }
}

/* 右侧内容区 */
.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #e4e7ed;

  .header-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .header-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 18px;
      font-weight: 600;
      color: #303133;

      .el-icon {
        color: #d4af37;
        font-size: 20px;
      }
    }
  }

  .header-actions {
    display: flex;
    gap: 12px;

    /* 黑金风格按钮 */
    :deep(.el-button--primary) {
      background: linear-gradient(135deg, #000000 0%, #2a2a2a 100%);
      border: 1px solid #d4af37;
      color: #d4af37;

      &:hover {
        background: linear-gradient(135deg, #1a1a1a 0%, #3a3a3a 100%);
        border-color: #f0c14b;
        color: #f0c14b;
        box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
      }

      &:active {
        background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
      }
    }

    :deep(.el-button:not(.el-button--primary)) {
      background: #fff;
      border: 1px solid rgba(212, 175, 55, 0.3);
      color: #606266;

      &:hover {
        border-color: #d4af37;
        color: #d4af37;
        background: rgba(212, 175, 55, 0.05);
      }
    }
  }
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

/* 创建角色对话框样式 */
.create-role-dialog {
  .create-role-content {
    padding: 0;
  }

  .form-section {
    margin-bottom: 20px;

    &.disabled {
      opacity: 0.5;
      pointer-events: none;
    }
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 15px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 12px;

    .el-icon {
      color: #d4af37;
      font-size: 18px;
    }

    .el-tag {
      margin-left: auto;
    }
  }

  .form-item {
    label {
      display: block;
      font-size: 13px;
      color: #606266;
      font-weight: 500;
      margin-bottom: 8px;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select) {
      border-radius: 6px;
    }
  }

  .rule-preview {
    background: linear-gradient(135deg, #f5f7fa 0%, rgba(212, 175, 55, 0.08) 100%);
    border-radius: 8px;
    padding: 16px;
    border: 1px solid rgba(212, 175, 55, 0.3);
    margin-top: 20px;

    .preview-title {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 14px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 12px;

      .el-icon {
        color: #d4af37;
      }
    }

    .preview-content {
      display: flex;
      flex-direction: column;
      gap: 10px;
    }

    .preview-item {
      display: flex;
      align-items: center;
      gap: 10px;

      .preview-label {
        font-size: 13px;
        color: #606266;
        font-weight: 500;
        min-width: 70px;
      }

      .el-tag {
        margin-right: 4px;
      }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;

    /* 黑金风格按钮 */
    :deep(.el-button--primary) {
      background: linear-gradient(135deg, #000000 0%, #2a2a2a 100%);
      border: 1px solid #d4af37;
      color: #d4af37;
      font-weight: 500;

      &:hover {
        background: linear-gradient(135deg, #1a1a1a 0%, #3a3a3a 100%);
        border-color: #f0c14b;
        color: #f0c14b;
        box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
      }

      &:active {
        background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
      }

      &:disabled {
        background: #909399;
        border-color: #909399;
        color: #fff;
        opacity: 0.6;
      }
    }

    :deep(.el-button:not(.el-button--primary)) {
      background: #fff;
      border: 1px solid rgba(212, 175, 55, 0.3);
      color: #606266;

      &:hover {
        border-color: #d4af37;
        color: #d4af37;
        background: rgba(212, 175, 55, 0.05);
      }
    }
  }
}

.role-list {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.role-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.role-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;

  &:hover {
    border-color: #d4af37;
    box-shadow: 0 2px 12px rgba(212, 175, 55, 0.15);
  }
}

.role-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;

  .role-name {
    font-size: 15px;
    font-weight: 600;
    color: #303133;
  }

  .role-actions {
    display: flex;
    gap: 8px;
    opacity: 0;
    transition: opacity 0.2s ease;

    /* 黑金风格按钮 */
    :deep(.el-button--primary) {
      background: transparent;
      border: none;
      color: #d4af37;
      padding: 4px 8px;

      &:hover {
        color: #f0c14b;
        background: rgba(212, 175, 55, 0.1);
      }
    }

    :deep(.el-button--danger) {
      background: transparent;
      border: none;
      color: #f56c6c;
      padding: 4px 8px;

      &:hover {
        color: #ff5252;
        background: rgba(245, 108, 108, 0.1);
      }
    }
  }
}

.role-card:hover .role-actions {
  opacity: 1;
}

.role-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .meta-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;

    .label {
      color: #909399;
    }

    .value {
      color: #303133;
      font-weight: 500;
    }
  }
}

/* 角色详情对话框 */
.role-detail {
  .detail-header {
    display: flex;
    gap: 32px;

    .detail-info {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;

      .label {
        color: #606266;
        font-weight: 500;
      }

      .value {
        color: #303133;
      }

      .el-icon {
        color: #d4af37;
        font-size: 16px;
      }

      .icon-cluster {
        color: #f56c6c;
      }

      .icon-namespace {
        color: #409eff;
      }

      .role-type {
        color: #303133;
        font-weight: 600;
      }
    }
  }

  .rules-section {
    margin-top: 20px;

    .rules-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 15px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 16px;

      .el-icon {
        color: #d4af37;
        font-size: 18px;
      }
    }
  }

  .rules-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .rule-item {
    background: #f5f7fa;
    border-radius: 6px;
    padding: 12px;
    border: 1px solid #e4e7ed;
  }

  .rule-main {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .rule-api {
    display: flex;
    align-items: center;
    gap: 6px;

    .api-label {
      font-size: 12px;
      color: #909399;
      font-weight: 500;
    }

    .api-value {
      font-size: 13px;
      color: #409eff;
      font-weight: 600;
    }
  }

  .rule-resources {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding-left: 44px;

    .resource-group {
      display: flex;
      align-items: center;
      gap: 6px;

      .res-label {
        font-size: 12px;
        color: #909399;
        font-weight: 500;
        min-width: 36px;
      }

      .res-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;

        .empty-text {
          font-size: 12px;
          color: #909399;
        }
      }
    }
  }

  .rule-verbs {
    display: flex;
    align-items: center;
    gap: 6px;
    padding-left: 44px;

    .verb-label {
      font-size: 12px;
      color: #909399;
      font-weight: 500;
      min-width: 36px;
    }

    .verb-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
    }
  }
}
</style>
