// Author: Caden Lund
package points_test

import (
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// TestReasonsAreSpelledAsTheContractsSpellThem: the ledger's check constraint and the wire enum
// use these strings, so a renamed value would break every write of it.
func TestReasonsAreSpelledAsTheContractsSpellThem(t *testing.T) {
	t.Parallel()
	got := []points.Reason{
		points.ReasonCook, points.ReasonScrappyBonus, points.ReasonAgainstAllOdds,
		points.ReasonStreak, points.ReasonComboCook,
	}
	want := []points.Reason{"cook", "scrappy_bonus", "against_all_odds", "streak", "combo_cook"}

	if !slices.Equal(got, want) {
		t.Errorf("reasons = %v, want %v", got, want)
	}
}

// TestAPlateEarnsTheContractsLines walks the contracts' table: the bonus edges at 34, 35, 59 and
// 60, the streak only on the day's first plate, the combo bonus, and the receipt's order.
func TestAPlateEarnsTheContractsLines(t *testing.T) {
	t.Parallel()
	cook := points.Line{Reason: points.ReasonCook, Points: 100}
	scrappy := points.Line{Reason: points.ReasonScrappyBonus, Points: 50}
	againstAllOdds := points.Line{Reason: points.ReasonAgainstAllOdds, Points: 100}
	cases := []struct {
		name string
		in   points.Plate
		want []points.Line
	}{
		{name: "a perfect fridge earns the cook alone", in: points.Plate{Score: 100}, want: []points.Line{cook}},
		{name: "60 earns no bonus", in: points.Plate{Score: 60}, want: []points.Line{cook}},
		{name: "59 is scrappy", in: points.Plate{Score: 59}, want: []points.Line{cook, scrappy}},
		{name: "35 is scrappy", in: points.Plate{Score: 35}, want: []points.Line{cook, scrappy}},
		{name: "34 is against all odds", in: points.Plate{Score: 34}, want: []points.Line{cook, againstAllOdds}},
		{name: "0 is against all odds", in: points.Plate{Score: 0}, want: []points.Line{cook, againstAllOdds}},
		{
			name: "the day's first plate adds the streak",
			in:   points.Plate{Score: 85, FirstOfDay: true, Streak: 3},
			want: []points.Line{cook, {Reason: points.ReasonStreak, Points: 30}},
		},
		{
			name: "a later plate that day earns no streak",
			in:   points.Plate{Score: 85, Streak: 3},
			want: []points.Line{cook},
		},
		{
			name: "a combo plate adds the combo bonus",
			in:   points.Plate{Score: 85, Combo: true},
			want: []points.Line{cook, {Reason: points.ReasonComboCook, Points: 50}},
		},
		{
			name: "everything at once",
			in:   points.Plate{Score: 20, FirstOfDay: true, Streak: 2, Combo: true},
			want: []points.Line{
				cook, againstAllOdds,
				{Reason: points.ReasonStreak, Points: 20},
				{Reason: points.ReasonComboCook, Points: 50},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.PlateAward(c.in)

			if !slices.Equal(got, c.want) {
				t.Errorf("PlateAward(%+v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestTheStreakBonusStopsAtSeventy: +10 for each day of the streak, today included, counting
// seven days at most.
func TestTheStreakBonusStopsAtSeventy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		streak int
		want   int
	}{
		{name: "one day", streak: 1, want: 10},
		{name: "six days", streak: 6, want: 60},
		{name: "seven days reach the cap", streak: 7, want: 70},
		{name: "eight days stay at the cap", streak: 8, want: 70},
		{name: "thirty days stay at the cap", streak: 30, want: 70},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.PlateAward(points.Plate{Score: 85, FirstOfDay: true, Streak: c.streak})

			want := []points.Line{
				{Reason: points.ReasonCook, Points: 100},
				{Reason: points.ReasonStreak, Points: c.want},
			}
			if !slices.Equal(got, want) {
				t.Errorf("a %d-day streak earned %v, want %v", c.streak, got, want)
			}
		})
	}
}
