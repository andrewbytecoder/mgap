export function formatBytes(value: number): string {
  if (!Number.isFinite(value)) return ''
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let current = Math.abs(value)
  let index = 0
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024
    index += 1
  }
  const output = current >= 100 || index === 0 ? current.toFixed(0) : current.toFixed(1)
  return `${output} ${units[index]}`
}

export function formatDuration(value: number): string {
  if (!Number.isFinite(value)) return ''
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)} s`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)} ms`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)} us`
  return `${value.toFixed(0)} ns`
}

export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return ''
  if (Math.abs(value) >= 100) return `${value.toFixed(0)}%`
  if (Math.abs(value) >= 10) return `${value.toFixed(1)}%`
  return `${value.toFixed(2)}%`
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value)
}

export function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp / 1_000_000)
  return date.toLocaleTimeString()
}
