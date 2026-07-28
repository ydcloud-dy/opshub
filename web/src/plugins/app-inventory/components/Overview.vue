<template>
  <div class="inventory-page">
    <PageHeader :icon="DataAnalysis" title="应用资产总览" description="从应用归属出发，查看运行环境、入口、部署资源和依赖健康度。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ counts.applications || 0 }}</strong><span>应用</span></div>
          <div class="inventory-header-metric"><strong>{{ counts.environments || 0 }}</strong><span>环境</span></div>
          <div class="inventory-header-metric"><strong>{{ counts.resources || 0 }}</strong><span>部署资源</span></div>
          <div class="inventory-header-metric"><strong>{{ counts.dependencies || 0 }}</strong><span>调用关系</span></div>
        </div>
      </template>
      <el-button :icon="Refresh" @click="loadData">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="router.push('/app-inventory/apps?create=1')">登记应用</el-button>
    </PageHeader>

    <div class="inventory-stat-grid">
      <div class="inventory-stat">
        <div class="inventory-stat__label">应用总数</div>
        <div class="inventory-stat__value">{{ counts.applications || 0 }}</div>
        <div class="inventory-stat__hint">已纳入台账的服务边界</div>
      </div>
      <div class="inventory-stat">
        <div class="inventory-stat__label">运行环境</div>
        <div class="inventory-stat__value">{{ counts.environments || 0 }}</div>
        <div class="inventory-stat__hint">开发、测试、生产等</div>
      </div>
      <div class="inventory-stat">
        <div class="inventory-stat__label">入口域名</div>
        <div class="inventory-stat__value">{{ counts.domains || 0 }}</div>
        <div class="inventory-stat__hint">可关联 SSL 证书</div>
      </div>
      <div class="inventory-stat">
        <div class="inventory-stat__label">部署资源</div>
        <div class="inventory-stat__value">{{ counts.resources || 0 }}</div>
        <div class="inventory-stat__hint">主机与 Kubernetes 资源</div>
      </div>
      <div class="inventory-stat inventory-stat--warning">
        <div class="inventory-stat__label">证书 30 天内到期</div>
        <div class="inventory-stat__value">{{ counts.expiringCertificates || 0 }}</div>
        <div class="inventory-stat__hint">需要关注续期计划</div>
      </div>
      <div class="inventory-stat inventory-stat--danger">
        <div class="inventory-stat__label">异常资源</div>
        <div class="inventory-stat__value">{{ (counts.unhealthyResources || 0) + (counts.unhealthyComponents || 0) }}</div>
        <div class="inventory-stat__hint">资源 {{ counts.unhealthyResources || 0 }} / 组件 {{ counts.unhealthyComponents || 0 }}</div>
      </div>
    </div>

    <div class="inventory-two-column">
      <section class="inventory-panel">
        <div class="inventory-panel__heading">
          <h3>最近登记的应用</h3>
          <el-button link type="primary" @click="router.push('/app-inventory/apps')">查看全部</el-button>
        </div>
        <el-table v-if="recentApplications.length" :data="recentApplications" size="small" stripe @row-click="openApplication">
          <el-table-column prop="name" label="应用" min-width="180">
            <template #default="{ row }">
              <span class="inventory-link">{{ row.name }}</span>
              <div class="inventory-table-cell__sub">{{ row.code }}</div>
            </template>
          </el-table-column>
          <el-table-column label="所属部门" min-width="110" show-overflow-tooltip>
            <template #default="{ row }">{{ row.departmentName || row.team || '未关联部门' }}</template>
          </el-table-column>
          <el-table-column prop="healthStatus" label="健康状态" width="100">
            <template #default="{ row }"><StatusTag :status="row.healthStatus" /></template>
          </el-table-column>
          <el-table-column label="资产" width="180">
            <template #default="{ row }">
              环境 {{ row.environmentCount || 0 }} · 资源 {{ row.resourceCount || 0 }}
            </template>
          </el-table-column>
        </el-table>
        <div v-else class="inventory-empty">还没有登记应用，先建立第一个应用边界。</div>
      </section>

      <section class="inventory-panel">
        <div class="inventory-panel__heading">
          <h3>应用健康分布</h3>
          <span>根据台账中的健康状态统计</span>
        </div>
        <div v-for="item in healthItems" :key="item.key" style="margin-bottom: 16px">
          <div style="display:flex;justify-content:space-between;margin-bottom:6px;font-size:13px">
            <span>{{ item.label }}</span><strong>{{ item.value }}</strong>
          </div>
          <el-progress :percentage="healthPercent(item.value)" :status="item.status" :show-text="false" :stroke-width="8" />
        </div>
        <el-divider />
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;font-size:13px">
          <div><span class="inventory-muted">数据库与中间件</span><strong style="display:block;font-size:20px;margin-top:5px">{{ counts.components || 0 }}</strong></div>
          <div><span class="inventory-muted">调用关系</span><strong style="display:block;font-size:20px;margin-top:5px">{{ counts.dependencies || 0 }}</strong></div>
          <div><span class="inventory-muted">凭据条目</span><strong style="display:block;font-size:20px;margin-top:5px">{{ counts.credentials || 0 }}</strong></div>
          <div><span class="inventory-muted">凭据即将过期</span><strong style="display:block;font-size:20px;margin-top:5px;color:#d97706">{{ counts.expiringCredentials || 0 }}</strong></div>
        </div>
      </section>
    </div>

    <section class="inventory-panel" style="margin-top:14px">
      <div class="inventory-panel__heading">
        <h3>资产登记边界</h3>
        <span>建议先登记应用和环境，再补充资源与依赖</span>
      </div>
      <el-steps :active="activeStep" finish-status="success" align-center>
        <el-step title="应用台账" description="明确负责人、部门和重要级别" />
        <el-step title="环境与入口" description="关联域名、证书和环境" />
        <el-step title="部署资源" description="主机或 Kubernetes 工作负载" />
        <el-step title="依赖关系" description="补充数据库、中间件和调用链" />
      </el-steps>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { DataAnalysis, Plus, Refresh } from '@element-plus/icons-vue'
