// Author: Caden Lund
package repository_test

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// A set as the handler stores it: checked, with its allergens and diets judged.
var (
	omelette = repository.NewRecipe{
		Title: "Spinach Cheddar Omelette", Minutes: 15, Difficulty: repository.DifficultyEasy,
		Equipment: []string{repository.EquipmentStovetop},
		Uses:      []string{"Eggs", "Spinach", "Cheddar", "Butter"},
		Allergens: []string{repository.AllergyEggs, repository.AllergyMilk},
		Diets:     []string{repository.DietPescatarian, repository.DietVegetarian},
		Steps:     []string{"Whisk the eggs.", "Wilt the spinach.", "Fold in the cheddar."},
	}
	// Exactly ten minutes: the shortest filter's edge.
	eggToast = repository.NewRecipe{
		Title: "Egg on Toast", Minutes: 10, Difficulty: repository.DifficultyEasy,
		Equipment: []string{repository.EquipmentStovetop},
		Uses:      []string{"Eggs", "Butter"}, Missing: []string{"Bread"},
		Allergens: []string{repository.AllergyEggs, repository.AllergyMilk, repository.AllergyWheat},
		Diets:     []string{repository.DietPescatarian, repository.DietVegetarian},
		Steps:     []string{"Toast the bread.", "Fry the egg on top."},
	}
	spinachSoup = repository.NewRecipe{
		Title: "Spinach Soup", Minutes: 40, Difficulty: repository.DifficultyMedium,
		Equipment: []string{repository.EquipmentStovetop, repository.EquipmentOven},
		Uses:      []string{"Spinach", "Oil", "Salt"},
		Diets:     []string{repository.DietPescatarian, repository.DietVegan, repository.DietVegetarian},
		Steps:     []string{"Simmer.", "Blend.", "Bake the croutons."},
	}
	set = []repository.NewRecipe{omelette, eggToast, spinachSoup}
)

// TestCreateRecipesStoresTheSetInOrder: the shelves read back exactly what was checked, in the
// model's order, each with its scan's score for the points preview.
func TestCreateRecipesStoresTheSetInOrder(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)

	stored, err := repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1, set)
	if err != nil || !stored {
		t.Fatalf("create recipes = %v, %v; want stored", stored, err)
	}
	got := listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{})

	if len(got) != len(set) {
		t.Fatalf("read %d recipes, want %d", len(got), len(set))
	}
	for i, want := range set {
		r := got[i]
		if r.RecipeID == 0 || r.Title != want.Title || r.Minutes != want.Minutes ||
			r.Difficulty != want.Difficulty || r.ScanScore != 79 {
			t.Errorf("recipe %d = id %d, %q, %d min, %s, scan score %d; want an id, %q, %d min, %s, 79",
				i, r.RecipeID, r.Title, r.Minutes, r.Difficulty, r.ScanScore,
				want.Title, want.Minutes, want.Difficulty)
		}
		for name, pair := range map[string][2][]string{
			"equipment": {r.Equipment, want.Equipment},
			"uses":      {r.Uses, want.Uses},
			"missing":   {r.Missing, want.Missing},
			"steps":     {r.Steps, want.Steps},
		} {
			if !slices.Equal(pair[0], pair[1]) {
				t.Errorf("recipe %d %s = %v, want %v", i, name, pair[0], pair[1])
			}
		}
	}
	// What the wire never carries is stored too: the filters read it.
	var allergens, diets []string
	err = pool.QueryRow(t.Context(), `
		select r.allergens, r.diets
		  from app.recipe r
		 where r.title = $1`, eggToast.Title).Scan(&allergens, &diets)
	if err != nil {
		t.Fatalf("read the egg toast's judgment: %v", err)
	}
	if !slices.Equal(allergens, eggToast.Allergens) || !slices.Equal(diets, eggToast.Diets) {
		t.Errorf("stored allergens %v, diets %v; want %v, %v", allergens, diets,
			eggToast.Allergens, eggToast.Diets)
	}
}

