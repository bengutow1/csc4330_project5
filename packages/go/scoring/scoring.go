// Author: Caden Lund
// Package scoring turns a fridge's ingredient list into its score, from 0 to 100, and a score
// into its tier. It is pure, so the same list always scores the same, in any order.
//
// The formula:
//
//  1. Only distinct foods count. A name is trimmed and lowercased, and one listed twice counts
//     once. If its categories disagree, the one that ranks first in categories below wins.
//  2. Each category pays for its distinct foods, less for each one after the first: protein and
//     vegetable 15, 10 and 5; grain and dairy 6 and 3; fruit and leftover 5 and 2; condiment and
//     snack 1; drink and other nothing. More foods than a category pays for add nothing.
//  3. Variety: +4 for each kind of real food present (protein, vegetable, grain, dairy, fruit,
//     leftover).
//  4. A meal: +20 when there is a protein and a vegetable.
//  5. The total stops at 100. An empty list scores 0.
//
// The meal bonus is why two weak fridges can score far more merged: an egg in one and frozen peas
// in the other each miss it, and together they earn it.
package scoring

import "strings"

// Category is the closed set an ingredient falls in, spelled as docs/contracts.md spells it.
type Category string

// The ten categories.
const (
	Protein   Category = "protein"
	Vegetable Category = "vegetable"
	Fruit     Category = "fruit"
	Grain     Category = "grain"
	Dairy     Category = "dairy"
	Condiment Category = "condiment"
	Drink     Category = "drink"
	Snack     Category = "snack"
	Leftover  Category = "leftover"
	Other     Category = "other"
)

// Item is a food as the score sees it. It has no quantity on purpose: how much of something
// there is never counts.
type Item struct {
	Name     string
	Category Category
}

// Tier is the band a score falls in, which decides the verdict's big button. It is computed from
// the score every time, never stored.
type Tier string

// The four tiers, spelled as docs/contracts.md spells them.
const (
	TierCertifiedChef Tier = "certified_chef"
	TierHomeCook      Tier = "home_cook"
	TierScrappy       Tier = "scrappy"
	TierFriesInTheBag Tier = "fries_in_the_bag"
)

// The lowest score of each tier. Below ScrappyFloor is fries_in_the_bag.
const (
	CertifiedChefFloor = 85
	HomeCookFloor      = 60
	ScrappyFloor       = 35
)

const (
	// varietyBonus is paid once for each kind of real food present.
	varietyBonus = 4
	// mealBonus is paid when there is a protein and a vegetable, the makings of a dinner.
	mealBonus = 20
	maxScore  = 100
)

// categories is what each category is worth, in rank order: a name filed under two categories
// counts in the one listed first.
var categories = []struct {
	category Category
	pays     []int // for the first distinct food, the second, the third
	food     bool  // real food, which counts toward variety
}{
	{category: Protein, pays: []int{15, 10, 5}, food: true},
	{category: Vegetable, pays: []int{15, 10, 5}, food: true},
	{category: Grain, pays: []int{6, 3}, food: true},
	{category: Dairy, pays: []int{6, 3}, food: true},
	{category: Fruit, pays: []int{5, 2}, food: true},
	{category: Leftover, pays: []int{5, 2}, food: true},
	{category: Condiment, pays: []int{1}},
	{category: Snack, pays: []int{1}},
	{category: Drink},
	{category: Other},
}

// Fridge scores an ingredient list from 0 to 100, by the formula in the package doc. The caller
// has already refused any category outside the closed set; one that slips through adds nothing.
func Fridge(items []Item) int {
	// 1. One category per distinct name, the best-ranked one when the list disagrees
	foods := map[string]int{} // name -> its category's rank
	for _, item := range items {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		r := rank(item.Category)
		have, seen := foods[name]
		if !seen || r < have {
			foods[name] = r
		}
	}

	// 2. How many distinct foods each category holds
	held := make([]int, len(categories))
	for _, r := range foods {
		if r < len(categories) {
			held[r]++
		}
	}

	// 3. What each category pays, then variety and the meal
	score := 0
	for r, c := range categories {
		n := held[r]
		for _, p := range c.pays[:min(n, len(c.pays))] {
			score += p
		}
		if c.food && n > 0 {
			score += varietyBonus
		}
	}
	if held[rank(Protein)] > 0 && held[rank(Vegetable)] > 0 {
		score += mealBonus
	}
	return min(score, maxScore)
}

// rank is c's place in categories, or past the end for a category outside the closed set.
func rank(c Category) int {
	for r, known := range categories {
		if known.category == c {
			return r
		}
	}
	return len(categories)
}

// TierOf is the tier a score falls in.
func TierOf(score int) Tier {
	switch {
	case score >= CertifiedChefFloor:
		return TierCertifiedChef
	case score >= HomeCookFloor:
		return TierHomeCook
	case score >= ScrappyFloor:
		return TierScrappy
	default:
		return TierFriesInTheBag
	}
}
