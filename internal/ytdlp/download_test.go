package ytdlp

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test doubles: a fake spawner/child lets us drive the whole job lifecycle
// (progress, pause, resume, cancel, failure) without spawning real processes.
// ---------------------------------------------------------------------------

type fakeChild struct {
	pr *io.PipeReader
	pw *io.PipeWriter

	mu         sync.Mutex
	exited     chan struct{}
	exitErr    error
	killed     bool
	suspended  bool
	resumed    bool
	suspendErr error
	resumeErr  error
	closed     bool
}

func newFakeChild() *fakeChild {
	pr, pw := io.Pipe()
	return &fakeChild{pr: pr, pw: pw, exited: make(chan struct{})}
}

func (c *fakeChild) stdout() io.ReadCloser { return c.pr }

// stderr is always empty for the fake; output is merged into stdout.
func (c *fakeChild) stderr() io.ReadCloser { return io.NopCloser(strings.NewReader("")) }

// emit writes a line as if yt-dlp printed it.
func (c *fakeChild) emit(line string) { _, _ = io.WriteString(c.pw, line+"\n") }

// finish ends the process with the given error (nil = clean exit).
func (c *fakeChild) finish(err error) {
	c.mu.Lock()
	if c.exitErr == nil && err != nil {
		c.exitErr = err
	}
	select {
	case <-c.exited:
	default:
		close(c.exited)
	}
	c.mu.Unlock()
	_ = c.pw.Close()
}

func (c *fakeChild) wait() error {
	<-c.exited
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.exitErr
}

func (c *fakeChild) suspend() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.suspendErr != nil {
		return c.suspendErr
	}
	c.suspended = true
	return nil
}

func (c *fakeChild) resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resumeErr != nil {
		return c.resumeErr
	}
	c.resumed = true
	c.suspended = false
	return nil
}

func (c *fakeChild) kill() error {
	c.mu.Lock()
	c.killed = true
	c.mu.Unlock()
	c.finish(errors.New("killed"))
	return nil
}

func (c *fakeChild) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	_ = c.pw.Close()
}

func (c *fakeChild) wasKilled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.killed
}

func (c *fakeChild) wasSuspended() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.suspended
}

func (c *fakeChild) wasResumed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resumed
}

// fakeSpawner records every child it hands out so tests can inspect the Nth
// process (important for the kill+restart pause fallback).
type fakeSpawner struct {
	mu   sync.Mutex
	made []*fakeChild
	err  error
	args [][]string
}

func (s *fakeSpawner) spawn(_ context.Context, _ string, args []string) (child, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	c := newFakeChild()
	s.made = append(s.made, c)
	s.args = append(s.args, args)
	return c, nil
}

func (s *fakeSpawner) child(i int) *fakeChild {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i < 0 || i >= len(s.made) {
		return nil
	}
	return s.made[i]
}

func (s *fakeSpawner) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.made)
}

func (s *fakeSpawner) lastArgs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.args) == 0 {
		return nil
	}
	return s.args[len(s.args)-1]
}

// collectProgress returns a callback plus a snapshot accessor.
func collectProgress() (func(Progress), func() []Progress) {
	var mu sync.Mutex
	var got []Progress
	return func(p Progress) {
			mu.Lock()
			got = append(got, p)
			mu.Unlock()
		}, func() []Progress {
			mu.Lock()
			defer mu.Unlock()
			out := make([]Progress, len(got))
			copy(out, got)
			return out
		}
}

// waitState polls until the job reaches a terminal state or the deadline hits.
func waitState(t *testing.T, j *Job, want JobState) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if j.State() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("state = %s, want %s (timed out)", j.State(), want)
}

// waitFor polls until cond is true or the deadline hits.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

// ---------------------------------------------------------------------------
// buildArgs: pure, table-driven
// ---------------------------------------------------------------------------

func testBins() Bins {
	return Bins{YtDlp: `C:\app\yt-dlp.exe`, FFmpeg: `C:\app\ffmpeg.exe`, FFprobe: `C:\app\ffprobe.exe`}
}

