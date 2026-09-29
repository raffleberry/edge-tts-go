package edgetts

import (
	"strings"
	"testing"
	"time"
)

func TestTimedeltaToSRTTimestamp(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{time.Hour + 23*time.Minute + 4*time.Second, "01:23:04,000"},
		{time.Second + 500*time.Millisecond, "00:00:01,500"},
		{0, "00:00:00,000"},
		{25 * time.Hour, "25:00:00,000"},
	}
	for _, c := range cases {
		if got := TimedeltaToSRTTimestamp(c.d); got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}

func TestComposeAndSubMaker(t *testing.T) {
	sm := &SubMaker{}
	// 10_000_000 ticks = 1 second.
	if err := sm.Feed(TTSChunk{Type: ChunkWordBoundary, Offset: 10_000_000, Duration: 5_000_000, Text: "Hello"}); err != nil {
		t.Fatal(err)
	}
	if err := sm.Feed(TTSChunk{Type: ChunkWordBoundary, Offset: 20_000_000, Duration: 5_000_000, Text: "world"}); err != nil {
		t.Fatal(err)
	}
	srt := sm.GetSRT()
	if !strings.Contains(srt, "00:00:01,000 --> 00:00:01,500") {
		t.Fatalf("bad first cue:\n%s", srt)
	}
	if !strings.Contains(srt, "Hello") || !strings.Contains(srt, "world") {
		t.Fatalf("missing content:\n%s", srt)
	}
	if !strings.HasPrefix(srt, "1\n") || !strings.Contains(srt, "\n2\n") {
		t.Fatalf("bad indexing:\n%s", srt)
	}
}

func TestSubMakerMixedTypesRejected(t *testing.T) {
	sm := &SubMaker{}
	if err := sm.Feed(TTSChunk{Type: ChunkWordBoundary, Offset: 0, Duration: 1, Text: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := sm.Feed(TTSChunk{Type: ChunkSentenceBoundary, Offset: 0, Duration: 1, Text: "b"}); err == nil {
		t.Fatal("expected error mixing boundary types")
	}
	if err := sm.Feed(TTSChunk{Type: ChunkAudio}); err == nil {
		t.Fatal("expected error for audio chunk")
	}
}

func TestSortAndReindexSkipsUseless(t *testing.T) {
	subs := []Subtitle{
		{Index: 99, Start: 2 * time.Second, End: 3 * time.Second, Content: "b"},
		{Index: 98, Start: time.Second, End: 2 * time.Second, Content: "a"},
		{Index: 97, Start: 4 * time.Second, End: 3 * time.Second, Content: "bad"}, // start >= end
		{Index: 96, Start: 5 * time.Second, End: 6 * time.Second, Content: "   "}, // blank
	}
	out := SortAndReindex(subs, 1, true)
	if len(out) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(out))
	}
	if out[0].Content != "a" || out[0].Index != 1 || out[1].Content != "b" || out[1].Index != 2 {
		t.Fatalf("bad sort/reindex: %+v", out)
	}
}

func TestMakeLegalContent(t *testing.T) {
	if got := MakeLegalContent("\nfoo\n\nbar\n"); got != "foo\nbar" {
		t.Fatalf("got %q", got)
	}
	if got := MakeLegalContent("clean"); got != "clean" {
		t.Fatalf("clean content altered: %q", got)
	}
}
