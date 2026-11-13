// TODO(Tian): Extend deterministic scenarios when media or scheduling contracts change.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *testClock) advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
func (c *testClock) Wait(ctx context.Context, d time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	c.advance(d)
	return d, nil
}

func fixtureEngine(t *testing.T, durations []int64, serve func(http.ResponseWriter, *http.Request)) (*Engine, []domain.Segment) {
	t.Helper()
	data := []byte("payload")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	segments := make([]domain.Segment, len(durations))
	for i, d := range durations {
		segments[i] = domain.Segment{ID: fmt.Sprintf("v--%d", i), Index: i, DurationMS: d, Size: int64(len(data)), SHA256: hash}
	}
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serve != nil {
			serve(w, r)
		} else {
			w.Write(data)
		}
	}))
	t.Cleanup(peer.Close)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/videos/v/manifest", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: segments})
	})
	mux.HandleFunc("GET /api/v1/segments/{id}/locations", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, protocol.Locations{SegmentID: r.PathValue("id"), Locations: []domain.Location{{Peer: domain.Peer{ID: "p", URL: peer.URL}, Version: 1}}})
	})
	meta := httptest.NewServer(mux)
	t.Cleanup(meta.Close)
	e := NewEngine([]string{meta.URL})
	e.Clock = &testClock{now: time.Unix(100, 0)}
	return e, segments
}
func TestFrozenDeadlinesAndStalls(t *testing.T) {
	stalled := make(chan struct{})
	var clock *testClock
	e, _ := fixtureEngine(t, []int64{5, 5}, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "v--1") {
			select {
			case <-stalled:
				clock.advance(80 * time.Millisecond)
			case <-r.Context().Done():
				return
			}
		}
		w.Write([]byte("payload"))
	})
	clock = e.Clock.(*testClock)
	result := e.Run(context.Background(), "v", Config{Window: 2, Concurrency: 2, Strategy: "deadline"}, func(event Event) {
		if event.Type == "stall_started" {
			close(stalled)
		}
	})
	if result.Status != "completed" || result.MediaMS != 10 || result.StallMS != 80 || result.DeadlineMisses != 1 {
		t.Fatalf("unexpected metrics: %+v", result)
	}
	if result.Requests != 2 || result.DownloadedBytes != 14 || result.VerifiedBytes != 14 {
		t.Fatalf("unexpected accounting: %+v", result)
	}
}
func TestStartupBufferSnapshotsAndCancellation(t *testing.T) {
	e, _ := fixtureEngine(t, []int64{1000, 1000, 1000}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	verified := 0
	started := false
	result := e.Run(ctx, "v", Config{Window: 2, Concurrency: 2, StartupBufferMS: 2000}, func(event Event) {
		if len(event.Snapshot.Buffer) > 2 {
			t.Error("buffer exceeded configured window")
		}
		if event.Type == "segment_verified" {
			verified++
		}
		if event.Type == "playback_started" {
			started = true
			if verified < 2 {
				t.Error("started before startup prefix was ready")
			}
		}
		if event.Type == "playback_progress" {
			if event.Snapshot.Summary.MediaMS != 250 {
				t.Error("missing live progress")
			}
			cancel()
		}
	})
	if !started || result.Status != "stopped" || result.MediaMS != 250 || result.SegmentsPlayed != 0 {
		t.Fatalf("unexpected cancellation: %+v", result)
	}
	result = e.Run(context.Background(), "v", Config{Window: 1, Concurrency: 1, StartupBufferMS: 2000}, nil)
	if result.Status != "failed" || !strings.Contains(result.Error, "window") {
		t.Fatalf("oversized startup buffer accepted: %+v", result)
	}
}
func TestDeadlineLearnsSourcesAndAccountsForQueues(t *testing.T) {
	d := newDeadline()
	seg := domain.Segment{Size: 1 << 20}
	locations := []domain.Location{{Peer: domain.Peer{ID: "slow"}}, {Peer: domain.Peer{ID: "fast"}}}
	for i := 0; i < 15; i++ {
		d.start("slow")
		d.finish("slow", 1<<20, time.Second, nil)
		d.start("fast")
		d.finish("fast", 1<<20, 100*time.Millisecond, nil)
	}
	if d.Candidates(seg, locations)[0].ID != "fast" {
		t.Fatal("did not learn faster source")
	}
	for i := 0; i < 20; i++ {
		d.start("fast")
	}
	if d.Candidates(seg, locations)[0].ID != "slow" {
		t.Fatal("ignored queued work")
	}
	if d.budget(seg, "fast", -time.Second) < 250*time.Millisecond {
		t.Fatal("overdue request has no recovery budget")
	}
	if _, err := NewScheduler("made-up"); err == nil {
		t.Fatal("unknown scheduler accepted")
	}
}
func TestMetadataCacheOutageAndAuthoritativeErrors(t *testing.T) {
	e, _ := fixtureEngine(t, []int64{1}, nil)
	c := e.Metadata
	c.retryTimeout = 30 * time.Millisecond
	m, err := c.Manifest(context.Background(), "v")
	if err != nil {
		t.Fatal(err)
	}
	m.Segments[0].Size = 12345
	if cached, _ := c.Manifest(context.Background(), "v"); cached.Segments[0].Size == 12345 {
		t.Fatal("caller mutated cache")
	}
	if _, err = c.Locations(context.Background(), "v--0"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "outage", 503) }))
	defer server.Close()
	c.URLs = []string{server.URL}
	c.mu.Lock()
	entry := c.manifests["v"]
	entry.fetched = time.Now().Add(-10 * time.Second)
	c.manifests["v"] = entry
	c.mu.Unlock()
	if _, err = c.Manifest(context.Background(), "v"); err != nil {
		t.Fatalf("cached manifest unavailable: %v", err)
	}
	if _, err = c.RefreshLocations(context.Background(), "v--0"); err != nil {
		t.Fatalf("cached locations unavailable: %v", err)
	}
	missing := httptest.NewServer(http.NotFoundHandler())
	defer missing.Close()
	c.URLs = []string{missing.URL}
	if _, err = c.Manifest(context.Background(), "v"); err == nil {
		t.Fatal("authoritative 404 hidden by cache")
	}
	if _, err = c.RefreshLocations(context.Background(), "v--0"); err == nil {
		t.Fatal("location 404 hidden by cache")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.Manifest(ctx, "v"); err != context.Canceled {
		t.Fatal("cancellation hidden by cache")
	}
}
func TestFailureAccountsCorruptAttemptsAndTerminalEvent(t *testing.T) {
	e, _ := fixtureEngine(t, []int64{1}, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("corrupt")) })
	terminal := ""
	result := e.Run(context.Background(), "v", Config{Window: 1, Concurrency: 1}, func(event Event) { terminal = event.Type })
	if result.Status != "failed" || result.FailedRequests != 2 || result.DownloadedBytes != 14 || result.VerifiedBytes != 0 || terminal != "session_failed" {
		t.Fatalf("bad failure accounting: %+v, %s", result, terminal)
	}
}
func TestSessionJournalReplayRecoveryAndRetention(t *testing.T) {
	dir := t.TempDir()
	id := "session-100"
	f, err := os.Create(filepath.Join(dir, id+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for i := int64(1); i <= 300; i++ {
		event := Event{Sequence: i, Type: "playback_progress", Snapshot: &Snapshot{Summary: Summary{VideoID: "v", Status: "running", MediaMS: float64(i)}}}
		if err := json.NewEncoder(f).Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	// Simulate a process dying in the middle of its last JSON record.
	f.WriteString(`{"sequence":301`)
	f.Close()
	s, err := OpenSessions(context.Background(), NewEngine(nil), dir)
	if err != nil {
		t.Fatal(err)
	}
	item, _ := s.snapshot(id)
	if item.Summary.Status != "stopped" || len(item.Events) != 256 || item.Events[255].Sequence != 301 {
		t.Fatalf("bad restore: %+v", item.Summary)
	}
	req := httptest.NewRequest("GET", "/api/v1/sessions/"+id+"/events", nil)
	req.Header.Set("Last-Event-ID", "3")
	response := httptest.NewRecorder()
	s.Handler().ServeHTTP(response, req)
	body := response.Body.String()
	if response.Code != 200 || strings.Contains(body, "id: 3\n") || !strings.Contains(body, "id: 4\n") || !strings.Contains(body, "id: 301\n") {
		t.Fatalf("bad replay: %d %s", response.Code, body)
	}
	if _, err := OpenSessions(context.Background(), NewEngine(nil), dir); err != nil {
		t.Fatalf("journal was not repaired: %v", err)
	}
	s = NewSessions(context.Background(), NewEngine(nil))
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("session-%03d", i)
		s.items[id] = &Session{ID: id, Summary: Summary{Status: "running"}}
	}
	if err := s.evict(); err == nil {
		t.Fatal("evicted active session")
	}
	s.items["session-000"].Summary.Status = "completed"
	if err := s.evict(); err != nil {
		t.Fatal(err)
	}
	if len(s.items) != 99 {
		t.Fatal("retention did not free capacity")
	}
}
func TestSessionAPIConfigAndPersistence(t *testing.T) {
	e, _ := fixtureEngine(t, []int64{1, 1}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	s, err := OpenSessions(ctx, e, dir)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	response, err := http.Post(server.URL+"/api/v1/sessions", "application/json", strings.NewReader(`{"video_id":"v","strategy":"deadline","window":2,"concurrency":2,"startup_buffer_ms":2}`))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	json.NewDecoder(response.Body).Decode(&out)
	response.Body.Close()
	if response.StatusCode != 202 {
		t.Fatal(response.StatusCode)
	}
	id := out["session_id"]
	deadline := time.Now().Add(3 * time.Second)
	for {
		item, _ := s.snapshot(id)
		s.mu.Lock()
		closed := s.items[id].journal == nil
		s.mu.Unlock()
		if item.Summary.Status == "completed" && closed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("session did not complete: %+v", item)
		}
		time.Sleep(time.Millisecond)
	}
	restored, err := OpenSessions(context.Background(), e, dir)
	if err != nil {
		t.Fatal(err)
	}
	item, _ := restored.snapshot(id)
	if item.Summary.Status != "completed" || item.Summary.VerifiedBytes != 14 {
		t.Fatalf("bad persisted summary: %+v", item.Summary)
	}
	for _, body := range []string{`{"video_id":"v","strategy":"unknown"}`, `{"video_id":"v","window":1,"concurrency":2}`, `{"video_id":"v","startup_buffer_ms":-1}`} {
		response, err := http.Post(server.URL+"/api/v1/sessions", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 400 {
			t.Fatalf("invalid config accepted: %s", body)
		}
	}
}

func TestDeadlineTimeoutRetriesAnotherReplica(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	data := []byte("payload")
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer fast.Close()
	seg := domain.Segment{ID: "v--0", DurationMS: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/manifest") {
			protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}})
			return
		}
		protocol.JSON(w, 200, protocol.Locations{SegmentID: seg.ID, Locations: []domain.Location{{Peer: domain.Peer{ID: "slow", URL: slow.URL}, Version: 1}, {Peer: domain.Peer{ID: "fast", URL: fast.URL}, Version: 1}}})
	}))
	defer meta.Close()
	e := NewEngine([]string{meta.URL})
	result := e.Run(context.Background(), "v", Config{Window: 1, Concurrency: 1, Strategy: "deadline"}, nil)
	if result.Status != "completed" || result.Requests != 2 || result.FailedRequests != 1 || result.VerifiedBytes != 7 {
		t.Fatalf("deadline retry failed: %+v", result)
	}
}

