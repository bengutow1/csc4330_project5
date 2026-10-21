// Author: Caden Lund
package vision_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// recipe is a canned recipe as the tests expect it, steps aside: the fields that decide what the
// server keeps and where its shelves and filters put it.
type recipe struct {
	title      string
	minutes    int
	difficulty string
	equipment  []string
	uses       []string
	missing    []string
	allergens  []string
}

// wantRecipes is every mock fridge's recipes written out by hand, by the pin of the fridge they
// belong to, in the order the mock serves them.
func wantRecipes() map[string][]recipe {
	return map[string][]recipe{
		"certified_chef": {
			{
				title: "Chicken Fajita Bowl", minutes: 25, difficulty: "medium", equipment: []string{"stovetop"},
				uses: []string{"Chicken", "Bell peppers", "Rice", "Oil", "Spices"},
			},
			{
				title: "Spinach Cheddar Omelette", minutes: 10, difficulty: "easy", equipment: []string{"stovetop"},
				uses:      []string{"Eggs", "Spinach", "Cheddar", "Butter", "Salt", "Pepper"},
				allergens: []string{"eggs", "milk"},
			},
			{
				title: "Yogurt Parfait", minutes: 5, difficulty: "easy",
				uses: []string{"Greek yogurt", "Sugar", "Spices"}, allergens: []string{"milk"},
			},
		},
		"home_cook": {
			{
				title: "Beef Tacos", minutes: 20, difficulty: "easy", equipment: []string{"stovetop"},
				uses:      []string{"Ground beef", "Tortillas", "Onion", "Shredded cheese", "Salsa", "Oil", "Spices"},
				allergens: []string{"milk", "wheat"},
			},
			{
				title: "Cheese Quesadilla", minutes: 10, difficulty: "easy", equipment: []string{"stovetop"},
				uses:      []string{"Tortillas", "Shredded cheese", "Salsa", "Butter"},
				allergens: []string{"milk", "wheat"},
			},
			{
				// The invalid one: steak and potatoes are on no list and not declared missing.
				title: "Steak Frites", minutes: 40, difficulty: "hard", equipment: []string{"stovetop", "oven"},
				uses: []string{"Steak", "Potatoes", "Oil", "Salt"},
			},
		},
		"scrappy": {
			{
				title: "Spicy Pea Ramen", minutes: 6, difficulty: "easy", equipment: []string{"microwave"},
				uses: []string{"Instant ramen", "Frozen peas", "Hot sauce"}, allergens: []string{"wheat", "soy"},
			},
			{
				title: "Microwave Baked Apple", minutes: 5, difficulty: "easy", equipment: []string{"microwave"},
				uses: []string{"Apple", "Butter", "Sugar", "Spices"}, allergens: []string{"milk"},
			},
		},
		"fries_in_the_bag": {
			{
				// The one item away: there is no bread to toast.
				title: "Sad Egg on Toast", minutes: 8, difficulty: "easy", equipment: []string{"stovetop"},
				uses: []string{"Egg", "Butter", "Salt"}, missing: []string{"Bread"},
				allergens: []string{"eggs", "milk", "wheat"},
			},
			{
				title: "Lemon Water (Fancy)", minutes: 2, difficulty: "easy",
				uses: []string{"Lemon"},
			},
		},
		"empty": nil,
	}
}

// staples is the contracts' closed set of staples, which a recipe may use off the list.
var staples = []string{"salt", "pepper", "oil", "butter", "rice", "pasta", "flour", "sugar", "spices"}

// TestEachMockFridgeServesItsRecipes: each tier's fridge has its own shelf of recipes, so every
// verdict a demo lands on has something to cook, and the empty fridge has nothing.
func TestEachMockFridgeServesItsRecipes(t *testing.T) {
	t.Parallel()
	want := wantRecipes()
	for _, f := range wantFridges() {
		t.Run(f.pin, func(t *testing.T) {
			t.Parallel()

			got := recipes(t, quickMock(t, ""), f.items)

			assertRecipes(t, got, want[f.pin])
		})
	}
}

// TestTheInvalidRecipeIsThereForTheServerToDrop: #21 must prove it drops a recipe that uses food
// the scan never listed, so the mock serves exactly one, from the home cook fridge.
func TestTheInvalidRecipeIsThereForTheServerToDrop(t *testing.T) {
	t.Parallel()
	var invalid []string
	for _, f := range wantFridges() {
		for _, r := range recipes(t, quickMock(t, ""), f.items) {
			if len(offTheList(r.Uses, f.items)) > 0 {
				invalid = append(invalid, f.pin+": "+r.Title)
			}
		}
	}

	want := []string{"home_cook: Steak Frites"}
	if !slices.Equal(invalid, want) {
		t.Errorf("recipes using food off the list = %v, want %v", invalid, want)
	}
}

