package queue

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// EngineJob is one running download owned by the engine adapter. The ytdlp
// package is adapted to this interface in the app layer.
type EngineJob interface {
	// Wait blocks until the download finishes and returns its error, if any.
	Wait() error
	// OutputPath returns the file path reported by the engine ("" if unknown).
	OutputPath() string
	Pause() error
	Resume() error
	Cancel() error
}

// Engine starts downloads. It is an interface so the queue can be tested with a
// fake and so the engine implementation can change without touching the queue.
type Engine interface {
	Start(req DownloadRequest) (EngineJob, error)
}

// ChangeType classifies a change event for the frontend.
type ChangeType string

const (
	ChangeAdded   ChangeType = "added"
	ChangeUpdated ChangeType = "updated"
	ChangeRemoved ChangeType = "removed"
)

// ChangeEvent describes one queue mutation. The Job is a snapshot (safe to
// marshal) taken at the moment of the change.
type ChangeEvent struct {
	Type ChangeType
	Job  Job
}

// ProgressUpdate carries the live progress fields pushed from the engine.
type ProgressUpdate struct {
	Percent         float64
	DownloadedBytes int64
	TotalBytes      int64
	SpeedBps        int64
	ETASec          int
}

// Options configures a Queue.
type Options struct {
	Engine        Engine
	MaxConcurrent int
	// Now is injectable for deterministic tests; defaults to time.Now.
	Now func() time.Time
	// OnChange, when set, is called after every mutation (never while holding
	// the lock, so it may call back into the queue).
	OnChange func(ChangeEvent)
}

// Queue owns every job and schedules them within MaxConcurrent.
type Queue struct {
	engine   Engine
	max      int
	now      func() time.Time
	onChange func(ChangeEvent)

	mu    sync.Mutex
	jobs  []*Job // insertion order
	index map[string]*Job
	// active counts slots in use by started jobs (a paused job still holds one).
	active int
	seq    atomic.Uint64
}

// New creates a Queue. MaxConcurrent below 1 is clamped to 1 so the queue always
// makes progress.
func New(opts Options) *Queue {
	max := opts.MaxConcurrent
	if max < 1 {
		max = 1
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Queue{
		engine:   opts.Engine,
		max:      max,
		now:      now,
		onChange: opts.OnChange,
		index:    map[string]*Job{},
	}
}

// Add validates the request, creates a queued job, and schedules it.
func (q *Queue) Add(req DownloadRequest) (Job, error) {
	if strings.TrimSpace(req.URL) == "" {
		return Job{}, errors.New("queue: URL is required")
	}

	id := q.nextID()

	q.mu.Lock()
	j := newJob(id, req, q.now())
	q.jobs = append(q.jobs, j)
	q.index[id] = j
	snapshot := j.clone()
	q.pumpLocked()
	q.mu.Unlock()

	q.emit(ChangeEvent{Type: ChangeAdded, Job: snapshot})
	return snapshot, nil
}

// nextID returns a process-unique job id. A counter keeps ids unique even when
// many jobs are added within the same millisecond.
func (q *Queue) nextID() string {
	n := q.seq.Add(1)
	return fmt.Sprintf("job-%d-%d", q.now().UnixNano(), n)
}

// Get returns a snapshot of one job.
func (q *Queue) Get(id string) (Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	j, ok := q.index[id]
	if !ok {
		return Job{}, false
	}
	return j.clone(), true
}

// List returns snapshots of every job in insertion order.
func (q *Queue) List() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		out = append(out, j.clone())
	}
	return out
}

// Running returns how many jobs are actively downloading (paused jobs are not
// running, though they still occupy a slot).
func (q *Queue) Running() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, j := range q.jobs {
		if j.State == StateDownloading {
			n++
		}
	}
	return n
}

// UpdateProgress applies a live progress update from the engine.
func (q *Queue) UpdateProgress(id string, p ProgressUpdate) {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok || isTerminal(j.State) {
		q.mu.Unlock()
		return
	}
	j.Percent = p.Percent
	j.DownloadedBytes = p.DownloadedBytes
	j.TotalBytes = p.TotalBytes
	j.SpeedBps = p.SpeedBps
	j.ETASec = p.ETASec
	snapshot := j.clone()
	q.mu.Unlock()

	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
}

// SetOutputPath records where the engine reported the finished file will land.
func (q *Queue) SetOutputPath(id, path string) {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return
	}
	j.OutputPath = path
	snapshot := j.clone()
	q.mu.Unlock()

	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
}