// TestASecondSetForTheSameVersionIsRefused: a version gets its set once, so asking again (or a
// second request that raced the first) must leave the first set as it was.
func TestASecondSetForTheSameVersionIsRefused(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)

	stored, err := repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1,
		[]repository.NewRecipe{spinachSoup, omelette, eggToast, omelette})

	if err != nil || stored {
		t.Errorf("second set = %v, %v; want not stored and no error", stored, err)
	}
	got := titles(listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{}))
	want := []string{omelette.Title, eggToast.Title, spinachSoup.Title}
	if !slices.Equal(got, want) {
		t.Errorf("titles = %v, want the first set %v", got, want)
	}
}

// TestTwentyRacingSetsStoreOne: twenty ListRecipes at the same moment each ask the model, and
// exactly one set may land, whole, never two sets mixed.
func TestTwentyRacingSetsStoreOne(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	// Sets of different sizes, so a mix of two would show.
	sets := [][]repository.NewRecipe{set, {spinachSoup}, {eggToast, spinachSoup}}

	var wg sync.WaitGroup
	results := make([]bool, 20)
	errs := make([]error, 20)
	for i := range 20 {
		wg.Go(func() {
			results[i], errs[i] = repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1,
				sets[i%len(sets)])
		})
	}
	wg.Wait()

	won := -1
	for i := range 20 {
		if errs[i] != nil {
			t.Errorf("racer %d: %v", i, errs[i])
		}
		if results[i] {
			if won >= 0 {
				t.Errorf("racers %d and %d both stored a set", won, i)
			}
			won = i
		}
	}
	if won < 0 {
		t.Fatal("no racer stored a set")
	}
	got := titles(listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{}))
	want := titles(asStored(sets[won%len(sets)]))
	if !slices.Equal(got, want) {
		t.Errorf("titles = %v, want the winner's set %v", got, want)
	}
}

// TestEachVersionOfTheListHasItsOwnSet: an edit makes a new version, whose set must not mix with
// the last one, while the old set stays for the plates that name it.
func TestEachVersionOfTheListHasItsOwnSet(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)
	changed, err := repo.ReplaceItems(t.Context(), maya.AccountID, scan.ScanID, fridge[:1])
	if err != nil || !changed {
		t.Fatalf("replace items = %v, %v", changed, err)
	}
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 2, []repository.NewRecipe{eggToast})

	for version, want := range map[int][]string{
		1: {omelette.Title, eggToast.Title, spinachSoup.Title},
		2: {eggToast.Title},
		3: nil,
	} {
		got := titles(listRecipes(t, repo, maya.AccountID, scan.ScanID, version, repository.RecipeFilter{}))
		if !slices.Equal(got, want) {
			t.Errorf("version %d: titles = %v, want %v", version, got, want)
		}
		has, err := repo.HasRecipes(t.Context(), maya.AccountID, scan.ScanID, version)
		if err != nil || has != (want != nil) {
			t.Errorf("version %d: has recipes = %v, %v; want %v", version, has, err, want != nil)
		}
	}
}

// TestAnotherPlayersScanTakesNoRecipesAndShowsNone: every query reaches the owner through the
// scan, so another player can neither store a set on it nor read one from it.
func TestAnotherPlayersScanTakesNoRecipesAndShowsNone(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	mine := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, mine.ScanID, 1, set)
	theirs := listRecipes(t, repo, maya.AccountID, mine.ScanID, 1, repository.RecipeFilter{})
	freshScan := createScan(t, repo, maya.AccountID, fridge)

	stored, err := repo.CreateRecipes(t.Context(), sam.AccountID, freshScan.ScanID, 1, set)
	if err != nil || stored {
		t.Errorf("sam's set on maya's scan = %v, %v; want not stored", stored, err)
	}
	has, err := repo.HasRecipes(t.Context(), sam.AccountID, mine.ScanID, 1)
	if err != nil || has {
		t.Errorf("sam's has recipes = %v, %v; want false", has, err)
	}
	seen := listRecipes(t, repo, sam.AccountID, mine.ScanID, 1, repository.RecipeFilter{})
	if len(seen) != 0 {
		t.Errorf("sam read %d of maya's recipes, want none", len(seen))
	}
	_, err = repo.RecipeByID(t.Context(), sam.AccountID, theirs[0].RecipeID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("sam's recipe by id err = %v, want ErrNotFound", err)
	}

	n := rows(t, pool, `select count(*) from app.recipe`)
	if n != len(set) {
		t.Errorf("%d recipes stored, want maya's %d alone", n, len(set))
	}
	again := listRecipes(t, repo, maya.AccountID, mine.ScanID, 1, repository.RecipeFilter{})
	if !slices.Equal(titles(again), titles(theirs)) {
		t.Errorf("maya's recipes = %v after sam's tries, want %v", titles(again), titles(theirs))
	}
}

