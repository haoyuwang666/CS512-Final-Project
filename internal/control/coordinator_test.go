// TODO(Yuquan): Extend coordinator tests with bandwidth stalls and retry backoff.
package control

import (
	"context"
	"cooperative.video/internal/domain"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCopyRejectsCorruptionAndLostLeadershipBeforePut(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++; w.WriteHeader(500) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("bad")) }))
	defer source.Close()
	task := domain.Task{ID: "t", SegmentID: "s", Version: 1, Size: 3, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("abc"))), Source: domain.Peer{URL: source.URL}, Target: domain.Peer{URL: target.URL}}
	c := Coordinator{Client: http.DefaultClient, Leader: func() error { return nil }}
	if _, err := c.copy(context.Background(), task); err == nil {
		t.Fatal("copied corrupt bytes")
	}
	task.SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte("bad")))
	c.Leader = func() error { return errors.New("no quorum") }
	if _, err := c.copy(context.Background(), task); err == nil {
		t.Fatal("copied without quorum")
	}
	if targetCalls != 0 {
		t.Fatal("unsafe PUT reached target")
	}
}
