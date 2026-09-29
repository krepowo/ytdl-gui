# Task List: Video Downloader (yt-dlp GUI)

Legend: **S** = 1–2 files, **M** = 3–5 files, **L** = 5–8 files.
Full context in `plan.md`; module contracts in `SPEC-*.md`.

---

## Phase 1 — Foundation

### Task 1: Scaffold the Wails project + frontend toolchain  [M]
**Description:** Upgrade the Wails CLI to the **latest v2 (≥ v2.13.0; 2.16.0
recommended)** — required later for `-installscope`. Then create the Wails v2 app
(`ytdl-gui`) and install/wire the frontend stack: React 19, MUI v9, Tailwind v4
via `@tailwindcss/vite`, Vite 8, **TypeScript 5.x**, Vitest. Configure
`wails.json`, `vite.config.ts`, npm scripts.

**Acceptance criteria:**
- [ ] Wails CLI + module upgraded to **≥ v2.13.0**; `wails build --help` shows `-installscope`
- [ ] `wails dev` launches a window rendering the React app
- [ ] Tailwind v4 works via `@import "tailwindcss"` + `@tailwindcss/vite` (no `tailwind.config.js`)
- [ ] TypeScript pinned to **5.x**
- [ ] `npm run typecheck`, `npm run lint`, `npm run test` scripts exist and run

**Verification:**
- [ ] `wails build --help | grep installscope` (flag present)
- [ ] Manual: `wails dev` shows the app
- [ ] `npm run typecheck` exits 0
- [ ] `go build ./...` exits 0

**Dependencies:** None
**Files:** `wails.json`, `go.mod`, `frontend/vite.config.ts`, `frontend/package.json`,
`frontend/src/main.tsx`, `frontend/src/index.css`, `main.go`, `app.go`

---

### Task 2: `settings` module  [S]
**Description:** Implement `Settings` struct, defaults, `Load`/`Save`/`Validate`,
and `ConfigPath()` resolving `<exe-dir>/config.json` with a read-only
`%APPDATA%` fallback. Atomic writes.

**Acceptance criteria:**
- [ ] First run returns documented defaults
- [ ] Save→Load round-trips; corrupt JSON falls back to defaults (no panic)
- [ ] `MaxConcurrent` clamped to 1–8
- [ ] Read-only exe dir transparently uses `%APPDATA%` fallback

**Verification:**
- [ ] `go test ./internal/settings/...` passes
- [ ] Manual: run app once, confirm `config.json` appears next to the exe

**Dependencies:** Task 1
**Files:** `internal/settings/settings.go`, `internal/settings/settings_test.go`

---

### Task 3: `ui-shell` — tokens, theme, TitleBar, Layout  [M]
**Description:** Define `tokens.ts` as the single source of truth; build the MUI
v9 theme from it (CSS variables mode); mirror tokens into Tailwind's `@theme`
block in `index.css`. Build the frameless custom dark TitleBar (drag/min/close)
and Layout.

**Acceptance criteria:**
- [ ] Frameless window; title bar drag + minimize + close work
- [ ] No hex values outside `tokens.ts`/`index.css`
- [ ] Tailwind utilities and MUI components coexist (no reset conflict)
- [ ] Icon-only controls have `aria-label`/Tooltip

**Verification:**
- [ ] Manual: `wails dev` — colors/radius match the reference image
- [ ] `npm run test` (TitleBar snapshot) and `npm run typecheck` pass

**Dependencies:** Task 1
**Files:** `frontend/src/theme/tokens.ts`, `frontend/src/theme/muiTheme.ts`,
`frontend/src/components/TitleBar.tsx`, `frontend/src/components/Layout.tsx`,
`frontend/src/index.css`, `frontend/src/App.tsx`

#### Checkpoint: Foundation
- [ ] `wails dev` opens the frameless dark window; title bar works
- [ ] `go test ./...` and `npm run typecheck` pass
- [ ] Review with human before proceeding

---

## Phase 2 — Engine

### Task 4: Bundled binary resolution + version checks  [S]
**Description:** Resolve `yt-dlp.exe`, `ffmpeg.exe`, `ffprobe.exe` relative to the
executable; expose `Bins` and version strings. Fail with a clear error if a
binary is missing. (All three are always installed — a missing binary means a
corrupted install, not a supported configuration.)

**Acceptance criteria:**
- [ ] Paths resolved from exe dir (not CWD)
- [ ] `yt-dlp --version` and `ffmpeg -version` parsed and returned
- [ ] Missing binary → descriptive error surfaced to the UI, not a panic

**Verification:** `go test ./internal/ytdlp/...`; manual `GetAppInfo` shows versions
**Dependencies:** Task 2
**Files:** `internal/ytdlp/binary.go`, `internal/ytdlp/ytdlp_test.go`, `resources/`

---

