import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * Kubernetes 容器管理插件
 * 提供集群管理、节点管理、工作负载、命名空间等完整功能
 */
class KubernetesPlugin implements Plugin {
  name = 'kubernetes'
  description = 'Kubernetes容器管理平台,提供集群管理、节点管理、工作负载、命名空间等完整功能'
  version = '1.0.0'
  author = 'OpsHub Team'

  /**
   * 安装插件
   */
  async install() {
    console.log('Kubernetes 插件安装中...')
    // 在这里可以进行一些初始化操作
    // 例如: 注册全局组件、配置等
  }

  /**
   * 卸载插件
   */
  async uninstall() {
    console.log('Kubernetes 插件卸载中...')
    // 清理插件创建的资源
  }

  /**
   * 获取插件菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    const parentPath = '/kubernetes'

    return [
      {
        name: 'Kubernetes管理',
        path: parentPath,
        icon: 'Platform',
        sort: 100,
        hidden: false,
        parentPath: '',
      },
      {
        name: '集群管理',
        path: '/kubernetes/clusters',
        icon: 'OfficeBuilding',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '节点管理',
        path: '/kubernetes/nodes',
        icon: 'Monitor',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '工作负载',
        path: '/kubernetes/workloads',
        icon: 'Tools',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '命名空间',
        path: '/kubernetes/namespaces',
        icon: 'FolderOpened',
        sort: 4,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '网络管理',
        path: '/kubernetes/network',
        icon: 'Connection',
        sort: 5,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '配置管理',
        path: '/kubernetes/config',
        icon: 'Document',
        sort: 6,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '存储管理',
        path: '/kubernetes/storage',
        icon: 'Files',
        sort: 7,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '访问控制',
        path: '/kubernetes/access',
        icon: 'Lock',
        sort: 8,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '终端审计',
        path: '/kubernetes/audit',
        icon: 'View',
        sort: 9,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '应用诊断',
        path: '/kubernetes/diagnostic',
        icon: 'Odometer',
        sort: 10,
        hidden: false,
        parentPath: parentPath,
      },
    ]
  }

  /**
   * 获取插件路由配置
   */
  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/kubernetes',
        name: 'Kubernetes',
        component: () => import('@/views/kubernetes/Index.vue'),
        meta: { title: 'Kubernetes管理' },
        children: [
          {
            path: 'clusters',
            name: 'K8sClusters',
            component: () => import('@/views/kubernetes/Clusters.vue'),
            meta: { title: '集群管理' },
          },
          {
            path: 'clusters/:id',
            name: 'K8sClusterDetail',
            component: () => import('@/views/kubernetes/ClusterDetail.vue'),
            meta: { title: '集群详情', activeMenu: '/kubernetes/clusters' },
          },
          {
            path: 'nodes',
            name: 'K8sNodes',
            component: () => import('@/views/kubernetes/Nodes.vue'),
            meta: { title: '节点管理' },
          },
          {
            path: 'clusters/:clusterId/nodes/:nodeName',
            name: 'K8sNodeDetail',
            component: () => import('@/views/kubernetes/NodeDetail.vue'),
            meta: { title: '节点详情', activeMenu: '/kubernetes/nodes' },
          },
          {
            path: 'workloads',
            name: 'K8sWorkloads',
            component: () => import('@/views/kubernetes/Workloads.vue'),
            meta: { title: '工作负载' },
          },
          {
            path: 'namespaces',
            name: 'K8sNamespaces',
            component: () => import('@/views/kubernetes/Namespaces.vue'),
            meta: { title: '命名空间' },
          },
          {
            path: 'roles',
            name: 'K8sRoles',
            component: () => import('@/views/kubernetes/Roles.vue'),
            meta: { title: '角色管理' },
          },
          {
            path: 'network',
            name: 'K8sNetwork',
            component: () => import('@/views/kubernetes/Network.vue'),
            meta: { title: '网络管理' },
          },
          {
            path: 'config',
            name: 'K8sConfig',
            component: () => import('@/views/kubernetes/Config.vue'),
            meta: { title: '配置管理' },
          },
          {
            path: 'storage',
            name: 'K8sStorage',
            component: () => import('@/views/kubernetes/Storage.vue'),
            meta: { title: '存储管理' },
          },
          {
            path: 'access',
            name: 'K8sAccess',
            component: () => import('@/views/kubernetes/Access.vue'),
            meta: { title: '访问控制' },
          },
          {
            path: 'audit',
            name: 'K8sAudit',
            component: () => import('@/views/kubernetes/Audit.vue'),
            meta: { title: '终端审计' },
          },
          {
            path: 'diagnostic',
            name: 'K8sDiagnostic',
            component: () => import('@/views/kubernetes/Diagnostic.vue'),
            meta: { title: '应用诊断' },
          },
        ],
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new KubernetesPlugin()
console.log('[Kubernetes Plugin] 正在注册插件...')
pluginManager.register(plugin)
console.log('[Kubernetes Plugin] 插件注册完成')

export default plugin
