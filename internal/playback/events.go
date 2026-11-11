// TODO(Tian): Persist event logs for experiments and implement reconnect replay; callbacks must remain fast and thread-safe.
package playback

import "time"

type Event struct {
	Sequence  int64     `json:"sequence"`
	Type      string    `json:"type"`
	Time      time.Time `json:"time"`
	SegmentID string    `json:"segment_id,omitempty"`
	PeerID    string    `json:"peer_id,omitempty"`
	Message   string    `json:"message,omitempty"`
}
