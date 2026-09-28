//go:build manual

// Decisive pause check: after pausing, progress must truly freeze (not merely
// drain a pipe buffer). We pause mid-download, then sample the reported progress
// three times, allowing the first gap to drain any buffered output; the later
// gaps must show zero advance.
//
//	go test -tags manual ./internal/ytdlp/ -run ManualPauseTrulyFreezes -v -timeout 300s
package ytdlp

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestManualPauseTrulyFreezes(t *testing.T) {
	bins := manualBins(t)
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	var mu sync.Mutex
	var last Progress
	j, err := StartJob(ctx, bins, DownloadOptions{
		URL:        "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		OutputDir:  dir,
		Mode:       ModeVideo,
		FormatID:   "137", // 1080p video-only: large enough to pause mid-flight
		NeedsMerge: true,
	}, JobCallbacks{OnProgress: func(p Progress) {
		mu.Lock()
		last = p
		mu.Unlock()
	}})
	if err != nil {
		t.Fatal(err)
	}

	sample := func() Progress {
		mu.Lock()
		defer mu.Unlock()
		return last
	}

	// Wait until we are clearly mid-download (progress is meaningful).
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if sample().DownloadedBytes > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if sample().DownloadedBytes == 0 {
		t.Skip("download did not start emitting progress in time")
	}

	if err := j.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Gap 1: allow any already-buffered output to drain.
	time.Sleep(3 * time.Second)
	s1 := sample()

	// Gap 2 and 3: a truly suspended process emits nothing more.
	time.Sleep(2 * time.Second)
	s2 := sample()
	time.Sleep(2 * time.Second)
	s3 := sample()

	t.Logf("paused samples: s1=%.1f%% (%d B), s2=%.1f%% (%d B), s3=%.1f%% (%d B)",
		s1.Percent, s1.DownloadedBytes, s2.Percent, s2.DownloadedBytes, s3.Percent, s3.DownloadedBytes)

	if s2.DownloadedBytes != s3.DownloadedBytes {
		t.Errorf("progress advanced while paused: %d -> %d bytes (suspend not effective)",
			s2.DownloadedBytes, s3.DownloadedBytes)
	}

	if err := j.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// After resume, progress must advance again.
	time.Sleep(3 * time.Second)
	s4 := sample()
	if s4.DownloadedBytes <= s3.DownloadedBytes {
		t.Errorf("progress did not advance after resume: %d -> %d bytes", s3.DownloadedBytes, s4.DownloadedBytes)
	}

	if err := j.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	t.Logf("verified: frozen at %d bytes while paused, resumed to %d bytes", s3.DownloadedBytes, s4.DownloadedBytes)

	_ = filepath.Join(dir, "unused")
}
