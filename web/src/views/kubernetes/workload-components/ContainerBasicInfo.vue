<template>
  <el-row :gutter="20">
    <el-col :span="12">
      <el-form label-position="top" size="default">
        <el-form-item label="容器名称">
          <el-input v-model="localContainer.name" placeholder="输入容器名称" @input="update">
            <template #prefix>
              <el-icon><Box /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="容器镜像">
          <el-input v-model="localContainer.image" placeholder="例如: nginx:latest" @input="update">
            <template #prefix>
              <el-icon><Picture /></el-icon>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
    </el-col>
    <el-col :span="12">
      <el-form label-position="top" size="default">
        <el-form-item label="拉取策略">
          <el-select v-model="localContainer.imagePullPolicy" style="width: 100%" placeholder="选择拉取策略" @change="update">
            <el-option label="IfNotPresent - 本地不存在时拉取" value="IfNotPresent" />
            <el-option label="Always - 总是拉取最新" value="Always" />
            <el-option label="Never - 从不拉取" value="Never" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { Box, Picture } from '@element-plus/icons-vue'

interface Container {
  name: string
  image: string
  imagePullPolicy?: string
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive({ ...props.container })

watch(() => props.container, (newVal) => {
  Object.assign(localContainer, newVal)
}, { deep: true })

const update = () => {
  emit('update', { ...localContainer })
}
</script>
