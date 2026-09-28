package queue

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// historyOf loads history from a queue's bound History, failing on error.
func historyOf(t *testing.T, h *History) []Job {
	t.Helper()
	jobs, err := h.Load()
	if err != nil {
		t.Fatalf("history Load: %v", err)
	}
	return jobs
}

// waitHistoryCount waits until history holds exactly n jobs. It tolerates the
// transient Windows sharing violation that can occur if a read races a
// concurrent atomic rename while a save is in flight.
func waitHistoryCount(t *testing.T, h *History, n int) []Job {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last []Job
	for time.Now().Before(deadline) {
		jobs, err := h.Load()
		if err == nil {
			last = jobs
			if len(jobs) == n {
				return jobs
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("history = %d jobs, want %d (last=%+v)", len(last), n, last)
	return last
}

func TestQueuePersistsCompletedJobToHistory(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "history.json"))
	e := &fakeEngine{}

	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now, History: h})
	a, _ := q.Add(DownloadRequest{URL: "a", Title: "Alpha"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")
	e.handle(0).finish(nil)
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "job did not complete")

	// History is written asynchronously; wait for it to appear.
	got := waitHistoryCount(t, h, 1)

	if got[0].ID != a.ID || got[0].State != StateCompleted {
		t.Errorf("history = %+v, want the completed job", got[0])
	}
	if got[0].Title != "Alpha" {
		t.Errorf("history title = %q, want Alpha", got[0].Title)
	}
}

func TestQueuePersistsFailedJobToHistory(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "history.json"))
	e := &fakeEngine{}

	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now, History: h})
	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")
	e.handle(0).finish(errors.New("network down"))
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateError }, "job did not fail")

	waitHistoryCount(t, h, 1)
	got := historyOf(t, h)
	if got[0].State != StateError || got[0].Error != "network down" {
		t.Errorf("history = %+v, want the failed job with its error", got[0])
	}
}

func TestQueuePersistsCanceledJobToHistory(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "history.json"))
	e := &fakeEngine{}

	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now, History: h})
	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	if err := q.Cancel(a.ID); err != nil {
		t.Fatal(err)
	}
	waitHistoryCount(t, h, 1)
	got := historyOf(t, h)
	if got[0].State != StateCanceled {
		t.Errorf("history state = %s, want canceled", got[0].State)
	}
}

func TestQueueDoesNotPersistRunningJob(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "history.json"))
	e := &fakeEngine{}

	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now, History: h})
	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")

	// While downloading, history must be empty.
	time.Sleep(30 * time.Millisecond)
	if got, err := h.Load(); err == nil && len(got) != 0 {
		t.Errorf("history has %d entries while a job is running, want 0", len(got))
	}
	if got := stateOf(t, q, a.ID); got != StateDownloading {
		t.Errorf("job state = %s, want downloading", got)
	}
}

func TestQueueRestoreHistoryAsPaused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Seed history with a mix, including an interrupted (downloading) job.
	seed := NewHistory(path)
	if err := seed.Save([]Job{
		{ID: "done", URL: "u1", State: StateCompleted, CreatedAt: 1},
		{ID: "interrupted", URL: "u2", State: StateDownloading, CreatedAt: 2},
	}); err != nil {
		t.Fatal(err)
	}

	e := &fakeEngine{}
	h := NewHistory(path)
	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now, History: h})

	if err := q.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	list := q.List()
	if len(list) != 2 {
		t.Fatalf("List() = %d jobs, want 2 restored", len(list))
	}

	byID := map[string]Job{}
	for _, j := range list {
		byID[j.ID] = j
	}
	if byID["done"].State != StateCompleted {
		t.Errorf("restored completed job = %s, want completed", byID["done"].State)
	}
	if byID["interrupted"].State != StatePaused {
		t.Errorf("restored interrupted job = %s, want paused", byID["interrupted"].State)
	}

	// Crucially, restoring must NOT auto-start anything.
	time.Sleep(50 * time.Millisecond)
	if e.count() != 0 {
		t.Errorf("engine started %d jobs after Restore, want 0 (nothing auto-starts)", e.count())
	}
}

func TestQueueWorksWithoutHistory(t *testing.T) {
	// History is optional; a nil History must not panic.
	e := &fakeEngine{}
	q := New(Options{Engine: e, MaxConcurrent: 1, Now: time.Now})

	a, _ := q.Add(DownloadRequest{URL: "a"})
	waitFor(t, func() bool { return e.count() == 1 }, "job did not start")
	e.handle(0).finish(nil)
	waitFor(t, func() bool { return stateOf(t, q, a.ID) == StateCompleted }, "job did not complete")

	if err := q.Restore(); err != nil {
		t.Errorf("Restore() with no history = %v, want nil", err)
	}
}
