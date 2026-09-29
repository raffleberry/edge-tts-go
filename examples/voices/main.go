// Command voices demonstrates dynamic voice selection with VoicesManager.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/raffleberry/edge-tts-go/edgetts"
)

const (
	text       = "Hoy es un buen día."
	outputFile = "spanish.mp3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	manager, err := edgetts.NewVoicesManager(ctx, nil)
	if err != nil {
		return err
	}
	// Also supports Locale, ShortName, Name, Status.
	voices, err := manager.Find(map[string]string{"Gender": "Male", "Language": "es"})
	if err != nil {
		return err
	}
	if len(voices) == 0 {
		return fmt.Errorf("no matching voices")
	}
	voice := voices[rand.Intn(len(voices))].ShortName
	comm, err := edgetts.NewCommunicate(text, voice, "+0%", "+0%", "+0Hz", edgetts.SentenceBoundary, "", 10, 60)
	if err != nil {
		return err
	}
	return comm.Save(ctx, outputFile, "")
}
