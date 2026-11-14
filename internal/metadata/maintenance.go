// TODO(Yuquan): Add report pagination and bounded task retention for larger deployments.
package metadata

import (
	"context"
	"cooperative.video/internal/control"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type reportStore struct {
	sync.Mutex
	reports map[string]reportedPeer
}
type reportedPeer struct {
	Report domain.PeerReport `json:"report"`
	Seen   time.Time         `json:"seen_at"`
}

func (s *HTTPServer) RunMaintenance(ctx context.Context, grace time.Duration) {
	commit := func(c Command) error {
		r, e := s.Node.Propose(c)
		if e != nil {
			return e
		}
		if r.Code != "" {
			return fmt.Errorf("%s: %s", r.Code, r.Message)
		}
		return nil
	}
	c := control.Coordinator{Peers: s.Peers, Grace: grace, Leader: s.Node.Read,
		Snapshot: func() control.View {
			st := s.Node.FSM.Copy()
			return control.View{Videos: st.Videos, Replicas: st.Replicas, Tasks: st.Tasks}
		},
		Create: func(t domain.Task) error {
			return commit(Command{Kind: "task-create", OperationID: t.ID + "-create", Task: &t})
		},
		Complete: func(t domain.Task, a domain.Receipt) error {
			return commit(Command{Kind: "task-complete", OperationID: t.ID + "-complete", TaskID: t.ID, Receipt: a})
		},
		Fail: func(t domain.Task) error {
			return commit(Command{Kind: "task-fail", OperationID: t.ID + "-fail", TaskID: t.ID})
		},
		Adopt: func(a domain.Receipt) error {
			raw, _ := json.Marshal(a)
			return commit(Command{Kind: "confirm", OperationID: fmt.Sprintf("adopt-%x", sha256.Sum256(raw)), Receipt: a})
		},
	}
	c.Run(ctx)
}
func (s *HTTPServer) peerReport(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	var body domain.PeerReport
	if e := protocol.Decode(w, r, &body); e != nil || body.PeerID != r.PathValue("id") || body.InstanceID == "" || body.StoredBytes < 0 || body.QuotaBytes <= 0 {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid peer report")
		return
	}
	configured := false
	for _, p := range s.Peers {
		if p.ID == body.PeerID {
			configured = true
		}
	}
	if !configured {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "unknown peer")
		return
	}
	// Advisory only: maintenance separately fetches inventory from configured URLs.
	s.Reports.Lock()
	s.Reports.reports[body.PeerID] = reportedPeer{body, time.Now().UTC()}
	s.Reports.Unlock()
	protocol.JSON(w, 200, map[string]bool{"ok": true})
}
func (s *HTTPServer) maintenanceStatus(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	s.Reports.Lock()
	defer s.Reports.Unlock()
	protocol.JSON(w, 200, map[string]any{"peers": s.Reports.reports, "tasks": s.Node.FSM.Copy().Tasks})
}
func (s *HTTPServer) peerTasks(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	tasks := []domain.Task{}
	for _, t := range s.Node.FSM.Copy().Tasks {
		if t.Target.ID == r.PathValue("id") && t.State == "pending" {
			tasks = append(tasks, t)
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	protocol.JSON(w, 200, tasks)
}
func (s *HTTPServer) taskAck(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	var body protocol.ConfirmReplica
	if err := protocol.Decode(w, r, &body); err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", err.Error())
		return
	}
	t, ok := s.Node.FSM.Copy().Tasks[r.PathValue("id")]
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "task not found")
		return
	}
	// Reject malformed ACKs before using the stable completion operation ID.
	// Otherwise a rejected command could permanently poison that retry key.
	want := domain.Receipt{OperationID: t.ID, PeerID: t.Target.ID, SegmentID: t.SegmentID, Version: t.Version, Size: t.Size, SHA256: t.SHA256}
	if t.Reason == "cleanup" || t.Reason == "video-delete" || (t.State != "pending" && t.State != "completed") || body.Receipt != want {
		protocol.Fail(w, 409, "VERSION_CONFLICT", "receipt does not match repair task")
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", t.Target.URL+"/api/v1/segments/"+t.SegmentID+"/receipt", nil)
	res, err := s.Client.Do(req)
	if err != nil {
		protocol.Fail(w, 503, "PEER_UNAVAILABLE", err.Error())
		return
	}
	defer res.Body.Close()
	var a domain.Receipt
	if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&a) != nil || !a.SameObject(body.Receipt) {
		protocol.Fail(w, 409, "VERSION_CONFLICT", "target receipt mismatch")
		return
	}
	s.apply(w, Command{Kind: "task-complete", OperationID: t.ID + "-complete", TaskID: t.ID, Receipt: body.Receipt})
}
