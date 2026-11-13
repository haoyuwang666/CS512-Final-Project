// TODO(Yuquan): Add long-running concurrency tests for simultaneous browser sessions.
package localconsole

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFaultRoutesRejectUnknownBusyAndCrossOrigin(t *testing.T) {
	m := New(context.Background())
	defer m.Close()
	h := m.Handler()
	for _, tc := range []struct {
		path, body, origin string
		busy               bool
		code               int
	}{
		{"/api/local/nodes/unowned/stop", "{}", "", false, 404},
		{"/api/local/nodes/unowned/remove", "{}", "", false, 400},
		{"/api/local/nodes/unowned/stop", "{}", "https://foreign.example", false, 403},
		{"/api/local/nodes/unowned/stop", "{}", "", true, 409},
		{"/api/local/nodes/unowned/stop", "broken", "", false, 400},
	} {
		m.busy = tc.busy
		r := httptest.NewRequest("POST", "http://127.0.0.1:8090"+tc.path, strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/json")
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%s status=%d body=%s", tc.path, w.Code, w.Body.String())
		}
	}
	m.busy = false
	r := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/local/maintenance", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out struct {
		Available bool         `json:"available"`
		Events    []FaultEvent `json:"events"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Available || out.Events == nil {
		t.Fatal(w.Body.String())
	}
}
