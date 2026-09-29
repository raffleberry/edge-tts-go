// Command streaming demonstrates streaming audio chunks while feeding
// boundary events to a SubMaker to generate subtitles.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/raffleberry/edge-tts-go/edgetts"
)

const (
	text      = "Hello World!"
	voice     = "en-GB-SoniaNeural"
	output    = "test.mp3"
	subtitles = "test.srt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	comm, err := edgetts.NewCommunicate(text, voice, "+0%", "+0%", "+0Hz", edgetts.SentenceBoundary, "", 10, 60)
	if err != nil {
		return err
	}
	submaker := &edgetts.SubMaker{}
	audio, err := os.Create(output)
	if err != nil {
		return err
	}
	defer audio.Close()
	if err := comm.Stream(ctx, func(chunk edgetts.TTSChunk) error {
		switch chunk.Type {
		case edgetts.ChunkAudio:
			_, err := audio.Write(chunk.Data)
			return err
		case edgetts.ChunkWordBoundary, edgetts.ChunkSentenceBoundary:
			return submaker.Feed(chunk)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.WriteFile(subtitles, []byte(submaker.GetSRT()), 0o644)
}
