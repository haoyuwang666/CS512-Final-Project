// TODO(Tian): Add per-source throughput estimates and deadline-driven retry; all successful objects must pass size/hash validation.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Downloader struct{ HTTP *http.Client }

func NewDownloader() *Downloader { return &Downloader{HTTP: &http.Client{Timeout: 8 * time.Second}} }
func (d *Downloader) Download(ctx context.Context, s domain.Segment, p domain.Location) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL+"/api/v1/segments/"+s.ID, nil)
	if err != nil {
		return nil, err
	}
	res, err := d.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("peer %s HTTP %d", p.ID, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, s.Size+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != s.Size || fmt.Sprintf("%x", sha256.Sum256(b)) != s.SHA256 {
		return nil, fmt.Errorf("HASH_MISMATCH: %s", s.ID)
	}
	return b, nil
}
