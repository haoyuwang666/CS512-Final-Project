// TODO(Tian): Revisit cache lifetime when mutable representations or deletion are introduced.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type cacheEntry[T any] struct {
	value   T
	fetched time.Time
}
type MetadataClient struct {
	URLs      []string
	HTTP      *http.Client
	mu        sync.Mutex
	manifests map[string]cacheEntry[domain.Manifest]
	locations map[string]cacheEntry[[]domain.Location]
}

func NewMetadataClient(urls []string) *MetadataClient {
	return &MetadataClient{URLs: urls, HTTP: &http.Client{Timeout: 4 * time.Second}, manifests: make(map[string]cacheEntry[domain.Manifest]), locations: make(map[string]cacheEntry[[]domain.Location])}
}

type metadataError struct {
	status       int
	path, detail string
}

func (e *metadataError) Error() string {
	return fmt.Sprintf("metadata HTTP %d for %s: %s", e.status, e.path, e.detail)
}
func (c *MetadataClient) get(ctx context.Context, path string, out any) (transient bool, err error) {
	transient = true
	for _, base := range c.URLs {
		req, e := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(strings.TrimSpace(base), "/")+"/api/v1"+path, nil)
		if e != nil {
			return false, e
		}
		res, e := c.HTTP.Do(req)
		if e != nil {
			err = e
			continue
		}
		if res.StatusCode != 200 {
			body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
			res.Body.Close()
			err = &metadataError{res.StatusCode, path, strings.TrimSpace(string(body))}
			if res.StatusCode < 500 && res.StatusCode != 429 {
				return false, err
			}
			continue
		}
		e = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(out)
		res.Body.Close()
		return false, e
	}
	if err == nil {
		err = fmt.Errorf("no controller URLs configured")
	}
	return transient, err
}
func cloneManifest(m domain.Manifest) domain.Manifest {
	m.Segments = append([]domain.Segment(nil), m.Segments...)
	if m.Media != nil {
		media := *m.Media
		m.Media = &media
	}
	return m
}

// Bound both caches; eviction uses fetch age rather than random map iteration.
func trimCache[T any](entries map[string]cacheEntry[T], limit int) {
	if len(entries) < limit {
		return
	}
	oldest := ""
	var at time.Time
	for id, e := range entries {
		if oldest == "" || e.fetched.Before(at) {
			oldest = id
			at = e.fetched
		}
	}
	delete(entries, oldest)
}
func (c *MetadataClient) Manifest(ctx context.Context, id string) (domain.Manifest, error) {
	if !domain.ValidID(id) {
		return domain.Manifest{}, fmt.Errorf("invalid video ID")
	}
	if err := ctx.Err(); err != nil {
		return domain.Manifest{}, err
	}
	c.mu.Lock()
	cached, ok := c.manifests[id]
	c.mu.Unlock()
	if ok && time.Since(cached.fetched) < 5*time.Second {
		return cloneManifest(cached.value), nil
	}
	var out domain.Manifest
	transient, err := c.get(ctx, "/videos/"+id+"/manifest", &out)
	if err == nil {
		err = out.Validate()
		if err == nil && out.VideoID != id {
			err = fmt.Errorf("manifest video ID mismatch")
		}
	}
	if err != nil {
		// Never hide an authoritative 404/409, invalid response, or cancellation behind stale data.
		if transient && ctx.Err() == nil && ok && time.Since(cached.fetched) < 5*time.Minute {
			return cloneManifest(cached.value), nil
		}
		if !transient {
			c.mu.Lock()
			delete(c.manifests, id)
			c.mu.Unlock()
		}
		return domain.Manifest{}, err
	}
	c.mu.Lock()
	trimCache(c.manifests, 128)
	c.manifests[id] = cacheEntry[domain.Manifest]{cloneManifest(out), time.Now()}
	c.mu.Unlock()
	return out, nil
}
func (c *MetadataClient) Locations(ctx context.Context, id string) ([]domain.Location, error) {
	return c.locationsFor(ctx, id, false)
}
func (c *MetadataClient) RefreshLocations(ctx context.Context, id string) ([]domain.Location, error) {
	return c.locationsFor(ctx, id, true)
}
func (c *MetadataClient) locationsFor(ctx context.Context, id string, refresh bool) ([]domain.Location, error) {
	if !domain.ValidID(id) {
		return nil, fmt.Errorf("invalid segment ID")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	cached, ok := c.locations[id]
	c.mu.Unlock()
	if !refresh && ok && time.Since(cached.fetched) < 5*time.Second {
		return append([]domain.Location(nil), cached.value...), nil
	}
	var out protocol.Locations
	transient, err := c.get(ctx, "/segments/"+id+"/locations", &out)
	if err == nil {
		if out.SegmentID != id {
			err = fmt.Errorf("locations segment ID mismatch")
		}
		for _, p := range out.Locations {
			if !domain.ValidID(p.ID) || p.Version != 1 || !(strings.HasPrefix(p.URL, "http://") || strings.HasPrefix(p.URL, "https://")) {
				err = fmt.Errorf("invalid location")
			}
		}
	}
	if err != nil {
		if transient && ctx.Err() == nil && ok && time.Since(cached.fetched) < 5*time.Minute {
			return append([]domain.Location(nil), cached.value...), nil
		}
		if !transient {
			c.mu.Lock()
			delete(c.locations, id)
			c.mu.Unlock()
		}
		return nil, err
	}
	c.mu.Lock()
	trimCache(c.locations, 4096)
	c.locations[id] = cacheEntry[[]domain.Location]{append([]domain.Location(nil), out.Locations...), time.Now()}
	c.mu.Unlock()
	return out.Locations, nil
}
