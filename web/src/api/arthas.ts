import request from '@/utils/request'

const BASE_URL = '/api/v1/plugins/kubernetes/arthas'
const ARTHAS_TIMEOUT = 120000

// Java进程信息
export interface JavaProcess {
  pid: string
  mainClass: string
  commandLine?: string
}

// Arthas检查结果
export interface ArthasCheckResult {
  hasJava: boolean
  hasArthas: boolean
  javaVersion: string
}

// 通用请求参数
export interface ArthasBaseParams {
  clusterId: number
  namespace: string
  pod: string
  container: string
  processId?: string
}

// 列出Java进程
export function listJavaProcesses(params: Omit<ArthasBaseParams, 'processId'>) {
  return request({
    url: `${BASE_URL}/java-processes`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 检查Arthas是否安装
export function checkArthasInstalled(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/check`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 安装Arthas
export function installArthas(data: Omit<ArthasBaseParams, 'processId'>) {
  return request({
    url: `${BASE_URL}/install`,
    method: 'post',
    data,
    timeout: ARTHAS_TIMEOUT
  })
}

// 执行Arthas命令
export function executeArthasCommand(data: ArthasBaseParams & { command: string }) {
  return request({
    url: `${BASE_URL}/command`,
    method: 'post',
    data,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取控制面板信息
export function getDashboard(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/dashboard`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取线程列表
export function getThreadList(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/thread`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取线程堆栈
export function getThreadStack(params: ArthasBaseParams & { threadId?: string }) {
  return request({
    url: `${BASE_URL}/thread/stack`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取JVM信息
export function getJvmInfo(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/jvm`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取系统环境变量
export function getSysEnv(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/sysenv`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取系统属性
export function getSysProp(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/sysprop`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取性能计数器
export function getPerfCounter(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/perfcounter`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取内存信息
export function getMemory(params: ArthasBaseParams) {
  return request({
    url: `${BASE_URL}/memory`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 反编译类（查看源码）
export function decompileClass(params: ArthasBaseParams & { className: string }) {
  return request({
    url: `${BASE_URL}/jad`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 获取静态字段
export function getStaticField(params: ArthasBaseParams & { className: string; fieldName?: string }) {
  return request({
    url: `${BASE_URL}/getstatic`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 搜索类
export function searchClass(params: ArthasBaseParams & { pattern: string }) {
  return request({
    url: `${BASE_URL}/sc`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 搜索方法
export function searchMethod(params: ArthasBaseParams & { className: string; methodName?: string }) {
  return request({
    url: `${BASE_URL}/sm`,
    method: 'get',
    params,
    timeout: ARTHAS_TIMEOUT
  })
}

// 生成火焰图
export function generateFlameGraph(params: ArthasBaseParams & {
  duration?: string
  event?: string        // cpu, alloc, lock, wall
  threadId?: string     // 指定线程ID
  includeThreads?: string  // 是否按线程分组
}) {
  return request({
    url: `${BASE_URL}/profiler`,
    method: 'get',
    params,
    timeout: 300000 // 5分钟超时
  })
}

// 创建Arthas WebSocket连接
export function createArthasWebSocket(params: ArthasBaseParams): WebSocket {
  const token = localStorage.getItem('token')
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host

  const queryParams = new URLSearchParams({
    clusterId: String(params.clusterId),
    namespace: params.namespace,
    pod: params.pod,
    container: params.container,
    processId: params.processId || '',
    token: token || ''
  })

  return new WebSocket(`${protocol}//${host}${BASE_URL}/ws?${queryParams.toString()}`)
}

// WebSocket消息类型
export interface ArthasWSMessage {
  type: 'command' | 'stop'
  command?: string
}

// WebSocket响应类型
export interface ArthasWSResponse {
  type: 'output' | 'info' | 'error'
  content: string
}
