// TODO(Yuquan): Add capacity reservations and matched-budget adaptive placement; current plans are advisory, with Peer quotas authoritative.
package placement

import (
	"cooperative.video/internal/domain"
	"fmt"
)

type Policy interface {
	Choose(segment domain.Segment, peers []domain.PeerObservation, copies int) ([]domain.Peer, error)
}
type Static struct{}

func (Static) Choose(s domain.Segment, peers []domain.PeerObservation, copies int) ([]domain.Peer, error) {
	if copies < 1 || len(peers) < copies {
		return nil, fmt.Errorf("insufficient distinct peers")
	}
	var out []domain.Peer
	seen := map[string]bool{}
	for i := 0; i < len(peers); i++ {
		p := peers[(s.Index+i)%len(peers)]
		if !seen[p.Peer.ID] && p.FreeBytes >= s.Size {
			out = append(out, p.Peer)
			seen[p.Peer.ID] = true
		}
		if len(out) == copies {
			return out, nil
		}
	}
	return nil, fmt.Errorf("insufficient peer capacity")
}
