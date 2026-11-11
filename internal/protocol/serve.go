// TODO(Haoyu): Add structured request logging and per-route deadlines; SSE intentionally has no global write timeout.
package protocol

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Environment(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
func Serve(ctx context.Context, address string, handler http.Handler) error {
	s := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}()
	err := s.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
