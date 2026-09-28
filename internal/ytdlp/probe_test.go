package ytdlp

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// youtubeJSON is a trimmed real-shaped YouTube probe result: separate video-only
// and audio-only streams, all optional metadata present.
const youtubeJSON = `{
  "id": "dQw4w9WgXcQ",
  "title": "Rick Astley - Never Gonna Give You Up",
  "uploader": "Rick Astley",
  "duration": 213.0,
  "thumbnail": "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
  "extractor": "youtube",
  "extractor_key": "Youtube",
  "webpage_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "is_live": false,
  "formats": [
    {"format_id": "137", "ext": "mp4", "height": 1080, "width": 1920, "vcodec": "avc1.640028", "acodec": "none", "filesize": 104857600, "fps": 30, "tbr": 2000, "format_note": "1080p"},
    {"format_id": "136", "ext": "mp4", "height": 720, "width": 1280, "vcodec": "avc1.4d401f", "acodec": "none", "filesize": 52428800, "fps": 30, "tbr": 1200, "format_note": "720p"},
    {"format_id": "135", "ext": "mp4", "height": 480, "width": 854, "vcodec": "avc1.4d401e", "acodec": "none", "filesize": 26214400, "fps": 30, "tbr": 800, "format_note": "480p"},
    {"format_id": "140", "ext": "m4a", "vcodec": "none", "acodec": "mp4a.40.2", "filesize": 3400000, "abr": 128, "format_note": "medium"},
    {"format_id": "251", "ext": "webm", "vcodec": "none", "acodec": "opus", "filesize": 3000000, "abr": 160, "format_note": "medium"},
    {"format_id": "18", "ext": "mp4", "height": 360, "width": 640, "vcodec": "avc1.42001E", "acodec": "mp4a.40.2", "filesize": 12000000, "fps": 30, "format_note": "360p"}
  ]
}`

// genericJSON is a generic-extractor page: combined A/V only, and the optional
// metadata (duration, thumbnail, uploader) is missing entirely.
const genericJSON = `{
  "id": "abc123",
  "title": "Some Page Video",
  "extractor": "generic",
  "extractor_key": "Generic",
  "webpage_url": "https://example.com/video",
  "formats": [
    {"format_id": "0", "ext": "mp4", "height": 480, "width": 854, "vcodec": "h264", "acodec": "aac", "filesize": 9000000, "format_note": "480p"}
  ]
}`

// audioOnlyJSON models an audio-only source (e.g. a podcast page).
const audioOnlyJSON = `{
  "id": "ep1",
  "title": "Episode 1",
  "extractor": "soundcloud",
  "extractor_key": "Soundcloud",
  "webpage_url": "https://soundcloud.com/x/ep1",
  "duration": 1800,
  "formats": [
    {"format_id": "hls_mp3", "ext": "mp3", "vcodec": "none", "acodec": "mp3", "abr": 128, "format_note": "128kbps"}
  ]
}`

// liveJSON models a live stream (is_live true, no fixed duration).
const liveJSON = `{
  "id": "live1",
  "title": "Live Now",
  "extractor": "twitch",
  "extractor_key": "Twitch",
  "webpage_url": "https://twitch.tv/x",
  "is_live": true,
  "formats": [
    {"format_id": "best", "ext": "mp4", "height": 1080, "vcodec": "h264", "acodec": "aac", "format_note": "1080p60"}
  ]
}`

func TestParseProbeExtractsCoreMetadata(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatalf("parseProbe() error = %v", err)
	}

	if info.ID != "dQw4w9WgXcQ" {
		t.Errorf("ID = %q", info.ID)
	}
	if info.Title == "" {
		t.Error("Title is empty")
	}
	if info.Uploader != "Rick Astley" {
		t.Errorf("Uploader = %q", info.Uploader)
	}
	if info.DurationSec != 213 {
		t.Errorf("DurationSec = %v, want 213", info.DurationSec)
	}
	if info.Thumbnail == "" {
		t.Error("Thumbnail is empty")
	}
	if info.WebpageURL == "" {
		t.Error("WebpageURL is empty")
	}
	if info.IsLive {
		t.Error("IsLive = true, want false")
	}
}

