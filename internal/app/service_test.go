package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ytdl-gui/internal/queue"
	"ytdl-gui/internal/settings"
	"ytdl-gui/internal/ytdlp"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeSettings struct {
	mu     sync.Mutex
	loaded settings.Settings
	saved  []settings.Settings
	path   string
}

func (f *fakeSettings) Load() (settings.Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.loaded, nil
}

func (f *fakeSettings) Save(s settings.Settings) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, s)
	f.loaded = s
	return nil
}

func (f *fakeSettings) Path() string { return f.path }

type fakeHandle struct {
	mu       sync.Mutex
	done     chan struct{}
	err      error
	paused   bool
	resumed  bool
	canceled bool
	out      string
}

func newFakeHandle() *fakeHandle { return &fakeHandle{done: make(chan struct{})} }

func (h *fakeHandle) Wait() error {
	<-h.done
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

func (h *fakeHandle) OutputPath() string { return h.out }

func (h *fakeHandle) Pause() error {
	h.mu.Lock()
	h.paused = true
	h.mu.Unlock()
	return nil
}

func (h *fakeHandle) Resume() error {
	h.mu.Lock()
	h.resumed = true
	h.mu.Unlock()
	return nil
}

func (h *fakeHandle) Cancel() error {
	h.mu.Lock()
	h.canceled = true
	h.mu.Unlock()
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	return nil
}

func (h *fakeHandle) finish(err error) {
	h.mu.Lock()
	h.err = err
	h.mu.Unlock()
	select {
	case <-h.done:
	default:
		close(h.done)
	}
}

type fakeEngine struct {
	mu         sync.Mutex
	probeInfo  *ytdlp.MediaInfo
	probeErr   error
	probeCalls []probeCall
	startErr   error
	startOpts  []ytdlp.DownloadOptions
	handles    []*fakeHandle
}

type probeCall struct {
	url  string
	opts ytdlp.ProbeOptions
}

func (e *fakeEngine) Probe(_ context.Context, url string, opts ytdlp.ProbeOptions) (*ytdlp.MediaInfo, error) {
	e.mu.Lock()
	e.probeCalls = append(e.probeCalls, probeCall{url: url, opts: opts})
	info, err := e.probeInfo, e.probeErr
	e.mu.Unlock()
	return info, err
}

func (e *fakeEngine) Start(_ context.Context, opts ytdlp.DownloadOptions, onProgress func(ytdlp.Progress), _ func(ytdlp.JobState)) (JobHandle, error) {
	e.mu.Lock()
	e.startOpts = append(e.startOpts, opts)
	if e.startErr != nil {
		err := e.startErr
		e.mu.Unlock()
		return nil, err
	}
	h := newFakeHandle()
	e.handles = append(e.handles, h)
	e.mu.Unlock()
	return h, nil
}

func (e *fakeEngine) lastStartOpts() ytdlp.DownloadOptions {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.startOpts) == 0 {
		return ytdlp.DownloadOptions{}
	}
	return e.startOpts[len(e.startOpts)-1]
}

func (e *fakeEngine) handle(i int) *fakeHandle {
	e.mu.Lock()
	defer e.mu.Unlock()
	if i < 0 || i >= len(e.handles) {
		return nil
	}
	return e.handles[i]
}

func (e *fakeEngine) startCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.handles)
}

// recorder captures emitted events.
type recorder struct {
	mu     sync.Mutex
	events []recordedEvent
}

type recordedEvent struct {
	name string
	data []any
}

func (r *recorder) emit(name string, data ...any) {
	r.mu.Lock()
	r.events = append(r.events, recordedEvent{name: name, data: data})
	r.mu.Unlock()
}

func (r *recorder) all() []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedEvent, len(r.events))
	copy(out, r.events)
	return out
}

func (r *recorder) countNamed(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.name == name {
			n++
		}
	}
	return n
}

