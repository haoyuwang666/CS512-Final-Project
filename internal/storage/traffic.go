// TODO(Yuquan): Maintain classified traffic accounting and repair policies under the shared upload limit.
package storage

import (
	"context"
	"cooperative.video/internal/domain"
	"fmt"
	"golang.org/x/time/rate"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type policySample struct {
	Sequence    int64   `json:"sequence"`
	UnixMS      int64   `json:"unix_ms"`
	Foreground  int     `json:"active_foreground"`
	Repair      int     `json:"active_repair"`
	OldestAgeMS int64   `json:"oldest_repair_age_ms"`
	Cap         float64 `json:"repair_bytes_per_second"`
}
type trafficState struct {
	mu            sync.Mutex
	policy        string
	total         float64
	repairLimiter *rate.Limiter
	active        map[string]int
	requests      map[string]int64
	bytes         map[string]int64
	pending       map[*http.Request]int64
	idleSince     time.Time
	history       []policySample
	sequence      int64
}

func newTraffic(total float64) trafficState {
	return trafficState{policy: "S", total: total, repairLimiter: rate.NewLimiter(rate.Limit(total), 32<<10), active: map[string]int{}, requests: map[string]int64{}, bytes: map[string]int64{}, pending: map[*http.Request]int64{}, idleSince: time.Now()}
}
func (s *Server) ConfigureRepairPolicy(policy string) error {
	if policy != "S" && policy != "F25" && policy != "F50" && policy != "L" {
		return fmt.Errorf("REPAIR_POLICY must be S, F25, F50, or L")
	}
	t := &s.Traffic
	t.mu.Lock()
	defer t.mu.Unlock()
	t.policy = policy
	t.update(time.Now())
	return nil
}

// The allowance is an extra cap, not a reservation. Both classes still consume
// the unchanged shared limiter (including its original 32 KiB token burst).
func repairFraction(policy string, busy bool, idle time.Duration, age time.Duration) float64 {
	switch policy {
	case "F25":
		return .25
	case "F50":
		return .5
	case "L":
		f := .25
		if !busy && idle >= 2*time.Second {
			f = 1
		}
		if age > 60*time.Second && f < .5 {
			f = .5
		}
		return f
	}
	return 1
}
func (t *trafficState) update(now time.Time) {
	var oldest int64
	for _, created := range t.pending {
		if created > 0 && (oldest == 0 || created < oldest) {
			oldest = created
		}
	}
	var age time.Duration
	if oldest > 0 {
		age = now.Sub(time.UnixMilli(oldest))
		if age < 0 {
			age = 0
		}
	}
	cap := t.total * repairFraction(t.policy, t.active["foreground"] > 0, now.Sub(t.idleSince), age)
	t.repairLimiter.SetLimitAt(now, rate.Limit(cap))
	t.sequence++
	t.history = append(t.history, policySample{t.sequence, now.UnixMilli(), t.active["foreground"], t.active["repair"], age.Milliseconds(), cap})
	if len(t.history) > 16 {
		t.history = t.history[len(t.history)-16:]
	}
}
func (s *Server) RunPolicy(ctx context.Context) {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			t := &s.Traffic
			t.mu.Lock()
			t.update(now)
			t.mu.Unlock()
		}
	}
}
func (s *Server) beginTransfer(r *http.Request) (string, func()) {
	class := "foreground"
	created, err := strconv.ParseInt(r.Header.Get("X-Repair-Created-Unix-MS"), 10, 64)
	// Trusted lab protocol. Old committed tasks may have no timestamp: classify
	// their traffic but do not invent a creation time for the aging rule.
	if r.Header.Get("X-Transfer-Class") == "probe" {
		class = "probe"
	} else if r.Header.Get("X-Transfer-Class") == "repair" && domain.ValidID(r.Header.Get("X-Repair-Task-ID")) {
		class = "repair"
	}
	t := &s.Traffic
	t.mu.Lock()
	t.active[class]++
	t.requests[class]++
	if class == "repair" && err == nil {
		t.pending[r] = created
	}
	t.mu.Unlock()
	return class, func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.active[class]--
		delete(t.pending, r)
		if class == "foreground" && t.active[class] == 0 {
			t.idleSince = time.Now()
		}
	}
}
func (s *Server) countBytes(class string, n int64) {
	t := &s.Traffic
	t.mu.Lock()
	defer t.mu.Unlock()
	t.bytes[class] += n
}
func (s *Server) trafficSnapshot() map[string]any {
	t := &s.Traffic
	t.mu.Lock()
	defer t.mu.Unlock()
	clone := func(m map[string]int64) map[string]int64 {
		out := map[string]int64{}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return map[string]any{"repair_policy": t.policy, "class_bytes": clone(t.bytes), "class_requests": clone(t.requests), "active_foreground": t.active["foreground"], "active_repair": t.active["repair"], "repair_bytes_per_second": float64(t.repairLimiter.Limit()), "policy_history": append([]policySample(nil), t.history...)}
}
