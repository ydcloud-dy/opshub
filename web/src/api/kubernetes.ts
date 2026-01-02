import request from '@/utils/request'

export interface Cluster {
  id: number
  name: string
  alias: string
  apiEndpoint: string
  version: string
  status: number
  nodeCount: number    // 节点数量（缓存）
  podCount: number     // Pod数量（缓存）
  statusSyncedAt: string | null  // 状态最后同步时间
  region: string
  provider: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface CreateClusterParams {
  name: string
  alias?: string
  apiEndpoint: string
  kubeConfig: string
  region?: string
  provider?: string
  description?: string
}

export interface UpdateClusterParams {
  name?: string
  alias?: string
  apiEndpoint?: string
  kubeConfig?: string
  region?: string
  provider?: string
  description?: string
}

/**
 * 获取集群列表
 */
export function getClusterList() {
  return request<Cluster[]>({
    url: '/api/v1/plugins/kubernetes/clusters',
    method: 'get'
  })
}

/**
 * 获取集群详情
 */
export function getClusterDetail(id: number) {
  return request<Cluster>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'get'
  })
}

/**
 * 创建集群
 */
export function createCluster(data: CreateClusterParams) {
  return request<Cluster>({
    url: '/api/v1/plugins/kubernetes/clusters',
    method: 'post',
    data
  })
}

/**
 * 更新集群
 */
export function updateCluster(id: number, data: UpdateClusterParams) {
  return request<Cluster>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除集群
 */
export function deleteCluster(id: number) {
  return request({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'delete'
  })
}

/**
 * 测试集群连接
 */
export function testClusterConnection(id: number) {
  return request<{
    status: string
    version: string
  }>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}/test`,
    method: 'post'
  })
}

/**
 * 获取集群凭证（解密后的 KubeConfig）
 */
export function getClusterConfig(id: number) {
  return request<string>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}/config`,
    method: 'get'
  })
}

// ==================== Kubernetes 资源类型定义 ====================

export interface NodeInfo {
  name: string
  status: string
  roles: string
  age: string
  version: string
  internalIP: string
  externalIP?: string
  osImage: string
  kernelVersion: string
  containerRuntime: string
  labels: Record<string, string>
  annotations: Record<string, string>
  cpuCapacity: string
  memoryCapacity: string
  podCount: number
  podCapacity: number
  schedulable: boolean
  taintCount: number
  taints?: TaintInfo[]
  podCIDR?: string
  providerID?: string
  conditions?: NodeCondition[]
}

export interface NodeCondition {
  type: string
  status: string
  lastHeartbeatTime: string
  lastTransitionTime: string
  reason: string
  message: string
}

export interface TaintInfo {
  key: string
  value: string
  effect: string
}

export interface NamespaceInfo {
  name: string
  status: string
  age: string
  labels: Record<string, string>
}

export interface PodInfo {
  name: string
  namespace: string
  ready: string
  status: string
  restarts: number
  age: string
  ip: string
  node: string
  labels: Record<string, string>
}

export interface DeploymentInfo {
  name: string
  namespace: string
  ready: string
  upToDate: number
  available: number
  age: string
  replicas: number
  selector: Record<string, string>
  labels: Record<string, string>
}

export interface ClusterStats {
  nodeCount: number
  workloadCount: number
  podCount: number
  cpuUsage: number
  memoryUsage: number
  cpuCapacity: number
  memoryCapacity: number
  cpuAllocatable: number
  memoryAllocatable: number
  cpuUsed: number
  memoryUsed: number
}

export interface ClusterNetworkInfo {
  serviceCIDR: string
  podCIDR: string
  apiServerAddress: string
  networkPlugin: string
  proxyMode: string
  dnsService: string
}

export interface ComponentInfo {
  name: string
  version: string
  status: string
}

export interface RuntimeInfo {
  containerRuntime: string
  version: string
}

export interface StorageInfo {
  name: string
  provisioner: string
  reclaimPolicy: string
}

export interface EventInfo {
  type: string          // 事件类型: Normal, Warning
  reason: string        // 原因
  message: string       // 消息
  source: string        // 来源
  count: number         // 次数
  firstTimestamp: string
  lastTimestamp: string // 最后发生时间
  involvedObject: {
    kind: string
    name: string
    namespace?: string
  }
}

export interface ClusterComponentInfo {
  components: ComponentInfo[]
  runtime: RuntimeInfo
  storage: StorageInfo[]
}

// ==================== Kubernetes 资源 API ====================

/**
 * 获取节点列表
 */
export function getNodes(clusterId: number) {
  return request<NodeInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/nodes',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取命名空间列表
 */
export function getNamespaces(clusterId: number) {
  return request<NamespaceInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/namespaces',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 Pod 列表
 */
export function getPods(clusterId: number, namespace?: string, nodeName?: string) {
  return request<PodInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/pods',
    method: 'get',
    params: { clusterId, namespace, nodeName }
  })
}

