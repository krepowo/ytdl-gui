// Package app is the single contract between Go and React. Every frontend
// capability maps to one method here or one emitted event; no other module is
// exposed to Wails.
package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ytdl-gui/internal/queue"
	"ytdl-gui/internal/settings"
	"ytdl-gui/internal/ytdlp"
)

// Event names emitted to the frontend. These are part of the API contract.
const (
	EventJobAdded   = "job:added"
	EventJobUpdated = "job:updated"
	EventJobRemoved = "job:removed"
	EventAppError   = "app:error"
)

// progressThrottle caps job:updated emissions to 4/second per job.
const progressThrottle = 250 * time.Millisecond

// JobHandle is a running download as seen by the app layer. The ytdlp package
// is adapted to this shape so the app layer does not depend on its concrete type.
type JobHandle interface {
	Wait() error
	OutputPath() string
	Pause() error
	Resume() error
	Cancel() error
}

// DownloadEngine probes URLs and starts downloads. Implemented over ytdlp in
// main; faked in tests.
type DownloadEngine interface {
	Probe(ctx context.Context, url string, opts ytdlp.ProbeOptions) (*ytdlp.MediaInfo, error)
	Start(ctx context.Context, opts ytdlp.DownloadOptions, onProgress func(ytdlp.Progress), onState func(ytdlp.JobState)) (JobHandle, error)
}

// SettingsStore reads and writes app configuration.
type SettingsStore interface {
	Load() (settings.Settings, error)
	Save(settings.Settings) error
	Path() string
}

// Deps are the collaborators a Service needs. Everything with a side effect is
// injected so the service is fully testable.
type Deps struct {
	Settings      SettingsStore
	Engine        DownloadEngine
	MaxConcurrent int
	// Emit publishes an event to the frontend (runtime.EventsEmit in main).
	Emit func(name string, data ...any)
	// PickDir opens the native folder picker.
	PickDir func() (string, error)
	// OpenPath opens a file/folder with the OS shell.
	OpenPath func(path string) error
	// Now is injectable for deterministic throttle tests.
	Now func() time.Time
	// Version is the app version reported in AppInfo.
	Version string
	// Versions reports the bundled binary versions.
	Versions func(ctx context.Context) (ytdlp.Versions, error)
	// DiskFree reports free bytes for a directory.
	DiskFree func(dir string) (int64, error)
	// History persists finished jobs. Optional.
	History *queue.History
}

// DownloadRequest is the frontend's request to enqueue a download.
type DownloadRequest struct {
	URL        string `json:"url"`
	Mode       string `json:"mode"` // "video" | "audio"
	FormatID   string `json:"formatId"`
	NeedsMerge bool   `json:"needsMerge"`
	Title      string `json:"title"`
}

// AppInfo describes the running app for the settings/about UI.
type AppInfo struct {
	Version       string `json:"version"`
	ConfigPath    string `json:"configPath"`
	YtDlpVersion  string `json:"ytDlpVersion"`
	FFmpegVersion string `json:"ffmpegVersion"`
}

// Service implements every bound method and owns the event bridge.
type Service struct {
	deps     Deps
	queue    *queue.Queue
	openPath func(string) error

	// engineAdapter maps queue requests to engine calls.
	engine DownloadEngine

	mu       sync.Mutex
	lastEmit map[string]time.Time // job id -> last job:updated time
	ctx      context.Context
}

// NewService wires the queue and the event bridge.
func NewService(deps Deps) *Service {
	s := &Service{
		deps:     deps,
		engine:   deps.Engine,
		openPath: deps.OpenPath,
		lastEmit: map[string]time.Time{},
		// Default to a background context so the service is usable before
		// startup() swaps in the Wails context. A nil context would panic in
		// exec.CommandContext if a call arrived early.
		ctx: context.Background(),
	}

	s.queue = queue.New(queue.Options{
		Engine:        s,
		MaxConcurrent: deps.MaxConcurrent,
		Now:           deps.Now,
		History:       deps.History,
		OnChange:      s.onChange,
	})
	return s
}

// SetContext stores the Wails startup context (used for probe/start cancellation).
func (s *Service) SetContext(ctx context.Context) { s.ctx = ctx }

// Restore loads persisted jobs. Interrupted jobs come back paused and are never
// auto-started.
func (s *Service) Restore() error { return s.queue.Restore() }

// ---------------------------------------------------------------------------
// queue.Engine adapter
// ---------------------------------------------------------------------------

// Start implements queue.Engine. It translates a queue request into an engine
// download, routing progress back to the right job.
func (s *Service) Start(jobID string, req queue.DownloadRequest) (queue.EngineJob, error) {
	onProgress := s.onProgressFor(jobID)

	handle, err := s.engine.Start(s.ctx, ytdlp.DownloadOptions{
		URL:            req.URL,
		OutputDir:      req.OutputDir,
		Mode:           ytdlp.Mode(req.Mode),
		FormatID:       req.FormatID,
		NeedsMerge:     req.NeedsMerge,
		CookiesBrowser: req.CookiesBrowser,
	}, onProgress, nil)
	if err != nil {
		return nil, err
	}
	return handle, nil
}

