// TODO(Tian): Add frozen-deadline misses and per-request JSONL; this is the sole definition of session-level playback metrics.
package playback

type Summary struct {
	VideoID           string  `json:"video_id"`
	Status            string  `json:"status"`
	Error             string  `json:"error,omitempty"`
	StartupMS         float64 `json:"startup_delay_ms"`
	StallMS           float64 `json:"stall_time_ms"`
	MediaMS           float64 `json:"media_playback_elapsed_ms"`
	RebufferingRatio  float64 `json:"rebuffering_ratio"`
	VerifiedBytes     int64   `json:"verified_bytes"`
	SegmentsPlayed    int     `json:"segments_played"`
	SimulatedPlayback bool    `json:"simulated_playback"`
}

func (s *Summary) finalize() {
	if total := s.MediaMS + s.StallMS; total > 0 {
		s.RebufferingRatio = s.StallMS / total
	}
}
