import request from '@/utils/request'

export interface Application {
  id: number
  code: string
  name: string
  description?: string
  ownerName?: string
  ownerUsername?: string
  ownerUserId?: number
  departmentId?: number
  departmentName?: string
  team?: string
  environmentId: number
  environmentName?: string
  criticality: string
  status: string
  lifecycle: string
  healthStatus: string
  healthCheckedAt?: string
  healthMessage?: string
  healthSource?: string
  repositoryUrl?: string
  documentationUrl?: string
  language?: string
  tags?: string
  environmentCount?: number
  domainCount?: number
  resourceCount?: number
  componentCount?: number
  dependencyCount?: number
  createdAt?: string
  updatedAt?: string
}

export interface Environment {
  id: number
  code: string
  name: string
  kind: string
  region?: string
  status: string
  description?: string
  applicationCount?: number
  createdAt?: string
  updatedAt?: string
}

export interface Domain {
  id: number
  applicationId: number
  environmentId: number
  domain: string
  protocol: string
  port: number
  path: string
  dnsProvider?: string
  certificateId?: number
  certificateName?: string
  certificateStatus?: string
  certificateExpiry?: string
  isPrimary: boolean
  status: string
  source: string
  description?: string
  lastCheckedAt?: string
  responseTimeMs?: number
  httpStatusCode?: number
  probeMessage?: string
  resolvedAddress?: string
  tlsExpiresAt?: string
  tlsIssuer?: string
}

export interface Resource {
  id: number
  applicationId: number
  environmentId: number
  kind: string
  name: string
  address?: string
  port?: number
  hostId?: number
  clusterId?: number
  namespace?: string
  externalId?: string
  credentialId?: number
  status: string
  source: string
  metadata?: string
  description?: string
  lastSyncedAt?: string
  lastCheckedAt?: string
  responseTimeMs?: number
  healthMessage?: string
}

export interface Component {
  id: number
  applicationId: number
  environmentId: number
  category: string
  type: string
  name: string
  address?: string
  port?: number
  databaseName?: string
  version?: string
  credentialId?: number
  tlsEnabled: boolean
  status: string
  source: string
  metadata?: string
  description?: string
  lastCheckedAt?: string
  responseTimeMs?: number
  healthMessage?: string
}

export interface Dependency {
  id: number
  sourceApplicationId: number
  sourceEnvironmentId: number
  targetApplicationId: number
  targetComponentId: number
  targetResourceId: number
  targetName?: string
  relationType: string
  protocol?: string
  endpoint?: string
  port?: number
  criticality: string
  status: string
  description?: string
}

