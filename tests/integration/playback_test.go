// TODO(Tian): Add explicit failed-session, cancellation and controlled-stall metric cases to these real HTTP integration tests.
package integration

import (
	"bytes"
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/playback"
	"cooperative.video/internal/protocol"
	"cooperative.video/internal/storage"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlaybackValidatesHTTPAndFallsBack(t *testing.T) {
	data := []byte("verified bytes")
	seg := domain.Segment{ID: "v--0", Index: 0, DurationMS: 5, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	store, err := storage.Open(t.TempDir(), "peer-2", 1024)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(domain.Receipt{OperationID: "put", PeerID: "peer-2", SegmentID: seg.ID, Version: 1, Size: seg.Size, SHA256: seg.SHA256}, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	peer := httptest.NewServer(storage.NewServer(store, "peer-2", 1e6).Handler())
	defer peer.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("corrupt bytes!")) }))
	defer bad.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/videos/v/manifest", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}})
	})
	mux.HandleFunc("GET /api/v1/segments/v--0/locations", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, protocol.Locations{SegmentID: seg.ID, Locations: []domain.Location{{Peer: domain.Peer{ID: "peer-1", URL: bad.URL}, Version: 1}, {Peer: domain.Peer{ID: "peer-2", URL: peer.URL}, Version: 1}}})
	})
	meta := httptest.NewServer(mux)
	defer meta.Close()
	failed := 0
	result := playback.NewEngine([]string{meta.URL}).Run(context.Background(), "v", playback.Config{Window: 2, Concurrency: 1}, func(e playback.Event) {
		if e.Type == "request_failed" {
			failed++
		}
	})
	if result.Status != "completed" || result.SegmentsPlayed != 1 || result.VerifiedBytes != seg.Size || failed != 1 {
		t.Fatalf("unexpected result: %+v, failed=%d", result, failed)
	}
}

func TestStallAccountingAndCancellation(t *testing.T) {
	data := []byte("payload")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	segments := []domain.Segment{{ID: "v--0", Index: 0, DurationMS: 5, Size: int64(len(data)), SHA256: hash}, {ID: "v--1", Index: 1, DurationMS: 5, Size: int64(len(data)), SHA256: hash}}
	started := make(chan struct{})
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/segments/v--1" {
			select {
			case <-started:
			case <-r.Context().Done():
				return
			}
			time.Sleep(80 * time.Millisecond)
		}
		_, _ = w.Write(data)
	}))
	defer peer.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/videos/v/manifest", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: segments})
	})
	mux.HandleFunc("GET /api/v1/segments/{id}/locations", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, protocol.Locations{SegmentID: r.PathValue("id"), Locations: []domain.Location{{Peer: domain.Peer{ID: "peer-1", URL: peer.URL}, Version: 1}}})
	})
	meta := httptest.NewServer(mux)
	defer meta.Close()
	result := playback.NewEngine([]string{meta.URL}).Run(context.Background(), "v", playback.Config{Window: 2, Concurrency: 2}, func(e playback.Event) {
		if e.Type == "playback_started" {
			close(started)
		}
	})
	if result.Status != "completed" || result.StallMS < 40 || result.RebufferingRatio <= 0 || result.RebufferingRatio >= 1 {
		t.Fatalf("stall accounting incorrect: %+v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result = playback.NewEngine([]string{meta.URL}).Run(ctx, "v", playback.Config{Window: 1, Concurrency: 1}, func(e playback.Event) {
		if e.Type == "playback_started" {
			cancel()
		}
	})
	if result.Status != "stopped" || result.SegmentsPlayed != 0 {
		t.Fatalf("cancellation ignored: %+v", result)
	}
}
