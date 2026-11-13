// TODO(Yuquan): Add global capacity reservations and bounded task retention; never perform network I/O in Apply.
package metadata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"cooperative.video/internal/domain"
	"github.com/hashicorp/raft"
)

type Command struct {
	Tasks           []domain.Task   `json:"tasks,omitempty"`
	Kind            string          `json:"kind"`
	OperationID     string          `json:"operation_id"`
	Manifest        domain.Manifest `json:"manifest,omitempty"`
	Receipt         domain.Receipt  `json:"receipt,omitempty"`
	VideoID         string          `json:"video_id,omitempty"`
	ExpectedVersion int             `json:"expected_version,omitempty"`
	Task            *domain.Task    `json:"task,omitempty"`
	TaskID          string          `json:"task_id,omitempty"`
}
type Result struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
type Operation struct {
	Digest string `json:"digest"`
	Result Result `json:"result"`
}
type State struct {
	Videos     map[string]domain.Video              `json:"videos"`
	Replicas   map[string]map[string]domain.Receipt `json:"replicas"`
	Operations map[string]Operation                 `json:"operations"`
	Tasks      map[string]domain.Task               `json:"tasks"`
}
type FSM struct {
	mu    sync.RWMutex
	state State
}

func NewFSM() *FSM {
	return &FSM{state: State{Videos: map[string]domain.Video{}, Replicas: map[string]map[string]domain.Receipt{}, Operations: map[string]Operation{}, Tasks: map[string]domain.Task{}}}
}
func (f *FSM) Apply(log *raft.Log) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var c Command
	if err := json.Unmarshal(log.Data, &c); err != nil {
		return Result{"INVALID_ARGUMENT", err.Error()}
	}
	if !domain.ValidID(c.OperationID) {
		return Result{"INVALID_ARGUMENT", "invalid operation_id"}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(log.Data))
	if op, ok := f.state.Operations[c.OperationID]; ok {
		if op.Digest != digest {
			return Result{"VERSION_CONFLICT", "operation_id reused with different content"}
		}
		return op.Result
	}
	result := f.applyCommand(c)
	f.state.Operations[c.OperationID] = Operation{digest, result}
	return result
}
func (f *FSM) applyCommand(c Command) Result {
	if f.state.Tasks == nil {
		f.state.Tasks = map[string]domain.Task{}
	}
	switch c.Kind {
	case "video-delete":
		v, ok := f.state.Videos[c.VideoID]
		if !ok {
			return Result{"NOT_FOUND", "video not found"}
		}
		if v.Status == "DELETING" || v.Status == "DELETED" {
			return Result{}
		}
		segments := map[string]bool{}
		count := 0
		for _, seg := range v.Manifest.Segments {
			segments[seg.ID] = true
			count += len(f.state.Replicas[seg.ID])
		}
		for _, t := range f.state.Tasks {
			if segments[t.SegmentID] && t.State == "pending" {
				return Result{"VERSION_CONFLICT", "wait for pending replica tasks before deleting"}
			}
		}
		if len(c.Tasks) != count {
			return Result{"VERSION_CONFLICT", "replica set changed"}
		}
		seen := map[string]bool{}
		for _, t := range c.Tasks {
			key := t.SegmentID + "/" + t.Target.ID
			if !domain.ValidID(t.ID) || seen[key] || !segments[t.SegmentID] || t.VideoID != c.VideoID || t.Reason != "video-delete" || t.State != "pending" || t.Removed == nil || f.state.Replicas[t.SegmentID][t.Target.ID] != *t.Removed {
				return Result{"VERSION_CONFLICT", "invalid deletion plan"}
			}
			seen[key] = true
		}
		for _, t := range c.Tasks {
			f.state.Tasks[t.ID] = t
			delete(f.state.Replicas[t.SegmentID], t.Target.ID)
		}
		v.Status = "DELETING"
		if count == 0 {
			v.Status = "DELETED"
		}
		v.Version++
		f.state.Videos[c.VideoID] = v

	case "cleanup-create":
		if c.Task == nil || c.Task.Removed == nil || len(c.Task.Keep) != 2 {
			return Result{"INVALID_ARGUMENT", "invalid cleanup task"}
		}
		t := *c.Task
		if !domain.ValidID(t.ID) || t.State != "pending" || t.Reason != "cleanup" || t.Removed.PeerID != t.Target.ID || t.Removed.SegmentID != t.SegmentID {
			return Result{"INVALID_ARGUMENT", "invalid cleanup identity"}
		}
		if _, exists := f.state.Tasks[t.ID]; exists {
			return Result{"VERSION_CONFLICT", "task exists"}
		}
		for _, old := range f.state.Tasks {
			if old.State == "pending" && old.SegmentID == t.SegmentID {
				return Result{"VERSION_CONFLICT", "segment has pending work"}
			}
		}
		replicas := f.state.Replicas[t.SegmentID]
		if replicas[t.Target.ID] != *t.Removed || len(replicas) <= 2 || t.Keep[0].PeerID == t.Keep[1].PeerID {
			return Result{"VERSION_CONFLICT", "replicas changed"}
		}
		for _, keep := range t.Keep {
			if keep.PeerID == t.Target.ID || replicas[keep.PeerID] != keep {
				return Result{"VERSION_CONFLICT", "keepers changed"}
			}
		}
		f.state.Tasks[t.ID] = t
		delete(replicas, t.Target.ID)
	case "cleanup-complete":
		t, ok := f.state.Tasks[c.TaskID]
		if !ok || (t.Reason != "cleanup" && t.Reason != "video-delete") || t.State != "pending" {
			return Result{"VERSION_CONFLICT", "cleanup not pending"}
		}
		t.State = "completed"
		f.state.Tasks[t.ID] = t
		if t.Reason == "video-delete" {
			done := true
			for _, other := range f.state.Tasks {
				if other.VideoID == t.VideoID && other.State == "pending" {
					done = false
				}
			}
			if done {
				v := f.state.Videos[t.VideoID]
				v.Status = "DELETED"
				f.state.Videos[t.VideoID] = v
			}
		}

	case "task-create":
		if c.Task == nil {
			return Result{"INVALID_ARGUMENT", "missing task"}
		}
		t := *c.Task
		if !domain.ValidID(t.ID) || !domain.ValidID(t.Source.ID) || !domain.ValidID(t.Target.ID) || t.Source.ID == t.Target.ID || t.State != "pending" || t.Version != 1 {
			return Result{"INVALID_ARGUMENT", "invalid repair task"}
		}
		if _, ok := f.state.Tasks[t.ID]; ok {
			return Result{"VERSION_CONFLICT", "task already exists"}
		}
		valid := false
		for _, v := range f.state.Videos {
			for _, seg := range v.Manifest.Segments {
				if v.Status == "READY" && seg.ID == t.SegmentID && seg.Size == t.Size && seg.SHA256 == t.SHA256 {
					valid = true
				}
			}
		}
		if !valid {
			return Result{"VERSION_CONFLICT", "task must match a published segment"}
		}
		if a, ok := f.state.Replicas[t.SegmentID][t.Source.ID]; !ok || a.SHA256 != t.SHA256 {
			return Result{"NOT_READY", "source is not committed"}
		}
		for _, old := range f.state.Tasks {
			if old.State == "pending" && old.SegmentID == t.SegmentID {
				return Result{"VERSION_CONFLICT", "segment already has a pending task"}
			}
		}
		f.state.Tasks[t.ID] = t
	case "task-complete":
		t, ok := f.state.Tasks[c.TaskID]
		if !ok {
			return Result{"NOT_FOUND", "task not found"}
		}
		a := c.Receipt
		if (t.Reason == "cleanup" || t.Reason == "video-delete") || t.State != "pending" || a.PeerID != t.Target.ID || a.SegmentID != t.SegmentID || a.Size != t.Size || a.SHA256 != t.SHA256 || a.Version != t.Version || a.OperationID != t.ID {
			return Result{"VERSION_CONFLICT", "receipt does not match pending task"}
		}
		if f.state.Replicas[t.SegmentID] == nil {
			f.state.Replicas[t.SegmentID] = map[string]domain.Receipt{}
		}
		f.state.Replicas[t.SegmentID][a.PeerID] = a
		t.State = "completed"
		f.state.Tasks[t.ID] = t
	case "task-fail":
		t, ok := f.state.Tasks[c.TaskID]
		if !ok || t.State != "pending" || t.Reason == "cleanup" || t.Reason == "video-delete" {
			return Result{"VERSION_CONFLICT", "task is not pending"}
		}
		t.State = "failed"
		f.state.Tasks[t.ID] = t
	case "create":
		if err := c.Manifest.Validate(); err != nil {
			return Result{"INVALID_ARGUMENT", err.Error()}
		}
		if _, ok := f.state.Videos[c.Manifest.VideoID]; ok {
			return Result{"VERSION_CONFLICT", "video already exists"}
		}
		for _, v := range f.state.Videos {
			for _, old := range v.Manifest.Segments {
				for _, s := range c.Manifest.Segments {
					if old.ID == s.ID {
						return Result{"VERSION_CONFLICT", "segment ID already belongs to another video"}
					}
				}
			}
		}
		f.state.Videos[c.Manifest.VideoID] = domain.Video{Manifest: c.Manifest, Status: "DRAFT", Version: 1}
	case "confirm":
		a := c.Receipt
		for _, t := range f.state.Tasks {
			if t.Reason == "cleanup" && t.Removed != nil && t.SegmentID == a.SegmentID && t.Target.ID == a.PeerID && (t.State == "pending" || a.OperationID == t.Removed.OperationID) {
				return Result{"VERSION_CONFLICT", "retired receipt"}
			}
		}
		if a.Version != 1 || !domain.ValidID(a.PeerID) || !domain.ValidID(a.OperationID) {
			return Result{"INVALID_ARGUMENT", "invalid receipt"}
		}
		found := false
		for _, v := range f.state.Videos {
			for _, s := range v.Manifest.Segments {
				if s.ID == a.SegmentID && (v.Status == "DELETING" || v.Status == "DELETED") {
					return Result{"VERSION_CONFLICT", "video deleted"}
				}
				if s.ID == a.SegmentID && s.Size == a.Size && s.SHA256 == a.SHA256 {
					found = true
				}
			}
		}
		if !found {
			return Result{"VERSION_CONFLICT", "receipt does not match manifest"}
		}
		if f.state.Replicas[a.SegmentID] == nil {
			f.state.Replicas[a.SegmentID] = map[string]domain.Receipt{}
		}
		f.state.Replicas[a.SegmentID][a.PeerID] = a
	case "publish":
		v, ok := f.state.Videos[c.VideoID]
		if !ok {
			return Result{"NOT_FOUND", "video not found"}
		}
		if v.Status == "DELETING" || v.Status == "DELETED" {
			return Result{"VERSION_CONFLICT", "video deleted"}
		}
		if v.Version != c.ExpectedVersion {
			return Result{"VERSION_CONFLICT", "video version mismatch"}
		}
		for _, s := range v.Manifest.Segments {
			if len(f.state.Replicas[s.ID]) < 2 {
				return Result{"NOT_READY", "every segment requires two distinct confirmed peers"}
			}
		}
		v.Status = "READY"
		v.Version++
		f.state.Videos[c.VideoID] = v
	default:
		return Result{"INVALID_ARGUMENT", "unknown command"}
	}
	return Result{}
}
func (f *FSM) Copy() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	b, _ := json.Marshal(f.state)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}

type snapshot struct{ data []byte }

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	b, err := json.Marshal(f.Copy())
	return &snapshot{b}, err
}
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write(s.data); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}
func (s *snapshot) Release() {}
func (f *FSM) Restore(reader io.ReadCloser) error {
	defer reader.Close()
	var state State
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		return err
	}
	if state.Videos == nil || state.Replicas == nil || state.Operations == nil {
		return fmt.Errorf("invalid snapshot")
	}
	if state.Tasks == nil {
		state.Tasks = map[string]domain.Task{}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = state
	return nil
}
