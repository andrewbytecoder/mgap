import type { GraphData, MetricPoint, MetricsSnapshot, SeriesPoint } from '../types'
import type { MetricKey } from '../types'

const stickySampleWindow = 4

export function newGraphData(): GraphData {
  return {
    lineTable: {},
    dates: [],
    stickyKeys: {},
    imported: false
  }
}

export function importedGraphData(snapshot: MetricsSnapshot, metric: MetricKey, cpuProfileSeconds: number): GraphData {
  const date = new Date(snapshot.timestamp / 1_000_000)
  const lineTable: GraphData['lineTable'] = {}

  for (const item of snapshot.items) {
    const key = getKey(item)
    lineTable[key] = {
      name: key,
      points: [
        {
          date,
          flat: normalizeMetricValue(metric, item.flat, cpuProfileSeconds, snapshot.total),
          cum: normalizeMetricValue(metric, item.cum, cpuProfileSeconds, snapshot.total)
        }
      ]
    }
  }

  return {
    lineTable,
    dates: [date],
    stickyKeys: {},
    imported: true,
    sourceLabel: snapshot.url,
    stacks: snapshot.stacks
  }
}

export function importedTimelineGraphData(
  snapshots: MetricsSnapshot[],
  metric: MetricKey,
  topN: number,
  retainedSamples: number,
  cpuProfileSeconds: number
): GraphData {
  return snapshots.reduce(
    (graphData, snapshot) => appendGraphData(graphData, snapshot, metric, topN, retainedSamples, cpuProfileSeconds),
    newGraphData()
  )
}

export function filterGraphDataByMinutes(graphData: GraphData, minutes: number): GraphData {
  if (graphData.dates.length === 0 || graphData.imported || minutes <= 0) return graphData

  const last = graphData.dates[graphData.dates.length - 1]
  const cutoff = last.getTime() - minutes * 60 * 1000
  const visibleDates = graphData.dates.filter(date => date.getTime() >= cutoff)
  const visibleSet = new Set(visibleDates.map(date => date.getTime()))

  return {
    ...graphData,
    dates: visibleDates,
    lineTable: Object.fromEntries(
      Object.entries(graphData.lineTable).map(([key, line]) => [
        key,
        {
          ...line,
          points: line.points.filter(point => visibleSet.has(point.date.getTime()))
        }
      ])
    )
  }
}

function getKey(item: MetricPoint): string {
  return `${item.function} ${item.line}`.trimEnd()
}

export function appendGraphData(
  graphData: GraphData,
  snapshot: MetricsSnapshot,
  metric: MetricKey,
  topN: number,
  retainedSamples: number,
  cpuProfileSeconds: number
): GraphData {
  const next: GraphData = {
    dates: [...graphData.dates],
    stickyKeys: { ...graphData.stickyKeys },
    imported: false,
    sourceLabel: snapshot.url,
    stacks: snapshot.stacks,
    lineTable: Object.fromEntries(
      Object.entries(graphData.lineTable).map(([key, line]) => [
        key,
        {
          ...line,
          points: [...line.points]
        }
      ])
    )
  }

  const date = new Date(snapshot.timestamp / 1_000_000)
  const sortedItems = [...snapshot.items]
    .filter(item => item.flat > 0)
    .sort((left, right) => right.flat - left.flat)

  next.dates.push(date)
  const currentSampleIndex = next.dates.length - 1

  if (next.dates.length > retainedSamples) {
    const removeCount = next.dates.length - retainedSamples
    const removedDates = next.dates.splice(0, removeCount)
    const removedTimeSet = new Set(removedDates.map(item => item.getTime()))
    for (const line of Object.values(next.lineTable)) {
      line.points = line.points.filter(point => !removedTimeSet.has(point.date.getTime()))
    }
  }

  for (const [key, lastSeen] of Object.entries(next.stickyKeys)) {
    if (currentSampleIndex-lastSeen > stickySampleWindow) {
      delete next.stickyKeys[key]
    }
  }

  const selectedKeys = new Set(sortedItems.slice(0, topN).map(getKey))
  for (const key of Object.keys(next.stickyKeys)) {
    selectedKeys.add(key)
  }
  const items = sortedItems.filter(item => selectedKeys.has(getKey(item)))

  for (const line of Object.values(next.lineTable)) {
    line.points.push({
      date,
      flat: undefined,
      cum: undefined
    })
  }

  for (const item of items) {
    const key = getKey(item)
    next.stickyKeys[key] = currentSampleIndex
    if (!next.lineTable[key]) {
      const nullPoints: SeriesPoint[] = next.dates.map(existingDate => ({
        date: existingDate,
        flat: undefined,
        cum: undefined
      }))
      next.lineTable[key] = {
        name: key,
        points: nullPoints
      }
    }
    const line = next.lineTable[key]
    line.points[line.points.length - 1] = {
      date,
      flat: normalizeMetricValue(metric, item.flat, cpuProfileSeconds, snapshot.total),
      cum: normalizeMetricValue(metric, item.cum, cpuProfileSeconds, snapshot.total)
    }
  }

  const totalLine = next.lineTable.total
  if (totalLine && totalLine.points.length > 0) {
    const lastPoint = totalLine.points[totalLine.points.length - 1]
    totalLine.points[totalLine.points.length - 1] = {
      ...lastPoint,
      flat: normalizeMetricValue(metric, snapshot.total, cpuProfileSeconds, snapshot.total),
      cum: normalizeMetricValue(metric, snapshot.total, cpuProfileSeconds, snapshot.total)
    }
  }

  return next
}

function normalizeMetricValue(metric: MetricKey, value: number, cpuProfileSeconds: number, cpuTotal: number): number {
  if (metric !== 'cpu') return value
  const seconds = Math.max(cpuProfileSeconds, 1)

  // CPU profiles may come back either as sampled counts or as CPU nanoseconds.
  // Counts are usually in the tens/hundreds range per second, while nanoseconds are huge.
  if (cpuTotal > seconds * 100_000) {
    return (value / (seconds * 1_000_000_000)) * 100
  }

  // With Go's default 100Hz CPU sampling, one fully busy core is roughly 100 samples/sec.
  // That makes "count per second" a good approximation of CPU percent.
  return value / seconds
}
