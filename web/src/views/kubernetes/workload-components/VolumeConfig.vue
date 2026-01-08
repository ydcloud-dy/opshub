<template>
  <div class="info-panel volume-panel">
    <div class="panel-header">
      <span class="panel-icon">💾</span>
      <span class="panel-title">数据卷</span>
      <el-button link type="primary" @click="emit('addVolume')" :icon="Plus" size="small">添加</el-button>
    </div>
    <div class="panel-content">
      <div class="volume-list">
        <div v-for="(volume, index) in volumes" :key="'volume-'+index" class="volume-card">
          <div class="volume-card-header">
            <span class="volume-name">{{ volume.name || '未命名' }}</span>
            <el-button link type="danger" @click="emit('removeVolume', index)" :icon="Delete" size="small" />
          </div>
          <div class="volume-card-body">
            <el-form label-width="60px" size="small">
              <el-form-item label="类型">
                <el-select v-model="volume.type" placeholder="选择类型">
                  <el-option label="EmptyDir" value="emptyDir" />
                  <el-option label="HostPath" value="hostPath" />
                  <el-option label="ConfigMap" value="configMap" />
                  <el-option label="Secret" value="secret" />
                  <el-option label="PVC" value="persistentVolumeClaim" />
                </el-select>
              </el-form-item>
              <el-form-item label="名称">
                <el-input v-model="volume.name" placeholder="volume-name" />
              </el-form-item>
              <el-form-item label="路径" v-if="volume.type === 'hostPath'">
                <el-input v-model="volume.path" placeholder="/host/path" />
              </el-form-item>
              <el-form-item label="声明名" v-if="volume.type === 'persistentVolumeClaim'">
                <el-input v-model="volume.claimName" placeholder="pvc-name" />
              </el-form-item>
            </el-form>
          </div>
        </div>
        <div v-if="volumes.length === 0" class="empty-tip">
          <el-empty description="暂无数据卷" :image-size="60" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

interface Volume {
  name: string
  type: string
  path?: string
  claimName?: string
}

const props = defineProps<{
  volumes: Volume[]
}>()

const emit = defineEmits<{
  addVolume: []
  removeVolume: [index: number]
}>()
</script>

<style scoped>
.info-panel {
  background: #fff;
}

.volume-panel {
  background: #fafbfc;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #e9ecef;
  background: #fff;
  position: sticky;
  top: 0;
  z-index: 10;
}

.panel-icon {
  font-size: 18px;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #212529;
  flex: 1;
}

.panel-content {
  padding: 12px 16px;
}

.volume-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.volume-card {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 8px;
  overflow: hidden;
  transition: box-shadow 0.2s;
}

.volume-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.volume-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-bottom: 1px solid #dee2e6;
}

.volume-name {
  font-size: 13px;
  font-weight: 600;
  color: #495057;
}

.volume-card-body {
  padding: 14px;
}

.volume-card-body .el-form-item {
  margin-bottom: 12px;
}

.empty-tip {
  text-align: center;
  padding: 20px;
  color: #adb5bd;
  font-size: 13px;
}
</style>
