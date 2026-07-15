<template>
  <div class="log-center-page log-overview-page">
    <div class="page-head">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DataAnalysis /></el-icon>
        </div>
        <div>
          <h2>日志总览</h2>
          <p>统一查看 OpsHub 自带采集链路、ClickHouse 存储和日志查询使用情况</p>
        </div>
      </div>
      <el-button type="primary" class="primary-action" @click="loadData">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <div v-loading="loading" class="overview-grid">
      <article class="metric-card">
        <div class="stat-icon stat-blue"><el-icon><Connection /></el-icon></div>
        <div>
          <span>采集链路</span>
          <strong class="status-text" :class="{ healthy: pipelineHealthy }">{{ pipelineHealthy ? '正常' : '异常' }}</strong>
          <em>Gateway、Writer 与 ClickHouse</em>
        </div>
      </article>
      <article class="metric-card">
        <div class="stat-icon stat-green"><el-icon><Search /></el-icon></div>
        <div>
          <span>今日查询</span>
          <strong>{{ overview.todayQueries || 0 }}</strong>
          <em>来自日志查询页和模板执行</em>
        </div>
      </article>
      <article class="metric-card">
        <div class="stat-icon stat-red"><el-icon><Warning /></el-icon></div>
        <div>
          <span>今日失败</span>
          <strong>{{ overview.todayFailures || 0 }}</strong>
          <em>查询语法、权限或存储异常</em>
        </div>
      </article>
      <article class="metric-card">
        <div class="stat-icon stat-amber"><el-icon><CollectionTag /></el-icon></div>
        <div>
          <span>查询模板</span>
          <strong>{{ overview.templateCount || 0 }}</strong>
          <em>可复用的内置结构化查询</em>
        </div>
      </article>
      <article class="metric-card">
        <div class="stat-icon stat-purple"><el-icon><FolderOpened /></el-icon></div>
        <div>
          <span>日志库</span>
          <strong>{{ overview.libraryCount || 0 }}</strong>
          <em>内置存储和字段元数据</em>
        </div>
      </article>
    </div>

    <div class="content-grid">
      <section class="panel">
        <div class="panel-head">
          <div>
            <h3>最近查询</h3>
            <p>最近执行的日志检索记录和结果状态</p>
          </div>
          <el-button link type="primary" @click="$router.push('/logs/query')">进入查询</el-button>
        </div>
        <el-table :data="overview.recentQueries || []" empty-text="暂无查询历史" height="300">
          <el-table-column prop="datasourceType" label="类型" width="120">
            <template #default>
              <el-tag size="small" type="info">内置日志</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="query" label="查询语句" min-width="260" show-overflow-tooltip />
          <el-table-column prop="resultCount" label="结果" width="90" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'success' ? 'success' : 'danger'">
                {{ row.status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="时间" width="170">
            <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
          </el-table-column>
        </el-table>
      </section>

      <section class="panel">
        <div class="panel-head">
          <div>
            <h3>常用模板</h3>
            <p>高频使用的结构化日志查询模板</p>
          </div>
          <el-button link type="primary" @click="$router.push('/logs/templates')">管理模板</el-button>
        </div>
        <div class="template-list">
          <div v-for="item in overview.hotTemplates || []" :key="item.id" class="template-item">
            <div>
              <strong>{{ item.name }}</strong>
              <small>{{ item.category || '常用查询' }} · OpsHub 内置日志</small>
            </div>
            <el-button link type="primary" @click="useTemplate(item)">执行</el-button>
          </div>
          <el-empty v-if="!overview.hotTemplates?.length" description="暂无查询模板" :image-size="96" />
        </div>
      </section>
    </div>

    <section class="panel">
      <div class="panel-head">
        <div>
          <h3>日志库元数据</h3>
          <p>内置 ClickHouse 日志资源与字段配置</p>
        </div>
        <el-button link type="primary" @click="$router.push('/logs/library')">查看日志库</el-button>
      </div>
      <el-table :data="overview.recentLibrary || []" empty-text="暂无日志库元数据">
        <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
        <el-table-column prop="itemType" label="类型" width="130">
          <template #default="{ row }">
            <el-tag size="small">{{ itemTypeName(row.itemType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="datasourceType" label="数据源" width="140">
          <template #default>
            <el-tag size="small" type="info">ClickHouse</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="owner" label="负责人" width="120">
          <template #default="{ row }">{{ row.owner || '-' }}</template>
        </el-table-column>
        <el-table-column prop="environment" label="环境" width="100">
          <template #default="{ row }">{{ row.environment || '-' }}</template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新时间" width="170">
          <template #default="{ row }">{{ formatTime(row.updatedAt) }}</template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { CollectionTag, Connection, DataAnalysis, FolderOpened, Refresh, Search, Warning } from '@element-plus/icons-vue'
import { getLogIngestStatus, getLogOverview, type LogIngestStatus } from '@/api/logcenter'

const router = useRouter()
const loading = ref(false)
const overview = ref<any>({})
const ingest = ref<LogIngestStatus>()
const pipelineHealthy = computed(() => Boolean(
  ingest.value?.gateway.reachable && ingest.value?.writer.reachable && ingest.value?.storage.reachable,
))

const loadData = async () => {
  loading.value = true
  try {
    const [overviewResult, ingestResult] = await Promise.all([getLogOverview(), getLogIngestStatus()])
    overview.value = overviewResult
    ingest.value = ingestResult as any
  } finally {
    loading.value = false
  }
}

const useTemplate = (item: any) => {
  router.push({
    path: '/logs/query',
    query: {
      templateId: item.id,
      datasourceId: item.datasourceId,
      datasourceType: item.datasourceType,
      queryLanguage: item.queryLanguage,
      index: item.index,
      q: item.query,
    },
  })
}

const itemTypeName = (type: string) => {
  const names: Record<string, string> = {
    index: '索引',
    alias: '别名',
    stream: '日志流',
    table: '日志表',
  }
  return names[type] || type
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

onMounted(loadData)
</script>

<style scoped>
.log-overview-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0;
  background: transparent;
  color: #111827;
}

.log-overview-page .page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin: 0;
  padding: 18px 20px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 22px;
}

.log-overview-page .page-head h2 {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.3;
}

.log-overview-page .page-head p {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.log-overview-page .primary-action.el-button,
.log-overview-page .primary-action.el-button:hover,
.log-overview-page .primary-action.el-button:focus {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.metric-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 108px;
  padding: 18px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.metric-card span {
  color: #667085;
  font-size: 13px;
  font-weight: 500;
}

.metric-card strong {
  display: block;
  margin: 7px 0;
  color: #111827;
  font-size: 32px;
  font-weight: 780;
  line-height: 1;
}

.metric-card strong.status-text {
  color: #dc2626;
  font-size: 25px;
}

.metric-card strong.status-text.healthy {
  color: #16a34a;
}

.metric-card em {
  display: block;
  color: #98a2b3;
  font-size: 13px;
  font-style: normal;
  font-weight: 500;
}

.stat-icon {
  width: 46px;
  height: 46px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  font-size: 23px;
}

.stat-blue { background: #eef4ff; border: 1px solid #dbe8ff; color: #2563eb; }
.stat-amber { background: #fffbeb; border: 1px solid #fde68a; color: #d97706; }
.stat-green { background: #ecfdf3; border: 1px solid #bbf7d0; color: #16a34a; }
.stat-red { background: #fff1f2; border: 1px solid #fecdd3; color: #e11d48; }
.stat-purple { background: #f5f3ff; border: 1px solid #ddd6fe; color: #7c3aed; }

.content-grid {
  display: grid;
  grid-template-columns: 1.45fr 1fr;
  gap: 14px;
}

.panel {
  padding: 18px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}

.panel-head h3 {
  margin: 0;
  color: #111827;
  font-size: 17px;
  font-weight: 780;
}

.panel-head p {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 300px;
}

.template-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border: 1px solid #eef2f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.template-item strong,
.template-item small {
  display: block;
}

.template-item strong {
  color: #111827;
}

.template-item small {
  margin-top: 5px;
  color: #667085;
}

@media (max-width: 1200px) {
  .overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .content-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .overview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
