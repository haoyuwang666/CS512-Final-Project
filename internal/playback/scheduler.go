// TODO(Tian): Extend source prediction only alongside reproducible comparison tests.
package playback

import (
	"cooperative.video/internal/domain"
	"fmt"
)

type Scheduler interface {
	Candidates(domain.Segment, []domain.Location) []domain.Location
}

func NewScheduler(strategy string) (Scheduler, error) {
	switch strategy {
	case "", "round_robin":
		return RoundRobin{}, nil
	case "deadline":
		return newDeadline(), nil
	default:
		return nil, fmt.Errorf("unknown playback strategy %q", strategy)
	}
}
