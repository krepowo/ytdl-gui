//go:build manual

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ytdl-gui/internal/app"
	"ytdl-gui/internal/queue"
	"ytdl-gui/internal/settings"
	"ytdl-gui/internal/ytdlp"
)

// TestE2EProbeThenDownload drives the same path the UI takes - ProbeURL then
// StartDownload - through the real ytdlpEngine adapter and the real
// yt-dlp/ffmpeg binaries staged in resources/, asserting the event stream the
// frontend listens to (job:added, job:updated with progress, terminal state).
// This is the end-to-end proof for Checkpoint 4 without clicking the GUI.
func TestE2EProbeThenDownload(t *testing.T) {
	binDir := "resources"
	if _, err := os.Stat(filepath.Join(binDir, "yt-dlp.exe")); err != nil {
		t.Skip("resources/ binaries not staged")
	}

	outDir := t.TempDir()
	cfg := settings.Defaults()
	cfg.DownloadDir = outDir
	cfg.MaxConcurrent = 2

	// Point the real engine at the staged binaries (same as app.go does for the
	// packaged layout).
	bins := ytdlp.Bins{
		YtDlp:   filepath.Join(binDir, "yt-dlp.exe"),
		FFmpeg:  filepath.Join(binDir, "ffmpeg.exe"),
		FFprobe: filepath.Join(binDir, "ffprobe.exe"),
	}
	engine := ytdlpEngine{bins: bins}

	events := make(chan string, 512)
	svc := app.NewService(app.Deps{
		Settings:      &memSettings{cfg: cfg},
		Engine:        engine,
		MaxConcurrent: 2,
		Emit:          func(name string, _ ...any) { events <- name },
		Version:       "test",
		Versions: func(ctx context.Context) (ytdlp.Versions, error) {
			return ytdlp.QueryVersions(ctx, bins)
		},
		DiskFree: func(string) (int64, error) { return 0, nil },
		Now:      time.Now,
		PickDir:  func() (string, error) { return outDir, nil },
		OpenPath: func(string) error { return nil },
	})

	// 1) Probe a real URL, exactly as UrlInput does.
	const url = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	info, err := svc.ProbeURL(url)
	if err != nil {
		t.Fatalf("ProbeURL: %v", err)
	}
	if len(info.VideoOptions) == 0 {
		t.Fatal("probe returned no video options")
	}
	t.Logf("probed %q: extractor=%s video=%d audio=%d",
		info.Title, info.Extractor, len(info.VideoOptions), len(info.AudioOptions))

	// 2) Start a download using a real option, as DownloadScreen does. Prefer a
	// small combined format to keep the test quick.
	opt := pickSmallest(info.VideoOptions)
	jobID, err := svc.StartDownload(app.DownloadRequest{
		URL:        url,
		Mode:       "video",
		FormatID:   opt.FormatID,
		NeedsMerge: opt.NeedsMerge,
		Title:      info.Title,
	})
	if err != nil {
		t.Fatalf("StartDownload: %v", err)
	}
	t.Logf("started job %s (format %s, merge=%v)", jobID, opt.FormatID, opt.NeedsMerge)

	// 3) Watch the event stream until the job reaches a terminal state.
	deadline := time.After(4 * time.Minute)
	sawAdded, sawUpdated, sawProgress := false, false, false
	for {
		select {
		case name := <-events:
			switch name {
			case app.EventJobAdded:
				sawAdded = true
			case app.EventJobUpdated:
				sawUpdated = true
				for _, j := range svc.ListJobs() {
					if j.ID != jobID {
						continue
					}
					if j.Percent > 0 && j.Percent < 100 {
						sawProgress = true
					}
					if j.State == queue.StateCompleted {
						if !sawAdded || !sawUpdated || !sawProgress {
							t.Fatalf("event gaps: added=%v updated=%v progress=%v", sawAdded, sawUpdated, sawProgress)
						}
						fi, err := os.Stat(j.OutputPath)
						if err != nil {
							t.Fatalf("output not on disk (%s): %v", j.OutputPath, err)
						}
						t.Logf("PASS: completed %s (%d bytes) via events added=%v updated=%v progress=%v",
							filepath.Base(j.OutputPath), fi.Size(), sawAdded, sawUpdated, sawProgress)
						return
					}
					if j.State == queue.StateError {
						t.Fatalf("job failed: %s", j.Error)
					}
				}
			}
		case <-deadline:
			t.Fatalf("timed out; added=%v updated=%v progress=%v", sawAdded, sawUpdated, sawProgress)
		}
	}
}

// pickSmallest returns the lowest-resolution video option, preferring a
// progressive (video+audio in one file) option so the e2e download avoids
// YouTube's frequent 403 on separate DASH streams and needs no merge.
func pickSmallest(opts []ytdlp.FormatOption) ytdlp.FormatOption {
	// Prefer progressive options (no merge), smallest first.
	var progressive *ytdlp.FormatOption
	for i := range opts {
		o := opts[i]
		if o.NeedsMerge {
			continue
		}
		if progressive == nil || (o.Height > 0 && (progressive.Height == 0 || o.Height < progressive.Height)) {
			progressive = &opts[i]
		}
	}
	if progressive != nil {
		return *progressive
	}

	// Otherwise fall back to the lowest-resolution merged option.
	best := opts[0]
	for _, o := range opts {
		if o.Height > 0 && (best.Height == 0 || o.Height < best.Height) {
			best = o
		}
	}
	return best
}

// memSettings is a minimal in-memory SettingsStore for the e2e test.
type memSettings struct{ cfg settings.Settings }

func (m *memSettings) Load() (settings.Settings, error) { return m.cfg, nil }
func (m *memSettings) Save(s settings.Settings) error   { m.cfg = s; return nil }
func (m *memSettings) Path() string                     { return filepath.Join(os.TempDir(), "config.json") }
