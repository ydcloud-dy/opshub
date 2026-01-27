<template>
  <div class="test-home-container">
    <el-card class="welcome-card">
      <template #header>
        <div class="card-header">
          <el-icon class="header-icon" color="#409eff"><Grape /></el-icon>
          <span class="header-title">测试插件</span>
        </div>
      </template>

      <div class="content">
        <h1>🎉 测试插件安装成功！</h1>
        <p class="subtitle">恭喜你，插件系统运行正常</p>

        <el-divider />

        <div class="info-section">
          <h3>插件信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="插件名称">测试插件</el-descriptions-item>
            <el-descriptions-item label="插件版本">1.0.0</el-descriptions-item>
            <el-descriptions-item label="插件作者">Test Team</el-descriptions-item>
            <el-descriptions-item label="安装时间">{{ currentTime }}</el-descriptions-item>
          </el-descriptions>
        </div>

        <el-divider />

        <div class="action-section">
          <h3>测试功能</h3>
          <el-space wrap>
            <el-button type="primary" @click="showMessage">显示消息</el-button>
            <el-button type="success" @click="counter++">计数器: {{ counter }}</el-button>
            <el-button type="warning" @click="toggleColor">切换颜色</el-button>
          </el-space>

          <div v-if="showColorBlock" class="color-block" :style="{ background: currentColor }">
            当前颜色: {{ currentColor }}
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Grape } from '@element-plus/icons-vue'

const currentTime = ref(new Date().toLocaleString('zh-CN'))
const counter = ref(0)
const showColorBlock = ref(false)
const currentColor = ref('#409eff')

const colors = ['#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#909399']
let colorIndex = 0

const showMessage = () => {
  ElMessage.success('测试插件功能正常！')
}

const toggleColor = () => {
  showColorBlock.value = true
  colorIndex = (colorIndex + 1) % colors.length
  currentColor.value = colors[colorIndex]
}
</script>

<style scoped lang="scss">
.test-home-container {
  padding: 24px;

  .welcome-card {
    max-width: 800px;
    margin: 0 auto;

    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      .header-icon {
        font-size: 28px;
      }

      .header-title {
        font-size: 20px;
        font-weight: 600;
      }
    }
  }

  .content {
    text-align: center;

    h1 {
      color: #303133;
      margin-bottom: 12px;
    }

    .subtitle {
      color: #606266;
      font-size: 16px;
      margin-bottom: 24px;
    }

    .info-section,
    .action-section {
      margin: 24px 0;

      h3 {
        margin-bottom: 16px;
        color: #303133;
      }
    }

    .color-block {
      margin-top: 20px;
      padding: 40px;
      border-radius: 8px;
      color: white;
      font-size: 18px;
      font-weight: 600;
      text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
      animation: fadeIn 0.3s;
    }

    @keyframes fadeIn {
      from {
        opacity: 0;
        transform: scale(0.9);
      }
      to {
        opacity: 1;
        transform: scale(1);
      }
    }
  }
}
</style>
