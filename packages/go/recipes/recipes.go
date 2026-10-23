// Author: Caden Lund
// Package recipes holds a recipe the model wrote to the player's kitchen before anyone sees it:
// it may use only the scan's list and the player's staples, lack at most one thing it names, and
// contain none of the player's allergens. It also judges which diets a dish fits. It has no I/O,
// so the same recipe always gets the same answer.
package recipes

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// Recipe is what the checks read from one dish.
type Recipe struct {
	// Title is the dish's name, which can give an allergen away ("Spicy Peanut Bowl").
	Title string
	// Uses is the food it takes that the player has: names from the scan's list and the staples.
	Uses []string
	// Missing is the food it takes that the player lacks: nothing, or one thing.
	Missing []string
	// Allergens are what the dish says it contains, from the closed set of allergies.
	Allergens []string
	// Steps are the instructions, which can name food the lists left out ("top with sesame").
	Steps []string
}

// The allergies, spelled as docs/contracts.md spells them.
const (
	peanuts   = "peanuts"
	treeNuts  = "tree_nuts"
	milk      = "milk"
	eggs      = "eggs"
	wheat     = "wheat"
	soy       = "soy"
	fish      = "fish"
	shellfish = "shellfish"
	sesame    = "sesame"
)

// The diets besides none, which every dish fits.
const (
	pescatarian = "pescatarian"
	vegan       = "vegan"
	vegetarian  = "vegetarian"
)

// words is how one kind of food gives itself away in a recipe. Each phrase is lowercase and
// singular, and a plural ending in s or es matches it too. A lookalike reads like the food and is
// not it (peanut butter holds no milk), so the search passes over it.
type words struct {
	phrases    []string
	lookalikes []string
}

// allergenWords is each allergen's words. A match errs toward the allergen: a needless drop costs
// the player one suggestion, and a missed allergen costs far more.
var allergenWords = map[string]words{
	peanuts: {phrases: []string{"groundnut", "peanut"}},
	treeNuts: {
		phrases: []string{"almond", "cashew", "chestnut", "hazelnut", "macadamia", "marzipan",
			"nut", "nutella", "pecan", "pistachio", "praline", "walnut"},
		lookalikes: []string{"water chestnut"},
	},
	milk: {
		phrases: []string{"brie", "butter", "buttermilk", "casein", "cheddar", "cheese", "cream",
			"custard", "feta", "ghee", "gouda", "half and half", "mascarpone", "milk", "mozzarella",
			"paneer", "parmesan", "queso", "ricotta", "whey", "yoghurt", "yogurt"},
		lookalikes: []string{"almond butter", "almond milk", "apple butter", "cashew butter",
			"cashew milk", "cocoa butter", "coconut cream", "coconut milk", "cream of tartar",
			"nut butter", "oat milk", "peanut butter", "rice milk", "soy milk"},
	},
	eggs: {
		phrases: []string{"aioli", "brioche", "carbonara", "custard", "egg", "eggnog",
			"french toast", "frittata", "hollandaise", "mayo", "mayonnaise", "meringue", "omelet",
			"omelette", "quiche"},
	},
	// Barley and rye are not wheat, but a reaction to one can come with the others, so they count.
	wheat: {
		phrases: []string{"bagel", "barley", "biscuit", "bread", "breadcrumb", "brioche", "bun",
			"burrito", "cake", "cookie", "couscous", "cracker", "croissant", "crouton", "crust",
			"dough", "dumpling", "fettuccine", "flour", "gnocchi", "lasagna", "linguine", "macaroni",
			"muffin", "noodle", "pancake", "panko", "pasta", "pastries", "pastry", "penne", "pie",
			"pita", "pizza", "pretzel", "ramen", "ravioli", "rye", "sandwich", "seitan", "semolina",
			"soy sauce", "spaghetti", "toast", "tortellini", "tortilla", "udon", "waffle", "wheat"},
		lookalikes: []string{"corn tortilla", "glass noodle", "rice noodle"},
	},
	soy: {phrases: []string{"edamame", "miso", "soy", "soya", "tamari", "tempeh", "tofu"}},
	fish: {
		phrases: []string{"anchovies", "anchovy", "catfish", "caviar", "cod", "eel", "fish",
			"haddock", "halibut", "herring", "lox", "mackerel", "mahi", "pollock", "salmon",
			"sardine", "sashimi", "snapper", "swordfish", "tilapia", "trout", "tuna",
			"worcestershire"},
	},
	// Squid and octopus are molluscs, which a shellfish allergy often takes in.
	shellfish: {
		phrases: []string{"calamari", "clam", "crab", "crawfish", "crayfish", "lobster", "mussel",
			"octopus", "oyster", "prawn", "scallop", "shrimp", "squid"},
	},
	sesame: {phrases: []string{"halva", "sesame", "tahini"}},
}

