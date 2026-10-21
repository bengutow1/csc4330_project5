// Author: Caden Lund
package scoring_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
)

// TestTierOfStartsEachTierAtItsFloor: the tier picks the verdict's big button, so a floor off by
// one sends a 35 to the fries.
func TestTierOfStartsEachTierAtItsFloor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		score int
		want  scoring.Tier
	}{
		{score: 0, want: scoring.TierFriesInTheBag},
		{score: 34, want: scoring.TierFriesInTheBag},
		{score: 35, want: scoring.TierScrappy},
		{score: 59, want: scoring.TierScrappy},
		{score: 60, want: scoring.TierHomeCook},
		{score: 84, want: scoring.TierHomeCook},
		{score: 85, want: scoring.TierCertifiedChef},
		{score: 100, want: scoring.TierCertifiedChef},
	}
	for _, c := range cases {
		got := scoring.TierOf(c.score)
		if got != c.want {
			t.Errorf("TierOf(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

// TestTheFloorsAreTheContracts: the points rules and the screens read these consts, so they must
// say what docs/contracts.md says.
func TestTheFloorsAreTheContracts(t *testing.T) {
	t.Parallel()
	if scoring.ScrappyFloor != 35 {
		t.Errorf("ScrappyFloor = %d, want 35", scoring.ScrappyFloor)
	}
	if scoring.HomeCookFloor != 60 {
		t.Errorf("HomeCookFloor = %d, want 60", scoring.HomeCookFloor)
	}
	if scoring.CertifiedChefFloor != 85 {
		t.Errorf("CertifiedChefFloor = %d, want 85", scoring.CertifiedChefFloor)
	}
}

// TestEveryMockFridgeLandsInItsTier: the mock serves one fridge per tier, so local dev and the
// demo show every verdict. A formula that moved one would quietly lose a screen state.
func TestEveryMockFridgeLandsInItsTier(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		items []scoring.Item
		score int
		tier  scoring.Tier
	}{
		{name: "certified_chef", items: chefFridge(), score: 91, tier: scoring.TierCertifiedChef},
		{name: "home_cook", items: homeCookFridge(), score: 79, tier: scoring.TierHomeCook},
		{name: "scrappy", items: scrappyFridge(), score: 39, tier: scoring.TierScrappy},
		{name: "fries_in_the_bag", items: friesFridge(), score: 29, tier: scoring.TierFriesInTheBag},
		{name: "empty", items: nil, score: 0, tier: scoring.TierFriesInTheBag},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := scoring.Fridge(c.items)

			if got != c.score {
				t.Errorf("score = %d, want %d", got, c.score)
			}
			if scoring.TierOf(got) != c.tier {
				t.Errorf("tier = %q, want %q", scoring.TierOf(got), c.tier)
			}
		})
	}
}

// TestTheConceptsDemoFridgesLandInTheirTiers: the concept's roast fridge must be roasted and its
// chef fridge certified, or the demo tells the wrong story.
func TestTheConceptsDemoFridgesLandInTheirTiers(t *testing.T) {
	t.Parallel()
	roast := []scoring.Item{
		{Name: "Ketchup", Category: scoring.Condiment},
		{Name: "Egg", Category: scoring.Protein},
		{Name: "Old lemon", Category: scoring.Fruit},
		{Name: "Energy drinks", Category: scoring.Drink},
		{Name: "Mystery container", Category: scoring.Other},
	}

	got := scoring.Fridge(roast)
	if got >= scoring.ScrappyFloor {
		t.Errorf("roast fridge scored %d, want under %d", got, scoring.ScrappyFloor)
	}
	got = scoring.Fridge(chefFridge())
	if got < scoring.CertifiedChefFloor {
		t.Errorf("chef fridge scored %d, want %d or more", got, scoring.CertifiedChefFloor)
	}
}

// TestAnEmptyFridgeScoresZero: nothing found is a verdict too, and it is the lowest one.
func TestAnEmptyFridgeScoresZero(t *testing.T) {
	t.Parallel()
	for _, items := range [][]scoring.Item{nil, {}} {
		got := scoring.Fridge(items)
		if got != 0 {
			t.Errorf("Fridge(%#v) = %d, want 0", items, got)
		}
	}
}

// TestOrderNeverChangesTheScore: the model lists what it sees in any order, and the same fridge
// must score the same however it is listed.
func TestOrderNeverChangesTheScore(t *testing.T) {
	t.Parallel()
	items := homeCookFridge()
	want := scoring.Fridge(items)

	reversed := slices.Clone(items)
	slices.Reverse(reversed)
	byName := slices.Clone(items)
	slices.SortFunc(byName, func(a, b scoring.Item) int { return cmp.Compare(a.Name, b.Name) })
	rotated := append(slices.Clone(items[3:]), items[:3]...)

	for name, order := range map[string][]scoring.Item{"reversed": reversed, "by name": byName, "rotated": rotated} {
		got := scoring.Fridge(order)
		if got != want {
			t.Errorf("%s: score = %d, want %d as listed", name, got, want)
		}
	}
}

