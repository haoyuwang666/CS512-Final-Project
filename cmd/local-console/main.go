// TODO(Haoyu): Keep the local console bound to loopback and stop only its own child processes.
package main

import (
	"cooperative.video/internal/localconsole"
	"cooperative.video/internal/protocol"
	"log"
)

func main() {
	ctx, cancel := protocol.SignalContext()
	defer cancel()
	manager := localconsole.New(ctx)
	defer manager.Close()
	log.Print("Local console: http://127.0.0.1:8090")
	if err := protocol.Serve(ctx, "127.0.0.1:8090", manager.Handler()); err != nil {
		log.Fatal(err)
	}
}
