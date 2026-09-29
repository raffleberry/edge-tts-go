# edge-tts-go

`edge-tts-go` is a Go port of the popular Python
[`edge-tts`](https://github.com/rany2/edge-tts) module. 

## Installation

Requires Go 1.24+.

```sh
go install github.com/raffleberry/edge-tts-go/cmd/edge-tts@latest
go install github.com/raffleberry/edge-tts-go/cmd/edge-playback@latest
```

Or clone and build:

```sh
git clone https://github.com/raffleberry/edge-tts-go
cd edge-tts-go
go build ./...
```

## CLI usage

```sh
# Basic synthesis
edge-tts --text "Hello, world!" --write-media hello.mp3 --write-subtitles hello.srt

# Play back immediately (requires mpv)
edge-playback --text "Hello, world!"

# List voices
edge-tts --list-voices

# Change voice
edge-tts --voice ar-EG-SalmaNeural --text "مرحبا كيف حالك؟" \
  --write-media hello_in_arabic.mp3 --write-subtitles hello_in_arabic.srt

# Change rate / volume / pitch
edge-tts --rate=-50% --text "Hello, world!" --write-media hello.mp3
edge-tts --volume=-50% --text "Hello, world!" --write-media hello.mp3
edge-tts --pitch=-50Hz --text "Hello, world!" --write-media hello.mp3

# Read text from a file (or stdin with -f -)
edge-tts -f book.txt --write-media book.mp3
cat book.txt | edge-tts -f - --write-media book.mp3

# Use a proxy
edge-tts --proxy http://localhost:8080 --text "hi" --write-media hi.mp3
```

All `edge-tts` flags work with `edge-playback`, except `--write-media`,
`--write-subtitles`, and `--list-voices` (managed internally via temp files).

## Library usage

```go
import (
    "context"

    "github.com/raffleberry/edge-tts-go/edgetts"
)

// One-shot save.
comm, err := edgetts.NewCommunicate(
    "Hello World!", "en-GB-SoniaNeural",
    "+0%", "+0%", "+0Hz",
    edgetts.SentenceBoundary, // or edgetts.WordBoundary for word subtitles
    "", 10, 60, // proxy, connect/read timeout seconds
)
if err != nil { /* ... */ }
if err := comm.Save(ctx, "test.mp3", ""); err != nil { /* ... */ }

// Streaming with subtitles.
submaker := &edgetts.SubMaker{}
err = comm.Stream(ctx, func(chunk edgetts.TTSChunk) error {
    switch chunk.Type {
    case edgetts.ChunkAudio:
        audioFile.Write(chunk.Data)
    case edgetts.ChunkWordBoundary, edgetts.ChunkSentenceBoundary:
        submaker.Feed(chunk)
    }
    return nil
})
srt := submaker.GetSRT()

// Channel-based streaming.
chunks, errs := comm.StreamChan(ctx)

// Voice discovery.
voices, err := edgetts.ListVoices(ctx, "")
manager, err := edgetts.NewVoicesManager(ctx, nil)
matches, err := manager.Find(map[string]string{"Gender": "Female", "Language": "en"})
```

See [`examples/`](examples/) for runnable programs.


## Testing

```sh
go test -short ./...   # unit tests (no network)
go test ./...          # includes live-service tests (ListVoices, synthesis)
go vet ./...
```

## Notes and differences from the Python package

- `Communicate.Stream` takes a callback (`func(TTSChunk) error`) instead of
  returning an async generator; `StreamChan` offers a channel-based variant.
  There is no separate "sync" API because Go has no async/colored functions.
- `SplitTextByByteLength` honors the documented maximum byte length in all
  cases. The Python implementation passes the *full* remaining text to its
  UTF-8 safe-split helper, so spaceless text (e.g. long CJK strings) yields
  chunks larger than 4096 bytes; the Go port correctly splits those.
- `edge-playback` always uses `mpv` (including on Windows); the Python
  version can use a Win32 API fallback when `--mpv` is not given.
- Clock-skew (HTTP 403 → `Date` header) retry logic is preserved for both
  the voice list and the WebSocket handshake.
- Inter-chunk subtitle offset compensation uses exact CBR arithmetic
  (48 kbps → ticks), as in the current Python release.
