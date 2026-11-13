// TODO(Haoyu): Extend video deletion tests when replica generations are introduced.
package metadata

import (
	"cooperative.video/internal/domain"
	"testing"
)

func TestVideoDeletionBlocksLateConfirmAndWaitsForAllPeers(t *testing.T) {
	f := NewFSM()
	a := domain.Receipt{OperationID: "put-a", PeerID: "p1", SegmentID: "s", Size: 3, SHA256: "hash", Version: 1}
	b := a
	b.PeerID = "p2"
	b.OperationID = "put-b"
	f.state.Videos["v"] = domain.Video{Manifest: domain.Manifest{VideoID: "v", Segments: []domain.Segment{{ID: "s", Size: 3, SHA256: "hash"}}}, Status: "READY", Version: 2}
	f.state.Replicas["s"] = map[string]domain.Receipt{"p1": a, "p2": b}
	task := func(id string, a domain.Receipt) domain.Task {
		return domain.Task{ID: id, VideoID: "v", SegmentID: "s", State: "pending", Reason: "video-delete", Target: domain.Peer{ID: a.PeerID}, Removed: &a}
	}
	tasks := []domain.Task{task("delete-a", a), task("delete-b", b)}
	if r := applyTest(t, f, Command{Kind: "video-delete", OperationID: "missing", VideoID: "v", Tasks: tasks[:1]}); r.Code == "" {
		t.Fatal("partial plan accepted")
	}
	if f.state.Videos["v"].Status != "READY" {
		t.Fatal("rejected plan changed state")
	}
	if r := applyTest(t, f, Command{Kind: "video-delete", OperationID: "delete", VideoID: "v", Tasks: tasks}); r.Code != "" {
		t.Fatal(r)
	}
	if len(f.state.Replicas["s"]) != 0 {
		t.Fatal("deleted locations published")
	}
	if r := applyTest(t, f, Command{Kind: "confirm", OperationID: "late", Receipt: a}); r.Code == "" {
		t.Fatal("late confirmation accepted")
	}
	applyTest(t, f, Command{Kind: "cleanup-complete", OperationID: "done-a", TaskID: "delete-a"})
	if f.state.Videos["v"].Status != "DELETING" {
		t.Fatal("completed before both peers")
	}
	applyTest(t, f, Command{Kind: "cleanup-complete", OperationID: "done-b", TaskID: "delete-b"})
	if f.state.Videos["v"].Status != "DELETED" {
		t.Fatal("not completed")
	}
	if r := applyTest(t, f, Command{Kind: "publish", OperationID: "republish", VideoID: "v", ExpectedVersion: f.state.Videos["v"].Version}); r.Code == "" {
		t.Fatal("deleted video republished")
	}
}
