// TODO(Tian): Introduce schema migration before changing persisted session records.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

const retainedEvents = 256
const retainedSessions = 100

type Session struct {
	ID               string         `json:"session_id"`
	Summary          Summary        `json:"summary"`
	Events           []Event        `json:"events"`
	Buffer           []BufferEntry  `json:"buffer"`
	Peers            []PeerEstimate `json:"peers"`
	PersistenceError string         `json:"persistence_error,omitempty"`
	cancel           context.CancelFunc
	journal          *os.File
}
type Sessions struct {
	mu     sync.Mutex
	items  map[string]*Session
	engine *Engine
	root   context.Context
	dir    string
}

func NewSessions(ctx context.Context, e *Engine) *Sessions {
	return &Sessions{items: map[string]*Session{}, engine: e, root: ctx}
}

// OpenSessions restores retained snapshots, but never silently restarts interrupted playback.
// Only one Demo API process may own a session directory at a time.
func OpenSessions(ctx context.Context, e *Engine, dir string) (*Sessions, error) {
	s := NewSessions(ctx, e)
	s.dir = dir
	if dir == "" {
		return s, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, "session-*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	for _, path := range paths {
		id := filepath.Base(path)
		id = id[:len(id)-len(".jsonl")]
		if !domain.ValidID(id) {
			continue
		}
		events, validBytes, truncated, err := readJournal(path, 0)
		if err != nil {
			return nil, fmt.Errorf("restore %s: %w", id, err)
		}
		if truncated {
			if err := os.Truncate(path, validBytes); err != nil {
				return nil, err
			}
			// Decoder offsets end before trailing whitespace. Restore the newline
			// so every surviving event remains a separate JSONL record.
			file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			_, err = file.WriteString("\n")
			if err == nil {
				err = file.Sync()
			}
			file.Close()
			if err != nil {
				return nil, err
			}
		}
		item := &Session{ID: id, Events: []Event{}, cancel: func() {}}
		for _, event := range events {
			s.apply(item, event)
		}
		if len(events) == 0 {
			continue
		}
		if item.Summary.Status == "running" {
			item.Summary.Status = "stopped"
			item.Summary.Error = "demo API restarted during playback"
			event := Event{Sequence: events[len(events)-1].Sequence + 1, Type: "session_stopped", Time: time.Now().UTC(), Message: item.Summary.Error, Snapshot: &Snapshot{Summary: item.Summary, Buffer: item.Buffer, Peers: item.Peers}}
			file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			err = writeEvent(file, event)
			file.Close()
			if err != nil {
				return nil, err
			}
			s.apply(item, event)
		}
		s.items[id] = item
	}
	for len(s.items) > retainedSessions {
		if err := s.evict(); err != nil {
			return nil, err
		}
	}
	return s, nil
}
func writeEvent(file *os.File, event Event) error {
	if err := json.NewEncoder(file).Encode(event); err != nil {
		return err
	}
	return file.Sync()
}

// Truncated final records are ignored after a crash. Other corruption is reported.
func scanJournal(file *os.File, visit func(Event) error) (valid int64, truncated bool, err error) {
	decoder := json.NewDecoder(file)
	for {
		var event Event
		err = decoder.Decode(&event)
		if err == io.EOF {
			return valid, false, nil
		}
		if err == io.ErrUnexpectedEOF {
			return valid, true, nil
		}
		if err != nil {
			return valid, false, err
		}
		valid = decoder.InputOffset()
		if err := visit(event); err != nil {
			return valid, false, err
		}
	}
}
func readJournal(path string, after int64) ([]Event, int64, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer file.Close()
	events := []Event{}
	valid, truncated, err := scanJournal(file, func(event Event) error {
		if event.Sequence > after {
			if len(events) > 0 {
				events[len(events)-1].Snapshot = nil
			}
			events = append(events, event)
			if len(events) > retainedEvents {
				events = events[len(events)-retainedEvents:]
			}
		}
		return nil
	})
	return events, valid, truncated, err
}

func (s *Sessions) apply(item *Session, event Event) {
	if event.Snapshot != nil {
		item.Summary = event.Snapshot.Summary
		item.Buffer = append([]BufferEntry{}, event.Snapshot.Buffer...)
		item.Peers = append([]PeerEstimate{}, event.Snapshot.Peers...)
	}
	// Snapshots are available through REST; don't duplicate them in the in-memory event ring.
	event.Snapshot = nil
	item.Events = append(item.Events, event)
	if len(item.Events) > retainedEvents {
		item.Events = item.Events[len(item.Events)-retainedEvents:]
	}
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
	out.Buffer = append([]BufferEntry{}, item.Buffer...)
	out.Peers = append([]PeerEstimate{}, item.Peers...)
	return out, true
}

