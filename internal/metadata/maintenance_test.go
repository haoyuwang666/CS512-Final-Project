// TODO(Yuquan): Extend durable task tests alongside new lifecycle transitions.
package metadata

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestRepairTaskAtomicCompletionAndSnapshotReplay(t *testing.T) {
	f := NewFSM()
	seg := domain.Segment{ID: "s", DurationMS: 10, Size: 3, SHA256: strings.Repeat("a", 64)}
	applyTest(t, f, Command{Kind: "create", OperationID: "create", Manifest: domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}}})
	for _, id := range []string{"p1", "p2"} {
		applyTest(t, f, Command{Kind: "confirm", OperationID: id, Receipt: domain.Receipt{PeerID: id, OperationID: "put-" + id, SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}})
	}
	applyTest(t, f, Command{Kind: "publish", OperationID: "pub", VideoID: "v", ExpectedVersion: 1})
	task := domain.Task{ID: "task", SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256, Source: domain.Peer{ID: "p1"}, Target: domain.Peer{ID: "p3"}, State: "pending"}
	create := Command{Kind: "task-create", OperationID: "task-create", Task: &task}
	if r := applyTest(t, f, create); r.Code != "" {
		t.Fatal(r)
	}
	other := task
	other.ID = "other"
	if r := applyTest(t, f, Command{Kind: "task-create", OperationID: "other-create", Task: &other}); r.Code != "VERSION_CONFLICT" {
		t.Fatal(r)
	}
	raw, _ := json.Marshal(f.Copy())
	restored := NewFSM()
	if err := restored.Restore(io.NopCloser(bytes.NewReader(raw))); err != nil {
		t.Fatal(err)
	}
	if restored.Copy().Tasks["task"].State != "pending" {
		t.Fatal("lost pending work")
	}
	a := domain.Receipt{OperationID: "task", PeerID: "p3", SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}
	wrong := a
	wrong.PeerID = "attacker"
	if r := applyTest(t, restored, Command{Kind: "task-complete", OperationID: "wrong", TaskID: "task", Receipt: wrong}); r.Code == "" {
		t.Fatal("accepted wrong target")
	}
	complete := Command{Kind: "task-complete", OperationID: "done", TaskID: "task", Receipt: a}
	for i := 0; i < 2; i++ {
		if r := applyTest(t, restored, complete); r.Code != "" {
			t.Fatal(r)
		}
	}
	state := restored.Copy()
	if state.Tasks["task"].State != "completed" || len(state.Replicas["s"]) != 3 {
		t.Fatal(state)
	}
}

func TestLegacyOperationDigestSurvivesTaskSchemaUpgrade(t *testing.T) {
	// Command's optional task must not change serialized pre-upgrade commands.
	type legacyCommand struct {
		Kind            string          `json:"kind"`
		OperationID     string          `json:"operation_id"`
		Manifest        domain.Manifest `json:"manifest,omitempty"`
		Receipt         domain.Receipt  `json:"receipt,omitempty"`
		VideoID         string          `json:"video_id,omitempty"`
		ExpectedVersion int             `json:"expected_version,omitempty"`
	}
	old := legacyCommand{Kind: "publish", OperationID: "legacy", VideoID: "video", ExpectedVersion: 1}
	raw, _ := json.Marshal(old)
	var current Command
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatal(err)
	}
	f := NewFSM()
	f.state.Operations["legacy"] = Operation{Digest: fmt.Sprintf("%x", sha256.Sum256(raw)), Result: Result{Code: "NOT_READY", Message: "original result"}}
	if r := applyTest(t, f, current); r.Code != "NOT_READY" || r.Message != "original result" {
		t.Fatal("upgrade broke retry identity", r)
	}
}
