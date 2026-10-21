// Author: Caden Lund
package scan

import (
	"strings"
	"testing"

	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
)

// TestTheCategoryTableHoldsEveryCategoryOnceInEachSet: the table is three hand-kept lists side by
// side. A category missing from it is food a verdict cannot show, and a row that pairs two
// different categories scores food as something it is not.
func TestTheCategoryTableHoldsEveryCategoryOnceInEachSet(t *testing.T) {
	t.Parallel()
	words := map[string]int{}
	wires := map[scanv1.Category]int{}
	scores := map[scoring.Category]int{}
	for _, row := range categories {
		words[row.word]++
		wires[row.wire]++
		scores[row.scoring]++
		// All three spell the category as docs/contracts.md does, so a crossed row shows here.
		if string(row.scoring) != row.word || row.wire.String() != "CATEGORY_"+strings.ToUpper(row.word) {
			t.Errorf("row %q pairs %v with %q", row.word, row.wire, row.scoring)
		}
	}

	for number, name := range scanv1.Category_name {
		c := scanv1.Category(number)
		if c != scanv1.Category_CATEGORY_UNSPECIFIED && wires[c] != 1 {
			t.Errorf("%s is in the table %d times, want once", name, wires[c])
		}
	}
	for _, word := range []string{
		repository.CategoryProtein, repository.CategoryVegetable, repository.CategoryFruit,
		repository.CategoryGrain, repository.CategoryDairy, repository.CategoryCondiment,
		repository.CategoryDrink, repository.CategorySnack, repository.CategoryLeftover,
		repository.CategoryOther,
	} {
		if words[word] != 1 {
			t.Errorf("%q is in the table %d times, want once", word, words[word])
		}
	}
	for _, c := range []scoring.Category{
		scoring.Protein, scoring.Vegetable, scoring.Fruit, scoring.Grain, scoring.Dairy,
		scoring.Condiment, scoring.Drink, scoring.Snack, scoring.Leftover, scoring.Other,
	} {
		if scores[c] != 1 {
			t.Errorf("scoring's %q is in the table %d times, want once", c, scores[c])
		}
	}
}

// TestEachScoreMapsToItsTierAtEveryBoundary: the tier decides the verdict's big button, so each
// side of each floor has to land in the right one on the wire.
func TestEachScoreMapsToItsTierAtEveryBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		score int
		want  scanv1.Tier
	}{
		{100, scanv1.Tier_TIER_CERTIFIED_CHEF},
		{85, scanv1.Tier_TIER_CERTIFIED_CHEF},
		{84, scanv1.Tier_TIER_HOME_COOK},
		{60, scanv1.Tier_TIER_HOME_COOK},
		{59, scanv1.Tier_TIER_SCRAPPY},
		{35, scanv1.Tier_TIER_SCRAPPY},
		{34, scanv1.Tier_TIER_FRIES_IN_THE_BAG},
		{0, scanv1.Tier_TIER_FRIES_IN_THE_BAG},
	}
	for _, c := range cases {
		got, err := toProtoVerdict(repository.Scan{Score: c.score})
		if err != nil {
			t.Errorf("score %d: %v", c.score, err)
			continue
		}
		if got.GetTier() != c.want || got.GetScore() != int32(c.score) {
			t.Errorf("score %d: got %d, %v; want %d, %v", c.score, got.GetScore(), got.GetTier(), c.score, c.want)
		}
	}
}

// TestEveryRoastLevelReachesTheModelAsItself: the player picks how hard the roast hits, and a
// level lost on the way would roast them at another.
func TestEveryRoastLevelReachesTheModelAsItself(t *testing.T) {
	t.Parallel()
	for _, level := range []string{repository.RoastMild, repository.RoastMedium, repository.RoastExtraCrispy} {
		got, err := roastLevel(level)
		if err != nil || string(got) != level {
			t.Errorf("%q: got %q, %v; want %q", level, got, err, level)
		}
	}
	got, err := roastLevel("scorched")
	if err == nil {
		t.Errorf("an unknown level: got %q, want an error rather than a guess", got)
	}
}
