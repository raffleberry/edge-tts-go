package edgetts

import (
	"strings"
	"testing"
	"time"
)

func TestRemoveIncompatibleCharacters(t *testing.T) {
	in := "a\x00b\x07c\x0bd\x0ce\x0ef\x1fg"
	want := "a b c d e f g"
	if got := RemoveIncompatibleCharacters(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// Tab, LF, CR must be preserved.
	in = "a\tb\nc\rd"
	if got := RemoveIncompatibleCharacters(in); got != in {
		t.Fatalf("tab/LF/CR altered: %q", got)
	}
}

func TestSplitTextByByteLengthBasic(t *testing.T) {
	chunks, err := SplitTextByByteLength("hello world foo", 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if len(c) > 5 {
			t.Fatalf("chunk exceeds limit: %q", c)
		}
	}
	joined := strings.Join(byteSlicesToStrings(chunks), " ")
	if joined != "hello world foo" {
		t.Fatalf("round trip failed: %q", joined)
	}
}

func TestSplitTextByByteLengthNewlinePriority(t *testing.T) {
	text := "line1\nline2 line3"
	chunks, err := SplitTextByByteLength(text, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	if string(chunks[0]) != "line1" {
		t.Fatalf("expected first chunk 'line1', got %q", chunks[0])
	}
}

func TestSplitTextByByteLengthUTF8Safety(t *testing.T) {
	// "é" is 2 bytes in UTF-8.
	text := strings.Repeat("é", 10)
	chunks, err := SplitTextByByteLength(text, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if len(c) > 5 {
			t.Fatalf("chunk exceeds limit: %q (%d bytes)", c, len(c))
		}
		if !validUTF8(c) {
			t.Fatalf("chunk is not valid UTF-8: %q", c)
		}
	}
	var total strings.Builder
	for _, c := range chunks {
		total.Write(c)
	}
	if total.String() != text {
		t.Fatalf("round trip failed")
	}
}

func TestSplitTextByByteLengthXMLEntity(t *testing.T) {
	text := "this &amp; that and more text here"
	chunks, err := SplitTextByByteLength(text, 12)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		s := string(c)
		// No chunk may end with a bare '&' starting an unterminated entity,
		// and '&amp;' must never be split.
		if strings.HasSuffix(s, "&") || strings.HasSuffix(s, "&a") ||
			strings.HasSuffix(s, "&am") || strings.HasSuffix(s, "&amp") {
			t.Fatalf("chunk splits entity: %q", s)
		}
	}
}

func TestSplitTextByByteLengthErrors(t *testing.T) {
	if _, err := SplitTextByByteLength("hi", 0); err == nil {
		t.Fatal("expected error for byte_length 0")
	}
	if _, err := SplitTextByByteLength("hi", -1); err == nil {
		t.Fatal("expected error for negative byte_length")
	}
}

func TestMkSSML(t *testing.T) {
	tc, err := NewTTSConfig("en-US-EmmaMultilingualNeural", "+0%", "+0%", "+0Hz", SentenceBoundary)
	if err != nil {
		t.Fatal(err)
	}
	ssml := MkSSML(tc, "Hello")
	if !strings.Contains(ssml, "<speak") || !strings.Contains(ssml, "Hello") ||
		!strings.Contains(ssml, "EmmaMultilingualNeural") {
		t.Fatalf("bad SSML: %q", ssml)
	}
	if !strings.Contains(ssml, "pitch='+0Hz'") || !strings.Contains(ssml, "rate='+0%'") {
		t.Fatalf("prosody missing: %q", ssml)
	}
}

func TestSsmlHeadersPlusData(t *testing.T) {
	got := SsmlHeadersPlusData("reqid", "sometime", "<speak/>")
	if !strings.Contains(got, "X-RequestId:reqid\r\n") ||
		!strings.Contains(got, "Path:ssml\r\n\r\n<speak/>") {
		t.Fatalf("bad headers+data: %q", got)
	}
}

func TestDateToString(t *testing.T) {
	s := DateToString()
	if !strings.Contains(s, "GMT+0000 (Coordinated Universal Time)") {
		t.Fatalf("bad date string: %q", s)
	}
	if _, err := time.Parse("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)", s); err != nil {
		t.Fatalf("date string does not parse: %v", err)
	}
}

func TestGetHeadersAndData(t *testing.T) {
	raw := []byte("Path:audio\r\nContent-Type:audio/mpeg\r\n\r\nPAYLOAD")
	sep := strings.Index(string(raw), "\r\n\r\n")
	h, data, err := GetHeadersAndData(raw, sep)
	if err != nil {
		t.Fatal(err)
	}
	if string(h["Path"]) != "audio" {
		t.Fatalf("Path = %q", h["Path"])
	}
	if string(data) != "\r\nPAYLOAD" {
		t.Fatalf("data = %q", data)
	}
}

func TestConnectID(t *testing.T) {
	id := ConnectID()
	if len(id) != 32 {
		t.Fatalf("expected 32 chars, got %q", id)
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("non-hex char %q in %q", c, id)
		}
	}
	if id == ConnectID() {
		t.Fatal("expected unique IDs")
	}
}

func TestEscapeUnescape(t *testing.T) {
	if got := EscapeText("a&b<c>d\"e'f"); got != "a&amp;b&lt;c&gt;d\"e'f" {
		t.Fatalf("escape = %q", got)
	}
	if got := UnescapeText("a&amp;b&lt;c&gt;d&quot;e&apos;f"); got != "a&b<c>d\"e'f" {
		t.Fatalf("unescape = %q", got)
	}
}

func byteSlicesToStrings(b [][]byte) []string {
	out := make([]string, len(b))
	for i, v := range b {
		out[i] = string(v)
	}
	return out
}

func validUTF8(b []byte) bool {
	for len(b) > 0 {
		_, size := decodeRune(b)
		if size == 0 {
			return false
		}
		b = b[size:]
	}
	return true
}

func decodeRune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0, 0
	}
	c := b[0]
	if c < 0x80 {
		return rune(c), 1
	}
	var need int
	switch {
	case c&0xE0 == 0xC0:
		need = 2
	case c&0xF0 == 0xE0:
		need = 3
	case c&0xF8 == 0xF0:
		need = 4
	default:
		return 0, 0
	}
	if len(b) < need {
		return 0, 0
	}
	for _, cont := range b[1:need] {
		if cont&0xC0 != 0x80 {
			return 0, 0
		}
	}
	return 1, need
}
