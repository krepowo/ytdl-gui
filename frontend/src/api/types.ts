/**
 * Frontend-facing types.
 *
 * These mirror the Go structs exposed through the Wails bindings (see
 * `wailsjs/go/models.ts`). They are declared here so feature code imports from a
 * stable local module instead of the generated files, and so tests can build
 * plain objects without the generated classes.
 */

/** A download lifecycle state. Values match the Go `queue.State` constants. */
export type JobState =
  | 'queued'
  | 'downloading'
  | 'paused'
  | 'completed'
  | 'error'
  | 'canceled'

/** One selectable format, derived from what the site actually returned. */
export type FormatOption = {
  formatId: string
  label: string
  ext: string
  height: number
  width: number
  fps: number
  bitrate: number
  filesize: number
  /** A video-only stream that the engine must merge with best audio. */
  needsMerge: boolean
}

/** Metadata + selectable formats for a probed URL. Every field is optional-safe. */
export type MediaInfo = {
  id: string
  title: string
  uploader: string
  /** Seconds; 0 when unknown (live streams, generic pages). */
  duration: number
  thumbnail: string
  /** Detected source, e.g. "youtube", "twitter", "generic". */
  extractor: string
  webpageUrl: string
  isLive: boolean
  videoOptions: FormatOption[]
  audioOptions: FormatOption[]
}

/** A queued/running/finished download. Mirrors `queue.Job`. */
export type Job = {
  id: string
  url: string
  title: string
  mode: string
  formatId: string
  state: JobState
  percent: number
  downloadedBytes: number
  totalBytes: number
  speedBps: number
  etaSec: number
  outputPath: string
  error: string
  createdAt: number
}

/** Persisted configuration. Mirrors `settings.Settings`. */
export type Settings = {
  downloadDir: string
  maxConcurrent: number
  defaultMode: string
  defaultQuality: string
  filenameTemplate: string
  cookiesBrowser: string
}

/** App/build info. Mirrors `app.AppInfo`. */
export type AppInfo = {
  version: string
  configPath: string
  ytDlpVersion: string
  ffmpegVersion: string
}

/** Request to enqueue a download. Mirrors `app.DownloadRequest`. */
export type DownloadRequest = {
  url: string
  mode: string
  formatId: string
  needsMerge: boolean
  title: string
}

/** Download modes. */
export const MODE_VIDEO = 'video'
export const MODE_AUDIO = 'audio'
