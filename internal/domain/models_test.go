// TODO(Haoyu): Extend manifest safety tests when additional media representations are introduced.
package domain

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestContinuousManifestRequiresConsistentCodecAndTimeline(t *testing.T) {
	manifest := Manifest{SchemaVersion: 1, VideoID: "v", Representation: "continuous-fmp4-h264-aac-v1", Media: &MediaInfo{MIMEType: `video/mp4; codecs="avc1.64000c, mp4a.40.2"`, DurationMS: 4000}, Segments: []Segment{
		{ID: "v--0", Index: 0, DurationMS: 2000, Size: 1, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("x")))},
		{ID: "v--1", Index: 1, DurationMS: 2000, Size: 1, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("x")))},
	}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, media := range []*MediaInfo{nil, {MIMEType: "video/mp4", DurationMS: 4000}, {MIMEType: `video/mp4; codecs="avc1.64000c"`, DurationMS: 3000}, {MIMEType: `video/mp4; codecs="avc1.64000c"`, DurationMS: 5000}} {
		invalid := manifest
		invalid.Media = media
		if invalid.Validate() == nil {
			t.Fatalf("invalid media accepted: %+v", media)
		}
	}
	manifest.Representation = "legacy"
	manifest.Media = nil
	if err := manifest.Validate(); err != nil {
		t.Fatalf("legacy manifest compatibility broken: %v", err)
	}
}
