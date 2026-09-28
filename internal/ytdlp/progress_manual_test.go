//go:build manual

// Manual end-to-end check: run a real download and feed its live stdout through
// the progress parser, asserting we actually see progress lines.
//
//	go test -tags manual ./internal/ytdlp/ -run ManualProgress -v
package ytdlp

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestManualProgressLiveParsing(t *testing.T) {
	dir := filepath.Join("..", "..", "resources")
	if _, err := os.Stat(filepath.Join(dir, ytDlpName)); err != nil {
		t.Skip("resources/ binaries not staged")
	}
	bins, err := resolveBins(dir, fileExists)
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// A small direct file downloads quickly and still emits real progress lines.
	url := "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4"
	cmd := exec.CommandContext(ctx, bins.YtDlp,
		"--newline", "--no-warnings", "--progress",
		"--ffmpeg-location", dir,
		"-o", filepath.Join(outDir, "%(title)s.%(ext)s"),
		url,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout // yt-dlp writes progress to stderr; merge for simplicity
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var progressCount int
	var last Progress
	var dest string
	for scanner.Scan() {
		line := scanner.Text()
		if p, ok := parseProgressLine(line); ok {
			progressCount++
			last = p
		}
		if d, ok := parseDestination(line); ok {
			dest = d
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("yt-dlp failed: %v", err)
	}

	t.Logf("parsed %d progress lines; last=%+v; destination=%q", progressCount, last, dest)
	if progressCount == 0 {
		t.Fatal("no progress lines parsed from a real download")
	}
	if last.Percent < 99 {
		t.Errorf("last percent = %v, want ~100 at completion", last.Percent)
	}
	if last.TotalBytes == 0 {
		t.Error("TotalBytes = 0, want a real size")
	}
	if dest == "" {
		t.Error("no destination path parsed")
	}
	if FormatBytes(last.TotalBytes) == "" || FormatDuration(last.ETASec) == "" {
		t.Error("format helpers returned empty output")
	}
}
