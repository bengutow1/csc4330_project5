// Author: Caden Lund
package recipes_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/recipes"
)

// fridge is a scan's list as the handler hands it over: the names the model gave.
var fridge = []string{"Eggs", "Spinach", "Cheddar", "Bell peppers"}

// staples are a new player's: every one, as the account stores them.
var staples = []string{"salt", "pepper", "oil", "butter", "rice", "pasta", "flour", "sugar", "spices"}

// allAllergens is the closed set of allergies in docs/contracts.md.
var allAllergens = []string{
	"peanuts", "tree_nuts", "milk", "eggs", "wheat", "soy", "fish", "shellfish", "sesame",
}

// TestCheckKeepsOnlyWhatThePlayerCanCook: a recipe the player cannot make with what they have
// (and one thing they lack, named) must never reach a shelf.
func TestCheckKeepsOnlyWhatThePlayerCanCook(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		recipe  recipes.Recipe
		staples []string
		kept    bool
		why     string // a word the reason must name; "" when kept
	}{
		{
			name:    "an item on the list",
			recipe:  recipes.Recipe{Title: "Wilted Spinach", Uses: []string{"Spinach"}},
			staples: staples, kept: true,
		},
		{
			name:    "a staple",
			recipe:  recipes.Recipe{Title: "Fried Eggs", Uses: []string{"Eggs", "Oil", "Salt"}},
			staples: staples, kept: true,
		},
		{
			name:    "names match whatever their case and spacing",
			recipe:  recipes.Recipe{Title: "Pepper Saute", Uses: []string{" bell PEPPERS ", "OIL"}},
			staples: staples, kept: true,
		},
		{
			name: "one declared missing item",
			recipe: recipes.Recipe{
				Title: "Egg on Toast", Uses: []string{"Eggs", "Butter"}, Missing: []string{"Bread"},
			},
			staples: staples, kept: true,
		},
		{
			name: "two missing",
			recipe: recipes.Recipe{
				Title: "Breakfast Sandwich", Uses: []string{"Eggs"}, Missing: []string{"Bread", "Ham"},
			},
			staples: staples, kept: false, why: "2",
		},
		{
			name:    "an undeclared item",
			recipe:  recipes.Recipe{Title: "Steak and Eggs", Uses: []string{"Eggs", "Steak"}},
			staples: staples, kept: false, why: "Steak",
		},
		{
			name:    "a staple the player does not keep",
			recipe:  recipes.Recipe{Title: "Buttered Spinach", Uses: []string{"Spinach", "Butter"}},
			staples: []string{"salt", "oil"}, kept: false, why: "Butter",
		},
		{
			name: "a missing item is not a use",
			recipe: recipes.Recipe{
				Title: "Egg on Toast", Uses: []string{"Eggs", "Bread"}, Missing: []string{"Bread"},
			},
			staples: staples, kept: false, why: "Bread",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			kept, why := recipes.Check(c.recipe, fridge, c.staples, nil)

			if kept != c.kept {
				t.Errorf("kept = %v (%q), want %v", kept, why, c.kept)
			}
			// A drop needs its reason for the log, and a keep has none.
			if c.kept && why != "" {
				t.Errorf("why = %q, want none for a kept recipe", why)
			}
			if !c.kept && !strings.Contains(why, c.why) {
				t.Errorf("why = %q, want it to name %q", why, c.why)
			}
		})
	}
}

