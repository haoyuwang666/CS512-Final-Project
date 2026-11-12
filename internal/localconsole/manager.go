// TODO(Haoyu): Extend local process supervision without changing distributed storage protocols.
package localconsole

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"cooperative.video/internal/protocol"
)

type node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Port int    `json:"port"`
	cmd  *exec.Cmd
	done chan struct{}
}
type Job struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Stage    string `json:"stage"`
	Error    string `json:"error,omitempty"`
	Segments int    `json:"segments"`
	Log      string `json:"log,omitempty"`
}
type Manager struct {
	mu                 sync.Mutex
	ctx                context.Context
	cancel             context.CancelFunc
	nodes              []*node
	job                *Job
	busy               bool
	peers, controllers int
	dir                string
	client             *http.Client
}

func New(parent context.Context) *Manager {
	ctx, cancel := context.WithCancel(parent)
	return &Manager{ctx: ctx, cancel: cancel, client: &http.Client{Timeout: 700 * time.Millisecond}}
}
func (m *Manager) Close() { m.cancel(); m.mu.Lock(); defer m.mu.Unlock(); m.stopLocked() }
func (m *Manager) stopLocked() {
	for i := len(m.nodes) - 1; i >= 0; i-- {
		n := m.nodes[i]
		select {
		case <-n.done:
			continue
		default:
		}
		_ = n.cmd.Process.Signal(os.Interrupt)
		select {
		case <-n.done:
		case <-time.After(3 * time.Second):
			_ = n.cmd.Process.Kill()
			<-n.done
		}
	}
	m.nodes = nil
}
func fail(w http.ResponseWriter, status int, err error) {
	protocol.Fail(w, status, "LOCAL_CONSOLE", err.Error())
}
func (m *Manager) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/local/status", m.status)
	mux.HandleFunc("POST /api/local/start", m.start)
	mux.HandleFunc("POST /api/local/stop", m.stop)
	mux.HandleFunc("POST /api/local/upload", m.upload)
	mux.HandleFunc("GET /api/local/videos", m.videos)
	target, _ := url.Parse("http://127.0.0.1:18080")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fail(w, 503, fmt.Errorf("请先启动集群：%v", err))
	}
	mux.Handle("/api/", proxy)
	mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A browser on another website must not be able to control local processes.
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil || (host != "localhost" && host != "127.0.0.1") {
			fail(w, 403, fmt.Errorf("仅允许本机访问"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, e := url.Parse(origin)
			if e != nil || u.Host != r.Host {
				fail(w, 403, fmt.Errorf("不允许跨来源操作"))
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}
func (m *Manager) launch(id, kind string, port int, env map[string]string) error {
	logFile, err := os.Create(filepath.Join(m.dir, id+".log"))
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(m.ctx, filepath.Join("bin", kind))
	commandGroup(cmd)
	// Remove inherited node settings so each generated configuration is authoritative.
	values := map[string]string{}
	for _, entry := range os.Environ() {
		k, v, _ := strings.Cut(entry, "=")
		values[k] = v
	}
	for _, key := range []string{"HTTP_ADDR", "DATA_DIR", "PEERS", "NODE_ID", "RAFT_BIND", "RAFT_ADVERTISE", "RAFT_MEMBERS", "BOOTSTRAP", "CONTROLLERS", "SESSION_DIR"} {
		delete(values, key)
	}
	for k, v := range env {
		values[k] = v
	}
	for k, v := range values {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err = cmd.Start(); err != nil {
		logFile.Close()
		return err
	}
	n := &node{ID: id, Kind: kind, Port: port, cmd: cmd, done: make(chan struct{})}
	m.nodes = append(m.nodes, n)
	go func() { _ = cmd.Wait(); _ = logFile.Close(); close(n.done) }()
	return nil
}
func (m *Manager) start(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Peers       int `json:"peers"`
		Controllers int `json:"controllers"`
	}
	if err := protocol.Decode(w, r, &body); err != nil {
		fail(w, 400, err)
		return
	}
	if body.Peers < 2 || body.Peers > 8 || (body.Controllers != 1 && body.Controllers != 3) {
		fail(w, 400, fmt.Errorf("Peer 数量须为 2–8；Controller 数量为 1 或 3"))
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy || len(m.nodes) > 0 {
		fail(w, 409, fmt.Errorf("请等待上传结束并停止现有集群后再启动"))
		return
	}
	ports := []int{18080}
	for i := 0; i < body.Peers; i++ {
		ports = append(ports, 19001+i)
	}
	for i := 0; i < body.Controllers; i++ {
		ports = append(ports, 18001+i, 17001+i)
	}
	for _, port := range ports {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			fail(w, 409, fmt.Errorf("端口 %d 已被占用；未终止任何已有服务", port))
			return
		}
		l.Close()
	}
	m.peers = body.Peers
	m.controllers = body.Controllers
	m.dir = fmt.Sprintf(".data/console/p%d-c%d", body.Peers, body.Controllers)
	if err := os.MkdirAll(m.dir, 0700); err != nil {
		fail(w, 500, err)
		return
	}
	peers := []map[string]string{}
	members := []map[string]string{}
	urls := []string{}
	for i := 1; i <= body.Peers; i++ {
		peers = append(peers, map[string]string{"peer_id": fmt.Sprintf("peer-%d", i), "url": fmt.Sprintf("http://127.0.0.1:%d", 19000+i)})
	}
	for i := 1; i <= body.Controllers; i++ {
		u := fmt.Sprintf("http://127.0.0.1:%d", 18000+i)
		urls = append(urls, u)
		members = append(members, map[string]string{"id": fmt.Sprintf("controller-%d", i), "raft_address": fmt.Sprintf("127.0.0.1:%d", 17000+i), "http_url": u})
	}
	p, _ := json.Marshal(peers)
	c, _ := json.Marshal(members)
	launch := func(id, kind string, port int, env map[string]string) bool {
		if err := m.launch(id, kind, port, env); err != nil {
			m.stopLocked()
			fail(w, 500, err)
			return false
		}
		return true
	}
	for i := 1; i <= body.Peers; i++ {
		id := fmt.Sprintf("peer-%d", i)
		port := 19000 + i
		if !launch(id, "peer", port, map[string]string{"HTTP_ADDR": fmt.Sprintf("127.0.0.1:%d", port), "PEER_ID": id, "DATA_DIR": filepath.Join(m.dir, id)}) {
			return
		}
	}
	for i := 1; i <= body.Controllers; i++ {
		id := fmt.Sprintf("controller-%d", i)
		port := 18000 + i
		if !launch(id, "controller", port, map[string]string{"HTTP_ADDR": fmt.Sprintf("127.0.0.1:%d", port), "NODE_ID": id, "RAFT_BIND": fmt.Sprintf("127.0.0.1:%d", 17000+i), "RAFT_ADVERTISE": fmt.Sprintf("127.0.0.1:%d", 17000+i), "RAFT_MEMBERS": string(c), "BOOTSTRAP": strconv.FormatBool(i == 1), "PEERS": string(p), "DATA_DIR": filepath.Join(m.dir, id)}) {
			return
		}
	}
	if !launch("demo-api", "demo-api", 18080, map[string]string{"HTTP_ADDR": "127.0.0.1:18080", "CONTROLLERS": strings.Join(urls, ","), "SESSION_DIR": filepath.Join(m.dir, "sessions")}) {
		return
	}
	protocol.JSON(w, 202, map[string]string{"status": "starting"})
}
func (m *Manager) stop(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy {
		fail(w, 409, fmt.Errorf("视频正在处理，请完成后再停止集群"))
		return
	}
	m.stopLocked()
	protocol.JSON(w, 200, map[string]bool{"ok": true})
}
func (m *Manager) status(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	nodes := append([]*node(nil), m.nodes...)
	busy := m.busy
	var job *Job
	if m.job != nil {
		v := *m.job
		job = &v
	}
	m.mu.Unlock()
	rows := []map[string]any{}
	for _, n := range nodes {
		row := map[string]any{"id": n.ID, "kind": n.Kind, "port": n.Port, "pid": n.cmd.Process.Pid, "status": "starting"}
		select {
		case <-n.done:
			row["status"] = "exited"
		default:
			path := "/api/v1/health"
			if n.Kind == "peer" {
				path = "/api/v1/stats"
			}
			if n.Kind == "controller" {
				path = "/api/v1/cluster"
			}
			res, err := m.client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", n.Port, path))
			if err == nil {
				if res.StatusCode == 200 {
					var data map[string]any
					if json.NewDecoder(res.Body).Decode(&data) == nil {
						row["details"] = data
						row["status"] = "running"
					}
				}
				res.Body.Close()
			}
		}
		rows = append(rows, row)
	}
	protocol.JSON(w, 200, map[string]any{"nodes": rows, "job": job, "busy": busy})
}
func (m *Manager) stage(stage, log string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.job.Stage = stage
	if len(log) > 4000 {
		log = log[len(log)-4000:]
	}
	m.job.Log = log
}
func (m *Manager) run(timeout time.Duration, command string, args ...string) error {
	ctx, cancel := context.WithTimeout(m.ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	commandGroup(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", command, err, string(out))
	}
	return nil
}

// Each job owns a process group, including FFmpeg children launched by bash.
func commandGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
}
func (m *Manager) upload(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	if m.busy || len(m.nodes) == 0 {
		m.mu.Unlock()
		fail(w, 409, fmt.Errorf("请先启动集群，并等待当前任务完成"))
		return
	}
	m.busy = true
	dir := m.dir
	count := m.controllers
	m.mu.Unlock()
	accepted := false
	defer func() {
		if !accepted {
			m.mu.Lock()
			m.busy = false
			m.mu.Unlock()
		}
	}()
	r.Body = http.MaxBytesReader(w, r.Body, 512<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		fail(w, 400, fmt.Errorf("上传失败（最大 512 MB）：%w", err))
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("video")
	if err != nil {
		fail(w, 400, err)
		return
	}
	defer file.Close()
	start, duration := 0.0, 0.0
	for key, value := range map[string]*float64{"start": &start, "duration": &duration} {
		raw := r.FormValue(key)
		if raw != "" {
			v, e := strconv.ParseFloat(raw, 64)
			if e != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 86400 {
				fail(w, 400, fmt.Errorf("裁剪参数须为 0–86400 秒"))
				return
			}
			*value = v
		}
	}
	token := make([]byte, 8)
	if _, err = rand.Read(token); err != nil {
		fail(w, 500, err)
		return
	}
	id := "video-" + hex.EncodeToString(token)
	work := filepath.Join(dir, "videos", id)
	if err = os.MkdirAll(work, 0700); err != nil {
		fail(w, 500, err)
		return
	}
	path := filepath.Join(work, "source.mp4")
	dest, err := os.Create(path)
	if err != nil {
		fail(w, 500, err)
		return
	}
	_, err = io.Copy(dest, file)
	closeErr := dest.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	m.mu.Lock()
	m.job = &Job{ID: id, Name: filepath.Base(header.Filename), Stage: "uploaded"}
	m.mu.Unlock()
	accepted = true
	go m.process(work, path, id, count, start, duration)
	protocol.JSON(w, 202, map[string]string{"video_id": id})
}
func (m *Manager) process(work, path, id string, count int, start, duration float64) {
	defer func() { m.mu.Lock(); m.busy = false; m.mu.Unlock() }()
	process := func() error {
		if start > 0 || duration > 0 {
			m.stage("trimming", "")
			trim := filepath.Join(work, "trimmed.mp4")
			args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", path, "-ss", strconv.FormatFloat(start, 'f', 3, 64)}
			if duration > 0 {
				args = append(args, "-t", strconv.FormatFloat(duration, 'f', 3, 64))
			}
			args = append(args, "-map", "0:v:0", "-map", "0:a:0?", "-c:v", "libx264", "-c:a", "aac", trim)
			if err := m.run(30*time.Minute, "ffmpeg", args...); err != nil {
				return err
			}
			path = trim
		}
		segments := filepath.Join(work, "segments")
		m.stage("segmenting", "")
		if err := m.run(30*time.Minute, "bash", "scripts/media/segment_video.sh", path, segments); err != nil {
			return err
		}
		m.stage("manifest", "")
		if err := m.run(10*time.Minute, "python3", "scripts/media/generate_manifest.py", segments, "--video-id", id); err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(segments, "manifest.json"))
		if err != nil {
			return err
		}
		var manifest struct {
			Segments []json.RawMessage `json:"segments"`
		}
		if err = json.Unmarshal(data, &manifest); err != nil {
			return err
		}
		m.mu.Lock()
		m.job.Segments = len(manifest.Segments)
		m.mu.Unlock()
		m.stage("importing", "")
		urls := []string{}
		for i := 1; i <= count; i++ {
			urls = append(urls, fmt.Sprintf("http://127.0.0.1:%d", 18000+i))
		}
		if err = m.run(30*time.Minute, "python3", "scripts/ingest/ingest.py", filepath.Join(segments, "manifest.json"), "--controllers", strings.Join(urls, ",")); err != nil {
			return err
		}
		m.mu.Lock()
		name := m.job.Name
		m.mu.Unlock()
		meta, _ := json.Marshal(Job{ID: id, Name: name, Stage: "ready", Segments: len(manifest.Segments)})
		if err = os.WriteFile(filepath.Join(work, "ready.json"), meta, 0600); err != nil {
			return err
		}
		m.stage("ready", "")
		return nil
	}
	if err := process(); err != nil {
		m.mu.Lock()
		m.job.Stage = "failed"
		m.job.Error = err.Error()
		m.mu.Unlock()
	}
}
func (m *Manager) videos(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	dir := m.dir
	m.mu.Unlock()
	out := []Job{}
	if dir != "" {
		paths, _ := filepath.Glob(filepath.Join(dir, "videos", "*", "ready.json"))
		for _, p := range paths {
			b, e := os.ReadFile(p)
			var j Job
			if e == nil && json.Unmarshal(b, &j) == nil {
				out = append(out, j)
			}
		}
	}
	protocol.JSON(w, 200, out)
}
