// TODO(Haoyu): Extend the real Raft FSM with reservations and durable repair transitions; never perform network I/O in Apply.
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
	Kind            string          `json:"kind"`
	OperationID     string          `json:"operation_id"`
	Manifest        domain.Manifest `json:"manifest,omitempty"`
	Receipt         domain.Receipt  `json:"receipt,omitempty"`
	VideoID         string          `json:"video_id,omitempty"`
	ExpectedVersion int             `json:"expected_version,omitempty"`
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
}
type FSM struct {
	mu    sync.RWMutex
	state State
}

func NewFSM() *FSM {
	return &FSM{state: State{Videos: map[string]domain.Video{}, Replicas: map[string]map[string]domain.Receipt{}, Operations: map[string]Operation{}}}
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
	switch c.Kind {
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
		if a.Version != 1 || !domain.ValidID(a.PeerID) || !domain.ValidID(a.OperationID) {
			return Result{"INVALID_ARGUMENT", "invalid receipt"}
		}
		found := false
		for _, v := range f.state.Videos {
			for _, s := range v.Manifest.Segments {
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
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = state
	return nil
}
