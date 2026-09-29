package edgetts

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewTTSConfigShortVoiceExpansion(t *testing.T) {
	tc, err := NewTTSConfig("en-US-EmmaMultilingualNeural", "+0%", "+0%", "+0Hz", SentenceBoundary)
	if err != nil {
		t.Fatal(err)
	}
	want := "Microsoft Server Speech Text to Speech Voice (en-US, EmmaMultilingualNeural)"
	if tc.Voice != want {
		t.Fatalf("got %q want %q", tc.Voice, want)
	}
}

func TestNewTTSConfigCompoundRegion(t *testing.T) {
	tc, err := NewTTSConfig("cy-GB-NiaNeural", "+0%", "+0%", "+0Hz", SentenceBoundary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tc.Voice, "cy-GB") {
		t.Fatalf("unexpected voice expansion: %q", tc.Voice)
	}
}

func TestNewTTSConfigInvalid(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		voice, rate, vol, pit string
	}{
		{"bad voice", "not-a-voice", "+0%", "+0%", "+0Hz"},
		{"bad rate", "en-US-EmmaMultilingualNeural", "fast", "+0%", "+0Hz"},
		{"bad volume", "en-US-EmmaMultilingualNeural", "+0%", "loud", "+0Hz"},
		{"bad pitch", "en-US-EmmaMultilingualNeural", "+0%", "+0%", "high"},
	} {
		if _, err := NewTTSConfig(tt.voice, tt.rate, tt.vol, tt.pit, SentenceBoundary); err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
	}
	if _, err := NewTTSConfig("en-US-EmmaMultilingualNeural", "+0%", "+0%", "+0Hz", "ParagraphBoundary"); err == nil {
		t.Fatal("expected error for bad boundary")
	}
}

func TestGenerateSecMSGEC(t *testing.T) {
	tok := GenerateSecMSGEC()
	if matched, _ := regexp.MatchString(`^[0-9A-F]{64}$`, tok); !matched {
		t.Fatalf("bad token: %q", tok)
	}
	// Deterministic within a 5-minute window.
	if tok2 := GenerateSecMSGEC(); tok != tok2 {
		t.Fatal("token changed within same window")
	}
}

func TestGenerateMUID(t *testing.T) {
	m := GenerateMUID()
	if matched, _ := regexp.MatchString(`^[0-9A-F]{32}$`, m); !matched {
		t.Fatalf("bad muid: %q", m)
	}
}

func TestHeadersWithMUID(t *testing.T) {
	base := map[string]string{"Origin": "x"}
	h := HeadersWithMUID(base)
	if !strings.HasPrefix(h["Cookie"], "muid=") {
		t.Fatalf("missing muid cookie: %v", h)
	}
	if _, ok := base["Cookie"]; ok {
		t.Fatal("input map mutated")
	}
}

func TestParseRFC2616Date(t *testing.T) {
	ts, ok := ParseRFC2616Date("Wed, 21 Oct 2015 07:28:00 GMT")
	if !ok {
		t.Fatal("parse failed")
	}
	want := time.Date(2015, 10, 21, 7, 28, 0, 0, time.UTC).Unix()
	if int64(ts) != want {
		t.Fatalf("got %v want %v", ts, want)
	}
	if _, ok := ParseRFC2616Date("not a date"); ok {
		t.Fatal("expected parse failure")
	}
}

func TestClockSkewAdjust(t *testing.T) {
	old := GetClockSkewSeconds()
	defer SetClockSkewSeconds(old)
	SetClockSkewSeconds(0)
	AdjClockSkewSeconds(5)
	if GetClockSkewSeconds() != 5 {
		t.Fatalf("skew = %v", GetClockSkewSeconds())
	}
	if err := AdjustSkewFromDateHeader(""); err == nil {
		t.Fatal("expected error for empty date")
	}
	if err := Handle403DateHeader(""); err == nil {
		t.Fatal("expected error for missing date header")
	}
}
