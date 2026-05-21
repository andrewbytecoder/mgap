<script setup lang="ts">
import { computed, onMounted } from 'vue'
import MetricChart from './components/MetricChart.vue'
import ImportedProfileView from './components/ImportedProfileView.vue'
import { useLivePprof } from './composables/useLivePprof'
import { formatBytes, formatNumber, formatPercent, formatTimestamp } from './utils/format'
import type { MetricKey } from './types'

const {
  state,
  enabledMetrics,
  hasEndpointError,
  filteredGraphData,
  bootstrap,
  detectEndpoints,
  startRecording,
  stopRecording,
  clearData,
  importProfile,
  exportProfile
} = useLivePprof()

const statusText = computed(() => (state.recording ? 'Sampling live profiles' : 'Idle'))
const hiddenMetrics = computed(() =>
  state.metricInfo.filter(metric => !state.preferences.metrics[metric.key as MetricKey]?.enabled)
)

onMounted(() => {
  void bootstrap()
})

function metricSummary(metric: MetricKey): string {
  const data = state.graphData[metric]
  const latestDate = data.dates[data.dates.length - 1]
  if (!latestDate) return 'No samples yet'

  const totalLine = data.lineTable.total
  const totalPoint = totalLine ? totalLine.points[totalLine.points.length - 1] : undefined
  const total = totalPoint?.flat ?? totalPoint?.cum
  if (total === undefined) return `Last sample ${latestDate.toLocaleTimeString()}`

  if (metric === 'cpu') return `${formatPercent(total)} at ${latestDate.toLocaleTimeString()}`
  if (metric === 'goroutine') return `${formatNumber(total)} goroutines at ${latestDate.toLocaleTimeString()}`
  return `${formatBytes(total)} at ${latestDate.toLocaleTimeString()}`
}
</script>

