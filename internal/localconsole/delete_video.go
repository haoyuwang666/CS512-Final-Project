// TODO(Haoyu): Move deletion reconciliation to a background worker for large libraries.
package localconsole

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (m *Manager) videoDeletionRequest(id, method string) (map[string]any, error) {
	m.mu.Lock()
	nodes := append([]*node(nil), m.nodes...)
	m.mu.Unlock()
	client := &http.Client{Timeout: 3 * time.Second}
	for _, n := range nodes {
		if n.Kind != "controller" {
			continue
		}
		req, _ := http.NewRequestWithContext(m.ctx, method, fmt.Sprintf("http://127.0.0.1:%d/api/v1/videos/%s/delete", n.Port, id), nil)
		res, err := client.Do(req)
		if err != nil {
			continue
		}
		var out map[string]any
		err = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&out)
		res.Body.Close()
		if err != nil || res.StatusCode == 503 {
			continue
		}
		if res.StatusCode >= 400 {
			return nil, fmt.Errorf("删除等待中：%v", out["error"])
		}
		return out, nil
	}
	return nil, fmt.Errorf("等待 Controller 恢复")
}
func (m *Manager) deleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.ValidID(id) {
		fail(w, 400, fmt.Errorf("invalid video ID"))
		return
	}
	m.mu.Lock()
	dir, busy := m.dir, m.busy
	m.mu.Unlock()
	if dir == "" || busy {
		fail(w, 409, fmt.Errorf("请等待上传结束并启动集群"))
		return
	}
	work := filepath.Join(dir, "videos", id)
	if _, err := os.Stat(filepath.Join(work, "ready.json")); err != nil {
		fail(w, 404, fmt.Errorf("视频不在当前视频库中"))
		return
	}
	// Persist user intent before contacting Raft: ambiguous HTTP outcomes remain
	// recoverable across console restarts without losing local source files early.
	file, err := os.OpenFile(filepath.Join(work, "delete-pending"), os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		err = file.Sync()
		file.Close()
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	d, err := os.Open(work)
	if err == nil {
		err = d.Sync()
		d.Close()
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	_, err = m.videoDeletionRequest(id, "POST")
	message := "删除已提交"
	if err != nil {
		message = err.Error()
	}
	protocol.JSON(w, 202, map[string]any{"state": "DELETING", "message": message})
}
func (m *Manager) reconcileVideoDeletion(dir string, j *Job) bool {
	work := filepath.Join(dir, "videos", j.ID)
	if _, err := os.Stat(filepath.Join(work, "delete-pending")); err != nil {
		return false
	}
	j.Stage = "deleting"
	status, err := m.videoDeletionRequest(j.ID, "GET")
	if err != nil {
		j.Error = err.Error()
		return false
	}
	switch status["state"] {
	case "DELETED":
		if err = os.RemoveAll(work); err != nil {
			j.Error = err.Error()
			return false
		}
		m.mu.Lock()
		if m.job != nil && m.job.ID == j.ID {
			m.job = nil
		}
		m.mu.Unlock()
		return true
	case "DELETING":
		j.Error = fmt.Sprintf("已删除 %v / %v 份副本；离线节点恢复后继续", status["completed"], status["total"])
	default:
		_, err = m.videoDeletionRequest(j.ID, "POST")
		if err != nil {
			j.Error = err.Error()
		}
	}
	return false
}
