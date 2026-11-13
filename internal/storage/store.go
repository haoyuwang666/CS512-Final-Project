// TODO(Yuquan): Extend immutable object storage with inventory and authorized cleanup; never acknowledge before durable install.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"cooperative.video/internal/domain"
)

var ErrCapacity = errors.New("capacity exceeded")
var ErrHash = errors.New("size or SHA-256 mismatch")
var ErrConflict = errors.New("immutable object or operation conflict")
var ErrInvalid = errors.New("invalid receipt fields")

type diskIndex struct {
	Objects    map[string]domain.Receipt `json:"objects"`
	Operations map[string]domain.Receipt `json:"operations"`
}
type Store struct {
	mu          sync.Mutex
	dir, peer   string
	quota, used int64
	index       diskIndex
}

func Open(dir, peer string, quota int64) (*Store, error) {
	if !domain.ValidID(peer) || quota <= 0 {
		return nil, fmt.Errorf("invalid peer or quota")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, peer: peer, quota: quota, index: diskIndex{map[string]domain.Receipt{}, map[string]domain.Receipt{}}}
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err == nil {
		if err = json.Unmarshal(b, &s.index); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.index.Objects == nil || s.index.Operations == nil {
		return nil, fmt.Errorf("invalid disk index")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.blob"))
	if err != nil {
		return nil, err
	}
	for _, name := range files {
		info, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		s.used += info.Size()
	}
	// Unindexed blobs count toward quota after crashes and are retained conservatively.
	return s, nil
}
func (s *Store) Put(a domain.Receipt, reader io.Reader) (domain.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash, hashErr := hex.DecodeString(a.SHA256)
	if !domain.ValidID(a.SegmentID) || !domain.ValidID(a.OperationID) || a.PeerID != s.peer || a.Version != 1 || a.Size <= 0 || a.Size > 64<<20 || hashErr != nil || len(hash) != 32 || fmt.Sprintf("%x", hash) != a.SHA256 {
		return a, ErrInvalid
	}
	if old, ok := s.index.Operations[a.OperationID]; ok && old != a {
		return a, ErrConflict
	}
	if old, ok := s.index.Objects[a.SegmentID]; ok && (old.SHA256 != a.SHA256 || old.Size != a.Size || old.Version != a.Version) {
		return a, ErrConflict
	}
	path := filepath.Join(s.dir, a.SHA256+".blob")
	_, err := os.Stat(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return a, err
	}
	// Never acknowledge an existing damaged blob as a valid idempotent retry.
	if exists {
		f, e := os.Open(path)
		if e != nil {
			return a, e
		}
		h := sha256.New()
		n, e := io.Copy(h, f)
		f.Close()
		if e != nil || n != a.Size || fmt.Sprintf("%x", h.Sum(nil)) != a.SHA256 {
			return a, ErrHash
		}
	}
	if !exists && s.used+a.Size > s.quota {
		return a, ErrCapacity
	}
	tmp, err := os.CreateTemp(s.dir, "upload-*")
	if err != nil {
		return a, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(reader, a.Size+1))
	if err != nil {
		return a, err
	}
	if n != a.Size || fmt.Sprintf("%x", h.Sum(nil)) != a.SHA256 {
		return a, ErrHash
	}
	if err = tmp.Sync(); err != nil {
		return a, err
	}
	if err = tmp.Close(); err != nil {
		return a, err
	}
	if !exists {
		if err = os.Rename(tmp.Name(), path); err != nil {
			return a, err
		}
		s.used += a.Size
		if err = syncDir(s.dir); err != nil {
			return a, err
		}
	}
	// Persist a candidate index before exposing the new object/operation in memory.
	b, _ := json.Marshal(s.index)
	var next diskIndex
	_ = json.Unmarshal(b, &next)
	next.Objects[a.SegmentID] = a
	next.Operations[a.OperationID] = a
	b, err = json.Marshal(next)
	if err != nil {
		return a, err
	}
	if err = atomicWrite(s.dir, "index.json", b); err != nil {
		return a, err
	}
	s.index = next
	return a, nil
}
func (s *Store) Lookup(id string) (domain.Receipt, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.index.Objects[id]
	return a, filepath.Join(s.dir, a.SHA256+".blob"), ok
}
func (s *Store) Stats() (int64, int64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.used, s.quota, len(s.index.Objects)
}

// Inventory returns only objects that still match their durable receipts.
// Corrupt/orphan objects are retained and continue to consume quota.
func (s *Store) Inventory() []domain.Receipt {
	s.mu.Lock()
	items := make([]domain.Receipt, 0, len(s.index.Objects))
	for _, a := range s.index.Objects {
		items = append(items, a)
	}
	s.mu.Unlock()
	out := []domain.Receipt{}
	for _, a := range items {
		f, err := os.Open(filepath.Join(s.dir, a.SHA256+".blob"))
		if err != nil {
			continue
		}
		h := sha256.New()
		n, err := io.Copy(h, f)
		f.Close()
		if err == nil && n == a.Size && fmt.Sprintf("%x", h.Sum(nil)) == a.SHA256 {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SegmentID < out[j].SegmentID })
	return out
}

func (s *Store) Verified(id string) bool {
	a, path, ok := s.Lookup(id)
	if !ok {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return err == nil && n == a.Size && fmt.Sprintf("%x", h.Sum(nil)) == a.SHA256
}
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func atomicWrite(dir, name string, b []byte) error {
	f, err := os.CreateTemp(dir, "index-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(b); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(dir, name)); err != nil {
		return err
	}
	return syncDir(dir)
}
