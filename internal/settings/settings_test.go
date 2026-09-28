package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Defaults()

	if d.DownloadDir == "" {
		t.Error("DownloadDir should default to a non-empty path")
	}
	if !strings.HasSuffix(d.DownloadDir, "Downloads") {
		t.Errorf("DownloadDir = %q, want it to end with Downloads", d.DownloadDir)
	}
	if d.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", d.MaxConcurrent)
	}
	if d.DefaultMode != ModeVideo {
		t.Errorf("DefaultMode = %q, want %q", d.DefaultMode, ModeVideo)
	}
	if d.DefaultQuality != "best" {
		t.Errorf("DefaultQuality = %q, want best", d.DefaultQuality)
	}
	if d.FilenameTmpl != "%(title)s.%(ext)s" {
		t.Errorf("FilenameTmpl = %q, want yt-dlp default", d.FilenameTmpl)
	}
	if d.CookiesBrowser != "" {
		t.Errorf("CookiesBrowser = %q, want empty", d.CookiesBrowser)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "config.json"))

	got, err := st.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got != Defaults() {
		t.Errorf("Load() = %+v, want defaults %+v", got, Defaults())
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "config.json"))

	want := Settings{
		DownloadDir:    filepath.Join(t.TempDir(), "videos"),
		MaxConcurrent:  4,
		DefaultMode:    ModeAudio,
		DefaultQuality: "720",
		FilenameTmpl:   "%(id)s.%(ext)s",
		CookiesBrowser: "firefox",
	}
	if err := st.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := st.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadCorruptFileFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := NewStore(path)
	got, err := st.Load()
	if err != nil {
		t.Fatalf("Load() on corrupt file error = %v, want nil (must not panic)", err)
	}
	if got != Defaults() {
		t.Errorf("Load() = %+v, want defaults", got)
	}
}

func TestLoadPartialFileKeepsDefaultsForMissingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	// Only MaxConcurrent is present; every other field must keep its default.
	if err := os.WriteFile(path, []byte(`{"maxConcurrent":6}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.MaxConcurrent != 6 {
		t.Errorf("MaxConcurrent = %d, want 6", got.MaxConcurrent)
	}
	if got.DefaultMode != ModeVideo || got.DefaultQuality != "best" {
		t.Errorf("missing keys did not keep defaults: %+v", got)
	}
}

func TestValidateClampsConcurrency(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero becomes 1", 0, 1},
		{"negative becomes 1", -3, 1},
		{"below range stays", 1, 1},
		{"in range stays", 5, 5},
		{"at ceiling stays", 8, 8},
		{"above ceiling clamps", 99, 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Defaults()
			s.MaxConcurrent = tc.in
			if got := Validate(s).MaxConcurrent; got != tc.want {
				t.Errorf("Validate().MaxConcurrent = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestValidateRepairsInvalidEnums(t *testing.T) {
	s := Settings{
		DownloadDir:    "",
		MaxConcurrent:  2,
		DefaultMode:    "bogus",
		DefaultQuality: "4k",
		FilenameTmpl:   "",
		CookiesBrowser: "netscape",
	}
	got := Validate(s)

	if got.DefaultMode != ModeVideo {
		t.Errorf("DefaultMode = %q, want %q", got.DefaultMode, ModeVideo)
	}
	if got.DefaultQuality != "best" {
		t.Errorf("DefaultQuality = %q, want best", got.DefaultQuality)
	}
	if got.CookiesBrowser != "" {
		t.Errorf("CookiesBrowser = %q, want empty", got.CookiesBrowser)
	}
	if got.FilenameTmpl == "" {
		t.Error("FilenameTmpl should be repaired to the default, not left empty")
	}
	if got.DownloadDir == "" {
		t.Error("DownloadDir should be repaired to the default, not left empty")
	}
}

func TestValidateAcceptsAllDocumentedValues(t *testing.T) {
	for _, q := range []string{"best", "1080", "720", "480", "360"} {
		s := Defaults()
		s.DefaultQuality = q
		if got := Validate(s).DefaultQuality; got != q {
			t.Errorf("quality %q was rejected (got %q)", q, got)
		}
	}
	for _, b := range []string{"", "chrome", "edge", "firefox", "brave"} {
		s := Defaults()
		s.CookiesBrowser = b
		if got := Validate(s).CookiesBrowser; got != b {
			t.Errorf("browser %q was rejected (got %q)", b, got)
		}
	}
	for _, m := range []string{ModeVideo, ModeAudio} {
		s := Defaults()
		s.DefaultMode = m
		if got := Validate(s).DefaultMode; got != m {
			t.Errorf("mode %q was rejected (got %q)", m, got)
		}
	}
}

func TestResolveConfigPathUsesWritableExeDir(t *testing.T) {
	exeDir := `C:\Program Files\Video Downloader`
	appData := `C:\Users\x\AppData\Roaming`

	got := resolveConfigPath(exeDir, appData, func(string) bool { return true })
	want := filepath.Join(exeDir, "config.json")
	if got != want {
		t.Errorf("resolveConfigPath() = %q, want %q", got, want)
	}
}

func TestResolveConfigPathFallsBackWhenReadOnly(t *testing.T) {
	exeDir := `C:\Program Files\Video Downloader`
	appData := `C:\Users\x\AppData\Roaming`

	got := resolveConfigPath(exeDir, appData, func(string) bool { return false })
	want := filepath.Join(appData, "VideoDownloader", "config.json")
	if got != want {
		t.Errorf("resolveConfigPath() = %q, want %q", got, want)
	}
}

func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(filepath.Join(dir, "config.json"))
	if err := st.Save(Defaults()); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "config.json" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "config.json")
	st := NewStore(path)
	if err := st.Save(Defaults()); err != nil {
		t.Fatalf("Save() error = %v, want it to create parent dirs", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSavedFileIsHumanReadableJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := NewStore(path).Save(Defaults()); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}
	for _, key := range []string{"downloadDir", "maxConcurrent", "defaultMode", "defaultQuality"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("saved JSON is missing key %q", key)
		}
	}
}