// TestRecipeByIDReadsOneRecipe: a recipe opened from its card must be the one on the card.
func TestRecipeByIDReadsOneRecipe(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)
	listed := listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{})

	got, err := repo.RecipeByID(t.Context(), maya.AccountID, listed[1].RecipeID)

	if err != nil {
		t.Fatalf("recipe by id: %v", err)
	}
	if got.RecipeID != listed[1].RecipeID || got.Title != eggToast.Title || got.ScanScore != 79 ||
		!slices.Equal(got.Missing, eggToast.Missing) || !slices.Equal(got.Steps, eggToast.Steps) {
		t.Errorf("recipe = %+v, want the egg toast with its scan score 79", got)
	}
	_, err = repo.RecipeByID(t.Context(), maya.AccountID, listed[2].RecipeID+1000)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("a recipe that is not there: err = %v, want ErrNotFound", err)
	}
}

// TestEachFilterNarrowsTheSetInSQL: the filters run in the query, so a narrowed shelf is the
// server's answer and never a page the app trimmed.
func TestEachFilterNarrowsTheSetInSQL(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)
	all := []string{omelette.Title, eggToast.Title, spinachSoup.Title}
	cases := []struct {
		name   string
		filter repository.RecipeFilter
		want   []string
	}{
		{name: "no filter", filter: repository.RecipeFilter{}, want: all},
		{name: "ten minutes", filter: repository.RecipeFilter{MaxMinutes: 10}, want: []string{eggToast.Title}},
		{name: "twenty minutes", filter: repository.RecipeFilter{MaxMinutes: 20}, want: []string{omelette.Title, eggToast.Title}},
		{name: "forty-five minutes", filter: repository.RecipeFilter{MaxMinutes: 45}, want: all},
		{name: "a stovetop", filter: repository.RecipeFilter{Equipment: []string{repository.EquipmentStovetop}}, want: []string{omelette.Title, eggToast.Title}},
		{name: "a stovetop and an oven", filter: repository.RecipeFilter{Equipment: []string{repository.EquipmentOven, repository.EquipmentStovetop}}, want: all},
		{name: "a microwave alone", filter: repository.RecipeFilter{Equipment: []string{repository.EquipmentMicrowave}}, want: nil},
		{name: "vegan", filter: repository.RecipeFilter{Diet: repository.DietVegan}, want: []string{spinachSoup.Title}},
		{name: "vegetarian", filter: repository.RecipeFilter{Diet: repository.DietVegetarian}, want: all},
		{name: "a diet of none filters nothing", filter: repository.RecipeFilter{Diet: repository.DietNone}, want: all},
		{
			name:   "every filter at once",
			filter: repository.RecipeFilter{MaxMinutes: 20, Equipment: []string{repository.EquipmentStovetop}, Diet: repository.DietVegan},
			want:   nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := titles(listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, c.filter))

			if !slices.Equal(got, c.want) {
				t.Errorf("titles = %v, want %v", got, c.want)
			}
		})
	}
}