func (r *recorder) lastNamed(name string) (recordedEvent, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].name == name {
			return r.events[i], true
		}
	}
	return recordedEvent{}, false
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type harness struct {
	svc      *Service
	settings *fakeSettings
	engine   *fakeEngine
	rec      *recorder
	now      time.Time
}

func newHarness(t *testing.T, cfg settings.Settings) *harness {
	t.Helper()
	h := &harness{
		settings: &fakeSettings{loaded: cfg, path: `C:\app\config.json`},
		engine:   &fakeEngine{probeInfo: &ytdlp.MediaInfo{Title: "T", Extractor: "youtube"}},
		rec:      &recorder{},
		now:      time.Unix(1_700_000_000, 0),
	}

	h.svc = NewService(Deps{
		Settings:      h.settings,
		Engine:        h.engine,
		MaxConcurrent: 2,
		Emit:          h.rec.emit,
		PickDir:       func() (string, error) { return `D:\Picked`, nil },
		OpenPath:      func(string) error { return nil },
		Now:           func() time.Time { return h.now },
		Version:       "1.2.3",
		Versions: func(context.Context) (ytdlp.Versions, error) {
			return ytdlp.Versions{YtDlp: "2026.08.19", FFmpeg: "9.0"}, nil
		},
		DiskFree: func(string) (int64, error) { return 123456789, nil },
	})
	return h
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func TestGetSettingsDelegates(t *testing.T) {
	cfg := settings.Defaults()
	cfg.MaxConcurrent = 4
	h := newHarness(t, cfg)

	got, err := h.svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent = %d, want 4", got.MaxConcurrent)
	}
}

func TestSaveSettingsDelegates(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	in := settings.Defaults()
	in.MaxConcurrent = 6
	if err := h.svc.SaveSettings(in); err != nil {
		t.Fatal(err)
	}

	h.settings.mu.Lock()
	defer h.settings.mu.Unlock()
	if len(h.settings.saved) != 1 {
		t.Fatalf("saved %d times, want 1", len(h.settings.saved))
	}
	if h.settings.saved[0].MaxConcurrent != 6 {
		t.Errorf("saved MaxConcurrent = %d, want 6", h.settings.saved[0].MaxConcurrent)
	}
}

func TestPickDownloadDirDelegates(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	dir, err := h.svc.PickDownloadDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != `D:\Picked` {
		t.Errorf("dir = %q, want the picker result", dir)
	}
}

// ---------------------------------------------------------------------------
// Probe
// ---------------------------------------------------------------------------

func TestProbeURLPassesCookiesFromSettings(t *testing.T) {
	cfg := settings.Defaults()
	cfg.CookiesBrowser = "firefox"
	h := newHarness(t, cfg)

	if _, err := h.svc.ProbeURL("https://example.com/v"); err != nil {
		t.Fatal(err)
	}

	h.engine.mu.Lock()
	defer h.engine.mu.Unlock()
	if len(h.engine.probeCalls) != 1 {
		t.Fatalf("probe calls = %d, want 1", len(h.engine.probeCalls))
	}
	if h.engine.probeCalls[0].opts.CookiesBrowser != "firefox" {
		t.Errorf("cookies = %q, want firefox (from settings)", h.engine.probeCalls[0].opts.CookiesBrowser)
	}
	if h.engine.probeCalls[0].url != "https://example.com/v" {
		t.Errorf("url = %q, want the requested url", h.engine.probeCalls[0].url)
	}
}

func TestProbeURLErrorIsReturnedNotPanic(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	h.engine.probeErr = errors.New("Unsupported URL")

	_, err := h.svc.ProbeURL("https://bad.example/x")
	if err == nil {
		t.Fatal("err = nil, want the engine error surfaced")
	}
	if !errors.Is(err, h.engine.probeErr) && err.Error() != "Unsupported URL" {
		t.Errorf("err = %v, want it to carry the engine message", err)
	}
}

// ---------------------------------------------------------------------------
// Downloads
// ---------------------------------------------------------------------------