func containsSubslice(args []string, want ...string) bool {
	for i := 0; i+len(want) <= len(args); i++ {
		match := true
		for j := range want {
			if args[i+j] != want[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestBuildArgsVideoMerge(t *testing.T) {
	opts := DownloadOptions{
		URL:        "https://example.com/v",
		OutputDir:  `D:\Downloads`,
		Mode:       ModeVideo,
		FormatID:   "248",
		NeedsMerge: true,
	}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "-f", "248+bestaudio") {
		t.Errorf("args = %v, want -f 248+bestaudio (video-only stream must merge audio)", args)
	}
	if !containsSubslice(args, "--merge-output-format", "mp4") {
		t.Errorf("args = %v, want --merge-output-format mp4", args)
	}
	if !containsSubslice(args, "--ffmpeg-location", `C:\app`) {
		t.Errorf("args = %v, want --ffmpeg-location with the bins dir", args)
	}
	if !containsArg(args, "--newline") {
		t.Errorf("args = %v, want --newline so progress is line-based", args)
	}
	if !containsArg(args, "--no-playlist") {
		t.Errorf("args = %v, want --no-playlist (single item)", args)
	}
	if args[len(args)-1] != opts.URL {
		t.Errorf("last arg = %q, want the URL", args[len(args)-1])
	}
}

func TestBuildArgsVideoCombinedNoMerge(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo, FormatID: "mp4", NeedsMerge: false}
	args := buildArgs(opts, testBins())

	if containsSubslice(args, "-f", "mp4+bestaudio") {
		t.Errorf("args = %v, must NOT add +bestaudio for a combined stream", args)
	}
	if !containsSubslice(args, "-f", "mp4") {
		t.Errorf("args = %v, want -f mp4", args)
	}
}

func TestBuildArgsAudioMp3(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeAudio, FormatID: "251"}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "-x", "--audio-format", "mp3") {
		t.Errorf("args = %v, want -x --audio-format mp3", args)
	}
	if !containsSubslice(args, "-f", "251") {
		t.Errorf("args = %v, want -f 251", args)
	}
	if containsSubslice(args, "--merge-output-format", "mp4") {
		t.Errorf("args = %v, audio mode must not request a video merge", args)
	}
}

func TestBuildArgsAudioDefaultsToBestAudio(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeAudio}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "-f", "bestaudio/best") {
		t.Errorf("args = %v, want -f bestaudio/best when no format chosen", args)
	}
}

func TestBuildArgsCookiesFromBrowser(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo, CookiesBrowser: "chrome"}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "--cookies-from-browser", "chrome") {
		t.Errorf("args = %v, want --cookies-from-browser chrome", args)
	}
}

func TestBuildArgsOmitsCookiesWhenUnset(t *testing.T) {
	args := buildArgs(DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo}, testBins())
	if containsArg(args, "--cookies-from-browser") {
		t.Errorf("args = %v, must not pass --cookies-from-browser when unset", args)
	}
}

func TestBuildArgsOutputTemplate(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "-o", `D:\D\%(title)s.%(ext)s`) {
		t.Errorf("args = %v, want -o with the output dir joined to the default template", args)
	}
}

func TestBuildArgsCustomTemplate(t *testing.T) {
	opts := DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo, FilenameTemplate: "%(id)s.%(ext)s"}
	args := buildArgs(opts, testBins())

	if !containsSubslice(args, "-o", `D:\D\%(id)s.%(ext)s`) {
		t.Errorf("args = %v, want the custom template honoured", args)
	}
}

// ---------------------------------------------------------------------------
// Filename safety: illegal characters and duplicates
// ---------------------------------------------------------------------------

func TestBuildArgsSanitizesIllegalFilenameChars(t *testing.T) {
	args := buildArgs(DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo}, testBins())

	// yt-dlp must rewrite characters Windows forbids in a filename.
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--replace-in-metadata") {
		t.Fatalf("args = %v, want --replace-in-metadata to sanitize the title", args)
	}
	if !containsSubslice(args, "--windows-filenames") {
		t.Errorf("args = %v, want --windows-filenames for Windows-safe output", args)
	}
}

func TestBuildArgsSanitizeReplacesWithUnderscore(t *testing.T) {
	args := buildArgs(DownloadOptions{URL: "u", OutputDir: `D:\D`, Mode: ModeVideo}, testBins())

	// Find the replace-in-metadata triple and confirm the replacement is "_".
	for i := 0; i+3 < len(args); i++ {
		if args[i] == "--replace-in-metadata" && args[i+1] == "title" {
			if args[i+3] != "_" {
				t.Errorf("replacement = %q, want %q", args[i+3], "_")
			}
			return
		}
	}
	t.Errorf("args = %v, no --replace-in-metadata title <regex> _ triple found", args)
}

func TestBuildArgsIncrementsDuplicateOutputName(t *testing.T) {
	dir := t.TempDir()
	// A file already occupying the natural output name.
	if err := os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A literal base template whose .%(ext)s resolves to the existing name.
	opts := DownloadOptions{URL: "u", OutputDir: dir, Mode: ModeVideo, FilenameTemplate: "clip.%(ext)s"}
	args := buildArgs(opts, testBins())

	out := ""
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" {
			out = args[i+1]
		}
	}
	if out == "" {
		t.Fatalf("args = %v, no -o template found", args)
	}
	// The output must no longer collide with the existing file.
	if strings.HasSuffix(out, "clip.%(ext)s") {
		t.Errorf("-o = %q, want a suffixed name so it does not overwrite the existing clip", out)
	}
	if !strings.Contains(out, "clip+1") && !strings.Contains(out, "clip (1)") {
		t.Errorf("-o = %q, want an increment marker (clip+1 or clip (1))", out)
	}
}

