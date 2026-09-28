package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// defaultMaxEntries caps how many history records are retained. Terminal jobs
// are kept newest-first up to this limit.
const defaultMaxEntries = 200

// historyDoc is the on-disk shape. A wrapper object (rather than a bare array)
// leaves room to add fields later without breaking old files.
type historyDoc struct {
	Jobs []Job `json:"jobs"`
}

// History persists finished jobs (completed/failed/canceled) so the list
// survives a restart. It writes atomically and never fails on a corrupt file.
type History struct {
	path string
	// MaxEntries caps retained records; 0 means defaultMaxEntries.
	MaxEntries int
}

// NewHistory returns a History bound to a file path.
func NewHistory(path string) *History {
	return &History{path: path}
}

// Path returns the file this history reads and writes.
func (h *History) Path() string { return h.path }

// limit returns the effective retention cap.
func (h *History) limit() int {
	if h.MaxEntries > 0 {
		return h.MaxEntries
	}
	return defaultMaxEntries
}

// Save writes the given jobs, keeping only the newest `limit` records. Records
// are stored in chronological order (oldest first) to match Queue.List's
// insertion order, so a restored list reads the same as a live one. It writes to
// a temp file and renames it into place so a crash cannot truncate history.
func (h *History) Save(jobs []Job) error {
	kept := trimToNewest(jobs, h.limit())

	doc := historyDoc{Jobs: kept}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("queue: marshal history: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("queue: create history dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("queue: create temp history: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("queue: write temp history: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("queue: sync temp history: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("queue: close temp history: %w", err)
	}
	if err := os.Rename(tmpName, h.path); err != nil {
		return fmt.Errorf("queue: rename history into place: %w", err)
	}
	return nil
}

// Load reads history and normalizes interrupted jobs to `paused` so a job that
// was downloading when the app closed is never silently restarted. A missing or
// corrupt file yields an empty list without error.
func (h *History) Load() ([]Job, error) {
	raw, err := os.ReadFile(h.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("queue: read history: %w", err)
	}

	var doc historyDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		// Corrupt file: treat as empty rather than breaking startup.
		return nil, nil
	}

	out := make([]Job, 0, len(doc.Jobs))
	for _, j := range doc.Jobs {
		out = append(out, normalizeLoaded(j))
	}
	return out, nil
}

// normalizeLoaded repairs a job loaded from disk. Non-terminal states
// (downloading/queued) become paused; terminal states are kept as-is.
func normalizeLoaded(j Job) Job {
	if !isTerminal(j.State) {
		j.State = StatePaused
	}
	return j
}

// trimToNewest keeps the `limit` most recent jobs but returns them in
// chronological order (oldest first), matching the queue's insertion order.
func trimToNewest(jobs []Job, limit int) []Job {
	out := make([]Job, len(jobs))
	copy(out, jobs)

	// Stable sort newest-first, drop the overflow, then restore chronology.
	sort.SliceStable(out, func(i, k int) bool {
		return out[i].CreatedAt > out[k].CreatedAt
	})
	if len(out) > limit {
		out = out[:limit]
	}
	sort.SliceStable(out, func(i, k int) bool {
		return out[i].CreatedAt < out[k].CreatedAt
	})
	return out
}
