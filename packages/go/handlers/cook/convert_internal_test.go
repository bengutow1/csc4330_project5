// Author: Caden Lund
package cook

import (
	"testing"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// TestEveryPointReasonHasOneWireValue: a reason added to the points rules or to cook.proto on one
// side only would fail every plate that earns it, or never reach the receipt.
func TestEveryPointReasonHasOneWireValue(t *testing.T) {
	t.Parallel()
	values := cookv1.PointReason(0).Descriptor().Values()
	want := values.Len() - 1 // every value but UNSPECIFIED
	seen := map[cookv1.PointReason]points.Reason{}
	for reason, wire := range pointReasons {
		if wire == cookv1.PointReason_POINT_REASON_UNSPECIFIED {
			t.Errorf("%s maps to unspecified", reason)
		}
		if other, ok := seen[wire]; ok {
			t.Errorf("%s and %s both map to %v", reason, other, wire)
		}
		seen[wire] = reason
	}
	if len(seen) != want {
		t.Errorf("%d wire reasons mapped, want all %d", len(seen), want)
	}
}

// TestNewBadgesAreOnlyTheOnesThisPlateEarned: the receipt celebrates a badge once, on the plate
// that earned it, and never one the player already had.
func TestNewBadgesAreOnlyTheOnesThisPlateEarned(t *testing.T) {
	t.Parallel()
	before := points.Stats{Scans: 1, BestScore: 29}
	after := points.Stats{Scans: 1, BestScore: 29, Cooks: 1, CooksUnder35: 1}

	got := newBadges(before, after)

	var names []string
	for _, b := range got {
		names = append(names, b.GetName())
		if !b.GetEarned() || b.GetHowToEarn() == "" {
			t.Errorf("%s = %v, want earned with its line", b.GetName(), b)
		}
	}
	want := []string{"On the Board", "Underdog"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Errorf("new badges = %v, want %v", names, want)
	}
	if again := newBadges(after, after); len(again) != 0 {
		t.Errorf("a plate that earned nothing new: %v, want none", again)
	}
}
