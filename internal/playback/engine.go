// TODO(Tian): Keep headless consumption and real media decoding as explicit, separately measured modes.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Config bounds logical buffering independently from simultaneous HTTP transfers.
type Config struct {
	Window          int
	Concurrency     int
	Strategy        string
	StartupBufferMS int64
}

func (c Config) Validate() error {
	if c.Window < 1 || c.Window > 64 || c.Concurrency < 1 || c.Concurrency > c.Window || c.StartupBufferMS < 0 || c.StartupBufferMS > 3600000 {
		return fmt.Errorf("invalid window/concurrency/startup buffer")
	}
	_, err := NewScheduler(c.Strategy)
	return err
}

type Engine struct {
	Metadata   *MetadataClient
	Downloader *Downloader
	Clock      Clock
}

func NewEngine(urls []string) *Engine {
	return &Engine{Metadata: NewMetadataClient(urls), Downloader: NewDownloader(), Clock: RealClock{}}
}

// Run owns all session state. Worker updates and callbacks are serialized, while
// HTTP requests run concurrently. A callback must not block or re-enter Run.
func (e *Engine) Run(parent context.Context, id string, cfg Config, emit func(Event)) (summary Summary) {
	clock := e.Clock
	if clock == nil {
		clock = RealClock{}
	}
	start := clock.Now()
	summary = Summary{VideoID: id, Status: "running", SimulatedPlayback: true}
	ctx, cancel := context.WithCancel(parent)
	var workers sync.WaitGroup
	var mu sync.Mutex
	var sequence int64
	entries := map[int]BufferEntry{}
	tracker := newDeadline()
	var origin time.Time
	var shift time.Duration
	publishLocked := func(event Event) {
		summary.finalize()
		buffer := make([]BufferEntry, 0, len(entries))
		for _, b := range entries {
			buffer = append(buffer, b)
		}
		sort.Slice(buffer, func(i, j int) bool { return buffer[i].Index < buffer[j].Index })
		sequence++
		event.Sequence = sequence
		event.Time = clock.Now().UTC()
		event.SinceStartMS = float64(clock.Now().Sub(start)) / float64(time.Millisecond)
		event.Snapshot = &Snapshot{Summary: summary, Buffer: buffer, Peers: tracker.snapshot()}
		if emit != nil {
			emit(event)
		}
	}
	update := func(event Event, change func()) {
		mu.Lock()
		defer mu.Unlock()
		if change != nil {
			change()
		}
		publishLocked(event)
	}
	defer func() {
		cancel()
		workers.Wait()
		mu.Lock()
		defer mu.Unlock()
		if summary.Status == "running" {
			summary.Status = "failed"
		}
		if summary.Status != "completed" && parent.Err() != nil {
			summary.Status = "stopped"
			summary.Error = parent.Err().Error()
		}
		kind := "session_" + summary.Status
		publishLocked(Event{Type: kind, Message: summary.Error})
	}()
	setError := func(err error) { mu.Lock(); summary.Error = err.Error(); mu.Unlock() }
	update(Event{Type: "session_started"}, nil)
	if !domain.ValidID(id) {
		setError(fmt.Errorf("invalid video ID"))
		return
	}
	if err := cfg.Validate(); err != nil {
		setError(err)
		return
	}
	scheduler, _ := NewScheduler(cfg.Strategy)
	if cfg.Strategy == "deadline" {
		scheduler = tracker
	}
	manifest, err := e.Metadata.Manifest(ctx, id)
	if err != nil {
		setError(err)
		return
	}
	// Media offsets serve both frozen measurement and stall-adjusted scheduling.
	offsets := make([]time.Duration, len(manifest.Segments))
	var total time.Duration
	for i, s := range manifest.Segments {
		offsets[i] = total
		total += time.Duration(s.DurationMS) * time.Millisecond
	}
	// Startup buffering is a contiguous prefix and must fit inside the bounded window.
	target := min(time.Duration(cfg.StartupBufferMS)*time.Millisecond, total)
	startupCount := 1
	var buffered time.Duration
	for i, s := range manifest.Segments {
		buffered += time.Duration(s.DurationMS) * time.Millisecond
		startupCount = i + 1
		if buffered >= target {
			break
		}
	}
	if startupCount > cfg.Window {
		setError(fmt.Errorf("startup buffer exceeds prefetch window"))
		return
	}
	slots := map[int]bufferSlot{}
	ready := map[int]downloadResult{}
	type job struct {
		index int
		slot  bufferSlot
	}
	jobs := make(chan job, cfg.Window)
	// A FIFO worker queue avoids semaphore races reordering the initial prefix.
	for w := 0; w < cfg.Concurrency; w++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				var j job
				select {
				case <-ctx.Done():
					return
				case j = <-jobs:
				}
				seg := manifest.Segments[j.index]
				var last error
				result := downloadResult{}
				for refresh := 0; refresh < 2; refresh++ {
					var locations []domain.Location
					if refresh == 0 {
						locations, last = e.Metadata.Locations(ctx, seg.ID)
					} else {
						locations, last = e.Metadata.RefreshLocations(ctx, seg.ID)
					}
					if last != nil {
						continue
					}
					for _, p := range scheduler.Candidates(seg, locations) {
						if ctx.Err() != nil {
							last = ctx.Err()
							break
						}
						mu.Lock()
						base := origin
						if base.IsZero() {
							base = start
						}
						due := base.Add(offsets[j.index] + shift)
						mu.Unlock()
						requestCtx := ctx
						stop := func() {}
						if cfg.Strategy == "deadline" {
							requestCtx, stop = context.WithTimeout(ctx, tracker.budget(seg, p.ID, due.Sub(clock.Now())))
						}
						tracker.start(p.ID)
						update(Event{Type: "segment_requested", SegmentID: seg.ID, PeerID: p.ID, Deadline: &due}, func() {
							summary.Requests++
							b := entries[j.index]
							b.State = "downloading"
							b.PeerID = p.ID
							entries[j.index] = b
						})
						began := clock.Now()
						data, n, downloadErr := e.Downloader.download(requestCtx, seg, p)
						elapsed := clock.Now().Sub(began)
						stop()
						tracker.finish(p.ID, n, elapsed, downloadErr)
						result.finished = clock.Now()
						kind := "segment_verified"
						message := ""
						if downloadErr != nil {
							kind = "request_failed"
							message = downloadErr.Error()
						}
						update(Event{Type: kind, SegmentID: seg.ID, PeerID: p.ID, Bytes: n, ElapsedMS: float64(elapsed) / float64(time.Millisecond), Message: message, Deadline: &due}, func() {
							summary.DownloadedBytes += n
							b := entries[j.index]
							if downloadErr != nil {
								summary.FailedRequests++
								b.State = "retrying"
							} else {
								summary.VerifiedBytes += int64(len(data))
								b.State = "ready"
							}
							entries[j.index] = b
						})
						if downloadErr == nil {
							result.size = len(data)
							last = nil
							break
						}
						last = downloadErr
					}
					if result.size > 0 {
						break
					}
				}
				if result.size == 0 {
					if last == nil {
						last = fmt.Errorf("no reachable source for %s", seg.ID)
					}
					result.err = last
					update(Event{Type: "segment_failed", SegmentID: seg.ID, Message: last.Error()}, func() { b := entries[j.index]; b.State = "failed"; entries[j.index] = b })
				}
				// The single-result channel is buffered so cancellation cannot strand a worker.
				j.slot <- result
			}
		}()
	}
	next := 0
	schedule := func(index int) {
		seg := manifest.Segments[index]
		slot := newSlot()
		slots[index] = slot
		update(Event{Type: "segment_queued", SegmentID: seg.ID}, func() {
			entries[index] = BufferEntry{SegmentID: seg.ID, Index: index, DurationMS: seg.DurationMS, State: "queued"}
		})
		jobs <- job{index, slot}
	}
	for next < len(manifest.Segments) && next < cfg.Window {
		schedule(next)
		next++
	}
	await := func(index int) (downloadResult, error) {
		if r, ok := ready[index]; ok {
			return r, r.err
		}
		select {
		case r := <-slots[index]:
			ready[index] = r
			return r, r.err
		case <-ctx.Done():
			return downloadResult{}, ctx.Err()
		}
	}
	for i := 0; i < startupCount; i++ {
		if _, err := await(i); err != nil {
			setError(err)
			return
		}
	}
	update(Event{Type: "playback_started"}, func() {
		origin = clock.Now()
		summary.StartupMS = float64(origin.Sub(start)) / float64(time.Millisecond)
	})
	for i, seg := range manifest.Segments {
		var result downloadResult
		var available bool
		if r, ok := ready[i]; ok {
			result = r
			available = true
		} else {
			select {
			case result = <-slots[i]:
				ready[i] = result
				available = true
			default:
			}
		}
		if !available {
			waitStart := clock.Now()
			update(Event{Type: "stall_started", SegmentID: seg.ID}, nil)
			accounted := time.Duration(0)
			recordStall := func(kind string) {
				update(Event{Type: kind, SegmentID: seg.ID}, func() {
					elapsed := clock.Now().Sub(waitStart)
					delta := elapsed - accounted
					summary.StallMS += float64(delta) / float64(time.Millisecond)
					shift += delta
					accounted = elapsed
				})
			}
			// Progress ticks observe the injected clock; they never advance media time.
			ticker := time.NewTicker(250 * time.Millisecond)
			waiting := true
			for waiting {
				select {
				case result = <-slots[i]:
					err = result.err
					waiting = false
				case <-ctx.Done():
					err = ctx.Err()
					waiting = false
				case <-ticker.C:
					recordStall("stall_progress")
				}
			}
			ticker.Stop()
			recordStall("stall_ended")
			if err != nil {
				setError(err)
				return
			}
		}
		if result.err != nil {
			setError(result.err)
			return
		}
		// Frozen deadlines measure lateness against the original playback timeline.
		// Scheduling deadlines separately move by accumulated stalls to allow recovery.
		update(Event{Type: "segment_consuming", SegmentID: seg.ID}, func() {
			if i > 0 && result.finished.After(origin.Add(offsets[i])) {
				summary.DeadlineMisses++
			}
			b := entries[i]
			b.State = "playing"
			entries[i] = b
		})
		remaining := time.Duration(seg.DurationMS) * time.Millisecond
		for remaining > 0 {
			elapsed, consumeErr := clock.Wait(ctx, min(remaining, 250*time.Millisecond))
			update(Event{Type: "playback_progress", SegmentID: seg.ID}, func() { summary.MediaMS += float64(elapsed) / float64(time.Millisecond) })
			remaining -= elapsed
			if consumeErr != nil {
				setError(consumeErr)
				return
			}
		}
		delete(slots, i)
		delete(ready, i)
		update(Event{Type: "segment_played", SegmentID: seg.ID}, func() { summary.SegmentsPlayed++; delete(entries, i) })
		for next < len(manifest.Segments) && next < i+1+cfg.Window {
			schedule(next)
			next++
		}
	}
	mu.Lock()
	summary.Status = "completed"
	mu.Unlock()
	return
}
