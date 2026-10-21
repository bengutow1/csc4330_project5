// Author: Caden Lund
package handlers_test

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1/scanv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/cook"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/scan"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// kitchenFridge is the list the fake eyes read: enough for every shelf and filter below.
var kitchenFridge = vision.Fridge{
	Items: []vision.Item{
		{Name: "Eggs", Category: "protein", Quantity: "x6"},
		{Name: "Spinach", Category: "vegetable", Quantity: "1 bag"},
		{Name: "Cheddar", Category: "dairy", Quantity: ""},
		{Name: "Chicken", Category: "protein", Quantity: "x2"},
		{Name: "Lemon", Category: "fruit", Quantity: "x1"},
	},
	Roast: "A fridge with options. Suspicious.",
}

// The fake chef's recipes for kitchenFridge, in its order: five the player can cook, one of them
// lacking bread, and Steak Frites, which uses food the list never had.
var (
	omeletteRecipe = vision.Recipe{
		Title: "Spinach Cheddar Omelette", Minutes: 15, Difficulty: "easy",
		Equipment: []string{"stovetop"}, Uses: []string{"Eggs", "Spinach", "Cheddar", "Butter"},
		Allergens: []string{"eggs", "milk"}, Steps: []string{"Whisk the eggs.", "Fold in the rest."},
	}
	eggToastRecipe = vision.Recipe{
		Title: "Egg on Toast", Minutes: 10, Difficulty: "easy", Equipment: []string{"stovetop"},
		Uses: []string{"Eggs", "Butter"}, Missing: []string{"Bread"},
		Allergens: []string{"eggs", "milk", "wheat"}, Steps: []string{"Toast the bread.", "Fry the egg."},
	}
	lemonWaterRecipe = vision.Recipe{
		Title: "Lemon Water (Fancy)", Minutes: 2, Difficulty: "easy", Uses: []string{"Lemon"},
		Steps: []string{"Squeeze the lemon into cold water."},
	}
	steakRecipe = vision.Recipe{
		Title: "Steak Frites", Minutes: 40, Difficulty: "hard",
		Equipment: []string{"stovetop", "oven"}, Uses: []string{"Steak", "Potatoes", "Oil"},
		Steps: []string{"Sear the steak.", "Bake the fries."},
	}
	chickenRecipe = vision.Recipe{
		Title: "Lemon Chicken Bake", Minutes: 45, Difficulty: "medium", Equipment: []string{"oven"},
		Uses: []string{"Chicken", "Lemon", "Oil", "Salt"}, Steps: []string{"Season.", "Bake."},
	}
	spinachRecipe = vision.Recipe{
		Title: "Microwave Spinach Rice", Minutes: 5, Difficulty: "easy",
		Equipment: []string{"microwave"}, Uses: []string{"Spinach", "Rice", "Salt"},
		Steps: []string{"Microwave the rice.", "Stir in the spinach."},
	}
	kitchenRecipes = []vision.Recipe{
		omeletteRecipe, eggToastRecipe, lemonWaterRecipe, steakRecipe, chickenRecipe, spinachRecipe,
	}
)

// TestEachMockFridgeCooksItsRecipes: with no key the API cooks from the mock, so every fridge must
// reach its shelves through the real checks, with what plating pays for its score: the cook, plus
// the bonus a bad fridge earns.
func TestEachMockFridgeCooksItsRecipes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pin     string
		now     []string
		away    []string
		preview int32
	}{
		{
			pin:     "certified_chef",
			now:     []string{"Chicken Fajita Bowl", "Spinach Cheddar Omelette", "Yogurt Parfait"},
			preview: 100,
		},
		{pin: "home_cook", now: []string{"Beef Tacos", "Cheese Quesadilla"}, preview: 100},
		{pin: "scrappy", now: []string{"Spicy Pea Ramen", "Microwave Baked Apple"}, preview: 150},
		{
			pin: "fries_in_the_bag", now: []string{"Lemon Water (Fancy)"},
			away: []string{"Sad Egg on Toast"}, preview: 200,
		},
		{pin: "empty"},
	}
	for _, c := range cases {
		t.Run(c.pin, func(t *testing.T) {
			t.Parallel()
			mock, err := vision.NewMock(c.pin, "")
			if err != nil {
				t.Fatalf("new mock: %v", err)
			}
			srv := cookStack(t, cookFixture{eyes: mock, chef: mock})
			maya := cookPlayerFor(t, srv, "test_maya")
			verdict := scanFridge(t, maya)

			got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

			now := titlesOf(got.GetMakeItNow())
			if !slices.Equal(now, c.now) {
				t.Errorf("make it now = %v, want %v", now, c.now)
			}
			away := titlesOf(got.GetOneItemAway())
			if !slices.Equal(away, c.away) {
				t.Errorf("one item away = %v, want %v", away, c.away)
			}
			for _, r := range slices.Concat(got.GetMakeItNow(), got.GetOneItemAway()) {
				if r.GetPointsPreview() != c.preview {
					t.Errorf("%s points preview = %d, want %d", r.GetTitle(), r.GetPointsPreview(), c.preview)
				}
			}
		})
	}
}

