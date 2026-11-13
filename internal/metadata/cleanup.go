// TODO(Yuquan): Add paginated cleanup previews for large video libraries.
package metadata

import (
	"bytes"
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

func (s *HTTPServer) verifyKeep(ctx context.Context, receipts []domain.Receipt) bool {
	for _, a := range receipts {
		found := false
		for _, p := range s.Peers {
			if p.ID == a.PeerID {
				req, _ := http.NewRequestWithContext(ctx, "GET", p.URL+"/api/v1/segments/"+a.SegmentID+"/receipt", nil)
				res, err := s.Client.Do(req)
				if err != nil {
					return false
				}
				var got domain.Receipt
				err = json.NewDecoder(res.Body).Decode(&got)
				res.Body.Close()
				if err != nil || res.StatusCode != 200 || got != a {
					return false
				}
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func (s *HTTPServer) cleanupPreview(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	state := s.Node.FSM.Copy()
	video, ok := state.Videos[r.PathValue("id")]
	if !ok || video.Status != "READY" {
		protocol.Fail(w, 404, "NOT_FOUND", "ready video not found")
		return
	}
	tasks := []domain.Task{}
	// Use all committed logical bytes, including other videos. Retired replicas
	// are already absent, so concurrent pending cleanups are not counted twice.
	projected := cleanupLoads(state.Replicas)
	segments := append([]domain.Segment(nil), video.Manifest.Segments...)
	// Handle large objects first to reduce the imbalance left by indivisible files.
	sort.Slice(segments, func(i, j int) bool {
		if segments[i].Size != segments[j].Size {
			return segments[i].Size > segments[j].Size
		}
		return segments[i].Index < segments[j].Index
	})
	for _, seg := range segments {
		pending := false
		for _, t := range state.Tasks {
			if t.SegmentID == seg.ID && t.State == "pending" {
				pending = true
			}
		}
		if pending {
			continue
		}
		replicas := state.Replicas[seg.ID]
		ids := []string{}
		for id := range replicas {
			ids = append(ids, id)
		}
		if len(ids) <= 2 {
			continue
		}
		sortCleanupTargets(ids, projected)
		healthy := map[string]bool{}
		for _, id := range ids {
			healthy[id] = s.verifyKeep(r.Context(), []domain.Receipt{replicas[id]})
		}
		// Remove from the most loaded eligible holder, then account for this plan
		// before choosing the next target. A failed holder may be retired only when
		// two other copies have actually passed verification.
		for _, id := range ids {
			keep := []domain.Receipt{}
			for i := len(ids) - 1; i >= 0 && len(keep) < 2; i-- {
				other := ids[i]
				if other != id && healthy[other] {
					keep = append(keep, replicas[other])
				}
			}
			if len(keep) != 2 {
				continue
			}
			found := false
			for _, p := range s.Peers {
				if p.ID != id {
					continue
				}
				a := replicas[id]
				raw, _ := json.Marshal([]domain.Receipt{keep[0], keep[1], a})
				tasks = append(tasks, domain.Task{ID: fmt.Sprintf("cleanup-%x", sha256.Sum256(raw)), SegmentID: seg.ID, Version: 1, Target: p, State: "pending", Size: seg.Size, SHA256: seg.SHA256, Reason: "cleanup", Keep: keep, Removed: &a})
				projected[id] -= a.Size
				found = true
				break
			}
			if found {
				break
			}
		}
	}
	protocol.JSON(w, 200, map[string]any{"tasks": tasks})
}
func (s *HTTPServer) cleanupSubmit(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	var body struct {
		Tasks []domain.Task `json:"tasks"`
	}
	if protocol.Decode(w, r, &body) != nil || len(body.Tasks) == 0 {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "empty cleanup plan")
		return
	}
	state := s.Node.FSM.Copy()
	v, ok := state.Videos[r.PathValue("id")]
	if !ok || v.Status != "READY" {
		protocol.Fail(w, 404, "NOT_FOUND", "video not found")
		return
	}
	accepted := []string{}
	for _, t := range body.Tasks {
		valid := false
		for _, seg := range v.Manifest.Segments {
			if seg.ID == t.SegmentID && seg.Size == t.Size && seg.SHA256 == t.SHA256 {
				valid = true
			}
		}
		configured := false
		for _, p := range s.Peers {
			if p == t.Target {
				configured = true
			}
		}
		if !valid || !configured || !s.verifyKeep(r.Context(), t.Keep) {
			protocol.Fail(w, 409, "PLAN_CHANGED", "plan is no longer safe; refresh preview (earlier tasks may have been accepted)")
			return
		}
		result, err := s.Node.Propose(Command{Kind: "cleanup-create", OperationID: t.ID + "-create", Task: &t})
		if err != nil || result.Code != "" {
			protocol.Fail(w, 409, "PLAN_CHANGED", "refresh preview; earlier tasks may have been accepted")
			return
		}
		accepted = append(accepted, t.ID)
	}
	protocol.JSON(w, 202, map[string]any{"accepted": accepted})
}

// A Peer authorizes against a quorum-backed task, never against a caller's claim
// to be leader. Pending tasks pin the retained replica set until deletion ends.
func (s *HTTPServer) cleanupAuthorize(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	t, ok := s.Node.FSM.Copy().Tasks[r.PathValue("id")]
	safe := ok && t.State == "pending"
	if t.Reason == "cleanup" {
		safe = safe && s.verifyKeep(r.Context(), t.Keep)
	} else if t.Reason == "video-delete" {
		safe = safe && s.Node.FSM.Copy().Videos[t.VideoID].Status == "DELETING"
	} else {
		safe = false
	}
	if !safe {
		protocol.Fail(w, 409, "NOT_SAFE", "cleanup not currently authorized")
		return
	}
	protocol.JSON(w, 200, t)
}

// RunCleanup handles manually requested work independently of automatic repair.
func (s *HTTPServer) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if s.Node.Read() != nil {
			continue
		}
		for _, t := range s.Node.FSM.Copy().Tasks {
			if (t.Reason != "cleanup" && t.Reason != "video-delete") || t.State != "pending" {
				continue
			}
			raw, _ := json.Marshal(t)
			req, _ := http.NewRequestWithContext(ctx, "POST", t.Target.URL+"/api/v1/cleanup", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			res, err := s.Client.Do(req)
			if err != nil {
				continue
			}
			res.Body.Close()
			if res.StatusCode == 200 {
				s.Node.Propose(Command{Kind: "cleanup-complete", OperationID: t.ID + "-complete", TaskID: t.ID})
			}
		}
	}
}

// Logical replica sizes are available even for offline peers. Physical usage
// can differ because of content deduplication and unindexed files.
func cleanupLoads(replicas map[string]map[string]domain.Receipt) map[string]int64 {
	loads := map[string]int64{}
	for _, holders := range replicas {
		for id, receipt := range holders {
			loads[id] += receipt.Size
		}
	}
	return loads
}
func sortCleanupTargets(ids []string, loads map[string]int64) {
	sort.Slice(ids, func(i, j int) bool {
		if loads[ids[i]] != loads[ids[j]] {
			return loads[ids[i]] > loads[ids[j]]
		}
		return ids[i] < ids[j]
	})
}