// TestHowMuchOfAFoodNeverCounts: a dozen eggs make the same fridge as one egg. Item has no
// quantity, so the only way to say "more" is to list a food again, and that must not pay.
func TestHowMuchOfAFoodNeverCounts(t *testing.T) {
	t.Parallel()
	one := friesFridge()
	dozen := slices.Clone(one)
	for range 11 {
		dozen = append(dozen, scoring.Item{Name: "Egg", Category: scoring.Protein})
	}

	got := scoring.Fridge(dozen)

	want := scoring.Fridge(one)
	if got != want {
		t.Errorf("twelve eggs scored %d, want %d like one", got, want)
	}
}

// TestARepeatedNameCountsOnce: names are trimmed and lowercased first, so "Eggs", " eggs " and
// "EGGS" are one food, not three proteins.
func TestARepeatedNameCountsOnce(t *testing.T) {
	t.Parallel()
	repeated := []scoring.Item{
		{Name: "Eggs", Category: scoring.Protein},
		{Name: " eggs ", Category: scoring.Protein},
		{Name: "EGGS", Category: scoring.Protein},
	}

	got := scoring.Fridge(repeated)

	want := scoring.Fridge(repeated[:1])
	if got != want {
		t.Errorf("Eggs three ways scored %d, want %d like once", got, want)
	}
}

// TestANameInTwoCategoriesCountsOnceWhateverTheOrder: a list that files one food under two
// categories counts it once, in the category that ranks first, so the order still cannot matter.
func TestANameInTwoCategoriesCountsOnceWhateverTheOrder(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		items []scoring.Item
		want  int
	}{
		// One protein: 15, and 4 for variety.
		{name: "dairy first", want: 19, items: []scoring.Item{
			{Name: "Eggs", Category: scoring.Dairy},
			{Name: "Eggs", Category: scoring.Protein},
		}},
		{name: "protein first", want: 19, items: []scoring.Item{
			{Name: "Eggs", Category: scoring.Protein},
			{Name: "Eggs", Category: scoring.Dairy},
		}},
		// Protein and vegetable pay the same, but which one wins decides the meal bonus. Two
		// proteins: 15 + 10, and 4 for variety.
		{name: "vegetable first, beside chicken", want: 29, items: []scoring.Item{
			{Name: "Beans", Category: scoring.Vegetable},
			{Name: "Beans", Category: scoring.Protein},
			{Name: "Chicken", Category: scoring.Protein},
		}},
		{name: "protein first, beside chicken", want: 29, items: []scoring.Item{
			{Name: "Chicken", Category: scoring.Protein},
			{Name: "Beans", Category: scoring.Protein},
			{Name: "Beans", Category: scoring.Vegetable},
		}},
	}
	for _, c := range cases {
		got := scoring.Fridge(c.items)
		if got != c.want {
			t.Errorf("%s: score = %d, want %d", c.name, got, c.want)
		}
	}
}

// TestTwoWeakFridgesScoreFarMoreMerged: Combo scores the merged list, and the concept's promise
// is that two weak fridges can add up to a strong one. Here the egg from one and the peas from
// the other make a meal neither had.
func TestTwoWeakFridgesScoreFarMoreMerged(t *testing.T) {
	t.Parallel()
	fries := scoring.Fridge(friesFridge())
	scrappy := scoring.Fridge(scrappyFridge())

	merged := scoring.Fridge(append(friesFridge(), scrappyFridge()...))

	if merged != 80 {
		t.Errorf("merged = %d, want 80", merged)
	}
	if merged <= fries || merged <= scrappy {
		t.Errorf("merged = %d, want more than either alone (%d and %d)", merged, fries, scrappy)
	}
}

// TestEachCategoryPaysItsWorth: proteins and vegetables add the most, and condiments, drinks and
// snacks add little or nothing. One food alone shows each category's worth, variety included.
func TestEachCategoryPaysItsWorth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		category scoring.Category
		want     int
	}{
		{category: scoring.Protein, want: 19},
		{category: scoring.Vegetable, want: 19},
		{category: scoring.Grain, want: 10},
		{category: scoring.Dairy, want: 10},
		{category: scoring.Fruit, want: 9},
		{category: scoring.Leftover, want: 9},
		{category: scoring.Condiment, want: 1},
		{category: scoring.Snack, want: 1},
		{category: scoring.Drink, want: 0},
		{category: scoring.Other, want: 0},
	}
	for _, c := range cases {
		got := scoring.Fridge([]scoring.Item{{Name: "test_food", Category: c.category}})
		if got != c.want {
			t.Errorf("one %s scored %d, want %d", c.category, got, c.want)
		}
	}
}

