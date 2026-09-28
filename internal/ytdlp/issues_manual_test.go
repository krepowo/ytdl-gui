//go:build manual

package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestManualIssuesEndToEnd proves the audio + duplicate-name fixes against a REAL
// download through the production code path:
//
//  1. Audio mode produces an .mp3 (ffmpeg conversion).
//  2. A second download of the same media gets a "+1" suffix instead of being
//     skipped as "already downloaded".
//
// The caller passes the title it learned from Probe (exactly as the app does),
// because that is what yt-dlp's %(title)s template resolves to.
func TestManualIssuesEndToEnd(t *testing.T) {
	bins := manualBins(t)
	ctx := context.Background()

	const url = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	info, err := Probe(ctx, bins, url, ProbeOptions{})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	t.Logf("probe title: %q", info.Title)

	dir := t.TempDir()
	opts := DownloadOptions{
		URL:       url,
		OutputDir: dir,
		Mode:      ModeAudio,
		Title:     info.Title,
	}

	runOnce := func(label string) string {
		c, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()

		job, err := StartJob(c, bins, opts, JobCallbacks{})
		if err != nil {
			t.Fatalf("%s: StartJob: %v", label, err)
		}
		if st := job.Wait(); st != StateCompleted {
			t.Fatalf("%s: state=%s err=%v", label, st, job.Err())
		}
		return job.OutputPath()
	}

	first := runOnce("first")
	t.Logf("first destination: %q", first)
	if !strings.HasSuffix(strings.ToLower(first), ".mp3") {
		t.Errorf("first download not mp3: %q", first)
	}
	if strings.ContainsAny(filepath.Base(first), `<>:"/\|?*`) {
		t.Errorf("filename still contains illegal chars: %q", filepath.Base(first))
	}

	// Second download of the SAME media: must NOT skip, must create a new file.
	second := runOnce("second")
	t.Logf("second destination: %q", second)

	if second == first {
		t.Fatalf("second download reused path %q (should have incremented)", second)
	}
	if !strings.Contains(filepath.Base(second), "+1") {
		t.Errorf("second filename %q does not contain +1", filepath.Base(second))
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Logf("file: %s", e.Name())
	}
	if len(entries) < 2 {
		t.Errorf("expected 2 files, got %d", len(entries))
	}
}
