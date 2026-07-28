<template>
  <div class="inventory-page unified-credentials-page">
    <PageHeader :icon="Lock" title="统一凭据中心" description="在同一入口管理应用密钥与主机 SSH 凭据，按用途隔离存储、授权和使用关系。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ applicationCredentialCount }}</strong><span>应用凭据</span></div>
          <div class="inventory-header-metric"><strong>{{ hostCredentialCount }}</strong><span>主机凭据</span></div>
          <div class="inventory-header-metric"><strong>AES-256</strong><span>密文存储</span></div>
          <div class="inventory-header-metric"><strong>双域</strong><span>用途隔离</span></div>
        </div>
      </template>
      <el-button :icon="Refresh" :loading="loading" @click="refreshAll">刷新</el-button>
      <el-button :icon="activeTab === 'application' ? Coin : Monitor" @click="openRelatedAssets">{{ activeTab === 'application' ? '资源与域名' : '主机管理' }}</el-button>
    </PageHeader>

    <div class="unified-credentials-surface">
      <el-tabs v-model="activeTab" class="unified-credentials-tabs" @tab-change="changeTab">
        <el-tab-pane name="application">
          <template #label><span class="unified-credentials-tab-label"><el-icon><Coin /></el-icon>应用与中间件凭据</span></template>
          <div class="unified-credentials-context">
            <strong>应用密钥域</strong><span>数据库账号、访问令牌、AccessKey 和中间件账号，支持按用户或角色授权、查看理由与审计。</span>
          </div>
          <ApplicationCredentials :key="`application-${refreshKey}`" embedded />
        </el-tab-pane>
        <el-tab-pane name="host">
          <template #label><span class="unified-credentials-tab-label"><el-icon><Monitor /></el-icon>主机 SSH 凭据</span></template>
          <div class="unified-credentials-context">
            <strong>主机连接域</strong><span>仅用于 SSH 密码或私钥认证，主机资源关联资产主机后会继承这里配置的连接凭据。</span>
          </div>
          <HostCredentials :key="`host-${refreshKey}`" embedded />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Coin, Lock, Monitor, Refresh } from '@element-plus/icons-vue'
import { getCredentialList } from '@/api/host'
import ApplicationCredentials from '@/plugins/app-inventory/components/Credentials.vue'
import PageHeader from '@/plugins/app-inventory/components/PageHeader.vue'
import { listCredentials as listApplicationCredentials } from '@/plugins/app-inventory/api'
import HostCredentials from '@/views/asset/Credentials.vue'

type CredentialTab = 'application' | 'host'

const route = useRoute()
const router = useRouter()
const activeTab = ref<CredentialTab>('application')
const loading = ref(false)
const refreshKey = ref(0)
const applicationCredentialCount = ref(0)
const hostCredentialCount = ref(0)

const routeDefaultTab = (): CredentialTab => {
  if (route.query.scope === 'host' || route.query.scope === 'application') return route.query.scope
  return route.path.startsWith('/asset/') ? 'host' : 'application'
}

const loadCounts = async () => {
  loading.value = true
  try {
    const [applicationData, hostData] = await Promise.all([
      listApplicationCredentials({ page: 1, page_size: 1 }).catch(() => ({ total: 0 } as any)),
      getCredentialList({ page: 1, pageSize: 1 }).catch(() => ({ total: 0 } as any)),
    ])
    applicationCredentialCount.value = applicationData?.total || 0
    hostCredentialCount.value = hostData?.total || 0
  } finally { loading.value = false }
}

const changeTab = (name: string | number) => {
  activeTab.value = String(name) as CredentialTab
  router.replace({ path: route.path, query: { ...route.query, scope: activeTab.value } })
}
const refreshAll = async () => { refreshKey.value += 1; await loadCounts() }
const openRelatedAssets = () => router.push(activeTab.value === 'application' ? '/app-inventory/resources' : '/asset/hosts')

watch(() => [route.path, route.query.scope], () => { activeTab.value = routeDefaultTab() })
onMounted(() => { activeTab.value = routeDefaultTab(); loadCounts() })
</script>
