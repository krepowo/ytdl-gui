# Spec: settings

## Objective
Own the persisted application configuration so every other module reads one
source of truth. Config lives **inside the installation folder** (next to the
installed executable), per the product decision. The user can change where
downloads go, how many run at once, the default quality/format, and whether to
use browser cookies for sites that need them. Config survives restarts and
upgrades.

## Tech Stack
Go stdlib (`encoding/json`, `os`, `path/filepath`), stored at
`<exe-dir>/config.json`.

## Commands
- Build: `wails build`
- Test: `go test ./internal/settings/...`
- Lint: `gofmt -l . && go vet ./...`
- Dev: `wails dev`

## Project Structure
```
internal/settings/
  settings.go     → Settings struct, defaults, Load/Save, path resolution
  settings_test.go
```

## Code Style
```go
// Settings is the persisted, user-editable app configuration.
type Settings struct {
	DownloadDir    string `json:"downloadDir"`
	MaxConcurrent  int    `json:"maxConcurrent"`
	DefaultMode    string `json:"defaultMode"`    // "video" | "audio"
	DefaultQuality string `json:"defaultQuality"` // "best" | "1080" | "720" | "480" | "360"
	FilenameTmpl   string `json:"filenameTemplate"`
	CookiesBrowser string `json:"cookiesBrowser"` // "" | "chrome" | "edge" | "firefox" | "brave"
}

// ConfigPath returns <exe-dir>/config.json, falling back to %APPDATA% only if
// the install folder is not writable (e.g. a machine-wide install).
func ConfigPath() (string, error) { /* ... */ }
```

## Storage Location & Fallback
- **Primary:** `<dir of the running .exe>/config.json` — writable because the
  NSIS installer uses **per-user scope** (`$LOCALAPPDATA\Programs\...`).
- **Fallback:** if the exe dir is read-only (machine-wide install, portable
  media, `Program Files`), write to `%APPDATA%\VideoDownloader\config.json` and
  surface a one-time notice in the UI. The app must never crash over this.

## Testing Strategy
Go `testing`. Table-driven tests for defaults, load-missing-file, save→load
round-trip, validation clamping (`MaxConcurrent` 1–8), and the read-only-dir
fallback path (via an injected "is writable" probe). Target: full coverage of
`Load`/`Save`/`Validate`/`ConfigPath`.

## Boundaries
- **Always:** write config atomically (temp file + rename); validate on load;
  resolve the path from the executable location, not the CWD.
- **Ask first:** adding new config keys; changing the storage location or the
  fallback behaviour.
- **Never:** store absolute user paths as compile-time constants; crash on a
  missing/corrupt config file (fall back to defaults); assume the exe dir is
  writable without checking.

## Success Criteria
- [ ] Config file is created at `<exe-dir>/config.json` on first save.
- [ ] First run with no config file returns documented defaults (dir = `%USERPROFILE%\Downloads`, concurrency = 2, mode = video, quality = best, cookies = none).
- [ ] `Save` then `Load` returns identical values.
- [ ] Corrupt JSON falls back to defaults and logs a warning (no panic).
- [ ] `MaxConcurrent` outside 1–8 is clamped.
- [ ] On a read-only exe dir, config transparently uses the `%APPDATA%` fallback and the UI is told which path is active.

## Open Questions
- None.
