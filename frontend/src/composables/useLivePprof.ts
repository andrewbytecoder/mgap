import { computed, onBeforeUnmount, watch } from 'vue'
import { useLivePprofStore } from '@/stores/livePprof'
import { savePreferences } from '@/services/preferences'
import type { MetricKey } from '@/types'

export function useLivePprof() {
  const store = useLivePprofStore()
  let timer: number | undefined

  // 生命周期：watch preferences 持久化
  watch(
    () => store.preferences,
    value => {
      savePreferences(value)
    },
    { deep: true }
  )

  onBeforeUnmount(() => {
    stopRecording()
  })

  async function tick() {
    if (!store.recording) return
    store.error = ''
    const sessionId = store.recordingSession
    const metrics = store.enabledMetrics.map(item => item.key)
    await Promise.all(metrics.map(metric => store.sampleMetric(metric, sessionId)))
    if (!store.recording || sessionId !== store.recordingSession) return
    timer = window.setTimeout(() => {
      void tick()
    }, store.preferences.sampleInterval)
  }

  function startRecording() {
    stopRecording()
    store.clearData()
    store.recordingSession += 1
    if (store.preferences.sampleInterval < store.preferences.cpuProfileSeconds * 1000) {
      store.preferences.sampleInterval = store.preferences.cpuProfileSeconds * 1000
    }
    store.recording = true
    void tick()
  }

  function stopRecording() {
    store.recording = false
    store.recordingSession += 1
    if (timer) {
      window.clearTimeout(timer)
      timer = undefined
    }
  }

  return {
    state: store,
    enabledMetrics: computed(() => store.enabledMetrics),
    hasEndpointError: computed(() => store.hasEndpointError),
    filteredGraphData: computed(() => store.filteredGraphData),
    bootstrap: store.bootstrap,
    detectEndpoints: store.detectEndpoints,
    refreshProfileCatalog: store.refreshProfileCatalog,
    startRecording,
    stopRecording,
    clearData: store.clearData,
    importProfile: store.importProfile,
    exportProfile: store.exportProfile,
    openProfileText: store.openProfileText,
    downloadProfile: store.downloadProfile,
    openProfileFlamegraph: store.openProfileFlamegraph
  }
}
