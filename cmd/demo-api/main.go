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
	sessions := playback.NewSessions(ctx, engine)
	m := http.NewServeMux()
	m.Handle("/api/v1/", sessions.Handler())
	m.Handle("/", http.FileServer(http.Dir(protocol.Environment("FRONTEND_DIR", "frontend/dist"))))
	address := protocol.Environment("HTTP_ADDR", "127.0.0.1:8080")
	log.Printf("demo API listening on %s", address)
	if err := protocol.Serve(ctx, address, m); err != nil {
		log.Fatal(err)
	}
}
