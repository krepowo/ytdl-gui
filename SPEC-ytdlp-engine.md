# Spec: ytdlp-engine

## Objective
Wrap the bundled `yt-dlp.exe` so the rest of the app never touches its CLI.
Provides three capabilities: **probe** (metadata + format list), **run**
(start a download and stream progress), and **control** (pause/resume/cancel).
**Works for any site yt-dlp supports** — 1000+ named extractors plus the
generic fallback. No provider-specific assumptions anywhere in this module.

## Tech Stack
Go `os/exec`, `bufio`, `context`. Bundled binaries in `resources/`:
`yt-dlp.exe`, `ffmpeg.exe`, `ffprobe.exe`, resolved relative to the executable.

## Commands
- Build: `wails build`
- Test: `go test ./internal/ytdlp/...`
- Lint: `gofmt -l . && go vet ./...`

## Project Structure
```
internal/ytdlp/
  binary.go     → resolve bundled yt-dlp/ffmpeg paths, version checks
  probe.go      → yt-dlp -J <url> → MediaInfo + []FormatOption (site-agnostic)
  download.go   → spawn process, parse --newline progress, Pause/Resume/Cancel
  progress.go   → parse "[download]  42.3% of 128.00MiB at 1.20MiB/s ETA 00:42"
  ytdlp_test.go
resources/
  yt-dlp.exe  ffmpeg.exe  ffprobe.exe
```

## Multi-Source Requirements
- **Never assume YouTube.** Read provider identity from yt-dlp's JSON:
  `extractor` / `extractor_key` / `webpage_url_domain`. Surface it as a
  *source badge* so the user sees where a link came from.
- **Format lists are dynamic and site-specific.** Some sites expose only
  combined A/V; others expose separate video-only + audio-only streams that must
  be merged by ffmpeg (`-f bestvideo+bestaudio --merge-output-format mp4`). The
  probe must derive options from what the site actually returns, not a fixed
  ladder.
- **Generic extractor fallback** is yt-dlp's own behaviour; we simply pass the
  URL through and report success/failure faithfully.
- **Optional metadata:** `duration`, `thumbnail`, `uploader`, `title` may be
  missing (live streams, generic pages). The UI must degrade gracefully.
- **Cookies:** sites with age gates / logins need browser cookies. When
  `settings.CookiesBrowser` is set, pass `--cookies-from-browser <browser>`.
- **ffmpeg is mandatory** for MP3 extraction (`-x --audio-format mp3`) and for
  merging separate streams. Pass `--ffmpeg-location <bundled ffmpeg dir>`.

## Code Style
```go
// Probe returns metadata and selectable formats for any yt-dlp-supported URL.
func Probe(ctx context.Context, bins Bins, url string, opts ProbeOptions) (*MediaInfo, error) {
	args := []string{"-J", "--no-warnings", "--no-playlist"}
	if opts.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesBrowser)
	}
	args = append(args, url)
	out, err := exec.CommandContext(ctx, bins.YtDlp, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("ytdlp: probe %q: %w", url, err)
	}
	// ... unmarshal into MediaInfo, build FormatOption list
}

// MediaInfo carries only fields we actually render; all optional.
type MediaInfo struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Uploader   string  `json:"uploader"`
	Duration   float64 `json:"duration"`   // may be 0 (live/unknown)
	Thumbnail  string  `json:"thumbnail"`  // may be ""
	Extractor  string  `json:"extractor"`  // e.g. "youtube", "twitter", "generic"
	WebpageURL string  `json:"webpage_url"`
	IsLive     bool    `json:"is_live"`
}
```
- Every exec call takes a `context.Context` (cancellation = kill process tree).
- Progress callbacks are throttled to ≤4 updates/second to avoid flooding the bridge.
- Pause/Resume use OS process suspension for a true pause; on failure, fall back
  to kill + resume with `--continue` (yt-dlp resumes from a `.part` file).

## Testing Strategy
Go `testing` + a **fake yt-dlp** batch script that replays canned output, so
tests never hit the network. Table-driven tests for the progress-line parser
(percentages, sizes B/KiB/MiB/GiB, speeds, ETA, the 100% line). Probe tests
cover a YouTube-shaped JSON, a generic-page JSON (missing duration/thumbnail),
and an audio-only site, to prove no provider assumption leaks in.

## Boundaries
- **Always:** pass args as an argv slice (never build a shell string); kill the
  whole process tree on cancel; surface yt-dlp stderr to the job's error field.
- **Ask first:** adding a new yt-dlp CLI flag; changing the pause strategy;
  enabling playlist downloads (default is single-item).
- **Never:** hardcode a provider/extractor name in logic; block the UI
  goroutine; leave orphan `yt-dlp.exe`/`ffmpeg.exe` processes after cancel/quit.

## Success Criteria
- [ ] `Probe` works on a YouTube URL, a Twitter/X URL, and a generic page, returning correct `Extractor`.
- [ ] Progress parser turns a real yt-dlp `--newline` line into `{percent, bytes, total, speed, eta}`; unit tests pass.
- [ ] Audio-only job produces a valid `.mp3` using bundled ffmpeg.
- [ ] A site exposing separate video/audio streams downloads and merges into one file.
- [ ] Cancel terminates the process tree within 2s; no orphan process remains.
- [ ] Pause halts progress updates; Resume continues from the same byte offset.

## Open Questions
- None.