func TestParseProbeReadsExtractorForSourceBadge(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"youtube", youtubeJSON, "youtube"},
		{"generic", genericJSON, "generic"},
		{"soundcloud", audioOnlyJSON, "soundcloud"},
		{"twitch", liveJSON, "twitch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parseProbe([]byte(tc.json))
			if err != nil {
				t.Fatal(err)
			}
			if info.Extractor != tc.want {
				t.Errorf("Extractor = %q, want %q", info.Extractor, tc.want)
			}
		})
	}
}

func TestParseProbeFallsBackToExtractorKey(t *testing.T) {
	// Some results only carry extractor_key; the source badge must still work.
	const only = `{"id":"x","title":"t","extractor_key":"Vimeo","webpage_url":"u","formats":[]}`
	info, err := parseProbe([]byte(only))
	if err != nil {
		t.Fatal(err)
	}
	if info.Extractor != "vimeo" {
		t.Errorf("Extractor = %q, want vimeo (lowercased from extractor_key)", info.Extractor)
	}
}

func TestParseProbeToleratesMissingOptionalMetadata(t *testing.T) {
	info, err := parseProbe([]byte(genericJSON))
	if err != nil {
		t.Fatalf("parseProbe() error = %v, want nil for a generic page", err)
	}
	if info.DurationSec != 0 {
		t.Errorf("DurationSec = %v, want 0 when absent", info.DurationSec)
	}
	if info.Thumbnail != "" {
		t.Errorf("Thumbnail = %q, want empty when absent", info.Thumbnail)
	}
	if info.Uploader != "" {
		t.Errorf("Uploader = %q, want empty when absent", info.Uploader)
	}
}

func TestParseProbeRejectsInvalidJSON(t *testing.T) {
	if _, err := parseProbe([]byte("{not json")); err == nil {
		t.Fatal("parseProbe() error = nil, want an error for invalid JSON")
	}
}

// realYoutubeJSON mirrors actual yt-dlp output: it includes storyboard entries
// (video_ext "none" but a height) and uses video_ext/audio_ext to mark streams.
const realYoutubeJSON = `{
  "id": "dQw4w9WgXcQ",
  "title": "Real YouTube",
  "extractor": "youtube",
  "webpage_url": "https://youtu.be/x",
  "formats": [
    {"format_id": "sb1", "ext": "mhtml", "height": 90, "vcodec": "none", "acodec": "none", "video_ext": "none", "audio_ext": "none", "format_note": "storyboard"},
    {"format_id": "602", "ext": "mp4", "height": 144, "vcodec": "vp09.00.10.08", "acodec": "none", "video_ext": "mp4", "audio_ext": "none"},
    {"format_id": "248", "ext": "webm", "height": 1080, "vcodec": "vp9", "acodec": "none", "video_ext": "webm", "audio_ext": "none"},
    {"format_id": "139", "ext": "m4a", "vcodec": "none", "acodec": "mp4a.40.5", "video_ext": "none", "audio_ext": "m4a", "abr": 48.8},
    {"format_id": "251", "ext": "webm", "vcodec": "none", "acodec": "opus", "video_ext": "none", "audio_ext": "webm", "abr": 160.0}
  ]
}`

// realDirectJSON mirrors a direct media URL: codecs are null, height/resolution
// are null, and only video_ext/audio_ext signal the streams.
const realDirectJSON = `{
  "id": "clip",
  "title": "clip",
  "extractor": "generic",
  "extractor_key": "Generic",
  "webpage_url": "https://example.com/clip.mp4",
  "duration": null,
  "formats": [
    {"format_id": "mp4", "ext": "mp4", "vcodec": null, "acodec": null, "video_ext": "mp4", "audio_ext": "none", "format_note": "mp4 - unknown"}
  ]
}`

func TestParseProbeExcludesStoryboardsFromVideoOptions(t *testing.T) {
	info, err := parseProbe([]byte(realYoutubeJSON))
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range info.VideoOptions {
		if o.Height == 90 {
			t.Fatal("storyboard (height 90, video_ext none) leaked into VideoOptions")
		}
	}
	// Only 144 and 1080 are real video streams.
	if len(info.VideoOptions) != 2 {
		t.Fatalf("VideoOptions = %d entries, want 2 (144, 1080): %+v", len(info.VideoOptions), info.VideoOptions)
	}
}

