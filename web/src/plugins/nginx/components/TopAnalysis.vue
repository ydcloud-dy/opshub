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
          <div v-loading="loadingUrls" class="table-container">
            <el-table :data="topUrls" v-if="topUrls.length > 0" class="custom-table">
              <el-table-column type="index" label="#" width="60" align="center">
                <template #default="{ $index }">
                  <span class="rank-badge" :class="getRankClass($index)">{{ $index + 1 }}</span>
                </template>
              </el-table-column>
              <el-table-column label="URL" prop="uri" min-width="400" show-overflow-tooltip>
                <template #default="{ row }">
                  <span class="url-text">{{ row.uri }}</span>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" width="220">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <div class="progress-bar-container">
                      <div
                        class="progress-bar"
                        :style="{ width: getPercent(row.count, totalUrlCount) + '%', backgroundColor: '#d4af37' }"
                      ></div>
                    </div>
                    <span class="progress-text">{{ getPercent(row.count, totalUrlCount).toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else description="暂无数据，请先采集日志" />
          </div>
        </el-tab-pane>

        <!-- Top IPs -->
        <el-tab-pane label="Top IPs" name="ips">
          <div v-loading="loadingIps" class="table-container">
            <el-table :data="topIps" v-if="topIps.length > 0" class="custom-table">
              <el-table-column type="index" label="#" width="60" align="center">
                <template #default="{ $index }">
                  <span class="rank-badge" :class="getRankClass($index)">{{ $index + 1 }}</span>
                </template>
              </el-table-column>
              <el-table-column label="IP 地址" prop="ip" width="160">
                <template #default="{ row }">
                  <span class="ip-text">{{ row.ip }}</span>
                </template>
              </el-table-column>
              <el-table-column label="国家/地区" width="110" align="center">
                <template #default="{ row }">
                  <el-tag v-if="row.country && row.country !== '-'" size="small" effect="plain" type="info">
                    {{ row.country }}
                  </el-tag>
                  <span v-else class="no-data">-</span>
                </template>
              </el-table-column>
              <el-table-column label="省份" width="100" align="center">
                <template #default="{ row }">
                  <span v-if="row.province && row.province !== '-'" class="geo-text">{{ row.province }}</span>
                  <span v-else class="no-data">-</span>
                </template>
              </el-table-column>
              <el-table-column label="城市" width="100" align="center">
                <template #default="{ row }">
                  <span v-if="row.city && row.city !== '-'" class="geo-text">{{ row.city }}</span>
                  <span v-else class="no-data">-</span>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="120" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" width="200">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <div class="progress-bar-container">
                      <div
                        class="progress-bar"
                        :style="{ width: getPercent(row.count, totalIpCount) + '%', backgroundColor: '#409EFF' }"
                      ></div>
                    </div>
                    <span class="progress-text">{{ getPercent(row.count, totalIpCount).toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else description="暂无数据，请先采集日志" />
          </div>
        </el-tab-pane>

        <!-- Top Browsers -->
        <el-tab-pane label="浏览器分布" name="browsers">
          <div v-loading="loadingBrowsers" class="table-container">
            <el-table :data="browserStats" v-if="browserStats.length > 0" class="custom-table">
              <el-table-column type="index" label="#" width="60" align="center">
                <template #default="{ $index }">
                  <span class="rank-badge" :class="getRankClass($index)">{{ $index + 1 }}</span>
                </template>
              </el-table-column>
              <el-table-column label="浏览器" width="200">
                <template #default="{ row }">
                  <div class="browser-cell">
                    <div class="browser-icon" :style="{ backgroundColor: getBrowserColor(row.browser) }">
                      {{ getBrowserEmoji(row.browser) }}
                    </div>
                    <span class="browser-name">{{ row.browser || '未知' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" min-width="250">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <div class="progress-bar-container">
                      <div
                        class="progress-bar"
                        :style="{ width: row.percent + '%', backgroundColor: getBrowserColor(row.browser) }"
                      ></div>
                    </div>
                    <span class="progress-text">{{ row.percent.toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else description="暂无浏览器数据，请重新采集日志以获取UA解析信息" />
          </div>
        </el-tab-pane>

        <!-- Top Devices -->
        <el-tab-pane label="设备分布" name="devices">
          <div v-loading="loadingDevices" class="table-container">
            <el-table :data="deviceStats" v-if="deviceStats.length > 0" class="custom-table">
              <el-table-column type="index" label="#" width="60" align="center">
                <template #default="{ $index }">
                  <span class="rank-badge" :class="getRankClass($index)">{{ $index + 1 }}</span>
                </template>
              </el-table-column>
              <el-table-column label="设备类型" width="200">
                <template #default="{ row }">
                  <div class="device-cell">
                    <div class="device-icon" :style="{ backgroundColor: getDeviceColor(row.deviceType) }">
                      <el-icon>
                        <component :is="getDeviceIcon(row.deviceType)" />
                      </el-icon>
                    </div>
                    <span class="device-name">{{ row.deviceType }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" prop="count" width="150" align="right">
                <template #default="{ row }">
                  <span class="count-cell">{{ formatNumber(row.count) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" min-width="250">
                <template #default="{ row }">
                  <div class="progress-wrapper">
                    <div class="progress-bar-container">
                      <div
                        class="progress-bar"
                        :style="{ width: row.percent + '%', backgroundColor: getDeviceColor(row.deviceType) }"
                      ></div>
                    </div>
                    <span class="progress-text">{{ row.percent.toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else description="暂无设备数据，请重新采集日志以获取UA解析信息" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Rank, Refresh, Monitor, Cellphone, Iphone, Platform } from '@element-plus/icons-vue'
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

const getRankClass = (index: number): string => {
  if (index === 0) return 'rank-gold'
  if (index === 1) return 'rank-silver'
  if (index === 2) return 'rank-bronze'
  return ''
}

const getBrowserColor = (browser: string): string => {
  const colors: Record<string, string> = {
    'Chrome': '#4285F4',
    'Firefox': '#FF7139',
    'Safari': '#000000',
    'Edge': '#0078D7',
    'IE': '#0078D7',
    'Opera': '#FF1B2D',
    'Samsung Browser': '#1428A0',
    'UC Browser': '#FF6600',
    'QQ Browser': '#12B7F5',
    'Other': '#909399'
  }
  return colors[browser] || '#909399'
}

const getBrowserEmoji = (browser: string): string => {
  const emojis: Record<string, string> = {
    'Chrome': 'C',
    'Firefox': 'F',
    'Safari': 'S',
    'Edge': 'E',
    'IE': 'I',
    'Opera': 'O',
    'Samsung Browser': 'S',
    'UC Browser': 'U',
    'QQ Browser': 'Q',
  }
  return emojis[browser] || '?'
}

const getDeviceColor = (type: string): string => {
  const typeLower = type?.toLowerCase() || ''
  if (typeLower.includes('桌面') || typeLower === 'desktop') return '#409EFF'
  if (typeLower.includes('移动') || typeLower === 'mobile') return '#67C23A'
  if (typeLower.includes('平板') || typeLower === 'tablet') return '#E6A23C'
  if (typeLower.includes('爬虫') || typeLower.includes('机器人') || typeLower === 'bot') return '#909399'
  return '#909399'
}

const getDeviceIcon = (type: string) => {
  const typeLower = type?.toLowerCase() || ''
  if (typeLower.includes('桌面') || typeLower === 'desktop') return Monitor
  if (typeLower.includes('移动') || typeLower === 'mobile') return Cellphone
  if (typeLower.includes('平板') || typeLower === 'tablet') return Iphone
  if (typeLower.includes('爬虫') || typeLower.includes('机器人') || typeLower === 'bot') return Platform
  return Monitor
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

.table-container {
  min-height: 300px;
}

.custom-table {
  border-radius: 8px;
  overflow: hidden;
}

.custom-table :deep(.el-table__header th) {
  background-color: #f8f9fa;
  font-weight: 600;
  color: #606266;
}

.custom-table :deep(.el-table__row) {
  transition: background-color 0.2s;
}

.custom-table :deep(.el-table__row:hover td) {
  background-color: #f5f7fa !important;
}

/* 排名徽章 */
.rank-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 13px;
  background-color: #f0f2f5;
  color: #606266;
}

.rank-badge.rank-gold {
  background: linear-gradient(135deg, #ffd700 0%, #ffb300 100%);
  color: #fff;
  box-shadow: 0 2px 6px rgba(255, 179, 0, 0.4);
}

.rank-badge.rank-silver {
  background: linear-gradient(135deg, #c0c0c0 0%, #a0a0a0 100%);
  color: #fff;
  box-shadow: 0 2px 6px rgba(160, 160, 160, 0.4);
}

.rank-badge.rank-bronze {
  background: linear-gradient(135deg, #cd7f32 0%, #b06c2c 100%);
  color: #fff;
  box-shadow: 0 2px 6px rgba(176, 108, 44, 0.4);
}

/* URL 文本 */
.url-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: #606266;
}

/* IP 文本 */
.ip-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  font-weight: 500;
  color: #303133;
}

/* 地理信息文本 */
.geo-text {
  font-size: 13px;
  color: #606266;
}

.no-data {
  color: #c0c4cc;
}

/* 计数单元格 */
.count-cell {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

/* 进度条 */
.progress-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar-container {
  flex: 1;
  height: 8px;
  background-color: #f0f2f5;
  border-radius: 4px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-text {
  min-width: 55px;
  text-align: right;
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}

/* 浏览器单元格 */
.browser-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.browser-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 12px;
}

.browser-name {
  font-weight: 500;
  color: #303133;
}

/* 设备单元格 */
.device-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.device-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 14px;
}

.device-name {
  font-weight: 500;
  color: #303133;
}

/* Tab 样式 */
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

/* 响应式 */
@media (max-width: 1200px) {
  .header-actions {
    flex-wrap: wrap;
  }
}
</style>
