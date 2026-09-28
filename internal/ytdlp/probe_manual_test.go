//go:build manual

// Manual probe check against real sources. Run with:
//
//	go test -tags manual ./internal/ytdlp/ -run ManualProbe -v
//
// Excluded from normal test runs because it needs network access.
package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManualProbeRealSites(t *testing.T) {
	dir := filepath.Join("..", "..", "resources")
	if _, err := os.Stat(filepath.Join(dir, ytDlpName)); err != nil {
		t.Skip("resources/ binaries not staged")
	}
	bins, err := resolveBins(dir, fileExists)
	if err != nil {
		t.Fatal(err)
	}

	urls := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		// Direct media URL exercises the generic extractor (null codecs, no height).
		"https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4",
	}

	for _, u := range urls {
		t.Run(u, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			info, err := Probe(ctx, bins, u, ProbeOptions{})
			if err != nil {
				t.Fatalf("Probe(%s): %v", u, err)
			}
			t.Logf("extractor=%s title=%q duration=%.0fs video=%d audio=%d live=%v",
				info.Extractor, info.Title, info.DurationSec,
				len(info.VideoOptions), len(info.AudioOptions), info.IsLive)
			for _, o := range info.VideoOptions {
				t.Logf("  video option: id=%s label=%q h=%d ext=%s merge=%v",
					o.FormatID, o.Label, o.Height, o.Ext, o.NeedsMerge)
			}
			if info.Extractor == "" {
				t.Error("extractor is empty (source badge would be blank)")
			}
			if len(info.VideoOptions) == 0 {
				t.Error("no video options -> user could not download this URL")
			}
		})
	}
}