func TestStartDownloadFillsOutputDirAndCookiesFromSettings(t *testing.T) {
	cfg := settings.Defaults()
	cfg.DownloadDir = `E:\Media`
	cfg.CookiesBrowser = "chrome"
	h := newHarness(t, cfg)

	id, err := h.svc.StartDownload(DownloadRequest{URL: "https://example.com/v", Mode: "video"})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("id is empty")
	}

	// The queued job must carry the settings-derived fields.
	jobs := h.svc.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	if jobs[0].ID != id {
		t.Errorf("job ID = %q, want %q", jobs[0].ID, id)
	}

	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "engine did not start")

	// The engine must receive the resolved output dir + cookies from settings.
	opts := h.engine.lastStartOpts()
	if opts.OutputDir != `E:\Media` {
		t.Errorf("engine OutputDir = %q, want the settings download dir", opts.OutputDir)
	}
	if opts.CookiesBrowser != "chrome" {
		t.Errorf("engine CookiesBrowser = %q, want chrome (from settings)", opts.CookiesBrowser)
	}

	if h.rec.countNamed("job:added") != 1 {
		t.Errorf("job:added emitted %d times, want 1", h.rec.countNamed("job:added"))
	}
}

func TestStartDownloadRejectsEmptyURL(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	if _, err := h.svc.StartDownload(DownloadRequest{URL: "  "}); err == nil {
		t.Error("StartDownload with an empty URL = nil, want an error")
	}
}

func TestStartDownloadEmitsAddedEventWithJob(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "https://example.com/v", Title: "My Video", Mode: "audio"})
	if err != nil {
		t.Fatal(err)
	}

	ev, ok := h.rec.lastNamed("job:added")
	if !ok {
		t.Fatal("no job:added event emitted")
	}
	job, ok := ev.data[0].(queue.Job)
	if !ok {
		t.Fatalf("job:added payload = %T, want queue.Job", ev.data[0])
	}
	if job.ID != id || job.Title != "My Video" || job.Mode != "audio" {
		t.Errorf("payload = %+v, want the new job", job)
	}
}

func TestJobControlsDelegate(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "job did not start")

	if err := h.svc.PauseJob(id); err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	if !h.engine.handle(0).paused {
		t.Error("engine handle was not paused")
	}
	if err := h.svc.ResumeJob(id); err != nil {
		t.Fatalf("ResumeJob: %v", err)
	}
	if !h.engine.handle(0).resumed {
		t.Error("engine handle was not resumed")
	}
	if err := h.svc.CancelJob(id); err != nil {
		t.Fatalf("CancelJob: %v", err)
	}
	if !h.engine.handle(0).canceled {
		t.Error("engine handle was not canceled")
	}
}

func TestRemoveJobDelegates(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "job did not start")

	// Removing a running job is refused.
	if err := h.svc.RemoveJob(id); err == nil {
		t.Error("RemoveJob on a running job = nil, want an error")
	}

	h.engine.handle(0).finish(nil)
	waitFor(t, func() bool {
		jobs := h.svc.ListJobs()
		return len(jobs) == 1 && jobs[0].State == queue.StateCompleted
	}, "job did not complete")

	if err := h.svc.RemoveJob(id); err != nil {
		t.Fatalf("RemoveJob on a completed job = %v, want nil", err)
	}
	if len(h.svc.ListJobs()) != 0 {
		t.Error("job still listed after RemoveJob")
	}
}

