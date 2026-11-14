package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRepairRule(t *testing.T) {
	cases := []struct {
		p         string
		busy      bool
		idle, age time.Duration
		want      float64
	}{
		{"S", true, 0, 0, 1}, {"F25", false, 9 * time.Second, 0, .25}, {"F50", true, 0, 0, .5},
		{"L", true, 0, 0, .25}, {"L", false, time.Second, 0, .25}, {"L", false, 2 * time.Second, 0, 1},
		{"L", true, 0, 61 * time.Second, .5}, {"L", true, 0, 60 * time.Second, .25},
	}
	for _, c := range cases {
		if got := repairFraction(c.p, c.busy, c.idle, c.age); got != c.want {
			t.Fatalf("%+v: %v", c, got)
		}
	}
}
func TestTransferClassesAndCommonCap(t *testing.T) {
	store, _ := Open(t.TempDir(), "peer-1", 1<<20)
	data := bytes.Repeat([]byte("x"), 96<<10)
	if _, err := store.Put(receipt("s", data), bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	s := NewServer(store, "peer-1", 128<<10)
	if err := s.ConfigureRepairPolicy("F25"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.RunPolicy(ctx)
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	start := time.Now()
	for _, class := range []string{"repair", "foreground", "probe"} {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/segments/s", nil)
		req.Header.Set("X-Transfer-Class", class)
		req.Header.Set("X-Repair-Task-ID", "task")
		req.Header.Set("X-Repair-Created-Unix-MS", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil || !bytes.Equal(got, data) {
			t.Fatal("truncated", err)
		}
	}
	// 96 KiB repair at 32 KiB/s with 32 KiB burst cannot take <2 s.
	// Total payload at 128 KiB/s with the same burst cannot take <2 s.
	if elapsed := time.Since(start); elapsed < 1900*time.Millisecond {
		t.Fatal("rate cap bypassed", elapsed)
	}
	snap := s.trafficSnapshot()
	counts := snap["class_bytes"].(map[string]int64)
	for _, class := range []string{"repair", "foreground", "probe"} {
		if counts[class] != int64(len(data)) {
			t.Fatal(counts)
		}
	}
	if s.Uploaded.Load() != 3*int64(len(data)) {
		t.Fatal("total accounting")
	}
}
func TestCommittedAgeAndWaitingForeground(t *testing.T) {
	s := NewServer(nil, "peer-1", 1024)
	_ = s.ConfigureRepairPolicy("L")
	fg := httptest.NewRequest("GET", "/", nil)
	_, endFG := s.beginTransfer(fg)
	repair := httptest.NewRequest("GET", "/", nil)
	repair.Header.Set("X-Transfer-Class", "repair")
	repair.Header.Set("X-Repair-Task-ID", "stable")
	repair.Header.Set("X-Repair-Created-Unix-MS", "1")
	_, end := s.beginTransfer(repair)
	s.Traffic.mu.Lock()
	s.Traffic.update(time.Now())
	s.Traffic.mu.Unlock()
	if float64(s.Traffic.repairLimiter.Limit()) != 512 {
		t.Fatal("lost committed age")
	}
	end()
	endFG()
}

func TestForegroundCannotBypassSharedLimiter(t *testing.T) {
	store, _ := Open(t.TempDir(), "peer-1", 1<<20)
	data := bytes.Repeat([]byte("z"), 96<<10)
	_, _ = store.Put(receipt("s", data), bytes.NewReader(data))
	s := NewServer(store, "peer-1", 64<<10)
	_ = s.ConfigureRepairPolicy("F25")
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	start := time.Now()
	res, err := http.Get(server.URL + "/api/v1/segments/s")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil || len(got) != len(data) {
		t.Fatal("transfer failed", err)
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Fatal("foreground escaped total cap")
	}
}
