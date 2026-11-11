// TODO(Haoyu): Keep controller startup/configuration here; protocol, FSM and policy logic belong in internal packages.
package main

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/metadata"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"log"
)

func main() {
	id := protocol.Environment("NODE_ID", "controller-1")
	bind := protocol.Environment("RAFT_BIND", "127.0.0.1:7001")
	address := protocol.Environment("HTTP_ADDR", "127.0.0.1:8001")
	members := []metadata.Member{{ID: id, RaftAddress: bind, HTTPURL: "http://" + address}}
	if raw := protocol.Environment("RAFT_MEMBERS", ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &members); err != nil {
			log.Fatal(err)
		}
	}
	peers := []domain.Peer{}
	if err := json.Unmarshal([]byte(protocol.Environment("PEERS", "[]")), &peers); err != nil {
		log.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range peers {
		if !domain.ValidID(p.ID) || seen[p.ID] {
			log.Fatal("invalid or duplicate peer ID")
		}
		seen[p.ID] = true
	}
	node, err := metadata.Open(metadata.Config{ID: id, Bind: bind, Advertise: protocol.Environment("RAFT_ADVERTISE", bind), DataDir: protocol.Environment("DATA_DIR", ".data/controller-1"), Bootstrap: protocol.Environment("BOOTSTRAP", "true") == "true", Members: members})
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()
	ctx, cancel := protocol.SignalContext()
	defer cancel()
	log.Printf("controller %s listening on %s", id, address)
	if err := protocol.Serve(ctx, address, (&metadata.HTTPServer{Node: node, Peers: peers}).Handler()); err != nil {
		log.Fatal(err)
	}
}