// TestAnAllergyDropsTheRecipeByItsWordOrByItsFood: an allergy is a hard filter, so a recipe must
// be caught when it declares the allergen and when it only gives it away in what it takes.
func TestAnAllergyDropsTheRecipeByItsWordOrByItsFood(t *testing.T) {
	t.Parallel()
	// One food for each allergen that the keyword list must catch, with nothing declared. The
	// food is put on the list, so only the allergy can drop it.
	byFood := map[string]string{
		"peanuts":   "Peanuts",
		"tree_nuts": "Walnuts",
		"milk":      "Mozzarella",
		"eggs":      "Egg",
		"wheat":     "Tortillas",
		"soy":       "Tofu",
		"fish":      "Salmon",
		"shellfish": "Shrimp",
		"sesame":    "Tahini",
	}
	for _, allergen := range allAllergens {
		t.Run(allergen+" declared", func(t *testing.T) {
			t.Parallel()
			recipe := recipes.Recipe{Title: "Plain Rice", Uses: []string{"Rice"}, Allergens: []string{allergen}}

			kept, why := recipes.Check(recipe, nil, staples, []string{allergen})

			if kept || !strings.Contains(why, allergen) {
				t.Errorf("kept = %v (%q), want dropped for %s", kept, why, allergen)
			}
		})
		t.Run(allergen+" by its food", func(t *testing.T) {
			t.Parallel()
			food, ok := byFood[allergen]
			if !ok {
				t.Fatalf("no food for %s: add one", allergen)
			}
			recipe := recipes.Recipe{Title: "A Plate", Uses: []string{food}}

			kept, why := recipes.Check(recipe, []string{food}, staples, []string{allergen})

			if kept || !strings.Contains(why, allergen) {
				t.Errorf("%s: kept = %v (%q), want dropped for %s", food, kept, why, allergen)
			}
		})
	}
}

// TestALookalikeNeverHidesAnotherAllergy: peanut butter is not milk, but it is still peanuts, so
// passing over it for one allergy must leave it for the next.
func TestALookalikeNeverHidesAnotherAllergy(t *testing.T) {
	t.Parallel()
	toast := recipes.Recipe{Title: "Spread", Uses: []string{"Peanut butter"}}
	have := []string{"Peanut butter"}

	milkOnly, why := recipes.Check(toast, have, staples, []string{"milk"})
	if !milkOnly {
		t.Errorf("milk allergy: dropped (%q), want kept: peanut butter holds no milk", why)
	}
	both, why := recipes.Check(toast, have, staples, []string{"milk", "peanuts"})
	if both || !strings.Contains(why, "peanuts") {
		t.Errorf("milk then peanut allergy: kept = %v (%q), want dropped for peanuts", both, why)
	}
}

// TestAnAllergenOnlyMattersToThePlayerWhoHasIt: dropping a recipe for an allergy nobody here has
// would empty the shelves for no reason.
func TestAnAllergenOnlyMattersToThePlayerWhoHasIt(t *testing.T) {
	t.Parallel()
	omelette := recipes.Recipe{
		Title: "Spinach Cheddar Omelette", Uses: []string{"Eggs", "Spinach", "Cheddar"},
		Allergens: []string{"eggs", "milk"},
	}
	for _, allergies := range [][]string{nil, {"peanuts"}, {"fish", "sesame"}} {
		kept, why := recipes.Check(omelette, fridge, staples, allergies)
		if !kept {
			t.Errorf("allergies %v: dropped (%q), want kept", allergies, why)
		}
	}
}

