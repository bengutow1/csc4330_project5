// Author: Caden Lund
package demo

import (
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// story.go is the demo week itself: who plays, what their fridges held, and when they gave up and
// went for fries. Each player's fridges score through packages/go/scoring like any scan, so every
// tier is earned. Most of this week happens on Monday and Tuesday, so its boards are busy early.

// slot is when something happened: a day of the week (0 is Monday) at a campus clock time, this
// week or the one before.
type slot struct {
	lastWeek     bool
	day          int
	hour, minute int
}

// this is a slot in the week the seed runs in.
func this(day, hour, minute int) slot {
	return slot{day: day, hour: hour, minute: minute}
}

// last is a slot in the week before.
func last(day, hour, minute int) slot {
	return slot{lastWeek: true, day: day, hour: hour, minute: minute}
}

// at is the slot as a time, given the Monday 00:00 that starts the seed's week. Dates are built
// from the calendar, so a week with a daylight saving change still lands on the clock time.
func (s slot) at(weekStart time.Time) time.Time {
	start := weekStart
	if s.lastWeek {
		start = weekStart.AddDate(0, 0, -7)
	}
	return time.Date(start.Year(), start.Month(), start.Day()+s.day, s.hour, s.minute, 0, 0, points.Campus())
}

// fridge is what one scan found, the line the model wrote about it, and the dish its scan offers
// (nil when nothing in it cooks).
type fridge struct {
	roast string
	items []repository.ScanItem
	dish  *repository.NewRecipe
}

// spot is a place a fry run went: the places mock's own ids and names.
type spot struct {
	id, name string
}

type scan struct {
	when   slot
	fridge fridge
}

type run struct {
	when slot
	spot spot
}

// plate is a cooked dish: the dish from one of the player's scans, by its place in their scans,
// plated at when.
type plate struct {
	when slot
	scan int
}

// player is one of the demo's regulars and their week. Plates come in the order they were cooked,
// as the streak counts them.
type player struct {
	handle, avatar string
	scans          []scan
	runs           []run
	plates         []plate
}

func item(name, category, quantity string) repository.ScanItem {
	return repository.ScanItem{Name: name, Category: category, Quantity: quantity}
}

// The fridges, best first. The comment on each is the score scoring.Fridge gives it.
var (
	// 100: the Fridge of the Week.
	stocked = fridge{
		roast: "Labeled containers and a plan. This fridge meal preps on Sundays.",
		items: []repository.ScanItem{
			item("Chicken thighs", repository.CategoryProtein, "1 pack"),
			item("Salmon", repository.CategoryProtein, "2 fillets"),
			item("Eggs", repository.CategoryProtein, "x12"),
			item("Spinach", repository.CategoryVegetable, "1 bag"),
			item("Bell peppers", repository.CategoryVegetable, "x3"),
			item("Broccoli", repository.CategoryVegetable, "1 head"),
			item("Brown rice", repository.CategoryGrain, "1 bag"),
			item("Greek yogurt", repository.CategoryDairy, "1 tub"),
			item("Cheddar", repository.CategoryDairy, ""),
			item("Blueberries", repository.CategoryFruit, "1 pint"),
		},
		dish: &repository.NewRecipe{
			Title: "Salmon rice bowl", Minutes: 25, Difficulty: repository.DifficultyMedium,
			Equipment: []string{repository.EquipmentStovetop},
			Uses:      []string{"Salmon", "Brown rice", "Spinach"},
			Allergens: []string{repository.AllergyFish},
			Diets:     []string{repository.DietPescatarian},
			Steps:     []string{"Cook the rice.", "Sear the salmon.", "Wilt the spinach and pile it all up."},
		},
	}
	// 89: Certified Chef, and a clear second to stocked.
	planned = fridge{
		roast: "Turkey, tortillas and salsa. Taco night was planned on purpose.",
		items: []repository.ScanItem{
			item("Ground turkey", repository.CategoryProtein, "1 lb"),
			item("Eggs", repository.CategoryProtein, "x6"),
			item("Zucchini", repository.CategoryVegetable, "x2"),
			item("Onions", repository.CategoryVegetable, "x2"),
			item("Tortillas", repository.CategoryGrain, "1 pack"),
			item("Salsa", repository.CategoryCondiment, "1 jar"),
		},
		dish: &repository.NewRecipe{
			Title: "Turkey tacos", Minutes: 20, Difficulty: repository.DifficultyEasy,
			Equipment: []string{repository.EquipmentStovetop},
			Uses:      []string{"Ground turkey", "Tortillas", "Onions", "Salsa"},
			Allergens: []string{repository.AllergyWheat},
			Steps:     []string{"Brown the turkey with the onions.", "Warm the tortillas.", "Fill and top with salsa."},
		},
	}
	// 68: Home Cook.
	breakfast = fridge{
		roast: "Eggs, spinach and good bread. Breakfast for dinner is ready.",
		items: []repository.ScanItem{
			item("Eggs", repository.CategoryProtein, "x4"),
			item("Spinach", repository.CategoryVegetable, "half a bag"),
			item("Sourdough", repository.CategoryGrain, "1 loaf"),
		},
		dish: &repository.NewRecipe{
			Title: "Spinach eggs on toast", Minutes: 10, Difficulty: repository.DifficultyEasy,
			Equipment: []string{repository.EquipmentStovetop},
			Uses:      []string{"Eggs", "Spinach", "Sourdough"},
			Allergens: []string{repository.AllergyEggs, repository.AllergyWheat},
			Diets:     []string{repository.DietVegetarian, repository.DietPescatarian},
			Steps:     []string{"Toast the bread.", "Scramble the eggs with the spinach.", "Pile it on the toast."},
		},
	}
	// 39: Scrappy.
	negotiation = fridge{
		roast: "Ramen, frozen peas and cold pizza. Dinner is a negotiation.",
		items: []repository.ScanItem{
			item("Instant ramen", repository.CategoryGrain, "x3"),
			item("Frozen peas", repository.CategoryVegetable, "1 bag"),
			item("Leftover pizza", repository.CategoryLeftover, "2 slices"),
			item("Hot sauce", repository.CategoryCondiment, ""),
		},
		dish: &repository.NewRecipe{
			Title: "Pea ramen", Minutes: 10, Difficulty: repository.DifficultyEasy,
			Equipment: []string{repository.EquipmentStovetop},
			Uses:      []string{"Instant ramen", "Frozen peas", "Hot sauce"},
			Allergens: []string{repository.AllergyWheat},
			Diets:     []string{repository.DietVegetarian, repository.DietPescatarian},
			Steps:     []string{"Boil the noodles with the peas.", "Season and finish with hot sauce."},
		},
	}
	// 10: Fries in the Bag.
	takeout = fridge{
		roast: "Half a takeout box and soy sauce packets. The fridge has given up.",
		items: []repository.ScanItem{
			item("Leftover lo mein", repository.CategoryLeftover, "half a box"),
			item("Soy sauce packets", repository.CategoryCondiment, "x9"),
			item("Sparkling water", repository.CategoryDrink, "x2"),
		},
		dish: &repository.NewRecipe{
			Title: "Crispy lo mein", Minutes: 10, Difficulty: repository.DifficultyEasy,
			Equipment: []string{repository.EquipmentStovetop},
			Uses:      []string{"Leftover lo mein", "Soy sauce packets"},
			Allergens: []string{repository.AllergyWheat, repository.AllergySoy},
			Steps:     []string{"Fry the noodles flat until crisp.", "Flip, then splash with soy sauce."},
		},
	}
	// 1: Fries in the Bag.
	museum = fridge{
		roast: "Ketchup, mustard and an energy drink. This fridge is a condiment museum.",
		items: []repository.ScanItem{
			item("Ketchup", repository.CategoryCondiment, ""),
			item("Mustard", repository.CategoryCondiment, ""),
			item("Energy drink", repository.CategoryDrink, "x1"),
			item("Ice", repository.CategoryOther, ""),
		},
	}
)

// The spots fry runs go to, from the places mock.
var (
	nightOwl  = spot{id: "mock-03", name: "Night Owl Fries"}
	drumstick = spot{id: "mock-05", name: "Drumstick Drive-Thru"}
	mainPizza = spot{id: "mock-06", name: "Main Street Pizza"}
	noodles   = spot{id: "mock-08", name: "Lucky Noodle House"}
	tacoTruck = spot{id: "mock-04", name: "Sunny Side Taco Truck"}
)

// players is the demo's fifteen regulars. The first is how a run knows the demo is already there.
// This week, chef_maya's Monday fridge is the clear Fridge of the Week and her plates lead Chefs,
// while meal_prep_sam's streak leads it all time. dorm_fridge_dan leads the Hall of Fries with 5
// runs, and ten more players have a run, so a newcomer's first run lands below the fold.
var players = []player{
	{
		handle: "chef_maya", avatar: "🍳",
		scans: []scan{{this(0, 9, 0), stocked}, {last(1, 18, 0), planned}},
		// Lunch and dinner every day this week: Chefs' leader this week.
		plates: []plate{
			{last(1, 19, 0), 1}, {last(2, 19, 0), 1}, {last(3, 19, 0), 1},
			{this(0, 12, 30), 0}, {this(0, 19, 0), 0}, {this(1, 12, 30), 0}, {this(1, 19, 0), 0},
			{this(2, 12, 30), 0},
		},
	},
	{
		handle: "gymbro_tyler", avatar: "💪",
		scans: []scan{{this(1, 7, 30), planned}, {last(0, 7, 0), stocked}},
		runs:  []run{{this(0, 20, 0), drumstick}, {this(1, 20, 30), drumstick}},
		plates: []plate{
			{last(0, 12, 0), 1}, {last(1, 12, 0), 1}, {last(2, 12, 0), 1}, {last(3, 12, 0), 1},
			{this(1, 12, 0), 0}, {this(2, 12, 0), 0},
		},
	},
	{
		handle: "meal_prep_sam", avatar: "🥡",
		scans: []scan{{this(0, 11, 0), planned}, {last(2, 12, 0), breakfast}},
		// A plate a day since last Wednesday: the longest streak, and all time's leader.
		plates: []plate{
			{last(2, 13, 0), 1}, {last(3, 13, 0), 1}, {last(4, 13, 0), 1}, {last(5, 13, 0), 1},
			{last(6, 13, 0), 1}, {this(0, 12, 0), 0}, {this(1, 12, 0), 0}, {this(2, 12, 0), 0},
		},
	},
	{
		handle: "ramen_king", avatar: "🍜",
		scans: []scan{{this(0, 13, 0), negotiation}, {last(3, 19, 0), negotiation}},
		runs: []run{
			{this(0, 19, 0), noodles}, {this(1, 12, 30), noodles}, {this(2, 13, 0), noodles},
			{last(4, 19, 30), noodles},
		},
		// Scrappy: a 39 fridge pays a bonus on top of each plate.
		plates: []plate{{this(0, 19, 30), 0}, {this(1, 19, 30), 0}},
	},
	{
		handle: "dorm_fridge_dan", avatar: "🍟",
		scans: []scan{
			{this(0, 14, 0), museum}, {this(1, 20, 0), takeout},
			{last(4, 22, 0), museum}, {last(5, 23, 30), takeout},
		},
		// Five meal windows: Monday's midday and late night, Tuesday's evening and late night (00:30
		// Wednesday is still Tuesday night), and Wednesday's midday.
		runs: []run{
			{this(0, 12, 30), nightOwl}, {this(0, 23, 45), nightOwl}, {this(1, 19, 0), nightOwl},
			{this(2, 0, 30), nightOwl}, {this(2, 12, 15), nightOwl},
			{last(4, 22, 30), nightOwl}, {last(5, 23, 45), nightOwl},
		},
		// Against all odds: one plate out of a 10 fridge.
		plates: []plate{{this(1, 21, 0), 1}},
	},
	{
		handle: "salad_szn", avatar: "🥗",
		scans:  []scan{{this(0, 12, 30), breakfast}, {last(6, 12, 0), breakfast}},
		plates: []plate{{this(0, 13, 0), 0}, {this(1, 13, 0), 0}},
	},
	{
		handle: "taco_tuesday", avatar: "🌮",
		scans:  []scan{{this(1, 18, 0), breakfast}},
		runs:   []run{{this(0, 12, 0), tacoTruck}, {this(1, 12, 0), tacoTruck}},
		plates: []plate{{this(1, 19, 0), 0}},
	},
	{
		handle: "pizza_pete", avatar: "🍕",
		scans: []scan{{this(1, 21, 0), takeout}, {last(5, 20, 0), museum}},
		runs: []run{
			{this(0, 12, 0), mainPizza}, {this(1, 19, 30), mainPizza}, {this(2, 12, 0), mainPizza},
		},
	},
	{
		handle: "cereal_for_dinner", avatar: "🥣",
		scans: []scan{{this(2, 8, 0), negotiation}},
		runs:  []run{{this(1, 8, 30), drumstick}},
	},
	{
		handle: "bagel_boss", avatar: "🥯",
		scans: []scan{{this(0, 8, 0), breakfast}},
		runs:  []run{{this(1, 9, 0), drumstick}},
	},
	{
		handle: "spice_queen", avatar: "🔥",
		scans:  []scan{{this(2, 17, 0), planned}},
		plates: []plate{{this(2, 18, 30), 0}},
	},
	{
		handle: "freezer_burn", avatar: "🧊",
		scans: []scan{{last(0, 19, 0), takeout}},
		runs: []run{
			{this(1, 22, 30), nightOwl},
			{last(0, 22, 30), nightOwl}, {last(1, 22, 30), nightOwl}, {last(2, 22, 30), nightOwl},
			{last(3, 22, 30), nightOwl}, {last(4, 12, 0), nightOwl}, {last(5, 12, 0), nightOwl},
		},
	},
	{
		handle: "leftover_larry", avatar: "🍲",
		scans: []scan{{this(1, 13, 0), negotiation}},
		runs:  []run{{this(0, 18, 0), noodles}},
	},
	{
		handle: "noodle_nora", avatar: "🍝",
		scans:  []scan{{this(2, 12, 0), breakfast}, {last(4, 13, 0), negotiation}},
		runs:   []run{{this(1, 18, 30), noodles}},
		plates: []plate{{last(4, 19, 0), 1}},
	},
	{
		handle: "midnight_mo", avatar: "🌙",
		scans: []scan{{this(1, 23, 30), museum}},
		runs: []run{
			{this(0, 23, 30), nightOwl}, {this(1, 23, 0), nightOwl},
			{last(2, 23, 30), nightOwl}, {last(3, 23, 30), nightOwl}, {last(4, 23, 0), nightOwl},
		},
	},
}