// TestTheMocksInvalidRecipeNeverAppears: Steak Frites uses steak and potatoes, which the
// home_cook list does not have, so it must be dropped before any shelf, with why in the log.
func TestTheMocksInvalidRecipeNeverAppears(t *testing.T) {
	t.Parallel()
	mock, err := vision.NewMock("home_cook", "")
	if err != nil {
		t.Fatalf("new mock: %v", err)
	}
	var logs lockedBuffer
	srv := cookStack(t, cookFixture{eyes: mock, chef: mock, log: jsonLog(&logs)})
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	for _, r := range slices.Concat(got.GetMakeItNow(), got.GetOneItemAway()) {
		if r.GetTitle() == "Steak Frites" {
			t.Errorf("Steak Frites reached a shelf: %v", r)
		}
	}
	stored := count(t, srv, `select count(*) from app.recipe where title = 'Steak Frites'`)
	if stored != 0 {
		t.Errorf("Steak Frites stored %d times, want never", stored)
	}
	if !strings.Contains(logs.String(), "Steak") {
		t.Errorf("log = %s, want why Steak Frites was dropped", logs.String())
	}
}

// TestTheShelvesSplitByWhatThePlayerLacks: a recipe that lacks one thing goes on One item away,
// and every other the player can cook goes on Make it now, in the model's order.
func TestTheShelvesSplitByWhatThePlayerLacks(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	wantNow := []string{omeletteRecipe.Title, lemonWaterRecipe.Title, chickenRecipe.Title, spinachRecipe.Title}
	now := titlesOf(got.GetMakeItNow())
	if !slices.Equal(now, wantNow) {
		t.Errorf("make it now = %v, want %v", now, wantNow)
	}
	away := got.GetOneItemAway()
	if len(away) != 1 {
		t.Fatalf("one item away = %v, want the egg toast alone", titlesOf(away))
	}
	toast := away[0]
	want := &cookv1.Recipe{
		RecipeId: toast.GetRecipeId(), Title: "Egg on Toast", Minutes: 10,
		Difficulty: cookv1.Difficulty_DIFFICULTY_EASY,
		Equipment:  []accountv1.Equipment{accountv1.Equipment_EQUIPMENT_STOVETOP},
		Uses:       []string{"Eggs", "Butter"}, Missing: []string{"Bread"},
		Steps:         []string{"Toast the bread.", "Fry the egg."},
		PointsPreview: verdictPreview(verdict),
	}
	if toast.GetRecipeId() == 0 || !proto.Equal(toast, want) {
		t.Errorf("egg toast = %v, want %v", toast, want)
	}
}

// TestGetRecipeReadsTheRecipeOnTheCard: the recipe page must show exactly what the card showed.
func TestGetRecipeReadsTheRecipeOnTheCard(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)
	shelves := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	for _, card := range slices.Concat(shelves.GetMakeItNow(), shelves.GetOneItemAway()) {
		got, err := maya.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{
			RecipeId: card.GetRecipeId(),
		}))
		if err != nil {
			t.Fatalf("get recipe %s: %v", card.GetTitle(), err)
		}
		if !proto.Equal(got.Msg.GetRecipe(), card) {
			t.Errorf("get recipe = %v, want the card %v", got.Msg.GetRecipe(), card)
		}
	}

	_, err := maya.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{
		RecipeId: firstOn(t, shelves.GetMakeItNow()).GetRecipeId() + 1000,
	}))
	code := connect.CodeOf(err)
	if code != connect.CodeNotFound {
		t.Errorf("a recipe that is not there: code %v (%v), want not_found", code, err)
	}
	_, err = maya.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{}))
	got := fieldViolations(t, err)
	want := "recipe id is required"
	if got["recipe_id"] != want || len(got) != 1 {
		t.Errorf("no recipe id: violations %v, want recipe_id: %q", got, want)
	}
}

