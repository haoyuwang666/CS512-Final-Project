// TODO(Yuquan): Add crash-at-each-fsync and concurrent quota tests before implementing cleanup or migrations.
package storage

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
)

func TestStoreIntegrityQuotaAndRestart(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "peer-1", 6)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("abc")
	a := domain.Receipt{OperationID: "put-1", PeerID: "peer-1", SegmentID: "v--0", Version: 1, Size: 3, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	if _, err = s.Put(a, bytes.NewReader([]byte("bad"))); !errors.Is(err, ErrHash) {
		t.Fatal(err)
	}
	if _, _, ok := s.Lookup(a.SegmentID); ok {
		t.Fatal("corrupt data advertised")
	}
	if _, err = s.Put(a, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put(a, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	used, _, _ := s.Stats()
	if used != 3 {
		t.Fatalf("duplicate consumed quota: %d", used)
	}
	bad := a
	bad.SegmentID = "other"
	if _, err = s.Put(bad, bytes.NewReader(data)); !errors.Is(err, ErrConflict) {
		t.Fatal("operation reuse must conflict", err)
	}
	reopened, err := Open(dir, "peer-1", 6)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reopened.Lookup(a.SegmentID); !ok {
		t.Fatal("restart lost object")
	}
	large := []byte("1234")
	a.OperationID = "put-2"
	a.SegmentID = "v--1"
	a.Size = 4
	a.SHA256 = fmt.Sprintf("%x", sha256.Sum256(large))
	if _, err = reopened.Put(a, bytes.NewReader(large)); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
}
