// TODO(Tian): Extend election recovery tests alongside controller discovery changes.
package playback

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"cooperative.video/internal/protocol"
)

func TestMetadataWaitsForElectionWithoutCachedLocations(t *testing.T) {
	var requests atomic.Int32
	recovering := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) < 3 {
			protocol.Fail(w, 503, "NOT_LEADER", "election in progress")
			return
		}
		protocol.JSON(w, 200, protocol.Locations{SegmentID: "v--4"})
	}))
	defer recovering.Close()
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	client := NewMetadataClient([]string{recovering.URL, dead.URL})
	if _, err := client.Locations(context.Background(), "v--4"); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatal("did not retry election")
	}
}

func TestMetadataRecoveryBoundedAndCancellable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { protocol.Fail(w, 503, "NOT_LEADER", "election") }))
	defer server.Close()
	client := NewMetadataClient([]string{server.URL})
	client.retryTimeout = 30 * time.Millisecond
	_, err := client.Locations(context.Background(), "v--4")
	var unavailable *metadataUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("missing retryable error: %v", err)
	}
	client.retryTimeout = 8 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = client.Locations(ctx, "v--4")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost cancellation: %v", err)
	}
}

func TestMetadataDoesNotRetryAuthoritativeOrMalformedResponses(t *testing.T) {
	for _, status := range []int{404, 409, 200} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
				w.Write([]byte("invalid JSON"))
			}))
			defer server.Close()
			client := NewMetadataClient([]string{server.URL})
			if _, err := client.Locations(context.Background(), "v--4"); err == nil {
				t.Fatal("accepted invalid metadata")
			}
			if requests.Load() != 1 {
				t.Fatal("retried permanent error")
			}
		})
	}
}
