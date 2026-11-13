// TODO(Yuquan): Bound/paginate reports for inventories larger than the lab JSON limit.
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Reports are advisory and never publish a replica. The leader verifies peers
// independently before planning repair. Retry each configured Controller.
func (s *Server) RunReports(ctx context.Context, controllers []string) {
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report := s.Report()
			payload, _ := json.Marshal(report)
			for _, route := range []string{"heartbeat", "inventory"} {
				for _, base := range controllers {
					req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(base, "/")+"/api/v1/peers/"+s.PeerID+"/"+route, bytes.NewReader(payload))
					if err != nil {
						continue
					}
					req.Header.Set("Content-Type", "application/json")
					res, err := client.Do(req)
					if err != nil {
						continue
					}
					io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
					res.Body.Close()
					if res.StatusCode == 200 {
						break
					}
				}
			}
		}
	}
}
