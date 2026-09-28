# Spec: app-api

## Objective
Be the single contract between Go and React. Every frontend capability maps to
one bound Go method or one emitted event. No other module is exposed to Wails.

## Tech Stack
Wails v2 bindings (`wailsjs/go/...`) + runtime events (`runtime.EventsEmit`).

## Commands
- Regenerate bindings: `wails generate module` (or automatically on `wails dev`/`build`).
- Test: `go test ./internal/app/...`

## Project Structure
```
app.go            → App struct + all bound methods (Wails entrypoint)
internal/app/
  api.go          → thin adapters over engine + queue + settings
```

## Code Style
```go
// ProbeURL returns metadata + selectable formats for any yt-dlp-supported URL.
func (a *App) ProbeURL(url string) (*ytdlp.MediaInfo, error) {
	s, _ := a.settings.Load()
	return a.engine.Probe(a.ctx, a.bins, url, ytdlp.ProbeOptions{CookiesBrowser: s.CookiesBrowser})
}

// StartDownload enqueues a job and returns its id.
func (a *App) StartDownload(req DownloadRequest) (string, error) {
	return a.queue.Add(req), nil
}
```

## Bound Methods (the interface)
| Method | Purpose |
|---|---|
| `GetSettings() Settings` | Read config |
| `SaveSettings(Settings) error` | Persist config |
| `PickDownloadDir() string` | Native folder dialog |
| `ProbeURL(url) (MediaInfo, error)` | Metadata + formats (any site) |
| `StartDownload(DownloadRequest) (id, error)` | Enqueue |
| `PauseJob(id) error` / `ResumeJob(id) error` / `CancelJob(id) error` | Controls |
| `RemoveJob(id) error` | Delete from list |
| `ListJobs() []Job` | Current queue + history |
| `OpenDownloadDir() error` / `OpenFile(path) error` | Shell open |
| `GetAppInfo() AppInfo` | App version, **active config path**, yt-dlp version, ffmpeg version |
| `GetDiskFree(dir) (bytes, error)` | Free space for the storage bar |

## Events (Go → React)
| Event | Payload | When |
|---|---|---|
| `job:updated` | `Job` | progress (≤4/s) or state change |
| `job:added` | `Job` | new job enqueued |
| `job:removed` | `id` | job deleted |
| `app:error` | `{message}` | fatal engine/binary error |

## Testing Strategy
`go test` with a mocked engine/queue to assert each bound method delegates
correctly and emits the right event payloads.

## Boundaries
- **Always:** return typed errors (not panics) to the frontend; throttle
  `job:updated` to ≤4/s per job; report the active config path in `AppInfo`.
- **Ask first:** renaming/removing any bound method or event (breaking change).
- **Never:** expose raw `os/exec` or file-system handles to the frontend; assume
  a specific source site.

## Success Criteria
- [ ] Every method above is callable from generated JS bindings.
- [ ] Progress updates arrive in React without polling (event-driven).
- [ ] A probe error returns a readable message, not a Go panic.
- [ ] `GetAppInfo` reports the config path actually in use (install dir or fallback).

## Open Questions
- None.
