<template>
  <div class="inventory-page__header">
    <div class="inventory-page__title">
      <div class="inventory-page__title-icon">
        <el-icon><component :is="icon" /></el-icon>
      </div>
      <div class="inventory-page__title-copy">
        <h2>{{ title }}</h2>
        <p>{{ description }}</p>
      </div>
    </div>

    <div v-if="$slots.metrics" class="inventory-page__metrics">
      <slot name="metrics" />
    </div>

    <div class="inventory-page__actions">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon: Component
  title: string
  description: string
}>()
</script>
