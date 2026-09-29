# Video Downloader

A Windows desktop GUI for [yt-dlp](https://github.com/yt-dlp/yt-dlp): paste a link,
pick a quality, download. Built with **Wails v2** (Go backend) and **React 19 +
MUI v9 + Tailwind v4** (TypeScript frontend).

It is not limited to YouTube — anything yt-dlp supports works, and the quality
list is built from what each site actually offers (no fixed 480/720/1080 ladder).

## Features

- **Any yt-dlp source** — YouTube, X/Twitter, Vimeo, Twitch, direct `.mp4` links,
  and everything else yt-dlp handles. A source badge shows the detected
  extractor, with a generic fallback when a site has no dedicated one.
- **Dynamic format list** — video options come from the site's real formats
  (e.g. `2160p60`, `1080p60`, `144p`), not a hard-coded resolution list.
- **Audio / MP3** — pick an audio-only stream, or choose **MP3** and the bundled
  ffmpeg converts it.
- **Parallel download queue** — several downloads at once (configurable 1–8),
  with per-item **pause**, **resume**, and **cancel**. Pausing truly freezes the
  process tree; cancelling leaves no orphaned processes.
- **Merged video + audio** — a video-only stream is automatically paired with the
  best audio and merged to `.mp4` by ffmpeg.
- **Safe filenames** — characters Windows forbids are rewritten to `_`, and a
  second download of the same media gets a `+1` suffix instead of being skipped.
- **Cookie support** — optionally read cookies from Chrome / Edge / Firefox /
  Brave / Opera / Vivaldi for sites that require a login.
- **Session history** — the queue is restored on restart; jobs that were
  interrupted come back as *paused*.
- **Portable config** — `config.json` lives in the install folder, falling back to
  `%APPDATA%` if that folder is read-only.
- **Icon-first UI** — dark, frameless window; every icon action has a tooltip and
  an accessible label.

## Requirements

- **Windows 10/11** (this app is Windows-only).
- To build from source: **Go 1.25+**, **Node.js 20+**, and the
  [Wails CLI](https://wails.io/docs/gettingstarted/installation) (`v2.16.0`).
- To build the installer: **NSIS** (`makensis` on `PATH`).

## Getting started

```bash
git clone https://github.com/krepowo/ytdl-gui.git
cd ytdl-gui

# 1. Fetch the bundled binaries into resources/ (~220 MB, not committed).
#    Git Bash / WSL:
./scripts/fetch-binaries.sh

# 2. Build and run.
wails build -clean
powershell -ExecutionPolicy Bypass -File scripts\run.ps1 -NoBuild
```

`scripts/fetch-binaries.sh` downloads the standalone **onefile** `yt-dlp.exe`
plus `ffmpeg.exe` and `ffprobe.exe`. The onefile build matters: the winget /
Program-Files copy is a PyInstaller *onedir* install that fails when moved
without its `_internal\` folder.

The app looks for the three binaries **next to the executable**. A plain
`wails build` only emits `VideoDownloader.exe`, so `scripts/run.ps1` stages the
binaries from `resources/` into `build/bin/` first. Without them the UI still
runs, but every probe/download reports a clear "binary not found" error.

### Building the installer

A single all-inclusive, LZMA-compressed NSIS installer (bundling yt-dlp, ffmpeg
and ffprobe) is produced per-user, so it needs no administrator rights:

```bash
wails build -nsis -clean -platform windows/amd64 -installscope user
```

The installer is written to `build/bin/`. Uninstalling removes `config.json` and
`history.json`; downloaded media is never touched.

## Testing

See [docs/TESTING.md](docs/TESTING.md) for the full walkthrough. In short:

```bash
go test ./...                 # Go: engine, queue, settings, app service
cd frontend && npm test       # Frontend: components, hooks, formatting

# Real-network end-to-end tests (build-tagged so the default suite stays offline):
go test -tags manual -run TestE2EProbeThenDownload -v -timeout 6m .
```

## Project layout

```
main.go, app.go, engine.go      Wails entry point, service wiring, engine adapter
internal/settings/              config load/save (install dir + %APPDATA% fallback)
internal/ytdlp/                 binary resolution, probe, progress parsing, download runner
internal/queue/                 job state machine, concurrency scheduler, history
internal/app/                   bound API surface + event bridge
frontend/src/features/download/ URL input, media summary, format picker, queue list
frontend/src/features/settings/ settings dialog
frontend/src/components/        title bar, layout
docs/                           capability map, module specs, plan, todo, testing guide
```

The project is spec-driven: `docs/CAPABILITY-MAP.md` maps the modules, each
`docs/SPEC-*.md` defines one of them, and `docs/plan.md` / `docs/todo.md` track
the work.

## Architecture notes

- **Pause/resume** suspends every process in a Windows **Job Object**, not just
  the direct child. The onefile `yt-dlp.exe` is a PyInstaller bootloader that
  re-executes as a `python.exe` child, so suspending only the parent would let
  the download keep running.
- **Cancellation** relies on `KILL_ON_JOB_CLOSE`, which tears down the whole
  process tree (yt-dlp plus any ffmpeg child) with no orphans.
- **Merged downloads**: the final output path is taken from the
  `[Merger] Merging formats into "..."` line, because the last
  `[download] Destination:` line names an intermediate file that is deleted.
- The frontend is **event-driven** (`job:added`, `job:updated`, `job:removed`,
  `app:error`) with no polling.

## Third-party components

This repository does **not** bundle the binaries; `scripts/fetch-binaries.sh`
downloads them. Distributions that package them are responsible for the
respective licenses:

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) — Unlicense
- [FFmpeg](https://ffmpeg.org/) — LGPL/GPL depending on the build

## License

[MIT](LICENSE) © 2026 krepowo