export interface Credential {
  id: number
  name: string
  kind: string
  username?: string
  keyVersion?: string
  scope: string
  status: string
  description?: string
  ownerUserId: number
  lastRotatedAt?: string
  expiresAt?: string
  hasSecret: boolean
  secretMask: string
  canReveal: boolean
  canManage: boolean
  grantCount: number
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface TopologyNode {
  id: string
  label: string
  type: string
  status: string
  appId?: number
  environmentId?: number
  metadata?: Record<string, any>
}

export interface TopologyEdge {
  id: string
  source: string
  target: string
  label: string
  protocol?: string
  status: string
  kind: string
}

const base = '/api/v1/plugins/app-inventory'

export const getOverview = () => request.get<any, any>(`${base}/overview`)
export const getTopology = (appId?: number) => request.get<any, { nodes: TopologyNode[]; edges: TopologyEdge[]; stats: Record<string, number> }>(`${base}/topology`, { params: appId ? { app_id: appId } : undefined })

export const listApplications = (params?: any) => request.get<any, PageResult<Application>>(`${base}/apps`, { params })
export const getApplication = (id: number) => request.get<any, any>(`${base}/apps/${id}`)
export const createApplication = (data: Partial<Application>) => request.post<any, Application>(`${base}/apps`, data)
export const updateApplication = (id: number, data: Partial<Application>) => request.put<any, Application>(`${base}/apps/${id}`, data)
export const deleteApplication = (id: number) => request.delete(`${base}/apps/${id}`)
export const probeApplication = (id: number) => request.post<any, Application>(`${base}/apps/${id}/probe`)

export const listEnvironments = (appId?: number) => request.get<any, Environment[]>(`${base}/environments`, { params: appId ? { app_id: appId } : undefined })
export const createEnvironment = (data: Partial<Environment>) => request.post<any, Environment>(`${base}/environments`, data)
export const updateEnvironment = (id: number, data: Partial<Environment>) => request.put<any, Environment>(`${base}/environments/${id}`, data)
export const deleteEnvironment = (id: number) => request.delete(`${base}/environments/${id}`)

export const listDomains = (params?: any) => request.get<any, PageResult<Domain>>(`${base}/domains`, { params })
export const createDomain = (data: Partial<Domain>) => request.post<any, Domain>(`${base}/domains`, data)
export const updateDomain = (id: number, data: Partial<Domain>) => request.put<any, Domain>(`${base}/domains/${id}`, data)
export const deleteDomain = (id: number) => request.delete(`${base}/domains/${id}`)
export const probeDomain = (id: number) => request.post<any, Domain>(`${base}/domains/${id}/probe`)

export const listResources = (params?: any) => request.get<any, PageResult<Resource>>(`${base}/resources`, { params })
export const createResource = (data: Partial<Resource>) => request.post<any, Resource>(`${base}/resources`, data)
export const updateResource = (id: number, data: Partial<Resource>) => request.put<any, Resource>(`${base}/resources/${id}`, data)
export const deleteResource = (id: number) => request.delete(`${base}/resources/${id}`)
export const probeResource = (id: number) => request.post<any, Resource>(`${base}/resources/${id}/probe`)

export const listComponents = (params?: any) => request.get<any, PageResult<Component>>(`${base}/components`, { params })
export const createComponent = (data: Partial<Component>) => request.post<any, Component>(`${base}/components`, data)
export const updateComponent = (id: number, data: Partial<Component>) => request.put<any, Component>(`${base}/components/${id}`, data)
export const deleteComponent = (id: number) => request.delete(`${base}/components/${id}`)
export const probeComponent = (id: number) => request.post<any, Component>(`${base}/components/${id}/probe`)

export const listDependencies = (params?: any) => request.get<any, PageResult<Dependency>>(`${base}/dependencies`, { params })
export const createDependency = (data: Partial<Dependency>) => request.post<any, Dependency>(`${base}/dependencies`, data)
export const updateDependency = (id: number, data: Partial<Dependency>) => request.put<any, Dependency>(`${base}/dependencies/${id}`, data)
export const deleteDependency = (id: number) => request.delete(`${base}/dependencies/${id}`)

export const listCredentials = (params?: any) => request.get<any, PageResult<Credential>>(`${base}/credentials`, { params })
export const createCredential = (data: any) => request.post<any, Credential>(`${base}/credentials`, data)
export const updateCredential = (id: number, data: any) => request.put<any, Credential>(`${base}/credentials/${id}`, data)
export const deleteCredential = (id: number) => request.delete(`${base}/credentials/${id}`)
export const revealCredential = (id: number, reason: string) => request.post<any, any>(`${base}/credentials/${id}/reveal`, { reason })
export const listCredentialGrants = (id: number) => request.get<any, any[]>(`${base}/credentials/${id}/grants`)
export const upsertCredentialGrant = (id: number, data: any) => request.post<any, any>(`${base}/credentials/${id}/grants`, data)
export const deleteCredentialGrant = (id: number) => request.delete(`${base}/credentials/grants/${id}`)
export const listSecretAudits = (credentialId?: number) => request.get<any, any[]>(`${base}/credentials/audits`, { params: credentialId ? { credential_id: credentialId } : undefined })

export const getReferences = () => request.get<any, Record<string, any[]>>(`${base}/references`)
export const listKubernetesNamespaces = (clusterId: number) => request.get<any, Array<{ name: string; status: string }>>(`${base}/discovery/kubernetes/namespaces`, { params: { cluster_id: clusterId } })
export const previewKubernetes = (data: any) => request.post<any, any>(`${base}/discovery/kubernetes/preview`, data)
export const importKubernetes = (data: any) => request.post<any, any>(`${base}/discovery/kubernetes/import`, data)
export const listDiscoveryRuns = (appId?: number) => request.get<any, any[]>(`${base}/discovery/runs`, { params: appId ? { app_id: appId } : undefined })
