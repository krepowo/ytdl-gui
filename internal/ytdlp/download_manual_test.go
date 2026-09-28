//go:build manual

// Manual end-to-end checks for the download runner against REAL binaries and
// network. Verifies the highest-risk acceptance criteria: audio MP3 output,
// merged video output, cancel killing the process tree (no orphans), and
// pause/resume continuing the download.
//
//	go test -tags manual ./internal/ytdlp/ -run ManualDownload -v -timeout 300s
package ytdlp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func manualBins(t *testing.T) Bins {
	t.Helper()
	dir := filepath.Join("..", "..", "resources")
	if _, err := os.Stat(filepath.Join(dir, ytDlpName)); err != nil {
		t.Skip("resources/ binaries not staged")
	}
	bins, err := resolveBins(dir, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	return bins
}

// countProcesses counts running processes whose image name contains needle.
func countProcesses(t *testing.T, needle string) int {
	t.Helper()
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		t.Fatalf("tasklist: %v", err)
	}
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(strings.ToLower(line), strings.ToLower(needle)) {
			n++
		}
	}
	return n
}

func TestManualDownloadAudioMp3(t *testing.T) {
	bins := manualBins(t)
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	var last Progress
	j, err := StartJob(ctx, bins, DownloadOptions{
		URL:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		OutputDir: dir,
		Mode:      ModeAudio,
	}, JobCallbacks{OnProgress: func(p Progress) { last = p }})
	if err != nil {
		t.Fatal(err)
	}

	if state := j.Wait(); state != StateCompleted {
		t.Fatalf("state = %s, err = %v", state, j.Err())
	}

	mp3s, _ := filepath.Glob(filepath.Join(dir, "*.mp3"))
	if len(mp3s) == 0 {
		t.Fatalf("no .mp3 produced in %s (last progress %+v)", dir, last)
	}
	info, _ := os.Stat(mp3s[0])
	t.Logf("MP3 produced: %s (%d bytes)", filepath.Base(mp3s[0]), info.Size())
	if info.Size() == 0 {
		t.Error("MP3 is empty")
	}
}

func TestManualDownloadVideoMerge(t *testing.T) {
	bins := manualBins(t)
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// A low-res video-only stream forces the ffmpeg merge path.
	j, err := StartJob(ctx, bins, DownloadOptions{
		URL:        "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		OutputDir:  dir,
		Mode:       ModeVideo,
		FormatID:   "160", // 144p video-only (DASH)
		NeedsMerge: true,
	}, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}

	if state := j.Wait(); state != StateCompleted {
		t.Fatalf("state = %s, err = %v", state, j.Err())
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.mp4"))
	if len(files) == 0 {
		t.Fatalf("no merged .mp4 produced in %s", dir)
	}
	info, _ := os.Stat(files[0])
	t.Logf("merged video: %s (%d bytes)", filepath.Base(files[0]), info.Size())
	if info.Size() == 0 {
		t.Error("merged video is empty")
	}
}

func TestManualCancelLeavesNoOrphans(t *testing.T) {
	bins := manualBins(t)
	dir := t.TempDir()

	beforeYtdlp := countProcesses(t, "yt-dlp")
	beforeFFmpeg := countProcesses(t, "ffmpeg")

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	// A large download so we can cancel it mid-flight.
	j, err := StartJob(ctx, bins, DownloadOptions{
		URL:        "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		OutputDir:  dir,
		Mode:       ModeVideo,
		FormatID:   "137", // 1080p video-only, big enough to still be running
		NeedsMerge: true,
	}, JobCallbacks{})
	if err != nil {
		t.Fatal(err)
	}

	// Give it a moment to actually start downloading.
	time.Sleep(4 * time.Second)

	start := time.Now()
	if err := j.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if state := j.Wait(); state != StateCanceled {
		t.Fatalf("state = %s, want canceled", state)
	}
	t.Logf("cancel returned after %v", time.Since(start))

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("cancel took %v, want < 2s", elapsed)
	}

	// Allow the OS a moment to reap.
	time.Sleep(1500 * time.Millisecond)

	afterYtdlp := countProcesses(t, "yt-dlp")
	afterFFmpeg := countProcesses(t, "ffmpeg")
	t.Logf("yt-dlp: %d -> %d ; ffmpeg: %d -> %d", beforeYtdlp, afterYtdlp, beforeFFmpeg, afterFFmpeg)

	if afterYtdlp > beforeYtdlp {
		t.Errorf("orphan yt-dlp processes: %d before, %d after", beforeYtdlp, afterYtdlp)
	}
	if afterFFmpeg > beforeFFmpeg {
		t.Errorf("orphan ffmpeg processes: %d before, %d after", beforeFFmpeg, afterFFmpeg)
	}
}

func TestManualPauseResumeContinues(t *testing.T) {
	bins := manualBins(t)
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var mu sync.Mutex
	var last Progress
	j, err := StartJob(ctx, bins, DownloadOptions{
		URL:        "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		OutputDir:  dir,
		Mode:       ModeVideo,
		FormatID:   "137",
		NeedsMerge: true,
	}, JobCallbacks{OnProgress: func(p Progress) {
		mu.Lock()
		last = p
		mu.Unlock()
	}})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)
	if err := j.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if j.State() != StatePaused {
		t.Fatalf("state = %s, want paused", j.State())
	}

	mu.Lock()
	atPause := last
	mu.Unlock()
	time.Sleep(2 * time.Second)
	mu.Lock()
	stillPaused := last
	mu.Unlock()

	t.Logf("progress at pause: %.1f%% ; after 2s paused: %.1f%%", atPause.Percent, stillPaused.Percent)

	if err := j.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if state := j.Wait(); state != StateCompleted {
		t.Fatalf("state = %s, err = %v", state, j.Err())
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.mp4"))
	if len(files) == 0 {
		t.Fatal("no .mp4 produced after pause/resume")
	}
	t.Logf("completed after pause/resume: %s", filepath.Base(files[0]))
}
