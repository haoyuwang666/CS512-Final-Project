// TODO(Yuquan): Add report interval controls and inventory pagination for larger libraries.
package main

import (
	"cooperative.video/internal/protocol"
	"cooperative.video/internal/storage"
	"log"
	"strconv"
	"strings"
)

func main() {
	id := protocol.Environment("PEER_ID", "peer-1")
	quota, err := strconv.ParseInt(protocol.Environment("QUOTA_BYTES", "1073741824"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	upload, err := strconv.ParseFloat(protocol.Environment("UPLOAD_BYTES_PER_SECOND", "5000000"), 64)
	if err != nil || upload <= 0 {
		log.Fatal("upload rate must be positive")
	}
	store, err := storage.Open(protocol.Environment("DATA_DIR", ".data/peer-1"), id, quota)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := protocol.SignalContext()
	defer cancel()
	address := protocol.Environment("HTTP_ADDR", "127.0.0.1:9001")
	log.Printf("peer %s listening on %s", id, address)
	server := storage.NewServer(store, id, upload)
	if raw := protocol.Environment("CONTROLLERS", ""); raw != "" {
		go server.RunReports(ctx, strings.Split(raw, ","))
	}
	if err := protocol.Serve(ctx, address, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
