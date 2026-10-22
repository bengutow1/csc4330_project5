// Author: Caden Lund
package points

// FryRatio is the share of a player's meals that were fry runs: runs over runs and cooks together,
// from 0 to 1. A player with no meals at all has 0, never a division by zero, and one who has only
// ever gone for fries has 1.
func FryRatio(fryRuns, cooks int) float64 {
	meals := fryRuns + cooks
	if meals == 0 {
		return 0
	}
	return float64(fryRuns) / float64(meals)
}
