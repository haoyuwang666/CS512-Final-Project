// TODO(Tian): Extend media gateway tests when new representations are introduced.
package playback

import (
	"context"
	"cooperative.video/internal/domain"
	"cooperative.video/internal/protocol"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mediaFixture(t *testing.T, representation string, valid bool) (*MediaSessions, []byte) {
	t.Helper()
	data := []byte("verified MP4 bytes")
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("corrupt")) }))
	t.Cleanup(bad.Close)
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if valid {
			w.Write(data)
		} else {
			w.Write([]byte("corrupt"))
		}
	}))
	t.Cleanup(good.Close)
	seg := domain.Segment{ID: "v--0", DurationMS: 2000, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/manifest") {
			protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: representation, Media: &domain.MediaInfo{MIMEType: `video/mp4; codecs="avc1.64000c, mp4a.40.2"`, DurationMS: 2000}, Segments: []domain.Segment{seg}})
			return
		}
		protocol.JSON(w, 200, protocol.Locations{SegmentID: seg.ID, Locations: []domain.Location{{Peer: domain.Peer{ID: "bad", URL: bad.URL}, Version: 1}, {Peer: domain.Peer{ID: "good", URL: good.URL}, Version: 1}}})
	}))
	t.Cleanup(meta.Close)
	return NewMediaSessions(context.Background(), NewEngine([]string{meta.URL})), data
}
func createMedia(t *testing.T, handler http.Handler, body string) (string, int) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("POST", "/api/v1/media/sessions", strings.NewReader(body)))
	var out struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(response.Body.Bytes(), &out)
	return out.SessionID, response.Code
}
func TestMediaGatewayVerifiedFallbackAndStop(t *testing.T) {
	sessions, data := mediaFixture(t, "continuous-fmp4-h264-aac-v1", true)
	handler := sessions.Handler()
	id, status := createMedia(t, handler, `{"video_id":"v","strategy":"deadline"}`)
	if status != 201 {
		t.Fatal(status)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/media/sessions/"+id+"/segments/0", nil))
	if response.Code != 200 || response.Body.String() != string(data) || response.Header().Get("X-Peer-ID") != "good" || response.Header().Get("X-Download-Failures") != "1" {
		t.Fatalf("bad delivery: %d %s", response.Code, response.Body.String())
	}
	request := httptest.NewRequest("GET", "/api/v1/media/sessions/"+id+"/segments/0", nil)
	request.Header.Set("Range", "bytes=0-3")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 206 || response.Body.String() != string(data[:4]) {
		t.Fatalf("bad range delivery: %d", response.Code)
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/media/sessions/"+id+"/stop", nil))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/media/sessions/"+id+"/segments/0", nil))
	if response.Code != 404 {
		t.Fatal("stopped session remained readable")
	}
}
func TestMediaGatewayRejectsFixtureAndInvalidConfig(t *testing.T) {
	sessions, _ := mediaFixture(t, "test-bytes", true)
	if _, status := createMedia(t, sessions.Handler(), `{"video_id":"v"}`); status != 422 {
		t.Fatal(status)
	}
	sessions, _ = mediaFixture(t, "continuous-fmp4-h264-v1", true)
	for _, body := range []string{`{"video_id":"v","strategy":"unknown"}`, `{"video_id":"v","window":1,"concurrency":2}`} {
		if _, status := createMedia(t, sessions.Handler(), body); status != 400 {
			t.Fatal(status)
		}
	}
	id, status := createMedia(t, sessions.Handler(), `{"video_id":"v"}`)
	if status != 201 {
		t.Fatal(status)
	}
	response := httptest.NewRecorder()
	sessions.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/media/sessions/"+id+"/segments/99", nil))
	if response.Code != 400 {
		t.Fatal(response.Code)
	}
}
func TestMediaGatewayNeverForwardsCorruptPayload(t *testing.T) {
	sessions, _ := mediaFixture(t, "continuous-fmp4-h264-v1", false)
	id, status := createMedia(t, sessions.Handler(), `{"video_id":"v"}`)
	if status != 201 {
		t.Fatal(status)
	}
	server := httptest.NewServer(sessions.Handler())
	defer server.Close()
	response, err := http.Get(server.URL + "/api/v1/media/sessions/" + id + "/segments/0")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 502 || !strings.Contains(string(body), "HASH_MISMATCH") {
		t.Fatalf("unverified data forwarded: %d %s", response.StatusCode, body)
	}
}

func TestStoppingMediaSessionCancelsPeerDownload(t *testing.T) {
	started := make(chan struct{})
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer peer.Close()
	seg := domain.Segment{ID: "v--0", DurationMS: 1000, Size: 1, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("x")))}
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/manifest") {
			protocol.JSON(w, 200, domain.Manifest{SchemaVersion: 1, VideoID: "v", Representation: "continuous-fmp4-h264-v1", Media: &domain.MediaInfo{MIMEType: `video/mp4; codecs="avc1.64000c"`, DurationMS: 1000}, Segments: []domain.Segment{seg}})
			return
		}
		protocol.JSON(w, 200, protocol.Locations{SegmentID: seg.ID, Locations: []domain.Location{{Peer: domain.Peer{ID: "p", URL: peer.URL}, Version: 1}}})
	}))
	defer meta.Close()
	sessions := NewMediaSessions(context.Background(), NewEngine([]string{meta.URL}))
	id, status := createMedia(t, sessions.Handler(), `{"video_id":"v"}`)
	if status != 201 {
		t.Fatal(status)
	}
	server := httptest.NewServer(sessions.Handler())
	defer server.Close()
	finished := make(chan int, 1)
	go func() {
		response, err := http.Get(server.URL + "/api/v1/media/sessions/" + id + "/segments/0")
		if err != nil {
			finished <- 0
			return
		}
		response.Body.Close()
		finished <- response.StatusCode
	}()
	<-started
	response, err := http.Post(server.URL+"/api/v1/media/sessions/"+id+"/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if status := <-finished; status != 410 {
		t.Fatalf("stopped request status %d", status)
	}
}

func TestMediaGatewayMarksExhaustedElectionRecoveryRetryable(t *testing.T) {
	sessions, _ := mediaFixture(t, "continuous-fmp4-h264-v1", true)
	id, status := createMedia(t, sessions.Handler(), `{"video_id":"v"}`)
	if status != 201 {
		t.Fatal(status)
	}
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protocol.Fail(w, 503, "NOT_LEADER", "election in progress")
	}))
	defer unavailable.Close()
	sessions.engine.Metadata.URLs = []string{unavailable.URL}
	sessions.engine.Metadata.retryTimeout = 30 * time.Millisecond
	response := httptest.NewRecorder()
	sessions.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/media/sessions/"+id+"/segments/0", nil))
	var body struct {
		Error protocol.APIError `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 503 || body.Error.Code != "METADATA_UNAVAILABLE" || !body.Error.Retryable {
		t.Fatalf("not retryable: %d %s", response.Code, response.Body.String())
	}
	// A failed request must not destroy the media session.
	if _, ok := sessions.get(id); !ok {
		t.Fatal("media session lost during election")
	}
}
