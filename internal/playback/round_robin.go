// TODO(Tian): Measure the deterministic baseline before modifying source selection.
package playback

import "cooperative.video/internal/domain"

type RoundRobin struct{}

func (RoundRobin) Candidates(seg domain.Segment, locations []domain.Location) []domain.Location {
	if len(locations) == 0 {
		return nil
	}
	out := make([]domain.Location, len(locations))
	for i := range out {
		out[i] = locations[(seg.Index+i)%len(locations)]
	}
	return out
}
