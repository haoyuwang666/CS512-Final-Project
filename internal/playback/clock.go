// TODO(Tian): Keep clock implementations cancellation-aware as playback evolves.
package playback

import (
	"context"
	"time"
)

// Clock separates media time from the wall clock so metric tests can advance time deterministically.
// Implementations must be safe for concurrent Now calls.
type Clock interface {
	Now() time.Time
	Wait(context.Context, time.Duration) (time.Duration, error)
}
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
func (RealClock) Wait(ctx context.Context, d time.Duration) (time.Duration, error) {
	start := time.Now()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return min(time.Since(start), d), ctx.Err()
	case <-timer.C:
		return d, nil
	}
}
