import { vi } from 'vitest'

/**
 * Installs a fake `window.go.main.App` so components can be tested outside a
 * Wails webview. The generated bindings call `window['go']['main']['App'][name]`
 * at call time, so replacing the object is enough.
 *
 * Returns the map of spies so tests can assert calls and stub return values.
 */
export function installWailsGoMock(overrides: Record<string, unknown> = {}) {
  const methods: Record<string, ReturnType<typeof vi.fn>> = {
    GetSettings: vi.fn(async () => ({
      downloadDir: 'D:\\Downloads',
      maxConcurrent: 2,
      defaultMode: 'video',
      defaultQuality: 'best',
      filenameTemplate: '%(title)s.%(ext)s',
      cookiesBrowser: '',
    })),
    SaveSettings: vi.fn(async () => undefined),
    PickDownloadDir: vi.fn(async () => 'D:\\Downloads'),
    ProbeURL: vi.fn(),
    StartDownload: vi.fn(async () => 'job-1'),
    PauseJob: vi.fn(async () => undefined),
    ResumeJob: vi.fn(async () => undefined),
    CancelJob: vi.fn(async () => undefined),
    RemoveJob: vi.fn(async () => undefined),
    ListJobs: vi.fn(async () => []),
    OpenDownloadDir: vi.fn(async () => undefined),
    OpenFile: vi.fn(async () => undefined),
    GetAppInfo: vi.fn(async () => ({
      version: '0.1.0',
      configPath: 'C:\\app\\config.json',
      ytDlpVersion: '2026.08.19',
      ffmpegVersion: '9.0',
    })),
    GetDiskFree: vi.fn(async () => 123456789),
  }

  for (const [name, fn] of Object.entries(overrides)) {
    if (fn && typeof fn === 'object' && 'mock' in (fn as object)) {
      methods[name] = fn as ReturnType<typeof vi.fn>
    } else {
      methods[name].mockResolvedValue(fn)
    }
  }

  const go = { main: { App: methods } }
  ;(window as unknown as { go: unknown }).go = go
  return methods
}
