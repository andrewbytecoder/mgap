import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { wailsApi } from '@/wails'
import {
  appendGraphData,
  filterGraphDataByMinutes,
  importedGraphData,
  importedTimelineGraphData,
  newGraphData
} from '@/utils/graph'
import { loadPreferences } from '@/services/preferences'
import type {
  EndpointResult,
  FlamegraphNode,
  GraphData,
  MetricInfo,
  MetricKey,
  Preferences,
  ProfileCatalogEntry,
  ProfileMeta
} from '@/types'

const metricKeys: MetricKey[] = [
  'cpu',
  'heap',
  'allocs',
  'goroutine',
  'block',
  'mutex',
  'threadcreate'
]

function createDefaultBusyMetrics(): Record<MetricKey, boolean> {
  return {
    cpu: false,
    heap: false,
    allocs: false,
    goroutine: false,
    block: false,
    mutex: false,
    threadcreate: false
  }
}

function createDefaultGraphData(): Record<MetricKey, GraphData> {
  return {
    cpu: newGraphData(),
    heap: newGraphData(),
    allocs: newGraphData(),
    goroutine: newGraphData(),
    block: newGraphData(),
    mutex: newGraphData(),
    threadcreate: newGraphData()
  }
}

function createDefaultProfileMeta(): Record<MetricKey, ProfileMeta> {
  return {
    cpu: { metric: 'cpu', source: '', fileName: '', imported: false, exportable: false },
    heap: { metric: 'heap', source: '', fileName: '', imported: false, exportable: false },
    allocs: { metric: 'allocs', source: '', fileName: '', imported: false, exportable: false },
    goroutine: {
      metric: 'goroutine',
      source: '',
      fileName: '',
      imported: false,
      exportable: false
    },
    block: { metric: 'block', source: '', fileName: '', imported: false, exportable: false },
    mutex: { metric: 'mutex', source: '', fileName: '', imported: false, exportable: false },
    threadcreate: {
      metric: 'threadcreate',
      source: '',
      fileName: '',
      imported: false,
      exportable: false
    }
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

export const useLivePprofStore = defineStore('livePprof', () => {
  // State
  const ready = ref(false)
  const loadingDetect = ref(false)
  const recording = ref(false)
  const busyMetrics = ref<Record<MetricKey, boolean>>(createDefaultBusyMetrics())
  const error = ref('')
  const detectResults = ref<EndpointResult[]>([])
  const profileCatalog = ref<ProfileCatalogEntry[]>([])
  const loadingProfiles = ref(false)
  const profileRawText = ref('')
  const profileFlamegraph = ref<FlamegraphNode | null>(null)
  const graphData = ref<Record<MetricKey, GraphData>>(createDefaultGraphData())
  const profileMeta = ref<Record<MetricKey, ProfileMeta>>(createDefaultProfileMeta())
  const preferences = ref<Preferences>(
    loadPreferences('http://localhost:6060/debug/pprof')
  )
  const metricInfo = ref<MetricInfo[]>([])
  const appInfo = ref<Record<string, string>>({})
  const recordingSession = ref(0)

  // Getters
  const enabledMetrics = computed(() =>
    metricInfo.value.filter(
      (item): item is MetricInfo & { key: MetricKey } =>
        isMetricKey(item.key) && preferences.value.metrics[item.key as MetricKey].enabled
    )
  )

  const hasEndpointError = computed(() => !preferences.value.endpointInput.trim())

  const filteredGraphData = computed(() => ({
    cpu: filterGraphDataByMinutes(graphData.value.cpu, preferences.value.timeRangeMinutes),
    heap: filterGraphDataByMinutes(
      graphData.value.heap,
      preferences.value.timeRangeMinutes
    ),
    allocs: filterGraphDataByMinutes(
      graphData.value.allocs,
      preferences.value.timeRangeMinutes
    ),
    goroutine: filterGraphDataByMinutes(
      graphData.value.goroutine,
      preferences.value.timeRangeMinutes
    ),
    block: filterGraphDataByMinutes(
      graphData.value.block,
      preferences.value.timeRangeMinutes
    ),
    mutex: filterGraphDataByMinutes(
      graphData.value.mutex,
      preferences.value.timeRangeMinutes
    ),
    threadcreate: filterGraphDataByMinutes(
      graphData.value.threadcreate,
      preferences.value.timeRangeMinutes
    )
  }))

  // Actions
  async function bootstrap() {
    try {
      const [
        initialURL,
        info,
        aInfo,
        cpuMeta,
        heapMeta,
        allocsMeta,
        goroutineMeta,
        blockMeta,
        mutexMeta,
        threadMeta
      ] = await Promise.all([
        wailsApi.initialURL(),
        wailsApi.availableMetrics(),
        wailsApi.appInfo(),
        wailsApi.profileMeta('cpu'),
        wailsApi.profileMeta('heap'),
        wailsApi.profileMeta('allocs'),
        wailsApi.profileMeta('goroutine'),
        wailsApi.profileMeta('block'),
        wailsApi.profileMeta('mutex'),
        wailsApi.profileMeta('threadcreate')
      ])
      metricInfo.value = info
      appInfo.value = aInfo
      profileMeta.value = {
        cpu: cpuMeta,
        heap: heapMeta,
        allocs: allocsMeta,
        goroutine: goroutineMeta,
        block: blockMeta,
        mutex: mutexMeta,
        threadcreate: threadMeta
      }
      preferences.value = loadPreferences(initialURL)
      ready.value = true
      if (preferences.value.endpointInput.trim()) {
        await detectEndpoints()
        await refreshProfileCatalog()
      }
    } catch (err) {
      error.value = toMessage(err)
      ready.value = true
    }
  }

  async function detectEndpoints() {
    loadingDetect.value = true
    error.value = ''
    try {
      detectResults.value = await wailsApi.detectURL(preferences.value.endpointInput)
    } catch (err) {
      error.value = toMessage(err)
      detectResults.value = []
    } finally {
      loadingDetect.value = false
    }
  }

  async function refreshProfileCatalog() {
    if (!preferences.value.endpointInput.trim()) return
    loadingProfiles.value = true
    try {
      profileCatalog.value = await wailsApi.fetchProfileCatalog(
        preferences.value.endpointInput
      )
    } catch (err) {
      error.value = `Profiles: ${toMessage(err)}`
      profileCatalog.value = []
    } finally {
      loadingProfiles.value = false
    }
  }

  function clearData() {
    graphData.value = createDefaultGraphData()
  }

  async function sampleMetric(metric: MetricKey, sessionId: number) {
    if (busyMetrics.value[metric]) return
    const profileSeconds = metric === 'cpu' ? preferences.value.cpuProfileSeconds : 1
    busyMetrics.value[metric] = true
    try {
      const snapshot = await wailsApi.fetchMetrics(
        preferences.value.endpointInput,
        metric,
        profileSeconds,
        preferences.value.useMock
      )
      if (!recording.value || sessionId !== recordingSession.value) return
      graphData.value[metric] = appendGraphData(
        graphData.value[metric],
        snapshot,
        metric,
        preferences.value.metrics[metric].topN,
        preferences.value.retainedSamples,
        preferences.value.cpuProfileSeconds
      )
      profileMeta.value[metric] = await wailsApi.profileMeta(metric)
    } catch (err) {
      if (!recording.value || sessionId !== recordingSession.value) return
      error.value = `${metric.toUpperCase()}: ${toMessage(err)}`
    } finally {
      busyMetrics.value[metric] = false
    }
  }

  async function importProfile(metric: MetricKey) {
    error.value = ''
    try {
      const snapshots = await wailsApi.importProfiles(metric)
      if (!snapshots || snapshots.length === 0) return
      graphData.value[metric] =
        snapshots.length === 1
          ? importedGraphData(snapshots[0], metric, preferences.value.cpuProfileSeconds)
          : importedTimelineGraphData(
              snapshots,
              metric,
              preferences.value.metrics[metric].topN,
              preferences.value.retainedSamples,
              preferences.value.cpuProfileSeconds
            )
      profileMeta.value[metric] = await wailsApi.profileMeta(metric)
      preferences.value.metrics[metric].enabled = true
    } catch (err) {
      error.value = `${metric.toUpperCase()} import: ${toMessage(err)}`
    }
  }

  async function exportProfile(metric: MetricKey) {
    error.value = ''
    try {
      await wailsApi.exportProfile(metric)
      profileMeta.value[metric] = await wailsApi.profileMeta(metric)
    } catch (err) {
      error.value = `${metric.toUpperCase()} export: ${toMessage(err)}`
    }
  }

  async function openProfileText(profile: string, debug: number) {
    error.value = ''
    profileFlamegraph.value = null
    try {
      profileRawText.value = await wailsApi.fetchProfileText(
        preferences.value.endpointInput,
        profile,
        debug,
        preferences.value.cpuProfileSeconds
      )
    } catch (err) {
      error.value = `${profile} text: ${toMessage(err)}`
    }
  }

  async function downloadProfile(profile: string, debug: number) {
    error.value = ''
    try {
      await wailsApi.downloadProfile(
        preferences.value.endpointInput,
        profile,
        debug,
        preferences.value.cpuProfileSeconds
      )
    } catch (err) {
      error.value = `${profile} download: ${toMessage(err)}`
    }
  }

  async function openProfileFlamegraph(profile: string) {
    error.value = ''
    profileRawText.value = ''
    try {
      profileFlamegraph.value = await wailsApi.getProfileFlamegraph(
        preferences.value.endpointInput,
        profile,
        preferences.value.cpuProfileSeconds
      )
    } catch (err) {
      error.value = `${profile} flamegraph: ${toMessage(err)}`
    }
  }

  return {
    // state
    ready,
    loadingDetect,
    recording,
    busyMetrics,
    error,
    detectResults,
    profileCatalog,
    loadingProfiles,
    profileRawText,
    profileFlamegraph,
    graphData,
    profileMeta,
    preferences,
    metricInfo,
    appInfo,
    recordingSession,
    // getters
    enabledMetrics,
    hasEndpointError,
    filteredGraphData,
    // actions
    bootstrap,
    detectEndpoints,
    refreshProfileCatalog,
    clearData,
    sampleMetric,
    importProfile,
    exportProfile,
    openProfileText,
    downloadProfile,
    openProfileFlamegraph
  }
})