// TestVarietyBeatsMoreOfTheSame: three kinds of food make more meals than three cuts of meat.
func TestVarietyBeatsMoreOfTheSame(t *testing.T) {
	t.Parallel()
	same := []scoring.Item{
		{Name: "Chicken", Category: scoring.Protein},
		{Name: "Ground beef", Category: scoring.Protein},
		{Name: "Pork chops", Category: scoring.Protein},
	}
	varied := []scoring.Item{
		{Name: "Chicken", Category: scoring.Protein},
		{Name: "Tortillas", Category: scoring.Grain},
		{Name: "Apple", Category: scoring.Fruit},
	}

	gotSame, gotVaried := scoring.Fridge(same), scoring.Fridge(varied)

	if gotSame != 34 || gotVaried != 38 {
		t.Errorf("three proteins = %d, a protein, a grain and a fruit = %d; want 34 and 38", gotSame, gotVaried)
	}
}

// TestSaucesAndDrinksNeverMakeAMeal: a door full of condiments, drinks and snacks is still a
// fries night, however much of it there is.
func TestSaucesAndDrinksNeverMakeAMeal(t *testing.T) {
	t.Parallel()
	door := []scoring.Item{
		{Name: "Ketchup", Category: scoring.Condiment},
		{Name: "Mustard", Category: scoring.Condiment},
		{Name: "Mayo", Category: scoring.Condiment},
		{Name: "Hot sauce", Category: scoring.Condiment},
		{Name: "Soda", Category: scoring.Drink},
		{Name: "Energy drinks", Category: scoring.Drink},
		{Name: "Chips", Category: scoring.Snack},
		{Name: "Candy", Category: scoring.Snack},
	}

	got := scoring.Fridge(door)

	if got != 2 {
		t.Errorf("score = %d, want 2", got)
	}
}

// TestAFullFridgeStopsAt100: the score is out of 100 however much is listed.
func TestAFullFridgeStopsAt100(t *testing.T) {
	t.Parallel()
	var full []scoring.Item
	for _, c := range []scoring.Category{
		scoring.Protein, scoring.Vegetable, scoring.Fruit, scoring.Grain, scoring.Dairy,
		scoring.Condiment, scoring.Drink, scoring.Snack, scoring.Leftover, scoring.Other,
	} {
		for _, n := range []string{"a", "b", "c", "d"} {
			full = append(full, scoring.Item{Name: "test_" + string(c) + "_" + n, Category: c})
		}
	}

	got := scoring.Fridge(full)

	if got != 100 {
		t.Errorf("score = %d, want 100", got)
	}
}

// The four stocked fridges vision.Mock serves (packages/go/vision/fridges.go), written out by
// hand. A change to one there belongs here too.

func chefFridge() []scoring.Item {
	return []scoring.Item{
		{Name: "Chicken", Category: scoring.Protein},
		{Name: "Spinach", Category: scoring.Vegetable},
		{Name: "Bell peppers", Category: scoring.Vegetable},
		{Name: "Greek yogurt", Category: scoring.Dairy},
		{Name: "Eggs", Category: scoring.Protein},
		{Name: "Cheddar", Category: scoring.Dairy},
	}
}

func homeCookFridge() []scoring.Item {
	return []scoring.Item{
		{Name: "Ground beef", Category: scoring.Protein},
		{Name: "Tortillas", Category: scoring.Grain},
		{Name: "Onion", Category: scoring.Vegetable},
		{Name: "Shredded cheese", Category: scoring.Dairy},
		{Name: "Salsa", Category: scoring.Condiment},
		{Name: "Orange juice", Category: scoring.Drink},
	}
}

func scrappyFridge() []scoring.Item {
	return []scoring.Item{
		{Name: "Instant ramen", Category: scoring.Grain},
		{Name: "Frozen peas", Category: scoring.Vegetable},
		{Name: "Hot sauce", Category: scoring.Condiment},
		{Name: "Apple", Category: scoring.Fruit},
		{Name: "Soda", Category: scoring.Drink},
	}
}

func friesFridge() []scoring.Item {
	return []scoring.Item{
		{Name: "Ketchup", Category: scoring.Condiment},
		{Name: "Egg", Category: scoring.Protein},
		{Name: "Lemon", Category: scoring.Fruit},
		{Name: "Energy drinks", Category: scoring.Drink},
		{Name: "Mystery container", Category: scoring.Other},
	}
}