// TestAnotherPlayersScanOrRecipeIsNotFound: every read reaches the owner through the scan, so
// another player's scan and recipe answer as missing ones do, and asking spends nothing of theirs.
func TestAnotherPlayersScanOrRecipeIsNotFound(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	sam := cookPlayerFor(t, srv, "test_sam")
	mine := scanFridge(t, maya)
	shelves := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: mine.GetScanId()})
	untouched := scanFridge(t, maya) // never listed, so its set is not made yet
	theirs := firstOn(t, shelves.GetMakeItNow()).GetRecipeId()

	rpcs := map[string]func() error{
		"ListRecipes on a scan with recipes": func() error {
			_, err := sam.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: mine.GetScanId()}))
			return err
		},
		"ListRecipes on a scan without": func() error {
			_, err := sam.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: untouched.GetScanId()}))
			return err
		},
		"GetRecipe": func() error {
			_, err := sam.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: theirs}))
			return err
		},
	}
	for name, call := range rpcs {
		err := call()
		code := connect.CodeOf(err)
		if code != connect.CodeNotFound {
			t.Errorf("%s: code %v (%v), want not_found", name, code, err)
		}
	}

	asked := len(chef.calls())
	if asked != 1 {
		t.Errorf("the model was asked %d times, want once, for maya's own list", asked)
	}
	stored := count(t, srv, fmt.Sprintf(`select count(*) from app.recipe where scan_id = %d`, untouched.GetScanId()))
	if stored != 0 {
		t.Errorf("%d recipes made for maya's untouched scan, want none", stored)
	}
	again := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: mine.GetScanId()})
	if !proto.Equal(again, shelves) {
		t.Errorf("maya's shelves = %v after sam's tries, want %v", again, shelves)
	}
}

// TestEachFilterNarrowsTheShelvesOnTheServer: a narrowed shelf is the server's answer from the
// stored set, and narrowing it again never asks the model.
func TestEachFilterNarrowsTheShelvesOnTheServer(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})
	t.Cleanup(func() {
		asked := len(chef.calls())
		if asked != 1 {
			t.Errorf("the model was asked %d times, want once for every filter", asked)
		}
	})
	var (
		omelette, toast, lemon = omeletteRecipe.Title, eggToastRecipe.Title, lemonWaterRecipe.Title
		chicken, spinach       = chickenRecipe.Title, spinachRecipe.Title
		microwave              = accountv1.Equipment_EQUIPMENT_MICROWAVE
		stovetop               = accountv1.Equipment_EQUIPMENT_STOVETOP
		oven                   = accountv1.Equipment_EQUIPMENT_OVEN
	)
	cases := []struct {
		name string
		req  *cookv1.ListRecipesRequest
		now  []string
		away []string
	}{
		{name: "no filter", req: &cookv1.ListRecipesRequest{}, now: []string{omelette, lemon, chicken, spinach}, away: []string{toast}},
		{name: "ten minutes", req: &cookv1.ListRecipesRequest{MaxMinutes: 10}, now: []string{lemon, spinach}, away: []string{toast}},
		{name: "twenty minutes", req: &cookv1.ListRecipesRequest{MaxMinutes: 20}, now: []string{omelette, lemon, spinach}, away: []string{toast}},
		{name: "forty-five minutes", req: &cookv1.ListRecipesRequest{MaxMinutes: 45}, now: []string{omelette, lemon, chicken, spinach}, away: []string{toast}},
		{name: "a microwave alone", req: &cookv1.ListRecipesRequest{Equipment: []accountv1.Equipment{microwave}}, now: []string{lemon, spinach}},
		{name: "a stovetop and an oven", req: &cookv1.ListRecipesRequest{Equipment: []accountv1.Equipment{stovetop, oven}}, now: []string{omelette, lemon, chicken}, away: []string{toast}},
		{name: "vegan", req: &cookv1.ListRecipesRequest{Diet: accountv1.Diet_DIET_VEGAN}, now: []string{lemon, spinach}},
		{name: "vegetarian", req: &cookv1.ListRecipesRequest{Diet: accountv1.Diet_DIET_VEGETARIAN}, now: []string{omelette, lemon, spinach}, away: []string{toast}},
		{name: "pescatarian", req: &cookv1.ListRecipesRequest{Diet: accountv1.Diet_DIET_PESCATARIAN}, now: []string{omelette, lemon, spinach}, away: []string{toast}},
		{name: "no diet", req: &cookv1.ListRecipesRequest{Diet: accountv1.Diet_DIET_NONE}, now: []string{omelette, lemon, chicken, spinach}, away: []string{toast}},
		{
			name: "every filter at once",
			req:  &cookv1.ListRecipesRequest{MaxMinutes: 10, Equipment: []accountv1.Equipment{microwave}, Diet: accountv1.Diet_DIET_VEGAN},
			now:  []string{lemon, spinach},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.req.ScanId = verdict.GetScanId()

			got := listRecipes(t, maya, c.req)

			now := titlesOf(got.GetMakeItNow())
			if !slices.Equal(now, c.now) {
				t.Errorf("make it now = %v, want %v", now, c.now)
			}
			away := titlesOf(got.GetOneItemAway())
			if !slices.Equal(away, c.away) {
				t.Errorf("one item away = %v, want %v", away, c.away)
			}
		})
	}
}

