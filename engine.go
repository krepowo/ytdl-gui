package main

import (
	"context"

	"ytdl-gui/internal/app"
	"ytdl-gui/internal/ytdlp"
)

// ytdlpEngine adapts the ytdlp package to the app.DownloadEngine interface. It
// is the ONLY place the app layer touches yt-dlp's concrete API, so a future
// engine swap changes only this file.
//
// binsErr captures a failure to locate the bundled binaries. When set, every
// call returns it as a readable error instead of the app crashing at startup.
type ytdlpEngine struct {
	bins    ytdlp.Bins
	binsErr error
}

func (e ytdlpEngine) Probe(ctx context.Context, url string, opts ytdlp.ProbeOptions) (*ytdlp.MediaInfo, error) {
	if e.binsErr != nil {
		return nil, e.binsErr
	}
	return ytdlp.Probe(ctx, e.bins, url, opts)
}

func (e ytdlpEngine) Start(ctx context.Context, opts ytdlp.DownloadOptions, onProgress func(ytdlp.Progress), onState func(ytdlp.JobState)) (app.JobHandle, error) {
	if e.binsErr != nil {
		return nil, e.binsErr
	}
	job, err := ytdlp.StartJob(ctx, e.bins, opts, ytdlp.JobCallbacks{
		OnProgress: onProgress,
		OnState:    onState,
	})
	if err != nil {
		return nil, err
	}
	return jobHandle{job: job}, nil
}

// jobHandle adapts *ytdlp.Job (whose Wait returns a JobState) to app.JobHandle
// (whose Wait returns an error), so the queue can treat both uniformly.
type jobHandle struct {
	job *ytdlp.Job
}

func (h jobHandle) Wait() error {
	state := h.job.Wait()
	// A canceled job is not a failure: the queue already tracks that state.
	if state == ytdlp.StateFailed {
		return h.job.Err()
	}
	return nil
}

func (h jobHandle) OutputPath() string { return h.job.OutputPath() }
func (h jobHandle) Pause() error       { return h.job.Pause() }
func (h jobHandle) Resume() error      { return h.job.Resume() }
func (h jobHandle) Cancel() error      { return h.job.Cancel() }
