// TODO(Haoyu): Add bounded retention for deleted video manifests and task history.
package metadata

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sort"
)

func (s *HTTPServer) deleteVideo(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	id := r.PathValue("id")
	state := s.Node.FSM.Copy()
	v, ok := state.Videos[id]
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "video not found")
		return
	}
	if v.Status == "DELETING" || v.Status == "DELETED" {
		s.deleteVideoStatus(w, r)
		return
	}
	tasks := []domain.Task{}
	for _, seg := range v.Manifest.Segments {
		for peer, a := range state.Replicas[seg.ID] {
			found := false
			for _, p := range s.Peers {
				if p.ID == peer {
					found = true
					receipt := a
					tasks = append(tasks, domain.Task{ID: fmt.Sprintf("delete-%x", sha256.Sum256([]byte(id+"/"+seg.ID+"/"+peer))), VideoID: id, SegmentID: seg.ID, Target: p, State: "pending", Reason: "video-delete", Size: seg.Size, SHA256: seg.SHA256, Version: 1, Removed: &receipt})
				}
			}
			if !found {
				protocol.Fail(w, 409, "UNKNOWN_PEER", "restore original peer configuration before deletion")
				return
			}
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	// Fresh operation IDs allow a rejected request to be retried after repair ends.
	s.apply(w, Command{Kind: "video-delete", OperationID: fmt.Sprintf("delete-%s-%d", id, len(state.Operations)), VideoID: id, Tasks: tasks})
}
func (s *HTTPServer) deleteVideoStatus(w http.ResponseWriter, r *http.Request) {
	if !s.readable(w) {
		return
	}
	state := s.Node.FSM.Copy()
	v, ok := state.Videos[r.PathValue("id")]
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "video not found")
		return
	}
	total, completed := 0, 0
	for _, t := range state.Tasks {
		if t.VideoID == v.Manifest.VideoID && t.Reason == "video-delete" {
			total++
			if t.State == "completed" {
				completed++
			}
		}
	}
	protocol.JSON(w, 200, map[string]any{"state": v.Status, "total": total, "completed": completed})
}
