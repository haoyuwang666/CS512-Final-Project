// TODO(Haoyu): Add controlled membership changes and quorum/failover tests; this uses persistent HashiCorp Raft, not a mock.
package metadata

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb/v2"
)

type Member struct {
	ID          string `json:"id"`
	RaftAddress string `json:"raft_address"`
	HTTPURL     string `json:"http_url"`
}
type Config struct {
	ID, Bind, Advertise, DataDir string
	Bootstrap                    bool
	Members                      []Member
}
type Node struct {
	Raft      *raft.Raft
	FSM       *FSM
	Store     *boltdb.BoltStore
	Transport *raft.NetworkTransport
	Members   []Member
}

func Open(cfg Config) (*Node, error) {
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, err
	}
	store, err := boltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft.db"))
	if err != nil {
		return nil, err
	}
	snaps, err := raft.NewFileSnapshotStore(cfg.DataDir, 2, os.Stderr)
	if err != nil {
		store.Close()
		return nil, err
	}
	advertise := cfg.Advertise
	if advertise == "" {
		advertise = cfg.Bind
	}
	addr, err := net.ResolveTCPAddr("tcp", advertise)
	if err != nil {
		store.Close()
		return nil, err
	}
	transport, err := raft.NewTCPTransport(cfg.Bind, addr, 3, 2*time.Second, os.Stderr)
	if err != nil {
		store.Close()
		return nil, err
	}
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(cfg.ID)
	exists, err := raft.HasExistingState(store, store, snaps)
	if err != nil {
		transport.Close()
		store.Close()
		return nil, err
	}
	fsm := NewFSM()
	r, err := raft.NewRaft(config, fsm, store, store, snaps, transport)
	if err != nil {
		transport.Close()
		store.Close()
		return nil, err
	}
	n := &Node{r, fsm, store, transport, cfg.Members}
	if cfg.Bootstrap && !exists {
		var servers []raft.Server
		for _, m := range cfg.Members {
			servers = append(servers, raft.Server{ID: raft.ServerID(m.ID), Address: raft.ServerAddress(m.RaftAddress), Suffrage: raft.Voter})
		}
		if err := r.BootstrapCluster(raft.Configuration{Servers: servers}).Error(); err != nil {
			n.Close()
			return nil, err
		}
	}
	return n, nil
}
func (n *Node) Close() { _ = n.Raft.Shutdown().Error(); _ = n.Transport.Close(); _ = n.Store.Close() }
func (n *Node) LeaderURL() string {
	_, id := n.Raft.LeaderWithID()
	for _, m := range n.Members {
		if m.ID == string(id) {
			return m.HTTPURL
		}
	}
	return ""
}
func (n *Node) Read() error {
	if err := n.Raft.VerifyLeader().Error(); err != nil {
		return err
	}
	return n.Raft.Barrier(3 * time.Second).Error()
}
func (n *Node) Propose(c Command) (Result, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return Result{}, err
	}
	f := n.Raft.Apply(b, 3*time.Second)
	if err = f.Error(); err != nil {
		return Result{}, err
	}
	result, ok := f.Response().(Result)
	if !ok {
		return Result{}, fmt.Errorf("invalid FSM response")
	}
	return result, nil
}
