// TODO(Haoyu): Keep controller startup/configuration here; protocol, FSM and policy logic belong in internal packages.
package main

import (
	"cooperative.video/internal/domain"
	"cooperative.video/internal/metadata"
	"cooperative.video/internal/placement"
	"cooperative.video/internal/protocol"
	"encoding/json"
	"log"
	"strconv"
	"time"
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
	server := &metadata.HTTPServer{Node: node, Peers: peers}
	extra, err := strconv.ParseInt(protocol.Environment("EXTRA_REPLICA_BYTES", "0"), 10, 64)
	if err != nil || extra < 0 {
		log.Fatal("invalid EXTRA_REPLICA_BYTES")
	}
	server.ExtraReplicaBytes = extra
	switch protocol.Environment("PLACEMENT_POLICY", "fixed") {
	case "adaptive":
		server.Placement = placement.Adaptive{}
	case "fixed":
	default:
		log.Fatal("PLACEMENT_POLICY must be fixed or adaptive")
	}
	handler := server.Handler()
	grace, err := time.ParseDuration(protocol.Environment("REPAIR_GRACE", "6s"))
	if err != nil || grace <= 0 {
		log.Fatal("invalid REPAIR_GRACE")
	}
	if protocol.Environment("REPAIR_ENABLED", "true") == "true" {
		go server.RunMaintenance(ctx, grace)
	}
	if err := protocol.Serve(ctx, address, handler); err != nil {
		log.Fatal(err)
	}
}
