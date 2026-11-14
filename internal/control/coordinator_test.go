// TODO(Yuquan): Extend coordinator tests with bandwidth stalls and retry backoff.
package control

import (
	"bytes"
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/storage"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCopyRejectsCorruptionAndLostLeadershipBeforePut(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			targetCalls++
		}
		w.WriteHeader(500)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("bad")) }))
	defer source.Close()
	task := domain.Task{ID: "t", SegmentID: "s", Version: 1, Size: 3, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("abc"))), Source: domain.Peer{URL: source.URL}, Target: domain.Peer{URL: target.URL}}
	c := Coordinator{Client: http.DefaultClient, Leader: func() error { return nil }}
	if _, err := c.copy(context.Background(), task); err == nil {
		t.Fatal("copied corrupt bytes")
	}
	task.SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte("bad")))
	c.Leader = func() error { return errors.New("no quorum") }
	if _, err := c.copy(context.Background(), task); err == nil {
		t.Fatal("copied without quorum")
	}
	if targetCalls != 0 {
		t.Fatal("unsafe PUT reached target")
	}
}

func TestLostRepairResponseIsReconciledFromVerifiedInventory(t *testing.T) {
	data := []byte("repair data")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	seg := domain.Segment{ID: "seg", Size: int64(len(data)), SHA256: hash}
	store, err := storage.Open(t.TempDir(), "target", seg.Size)
	if err != nil {
		t.Fatal(err)
	}
	peer := storage.NewServer(store, "target", 1000000).Handler()
	puts := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
			// Install the object but close the connection instead of delivering ACK.
			recorder := httptest.NewRecorder()
			peer.ServeHTTP(recorder, r)
			if recorder.Code != 200 {
				t.Error("target install failed", recorder.Body.String())
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		peer.ServeHTTP(w, r)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer source.Close()
	task := domain.Task{ID: "repair-lost-ack", State: "pending", Reason: "under-replicated", SegmentID: seg.ID, Version: 1, Size: seg.Size, SHA256: hash, Source: domain.Peer{ID: "source", URL: source.URL}, Target: domain.Peer{ID: "target", URL: target.URL}}
	state := View{Videos: map[string]domain.Video{"v": {Status: "READY", Manifest: domain.Manifest{Segments: []domain.Segment{seg}}}}, Replicas: map[string]map[string]domain.Receipt{seg.ID: {}}, Tasks: map[string]domain.Task{task.ID: task}}
	adoptions := 0
	c := Coordinator{Client: http.DefaultClient, Peers: []domain.Peer{task.Target}, Leader: func() error { return nil }, Snapshot: func() View { return state },
		Fail:     func(t domain.Task) error { t.State = "failed"; state.Tasks[t.ID] = t; return nil },
		Complete: func(domain.Task, domain.Receipt) error { t.Error("lost ACK unexpectedly completed"); return nil },
		Adopt:    func(a domain.Receipt) error { adoptions++; state.Replicas[seg.ID][a.PeerID] = a; return nil },
	}
	c.reconcile(context.Background(), map[string]time.Time{})
	if state.Tasks[task.ID].State != "failed" || puts != 1 {
		t.Fatal("did not exercise lost response", state.Tasks, puts)
	}
	// No source or free target capacity is needed to adopt already-installed bytes.
	source.Close()
	c.reconcile(context.Background(), map[string]time.Time{})
	if adoptions != 1 || state.Replicas[seg.ID]["target"].OperationID != task.ID || puts != 1 {
		t.Fatal("valid target was not recovered", state.Replicas, adoptions, puts)
	}
}

func TestPendingRepairResumesInstalledTargetWithoutSource(t *testing.T) {
	data := []byte("installed")
	a := domain.Receipt{OperationID: "repair", PeerID: "target", SegmentID: "seg", Version: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	store, err := storage.Open(t.TempDir(), "target", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Put(a, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	target := httptest.NewServer(storage.NewServer(store, "target", 1000000).Handler())
	defer target.Close()
	task := domain.Task{ID: a.OperationID, SegmentID: a.SegmentID, Version: a.Version, Size: a.Size, SHA256: a.SHA256, Source: domain.Peer{URL: "http://127.0.0.1:1"}, Target: domain.Peer{ID: a.PeerID, URL: target.URL}}
	c := Coordinator{Client: http.DefaultClient, Leader: func() error { return nil }}
	if got, err := c.copy(context.Background(), task); err != nil || got != a {
		t.Fatal("resume tried dead source", got, err)
	}
	c.Leader = func() error { return errors.New("lost quorum") }
	if _, err := c.copy(context.Background(), task); err == nil {
		t.Fatal("resumed without quorum")
	}
}

func TestAdoptionRejectsInvalidInventoryAndLostLeadership(t *testing.T) {
	for _, mode := range []string{"bad-hash", "bad-size", "bad-version", "lost-quorum"} {
		t.Run(mode, func(t *testing.T) {
			seg := domain.Segment{ID: "seg", Size: 3, SHA256: "expected"}
			a := domain.Receipt{OperationID: "put", PeerID: "target", SegmentID: seg.ID, Size: seg.Size, SHA256: seg.SHA256, Version: 1}
			switch mode {
			case "bad-hash":
				a.SHA256 = "wrong"
			case "bad-size":
				a.Size++
			case "bad-version":
				a.Version++
			}
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(domain.PeerReport{PeerID: "target", QuotaBytes: 100, Receipts: []domain.Receipt{a}})
			}))
			defer target.Close()
			checks := 0
			c := Coordinator{Client: http.DefaultClient, Peers: []domain.Peer{{ID: "target", URL: target.URL}},
				Snapshot: func() View {
					return View{Videos: map[string]domain.Video{"v": {Status: "READY", Manifest: domain.Manifest{Segments: []domain.Segment{seg}}}}}
				},
				Leader: func() error {
					checks++
					if mode == "lost-quorum" && checks > 1 {
						return errors.New("lost quorum")
					}
					return nil
				},
				Adopt: func(domain.Receipt) error { t.Error("unsafe adoption"); return nil },
			}
			c.reconcile(context.Background(), map[string]time.Time{})
		})
	}
}
