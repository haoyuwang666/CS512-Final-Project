// TODO(Yuquan): Expand cleanup tests with injected fsync failures.
package storage

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestEquivalentPutKeepsReceiptAndRetiresEveryAlias(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("shared")
	a := domain.Receipt{OperationID: "first", PeerID: "p", SegmentID: "seg", Version: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	b := a
	b.OperationID = "retry-alias"
	for _, receipt := range []domain.Receipt{a, b, a} {
		got, err := s.Put(receipt, bytes.NewReader(data))
		if err != nil || got != receipt {
			t.Fatalf("PUT %v: %v %v", receipt, got, err)
		}
	}
	if got, _, _ := s.Lookup(a.SegmentID); got != a {
		t.Fatal("installed receipt changed", got)
	}
	// Metadata may have confirmed an equivalent PUT alias rather than the first.
	if err = s.DeleteReplica("delete", b); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range []domain.Receipt{a, b} {
		if _, err := s.Put(old, bytes.NewReader(data)); err != ErrConflict {
			t.Fatal("retired alias revived", old, err)
		}
	}
	// A new repair is allowed, but replaying a completed delete cannot remove it.
	c := a
	c.OperationID = "new-repair"
	if _, err = s.Put(c, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteReplica("delete", b); err != nil {
		t.Fatal(err)
	}
	if got, _, ok := s.Lookup(c.SegmentID); !ok || got != c || !s.Verified(c.SegmentID) {
		t.Fatal("old deletion removed new replica", got)
	}
	if _, err = s.Put(a, bytes.NewReader(data)); err != ErrConflict {
		t.Fatal("old alias replaced new replica", err)
	}
}

func TestLegacyReceiptAliasCanBeDeletedAndRetriedAfterRestart(t *testing.T) {
	dir := t.TempDir()
	data := []byte("legacy")
	a := domain.Receipt{OperationID: "original", PeerID: "p", SegmentID: "seg", Version: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	b := a
	b.OperationID = "legacy-overwrite"
	// Old Peers replaced Objects on every equivalent PUT, but Raft could still
	// contain the original receipt. Exercise a real persisted pre-upgrade index.
	raw, _ := json.Marshal(diskIndex{Objects: map[string]domain.Receipt{"seg": b}, Operations: map[string]domain.Receipt{a.OperationID: a, b.OperationID: b}})
	if err := os.WriteFile(filepath.Join(dir, "index.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, a.SHA256+".blob"), data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteReplica("legacy-delete", a); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteReplica("legacy-delete", a); err != nil {
		t.Fatal(err)
	}
	for _, old := range []domain.Receipt{a, b} {
		if _, err = s.Put(old, bytes.NewReader(data)); err != ErrConflict {
			t.Fatal("legacy alias revived", err)
		}
	}
	// Migration of a deletion index written before retired_operations existed.
	raw, _ = json.Marshal(diskIndex{Objects: map[string]domain.Receipt{}, Operations: map[string]domain.Receipt{a.OperationID: a, b.OperationID: b}, Deleted: map[string]domain.Receipt{"legacy-delete": a}})
	if err = os.WriteFile(filepath.Join(dir, "index.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put(b, bytes.NewReader(data)); err != ErrConflict {
		t.Fatal("pre-upgrade tombstone lost alias", err)
	}
}
