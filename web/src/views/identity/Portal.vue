<template>
  <div class="portal-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Grid /></el-icon>
        </div>
        <div>
          <h2 class="page-title">应用门户</h2>
          <p class="page-subtitle">一站式访问所有运维应用，支持收藏和快速搜索</p>
        </div>
      </div>
      <div class="header-stats">
        <div class="stat-card">
          <span class="stat-value">{{ apps.length }}</span>
          <span class="stat-label">可用应用</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{{ favoriteApps.length }}</span>
          <span class="stat-label">已收藏</span>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 搜索和筛选栏 -->
      <div class="filter-bar">
        <div class="filter-left">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索应用名称..."
            clearable
            class="search-input"
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
        <div class="filter-right">
          <el-radio-group v-model="selectedCategory" @change="loadApps" class="category-group">
            <el-radio-button label="">全部</el-radio-button>
            <el-radio-button label="cicd">CI/CD</el-radio-button>
            <el-radio-button label="code">代码管理</el-radio-button>
            <el-radio-button label="monitor">监控告警</el-radio-button>
            <el-radio-button label="registry">镜像仓库</el-radio-button>
            <el-radio-button label="other">其他</el-radio-button>
          </el-radio-group>
        </div>
      </div>

      <!-- 收藏的应用 -->
      <div v-if="favoriteApps.length > 0 && !selectedCategory && !searchKeyword" class="app-section">
        <div class="section-header">
          <div class="section-title">
            <el-icon class="section-icon" color="#d4af37"><StarFilled /></el-icon>
            <span>我的收藏</span>
          </div>
        </div>
        <div class="app-grid">
          <div
            v-for="app in favoriteApps"
            :key="'fav-' + app.id"
            class="app-card favorite"
            @click="handleAccessApp(app)"
          >
            <div class="app-card-header">
              <div class="app-icon">
                <img v-if="app.icon" :src="app.icon" :alt="app.name" />
                <el-icon v-else :size="24" color="#d4af37"><Grid /></el-icon>
              </div>
              <el-button
                link
                @click.stop="handleToggleFavorite(app)"
                class="favorite-btn active"
              >
                <el-icon :size="18"><StarFilled /></el-icon>
              </el-button>
            </div>
            <div class="app-card-body">
              <h4 class="app-name">{{ app.name }}</h4>
              <p class="app-desc">{{ app.description || '暂无描述' }}</p>
            </div>
            <div class="app-card-footer">
              <span class="category-tag">{{ getCategoryLabel(app.category) }}</span>
              <el-icon class="access-icon"><Right /></el-icon>
            </div>
          </div>
        </div>
      </div>

      <!-- 所有应用 -->
      <div class="app-section">
        <div class="section-header">
          <div class="section-title">
            <el-icon class="section-icon"><Grid /></el-icon>
            <span>{{ selectedCategory ? getCategoryLabel(selectedCategory) : '全部应用' }}</span>
            <el-tag size="small" type="info" style="margin-left: 8px;">{{ filteredApps.length }} 个</el-tag>
          </div>
        </div>
        <div v-if="filteredApps.length > 0" class="app-grid" v-loading="loading">
          <div
            v-for="app in filteredApps"
            :key="app.id"
            class="app-card"
            @click="handleAccessApp(app)"
          >
            <div class="app-card-header">
              <div class="app-icon">
                <img v-if="app.icon" :src="app.icon" :alt="app.name" />
                <el-icon v-else :size="24" color="#d4af37"><Grid /></el-icon>
              </div>
              <el-button
                link
                @click.stop="handleToggleFavorite(app)"
                :class="['favorite-btn', { active: app.isFavorite }]"
              >
                <el-icon :size="18">
                  <StarFilled v-if="app.isFavorite" />
                  <Star v-else />
                </el-icon>
              </el-button>
            </div>
            <div class="app-card-body">
              <h4 class="app-name">{{ app.name }}</h4>
              <p class="app-desc">{{ app.description || '暂无描述' }}</p>
            </div>
            <div class="app-card-footer">
              <span class="category-tag">{{ getCategoryLabel(app.category) }}</span>
              <el-icon class="access-icon"><Right /></el-icon>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无应用" :image-size="120" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Star, StarFilled, Grid, Search, Right } from '@element-plus/icons-vue'
