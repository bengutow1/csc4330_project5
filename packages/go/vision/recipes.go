// Author: Caden Lund
package vision

import (
	"slices"
	"strings"
)

// mockRecipes is every mock fridge's recipes, by the fridge's name, as the model would suggest
// them. Two are there for the server to sort: Sad Egg on Toast is one item away, and Steak Frites
// uses food no list holds without declaring it, so the recipe check must drop it. The empty
// fridge has none. Read-only: Recipes hands out copies.
var mockRecipes = map[string][]Recipe{
	"certified_chef": {
		{
			Title: "Chicken Fajita Bowl", Minutes: 25, Difficulty: "medium", Equipment: []string{"stovetop"},
			Uses: []string{"Chicken", "Bell peppers", "Rice", "Oil", "Spices"},
			Steps: []string{
				"Cook the rice.",
				"Slice the chicken and the bell peppers into strips.",
				"Sear the chicken in oil with the spices until cooked through, about 8 minutes.",
				"Add the peppers for 3 more minutes, then serve it all over the rice.",
			},
		},
		{
			Title: "Spinach Cheddar Omelette", Minutes: 10, Difficulty: "easy", Equipment: []string{"stovetop"},
			Uses:      []string{"Eggs", "Spinach", "Cheddar", "Butter", "Salt", "Pepper"},
			Allergens: []string{"eggs", "milk"},
			Steps: []string{
				"Whisk three eggs with a pinch of salt and pepper.",
				"Wilt a handful of spinach in butter.",
				"Pour in the eggs and cook until almost set.",
				"Scatter the cheddar over half, fold, and slide it onto a plate.",
			},
		},
		{
			Title: "Yogurt Parfait", Minutes: 5, Difficulty: "easy",
			Uses: []string{"Greek yogurt", "Sugar", "Spices"}, Allergens: []string{"milk"},
			Steps: []string{
				"Spoon half the yogurt into a glass.",
				"Sprinkle on a little sugar and a pinch of cinnamon.",
				"Add the rest of the yogurt and finish with more of both.",
			},
		},
	},
	"home_cook": {
		{
			Title: "Beef Tacos", Minutes: 20, Difficulty: "easy", Equipment: []string{"stovetop"},
			Uses:      []string{"Ground beef", "Tortillas", "Onion", "Shredded cheese", "Salsa", "Oil", "Spices"},
			Allergens: []string{"milk", "wheat"},
			Steps: []string{
				"Dice the onion and soften it in a little oil.",
				"Brown the ground beef with the onion and the spices.",
				"Warm the tortillas in a dry pan.",
				"Fill them with the beef, then top with cheese and salsa.",
			},
		},
		{
			Title: "Cheese Quesadilla", Minutes: 10, Difficulty: "easy", Equipment: []string{"stovetop"},
			Uses:      []string{"Tortillas", "Shredded cheese", "Salsa", "Butter"},
			Allergens: []string{"milk", "wheat"},
			Steps: []string{
				"Butter one side of two tortillas.",
				"Lay one butter side down in a pan and cover it with cheese.",
				"Top it with the other tortilla and cook until both sides are golden.",
				"Cut it into wedges and serve it with salsa.",
			},
		},
		{
			Title: "Steak Frites", Minutes: 40, Difficulty: "hard", Equipment: []string{"stovetop", "oven"},
			Uses: []string{"Steak", "Potatoes", "Oil", "Salt"},
			Steps: []string{
				"Cut the potatoes into fries, toss them in oil and salt, and bake them until crisp.",
				"Sear the steak in a hot pan, 4 minutes a side.",
				"Rest the steak, slice it and serve it with the fries.",
			},
		},
	},
	"scrappy": {
		{
			Title: "Spicy Pea Ramen", Minutes: 6, Difficulty: "easy", Equipment: []string{"microwave"},
			Uses: []string{"Instant ramen", "Frozen peas", "Hot sauce"}, Allergens: []string{"wheat", "soy"},
			Steps: []string{
				"Break the ramen into a microwave-safe bowl with a handful of peas.",
				"Cover it with water and microwave for 4 minutes.",
				"Stir in the seasoning and as much hot sauce as you dare.",
			},
		},
		{
			Title: "Microwave Baked Apple", Minutes: 5, Difficulty: "easy", Equipment: []string{"microwave"},
			Uses: []string{"Apple", "Butter", "Sugar", "Spices"}, Allergens: []string{"milk"},
			Steps: []string{
				"Core the apple and stand it in a bowl.",
				"Fill the middle with butter, sugar and a pinch of cinnamon.",
				"Microwave it for 3 minutes, until soft.",
			},
		},
	},
	"fries_in_the_bag": {
		{
			Title: "Sad Egg on Toast", Minutes: 8, Difficulty: "easy", Equipment: []string{"stovetop"},
			Uses: []string{"Egg", "Butter", "Salt"}, Missing: []string{"Bread"},
			Allergens: []string{"eggs", "milk", "wheat"},
			Steps: []string{
				"Toast a slice of bread.",
				"Fry the egg in butter with a pinch of salt.",
				"Slide the egg onto the toast. It is dinner now.",
			},
		},
		{
			Title: "Lemon Water (Fancy)", Minutes: 2, Difficulty: "easy",
			Uses: []string{"Lemon"},
			Steps: []string{
				"Slice the lemon.",
				"Drop a slice into a tall glass of cold water.",
				"Sip it slowly. It is fancy now.",
			},
		},
	},
}

// recipesFor is the recipes of the mock fridge whose list shares the most names with items,
// case aside, the first fridge winning a tie, and none when nothing is shared. So a list the
// player edited still finds its fridge, and the server's checks decide what still fits.
func recipesFor(items []Item) []Recipe {
	best, bestShared := "", 0
	for _, f := range mockFridges {
		shared := 0
		for _, have := range f.items {
			if slices.ContainsFunc(items, func(it Item) bool { return strings.EqualFold(it.Name, have.Name) }) {
				shared++
			}
		}
		if shared > bestShared {
			best, bestShared = f.name, shared
		}
	}
	return cloneRecipes(mockRecipes[best])
}

// cloneRecipes copies recipes down to their lists, so a caller that edits what it got cannot edit
// what every later call serves.
func cloneRecipes(rs []Recipe) []Recipe {
	if len(rs) == 0 {
		return nil
	}
	out := make([]Recipe, len(rs))
	for i, r := range rs {
		r.Equipment = slices.Clone(r.Equipment)
		r.Uses = slices.Clone(r.Uses)
		r.Missing = slices.Clone(r.Missing)
		r.Allergens = slices.Clone(r.Allergens)
		r.Steps = slices.Clone(r.Steps)
		out[i] = r
	}
	return out
}
