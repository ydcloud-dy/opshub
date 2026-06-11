<template>
  <div class="plugin-list-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Grid /></el-icon>
        </div>
        <div>
          <h2 class="page-title">插件列表</h2>
          <p class="page-subtitle">查看和管理系统中已安装的所有插件</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="loadPlugins" :icon="Refresh">刷新</el-button>
        <el-button type="primary" @click="handleGoToInstall" class="black-button">
          <el-icon style="margin-right: 6px;"><Upload /></el-icon>
          安装插件
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon total">
          <el-icon><Grid /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ plugins.length }}</div>
          <div class="stat-label">总插件数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon enabled">
          <el-icon><Check /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ enabledCount }}</div>
          <div class="stat-label">已启用</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon disabled">
          <el-icon><Close /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ disabledCount }}</div>
          <div class="stat-label">已禁用</div>
        </div>
      </div>
    </div>

    <!-- 插件表格 -->
    <div class="table-container">
      <el-table :data="plugins" v-loading="loading" style="width: 100%" :header-cell-style="{background: '#fafafa'}">
        <el-table-column type="index" label="序号" width="80" align="center" />

        <el-table-column label="插件信息" min-width="300">
          <template #default="{ row }">
            <div class="plugin-info">
              <div class="plugin-icon">
                <el-icon><Grid /></el-icon>
              </div>
              <div class="plugin-details">
                <div class="plugin-name">{{ row.name }}</div>
                <div class="plugin-desc">{{ row.description }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="版本" prop="version" width="120" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.version }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="作者" prop="author" width="150" align="center" />

        <el-table-column label="状态" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="!row.enabled"
              type="primary"
              size="small"
              @click="handleEnable(row)"
              :loading="row.loading"
              link
            >
              启用
            </el-button>
            <el-button
              v-else
              type="danger"
              size="small"
              @click="handleDisable(row)"
              :loading="row.loading"
              link
            >
              禁用
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="handleUninstall(row)"
              :loading="row.loading"
              link
            >
              卸载
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <el-empty v-if="!loading && plugins.length === 0" description="暂无插件" :image-size="80">
        <el-button type="primary" @click="handleGoToInstall" class="black-button">立即安装</el-button>
      </el-empty>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Grid, Upload, Refresh, Check, Close } from '@element-plus/icons-vue'
import { pluginManager } from '@/plugins/manager'
import { enablePlugin, disablePlugin, uninstallPlugin } from '@/api/plugin'
import type { Plugin } from '@/plugins/types'

const router = useRouter()
const loading = ref(false)
const plugins = ref<(Plugin & { loading?: boolean; enabled?: boolean })[]>([])

// 计算启用和禁用的插件数量
const enabledCount = computed(() => plugins.value.filter(p => p.enabled).length)
const disabledCount = computed(() => plugins.value.filter(p => !p.enabled).length)

// 加载插件列表
const loadPlugins = async () => {
  loading.value = true
  try {
    // 从前端插件管理器获取所有已注册的插件
    const allPlugins = pluginManager.getAll()

    // 同时从后端API获取插件启用状态
    try {
      const { listPlugins } = await import('@/api/plugin')
      const backendPlugins = await listPlugins()

      // 合并前端插件和后端状态
      plugins.value = allPlugins.map(plugin => {
        const backendPlugin = backendPlugins.find((p: any) => p.name === plugin.name)
        return {
          ...plugin,
          enabled: backendPlugin?.enabled ?? false,
          loading: false
        }
      })
    } catch (error) {
      // 如果后端API失败，仍然显示前端插件，默认状态为未启用
      plugins.value = allPlugins.map(plugin => ({
        ...plugin,
        enabled: false,
        loading: false
      }))
    }
  } catch (error) {
    ElMessage.error('加载插件列表失败')
  } finally {
    loading.value = false
  }
}

// 启用插件
const handleEnable = async (plugin: Plugin & { loading?: boolean; enabled?: boolean }) => {
  try {
    await ElMessageBox.confirm(
      `确定要启用插件 "${plugin.name}" 吗？启用后页面将自动刷新以加载插件功能。`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    plugin.loading = true
    await enablePlugin(plugin.name)
    ElMessage.success('插件启用成功，页面即将刷新...')

    // 等待1秒后刷新页面，让用户看到成功提示
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '启用插件失败')
    }
    plugin.loading = false
  }
}

// 禁用插件
const handleDisable = async (plugin: Plugin & { loading?: boolean; enabled?: boolean }) => {
  try {
    await ElMessageBox.confirm(
      `确定要禁用插件 "${plugin.name}" 吗？禁用后该插件的菜单和功能将不可用，页面将自动刷新。`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    plugin.loading = true
    await disablePlugin(plugin.name)
    ElMessage.success('插件禁用成功，页面即将刷新...')

    // 等待1秒后刷新页面
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '禁用插件失败')
    }
    plugin.loading = false
  }
}

// 卸载插件
const handleUninstall = async (plugin: Plugin & { loading?: boolean; enabled?: boolean }) => {
  try {
    await ElMessageBox.confirm(
      `确定要卸载插件 "${plugin.name}" 吗？这将删除插件的所有文件，此操作不可恢复！卸载后需要重启服务才能完全生效。`,
      '警告',
      {
        confirmButtonText: '确定卸载',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    plugin.loading = true
    await uninstallPlugin(plugin.name)
    ElMessage.success('插件卸载成功，请重启服务以完全生效')

    // 从列表中移除
    const index = plugins.value.findIndex(p => p.name === plugin.name)
    if (index !== -1) {
      plugins.value.splice(index, 1)
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '卸载插件失败')
    }
    plugin.loading = false
  }
}

// 跳转到安装页面
const handleGoToInstall = () => {
  router.push('/plugin/install')
}

onMounted(async () => {
  // 等待一小段时间确保插件完全加载
  await new Promise(resolve => setTimeout(resolve, 100))
  loadPlugins()
})
</script>

<style scoped lang="scss">
.plugin-list-container {
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  background-color: transparent;
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
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  background: #f5f3ff;
  border: 1px solid #ddd6fe;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #7c3aed;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 28px;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: #909399;
  line-height: 20px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;

  &:hover {
    background-color: #1a1a1a !important;
  }
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  align-items: center;
  gap: 16px;

  .stat-icon {
    width: 56px;
    height: 56px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 28px;

    &.total {
      background: #f5f3ff;
      border: 1px solid #ddd6fe;
      color: #7c3aed;
    }

    &.enabled {
      background: #ecfdf3;
      border: 1px solid #bbf7d0;
      color: #16a34a;
    }

    &.disabled {
      background: #f8fafc;
      border: 1px solid #e2e8f0;
      color: #64748b;
    }
  }

  .stat-content {
    flex: 1;

    .stat-value {
      font-size: 28px;
      font-weight: 700;
      color: #303133;
      line-height: 1;
      margin-bottom: 8px;
    }

    .stat-label {
      font-size: 14px;
      color: #909399;
    }
  }
}

.table-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;

  .plugin-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .plugin-icon {
      width: 40px;
      height: 40px;
      border-radius: 8px;
      background: #f5f3ff;
      border: 1px solid #ddd6fe;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #7c3aed;
      font-size: 20px;
      flex-shrink: 0;
    }

    .plugin-details {
      flex: 1;
      min-width: 0;

      .plugin-name {
        font-size: 14px;
        font-weight: 600;
        color: #303133;
        margin-bottom: 4px;
      }

      .plugin-desc {
        font-size: 12px;
        color: #909399;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
  }
}
</style>
