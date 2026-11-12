// TODO(Tian): Extend demo routes with live Peer/buffer snapshots; the UI never bypasses the shared engine.
package main

import (
	"cooperative.video/internal/playback"
	"cooperative.video/internal/protocol"
	"log"
	"net/http"
	"strings"
)

func main() {
	ctx, cancel := protocol.SignalContext()
	defer cancel()
	engine := playback.NewEngine(strings.Split(protocol.Environment("CONTROLLERS", "http://127.0.0.1:8001"), ","))
	// A durable event journal enables replay and marks interrupted sessions after restart.
	sessions, err := playback.OpenSessions(ctx, engine, protocol.Environment("SESSION_DIR", ".data/sessions"))
	if err != nil {
		log.Fatal(err)
	}
	m := http.NewServeMux()
	// Real media delivery has its own lifecycle; simulated Session metrics remain separate.
	media := playback.NewMediaSessions(ctx, engine)
	m.Handle("/api/v1/media/", media.Handler())
	m.Handle("/api/v1/", sessions.Handler())
	m.Handle("/", http.FileServer(http.Dir(protocol.Environment("FRONTEND_DIR", "frontend/dist"))))
	address := protocol.Environment("HTTP_ADDR", "127.0.0.1:8080")
	log.Printf("demo API listening on %s", address)
	if err := protocol.Serve(ctx, address, m); err != nil {
		log.Fatal(err)
	}
}
