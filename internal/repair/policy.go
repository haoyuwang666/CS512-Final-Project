// TODO(Yuquan): Implement grace-period repair planning from verified inventory; this policy must not mutate Raft or delete data.
package repair

import "cooperative.video/internal/domain"

type Snapshot struct {
	Segment    domain.Segment
	Reachable  []domain.Peer
	Candidates []domain.PeerObservation
}
type Policy interface {
	Plan(Snapshot) (domain.Task, error)
}