### Task 5: `Probe` — site-agnostic metadata + dynamic formats  [M]
**Description:** Run `yt-dlp -J --no-warnings --no-playlist <url>` (plus
`--cookies-from-browser` when configured), unmarshal into `MediaInfo`, and build
the `FormatOption` list from what the site actually returns. Read
`extractor`/`extractor_key` for the source badge. Handle missing
duration/thumbnail and live streams.

**Acceptance criteria:**
- [ ] Works for a YouTube URL, a Twitter/X URL, and a generic page
- [ ] Formats derived dynamically; no hardcoded quality ladder
- [ ] Missing optional metadata doesn't error
- [ ] Cookies flag passed when `settings.CookiesBrowser` is set

**Verification:** `go test` against fake yt-dlp JSON fixtures; manual probe of 3 sites
**Dependencies:** Task 4
**Files:** `internal/ytdlp/probe.go`, `internal/ytdlp/probe_test.go`

---

### Task 6: Progress-line parser  [S]
**Description:** Parse yt-dlp `--newline` output lines into
`{percent, downloadedBytes, totalBytes, speedBps, etaSec}`, handling B/KiB/MiB/GiB,
speeds, ETA, and the 100% line.

**Acceptance criteria:**
- [ ] All unit cases pass (units, speeds, ETA, completion)
- [ ] Unknown lines are ignored, not fatal

**Verification:** `go test ./internal/ytdlp/... -run Progress`
**Dependencies:** Task 4
**Files:** `internal/ytdlp/progress.go`, `internal/ytdlp/progress_test.go`

---

### Task 7: Download runner — spawn, stream, pause/resume/cancel  [L]
**Description:** Spawn yt-dlp with the right args per mode (video merge vs. audio
`-x --audio-format mp3`), pass `--ffmpeg-location`, stream parsed progress via a
callback (throttled ≤4/s), and implement pause/resume/cancel. Kill the **whole
process tree** (Windows Job Object) on cancel/quit.

**Acceptance criteria:**
- [ ] Progress callbacks fire during a real download
- [ ] Audio job yields a valid `.mp3`; separate-stream site merges to one file
- [ ] Cancel kills yt-dlp + ffmpeg within 2s; no orphans
- [ ] Pause halts updates; resume continues from the same offset (suspend, else kill+`--continue`)

**Verification:** `go test` with a fake binary; manual download of an MP3 + a merged video
**Dependencies:** Task 5, Task 6
**Files:** `internal/ytdlp/download.go`, `internal/ytdlp/download_test.go`

#### Checkpoint: Engine
- [ ] Probe works for 3 different sites
- [ ] MP3 + merged download both succeed
- [ ] Cancel leaves no orphan processes
- [ ] Review with human before proceeding

---

## Phase 3 — Queue & API

### Task 8: `download-queue` — state machine + scheduler  [M]
**Description:** Job model with a validated state machine
(queued/downloading/paused/completed/error/canceled), a worker pool honoring
`MaxConcurrent`, and slot release on every terminal state. Engine injected via an
interface.

**Acceptance criteria:**
- [ ] 5 jobs @ concurrency 2 never exceed 2 running
- [ ] Finishing a job promotes the next queued job
- [ ] Illegal transitions rejected

**Verification:** `go test ./internal/queue/...` (deterministic concurrency tests — see plan risk note on `-race`/no gcc)
**Dependencies:** Task 7
**Files:** `internal/queue/queue.go`, `internal/queue/job.go`, `internal/queue/queue_test.go`

---

### Task 9: History persistence  [S]
**Description:** Persist completed/failed history to
`<exe-dir>/history.json`; restore on startup. Interrupted jobs reload as `paused`
(never auto-started).

**Acceptance criteria:**
- [ ] History round-trips across restart
- [ ] Interrupted jobs load as `paused`

**Verification:** `go test ./internal/queue/... -run History`
**Dependencies:** Task 8
**Files:** `internal/queue/history.go`, `internal/queue/history_test.go`

---

### Task 10: `app-api` — bound methods + event bridge  [M]
**Description:** Implement all bound methods (settings, probe, start/control/remove
job, list jobs, open dir/file, app info, disk free) and emit `job:*` / `app:error`
events with throttling.

**Acceptance criteria:**
- [ ] Every method in `SPEC-app-api.md` callable from JS bindings
- [ ] `job:updated` throttled ≤4/s per job
- [ ] `GetAppInfo` reports the active config path

**Verification:** `go test ./internal/app/...`; manual call from React devtools
**Dependencies:** Task 8, Task 9, Task 2
**Files:** `app.go`, `internal/app/api.go`, `internal/app/api_test.go`

#### Checkpoint: Queue & API
- [ ] `go test -race ./...` passes
- [ ] Bound methods + events work from the frontend
- [ ] Review with human before proceeding

---

## Phase 4 — UI

