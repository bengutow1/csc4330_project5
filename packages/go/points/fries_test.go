// Author: Caden Lund
package points_test

import (
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// TestTheFryRatioIsTheShareOfMealsThatWereFries: screen 10 shows 3 fry runs beside 12 cooks as
// 20%. A player with no meals at all has 0, never a division by zero, and one who has only ever
// gone for fries has 1.
func TestTheFryRatioIsTheShareOfMealsThatWereFries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		fryRuns int
		cooks   int
		want    float64
	}{
		{"screen 10's player", 3, 12, 0.2},
		{"no meals yet", 0, 0, 0},
		{"no cooks, only fries", 4, 0, 1},
		{"no fries", 0, 9, 0},
		{"one of each", 1, 1, 0.5},
	}
	for _, c := range cases {
		got := points.FryRatio(c.fryRuns, c.cooks)
		if got != c.want {
			t.Errorf("%s: FryRatio(%d, %d) = %v, want %v", c.name, c.fryRuns, c.cooks, got, c.want)
		}
	}
}
