// TODO(Yuquan): Add measured popularity only after defining an observation window.
package placement

import (
	"cooperative.video/internal/domain"
	"fmt"
	"math"
	"sort"
)

// Adaptive uses current capacity, upload bandwidth and availability at planning
// time. It does not claim to learn popularity or migrate existing objects.
type Adaptive struct{}

func (Adaptive) Choose(s domain.Segment, peers []domain.PeerObservation, copies int) ([]domain.Peer, error) {
	if copies < 1 {
		return nil, fmt.Errorf("copies must be positive")
	}
	eligible := []domain.PeerObservation{}
	seen := map[string]bool{}
	for _, p := range peers {
		if domain.ValidID(p.Peer.ID) && !seen[p.Peer.ID] && p.FreeBytes >= s.Size && p.OnlineFraction > 0 && !math.IsNaN(p.UploadBytesPerSecond) && !math.IsInf(p.UploadBytesPerSecond, 0) {
			eligible = append(eligible, p)
			seen[p.Peer.ID] = true
		}
	}
	if len(eligible) < copies {
		return nil, fmt.Errorf("insufficient distinct healthy capacity")
	}
	maxFree, maxRate := 1.0, 1.0
	for _, p := range eligible {
		maxFree = math.Max(maxFree, float64(p.FreeBytes))
		maxRate = math.Max(maxRate, p.UploadBytesPerSecond)
	}
	weight := 0.5
	if s.Index < 3 {
		weight = 0.8
	}
	score := func(p domain.PeerObservation) float64 {
		return math.Min(1, p.OnlineFraction) * (weight*p.UploadBytesPerSecond/maxRate + (1-weight)*float64(p.FreeBytes)/maxFree)
	}
	sort.Slice(eligible, func(i, j int) bool {
		a, b := score(eligible[i]), score(eligible[j])
		if a == b {
			return eligible[i].Peer.ID < eligible[j].Peer.ID
		}
		return a > b
	})
	out := make([]domain.Peer, copies)
	for i := range out {
		out[i] = eligible[i].Peer
	}
	return out, nil
}
