<template>
  <el-tag :type="tagType" size="small" effect="plain">{{ label }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status?: string }>()
const labels: Record<string, string> = {
  healthy: '健康',
  active: '正常',
  warning: '关注',
  unhealthy: '异常',
  error: '异常',
  down: '不可用',
  checking: '检测中',
  unknown: '未知',
  disabled: '已停用',
  planned: '规划中',
  expiring: '即将过期',
  expired: '已过期',
  running: '执行中',
  success: '成功',
  failed: '失败',
  production: '生产',
  staging: '预发布',
  test: '测试',
  development: '开发',
}
const label = computed(() => labels[props.status || 'unknown'] || props.status || '未知')
const tagType = computed(() => {
  if (['healthy', 'active', 'success'].includes(props.status || '')) return 'success'
  if (['warning', 'checking', 'running', 'expiring', 'staging', 'test'].includes(props.status || '')) return 'warning'
  if (['unhealthy', 'error', 'down', 'expired', 'failed'].includes(props.status || '')) return 'danger'
  if (['disabled', 'planned'].includes(props.status || '')) return 'info'
  return 'info'
})
</script>
