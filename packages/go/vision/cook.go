// Author: Caden Lund
package vision

// Kitchen is what the model's recipes must respect, and all it learns about the player
// (go-vendors.md). Each field holds values from its closed set in docs/contracts.md.
type Kitchen struct {
	// Equipment is microwave, stovetop, oven or air_fryer.
	Equipment []string
	// Diet is none, vegetarian, vegan or pescatarian.
	Diet string
	// Allergies are peanuts, tree_nuts, milk, eggs, wheat, soy, fish, shellfish or sesame.
	Allergies []string
}

// Recipe is one dish the model suggests for a scan's list. Like an Item, it is the model's claim:
// the server checks what it uses, what it misses and its allergens before anyone sees it.
type Recipe struct {
	Title string
	// Minutes is the whole time to the plate.
	Minutes int
	// Difficulty is easy, medium or hard.
	Difficulty string
	// Equipment is what the dish needs, from Kitchen's set. Empty needs none.
	Equipment []string
	// Uses is the food the dish takes from the list, by the list's names, and from the staples.
	Uses []string
	// Missing is the food it takes that is neither: nothing, or the one item a "one item away"
	// recipe needs. It is not in Uses.
	Missing []string
	// Allergens are what the dish contains, from Kitchen's set of allergies.
	Allergens []string
	// Steps are the instructions, in order.
	Steps []string
	// StepTimers aligns with Steps: seconds, 0 for no timer; empty means all untimed.
	StepTimers []int
}

// PlateCheck is the model's word on a plate photo. The server decides what a pass earns.
type PlateCheck struct {
	// Passed says the photo shows food that matches the recipe.
	Passed bool
	// Reason is the model's few words on a failed check, for the server log. It is never shown
	// to the player as written.
	Reason string
}
