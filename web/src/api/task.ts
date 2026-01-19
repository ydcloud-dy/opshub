import request from '@/utils/request'

// ==================== 任务作业 ====================

export interface JobTask {
  id: number
  name: string
  templateId?: number
  taskType: string
  status: string
  targetHosts?: string
  parameters?: string
  executeTime?: string
  result?: string
  errorMessage?: string
  createdBy: number
  createdAt: string
  updatedAt: string
}

export interface JobTaskListParams {
  page?: number
  pageSize?: number
  keyword?: string
  taskType?: string
  status?: string
}

export const getJobTaskList = (params: JobTaskListParams) => {
  return request.get<any, any>('/api/v1/plugins/jobs', { params })
}

export const getJobTaskDetail = (id: number) => {
  return request.get<any, JobTask>(`/api/v1/plugins/jobs/${id}`)
}

export const createJobTask = (data: any) => {
  return request.post<any, JobTask>('/api/v1/plugins/jobs', data)
}

export const updateJobTask = (id: number, data: any) => {
  return request.put<any, JobTask>(`/api/v1/plugins/jobs/${id}`, data)
}

export const deleteJobTask = (id: number) => {
  return request.delete<any, any>(`/api/v1/plugins/jobs/${id}`)
}

// ==================== 任务模板 ====================

export interface JobTemplate {
  id: number
  name: string
  code: string
  description?: string
  content: string
  variables?: string
  category: string
  platform?: string
  timeout: number
  sort: number
  status: number
  createdBy: number
  createdAt: string
  updatedAt: string
}

export interface JobTemplateListParams {
  page?: number
  pageSize?: number
  keyword?: string
  category?: string
  platform?: string
  status?: number
}

export const getJobTemplateList = (params: JobTemplateListParams) => {
  return request.get<any, any>('/api/v1/plugins/templates', { params })
}

export const getAllJobTemplates = (category?: string) => {
  return request.get<any, JobTemplate[]>('/api/v1/plugins/templates/all', { params: { category } })
}

export const getJobTemplateDetail = (id: number) => {
  return request.get<any, JobTemplate>(`/api/v1/plugins/templates/${id}`)
}

export const createJobTemplate = (data: any) => {
  return request.post<any, JobTemplate>('/api/v1/plugins/templates', data)
}

export const updateJobTemplate = (id: number, data: any) => {
  return request.put<any, JobTemplate>(`/api/v1/plugins/templates/${id}`, data)
}

export const deleteJobTemplate = (id: number) => {
  return request.delete<any, any>(`/api/v1/plugins/templates/${id}`)
}

// ==================== Ansible任务 ====================

export interface AnsibleTask {
  id: number
  name: string
  playbookContent?: string
  playbookPath?: string
  inventory?: string
  extraVars?: string
  tags?: string
  fork: number
  timeout: number
  verbose: string
  status: string
  lastRunTime?: string
  lastRunResult?: string
  createdBy: number
  createdAt: string
  updatedAt: string
}

export interface AnsibleTaskListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
}

export const getAnsibleTaskList = (params: AnsibleTaskListParams) => {
  return request.get<any, any>('/api/v1/plugins/ansible', { params })
}

export const getAnsibleTaskDetail = (id: number) => {
  return request.get<any, AnsibleTask>(`/api/v1/plugins/ansible/${id}`)
}

export const createAnsibleTask = (data: any) => {
  return request.post<any, AnsibleTask>('/api/v1/plugins/ansible', data)
}

export const updateAnsibleTask = (id: number, data: any) => {
  return request.put<any, AnsibleTask>(`/api/v1/plugins/ansible/${id}`, data)
}

export const deleteAnsibleTask = (id: number) => {
  return request.delete<any, any>(`/api/v1/plugins/ansible/${id}`)
}
