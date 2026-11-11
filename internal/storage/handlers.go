// TODO(Yuquan): Add authenticated task execution/telemetry; upload bytes are counted at ResponseWriter, not TCP-wire level.
package storage

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"golang.org/x/time/rate"
)

type Server struct {
	Store      *Store
	PeerID     string
	Limiter    *rate.Limiter
	Uploaded   atomic.Int64
	InstanceID string
}

func NewServer(store *Store, peer string, bytesPerSecond float64) *Server {
	return &Server{Store: store, PeerID: peer, Limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), 32<<10), InstanceID: strconv.FormatInt(time.Now().UnixNano(), 10)}
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/health", protocol.Health)
	m.HandleFunc("PUT /api/v1/segments/{id}", s.put)
	m.HandleFunc("GET /api/v1/segments/{id}", s.get)
	m.HandleFunc("GET /api/v1/segments/{id}/receipt", s.receipt)
	m.HandleFunc("GET /api/v1/stats", s.stats)
	return m
}
func (s *Server) put(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.ParseInt(r.Header.Get("X-Size-Bytes"), 10, 64)
	if err != nil {
		protocol.Fail(w, 400, "INVALID_ARGUMENT", "invalid size")
		return
	}
	version, _ := strconv.Atoi(r.Header.Get("X-Representation-Version"))
	a := domain.Receipt{OperationID: r.Header.Get("X-Operation-ID"), PeerID: s.PeerID, SegmentID: r.PathValue("id"), Version: version, Size: size, SHA256: r.Header.Get("X-SHA256")}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20+1)
	defer r.Body.Close()
	a, err = s.Store.Put(a, r.Body)
	if err != nil {
		status, code := 400, "INVALID_ARGUMENT"
		if errors.Is(err, ErrCapacity) {
			status, code = 507, "CAPACITY_EXCEEDED"
		}
		if errors.Is(err, ErrHash) {
			code = "HASH_MISMATCH"
		}
		if errors.Is(err, ErrConflict) {
			status, code = 409, "VERSION_CONFLICT"
		}
		protocol.Fail(w, status, code, err.Error())
		return
	}
	protocol.JSON(w, 200, a)
}
func (s *Server) receipt(w http.ResponseWriter, r *http.Request) {
	a, _, ok := s.Store.Lookup(r.PathValue("id"))
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "object not found")
		return
	}
	protocol.JSON(w, 200, a)
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	a, path, ok := s.Store.Lookup(r.PathValue("id"))
	if !ok {
		protocol.Fail(w, 404, "NOT_FOUND", "object not found")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		protocol.Fail(w, 500, "STORAGE_ERROR", err.Error())
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(a.Size, 10))
	w.Header().Set("X-SHA256", a.SHA256)
	buf := make([]byte, 32<<10)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if e := s.Limiter.WaitN(r.Context(), n); e != nil {
				return
			}
			sent, e := w.Write(buf[:n])
			s.Uploaded.Add(int64(sent))
			if e != nil {
				return
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}
	}
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	used, quota, count := s.Store.Stats()
	protocol.JSON(w, 200, map[string]any{"peer_id": s.PeerID, "instance_id": s.InstanceID, "stored_bytes": used, "quota_bytes": quota, "objects": count, "uploaded_bytes": s.Uploaded.Load(), "upload_bytes_per_second": float64(s.Limiter.Limit())})
}