func TestParseProbeHandlesNullCodecsOnDirectMedia(t *testing.T) {
	info, err := parseProbe([]byte(realDirectJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.VideoOptions) != 1 {
		t.Fatalf("VideoOptions = %d, want 1 (direct media must still be downloadable)", len(info.VideoOptions))
	}
	opt := info.VideoOptions[0]
	if opt.FormatID != "mp4" {
		t.Errorf("FormatID = %q, want mp4", opt.FormatID)
	}
	if opt.Label == "" {
		t.Error("Label is empty; the picker needs something to show")
	}
	// A direct file is a single combined stream -> no merge.
	if opt.NeedsMerge {
		t.Error("direct media should not need merging")
	}
}

func TestParseProbeUsesVideoExtWhenCodecsMissing(t *testing.T) {
	// video_ext present, codecs absent: must still be recognised as video.
	const j = `{"id":"x","title":"t","extractor":"generic","webpage_url":"u","formats":[
	  {"format_id":"a","ext":"mp4","video_ext":"mp4","audio_ext":"none","height":720}
	]}`
	info, err := parseProbe([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.VideoOptions) != 1 {
		t.Fatalf("VideoOptions = %d, want 1", len(info.VideoOptions))
	}
	if info.VideoOptions[0].Height != 720 {
		t.Errorf("Height = %d, want 720", info.VideoOptions[0].Height)
	}
}

func TestParseProbeParsesResolutionWhenHeightAbsent(t *testing.T) {
	const j = `{"id":"x","title":"t","extractor":"generic","webpage_url":"u","formats":[
	  {"format_id":"a","ext":"mp4","video_ext":"mp4","resolution":"1920x1080"}
	]}`
	info, err := parseProbe([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.VideoOptions) != 1 {
		t.Fatalf("VideoOptions = %d, want 1", len(info.VideoOptions))
	}
	if got := info.VideoOptions[0].Height; got != 1080 {
		t.Errorf("Height = %d, want 1080 (parsed from resolution)", got)
	}
	if got := info.VideoOptions[0].Width; got != 1920 {
		t.Errorf("Width = %d, want 1920", got)
	}
}

func TestParseProbeDerivesVideoQualitiesDynamically(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatal(err)
	}

	// YouTube fixture offers separate video-only streams at 1080/720/480 plus a
	// combined 360. The picker must list exactly those heights, highest first.
	var heights []int
	for _, o := range info.VideoOptions {
		heights = append(heights, o.Height)
	}
	want := []int{1080, 720, 480, 360}
	if len(heights) != len(want) {
		t.Fatalf("VideoOptions heights = %v, want %v", heights, want)
	}
	for i := range want {
		if heights[i] != want[i] {
			t.Fatalf("VideoOptions heights = %v, want %v (descending)", heights, want)
		}
	}
}

func TestParseProbeMarksMergedStreams(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatal(err)
	}

	byHeight := map[int]FormatOption{}
	for _, o := range info.VideoOptions {
		byHeight[o.Height] = o
	}

	// 1080 is video-only -> needs merging with a separate audio stream.
	if !byHeight[1080].NeedsMerge {
		t.Error("1080p should be marked NeedsMerge (video-only stream)")
	}
	// 360 is a combined A/V stream -> no merge needed.
	if byHeight[360].NeedsMerge {
		t.Error("360p should NOT be marked NeedsMerge (combined stream)")
	}
}

func TestParseProbeListsAudioOptions(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.AudioOptions) == 0 {
		t.Fatal("AudioOptions is empty, want the two audio-only formats")
	}
	for _, o := range info.AudioOptions {
		if o.Ext == "" {
			t.Error("audio option has empty Ext")
		}
	}
}