func TestSSERejectsExpiredAndInvalidCursors(t *testing.T) {
	s := NewSessions(context.Background(), NewEngine(nil))
	item := &Session{ID: "session-1", Summary: Summary{Status: "completed"}, Events: []Event{{Sequence: 10, Type: "session_completed"}}}
	s.items[item.ID] = item
	for cursor, status := range map[string]int{"0": 409, "-1": 400, "bad": 400, "11": 400, "10": 200} {
		request := httptest.NewRequest("GET", "/api/v1/sessions/session-1/events", nil)
		request.Header.Set("Last-Event-ID", cursor)
		response := httptest.NewRecorder()
		s.Handler().ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("cursor %s: got %d want %d", cursor, response.Code, status)
		}
	}
}

func TestCachedMediaInfoCannotBeMutatedByCaller(t *testing.T) {
	client := NewMetadataClient(nil)
	manifest := domain.Manifest{VideoID: "v", Media: &domain.MediaInfo{MIMEType: `video/mp4; codecs="avc1.64000c"`, DurationMS: 1000}}
	client.manifests["v"] = cacheEntry[domain.Manifest]{value: manifest, fetched: time.Now()}
	first, err := client.Manifest(context.Background(), "v")
	if err != nil {
		t.Fatal(err)
	}
	first.Media.DurationMS = 1
	next, err := client.Manifest(context.Background(), "v")
	if err != nil || next.Media.DurationMS != 1000 {
		t.Fatal("caller mutated cached media info")
	}
}
