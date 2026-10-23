// Author: LAPTOP-BPMQJP10\andyk
package combo

import (
	"testing"

	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestTwoWeakFridgesEarnTheCombinedMealBonus prevents summing or copying individual scores.
func TestTwoWeakFridgesEarnTheCombinedMealBonus(t *testing.T) {
	t.Parallel()
	c := repository.Combo{State: repository.ComboOpen, Members: []repository.ComboMember{
		{AccountID: 1, Scan: &repository.Scan{Score: 19, Items: []repository.ScanItem{{Name: "Eggs", Category: repository.CategoryProtein}}}},
		{AccountID: 2, Scan: &repository.Scan{Score: 19, Items: []repository.ScanItem{{Name: "Peas", Category: repository.CategoryVegetable}}}},
	}}
	got, err := toProtoCombo(c)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if got.Verdict.GetScore() != 58 || got.Verdict.GetTier() != scanv1.Tier_TIER_SCRAPPY {
		t.Errorf("combined = %v, want 58, scrappy", got.Verdict)
	}
	if got.Players[0].Scan.Score != 19 || got.Players[1].Scan.Score != 19 {
		t.Error("combined scoring changed individual scores")
	}
	if len(got.Verdict.Ingredients) != 2 || got.Verdict.Ingredients[0].AccountId != 1 || got.Verdict.Ingredients[1].AccountId != 2 {
		t.Errorf("provenance = %v", got.Verdict.Ingredients)
	}
}

// TestTwoEmptyScansStillRevealACombinedVerdict distinguishes empty food from a missing scan.
func TestTwoEmptyScansStillRevealACombinedVerdict(t *testing.T) {
	t.Parallel()
	c := repository.Combo{State: repository.ComboOpen, Members: []repository.ComboMember{
		{AccountID: 1, Scan: &repository.Scan{}}, {AccountID: 2, Scan: &repository.Scan{}},
	}}
	got, err := toProtoCombo(c)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if got.Verdict == nil || got.Verdict.Score != 0 || got.Verdict.Tier != scanv1.Tier_TIER_FRIES_IN_THE_BAG || len(got.Verdict.Ingredients) != 0 {
		t.Errorf("combined = %v, want empty fries verdict", got.Verdict)
	}
}
