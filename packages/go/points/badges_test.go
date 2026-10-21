// Author: Caden Lund
package points_test

import (
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// earned names the badges a player has, in shelf order.
func earned(badges []points.Badge) []string {
	var names []string
	for _, b := range badges {
		if b.Earned {
			names = append(names, b.Name)
		}
	}
	return names
}

// TestTheElevenBadgesAndHowToEarnThem keeps the shelf complete and its copy the concept's: a
// locked badge shows its how-to-earn line, so a typo here is a typo on screen.
func TestTheElevenBadgesAndHowToEarnThem(t *testing.T) {
	t.Parallel()
	want := []points.Badge{
		{Name: "First Roast", HowToEarn: "Scan your first fridge"},
		{Name: "Certified Chef", HowToEarn: "Score 85 or higher on a scan"},
		{Name: "On the Board", HowToEarn: "Earn your first points"},
		{Name: "Underdog", HowToEarn: "Cook from a fridge that scored under 35"},
		{Name: "Three's a Meal", HowToEarn: "Cook a recipe with three ingredients or fewer"},
		{Name: "Combo Meal", HowToEarn: "Plate a meal in Combo mode"},
		{Name: "Clean Plate Club", HowToEarn: "Cook seven days in a row"},
		{Name: "Late Night Legend", HowToEarn: "Plate a meal between midnight and 4 am"},
		{Name: "Glow Up", HowToEarn: "Raise your fridge score by 40 or more within a week"},
		{Name: "Microwave Maestro", HowToEarn: "Cook five microwave-only recipes"},
		{Name: "Bag Secured", HowToEarn: "Top the Hall of Fries for a week"},
	}

	got := points.Badges(points.Stats{})

	if !slices.Equal(got, want) {
		t.Errorf("Badges(no history) = %+v, want %+v", got, want)
	}
}

// TestEachBadgeTurnsOnAtItsEdge: one short of a feat earns nothing, and the feat itself earns
// that badge and no other, so no predicate reads the wrong stat or misses its threshold.
func TestEachBadgeTurnsOnAtItsEdge(t *testing.T) {
	t.Parallel()
	cases := []struct {
		badge string
		short points.Stats
		edge  points.Stats
	}{
		{badge: "First Roast", short: points.Stats{Scans: 0}, edge: points.Stats{Scans: 1}},
		{badge: "Certified Chef", short: points.Stats{BestScore: 84}, edge: points.Stats{BestScore: 85}},
		{badge: "On the Board", short: points.Stats{Cooks: 0}, edge: points.Stats{Cooks: 1}},
		{badge: "Underdog", short: points.Stats{CooksUnder35: 0}, edge: points.Stats{CooksUnder35: 1}},
		{
			badge: "Three's a Meal",
			short: points.Stats{CooksWithThreeOrFewer: 0}, edge: points.Stats{CooksWithThreeOrFewer: 1},
		},
		{badge: "Combo Meal", short: points.Stats{ComboCooks: 0}, edge: points.Stats{ComboCooks: 1}},
		{badge: "Clean Plate Club", short: points.Stats{LongestStreak: 6}, edge: points.Stats{LongestStreak: 7}},
		{
			badge: "Late Night Legend",
			short: points.Stats{LateNightPlates: 0}, edge: points.Stats{LateNightPlates: 1},
		},
		{badge: "Glow Up", short: points.Stats{BiggestWeeklyRise: 39}, edge: points.Stats{BiggestWeeklyRise: 40}},
		{
			badge: "Microwave Maestro",
			short: points.Stats{MicrowaveOnlyCooks: 4}, edge: points.Stats{MicrowaveOnlyCooks: 5},
		},
		{badge: "Bag Secured", short: points.Stats{HallOfFriesWeeks: 0}, edge: points.Stats{HallOfFriesWeeks: 1}},
	}
	for _, c := range cases {
		t.Run(c.badge, func(t *testing.T) {
			t.Parallel()

			short := earned(points.Badges(c.short))
			edge := earned(points.Badges(c.edge))

			if len(short) != 0 {
				t.Errorf("one short earned %v, want none", short)
			}
			if !slices.Equal(edge, []string{c.badge}) {
				t.Errorf("at the edge earned %v, want [%s]", edge, c.badge)
			}
		})
	}
}

// TestAPlayerWhoHasDoneEverythingHasEveryBadge: no badge has a ceiling, so going past a feat
// never takes its badge away.
func TestAPlayerWhoHasDoneEverythingHasEveryBadge(t *testing.T) {
	t.Parallel()
	veteran := points.Stats{
		Scans: 40, BestScore: 97, Cooks: 60, CooksUnder35: 8, CooksWithThreeOrFewer: 12,
		ComboCooks: 3, LongestStreak: 14, LateNightPlates: 5, MicrowaveOnlyCooks: 9,
		BiggestWeeklyRise: 55, HallOfFriesWeeks: 2,
	}

	got := points.Badges(veteran)

	if len(got) != 11 {
		t.Fatalf("got %d badges, want 11", len(got))
	}
	for _, b := range got {
		if !b.Earned {
			t.Errorf("%s is locked for a player past every feat", b.Name)
		}
	}
}
