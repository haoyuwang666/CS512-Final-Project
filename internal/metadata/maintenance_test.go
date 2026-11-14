// TODO(Yuquan): Extend durable task tests alongside new lifecycle transitions.
package metadata

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

func TestRepairTaskAtomicCompletionAndSnapshotReplay(t *testing.T) {
	f := NewFSM()
	seg := domain.Segment{ID: "s", DurationMS: 10, Size: 3, SHA256: strings.Repeat("a", 64)}
	applyTest(t, f, Command{Kind: "create", OperationID: "create", Manifest: domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}}})
	for _, id := range []string{"p1", "p2"} {
		applyTest(t, f, Command{Kind: "confirm", OperationID: id, Receipt: domain.Receipt{PeerID: id, OperationID: "put-" + id, SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}})
	}
	applyTest(t, f, Command{Kind: "publish", OperationID: "pub", VideoID: "v", ExpectedVersion: 1})
	task := domain.Task{CreatedUnixMS: 123456789, ID: "task", SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256, Source: domain.Peer{ID: "p1"}, Target: domain.Peer{ID: "p3"}, State: "pending"}
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
	if restored.Copy().Tasks["task"].State != "pending" || restored.Copy().Tasks["task"].CreatedUnixMS != task.CreatedUnixMS {
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

func TestTaskAckAcceptsPutAliasWithoutPoisoningCompletion(t *testing.T) {
	f := NewFSM()
	seg := domain.Segment{ID: "s", DurationMS: 10, Size: 3, SHA256: strings.Repeat("a", 64)}
	applyTest(t, f, Command{Kind: "create", OperationID: "create", Manifest: domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "test", Segments: []domain.Segment{seg}}})
	for _, id := range []string{"p1", "p2"} {
		applyTest(t, f, Command{Kind: "confirm", OperationID: id, Receipt: domain.Receipt{PeerID: id, OperationID: "put-" + id, SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}})
	}
	applyTest(t, f, Command{Kind: "publish", OperationID: "publish", VideoID: "v", ExpectedVersion: 1})
	canonical := domain.Receipt{OperationID: "earlier-put", PeerID: "p3", SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(canonical) }))
	defer peer.Close()
	task := domain.Task{ID: "repair-alias", State: "pending", Source: domain.Peer{ID: "p1"}, Target: domain.Peer{ID: "p3", URL: peer.URL}, SegmentID: "s", Version: 1, Size: 3, SHA256: seg.SHA256}
	applyTest(t, f, Command{Kind: "task-create", OperationID: "repair-create", Task: &task})
	config := raft.DefaultConfig()
	config.LocalID = "test"
	store := raft.NewInmemStore()
	addr, transport := raft.NewInmemTransport("test")
	defer transport.Close()
	r, err := raft.NewRaft(config, f, store, store, raft.NewInmemSnapshotStore(), transport)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Shutdown().Error() }()
	if err = r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{ID: "test", Address: addr, Suffrage: raft.Voter}}}).Error(); err != nil {
		t.Fatal(err)
	}
	node := &Node{Raft: r, FSM: f}
	deadline := time.Now().Add(5 * time.Second)
	for node.Read() != nil {
		if time.Now().After(deadline) {
			t.Fatal("in-memory leader unavailable")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s := HTTPServer{Node: node, Client: http.DefaultClient}
	ack := func(a domain.Receipt) int {
		raw, _ := json.Marshal(map[string]any{"operation_id": "caller", "receipt": a})
		req := httptest.NewRequest("POST", "/api/v1/tasks/repair-alias/ack", bytes.NewReader(raw))
		req.SetPathValue("id", task.ID)
		response := httptest.NewRecorder()
		s.taskAck(response, req)
		return response.Code
	}
	// A canonical receipt for an earlier upload proves bytes exist, but is not
	// this task's ACK. Reject it without recording a failed completion command.
	if status := ack(canonical); status != 409 {
		t.Fatal(status)
	}
	if _, poisoned := f.Copy().Operations[task.ID+"-complete"]; poisoned {
		t.Fatal("bad ACK poisoned completion key")
	}
	alias := canonical
	alias.OperationID = task.ID
	for i := 0; i < 2; i++ {
		if status := ack(alias); status != 200 {
			t.Fatal("matching alias ACK rejected", status)
		}
	}
	if f.Copy().Tasks[task.ID].State != "completed" {
		t.Fatal("repair not completed")
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
