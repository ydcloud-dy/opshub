import type { FormInstance, FormItemRule } from 'element-plus'

export const parseTagList = (value?: string | string[]) => {
  if (Array.isArray(value)) return normalizeTagList(value)
  if (!value?.trim()) return []
  try {
    const parsed = JSON.parse(value)
    if (Array.isArray(parsed)) return normalizeTagList(parsed.map(String))
    if (parsed && typeof parsed === 'object' && typeof parsed.value === 'string') return normalizeTagList(parsed.value.split(','))
  } catch {
    return normalizeTagList(value.split(','))
  }
  return []
}

export const normalizeTagList = (values: string[]) => Array.from(new Set(values.map(value => value.trim()).filter(Boolean))).slice(0, 12)

export const serializeTagList = (values: string[]) => JSON.stringify(normalizeTagList(values))

export const validateOptionalURL: FormItemRule['validator'] = (_rule, value, callback) => {
  if (!value) return callback()
  try {
    const url = new URL(value)
    if (!['http:', 'https:'].includes(url.protocol)) return callback(new Error('仅支持 HTTP 或 HTTPS 地址'))
    callback()
  } catch {
    callback(new Error('请输入完整的 URL 地址'))
  }
}

export const validateDomainName: FormItemRule['validator'] = (_rule, value, callback) => {
	const normalized = String(value || '').trim().replace(/\.$/, '')
	if (!normalized || /^[a-z][a-z0-9+.-]*:\/\//i.test(normalized) || normalized.includes('/') || normalized.includes(':')) {
		return callback(new Error('只填写域名、内网主机名或 IPv4 地址，不要包含协议、端口和路径'))
	}
	const ipv4 = normalized.split('.')
	const validIPv4 = ipv4.length === 4 && ipv4.every(part => /^\d{1,3}$/.test(part) && Number(part) <= 255)
	const validHostname = /^(?=.{1,253}$)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/.test(normalized)
	validIPv4 || validHostname ? callback() : callback(new Error('请输入有效域名，例如 api.example.com 或 order-api'))
}

export const validateForm = async (formRef?: FormInstance) => Boolean(formRef && await formRef.validate().catch(() => false))
