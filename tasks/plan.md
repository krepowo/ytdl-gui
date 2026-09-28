# Implementation Plan: Video Downloader (yt-dlp GUI)

## Overview

Build a Windows desktop GUI for `yt-dlp` using **Wails v2** (Go backend) and a
**React 19 + MUI v9 + Tailwind v4** frontend, shipped as an **NSIS installer**.
The app probes a URL from *any* yt-dlp-supported site, lets the user pick
quality/format (video or audio/MP3), and manages a parallel download queue with
pause/resume/cancel. Bundled `yt-dlp.exe` + `ffmpeg.exe`/`ffprobe.exe` mean zero
external installs.

## Architecture Decisions

| Decision | Rationale |
|---|---|
| **8 modules, strict dependency direction** (see `CAPABILITY-MAP.md`) | `app-api` is the only Go↔React boundary; `ytdlp-engine` is the only module that knows yt-dlp's CLI. Swappable engine, testable queue. |
| **Config in the install folder** with `%APPDATA%` fallback | Per user request. Safe because NSIS installs **per-user** to `$LOCALAPPDATA\Programs\...` (writable, no UAC). Fallback protects machine-wide installs. |
| **Bundle ffmpeg + ffprobe, always** | MP3 extraction and merging separate video/audio streams *require* ffmpeg. One installer with everything included = no broken states for the user. |
| **Event-driven progress** (`job:updated`), throttled ≤4/s | No polling; keeps UI responsive and bridge traffic bounded. |
| **Pause = OS process suspend**, fallback kill + `--continue` | yt-dlp has no native pause; `--continue` resumes from `.part`. |
| **Probe returns dynamic formats, never a fixed ladder** | Sites differ wildly (combined A/V vs. separate streams, missing metadata). |
| **Tailwind v4 CSS-first (`@theme`), no `tailwind.config.js`** | v4's supported setup; tokens live in `index.css` + `tokens.ts` for MUI. |
| **MUI v9 with CSS variables mode** | Shares derived colors with Tailwind; `color-mix()` based. |
| **Icon-first UI** (Material icons + tooltips) | Per user request; text only where an icon is ambiguous. |

## Tech Stack (pinned)

Backend: Go 1.26, **Wails v2 ≥ 2.13 (upgrade to latest v2 — 2.16.0 — for `-installscope`; 2.12.0 lacks it)**. Frontend: React 19.3, @mui/material 9.4,
@mui/icons-material 9.4, @emotion 11.14, tailwindcss 4.3 + @tailwindcss/vite,
Vite 8.3, TypeScript 5.x (pinned), Vitest 5.

## Task List

### Phase 1 — Foundation

- [ ] **Task 1: Scaffold the Wails project + frontend toolchain**
- [ ] **Task 2: `settings` module (Go) — install-dir config + fallback**
- [ ] **Task 3: `ui-shell` — tokens, MUI theme, Tailwind `@theme`, TitleBar, Layout**

#### Checkpoint: Foundation
- [ ] `wails dev` opens the frameless dark window; title bar drag/min/close work
- [ ] `go test ./...` and `npm run typecheck` pass

### Phase 2 — Engine

- [ ] **Task 4: Bundled binary resolution + version checks**
- [ ] **Task 5: `Probe` — site-agnostic metadata + dynamic formats**
- [ ] **Task 6: Progress-line parser**
- [ ] **Task 7: Download runner — spawn, stream progress, pause/resume/cancel**

#### Checkpoint: Engine
- [ ] Probe works for YouTube, Twitter/X, and a generic page
- [ ] MP3 job produces a valid file; a separate-stream site merges to one file
- [ ] Cancel leaves no orphan processes

### Phase 3 — Queue & API

- [ ] **Task 8: `download-queue` — state machine + concurrency scheduler**
- [ ] **Task 9: History persistence (survive restart)**
- [ ] **Task 10: `app-api` — bound methods + event bridge**

#### Checkpoint: Queue & API
- [ ] 5 jobs @ concurrency 2 never exceed 2 running; `go test -race` passes
- [ ] All bound methods callable from generated JS bindings

### Phase 4 — UI

- [ ] **Task 11: `download-screen` — URL input, media summary, format picker**
- [ ] **Task 12: Queue list — items, progress, per-item actions**
- [ ] **Task 13: Storage bar + `settings-screen`**
- [ ] **Task 14: Wire live events (`useDownloadQueue`) end-to-end**

#### Checkpoint: UI
- [ ] Paste → Download → live progress; pause/resume/cancel work
- [ ] Design matches the reference image (dark, coral, rounded)

### Phase 5 — Packaging

- [ ] **Task 15: Bundle binaries + NSIS installer (upgrade Wails ≥2.13, per-user, LZMA, all-inclusive)**

#### Checkpoint: Complete
- [ ] Wails upgraded to ≥ 2.13 (2.16.0); `-installscope user` works
- [ ] `wails build -nsis -installscope user` produces a ~65–75 MB LZMA installer; no UAC
- [ ] All three binaries installed next to the exe; downloads work (video + MP3)
- [ ] `config.json` lands in the install folder
- [ ] Uninstaller removes the app + config/history

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| MUI v9 + Tailwind v4 preflight/CssBaseline reset conflicts | Med | Import Tailwind first, mount `CssBaseline` after; add `@layer base` overrides only if needed (Task 3). |
| Process-tree kill unreliable for yt-dlp children (ffmpeg) | High | Use Windows Job Objects (`CREATE_NEW_PROCESS_GROUP` + job) so cancel/quit kills the whole tree (Task 7). |
| `NtSuspendProcess` pause is fragile | Med | Fallback to kill + `--continue`; keep pause behind an interface (Task 7). |
| NSIS not installed / not on PATH | Low | Already present (NSIS 3.13). Verify `makensis` on PATH in Task 15. |
| **Wails 2.12.0 lacks `-installscope`** (per-user install) | **High** | Verified: the flag + `${WAILS_INSTALL_SCOPE}` template handling landed in **v2.13.0**. Upgrade Wails CLI + module to latest v2 (2.16.0) in Task 15 before building. |
| Wails default NSIS template uses **zlib, not LZMA** | Med | Verified: no `SetCompressor` in the default `project.nsi`. We add `SetCompressor /SOLID lzma` ourselves (Task 15) or the installer stays ~200 MB. |
| Extra binaries aren't embedded by Wails' `wails.files` | Med | Verified: it only embeds the app exe. We add explicit `File` lines for yt-dlp/ffmpeg/ffprobe to `project.nsi` (Task 15); Wails preserves an existing `project.nsi`. |
| Bundling ffmpeg bloats the payload (~212 MB on disk) | Low | NSIS **LZMA solid** compression → installer ≈ 65–75 MB. Verify the flag is active in Task 15. |
| Site extractors break over time | Med | Bundled yt-dlp can be refreshed; add a version display + note in `GetAppInfo` (Task 4). |

## Open Questions

- None. (TypeScript pinned to 5.x; installer is all-inclusive with LZMA; uninstaller deletes `config.json`.)
