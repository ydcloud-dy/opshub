import { ElMessage } from 'element-plus'

const READONLY_USERNAME = 'test'

const dangerousKeywords = [
  '新增',
  '添加',
  '创建',
  '编辑',
  '修改',
  '删除',
  '移除',
  '保存',
  '提交',
  '确定',
  '确认',
  '执行',
  '运行',
  '重启',
  '停止',
  '启动',
  '启用',
  '禁用',
  '测试',
  '连接测试',
  '下载',
  '上传',
  '导入',
  '导出',
  '发布',
  '同步',
  '授权',
  '分配',
  '解绑',
  '回滚',
  '部署',
  '续期',
  '采集',
  '终端',
  'shell',
  'yaml',
  'kubeconfig',
  '密码',
  '解锁',
]

const safeKeywords = [
  '查询',
  '搜索',
  '刷新',
  '重置',
  '查看',
  '详情',
  '关闭',
  '取消',
  '返回',
  '播放',
  '预览',
  '展开',
  '收起',
  '实时概览',
]

const dangerousClassFragments = [
  'delete',
  'remove',
  'edit',
  'save',
  'submit',
  'create',
  'add',
  'test',
  'download',
  'upload',
  'terminal',
  'shell',
  'yaml',
  'deploy',
  'execute',
  'danger',
]

export const isReadonlyUser = () => {
  const raw = localStorage.getItem('userInfo')
  if (!raw) return false

  try {
    const user = JSON.parse(raw)
    return String(user?.username || '').trim().toLowerCase() === READONLY_USERNAME
  } catch {
    return false
  }
}

export const isDangerousActionText = (text = '') => {
  const normalized = text.replace(/\s+/g, '').toLowerCase()
  if (!normalized) return false
  if (safeKeywords.some(keyword => normalized.includes(keyword.toLowerCase()))) {
    return false
  }
  return dangerousKeywords.some(keyword => normalized.includes(keyword.toLowerCase()))
}

const getElementText = (el: Element) => {
  const parts = [
    el.textContent || '',
    el.getAttribute('title') || '',
    el.getAttribute('aria-label') || '',
    el.getAttribute('data-readonly-action') || '',
    el.getAttribute('class') || '',
  ]
  return parts.join(' ')
}

const isDangerousElement = (el: Element) => {
  const text = getElementText(el)
  if (isDangerousActionText(text)) return true

  const className = String(el.getAttribute('class') || '').toLowerCase()
  return dangerousClassFragments.some(fragment => className.includes(fragment))
}

const markDisabled = (el: HTMLElement) => {
  if (!el.classList.contains('opshub-readonly-disabled')) {
    el.classList.add('opshub-readonly-disabled')
  }
  if (el.getAttribute('aria-disabled') !== 'true') {
    el.setAttribute('aria-disabled', 'true')
  }
  if (el.getAttribute('data-readonly-disabled') !== 'true') {
    el.setAttribute('data-readonly-disabled', 'true')
  }
  if (el instanceof HTMLButtonElement || el instanceof HTMLInputElement) {
    el.disabled = true
  }
}

const unmarkDisabled = (el: HTMLElement) => {
  if (el.classList.contains('opshub-readonly-disabled')) {
    el.classList.remove('opshub-readonly-disabled')
  }
  if (el.hasAttribute('aria-disabled')) {
    el.removeAttribute('aria-disabled')
  }
  if (el.hasAttribute('data-readonly-disabled')) {
    el.removeAttribute('data-readonly-disabled')
  }
  if ((el instanceof HTMLButtonElement || el instanceof HTMLInputElement) && el.disabled) {
    el.disabled = false
  }
}

export const applyReadonlyGuards = (root: ParentNode = document) => {
  const elements = root.querySelectorAll<HTMLElement>(
    [
      '.el-button',
      '.el-dropdown-menu__item',
      '.el-link',
      'button',
      '[role="button"]',
      '[data-readonly-action]',
    ].join(','),
  )

  elements.forEach(el => {
    if (!isReadonlyUser()) {
      unmarkDisabled(el)
      return
    }

    if (isDangerousElement(el)) {
      markDisabled(el)
    } else {
      unmarkDisabled(el)
    }
  })
}

const readonlyClickGuard = (event: Event) => {
  if (!isReadonlyUser()) return

  const target = event.target as Element | null
  const actionEl = target?.closest?.(
    [
      '.opshub-readonly-disabled',
      '.el-button',
      '.el-dropdown-menu__item',
      '.el-link',
      'button',
      '[role="button"]',
      '[data-readonly-action]',
    ].join(','),
  ) as HTMLElement | null

  if (!actionEl || !isDangerousElement(actionEl)) return

  event.preventDefault()
  event.stopPropagation()
  event.stopImmediatePropagation()
  ElMessage.warning('test账号仅允许查看，不能执行修改、测试、下载、终端或敏感配置操作')
}

export const installReadonlyGuards = () => {
  document.addEventListener('click', readonlyClickGuard, true)
  document.addEventListener('mousedown', readonlyClickGuard, true)
}
