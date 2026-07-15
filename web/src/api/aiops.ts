import request from '@/utils/request'

const aiModelRequestConfig = {
  timeout: 300000
}

export interface AIProvider {
  id: number
  name: string
  provider: string
  baseUrl: string
  apiKey?: string
  model: string
  temperature: number
  maxTokens: number
  timeout: number
  reasoningEffort?: string
  enabled: boolean
  isDefault: boolean
  remark?: string
  lastTestAt?: string
  lastTestMsg?: string
  createdAt: string
  updatedAt: string
}

export interface AIProviderOption {
  id: number
  name: string
  model: string
  provider: string
  isDefault: boolean
  remark?: string
}

export interface AIProviderPayload {
  name: string
  provider?: string
  baseUrl: string
  apiKey?: string
  model: string
  temperature: number
  maxTokens: number
  timeout: number
  reasoningEffort?: string
  enabled: boolean
  isDefault: boolean
  remark?: string
}

export interface AIChatResponse {
  sessionId: number
  answer: string
  model: string
  fallback: boolean
  finishReason?: string
  thinkingSteps?: string[]
  message?: any
}

export interface AIChatStreamEvent {
  type: 'meta' | 'delta' | 'done' | 'error' | 'ping'
  sessionId?: number
  delta?: string
  answer?: string
  model?: string
  fallback?: boolean
  finishReason?: string
  thinkingSteps?: string[]
  message?: any
  error?: string
}

export interface AIChatPayload {
  sessionId?: number
  message: string
  providerId?: number
  continue?: boolean
  continueFromMessageId?: number
  originalQuestion?: string
  previousAnswer?: string
}

export interface AIDiagnosisResponse {
  sessionId: number
  taskId: number
  conclusion: string
  evidence: any
  suggestion: string
  model: string
  fallback: boolean
}

export interface AIAlertEvent {
  id: number
  ruleId: number
  ruleName: string
  dataSourceName: string
  dataSourceType: string
  severity: string
  state: string
  value: number
  message: string
  labels: string
  startedAt: string
  lastEvalAt: string
}

export interface AIRootCauseAnalysis {
  id: number
  alertEventId: number
  ruleId: number
  ruleName: string
  severity: string
  state: string
  summary: string
  rootCause: string
  evidenceJson: string
  suggestion: string
  model: string
  fallback: boolean
  status: string
  createdAt: string
  updatedAt: string
}

export const getAIProviders = () => {
  return request.get('/api/v1/aiops/providers') as Promise<AIProvider[]>
}

export const getAIProviderOptions = () => {
  return request.get('/api/v1/aiops/providers/options') as Promise<AIProviderOption[]>
}

export const createAIProvider = (data: AIProviderPayload) => {
  return request.post('/api/v1/aiops/providers', data) as Promise<AIProvider>
}

export const updateAIProvider = (id: number, data: AIProviderPayload) => {
  return request.put(`/api/v1/aiops/providers/${id}`, data) as Promise<AIProvider>
}

export const deleteAIProvider = (id: number) => {
  return request.delete(`/api/v1/aiops/providers/${id}`)
}

export const testAIProvider = (id: number) => {
  return request.post(`/api/v1/aiops/providers/${id}/test`, undefined, aiModelRequestConfig)
}

export const chatWithAI = (data: AIChatPayload) => {
  return request.post('/api/v1/aiops/chat', data, aiModelRequestConfig) as Promise<AIChatResponse>
}

export const stopAIChat = (data: { sessionId: number; messageId?: number }) => {
  return request.post('/api/v1/aiops/chat/stop', data) as Promise<{ stopped: boolean }>
}

