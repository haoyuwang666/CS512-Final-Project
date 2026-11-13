// TODO(Yuquan): Implement grace-period repair planning from verified inventory; this policy must not mutate Raft or delete data.
package repair

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/placement"
	"fmt"
)

type Snapshot struct {
	Segment    domain.Segment
	Reachable  []domain.Peer
	Candidates []domain.PeerObservation
}
type Policy interface {
	Plan(Snapshot) (domain.Task, error)
}

// MinimumTwo chooses a missing holder without deleting historical replicas.
// The coordinator handles grace periods and persists the resulting plan.
type MinimumTwo struct{}

func (MinimumTwo) Plan(s Snapshot) (domain.Task, error) {
	if len(s.Reachable) == 0 || len(s.Reachable) >= 2 {
		return domain.Task{}, fmt.Errorf("repair requires exactly one reachable committed replica")
	}
	candidates := []domain.PeerObservation{}
	for _, p := range s.Candidates {
		if p.Peer.ID != s.Reachable[0].ID {
			candidates = append(candidates, p)
		}
	}
	targets, err := (placement.Adaptive{}).Choose(s.Segment, candidates, 1)
	if err != nil {
		return domain.Task{}, err
	}
	return domain.Task{SegmentID: s.Segment.ID, Version: 1, Source: s.Reachable[0], Target: targets[0], State: "pending", Size: s.Segment.Size, SHA256: s.Segment.SHA256, Reason: "under-replicated"}, nil
}