// meatWords rule out every diet but none. Fish and shellfish are not here: as allergens, they
// already rule out vegetarian and vegan.
var meatWords = words{
	phrases: []string{"bacon", "barbacoa", "beef", "birria", "bison", "bologna", "bratwurst",
		"brisket", "burger", "carne", "carnitas", "chicken", "chorizo", "duck", "frankfurter",
		"gelatin", "gelatine", "goose", "gyro", "ham", "hamburger", "hot dog", "jerky", "kebab",
		"kielbasa", "lamb", "lard", "liver", "meat", "meatball", "meatloaf", "mutton", "oxtail",
		"pancetta", "pastrami", "pepperoni", "pollo", "pork", "prosciutto", "rib", "ribeye",
		"salami", "sausage", "shawarma", "sirloin", "spam", "steak", "tenderloin", "turkey",
		"veal", "venison"},
	lookalikes: []string{"bean burger", "veggie burger"},
}

// honeyWords rule out vegan alone.
var honeyWords = words{phrases: []string{"honey"}}

// Check keeps recipe when the player can cook it: everything it uses is on items or among
// staples, it lacks at most one thing and names it in Missing, and it contains none of
// allergies, by its own word or by what its food gives away. Names match whatever their case and
// spacing. When it is dropped, why says what failed, for the server log.
func Check(recipe Recipe, items, staples, allergies []string) (kept bool, why string) {
	// 1. At most one thing the player lacks
	if len(recipe.Missing) > 1 {
		return false, fmt.Sprintf("lacks %d things (%s), and a recipe may lack 1",
			len(recipe.Missing), strings.Join(recipe.Missing, ", "))
	}

	// 2. Everything else it takes, the player has. Food on neither list was never declared.
	have := make(map[string]bool, len(items)+len(staples))
	for _, name := range slices.Concat(items, staples) {
		have[fold(name)] = true
	}
	for _, name := range recipe.Uses {
		if !have[fold(name)] {
			return false, fmt.Sprintf("uses %s, which is on neither the list nor the staples",
				strings.TrimSpace(name))
		}
	}

	// 3. None of the player's allergens, by its word or by its food
	text := wordsOf(recipe)
	for _, allergy := range allergies {
		allergen := fold(allergy)
		if declares(recipe, allergen) {
			return false, fmt.Sprintf("declares %s, one of the player's allergies", allergen)
		}
		w, known := allergenWords[allergen]
		if known && w.in(text) {
			return false, fmt.Sprintf("its food gives away %s, one of the player's allergies", allergen)
		}
	}
	return true, ""
}

// Allergens is every allergen recipe contains, sorted: those it declares, and those its title,
// food and steps give away (cheese is milk). A declared word outside the closed set is not one.
func Allergens(recipe Recipe) []string {
	text := wordsOf(recipe)
	var found []string
	for allergen, w := range allergenWords {
		if declares(recipe, allergen) || w.in(text) {
			found = append(found, allergen)
		}
	}
	slices.Sort(found)
	return found
}

// Diets is every diet besides none that recipe fits, sorted, judged from what it contains:
// pescatarian rules out meat, vegetarian rules out fish and shellfish too, and vegan rules out
// milk, eggs and honey as well.
func Diets(recipe Recipe) []string {
	contains := Allergens(recipe)
	text := wordsOf(recipe)
	meatFree := !meatWords.in(text)
	vegetarianFits := meatFree && !slices.Contains(contains, fish) &&
		!slices.Contains(contains, shellfish)
	veganFits := vegetarianFits && !slices.Contains(contains, milk) &&
		!slices.Contains(contains, eggs) && !honeyWords.in(text)

	var fits []string
	if meatFree {
		fits = append(fits, pescatarian)
	}
	if veganFits {
		fits = append(fits, vegan)
	}
	if vegetarianFits {
		fits = append(fits, vegetarian)
	}
	return fits
}

// declares reports whether recipe lists allergen among its own.
func declares(recipe Recipe, allergen string) bool {
	return slices.ContainsFunc(recipe.Allergens, func(a string) bool { return fold(a) == allergen })
}

// fold is a name as the checks compare it: lowercase, its words single-spaced.
func fold(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// wordsOf is every word recipe names, lowercase, with an empty word after each of its strings,
// so a phrase never runs from one food into the next.
func wordsOf(recipe Recipe) []string {
	var text []string
	for _, s := range slices.Concat([]string{recipe.Title}, recipe.Uses, recipe.Missing, recipe.Steps) {
		text = append(text, strings.FieldsFunc(strings.ToLower(s), notLetter)...)
		text = append(text, "")
	}
	return text
}

func notLetter(r rune) bool {
	return !unicode.IsLetter(r)
}

// in reports whether text names this food anywhere outside its lookalikes.
func (w words) in(text []string) bool {
	text = slices.Clone(text)
	for _, phrase := range w.lookalikes {
		blank(text, strings.Fields(phrase))
	}
	for _, phrase := range w.phrases {
		p := strings.Fields(phrase)
		for i := range text {
			if startsAt(text, i, p) {
				return true
			}
		}
	}
	return false
}

// blank empties every place phrase appears in text, so no other phrase can match its words.
func blank(text, phrase []string) {
	for i := range text {
		if startsAt(text, i, phrase) {
			for j := range phrase {
				text[i+j] = ""
			}
		}
	}
}

// startsAt reports whether phrase appears in text from i on, each word as written or with a
// plural s or es.
func startsAt(text []string, i int, phrase []string) bool {
	if i+len(phrase) > len(text) {
		return false
	}
	for j, word := range phrase {
		got := text[i+j]
		if got != word && got != word+"s" && got != word+"es" {
			return false
		}
	}
	return true
}