// TestAnAllergyHidesARecipe: an allergy is a hard filter, so a recipe holding one never reaches
// the player, whether it declares it, gives it away in its food, or was made before the allergy.
func TestAnAllergyHidesARecipe(t *testing.T) {
	t.Parallel()
	cheddarSnack := vision.Recipe{
		Title: "Cheddar Cubes", Minutes: 1, Difficulty: "easy", Uses: []string{"Cheddar"},
		Steps: []string{"Cube the cheddar."},
	}
	t.Run("an allergy the player had when the set was made", func(t *testing.T) {
		t.Parallel()
		f := cookFixture{
			eyes: newFakeVision(kitchenFridge),
			chef: &fakeChef{recipes: append(slices.Clone(kitchenRecipes), cheddarSnack)},
		}
		srv := cookStack(t, f)
		maya := cookPlayerFor(t, srv, "test_maya")
		setKitchen(t, srv, "test_maya", nil, repository.DietNone, []string{repository.AllergyMilk})
		verdict := scanFridge(t, maya)

		got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

		// The omelette and the toast declare milk; the cheddar cubes only give it away.
		want := []string{lemonWaterRecipe.Title, chickenRecipe.Title, spinachRecipe.Title}
		now := titlesOf(got.GetMakeItNow())
		if !slices.Equal(now, want) || len(got.GetOneItemAway()) != 0 {
			t.Errorf("shelves = %v and %v, want %v and nothing one item away",
				now, titlesOf(got.GetOneItemAway()), want)
		}
		stored := count(t, srv, `select count(*) from app.recipe where 'milk' = any (allergens)`)
		if stored != 0 {
			t.Errorf("%d recipes with milk stored, want none", stored)
		}
	})
	t.Run("an allergy added after the set was made", func(t *testing.T) {
		t.Parallel()
		f := cookFixture{
			eyes: newFakeVision(kitchenFridge),
			chef: &fakeChef{recipes: append(slices.Clone(kitchenRecipes), cheddarSnack)},
		}
		srv := cookStack(t, f)
		maya := cookPlayerFor(t, srv, "test_maya")
		verdict := scanFridge(t, maya)
		before := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})
		omeletteID := firstOn(t, before.GetMakeItNow()).GetRecipeId()

		setKitchen(t, srv, "test_maya", nil, repository.DietNone, []string{repository.AllergyMilk})

		// The cheddar cubes declare nothing, so only the milk their food gives away hides them.
		got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})
		want := []string{lemonWaterRecipe.Title, chickenRecipe.Title, spinachRecipe.Title}
		now := titlesOf(got.GetMakeItNow())
		if !slices.Equal(now, want) || len(got.GetOneItemAway()) != 0 {
			t.Errorf("shelves = %v and %v, want %v and nothing one item away",
				now, titlesOf(got.GetOneItemAway()), want)
		}
		_, err := maya.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: omeletteID}))
		code := connect.CodeOf(err)
		if code != connect.CodeNotFound {
			t.Errorf("the omelette by id: code %v (%v), want not_found", code, err)
		}
	})
}

// TestTheModelHearsTheKitchenAndTheListAlone: the recipes must suit the player's equipment, diet
// and allergies, and go-vendors.md lets nothing else about them reach the vendor.
func TestTheModelHearsTheKitchenAndTheListAlone(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	setKitchen(t, srv, "test_maya",
		[]string{repository.EquipmentMicrowave, repository.EquipmentStovetop}, repository.DietVegetarian,
		[]string{repository.AllergyPeanuts})
	verdict := scanFridge(t, maya)

	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	calls := chef.calls()
	if len(calls) != 1 {
		t.Fatalf("the model was asked %d times, want once", len(calls))
	}
	wantKitchen := vision.Kitchen{
		Equipment: []string{"microwave", "stovetop"}, Diet: "vegetarian", Allergies: []string{"peanuts"},
	}
	got := calls[0]
	if !slices.Equal(got.kitchen.Equipment, wantKitchen.Equipment) || got.kitchen.Diet != wantKitchen.Diet ||
		!slices.Equal(got.kitchen.Allergies, wantKitchen.Allergies) {
		t.Errorf("kitchen = %+v, want %+v", got.kitchen, wantKitchen)
	}
	if !slices.Equal(got.items, kitchenFridge.Items) {
		t.Errorf("items = %v, want the stored list %v", got.items, kitchenFridge.Items)
	}
}

