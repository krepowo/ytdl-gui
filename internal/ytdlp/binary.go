// Package ytdlp wraps the bundled yt-dlp binary. It is the ONLY place in the app
// that knows yt-dlp's command-line surface, so a future engine swap touches only
// this package.
//
// It is deliberately site-agnostic: it works for any URL yt-dlp supports (1000+
// named extractors plus the generic fallback) and never hardcodes a provider.
package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Binary file names shipped next to the executable (see the packaging spec).
const (
	ytDlpName   = "yt-dlp.exe"
	ffmpegName  = "ffmpeg.exe"
	ffprobeName = "ffprobe.exe"
)

// ErrBinaryMissing is wrapped by every error returned when a required bundled
// binary is absent. Callers can test for it with errors.Is.
var ErrBinaryMissing = errors.New("ytdlp: required binary is missing")

// MissingBinaryError names the binary that could not be found so the UI can tell
// the user exactly what is wrong (a corrupted install, not a normal state).
type MissingBinaryError struct {
	Name string
	Path string
}

func (e *MissingBinaryError) Error() string {
	return fmt.Sprintf("%s: %s not found at %s (the installation may be corrupted)", ErrBinaryMissing, e.Name, e.Path)
}

// Unwrap lets errors.Is(err, ErrBinaryMissing) succeed.
func (e *MissingBinaryError) Unwrap() error { return ErrBinaryMissing }

// Bins holds the resolved absolute paths to the bundled binaries.
type Bins struct {
	YtDlp   string
	FFmpeg  string
	FFprobe string
}

// Dir returns the directory containing the binaries. yt-dlp needs this for
// --ffmpeg-location so it can find ffmpeg/ffprobe.
func (b Bins) Dir() string { return filepath.Dir(b.YtDlp) }

// Versions holds the reported versions of the bundled binaries.
type Versions struct {
	YtDlp  string
	FFmpeg string
}

// runner runs a binary and returns its combined stdout. It is a seam so tests
// can exercise version parsing without real binaries.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// resolveBins builds the expected paths for the binaries in dir, verifying each
// with exists. The first missing binary produces a *MissingBinaryError.
//
// It is pure (no filesystem access) so it is fully unit-testable.
func resolveBins(dir string, exists func(string) bool) (Bins, error) {
	pick := func(name string) (string, error) {
		p := filepath.Join(dir, name)
		if !exists(p) {
			return "", &MissingBinaryError{Name: name, Path: p}
		}
		return p, nil
	}

	ytdlp, err := pick(ytDlpName)
	if err != nil {
		return Bins{}, err
	}
	ffmpeg, err := pick(ffmpegName)
	if err != nil {
		return Bins{}, err
	}
	ffprobe, err := pick(ffprobeName)
	if err != nil {
		return Bins{}, err
	}

	return Bins{YtDlp: ytdlp, FFmpeg: ffmpeg, FFprobe: ffprobe}, nil
}

// ResolveBins locates the bundled binaries next to the running executable.
func ResolveBins() (Bins, error) {
	exe, err := os.Executable()
	if err != nil {
		return Bins{}, fmt.Errorf("ytdlp: locate executable: %w", err)
	}
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	return resolveBins(filepath.Dir(exe), fileExists)
}

// fileExists reports whether path is an existing regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// execRunner is the production runner: it executes a binary and returns stdout.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			return nil, fmt.Errorf("%s: %s", name, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// versions queries the reported versions of yt-dlp and ffmpeg.
func versions(ctx context.Context, run runner, bins Bins) (Versions, error) {
	var v Versions

	ytOut, err := run(ctx, bins.YtDlp, "--version")
	if err != nil {
		return v, fmt.Errorf("ytdlp: query yt-dlp version: %w", err)
	}
	v.YtDlp = firstLine(string(ytOut))

	ffOut, err := run(ctx, bins.FFmpeg, "-version")
	if err != nil {
		return v, fmt.Errorf("ytdlp: query ffmpeg version: %w", err)
	}
	v.FFmpeg = parseFFmpegVersion(string(ffOut))

	return v, nil
}

// Versions reports the versions of the bundled binaries.
func QueryVersions(ctx context.Context, bins Bins) (Versions, error) {
	return versions(ctx, execRunner, bins)
}

// firstLine returns the first non-empty line, trimmed of surrounding whitespace.
// yt-dlp --version prints a single line like "2026.06.09".
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// parseFFmpegVersion extracts the version token from `ffmpeg -version` output,
// whose first line looks like:
//
//	ffmpeg version 9.0-essentials_build-www.gyan.dev Copyright (c) 2000-2026
func parseFFmpegVersion(out string) string {
	line := firstLine(out)
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	// Expect: ["ffmpeg", "version", "<token>", ...]
	if len(fields) >= 3 && fields[0] == "ffmpeg" && fields[1] == "version" {
		return fields[2]
	}
	// Fallback: return the last field of the first line rather than nothing.
	return fields[len(fields)-1]
}
