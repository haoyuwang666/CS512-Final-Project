// TODO(Tian): Preserve bounded buffering when adding a media decoder.
package playback

import "time"

type downloadResult struct {
	size     int
	err      error
	finished time.Time
}
type bufferSlot chan downloadResult

func newSlot() bufferSlot { return make(bufferSlot, 1) }

// BufferEntry describes logical readiness, not retained MP4 payloads.
type BufferEntry struct {
	SegmentID  string `json:"segment_id"`
	Index      int    `json:"index"`
	DurationMS int64  `json:"duration_ms"`
	State      string `json:"state"`
	PeerID     string `json:"peer_id,omitempty"`
}
type Snapshot struct {
	Summary Summary        `json:"summary"`
	Buffer  []BufferEntry  `json:"buffer"`
	Peers   []PeerEstimate `json:"peers"`
}