// TestAllergensAreCaughtAnywhereTheRecipeNamesThem: the title and the steps can name food the
// lists left out, and the model's spelling (a plural, capitals) must not hide it.
func TestAllergensAreCaughtAnywhereTheRecipeNamesThem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		recipe recipes.Recipe
		want   []string
	}{
		{name: "nothing", recipe: recipes.Recipe{Title: "Lemon Water", Uses: []string{"Lemon"}}, want: nil},
		{
			name:   "in the title",
			recipe: recipes.Recipe{Title: "Spicy Peanut Bowl", Uses: []string{"Rice"}},
			want:   []string{"peanuts"},
		},
		{
			name: "in a step",
			recipe: recipes.Recipe{
				Title: "Rice Bowl", Uses: []string{"Rice"}, Steps: []string{"Top with sesame seeds."},
			},
			want: []string{"sesame"},
		},
		{
			name:   "in what it lacks",
			recipe: recipes.Recipe{Title: "Toast", Uses: []string{"Oil"}, Missing: []string{"Bread"}},
			want:   []string{"wheat"},
		},
		{
			name:   "a plural and capitals",
			recipe: recipes.Recipe{Title: "Trail Mix", Uses: []string{"ALMONDS", "Cashews"}},
			want:   []string{"tree_nuts"},
		},
		{
			name: "declared and found, merged in order without repeats",
			recipe: recipes.Recipe{
				Title: "Cheese Omelette", Uses: []string{"Eggs", "Cheddar", "Butter"},
				Allergens: []string{"milk", "eggs", "milk"},
			},
			want: []string{"eggs", "milk"},
		},
		{
			name:   "a declared allergen in the model's capitals",
			recipe: recipes.Recipe{Title: "Rice", Uses: []string{"Rice"}, Allergens: []string{" Soy "}},
			want:   []string{"soy"},
		},
		{
			name:   "a phrase across two words",
			recipe: recipes.Recipe{Title: "Stir Fry", Uses: []string{"Soy sauce"}},
			want:   []string{"soy", "wheat"},
		},
		{
			name:   "worcestershire hides anchovies",
			recipe: recipes.Recipe{Title: "Burger Sauce", Uses: []string{"Worcestershire sauce"}},
			want:   []string{"fish"},
		},
		// Dishes that name no ingredient still give theirs away.
		{name: "a frittata is eggs", recipe: recipes.Recipe{Title: "Veggie Frittata", Uses: []string{"Spinach"}}, want: []string{"eggs"}},
		{name: "pizza dough is wheat", recipe: recipes.Recipe{Title: "Flatbread", Uses: []string{"Pizza dough"}}, want: []string{"wheat"}},
		{name: "barley counts as wheat", recipe: recipes.Recipe{Title: "Barley Soup", Uses: []string{"Barley"}}, want: []string{"wheat"}},
		{name: "calamari is shellfish", recipe: recipes.Recipe{Title: "Fried Calamari", Uses: []string{"Squid"}}, want: []string{"shellfish"}},
		{name: "lox is fish", recipe: recipes.Recipe{Title: "Lox Bowl", Uses: []string{"Lox"}}, want: []string{"fish"}},
		// The lookalikes: food that reads like an allergen and is not one.
		{
			name:   "peanut butter is peanuts and not milk",
			recipe: recipes.Recipe{Title: "Peanut Butter Spoon", Uses: []string{"Peanut butter"}},
			want:   []string{"peanuts"},
		},
		{
			name:   "oat milk is not milk",
			recipe: recipes.Recipe{Title: "Overnight Oats", Uses: []string{"Oats", "Oat milk"}},
			want:   nil,
		},
		{
			name:   "coconut milk is not milk, nor a tree nut",
			recipe: recipes.Recipe{Title: "Curry", Uses: []string{"Coconut milk"}},
			want:   nil,
		},
		{
			name:   "eggplant is not eggs",
			recipe: recipes.Recipe{Title: "Roast Eggplant", Uses: []string{"Eggplant"}},
			want:   nil,
		},
		{
			name:   "corn tortillas and rice noodles are not wheat",
			recipe: recipes.Recipe{Title: "Street Tacos", Uses: []string{"Corn tortillas", "Rice noodles"}},
			want:   nil,
		},
		{
			name:   "nutmeg and butternut are not nuts",
			recipe: recipes.Recipe{Title: "Squash Soup", Uses: []string{"Butternut squash", "Nutmeg"}},
			want:   nil,
		},
		// Two separate foods never join into one phrase: Peanuts and Butter are milk after all.
		{
			name:   "two foods are not one phrase",
			recipe: recipes.Recipe{Title: "Snack", Uses: []string{"Peanuts", "Butter"}},
			want:   []string{"milk", "peanuts"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := recipes.Allergens(c.recipe)

			if !slices.Equal(got, c.want) {
				t.Errorf("allergens = %v, want %v", got, c.want)
			}
		})
	}
}

