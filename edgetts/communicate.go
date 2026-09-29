package edgetts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// Communicate talks to the Edge TTS service.
type Communicate struct {
	Config         TTSConfig
	Texts          [][]byte
	Proxy          string
	ConnectTimeout time.Duration
	ReceiveTimeout time.Duration

	offsetCompensation   int64
	chunkAudioBytes      int64
	cumulativeAudioBytes int64
	streamWasCalled      bool
}

// NewCommunicate validates its arguments, splits text, and returns a Communicate.
// Timeouts are seconds; pass 0 to use defaults (10s connect, 60s receive).
func NewCommunicate(text, voice, rate, volume, pitch string, boundary Boundary, proxy string, connectTimeoutSecs, receiveTimeoutSecs int) (*Communicate, error) {
	if rate == "" {
		rate = "+0%"
	}
	if volume == "" {
		volume = "+0%"
	}
	if pitch == "" {
		pitch = "+0Hz"
	}
	if voice == "" {
		voice = DefaultVoice
	}
	if boundary == "" {
		boundary = SentenceBoundary
	}
	tc, err := NewTTSConfig(voice, rate, volume, pitch, boundary)
	if err != nil {
		return nil, err
	}
	texts, err := SplitTextByByteLength(EscapeText(RemoveIncompatibleCharacters(text)), 4096)
	if err != nil {
		return nil, err
	}
	ct := time.Duration(connectTimeoutSecs) * time.Second
	if connectTimeoutSecs == 0 {
		ct = 10 * time.Second
	}
	rt := time.Duration(receiveTimeoutSecs) * time.Second
	if receiveTimeoutSecs == 0 {
		rt = 60 * time.Second
	}
	return &Communicate{
		Config:         tc,
		Texts:          texts,
		Proxy:          proxy,
		ConnectTimeout: ct,
		ReceiveTimeout: rt,
	}, nil
}

// Stream consumes the service and calls yield for every chunk, mirroring
// Python's `async for chunk in communicate.stream()`.
func (c *Communicate) Stream(ctx context.Context, yield func(TTSChunk) error) error {
	if c.streamWasCalled {
		return fmt.Errorf("stream can only be called once")
	}
	c.streamWasCalled = true
	for _, partial := range c.Texts {
		c.chunkAudioBytes = 0
		err := c.streamOne(ctx, partial, yield)
		if err == nil {
			continue
		}
		if isForbiddenError(err) {
			c.chunkAudioBytes = 0
			if err2 := c.streamOne(ctx, partial, yield); err2 != nil {
				return err2
			}
			continue
		}
		return err
	}
	return nil
}

