<template>
  <BaseEdge :id="id" :path="edgePath" :marker-end="markerEnd" :style="edgeStyle" />
  <path :d="edgePath" class="topology-flow-edge__stream" :style="streamStyle" />
  <circle class="topology-flow-edge__particle" :fill="edgeColor" :r="data?.kind === 'dependency' ? 3.8 : 2.8">
    <animateMotion :dur="data?.kind === 'dependency' ? '2.2s' : '3.2s'" repeatCount="indefinite" :path="edgePath" />
  </circle>
  <EdgeLabelRenderer v-if="data?.label">
    <div class="topology-flow-edge__label" :class="{ 'is-dependency': data?.kind === 'dependency' }" :style="labelStyle">
      <span>{{ data.label }}</span><b v-if="data.protocol">{{ data.protocol }}</b>
    </div>
  </EdgeLabelRenderer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getBezierPath, getSmoothStepPath, type EdgeProps } from '@vue-flow/core'

const props = defineProps<EdgeProps>()

const pathResult = computed(() => props.data?.layout === 'panorama'
  ? getBezierPath({ sourceX: props.sourceX, sourceY: props.sourceY, sourcePosition: props.sourcePosition, targetX: props.targetX, targetY: props.targetY, targetPosition: props.targetPosition, curvature: 0.28 })
  : getSmoothStepPath({ sourceX: props.sourceX, sourceY: props.sourceY, sourcePosition: props.sourcePosition, targetX: props.targetX, targetY: props.targetY, targetPosition: props.targetPosition, borderRadius: 14, offset: 24 }))
const edgePath = computed(() => pathResult.value[0])
const labelX = computed(() => pathResult.value[1])
const labelY = computed(() => pathResult.value[2])
const edgeColor = computed(() => props.data?.color || '#52d7c8')
const edgeStyle = computed(() => ({ stroke: props.data?.baseColor || edgeColor.value, strokeWidth: props.data?.kind === 'dependency' ? 2.2 : 1.5, opacity: props.data?.kind === 'dependency' ? 0.82 : 0.5 }))
const streamStyle = computed(() => ({ stroke: edgeColor.value, strokeWidth: props.data?.kind === 'dependency' ? 2.4 : 1.6 }))
const labelStyle = computed(() => ({ transform: `translate(-50%, -50%) translate(${labelX.value}px, ${labelY.value}px)`, '--edge-accent': edgeColor.value }))
</script>