<template>
  <v-app>
    <v-main>
      <div class="shell">
        <section class="hero">
          <div class="hero-copy">
            <p class="eyebrow">Wails Desktop Monitor</p>
            <h1>live-pprof</h1>
            <p class="hero-text">
              Track heap, allocs, CPU and goroutine profiles from a desktop app instead of juggling a browser tab and a
              local web server.
            </p>
            <div class="hero-meta mono">
              <span>{{ statusText }}</span>
              <span>{{ state.appInfo.stack }}</span>
            </div>
          </div>
          <div class="hero-blobs">
            <div class="blob blob-one"></div>
            <div class="blob blob-two"></div>
          </div>
        </section>

        <v-row class="mt-2" dense>
          <v-col cols="12" lg="4">
            <v-card class="control-card" elevation="0">
              <div class="card-title">
                <div>
                  <p class="section-kicker">Connection</p>
                  <h2>Endpoint</h2>
                </div>
                <v-chip color="secondary" variant="flat" class="mono">{{ state.preferences.useMock ? 'Mock' : 'Live' }}</v-chip>
              </div>

              <v-text-field
                v-model="state.preferences.endpointInput"
                label="pprof URL or port"
                hint="Examples: 6060, localhost:6060, http://localhost:6060/debug/pprof"
                placeholder="Enter endpoint manually"
                persistent-hint
                prepend-inner-icon="mdi-lan-connect"
              />

              <div class="action-row">
                <v-btn
                  color="primary"
                  variant="flat"
                  prepend-icon="mdi-radar"
                  :loading="state.loadingDetect"
                  :disabled="hasEndpointError"
                  @click="detectEndpoints"
                >
                  Detect
                </v-btn>
                <v-btn color="secondary" variant="tonal" prepend-icon="mdi-delete-outline" @click="clearData">
                  Clear
                </v-btn>
                <v-btn
                  :color="state.recording ? 'error' : 'success'"
                  variant="flat"
                  :disabled="hasEndpointError"
                  :prepend-icon="state.recording ? 'mdi-stop-circle-outline' : 'mdi-play-circle-outline'"
                  @click="state.recording ? stopRecording() : startRecording()"
                >
                  {{ state.recording ? 'Stop' : 'Start' }}
                </v-btn>
              </div>

              <div class="field-grid">
                <v-text-field
                  v-model.number="state.preferences.sampleInterval"
                  label="Sample Interval (ms)"
                  type="number"
                  min="100"
                  step="100"
                />
                <v-text-field
                  v-model.number="state.preferences.retainedSamples"
                  label="Retained Samples"
                  type="number"
                  min="1"
                  step="10"
                />
                <v-text-field
                  v-model.number="state.preferences.cpuProfileSeconds"
                  label="CPU Profile (s)"
                  type="number"
                  min="1"
                  step="1"
                />
              </div>

              <div class="switches">
                <v-switch v-model="state.preferences.smooth" color="secondary" label="Smooth chart lines" hide-details />
                <v-switch v-model="state.preferences.useMock" color="accent" label="Use embedded mock data" hide-details />
              </div>

              <v-alert v-if="state.error" type="error" variant="tonal" class="mt-3">
                {{ state.error }}
              </v-alert>
            </v-card>
          </v-col>

          <v-col cols="12" lg="8">
            <v-card class="detect-card" elevation="0">
              <div class="card-title">
                <div>
                  <p class="section-kicker">Inspection</p>
                  <h2>Endpoint detection</h2>
                </div>
              </div>

              <v-skeleton-loader v-if="!state.ready" type="list-item-three-line" />
              <v-expansion-panels v-else variant="accordion">
                <v-expansion-panel v-for="result in state.detectResults" :key="result.endpoint">
                  <v-expansion-panel-title>
                    <div class="endpoint-row">
                      <span class="mono endpoint-name">{{ result.endpoint }}</span>
                      <v-chip
                        v-if="result.statusText"
                        :color="result.statusCode >= 200 && result.statusCode < 400 ? 'success' : 'error'"
                        size="small"
                        variant="flat"
                      >
                        {{ result.statusText }}
                      </v-chip>
                      <v-chip v-if="result.error" color="error" size="small" variant="tonal">Error</v-chip>
                    </div>
                  </v-expansion-panel-title>
                  <v-expansion-panel-text>
                    <p v-if="result.error" class="error-copy">{{ result.error }}</p>
                    <pre v-if="result.body" class="endpoint-body mono">{{ result.body }}</pre>
                    <p v-if="!result.body && !result.error" class="subtle-copy">No response body returned.</p>
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>
            </v-card>
          </v-col>
        </v-row>

        <section class="metrics-section">
          <div class="card-title chart-title">
            <div>
              <p class="section-kicker">Charts</p>
              <h2>Profile timelines</h2>
            </div>
            <v-select
              v-model="state.preferences.timeRangeMinutes"
              :items="[
                { title: 'Last 5m', value: 5 },
                { title: 'Last 15m', value: 15 },
                { title: 'Last 30m', value: 30 },
                { title: 'Last 1h', value: 60 },
                { title: 'Last 6h', value: 360 },
                { title: 'All data', value: 0 }
              ]"
              class="time-range-select"
              item-title="title"
              item-value="value"
              label="Time range"
              hide-details
            />
          </div>

          <div class="metric-toggle-bar">
            <v-chip
              v-for="metric in state.metricInfo"
              :key="metric.key"
              :color="state.preferences.metrics[metric.key as MetricKey].enabled ? 'primary' : undefined"
              :prepend-icon="
                state.preferences.metrics[metric.key as MetricKey].enabled
                  ? 'mdi-chart-line'
                  : 'mdi-chart-line-variant'
              "
              :variant="state.preferences.metrics[metric.key as MetricKey].enabled ? 'flat' : 'outlined'"
              class="metric-toggle-chip"
              @click="state.preferences.metrics[metric.key as MetricKey].enabled = !state.preferences.metrics[metric.key as MetricKey].enabled"
            >
              {{ metric.label }}
            </v-chip>
          </div>

          <p v-if="hiddenMetrics.length" class="hidden-metric-hint">
            Hidden charts can be restored here at any time.
          </p>

          <v-row dense>
            <v-col v-for="metric in enabledMetrics" :key="metric.key" cols="12" xl="6">
              <v-card class="metric-card" elevation="0">
                <div class="metric-header">
                  <div>
                    <div class="metric-heading">
                      <div>
                        <h3>{{ metric.label }}</h3>
                        <p class="subtle-copy">{{ metric.description }}</p>
                      </div>
                    </div>
                    <p class="metric-summary">{{ metricSummary(metric.key) }}</p>
                  </div>

                  <div class="metric-controls">
                    <v-btn
                      size="small"
                      variant="tonal"
                      prepend-icon="mdi-file-import-outline"
                      @click="importProfile(metric.key)"
                    >
                      Import
                    </v-btn>
                    <v-btn
                      size="small"
                      variant="tonal"
                      prepend-icon="mdi-file-export-outline"
                      :disabled="!state.profileMeta[metric.key].exportable"
                      @click="exportProfile(metric.key)"
                    >
                      Export
                    </v-btn>
                    <v-select
                      v-model="state.preferences.metrics[metric.key].flatOrCum"
                      :items="[
                        { title: 'Flat', value: 'flat' },
                        { title: 'Cum', value: 'cum' }
                      ]"
                      item-title="title"
                      item-value="value"
                      label="Metric mode"
                      hide-details
                    />
                    <v-text-field
                      v-model.number="state.preferences.metrics[metric.key].topN"
                      type="number"
                      min="1"
                      max="50"
                      step="1"
                      label="Top N"
                      hide-details
                    />
                    <v-checkbox
                      v-model="state.preferences.metrics[metric.key].total"
                      label="Show total"
                      color="secondary"
                      hide-details
                    />
                  </div>
                </div>

                <p v-if="state.profileMeta[metric.key].source" class="profile-meta">
                  {{ state.profileMeta[metric.key].imported ? 'Imported' : 'Captured' }}:
                  <span class="mono">{{ state.profileMeta[metric.key].fileName || state.profileMeta[metric.key].source }}</span>
                </p>

                <div v-if="state.preferences.metrics[metric.key].enabled" class="chart-shell">
                  <ImportedProfileView
                    v-if="filteredGraphData[metric.key].imported"
                    :data="filteredGraphData[metric.key]"
                    :metric="metric.key"
                    :mode="state.preferences.metrics[metric.key].flatOrCum"
                    :top-n="state.preferences.metrics[metric.key].topN"
                  />
                  <MetricChart
                    v-else
                    :data="filteredGraphData[metric.key]"
                    :metric="metric"
                    :preference="{
                      total: state.preferences.metrics[metric.key].total,
                      flatOrCum: state.preferences.metrics[metric.key].flatOrCum,
                      smooth: state.preferences.smooth
                    }"
                  />
                </div>
                <div v-else class="empty-chart">
                  <p>This metric is currently hidden. Toggle it back on when you want it in the sampling loop.</p>
                </div>
              </v-card>
            </v-col>
          </v-row>
        </section>
      </div>
    </v-main>
  </v-app>
