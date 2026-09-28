// Package queue owns the lifecycle of every download: it accepts jobs, runs up
// to MaxConcurrent at once, drives each job through a validated state machine,
// and exposes live status for the UI.
package queue

import (
	"fmt"
	"time"
)

// State is a job's lifecycle state. The string values are the wire format used
// by the frontend, so they must stay stable.
type State string

const (
	StateQueued      State = "queued"
	StateDownloading State = "downloading"
	StatePaused      State = "paused"
	StateCompleted   State = "completed"
	StateError       State = "error"
	StateCanceled    State = "canceled"
)

// isTerminal reports whether a state ends the job's life.
func isTerminal(s State) bool {
	return s == StateCompleted || s == StateError || s == StateCanceled
}

// canTransition reports whether a state change is legal. Same-state "changes"
// are not transitions and are rejected.
func canTransition(from, to State) bool {
	switch from {
	case StateQueued:
		return to == StateDownloading || to == StateCanceled
	case StateDownloading:
		return to == StatePaused || to == StateCompleted || to == StateError || to == StateCanceled
	case StatePaused:
		return to == StateDownloading || to == StateCanceled
	default:
		// Terminal states never change.
		return false
	}
}

// DownloadRequest is the immutable input that starts a job. It is kept off the
// wire format (the UI already knows what it asked for).
type DownloadRequest struct {
	URL              string
	OutputDir        string
	Mode             string // "video" | "audio"
	FormatID         string
	NeedsMerge       bool
	CookiesBrowser   string
	FilenameTemplate string
	Title            string
}

// Job is one download task and its live status. The JSON tags define the wire
// format shared with the frontend.
type Job struct {
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	Title           string  `json:"title"`
	Mode            string  `json:"mode"`
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

	// Runtime-only fields, deliberately unexported so they never reach the wire.
	req    DownloadRequest
	handle any // engine handle (cancel/pause/resume); type owned by the engine adapter
}

// newJob creates a queued job from a request.
func newJob(id string, req DownloadRequest, now time.Time) *Job {
	return &Job{
		ID:        id,
		URL:       req.URL,
		Title:     req.Title,
		Mode:      req.Mode,
		FormatID:  req.FormatID,
		State:     StateQueued,
		CreatedAt: now.Unix(),
		req:       req,
	}
}

// setState transitions the job, rejecting illegal moves. The state is left
// untouched when the move is rejected.
func (j *Job) setState(to State) error {
	if !canTransition(j.State, to) {
		return fmt.Errorf("queue: illegal transition %s -> %s", j.State, to)
	}
	j.State = to
	return nil
}

// clone returns a deep copy safe to hand to another goroutine or marshal. Only
// value fields exist, so a shallow copy suffices, but the method documents the
// intent and gives one place to change if a slice/map field is ever added.
func (j *Job) clone() Job {
	c := *j
	return c
}