// Pause pauses a running job. A queued or finished job cannot be paused.
func (q *Queue) Pause(id string) error {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("queue: job %s not found", id)
	}
	if j.State != StateDownloading {
		state := j.State
		q.mu.Unlock()
		return fmt.Errorf("queue: cannot pause a job in state %q", state)
	}
	handle, _ := j.handle.(EngineJob)
	if err := j.setState(StatePaused); err != nil {
		q.mu.Unlock()
		return err
	}
	snapshot := j.clone()
	q.mu.Unlock()

	if handle != nil {
		if err := handle.Pause(); err != nil {
			return err
		}
	}
	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
	return nil
}

// Resume resumes a paused job back to downloading.
func (q *Queue) Resume(id string) error {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("queue: job %s not found", id)
	}
	if j.State != StatePaused {
		state := j.State
		q.mu.Unlock()
		return fmt.Errorf("queue: cannot resume a job in state %q", state)
	}
	handle, _ := j.handle.(EngineJob)
	if err := j.setState(StateDownloading); err != nil {
		q.mu.Unlock()
		return err
	}
	snapshot := j.clone()
	q.mu.Unlock()

	if handle != nil {
		if err := handle.Resume(); err != nil {
			return err
		}
	}
	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
	return nil
}

// Cancel cancels a job. A queued job is canceled immediately (it never starts);
// a running job is told to stop and its slot is released once the engine exits.
// Canceling an already-finished job is a no-op.
func (q *Queue) Cancel(id string) error {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("queue: job %s not found", id)
	}
	if isTerminal(j.State) {
		q.mu.Unlock()
		return nil
	}

	wasQueued := j.State == StateQueued
	handle, _ := j.handle.(EngineJob)
	if err := j.setState(StateCanceled); err != nil {
		q.mu.Unlock()
		return err
	}
	snapshot := j.clone()
	q.mu.Unlock()

	if handle != nil {
		_ = handle.Cancel()
	}
	if wasQueued {
		// No watcher owns this job, so release/schedule here.
		q.mu.Lock()
		q.pumpLocked()
		q.mu.Unlock()
	}
	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
	return nil
}

// Remove deletes a finished job from the list. A job that is still running or
// queued must be canceled first.
func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("queue: job %s not found", id)
	}
	if !isTerminal(j.State) {
		state := j.State
		q.mu.Unlock()
		return fmt.Errorf("queue: cannot remove a job in state %q", state)
	}
	delete(q.index, id)
	for i, item := range q.jobs {
		if item == j {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			break
		}
	}
	snapshot := j.clone()
	q.mu.Unlock()

	q.emit(ChangeEvent{Type: ChangeRemoved, Job: snapshot})
	return nil
}

// pumpLocked starts queued jobs while slots are free. Callers must hold the lock.
func (q *Queue) pumpLocked() {
	for q.active < q.max {
		j := q.nextQueuedLocked()
		if j == nil {
			return
		}
		if err := j.setState(StateDownloading); err != nil {
			return
		}
		q.active++
		go q.run(j)
	}
}

// nextQueuedLocked returns the oldest queued job, or nil.
func (q *Queue) nextQueuedLocked() *Job {
	for _, j := range q.jobs {
		if j.State == StateQueued {
			return j
		}
	}
	return nil
}

// run starts the engine job and waits for it. It is the ONLY place a slot is
// released for a started job, which guarantees exactly-once release.
func (q *Queue) run(j *Job) {
	handle, err := q.engine.Start(j.req)
	if err != nil {
		q.settle(j.ID, nil, err)
		return
	}

	q.mu.Lock()
	if cur, ok := q.index[j.ID]; ok {
		cur.handle = handle
	}
	q.mu.Unlock()

	waitErr := handle.Wait()
	q.settle(j.ID, handle, waitErr)
}

// settle applies a job's outcome, frees its slot, and schedules the next job. A
// job already moved to a terminal state (e.g. canceled) keeps that state.
func (q *Queue) settle(id string, handle EngineJob, err error) {
	q.mu.Lock()
	j, ok := q.index[id]
	if !ok {
		q.mu.Unlock()
		return
	}

	// The slot for a started job is released here, exactly once.
	if q.active > 0 {
		q.active--
	}

	if !isTerminal(j.State) {
		if err != nil {
			j.State = StateError
			j.Error = err.Error()
		} else {
			j.State = StateCompleted
			j.Percent = 100
			if handle != nil {
				if p := handle.OutputPath(); p != "" {
					j.OutputPath = p
				}
			}
		}
	}
	snapshot := j.clone()
	q.pumpLocked()
	q.mu.Unlock()

	q.emit(ChangeEvent{Type: ChangeUpdated, Job: snapshot})
}

// emit calls the change callback outside the lock.
func (q *Queue) emit(ev ChangeEvent) {
	if q.onChange != nil {
		q.onChange(ev)
	}
}
