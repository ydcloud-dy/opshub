export interface PluginMenuConfig {
  name: string
  path: string
  icon: string
  sort: number
  hidden: boolean
  parentPath: string
  permission?: string
}

export interface PluginRouteConfig {
  path: string
  name: string
  component: () => Promise<any>
  redirect?: string
  meta?: {
    title?: string
    icon?: string
    hidden?: boolean
    permission?: string
    activeMenu?: string
  }
  children?: PluginRouteConfig[]
}

export interface Plugin {
  name: string
  description: string
  version: string
  author: string
  install: () => void | Promise<void>
  uninstall: () => void | Promise<void>
  getMenus?: () => PluginMenuConfig[]
  getRoutes?: () => PluginRouteConfig[]
}

export interface PluginInfo {
  name: string
  description: string
  version: string
  author: string
  enabled?: boolean
}
