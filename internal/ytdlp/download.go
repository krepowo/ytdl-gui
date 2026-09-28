package ytdlp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Mode selects what a download produces.
type Mode string

const (
	// ModeVideo downloads a video (merging a separate audio stream when needed).
	ModeVideo Mode = "video"
	// ModeAudio extracts audio only as MP3 via ffmpeg.
	ModeAudio Mode = "audio"
)

// DownloadOptions describes one download request. It is site-agnostic: the same
// options work for YouTube, Twitter, or a generic page.
type DownloadOptions struct {
	URL       string
	OutputDir string
	Mode      Mode
	// FormatID is the yt-dlp format chosen in the picker (may be empty).
	FormatID string
	// NeedsMerge marks a video-only stream that must be merged with best audio.
	NeedsMerge bool
	// CookiesBrowser, when set, is passed as --cookies-from-browser.
	CookiesBrowser string
	// FilenameTemplate overrides the default "%(title)s.%(ext)s" output template.
	FilenameTemplate string
}

// JobState is the lifecycle state of a download job. The string values are the
// wire format used by the frontend, so they must stay stable.
type JobState string

const (
	StateQueued    JobState = "queued"
	StateRunning   JobState = "downloading"
	StatePaused    JobState = "paused"
	StateCompleted JobState = "completed"
	StateFailed    JobState = "error"
	StateCanceled  JobState = "canceled"
)

// isTerminal reports whether a state ends the job's life.
func isTerminal(s JobState) bool {
	return s == StateCompleted || s == StateFailed || s == StateCanceled
}

// JobCallbacks receive lifecycle updates. Both are optional and are invoked from
// the job's own goroutine, never while a lock is held.
type JobCallbacks struct {
	OnProgress func(Progress)
	OnState    func(JobState)
}

// defaultThrottle caps progress callbacks at 4/second (see the engine spec).
const defaultThrottle = 250 * time.Millisecond

// child is one spawned download process. suspend/resume implement a true pause;
// kill terminates the whole process tree (yt-dlp + any ffmpeg child).
type child interface {
	stdout() io.ReadCloser
	wait() error
	suspend() error
	resume() error
	kill() error
	close()
}

// spawner starts a child process. It is a seam so tests can drive the lifecycle
// with a fake process instead of a real binary.
type spawner interface {
	spawn(ctx context.Context, name string, args []string) (child, error)
}

// realSpawner is the production spawner backed by the OS.
type realSpawner struct{}

func (realSpawner) spawn(ctx context.Context, name string, args []string) (child, error) {
	return spawnProcess(ctx, name, args)
}

// Job is a single running (or finished) download. All methods are safe for
// concurrent use.
type Job struct {
	opts     DownloadOptions
	bins     Bins
	sp       spawner
	ctx      context.Context
	cancel   context.CancelFunc
	cb       JobCallbacks
	throttle time.Duration

	mu   sync.Mutex
	done chan struct{}

	state        JobState
	err          error
	outputPath   string
	child        child
	errLines     []string
	hasEmitted   bool
	lastEmit     time.Time
	suspended    bool
	pausedByKill bool
	// expectExit is the child whose non-zero exit must be ignored because we
	// killed it deliberately (pause fallback). It is per-child so a race with
	// Resume cannot misattribute the exit as a real failure.
	expectExit child
}

// StartJob begins a download using the real OS spawner.
func StartJob(ctx context.Context, bins Bins, opts DownloadOptions, cb JobCallbacks) (*Job, error) {
	return startJobWith(ctx, realSpawner{}, bins, opts, defaultThrottle, cb)
}

// startJobWith begins a download with an injected spawner and throttle interval
// (0 = no throttling). A spawn failure is reported through the job's state, not
// as a returned error, so callers always get a job they can observe.
func startJobWith(ctx context.Context, sp spawner, bins Bins, opts DownloadOptions, throttle time.Duration, cb JobCallbacks) (*Job, error) {
	if strings.TrimSpace(opts.URL) == "" {
		return nil, errors.New("ytdlp: URL is required")
	}

	ctx, cancel := context.WithCancel(ctx)
	j := &Job{
		opts:     opts,
		bins:     bins,
		sp:       sp,
		ctx:      ctx,
		cancel:   cancel,
		cb:       cb,
		throttle: throttle,
		state:    StateQueued,
		done:     make(chan struct{}),
	}

	if err := j.launch(); err != nil {
		j.mu.Lock()
		j.err = err
		notify := j.setStateLocked(StateFailed)
		j.mu.Unlock()
		if notify != nil {
			notify(StateFailed)
		}
		return j, nil
	}
	return j, nil
}