func TestParseProbeAudioOnlySourceHasNoVideoOptions(t *testing.T) {
	info, err := parseProbe([]byte(audioOnlyJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.VideoOptions) != 0 {
		t.Errorf("VideoOptions = %v, want none for an audio-only source", info.VideoOptions)
	}
	if len(info.AudioOptions) == 0 {
		t.Error("AudioOptions is empty, want the mp3 format")
	}
}

func TestParseProbeDetectsLive(t *testing.T) {
	info, err := parseProbe([]byte(liveJSON))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsLive {
		t.Error("IsLive = false, want true")
	}
}

func TestFormatOptionLabelIsHumanReadable(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range info.VideoOptions {
		if !strings.Contains(o.Label, "p") {
			t.Errorf("video label %q should contain the resolution, e.g. 1080p", o.Label)
		}
	}
}

// noHeightVideoJSON models real YouTube formats where height is null: audio
// formats 233/234 (audio_ext mp4, no abr) and a video format with video_ext set
// but no height. Both made the picker render "mp4 • mp4".
const noHeightVideoJSON = `{
  "id": "x", "title": "T", "extractor": "generic", "webpage_url": "u",
  "formats": [
    {"format_id": "233", "ext": "mp4", "vcodec": "none", "acodec": "none", "video_ext": "none", "audio_ext": "mp4"},
    {"format_id": "999", "ext": "mp4", "vcodec": "avc1", "acodec": "none", "video_ext": "mp4", "audio_ext": "none", "tbr": 500}
  ]
}`

func TestAudioLabelNeverEqualsBareExt(t *testing.T) {
	info, err := parseProbe([]byte(noHeightVideoJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range info.AudioOptions {
		if strings.EqualFold(o.Label, o.Ext) {
			t.Errorf("audio label %q equals its ext %q; picker would show %q", o.Label, o.Ext, o.Label+" • "+o.Ext)
		}
	}
}

func TestVideoLabelNeverEqualsBareExt(t *testing.T) {
	info, err := parseProbe([]byte(noHeightVideoJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range info.VideoOptions {
		if strings.EqualFold(o.Label, o.Ext) {
			t.Errorf("video label %q equals its ext %q; picker would show %q", o.Label, o.Ext, o.Label+" • "+o.Ext)
		}
	}
}

func TestAudioOptionsAlwaysOfferMp3(t *testing.T) {
	for _, fixture := range []string{youtubeJSON, audioOnlyJSON} {
		info, err := parseProbe([]byte(fixture))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, o := range info.AudioOptions {
			if strings.EqualFold(o.Ext, "mp3") || strings.Contains(strings.ToUpper(o.Label), "MP3") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AudioOptions for fixture lack an MP3 choice: %+v", info.AudioOptions)
		}
	}
}

func TestMp3OptionUsesBestAudio(t *testing.T) {
	info, err := parseProbe([]byte(youtubeJSON))
	if err != nil {
		t.Fatal(err)
	}
	// The synthetic MP3 option selects bestaudio; MUI's Select needs a non-empty
	// value, so it is not "" (empty would render a blank control).
	for _, o := range info.AudioOptions {
		if strings.EqualFold(o.Ext, "mp3") && o.FormatID != "bestaudio" {
			t.Errorf("mp3 option FormatID = %q, want %q", o.FormatID, "bestaudio")
		}
	}
}

func TestProbeUsesInjectedRunnerAndPassesExpectedArgs(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		gotArgs = args
		return []byte(youtubeJSON), nil
	}

	info, err := probe(context.Background(), run, Bins{YtDlp: "yt-dlp.exe"},
		"https://youtu.be/x", ProbeOptions{})
	if err != nil {
		t.Fatalf("probe() error = %v", err)
	}
	if info.Title == "" {
		t.Error("probe() did not parse the result")
	}

	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"-J", "--no-warnings", "--no-playlist", "https://youtu.be/x"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "--cookies-from-browser") {
		t.Errorf("args %q should not include cookies when none configured", joined)
	}
}

func TestProbePassesCookiesBrowserWhenConfigured(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		gotArgs = args
		return []byte(youtubeJSON), nil
	}

	_, err := probe(context.Background(), run, Bins{YtDlp: "yt-dlp.exe"},
		"https://youtu.be/x", ProbeOptions{CookiesBrowser: "firefox"})
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--cookies-from-browser firefox") {
		t.Errorf("args %q missing the cookies flag", joined)
	}
}

func TestProbeWrapsRunnerError(t *testing.T) {
	run := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("network down")
	}
	_, err := probe(context.Background(), run, Bins{YtDlp: "yt-dlp.exe"}, "u", ProbeOptions{})
	if err == nil {
		t.Fatal("probe() error = nil, want the runner error to surface")
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Errorf("error = %v, want it to include the underlying cause", err)
	}
}
