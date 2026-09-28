package queue

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Fake engine: lets tests drive job lifecycles deterministically (no real
// processes) while still exercising the real scheduler.
// ---------------------------------------------------------------------------

// fakeHandle is one running engine job. The test controls when it finishes.
type fakeHandle struct {
	url string

	mu         sync.Mutex
	paused     bool
	resumed    bool
	canceled   bool
	outputPath string
	done       chan struct{}
	err        error
}

func (h *fakeHandle) Wait() error {
	<-h.done
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

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
	h.finish(errors.New("canceled"))
	return nil
}

func (h *fakeHandle) OutputPath() string { return h.outputPath }

// finish ends the engine job; err != nil marks it failed.
func (h *fakeHandle) finish(err error) {
	h.mu.Lock()
	h.err = err
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	h.mu.Unlock()
}

func (h *fakeHandle) wasPaused() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.paused
}

func (h *fakeHandle) wasResumed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.resumed
}

func (h *fakeHandle) wasCanceled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.canceled
}

type fakeEngine struct {
	mu       sync.Mutex
	started  []*fakeHandle
	startErr error
	// onStart, when set, runs after a handle is created (e.g. to auto-finish).
	onStart func(*fakeHandle)
}

func (e *fakeEngine) Start(jobID string, req DownloadRequest) (EngineJob, error) {
	e.mu.Lock()
	if e.startErr != nil {
		err := e.startErr
		e.mu.Unlock()
		return nil, err
	}
	h := &fakeHandle{url: req.URL, done: make(chan struct{})}
	e.started = append(e.started, h)
	cb := e.onStart
	e.mu.Unlock()

	if cb != nil {
		cb(h)
	}
	return h, nil
}

func (e *fakeEngine) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.started)
}

func (e *fakeEngine) handle(i int) *fakeHandle {
	e.mu.Lock()
	defer e.mu.Unlock()
	if i < 0 || i >= len(e.started) {
		return nil
	}
	return e.started[i]
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestQueue(engine Engine, maxConcurrent int) *Queue {
	return New(Options{Engine: engine, MaxConcurrent: maxConcurrent, Now: time.Now})
}

// waitFor polls cond until true or the deadline passes.
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

func stateOf(t *testing.T, q *Queue, id string) State {
	t.Helper()
	j, ok := q.Get(id)
	if !ok {
		t.Fatalf("job %s not found", id)
	}
	return j.State
}

// ---------------------------------------------------------------------------
// Scheduler
// ---------------------------------------------------------------------------

func TestQueueStartsUpToMaxConcurrent(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 2)

	var ids []string
	for i := 0; i < 5; i++ {
		j, err := q.Add(DownloadRequest{URL: "u"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, j.ID)
	}

	// Only two engine processes may be running; the rest stay queued.
	waitFor(t, func() bool { return e.count() == 2 }, "engine did not start exactly 2 jobs")

	if got := stateOf(t, q, ids[0]); got != StateDownloading {
		t.Errorf("job0 state = %s, want downloading", got)
	}
	if got := stateOf(t, q, ids[1]); got != StateDownloading {
		t.Errorf("job1 state = %s, want downloading", got)
	}
	for _, id := range ids[2:] {
		if got := stateOf(t, q, id); got != StateQueued {
			t.Errorf("job state = %s, want queued", got)
		}
	}

	// Never exceed the limit at any moment.
	if n := q.Running(); n != 2 {
		t.Errorf("Running() = %d, want 2", n)
	}
}

func TestQueuePromotesNextJobOnCompletion(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	b, _ := q.Add(DownloadRequest{URL: "b"})

	waitFor(t, func() bool { return e.count() == 1 }, "first job did not start")
	if got := stateOf(t, q, a.ID); got != StateDownloading {
		t.Fatalf("a state = %s, want downloading", got)
	}
	if got := stateOf(t, q, b.ID); got != StateQueued {
		t.Fatalf("b state = %s, want queued", got)
	}

	// Finishing A must immediately start B.
	e.handle(0).finish(nil)

	waitFor(t, func() bool { return e.count() == 2 }, "next job was not promoted")
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "a did not complete")
	if got := stateOf(t, q, b.ID); got != StateDownloading {
		t.Errorf("b state = %s, want downloading after promotion", got)
	}
}

