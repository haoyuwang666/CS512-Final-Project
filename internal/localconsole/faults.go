// TODO(Yuquan): Extend supervised faults with explicit trace logging and network faults.
package localconsole

import (
	"context"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Only processes started by this manager can be stopped or restarted.
func (m *Manager) peerFault(w http.ResponseWriter, r *http.Request) {
	m.changeNode(w, r, true)
}

type FaultEvent struct {
	Sequence uint64    `json:"sequence"`
	NodeID   string    `json:"node_id"`
	Kind     string    `json:"kind"`
	Action   string    `json:"action"`
	At       time.Time `json:"at"`
}

func (m *Manager) nodeFault(w http.ResponseWriter, r *http.Request) { m.changeNode(w, r, false) }
func (m *Manager) changeNode(w http.ResponseWriter, r *http.Request, peerOnly bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy {
		fail(w, 409, fmt.Errorf("wait for the upload job before fault injection"))
		return
	}
	action := r.PathValue("action")
	if action != "stop" && action != "start" && action != "restart" {
		fail(w, 400, fmt.Errorf("操作必须是 stop、start 或 restart"))
		return
	}
	var body struct {
		ExpectedRole string `json:"expected_role"`
	}
	if err := protocol.Decode(w, r, &body); err != nil {
		fail(w, 400, err)
		return
	}
	for i, n := range m.nodes {
		if n.ID != r.PathValue("id") || (peerOnly && n.Kind != "peer") || (n.Kind != "peer" && n.Kind != "controller" && n.Kind != "demo-api") {
			continue
		}
		if body.ExpectedRole != "" {
			if body.ExpectedRole != "Leader" || n.Kind != "controller" {
				fail(w, 400, fmt.Errorf("无效节点角色"))
				return
			}
			req, _ := http.NewRequestWithContext(r.Context(), "GET", fmt.Sprintf("http://127.0.0.1:%d/api/v1/cluster", n.Port), nil)
			res, err := m.client.Do(req)
			if err != nil {
				fail(w, 409, fmt.Errorf("Leader 状态已变化，请刷新后重试"))
				return
			}
			var details struct {
				State string `json:"state"`
			}
			err = json.NewDecoder(res.Body).Decode(&details)
			res.Body.Close()
			if err != nil || res.StatusCode != 200 || details.State != "Leader" {
				fail(w, 409, fmt.Errorf("该节点已不是 Leader，请刷新后重试"))
				return
			}
		}
		stopped := false
		select {
		case <-n.done:
			stopped = true
		default:
		}
		changed := false
		if (action == "stop" || action == "restart") && !stopped {
			if err := n.cmd.Process.Kill(); err != nil {
				fail(w, 500, err)
				return
			}
			select {
			case <-n.done:
				stopped = true
				changed = true
			case <-time.After(5 * time.Second):
				fail(w, 503, fmt.Errorf("peer has not exited"))
				return
			}
		}
		if (action == "start" || action == "restart") && stopped {
			env := map[string]string{}
			for _, entry := range n.cmd.Env {
				k, v, _ := strings.Cut(entry, "=")
				env[k] = v
			}
			if err := m.launch(n.ID, n.Kind, n.Port, env); err != nil {
				fail(w, 500, err)
				return
			}
			m.nodes = append(m.nodes[:i], m.nodes[i+1:]...)
			changed = true
		}
		if changed {
			m.faultSequence++
			m.faultEvents = append(m.faultEvents, FaultEvent{m.faultSequence, n.ID, n.Kind, action, time.Now().UTC()})
			if len(m.faultEvents) > 100 {
				m.faultEvents = m.faultEvents[len(m.faultEvents)-100:]
			}
		}
		protocol.JSON(w, 200, map[string]any{"node_id": n.ID, "peer_id": n.ID, "action": action, "changed": changed})
		return
	}
	fail(w, 404, fmt.Errorf("该节点不属于当前控制台"))
}

// Management stays reachable even when Raft has no quorum or Demo API is down.
func (m *Manager) maintenance(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	nodes := append([]*node(nil), m.nodes...)
	events := append([]FaultEvent{}, m.faultEvents...)
	m.mu.Unlock()
	result := map[string]any{"available": false, "events": events, "tasks": map[string]any{}, "peers": map[string]any{}, "error": "暂无可读 Leader；请检查 Controller 状态或恢复节点"}
	for _, n := range nodes {
		if n.Kind != "controller" {
			continue
		}
		select {
		case <-n.done:
			continue
		default:
		}
		ctx, cancel := context.WithTimeout(r.Context(), 900*time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://127.0.0.1:%d/api/v1/maintenance", n.Port), nil)
		res, err := m.client.Do(req)
		if err == nil {
			var data map[string]any
			err = json.NewDecoder(res.Body).Decode(&data)
			res.Body.Close()
			if res.StatusCode == 200 && err == nil {
				result["available"] = true
				result["leader_id"] = n.ID
				result["tasks"] = data["tasks"]
				result["peers"] = data["peers"]
				result["error"] = ""
				cancel()
				break
			}
		}
		cancel()
	}
	protocol.JSON(w, 200, result)
}