/**
 * 获取 Deployment 列表
 */
export function getDeployments(clusterId: number, namespace?: string) {
  return request<DeploymentInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/deployments',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取集群统计信息
 */
export function getClusterStats(clusterId: number) {
  return request<ClusterStats>({
    url: '/api/v1/plugins/kubernetes/resources/stats',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取集群网络信息
 */
export function getClusterNetworkInfo(clusterId: number) {
  return request<ClusterNetworkInfo>({
    url: '/api/v1/plugins/kubernetes/resources/network',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取集群组件信息
 */
export function getClusterComponentInfo(clusterId: number) {
  return request<ClusterComponentInfo>({
    url: '/api/v1/plugins/kubernetes/resources/components',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取集群事件列表
 */
export function getClusterEvents(clusterId: number, namespace?: string) {
  return request<EventInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/events',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取集群API组列表
 */
export function getAPIGroups(clusterId: number) {
  return request<string[]>({
    url: '/api/v1/plugins/kubernetes/resources/api-groups',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 根据API组获取资源列表
 */
export function getResourcesByAPIGroup(clusterId: number, apiGroups: string[] | string) {
  // 确保 apiGroups 是数组
  const groups = Array.isArray(apiGroups) ? apiGroups : [apiGroups]
  return request<string[]>({
    url: '/api/v1/plugins/kubernetes/resources/api-resources',
    method: 'get',
    params: { clusterId, apiGroups: groups.join(',') }
  })
}

/**
 * 生成集群 KubeConfig 凭据
 */
export function generateKubeConfig(clusterId: number, username: string) {
  return request<{ kubeconfig: string; username: string }>({
    url: '/api/v1/plugins/kubernetes/clusters/kubeconfig',
    method: 'post',
    data: { clusterId, username }
  })
}

/**
 * 吊销集群 KubeConfig 凭据
 */
export function revokeKubeConfig(clusterId: number, username: string) {
  return request({
    url: '/api/v1/plugins/kubernetes/clusters/kubeconfig',
    method: 'delete',
    data: { clusterId, username }
  })
}

/**
 * 完全吊销用户凭据（删除 SA、RoleBinding 和数据库记录）
 */
export function revokeCredentialFully(clusterId: number, serviceAccount: string, username: string) {
  return request({
    url: '/api/v1/plugins/kubernetes/clusters/kubeconfig/revoke',
    method: 'delete',
    data: { clusterId, serviceAccount, username }
  })
}

// ==================== Kubernetes 角色相关 ====================

export interface Role {
  name: string
  namespace?: string
  labels: Record<string, string>
  age: string
  rules: any[]
  isCustom?: boolean // 是否为自定义角色（系统角色和默认角色为 false）
}

export interface RoleDetail {
  name: string
  namespace?: string
  labels: Record<string, string>
  age: string
  rules: any[]
}

/**
 * 获取集群角色列表
 */
export function getClusterRoles(clusterId: number) {
  return request<Role[]>({
    url: '/api/v1/plugins/kubernetes/roles/cluster',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 创建默认集群角色
 */
export function createDefaultClusterRoles(clusterId: number) {
  return request<{
    created: string[]
    existing: string[]
  }>({
    url: '/api/v1/plugins/kubernetes/roles/create-defaults',
    method: 'post',
    params: { clusterId }
  })
}

/**
 * 创建默认命名空间角色
 */
export function createDefaultNamespaceRoles(clusterId: number, namespace: string) {
  return request<{
    created: string[]
    existing: string[]
  }>({
    url: '/api/v1/plugins/kubernetes/roles/create-defaults-namespace',
    method: 'post',
    params: { clusterId, namespace }
  })
}

/**
 * 获取命名空间列表（用于角色管理）
 */
export function getNamespacesForRoles(clusterId: number) {
  return request<{ name: string; podCount?: number }[]>({
    url: '/api/v1/plugins/kubernetes/roles/namespaces',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取命名空间角色列表
 */
export function getNamespaceRoles(clusterId: number, namespace: string) {
  return request<Role[]>({
    url: '/api/v1/plugins/kubernetes/roles/namespace',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取角色详情
 */
export function getRoleDetail(clusterId: number, namespace: string, roleName: string) {
  // 当 namespace 为空时（集群角色），使用 'cluster' 代替空字符串避免双斜杠问题
  const ns = namespace || 'cluster'
  return request<RoleDetail>({
    url: `/api/v1/plugins/kubernetes/roles/${ns}/${roleName}`,
    method: 'get',
    params: { clusterId }
  })
}

// ==================== Kubernetes 角色绑定相关 ====================

export interface BindUserToRoleParams {
  clusterId: number
  userId: number
  roleName: string
  roleNamespace: string
  roleType: string
}

export interface UnbindUserFromRoleParams {
  clusterId: number
  userId: number
  roleName: string
  roleNamespace: string
}

export interface AvailableUser {
  id: number
  username: string
  realName: string
  email: string
}

export interface BoundUser {
  userId: number
  username: string
  realName: string
  boundAt: string
}

/**
 * 绑定用户到K8s角色
 */
export function bindUserToRole(data: BindUserToRoleParams) {
  return request({
    url: '/api/v1/plugins/kubernetes/role-bindings/bind',
    method: 'post',
    data
  })
}

/**
 * 解绑用户K8s角色
 */
export function unbindUserFromRole(data: UnbindUserFromRoleParams) {
  return request({
    url: '/api/v1/plugins/kubernetes/role-bindings/unbind',
    method: 'delete',
    data
  })
}

/**
 * 获取角色已绑定的用户列表
 */
export function getRoleBoundUsers(clusterId: number, roleName: string, roleNamespace: string) {
  return request<BoundUser[]>({
    url: '/api/v1/plugins/kubernetes/role-bindings/users',
    method: 'get',
    params: { clusterId, roleName, roleNamespace }
  })
}

/**
 * 获取可绑定的用户列表
 */
export function getAvailableUsers(keyword: string, page: number, pageSize: number) {
  return request<{
    list: AvailableUser[]
    total: number
    page: number
    pageSize: number
  }>({
    url: '/api/v1/plugins/kubernetes/role-bindings/available-users',
    method: 'get',
    params: { keyword, page, pageSize }
  })
}

/**
 * 删除角色
 */
export function deleteRole(clusterId: number, namespace: string, roleName: string) {
  // 当 namespace 为空时（集群角色），使用 'cluster' 代替空字符串避免双斜杠问题
  const ns = namespace || 'cluster'
  return request({
    url: `/api/v1/plugins/kubernetes/roles/${ns}/${roleName}`,
    method: 'delete',
    params: { clusterId }
  })
}

/**
 * 创建角色请求接口
 */
export interface CreateRoleRequest {
  namespace: string
  name: string
  rules: {
    apiGroups: string[]
    resources: string[]
    resourceNames: string[]
    verbs: string[]
  }[]
}

/**
 * 创建角色
 */
export function createRole(clusterId: number, data: CreateRoleRequest) {
  return request({
    url: `/api/v1/plugins/kubernetes/clusters/${clusterId}/roles`,
    method: 'post',
    data
  })
}

/**
 * 凭据用户接口
 */
export interface CredentialUser {
  username: string       // 平台用户名
  serviceAccount: string // K8s ServiceAccount 完整名称
  namespace: string      // 命名空间
  userId: number         // 平台用户ID
  createdAt: string      // 创建时间
}

/**
 * 获取集群的凭据用户列表
 */
export function getClusterCredentialUsers(clusterId: number) {
  return request<CredentialUser[]>({
    url: '/api/v1/plugins/kubernetes/role-bindings/credential-users',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取用户现有的KubeConfig
 */
export function getExistingKubeConfig(clusterId: number) {
  return request<{
    kubeconfig: string
    username: string
  }>({
    url: '/api/v1/plugins/kubernetes/clusters/kubeconfig/existing',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 根据ServiceAccount获取KubeConfig
 */
export function getServiceAccountKubeConfig(clusterId: number, serviceAccount: string) {
  return request<{
    kubeconfig: string
  }>({
    url: '/api/v1/plugins/kubernetes/clusters/kubeconfig/sa',
    method: 'post',
    data: { clusterId, serviceAccount }
  })
}

// ==================== 用户角色绑定 ====================

export interface UserRoleBinding {
  id: number
  clusterId: number
  clusterName: string
  userId: number
  username: string
  realName: string
  roleName: string
  roleNamespace: string
  roleType: 'ClusterRole' | 'Role'
  createdAt: string
}

/**
 * 获取用户的所有K8s角色绑定
 */
export function getUserRoleBindings(clusterId: number, userId?: number) {
  return request<UserRoleBinding[]>({
    url: '/api/v1/plugins/kubernetes/role-bindings/user-bindings',
    method: 'get',
    params: { clusterId, userId }
  })
}

/**
 * 同步单个集群状态
 */
export function syncClusterStatus(id: number) {
  return request({
    url: `/api/v1/plugins/kubernetes/clusters/${id}/sync`,
    method: 'post'
  })
}

/**
 * 同步所有集群状态
 */
export function syncAllClustersStatus() {
  return request({
    url: '/api/v1/plugins/kubernetes/clusters/sync-all',
    method: 'post'
  })
}
