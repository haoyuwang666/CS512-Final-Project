// TODO(Tian): Implement predicted completion, EWMA throughput and stall-shifted deadlines; do not alias the baseline as an optimization.
package playback

import "errors"

var ErrDeadlineNotImplemented = errors.New("deadline strategy not implemented; use round_robin")
