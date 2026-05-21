import {
  AppInfo,
  AvailableMetrics,
  DetectURL,
  ExportProfile,
  FetchMetrics,
  ImportProfile,
  ImportProfiles,
  InitialURL,
  NormalizeURL,
  ProfileMeta
} from '../wailsjs/go/main/App'

export const wailsApi = {
  detectURL(input: string) {
    return DetectURL(input)
  },
  fetchMetrics(input: string, metric: string, profileSeconds: number, useMock: boolean) {
    return FetchMetrics(input, metric, profileSeconds, useMock)
  },
  importProfile(metric: string) {
    return ImportProfile(metric)
  },
  importProfiles(metric: string) {
    return ImportProfiles(metric)
  },
  exportProfile(metric: string) {
    return ExportProfile(metric)
  },
  profileMeta(metric: string) {
    return ProfileMeta(metric)
  },
  normalizeURL(input: string) {
    return NormalizeURL(input)
  },
  initialURL() {
    return InitialURL()
  },
  availableMetrics() {
    return AvailableMetrics()
  },
  appInfo() {
    return AppInfo()
  }
}