func TestBuildArgsNoIncrementWhenFree(t *testing.T) {
	dir := t.TempDir()
	opts := DownloadOptions{URL: "u", OutputDir: dir, Mode: ModeVideo, FilenameTemplate: "clip.%(ext)s"}
	args := buildArgs(opts, testBins())

	out := ""
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" {
			out = args[i+1]
		}
	}
	if !strings.HasSuffix(out, "clip.%(ext)s") {
		t.Errorf("-o = %q, want the plain name when no duplicate exists", out)
	}
}

// ---------------------------------------------------------------------------
// Job lifecycle
// ---------------------------------------------------------------------------

func TestJobEmitsProgressAndCompletes(t *testing.T) {
	sp := &fakeSpawner{}
	onProgress, snapshot := collectProgress()

	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{OnProgress: onProgress})
	if err != nil {
		t.Fatal(err)
	}

	ch := sp.child(0)
	ch.emit(`[download]  10.0% of  100.00MiB at 1.00MiB/s ETA 00:10`)
	ch.emit(`[download]  50.0% of  100.00MiB at 2.00MiB/s ETA 00:05`)
	ch.emit(`[download] Destination: D:\D\clip.mp4`)
	ch.emit(`[download] 100% of  100.00MiB in 00:00:05 at 20.00MiB/s`)
	ch.finish(nil)

	waitState(t, j, StateCompleted)

	got := snapshot()
	if len(got) != 3 {
		t.Fatalf("progress callbacks = %d, want 3 (%+v)", len(got), got)
	}
	if got[2].Percent != 100 {
		t.Errorf("last percent = %v, want 100", got[2].Percent)
	}
	if j.OutputPath() != `D:\D\clip.mp4` {
		t.Errorf("OutputPath = %q, want the parsed destination", j.OutputPath())
	}
	if err := j.Err(); err != nil {
		t.Errorf("Err = %v, want nil", err)
	}
}

func TestJobThrottlesProgressButAlwaysEmitsFinal(t *testing.T) {
	sp := &fakeSpawner{}
	onProgress, snapshot := collectProgress()

	// A huge interval forces throttling of everything except the first and 100%.
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, time.Hour, JobCallbacks{OnProgress: onProgress})
	if err != nil {
		t.Fatal(err)
	}

	ch := sp.child(0)
	for i := 1; i <= 20; i++ {
		ch.emit(`[download]  ` + strconv.Itoa(i) + `.0% of  100.00MiB at 1.00MiB/s ETA 00:10`)
	}
	ch.emit(`[download] 100% of  100.00MiB in 00:00:10 at 10.00MiB/s`)
	ch.finish(nil)

	waitState(t, j, StateCompleted)

	got := snapshot()
	// First line passes (no previous emit) and the final 100% always passes;
	// everything in between is throttled away.
	if len(got) != 2 {
		t.Fatalf("progress callbacks = %d, want 2 (first + final 100%%): %+v", len(got), got)
	}
	if got[0].Percent != 1.0 {
		t.Errorf("first callback percent = %v, want 1", got[0].Percent)
	}
	if got[1].Percent != 100 {
		t.Errorf("last callback percent = %v, want 100", got[1].Percent)
	}
}

func TestJobFailsWithStderrTail(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}

	ch := sp.child(0)
	ch.emit(`ERROR: [generic] Unable to download webpage`)
	ch.finish(errors.New("exit status 1"))

	waitState(t, j, StateFailed)

	if j.Err() == nil {
		t.Fatal("Err = nil, want the failure reason")
	}
	if !strings.Contains(j.Err().Error(), "Unable to download webpage") {
		t.Errorf("Err = %v, want it to include yt-dlp's error output", j.Err())
	}
}

func TestJobCancelKillsProcess(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	ch := sp.child(0)
	ch.emit(`[download]  10.0% of  100.00MiB at 1.00MiB/s ETA 00:10`)

	if err := j.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	waitState(t, j, StateCanceled)

	if !ch.wasKilled() {
		t.Error("child was not killed on cancel")
	}
	if j.Err() != nil {
		t.Errorf("Err = %v, want nil for a user cancel", j.Err())
	}
}

