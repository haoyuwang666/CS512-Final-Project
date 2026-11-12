// TODO(Tian): Extend CLI configuration alongside the shared Go engine; never implement a separate CLI scheduler.
package main

import (
	"context"
	"cooperative.video/internal/playback"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	controllers := flag.String("controllers", "http://127.0.0.1:8001", "comma-separated controller URLs")
	id := flag.String("video", "fixture", "video ID")
	window := flag.Int("window", 3, "prefetch window")
	parallel := flag.Int("concurrency", 2, "concurrent downloads")
	strategy := flag.String("strategy", "round_robin", "round_robin or deadline")
	startup := flag.Int64("startup-buffer-ms", 0, "contiguous media to buffer before starting (0: first segment)")
	eventsPath := flag.String("events-file", "", "optional JSONL event journal (events also go to stderr)")
	timeout := flag.Duration("timeout", 2*time.Minute, "session timeout")
	flag.Parse()
	ctx, stop := protocol.SignalContext()
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	var journal *os.File
	if *eventsPath != "" {
		if err := os.MkdirAll(filepath.Dir(*eventsPath), 0700); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var err error
		journal, err = os.Create(*eventsPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer journal.Close()
	}
	var journalErr error
	result := playback.NewEngine(strings.Split(*controllers, ",")).Run(ctx, *id, playback.Config{Window: *window, Concurrency: *parallel, Strategy: *strategy, StartupBufferMS: *startup}, func(event playback.Event) {
		_ = json.NewEncoder(os.Stderr).Encode(event)
		if journal != nil && journalErr == nil {
			journalErr = json.NewEncoder(journal).Encode(event)
			if journalErr == nil {
				journalErr = journal.Sync()
			}
			if journalErr != nil {
				cancel()
			}
		}
	})
	if journalErr != nil {
		result.Status = "failed"
		result.Error = "event persistence failed: " + journalErr.Error()
	}
	_ = json.NewEncoder(os.Stdout).Encode(result)
	if result.Status != "completed" {
		os.Exit(1)
	}
}
