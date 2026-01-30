<template>
  <div class="logs-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">认证日志</h2>
          <p class="page-subtitle">查看用户登录和应用访问的审计记录</p>
        </div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon">
          <el-icon :size="22"><User /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.totalLogins }}</div>
          <div class="stat-label">总登录次数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon blue">
          <el-icon :size="22"><Calendar /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.todayLogins }}</div>
          <div class="stat-label">今日登录</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon red">
          <el-icon :size="22"><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.failedLogins }}</div>
          <div class="stat-label">失败次数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon green">
          <el-icon :size="22"><Grid /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.appAccessCount }}</div>
          <div class="stat-label">应用访问</div>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 搜索栏 -->
      <div class="filter-bar">
        <div class="filter-inputs">
          <el-select v-model="searchForm.action" placeholder="动作" clearable class="filter-input" @change="loadLogs">
            <el-option label="登录" value="login" />
            <el-option label="退出" value="logout" />
            <el-option label="访问应用" value="access_app" />
          </el-select>
          <el-select v-model="searchForm.result" placeholder="结果" clearable class="filter-input" @change="loadLogs">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
          <el-date-picker
            v-model="searchForm.dateRange"
            type="daterange"
            range-separator="-"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            class="date-picker"
            @change="loadLogs"
          />
        </div>
        <div class="filter-actions">
          <el-button class="black-button" @click="loadLogs">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
      </div>

      <!-- 表格 -->
      <div class="table-wrapper">
        <el-table :data="logList" v-loading="loading" border stripe>
          <el-table-column prop="username" label="用户" width="120" />
          <el-table-column label="动作" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="getActionTag(row.action)">
                {{ getActionLabel(row.action) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="appName" label="应用" min-width="120">
            <template #default="{ row }">
              {{ row.appName || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="loginType" label="登录方式" width="100">
            <template #default="{ row }">
              {{ row.loginType || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="ip" label="IP地址" width="130" />
          <el-table-column prop="location" label="位置" min-width="120">
            <template #default="{ row }">
              {{ row.location || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="结果" width="80" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
                {{ row.result === 'success' ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="failReason" label="失败原因" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              {{ row.failReason || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="时间" width="170" />
        </el-table>

        <!-- 分页 -->
        <div class="pagination-wrapper">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="loadLogs"
            @current-change="loadLogs"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Document, User, Calendar, Warning, Grid } from '@element-plus/icons-vue'
import { getAuthLogs, getAuthStats } from '@/api/identity'

const logList = ref<any[]>([])
const loading = ref(false)

const stats = reactive({
  totalLogins: 0,
  todayLogins: 0,
  failedLogins: 0,
  appAccessCount: 0
})

const searchForm = reactive({
  action: '',
  result: '',
  dateRange: null as [string, string] | null
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const getActionLabel = (action: string) => {
  const map: Record<string, string> = {
    login: '登录',
    logout: '退出',
    access_app: '访问应用'
  }
  return map[action] || action
}

const getActionTag = (action: string) => {
  const map: Record<string, string> = {
    login: 'success',
    logout: 'info',
    access_app: ''
  }
  return map[action] || ''
}

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
    console.error('加载日志失败:', error)
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    const res = await getAuthStats({})
    if (res.data.code === 0) {
      const data = res.data.data || {}
      stats.totalLogins = data.totalLogins || 0
      stats.todayLogins = data.todayLogins || 0
      stats.failedLogins = data.failedLogins || 0
      stats.appAccessCount = data.appAccessCount || 0
    }
  } catch (error) {
    console.error('加载统计失败:', error)
  }
}

const resetSearch = () => {
  searchForm.action = ''
  searchForm.result = ''
  searchForm.dateRange = null
  pagination.page = 1
  loadLogs()
}

onMounted(() => {
  loadLogs()
  loadStats()
})
</script>

<style scoped>
.logs-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
}

/* 统计卡片 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.stat-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon.blue {
  background: linear-gradient(135deg, #409eff 0%, #337ecc 100%);
  border-color: #409eff;
  color: #fff;
}

.stat-icon.red {
  background: linear-gradient(135deg, #f56c6c 0%, #c45656 100%);
  border-color: #f56c6c;
  color: #fff;
}

.stat-icon.green {
  background: linear-gradient(135deg, #67c23a 0%, #529b2e 100%);
  border-color: #67c23a;
  color: #fff;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

/* 主内容 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.filter-inputs {
  display: flex;
  gap: 12px;
}

.filter-input {
  width: 150px;
}

.date-picker {
  width: 260px;
}

.filter-actions {
  display: flex;
  gap: 8px;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

.black-button {
  background-color: #000 !important;
  border-color: #000 !important;
  color: #d4af37 !important;
}

.black-button:hover {
  background-color: #1a1a1a !important;
  border-color: #1a1a1a !important;
}
</style>