// TestAnAllergyHidesARecipeEvenOneMadeBeforeIt: the set was checked against the allergies of the
// moment it was made, so one added since must still keep its recipes off every shelf and page.
func TestAnAllergyHidesARecipeEvenOneMadeBeforeIt(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)
	before := listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{})

	// UpdateKitchen is #14's; until it lands, the allergy goes straight onto the row.
	_, err := pool.Exec(t.Context(), `
		update app.account
		   set allergies = array['wheat']
		 where account_id = $1`, maya.AccountID)
	if err != nil {
		t.Fatalf("add a wheat allergy: %v", err)
	}

	got := titles(listRecipes(t, repo, maya.AccountID, scan.ScanID, 1, repository.RecipeFilter{}))
	want := []string{omelette.Title, spinachSoup.Title}
	if !slices.Equal(got, want) {
		t.Errorf("titles = %v, want %v without the egg toast's wheat", got, want)
	}
	_, err = repo.RecipeByID(t.Context(), maya.AccountID, before[1].RecipeID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("the egg toast by id: err = %v, want ErrNotFound", err)
	}
	_, err = repo.RecipeByID(t.Context(), maya.AccountID, before[0].RecipeID)
	if err != nil {
		t.Errorf("the omelette by id: %v, want it still there", err)
	}
}

// TestAnEmptySetStoresNothing: a list the model found nothing for has no set to store, and the
// call must not pretend it stored one.
func TestAnEmptySetStoresNothing(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)

	stored, err := repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1, nil)

	if err != nil || stored {
		t.Errorf("empty set = %v, %v; want not stored", stored, err)
	}
	n := rows(t, pool, `select count(*) from app.recipe`)
	if n != 0 {
		t.Errorf("%d recipes stored, want none", n)
	}
}

// TestTheSchemaHoldsTheRecipeShape: the handler validates the model's recipes first, and these
// checks are what still hold if a write ever skips it.
func TestTheSchemaHoldsTheRecipeShape(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	with := func(edit func(r *repository.NewRecipe)) repository.NewRecipe {
		r := spinachSoup
		r.Equipment, r.Uses, r.Missing = slices.Clone(r.Equipment), slices.Clone(r.Uses), slices.Clone(r.Missing)
		r.Allergens, r.Diets, r.Steps = slices.Clone(r.Allergens), slices.Clone(r.Diets), slices.Clone(r.Steps)
		edit(&r)
		return r
	}
	twenty := make([]string, 20)
	for i := range twenty {
		twenty[i] = "Step."
	}
	cases := []struct {
		name   string
		recipe repository.NewRecipe
		ok     bool
	}{
		{name: "as checked", recipe: with(func(*repository.NewRecipe) {}), ok: true},
		{name: "an eighty-character title", recipe: with(func(r *repository.NewRecipe) { r.Title = strings.Repeat("a", 80) }), ok: true},
		{name: "an empty title", recipe: with(func(r *repository.NewRecipe) { r.Title = "" }), ok: false},
		{name: "an eighty-one-character title", recipe: with(func(r *repository.NewRecipe) { r.Title = strings.Repeat("a", 81) }), ok: false},
		{name: "a minute", recipe: with(func(r *repository.NewRecipe) { r.Minutes = 1 }), ok: true},
		{name: "four hours", recipe: with(func(r *repository.NewRecipe) { r.Minutes = 240 }), ok: true},
		{name: "no minutes", recipe: with(func(r *repository.NewRecipe) { r.Minutes = 0 }), ok: false},
		{name: "past four hours", recipe: with(func(r *repository.NewRecipe) { r.Minutes = 241 }), ok: false},
		{name: "a difficulty outside the set", recipe: with(func(r *repository.NewRecipe) { r.Difficulty = "expert" }), ok: false},
		{name: "no equipment", recipe: with(func(r *repository.NewRecipe) { r.Equipment = nil }), ok: true},
		{name: "equipment outside the set", recipe: with(func(r *repository.NewRecipe) { r.Equipment = []string{"grill"} }), ok: false},
		{name: "nothing used", recipe: with(func(r *repository.NewRecipe) { r.Uses = nil }), ok: false},
		{name: "two missing", recipe: with(func(r *repository.NewRecipe) { r.Missing = []string{"Bread", "Ham"} }), ok: false},
		{name: "an allergen outside the set", recipe: with(func(r *repository.NewRecipe) { r.Allergens = []string{"gluten"} }), ok: false},
		{name: "none among the diets", recipe: with(func(r *repository.NewRecipe) { r.Diets = []string{repository.DietNone} }), ok: false},
		{name: "twenty steps", recipe: with(func(r *repository.NewRecipe) { r.Steps = twenty }), ok: true},
		{name: "no steps", recipe: with(func(r *repository.NewRecipe) { r.Steps = nil }), ok: false},
		{name: "twenty-one steps", recipe: with(func(r *repository.NewRecipe) { r.Steps = append(slices.Clone(twenty), "One more.") }), ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// A scan each, so every case stores into a version of its own.
			scan := createScan(t, repo, maya.AccountID, fridge)

			stored, err := repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1,
				[]repository.NewRecipe{c.recipe})

			switch {
			case c.ok && (err != nil || !stored):
				t.Errorf("refused: %v, %v", stored, err)
			case !c.ok && !isCheckViolation(err) && !isNotNullViolation(err):
				t.Errorf("err = %v, want a check or not-null violation", err)
			}
		})
	}
}

