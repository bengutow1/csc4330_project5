// Author: Caden Lund
// Package points holds the rules behind every number the board, the profile and the points
// receipt show: what a checked plate earns, streaks, ranks, badges, and the campus clock they all
// run on. Nothing here does I/O or reads the clock; every function is handed the time it needs.
package points

// Reason is why an award was made. Each award is one row in the points ledger, and the values are
// spelled the way the contracts spell them, in SQL and on the wire alike.
type Reason string

const (
	ReasonCook           Reason = "cook"             // a checked plate, one per meal window
	ReasonScrappyBonus   Reason = "scrappy_bonus"    // the recipe's scan scored 35 to 59
	ReasonAgainstAllOdds Reason = "against_all_odds" // the recipe's scan scored under 35
	ReasonStreak         Reason = "streak"           // the day's first plate, per streak day
	ReasonComboCook      Reason = "combo_cook"       // a combo plate, to each player (Next)
)

// Line is one award: a line on the points receipt and a row in the ledger.
type Line struct {
	Reason Reason
	Points int
}

// Plate is what a checked plate's award depends on.
type Plate struct {
	Score      int  // the score of the scan the recipe came from
	FirstOfDay bool // the player's first checked plate of the campus day
	Streak     int  // the current streak in days, this plate's day included (see Streak)
	Combo      bool // cooked in a combo (Next), where one plate counts for both players
}

// streakCap is the most streak days the bonus counts, so it tops out at +70.
const streakCap = 7

// PlateAward is the award lines for one checked plate, in receipt order. The handler has already
// claimed the meal window's cook credit: a plate whose window is taken earns nothing and never
// gets here.
func PlateAward(in Plate) []Line {
	lines := []Line{{Reason: ReasonCook, Points: 100}}
	// The biggest bonuses go to cooking out of a bad fridge, and 60 or more earns none.
	switch {
	case in.Score < 35:
		lines = append(lines, Line{Reason: ReasonAgainstAllOdds, Points: 100})
	case in.Score < 60:
		lines = append(lines, Line{Reason: ReasonScrappyBonus, Points: 50})
	}
	if in.FirstOfDay {
		lines = append(lines, Line{Reason: ReasonStreak, Points: 10 * min(in.Streak, streakCap)})
	}
	if in.Combo {
		lines = append(lines, Line{Reason: ReasonComboCook, Points: 50})
	}
	return lines
}
