package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveBinsRealBinaries is an integration check that runs only when the
// real binaries have been staged in resources/ (see scripts/fetch-binaries.sh).
// It verifies version parsing against actual tool output.
func TestResolveBinsRealBinaries(t *testing.T) {
	dir := filepath.Join("..", "..", "resources")
	if _, err := os.Stat(filepath.Join(dir, ytDlpName)); err != nil {
		t.Skip("resources/ binaries not staged; run scripts/fetch-binaries.sh")
	}

	bins, err := resolveBins(dir, fileExists)
	if err != nil {
		t.Fatalf("resolveBins() on real binaries: %v", err)
	}

	v, err := QueryVersions(context.Background(), bins)
	if err != nil {
		t.Fatalf("QueryVersions() error = %v", err)
	}
	if v.YtDlp == "" {
		t.Error("yt-dlp version is empty")
	}
	if v.FFmpeg == "" {
		t.Error("ffmpeg version is empty")
	}
	t.Logf("yt-dlp=%s ffmpeg=%s", v.YtDlp, v.FFmpeg)
}
