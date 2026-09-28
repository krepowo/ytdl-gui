/**
 * Typed wrapper around the Wails bindings.
 *
 * Feature code imports these functions instead of reaching into
 * `window.go.main.App` directly, so:
 *   - the call surface is typed and centralised, and
 *   - tests can install a fake `window.go` without touching feature code.
 *
 * In a real Wails build the generated bindings resolve at import time; in tests
 * `window.go` is provided by `installWailsGoMock`.
 */
import {
  CancelJob,
  GetAppInfo,
  GetDiskFree,
  GetSettings,
  ListJobs,
  OpenDownloadDir,
  OpenFile,
  PauseJob,
  PickDownloadDir,
  ProbeURL,
  RemoveJob,
  ResumeJob,
  SaveSettings,
  StartDownload,
} from '../../wailsjs/go/main/App'
import type { AppInfo, DownloadRequest, Job, MediaInfo, Settings } from './types'

export const api = {
  getSettings: (): Promise<Settings> => GetSettings() as Promise<Settings>,
  saveSettings: (s: Settings): Promise<void> => SaveSettings(s as never),
  pickDownloadDir: (): Promise<string> => PickDownloadDir(),
  probeURL: (url: string): Promise<MediaInfo> => ProbeURL(url) as Promise<MediaInfo>,
  startDownload: (req: DownloadRequest): Promise<string> => StartDownload(req as never),
  pauseJob: (id: string): Promise<void> => PauseJob(id),
  resumeJob: (id: string): Promise<void> => ResumeJob(id),
  cancelJob: (id: string): Promise<void> => CancelJob(id),
  removeJob: (id: string): Promise<void> => RemoveJob(id),
  // The generated bindings type `state` as a plain string; the app narrows it to
  // the JobState union at this boundary so feature code gets exhaustiveness.
  listJobs: (): Promise<Job[]> => ListJobs() as unknown as Promise<Job[]>,
  openDownloadDir: (): Promise<void> => OpenDownloadDir(),
  openFile: (path: string): Promise<void> => OpenFile(path),
  getAppInfo: (): Promise<AppInfo> => GetAppInfo() as Promise<AppInfo>,
  getDiskFree: (dir: string): Promise<number> => GetDiskFree(dir),
}

/**
 * Turns an unknown thrown value into a readable message. Wails rejects with a
 * string, so `String(err)` is usually already correct.
 */
export function errorMessage(err: unknown): string {
  if (typeof err === 'string') return err
  if (err instanceof Error) return err.message
  return 'Terjadi kesalahan yang tidak diketahui.'
}
