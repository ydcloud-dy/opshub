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
  cpuUsed: number
  memoryUsed: number
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
 * 获取 Pod 详情
 */
export function getPodDetail(clusterId: number, namespace: string, podName: string) {
  return request<any>({
    url: `/api/v1/plugins/kubernetes/resources/pods/${namespace}/${podName}`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 Pod 事件
 */
export function getPodEvents(clusterId: number, namespace: string, podName: string) {
  return request<{ events: EventInfo[] }>({
    url: `/api/v1/plugins/kubernetes/resources/pods/${namespace}/${podName}/events`,
    method: 'get',
    params: { clusterId }
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
 * 创建默认命名空间角色（ClusterRole）
 */
export function createDefaultNamespaceRoles(clusterId: number) {
  return request<{
    created: string[]
    existing: string[]
  }>({
    url: '/api/v1/plugins/kubernetes/roles/create-defaults-namespace',
    method: 'post',
    params: { clusterId }
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

/**
 * 更新工作负载
 * 请求体直接是 Kubernetes 对象，参数通过 URL 查询参数传递
 */
export interface UpdateWorkloadParams {
  cluster: string
  namespace: string
  type: string
  name: string
  yaml: string
}

export function updateWorkload(params: UpdateWorkloadParams) {
  // 将 YAML 字符串解析为对象
  const kubernetesObject = JSON.parse(params.yaml)

  return request({
    url: '/api/v1/plugins/kubernetes/workloads/update',
    method: 'post',
    params: {
      cluster: params.cluster,
      namespace: params.namespace,
      type: params.type,
      name: params.name
    },
    data: kubernetesObject  // 直接发送 Kubernetes 对象，而不是包装在请求体中
  })
}

// ==================== 网络资源相关 ====================

// -------------------- Service --------------------

export interface ServicePort {
  name?: string
  protocol: string
  port: number
  targetPort?: string
  nodePort?: number
}

export interface ServiceInfo {
  name: string
  namespace: string
  type: 'ClusterIP' | 'NodePort' | 'LoadBalancer' | 'ExternalName'
  clusterIP: string
  externalIP: string
  ports: ServicePort[]
  selector: Record<string, string>
  sessionAffinity: string
  age: string
  labels: Record<string, string>
  endpoints: number
}

/**
 * 获取服务列表
 */
export function getServices(clusterId: number, namespace?: string) {
  return request<ServiceInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/services',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取服务 YAML
 */
export function getServiceYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/services/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新服务 YAML
 */
export function updateServiceYAML(clusterId: number, namespace: string, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/services/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, ...data }
  })
}

/**
 * 创建服务
 */
export function createService(clusterId: number, namespace: string, data: {
  name: string
  type: string
  clusterIP?: string
  ports: ServicePort[]
  selector?: Record<string, string>
  sessionAffinity?: string
}) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/services/${namespace}/${data.name}`,
    method: 'post',
    data: { clusterId, namespace, ...data }
  })
}

/**
 * 创建服务 (从 YAML)
 */
export function createServiceYAML(clusterId: number, namespace: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/services/${namespace}/yaml`,
    method: 'post',
    data: { clusterId, ...data }
  })
}

/**
 * 删除服务
 */
export function deleteService(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/services/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// -------------------- Ingress --------------------

export interface IngressPath {
  host: string
  path: string
  pathType: string
  service: string
  port: number
}

export interface IngressTLS {
  hosts: string[]
  secretName: string
}

export interface IngressInfo {
  name: string
  namespace: string
  hosts: string[]
  paths: IngressPath[]
  tls: IngressTLS[]
  ingressClass: string
  age: string
  labels: Record<string, string>
  addresses: string[]
}

/**
 * 获取 Ingress 列表
 */
export function getIngresses(clusterId: number, namespace?: string) {
  return request<IngressInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/ingresses',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 Ingress YAML
 */
export function getIngressYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/ingresses/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 Ingress YAML
 */
export function updateIngressYAML(clusterId: number, namespace: string, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/ingresses/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, ...data }
  })
}

/**
 * 创建 Ingress
 */
export function createIngress(clusterId: number, namespace: string, data: {
  name: string
  ingressClass?: string
  rules: Array<{
    host?: string
    paths: Array<{
      path: string
      pathType: string
      service: string
      port: number
    }>
  }>
  tls?: Array<{
    hosts: string[]
    secretName: string
  }>
}) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/ingresses/${namespace}/${data.name}`,
    method: 'post',
    data: { clusterId, namespace, ...data }
  })
}

/**
 * 创建 Ingress (从 YAML)
 */
export function createIngressYAML(clusterId: number, namespace: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/ingresses/${namespace}/yaml`,
    method: 'post',
    data: { clusterId, ...data }
  })
}