// TestExactlyOneRecipeIsOneItemAway: the One item away shelf needs a recipe that misses one
// thing and says so, and a recipe that misses two is something #21 must never be handed.
func TestExactlyOneRecipeIsOneItemAway(t *testing.T) {
	t.Parallel()
	var oneAway []string
	for _, f := range wantFridges() {
		for _, r := range recipes(t, quickMock(t, ""), f.items) {
			switch len(r.Missing) {
			case 0:
			case 1:
				oneAway = append(oneAway, f.pin+": "+r.Title+" needs "+r.Missing[0])
				if len(offTheList(r.Missing, f.items)) != 1 {
					t.Errorf("%s declares %v missing, but it is on the list or a staple", r.Title, r.Missing)
				}
			default:
				t.Errorf("%s misses %v, want one item at most", r.Title, r.Missing)
			}
		}
	}

	want := []string{"fries_in_the_bag: Sad Egg on Toast needs Bread"}
	if !slices.Equal(oneAway, want) {
		t.Errorf("one item away = %v, want %v", oneAway, want)
	}
}

// TestEveryRecipeIsWellFormed: the cards and filters read every field, so a blank title, a zero
// time, or a difficulty, piece of equipment or allergen outside its closed set would break them.
func TestEveryRecipeIsWellFormed(t *testing.T) {
	t.Parallel()
	difficulties := []string{"easy", "medium", "hard"}
	equipment := []string{"microwave", "stovetop", "oven", "air_fryer"}
	allergens := []string{"peanuts", "tree_nuts", "milk", "eggs", "wheat", "soy", "fish", "shellfish", "sesame"}
	titles := map[string]bool{}

	for _, f := range wantFridges() {
		for _, r := range recipes(t, quickMock(t, ""), f.items) {
			if r.Title == "" || titles[r.Title] {
				t.Errorf("%s serves the title %q, want a unique, non-empty one", f.pin, r.Title)
			}
			titles[r.Title] = true
			// 45 minutes is the longest time filter, so a recipe over it could never be found.
			if r.Minutes < 1 || r.Minutes > 45 {
				t.Errorf("%s takes %d minutes, want 1 to 45", r.Title, r.Minutes)
			}
			if !slices.Contains(difficulties, r.Difficulty) {
				t.Errorf("%s is %q, want one of %v", r.Title, r.Difficulty, difficulties)
			}
			for _, e := range r.Equipment {
				if !slices.Contains(equipment, e) {
					t.Errorf("%s needs %q, want one of %v", r.Title, e, equipment)
				}
			}
			for _, a := range r.Allergens {
				if !slices.Contains(allergens, a) {
					t.Errorf("%s contains %q, want one of %v", r.Title, a, allergens)
				}
			}
			if len(r.Steps) == 0 || slices.Contains(r.Steps, "") {
				t.Errorf("%s has steps %q, want at least one and none blank", r.Title, r.Steps)
			}
		}
	}
	// Otherwise an empty shelf would pass every check above.
	if len(titles) == 0 {
		t.Error("the mock served no recipes to check")
	}
}

// TestRecipesFollowTheList: a list the player edited still finds its fridge, so regenerating
// after an edit hands the server recipes whose checks now decide what stays.
func TestRecipesFollowTheList(t *testing.T) {
	t.Parallel()
	chef := wantFridges()[0].items // certified_chef
	cases := []struct {
		name  string
		items []vision.Item
		want  []string
	}{
		{
			name:  "one item removed",
			items: chef[1:],
			want:  []string{"Chicken Fajita Bowl", "Spinach Cheddar Omelette", "Yogurt Parfait"},
		},
		{
			name: "renamed in another case and order",
			items: []vision.Item{
				{Name: "cheddar", Category: "dairy"}, {Name: "SPINACH", Category: "vegetable"},
			},
			want: []string{"Chicken Fajita Bowl", "Spinach Cheddar Omelette", "Yogurt Parfait"},
		},
		{
			name: "the most shared names win",
			items: []vision.Item{
				{Name: "Eggs", Category: "protein"}, {Name: "Tortillas", Category: "grain"},
				{Name: "Salsa", Category: "condiment"},
			},
			want: []string{"Beef Tacos", "Cheese Quesadilla", "Steak Frites"},
		},
		{
			name: "a tie goes to the first fridge",
			items: []vision.Item{
				{Name: "Ground beef", Category: "protein"}, {Name: "Chicken", Category: "protein"},
			},
			want: []string{"Chicken Fajita Bowl", "Spinach Cheddar Omelette", "Yogurt Parfait"},
		},
		{
			name:  "nothing shared",
			items: []vision.Item{{Name: "test_mystery_food", Category: "other"}},
			want:  nil,
		},
		{name: "nothing listed", items: nil, want: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := recipes(t, quickMock(t, ""), c.items)

			titles := make([]string, len(got))
			for i, r := range got {
				titles[i] = r.Title
			}
			if !slices.Equal(titles, c.want) {
				t.Errorf("recipes = %v, want %v", titles, c.want)
			}
		})
	}
}

