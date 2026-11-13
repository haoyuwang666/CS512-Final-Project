// TODO(Yuquan): Paginate inventories and bound parallel repair before scaling beyond lab workloads.
package control

import (
	"bytes"
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/repair"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Callbacks keep network and clock effects outside the Raft FSM.
type View struct {
	Videos   map[string]domain.Video
	Replicas map[string]map[string]domain.Receipt
	Tasks    map[string]domain.Task
}
type Coordinator struct {
	Peers    []domain.Peer
	Client   *http.Client
	Grace    time.Duration
	Interval time.Duration
	Snapshot func() View
	Leader   func() error
	Create   func(domain.Task) error
	Complete func(domain.Task, domain.Receipt) error
	Fail     func(domain.Task) error
}

func (c *Coordinator) Run(ctx context.Context) {
	if c.Client == nil {
		c.Client = &http.Client{Timeout: 30 * time.Second}
	}
	if c.Interval <= 0 {
		c.Interval = 2 * time.Second
	}
	if c.Grace <= 0 {
		c.Grace = 6 * time.Second
	}
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	missing := map[string]time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.Leader() != nil {
				missing = map[string]time.Time{}
				continue
			}
			c.reconcile(ctx, missing)
		}
	}
}
func (c *Coordinator) inventory(ctx context.Context, p domain.Peer) (domain.PeerReport, error) {
	var out domain.PeerReport
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(probe, "GET", p.URL+"/api/v1/inventory", nil)
	res, err := c.Client.Do(req)
	if err != nil {
		return out, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&out)
	if err != nil || res.StatusCode != 200 || out.PeerID != p.ID || out.QuotaBytes <= 0 || out.StoredBytes < 0 {
		return out, fmt.Errorf("invalid inventory from %s", p.ID)
	}
	return out, nil
}
func (c *Coordinator) reconcile(ctx context.Context, missing map[string]time.Time) {
	state := c.Snapshot()
	ids := make([]string, 0, len(state.Tasks))
	for id := range state.Tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		t := state.Tasks[id]
		if t.State == "pending" {
			if c.Leader() != nil {
				return
			}
			receipt, err := c.copy(ctx, t)
			if err != nil {
				_ = c.Fail(t)
			} else {
				_ = c.Complete(t, receipt)
			}
			return
		}
	}
	reports := map[string]domain.PeerReport{}
	for _, p := range c.Peers {
		report, err := c.inventory(ctx, p)
		if err == nil {
			reports[p.ID] = report
		}
	}
	if c.Leader() != nil {
		return
	}
	videos := make([]string, 0, len(state.Videos))
	for id := range state.Videos {
		videos = append(videos, id)
	}
	sort.Strings(videos)
	for _, id := range videos {
		v := state.Videos[id]
		if v.Status != "READY" {
			continue
		}
		for _, seg := range v.Manifest.Segments {
			sources := []domain.Peer{}
			candidates := []domain.PeerObservation{}
			for _, p := range c.Peers {
				report, online := reports[p.ID]
				if !online {
					continue
				}
				valid := false
				for _, a := range report.Receipts {
					if a.PeerID == p.ID && a.SegmentID == seg.ID && a.Size == seg.Size && a.SHA256 == seg.SHA256 && a.Version == 1 {
						valid = true
					}
				}
				_, known := state.Replicas[seg.ID][p.ID]
				if valid {
					if known {
						sources = append(sources, p)
					}
					continue
				}
				// Missing committed data might be damaged: retain it instead of overwriting.
				if !known {
					candidates = append(candidates, domain.PeerObservation{Peer: p, FreeBytes: report.QuotaBytes - report.StoredBytes, UploadBytesPerSecond: report.UploadBytesPerSecond, OnlineFraction: 1})
				}
			}
			if len(sources) >= 2 {
				delete(missing, seg.ID)
				continue
			}
			if _, ok := missing[seg.ID]; !ok {
				missing[seg.ID] = time.Now()
				continue
			}
			if len(sources) == 0 || time.Since(missing[seg.ID]) < c.Grace {
				continue
			}
			t, err := (repair.MinimumTwo{}).Plan(repair.Snapshot{Segment: seg, Reachable: sources, Candidates: candidates})
			if err != nil {
				continue
			}
			t.ID = fmt.Sprintf("repair-%x", sha256.Sum256([]byte(fmt.Sprintf("%s/%s/%d", seg.ID, t.Target.ID, len(state.Tasks)))))
			_ = c.Create(t)
			return
		}
	}
}
func (c *Coordinator) copy(ctx context.Context, t domain.Task) (domain.Receipt, error) {
	var receipt domain.Receipt
	transfer, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(transfer, "GET", t.Source.URL+"/api/v1/segments/"+t.SegmentID, nil)
	res, err := c.Client.Do(req)
	if err != nil {
		return receipt, err
	}
	data, readErr := io.ReadAll(io.LimitReader(res.Body, t.Size+1))
	res.Body.Close()
	if readErr != nil || res.StatusCode != 200 || int64(len(data)) != t.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != t.SHA256 {
		return receipt, fmt.Errorf("repair source verification failed")
	}
	// Recheck quorum. An old leader can install immutable extra bytes but cannot
	// publish without quorum, overwrite a different object, or delete anything.
	if err = c.Leader(); err != nil {
		return receipt, err
	}
	req, _ = http.NewRequestWithContext(transfer, "PUT", t.Target.URL+"/api/v1/segments/"+t.SegmentID, bytes.NewReader(data))
	req.Header.Set("X-Operation-ID", t.ID)
	req.Header.Set("X-Size-Bytes", fmt.Sprint(t.Size))
	req.Header.Set("X-SHA256", t.SHA256)
	req.Header.Set("X-Representation-Version", "1")
	res, err = c.Client.Do(req)
	if err != nil {
		return receipt, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&receipt)
	if err != nil || res.StatusCode != 200 || receipt.OperationID != t.ID || receipt.PeerID != t.Target.ID || receipt.SegmentID != t.SegmentID || receipt.Size != t.Size || receipt.SHA256 != t.SHA256 || receipt.Version != 1 {
		return receipt, fmt.Errorf("repair target acknowledgement mismatch")
	}
	return receipt, nil
}
