package ytdlp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Progress is one parsed `[download]` status line. Zero values mean "unknown":
// yt-dlp prints "Unknown B/s" and "ETA Unknown" at the start of a download, and
// the final summary line carries no ETA at all.
type Progress struct {
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloadedBytes"`
	TotalBytes      int64   `json:"totalBytes"`
	SpeedBps        int64   `json:"speedBps"`
	ETASec          int     `json:"etaSec"`
}

// The regexes below are matched against REAL yt-dlp --newline output, e.g.:
//
//	[download]  42.3% of  128.00MiB at    1.90MiB/s ETA 00:42
//	[download]   0.1% of  967.79KiB at  Unknown B/s ETA Unknown
//	[download]  12.3% of ~  3.27MiB at  ~  1.23MiB/s ETA 00:01 (frag 5/20)
//	[download] 100% of  967.79KiB in 00:00:00 at 4.19MiB/s
var (
	percentRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)%`)
	// A leading '~' marks an estimated size; it must be tolerated.
	totalRe = regexp.MustCompile(`of\s+~?\s*([0-9]+(?:\.[0-9]+)?)\s*(B|KiB|MiB|GiB)\b`)
	// "Unknown B/s" simply yields no match -> speed stays 0.
	speedRe = regexp.MustCompile(`at\s+~?\s*([0-9]+(?:\.[0-9]+)?)\s*(B|KiB|MiB|GiB)/s\b`)
	// ETA is mm:ss (e.g. "ETA 00:42") or hh:mm:ss (e.g. "ETA 01:02:03");
	// "ETA Unknown" yields no match.
	etaRe = regexp.MustCompile(`ETA\s+(?:([0-9]+):)?([0-9]{1,2}):([0-9]{2})\b`)
)

// sizeUnits maps yt-dlp's unit suffixes to byte multipliers.
var sizeUnits = map[string]int64{
	"B":   1,
	"KiB": 1024,
	"MiB": 1024 * 1024,
	"GiB": 1024 * 1024 * 1024,
}

// parseProgressLine parses a single output line. It returns ok=false for any
// line that is not a download progress report (log lines, "Destination:" lines,
// merger/extractor notices), so callers can safely feed it every line.
func parseProgressLine(line string) (Progress, bool) {
	if !strings.HasPrefix(line, "[download]") {
		return Progress{}, false
	}

	pm := percentRe.FindStringSubmatch(line)
	if pm == nil {
		return Progress{}, false
	}
	percent, err := strconv.ParseFloat(pm[1], 64)
	if err != nil {
		return Progress{}, false
	}

	p := Progress{Percent: percent}

	if tm := totalRe.FindStringSubmatch(line); tm != nil {
		p.TotalBytes = parseSize(tm[1], tm[2])
	}
	if sm := speedRe.FindStringSubmatch(line); sm != nil {
		p.SpeedBps = parseSize(sm[1], sm[2])
	}
	if em := etaRe.FindStringSubmatch(line); em != nil {
		// em[1] = hours (may be empty), em[2] = minutes, em[3] = seconds.
		h := 0
		if em[1] != "" {
			h, _ = strconv.Atoi(em[1])
		}
		m, _ := strconv.Atoi(em[2])
		s, _ := strconv.Atoi(em[3])
		p.ETASec = h*3600 + m*60 + s
	}

	// Downloaded bytes are derived, since yt-dlp only prints the percentage.
	if p.TotalBytes > 0 {
		p.DownloadedBytes = int64(float64(p.TotalBytes) * percent / 100)
	}
	return p, true
}

// parseSize converts a numeric string plus unit into bytes.
func parseSize(num, unit string) int64 {
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	return int64(f * float64(sizeUnits[unit]))
}

// destinationRe matches the artefact path yt-dlp writes. [download] covers plain
// downloads and [ExtractAudio] covers MP3 conversion; [Merger] is intentionally
// excluded because its output is quoted differently and is not the final name.
var destinationRe = regexp.MustCompile(`^\[(?:download|ExtractAudio|ffmpeg|Merger)\]\s+Destination:\s+(.+)$`)

// parseDestination extracts the output path from a Destination line, which tells
// the UI where the finished file landed.
func parseDestination(line string) (string, bool) {
	m := destinationRe.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

// FormatBytes renders a byte count for humans, e.g. 1536 -> "1.5 KiB". Values
// below 1 KiB are shown as whole bytes.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	value := float64(n)
	i := -1
	for value >= unit && i < len(units)-1 {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", value, units[i])
}

// FormatDuration renders seconds as mm:ss, or hh:mm:ss past an hour. Zero (an
// unknown ETA) renders as "--:--" so the UI can show a placeholder.
func FormatDuration(sec int) string {
	if sec <= 0 {
		return "--:--"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
