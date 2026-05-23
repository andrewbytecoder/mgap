<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  WindowMinimise,
  WindowToggleMaximise,
  WindowIsMaximised,
  WindowUnmaximise,
  WindowSetPosition,
  Quit
} from '../../wailsjs/runtime'
import GoLearnIcon from '@/assets/icons/go-learn.svg'

function goLearn() {
  // TODO: add learn/help action here
}


const isDragging = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const windowStartX = ref(0)
const windowStartY = ref(0)
const isMaximised = ref(false)

async function checkMaximised() {
  isMaximised.value = await WindowIsMaximised()
}

async function onToggleMaximise() {
  WindowToggleMaximise()
  // 等待窗口状态更新后检测
  await new Promise(r => setTimeout(r, 150))
  await checkMaximised()
}

async function onMouseDown(e: MouseEvent) {
  // skip if clicking buttons
  const target = e.target as HTMLElement
  if (target.closest('.titlebar-button')) return

  isDragging.value = true
  dragStartX.value = e.screenX
  dragStartY.value = e.screenY

  // If window is maximised, first restore it before dragging
  const max = await WindowIsMaximised()
  if (max) {
    WindowUnmaximise()
    isMaximised.value = false
    // After unmaximising, the window position changes — use mouse offset to set position
    windowStartX.value = e.screenX - 400 // estimate half width
    windowStartY.value = e.screenY - 20  // near titlebar
    WindowSetPosition(windowStartX.value, windowStartY.value)
    return
  }

  windowStartX.value = window.screenX
  windowStartY.value = window.screenY
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  //  计算出移动的坐标
  const deltaX = e.screenX - dragStartX.value
  const deltaY = e.screenY - dragStartY.value
  WindowSetPosition(windowStartX.value + deltaX, windowStartY.value + deltaY)
}

function onMouseUp() {
  isDragging.value = false
}

onMounted(() => {
  checkMaximised()
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
})

onUnmounted(() => {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="titlebar">
    <div class="titlebar-drag-region" @mousedown="onMouseDown">
      <v-btn
          size="small"
          variant="tonal"
          :prepend-icon="GoLearnIcon"
          @click="goLearn">MGAP</v-btn>
    </div>
    <div class="titlebar-buttons">
      <button class="titlebar-button" @click="WindowMinimise" title="最小化">
        <svg width="12" height="12" viewBox="0 0 12 12">
          <line x1="0" y1="6" x2="12" y2="6" stroke="currentColor" stroke-width="1.5" />
        </svg>
      </button>
      <button class="titlebar-button" @click="onToggleMaximise" :title="isMaximised ? '还原' : '最大化'">
        <!-- 最大化图标 -->
        <svg v-if="!isMaximised" width="12" height="12" viewBox="0 0 12 12">
          <rect x="1.5" y="1.5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1.5" />
        </svg>
        <!-- 还原图标（重叠方块） -->
        <svg v-else width="12" height="12" viewBox="0 0 12 12">
          <rect x="3.5" y="0.5" width="8" height="8" fill="none" stroke="currentColor" stroke-width="1.5" />
          <rect x="0.5" y="3.5" width="8" height="8" fill="currentColor" opacity="0.15" stroke="currentColor" stroke-width="1.5" />
        </svg>
      </button>
      <button class="titlebar-button titlebar-close" @click="Quit" title="关闭">
        <svg width="12" height="12" viewBox="0 0 12 12">
          <line x1="1" y1="1" x2="11" y2="11" stroke="currentColor" stroke-width="1.5" />
          <line x1="11" y1="1" x2="1" y2="11" stroke="currentColor" stroke-width="1.5" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.titlebar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 36px;
  background: linear-gradient(135deg, #1c2b21 0%, #304a3d 100%);
  color: #f7f6f0;
  user-select: none;
  -webkit-user-select: none;
  position: sticky;
  top: 0;
  flex-shrink: 0;
  z-index: 9999;
}

.titlebar-drag-region {
  flex: 1;
  display: flex;
  align-items: center;
  height: 100%;
  padding-left: 12px;
  cursor: grab;
}

.titlebar-drag-region:active {
  cursor: grabbing;
}

.titlebar-title {
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.3px;
  pointer-events: none;
}

.titlebar-buttons {
  display: flex;
  height: 100%;
}

.titlebar-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 100%;
  border: none;
  background: transparent;
  color: #f7f6f0;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.titlebar-button:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.titlebar-button:active {
  background-color: rgba(255, 255, 255, 0.15);
}

.titlebar-close:hover {
  background-color: #e81123;
}

.titlebar-close:active {
  background-color: #c50b1a;
}
</style>