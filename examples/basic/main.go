// Command basic demonstrates generating an audio file with a predefined voice.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/raffleberry/edge-tts-go/edgetts"
)

const (
	text       = "Hello World!"
	voice      = "en-GB-SoniaNeural"
	outputFile = "test.mp3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	comm, err := edgetts.NewCommunicate(text, voice, "+0%", "+0%", "+0Hz", edgetts.SentenceBoundary, "", 10, 60)
	if err != nil {
		return err
	}
	return comm.Save(context.Background(), outputFile, "")
}
