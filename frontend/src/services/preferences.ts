import type { Preferences } from '../types'

const key = 'live-pprof-wails-preferences-v1'

export function defaultPreferences(initialURL: string): Preferences {
  return {
    endpointInput: initialURL,
    sampleInterval: 1000,
    retainedSamples: 120,
    cpuProfileSeconds: 1,
    smooth: false,
    useMock: false,
    timeRangeMinutes: 30,
    metrics: {
      cpu: { enabled: true, total: false, flatOrCum: 'flat', topN: 10 },
      heap: { enabled: true, total: false, flatOrCum: 'flat', topN: 10 },
      allocs: { enabled: true, total: false, flatOrCum: 'flat', topN: 10 },
      goroutine: { enabled: true, total: false, flatOrCum: 'flat', topN: 10 }
    }
  }
}

export function loadPreferences(initialURL: string): Preferences {
  const defaults = defaultPreferences(initialURL)
  if (typeof window === 'undefined') return defaults

  try {
    const raw = localStorage.getItem(key)
    if (!raw) return defaults
    const parsed = JSON.parse(raw) as Partial<Preferences>
    return {
      ...defaults,
      ...parsed,
      metrics: {
        ...defaults.metrics,
        ...(parsed.metrics ?? {})
      }
    }
  } catch {
    return defaults
  }
}

export function savePreferences(value: Preferences): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(key, JSON.stringify(value))
}
