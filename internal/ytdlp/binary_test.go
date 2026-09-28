package ytdlp

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBinsFindsAllBinaries(t *testing.T) {
	dir := `C:\Program Files\Video Downloader`
	allExist := func(string) bool { return true }

	got, err := resolveBins(dir, allExist)
	if err != nil {
		t.Fatalf("resolveBins() error = %v, want nil", err)
	}

	if want := filepath.Join(dir, "yt-dlp.exe"); got.YtDlp != want {
		t.Errorf("YtDlp = %q, want %q", got.YtDlp, want)
	}
	if want := filepath.Join(dir, "ffmpeg.exe"); got.FFmpeg != want {
		t.Errorf("FFmpeg = %q, want %q", got.FFmpeg, want)
	}
	if want := filepath.Join(dir, "ffprobe.exe"); got.FFprobe != want {
		t.Errorf("FFprobe = %q, want %q", got.FFprobe, want)
	}
}

func TestResolveBinsReportsWhichBinaryIsMissing(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name      string
		missing   string
		wantField string
	}{
		{"yt-dlp missing", "yt-dlp.exe", "yt-dlp.exe"},
		{"ffmpeg missing", "ffmpeg.exe", "ffmpeg.exe"},
		{"ffprobe missing", "ffprobe.exe", "ffprobe.exe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exists := func(p string) bool { return filepath.Base(p) != tc.missing }

			_, err := resolveBins(dir, exists)
			if err == nil {
				t.Fatal("resolveBins() error = nil, want an error")
			}
			if !errors.Is(err, ErrBinaryMissing) {
				t.Errorf("error = %v, want it to wrap ErrBinaryMissing", err)
			}

			var missing *MissingBinaryError
			if !errors.As(err, &missing) {
				t.Fatalf("error = %v, want *MissingBinaryError", err)
			}
			if filepath.Base(missing.Path) != tc.wantField {
				t.Errorf("missing binary = %q, want %q", missing.Path, tc.wantField)
			}
		})
	}
}

func TestMissingBinaryErrorMentionsTheFileName(t *testing.T) {
	err := &MissingBinaryError{Name: "ffmpeg.exe", Path: `C:\app\ffmpeg.exe`}
	if got := err.Error(); !strings.Contains(got, "ffmpeg.exe") {
		t.Errorf("Error() = %q, want it to name the missing file", got)
	}
}

func TestFirstLineTrimsWhitespace(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"2026.06.09\n", "2026.06.09"},
		{"2026.06.09\r\n", "2026.06.09"},
		{"  2026.06.09  \n", "2026.06.09"},
		{"\n2026.06.09", "2026.06.09"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := firstLine(tc.in); got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseFFmpegVersion(t *testing.T) {
	// ffmpeg -version prints a banner; we want the version token of line one.
	out := "ffmpeg version 9.0-essentials_build-www.gyan.dev Copyright (c) 2000-2026\nbuilt with gcc\n"
	if got, want := parseFFmpegVersion(out), "9.0-essentials_build-www.gyan.dev"; got != want {
		t.Errorf("parseFFmpegVersion() = %q, want %q", got, want)
	}
}

func TestParseFFmpegVersionHandlesEmpty(t *testing.T) {
	if got := parseFFmpegVersion(""); got != "" {
		t.Errorf("parseFFmpegVersion(\"\") = %q, want empty", got)
	}
}

func TestVersionsUsesInjectedRunner(t *testing.T) {
	run := func(_ context.Context, name string, _ ...string) ([]byte, error) {
		switch filepath.Base(name) {
		case "yt-dlp.exe":
			return []byte("2026.06.09\n"), nil
		case "ffmpeg.exe":
			return []byte("ffmpeg version 9.0-test Copyright (c)\n"), nil
		default:
			return nil, errors.New("unexpected binary: " + name)
		}
	}

	got, err := versions(context.Background(), run, Bins{YtDlp: "yt-dlp.exe", FFmpeg: "ffmpeg.exe"})
	if err != nil {
		t.Fatalf("versions() error = %v", err)
	}
	if got.YtDlp != "2026.06.09" {
		t.Errorf("YtDlp = %q, want 2026.06.09", got.YtDlp)
	}
	if got.FFmpeg != "9.0-test" {
		t.Errorf("FFmpeg = %q, want 9.0-test", got.FFmpeg)
	}
}

func TestVersionsSurfacesRunnerError(t *testing.T) {
	run := func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if filepath.Base(name) == "yt-dlp.exe" {
			return nil, errors.New("boom")
		}
		return []byte("ffmpeg version 9.0\n"), nil
	}

	_, err := versions(context.Background(), run, Bins{YtDlp: "yt-dlp.exe", FFmpeg: "ffmpeg.exe"})
	if err == nil {
		t.Fatal("versions() error = nil, want the runner error to surface")
	}
}
