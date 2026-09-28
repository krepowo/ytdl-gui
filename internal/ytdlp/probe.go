package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ProbeOptions controls how a probe is performed.
type ProbeOptions struct {
	// CookiesBrowser, when non-empty, is passed as --cookies-from-browser so
	// age-gated or login-only sites can be probed. Values: chrome, edge,
	// firefox, brave.
	CookiesBrowser string
}

// MediaInfo is the metadata we render for a probed URL. Every optional field may
// be absent depending on the site (live streams have no duration, generic pages
// often have no thumbnail or uploader), so the UI must degrade gracefully.
type MediaInfo struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Uploader     string         `json:"uploader"`
	DurationSec  float64        `json:"duration"`
	Thumbnail    string         `json:"thumbnail"`
	Extractor    string         `json:"extractor"` // source badge, e.g. "youtube", "twitter", "generic"
	WebpageURL   string         `json:"webpageUrl"`
	IsLive       bool           `json:"isLive"`
	VideoOptions []FormatOption `json:"videoOptions"`
	AudioOptions []FormatOption `json:"audioOptions"`
}

// FormatOption is one selectable choice in the format picker. The list is built
// from whatever the site actually returns — never a fixed quality ladder — so it
// works for combined-only pages and separate video/audio sources alike.
type FormatOption struct {
	FormatID   string  `json:"formatId"`
	Label      string  `json:"label"`
	Ext        string  `json:"ext"`
	Height     int     `json:"height"`     // 0 when the site reports no resolution
	Width      int     `json:"width"`      // 0 when unknown
	FPS        float64 `json:"fps"`        // 0 when unknown
	Bitrate    float64 `json:"bitrate"`    // tbr for video, abr for audio (kbps)
	Filesize   int64   `json:"filesize"`   // 0 when unknown
	NeedsMerge bool    `json:"needsMerge"` // video-only stream: merge with best audio
}

// probeResult mirrors the subset of yt-dlp's `-J` output we consume. Unknown
// fields are ignored, and missing fields decode to their zero value.
type probeResult struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Uploader     string        `json:"uploader"`
	Channel      string        `json:"channel"`
	Duration     float64       `json:"duration"`
	Thumbnail    string        `json:"thumbnail"`
	Extractor    string        `json:"extractor"`
	ExtractorKey string        `json:"extractor_key"`
	WebpageURL   string        `json:"webpage_url"`
	IsLive       bool          `json:"is_live"`
	Formats      []probeFormat `json:"formats"`
}

