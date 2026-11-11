// TODO(Tian): Keep source selection separate from downloading; extend assignments with explicit deadline and queue estimates.
package playback

import "cooperative.video/internal/domain"

type Scheduler interface {
	Candidates(domain.Segment, []domain.Location) []domain.Location
}

func NewScheduler(strategy string) (Scheduler, error) {
	switch strategy {
	case "", "round_robin":
		return RoundRobin{}, nil
	default:
		return nil, ErrDeadlineNotImplemented
	}
}
