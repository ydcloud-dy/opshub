import request from '@/utils/request'

// ========== 告警通道 ==========

export interface AlertChannel {
  id?: number
  name: string
  channelType: 'email' | 'webhook' | 'wechat' | 'dingtalk' | 'feishu'
  enabled: boolean
  config: string
  createdAt?: string
  updatedAt?: string
}

export const getAlertChannels = () => {
  return request.get('/api/v1/plugins/monitor/alerts/channels')
}

export const getAlertChannel = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/alerts/channels/${id}`)
}

export const createAlertChannel = (data: AlertChannel) => {
  return request.post('/api/v1/plugins/monitor/alerts/channels', data)
}

export const updateAlertChannel = (id: number, data: AlertChannel) => {
  return request.put(`/api/v1/plugins/monitor/alerts/channels/${id}`, data)
}

export const deleteAlertChannel = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/alerts/channels/${id}`)
}

// ========== 告警接收人 ==========

export interface AlertReceiver {
  id?: number
  name: string
  email?: string
  phone?: string
  wechatId?: string
  dingtalkId?: string
  feishuId?: string
  userId?: number
  enableEmail: boolean
  enableWebhook: boolean
  enableWeChat: boolean
  enableDingTalk: boolean
  enableFeishu: boolean
  enableSystemMsg: boolean
  createdAt?: string
  updatedAt?: string
}

export const getAlertReceivers = () => {
  return request.get('/api/v1/plugins/monitor/alerts/receivers')
}

export const getAlertReceiver = (id: number) => {
  return request.get(`/api/v1/plugins/monitor/alerts/receivers/${id}`)
}

export const createAlertReceiver = (data: AlertReceiver) => {
  return request.post('/api/v1/plugins/monitor/alerts/receivers', data)
}

export const updateAlertReceiver = (id: number, data: AlertReceiver) => {
  return request.put(`/api/v1/plugins/monitor/alerts/receivers/${id}`, data)
}

export const deleteAlertReceiver = (id: number) => {
  return request.delete(`/api/v1/plugins/monitor/alerts/receivers/${id}`)
}

// ========== 告警日志 ==========

export interface AlertLog {
  id: number
  alertType: string
  domainMonitorId: number
  domain: string
  status: string
  message: string
  channelType: string
  errorMsg: string
  sentAt: string
  createdAt: string
}

export const getAlertLogs = (params?: { page?: number; pageSize?: number; domainMonitorId?: number; alertType?: string }) => {
  return request.get('/api/v1/plugins/monitor/alerts/logs', { params })
}

export const getAlertStats = () => {
  return request.get('/api/v1/plugins/monitor/alerts/logs/stats')
}