import { getPortalApps, toggleFavoriteApp, accessApp, type PortalApp } from '@/api/identity'

const apps = ref<PortalApp[]>([])
const searchKeyword = ref('')
const selectedCategory = ref('')
const loading = ref(false)

const categoryMap: Record<string, string> = {
  cicd: 'CI/CD',
  code: '代码管理',
  monitor: '监控告警',
  registry: '镜像仓库',
  other: '其他'
}

const getCategoryLabel = (category: string) => {
  return categoryMap[category] || category || '未分类'
}

const favoriteApps = computed(() => {
  return apps.value.filter(app => app.isFavorite)
})

const filteredApps = computed(() => {
  let result = apps.value
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(app =>
      app.name.toLowerCase().includes(keyword) ||
      app.code?.toLowerCase().includes(keyword) ||
      (app.description && app.description.toLowerCase().includes(keyword))
    )
  }
  return result
})

const handleSearch = () => {
  // 搜索由computed属性自动处理
}

const loadApps = async () => {
  loading.value = true
  try {
    const res = await getPortalApps({ category: selectedCategory.value })
    if (res.data.code === 0) {
      apps.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载应用列表失败:', error)
  } finally {
    loading.value = false
  }
}

const handleAccessApp = async (app: PortalApp) => {
  try {
    const res = await accessApp(app.id)
    if (res.data.code === 0 && res.data.data?.url) {
      window.open(res.data.data.url, '_blank')
    }
  } catch (error) {
    ElMessage.error('访问应用失败')
  }
}

const handleToggleFavorite = async (app: PortalApp) => {
  try {
    const res = await toggleFavoriteApp(app.id)
    if (res.data.code === 0) {
      app.isFavorite = res.data.data?.isFavorite ?? !app.isFavorite
      ElMessage.success(app.isFavorite ? '已收藏' : '已取消收藏')
    }
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

onMounted(() => {
  loadApps()
})
</script>

<style scoped>
.portal-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 页面头部 */
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

.header-stats {
  display: flex;
  gap: 16px;
}

.stat-card {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  border-radius: 8px;
  padding: 12px 20px;
  text-align: center;
  min-width: 80px;
}

.stat-value {
  display: block;
  font-size: 24px;
  font-weight: 700;
  color: #d4af37;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

/* 主内容区域 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 筛选栏 */
.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.filter-left {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.category-group :deep(.el-radio-button__inner) {
  border-color: #dcdfe6;
}

.category-group :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background-color: #000;
  border-color: #000;
  color: #d4af37;
  box-shadow: -1px 0 0 0 #000;
}

/* 应用区块 */
.app-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.section-header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #ebeef5;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.section-icon {
  font-size: 18px;
}

/* 应用网格 */
.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

/* 应用卡片 */
.app-card {
  background: #fafafa;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
}

.app-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 16px rgba(212, 175, 55, 0.15);
  transform: translateY(-2px);
}

.app-card.favorite {
  background: linear-gradient(135deg, #fffef8 0%, #fafafa 100%);
  border-color: rgba(212, 175, 55, 0.4);
}

.app-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.app-icon {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.favorite-btn {
  padding: 4px;
  color: #c0c4cc;
}

.favorite-btn.active {
  color: #d4af37;
}

.favorite-btn:hover {
  color: #d4af37;
}

.app-card-body {
  flex: 1;
  margin-bottom: 12px;
}

.app-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 6px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.app-desc {
  font-size: 12px;
  color: #909399;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.5;
}

.app-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.category-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: rgba(212, 175, 55, 0.1);
  color: #b8960c;
  border: 1px solid rgba(212, 175, 55, 0.2);
}

.access-icon {
  color: #c0c4cc;
  transition: all 0.3s;
}

.app-card:hover .access-icon {
  color: #d4af37;
  transform: translateX(4px);
}
</style>