func TestQueueFreesSlotOnFailure(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	if _, err := q.Add(DownloadRequest{URL: "b"}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool { return e.count() == 1 }, "first job did not start")
	e.handle(0).finish(errors.New("boom"))

	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateError }, "a did not fail")
	waitFor(t, func() bool { return e.count() == 2 }, "a failed job did not free its slot")

	if got := stateOf(t, q, a.ID); got != StateError {
		t.Errorf("a state = %s, want error", got)
	}
}

func TestQueueCancelRunningFreesSlotAndStartsNext(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	if _, err := q.Add(DownloadRequest{URL: "b"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return e.count() == 1 }, "first job did not start")

	if err := q.Cancel(a.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got := stateOf(t, q, a.ID); got != StateCanceled {
		t.Errorf("a state = %s, want canceled", got)
	}
	if !e.handle(0).wasCanceled() {
		t.Error("engine handle was not canceled")
	}
	waitFor(t, func() bool { return e.count() == 2 }, "cancel did not free the slot")
}

func TestQueueCancelQueuedNeverStarts(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	b, _ := q.Add(DownloadRequest{URL: "b"})
	waitFor(t, func() bool { return e.count() == 1 }, "first job did not start")

	if err := q.Cancel(b.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got := stateOf(t, q, b.ID); got != StateCanceled {
		t.Errorf("b state = %s, want canceled", got)
	}

	// Finishing A must NOT start the canceled B.
	e.handle(0).finish(nil)
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "a did not complete")
	time.Sleep(50 * time.Millisecond)
	if e.count() != 1 {
		t.Errorf("engine started %d jobs, want 1 (canceled job must not run)", e.count())
	}
}

func TestQueuePauseAndResumeRunning(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	if err := q.Pause(a.ID); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if got := stateOf(t, q, a.ID); got != StatePaused {
		t.Errorf("state = %s, want paused", got)
	}
	if !e.handle(0).wasPaused() {
		t.Error("engine handle was not paused")
	}
	// A paused job still occupies its slot, so the count must not change.
	if n := q.Running(); n != 0 {
		t.Errorf("Running() = %d, want 0 while paused (paused is not running)", n)
	}

	if err := q.Resume(a.ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if got := stateOf(t, q, a.ID); got != StateDownloading {
		t.Errorf("state = %s, want downloading", got)
	}
	if !e.handle(0).wasResumed() {
		t.Error("engine handle was not resumed")
	}
}

func TestQueuePauseQueuedIsRejected(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	if _, err := q.Add(DownloadRequest{URL: "a"}); err != nil {
		t.Fatal(err)
	}
	b, _ := q.Add(DownloadRequest{URL: "b"})
	waitFor(t, func() bool { return e.count() == 1 }, "first job did not start")

	if err := q.Pause(b.ID); err == nil {
		t.Error("Pause on a queued job = nil, want an error")
	}
	if got := stateOf(t, q, b.ID); got != StateQueued {
		t.Errorf("b state = %s, want queued", got)
	}
}

func TestQueueStartErrorMarksJobFailedAndFreesSlot(t *testing.T) {
	e := &fakeEngine{startErr: errors.New("cannot spawn")}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateError }, "job did not become error")

	j, _ := q.Get(a.ID)
	if j.Error == "" {
		t.Error("Error field is empty, want the engine failure message")
	}
	if n := q.Running(); n != 0 {
		t.Errorf("Running() = %d, want 0 after a start failure", n)
	}
}

func TestQueueAddRejectsEmptyURL(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)
	if _, err := q.Add(DownloadRequest{URL: "  "}); err == nil {
		t.Error("Add with an empty URL = nil, want an error")
	}
}

func TestQueueGetUnknownJob(t *testing.T) {
	q := newTestQueue(&fakeEngine{}, 1)
	if _, ok := q.Get("nope"); ok {
		t.Error("Get(unknown) = ok, want false")
	}
}

