package queue

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistorySaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h := NewHistory(path)

	jobs := []Job{
		{ID: "a", URL: "u1", Title: "One", Mode: "video", State: StateCompleted, OutputPath: `D:\D\a.mp4`, CreatedAt: 100},
		{ID: "b", URL: "u2", Title: "Two", Mode: "audio", State: StateError, Error: "boom", CreatedAt: 200},
	}
	if err := h.Save(jobs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := h.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load() = %d jobs, want 2", len(got))
	}
	if got[0].ID != "a" || got[0].State != StateCompleted || got[0].OutputPath != `D:\D\a.mp4` {
		t.Errorf("job0 = %+v, want the saved completed job", got[0])
	}
	if got[1].ID != "b" || got[1].State != StateError || got[1].Error != "boom" {
		t.Errorf("job1 = %+v, want the saved failed job", got[1])
	}
}

func TestHistoryLoadMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "does-not-exist.json"))

	got, err := h.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for a missing file", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d jobs, want 0", len(got))
	}
}

func TestHistoryLoadCorruptFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewHistory(path)

	got, err := h.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil (corrupt file must not crash the app)", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d jobs, want 0", len(got))
	}
}

func TestHistorySaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h := NewHistory(path)

	if err := h.Save([]Job{{ID: "a", URL: "u", State: StateCompleted}}); err != nil {
		t.Fatal(err)
	}
	if err := h.Save([]Job{{ID: "b", URL: "u2", State: StateCompleted}}); err != nil {
		t.Fatal(err)
	}

	// No temp files left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "history.json" {
			t.Errorf("unexpected leftover file %q", e.Name())
		}
	}

	got, _ := h.Load()
	if len(got) != 1 || got[0].ID != "b" {
		t.Errorf("Load() = %+v, want only the second save", got)
	}
}

func TestHistoryNormalizesInterruptedJobsToPaused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h := NewHistory(path)

	// Simulate a file written before an unclean shutdown: jobs that were still
	// downloading must come back paused, never auto-started.
	jobs := []Job{
		{ID: "a", URL: "u1", State: StateCompleted},
		{ID: "b", URL: "u2", State: StateError, Error: "x"},
		{ID: "c", URL: "u3", State: StateDownloading},
		{ID: "d", URL: "u4", State: StateQueued},
		{ID: "e", URL: "u5", State: StatePaused},
		{ID: "f", URL: "u6", State: StateCanceled},
	}
	if err := h.Save(jobs); err != nil {
		t.Fatal(err)
	}

	got, err := h.Load()
	if err != nil {
		t.Fatal(err)
	}

	byID := map[string]Job{}
	for _, j := range got {
		byID[j.ID] = j
	}

	if byID["a"].State != StateCompleted {
		t.Errorf("completed job state = %s, want completed (kept)", byID["a"].State)
	}
	if byID["b"].State != StateError {
		t.Errorf("failed job state = %s, want error (kept)", byID["b"].State)
	}
	if byID["c"].State != StatePaused {
		t.Errorf("interrupted downloading job state = %s, want paused", byID["c"].State)
	}
	if byID["d"].State != StatePaused {
		t.Errorf("interrupted queued job state = %s, want paused", byID["d"].State)
	}
	if byID["e"].State != StatePaused {
		t.Errorf("paused job state = %s, want paused", byID["e"].State)
	}
	// A canceled job is terminal and stays canceled.
	if byID["f"].State != StateCanceled {
		t.Errorf("canceled job state = %s, want canceled", byID["f"].State)
	}
}

func TestHistorySaveOmitsRuntimeOnlyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h := NewHistory(path)

	// Build a job with runtime fields populated, then confirm they are absent.
	j := newJob("a", DownloadRequest{URL: "u", OutputDir: `D:\D`, CookiesBrowser: "chrome"}, time.Now())
	j.handle = "some-handle"
	j.setState(StateCompleted)

	if err := h.Save([]Job{j.clone()}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Jobs) != 1 {
		t.Fatalf("saved %d jobs, want 1", len(doc.Jobs))
	}
	for _, leak := range []string{"req", "handle", "outputDir", "cookiesBrowser"} {
		if _, ok := doc.Jobs[0][leak]; ok {
			t.Errorf("history leaks runtime field %q", leak)
		}
	}
}

func TestHistoryPruneCapsRetainedJobs(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(filepath.Join(dir, "history.json"))
	h.MaxEntries = 3

	var jobs []Job
	for i := 0; i < 10; i++ {
		jobs = append(jobs, Job{ID: string(rune('a' + i)), URL: "u", State: StateCompleted, CreatedAt: int64(i)})
	}
	if err := h.Save(jobs); err != nil {
		t.Fatal(err)
	}

	got, err := h.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("Load() = %d jobs, want 3 (capped)", len(got))
	}
	// The most recent (highest CreatedAt) entries are kept.
	if got[len(got)-1].CreatedAt != 9 {
		t.Errorf("last kept job CreatedAt = %d, want 9 (newest)", got[len(got)-1].CreatedAt)
	}
}
