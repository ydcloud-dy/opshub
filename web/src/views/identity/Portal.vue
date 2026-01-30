<template>
  <div class="portal-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">应用门户</h2>
      <div class="header-stats">
        <div class="stat-item">
          <span class="stat-value">{{ apps.length }}</span>
          <span class="stat-label">可用应用</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">{{ favoriteApps.length }}</span>
          <span class="stat-label">已收藏</span>
        </div>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="filter-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索应用..."
        prefix-icon="Search"
        clearable
        class="search-input"
      />
      <div class="category-tabs">
        <el-radio-group v-model="selectedCategory" @change="loadApps">
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
    <div v-if="favoriteApps.length > 0 && !selectedCategory" class="section">
      <div class="section-header">
        <h3 class="section-title">
          <el-icon><Star /></el-icon>
          我的收藏
        </h3>
      </div>
      <div class="app-grid">
        <div
          v-for="app in favoriteApps"
          :key="app.id"
          class="app-card favorite"
          @click="handleAccessApp(app)"
        >
          <div class="app-icon">
            <img v-if="app.icon" :src="app.icon" :alt="app.name" />
            <el-icon v-else :size="32"><Grid /></el-icon>
          </div>
          <div class="app-info">
            <h4 class="app-name">{{ app.name }}</h4>
            <p class="app-desc">{{ app.description || '暂无描述' }}</p>
          </div>
          <div class="app-actions">
            <el-button
              type="text"
              @click.stop="handleToggleFavorite(app)"
              class="favorite-btn active"
            >
              <el-icon><StarFilled /></el-icon>
            </el-button>
          </div>
          <div class="category-tag">{{ getCategoryLabel(app.category) }}</div>
        </div>
      </div>
    </div>

    <!-- 所有应用 -->
    <div class="section">
      <div class="section-header">
        <h3 class="section-title">
          <el-icon><Grid /></el-icon>
          {{ selectedCategory ? getCategoryLabel(selectedCategory) : '全部应用' }}
        </h3>
      </div>
      <div v-if="filteredApps.length > 0" class="app-grid">
        <div
          v-for="app in filteredApps"
          :key="app.id"
          class="app-card"
          @click="handleAccessApp(app)"
        >
          <div class="app-icon">
            <img v-if="app.icon" :src="app.icon" :alt="app.name" />
            <el-icon v-else :size="32"><Grid /></el-icon>
          </div>
          <div class="app-info">
            <h4 class="app-name">{{ app.name }}</h4>
            <p class="app-desc">{{ app.description || '暂无描述' }}</p>
          </div>
          <div class="app-actions">
            <el-button
              type="text"
              @click.stop="handleToggleFavorite(app)"
              :class="['favorite-btn', { active: app.isFavorite }]"
            >
              <el-icon>
                <StarFilled v-if="app.isFavorite" />
                <Star v-else />
              </el-icon>
            </el-button>
          </div>
          <div class="category-tag">{{ getCategoryLabel(app.category) }}</div>
        </div>
      </div>
      <el-empty v-else description="暂无应用" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Star, StarFilled, Grid, Search } from '@element-plus/icons-vue'
import { getPortalApps, toggleFavoriteApp, accessApp, type PortalApp } from '@/api/identity'

const apps = ref<PortalApp[]>([])
const searchKeyword = ref('')
const selectedCategory = ref('')
const loading = ref(false)

// 分类映射
const categoryMap: Record<string, string> = {
  cicd: 'CI/CD',
  code: '代码管理',
  monitor: '监控告警',
  registry: '镜像仓库',
  other: '其他'
}

const getCategoryLabel = (category: string) => {
  return categoryMap[category] || category
}

// 收藏的应用
const favoriteApps = computed(() => {
  return apps.value.filter(app => app.isFavorite)
})

// 过滤后的应用
const filteredApps = computed(() => {
  let result = apps.value
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(app =>
      app.name.toLowerCase().includes(keyword) ||
      app.code.toLowerCase().includes(keyword) ||
      (app.description && app.description.toLowerCase().includes(keyword))
    )
  }
  return result
})

// 加载应用列表
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

// 访问应用
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

// 切换收藏
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
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.header-stats {
  display: flex;
  gap: 24px;
}

.stat-item {
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 24px;
  font-weight: 700;
  color: #d4af37;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.filter-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.search-input {
  width: 280px;
}

.category-tabs :deep(.el-radio-button__inner) {
  border-color: #dcdfe6;
}

.category-tabs :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background-color: #000;
  border-color: #000;
  color: #d4af37;
}

.section {
  margin-bottom: 32px;
}

.section-header {
  margin-bottom: 16px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.app-card {
  position: relative;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.app-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 16px rgba(212, 175, 55, 0.15);
  transform: translateY(-2px);
}

.app-card.favorite {
  background: linear-gradient(135deg, #fffef5 0%, #fff 100%);
  border-color: rgba(212, 175, 55, 0.3);
}

.app-icon {
  width: 48px;
  height: 48px;
  min-width: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.app-icon .el-icon {
  color: #d4af37;
}

.app-info {
  flex: 1;
  min-width: 0;
}

.app-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 4px 0;
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
}

.app-actions {
  position: absolute;
  top: 12px;
  right: 12px;
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

.category-tag {
  position: absolute;
  bottom: 12px;
  right: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 10px;
  background: rgba(212, 175, 55, 0.1);
  color: #d4af37;
  border: 1px solid rgba(212, 175, 55, 0.3);
}
</style>
