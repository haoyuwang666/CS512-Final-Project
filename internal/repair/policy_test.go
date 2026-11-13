// TODO(Yuquan): Extend repair policy checks when replica budgets become configurable.
package repair

import (
	"cooperative.video/internal/domain"
	"testing"
)

func TestMinimumTwoNeverCopiesFromNothingOrToSource(t *testing.T) {
	s := Snapshot{Segment: domain.Segment{ID: "s", Size: 3}, Reachable: []domain.Peer{{ID: "p1"}}, Candidates: []domain.PeerObservation{{Peer: domain.Peer{ID: "p1"}, FreeBytes: 10, OnlineFraction: 1}, {Peer: domain.Peer{ID: "p3"}, FreeBytes: 10, OnlineFraction: 1}}}
	task, err := (MinimumTwo{}).Plan(s)
	if err != nil || task.Source.ID != "p1" || task.Target.ID != "p3" {
		t.Fatal(task, err)
	}
	s.Reachable = nil
	if _, err := (MinimumTwo{}).Plan(s); err == nil {
		t.Fatal("cannot invent a source")
	}
	s.Reachable = []domain.Peer{{ID: "p1"}, {ID: "p2"}}
	if _, err := (MinimumTwo{}).Plan(s); err == nil {
		t.Fatal("already protected")
	}
}