// StreamChan is a channel-based variant of Stream.
func (c *Communicate) StreamChan(ctx context.Context) (<-chan TTSChunk, <-chan error) {
	ch := make(chan TTSChunk)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errCh)
		errCh <- c.Stream(ctx, func(chunk TTSChunk) error {
			select {
			case ch <- chunk:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	return ch, errCh
}

// Save writes audio to audioFname and, when metadataFname != "", JSON-lines
// boundary metadata to metadataFname.
func (c *Communicate) Save(ctx context.Context, audioFname, metadataFname string) error {
	audio, err := os.Create(audioFname)
	if err != nil {
		return err
	}
	defer audio.Close()
	var meta *os.File
	if metadataFname != "" {
		meta, err = os.Create(metadataFname)
		if err != nil {
			return err
		}
		defer meta.Close()
	}
	enc := json.NewEncoder(nil)
	_ = enc
	return c.Stream(ctx, func(chunk TTSChunk) error {
		switch chunk.Type {
		case ChunkAudio:
			if _, err := audio.Write(chunk.Data); err != nil {
				return err
			}
		case ChunkWordBoundary, ChunkSentenceBoundary:
			if meta != nil {
				m := map[string]any{
					"type":     string(chunk.Type),
					"offset":   chunk.Offset,
					"duration": chunk.Duration,
					"text":     chunk.Text,
				}
				b, _ := json.Marshal(m)
				b = append(b, '\n')
				if _, err := meta.Write(b); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func isForbiddenError(err error) bool {
	var e *handshakeError
	if asErr(err, &e) {
		return e.statusCode == http.StatusForbidden
	}
	return false
}

func asErr(err error, target **handshakeError) bool {
	if he, ok := err.(*handshakeError); ok {
		*target = he
		return true
	}
	return false
}

type handshakeError struct {
	statusCode int
	dateHeader string
	msg        string
}

func (e *handshakeError) Error() string { return e.msg }

type wsMetadata struct {
	Metadata []struct {
		Type string `json:"Type"`
		Data struct {
			Offset   int64 `json:"Offset"`
			Duration int64 `json:"Duration"`
			Text     struct {
				Text string `json:"Text"`
			} `json:"text"`
		} `json:"Data"`
	} `json:"Metadata"`
}

func (c *Communicate) compensateOffset() {
	c.cumulativeAudioBytes += c.chunkAudioBytes
	c.offsetCompensation = c.cumulativeAudioBytes * 8 * TicksPerSecond / MP3BitrateBPS
	c.chunkAudioBytes = 0
}

func (c *Communicate) parseMetadata(data []byte) (TTSChunk, error) {
	var m wsMetadata
	if err := json.Unmarshal(data, &m); err != nil {
		return TTSChunk{}, &UnexpectedResponseError{Msg: "invalid metadata JSON"}
	}
	for _, obj := range m.Metadata {
		switch obj.Type {
		case "WordBoundary", "SentenceBoundary":
			return TTSChunk{
				Type:     ChunkType(obj.Type),
				Offset:   obj.Data.Offset + c.offsetCompensation,
				Duration: obj.Data.Duration,
				Text:     UnescapeText(obj.Data.Text.Text),
			}, nil
		case "SessionEnd":
			continue
		default:
			return TTSChunk{}, &UnknownResponseError{Msg: "unknown metadata type: " + obj.Type}
		}
	}
	return TTSChunk{}, &UnexpectedResponseError{Msg: "no WordBoundary metadata found"}
}

func (c *Communicate) dialer() (*websocket.Dialer, error) {
	d := &websocket.Dialer{
		HandshakeTimeout: c.ConnectTimeout,
	}
	if c.Proxy != "" {
		pu, err := url.Parse(c.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy %q: %w", c.Proxy, err)
		}
		d.Proxy = http.ProxyURL(pu)
	} else {
		d.Proxy = http.ProxyFromEnvironment
	}
	return d, nil
}

func (c *Communicate) streamOne(ctx context.Context, partialText []byte, yield func(TTSChunk) error) error {
	dialer, err := c.dialer()
	if err != nil {
		return err
	}
	endpoint := WSSURL + "&ConnectionId=" + ConnectID() +
		"&Sec-MS-GEC=" + GenerateSecMSGEC() +
		"&Sec-MS-GEC-Version=" + SecMSGECVersion
	headers := http.Header{}
	for k, v := range HeadersWithMUID(WSSHeaders()) {
		headers.Set(k, v)
	}
	conn, resp, err := dialer.DialContext(ctx, endpoint, headers)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			date := ""
			if resp.Header != nil {
				date = resp.Header.Get("Date")
			}
			derr := Handle403DateHeader(date)
			if derr != nil {
				return derr
			}
			return &handshakeError{statusCode: 403, dateHeader: date, msg: "websocket handshake forbidden (403)"}
		}
		return &WebSocketError{Msg: err.Error()}
	}
	defer conn.Close()

	if c.ReceiveTimeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(c.ReceiveTimeout))
	}

	wordBoundary := c.Config.Boundary == WordBoundary
	wd, sq := "false", "true"
	if wordBoundary {
		wd, sq = "true", "false"
	}
	configMsg := "X-Timestamp:" + DateToString() + "\r\n" +
		"Content-Type:application/json; charset=utf-8\r\n" +
		"Path:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{` +
		`"sentenceBoundaryEnabled":"` + sq + `","wordBoundaryEnabled":"` + wd + `"},` +
		`"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}` + "\r\n"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
		return &WebSocketError{Msg: err.Error()}
	}
	ssml := SsmlHeadersPlusData(ConnectID(), DateToString(), MkSSML(c.Config, string(partialText)))
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ssml)); err != nil {
		return &WebSocketError{Msg: err.Error()}
	}

	audioReceived := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if c.ReceiveTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(c.ReceiveTimeout))
		}
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err) {
				break
			}
			// Normal closure after turn.end break is handled below; EOF here
			// means the server closed the stream.
			break
		}
		switch msgType {
		case websocket.TextMessage:
			sep := bytes.Index(data, []byte("\r\n\r\n"))
			if sep < 0 {
				return &UnexpectedResponseError{Msg: "text message missing header separator"}
			}
			params, payload, err := GetHeadersAndData(data, sep)
			if err != nil {
				return err
			}
			path := string(params["Path"])
			switch path {
			case "audio.metadata":
				parsed, err := c.parseMetadata(payload)
				if err != nil {
					if _, ok := err.(*UnknownResponseError); ok {
						return err
					}
					// SessionEnd-only metadata raises UnexpectedResponse in Python;
					// treat as skippable.
					continue
				}
				if err := yield(parsed); err != nil {
					return err
				}
			case "turn.end":
				c.compensateOffset()
				goto done
			case "response", "turn.start":
				// ignore
			default:
				return &UnknownResponseError{Msg: "unknown path received: " + path}
			}
		case websocket.BinaryMessage:
			if len(data) < 2 {
				return &UnexpectedResponseError{Msg: "binary message missing header length"}
			}
			headerLen := int(data[0])<<8 | int(data[1])
			if headerLen > len(data) {
				return &UnexpectedResponseError{Msg: "header length greater than data length"}
			}
			params, payload, err := GetHeadersAndData(data[2:], headerLen-2)
			if err != nil {
				return err
			}
			if string(params["Path"]) != "audio" {
				return &UnexpectedResponseError{Msg: "binary message path is not audio"}
			}
			ct, hasCT := params["Content-Type"]
			if !hasCT {
				if len(payload) == 0 {
					continue
				}
				return &UnexpectedResponseError{Msg: "binary message with no Content-Type but with data"}
			}
			if string(ct) != "audio/mpeg" {
				return &UnexpectedResponseError{Msg: "binary message with unexpected Content-Type"}
			}
			if len(payload) == 0 {
				return &UnexpectedResponseError{Msg: "binary message missing audio data"}
			}
			audioReceived = true
			c.chunkAudioBytes += int64(len(payload))
			cp := make([]byte, len(payload))
			copy(cp, payload)
			if err := yield(TTSChunk{Type: ChunkAudio, Data: cp}); err != nil {
				return err
			}
		default:
			return &WebSocketError{Msg: "unexpected websocket message type"}
		}
	}
done:
	if !audioReceived {
		return &NoAudioReceivedError{Msg: "No audio was received. Please verify that your parameters are correct."}
	}
	return nil
}
