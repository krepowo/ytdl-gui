package queue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCanTransitionLegal(t *testing.T) {
	legal := []struct{ from, to State }{
		{StateQueued, StateDownloading},
		{StateQueued, StateCanceled},
		{StateDownloading, StatePaused},
		{StateDownloading, StateCompleted},
		{StateDownloading, StateError},
		{StateDownloading, StateCanceled},
		{StatePaused, StateDownloading},
		{StatePaused, StateCanceled},
	}
	for _, tc := range legal {
		if !canTransition(tc.from, tc.to) {
			t.Errorf("canTransition(%s, %s) = false, want true", tc.from, tc.to)
		}
	}
}

func TestCanTransitionIllegal(t *testing.T) {
	illegal := []struct{ from, to State }{
		{StateQueued, StatePaused},         // cannot pause before it runs
		{StateQueued, StateCompleted},      // cannot complete before it runs
		{StateQueued, StateError},          // cannot fail before it runs
		{StatePaused, StateCompleted},      // must resume before finishing
		{StatePaused, StateError},          //
		{StateCompleted, StateDownloading}, // terminal states are final
		{StateCompleted, StateCanceled},
		{StateError, StateDownloading},
		{StateError, StateCanceled},
		{StateCanceled, StateDownloading},
		{StateCanceled, StatePaused},
		{StateDownloading, StateQueued}, // no going back to the queue
	}
	for _, tc := range illegal {
		if canTransition(tc.from, tc.to) {
			t.Errorf("canTransition(%s, %s) = true, want false", tc.from, tc.to)
		}
	}
}

func TestCanTransitionRejectsSameState(t *testing.T) {
	// A same-state "transition" is a no-op, not a legal move.
	for _, s := range []State{StateQueued, StateDownloading, StatePaused, StateCompleted, StateError, StateCanceled} {
		if canTransition(s, s) {
			t.Errorf("canTransition(%s, %s) = true, want false (no-op)", s, s)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := map[State]bool{StateCompleted: true, StateError: true, StateCanceled: true}
	for _, s := range []State{StateQueued, StateDownloading, StatePaused, StateCompleted, StateError, StateCanceled} {
		if got := isTerminal(s); got != terminal[s] {
			t.Errorf("isTerminal(%s) = %v, want %v", s, got, terminal[s])
		}
	}
}

func TestNewJobStartsQueued(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	j := newJob("id-1", DownloadRequest{
		URL:       "https://example.com/v",
		OutputDir: `D:\Downloads`,
		Mode:      "video",
		FormatID:  "248",
	}, now)

	if j.ID != "id-1" {
		t.Errorf("ID = %q, want id-1", j.ID)
	}
	if j.State != StateQueued {
		t.Errorf("State = %s, want queued", j.State)
	}
	if j.CreatedAt != now.Unix() {
		t.Errorf("CreatedAt = %d, want %d", j.CreatedAt, now.Unix())
	}
	if j.URL != "https://example.com/v" {
		t.Errorf("URL = %q, want the request URL", j.URL)
	}
	if j.Percent != 0 || j.DownloadedBytes != 0 || j.TotalBytes != 0 {
		t.Errorf("progress fields = %v, want zeroed", j)
	}
}

func TestJobSetStateValidatesTransitions(t *testing.T) {
	j := newJob("id-1", DownloadRequest{URL: "u"}, time.Now())

	if err := j.setState(StateDownloading); err != nil {
		t.Fatalf("queued->downloading = %v, want nil", err)
	}
	if err := j.setState(StateCompleted); err != nil {
		t.Fatalf("downloading->completed = %v, want nil", err)
	}
	// Terminal: any further move must be rejected and leave the state untouched.
	if err := j.setState(StateDownloading); err == nil {
		t.Error("completed->downloading = nil, want an error")
	}
	if j.State != StateCompleted {
		t.Errorf("State = %s, want completed (unchanged after rejected transition)", j.State)
	}
}

func TestJobSetStateRejectsIllegalMove(t *testing.T) {
	j := newJob("id-1", DownloadRequest{URL: "u"}, time.Now())
	if err := j.setState(StatePaused); err == nil {
		t.Error("queued->paused = nil, want an error")
	}
	if j.State != StateQueued {
		t.Errorf("State = %s, want queued (unchanged)", j.State)
	}
}

func TestJobCloneIsIndependentAndJSONSafe(t *testing.T) {
	j := newJob("id-1", DownloadRequest{URL: "u", OutputDir: `D:\D`}, time.Now())
	j.setState(StateDownloading)
	j.Percent = 42.5
	j.DownloadedBytes = 100
	j.TotalBytes = 200
	j.SpeedBps = 50
	j.ETASec = 7
	j.OutputPath = `D:\D\clip.mp4`

	c := j.clone()
	c.Percent = 99

	if j.Percent != 42.5 {
		t.Error("clone shares state with the original (mutating the clone changed the source)")
	}

	// The clone must marshal to the documented wire shape.
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"id", "url", "title", "mode", "formatId", "state", "percent",
		"downloadedBytes", "totalBytes", "speedBps", "etaSec", "outputPath",
		"error", "createdAt",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("marshalled job is missing key %q", key)
		}
	}
	// Runtime-only fields must not leak into the wire format.
	for _, key := range []string{"handle", "req", "OutputDir", "outputDir"} {
		if _, ok := got[key]; ok {
			t.Errorf("marshalled job leaks runtime field %q", key)
		}
	}
	if got["state"] != "downloading" {
		t.Errorf("state = %v, want downloading", got["state"])
	}
}
