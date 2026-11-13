// TODO(Yuquan): Compare observed-resource placement with workload-specific policies.
package placement

import (
	"cooperative.video/internal/domain"
	"testing"
)

func TestAdaptiveExcludesOfflineFullDuplicatePeers(t *testing.T) {
	s := domain.Segment{Size: 10}
	peers := []domain.PeerObservation{
		{Peer: domain.Peer{ID: "slow"}, FreeBytes: 100, UploadBytesPerSecond: 10, OnlineFraction: 1},
		{Peer: domain.Peer{ID: "fast"}, FreeBytes: 100, UploadBytesPerSecond: 100, OnlineFraction: 1},
		{Peer: domain.Peer{ID: "fast"}, FreeBytes: 100, UploadBytesPerSecond: 100, OnlineFraction: 1},
		{Peer: domain.Peer{ID: "offline"}, FreeBytes: 100, UploadBytesPerSecond: 1000},
		{Peer: domain.Peer{ID: "full"}, FreeBytes: 9, UploadBytesPerSecond: 1000, OnlineFraction: 1},
	}
	got, err := (Adaptive{}).Choose(s, peers, 2)
	if err != nil || len(got) != 2 || got[0].ID != "fast" || got[1].ID != "slow" {
		t.Fatal(got, err)
	}
	if _, err := (Adaptive{}).Choose(s, peers, 3); err == nil {
		t.Fatal("must reject unavailable capacity")
	}
}

func TestOptionalBudgetCannotStarveMandatoryPairs(t *testing.T) {
	peers := []domain.PeerObservation{}
	for _, id := range []string{"p1", "p2", "p3"} {
		peers = append(peers, domain.PeerObservation{Peer: domain.Peer{ID: id}, FreeBytes: 6, OnlineFraction: 1})
	}
	segments := []domain.Segment{{ID: "s0", Index: 0, Size: 3}, {ID: "s1", Index: 1, Size: 3}, {ID: "s2", Index: 2, Size: 3}}
	plans, spent, err := Plan(segments, peers, Static{}, 3)
	if err != nil || spent != 0 {
		t.Fatal(plans, spent, err)
	}
	for _, targets := range plans {
		if len(targets) != 2 {
			t.Fatal("optional copy starved mandatory pair")
		}
	}
	if peers[0].FreeBytes != 6 {
		t.Fatal("mutated caller's capacity snapshot")
	}
	for i := range peers {
		peers[i].FreeBytes = 9
	}
	plans, spent, err = Plan(segments, peers, Static{}, 4)
	if err != nil || spent != 3 {
		t.Fatal(plans, spent, err)
	}
}