// TestTheMockLeavesFilteringToTheServer: the kitchen narrows recipes in the server's checks and
// SQL, so a mock that filtered too would let those go untested.
func TestTheMockLeavesFilteringToTheServer(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "")
	chef := wantFridges()[0].items
	strict := vision.Kitchen{
		Equipment: []string{"microwave"},
		Diet:      "vegan",
		Allergies: []string{"peanuts", "tree_nuts", "milk", "eggs", "wheat", "soy", "fish", "shellfish", "sesame"},
	}

	got, err := mock.Recipes(t.Context(), chef, strict)
	if err != nil {
		t.Fatalf("recipes: %v", err)
	}

	assertRecipes(t, got, wantRecipes()["certified_chef"])
}

// TestEditingRecipesLeavesTheMockAlone: every scan shares the canned recipes, so a caller that
// changes what it got back must not change the next player's.
func TestEditingRecipesLeavesTheMockAlone(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "")
	chef := wantFridges()[0].items
	first := recipes(t, mock, chef)
	if len(first) == 0 {
		t.Fatal("certified_chef served no recipes to edit")
	}

	first[0].Title = "test_edited"
	first[0].Uses[0] = "test_edited"
	first[0].Steps[0] = "test_edited"

	second := recipes(t, mock, chef)
	if second[0].Title != "Chicken Fajita Bowl" || second[0].Uses[0] != "Chicken" ||
		second[0].Steps[0] == "test_edited" {
		t.Errorf("first recipe after an edit = %+v, want Chicken Fajita Bowl as served", second[0])
	}
}

// TestTheErrorPinFailsRecipes: VISION_MOCK=error is a model outage, which the recipe screen
// shows as "try again" too, not as an empty shelf.
func TestTheErrorPinFailsRecipes(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "error")

	got, err := mock.Recipes(t.Context(), wantFridges()[0].items, aKitchen())

	if !errors.Is(err, vision.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
	if got != nil {
		t.Errorf("recipes = %v, want none beside the error", got)
	}
}

// TestRecipesTakeTwoSecondsAndStopWhenCancelled: the pause makes the wait real on screen, and a
// player who leaves frees the request at once.
func TestRecipesTakeTwoSecondsAndStopWhenCancelled(t *testing.T) {
	t.Parallel()
	t.Run("the pause", func(t *testing.T) {
		t.Parallel()
		start := time.Now()

		recipes(t, newMock(t, ""), wantFridges()[2].items)

		elapsed := time.Since(start)
		if elapsed < 2*time.Second {
			t.Errorf("recipes took %v, want at least 2s", elapsed)
		}
	})
	t.Run("a cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			_, err := newMock(t, "").Recipes(ctx, wantFridges()[2].items, aKitchen())
			done <- err
		}()

		cancel()

		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Recipes was still waiting a second after its context was cancelled")
		}
	})
}

// recipes is one Recipes call that must succeed.
func recipes(t *testing.T, mock *vision.Mock, items []vision.Item) []vision.Recipe {
	t.Helper()
	got, err := mock.Recipes(t.Context(), items, aKitchen())
	if err != nil {
		t.Fatalf("recipes for %v: %v", items, err)
	}
	return got
}

// assertRecipes compares what the mock served with what a test wrote out, steps aside: those must
// only be there.
func assertRecipes(t *testing.T, got []vision.Recipe, want []recipe) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("served %d recipes (%v), want %d", len(got), got, len(want))
	}
	for i, r := range got {
		w := want[i]
		if r.Title != w.title || r.Minutes != w.minutes || r.Difficulty != w.difficulty {
			t.Errorf("recipe %d = %q, %d min, %s; want %q, %d min, %s",
				i, r.Title, r.Minutes, r.Difficulty, w.title, w.minutes, w.difficulty)
		}
		if !slices.Equal(r.Equipment, w.equipment) || !slices.Equal(r.Uses, w.uses) ||
			!slices.Equal(r.Missing, w.missing) || !slices.Equal(r.Allergens, w.allergens) {
			t.Errorf("%s equipment, uses, missing, allergens = %v, %v, %v, %v; want %v, %v, %v, %v",
				r.Title, r.Equipment, r.Uses, r.Missing, r.Allergens,
				w.equipment, w.uses, w.missing, w.allergens)
		}
		if len(r.Steps) == 0 {
			t.Errorf("%s has no steps", r.Title)
		}
	}
}

// offTheList is the foods that are neither on the list nor a staple, matched the way a player
// would read them, case aside.
func offTheList(foods []string, items []vision.Item) []string {
	var off []string
	for _, food := range foods {
		onList := slices.ContainsFunc(items, func(it vision.Item) bool { return strings.EqualFold(it.Name, food) })
		if !onList && !slices.Contains(staples, strings.ToLower(food)) {
			off = append(off, food)
		}
	}
	return off
}

// aKitchen is an everyday kitchen: two kinds of heat, no diet, no allergies.
func aKitchen() vision.Kitchen {
	return vision.Kitchen{Equipment: []string{"stovetop", "microwave"}, Diet: "none"}
}
