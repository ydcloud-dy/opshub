<template>
  <div class="topology-node-card" :class="[`topology-node-card--${data.type}`, { 'is-selected': selected }]" :style="nodeStyle">
    <Handle type="target" :position="Position.Left" class="topology-node-card__handle" />
    <div class="topology-node-card__scan" />
    <div class="topology-node-card__header">
      <div class="topology-node-card__icon"><el-icon><component :is="nodeIcon" /></el-icon></div>
      <div class="topology-node-card__identity">
        <span>{{ data.typeLabel }}</span>
        <strong>{{ data.label }}</strong>
      </div>
      <i class="topology-node-card__status" :class="`is-${statusTone}`" />
    </div>
    <div class="topology-node-card__meta">
      <span>{{ data.detail || '暂无补充信息' }}</span>
      <b>{{ statusLabel }}</b>
    </div>
    <Handle type="source" :position="Position.Right" class="topology-node-card__handle" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { Coin, Connection, Link, Monitor, Platform, SetUp } from '@element-plus/icons-vue'

const props = defineProps<{ data: Record<string, any>; selected?: boolean }>()

const icons: Record<string, any> = {
  application: Platform,
  environment: SetUp,
  domain: Link,
  resource: Monitor,
  component: Coin,
  external: Connection,
}

const nodeIcon = computed(() => icons[props.data.type] || Connection)
const statusTone = computed(() => {
  const status = String(props.data.status || '').toLowerCase()
  if (['healthy', 'active', 'online', 'running', 'success'].includes(status)) return 'healthy'
  if (['warning', 'degraded', 'expiring'].includes(status)) return 'warning'
  if (['unhealthy', 'error', 'down', 'failed', 'disabled'].includes(status)) return 'danger'
  return 'unknown'
})
const statusLabel = computed(() => ({ healthy: '运行正常', warning: '需要关注', danger: '存在异常', unknown: '状态未知' }[statusTone.value]))
const nodeStyle = computed(() => ({ '--node-accent': props.data.color, '--node-glow': props.data.glow }))
</script>
