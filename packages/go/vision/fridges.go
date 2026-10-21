// Author: Caden Lund
package vision

import "fmt"

// mockFridge is one canned answer: the name that pins it, what the model would list, and its
// roast at each level.
type mockFridge struct {
	name   string
	items  []Item
	roasts roasts
}

// roasts is a fridge's line at each roast level.
type roasts struct {
	mild, medium, extraCrispy string
}

// at is the line for level. A level outside the three is the caller's bug, refused rather than
// quietly served as medium.
func (r roasts) at(level RoastLevel) (string, error) {
	switch level {
	case RoastMild:
		return r.mild, nil
	case RoastMedium:
		return r.medium, nil
	case RoastExtraCrispy:
		return r.extraCrispy, nil
	default:
		return "", fmt.Errorf("vision: unknown roast level %q", level)
	}
}

// mockFridges is every canned fridge, in the order the mock serves them: one per tier, best
// first, then an empty one. They are built from the concept's sample verdicts, whose lines are
// the medium roasts, and every roast is about the fridge, never the person. Read-only: reads hand
// out copies.
var mockFridges = []mockFridge{
	{
		name: "certified_chef",
		items: []Item{
			{Name: "Chicken", Category: "protein", Quantity: "x2"},
			{Name: "Spinach", Category: "vegetable", Quantity: "1 bag"},
			{Name: "Bell peppers", Category: "vegetable", Quantity: "x3"},
			{Name: "Greek yogurt", Category: "dairy", Quantity: "1 tub"},
			{Name: "Eggs", Category: "protein", Quantity: "x12"},
			{Name: "Cheddar", Category: "dairy", Quantity: "1 block"},
		},
		roasts: roasts{
			mild:        "Stocked, fresh and organized. This fridge is ready for anything.",
			medium:      "Fresh veggies, real protein, labeled leftovers. Your fridge has its life together.",
			extraCrispy: "Fully stocked and not one science experiment in sight. This fridge is showing off.",
		},
	},
	{
		name: "home_cook",
		items: []Item{
			{Name: "Ground beef", Category: "protein", Quantity: "1 lb"},
			{Name: "Tortillas", Category: "grain", Quantity: "x10"},
			{Name: "Onion", Category: "vegetable", Quantity: "x1"},
			{Name: "Shredded cheese", Category: "dairy", Quantity: "1 bag"},
			{Name: "Salsa", Category: "condiment", Quantity: "1 jar"},
			{Name: "Orange juice", Category: "drink", Quantity: "half a carton"},
		},
		roasts: roasts{
			mild:        "Plenty to work with in here. A good dinner is waiting to happen.",
			medium:      "There's a real dinner in here. Go find it.",
			extraCrispy: "Respectable. Nothing in here will win an award, and nothing in here will hurt anyone either.",
		},
	},
	{
		name: "scrappy",
		items: []Item{
			{Name: "Instant ramen", Category: "grain", Quantity: "x3"},
			{Name: "Frozen peas", Category: "vegetable", Quantity: "half a bag"},
			{Name: "Hot sauce", Category: "condiment", Quantity: ""},
			{Name: "Apple", Category: "fruit", Quantity: "x1"},
			{Name: "Soda", Category: "drink", Quantity: "x2"},
		},
		roasts: roasts{
			mild:        "Not a lot in here, but there's a meal for anyone willing to get creative.",
			medium:      "It's giving creative. We can work with this.",
			extraCrispy: "Ramen, frozen peas and hot sauce. This fridge is running on sodium and optimism.",
		},
	},
	{
		name: "fries_in_the_bag",
		items: []Item{
			{Name: "Ketchup", Category: "condiment", Quantity: ""},
			{Name: "Egg", Category: "protein", Quantity: "x1"},
			{Name: "Lemon", Category: "fruit", Quantity: "x1"},
			{Name: "Energy drinks", Category: "drink", Quantity: "x4"},
			{Name: "Mystery container", Category: "other", Quantity: ""},
		},
		roasts: roasts{
			mild:        "Slim pickings in here. Tonight might be a fries night.",
			medium:      "Ketchup, one egg and a lemon from last semester. Put the fries in the bag.",
			extraCrispy: "Four energy drinks, one egg and a container nobody should open. This fridge has given up. Put the fries in the bag.",
		},
	},
	{
		// Nothing spotted at all, which is a verdict too: score 0.
		name: "empty",
		roasts: roasts{
			mild:        "Nothing in here to judge yet. Try a closer photo.",
			medium:      "Either your fridge is empty or the photo is.",
			extraCrispy: "Just shelves and cold air. This fridge is mostly a light bulb.",
		},
	},
}