</template>

<style scoped>
.shell {
  padding: 28px;
}

.hero {
  position: relative;
  overflow: hidden;
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 32px;
  border-radius: 28px;
  background: linear-gradient(135deg, #1c2b21 0%, #304a3d 48%, #486d5a 100%);
  color: #f7f6f0;
}

.hero-copy {
  position: relative;
  z-index: 1;
  max-width: 760px;
}

.eyebrow,
.section-kicker {
  margin: 0 0 8px;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  font-size: 12px;
  opacity: 0.75;
}

h1 {
  margin: 0;
  font-size: clamp(42px, 7vw, 76px);
  line-height: 0.95;
}

.hero-text {
  max-width: 640px;
  margin: 18px 0;
  font-size: 18px;
  color: rgba(247, 246, 240, 0.88);
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  font-size: 13px;
  color: rgba(247, 246, 240, 0.75);
}

.hero-blobs {
  position: absolute;
  inset: 0;
}

.blob {
  position: absolute;
  border-radius: 999px;
  filter: blur(10px);
}

.blob-one {
  width: 200px;
  height: 200px;
  right: -18px;
  top: -30px;
  background: rgba(216, 110, 63, 0.32);
}

.blob-two {
  width: 180px;
  height: 180px;
  right: 120px;
  bottom: -72px;
  background: rgba(201, 137, 40, 0.26);
}

.control-card,
.detect-card,
.metric-card {
  height: 100%;
  padding: 22px;
  border: 1px solid rgba(48, 74, 61, 0.08);
  background: rgba(255, 253, 247, 0.9);
  backdrop-filter: blur(12px);
}

.card-title {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 18px;
}

.card-title h2,
.metric-card h3 {
  margin: 0;
}

.action-row,
.field-grid,
.switches,
.metric-controls,
.metric-heading,
.endpoint-row {
  display: flex;
  gap: 12px;
}

.action-row,
.switches {
  flex-wrap: wrap;
}

.field-grid {
  flex-direction: column;
  margin-top: 14px;
}

.endpoint-row {
  width: 100%;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
}

.endpoint-name {
  overflow: hidden;
  text-overflow: ellipsis;
}

.endpoint-body {
  overflow: auto;
  max-height: 320px;
  padding: 14px;
  border-radius: 16px;
  background: #172118;
  color: #f5f6f0;
  white-space: pre-wrap;
}

.metrics-section {
  margin-top: 22px;
}

.chart-title {
  margin-bottom: 12px;
  align-items: end;
}

.time-range-select {
  min-width: 170px;
  max-width: 190px;
}

.metric-toggle-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 10px;
}

.metric-toggle-chip {
  cursor: pointer;
}

.hidden-metric-hint {
  margin: 0 0 14px;
  color: #56665d;
  font-size: 14px;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.metric-heading {
  align-items: center;
}

.metric-controls {
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
}

.metric-summary,
.subtle-copy,
.error-copy,
.profile-meta {
  margin: 8px 0 0;
  color: #56665d;
}

.chart-shell {
  height: 380px;
}

.empty-chart {
  display: grid;
  place-items: center;
  min-height: 240px;
  border-radius: 18px;
  background: repeating-linear-gradient(
    -45deg,
    rgba(48, 74, 61, 0.04),
    rgba(48, 74, 61, 0.04) 12px,
    rgba(216, 110, 63, 0.03) 12px,
    rgba(216, 110, 63, 0.03) 24px
  );
  text-align: center;
  color: #56665d;
  padding: 20px;
}

@media (max-width: 960px) {
  .shell {
    padding: 16px;
  }

  .hero {
    padding: 22px;
  }

  .metric-header {
    flex-direction: column;
  }

  .metric-controls {
    justify-content: flex-start;
  }
}
</style>
