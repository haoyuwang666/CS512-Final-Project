// TODO(Yuquan): Add generation-based cleanup retention before pruning tombstones.
package storage

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) cleanup(w http.ResponseWriter, r *http.Request) {
	var t domain.Task
	if protocol.Decode(w, r, &t) != nil || !domain.ValidID(t.ID) || t.Target.ID != s.PeerID || t.Removed == nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid cleanup")
		return
	}
	// Only configured Controllers can authorize destructive operations.
	authorized := false
	client := &http.Client{Timeout: 10 * time.Second}
	for _, base := range strings.Split(os.Getenv("CONTROLLERS"), ",") {
		if strings.TrimSpace(base) == "" {
			continue
		}
		req, err := http.NewRequestWithContext(r.Context(), "GET", strings.TrimRight(base, "/")+"/api/v1/cleanup/"+t.ID+"/authorize", nil)
		if err != nil {
			continue
		}
		res, err := client.Do(req)
		if err != nil {
			continue
		}
		var got domain.Task
		err = json.NewDecoder(res.Body).Decode(&got)
		res.Body.Close()
		a, _ := json.Marshal(t)
		b, _ := json.Marshal(got)
		if err == nil && res.StatusCode == 200 && string(a) == string(b) {
			authorized = true
			break
		}
	}
	if !authorized {
		protocol.Fail(w, 409, "NOT_SAFE", "cleanup authorization unavailable")
		return
	}
	if err := s.Store.DeleteReplica(t.ID, *t.Removed); err != nil {
		protocol.Fail(w, 409, "DELETE_FAILED", err.Error())
		return
	}
	protocol.JSON(w, 200, map[string]bool{"ok": true})
}
func (s *Store) DeleteReplica(task string, a domain.Receipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !domain.ValidID(task) || a.PeerID != s.peer {
		return ErrInvalid
	}
	if old, ok := s.index.Deleted[task]; ok {
		if old != a {
			return ErrConflict
		}
	} else {
		if current, ok := s.index.Objects[a.SegmentID]; !ok || current != a {
			return ErrConflict
		}
		raw, _ := json.Marshal(s.index)
		var next diskIndex
		if err := json.Unmarshal(raw, &next); err != nil {
			return err
		}
		if next.Deleted == nil {
			next.Deleted = map[string]domain.Receipt{}
		}
		next.Deleted[task] = a
		delete(next.Objects, a.SegmentID)
		raw, _ = json.Marshal(next)
		// Persist the tombstone first. A crash can leak space, never revive a replica.
		if err := atomicWrite(s.dir, "index.json", raw); err != nil {
			return err
		}
		s.index = next
	}
	for _, other := range s.index.Objects {
		if other.SHA256 == a.SHA256 {
			return nil
		}
	}
	path := filepath.Join(s.dir, a.SHA256+".blob")
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return syncDir(s.dir)
	}
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil {
		return fmt.Errorf("remove blob: %w", err)
	}
	s.used -= info.Size()
	return syncDir(s.dir)
}