// TestEditingTheListRemakesTheRecipes: the recipes come from the list, so a fixed list must get
// recipes of its own, while a filter change, or an edit that changes nothing, reuses the set.
func TestEditingTheListRemakesTheRecipes(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)
	scanID := verdict.GetScanId()
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: scanID})
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: scanID, MaxMinutes: 20})
	editList(t, maya, scanID, verdict.GetIngredients())
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: scanID})
	asked := len(chef.calls())
	if asked != 1 {
		t.Fatalf("the model was asked %d times before the list changed, want once", asked)
	}

	// The chicken is gone, so its bake no longer fits.
	withoutChicken := slices.DeleteFunc(slices.Clone(verdict.GetIngredients()), func(i *scanv1.Ingredient) bool {
		return i.GetName() == "Chicken"
	})
	editList(t, maya, scanID, withoutChicken)
	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: scanID})

	calls := chef.calls()
	if len(calls) != 2 {
		t.Fatalf("the model was asked %d times, want twice: once per list", len(calls))
	}
	var sent []string
	for _, item := range calls[1].items {
		sent = append(sent, item.Name)
	}
	fixed := []string{"Eggs", "Spinach", "Cheddar", "Lemon"}
	if !slices.Equal(sent, fixed) {
		t.Errorf("the second call sent %v, want the fixed list %v", sent, fixed)
	}
	want := []string{omeletteRecipe.Title, lemonWaterRecipe.Title, spinachRecipe.Title}
	now := titlesOf(got.GetMakeItNow())
	if !slices.Equal(now, want) {
		t.Errorf("make it now = %v, want %v", now, want)
	}
}

// TestAModelOutageWritingRecipesAnswersUnavailable: a failed call must reach the player as "try
// again", with nothing stored, so the next try asks the model afresh.
func TestAModelOutageWritingRecipesAnswersUnavailable(t *testing.T) {
	t.Parallel()
	f := cookFixture{
		eyes: newFakeVision(kitchenFridge),
		chef: &fakeChef{err: fmt.Errorf("vision: recipes: %w", vision.ErrUnavailable)},
	}
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	_, err := maya.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()}))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnavailable || cerr.Message() != vision.ErrUnavailable.Error() {
		t.Errorf("got %v %q, want unavailable %q", cerr.Code(), cerr.Message(), vision.ErrUnavailable.Error())
	}
	stored := count(t, srv, `select count(*) from app.recipe`)
	if stored != 0 {
		t.Errorf("%d recipes stored after an outage, want none", stored)
	}
}

// TestAListWithNothingOnItCooksNothing: an empty fridge has nothing to cook, so its shelves come
// back empty without spending a call on the model.
func TestAListWithNothingOnItCooksNothing(t *testing.T) {
	t.Parallel()
	chef := &fakeChef{recipes: kitchenRecipes}
	f := cookFixture{eyes: newFakeVision(vision.Fridge{Roast: "Nothing. Not even ice."}), chef: chef}
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	if len(got.GetMakeItNow()) != 0 || len(got.GetOneItemAway()) != 0 {
		t.Errorf("shelves = %v and %v, want both empty", titlesOf(got.GetMakeItNow()), titlesOf(got.GetOneItemAway()))
	}
	asked := len(chef.calls())
	if asked != 0 {
		t.Errorf("the model was asked %d times about an empty list, want never", asked)
	}
}

