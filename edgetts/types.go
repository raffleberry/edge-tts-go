package edgetts

import (
	"fmt"
	"regexp"
)

// Boundary selects which subtitle boundary events the service returns.
type Boundary string

const (
	WordBoundary     Boundary = "WordBoundary"
	SentenceBoundary Boundary = "SentenceBoundary"
)

// ChunkType identifies the kind of streaming chunk.
type ChunkType string

const (
	ChunkAudio            ChunkType = "audio"
	ChunkWordBoundary     ChunkType = "WordBoundary"
	ChunkSentenceBoundary ChunkType = "SentenceBoundary"
)

// TTSChunk is a single unit streamed from the service.
type TTSChunk struct {
	Type     ChunkType
	Data     []byte // only for audio
	Offset   int64  // ticks (100ns), only for boundaries
	Duration int64  // ticks (100ns), only for boundaries
	Text     string // only for boundaries
}

// VoiceTag holds voice content categories and personalities.
type VoiceTag struct {
	ContentCategories  []string `json:"ContentCategories"`
	VoicePersonalities []string `json:"VoicePersonalities"`
}

// Voice describes a single available voice.
type Voice struct {
	Name           string   `json:"Name"`
	ShortName      string   `json:"ShortName"`
	Gender         string   `json:"Gender"`
	Locale         string   `json:"Locale"`
	SuggestedCodec string   `json:"SuggestedCodec"`
	FriendlyName   string   `json:"FriendlyName"`
	Status         string   `json:"Status"`
	VoiceTag       VoiceTag `json:"VoiceTag"`
}

// ManagerVoice is a Voice with the derived Language field.
type ManagerVoice struct {
	Voice
	Language string `json:"Language"`
}

var (
	voiceShortRe = regexp.MustCompile(`^([a-z]{2,})-([A-Z]{2,})-(.+Neural)$`)
	voiceFullRe  = regexp.MustCompile(`^Microsoft Server Speech Text to Speech Voice \(.+,.+\)$`)
	rateRe       = regexp.MustCompile(`^[+-]\d+%$`)
	pitchRe      = regexp.MustCompile(`^[+-]\d+Hz$`)
)

// TTSConfig holds validated TTS parameters.
type TTSConfig struct {
	Voice    string
	Rate     string
	Volume   string
	Pitch    string
	Boundary Boundary
}

// NewTTSConfig validates parameters and returns a TTSConfig.
// It mirrors Python's TTSConfig.__post_init__ including the short voice
// name expansion to the full "Microsoft Server Speech..." form.
func NewTTSConfig(voice, rate, volume, pitch string, boundary Boundary) (TTSConfig, error) {
	tc := TTSConfig{Voice: voice, Rate: rate, Volume: volume, Pitch: pitch, Boundary: boundary}
	if m := voiceShortRe.FindStringSubmatch(voice); m != nil {
		lang, region, name := m[1], m[2], m[3]
		if i := indexOf(name, "-"); i != -1 {
			region = region + "-" + name[:i]
			name = name[i+1:]
		}
		tc.Voice = "Microsoft Server Speech Text to Speech Voice (" + lang + "-" + region + ", " + name + ")"
	}
	if !voiceFullRe.MatchString(tc.Voice) {
		return TTSConfig{}, fmt.Errorf("invalid voice '%s'", voice)
	}
	if !rateRe.MatchString(tc.Rate) {
		return TTSConfig{}, fmt.Errorf("invalid rate '%s'", rate)
	}
	if !rateRe.MatchString(tc.Volume) {
		return TTSConfig{}, fmt.Errorf("invalid volume '%s'", volume)
	}
	if !pitchRe.MatchString(tc.Pitch) {
		return TTSConfig{}, fmt.Errorf("invalid pitch '%s'", pitch)
	}
	if tc.Boundary != WordBoundary && tc.Boundary != SentenceBoundary {
		return TTSConfig{}, fmt.Errorf("invalid boundary '%s'", boundary)
	}
	return tc, nil
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
