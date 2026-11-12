// TODO(Haoyu): Own the common domain model; extend versioned task and coding types here.
package domain

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var mediaMIME = regexp.MustCompile(`^video/mp4; codecs="avc1\.[0-9a-fA-F]{6}(, mp4a\.40\.2)?"$`)

var identifier = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func ValidID(id string) bool { return identifier.MatchString(id) }

type Segment struct {
	ID         string `json:"segment_id"`
	Index      int    `json:"index"`
	DurationMS int64  `json:"duration_ms"`
	Size       int64  `json:"size_bytes"`
	SHA256     string `json:"sha256"`
}

// MediaInfo describes one continuous encoded timeline shared by all fMP4 objects.
type MediaInfo struct {
	MIMEType   string `json:"mime_type"`
	DurationMS int64  `json:"duration_ms"`
}
type Manifest struct {
	SchemaVersion  int        `json:"schema_version"`
	VideoID        string     `json:"video_id"`
	Representation string     `json:"representation"`
	Segments       []Segment  `json:"segments"`
	Media          *MediaInfo `json:"media,omitempty"`
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != 1 || !ValidID(m.VideoID) || m.Representation == "" || len(m.Segments) == 0 {
		return fmt.Errorf("invalid manifest header")
	}
	continuous := strings.HasPrefix(m.Representation, "continuous-fmp4-")
	if continuous && (m.Media == nil || m.Media.DurationMS <= 0 || len(m.Media.MIMEType) > 256 || !mediaMIME.MatchString(m.Media.MIMEType)) {
		return fmt.Errorf("invalid continuous media configuration")
	}
	seen := map[string]bool{}
	var total int64
	for i, s := range m.Segments {
		h, err := hex.DecodeString(s.SHA256)
		if !ValidID(s.ID) || seen[s.ID] || s.Index != i || s.DurationMS <= 0 || s.Size <= 0 || s.Size > 64<<20 || err != nil || len(h) != 32 || strings.ToLower(s.SHA256) != s.SHA256 {
			return fmt.Errorf("invalid segment at index %d", i)
		}
		seen[s.ID] = true
		if continuous {
			if s.DurationMS > m.Media.DurationMS-total {
				return fmt.Errorf("invalid continuous media duration")
			}
			total += s.DurationMS
		}
	}
	if continuous && total != m.Media.DurationMS {
		return fmt.Errorf("continuous media duration mismatch")
	}
	return nil
}

type Peer struct {
	ID  string `json:"peer_id"`
	URL string `json:"url"`
}
type Receipt struct {
	OperationID string `json:"operation_id"`
	PeerID      string `json:"peer_id"`
	SegmentID   string `json:"segment_id"`
	Version     int    `json:"representation_version"`
	Size        int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
}
type Location struct {
	Peer
	Version int `json:"representation_version"`
}
type Video struct {
	Manifest Manifest `json:"manifest"`
	Status   string   `json:"status"`
	Version  int      `json:"version"`
}
type PeerObservation struct {
	Peer                 Peer    `json:"peer"`
	FreeBytes            int64   `json:"free_bytes"`
	UploadBytesPerSecond float64 `json:"upload_bytes_per_second"`
	OnlineFraction       float64 `json:"online_fraction"`
}
type Task struct {
	ID        string `json:"task_id"`
	SegmentID string `json:"segment_id"`
	Version   int    `json:"representation_version"`
	Source    Peer   `json:"source"`
	Target    Peer   `json:"target"`
	State     string `json:"state"`
}
