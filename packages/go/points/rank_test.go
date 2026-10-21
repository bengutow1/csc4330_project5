// Author: Caden Lund
package points_test

import (
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// TestRanksFollowAllTimePoints pins every title's floor and the point before it: 499 is a Fry
// Cook one point from Line Cook, and 500 is a Line Cook at the start of the bar.
func TestRanksFollowAllTimePoints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		total int
		want  points.Rank
	}{
		{
			name: "a new player", total: 0,
			want: points.Rank{Name: "Fry Cook", Next: "Line Cook", ToNext: 500, Progress: 0},
		},
		{
			name: "halfway", total: 250,
			want: points.Rank{Name: "Fry Cook", Next: "Line Cook", ToNext: 250, Progress: 0.5},
		},
		{
			name: "499 is one short", total: 499,
			want: points.Rank{Name: "Fry Cook", Next: "Line Cook", ToNext: 1, Progress: 0.998},
		},
		{
			name: "500 is line cook", total: 500,
			want: points.Rank{Name: "Line Cook", Next: "Sous Chef", ToNext: 1000, Progress: 0},
		},
		{
			name: "1,499 is one short", total: 1499,
			want: points.Rank{Name: "Line Cook", Next: "Sous Chef", ToNext: 1, Progress: 0.999},
		},
		{
			name: "1,500 is sous chef", total: 1500,
			want: points.Rank{Name: "Sous Chef", Next: "Head Chef", ToNext: 2500, Progress: 0},
		},
		{
			name: "3,999 is one short", total: 3999,
			want: points.Rank{Name: "Sous Chef", Next: "Head Chef", ToNext: 1, Progress: 0.9996},
		},
		{
			name: "4,000 is head chef", total: 4000,
			want: points.Rank{Name: "Head Chef", Next: "Executive Chef", ToNext: 6000, Progress: 0},
		},
		{
			name: "9,999 is one short", total: 9999,
			want: points.Rank{Name: "Head Chef", Next: "Executive Chef", ToNext: 1, Progress: 5999.0 / 6000},
		},
		{name: "10,000 is the top", total: 10000, want: points.Rank{Name: "Executive Chef", Progress: 1}},
		{name: "past the top", total: 25000, want: points.Rank{Name: "Executive Chef", Progress: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.RankOf(c.total)

			if got != c.want {
				t.Errorf("RankOf(%d) = %+v, want %+v", c.total, got, c.want)
			}
		})
	}
}
