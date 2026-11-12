// TODO(Tian): Keep verified media delivery independent of browser buffering policy.
package playback

import (
	"bytes"
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// MediaSessions serves verified MP4 objects to a real browser player. It does not
// run the simulated media clock or report server download time as playback time.
type MediaSessions struct {
	mu     sync.Mutex
	items  map[string]*mediaSession
	root   context.Context
	engine *Engine
}
type mediaSession struct {
	manifest  domain.Manifest
	config    Config
	scheduler Scheduler
	tracker   *Deadline
	slots     chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	touched   time.Time // guarded by MediaSessions.mu
}

func NewMediaSessions(ctx context.Context, engine *Engine) *MediaSessions {
	return &MediaSessions{items: make(map[string]*mediaSession), root: ctx, engine: engine}
}
func (s *MediaSessions) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /api/v1/media/sessions", s.create)
	m.HandleFunc("GET /api/v1/media/sessions/{id}/segments/{index}", s.segment)
	m.HandleFunc("POST /api/v1/media/sessions/{id}/stop", s.stop)
	return m
}
func (s *MediaSessions) create(w http.ResponseWriter, r *http.Request) {
	var body protocol.SessionRequest
	if err := protocol.Decode(w, r, &body); err != nil || !domain.ValidID(body.VideoID) {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid media request")
		return
	}
	cfg := Config{Window: body.Window, Concurrency: body.Concurrency, Strategy: body.Strategy, StartupBufferMS: body.StartupBufferMS}
	if cfg.Window == 0 {
		cfg.Window = 3
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = min(2, cfg.Window)
	}
	if err := cfg.Validate(); err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", err.Error())
		return
	}
	manifest, err := s.engine.Metadata.Manifest(r.Context(), body.VideoID)
	if err != nil {
		status := 503
		var me *metadataError
		if errors.As(err, &me) {
			status = me.status
		}
		protocol.Fail(w, status, "METADATA_ERROR", err.Error())
		return
	}
	switch manifest.Representation {
	case "continuous-fmp4-h264-v1", "continuous-fmp4-h264-aac-v1":
	case "independent-mp4-h264-v1", "independent-mp4-h264-aac-v1":
		protocol.Fail(w, 422, "REPACKAGE_REQUIRED", "旧视频需要从原文件重新生成连续 fMP4 分片，并使用新视频 ID 导入，才能无缝播放。")
		return
	default:
		protocol.Fail(w, 422, "UNSUPPORTED_MEDIA", "this video is not a continuous fMP4 representation; byte fixture data cannot be decoded")
		return
	}
	var prefix int64
	for i, seg := range manifest.Segments {
		prefix += seg.DurationMS
		if prefix >= cfg.StartupBufferMS || i == len(manifest.Segments)-1 {
			if i+1 > cfg.Window {
				protocol.Fail(w, 400, "INVALID_ARGUMENT", "startup buffer exceeds prefetch window")
				return
			}
			break
		}
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		protocol.Fail(w, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	id := "media-" + hex.EncodeToString(token)
	ctx, cancel := context.WithCancel(s.root)
	tracker := newDeadline()
	scheduler, _ := NewScheduler(cfg.Strategy)
	if cfg.Strategy == "deadline" {
		scheduler = tracker
	}
	item := &mediaSession{manifest: manifest, config: cfg, scheduler: scheduler, tracker: tracker, slots: make(chan struct{}, cfg.Concurrency), ctx: ctx, cancel: cancel, touched: time.Now()}
	s.mu.Lock()
	// Idle sessions are leases, not durable simulated sessions. Closing a tab does
	// not leave a worker running, and old leases cannot exhaust capacity forever.
	for key, old := range s.items {
		if time.Since(old.touched) > 30*time.Minute {
			old.cancel()
			delete(s.items, key)
		}
	}
	if len(s.items) >= 100 {
		s.mu.Unlock()
		cancel()
		protocol.Fail(w, 429, "CAPACITY_EXCEEDED", "too many active media sessions")
		return
	}
	s.items[id] = item
	s.mu.Unlock()
	protocol.JSON(w, 201, map[string]any{"session_id": id, "manifest": manifest, "window": cfg.Window, "concurrency": cfg.Concurrency, "startup_buffer_ms": cfg.StartupBufferMS})
}
func (s *MediaSessions) get(id string) (*mediaSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil, false
	}
	if time.Since(item.touched) > 30*time.Minute {
		item.cancel()
		delete(s.items, id)
		return nil, false
	}
	item.touched = time.Now()
	return item, true
}
func (s *MediaSessions) stop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	item, ok := s.items[r.PathValue("id")]
	if ok {
		item.cancel()
		delete(s.items, r.PathValue("id"))
	}
	s.mu.Unlock()
	// Idempotence lets browser cleanup safely race expiry, explicit stop and unload.
	protocol.JSON(w, 200, map[string]bool{"ok": true})
}
func (s *MediaSessions) segment(w http.ResponseWriter, r *http.Request) {
	item, ok := s.get(r.PathValue("id"))
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "media session expired or stopped")
		return
	}
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 || index >= len(item.manifest.Segments) {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid segment index")
		return
	}
	// Both disconnects and explicit stop cancel an in-progress Peer request.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	detach := context.AfterFunc(item.ctx, cancel)
	defer detach()
	cancelled := func() {
		if r.Context().Err() == nil {
			protocol.Fail(w, 410, "MEDIA_STOPPED", "media session stopped")
		}
	}
	select {
	case item.slots <- struct{}{}:
		defer func() { <-item.slots }()
	case <-ctx.Done():
		cancelled()
		return
	}
	seg := item.manifest.Segments[index]
	began := time.Now()
	var last error
	var attempts, failures int
	var received int64
	for refresh := 0; refresh < 2; refresh++ {
		var locations []domain.Location
		if refresh == 0 {
			locations, last = s.engine.Metadata.Locations(ctx, seg.ID)
		} else {
			locations, last = s.engine.Metadata.RefreshLocations(ctx, seg.ID)
		}
		if last != nil {
			continue
		}
		for _, p := range item.scheduler.Candidates(seg, locations) {
			if ctx.Err() != nil {
				cancelled()
				return
			}
			requestCtx := ctx
			stop := func() {}
			// Browser request arrival expresses demand. Prediction bounds a slow-source
			// attempt; only the browser can measure actual decode/playback deadlines.
			if item.config.Strategy == "deadline" {
				requestCtx, stop = context.WithTimeout(ctx, item.tracker.budget(seg, p.ID, 0))
			}
			item.tracker.start(p.ID)
			attemptStart := time.Now()
			attempts++
			data, n, downloadErr := s.engine.Downloader.download(requestCtx, seg, p)
			stop()
			received += n
			item.tracker.finish(p.ID, n, time.Since(attemptStart), downloadErr)
			if downloadErr != nil {
				last = downloadErr
				failures++
				continue
			}
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Peer-ID", p.ID)
			w.Header().Set("X-Verified-SHA256", seg.SHA256)
			w.Header().Set("X-Download-Bytes", strconv.FormatInt(received, 10))
			w.Header().Set("X-Download-Attempts", strconv.Itoa(attempts))
			w.Header().Set("X-Download-Failures", strconv.Itoa(failures))
			w.Header().Set("X-Download-Elapsed-MS", fmt.Sprintf("%.3f", float64(time.Since(began))/float64(time.Millisecond)))
			// Only fully verified bytes leave the gateway; ServeContent also handles ranges.
			http.ServeContent(w, r, seg.ID+".mp4", time.Time{}, bytes.NewReader(data))
			return
		}
	}
	if ctx.Err() != nil {
		cancelled()
		return
	}
	if last == nil {
		last = fmt.Errorf("no reachable replica for %s", seg.ID)
	}
	protocol.Fail(w, 502, "MEDIA_UNAVAILABLE", last.Error())
}
