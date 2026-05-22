<script setup lang="ts">
import { computed, ref } from 'vue'
import type { FlamegraphNode } from '../types'

const props = defineProps<{
  root: FlamegraphNode | null
}>()

type Rect = {
  id: string
  name: string
  fullName: string
  fileName: string
  value: number
  x: number
  y: number
  width: number
  depth: number
}

const hovered = ref<Rect | null>(null)
const rowHeight = 26
const totalWidth = 1200

const rects = computed<Rect[]>(() => {
  if (!props.root) return []
  const out: Rect[] = []
  const total = Math.max(props.root.value, 1)

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
        x: cursor,
        y: depth * rowHeight,
        width,
        depth
      })
      walk(child, cursor, depth + 1, child.value)
      cursor += width
    }
  }

  walk(props.root, 0, 0, props.root.value)
  return out
})

const maxDepth = computed(() => rects.value.reduce((max, rect) => Math.max(max, rect.depth), 0))
const svgHeight = computed(() => (maxDepth.value + 1) * rowHeight + 10)

function colorFor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i += 1) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  const hue = hash % 360
  return `hsl(${hue} 62% 62%)`
}

function shortLabel(name: string, width: number): string {
  const maxChars = Math.max(Math.floor(width / 7), 3)
  if (name.length <= maxChars) return name
  return `${name.slice(0, Math.max(maxChars - 1, 1))}…`
}
</script>

<template>
  <div v-if="root" class="flamegraph-shell">
    <div class="flamegraph-toolbar">
      <span class="mono">total: {{ root.value }}</span>
      <span v-if="hovered" class="mono">{{ hovered.fullName }} · {{ hovered.value }}</span>
    </div>
    <div class="flamegraph-scroll">
      <svg :viewBox="`0 0 ${totalWidth} ${svgHeight}`" class="flamegraph-svg">
        <g v-for="rect in rects" :key="rect.id" @mouseenter="hovered = rect" @mouseleave="hovered = null">
          <rect
            :x="rect.x"
            :y="rect.y"
            :width="rect.width"
            :height="rowHeight - 4"
            :fill="colorFor(rect.fullName)"
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
