// TODO(Tian): Extend CLI configuration alongside the shared Go engine; never implement a separate CLI scheduler.
package main

import (
	"context"
	"cooperative.video/internal/playback"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"time"
)

func main() {
	controllers := flag.String("controllers", "http://127.0.0.1:8001", "comma-separated controller URLs")
	id := flag.String("video", "fixture", "video ID")
	window := flag.Int("window", 3, "prefetch window")
	parallel := flag.Int("concurrency", 2, "concurrent downloads")
	strategy := flag.String("strategy", "round_robin", "round_robin (deadline is TODO)")
	timeout := flag.Duration("timeout", 2*time.Minute, "session timeout")
	flag.Parse()
	ctx, stop := protocol.SignalContext()
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	result := playback.NewEngine(strings.Split(*controllers, ",")).Run(ctx, *id, playback.Config{Window: *window, Concurrency: *parallel, Strategy: *strategy}, func(event playback.Event) { _ = json.NewEncoder(os.Stderr).Encode(event) })
	_ = json.NewEncoder(os.Stdout).Encode(result)
	if result.Status != "completed" {
		os.Exit(1)
	}
}