// onProgressFor returns a throttled progress callback bound to one job. The
// first update and any update after the throttle window are forwarded; a
// terminal (100%) update always passes so the UI never looks stuck.
func (s *Service) onProgressFor(jobID string) func(ytdlp.Progress) {
	return func(p ytdlp.Progress) {
		if !s.allowEmit(jobID, p.Percent) {
			return
		}
		s.queue.UpdateProgress(jobID, queue.ProgressUpdate{
			Percent:         p.Percent,
			DownloadedBytes: p.DownloadedBytes,
			TotalBytes:      p.TotalBytes,
			SpeedBps:        p.SpeedBps,
			ETASec:          p.ETASec,
		})
	}
}

// allowEmit reports whether a progress update for jobID should be forwarded.
func (s *Service) allowEmit(jobID string, percent float64) bool {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	last, seen := s.lastEmit[jobID]
	if seen && percent < 100 && now.Sub(last) < progressThrottle {
		return false
	}
	s.lastEmit[jobID] = now
	return true
}

// now returns the injected clock or the real one.
func (s *Service) now() time.Time {
	if s.deps.Now != nil {
		return s.deps.Now()
	}
	return time.Now()
}

// ---------------------------------------------------------------------------
// Event bridge
// ---------------------------------------------------------------------------

// onChange maps a queue change to the frontend event.
func (s *Service) onChange(ev queue.ChangeEvent) {
	switch ev.Type {
	case queue.ChangeAdded:
		s.emit(EventJobAdded, ev.Job)
	case queue.ChangeRemoved:
		s.emit(EventJobRemoved, ev.Job.ID)
	default:
		s.emit(EventJobUpdated, ev.Job)
	}
}

// emit publishes an event if a sink is configured.
func (s *Service) emit(name string, data ...any) {
	if s.deps.Emit != nil {
		s.deps.Emit(name, data...)
	}
}

// ---------------------------------------------------------------------------
// Bound methods
// ---------------------------------------------------------------------------

// GetSettings returns the current configuration.
func (s *Service) GetSettings() (settings.Settings, error) {
	return s.deps.Settings.Load()
}

// SaveSettings validates and persists the configuration.
func (s *Service) SaveSettings(in settings.Settings) error {
	return s.deps.Settings.Save(in)
}

// PickDownloadDir opens the native folder picker and returns the chosen path.
func (s *Service) PickDownloadDir() (string, error) {
	if s.deps.PickDir == nil {
		return "", nil
	}
	return s.deps.PickDir()
}

// ProbeURL returns metadata and selectable formats for any yt-dlp-supported URL.
// The configured cookies browser is applied automatically.
func (s *Service) ProbeURL(url string) (*ytdlp.MediaInfo, error) {
	cfg, err := s.deps.Settings.Load()
	if err != nil {
		return nil, err
	}
	info, err := s.engine.Probe(s.ctx, url, ytdlp.ProbeOptions{CookiesBrowser: cfg.CookiesBrowser})
	if err != nil {
		// Surface a readable message; the frontend shows it, never a panic.
		return nil, err
	}
	return info, nil
}

// StartDownload enqueues a job and returns its id. The output directory and
// cookies browser come from settings so the frontend does not have to send them.
func (s *Service) StartDownload(req DownloadRequest) (string, error) {
	cfg, err := s.deps.Settings.Load()
	if err != nil {
		return "", err
	}

	mode := req.Mode
	if mode == "" {
		mode = cfg.DefaultMode
	}

	job, err := s.queue.Add(queue.DownloadRequest{
		URL:            req.URL,
		OutputDir:      cfg.DownloadDir,
		Mode:           mode,
		FormatID:       req.FormatID,
		NeedsMerge:     req.NeedsMerge,
		CookiesBrowser: cfg.CookiesBrowser,
		Title:          req.Title,
	})
	if err != nil {
		return "", err
	}
	return job.ID, nil
}

// PauseJob pauses a running job.
func (s *Service) PauseJob(id string) error { return s.queue.Pause(id) }

// ResumeJob resumes a paused job.
func (s *Service) ResumeJob(id string) error { return s.queue.Resume(id) }

// CancelJob cancels a queued or running job.
func (s *Service) CancelJob(id string) error { return s.queue.Cancel(id) }

// RemoveJob deletes a finished job from the list.
func (s *Service) RemoveJob(id string) error { return s.queue.Remove(id) }

// ListJobs returns the current queue and history.
func (s *Service) ListJobs() []queue.Job { return s.queue.List() }

// OpenDownloadDir opens the configured download directory in the OS shell.
func (s *Service) OpenDownloadDir() error {
	cfg, err := s.deps.Settings.Load()
	if err != nil {
		return err
	}
	if s.openPath == nil {
		return nil
	}
	return s.openPath(cfg.DownloadDir)
}

// OpenFile opens a file (e.g. a finished download) with the OS shell.
func (s *Service) OpenFile(path string) error {
	if s.openPath == nil {
		return nil
	}
	return s.openPath(path)
}

// GetAppInfo reports the app version, the active config path, and the bundled
// binary versions.
func (s *Service) GetAppInfo() AppInfo {
	info := AppInfo{
		Version:    s.deps.Version,
		ConfigPath: s.deps.Settings.Path(),
	}
	if s.deps.Versions != nil {
		if v, err := s.deps.Versions(s.ctx); err == nil {
			info.YtDlpVersion = v.YtDlp
			info.FFmpegVersion = v.FFmpeg
		}
	}
	return info
}

// GetDiskFree reports free bytes for the storage bar.
func (s *Service) GetDiskFree(dir string) (int64, error) {
	if s.deps.DiskFree == nil {
		return 0, fmt.Errorf("app: disk free not configured")
	}
	return s.deps.DiskFree(dir)
}