import { getOverview, type Application } from '../api'
import PageHeader from './PageHeader.vue'
import StatusTag from './StatusTag.vue'

const router = useRouter()
const loading = ref(false)
const counts = ref<Record<string, number>>({})
const health = ref<Record<string, number>>({})
const recentApplications = ref<Application[]>([])

const healthItems = computed(() => [
  { key: 'healthy', label: '健康', value: health.value.healthy || 0, status: 'success' as const },
  { key: 'warning', label: '关注', value: health.value.warning || 0, status: 'warning' as const },
  { key: 'unhealthy', label: '异常', value: health.value.unhealthy || 0, status: 'exception' as const },
  { key: 'unknown', label: '未知', value: health.value.unknown || 0, status: undefined },
])

const activeStep = computed(() => {
  if (!counts.value.applications) return 0
  if (!counts.value.domains && !counts.value.resources) return 1
  if (!counts.value.dependencies) return 3
  return 4
})

const healthPercent = (value: number) => {
  const total = Object.values(health.value).reduce((sum, item) => sum + item, 0)
  return total ? Math.round(value / total * 100) : 0
}

const loadData = async () => {
  loading.value = true
  try {
    const data = await getOverview()
    counts.value = data?.counts || {}
    health.value = data?.health || {}
    recentApplications.value = data?.recentApplications || []
  } catch (error) {
    ElMessage.error('加载应用资产总览失败')
  } finally {
    loading.value = false
  }
}

const openApplication = (row: Application) => router.push(`/app-inventory/apps/${row.id}`)

onMounted(loadData)
</script>

<style scoped>
.el-table { cursor: pointer; }
</style>
