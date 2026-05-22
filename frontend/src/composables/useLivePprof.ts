import { computed, onBeforeUnmount, reactive, watch } from 'vue'
import { wailsApi } from '../wails'
import { appendGraphData, filterGraphDataByMinutes, importedGraphData, importedTimelineGraphData, newGraphData } from '../utils/graph'
import { loadPreferences, savePreferences } from '../services/preferences'
import type { EndpointResult, GraphData, MetricInfo, MetricKey, Preferences, ProfileMeta } from '../types'

const metricKeys: MetricKey[] = ['cpu', 'heap', 'allocs', 'goroutine']

interface State {
  ready: boolean
  loadingDetect: boolean
  recording: boolean
  busyMetrics: Record<MetricKey, boolean>
  error: string
  detectResults: EndpointResult[]
  graphData: Record<MetricKey, GraphData>
  profileMeta: Record<MetricKey, ProfileMeta>
  preferences: Preferences
  metricInfo: MetricInfo[]
  appInfo: Record<string, string>
}

let timer: number | undefined
let recordingSession = 0

export function useLivePprof() {
  const state = reactive<State>({
    ready: false,
    loadingDetect: false,
    recording: false,
    busyMetrics: {
      cpu: false,
      heap: false,
      allocs: false,
      goroutine: false
    },
    error: '',
    detectResults: [],
    graphData: {
      cpu: newGraphData(),
      heap: newGraphData(),
      allocs: newGraphData(),
      goroutine: newGraphData()
    },
    profileMeta: {
      cpu: { metric: 'cpu', source: '', fileName: '', imported: false, exportable: false },
      heap: { metric: 'heap', source: '', fileName: '', imported: false, exportable: false },
      allocs: { metric: 'allocs', source: '', fileName: '', imported: false, exportable: false },
      goroutine: { metric: 'goroutine', source: '', fileName: '', imported: false, exportable: false }
    },
    preferences: loadPreferences('http://localhost:6060/debug/pprof'),
    metricInfo: [],
    appInfo: {}
  })

  const enabledMetrics = computed(() =>
    state.metricInfo.filter((item): item is MetricInfo & { key: MetricKey } => isMetricKey(item.key) && state.preferences.metrics[item.key].enabled)
  )
  const hasEndpointError = computed(() => !state.preferences.endpointInput.trim())

  async function bootstrap() {
    try {
      const [initialURL, metricInfo, appInfo, cpuMeta, heapMeta, allocsMeta, goroutineMeta] = await Promise.all([
        wailsApi.initialURL(),
        wailsApi.availableMetrics(),
        wailsApi.appInfo(),
        wailsApi.profileMeta('cpu'),
        wailsApi.profileMeta('heap'),
        wailsApi.profileMeta('allocs'),
        wailsApi.profileMeta('goroutine')
      ])
      state.metricInfo = metricInfo
      state.appInfo = appInfo
      state.profileMeta = {
        cpu: cpuMeta,
        heap: heapMeta,
        allocs: allocsMeta,
        goroutine: goroutineMeta
      }
      state.preferences = loadPreferences(initialURL)
      state.ready = true
      if (state.preferences.endpointInput.trim()) {
        await detectEndpoints()
      }
    } catch (error) {
      state.error = toMessage(error)
      state.ready = true
    }
  }

  async function detectEndpoints() {
    state.loadingDetect = true
    state.error = ''
    try {
      state.detectResults = await wailsApi.detectURL(state.preferences.endpointInput)
    } catch (error) {
      state.error = toMessage(error)
      state.detectResults = []
    } finally {
      state.loadingDetect = false
    }
  }

  function clearData() {
    state.graphData = {
      cpu: newGraphData(),
      heap: newGraphData(),
      allocs: newGraphData(),
      goroutine: newGraphData()
    }
  }

  function stopRecording() {
    state.recording = false
    recordingSession += 1
    if (timer) {
      window.clearTimeout(timer)
      timer = undefined
    }
  }

  async function sampleMetric(metric: MetricKey, sessionId: number) {
    if (state.busyMetrics[metric]) return

    const profileSeconds = metric === 'cpu' ? state.preferences.cpuProfileSeconds : 1
    state.busyMetrics[metric] = true
    try {
      const snapshot = await wailsApi.fetchMetrics(
        state.preferences.endpointInput,
        metric,
        profileSeconds,
        state.preferences.useMock
      )
      if (!state.recording || sessionId !== recordingSession) return
      state.graphData[metric] = appendGraphData(
        state.graphData[metric],
        snapshot,
        metric,
        state.preferences.metrics[metric].topN,
        state.preferences.retainedSamples,
        state.preferences.cpuProfileSeconds
      )
      state.profileMeta[metric] = await wailsApi.profileMeta(metric)
    } catch (error) {
      if (!state.recording || sessionId !== recordingSession) return
      state.error = `${metric.toUpperCase()}: ${toMessage(error)}`
    } finally {
      state.busyMetrics[metric] = false
    }
  }

  async function importProfile(metric: MetricKey) {
    state.error = ''
    try {
      const snapshots = await wailsApi.importProfiles(metric)
      if (!snapshots || snapshots.length === 0) return
      state.graphData[metric] =
        snapshots.length === 1
          ? importedGraphData(snapshots[0], metric, state.preferences.cpuProfileSeconds)
          : importedTimelineGraphData(
              snapshots,
              metric,
              state.preferences.metrics[metric].topN,
              state.preferences.retainedSamples,
              state.preferences.cpuProfileSeconds
            )
      state.profileMeta[metric] = await wailsApi.profileMeta(metric)
      state.preferences.metrics[metric].enabled = true
    } catch (error) {
      state.error = `${metric.toUpperCase()} import: ${toMessage(error)}`
    }
  }

  async function exportProfile(metric: MetricKey) {
    state.error = ''
    try {
      await wailsApi.exportProfile(metric)
      state.profileMeta[metric] = await wailsApi.profileMeta(metric)
    } catch (error) {
      state.error = `${metric.toUpperCase()} export: ${toMessage(error)}`
    }
  }

  async function tick() {
    if (!state.recording) return
    state.error = ''
    const sessionId = recordingSession

    const metrics = enabledMetrics.value.map(item => item.key)
    await Promise.all(metrics.map(metric => sampleMetric(metric, sessionId)))

    if (!state.recording || sessionId !== recordingSession) return
    timer = window.setTimeout(() => {
      void tick()
    }, state.preferences.sampleInterval)
  }

  function startRecording() {
    stopRecording()
    clearData()
    recordingSession += 1
    if (state.preferences.sampleInterval < state.preferences.cpuProfileSeconds * 1000) {
      state.preferences.sampleInterval = state.preferences.cpuProfileSeconds * 1000
    }
    state.recording = true
    void tick()
  }

  watch(
    () => state.preferences,
    value => {
      savePreferences(value)
    },
    { deep: true }
  )

  onBeforeUnmount(() => {
    stopRecording()
  })

  return {
    state,
    enabledMetrics,
    hasEndpointError,
    filteredGraphData: computed(() => ({
      cpu: filterGraphDataByMinutes(state.graphData.cpu, state.preferences.timeRangeMinutes),
      heap: filterGraphDataByMinutes(state.graphData.heap, state.preferences.timeRangeMinutes),
      allocs: filterGraphDataByMinutes(state.graphData.allocs, state.preferences.timeRangeMinutes),
      goroutine: filterGraphDataByMinutes(state.graphData.goroutine, state.preferences.timeRangeMinutes)
    })),
    bootstrap,
    detectEndpoints,
    startRecording,
    stopRecording,
    clearData,
    importProfile,
    exportProfile
  }
}

function toMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  return 'Unknown error'
}

function isMetricKey(value: string): value is MetricKey {
  return metricKeys.includes(value as MetricKey)
}