// Evict completed sessions only. Active sessions are never displaced by new requests.
func (s *Sessions) evict() error {
	ids := []string{}
	for id, item := range s.items {
		if item.Summary.Status != "running" && item.journal == nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return fmt.Errorf("all retained sessions are active")
	}
	id := ids[0]
	if s.dir != "" {
		if err := os.Remove(filepath.Join(s.dir, id+".jsonl")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	delete(s.items, id)
	return nil
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
		id := fmt.Sprintf("session-%d", time.Now().UnixNano())
		ctx, cancel := context.WithTimeout(s.root, 10*time.Minute)
		s.mu.Lock()
		if len(s.items) >= retainedSessions {
			if err := s.evict(); err != nil {
				s.mu.Unlock()
				cancel()
				protocol.Fail(w, 429, "CAPACITY_EXCEEDED", err.Error())
				return
			}
		}
		item := &Session{ID: id, Summary: Summary{VideoID: body.VideoID, Status: "running", SimulatedPlayback: true}, Events: []Event{}, Buffer: []BufferEntry{}, Peers: []PeerEstimate{}, cancel: cancel}
		if s.dir != "" {
			file, err := os.OpenFile(filepath.Join(s.dir, id+".jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				s.mu.Unlock()
				cancel()
				protocol.Fail(w, 500, "PERSISTENCE_ERROR", err.Error())
				return
			}
			item.journal = file
		}
		s.items[id] = item
		s.mu.Unlock()
		go func() {
			defer cancel()
			result := s.engine.Run(ctx, body.VideoID, cfg, func(event Event) {
				s.mu.Lock()
				defer s.mu.Unlock()
				if item.journal != nil && item.PersistenceError == "" {
					if err := writeEvent(item.journal, event); err != nil {
						item.PersistenceError = err.Error()
						cancel()
					}
				}
				s.apply(item, event)
			})
			s.mu.Lock()
			defer s.mu.Unlock()
			if item.journal != nil {
				if err := item.journal.Close(); err != nil {
					item.PersistenceError = err.Error()
				}
				item.journal = nil
			}
			if item.PersistenceError != "" {
				result.Status = "failed"
				result.Error = "event persistence failed: " + item.PersistenceError
			}
			item.Summary = result
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
		if ok {
			item.cancel()
		}
		s.mu.Unlock()
		if !ok {
			protocol.Fail(w, 404, "NOT_FOUND", "session not found")
			return
		}
		protocol.JSON(w, 202, map[string]bool{"ok": true})
	})
	m.HandleFunc("GET /api/v1/sessions/{id}/events", s.events)
	return m
}
func (s *Sessions) events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, ok := s.snapshot(id)
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "session not found")
		return
	}
	last := int64(0)
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		var err error
		last, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || last < 0 {
			protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid Last-Event-ID")
			return
		}
	}
	if len(item.Events) > 0 && last > item.Events[len(item.Events)-1].Sequence {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "Last-Event-ID is ahead of session")
		return
	}
	replay := item.Events
	var archive *os.File
	if len(replay) > 0 && last < replay[0].Sequence-1 {
		if s.dir == "" {
			protocol.Fail(w, 409, "EVENT_HISTORY_EXPIRED", "fetch the session snapshot and reconnect with its latest sequence")
			return
		}
		var err error
		archive, err = os.Open(filepath.Join(s.dir, id+".jsonl"))
		if err != nil {
			protocol.Fail(w, 500, "PERSISTENCE_ERROR", err.Error())
			return
		}
		defer archive.Close()
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		protocol.Fail(w, 500, "STREAM_ERROR", "streaming unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	// Stream older records from disk instead of loading an unbounded history into memory.
	if archive != nil {
		_, _, err := scanJournal(archive, func(event Event) error {
			if event.Sequence <= last {
				return nil
			}
			event.Snapshot = nil
			b, _ := json.Marshal(event)
			if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", event.Sequence, b); err != nil {
				return err
			}
			last = event.Sequence
			return nil
		})
		if err != nil {
			fmt.Fprint(w, "event: reset\ndata: {\"reason\":\"event journal unavailable\"}\n\n")
			flusher.Flush()
			return
		}
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		for _, event := range replay {
			if event.Sequence > last {
				event.Snapshot = nil
				b, _ := json.Marshal(event)
				if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", event.Sequence, b); err != nil {
					return
				}
				last = event.Sequence
			}
		}
		flusher.Flush()
		// Re-read after replay: a terminal event can arrive while the response is written.
		item, ok = s.snapshot(id)
		if !ok {
			return
		}
		if len(item.Events) > 0 && last < item.Events[0].Sequence-1 {
			fmt.Fprint(w, "event: reset\ndata: {\"reason\":\"event history expired\"}\n\n")
			flusher.Flush()
			return
		}
		replay = item.Events
		latest := int64(0)
		if len(replay) > 0 {
			latest = replay[len(replay)-1].Sequence
		}
		if item.Summary.Status != "running" && last >= latest {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
