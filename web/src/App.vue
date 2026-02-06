<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body,
#app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
    sans-serif;
}

/* 全局字体大小增加 */
html {
  font-size: 20px; /* 增加到20px */
  overflow-y: scroll; /* 始终显示滚动条，防止弹窗打开时页面抖动 */
}

body {
  font-size: 1rem; /* 20px */
}

/* 修复 Element Plus Dialog 打开时页面抖动 */
body.el-popup-parent--hidden {
  padding-right: 0 !important;
  overflow: hidden !important;
  width: 100% !important;
}

/* 隐藏弹窗遮罩层的滚动条 */
.el-overlay {
  overflow: hidden !important;
  scrollbar-width: none !important; /* Firefox */
  -ms-overflow-style: none !important; /* IE/Edge */
}

.el-overlay::-webkit-scrollbar {
  display: none !important; /* Chrome/Safari/Opera */
  width: 0 !important;
  height: 0 !important;
}

/* 弹窗wrapper允许滚动，但隐藏滚动条 */
.el-overlay-dialog {
  overflow: auto !important;
  scrollbar-width: none !important;
  -ms-overflow-style: none !important;
}

.el-overlay-dialog::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
  height: 0 !important;
}

/* 调整Element Plus组件字体大小 */
.el-button {
  font-size: 17px !important;
}

.el-input__inner,
.el-textarea__inner {
  font-size: 17px !important;
}

.el-form-item__label {
  font-size: 17px !important;
}

.el-table {
  font-size: 17px !important;
}

.el-table th,
.el-table td {
  font-size: 17px !important;
}

.el-menu-item {
  font-size: 17px !important;
}

.el-dropdown-menu__item {
  font-size: 17px !important;
}

.el-dialog__title {
  font-size: 20px !important;
}

.el-breadcrumb__inner {
  font-size: 17px !important;
}

.el-tag {
  font-size: 16px !important;
}

.el-pagination {
  font-size: 17px !important;
}

.el-card__header {
  font-size: 18px !important;
}

/* 侧边栏特殊字体 */
.el-aside .user-name {
  font-size: 16px !important;
}

.el-aside .user-role {
  font-size: 14px !important;
}

.logo h3 {
  font-size: 22px !important;
}
</style>
