package edgetts

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// GetHeadersAndData splits a raw protocol message into headers and payload.
// headerLength is the byte offset of the "\r\n\r\n" separator.
func GetHeadersAndData(data []byte, headerLength int) (map[string][]byte, []byte, error) {
	if headerLength < 0 || headerLength > len(data) {
		return nil, nil, &UnexpectedResponseError{Msg: "header length out of range"}
	}
	headers := make(map[string][]byte)
	for _, line := range bytes.Split(data[:headerLength], []byte("\r\n")) {
		if len(line) == 0 {
			continue
		}
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			return nil, nil, &UnexpectedResponseError{Msg: fmt.Sprintf("malformed header line: %q", line)}
		}
		headers[string(line[:idx])] = line[idx+1:]
	}
	rest := []byte{}
	if headerLength+2 <= len(data) {
		rest = data[headerLength+2:]
	}
	return headers, rest, nil
}

// RemoveIncompatibleCharacters replaces control characters unsupported by the
// service (0-8, 11-12, 14-31) with spaces.
func RemoveIncompatibleCharacters(s string) string {
	return strings.Map(func(r rune) rune {
		if (0 <= r && r <= 8) || (11 <= r && r <= 12) || (14 <= r && r <= 31) {
			return ' '
		}
		return r
	}, s)
}

// ConnectID returns a random 32-char lowercase hex ID (UUID without dashes).
func ConnectID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), time.Now().UnixNano())
	}
	// Set UUIDv4 version/variant bits to match Python's uuid4().hex.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:])
}

func findLastNewlineOrSpaceWithinLimit(text []byte, limit int) int {
	if limit > len(text) {
		limit = len(text)
	}
	if i := bytes.LastIndexByte(text[:limit], '\n'); i >= 0 {
		return i
	}
	if i := bytes.LastIndexByte(text[:limit], ' '); i >= 0 {
		return i
	}
	return -1
}

func findSafeUTF8SplitPoint(segment []byte) int {
	splitAt := len(segment)
	for splitAt > 0 {
		if utf8.Valid(segment[:splitAt]) {
			return splitAt
		}
		splitAt--
	}
	return 0
}

func adjustSplitPointForXMLEntity(text []byte, splitAt int) int {
	for splitAt > 0 {
		amp := bytes.LastIndexByte(text[:splitAt], '&')
		if amp < 0 {
			break
		}
		if bytes.IndexByte(text[amp:splitAt], ';') != -1 {
			break
		}
		splitAt = amp
	}
	return splitAt
}

// SplitTextByByteLength splits text into chunks of at most byteLength bytes,
// preferring newline/space boundaries while never splitting a UTF-8 rune or
// an XML entity such as &amp;. Chunks are trimmed of surrounding whitespace.
func SplitTextByByteLength(text string, byteLength int) ([][]byte, error) {
	if byteLength <= 0 {
		return nil, fmt.Errorf("byte_length must be greater than 0")
	}
	raw := []byte(text)
	var out [][]byte
	for len(raw) > byteLength {
		splitAt := findLastNewlineOrSpaceWithinLimit(raw, byteLength)
		if splitAt < 0 {
			limit := byteLength
			if limit > len(raw) {
				limit = len(raw)
			}
			splitAt = findSafeUTF8SplitPoint(raw[:limit])
		}
		splitAt = adjustSplitPointForXMLEntity(raw, splitAt)
		if splitAt < 0 {
			return nil, fmt.Errorf("maximum byte length is too small or invalid text structure near '&' or invalid UTF-8")
		}
		if chunk := bytes.TrimSpace(raw[:splitAt]); len(chunk) > 0 {
			c := make([]byte, len(chunk))
			copy(c, chunk)
			out = append(out, c)
		}
		// If splitAt is 0 (e.g. the chunk starts with a space or an XML
		// entity boundary), advance by one byte to avoid an infinite loop,
		// mirroring the Python implementation.
		if splitAt == 0 {
			raw = raw[1:]
		} else {
			raw = raw[splitAt:]
		}
	}
	if chunk := bytes.TrimSpace(raw); len(chunk) > 0 {
		c := make([]byte, len(chunk))
		copy(c, chunk)
		out = append(out, c)
	}
	return out, nil
}

// MkSSML builds the SSML document for the given config and escaped text.
func MkSSML(tc TTSConfig, escapedText string) string {
	return "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'>" +
		"<voice name='" + tc.Voice + "'>" +
		"<prosody pitch='" + tc.Pitch + "' rate='" + tc.Rate + "' volume='" + tc.Volume + "'>" +
		escapedText +
		"</prosody>" +
		"</voice>" +
		"</speak>"
}

// DateToString returns a JavaScript-style date string in UTC.
func DateToString() string {
	return time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
}

// SsmlHeadersPlusData builds the SSML request payload.
func SsmlHeadersPlusData(requestID, timestamp, ssml string) string {
	return "X-RequestId:" + requestID + "\r\n" +
		"Content-Type:application/ssml+xml\r\n" +
		"X-Timestamp:" + timestamp + "Z\r\n" +
		"Path:ssml\r\n\r\n" +
		ssml
}

// EscapeText XML-escapes text like Python's xml.sax.saxutils.escape
// (only &, <, > — quotes are left alone to match SSML expectations).
func EscapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// UnescapeText reverses EscapeText plus &quot; and &apos;.
func UnescapeText(s string) string {
	r := strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&apos;", "'",
		"&amp;", "&",
	)
	return r.Replace(s)
}
