# Capability Map: Video Downloader (yt-dlp GUI)

A desktop GUI for **yt-dlp**, built with **Wails v2 (Go backend)** and a
**React 19 + Material UI v9 + Tailwind CSS v4** frontend. Target platform:
**Windows only**. Ships as an **NSIS installer**. `yt-dlp.exe` (+ `ffmpeg.exe` /
`ffprobe.exe`) are **bundled inside the app** so it runs with zero external
installs.

**Multi-source by design:** the app targets *every site yt-dlp supports*
(1000+ extractors + the generic fallback), not just YouTube. Nothing in the
engine or UI may assume a specific provider.

## Modules

| Module id | Responsibility | Depends on |
|---|---|---|
| `settings` | Persisted app config (stored **next to the installed exe**): download dir, max concurrency, default quality/format, filename template, cookies preference. | — |
| `ytdlp-engine` | Wrap the bundled `yt-dlp.exe`: probe metadata & formats for **any supported site**, spawn a download, parse live progress, pause/resume/cancel. Uses bundled `ffmpeg` for merging/conversion. | `settings` |
| `download-queue` | Job queue + concurrency scheduler + per-job state machine + persisted history. | `ytdlp-engine`, `settings` |
| `app-api` | Wails-bound method surface and event bridge — the single Go↔React contract. | `ytdlp-engine`, `download-queue`, `settings` |
| `ui-shell` | Frameless window chrome (custom dark title bar), theme tokens shared by MUI v9 + Tailwind v4, icon-first layout. | `app-api` |
| `download-screen` | Main screen: URL input, dynamic format picker, queue list with per-item controls & progress. | `app-api`, `ui-shell` |
| `settings-screen` | Settings screen: download dir, concurrency, default format, cookies. | `app-api`, `ui-shell` |
| `packaging` | Bundle binaries as Wails assets; NSIS installer config (per-user scope); version stamping. | all |

## Build order

```
settings → ytdlp-engine → download-queue → app-api → ui-shell → download-screen → settings-screen → packaging
```

## Dependency rules

- `app-api` is the **only** module the frontend talks to; UI never shells out or
  reads Go internals directly.
- `ytdlp-engine` is the **only** module that knows yt-dlp's CLI surface. A future
  engine swap touches only this module.
- `packaging` depends on all modules only at build time (it wires assets + the
  NSIS script); it holds no runtime logic.
- No cycles: arrows point one way only.
