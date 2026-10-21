// Author: Caden Lund
// Package vision is the API's one line to the vision model, which reads a kitchen's photos into
// an ingredient list and writes the roast. It returns what the model said and decides nothing:
// the caller validates the list, and packages/go/scoring turns it into the score, so the model
// never sets a number we store. Without a key the API serves Mock instead.
package vision

// Item is one food the model spotted, in its own words. The caller checks every field like a
// request body before trusting it.
type Item struct {
	// Name is what the food is ("Greek yogurt").
	Name string
	// Category is meant to be one of the closed set in docs/contracts.md: protein, vegetable,
	// fruit, grain, dairy, condiment, drink, snack, leftover or other.
	Category string
	// Quantity is free text such as "x4" or "1 bag", and "" when the model could not tell.
	Quantity string
}

// Fridge is what the model read from one scan's photos.
type Fridge struct {
	// Items is everything spotted, in the model's order. Empty when the photos show no food.
	Items []Item
	// Roast is the model's line about the fridge, never the person, at the level asked for.
	Roast string
}

// RoastLevel is how hard the roast hits, picked by the player in their kitchen settings.
type RoastLevel string

// The three roast levels, spelled as the closed set in docs/contracts.md spells them.
const (
	RoastMild        RoastLevel = "mild"
	RoastMedium      RoastLevel = "medium"
	RoastExtraCrispy RoastLevel = "extra_crispy"
)