// probeFormat mirrors one entry of yt-dlp's `formats` array.
//
// Field notes learned from real output:
//   - vcodec/acodec may be null (generic/direct URLs), "none" (absent stream),
//     or a codec string.
//   - video_ext/audio_ext are the RELIABLE presence indicators ("none" = absent).
//   - height may be null for direct media; resolution is then null too.
//   - storyboard/thumbnail entries have a height but video_ext == "none", so
//     classifying on video_ext (not height) keeps them out of the quality list.
type probeFormat struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	Height         int     `json:"height"`
	Width          int     `json:"width"`
	Resolution     string  `json:"resolution"`
	FPS            float64 `json:"fps"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	VideoExt       string  `json:"video_ext"`
	AudioExt       string  `json:"audio_ext"`
	Filesize       int64   `json:"filesize"`
	FilesizeApprox int64   `json:"filesize_approx"`
	TBR            float64 `json:"tbr"`
	ABR            float64 `json:"abr"`
	FormatNote     string  `json:"format_note"`
}

// Probe fetches metadata and the selectable formats for any URL yt-dlp supports.
// It never assumes a provider: the source is read from the result's extractor.
func Probe(ctx context.Context, bins Bins, url string, opts ProbeOptions) (*MediaInfo, error) {
	return probe(ctx, execRunner, bins, url, opts)
}

// probe is Probe with an injected runner, so tests need no real binary/network.
func probe(ctx context.Context, run runner, bins Bins, url string, opts ProbeOptions) (*MediaInfo, error) {
	args := []string{"-J", "--no-warnings", "--no-playlist"}
	if opts.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesBrowser)
	}
	args = append(args, url)

	out, err := run(ctx, bins.YtDlp, args...)
	if err != nil {
		return nil, fmt.Errorf("ytdlp: probe %q: %w", url, err)
	}

	info, err := parseProbe(out)
	if err != nil {
		return nil, fmt.Errorf("ytdlp: probe %q: %w", url, err)
	}
	return info, nil
}

// parseProbe turns yt-dlp's JSON into MediaInfo with derived format options.
func parseProbe(raw []byte) (*MediaInfo, error) {
	var r probeResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decode probe result: %w", err)
	}

	info := &MediaInfo{
		ID:          r.ID,
		Title:       r.Title,
		Uploader:    firstNonEmpty(r.Uploader, r.Channel),
		DurationSec: r.Duration,
		Thumbnail:   r.Thumbnail,
		Extractor:   normalizeExtractor(r.Extractor, r.ExtractorKey),
		WebpageURL:  r.WebpageURL,
		IsLive:      r.IsLive,
	}

	// Whether the site exposes standalone audio streams decides if video-only
	// formats must be merged. A combined-only source (e.g. a direct .mp4) needs
	// no merge, even though yt-dlp may not report its audio stream separately.
	separateAudio := false
	for _, f := range r.Formats {
		if !hasVideo(f) && hasAudio(f) {
			separateAudio = true
			break
		}
	}

	info.VideoOptions = buildVideoOptions(r.Formats, separateAudio)
	info.AudioOptions = buildAudioOptions(r.Formats)
	return info, nil
}

// normalizeExtractor picks a lowercase source name for the badge. It prefers the
// friendly `extractor` value and falls back to `extractor_key`.
func normalizeExtractor(extractor, key string) string {
	if extractor != "" {
		return strings.ToLower(extractor)
	}
	return strings.ToLower(key)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// isAbsent reports whether a codec/stream field means "no stream". yt-dlp uses
// the literal "none" and may also emit null (which decodes to "").
func isAbsent(s string) bool { return s == "" || s == "none" }

// hasVideo reports whether a format carries a video stream. video_ext is the
// reliable indicator; it excludes storyboards (video_ext == "none") even though
// they report a height.
func hasVideo(f probeFormat) bool {
	if f.VideoExt != "" {
		return !isAbsent(f.VideoExt)
	}
	// Fall back to the codec field when video_ext is not reported at all.
	return !isAbsent(f.VCodec)
}

// hasAudio reports whether a format carries an audio stream.
func hasAudio(f probeFormat) bool {
	if f.AudioExt != "" {
		return !isAbsent(f.AudioExt)
	}
	return !isAbsent(f.ACodec)
}

// effectiveHeight returns the vertical resolution, preferring the `height` field
// and falling back to parsing `resolution` ("1920x1080"). Returns 0 if unknown.
func effectiveHeight(f probeFormat) int {
	if f.Height > 0 {
		return f.Height
	}
	if f.Resolution != "" {
		if _, h, ok := strings.Cut(f.Resolution, "x"); ok {
			if n, err := strconv.Atoi(h); err == nil {
				return n
			}
		}
	}
	return 0
}

// effectiveWidth mirrors effectiveHeight for the horizontal axis.
func effectiveWidth(f probeFormat) int {
	if f.Width > 0 {
		return f.Width
	}
	if f.Resolution != "" {
		if w, _, ok := strings.Cut(f.Resolution, "x"); ok {
			if n, err := strconv.Atoi(w); err == nil {
				return n
			}
		}
	}
	return 0
}

// buildVideoOptions derives one option per distinct resolution, highest first.
// When the site exposes separate audio streams, video-only entries are marked
// NeedsMerge so the engine merges them with the best audio.
func buildVideoOptions(formats []probeFormat, separateAudio bool) []FormatOption {
	// Keep the best entry per resolution (highest bitrate wins for duplicates).
	best := map[int]FormatOption{}

	for _, f := range formats {
		if !hasVideo(f) {
			continue
		}

		height := effectiveHeight(f)
		opt := FormatOption{
			FormatID: f.FormatID,
			Label:    videoLabel(f, height),
			Ext:      f.Ext,
			Height:   height,
			Width:    effectiveWidth(f),
			FPS:      f.FPS,
			Bitrate:  f.TBR,
			Filesize: effectiveFilesize(f),
			// Only merge when the site actually offers separate audio.
			NeedsMerge: separateAudio && !hasAudio(f),
		}

		existing, ok := best[height]
		if !ok || opt.Bitrate > existing.Bitrate {
			best[height] = opt
		}
	}

	out := make([]FormatOption, 0, len(best))
	for _, o := range best {
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Height != out[j].Height {
			return out[i].Height > out[j].Height
		}
		return out[i].Bitrate > out[j].Bitrate
	})
	return out
}

// videoLabel builds a human label such as "1080p60". When no resolution is known
// (direct media), it falls back to the format note or container so the picker
// still shows something meaningful.
func videoLabel(f probeFormat, height int) string {
	if height <= 0 {
		if f.FormatNote != "" {
			return f.FormatNote
		}
		if f.Ext != "" {
			return f.Ext
		}
		return "video"
	}
	label := fmt.Sprintf("%dp", height)
	if f.FPS >= 50 {
		label += strconv.Itoa(int(f.FPS))
	}
	return label
}

// buildAudioOptions derives audio-only options, highest bitrate first.
func buildAudioOptions(formats []probeFormat) []FormatOption {
	var out []FormatOption

	for _, f := range formats {
		if hasVideo(f) || !hasAudio(f) {
			continue // audio-only formats only
		}
		out = append(out, FormatOption{
			FormatID: f.FormatID,
			Label:    audioLabel(f),
			Ext:      f.Ext,
			Bitrate:  f.ABR,
			Filesize: effectiveFilesize(f),
		})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Bitrate > out[j].Bitrate })
	return out
}

// audioLabel builds a label such as "mp3 128kbps".
func audioLabel(f probeFormat) string {
	ext := f.Ext
	if ext == "" {
		ext = "audio"
	}
	if f.ABR > 0 {
		return fmt.Sprintf("%s %dkbps", ext, int(f.ABR))
	}
	return ext
}

// effectiveFilesize prefers the exact size and falls back to yt-dlp's estimate.
func effectiveFilesize(f probeFormat) int64 {
	if f.Filesize > 0 {
		return f.Filesize
	}
	return f.FilesizeApprox
}
