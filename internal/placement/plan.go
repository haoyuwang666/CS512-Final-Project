// TODO(Yuquan): Persist capacity reservations when supporting concurrent ingestion guarantees.
package placement

import "cooperative.video/internal/domain"

// Plan budgets all mandatory replicas first; optional copies must never starve
// a later segment's mandatory pair. The input observation slice is not mutated.
func Plan(segments []domain.Segment, peers []domain.PeerObservation, policy Policy, extraBytes int64) (map[string][]domain.Peer, int64, error) {
	observations := append([]domain.PeerObservation(nil), peers...)
	plans := map[string][]domain.Peer{}
	deduct := func(targets []domain.Peer, size int64) {
		for i := range observations {
			for _, p := range targets {
				if observations[i].Peer.ID == p.ID {
					observations[i].FreeBytes -= size
				}
			}
		}
	}
	for _, seg := range segments {
		targets, err := policy.Choose(seg, observations, 2)
		if err != nil {
			return nil, 0, err
		}
		plans[seg.ID] = targets
		deduct(targets, seg.Size)
	}
	remaining := extraBytes
	for _, seg := range segments {
		if remaining < seg.Size {
			continue
		}
		candidates := []domain.PeerObservation{}
		for _, p := range observations {
			used := false
			for _, target := range plans[seg.ID] {
				if target.ID == p.Peer.ID {
					used = true
				}
			}
			if !used {
				candidates = append(candidates, p)
			}
		}
		if extra, err := policy.Choose(seg, candidates, 1); err == nil {
			plans[seg.ID] = append(plans[seg.ID], extra[0])
			remaining -= seg.Size
			deduct(extra, seg.Size)
		}
	}
	return plans, extraBytes - remaining, nil
}