export const chatWithAIStream = async (
  data: AIChatPayload,
  onEvent: (event: AIChatStreamEvent) => void,
  options?: { signal?: AbortSignal }
) => {
  const token = localStorage.getItem('token')
  const response = await fetch('/api/v1/aiops/chat/stream', {
    method: 'POST',
    credentials: 'include',
    signal: options?.signal,
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      'Cache-Control': 'no-cache',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(data)
  })

  if (!response.ok || !response.body) {
    const text = await response.text().catch(() => '')
    throw new Error(text || `流式问答请求失败：HTTP ${response.status}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  const parseFrame = (frame: string) => {
    const dataLines = frame
      .split('\n')
      .map(line => line.trimEnd())
      .filter(line => line.startsWith('data:'))
      .map(line => line.replace(/^data:\s?/, ''))
    if (!dataLines.length) return
    let event: AIChatStreamEvent
    try {
      event = JSON.parse(dataLines.join('\n')) as AIChatStreamEvent
    } catch {
      // 忽略不完整或非 JSON 的 SSE 片段，后续帧会继续处理。
      return
    }
    onEvent(event)
  }

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const frames = buffer.split(/\r?\n\r?\n/)
    buffer = frames.pop() || ''
    frames.forEach(parseFrame)
  }

  buffer += decoder.decode()
  if (buffer.trim()) {
    parseFrame(buffer)
  }
}

export const analyzeLogs = (data: {
  logs?: string
  source?: string
  sourceType?: 'manual' | 'host' | 'kubernetes'
  providerId?: number
  k8sObjectType?: 'pod' | 'deployment' | 'statefulset' | 'daemonset' | 'job' | 'cronjob'
  title?: string
  sessionId?: number
  hostId?: number
  logPath?: string
  clusterId?: number
  namespace?: string
  podName?: string
  container?: string
  tailLines?: number
}) => {
  return request.post('/api/v1/aiops/logs/analyze', data, aiModelRequestConfig) as Promise<AIDiagnosisResponse>
}

export type AIKubernetesDiagnosisObjectType =
  | 'pod'
  | 'deployment'
  | 'statefulset'
  | 'daemonset'
  | 'job'
  | 'cronjob'
  | 'service'
  | 'ingress'
  | 'node'
  | 'namespace'
  | 'configmap'
  | 'secret'
  | 'persistentvolumeclaim'
  | 'persistentvolume'
  | 'storageclass'
  | 'endpoints'
  | 'networkpolicy'

export const diagnoseKubernetes = (data: {
  objectType: AIKubernetesDiagnosisObjectType
  clusterId: number
  namespace?: string
  name: string
  container?: string
  tailLines?: number
  providerId?: number
}) => {
  return request.post('/api/v1/aiops/diagnosis/kubernetes', data, aiModelRequestConfig) as Promise<AIDiagnosisResponse>
}

export const diagnoseHost = (hostId: number, data?: { focus?: string; providerId?: number }) => {
  return request.post(`/api/v1/aiops/diagnosis/hosts/${hostId}`, data || {}, aiModelRequestConfig) as Promise<AIDiagnosisResponse>
}

export const getAIAlertEvents = (params: { page?: number; pageSize?: number; state?: string; severity?: string }) => {
  return request.get('/api/v1/aiops/alerts/events', { params }) as Promise<any>
}

export const analyzeAIAlert = (data: { alertEventId?: number; query?: string; providerId?: number }) => {
  return request.post('/api/v1/aiops/alerts/analyze', data, aiModelRequestConfig) as Promise<AIRootCauseAnalysis>
}

export const getAIRootCauseAnalyses = (params: { page?: number; pageSize?: number }) => {
  return request.get('/api/v1/aiops/alerts/analyses', { params }) as Promise<any>
}

export const getAISessions = (params: { page?: number; pageSize?: number; type?: string }) => {
  return request.get('/api/v1/aiops/sessions', { params }) as Promise<any>
}

export const getAISessionDetail = (id: number) => {
  return request.get(`/api/v1/aiops/sessions/${id}`) as Promise<any>
}

export const deleteAISession = (id: number) => {
  return request.delete(`/api/v1/aiops/sessions/${id}`)
}

export const getAIDiagnosisTasks = (params: { page?: number; pageSize?: number }) => {
  return request.get('/api/v1/aiops/diagnosis/tasks', { params }) as Promise<any>
}
