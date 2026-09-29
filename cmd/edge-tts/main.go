package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/raffleberry/edge-tts-go/edgetts"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("edge-tts", flag.ContinueOnError)
	text := fs.String("text", "", "what TTS will say")
	file := fs.String("file", "", "same as --text but read from file")
	voice := fs.String("voice", edgetts.DefaultVoice, "voice for TTS")
	listVoices := fs.Bool("list-voices", false, "lists available voices and exits")
	rate := fs.String("rate", "+0%", "set TTS rate")
	volume := fs.String("volume", "+0%", "set TTS volume")
	pitch := fs.String("pitch", "+0Hz", "set TTS pitch")
	writeMedia := fs.String("write-media", "", "send media output to file instead of stdout")
	writeSubtitles := fs.String("write-subtitles", "", "send subtitle output to provided file instead of stderr")
	proxy := fs.String("proxy", "", "use a proxy for TTS and voice list")
	showVersion := fs.Bool("version", false, "show version and exit")
	// Short aliases.
	var t, f, v string
	var lv bool
	fs.StringVar(&t, "t", "", "what TTS will say")
	fs.StringVar(&f, "f", "", "same as --text but read from file")
	fs.StringVar(&v, "v", "", "voice for TTS")
	fs.BoolVar(&lv, "l", false, "lists available voices and exits")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	if t != "" {
		*text = t
	}
	if f != "" {
		*file = f
	}
	if v != "" {
		*voice = v
	}
	if lv {
		*listVoices = true
	}
	if *showVersion {
		fmt.Printf("edge-tts %s\n", edgetts.Version)
		return nil
	}

	ctx := context.Background()

	if *listVoices {
		return printVoices(ctx, *proxy)
	}

	input := *text
	if *file != "" {
		if *text != "" {
			return fmt.Errorf("--text and --file are mutually exclusive")
		}
		var data []byte
		var err error
		if *file == "-" || *file == "/dev/stdin" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(*file)
		}
		if err != nil {
			return err
		}
		input = string(data)
	}
	if input == "" {
		return fmt.Errorf("one of --text, --file or --list-voices is required")
	}

	return runTTS(ctx, input, *voice, *rate, *volume, *pitch, *proxy, *writeMedia, *writeSubtitles)
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func printVoices(ctx context.Context, proxy string) error {
	voices, err := edgetts.ListVoices(ctx, proxy)
	if err != nil {
		return err
	}
	sort.Slice(voices, func(i, j int) bool { return voices[i].ShortName < voices[j].ShortName })
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "Name\tGender\tContentCategories\tVoicePersonalities")
	for _, vc := range voices {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			vc.ShortName,
			vc.Gender,
			strings.Join(vc.VoiceTag.ContentCategories, ", "),
			strings.Join(vc.VoiceTag.VoicePersonalities, ", "),
		)
	}
	return w.Flush()
}

func runTTS(ctx context.Context, text, voice, rate, volume, pitch, proxy, writeMedia, writeSubtitles string) error {
	if writeMedia == "" || writeMedia == "-" {
		if isTerminal(os.Stdout) && isTerminal(os.Stdin) {
			fmt.Fprintln(os.Stderr, "Warning: TTS output will be written to the terminal. "+
				"Use --write-media to write to a file.")
			fmt.Fprintln(os.Stderr, "Press Ctrl+C to cancel the operation. Press Enter to continue.")
			var line [1]byte
			_, _ = os.Stdin.Read(line[:])
		}
	}
	comm, err := edgetts.NewCommunicate(text, voice, rate, volume, pitch, edgetts.SentenceBoundary, proxy, 10, 60)
	if err != nil {
		return err
	}

	var audioOut *os.File
	if writeMedia == "" || writeMedia == "-" {
		audioOut = os.Stdout
	} else {
		audioOut, err = os.Create(writeMedia)
		if err != nil {
			return err
		}
		defer audioOut.Close()
	}

	var subOut *os.File
	if writeSubtitles == "-" {
		subOut = os.Stderr
	} else if writeSubtitles != "" {
		subOut, err = os.Create(writeSubtitles)
		if err != nil {
			return err
		}
		defer subOut.Close()
	}

	submaker := &edgetts.SubMaker{}
	err = comm.Stream(ctx, func(chunk edgetts.TTSChunk) error {
		switch chunk.Type {
		case edgetts.ChunkAudio:
			if _, err := audioOut.Write(chunk.Data); err != nil {
				return err
			}
		case edgetts.ChunkWordBoundary, edgetts.ChunkSentenceBoundary:
			if subOut != nil {
				if err := submaker.Feed(chunk); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if subOut != nil {
		if _, err := io.WriteString(subOut, submaker.GetSRT()); err != nil {
			return err
		}
	}
	return nil
}
