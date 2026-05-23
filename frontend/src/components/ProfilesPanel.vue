<script setup lang="ts">
import { ref } from 'vue'
import FlamegraphView from './FlamegraphView.vue'
import type { FlamegraphNode, ProfileCatalogEntry } from '@/types'
import colors from 'vuetify/util/colors'


const props = defineProps<{
  entries: ProfileCatalogEntry[]
  endpoint: string
  loading: boolean
  rawText: string
  flamegraph: FlamegraphNode | null
}>()

const emit = defineEmits<{
  refresh: []
  importProfile: [profile: string]
  downloadProfile: [profile: string, debug: number]
  openText: [profile: string, debug: number]
  openFlame: [profile: string]
}>()

const expanded = ref<string | null>(null)

function toggle(name: string) {
  expanded.value = expanded.value === name ? null : name
}
</script>

<template>
  <v-card class="profiles-card" elevation="0">
    <div class="profiles-header">
      <div>
        <p class="section-kicker">Profiles</p>
        <h2>Browser Replacement</h2>
        <p class="profiles-copy">
          Mirrors the information you would normally inspect under <span class="mono">/debug/pprof/</span>.
        </p>
      </div>
      <v-btn
        color="primary"
        variant="flat"
        prepend-icon="mdi-refresh"
        :loading="loading"
        :disabled="!endpoint.trim()"
        @click="emit('refresh')"
      >
        Refresh Profiles
      </v-btn>
    </div>

    <div class="profiles-grid">
      <div class="profiles-table-head mono">
        <span>Profile</span>
        <span>Count</span>
        <span>Description</span>
        <span>Actions</span>
      </div>

      <div v-for="entry in entries" :key="entry.name" class="profiles-row">
        <div class="profiles-main">
          <button class="profile-name" type="button" @click="toggle(entry.name)">
            <span>{{ entry.name }}</span>
            <v-icon size="18">{{ expanded === entry.name ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
          </button>
          <span class="mono">{{ entry.count }}</span>
          <span class="description">{{ entry.description || 'No description available.' }}</span>
          <div class="action-buttons">
            <v-fab
              v-if="entry.supportsImport || entry.supportsExport || entry.supportsRawText || entry.supportsFlame"
              size="small"
              icon
              variant="flat"
              color="#f6fff7"
            >
            <v-icon>mdi-dots-vertical</v-icon>

            <v-speed-dial
              location="bottom center"
              transition="slide-y-transition"
              activator="parent"
            >
              <v-btn
                  v-if="entry.supportsRawText"
                  key="text"
                  class="speed-dial-btn"
                  variant="flat"
                  color="#1e88e5"
                  prepend-icon="mdi-text-box-search-outline"
                  @click="emit('openText', entry.name, entry.name === 'goroutine' ? 2 : 1)"
              >
                Text
              </v-btn>

              <v-btn
                  v-if="entry.supportsFlame"
                  key="flame"
                  class="speed-dial-btn"
                  variant="flat"
                  color="#ff8a65"
                  prepend-icon="mdi-fire"
                  @click="emit('openFlame', entry.name)"
              >
                Flame
              </v-btn>
              <v-btn
                  v-if="entry.supportsExport"
                  key="export"
                  class="speed-dial-btn"
                  variant="flat"
                  color="#273d31"
                  prepend-icon="mdi-download"
                  @click="emit('downloadProfile', entry.name, entry.name === 'goroutine' ? 2 : 1)"
              >
                Download
              </v-btn>
              <v-btn
                v-if="entry.supportsImport"
                key="import"
                class="speed-dial-btn"
                variant="flat"
                color="#26a69a"
                prepend-icon="mdi-file-import-outline"
                @click="emit('importProfile', entry.name)"
              >
                Import
              </v-btn>

            </v-speed-dial>
          </v-fab>
          </div>
        </div>

        <div v-if="expanded === entry.name" class="profiles-detail">
          <div class="detail-chip-row">
            <v-chip size="small" variant="outlined">chart: {{ entry.supportsChart ? 'yes' : 'no' }}</v-chip>
            <v-chip size="small" variant="outlined">raw text: {{ entry.supportsRawText ? 'yes' : 'no' }}</v-chip>
            <v-chip size="small" variant="outlined">flame: {{ entry.supportsFlame ? 'yes' : 'no' }}</v-chip>
          </div>
        </div>
      </div>
    </div>

    <div v-if="rawText" class="raw-view">
      <div class="raw-view-header">Profile Text View</div>
      <pre class="raw-view-content">{{ rawText }}</pre>
    </div>

    <div v-if="flamegraph" class="flame-view">
      <div class="raw-view-header">Flamegraph View</div>
      <FlamegraphView :root="flamegraph" />
    </div>
  </v-card>
</template>

<style scoped>
.profiles-card {
  padding: 22px;
  border: 1px solid rgba(48, 74, 61, 0.08);
  background: rgba(255, 253, 247, 0.9);
  backdrop-filter: blur(12px);
}

.profiles-header {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.profiles-header h2 {
  margin: 0;
}

.profiles-copy {
  margin: 8px 0 0;
  color: #56665d;
}

.profiles-grid {
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  overflow: hidden;
}

.profiles-table-head,
.profiles-main {
  display: grid;
  grid-template-columns: 160px 80px minmax(0, 1fr) 360px;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
}

.profiles-table-head {
  background: #eef0e7;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.profiles-row {
  border-top: 1px solid rgba(48, 74, 61, 0.08);
}

.profile-name {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  cursor: pointer;
  font: inherit;
  padding: 0;
  color: #142018;
}

.description {
  color: #56665d;
}

.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
  justify-self: end;
}

.profiles-detail {
  padding: 0 14px 14px;
}

.detail-chip-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.raw-view {
  margin-top: 18px;
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  overflow: hidden;
}

.raw-view-header {
  padding: 10px 14px;
  background: #eef0e7;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.raw-view-content {
  margin: 0;
  padding: 14px;
  max-height: 360px;
  overflow: auto;
  background: rgba(48, 74, 61, 0.03);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.flame-view {
  margin-top: 18px;
  border: 1px solid rgba(48, 74, 61, 0.08);
  border-radius: 16px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.7);
}

@media (max-width: 1280px) {
  .profiles-table-head,
  .profiles-main {
    grid-template-columns: 150px 64px minmax(0, 1fr);
  }

  .action-buttons {
    grid-column: 1 / -1;
    justify-content: flex-end;
  }
}

.speed-dial-btn {
  min-width: 140px !important;
  justify-content: flex-start !important;
}
</style>