/**
 * 删除 Ingress
 */
export function deleteIngress(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/ingresses/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// -------------------- Endpoints --------------------

export interface EndpointAddress {
  ip: string
  hostname?: string
  nodeName?: string
  targetRef?: string
  ready: boolean
}

export interface EndpointPort {
  name?: string
  protocol: string
  port: number
}

export interface EndpointSubset {
  addresses: EndpointAddress[]
  notReadyAddresses: EndpointAddress[]
  ports: EndpointPort[]
}

export interface EndpointsInfo {
  name: string
  namespace: string
  subsets: EndpointSubset[]
  age: string
  labels: Record<string, string>
}

/**
 * 获取端点列表
 */
export function getEndpoints(clusterId: number, namespace?: string) {
  return request<EndpointsInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/endpoints',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取端点详情
 */
export function getEndpointsDetail(clusterId: number, namespace: string, name: string) {
  return request<EndpointsInfo>({
    url: `/api/v1/plugins/kubernetes/resources/endpoints/${namespace}/${name}`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 创建端点 (从 YAML)
 */
export function createEndpointYAML(clusterId: number, namespace: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/endpoints/${namespace}/yaml`,
    method: 'post',
    data: { clusterId, ...data }
  })
}

/**
 * 删除端点
 */
export function deleteEndpoint(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/endpoints/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

/**
 * 获取端点 YAML
 */
export function getEndpointYAML(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/endpoints/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新端点 YAML
 */
export function updateEndpointYAML(clusterId: number, namespace: string, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/endpoints/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, ...data }
  })
}

// -------------------- NetworkPolicy --------------------

export interface PolicyPort {
  protocol?: string
  port?: string | number
}

export interface PolicyPeer {
  podSelector?: Record<string, string>
  namespaceSelector?: Record<string, string>
  ipBlock?: {
    cidr: string
    except?: string[]
  }
}

export interface PolicyRule {
  ports: PolicyPort[]
  from?: PolicyPeer[]
  to?: PolicyPeer[]
}

export interface NetworkPolicyDetailInfo {
  name: string
  namespace: string
  podSelector: Record<string, string>
  ingress: PolicyRule[]
  egress: PolicyRule[]
  age: string
  labels: Record<string, string>
}

/**
 * 获取网络策略列表
 */
export function getNetworkPolicies(clusterId: number, namespace?: string) {
  return request<NetworkPolicyDetailInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/networkpolicies',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取网络策略 YAML
 */
export function getNetworkPolicyYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/networkpolicies/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新网络策略 YAML
 */
export function updateNetworkPolicyYAML(clusterId: number, namespace: string, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/networkpolicies/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, ...data }
  })
}

/**
 * 创建网络策略
 */
export function createNetworkPolicy(clusterId: number, namespace: string, data: {
  name: string
  podSelector?: Record<string, string>
  policyTypes?: string[]
  ingress?: Array<{
    ports?: Array<{
      protocol?: string
      port?: number
      endPort?: number
      namedPort?: string
    }>
    from?: Array<{
      podSelector?: Record<string, string>
      namespaceSelector?: Record<string, string>
      ipBlock?: {
        cidr: string
        except?: string[]
      }
    }>
  }>
  egress?: Array<{
    ports?: Array<{
      protocol?: string
      port?: number
      endPort?: number
      namedPort?: string
    }>
    to?: Array<{
      podSelector?: Record<string, string>
      namespaceSelector?: Record<string, string>
      ipBlock?: {
        cidr: string
        except?: string[]
      }
    }>
  }>
}) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/networkpolicies/${namespace}/${data.name}`,
    method: 'post',
    data: { clusterId, namespace, ...data }
  })
}

/**
 * 创建网络策略 (从 YAML)
 */
export function createNetworkPolicyYAML(clusterId: number, namespace: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/networkpolicies/${namespace}/yaml`,
    method: 'post',
    data: { clusterId, ...data }
  })
}

/**
 * 删除网络策略
 */
export function deleteNetworkPolicy(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/networkpolicies/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== ConfigMaps ====================

export interface ConfigMapInfo {
  name: string
  namespace: string
  dataCount: number
  age: string
  createdAt?: string
}

/**
 * 获取 ConfigMap 列表
 */
export function getConfigMaps(clusterId: number, namespace?: string) {
  return request<ConfigMapInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/configmaps',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 ConfigMap YAML
 */
export function getConfigMapYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 ConfigMap YAML
 */
export function updateConfigMapYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 ConfigMap
 */
export function deleteConfigMap(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== Secrets ====================

export interface SecretInfo {
  name: string
  namespace: string
  type: string
  dataCount: number
  age: string
}

/**
 * 获取 Secret 列表
 */
export function getSecrets(clusterId: number, namespace?: string) {
  return request<SecretInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/secrets',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 Secret YAML
 */
export function getSecretYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/secrets/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 Secret YAML
 */
export function updateSecretYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/secrets/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 Secret
 */
export function deleteSecret(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/secrets/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== 访问控制相关 ====================

export interface ServiceAccountInfo {
  name: string
  namespace: string
  secrets: string[]
  age: string
  labels: Record<string, string>
}

export interface RoleInfo {
  name: string
  namespace: string
  age: string
  labels: Record<string, string>
  rules: any[]
}

export interface RoleBindingInfo {
  name: string
  namespace: string
  roleKind: string // Role 或 ClusterRole
  roleName: string
  subjects: any[]
  age: string
  labels: Record<string, string>
}

export interface ClusterRoleInfo {
  name: string
  age: string
  labels: Record<string, string>
  rules: any[]
}

export interface ClusterRoleBindingInfo {
  name: string
  roleName: string
  subjects: any[]
  age: string
  labels: Record<string, string>
}

export interface PodSecurityPolicyInfo {
  name: string
  age: string
  labels: Record<string, string>
  spec: any
}

/**
 * 获取 ServiceAccount 列表
 */
export function getServiceAccounts(clusterId: number, namespace?: string) {
  return request<ServiceAccountInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/serviceaccounts',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取命名空间 Roles 列表
 */
export function getRoles(clusterId: number, namespace: string) {
  return request<RoleInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/roles',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 RoleBindings 列表
 */
export function getRoleBindings(clusterId: number, namespace: string) {
  return request<RoleBindingInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/rolebindings',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 ClusterRoleBindings 列表
 */
export function getClusterRoleBindings(clusterId: number) {
  return request<ClusterRoleBindingInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/clusterrolebindings',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 PodSecurityPolicies 列表
 */
export function getPodSecurityPolicies(clusterId: number) {
  return request<PodSecurityPolicyInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/podsecuritypolicies',
    method: 'get',
    params: { clusterId }
  })
}

// ==================== Storage Types ====================

export interface PVCInfo {
  name: string
  namespace: string
  status: string
  capacity: string
  accessModes: string[]
  storageClass: string
  volumeName: string
  age: string
  labels: Record<string, string>
}

export interface PVInfo {
  name: string
  capacity: string
  accessModes: string[]
  reclaimPolicy: string
  status: string
  claim: string
  storageClass: string
  reason: string
  age: string
  labels: Record<string, string>
}

export interface StorageClassInfo {
  name: string
  provisioner: string
  reclaimPolicy: string
  volumeBindingMode: string
  allowVolumeExpansion: boolean
  age: string
  labels: Record<string, string>
}

// ==================== PVC Functions ====================

/**
 * 获取 PVC 列表
 */
export function getPersistentVolumeClaims(clusterId: number, namespace?: string) {
  return request<PVCInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 PVC YAML
 */
export function getPersistentVolumeClaimYAML(clusterId: number, namespace: string, name: string) {
  return request<any>({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 PVC YAML
 */
export function updatePersistentVolumeClaimYAML(clusterId: number, namespace: string, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/${namespace}/${name}/yaml`,
    method: 'put',
    params: { clusterId },
    data: data
  })
}

/**
 * 删除 PVC
 */
export function deletePersistentVolumeClaim(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/${namespace}/${name}`,
    method: 'delete',
    params: { clusterId }
  })
}

/**
 * 通过 YAML 创建 PVC
 */
export function createPersistentVolumeClaimYAML(clusterId: number, namespace: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/${namespace}/yaml`,
    method: 'post',
    params: { clusterId },
    data: data
  })
}

