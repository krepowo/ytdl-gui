// Package settings owns the persisted application configuration.
//
// The config file lives inside the installation folder, next to the installed
// executable (per product decision). Because the NSIS installer uses per-user
// scope ($LOCALAPPDATA\Programs\...), that folder is writable without admin
// rights. If it is not writable — a machine-wide install, or the app running
// from read-only media — we transparently fall back to
// %APPDATA%\VideoDownloader\config.json so the app never crashes over config.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Download modes.
const (
	ModeVideo = "video"
	ModeAudio = "audio"
)

// appFolderName is used for the %APPDATA% fallback directory.
const appFolderName = "VideoDownloader"

// configFileName is the file name used in both the primary and fallback locations.
const configFileName = "config.json"

// validQualities are the quality values the UI offers. "best" lets yt-dlp pick.
var validQualities = map[string]bool{
	"best": true,
	"1080": true,
	"720":  true,
	"480":  true,
	"360":  true,
}

// validBrowsers are the browsers we can read cookies from. "" means disabled.
var validBrowsers = map[string]bool{
	"":        true,
	"chrome":  true,
	"edge":    true,
	"firefox": true,
	"brave":   true,
}

// Settings is the persisted, user-editable app configuration.
type Settings struct {
	DownloadDir    string `json:"downloadDir"`
	MaxConcurrent  int    `json:"maxConcurrent"`
	DefaultMode    string `json:"defaultMode"`    // "video" | "audio"
	DefaultQuality string `json:"defaultQuality"` // "best" | "1080" | "720" | "480" | "360"
	FilenameTmpl   string `json:"filenameTemplate"`
	CookiesBrowser string `json:"cookiesBrowser"` // "" | "chrome" | "edge" | "firefox" | "brave"
}

// Store loads and saves Settings at a resolved path.
type Store struct {
	path string
}

// NewStore returns a Store bound to the given config file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Path returns the file path this store reads and writes.
func (s *Store) Path() string { return s.path }

// Defaults returns the documented default configuration. It is also the base
// that Load() decodes into, so missing keys keep their default value.
func Defaults() Settings {
	return Settings{
		DownloadDir:    defaultDownloadDir(),
		MaxConcurrent:  2,
		DefaultMode:    ModeVideo,
		DefaultQuality: "best",
		FilenameTmpl:   "%(title)s.%(ext)s",
		CookiesBrowser: "",
	}
}

// defaultDownloadDir returns the user's Downloads folder, falling back to the
// current directory if the OS cannot report a home directory.
func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return filepath.Join(home, "Downloads")
}

// Load reads the config file. A missing file returns defaults without error;
// a corrupt file also returns defaults (never a panic) so a bad edit cannot
// brick the app. Missing keys fall back to their default value.
func (s *Store) Load() (Settings, error) {
	base := Defaults()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return base, nil
		}
		return base, fmt.Errorf("settings: read %s: %w", s.path, err)
	}

	// Decode over the defaults so any key absent from the file keeps its default.
	if err := json.Unmarshal(raw, &base); err != nil {
		// Corrupt file: fall back to defaults rather than failing.
		return Defaults(), nil
	}

	return Validate(base), nil
}

// Save validates the settings and writes them atomically (temp file + rename)
// so a crash mid-write cannot leave a truncated config behind.
func (s *Store) Save(in Settings) error {
	clean := Validate(in)

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("settings: create dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return fmt.Errorf("settings: marshal: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("settings: create temp: %w", err)
	}
	tmpName := tmp.Name()

	// Best-effort cleanup if anything below fails.
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("settings: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("settings: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("settings: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("settings: rename into place: %w", err)
	}

	return nil
}

// Validate returns a repaired copy of s: out-of-range values are clamped and
// unknown enum values are replaced with their defaults. It never returns an
// error — callers can rely on the result being usable.
func Validate(s Settings) Settings {
	d := Defaults()

	if s.MaxConcurrent < 1 {
		s.MaxConcurrent = 1
	}
	if s.MaxConcurrent > 8 {
		s.MaxConcurrent = 8
	}

	if s.DefaultMode != ModeVideo && s.DefaultMode != ModeAudio {
		s.DefaultMode = d.DefaultMode
	}
	if !validQualities[s.DefaultQuality] {
		s.DefaultQuality = d.DefaultQuality
	}
	if !validBrowsers[s.CookiesBrowser] {
		s.CookiesBrowser = d.CookiesBrowser
	}
	if s.FilenameTmpl == "" {
		s.FilenameTmpl = d.FilenameTmpl
	}
	if s.DownloadDir == "" {
		s.DownloadDir = d.DownloadDir
	}

	return s
}

// ConfigPath resolves where config.json should live for the running app.
//
// It prefers the executable's own directory (the install folder). If that
// directory is not writable it falls back to %APPDATA%\VideoDownloader. The
// chosen path is returned so the UI can display which one is active.
func ConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("settings: locate executable: %w", err)
	}
	// Resolve symlinks so we test the real install directory.
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}

	appData, err := os.UserConfigDir()
	if err != nil || appData == "" {
		// Last resort: keep config beside the exe even if we cannot probe it.
		return filepath.Join(filepath.Dir(exe), configFileName), nil
	}

	path := resolveConfigPath(filepath.Dir(exe), appData, isDirWritable)
	return path, nil
}

// resolveConfigPath is the pure decision function behind ConfigPath, factored
// out so it can be tested without touching the real filesystem.
//
//	writable(exeDir) == true  -> <exeDir>\config.json
//	writable(exeDir) == false -> <appData>\VideoDownloader\config.json
func resolveConfigPath(exeDir, appData string, writable func(string) bool) string {
	if writable(exeDir) {
		return filepath.Join(exeDir, configFileName)
	}
	return filepath.Join(appData, appFolderName, configFileName)
}

// isDirWritable reports whether we can create files in dir by probing it with a
// temporary file. This is more reliable than checking permission bits.
func isDirWritable(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}

	f, err := os.CreateTemp(dir, ".write-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