func TestListJobsReturnsQueueContents(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	for i := 0; i < 3; i++ {
		if _, err := h.svc.StartDownload(DownloadRequest{URL: "u"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(h.svc.ListJobs()); got != 3 {
		t.Errorf("ListJobs() = %d, want 3", got)
	}
}

// ---------------------------------------------------------------------------
// Event bridge
// ---------------------------------------------------------------------------

func TestEventBridgeMapsChangeTypes(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "job did not start")

	// Progress -> job:updated with the job.
	h.svc.onProgressFor(id)(ytdlp.Progress{Percent: 50, TotalBytes: 1000, DownloadedBytes: 500, SpeedBps: 10, ETASec: 3})
	ev, ok := h.rec.lastNamed("job:updated")
	if !ok {
		t.Fatal("no job:updated emitted for progress")
	}
	job := ev.data[0].(queue.Job)
	if job.Percent != 50 || job.TotalBytes != 1000 {
		t.Errorf("job:updated payload = %+v, want the progress applied", job)
	}

	// Completion -> job:updated with state completed.
	h.engine.handle(0).finish(nil)
	waitFor(t, func() bool {
		jobs := h.svc.ListJobs()
		return len(jobs) == 1 && jobs[0].State == queue.StateCompleted
	}, "job did not complete")
	ev, _ = h.rec.lastNamed("job:updated")
	if ev.data[0].(queue.Job).State != queue.StateCompleted {
		t.Errorf("last job:updated state = %s, want completed", ev.data[0].(queue.Job).State)
	}

	// Removal -> job:removed with the id.
	if err := h.svc.RemoveJob(id); err != nil {
		t.Fatal(err)
	}
	ev, ok = h.rec.lastNamed("job:removed")
	if !ok {
		t.Fatal("no job:removed emitted")
	}
	if ev.data[0].(string) != id {
		t.Errorf("job:removed payload = %v, want the job id", ev.data[0])
	}
}

func TestServiceUsableBeforeStartup(t *testing.T) {
	// NewService must hand out a usable context: calling a bound method before
	// startup() swaps in the Wails context must not panic with a nil context
	// (exec.CommandContext panics on nil). The fake engine ignores ctx, so assert
	// the field directly - that is what the real engine relies on.
	h := newHarness(t, settings.Defaults())

	if h.svc.ctx == nil {
		t.Fatal("Service.ctx is nil before startup(); a real engine call would panic")
	}
	if _, err := h.svc.GetSettings(); err != nil {
		t.Fatalf("GetSettings before startup: %v", err)
	}
	if _, err := h.svc.ProbeURL("https://example.com/v"); err != nil {
		t.Fatalf("ProbeURL before startup: %v", err)
	}
}

func TestProgressRoutedToCorrectJob(t *testing.T) {
	h := newHarness(t, settings.Defaults()) // MaxConcurrent: 2

	idA, err := h.svc.StartDownload(DownloadRequest{URL: "a"})
	if err != nil {
		t.Fatal(err)
	}
	idB, err := h.svc.StartDownload(DownloadRequest{URL: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if idA == idB {
		t.Fatalf("expected two distinct job ids, both = %q", idA)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 2 }, "both jobs did not start")

	// Two jobs run concurrently; feed different progress to each.
	h.svc.onProgressFor(idA)(ytdlp.Progress{Percent: 11, TotalBytes: 100})
	h.svc.onProgressFor(idB)(ytdlp.Progress{Percent: 77, TotalBytes: 200})

	// Each job:updated payload must carry its own id and its own numbers.
	// Walking every job:updated and keying by id proves no cross-talk.
	seen := map[string]float64{}
	for _, e := range h.rec.all() {
		if e.name != EventJobUpdated {
			continue
		}
		job := e.data[0].(queue.Job)
		seen[job.ID] = job.Percent
	}
	if seen[idA] != 11 {
		t.Errorf("job A percent = %v, want 11 (progress leaked between jobs?)", seen[idA])
	}
	if seen[idB] != 77 {
		t.Errorf("job B percent = %v, want 77 (progress leaked between jobs?)", seen[idB])
	}

	// Finishing one job must not disturb the other's state. Which handle maps to
	// which id depends on scheduler order, so assert order-independently:
	// exactly one job completes, the other stays non-terminal.
	h.engine.handle(0).finish(nil)
	waitFor(t, func() bool {
		jobs := h.svc.ListJobs()
		if len(jobs) != 2 {
			return false
		}
		completed := 0
		for _, j := range jobs {
			if j.State == queue.StateCompleted {
				completed++
			}
		}
		return completed == 1
	}, "finishing one job did not leave exactly one completed and one active")
}

func TestProgressThrottledPerJob(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "job did not start")

	progress := h.svc.onProgressFor(id)
	base := h.rec.countNamed("job:updated")

	// Rapid updates within the throttle window: only the first should emit.
	for i := 1; i <= 10; i++ {
		progress(ytdlp.Progress{Percent: float64(i)})
	}
	got := h.rec.countNamed("job:updated") - base
	if got != 1 {
		t.Errorf("emitted %d job:updated within the throttle window, want 1", got)
	}

	// Advance the clock past the window: the next update emits.
	h.now = h.now.Add(300 * time.Millisecond)
	progress(ytdlp.Progress{Percent: 99})
	if got := h.rec.countNamed("job:updated") - base; got != 2 {
		t.Errorf("emitted %d job:updated after the window, want 2", got)
	}
}

func TestTerminalStateAlwaysEmitsDespiteThrottle(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	id, err := h.svc.StartDownload(DownloadRequest{URL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.engine.startCount() == 1 }, "job did not start")

	progress := h.svc.onProgressFor(id)
	progress(ytdlp.Progress{Percent: 10}) // consumes the throttle window
	progress(ytdlp.Progress{Percent: 20}) // throttled away

	// Completing the job must always emit, even inside the window.
	h.engine.handle(0).finish(nil)
	waitFor(t, func() bool {
		jobs := h.svc.ListJobs()
		return len(jobs) == 1 && jobs[0].State == queue.StateCompleted
	}, "job did not complete")

	ev, _ := h.rec.lastNamed("job:updated")
	if ev.data[0].(queue.Job).State != queue.StateCompleted {
		t.Error("the completion update was throttled away; terminal states must always emit")
	}
}

// ---------------------------------------------------------------------------
// App info, disk, shell
// ---------------------------------------------------------------------------

func TestGetAppInfoReportsConfigPathAndVersions(t *testing.T) {
	h := newHarness(t, settings.Defaults())

	info := h.svc.GetAppInfo()
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", info.Version)
	}
	if info.ConfigPath != `C:\app\config.json` {
		t.Errorf("ConfigPath = %q, want the active store path", info.ConfigPath)
	}
	if info.YtDlpVersion != "2026.08.19" {
		t.Errorf("YtDlpVersion = %q, want 2026.08.19", info.YtDlpVersion)
	}
	if info.FFmpegVersion != "9.0" {
		t.Errorf("FFmpegVersion = %q, want 9.0", info.FFmpegVersion)
	}
}

func TestGetDiskFreeDelegates(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	free, err := h.svc.GetDiskFree(`E:\Media`)
	if err != nil {
		t.Fatal(err)
	}
	if free != 123456789 {
		t.Errorf("free = %d, want 123456789", free)
	}
}

func TestOpenDownloadDirUsesSettingsDir(t *testing.T) {
	cfg := settings.Defaults()
	cfg.DownloadDir = `E:\Media`
	h := newHarness(t, cfg)

	var opened string
	h.svc.openPath = func(p string) error { opened = p; return nil }

	if err := h.svc.OpenDownloadDir(); err != nil {
		t.Fatal(err)
	}
	if opened != `E:\Media` {
		t.Errorf("opened %q, want the settings download dir", opened)
	}
}

func TestOpenFileDelegates(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	var opened string
	h.svc.openPath = func(p string) error { opened = p; return nil }

	if err := h.svc.OpenFile(`D:\D\clip.mp4`); err != nil {
		t.Fatal(err)
	}
	if opened != `D:\D\clip.mp4` {
		t.Errorf("opened %q, want the given path", opened)
	}
}

// ---------------------------------------------------------------------------
// Startup / restore
// ---------------------------------------------------------------------------

func TestRestoreDoesNotAutoStart(t *testing.T) {
	h := newHarness(t, settings.Defaults())
	if err := h.svc.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if h.engine.startCount() != 0 {
		t.Errorf("engine started %d jobs on restore, want 0", h.engine.startCount())
	}
}
