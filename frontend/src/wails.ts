import {
  AppInfo,
  AvailableMetrics,
  DetectURL,
  DownloadProfile,
  ExportProfile,
  FetchProfileCatalog,
  FetchProfileText,
  FetchMetrics,
  GetProfileFlamegraph,
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
  fetchProfileCatalog(input: string) {
    return FetchProfileCatalog(input)
  },
  fetchProfileText(input: string, profile: string, debug: number, seconds: number) {
    return FetchProfileText(input, profile, debug, seconds)
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
  downloadProfile(input: string, profile: string, debug: number, seconds: number) {
    return DownloadProfile(input, profile, debug, seconds)
  },
  getProfileFlamegraph(input: string, profile: string, seconds: number) {
    return GetProfileFlamegraph(input, profile, seconds)
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