// ==================== PV Functions ====================

/**
 * 获取 PV 列表
 */
export function getPersistentVolumes(clusterId: number) {
  return request<PVInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/persistentvolumes',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 PV YAML
 */
export function getPersistentVolumeYAML(clusterId: number, name: string) {
  return request<any>({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumes/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 PV YAML
 */
export function updatePersistentVolumeYAML(clusterId: number, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumes/${name}/yaml`,
    method: 'put',
    params: { clusterId },
    data: data
  })
}

/**
 * 删除 PV
 */
export function deletePersistentVolume(clusterId: number, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/persistentvolumes/${name}`,
    method: 'delete',
    params: { clusterId }
  })
}

/**
 * 通过 YAML 创建 PV
 */
export function createPersistentVolumeYAML(clusterId: number, data: any) {
  return request({
    url: '/api/v1/plugins/kubernetes/resources/persistentvolumes/yaml',
    method: 'post',
    params: { clusterId },
    data: data
  })
}

// ==================== StorageClass Functions ====================

/**
 * 获取 StorageClass 列表
 */
export function getStorageClasses(clusterId: number) {
  return request<StorageClassInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/storageclasses',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 StorageClass YAML
 */
export function getStorageClassYAML(clusterId: number, name: string) {
  return request<any>({
    url: `/api/v1/plugins/kubernetes/resources/storageclasses/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 StorageClass YAML
 */
export function updateStorageClassYAML(clusterId: number, name: string, data: any) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/storageclasses/${name}/yaml`,
    method: 'put',
    params: { clusterId },
    data: data
  })
}

/**
 * 删除 StorageClass
 */
export function deleteStorageClass(clusterId: number, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/storageclasses/${name}`,
    method: 'delete',
    params: { clusterId }
  })
}

/**
 * 通过 YAML 创建 StorageClass
 */
export function createStorageClassYAML(clusterId: number, data: any) {
  return request({
    url: '/api/v1/plugins/kubernetes/resources/storageclasses/yaml',
    method: 'post',
    params: { clusterId },
    data: data
  })
}

// ==================== ResourceQuota ====================

export interface ResourceQuotaInfo {
  name: string
  namespace: string
  requestsCpu?: string
  requestsMemory?: string
  limitsCpu?: string
  limitsMemory?: string
  age: string
  createdAt?: string
}

/**
 * 获取 ResourceQuota 列表
 */
export function getResourceQuotas(clusterId: number, namespace?: string) {
  return request<ResourceQuotaInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/resourcequotas',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 ResourceQuota YAML
 */
export function getResourceQuotaYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/resourcequotas/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 ResourceQuota YAML
 */
export function updateResourceQuotaYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/resourcequotas/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 ResourceQuota
 */
export function deleteResourceQuota(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/resourcequotas/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== LimitRange ====================

export interface LimitRangeInfo {
  name: string
  namespace: string
  type?: string
  resource?: string
  min?: string
  max?: string
  defaultLimit?: string
  defaultRequest?: string
  maxLimitRequestRatio?: string
  age: string
}

/**
 * 获取 LimitRange 列表
 */
export function getLimitRanges(clusterId: number, namespace?: string) {
  return request<LimitRangeInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/limitranges',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 LimitRange YAML
 */
export function getLimitRangeYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/limitranges/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 LimitRange YAML
 */
export function updateLimitRangeYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/limitranges/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 LimitRange
 */
export function deleteLimitRange(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/limitranges/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== HorizontalPodAutoscaler ====================

export interface HPAInfo {
  name: string
  namespace: string
  referenceTarget?: string
  minReplicas?: number
  maxReplicas?: number
  currentReplicas?: number
  targetCPU?: string
  targetMemory?: string
  age: string
  createdAt?: string
}

/**
 * 获取 HPA 列表
 */
export function getHorizontalPodAutoscalers(clusterId: number, namespace?: string) {
  return request<HPAInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 HPA YAML
 */
export function getHorizontalPodAutoscalerYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 HPA YAML
 */
export function updateHorizontalPodAutoscalerYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 HPA
 */
export function deleteHorizontalPodAutoscaler(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}

// ==================== PodDisruptionBudget ====================

export interface PDBInfo {
  name: string
  namespace: string
  minAvailable?: string
  maxUnavailable?: string
  allowedDisruptions?: number
  currentHealthy?: number
  desiredHealthy?: number
  age: string
  createdAt?: string
}

/**
 * 获取 PodDisruptionBudget 列表
 */
export function getPodDisruptionBudgets(clusterId: number, namespace?: string) {
  return request<PDBInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 PodDisruptionBudget YAML
 */
export function getPodDisruptionBudgetYAML(clusterId: number, namespace: string, name: string) {
  return request<{ items: Record<string, any> }>({
    url: `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${namespace}/${name}/yaml`,
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 更新 PodDisruptionBudget YAML
 */
export function updatePodDisruptionBudgetYAML(clusterId: number, namespace: string, name: string, yaml: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${namespace}/${name}/yaml`,
    method: 'put',
    data: { clusterId, yaml }
  })
}

/**
 * 删除 PodDisruptionBudget
 */
export function deletePodDisruptionBudget(clusterId: number, namespace: string, name: string) {
  return request({
    url: `/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/${namespace}/${name}`,
    method: 'delete',
    data: { clusterId }
  })
}
