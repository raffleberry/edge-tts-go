// Command edge-playback speaks text using Microsoft Edge's online
// text-to-speech service and plays it back with mpv.
//
// It forwards all TTS flags to the edge-tts binary, saves media and
// subtitles to temporary files, plays them with mpv, then cleans up.
//
// Environment overrides (mainly for debugging):
//
//	EDGE_PLAYBACK_DEBUG     print temp file names and keep output verbose
//	EDGE_PLAYBACK_KEEP_TEMP keep temp files instead of deleting them
//	EDGE_PLAYBACK_MP3_FILE  use this file for media instead of a temp file
//	EDGE_PLAYBACK_SRT_FILE  use this file for subtitles instead of a temp file
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func prErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func main() {
	if err := run(); err != nil {
		prErr("error: " + err.Error())
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("edge-playback", flag.ContinueOnError)
	useMPV := fs.Bool("mpv", true, "use mpv to play audio")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	ttsArgs := fs.Args()

	for _, dep := range []string{"edge-tts"} {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s is not installed (missing dependency); please install it", dep)
		}
	}
	if *useMPV {
		if _, err := exec.LookPath("mpv"); err != nil {
			return fmt.Errorf("mpv is not installed; please install it")
		}
	}

	debug := os.Getenv("EDGE_PLAYBACK_DEBUG") != ""
	keep := os.Getenv("EDGE_PLAYBACK_KEEP_TEMP") != ""
	mp3Fname := os.Getenv("EDGE_PLAYBACK_MP3_FILE")
	srtFname := os.Getenv("EDGE_PLAYBACK_SRT_FILE")

	var err error
	mp3Fname, srtFname, err = createTempFiles(*useMPV, mp3Fname, srtFname, debug)
	if err != nil {
		return err
	}
	if !keep {
		defer cleanup(mp3Fname, srtFname)
	} else {
		fmt.Printf("\nKeeping temporary files: %s", mp3Fname)
		if srtFname != "" {
			fmt.Printf(" and %s", srtFname)
		}
		fmt.Println()
	}

	if err := runEdgeTTS(mp3Fname, srtFname, ttsArgs); err != nil {
		return err
	}
	return playMedia(*useMPV, mp3Fname, srtFname)
}

func createTempFiles(useMPV bool, mp3Fname, srtFname string, debug bool) (string, string, error) {
	if mp3Fname == "" {
		f, err := os.CreateTemp("", "edge-playback-*.mp3")
		if err != nil {
			return "", "", err
		}
		mp3Fname = f.Name()
		f.Close()
		if debug {
			fmt.Println("Media file:", mp3Fname)
		}
	}
	if srtFname == "" && useMPV {
		f, err := os.CreateTemp("", "edge-playback-*.srt")
		if err != nil {
			return "", "", err
		}
		srtFname = f.Name()
		f.Close()
	}
	if debug && srtFname != "" {
		fmt.Println("Subtitle file:", srtFname)
	}
	return mp3Fname, srtFname, nil
}

func runEdgeTTS(mp3Fname, srtFname string, ttsArgs []string) error {
	args := []string{"--write-media=" + mp3Fname}
	if srtFname != "" {
		args = append(args, "--write-subtitles="+srtFname)
	}
	args = append(args, ttsArgs...)
	cmd := exec.Command("edge-tts", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func playMedia(useMPV bool, mp3Fname, srtFname string) error {
	if !useMPV {
		return fmt.Errorf("non-mpv playback is not supported by the Go port; install mpv")
	}
	args := []string{"--msg-level=all=error,statusline=status"}
	if srtFname != "" {
		args = append(args, "--sub-file="+srtFname)
	}
	args = append(args, mp3Fname)
	cmd := exec.Command("mpv", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func cleanup(mp3Fname, srtFname string) {
	if mp3Fname != "" {
		os.Remove(mp3Fname)
	}
	if srtFname != "" {
		os.Remove(srtFname)
	}
}