// TestDietsComeFromWhatTheDishContains: the diet filter runs in SQL on what this judged, so a meat
// dish on a vegan shelf would be this function's fault.
func TestDietsComeFromWhatTheDishContains(t *testing.T) {
	t.Parallel()
	every := []string{"pescatarian", "vegan", "vegetarian"}
	cases := []struct {
		name   string
		recipe recipes.Recipe
		want   []string
	}{
		{name: "fruit and water fit every diet", recipe: recipes.Recipe{Title: "Lemon Water", Uses: []string{"Lemon"}}, want: every},
		{name: "chicken fits none", recipe: recipes.Recipe{Title: "Fajita Bowl", Uses: []string{"Chicken", "Rice"}}, want: nil},
		{name: "ground beef fits none", recipe: recipes.Recipe{Title: "Tacos", Uses: []string{"Ground beef"}}, want: nil},
		{name: "meat in the title fits none", recipe: recipes.Recipe{Title: "Bacon Pasta", Uses: []string{"Pasta"}}, want: nil},
		{name: "pastrami fits none", recipe: recipes.Recipe{Title: "Deli Plate", Uses: []string{"Pastrami"}}, want: nil},
		{name: "bratwurst fits none", recipe: recipes.Recipe{Title: "Brats", Uses: []string{"Bratwursts"}}, want: nil},
		{name: "liver fits none", recipe: recipes.Recipe{Title: "Pate", Uses: []string{"Chicken livers"}}, want: nil},
		{name: "gelatin is meat", recipe: recipes.Recipe{Title: "Jello", Uses: []string{"Gelatin", "Sugar"}}, want: nil},
		{name: "salmon is pescatarian", recipe: recipes.Recipe{Title: "Salmon Rice", Uses: []string{"Salmon", "Rice"}}, want: []string{"pescatarian"}},
		{name: "shrimp is pescatarian", recipe: recipes.Recipe{Title: "Shrimp Rice", Uses: []string{"Shrimp", "Rice"}}, want: []string{"pescatarian"}},
		{name: "eggs and cheese are vegetarian", recipe: recipes.Recipe{Title: "Omelette", Uses: []string{"Eggs", "Cheddar"}}, want: []string{"pescatarian", "vegetarian"}},
		{name: "eggs alone are not vegan", recipe: recipes.Recipe{Title: "Fried Rice", Uses: []string{"Eggs", "Rice"}}, want: []string{"pescatarian", "vegetarian"}},
		{name: "yogurt is vegetarian", recipe: recipes.Recipe{Title: "Parfait", Uses: []string{"Greek yogurt"}}, want: []string{"pescatarian", "vegetarian"}},
		{name: "honey is not vegan", recipe: recipes.Recipe{Title: "Honey Toast", Uses: []string{"Bread", "Honey"}}, want: []string{"pescatarian", "vegetarian"}},
		{name: "peanut butter toast is vegan", recipe: recipes.Recipe{Title: "PB Toast", Uses: []string{"Bread", "Peanut butter"}}, want: every},
		{
			name:   "a declared milk is not vegan",
			recipe: recipes.Recipe{Title: "Mystery Sauce", Uses: []string{"Rice"}, Allergens: []string{"milk"}},
			want:   []string{"pescatarian", "vegetarian"},
		},
		{
			name:   "a declared fish is not vegetarian",
			recipe: recipes.Recipe{Title: "Mystery Broth", Uses: []string{"Rice"}, Allergens: []string{"fish"}},
			want:   []string{"pescatarian"},
		},
		{
			name: "meat in a step fits none",
			recipe: recipes.Recipe{
				Title: "Fried Rice", Uses: []string{"Rice"}, Steps: []string{"Add the diced ham."},
			},
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := recipes.Diets(c.recipe)

			if !slices.Equal(got, c.want) {
				t.Errorf("diets = %v, want %v", got, c.want)
			}
		})
	}
}

// TestTheChecksLeaveTheRecipeAlone: the handler stores the recipe it checked, so a check that
// sorted or trimmed the model's lists in place would store something nobody checked.
func TestTheChecksLeaveTheRecipeAlone(t *testing.T) {
	t.Parallel()
	recipe := recipes.Recipe{
		Title: "Cheese Omelette", Uses: []string{" Eggs", "Cheddar"}, Missing: []string{"Chives"},
		Allergens: []string{"milk", "Eggs"}, Steps: []string{"Whisk.", "Fold."},
	}
	before := recipes.Recipe{
		Title: recipe.Title, Uses: slices.Clone(recipe.Uses), Missing: slices.Clone(recipe.Missing),
		Allergens: slices.Clone(recipe.Allergens), Steps: slices.Clone(recipe.Steps),
	}

	recipes.Check(recipe, fridge, staples, []string{"peanuts"})
	recipes.Allergens(recipe)
	recipes.Diets(recipe)

	for name, pair := range map[string][2][]string{
		"uses":      {recipe.Uses, before.Uses},
		"missing":   {recipe.Missing, before.Missing},
		"allergens": {recipe.Allergens, before.Allergens},
		"steps":     {recipe.Steps, before.Steps},
	} {
		if !slices.Equal(pair[0], pair[1]) {
			t.Errorf("%s = %v after the checks, want %v as it came", name, pair[0], pair[1])
		}
	}
}
