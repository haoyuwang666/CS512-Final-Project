// TODO(Yuquan): Implement popularity/startup-weighted optional replicas after matching fixed-placement storage budgets.
package placement

import (
	"cooperative.video/internal/domain"
	"fmt"
)

type Adaptive struct{}

func (Adaptive) Choose(domain.Segment, []domain.PeerObservation, int) ([]domain.Peer, error) {
	return nil, fmt.Errorf("adaptive placement not implemented")
}
