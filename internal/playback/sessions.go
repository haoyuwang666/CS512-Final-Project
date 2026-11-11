// TODO(Tian): Add persistent session/event retention and active buffer snapshots; current process retains at most 100 sessions and 256 events each.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	ID      string  `json:"session_id"`
	Summary Summary `json:"summary"`
	Events  []Event `json:"events"`
	cancel  context.CancelFunc
}
type Sessions struct {
	mu     sync.Mutex
	items  map[string]*Session
	engine *Engine
	root   context.Context
}

func NewSessions(ctx context.Context, e *Engine) *Sessions {
	return &Sessions{items: map[string]*Session{}, engine: e, root: ctx}
}
func (s *Sessions) snapshot(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Session{}, false
	}
	out := *item
	out.Events = append([]Event{}, item.Events...)
	return out, true
}
func (s *Sessions) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/health", protocol.Health)
	m.HandleFunc("POST /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		var body protocol.SessionRequest
		if err := protocol.Decode(w, r, &body); err != nil || !domain.ValidID(body.VideoID) {
			protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid session request")
			return
		}
		if body.Strategy != "" && body.Strategy != "round_robin" {
			protocol.Fail(w, 501, "NOT_IMPLEMENTED", ErrDeadlineNotImplemented.Error())
			return
		}
		id := fmt.Sprintf("session-%d", time.Now().UnixNano())
		ctx, cancel := context.WithTimeout(s.root, 10*time.Minute)
		s.mu.Lock()
		if len(s.items) >= 100 {
			s.mu.Unlock()
			cancel()
			protocol.Fail(w, 429, "CAPACITY_EXCEEDED", "session retention full; restart demo API")
			return
		}
		s.items[id] = &Session{ID: id, Summary: Summary{VideoID: body.VideoID, Status: "running", SimulatedPlayback: true}, Events: []Event{}, cancel: cancel}
		s.mu.Unlock()
		go func() {
			defer cancel()
			result := s.engine.Run(ctx, body.VideoID, Config{Window: 3, Concurrency: 2, Strategy: body.Strategy}, func(event Event) {
				s.mu.Lock()
				defer s.mu.Unlock()
				item := s.items[id]
				item.Events = append(item.Events, event)
				if len(item.Events) > 256 {
					item.Events = item.Events[len(item.Events)-256:]
				}
			})
			s.mu.Lock()
			s.items[id].Summary = result
			s.mu.Unlock()
		}()
		protocol.JSON(w, 202, map[string]string{"session_id": id})
	})
	m.HandleFunc("GET /api/v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		out, ok := s.snapshot(r.PathValue("id"))
		if !ok {
			protocol.Fail(w, 404, "NOT_FOUND", "session not found")
			return
		}
		protocol.JSON(w, 200, out)
	})
	m.HandleFunc("POST /api/v1/sessions/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		item, ok := s.items[r.PathValue("id")]
		s.mu.Unlock()
		if !ok {
			protocol.Fail(w, 404, "NOT_FOUND", "session not found")
			return
		}
		item.cancel()
		protocol.JSON(w, 202, map[string]bool{"ok": true})
	})
	m.HandleFunc("GET /api/v1/sessions/{id}/events", s.events)
	return m
}
func (s *Sessions) events(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.snapshot(r.PathValue("id")); !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "session not found")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		protocol.Fail(w, 500, "STREAM_ERROR", "streaming unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	last := int64(0)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		item, _ := s.snapshot(r.PathValue("id"))
		for _, event := range item.Events {
			if event.Sequence > last {
				b, _ := json.Marshal(event)
				if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", event.Sequence, b); err != nil {
					return
				}
				last = event.Sequence
			}
		}
		flusher.Flush()
		if item.Summary.Status != "running" {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
