# Spec: download-screen

## Objective
The main screen: paste a URL **from any supported site**, pick quality/format,
hit Download, and watch the queue. Mirrors the reference design — URL input +
Download button, then an "Antrean Download" list with per-item status, progress,
and controls. Icon-first throughout.

## Tech Stack
React 19 + TypeScript, MUI v9 (Button, TextField, LinearProgress, IconButton,
Chip, Tooltip, Select), Tailwind v4 for layout, Wails events for live updates.

## Commands
- Dev: `wails dev`
- Test: `npm run test`
- Typecheck: `npm run typecheck`

## Project Structure
```
frontend/src/features/download/
  DownloadScreen.tsx     → page layout
  UrlInput.tsx           → label + input (link icon) + Download button
  MediaSummary.tsx       → post-probe: thumbnail, title, source badge, duration
  FormatPicker.tsx       → video/audio toggle + dynamic quality dropdown
  QueueList.tsx          → header ("Antrean Download") + summary count
  QueueItem.tsx          → status icon, title, status line, progress bar, actions
  useDownloadQueue.ts    → subscribes to job:* events, calls bound methods
  StorageBar.tsx         → footer: path + free space + icon actions
```

## Multi-Source Behaviour
- The input accepts **any URL yt-dlp supports** — no YouTube-specific
  validation or placeholder wording.
- After a successful probe, show a **source badge** (from `MediaInfo.Extractor`)
  so the user sees the detected site, plus title/thumbnail/duration **when
  present** (these are optional — hide missing fields, never show "undefined").
- `FormatPicker` is built from the **actual** returned formats. If the site only
  offers combined streams, show quality options; if it offers separate
  video/audio, offer a "merge to MP4" path (engine handles ffmpeg).
- Live streams (`is_live`) show a "LIVE" chip and a byte-counter progress bar
  instead of a percentage.

## Code Style
```tsx
// QueueItem.tsx — status drives icon + color, matching the reference.
const status = {
  downloading: { icon: <DownloadIcon />,         color: "primary.main", action: "pause"  },
  queued:      { icon: <ScheduleIcon />,         color: "text.disabled", action: "cancel" },
  completed:   { icon: <CheckCircleIcon />,      color: "success.main", action: "open"   },
  paused:      { icon: <PauseIcon />,            color: "text.disabled", action: "resume" },
  error:       { icon: <ErrorIcon />,            color: "error.main",   action: "retry"  },
}[job.state];
```
- All colors via theme/tokens; no inline hex.
- Status line format: `Mengunduh • 42.3 MB / 128 MB • 33%`.
- Progress via MUI `LinearProgress` with coral fill.
- Actions are **icon buttons with tooltips** (pause/resume/cancel/open/retry).

## Testing Strategy
Vitest + RTL. Tests: pasting a URL and clicking Download calls `StartDownload`;
a `job:updated` event updates the matching item's percent; each state renders
the correct icon/action; a probe result with a missing thumbnail renders without
errors; a non-YouTube extractor shows the right source badge. Mock the Wails bindings.

## Boundaries
- **Always:** disable Download while the URL is empty/invalid; show a spinner
  while probing; keep controls reachable by keyboard; every icon button has an
  `aria-label`.
- **Ask first:** adding a bulk action (download-all, clear-finished).
- **Never:** poll the backend on a timer (use events); block the UI thread;
  assume the source is YouTube.

## Success Criteria
- [ ] Paste URL → Download → item appears with `queued` then `downloading`.
- [ ] Works for at least YouTube, Twitter/X, and a generic page URL.
- [ ] Progress bar advances live from events (no polling).
- [ ] Pause/resume/cancel work from the item's action icon and update state.
- [ ] Completed item shows green check + "open folder" action; failed item shows error + retry.
- [ ] Queue header shows `N antrean • M sedang diunduh`.
- [ ] Missing optional metadata (thumbnail/duration) degrades gracefully.

## Open Questions
- Format picker appears **after** a successful probe (auto-probe on paste), to
  keep the default view as simple as the reference.
