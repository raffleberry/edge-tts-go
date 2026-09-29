package edgetts

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	winEpoch = 11644473600
	sToNS    = 1e9
)

var (
	skewMu           sync.RWMutex
	clockSkewSeconds float64
)

// AdjClockSkewSeconds adjusts the clock skew in case the system clock is off.
func AdjClockSkewSeconds(skewSeconds float64) {
	skewMu.Lock()
	defer skewMu.Unlock()
	clockSkewSeconds += skewSeconds
}

// SetClockSkewSeconds sets the absolute clock skew (mainly for tests).
func SetClockSkewSeconds(v float64) {
	skewMu.Lock()
	defer skewMu.Unlock()
	clockSkewSeconds = v
}

// GetClockSkewSeconds returns the current clock skew.
func GetClockSkewSeconds() float64 {
	skewMu.RLock()
	defer skewMu.RUnlock()
	return clockSkewSeconds
}

// GetUnixTimestamp returns the current Unix timestamp with skew correction.
func GetUnixTimestamp() float64 {
	return float64(time.Now().UTC().UnixNano())/1e9 + GetClockSkewSeconds()
}

// ParseRFC2616Date parses an RFC 2616 date string into a Unix timestamp.
// Returns (timestamp, true) on success.
func ParseRFC2616Date(date string) (float64, bool) {
	for _, layout := range []string{
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"Monday, 02-Jan-06 15:04:05 MST",
	} {
		if t, err := time.Parse(layout, date); err == nil {
			return float64(t.UTC().UnixNano()) / 1e9, true
		}
	}
	return 0, false
}

// AdjustSkewFromDateHeader adjusts the clock skew from an HTTP Date header value.
func AdjustSkewFromDateHeader(serverDate string) error {
	parsed, ok := ParseRFC2616Date(serverDate)
	if !ok {
		return &SkewAdjustmentError{Msg: fmt.Sprintf("failed to parse server date: %s", serverDate)}
	}
	AdjClockSkewSeconds(parsed - GetUnixTimestamp())
	return nil
}

// Handle403DateHeader handles a 403 response by adjusting skew from its Date header.
// dateHeader may be empty when the server did not supply one.
func Handle403DateHeader(dateHeader string) error {
	if strings.TrimSpace(dateHeader) == "" {
		return &SkewAdjustmentError{Msg: "no server date in headers"}
	}
	return AdjustSkewFromDateHeader(dateHeader)
}

// GenerateSecMSGEC generates the Sec-MS-GEC token value.
func GenerateSecMSGEC() string {
	ticks := GetUnixTimestamp()
	ticks += winEpoch
	ticks -= ticksMod(ticks, 300)
	ticks *= sToNS / 100
	strToHash := fmt.Sprintf("%.0f%s", ticks, TrustedClientToken)
	sum := sha256.Sum256([]byte(strToHash))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func ticksMod(ticks, m float64) float64 {
	// integer-style modulo for positive floats
	n := int64(ticks / m)
	return ticks - float64(n)*m
}

// GenerateMUID generates a random MUID (32 uppercase hex chars).
func GenerateMUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp-based value; should never happen.
		return strings.ToUpper(fmt.Sprintf("%016X%016X", time.Now().UnixNano(), time.Now().UnixNano()))
	}
	return strings.ToUpper(hex.EncodeToString(b[:]))
}

// HeadersWithMUID returns a copy of headers with the MUID cookie added.
func HeadersWithMUID(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		out[k] = v
	}
	out["Cookie"] = "muid=" + GenerateMUID() + ";"
	return out
}