// TestJunkFromTheModelWritingRecipesIsDropped: what the model wrote is as untrusted as a request,
// so a recipe that fails the shape is dropped, the rest are kept, and the log gets the count,
// never the content.
func TestJunkFromTheModelWritingRecipesIsDropped(t *testing.T) {
	t.Parallel()
	junk := func(edit func(r *vision.Recipe)) vision.Recipe {
		r := lemonWaterRecipe
		r.Title = "Junk"
		r.Uses, r.Steps = slices.Clone(r.Uses), slices.Clone(r.Steps)
		edit(&r)
		return r
	}
	written := []vision.Recipe{
		junk(func(r *vision.Recipe) { r.Title = strings.Repeat("Soup ", 17) }),
		junk(func(r *vision.Recipe) { r.Title = "" }),
		junk(func(r *vision.Recipe) { r.Minutes = 0 }),
		junk(func(r *vision.Recipe) { r.Minutes = 241 }),
		junk(func(r *vision.Recipe) { r.Difficulty = "expert" }),
		junk(func(r *vision.Recipe) { r.Equipment = []string{"grill"} }),
		junk(func(r *vision.Recipe) { r.Uses = nil }),
		junk(func(r *vision.Recipe) { r.Uses = []string{strings.Repeat("a", 41)} }),
		junk(func(r *vision.Recipe) { r.Steps = nil }),
		junk(func(r *vision.Recipe) { r.Steps = []string{"Stir.\x00"} }),
		junk(func(r *vision.Recipe) { r.Allergens = []string{"gluten"} }),
		// The model's capitals and spacing are tidied, not refused.
		func() vision.Recipe {
			r := spinachRecipe
			r.Difficulty, r.Equipment = " Easy ", []string{"MICROWAVE "}
			return r
		}(),
		lemonWaterRecipe,
	}
	var logs lockedBuffer
	f := cookFixture{eyes: newFakeVision(kitchenFridge), chef: &fakeChef{recipes: written}, log: jsonLog(&logs)}
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	want := []string{spinachRecipe.Title, lemonWaterRecipe.Title}
	now := titlesOf(got.GetMakeItNow())
	if !slices.Equal(now, want) {
		t.Errorf("make it now = %v, want %v", now, want)
	}
	tidy := got.GetMakeItNow()
	if len(tidy) > 0 &&
		(tidy[0].GetDifficulty() != cookv1.Difficulty_DIFFICULTY_EASY ||
			!slices.Equal(tidy[0].GetEquipment(), []accountv1.Equipment{accountv1.Equipment_EQUIPMENT_MICROWAVE})) {
		t.Errorf("tidied recipe = %v, want easy and a microwave", tidy[0])
	}
	line := logs.String()
	if !strings.Contains(line, `"dropped":11`) {
		t.Errorf("log = %s, want the 11 dropped counted", line)
	}
	if strings.Contains(line, "Soup Soup") || strings.Contains(line, "gluten") {
		t.Errorf("log = %s, want the count and none of the content", line)
	}
}

// TestASetKeepsAHandful: the model is asked for a handful, and a reply that runs long must not
// fill a shelf past what the screen was built for.
func TestASetKeepsAHandful(t *testing.T) {
	t.Parallel()
	long := make([]vision.Recipe, 15)
	for i := range long {
		long[i] = lemonWaterRecipe
		long[i].Title = fmt.Sprintf("Lemon Water No. %d", i+1)
	}
	f := cookFixture{eyes: newFakeVision(kitchenFridge), chef: &fakeChef{recipes: long}}
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})

	now := titlesOf(got.GetMakeItNow())
	if len(now) != 12 || now[0] != "Lemon Water No. 1" || now[11] != "Lemon Water No. 12" {
		t.Errorf("make it now = %v, want the first 12 in the model's order", now)
	}
}

// TestWritingRecipesIsLimitedPerPlayer: every set the model writes costs money, so a player who
// has spent their writes is refused until later, while rereading a set spends nothing and another
// player keeps their own.
func TestWritingRecipesIsLimitedPerPlayer(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	f.limit = handlers.NewLimiter(2, time.Hour)
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	sam := cookPlayerFor(t, srv, "test_sam")
	first, second, third := scanFridge(t, maya), scanFridge(t, maya), scanFridge(t, maya)
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: first.GetScanId()})
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: second.GetScanId()})

	_, err := maya.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: third.GetScanId()}))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeResourceExhausted || cerr.Message() != handlers.ErrTooManyRequests.Error() {
		t.Errorf("third set: %v %q, want resource_exhausted %q", cerr.Code(), cerr.Message(), handlers.ErrTooManyRequests.Error())
	}
	listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: first.GetScanId(), MaxMinutes: 10})
	listRecipes(t, sam, &cookv1.ListRecipesRequest{ScanId: scanFridge(t, sam).GetScanId()})
	asked := len(chef.calls())
	if asked != 3 {
		t.Errorf("the model was asked %d times, want 3: maya's two and sam's", asked)
	}
}

// TestTwentyRacingRequestsStoreOneSet: a screen that opens twice at once must not double the
// shelves, and every request must answer with the one set that landed.
func TestTwentyRacingRequestsStoreOneSet(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	verdict := scanFridge(t, maya)

	var wg sync.WaitGroup
	answers := make([]*cookv1.ListRecipesResponse, 20)
	errs := make([]error, 20)
	for i := range 20 {
		wg.Go(func() {
			got, err := maya.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()}))
			errs[i] = err
			if err == nil {
				answers[i] = got.Msg
			}
		})
	}
	wg.Wait()

	for i := range 20 {
		if errs[i] != nil {
			t.Errorf("request %d: %v", i, errs[i])
			continue
		}
		if !proto.Equal(answers[i], answers[0]) {
			t.Errorf("request %d answered %v, want the same set as request 0 %v", i, answers[i], answers[0])
		}
	}
	stored := count(t, srv, `select count(*) from app.recipe`)
	if stored != 5 {
		t.Errorf("%d recipes stored, want one set of 5", stored)
	}
}

