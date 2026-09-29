package edgetts

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

var multiWSRegex = regexp.MustCompile(`\n\n+`)

// Subtitle holds the metadata for a single subtitle cue.
type Subtitle struct {
	Index   int
	Start   time.Duration
	End     time.Duration
	Content string
}

// Less orders subtitles by start, end, then index.
func (s Subtitle) Less(o Subtitle) bool {
	if s.Start != o.Start {
		return s.Start < o.Start
	}
	if s.End != o.End {
		return s.End < o.End
	}
	return s.Index < o.Index
}

// ToSRT converts the subtitle to an SRT block.
func (s Subtitle) ToSRT(eol string) string {
	if eol == "" {
		eol = "\n"
	}
	content := MakeLegalContent(s.Content)
	if eol != "\n" {
		content = strings.ReplaceAll(content, "\n", eol)
	}
	idx := s.Index
	return fmt.Sprintf("%d%s%s --> %s%s%s%s%s",
		idx, eol,
		TimedeltaToSRTTimestamp(s.Start), TimedeltaToSRTTimestamp(s.End), eol,
		content, eol, eol)
}

// MakeLegalContent removes blank lines and leading/trailing newlines.
func MakeLegalContent(content string) string {
	if content != "" && content[0] != '\n' && !strings.Contains(content, "\n\n") {
		return content
	}
	legal := multiWSRegex.ReplaceAllString(strings.Trim(content, "\n"), "\n")
	log.Printf("Legalised content %q to %q", content, legal)
	return legal
}

// TimedeltaToSRTTimestamp converts a duration to "HH:MM:SS,mmm".
func TimedeltaToSRTTimestamp(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMS := d.Milliseconds()
	ms := totalMS % 1000
	secs := (totalMS / 1000) % 60
	mins := (totalMS / (1000 * 60)) % 60
	hrs := totalMS / (1000 * 60 * 60)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hrs, mins, secs, ms)
}

type skipCondition struct {
	msg string
	fn  func(Subtitle) bool
}

var subtitleSkipConditions = []skipCondition{
	{"No content", func(s Subtitle) bool { return strings.TrimSpace(s.Content) == "" }},
	{"Start time < 0 seconds", func(s Subtitle) bool { return s.Start < 0 }},
	{"Subtitle start time >= end time", func(s Subtitle) bool { return s.Start >= s.End }},
}

func shouldSkip(s Subtitle) string {
	for _, c := range subtitleSkipConditions {
		if c.fn(s) {
			return c.msg
		}
	}
	return ""
}

// SortAndReindex sorts subtitles by start time, drops useless ones when
// skip is true, and rewrites indexes starting at startIndex.
func SortAndReindex(subs []Subtitle, startIndex int, skip bool) []Subtitle {
	cp := make([]Subtitle, len(subs))
	copy(cp, subs)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Less(cp[j]) })
	out := make([]Subtitle, 0, len(cp))
	next := startIndex
	for _, s := range cp {
		if skip {
			if reason := shouldSkip(s); reason != "" {
				log.Printf("Skipped subtitle at index %d: %s", s.Index, reason)
				continue
			}
		}
		s.Index = next
		next++
		out = append(out, s)
	}
	return out
}

// Compose converts subtitles to a single SRT string, reindexing by default.
func Compose(subs []Subtitle, reindex bool, startIndex int, eol string) string {
	if reindex {
		subs = SortAndReindex(subs, startIndex, true)
	}
	var b strings.Builder
	for _, s := range subs {
		b.WriteString(s.ToSRT(eol))
	}
	return b.String()
}