### Task 11: `download-screen` — input, summary, format picker  [M]
**Description:** URL input (any site) + Download button; auto-probe on paste;
post-probe media summary with source badge, thumbnail, title, duration; dynamic
format picker (video/audio toggle + quality dropdown).

**Acceptance criteria:**
- [ ] Non-YouTube URLs accepted; source badge shows detected extractor
- [ ] Missing thumbnail/duration degrades gracefully
- [ ] Download disabled while URL empty/invalid; spinner while probing

**Verification:** `npm run test`; manual paste of 3 site URLs
**Dependencies:** Task 3, Task 10
**Files:** `frontend/src/features/download/DownloadScreen.tsx`,
`UrlInput.tsx`, `MediaSummary.tsx`, `FormatPicker.tsx`

---

### Task 12: Queue list — items, progress, actions  [M]
**Description:** "Download Queue" header with `N queued • M downloading`;
`QueueItem` with status icon, title, status line, `LinearProgress`, and icon
actions (pause/resume/cancel/open/retry).

**Acceptance criteria:**
- [ ] Each state renders the correct icon/color/action
- [ ] Progress bar advances from events (no polling)
- [ ] Actions are icon buttons with tooltips

**Verification:** `npm run test`; manual run through all states
**Dependencies:** Task 11
**Files:** `frontend/src/features/download/QueueList.tsx`,
`QueueItem.tsx`, `useDownloadQueue.ts`

---

### Task 13: Storage bar + `settings-screen`  [M]
**Description:** Footer storage bar (path, free space, folder/settings icon
buttons) and the settings form (dir picker, concurrency 1–8, default
quality/format, cookies) plus a config-path note.

**Acceptance criteria:**
- [ ] Settings persist after save + restart
- [ ] Concurrency limited to 1–8
- [ ] Folder icon opens the native dialog; active config path displayed

**Verification:** `npm run test`; manual settings change + restart
**Dependencies:** Task 11, Task 10
**Files:** `frontend/src/features/download/StorageBar.tsx`,
`frontend/src/features/settings/SettingsScreen.tsx`, `DirPicker.tsx`,
`ConfigPathNote.tsx`

---

### Task 14: Wire live events end-to-end  [S]
**Description:** Ensure `useDownloadQueue` subscribes to `job:added/updated/removed`
and `app:error` and reconciles state without polling; verify throttling and
cleanup on unmount.

**Acceptance criteria:**
- [ ] No `setInterval` polling anywhere in the frontend
- [ ] Events update the correct item; unsubscribe on unmount
- [ ] `app:error` surfaces a toast/snackbar

**Verification:** `npm run test`; manual download while watching the list
**Dependencies:** Task 12, Task 13
**Files:** `frontend/src/features/download/useDownloadQueue.ts`,
`frontend/src/App.tsx`

#### Checkpoint: UI
- [ ] Paste → Download → live progress; pause/resume/cancel all work
- [ ] Design matches the reference image
- [ ] Review with human before proceeding

---

## Phase 5 — Packaging

### Task 15: Bundle binaries + NSIS installer  [M]
**Description:** Upgrade Wails to ≥ v2.13 (required for `-installscope`), then
configure NSIS: per-user scope, **LZMA solid** compression, explicit `File` lines
for the three binaries, product metadata, and **no components page**. Produce
`wails build -nsis -installscope user`. Confirm `config.json` is writable in the
install folder and that the uninstaller removes it.

**Acceptance criteria:**
- [ ] Wails CLI + module upgraded to **≥ v2.13.0** (v2.16.0 recommended); `wails build --help` shows `-installscope`
- [ ] `project.nsi` committed with `SetCompressor /SOLID lzma` + `File` lines for yt-dlp/ffmpeg/ffprobe
- [ ] `wails build -nsis -installscope user` emits `build/bin/*-installer.exe`
- [ ] Installer is LZMA-compressed, ~65–75 MB
- [ ] No checkbox/components page; all three binaries always installed
- [ ] Installs per-user to `$LOCALAPPDATA\Programs\...` with no UAC prompt
- [ ] Installed app finds bundled binaries and downloads (video + MP3)
- [ ] `config.json` created in the install folder
- [ ] Uninstaller removes the program + `config.json`/`history.json`; leaves downloads untouched

**Verification:**
- [ ] `wails build --help | grep installscope` (flag present)
- [ ] Manual install on a clean account; download a video + an MP3
- [ ] Confirm `config.json` path in-app via `GetAppInfo`
- [ ] Confirm installer size (~65–75 MB) and that no components page appears

**Dependencies:** Task 1–14
**Files:** `wails.json`, `go.mod`, `build/windows/installer/project.nsi`,
`build/windows/installer/info.json`, `resources/`

#### Checkpoint: Complete
- [ ] All acceptance criteria met across specs
- [ ] Installer verified on a clean install
- [ ] Ready for review / release
