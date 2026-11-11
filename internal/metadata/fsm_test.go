// TODO(Haoyu): Extend safety tests to durable task transitions and concurrent removals when those features land.
package metadata

import (
	"bytes"
	"cooperative.video/internal/domain"
	"encoding/json"
	"github.com/hashicorp/raft"
	"io"
	"strings"
	"testing"
)

func applyTest(t *testing.T, f *FSM, c Command) Result {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return f.Apply(&raft.Log{Data: b}).(Result)
}
func TestPublishRequiresDistinctReplicasAndIdempotency(t *testing.T) {
	f := NewFSM()
	seg := domain.Segment{ID: "v--0", Index: 0, DurationMS: 100, Size: 3, SHA256: strings.Repeat("a", 64)}
	m := domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}}
	if r := applyTest(t, f, Command{Kind: "create", OperationID: "create", Manifest: m}); r.Code != "" {
		t.Fatal(r)
	}
	if r := applyTest(t, f, Command{Kind: "publish", OperationID: "too-early", VideoID: "v", ExpectedVersion: 1}); r.Code != "NOT_READY" {
		t.Fatal(r)
	}
	a := domain.Receipt{OperationID: "put-1", PeerID: "peer-1", SegmentID: seg.ID, Version: 1, Size: seg.Size, SHA256: seg.SHA256}
	for _, op := range []string{"ack-1", "ack-again"} {
		if r := applyTest(t, f, Command{Kind: "confirm", OperationID: op, Receipt: a}); r.Code != "" {
			t.Fatal(r)
		}
	}
	if r := applyTest(t, f, Command{Kind: "publish", OperationID: "one-peer", VideoID: "v", ExpectedVersion: 1}); r.Code != "NOT_READY" {
		t.Fatal(r)
	}
	a.PeerID = "peer-2"
	a.OperationID = "put-2"
	applyTest(t, f, Command{Kind: "confirm", OperationID: "ack-2", Receipt: a})
	c := Command{Kind: "publish", OperationID: "ready", VideoID: "v", ExpectedVersion: 1}
	if r := applyTest(t, f, c); r.Code != "" {
		t.Fatal(r)
	}
	if r := applyTest(t, f, c); r.Code != "" {
		t.Fatal("retry must return original success", r)
	}
	c.ExpectedVersion = 2
	if r := applyTest(t, f, c); r.Code != "VERSION_CONFLICT" {
		t.Fatal(r)
	}
	state := f.Copy()
	if state.Videos["v"].Status != "READY" || state.Videos["v"].Version != 2 {
		t.Fatal(state)
	}
	b, _ := json.Marshal(state)
	restored := NewFSM()
	if err := restored.Restore(io.NopCloser(bytes.NewReader(b))); err != nil {
		t.Fatal(err)
	}
	if restored.Copy().Videos["v"].Status != "READY" {
		t.Fatal("snapshot lost published video")
	}
}
