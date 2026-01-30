<template>
  <div class="nginx-top-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Rank /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Top 分析</h2>
          <p class="page-subtitle">查看热门 URL、IP 和 Referer 统计</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select v-model="selectedSourceId" placeholder="选择数据源" style="width: 200px">
          <el-option
            v-for="source in sources"
            :key="source.id"
            :label="source.name"
            :value="source.id"
          />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          :shortcuts="dateShortcuts"
          value-format="YYYY-MM-DD"
          style="width: 280px"
        />
        <el-button class="black-button" @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="content-section">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- Top URLs -->
        <el-tab-pane label="Top URLs" name="urls">
          <div v-loading="loadingUrls">
            <el-table :data="topUrls" stripe>
              <el-table-column type="index" label="#" width="60" />
              <el-table-column label="URL" prop="uri" min-width="400" show-overflow-tooltip />
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" width="200">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <el-progress
                      :percentage="getPercent(row.count, totalUrlCount)"
                      :stroke-width="10"
                      :show-text="false"
                      color="#d4af37"
                    />
                    <span class="progress-text">{{ getPercent(row.count, totalUrlCount).toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Top IPs -->
        <el-tab-pane label="Top IPs" name="ips">
          <div v-loading="loadingIps">
            <el-table :data="topIps" stripe>
              <el-table-column type="index" label="#" width="60" />
              <el-table-column label="IP 地址" prop="ip" width="180" />
              <el-table-column label="国家/地区" width="120">
                <template #default="{ row }">
                  {{ row.country || '-' }}
                </template>
              </el-table-column>
              <el-table-column label="省份" width="100">
                <template #default="{ row }">
                  {{ row.province || '-' }}
                </template>
              </el-table-column>
              <el-table-column label="城市" width="100">
                <template #default="{ row }">
                  {{ row.city || '-' }}
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" width="200">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <el-progress
                      :percentage="getPercent(row.count, totalIpCount)"
                      :stroke-width="10"
                      :show-text="false"
                      color="#409EFF"
                    />
                    <span class="progress-text">{{ getPercent(row.count, totalIpCount).toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Top Browsers -->
        <el-tab-pane label="浏览器分布" name="browsers">
          <div v-loading="loadingBrowsers">
            <el-table :data="browserStats" stripe>
              <el-table-column type="index" label="#" width="60" />
              <el-table-column label="浏览器" width="180">
                <template #default="{ row }">
                  <div class="browser-cell">
                    <span class="browser-icon" :style="{ backgroundColor: getBrowserColor(row.browser) }"></span>
                    {{ row.browser || '未知' }}
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" min-width="200">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <el-progress
                      :percentage="row.percent"
                      :stroke-width="10"
                      :show-text="false"
                      :color="getBrowserColor(row.browser)"
                    />
                    <span class="progress-text">{{ row.percent.toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Top Devices -->
        <el-tab-pane label="设备分布" name="devices">
          <div v-loading="loadingDevices">
            <el-table :data="deviceStats" stripe>
              <el-table-column type="index" label="#" width="60" />
              <el-table-column label="设备类型" width="180">
                <template #default="{ row }">
                  <div class="device-cell">
                    <el-icon :style="{ color: getDeviceColor(row.deviceType) }">
                      <component :is="getDeviceIcon(row.deviceType)" />
                    </el-icon>
                    {{ getDeviceName(row.deviceType) }}
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" min-width="200">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <el-progress
                      :percentage="row.percent"
                      :stroke-width="10"
                      :show-text="false"
                      :color="getDeviceColor(row.deviceType)"
                    />
                    <span class="progress-text">{{ row.percent.toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Rank, Refresh, Monitor, Cellphone, Iphone } from '@element-plus/icons-vue'
import {
  getNginxSources,
  getNginxTopURLs,
  getNginxTopIPsWithGeo,
  getNginxBrowserDistribution,
  getNginxDeviceDistribution,
  type NginxSource,
  type BrowserStats,
  type DeviceStats,
  type TopIPWithGeo,
} from '@/api/nginx'

const selectedSourceId = ref<number | undefined>(undefined)
const sources = ref<NginxSource[]>([])
const dateRange = ref<[string, string] | null>(null)
const activeTab = ref('urls')

const loadingUrls = ref(false)
const loadingIps = ref(false)
const loadingBrowsers = ref(false)
const loadingDevices = ref(false)

const topUrls = ref<Array<{ uri: string; count: number }>>([])
const topIps = ref<TopIPWithGeo[]>([])
const browserStats = ref<BrowserStats[]>([])
const deviceStats = ref<DeviceStats[]>([])

const dateShortcuts = [
  { text: '今天', value: () => { const d = new Date(); return [d, d] } },
  { text: '最近7天', value: () => { const d = new Date(); return [new Date(d.getTime() - 7 * 24 * 3600 * 1000), d] } },
  { text: '最近30天', value: () => { const d = new Date(); return [new Date(d.getTime() - 30 * 24 * 3600 * 1000), d] } },
]

const totalUrlCount = computed(() => {
  return topUrls.value.reduce((sum, item) => sum + item.count, 0)
})

const totalIpCount = computed(() => {
  return topIps.value.reduce((sum, item) => sum + item.count, 0)
})

// 加载数据源列表
const loadSources = async () => {
  try {
    const res = await getNginxSources({ pageSize: 100, status: 1 })
    sources.value = res.list || res || []
    if (sources.value.length > 0 && !selectedSourceId.value) {
      selectedSourceId.value = sources.value[0].id
    }
  } catch (error) {
    console.error('获取数据源列表失败:', error)
  }
}

// 构建查询参数
const buildParams = () => {
  const params: any = {
    sourceId: selectedSourceId.value,
    limit: 50
  }
  if (dateRange.value) {
    params.startTime = dateRange.value[0] + ' 00:00:00'
    params.endTime = dateRange.value[1] + ' 23:59:59'
  }
  return params
}

// 加载所有数据
const loadData = () => {
  if (!selectedSourceId.value) {
    ElMessage.warning('请选择数据源')
    return
  }

  switch (activeTab.value) {
    case 'urls':
      loadTopUrls()
      break
    case 'ips':
      loadTopIps()
      break
    case 'browsers':
      loadBrowsers()
      break
    case 'devices':
      loadDevices()
      break
  }
}

// Tab 切换
const handleTabChange = () => {
  loadData()
}

// 加载 Top URLs
const loadTopUrls = async () => {
  if (!selectedSourceId.value) return
  loadingUrls.value = true
  try {
    const params = buildParams()
    const res = await getNginxTopURLs(params)
    topUrls.value = res || []
  } catch (error) {
    console.error('获取 Top URLs 失败:', error)
    topUrls.value = []
  } finally {
    loadingUrls.value = false
  }
}

// 加载 Top IPs
const loadTopIps = async () => {
  if (!selectedSourceId.value) return
  loadingIps.value = true
  try {
    const params = buildParams()
    const res = await getNginxTopIPsWithGeo(params)
    topIps.value = res || []
  } catch (error) {
    console.error('获取 Top IPs 失败:', error)
    topIps.value = []
  } finally {
    loadingIps.value = false
  }
}

// 加载浏览器分布
const loadBrowsers = async () => {
  loadingBrowsers.value = true
  try {
    const params: any = {}
    if (selectedSourceId.value) {
      params.sourceId = selectedSourceId.value
    }
    if (dateRange.value) {
      params.startTime = dateRange.value[0]
      params.endTime = dateRange.value[1]
    }
    const res = await getNginxBrowserDistribution(params)
    browserStats.value = res || []
  } catch (error) {
    console.error('获取浏览器分布失败:', error)
    browserStats.value = []
  } finally {
    loadingBrowsers.value = false
  }
}

// 加载设备分布
const loadDevices = async () => {
  loadingDevices.value = true
  try {
    const params: any = {}
    if (selectedSourceId.value) {
      params.sourceId = selectedSourceId.value
    }
    if (dateRange.value) {
      params.startTime = dateRange.value[0]
      params.endTime = dateRange.value[1]
    }
    const res = await getNginxDeviceDistribution(params)
    deviceStats.value = res || []
  } catch (error) {
    console.error('获取设备分布失败:', error)
    deviceStats.value = []
  } finally {
    loadingDevices.value = false
  }
}

// 辅助函数
const formatNumber = (num: number): string => {
  return num?.toLocaleString() || '0'
}

const getPercent = (count: number, total: number): number => {
  if (!total) return 0
  return (count / total) * 100
}

const getBrowserColor = (browser: string): string => {
  const colors: Record<string, string> = {
    'Chrome': '#4285F4',
    'Firefox': '#FF7139',
    'Safari': '#000000',
    'Edge': '#0078D7',
    'IE': '#0078D7',
    'Opera': '#FF1B2D'
  }
  return colors[browser] || '#909399'
}

const getDeviceName = (type: string): string => {
  const names: Record<string, string> = {
    'desktop': '桌面端',
    'mobile': '移动端',
    'tablet': '平板',
    'bot': '机器人'
  }
  return names[type] || type
}

const getDeviceColor = (type: string): string => {
  const colors: Record<string, string> = {
    'desktop': '#409EFF',
    'mobile': '#67C23A',
    'tablet': '#E6A23C',
    'bot': '#909399'
  }
  return colors[type] || '#909399'
}

const getDeviceIcon = (type: string) => {
  const icons: Record<string, any> = {
    'desktop': Monitor,
    'mobile': Cellphone,
    'tablet': Iphone,
    'bot': Monitor
  }
  return icons[type] || Monitor
}

// 监听筛选条件变化
watch([selectedSourceId, dateRange], () => {
  loadData()
})

onMounted(() => {
  loadSources()
})
</script>

<style scoped>
.nginx-top-container {
  padding: 0;
  background-color: transparent;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
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
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

.content-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.count-cell {
  font-weight: 600;
  color: #303133;
}

.progress-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-wrapper :deep(.el-progress) {
  flex: 1;
}

.progress-text {
  min-width: 50px;
  text-align: right;
  font-size: 13px;
  color: #606266;
}

.browser-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.browser-icon {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.device-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.content-section :deep(.el-tabs__item) {
  font-size: 14px;
  font-weight: 500;
}

.content-section :deep(.el-tabs__item.is-active) {
  color: #d4af37;
}

.content-section :deep(.el-tabs__active-bar) {
  background-color: #d4af37;
}

@media (max-width: 1200px) {
  .header-actions {
    flex-wrap: wrap;
  }
}
</style>
