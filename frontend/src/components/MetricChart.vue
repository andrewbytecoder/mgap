<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { ECharts, EChartsOption, SeriesOption } from 'echarts'
import type { GraphData, MetricInfo, MetricKey } from '../types'
import { formatBytes, formatNumber, formatPercent } from '../utils/format'

type ViewMetricInfo = MetricInfo & { key: MetricKey }

const props = defineProps<{
  data: GraphData
  metric: ViewMetricInfo
  preference: {
    total: boolean
    flatOrCum: 'flat' | 'cum'
    smooth: boolean
  }
}>()

const chartRef = ref<HTMLDivElement | null>(null)
let chart: ECharts | undefined

const option = computed<EChartsOption>(() => {
  const axisValues = Object.values(props.data.lineTable).flatMap(line =>
    line.points.map(point => point[props.preference.flatOrCum]).filter((value): value is number => value !== undefined)
  )
  const leftPadding = estimateAxisPadding(props.metric.key, axisValues)
  const keys = Object.keys(props.data.lineTable).filter(key => props.preference.total || key !== 'total')
  const dataset = keys.map(key => ({
    id: key,
    source: props.data.lineTable[key].points.map(point => ({
      date: point.date,
      flat: point.flat,
      cum: point.cum
    }))
  }))
  const series: SeriesOption[] = keys.map(key => ({
    type: 'line',
    datasetId: key,
    name: key,
    smooth: props.preference.smooth,
    showSymbol: false,
    lineStyle: {
      width: key === 'total' ? 3 : 2
    },
    emphasis: {
      focus: 'series'
    },
    encode: {
      x: 'date',
      y: props.preference.flatOrCum,
      tooltip: [props.preference.flatOrCum]
    }
  }))

  return {
    animationDuration: 150,
    color: ['#d86e3f', '#304a3d', '#3f6a8f', '#c98928', '#8d5d8b', '#4d7c53', '#7f5539'],
    grid: {
      containLabel: true,
      left: leftPadding,
      right: 24,
      top: 48,
      bottom: 60
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#132019',
      borderColor: '#132019',
      textStyle: {
        color: '#f8f7f2'
      },
      formatter: (value: any) => formatTooltip(value, props.metric.key, props.preference.flatOrCum)
    },
    dataset,
    xAxis: {
      type: 'time',
      axisLabel: {
        color: '#526158'
      }
    },
    yAxis: {
      axisLabel: {
        color: '#526158',
        margin: 18,
        formatter: (value: number) => formatMetricValue(props.metric.key, value)
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(48, 74, 61, 0.12)'
        }
      }
    },
    series,
    toolbox: {
      right: 8,
      feature: {
        restore: {},
        saveAsImage: {}
      }
    }
  }
})

watch(
  option,
  value => {
    chart?.setOption(value, true)
  },
  { deep: true }
)

onMounted(() => {
  if (!chartRef.value) return
  chart = echarts.init(chartRef.value)
  chart.setOption(option.value, true)
  window.addEventListener('resize', resize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  chart?.dispose()
})

function resize() {
  chart?.resize()
}

function formatMetricValue(metric: MetricKey, value: number): string {
  if (metric === 'cpu') return formatPercent(value)
  if (metric === 'goroutine') return formatNumber(value)
  return formatBytes(value)
}

function estimateAxisPadding(metric: MetricKey, values: number[]): number {
  const baseline = metric === 'goroutine' ? 64 : 72
  if (values.length === 0) return baseline

  const labelLength = values
    .map(value => formatMetricValue(metric, value).length)
    .reduce((max, current) => Math.max(max, current), 0)

  const estimated = 22 + labelLength * 7
  return Math.max(baseline, Math.min(estimated, 94))
}

function formatTooltip(items: any[], metric: MetricKey, mode: 'flat' | 'cum'): string {
  const lines = items
    .filter(item => item.value[mode] !== undefined)
    .sort((left, right) => right.value[mode] - left.value[mode])
    .slice(0, 20)
    .map(item => {
      const value = formatMetricValue(metric, item.value[mode])
      return `<div style="display:flex;justify-content:space-between;gap:16px;"><span>${item.marker}${item.seriesName}</span><strong>${value}</strong></div>`
    })
    .join('')

  const title = items[0]?.axisValueLabel ?? ''
  return `<div style="min-width:260px"><div style="font-weight:700;margin-bottom:8px">${title}</div>${lines}</div>`
}
</script>

<template>
  <div ref="chartRef" style="height: 100%; min-height: 320px; width: 100%" />
</template>
