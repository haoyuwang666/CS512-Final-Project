// TODO(Yuquan): Expand cleanup tests with injected fsync failures.
package storage

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestCleanupSharedBlobRestartAndStaleUpload(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("shared")
	a := domain.Receipt{OperationID: "a", PeerID: "p", SegmentID: "a", Version: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	b := a
	b.OperationID = "b"
	b.SegmentID = "b"
	for _, r := range []domain.Receipt{a, b} {
		if _, err = s.Put(r, bytes.NewReader(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.DeleteReplica("delete-a", a); err != nil {
		t.Fatal(err)
	}
	if !s.Verified("b") {
		t.Fatal("shared blob lost")
	}
	s, err = Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put(a, bytes.NewReader(data)); err != ErrConflict {
		t.Fatal("stale upload revived", err)
	}
	if err = s.DeleteReplica("delete-a", a); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteReplica("delete-b", b); err != nil {
		t.Fatal(err)
	}
	used, _, count := s.Stats()
	if used != 0 || count != 0 {
		t.Fatal(used, count)
	}
	if err = s.DeleteReplica("delete-b", b); err != nil {
		t.Fatal(err)
	}
}
