// TODO(Haoyu): Exercise deletion reconciliation with a real browser console.
package localconsole

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestDeletedVideoRemovesLocalFilesOnlyAfterCompletion(t *testing.T) {
	state := "DELETING"
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"state": state, "total": 2, "completed": 1})
	}))
	defer controller.Close()
	u, _ := url.Parse(controller.URL)
	port, _ := strconv.Atoi(u.Port())
	dir := t.TempDir()
	work := filepath.Join(dir, "videos", "video-test")
	os.MkdirAll(work, 0700)
	for _, name := range []string{"ready.json", "delete-pending", "source.mp4"} {
		if err := os.WriteFile(filepath.Join(work, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := &Manager{ctx: context.Background(), dir: dir, nodes: []*node{{Kind: "controller", Port: port}}}
	j := Job{ID: "video-test"}
	if m.reconcileVideoDeletion(dir, &j) {
		t.Fatal("deleted local source before peer completion")
	}
	if _, err := os.Stat(filepath.Join(work, "source.mp4")); err != nil {
		t.Fatal(err)
	}
	state = "DELETED"
	if !m.reconcileVideoDeletion(dir, &j) {
		t.Fatal("did not remove completed video")
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatal("video files remain", err)
	}
}
