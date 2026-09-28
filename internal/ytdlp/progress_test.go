package ytdlp

import (
	"testing"
)

// mib/kiB/gib are runtime helpers so float maths truncates like the parser does.
func mib(f float64) int64 { return int64(f * 1024 * 1024) }
func kib(f float64) int64 { return int64(f * 1024) }
func gib(f float64) int64 { return int64(f * 1024 * 1024 * 1024) }

// The fixtures below are copied from REAL yt-dlp --newline output captured while
// downloading, so the parser is tested against the actual format, not a guess.

func TestParseProgressStandardLine(t *testing.T) {
	line := `[download]  42.3% of  128.00MiB at    1.90MiB/s ETA 00:42`

	p, ok := parseProgressLine(line)
	if !ok {
		t.Fatal("ok = false, want true for a progress line")
	}
	if p.Percent != 42.3 {
		t.Errorf("Percent = %v, want 42.3", p.Percent)
	}
	if p.TotalBytes != mib(128) {
		t.Errorf("TotalBytes = %d, want %d", p.TotalBytes, mib(128))
	}
	if p.SpeedBps != mib(1.90) {
		t.Errorf("SpeedBps = %d, want %d", p.SpeedBps, mib(1.90))
	}
	if p.ETASec != 42 {
		t.Errorf("ETASec = %d, want 42", p.ETASec)
	}
}

func TestParseProgressComputesDownloadedBytes(t *testing.T) {
	p, ok := parseProgressLine(`[download]  50.0% of  100.00MiB at  1.00MiB/s ETA 00:50`)
	if !ok {
		t.Fatal("ok = false")
	}
	if p.DownloadedBytes != mib(50) {
		t.Errorf("DownloadedBytes = %d, want %d", p.DownloadedBytes, mib(50))
	}
}

func TestParseProgressUnknownSpeedAndETA(t *testing.T) {
	// The very first lines of a real download look exactly like this.
	line := `[download]   0.1% of  967.79KiB at  Unknown B/s ETA Unknown`

	p, ok := parseProgressLine(line)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if p.Percent != 0.1 {
		t.Errorf("Percent = %v, want 0.1", p.Percent)
	}
	if p.SpeedBps != 0 {
		t.Errorf("SpeedBps = %d, want 0 (unknown)", p.SpeedBps)
	}
	if p.ETASec != 0 {
		t.Errorf("ETASec = %d, want 0 (unknown)", p.ETASec)
	}
	if p.TotalBytes != kib(967.79) {
		t.Errorf("TotalBytes = %d, want %d", p.TotalBytes, kib(967.79))
	}
}

func TestParseProgressApproxSizeTilde(t *testing.T) {
	// yt-dlp marks estimated totals/speeds with a leading '~'.
	line := `[download]  12.3% of ~  3.27MiB at  ~  1.23MiB/s ETA 00:01 (frag 5/20)`

	p, ok := parseProgressLine(line)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if p.Percent != 12.3 {
		t.Errorf("Percent = %v, want 12.3", p.Percent)
	}
	if p.SpeedBps != mib(1.23) {
		t.Errorf("SpeedBps = %d, want %d", p.SpeedBps, mib(1.23))
	}
	if p.ETASec != 1 {
		t.Errorf("ETASec = %d, want 1", p.ETASec)
	}
}

func TestParseProgressFinalSummaryLine(t *testing.T) {
	// The last line uses "in <elapsed> at <speed>" instead of "at <speed> ETA".
	line := `[download] 100% of  967.79KiB in 00:00:00 at 4.19MiB/s`

	p, ok := parseProgressLine(line)
	if !ok {
		t.Fatal("ok = false, want true for the final summary line")
	}
	if p.Percent != 100 {
		t.Errorf("Percent = %v, want 100", p.Percent)
	}
	if p.ETASec != 0 {
		t.Errorf("ETASec = %d, want 0", p.ETASec)
	}
	if p.SpeedBps != mib(4.19) {
		t.Errorf("SpeedBps = %d, want %d", p.SpeedBps, mib(4.19))
	}
}

func TestParseProgressHundredDecimal(t *testing.T) {
	p, ok := parseProgressLine(`[download] 100.0% of    1.44MiB at   11.09MiB/s ETA 00:00`)
	if !ok {
		t.Fatal("ok = false")
	}
	if p.Percent != 100.0 {
		t.Errorf("Percent = %v, want 100", p.Percent)
	}
}

