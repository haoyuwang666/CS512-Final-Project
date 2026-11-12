// TODO(Tian): Keep session metric definitions stable across experiment versions.
package playback

// Summary uses milliseconds for durations and response-body bytes for client traffic.
// DeadlineMisses counts consumed noninitial segments against the frozen timeline.
type Summary struct {
	VideoID           string  `json:"video_id"`
	Status            string  `json:"status"`
	Error             string  `json:"error,omitempty"`
	StartupMS         float64 `json:"startup_delay_ms"`
	StallMS           float64 `json:"stall_time_ms"`
	MediaMS           float64 `json:"media_playback_elapsed_ms"`
	RebufferingRatio  float64 `json:"rebuffering_ratio"`
	VerifiedBytes     int64   `json:"verified_bytes"`
	DownloadedBytes   int64   `json:"downloaded_bytes"`
	Requests          int     `json:"requests"`
	FailedRequests    int     `json:"failed_requests"`
	DeadlineMisses    int     `json:"deadline_misses"`
	SegmentsPlayed    int     `json:"segments_played"`
	SimulatedPlayback bool    `json:"simulated_playback"`
}

func (s *Summary) finalize() {
	if total := s.MediaMS + s.StallMS; total > 0 {
		s.RebufferingRatio = s.StallMS / total
	}
}
