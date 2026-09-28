# Spec: download-queue

## Objective
Own the lifecycle of every download. Accepts jobs, runs up to `MaxConcurrent`
at a time, drives each job through a state machine, exposes live status, and
persists history so the list survives restarts.

## Tech Stack
Go goroutines + `sync.Mutex`, channel-based worker pool. History at
`%APPDATA%/VideoDownloader/history.json`.

## Commands
- Test: `go test ./internal/queue/...`
- Race check: `go test -race ./internal/queue/...`

## Project Structure
```
internal/queue/
  queue.go      → Queue, Add, worker scheduler, concurrency limit
  job.go        → Job struct + State enum + transitions
  history.go    → Load/Save completed+failed history
  queue_test.go
```

## Code Style
```go
type State string
const (
	StateQueued      State = "queued"
	StateDownloading State = "downloading"
	StatePaused      State = "paused"
	StateCompleted   State = "completed"
	StateError       State = "error"
	StateCanceled    State = "canceled"
)

// Job is one download task and its live status.
type Job struct {
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	Title           string  `json:"title"`
	Mode            string  `json:"mode"`   // video | audio
	FormatID        string  `json:"formatId"`
	State           State   `json:"state"`
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloadedBytes"`
	TotalBytes      int64   `json:"totalBytes"`
	SpeedBps        int64   `json:"speedBps"`
	ETASec          int     `json:"etaSec"`
	OutputPath      string  `json:"outputPath"`
	Error           string  `json:"error"`
	CreatedAt       int64   `json:"createdAt"`
}
```
- State changes only via `job.transition(to)` which rejects illegal moves.

## Testing Strategy
`go test -race`. Tests: concurrency limit is never exceeded; queued jobs start
as slots free; pause/resume/cancel transitions; a failed job frees its slot;
history round-trips. Engine is injected via an interface so tests use a fake.

## Boundaries
- **Always:** guard all shared state with a mutex; free the worker slot on
  *every* terminal state; persist history after each terminal transition.
- **Ask first:** raising the concurrency ceiling; changing history retention.
- **Never:** run more than `MaxConcurrent` engine processes; lose a job on
  restart (queued/paused jobs are restored as `paused`, not auto-started).

## Success Criteria
- [ ] Adding 5 jobs with `MaxConcurrent=2` never runs more than 2 at once.
- [ ] Finishing a job immediately promotes the next `queued` job.
- [ ] Cancel a running job → slot frees, next job starts, no orphan process.
- [ ] After restart, completed/failed history is restored; interrupted jobs load as `paused`.
- [ ] `go test -race ./internal/queue/...` passes.

## Open Questions
- None.
