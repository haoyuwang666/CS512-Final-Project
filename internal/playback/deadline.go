// TODO(Tian): Calibrate prediction defaults against representative workloads.
package playback

import (
	"cooperative.video/internal/domain"
	"sort"
	"sync"
	"time"
)

type PeerEstimate struct {
	PeerID         string  `json:"peer_id"`
	BytesPerSecond float64 `json:"bytes_per_second"`
	InFlight       int     `json:"in_flight"`
	Failures       int     `json:"failures"`
}

// Deadline ranks replicas by predicted completion, including concurrent work on each source.
// Samples belong to one session; independent sessions cannot contaminate experiment results.
type Deadline struct {
	mu    sync.Mutex
	peers map[string]PeerEstimate
}

func newDeadline() *Deadline { return &Deadline{peers: make(map[string]PeerEstimate)} }
func (d *Deadline) estimate(id string) PeerEstimate {
	p, ok := d.peers[id]
	if !ok {
		p = PeerEstimate{PeerID: id, BytesPerSecond: 1 << 20}
	}
	return p
}
func (d *Deadline) prediction(s domain.Segment, id string) time.Duration {
	p := d.estimate(id)
	return time.Duration((float64(s.Size)*float64(p.InFlight+1)/p.BytesPerSecond + .05 + float64(p.Failures)*.25) * float64(time.Second))
}
func (d *Deadline) Candidates(s domain.Segment, locations []domain.Location) []domain.Location {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := (RoundRobin{}).Candidates(s, locations)
	sort.SliceStable(out, func(i, j int) bool { return d.prediction(s, out[i].ID) < d.prediction(s, out[j].ID) })
	return out
}
func (d *Deadline) start(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p := d.estimate(id)
	p.InFlight++
	d.peers[id] = p
}
func (d *Deadline) finish(id string, bytes int64, elapsed time.Duration, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p := d.estimate(id)
	p.InFlight--
	if err != nil {
		p.Failures++
	} else {
		if elapsed > 0 {
			p.BytesPerSecond = .7*p.BytesPerSecond + .3*float64(bytes)/elapsed.Seconds()
		}
		p.Failures = 0
	}
	d.peers[id] = p
}

// A missed deadline must not make every remaining replica immediately fail. Keep a
// recovery floor, while avoiding spending the full HTTP timeout on an urgent source.
func (d *Deadline) budget(s domain.Segment, id string, remaining time.Duration) time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	predicted := 2 * d.prediction(s, id)
	if remaining > 0 {
		predicted = min(predicted, remaining)
	}
	return max(250*time.Millisecond, min(predicted, 8*time.Second))
}
func (d *Deadline) snapshot() []PeerEstimate {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]PeerEstimate, 0, len(d.peers))
	for _, p := range d.peers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeerID < out[j].PeerID })
	return out
}
