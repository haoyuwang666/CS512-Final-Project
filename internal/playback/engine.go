// TODO(Tian): Add configurable startup buffering, cached metadata fallback and deadline metrics; baseline uses bounded concurrent prefetch.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"fmt"
	"sync"
	"time"
)

type Config struct {
	Window      int
	Concurrency int
	Strategy    string
}
type Engine struct {
	Metadata   *MetadataClient
	Downloader *Downloader
}

func NewEngine(urls []string) *Engine {
	return &Engine{Metadata: NewMetadataClient(urls), Downloader: NewDownloader()}
}
func (e *Engine) Run(parent context.Context, id string, cfg Config, emit func(Event)) (summary Summary) {
	start := time.Now()
	summary = Summary{VideoID: id, Status: "failed", SimulatedPlayback: true}
	var workers sync.WaitGroup
	ctx, cancel := context.WithCancel(parent)
	defer func() { cancel(); workers.Wait(); summary.finalize() }()
	if !domain.ValidID(id) || cfg.Window < 1 || cfg.Window > 64 || cfg.Concurrency < 1 || cfg.Concurrency > cfg.Window {
		summary.Error = "invalid video/window/concurrency"
		return
	}
	scheduler, err := NewScheduler(cfg.Strategy)
	if err != nil {
		summary.Error = err.Error()
		return
	}
	var eventMu sync.Mutex
	var sequence int64
	send := func(kind string, s domain.Segment, peer, message string) {
		eventMu.Lock()
		defer eventMu.Unlock()
		sequence++
		if emit != nil {
			emit(Event{Sequence: sequence, Type: kind, Time: time.Now().UTC(), SegmentID: s.ID, PeerID: peer, Message: message})
		}
	}
	send("session_started", domain.Segment{}, "", "")
	manifest, err := e.Metadata.Manifest(ctx, id)
	if err != nil {
		summary.Error = err.Error()
		return
	}
	slots := map[int]bufferSlot{}
	sem := make(chan struct{}, cfg.Concurrency)
	next := 0
	schedule := func(index int) {
		seg := manifest.Segments[index]
		slot := newSlot()
		slots[index] = slot
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				slot <- downloadResult{err: ctx.Err()}
				return
			}
			defer func() { <-sem }()
			var last error
			for refresh := 0; refresh < 2; refresh++ {
				locations, err := e.Metadata.Locations(ctx, seg.ID)
				if err != nil {
					last = err
					continue
				}
				for _, p := range scheduler.Candidates(seg, locations) {
					send("segment_requested", seg, p.ID, "")
					data, err := e.Downloader.Download(ctx, seg, p)
					if err != nil {
						last = err
						send("request_failed", seg, p.ID, err.Error())
						continue
					}
					send("segment_verified", seg, p.ID, "")
					slot <- downloadResult{size: len(data)}
					return
				}
			}
			if last == nil {
				last = fmt.Errorf("no reachable source for %s", seg.ID)
			}
			slot <- downloadResult{err: last}
		}()
	}
	for i, seg := range manifest.Segments {
		for next < len(manifest.Segments) && next < i+cfg.Window {
			schedule(next)
			next++
		}
		waitStart := time.Now()
		var result downloadResult
		select {
		case result = <-slots[i]:
		default:
			if i > 0 {
				send("stall_started", seg, "", "")
			}
			select {
			case result = <-slots[i]:
			case <-ctx.Done():
				result.err = ctx.Err()
			}
			if i > 0 {
				summary.StallMS += float64(time.Since(waitStart)) / float64(time.Millisecond)
				send("stall_ended", seg, "", "")
			}
		}
		delete(slots, i)
		if result.err != nil {
			summary.Error = result.err.Error()
			if parent.Err() != nil {
				summary.Status = "stopped"
			}
			send("session_failed", seg, "", summary.Error)
			return
		}
		summary.VerifiedBytes += int64(result.size)
		if i == 0 {
			summary.StartupMS = float64(time.Since(start)) / float64(time.Millisecond)
			send("playback_started", seg, "", "")
		}
		elapsed, err := consume(ctx, time.Duration(seg.DurationMS)*time.Millisecond)
		summary.MediaMS += float64(elapsed) / float64(time.Millisecond)
		if err != nil {
			summary.Status = "stopped"
			summary.Error = err.Error()
			return
		}
		summary.SegmentsPlayed++
		send("segment_played", seg, "", "")
	}
	summary.Status = "completed"
	send("session_completed", domain.Segment{}, "", "")
	return
}