func TestQueueListReturnsStableOrder(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	var ids []string
	for i := 0; i < 3; i++ {
		j, _ := q.Add(DownloadRequest{URL: "u"})
		ids = append(ids, j.ID)
	}

	list := q.List()
	if len(list) != 3 {
		t.Fatalf("List() = %d jobs, want 3", len(list))
	}
	for i, id := range ids {
		if list[i].ID != id {
			t.Errorf("List()[%d].ID = %s, want %s (insertion order)", i, list[i].ID, id)
		}
	}
}

func TestQueueRemoveTerminalJob(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	// Removing a running job is refused.
	if err := q.Remove(a.ID); err == nil {
		t.Error("Remove on a running job = nil, want an error")
	}

	e.handle(0).finish(nil)
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "job did not complete")

	if err := q.Remove(a.ID); err != nil {
		t.Fatalf("Remove on a completed job = %v, want nil", err)
	}
	if _, ok := q.Get(a.ID); ok {
		t.Error("job still present after Remove")
	}
}

func TestQueueUniqueIDs(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		j, _ := q.Add(DownloadRequest{URL: "u"})
		if seen[j.ID] {
			t.Fatalf("duplicate job ID %q", j.ID)
		}
		seen[j.ID] = true
	}
}

func TestQueueMaxConcurrentClampedToAtLeastOne(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 0) // invalid
	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateDownloading }, "job never started with MaxConcurrent=0")
}

func TestQueueUpdateProgress(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	q.UpdateProgress(a.ID, ProgressUpdate{
		Percent:         55.5,
		DownloadedBytes: 1000,
		TotalBytes:      2000,
		SpeedBps:        500,
		ETASec:          10,
	})

	j, _ := q.Get(a.ID)
	if j.Percent != 55.5 || j.DownloadedBytes != 1000 || j.TotalBytes != 2000 || j.SpeedBps != 500 || j.ETASec != 10 {
		t.Errorf("progress not applied: %+v", j)
	}
}

func TestQueueSetOutputPath(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 1)

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	q.SetOutputPath(a.ID, `D:\D\clip.mp4`)
	j, _ := q.Get(a.ID)
	if j.OutputPath != `D:\D\clip.mp4` {
		t.Errorf("OutputPath = %q, want the set path", j.OutputPath)
	}
}

func TestQueueEmitsChangeCallbacks(t *testing.T) {
	e := &fakeEngine{}
	var mu sync.Mutex
	var events []string

	q := New(Options{
		Engine:        e,
		MaxConcurrent: 1,
		Now:           time.Now,
		OnChange: func(ev ChangeEvent) {
			mu.Lock()
			events = append(events, string(ev.Type)+":"+string(ev.Job.State))
			mu.Unlock()
		},
	})

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")
	e.handle(0).finish(nil)
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "job did not complete")

	mu.Lock()
	defer mu.Unlock()
	if len(events) == 0 {
		t.Fatal("no change events emitted")
	}
	if events[len(events)-1] != "updated:completed" {
		t.Errorf("last event = %q, want updated:completed", events[len(events)-1])
	}
}

func TestQueueConcurrencyNeverExceedsLimitUnderLoad(t *testing.T) {
	e := &fakeEngine{}
	q := newTestQueue(e, 3)

	const n = 30
	var mu sync.Mutex
	peak := 0

	// Track the number of concurrently running engine jobs from the queue's view.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if r := q.Running(); r > 0 {
				mu.Lock()
				if r > peak {
					peak = r
				}
				mu.Unlock()
			}
			time.Sleep(time.Millisecond)
		}
	}()

	var ids []string
	for i := 0; i < n; i++ {
		j, err := q.Add(DownloadRequest{URL: "u"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, j.ID)
	}

	// Finish every job as it starts, in waves, so slots keep freeing.
	waitFor(t, func() bool { return e.count() >= 3 }, "jobs did not start")
	for i := 0; i < n; i++ {
		waitFor(t, func() bool { return e.handle(i) != nil }, "engine job did not start")
		e.handle(i).finish(nil)
	}

	for _, id := range ids {
		waitFor(t, func() bool { return stateOf(t, q, id) == StateCompleted }, "not all jobs completed")
	}
	close(stop)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if peak > 3 {
		t.Errorf("peak concurrent jobs = %d, want <= 3", peak)
	}
	if peak == 0 {
		t.Error("peak was never observed; the test did not exercise concurrency")
	}
}
