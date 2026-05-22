<script setup lang="ts">
import { computed } from 'vue'
import type { EChartsOption } from 'echarts'
import VChart from 'vue-echarts'
import type { GraphData, MetricKey } from '../types'
import { formatBytes, formatNumber, formatPercent } from '../utils/format'

const props = defineProps<{
  data: GraphData
  metric: MetricKey
  mode: 'flat' | 'cum'
  topN: number
}>()

type Row = {
  name: string
  flat: number
  cum: number
}

const rows = computed<Row[]>(() =>
  Object.values(props.data.lineTable)
    .map(line => {
      const point = line.points[0]
      return {
        name: line.name || 'unknown',
        flat: point?.flat ?? 0,
        cum: point?.cum ?? 0
      }
    })
    .filter(row => row.name !== 'total')
    .sort((left, right) => right[props.mode] - left[props.mode])
)

const topRows = computed(() => rows.value.slice(0, props.topN))

const option = computed<EChartsOption>(() => {
  const labels = [...topRows.value].reverse().map(item => shorten(item.name))
  const values = [...topRows.value].reverse().map(item => item[props.mode])

  return {
    animationDuration: 150,
    color: ['#d86e3f'],
    grid: {
      left: 220,
      right: 28,
      top: 20,
      bottom: 28,
      containLabel: false
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      backgroundColor: '#132019',
      borderColor: '#132019',
      textStyle: {
        color: '#f8f7f2'
      },
      formatter: (params: any) => {
        const item = Array.isArray(params) ? params[0] : params
        if (!item) return ''
        const row = [...topRows.value].reverse()[item.dataIndex]
        return `
          <div style="min-width:280px">
            <div style="font-weight:700;margin-bottom:8px">${row.name}</div>
            <div style="display:flex;justify-content:space-between;gap:16px"><span>Flat</span><strong>${formatMetricValue(props.metric, row.flat)}</strong></div>
            <div style="display:flex;justify-content:space-between;gap:16px"><span>Cum</span><strong>${formatMetricValue(props.metric, row.cum)}</strong></div>
          </div>
        `
      }
    },
    xAxis: {
      type: 'value',
      axisLabel: {
        color: '#526158',
        formatter: (value: number) => formatMetricValue(props.metric, value)
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(48, 74, 61, 0.12)'
        }
      }
    },
    yAxis: {
      type: 'category',
      data: labels,
      axisLabel: {
        color: '#526158',
        width: 190,
        overflow: 'truncate'
      }
    },
    series: [
      {
        type: 'bar',
        data: values,
        barMaxWidth: 24,
        itemStyle: {
          borderRadius: [0, 6, 6, 0]
        }
      }
    ]
  }
})

function formatMetricValue(metric: MetricKey, value: number): string {
  if (metric === 'cpu') return formatPercent(value)
  if (metric === 'goroutine' || metric === 'threadcreate') return formatNumber(value)
  if (metric === 'block' || metric === 'mutex') return formatNumber(value)
  return formatBytes(value)
}

function shorten(value: string): string {
  return value.length > 44 ? `${value.slice(0, 41)}...` : value
}
</script>

<template>
  <div class="imported-layout">
    <VChart class="imported-chart" :option="option" autoresize />
    <div class="imported-table">
      <div class="table-header mono">
        <span>Function</span>
        <span>Flat</span>
        <span>Cum</span>
      </div>
      <div v-for="row in topRows" :key="row.name" class="table-row">
        <span class="function-name mono" :title="row.name">{{ row.name }}</span>
        <span class="mono">{{ formatMetricValue(metric, row.flat) }}</span>
        <span class="mono">{{ formatMetricValue(metric, row.cum) }}</span>
      </div>
    </div>
  </div>

  <!-- Goroutine Stacks -->
  <div v-if="metric === 'goroutine'" class="stacks-section">
    <div class="stacks-header">
      <span>Goroutine Stacks</span>
      <span class="stacks-count">{{ props.data.stacks?.length || 0 }} unique stack(s)</span>
    </div>

    <div v-if="props.data.stacks?.length" class="stacks-list">
      <div v-for="(stack, idx) in props.data.stacks" :key="idx" class="stack-card">
        <div class="stack-count">{{ stack.count }} goroutine(s)</div>
        <div class="stack-frames">
          <div v-for="(frame, fidx) in stack.frames" :key="fidx" class="stack-frame">
            <span class="frame-func mono">{{ frame.func }}</span>
            <span v-if="frame.file" class="frame-location mono">{{ frame.file }}:{{ frame.line }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Raw Text (browser-like view) -->
    <div v-if="props.data.rawText" class="raw-text-section">
      <div class="raw-text-header">Raw Profile Text (browser view)</div>
      <pre class="raw-text-content">{{ props.data.rawText }}</pre>
    </div>
  </div>
</template>

<style scoped>
.imported-layout {
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  gap: 16px;
  height: 100%;
}

.imported-chart {
  display: block;
  min-height: 360px;
  width: 100%;
}

.imported-table {
  overflow: auto;
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.55);
}

.table-header,
.table-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 88px 88px;
  gap: 12px;
  align-items: center;
  padding: 10px 14px;
}

.table-header {
  position: sticky;
  top: 0;
  background: #eef0e7;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.table-row {
  border-top: 1px solid rgba(48, 74, 61, 0.06);
  font-size: 13px;
}

.function-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stacks-section {
  margin-top: 16px;
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.55);
  overflow: hidden;
}

.stacks-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #eef0e7;
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #304a3d;
}

.stacks-count {
  font-size: 12px;
  font-weight: 400;
  color: #56665d;
  text-transform: none;
  letter-spacing: normal;
}

.stacks-list {
  max-height: 400px;
  overflow-y: auto;
  padding: 12px 16px;
}

.stack-card {
  margin-bottom: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(48, 74, 61, 0.04);
  border: 1px solid rgba(48, 74, 61, 0.06);
}

.stack-card:last-child {
  margin-bottom: 0;
}

.stack-count {
  font-size: 13px;
  font-weight: 600;
  color: #d86e3f;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid rgba(48, 74, 61, 0.08);
}

.stack-frames {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stack-frame {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  line-height: 1.5;
  padding-left: 12px;
  position: relative;
}

.stack-frame::before {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(48, 74, 61, 0.25);
}

.stack-frame:not(:last-child)::after {
  content: '';
  position: absolute;
  left: 2.5px;
  top: 14px;
  width: 1px;
  height: calc(100% + 2px);
  background: rgba(48, 74, 61, 0.12);
}

.frame-func {
  color: #142018;
  word-break: break-all;
}

.frame-location {
  color: #56665d;
  font-size: 11px;
}

.raw-text-section {
  border-top: 1px solid rgba(48, 74, 61, 0.12);
}

.raw-text-header {
  padding: 10px 16px;
  background: #eef0e7;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #304a3d;
}

.raw-text-content {
  margin: 0;
  padding: 12px 16px;
  font-size: 11px;
  line-height: 1.6;
  color: #142018;
  background: rgba(48, 74, 61, 0.03);
  overflow-x: auto;
  white-space: pre;
  max-height: 500px;
  overflow-y: auto;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
}

@media (max-width: 1200px) {
  .imported-layout {
    grid-template-columns: 1fr;
  }

  .imported-chart {
    min-height: 320px;
  }
}
</style>