// launch spawns the process and starts streaming its output. It assumes the
// caller has already validated the options.
func (j *Job) launch() error {
	args := buildArgs(j.opts, j.bins)
	c, err := j.sp.spawn(j.ctx, j.bins.YtDlp, args)
	if err != nil {
		return err
	}

	j.mu.Lock()
	j.child = c
	j.mu.Unlock()

	j.setState(StateRunning)
	go j.pump(c)
	return nil
}

// pump reads the process output line by line until it exits, then records the
// terminal state. It is the only goroutine that owns the process's output.
func (j *Job) pump(c child) {
	defer c.close()

	scanner := bufio.NewScanner(c.stdout())
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		j.consume(scanner.Text())
	}

	j.finish(c, c.wait())
}

// consume routes one output line to the right handler.
func (j *Job) consume(line string) {
	if p, ok := parseProgressLine(line); ok {
		j.emitProgress(p)
		return
	}
	if dest, ok := parseDestination(line); ok {
		j.mu.Lock()
		j.outputPath = dest
		j.mu.Unlock()
		return
	}

	// Keep the tail of error output so a failure can explain itself.
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "ERROR") || strings.Contains(trimmed, "ERROR:") {
		j.mu.Lock()
		j.errLines = append(j.errLines, trimmed)
		if len(j.errLines) > 5 {
			j.errLines = j.errLines[len(j.errLines)-5:]
		}
		j.mu.Unlock()
	}
}

// emitProgress delivers a progress update, throttled to j.throttle but always
// letting the first update and the final 100% through.
func (j *Job) emitProgress(p Progress) {
	j.mu.Lock()
	now := time.Now()
	if j.hasEmitted && p.Percent < 100 && now.Sub(j.lastEmit) < j.throttle {
		j.mu.Unlock()
		return
	}
	j.hasEmitted = true
	j.lastEmit = now
	cb := j.cb.OnProgress
	j.mu.Unlock()

	if cb != nil {
		cb(p)
	}
}

// finish records the terminal state after a process exits. A pause fallback
// (kill + restart) exits the process deliberately, so that specific child's exit
// is ignored rather than reported as a failure.
func (j *Job) finish(c child, exitErr error) {
	j.mu.Lock()
	if j.expectExit != nil && j.expectExit == c {
		j.expectExit = nil
		j.mu.Unlock()
		return
	}
	if isTerminal(j.state) {
		j.mu.Unlock()
		return
	}

	next := StateCompleted
	if exitErr != nil {
		j.err = j.decorateErrorLocked(exitErr)
		next = StateFailed
	}
	notify := j.setStateLocked(next)
	j.mu.Unlock()

	if notify != nil {
		notify(next)
	}
}

// decorateErrorLocked attaches the captured yt-dlp error lines to err.
func (j *Job) decorateErrorLocked(err error) error {
	if len(j.errLines) == 0 {
		return err
	}
	return fmt.Errorf("%w: %s", err, strings.Join(j.errLines, "; "))
}

// setState transitions under the lock and returns the OnState callback to run
// after the lock is released (nil if nothing changed).
func (j *Job) setStateLocked(s JobState) func(JobState) {
	if j.state == s {
		return nil
	}
	j.state = s
	if isTerminal(s) {
		select {
		case <-j.done:
		default:
			close(j.done)
		}
	}
	return j.cb.OnState
}

// setState transitions and notifies, taking the lock itself.
func (j *Job) setState(s JobState) {
	j.mu.Lock()
	notify := j.setStateLocked(s)
	j.mu.Unlock()
	if notify != nil {
		notify(s)
	}
}

// State returns the current lifecycle state.
func (j *Job) State() JobState {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state
}

// Err returns the failure reason, or nil if the job has not failed.
func (j *Job) Err() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.err
}