func TestParseProgressSizeUnits(t *testing.T) {
	tests := []struct {
		line string
		want int64
	}{
		{`[download]   1.0% of  500.00B at 1.00B/s ETA 00:01`, 500},
		{`[download]   1.0% of  1.00KiB at 1.00KiB/s ETA 00:01`, 1024},
		{`[download]   1.0% of  1.00MiB at 1.00MiB/s ETA 00:01`, 1024 * 1024},
		{`[download]   1.0% of  1.00GiB at 1.00GiB/s ETA 00:01`, 1024 * 1024 * 1024},
	}
	for _, tc := range tests {
		p, ok := parseProgressLine(tc.line)
		if !ok {
			t.Fatalf("ok = false for %q", tc.line)
		}
		if p.TotalBytes != tc.want {
			t.Errorf("%q -> TotalBytes = %d, want %d", tc.line, p.TotalBytes, tc.want)
		}
	}
}

func TestParseProgressIgnoresNonProgressLines(t *testing.T) {
	nonProgress := []string{
		`[generic] Extracting URL: https://example.com/v`,
		`[info] Big_Buck_Bunny: Downloading 1 format(s): mp4`,
		`[download] Destination: \tmp\file.mp4`,
		`[Merger] Merging formats into "out.mp4"`,
		`[ExtractAudio] Destination: out.mp3`,
		`ERROR: something went wrong`,
		``,
	}
	for _, line := range nonProgress {
		if _, ok := parseProgressLine(line); ok {
			t.Errorf("parseProgressLine(%q) = ok, want false", line)
		}
	}
}

func TestParseProgressDestination(t *testing.T) {
	got, ok := parseDestination(`[download] Destination: \tmp\dl\My Video.mp4`)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got != `\tmp\dl\My Video.mp4` {
		t.Errorf("path = %q, want the destination path", got)
	}
}

func TestParseProgressDestinationExtractAudio(t *testing.T) {
	// After MP3 conversion the final artefact is reported by [ExtractAudio].
	got, ok := parseDestination(`[ExtractAudio] Destination: \tmp\dl\song.mp3`)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got != `\tmp\dl\song.mp3` {
		t.Errorf("path = %q, want the mp3 path", got)
	}
}

func TestParseProgressDestinationIgnoresOtherLines(t *testing.T) {
	if _, ok := parseDestination(`[download]  42.3% of  128.00MiB at 1.90MiB/s ETA 00:42`); ok {
		t.Error("ok = true for a progress line, want false")
	}
}

func TestFormatBytesHumanReadable(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{128 * 1024 * 1024, "128.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
	}
	for _, tc := range tests {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseProgressSpeedUnits(t *testing.T) {
	tests := []struct {
		line string
		want int64
	}{
		{`[download]   1.0% of  1.00MiB at 512.00B/s ETA 00:01`, 512},
		{`[download]   1.0% of  1.00MiB at 2.00KiB/s ETA 00:01`, 2048},
		{`[download]   1.0% of  1.00MiB at 3.00MiB/s ETA 00:01`, 3 * 1024 * 1024},
		{`[download]   1.0% of  1.00GiB at 1.50GiB/s ETA 00:01`, int64(1.5 * 1024 * 1024 * 1024)},
	}
	for _, tc := range tests {
		p, ok := parseProgressLine(tc.line)
		if !ok {
			t.Fatalf("ok = false for %q", tc.line)
		}
		if p.SpeedBps != tc.want {
			t.Errorf("%q -> SpeedBps = %d, want %d", tc.line, p.SpeedBps, tc.want)
		}
	}
}

func TestParseProgressLongETA(t *testing.T) {
	p, ok := parseProgressLine(`[download]   5.0% of    1.00GiB at 1.00MiB/s ETA 01:02:03`)
	if !ok {
		t.Fatal("ok = false")
	}
	want := 1*3600 + 2*60 + 3
	if p.ETASec != want {
		t.Errorf("ETASec = %d, want %d", p.ETASec, want)
	}
}

func TestFormatDurationHumanReadable(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "--:--"},
		{42, "00:42"},
		{90, "01:30"},
		{3661, "01:01:01"},
	}
	for _, tc := range tests {
		if got := FormatDuration(tc.in); got != tc.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
