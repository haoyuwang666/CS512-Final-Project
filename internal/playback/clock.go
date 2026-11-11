// TODO(Tian): Add deterministic clock injection for metric tests; this clock consumes media in real wall time without decoding.
package playback

import (
	"context"
	"time"
)

func consume(ctx context.Context, d time.Duration) (time.Duration, error) {
	start := time.Now()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		elapsed := time.Since(start)
		if elapsed > d {
			elapsed = d
		}
		return elapsed, ctx.Err()
	case <-timer.C:
		return d, nil
	}
}