// OutputPath returns the file path yt-dlp reported, or "" if unknown.
func (j *Job) OutputPath() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.outputPath
}

// Wait blocks until the job reaches a terminal state and returns it.
func (j *Job) Wait() JobState {
	<-j.done
	return j.State()
}

// Pause stops the download. It first tries to suspend the process (a true pause);
// if that is unsupported it kills the process and marks the job for a
// --continue restart on Resume.
func (j *Job) Pause() error {
	j.mu.Lock()
	if j.state != StateRunning {
		state := j.state
		j.mu.Unlock()
		return fmt.Errorf("ytdlp: cannot pause a job in state %q", state)
	}
	c := j.child
	j.mu.Unlock()

	if c == nil {
		return errors.New("ytdlp: no process to pause")
	}

	if err := c.suspend(); err == nil {
		j.mu.Lock()
		j.suspended = true
		notify := j.setStateLocked(StatePaused)
		j.mu.Unlock()
		if notify != nil {
			notify(StatePaused)
		}
		return nil
	}

	// Fallback: kill now; Resume relaunches so yt-dlp continues from the .part.
	// Record this exact child so its deliberate exit is not reported as an error.
	j.mu.Lock()
	j.expectExit = c
	j.pausedByKill = true
	notify := j.setStateLocked(StatePaused)
	j.mu.Unlock()
	if notify != nil {
		notify(StatePaused)
	}
	_ = c.kill()
	return nil
}

// Resume continues a paused job, either by resuming the suspended process or by
// relaunching it (which yt-dlp continues from the partial file).
func (j *Job) Resume() error {
	j.mu.Lock()
	if j.state != StatePaused {
		state := j.state
		j.mu.Unlock()
		return fmt.Errorf("ytdlp: cannot resume a job in state %q", state)
	}
	c := j.child
	suspended := j.suspended
	restart := j.pausedByKill
	j.pausedByKill = false
	j.suspended = false
	j.mu.Unlock()

	if restart {
		if err := j.launch(); err != nil {
			j.mu.Lock()
			j.err = err
			notify := j.setStateLocked(StateFailed)
			j.mu.Unlock()
			if notify != nil {
				notify(StateFailed)
			}
			return err
		}
		return nil
	}

	if c != nil && suspended {
		if err := c.resume(); err != nil {
			return err
		}
	}

	j.mu.Lock()
	notify := j.setStateLocked(StateRunning)
	j.mu.Unlock()
	if notify != nil {
		notify(StateRunning)
	}
	return nil
}

// Cancel terminates the job and its whole process tree. It is idempotent and
// returns nil for an already-finished job.
func (j *Job) Cancel() error {
	j.mu.Lock()
	if isTerminal(j.state) {
		j.mu.Unlock()
		return nil
	}
	notify := j.setStateLocked(StateCanceled)
	c := j.child
	j.mu.Unlock()

	if notify != nil {
		notify(StateCanceled)
	}
	if c != nil {
		_ = c.kill()
	}
	return nil
}

// buildArgs translates options into the yt-dlp argv. Arguments are always a
// slice (never a shell string) so user input cannot be interpreted by a shell.
func buildArgs(opts DownloadOptions, bins Bins) []string {
	args := []string{"--newline", "--no-warnings", "--no-playlist", "--progress"}
	args = append(args, "--ffmpeg-location", bins.Dir())

	if opts.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesBrowser)
	}

	switch opts.Mode {
	case ModeAudio:
		args = append(args, "-x", "--audio-format", "mp3")
		if opts.FormatID != "" {
			args = append(args, "-f", opts.FormatID)
		} else {
			args = append(args, "-f", "bestaudio/best")
		}
	default: // ModeVideo
		if opts.FormatID != "" {
			format := opts.FormatID
			if opts.NeedsMerge {
				// A video-only stream must be paired with the best audio.
				format += "+bestaudio"
			}
			args = append(args, "-f", format)
			if opts.NeedsMerge {
				args = append(args, "--merge-output-format", "mp4")
			}
		}
	}

	template := opts.FilenameTemplate
	if template == "" {
		template = "%(title)s.%(ext)s"
	}
	args = append(args, "-o", filepath.Join(opts.OutputDir, template))
	args = append(args, opts.URL)
	return args
}
