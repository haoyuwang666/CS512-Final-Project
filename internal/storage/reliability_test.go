// TODO(Yuquan): Add subprocess power-loss injection at each persistence boundary.
package storage

import (
	"bytes"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func receipt(id string, data []byte) domain.Receipt {
	return domain.Receipt{OperationID: "put-" + id, PeerID: "peer-1", SegmentID: id, Version: 1, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
}
func TestConcurrentQuotaIsAtomic(t *testing.T) {
	s, err := Open(t.TempDir(), "peer-1", 8)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	outcomes := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("%08d", i))
			_, err := s.Put(receipt(fmt.Sprint(i), data), bytes.NewReader(data))
			outcomes <- err
		}(i)
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrCapacity) {
			t.Fatal(err)
		}
	}
	used, _, count := s.Stats()
	if successes != 1 || used != 8 || count != 1 {
		t.Fatalf("success=%d used=%d count=%d", successes, used, count)
	}
}

type brokenReader struct{ sent bool }

func (r *brokenReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.ErrUnexpectedEOF
	}
	r.sent = true
	copy(p, "a")
	return 1, nil
}
func TestInterruptedUploadDoesNotAdvertiseOrLeakTemp(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir, "peer-1", 64)
	a := receipt("seg", []byte("abc"))
	if _, err := s.Put(a, &brokenReader{}); err == nil {
		t.Fatal("accepted interrupted upload")
	}
	if len(s.Inventory()) != 0 {
		t.Fatal("partial object advertised")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "upload-*"))
	if len(files) != 0 {
		t.Fatal(files)
	}
	used, _, _ := s.Stats()
	if used != 0 {
		t.Fatal(used)
	}
}
func TestCorruptBlobExcludedAndRetryRejected(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir, "peer-1", 64)
	data := []byte("abc")
	a := receipt("seg", data)
	if _, err := s.Put(a, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, a.SHA256+".blob"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if len(s.Inventory()) != 0 {
		t.Fatal("corrupt object in inventory")
	}
	if _, err := s.Put(a, bytes.NewReader(data)); !errors.Is(err, ErrHash) {
		t.Fatal("corrupt idempotent acknowledgement", err)
	}
	reopened, err := Open(dir, "peer-1", 64)
	if err != nil {
		t.Fatal(err)
	}
	used, _, _ := reopened.Stats()
	if used != 3 || len(reopened.Inventory()) != 0 {
		t.Fatal("corrupt bytes must count but not advertise")
	}
}
func TestRestartDownloadAndInvalidHash(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir, "peer-1", 64)
	data := []byte("abc")
	a := receipt("seg", data)
	if _, err := s.Put(a, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, "peer-1", 64)
	if err != nil {
		t.Fatal(err)
	}
	_, path, ok := s.Lookup(a.SegmentID)
	got, err := os.ReadFile(path)
	if !ok || err != nil || !bytes.Equal(data, got) {
		t.Fatal("restart lost bytes", err)
	}
	a.SHA256 = "../../../../../../../../../../../../../../../../../../../../../../../../../../../../../../../../"
	if _, err := s.Put(a, bytes.NewReader(data)); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}

func TestCrashResidueCountsQuotaAndNeverAdvertises(t *testing.T) {
	dir := t.TempDir()
	data := []byte("orphan")
	a := receipt("unknown", data)
	if err := os.WriteFile(filepath.Join(dir, a.SHA256+".blob"), data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, "peer-1", 8)
	if err != nil {
		t.Fatal(err)
	}
	used, _, count := s.Stats()
	if used != 6 || count != 0 || len(s.Inventory()) != 0 {
		t.Fatal("orphan must count without advertising")
	}
	other := []byte("new")
	if _, err := s.Put(receipt("new", other), bytes.NewReader(other)); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, a.SHA256+".blob")); err != nil {
		t.Fatal("must preserve crash residue", err)
	}
}
