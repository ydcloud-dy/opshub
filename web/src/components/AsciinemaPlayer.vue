<template>
  <div class="asciinema-player-container">
    <div class="asciinema-player-wrapper" ref="playerRef"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onBeforeUnmount } from 'vue'

interface Props {
  src: string
  cols?: number
  rows?: number
  autoplay?: boolean
  preload?: boolean
  startTime?: number
  speed?: number
  loop?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  cols: 80,
  rows: 24,
  autoplay: false,
  preload: true,
  startTime: 0,
  speed: 1,
  loop: false
})

const emit = defineEmits(['ready', 'play', 'pause', 'finish', 'progress'])

const playerRef = ref<HTMLDivElement>()
let player: any = null

// 动态加载 AsciinemaPlayer
const loadAsciinemaPlayer = async () => {
  return new Promise<void>((resolve, reject) => {
    if ((window as any).AsciinemaPlayer) {
      resolve()
      return
    }

    // 加载 CSS
    const css = document.createElement('link')
    css.rel = 'stylesheet'
    css.href = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.6.3/dist/bundle/asciinema-player.css'
    document.head.appendChild(css)

    // 加载 JS
    const script = document.createElement('script')
    script.src = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.6.3/dist/bundle/asciinema-player.min.js'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('Failed to load AsciinemaPlayer'))
    document.head.appendChild(script)
  })
}

// 创建播放器
const createPlayer = async () => {
  if (!playerRef.value || !props.src) return

  try {
    await loadAsciinemaPlayer()

    // 清除旧播放器
    if (player) {
      playerRef.value.innerHTML = ''
    }

    // 创建新播放器
    const AsciinemaPlayer = (window as any).AsciinemaPlayer
    player = new AsciinemaPlayer(props.src, playerRef.value, {
      cols: props.cols,
      rows: props.rows,
      autoplay: props.autoplay,
      preload: props.preload ? 'auto' : 'none',
      startTime: props.startTime,
      speed: props.speed,
      loop: props.loop,
      theme: 'tango',
      poster: 'npt:0:01',
    })

    // 监听事件
    if (player.addEventListener) {
      player.addEventListener('ready', () => emit('ready'))
      player.addEventListener('play', () => emit('play'))
      player.addEventListener('pause', () => emit('pause'))
      player.addEventListener('ended', () => emit('finish'))
      player.addEventListener('progress', (e: any) => emit('progress', e))
    }

    emit('ready')
  } catch (error) {
    console.error('Failed to create AsciinemaPlayer:', error)
  }
}

// 监听 src 变化
watch(() => props.src, () => {
  createPlayer()
})

onMounted(() => {
  createPlayer()
})

onBeforeUnmount(() => {
  if (player && playerRef.value) {
    playerRef.value.innerHTML = ''
    player = null
  }
})

// 暴露方法
defineExpose({
  play: () => player?.play(),
  pause: () => player?.pause(),
  seek: (time: number) => player?.seek(time),
  getDuration: () => player?.duration,
  getCurrentTime: () => player?.currentTime,
})
</script>

<style scoped>
.asciinema-player-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
}

.asciinema-player-wrapper {
  width: 100%;
  height: 100%;
}

/* 深度样式覆盖 - 修改 AsciinemaPlayer 的颜色 */
.asciinema-player-wrapper :deep(.asciinema-player) {
  background-color: #000 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-terminal) {
  background-color: #000 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-control-bar) {
  background: linear-gradient(to top, rgba(0, 0, 0, 0.8), transparent) !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-progress-container) {
  background-color: rgba(212, 175, 55, 0.2) !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-progress-bar) {
  background-color: #d4af37 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-controls) {
  color: #d4af37 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-icon-button) {
  color: #d4af37 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-icon-button:hover) {
  color: #bfa13f !important;
}
</style>
