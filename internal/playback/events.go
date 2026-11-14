// TODO(Tian): Version event schemas before changing persisted field semantics.
package playback

import "time"

type Event struct {
	SinceStartMS float64    `json:"since_start_ms"`
	Sequence     int64      `json:"sequence"`
	Type         string     `json:"type"`
	Time         time.Time  `json:"time"`
	SegmentID    string     `json:"segment_id,omitempty"`
	PeerID       string     `json:"peer_id,omitempty"`
	Message      string     `json:"message,omitempty"`
	Bytes        int64      `json:"bytes,omitempty"`
	ElapsedMS    float64    `json:"elapsed_ms,omitempty"`
	Deadline     *time.Time `json:"deadline,omitempty"`
	Snapshot     *Snapshot  `json:"snapshot,omitempty"`
}
