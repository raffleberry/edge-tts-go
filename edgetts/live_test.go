package edgetts

import (
	"context"
	"testing"
	"time"
)

// Live-service tests. Skipped with `go test -short`.

func TestListVoicesLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	voices, err := ListVoices(ctx, "")
	if err != nil {
		t.Fatalf("ListVoices: %v", err)
	}
	if len(voices) == 0 {
		t.Fatal("no voices returned")
	}
	found := false
	for _, v := range voices {
		if v.ShortName == "en-US-EmmaMultilingualNeural" {
			found = true
		}
		if v.VoiceTag.ContentCategories == nil || v.VoiceTag.VoicePersonalities == nil {
			t.Fatalf("VoiceTag not normalized: %+v", v)
		}
	}
	if !found {
		t.Fatal("default voice missing from list")
	}
}

func TestCommunicateLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	comm, err := NewCommunicate("Hello world!", "en-US-EmmaMultilingualNeural",
		"+0%", "+0%", "+0Hz", SentenceBoundary, "", 10, 60)
	if err != nil {
		t.Fatal(err)
	}
	var audioBytes int
	var boundaries int
	err = comm.Stream(ctx, func(chunk TTSChunk) error {
		switch chunk.Type {
		case ChunkAudio:
			audioBytes += len(chunk.Data)
		case ChunkWordBoundary, ChunkSentenceBoundary:
			boundaries++
		default:
			t.Fatalf("unknown chunk type %q", chunk.Type)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if audioBytes == 0 {
		t.Fatal("no audio received")
	}
	if boundaries == 0 {
		t.Fatal("no boundary events received")
	}
}

func TestVoicesManagerLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, err := NewVoicesManager(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found, err := m.Find(map[string]string{"Language": "en", "Gender": "Female"})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("no matching voices")
	}
}
