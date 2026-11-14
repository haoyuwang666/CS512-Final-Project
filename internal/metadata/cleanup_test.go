// TODO(Yuquan): Extend cleanup concurrency tests alongside future membership changes.
package metadata

import (
	"context"
	"cooperative.video/internal/domain"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeeperVerificationAcceptsEquivalentPutAndRejectsChangedObject(t *testing.T) {
	a := domain.Receipt{PeerID: "p", OperationID: "confirmed-alias", SegmentID: "seg", Version: 1, Size: 3, SHA256: "hash"}
	stored := a
	stored.OperationID = "canonical-put"
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(stored)
	}))
	defer peer.Close()
	s := HTTPServer{Client: http.DefaultClient, Peers: []domain.Peer{{ID: "p", URL: peer.URL}}}
	if !s.verifyKeep(context.Background(), []domain.Receipt{a}) {
		t.Fatal("equivalent PUT receipt prevented cleanup")
	}
	for _, field := range []string{"peer", "segment", "version", "size", "hash"} {
		bad := a
		switch field {
		case "peer":
			bad.PeerID = "other"
		case "segment":
			bad.SegmentID = "other"
		case "version":
			bad.Version++
		case "size":
			bad.Size++
		case "hash":
			bad.SHA256 = "different"
		}
		if s.verifyKeep(context.Background(), []domain.Receipt{bad}) {
			t.Fatal("accepted changed keeper", field)
		}
	}
}

func TestCleanupPinsKeepersAndRejectsStaleReceipts(t *testing.T) {
	f := NewFSM()
	replicas := map[string]domain.Receipt{}
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		replicas[id] = domain.Receipt{PeerID: id, OperationID: "put-" + id, SegmentID: "seg", SHA256: "hash", Size: 3, Version: 1}
	}
	f.state.Replicas["seg"] = replicas
	removed := replicas["p3"]
	task := domain.Task{ID: "cleanup-one", SegmentID: "seg", Target: domain.Peer{ID: "p3"}, State: "pending", Reason: "cleanup", Keep: []domain.Receipt{replicas["p1"], replicas["p2"]}, Removed: &removed}
	if r := applyTest(t, f, Command{Kind: "cleanup-create", OperationID: "start", Task: &task}); r.Code != "" {
		t.Fatal(r)
	}
	second := task
	second.ID = "cleanup-two"
	second.Target.ID = "p4"
	other := replicas["p4"]
	second.Removed = &other
	if r := applyTest(t, f, Command{Kind: "cleanup-create", OperationID: "conflict", Task: &second}); r.Code == "" {
		t.Fatal("overlapping cleanup accepted")
	}
	if r := applyTest(t, f, Command{Kind: "task-fail", OperationID: "cancel", TaskID: task.ID}); r.Code == "" {
		t.Fatal("unsafe cancellation accepted")
	}
	if r := applyTest(t, f, Command{Kind: "confirm", OperationID: "stale", Receipt: removed}); r.Code == "" {
		t.Fatal("stale receipt accepted")
	}
	if r := applyTest(t, f, Command{Kind: "cleanup-complete", OperationID: "done", TaskID: task.ID}); r.Code != "" {
		t.Fatal(r)
	}
	if r := applyTest(t, f, Command{Kind: "confirm", OperationID: "stale-after", Receipt: removed}); r.Code == "" {
		t.Fatal("deleted receipt resurrected")
	}
}

func TestCleanupBalancesProjectedBytesAcrossSegments(t *testing.T) {
	loads := map[string]int64{"p1": 600, "p2": 600, "p3": 600}
	for i := 0; i < 6; i++ {
		ids := []string{"p3", "p1", "p2"}
		sortCleanupTargets(ids, loads)
		loads[ids[0]] -= 100
	}
	for id, load := range loads {
		if load != 400 {
			t.Fatalf("unbalanced %s: %v", id, loads)
		}
	}
	// Byte load, rather than ID or object count, determines the target.
	loads = cleanupLoads(map[string]map[string]domain.Receipt{
		"other-video":   {"p3": {Size: 900}},
		"current-video": {"p1": {Size: 100}, "p2": {Size: 100}, "p3": {Size: 100}},
	})
	ids := []string{"p1", "p2", "p3"}
	sortCleanupTargets(ids, loads)
	if ids[0] != "p3" {
		t.Fatal("ignored bytes belonging to another video", ids)
	}
}
