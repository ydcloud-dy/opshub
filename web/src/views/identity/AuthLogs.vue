<template>
  <div class="auth-logs-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">认证日志</h2>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">
          <el-icon :size="24"><User /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.totalLogins }}</div>
          <div class="stat-label">总登录次数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">
          <el-icon :size="24"><Calendar /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.todayLogins }}</div>
          <div class="stat-label">今日登录</div>
        </div>
      </div>
      <div class="stat-card warning">
        <div class="stat-icon">
          <el-icon :size="24"><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.failedLogins }}</div>
          <div class="stat-label">失败次数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">
          <el-icon :size="24"><Grid /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.appAccessCount }}</div>
          <div class="stat-label">应用访问</div>
        </div>
      </div>
    </div>

    <!-- 搜索表单 -->
    <el-form :inline="true" :model="searchForm" class="search-form">
      <el-form-item label="动作">
        <el-select v-model="searchForm.action" placeholder="请选择" clearable>
          <el-option label="登录" value="login" />
          <el-option label="登出" value="logout" />
          <el-option label="访问应用" value="access_app" />
        </el-select>
      </el-form-item>
      <el-form-item label="结果">
        <el-select v-model="searchForm.result" placeholder="请选择" clearable>
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
        </el-select>
      </el-form-item>
      <el-form-item label="时间范围">
        <el-date-picker
          v-model="searchForm.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
        />
      </el-form-item>
      <el-form-item>
        <el-button class="black-button" @click="loadLogs">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 表格 -->
    <el-table :data="logList" border stripe v-loading="loading" style="width: 100%">
      <el-table-column label="时间" width="180" prop="createdAt" />
      <el-table-column label="用户" width="120" prop="username" />
      <el-table-column label="动作" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="getActionTag(row.action)">
            {{ getActionLabel(row.action) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="应用" width="120">
        <template #default="{ row }">
          {{ row.appName || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="登录类型" width="100">
        <template #default="{ row }">
          {{ row.loginType || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="IP地址" width="140" prop="ip" />
      <el-table-column label="地理位置" width="120" prop="location" show-overflow-tooltip />
      <el-table-column label="结果" width="80">
        <template #default="{ row }">
          <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
            {{ row.result === 'success' ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="失败原因" min-width="150" prop="failReason" show-overflow-tooltip />
    </el-table>

    <!-- 分页 -->
    <el-pagination
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="loadLogs"
      @current-change="loadLogs"
      style="margin-top: 20px; justify-content: center"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { User, Calendar, Warning, Grid } from '@element-plus/icons-vue'
import {
  getAuthLogs,
  getAuthLogStats,
  type AuthLog,
  type AuthLogStats
} from '@/api/identity'

const loading = ref(false)
const logList = ref<AuthLog[]>([])
const stats = ref<AuthLogStats>({
  totalLogins: 0,
  todayLogins: 0,
  failedLogins: 0,
  uniqueUsers: 0,
  appAccessCount: 0,
  loginTrend: [],
  topApps: [],
  topUsers: []
})

const searchForm = reactive({
  action: '',
  result: '',
  dateRange: [] as string[]
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const actionMap: Record<string, { label: string; tag: string }> = {
  login: { label: '登录', tag: '' },
  logout: { label: '登出', tag: 'info' },
  access_app: { label: '访问应用', tag: 'success' }
}

const getActionLabel = (action: string) => actionMap[action]?.label || action
const getActionTag = (action: string) => actionMap[action]?.tag || ''

// 加载日志列表
const loadLogs = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      action: searchForm.action,
      result: searchForm.result
    }
    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.startTime = searchForm.dateRange[0]
      params.endTime = searchForm.dateRange[1]
    }
    const res = await getAuthLogs(params)
    if (res.data.code === 0) {
      logList.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    console.error('加载日志列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载统计数据
const loadStats = async () => {
  try {
    const res = await getAuthLogStats()
    if (res.data.code === 0) {
      stats.value = res.data.data || stats.value
    }
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

const resetSearch = () => {
  searchForm.action = ''
  searchForm.result = ''
  searchForm.dateRange = []
  pagination.page = 1
  loadLogs()
}

onMounted(() => {
  loadLogs()
  loadStats()
})
</script>

<style scoped>
.auth-logs-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-card.warning .stat-value {
  color: #f56c6c;
}

.stat-icon {
  width: 48px;
  height: 48px;
  background: rgba(212, 175, 55, 0.1);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #d4af37;
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 13px;
  color: #909399;
}

.search-form {
  margin-bottom: 20px;
}

.black-button {
  background-color: #000 !important;
  border-color: #000 !important;
  color: #d4af37 !important;
}

.black-button:hover {
  background-color: #1a1a1a !important;
}
</style>