// TestEveryDifficultyConstIsOneTheSchemaTakes: the consts mirror app.recipe's check by hand, so a
// misspelled one would pass review and fail on its first recipe.
func TestEveryDifficultyConstIsOneTheSchemaTakes(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	var each []repository.NewRecipe
	for _, d := range []string{repository.DifficultyEasy, repository.DifficultyMedium, repository.DifficultyHard} {
		r := spinachSoup
		r.Difficulty = d
		each = append(each, r)
	}

	stored, err := repo.CreateRecipes(t.Context(), maya.AccountID, scan.ScanID, 1, each)

	if err != nil || !stored {
		t.Errorf("one recipe per difficulty = %v, %v; want stored", stored, err)
	}
}

// TestDeletingAScanTakesItsRecipesWithIt: the recipes belong to the scan, so a deleted scan must
// leave no set behind.
func TestDeletingAScanTakesItsRecipesWithIt(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	createRecipes(t, repo, maya.AccountID, scan.ScanID, 1, set)

	_, err := pool.Exec(t.Context(), `delete from app.scan where scan_id = $1`, scan.ScanID)
	if err != nil {
		t.Fatalf("delete scan: %v", err)
	}

	n := rows(t, pool, `select count(*) from app.recipe`)
	if n != 0 {
		t.Errorf("%d recipes left, want none", n)
	}
}

func createRecipes(
	t *testing.T, repo *repository.Repository, accountID, scanID int64, version int,
	recipes []repository.NewRecipe,
) {
	t.Helper()
	stored, err := repo.CreateRecipes(t.Context(), accountID, scanID, version, recipes)
	if err != nil || !stored {
		t.Fatalf("create recipes = %v, %v; want stored", stored, err)
	}
}

func listRecipes(
	t *testing.T, repo *repository.Repository, accountID, scanID int64, version int,
	f repository.RecipeFilter,
) []repository.Recipe {
	t.Helper()
	got, err := repo.ListRecipes(t.Context(), accountID, scanID, version, f)
	if err != nil {
		t.Fatalf("list recipes: %v", err)
	}
	return got
}

func titles(recipes []repository.Recipe) []string {
	var out []string
	for _, r := range recipes {
		out = append(out, r.Title)
	}
	return out
}

// asStored is a set as ListRecipes reads it back, for comparing titles.
func asStored(set []repository.NewRecipe) []repository.Recipe {
	out := make([]repository.Recipe, 0, len(set))
	for _, r := range set {
		out = append(out, repository.Recipe{Title: r.Title})
	}
	return out
}

// isNotNullViolation reports a value the table requires and the write left out.
func isNotNullViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23502" // not_null_violation
}
