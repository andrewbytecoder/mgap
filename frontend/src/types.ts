export type MetricKey = 'cpu' | 'heap' | 'allocs' | 'goroutine' | 'block' | 'mutex' | 'threadcreate'

export interface MetricInfo {
  key: string
  label: string
  description: string
  unit: string
}

export interface MetricPoint {
  function: string
  line: string
  flat: number
  cum: number
}

export interface StackFrame {
  func: string
  file: string
  line: number
}

export interface GoroutineStack {
  count: number
  frames: StackFrame[]
}

export interface MetricsSnapshot {
  type: string
  url: string
  timestamp: number
  total: number
  items: MetricPoint[]
  stacks?: GoroutineStack[]
  rawText?: string
  defaultSampleType?: string
  defaultSampleUnit?: string
  durationNanos?: number
  period?: number
  periodType?: string
  periodUnit?: string
}

export interface ProfileMeta {
  metric: string
  source: string
  fileName: string
  imported: boolean
  exportable: boolean
}

export interface ProfileCatalogEntry {
  name: string
  count: number
  description: string
  supportsChart: boolean
  supportsRawText: boolean
  supportsImport: boolean
  supportsExport: boolean
  supportsFlame: boolean
}

export interface FlamegraphNode {
  name: string
  fullName: string
  fileName: string
  value: number
  children: FlamegraphNode[]
}

export interface EndpointResult {
  endpoint: string
  statusCode: number
  statusText: string
  body: string
  error: string
}

export interface SeriesPoint {
  date: Date
  flat?: number
  cum?: number
}

export interface SeriesLine {
  name: string
  points: SeriesPoint[]
}

export type LineTable = Record<string, SeriesLine>

export interface GraphData {
  lineTable: LineTable
  dates: Date[]
  stickyKeys: Record<string, number>
  imported: boolean
  sourceLabel?: string
  stacks?: GoroutineStack[]
  rawText?: string
}

export interface GraphPreference {
  enabled: boolean
  total: boolean
  flatOrCum: 'flat' | 'cum'
  topN: number
}

export interface Preferences {
  endpointInput: string
  sampleInterval: number
  retainedSamples: number
  cpuProfileSeconds: number
  smooth: boolean
  useMock: boolean
  timeRangeMinutes: number
  metrics: Record<MetricKey, GraphPreference>
}
