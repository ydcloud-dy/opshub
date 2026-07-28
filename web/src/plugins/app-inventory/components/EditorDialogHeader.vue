<template>
  <div class="inventory-dialog-heading">
    <div class="inventory-dialog-heading__icon">
      <el-icon><component :is="icon" /></el-icon>
    </div>
    <div class="inventory-dialog-heading__copy">
      <span v-if="eyebrow" class="inventory-dialog-heading__eyebrow">{{ eyebrow }}</span>
      <h3>{{ title }}</h3>
      <p>{{ description }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon: Component
  title: string
  description: string
  eyebrow?: string
}>()
</script>
