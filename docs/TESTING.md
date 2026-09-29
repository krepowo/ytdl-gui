# How to test the app

There are three levels, from quickest to most thorough.

## 1. Run the app and use it (manual / exploratory)

The app looks for `yt-dlp.exe`, `ffmpeg.exe`, `ffprobe.exe` **next to the
executable**. The Wails build only emits `VideoDownloader.exe`, so use the run
script, which stages the three binaries from `resources/` first:

```powershell
# from the repo root, in PowerShell
powershell -ExecutionPolicy Bypass -File scripts\run.ps1
```

Options:

| Command | What it does |
|---|---|
| `scripts\run.ps1` | Build, stage binaries, launch |
| `scripts\run.ps1 -NoBuild` | Stage + launch the existing build (fast) |
| `scripts\run.ps1 -Dev` | `wails dev` with hot reload (frontend edits live-update) |

> If `resources/` is empty, run `scripts/fetch-binaries.sh` first (it downloads
> the standalone **onefile** yt-dlp — not the winget/Program-Files onedir copy —
> plus ffmpeg/ffprobe).

**What to try:**

1. Paste any video URL (YouTube, X/Twitter, Vimeo, a direct `.mp4` link…) into
   the field and press the **Download** button. The app probes it first, then shows
   the title, a source badge, and a format picker.
2. Pick a quality (or switch to **Audio/MP3**) and start the download. The item
   appears in the **Download Queue** and its progress bar advances live.
3. Exercise the per-item icons: **pause** → **resume**, **cancel**, and on
   completion **open folder** (folder icon).
4. Open **Settings** (gear, bottom-right) to change the download folder,
   concurrency (1–8), default mode, and the filename template. Settings save
   automatically; the config-file path is shown at the bottom.
5. Confirm the config file location: it lives in the install folder, shown in
   **Settings** as "Config file". If that folder is read-only it falls
   back to `%APPDATA%`.

## 2. Automated tests (fast, no network)

```bash
# Go: engine, queue, settings, app service
go test ./...

# Frontend: components, hooks, formatting
cd frontend
npm run typecheck
npm run lint
npm test
```

## 3. Real-network / end-to-end tests (slower, hits the internet)

These use the real bundled binaries and are gated behind a build tag so the
normal suite stays offline:

```bash
# Probe + download through the exact UI code path, asserting the event stream
go test -tags manual -run TestE2EProbeThenDownload -v -timeout 6m .

# Real MP3 + merged video + cancel-leaves-no-orphans + pause truly freezes
go test -tags manual ./internal/ytdlp/ -v -timeout 10m

# Version parsing against the real binaries
go test ./internal/ytdlp/ -run TestResolveBinsRealBinaries -v
```

## Known environment limits

- `go test -race` is unavailable on this machine (no C compiler). Concurrency is
  covered by deterministic tests instead.
- Some sites (e.g. Vimeo) may fail behind a restrictive local network/SSL
  proxy — that is the environment, not the app.
- `wails build` needs a lowercase `tmp` env var **unset** in Git Bash; the run
  script handles this for you.
