// TODO(Tian): Add cached manifests and controlled refresh during controller outages; retries use configured controller URLs only.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MetadataClient struct {
	URLs []string
	HTTP *http.Client
}

func NewMetadataClient(urls []string) *MetadataClient {
	return &MetadataClient{URLs: urls, HTTP: &http.Client{Timeout: 4 * time.Second}}
}
func (c *MetadataClient) get(ctx context.Context, path string, out any) error {
	var last error
	for _, base := range c.URLs {
		req, err := http.NewRequestWithContext(ctx, "GET", base+"/api/v1"+path, nil)
		if err != nil {
			return err
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			last = err
			continue
		}
		if res.StatusCode != 200 {
			last = fmt.Errorf("metadata HTTP %d for %s", res.StatusCode, path)
			res.Body.Close()
			continue
		}
		err = json.NewDecoder(res.Body).Decode(out)
		res.Body.Close()
		if err != nil {
			return err
		}
		return nil
	}
	if last == nil {
		last = fmt.Errorf("no controller URLs configured")
	}
	return last
}
func (c *MetadataClient) Manifest(ctx context.Context, id string) (domain.Manifest, error) {
	var out domain.Manifest
	err := c.get(ctx, "/videos/"+id+"/manifest", &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}
func (c *MetadataClient) Locations(ctx context.Context, id string) ([]domain.Location, error) {
	var out protocol.Locations
	err := c.get(ctx, "/segments/"+id+"/locations", &out)
	return out.Locations, err
}