func TestJobPauseSuspendsAndResumeContinues(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	ch := sp.child(0)
	ch.emit(`[download]  10.0% of  100.00MiB at 1.00MiB/s ETA 00:10`)

	if err := j.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if j.State() != StatePaused {
		t.Errorf("state = %s, want paused", j.State())
	}
	if !ch.wasSuspended() {
		t.Error("process was not suspended; want a true pause")
	}
	if ch.wasKilled() {
		t.Error("process was killed; suspend should have succeeded")
	}
	if sp.count() != 1 {
		t.Errorf("spawner made %d processes, want 1 (no restart when suspend works)", sp.count())
	}

	if err := j.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if j.State() != StateRunning {
		t.Errorf("state = %s, want running after resume", j.State())
	}
	if !ch.wasResumed() {
		t.Error("process was not resumed")
	}

	ch.emit(`[download] 100% of  100.00MiB in 00:00:10 at 10.00MiB/s`)
	ch.finish(nil)
	waitState(t, j, StateCompleted)
}

func TestJobPauseFallsBackToKillAndRestart(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}

	// Make suspend fail so the fallback (kill + --continue restart) is used.
	ch0 := sp.child(0)
	ch0.mu.Lock()
	ch0.suspendErr = errors.New("suspend unsupported")
	ch0.mu.Unlock()
	ch0.emit(`[download]  10.0% of  100.00MiB at 1.00MiB/s ETA 00:10`)

	if err := j.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if j.State() != StatePaused {
		t.Errorf("state = %s, want paused", j.State())
	}
	if !ch0.wasKilled() {
		t.Error("fallback pause must kill the process (so --continue can resume)")
	}

	// Resume must spawn a NEW process (the old one is dead).
	if err := j.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	waitFor(t, func() bool { return sp.count() == 2 }, "resume did not restart the download")

	if j.State() != StateRunning {
		t.Errorf("state = %s, want running after resume", j.State())
	}
	// The restarted process keeps the same args (so --continue resumes the .part).
	if a := sp.lastArgs(); !containsArg(a, "u") {
		t.Errorf("restart args = %v, want the same URL", a)
	}

	sp.child(1).finish(nil)
	waitState(t, j, StateCompleted)
}

func TestJobPauseIllegalWhenNotRunning(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	sp.child(0).finish(nil)
	waitState(t, j, StateCompleted)

	if err := j.Pause(); err == nil {
		t.Error("Pause() on a completed job = nil, want an error")
	}
	if err := j.Resume(); err == nil {
		t.Error("Resume() on a completed job = nil, want an error")
	}
}

func TestJobCancelIsIdempotent(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	sp.child(0).finish(nil)
	waitState(t, j, StateCompleted)

	if err := j.Cancel(); err != nil {
		t.Errorf("Cancel() on a finished job = %v, want nil (idempotent)", err)
	}
}

func TestJobEmptyURLRejected(t *testing.T) {
	sp := &fakeSpawner{}
	if _, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "  "}, 0, JobCallbacks{}); err == nil {
		t.Error("startJobWith() with an empty URL = nil, want an error")
	}
	if sp.count() != 0 {
		t.Error("no process should be spawned for an empty URL")
	}
}

func TestJobSpawnErrorSurfaced(t *testing.T) {
	sp := &fakeSpawner{err: errors.New("cannot start process")}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatalf("startJobWith() error = %v, want nil (failure is reported via the job)", err)
	}
	waitState(t, j, StateFailed)
	if j.Err() == nil || !strings.Contains(j.Err().Error(), "cannot start process") {
		t.Errorf("Err = %v, want it to mention the spawn failure", j.Err())
	}
}

func TestJobStateCallbackFires(t *testing.T) {
	sp := &fakeSpawner{}
	var mu sync.Mutex
	var states []JobState

	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{
		OnState: func(s JobState) { mu.Lock(); states = append(states, s); mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	sp.child(0).finish(nil)
	waitState(t, j, StateCompleted)

	mu.Lock()
	defer mu.Unlock()
	if len(states) == 0 || states[len(states)-1] != StateCompleted {
		t.Errorf("states = %v, want the last to be completed", states)
	}
}

func TestJobWaitReturnsTerminalState(t *testing.T) {
	sp := &fakeSpawner{}
	j, err := startJobWith(context.Background(), sp, testBins(), DownloadOptions{URL: "u", OutputDir: `D:\D`}, 0, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	sp.child(0).emit(`[download] 100% of  1.00MiB in 00:00:01 at 1.00MiB/s`)
	sp.child(0).finish(nil)

	done := make(chan JobState, 1)
	go func() { done <- j.Wait() }()

	select {
	case s := <-done:
		if s != StateCompleted {
			t.Errorf("Wait() = %s, want completed", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Wait() did not return")
	}
}