// TestAnInvalidListRequestIsRefusedFieldByField: every problem arrives at once under its field,
// and a refused request never reaches the model.
func TestAnInvalidListRequestIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	scanID := scanFridge(t, maya).GetScanId()
	const (
		equipmentRule = "equipment must be microwave, stovetop, oven or air fryer"
		dietRule      = "diet must be none, vegetarian, vegan or pescatarian"
		minutesRule   = "max minutes must be 0 or 10 or 20 or 45"
	)
	cases := []struct {
		name string
		req  *cookv1.ListRecipesRequest
		want map[string]string
	}{
		{name: "no scan", req: &cookv1.ListRecipesRequest{}, want: map[string]string{"scan_id": "scan id is required"}},
		{name: "fifteen minutes", req: &cookv1.ListRecipesRequest{ScanId: scanID, MaxMinutes: 15}, want: map[string]string{"max_minutes": minutesRule}},
		{
			name: "unspecified equipment",
			req:  &cookv1.ListRecipesRequest{ScanId: scanID, Equipment: []accountv1.Equipment{accountv1.Equipment_EQUIPMENT_UNSPECIFIED}},
			want: map[string]string{"equipment[0]": equipmentRule},
		},
		{
			name: "equipment newer than this server",
			req:  &cookv1.ListRecipesRequest{ScanId: scanID, Equipment: []accountv1.Equipment{accountv1.Equipment_EQUIPMENT_OVEN, 99}},
			want: map[string]string{"equipment[1]": equipmentRule},
		},
		{name: "a diet newer than this server", req: &cookv1.ListRecipesRequest{ScanId: scanID, Diet: 99}, want: map[string]string{"diet": dietRule}},
		{
			name: "everything at once",
			req:  &cookv1.ListRecipesRequest{MaxMinutes: -1, Equipment: []accountv1.Equipment{99}, Diet: 99},
			want: map[string]string{"scan_id": "scan id is required", "max_minutes": minutesRule, "equipment[0]": equipmentRule, "diet": dietRule},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := maya.cook.ListRecipes(t.Context(), connect.NewRequest(c.req))

			code := connect.CodeOf(err)
			if code != connect.CodeInvalidArgument {
				t.Fatalf("code %v (%v), want invalid_argument", code, err)
			}
			got := fieldViolations(t, err)
			if !maps.Equal(got, c.want) {
				t.Errorf("violations = %v, want %v", got, c.want)
			}
		})
	}
	t.Cleanup(func() {
		asked := len(chef.calls())
		if asked != 0 {
			t.Errorf("the model was asked %d times, want never for a refused request", asked)
		}
	})
}

// TestCookingNeedsAnAccount: recipes belong to a player, so a phone with no session gets
// Unauthenticated, which the app reads as "back to onboarding".
func TestCookingNeedsAnAccount(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	srv := cookStack(t, f)
	stranger := cookv1connect.NewCookServiceClient(http.DefaultClient, srv.url)

	_, listErr := stranger.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ScanId: 1}))
	_, getErr := stranger.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: 1}))

	for name, err := range map[string]error{"ListRecipes": listErr, "GetRecipe": getErr} {
		code := connect.CodeOf(err)
		if code != connect.CodeUnauthenticated {
			t.Errorf("%s: code %v (%v), want unauthenticated", name, code, err)
		}
	}
}

// cookFixture is what a cook stack runs with: the eyes that read fridges into scans, the chef
// that writes recipes, the limit on writing them (nil limits nothing) and the cook service's log
// (nil discards).
type cookFixture struct {
	eyes  scan.Vision
	chef  cook.Vision
	limit *handlers.Limiter
	log   *slog.Logger
}

// kitchenFixture reads every fridge as kitchenFridge and writes kitchenRecipes for it. The chef
// comes back too, so a test can count what it was asked.
func kitchenFixture() (cookFixture, *fakeChef) {
	chef := &fakeChef{recipes: kitchenRecipes}
	return cookFixture{eyes: newFakeVision(kitchenFridge), chef: chef}, chef
}

