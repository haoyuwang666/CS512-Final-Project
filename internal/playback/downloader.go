// TODO(Tian): Preserve size and hash verification for every download path.
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
	data, _, err := d.download(ctx, s, p)
	return data, err
}

// download also reports response-body bytes for failed or corrupt attempts. This is
// application payload accounting; HTTP headers and TCP retransmissions are excluded.
func (d *Downloader) download(ctx context.Context, s domain.Segment, p domain.Location) ([]byte, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL+"/api/v1/segments/"+s.ID, nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := d.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, 0, fmt.Errorf("peer %s HTTP %d", p.ID, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, s.Size+1))
	if err != nil {
		return nil, int64(len(b)), err
	}
	if int64(len(b)) != s.Size || fmt.Sprintf("%x", sha256.Sum256(b)) != s.SHA256 {
		return nil, int64(len(b)), fmt.Errorf("HASH_MISMATCH: %s", s.ID)
	}
	return b, int64(len(b)), nil
}
