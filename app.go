package main

import (
	"context"
	"os/exec"
	"path/filepath"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ytdl-gui/internal/app"
	"ytdl-gui/internal/queue"
	"ytdl-gui/internal/settings"
	"ytdl-gui/internal/ytdlp"
)

// App is the Wails-bound application struct. Every method the frontend calls
// lives on Service; App forwards to it and adds the Wails-specific plumbing
// (context, native dialogs, shell, events). No internal module is exposed
// directly.
type App struct {
	ctx     context.Context
	svc     *app.Service
	version string
}

// NewApp wires the settings store, history, engine, and the event bridge, then
// builds the service the frontend talks to.
func NewApp(version string) *App {
	// Config lives in the install folder (falling back to %APPDATA% if the
	// install folder is read-only). History sits beside it.
	cfgPath, err := settings.ConfigPath()
	if err != nil {
		// Fall back to a relative path so the app still starts.
		cfgPath = "config.json"
	}
	store := settings.NewStore(cfgPath)
	history := queue.NewHistory(filepath.Join(filepath.Dir(cfgPath), "history.json"))

	// Resolve the bundled binaries next to the executable. A missing binary is
	// surfaced through the UI rather than crashing the app.
	bins, binsErr := ytdlp.ResolveBins()

	a := &App{version: version}

	engine := ytdlpEngine{bins: bins}
	if binsErr != nil {
		engine.binsErr = binsErr
	}

	svc := app.NewService(app.Deps{
		Settings:      store,
		Engine:        engine,
		MaxConcurrent: settings.Defaults().MaxConcurrent,
		Emit: func(name string, data ...any) {
			if a.ctx != nil {
				wruntime.EventsEmit(a.ctx, name, data...)
			}
		},
		PickDir: func() (string, error) {
			return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
				Title: "Choose download folder",
			})
		},
		OpenPath: openInShell,
		Now:      nil, // real clock
		Version:  version,
		Versions: func(ctx context.Context) (ytdlp.Versions, error) {
			return ytdlp.QueryVersions(ctx, bins)
		},
		DiskFree: diskFree,
		History:  history,
	})
	a.svc = svc
	return a
}

// startup stores the Wails context (for dialogs/events) and restores history.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.svc.SetContext(ctx)
	_ = a.svc.Restore()
}

// openInShell opens a path with the OS default handler.
func openInShell(path string) error {
	return exec.Command("cmd", "/c", "start", "", path).Start()
}

// ---------------------------------------------------------------------------
// Bound methods (the Wails binding surface)
//
// These are thin forwards to the service so the frontend's call path is stable
// (window.go.main.App.*) and the API contract lives in one place.
// ---------------------------------------------------------------------------

// GetSettings returns the current configuration.
func (a *App) GetSettings() (settings.Settings, error) { return a.svc.GetSettings() }

// SaveSettings validates and persists the configuration.
func (a *App) SaveSettings(s settings.Settings) error { return a.svc.SaveSettings(s) }

// PickDownloadDir opens the native folder picker.
func (a *App) PickDownloadDir() (string, error) { return a.svc.PickDownloadDir() }

// ProbeURL returns metadata and selectable formats for any supported URL.
func (a *App) ProbeURL(url string) (*ytdlp.MediaInfo, error) { return a.svc.ProbeURL(url) }

// StartDownload enqueues a job and returns its id.
func (a *App) StartDownload(req app.DownloadRequest) (string, error) { return a.svc.StartDownload(req) }

// PauseJob pauses a running job.
func (a *App) PauseJob(id string) error { return a.svc.PauseJob(id) }

// ResumeJob resumes a paused job.
func (a *App) ResumeJob(id string) error { return a.svc.ResumeJob(id) }

// CancelJob cancels a queued or running job.
func (a *App) CancelJob(id string) error { return a.svc.CancelJob(id) }

// RemoveJob deletes a finished job from the list.
func (a *App) RemoveJob(id string) error { return a.svc.RemoveJob(id) }

// ListJobs returns the current queue and history.
func (a *App) ListJobs() []queue.Job { return a.svc.ListJobs() }

// OpenDownloadDir opens the configured download directory.
func (a *App) OpenDownloadDir() error { return a.svc.OpenDownloadDir() }

// OpenFile opens a file with the OS shell.
func (a *App) OpenFile(path string) error { return a.svc.OpenFile(path) }

// GetAppInfo reports the app version, config path, and binary versions.
func (a *App) GetAppInfo() app.AppInfo { return a.svc.GetAppInfo() }

// GetDiskFree reports free bytes for a directory.
func (a *App) GetDiskFree(dir string) (int64, error) { return a.svc.GetDiskFree(dir) }
