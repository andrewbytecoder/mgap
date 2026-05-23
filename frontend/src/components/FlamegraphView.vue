<script setup lang="ts">
import { computed, ref } from 'vue'
import type { FlamegraphNode } from '@/types'
import { formatBytes, formatNumber, formatPercent } from '@/utils/format'

const props = defineProps<{
  root: FlamegraphNode | null
  metric?: string
}>()

type Rect = {
  id: string
  name: string
  fullName: string
  fileName: string
  value: number
  selfValue: number
  x: number
  y: number
  width: number
  depth: number
  kind: 'runtime' | 'stdlib' | 'user' | 'unknown'
}

const hovered = ref<Rect | null>(null)
const search = ref('')
const valueMode = ref<'total' | 'self' | 'percent'>('total')
const focusPath = ref<string[]>([])
const rowHeight = 26
const totalWidth = 1200

const focusedRoot = computed(() => {
  if (!props.root) return null
  let current = props.root
  for (const segment of focusPath.value) {
    const next = current.children.find(child => child.fullName === segment)
    if (!next) break
    current = next
  }
  return current
})

const rects = computed<Rect[]>(() => {
  if (!focusedRoot.value) return []
  const out: Rect[] = []
  const total = Math.max(focusedRoot.value.value, 1)

  const walk = (node: FlamegraphNode, x: number, depth: number, parentValue: number) => {
    if (!node.children?.length) return

    let cursor = x
    for (const child of node.children) {
      const width = Math.max((child.value / total) * totalWidth, 1)
      out.push({
        id: `${depth}-${cursor}-${child.fullName}`,
        name: child.name,
        fullName: child.fullName,
        fileName: child.fileName,
        value: child.value,
        selfValue: child.selfValue ?? 0,
        x: cursor,
        y: depth * rowHeight,
        width,
        depth
        ,
        kind: classifyNode(child)
      })
      walk(child, cursor, depth + 1, child.value)
      cursor += width
    }
  }

  walk(focusedRoot.value, 0, 0, focusedRoot.value.value)
  return out.filter(rect => {
    if (!search.value.trim()) return true
    const needle = search.value.trim().toLowerCase()
    return rect.fullName.toLowerCase().includes(needle) || rect.fileName.toLowerCase().includes(needle)
  })
})

const maxDepth = computed(() => rects.value.reduce((max, rect) => Math.max(max, rect.depth), 0))
const svgHeight = computed(() => (maxDepth.value + 1) * rowHeight + 10)

function colorFor(name: string): string {
  switch (name as any) {
    case 'runtime':
      return '#d86e3f'
    case 'stdlib':
      return '#3f6a8f'
    case 'user':
      return '#4d7c53'
    default:
      return '#8d5d8b'
  }
}

function classifyNode(node: FlamegraphNode): Rect['kind'] {
  if (node.fullName.startsWith('runtime.') || node.fullName.startsWith('runtime/')) return 'runtime'
  if (node.fullName.startsWith('net/') || node.fullName.startsWith('internal/') || node.fullName.startsWith('sync.') || node.fullName.startsWith('syscall.')) {
    return 'stdlib'
  }
  if (node.fullName === 'unknown' || node.fullName.startsWith('0x')) return 'unknown'
  return 'user'
}

function shortLabel(name: string, width: number): string {
  const maxChars = Math.max(Math.floor(width / 7), 3)
  if (name.length <= maxChars) return name
  return `${name.slice(0, Math.max(maxChars - 1, 1))}…`
}

function displayValue(rect: Rect): string {
  if (valueMode.value === 'self') {
    return formatMetricValue(rect.selfValue)
  }
  if (valueMode.value === 'percent') {
    return `${((rect.value / Math.max(focusedRoot.value?.value ?? 1, 1)) * 100).toFixed(2)}%`
  }
  return formatMetricValue(rect.value)
}

function formatMetricValue(value: number): string {
  if (props.metric === 'cpu') return formatPercent(value)
  if (props.metric === 'goroutine' || props.metric === 'threadcreate') return formatNumber(value)
  if (props.metric === 'heap' || props.metric === 'allocs') return formatBytes(value)
  return formatNumber(value)
}

function drill(rect: Rect) {
  focusPath.value = [...focusPath.value, rect.fullName]
}

function resetDrill() {
  focusPath.value = []
}

function stepBack() {
  focusPath.value = focusPath.value.slice(0, -1)
}
</script>

<template>
  <div v-if="root" class="flamegraph-shell">
    <div class="flamegraph-toolbar">
      <div class="toolbar-left">
        <v-btn size="small" variant="tonal" prepend-icon="mdi-home" @click="resetDrill">Root</v-btn>
        <v-btn size="small" variant="tonal" prepend-icon="mdi-arrow-left" :disabled="focusPath.length === 0" @click="stepBack">Back</v-btn>
        <span class="mono">total: {{ focusedRoot?.value ?? 0 }}</span>
      </div>
      <div class="toolbar-right">
        <v-select
          v-model="valueMode"
          :items="[
            { title: 'Total', value: 'total' },
            { title: 'Self', value: 'self' },
            { title: 'Percent', value: 'percent' }
          ]"
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          style="width: 120px"
        />
        <v-text-field v-model="search" density="compact" hide-details placeholder="Search function" prepend-inner-icon="mdi-magnify" style="width: 220px" />
      </div>
    </div>
    <div class="legend">
      <span class="legend-item"><span class="legend-dot runtime"></span>runtime</span>
      <span class="legend-item"><span class="legend-dot stdlib"></span>stdlib</span>
      <span class="legend-item"><span class="legend-dot user"></span>user code</span>
      <span class="legend-item"><span class="legend-dot unknown"></span>unknown/raw</span>
      <span v-if="hovered" class="mono legend-detail">{{ hovered.fullName }} · {{ displayValue(hovered) }}</span>
    </div>
    <div class="flamegraph-scroll">
      <svg :viewBox="`0 0 ${totalWidth} ${svgHeight}`" class="flamegraph-svg">
        <g v-for="rect in rects" :key="rect.id" @mouseenter="hovered = rect" @mouseleave="hovered = null" @click="drill(rect)">
          <rect
            :x="rect.x"
            :y="rect.y"
            :width="rect.width"
            :height="rowHeight - 4"
            :fill="colorFor(rect.kind)"
            rx="4"
            ry="4"
          />
          <text
            v-if="rect.width > 28"
            :x="rect.x + 6"
            :y="rect.y + 16"
            class="flamegraph-label"
          >
            {{ shortLabel(rect.name, rect.width) }}
          </text>
        </g>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.flamegraph-shell {
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.72);
}

.flamegraph-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  background: #eef0e7;
  font-size: 12px;
  flex-wrap: wrap;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.legend {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  align-items: center;
  padding: 8px 14px;
  border-top: 1px solid rgba(48, 74, 61, 0.08);
  border-bottom: 1px solid rgba(48, 74, 61, 0.08);
  background: rgba(48, 74, 61, 0.03);
  font-size: 12px;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}

.legend-dot.runtime { background: #d86e3f; }
.legend-dot.stdlib { background: #3f6a8f; }
.legend-dot.user { background: #4d7c53; }
.legend-dot.unknown { background: #8d5d8b; }

.legend-detail {
  margin-left: auto;
}

.flamegraph-scroll {
  overflow: auto;
}

.flamegraph-svg {
  display: block;
  width: 100%;
  min-width: 1200px;
  height: auto;
  background: rgba(48, 74, 61, 0.03);
}

.flamegraph-label {
  font-size: 11px;
  fill: #142018;
  pointer-events: none;
}
</style>
