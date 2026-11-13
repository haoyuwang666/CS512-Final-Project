// TODO(Yuquan): Replace advisory ingestion plans with global persistent capacity reservations.
package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"cooperative.video/internal/domain"
	"cooperative.video/internal/placement"
	"cooperative.video/internal/protocol"
)

type HTTPServer struct {
	Node              *Node
	Peers             []domain.Peer
	Client            *http.Client
	Placement         placement.Policy
	Reports           *reportStore
	ExtraReplicaBytes int64
}

func (s *HTTPServer) Handler() http.Handler {
	s.Reports = &reportStore{reports: map[string]reportedPeer{}}
	if s.Placement == nil {
		s.Placement = placement.Static{}
	}
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 3 * time.Second}
	}
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/health", protocol.Health)
	m.HandleFunc("POST /api/v1/videos/{id}/delete", s.deleteVideo)
	m.HandleFunc("GET /api/v1/videos/{id}/delete", s.deleteVideoStatus)
	m.HandleFunc("GET /api/v1/videos/{id}/cleanup", s.cleanupPreview)
	m.HandleFunc("POST /api/v1/videos/{id}/cleanup", s.cleanupSubmit)
	m.HandleFunc("GET /api/v1/cleanup/{id}/authorize", s.cleanupAuthorize)
	m.HandleFunc("GET /api/v1/cluster", func(w http.ResponseWriter, r *http.Request) {
		protocol.JSON(w, 200, map[string]any{"leader_url": s.Node.LeaderURL(), "state": s.Node.Raft.State().String(), "members": s.Node.Members, "stats": s.Node.Raft.Stats(), "peers": s.Peers})
	})
	m.HandleFunc("POST /api/v1/videos", s.create)
	m.HandleFunc("POST /api/v1/videos/{id}/publish", s.publish)
	m.HandleFunc("GET /api/v1/videos/{id}/manifest", s.manifest)
	m.HandleFunc("POST /api/v1/videos/{id}/placement-plan", s.plan)
	m.HandleFunc("POST /api/v1/segments/{id}/replicas", s.confirm)
	m.HandleFunc("GET /api/v1/segments/{id}/locations", s.locations)
	m.HandleFunc("POST /api/v1/peers/{id}/heartbeat", s.peerReport)
	m.HandleFunc("POST /api/v1/peers/{id}/inventory", s.peerReport)
	m.HandleFunc("GET /api/v1/peers/{id}/tasks", s.peerTasks)
	m.HandleFunc("POST /api/v1/tasks/{id}/ack", s.taskAck)
	m.HandleFunc("GET /api/v1/maintenance", s.maintenanceStatus)
	return m
}
func (s *HTTPServer) readable(w http.ResponseWriter) bool {
	if err := s.Node.Read(); err != nil {
		protocol.JSON(w, 503, map[string]any{"error": protocol.APIError{Code: "NOT_LEADER", Message: err.Error(), Retryable: true, LeaderURL: s.Node.LeaderURL()}})
		return false
	}
	return true
}
func (s *HTTPServer) apply(w http.ResponseWriter, c Command) {
	if !domain.ValidID(c.OperationID) {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid operation_id")
		return
	}
	result, err := s.Node.Propose(c)
	if err != nil {
		protocol.JSON(w, 503, map[string]any{"error": protocol.APIError{Code: "UNKNOWN_OUTCOME", Message: err.Error(), Retryable: true, LeaderURL: s.Node.LeaderURL()}})
		return
	}
	if result.Code != "" {
		status := 409
		if result.Code == "INVALID_ARGUMENT" {
			status = 400
		}
		if result.Code == "NOT_FOUND" {
			status = 404
		}
		protocol.Fail(w, status, result.Code, result.Message)
		return
	}
	protocol.JSON(w, 200, map[string]bool{"ok": true})
}
func (s *HTTPServer) create(w http.ResponseWriter, r *http.Request) {
	var body protocol.CreateVideo
	if err := protocol.Decode(w, r, &body); err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", err.Error())
		return
	}
	s.apply(w, Command{Kind: "create", OperationID: body.OperationID, Manifest: body.Manifest})
}
func (s *HTTPServer) publish(w http.ResponseWriter, r *http.Request) {
	var body protocol.PublishVideo
	if err := protocol.Decode(w, r, &body); err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", err.Error())
		return
	}
	s.apply(w, Command{Kind: "publish", OperationID: body.OperationID, VideoID: r.PathValue("id"), ExpectedVersion: body.ExpectedVersion})
}
func (s *HTTPServer) manifest(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	v, ok := s.Node.FSM.Copy().Videos[r.PathValue("id")]
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "video not found")
		return
	}
	if v.Status != "READY" {
		protocol.Fail(w, 409, "NOT_READY", "video is not ready")
		return
	}
	protocol.JSON(w, 200, v.Manifest)
}
func (s *HTTPServer) locations(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	state := s.Node.FSM.Copy()
	id := r.PathValue("id")
	ready := false
	for _, v := range state.Videos {
		if v.Status == "READY" {
			for _, seg := range v.Manifest.Segments {
				if seg.ID == id {
					ready = true
				}
			}
		}
	}
	if !ready {
		protocol.Fail(w, 404, "NOT_FOUND", "segment not published")
		return
	}
	out := protocol.Locations{SegmentID: id, Locations: []domain.Location{}}
	for _, p := range s.Peers {
		if a, ok := state.Replicas[id][p.ID]; ok {
			out.Locations = append(out.Locations, domain.Location{Peer: p, Version: a.Version})
		}
	}
	sort.Slice(out.Locations, func(i, j int) bool { return out.Locations[i].ID < out.Locations[j].ID })
	protocol.JSON(w, 200, out)
}
func (s *HTTPServer) confirm(w http.ResponseWriter, r *http.Request) {
	var body protocol.ConfirmReplica
	if err := protocol.Decode(w, r, &body); err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", err.Error())
		return
	}
	if !s.readable(w) {
		return
	}
	a := body.Receipt
	if a.SegmentID != r.PathValue("id") || !domain.ValidID(a.SegmentID) {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "segment mismatch")
		return
	}
	var peerURL string
	for _, p := range s.Peers {
		if p.ID == a.PeerID {
			peerURL = p.URL
		}
	}
	if peerURL == "" {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "peer is not configured")
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", peerURL+"/api/v1/segments/"+a.SegmentID+"/receipt", nil)
	res, err := s.Client.Do(req)
	if err != nil {
		protocol.Fail(w, 503, "PEER_UNAVAILABLE", err.Error())
		return
	}
	defer res.Body.Close()
	var stored domain.Receipt
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&stored) != nil || stored.SegmentID != a.SegmentID || stored.PeerID != a.PeerID || stored.Version != a.Version || stored.Size != a.Size || stored.SHA256 != a.SHA256 {
		protocol.Fail(w, 409, "VERSION_CONFLICT", "peer did not confirm matching durable object")
		return
	}
	s.apply(w, Command{Kind: "confirm", OperationID: body.OperationID, Receipt: a})
}
func (s *HTTPServer) plan(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	v, ok := s.Node.FSM.Copy().Videos[r.PathValue("id")]
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "video not found")
		return
	}
	observations := []domain.PeerObservation{}
	for _, p := range s.Peers {
		req, _ := http.NewRequestWithContext(r.Context(), "GET", p.URL+"/api/v1/stats", nil)
		res, err := s.Client.Do(req)
		if err != nil {
			continue
		}
		var stats struct {
			Used  int64   `json:"stored_bytes"`
			Quota int64   `json:"quota_bytes"`
			Rate  float64 `json:"upload_bytes_per_second"`
		}
		err = json.NewDecoder(res.Body).Decode(&stats)
		res.Body.Close()
		if err == nil && res.StatusCode == 200 {
			observations = append(observations, domain.PeerObservation{Peer: p, FreeBytes: stats.Quota - stats.Used, UploadBytesPerSecond: stats.Rate, OnlineFraction: 1})
		}
	}
	plans, extraUsed, err := placement.Plan(v.Manifest.Segments, observations, s.Placement, s.ExtraReplicaBytes)
	if err != nil {
		protocol.Fail(w, 507, "CAPACITY_EXCEEDED", fmt.Sprint(err))
		return
	}
	protocol.JSON(w, 200, map[string]any{"advisory": true, "placements": plans, "extra_replica_bytes": extraUsed})
}