// cookStack mounts CookService beside AccountService, which makes the players, and ScanService,
// which makes their scans, the way main.go mounts them.
func cookStack(t *testing.T, f cookFixture) stack {
	t.Helper()
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		mux.Handle(accountv1connect.NewAccountServiceHandler(
			account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		mux.Handle(scanv1connect.NewScanServiceHandler(
			scan.NewScan(scan.Config{Repo: repo, Auth: authn, Vision: f.eyes})))
		svc := cook.NewCook(cook.Config{Repo: repo, Auth: authn, Vision: f.chef, Limit: f.limit, Log: f.log})
		mux.Handle(cookv1connect.NewCookServiceHandler(svc, connect.WithReadMaxBytes(4<<20)))
	})
}

// cookPlayer is a player with an account, holding the scan and cook services as their phone
// calls them.
type cookPlayer struct {
	scan scanv1connect.ScanServiceClient
	cook cookv1connect.CookServiceClient
}

func cookPlayerFor(t *testing.T, srv stack, handle string) cookPlayer {
	t.Helper()
	p := player(t, srv.url, handle)
	client := &http.Client{Jar: p.jar}
	return cookPlayer{
		scan: scanv1connect.NewScanServiceClient(client, srv.url),
		cook: cookv1connect.NewCookServiceClient(client, srv.url),
	}
}

// scanFridge makes a scan from one fridge photo and returns its verdict.
func scanFridge(t *testing.T, p cookPlayer) *scanv1.Verdict {
	t.Helper()
	return createScan(t, p.scan)
}

func listRecipes(t *testing.T, p cookPlayer, req *cookv1.ListRecipesRequest) *cookv1.ListRecipesResponse {
	t.Helper()
	got, err := p.cook.ListRecipes(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("list recipes: %v", err)
	}
	return got.Msg
}

// editList replaces a scan's list through the real rpc.
func editList(t *testing.T, p cookPlayer, scanID int64, items []*scanv1.Ingredient) {
	t.Helper()
	_, err := p.scan.EditIngredients(t.Context(), connect.NewRequest(&scanv1.EditIngredientsRequest{
		ScanId: scanID, Ingredients: items,
	}))
	if err != nil {
		t.Fatalf("edit ingredients: %v", err)
	}
}

// setKitchen writes a player's kitchen straight onto their row. UpdateKitchen is #14's; until it
// lands, this is how a test gives a player equipment, a diet or allergies.
func setKitchen(t *testing.T, srv stack, handle string, equipment []string, diet string, allergies []string) {
	t.Helper()
	if equipment == nil {
		equipment = []string{}
	}
	if allergies == nil {
		allergies = []string{}
	}
	tag, err := srv.pool.Exec(t.Context(), `
		update app.account
		   set equipment = $2, diet = $3, allergies = $4
		 where handle = $1`, handle, equipment, diet, allergies)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("set kitchen %s: %v (%d rows)", handle, err, tag.RowsAffected())
	}
}

// verdictPreview is what plating pays on this verdict's score, worked out by hand from the
// points rules: 100 for the cook, 50 more under 60, or 100 more under 35.
func verdictPreview(v *scanv1.Verdict) int32 {
	switch score := v.GetScore(); {
	case score < 35:
		return 200
	case score < 60:
		return 150
	default:
		return 100
	}
}

// firstOn is the first recipe on a shelf the test needs to hold one.
func firstOn(t *testing.T, shelf []*cookv1.Recipe) *cookv1.Recipe {
	t.Helper()
	if len(shelf) == 0 {
		t.Fatal("the shelf is empty, want at least one recipe on it")
	}
	return shelf[0]
}

func titlesOf(recipes []*cookv1.Recipe) []string {
	var out []string
	for _, r := range recipes {
		out = append(out, r.GetTitle())
	}
	return out
}

// fakeChef writes the recipes a test gave it for any list, at once, or fails with err, and
// records what it was sent. It is safe because the model is the true edge here: the checks, the
// store and the filters behind it are all real.
type fakeChef struct {
	recipes []vision.Recipe
	err     error

	mu    sync.Mutex
	asked []chefCall
}

// chefCall is one call the fake answered.
type chefCall struct {
	items   []vision.Item
	kitchen vision.Kitchen
}

func (f *fakeChef) Recipes(_ context.Context, items []vision.Item, kitchen vision.Kitchen) ([]vision.Recipe, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, chefCall{items: slices.Clone(items), kitchen: kitchen})
	if f.err != nil {
		return nil, f.err
	}
	out := make([]vision.Recipe, 0, len(f.recipes))
	for _, r := range f.recipes {
		r.Equipment, r.Uses, r.Missing = slices.Clone(r.Equipment), slices.Clone(r.Uses), slices.Clone(r.Missing)
		r.Allergens, r.Steps = slices.Clone(r.Allergens), slices.Clone(r.Steps)
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeChef) calls() []chefCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.asked)
}
