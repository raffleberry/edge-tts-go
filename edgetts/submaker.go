package edgetts

import (
	"fmt"
	"time"
)

// SubMaker generates subtitles from WordBoundary/SentenceBoundary events.
type SubMaker struct {
	Cues []Subtitle
	typ  string
}

// Feed adds a WordBoundary or SentenceBoundary chunk.
func (s *SubMaker) Feed(msg TTSChunk) error {
	if msg.Type != ChunkWordBoundary && msg.Type != ChunkSentenceBoundary {
		return fmt.Errorf("invalid message type, expected 'WordBoundary' or 'SentenceBoundary'")
	}
	if s.typ == "" {
		s.typ = string(msg.Type)
	} else if s.typ != string(msg.Type) {
		return fmt.Errorf("expected message type '%s', but got '%s'", s.typ, msg.Type)
	}
	s.Cues = append(s.Cues, Subtitle{
		Index:   len(s.Cues) + 1,
		Start:   time.Duration(msg.Offset*100) * time.Nanosecond,
		End:     time.Duration((msg.Offset+msg.Duration)*100) * time.Nanosecond,
		Content: msg.Text,
	})
	return nil
}

// GetSRT returns the cues formatted as SRT.
func (s *SubMaker) GetSRT() string { return Compose(s.Cues, true, 1, "\n") }

func (s *SubMaker) String() string { return s.GetSRT() }
