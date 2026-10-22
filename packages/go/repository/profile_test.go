// Author: Caden Lund
package repository_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// thisWeek is the Monday 00:00, campus time, that the profile tests stand in.
var thisWeek = time.Date(2026, 10, 19, 0, 0, 0, 0, campus)

// TestFryRunCountIsTheCallersOwn: a player's count is their runs alone, all time.
func TestFryRunCountIsTheCallersOwn(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	ranOn(t, repo, maya, thisWeek.AddDate(0, 0, -9), thisWeek.AddDate(0, 0, 1))
	ranOn(t, repo, ben, thisWeek.AddDate(0, 0, 1), thisWeek.AddDate(0, 0, 2), thisWeek.AddDate(0, 0, 3))

	mayas, err := repo.FryRunCount(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("maya's count: %v", err)
	}
	none, err := repo.FryRunCount(t.Context(), createAccount(t, repo, "test_eve").AccountID)
	if err != nil {
		t.Fatalf("eve's count: %v", err)
	}

	if mayas != 2 || none != 0 {
		t.Errorf("counts = %d and %d, want maya's 2 and eve's 0", mayas, none)
	}
}

// TestHallOfFriesWeeksCountsTheFinishedWeeksAPlayerTopped: a week counts once it is over, a shared
// top counts for both, a run late on Sunday night belongs to that campus week, and the week still
// running does not count yet.
func TestHallOfFriesWeeksCountsTheFinishedWeeksAPlayerTopped(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	threeAgo, twoAgo, lastWeek := thisWeek.AddDate(0, 0, -21), thisWeek.AddDate(0, 0, -14), thisWeek.AddDate(0, 0, -7)
	// Three weeks ago maya tops it, 2 to 1, her second run at 23:30 on that week's Sunday.
	ranOn(t, repo, maya, threeAgo.Add(12*time.Hour), threeAgo.AddDate(0, 0, 6).Add(23*time.Hour+30*time.Minute))
	ranOn(t, repo, ben, threeAgo.Add(12*time.Hour))
	// Two weeks ago they share the top.
	ranOn(t, repo, maya, twoAgo.Add(12*time.Hour))
	ranOn(t, repo, ben, twoAgo.Add(12*time.Hour))
	// Last week ben tops it, 2 to 1.
	ranOn(t, repo, ben, lastWeek.Add(12*time.Hour), lastWeek.AddDate(0, 0, 1).Add(12*time.Hour))
	ranOn(t, repo, maya, lastWeek.Add(12*time.Hour))
	// This week maya leads, but it is not over.
	ranOn(t, repo, maya, thisWeek.Add(12*time.Hour), thisWeek.AddDate(0, 0, 1).Add(12*time.Hour))

	got := map[string]int{}
	for _, a := range []repository.Account{maya, ben, createAccount(t, repo, "test_eve")} {
		n, err := repo.HallOfFriesWeeks(t.Context(), a.AccountID, thisWeek, campus)
		if err != nil {
			t.Fatalf("%s's weeks: %v", a.Handle, err)
		}
		got[a.Handle] = n
	}

	want := map[string]int{"test_maya": 2, "test_ben": 2, "test_eve": 0}
	for handle, n := range want {
		if got[handle] != n {
			t.Errorf("%s topped %d weeks, want %d (all: %v)", handle, got[handle], n, got)
		}
	}
}

// TestListPlatesPagesNewestFirst: a walk over pages of two returns every plate once, newest first,
// with plates from the same moment in a fixed order, and ends.
func TestListPlatesPagesNewestFirst(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	recipe := plateableRecipe(t, repo, maya.AccountID, 79)
	at := thisWeek.Add(12 * time.Hour)
	for i, when := range []time.Time{at, at.Add(time.Hour), at.Add(2 * time.Hour), at.Add(2 * time.Hour), at.Add(3 * time.Hour)} {
		createPlate(t, repo, maya.AccountID, recipe, fmt.Sprintf("plate %d", i), when)
	}

	var walked []string
	var after *repository.PlateKey
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatalf("still paging after %d pages: %v", pages, walked)
		}
		page, err := repo.ListPlates(t.Context(), maya.AccountID, after, 2)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		if len(page) == 0 {
			break
		}
		for _, p := range page {
			walked = append(walked, fmt.Sprintf("%s %s %s", p.At.In(campus).Format("15:04"), p.RecipeTitle, p.Photo[4:]))
		}
		last := page[len(page)-1]
		after = &repository.PlateKey{At: last.At, PlateID: last.PlateID}
	}

	// Plates 2 and 3 share 14:00; the one stored later comes first.
	want := []string{
		"15:00 Spinach Soup plate 4", "14:00 Spinach Soup plate 3", "14:00 Spinach Soup plate 2",
		"13:00 Spinach Soup plate 1", "12:00 Spinach Soup plate 0",
	}
	if !slices.Equal(walked, want) {
		t.Errorf("walked %v\nwant %v", walked, want)
	}
}

// TestListPlatesNeverShowsAnotherPlayersPlates: the account in the where clause is the fence, so a
// player's grid holds their plates and nobody else's.
func TestListPlatesNeverShowsAnotherPlayersPlates(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79), "maya's", thisWeek)
	createPlate(t, repo, ben.AccountID, plateableRecipe(t, repo, ben.AccountID, 79), "ben's", thisWeek)

	mine, err := repo.ListPlates(t.Context(), maya.AccountID, nil, 10)
	if err != nil {
		t.Fatalf("maya's plates: %v", err)
	}
	eves, err := repo.ListPlates(t.Context(), createAccount(t, repo, "test_eve").AccountID, nil, 10)
	if err != nil {
		t.Fatalf("eve's plates: %v", err)
	}

	if len(mine) != 1 || string(mine[0].Photo[4:]) != "maya's" {
		t.Errorf("maya's grid = %d plates, want only her own", len(mine))
	}
	if len(eves) != 0 {
		t.Errorf("eve's grid = %d plates, want none", len(eves))
	}
}

// ranOn logs one fry run for a at each time, each in a meal window of its own.
func ranOn(t *testing.T, repo *repository.Repository, a repository.Account, at ...time.Time) {
	t.Helper()
	for i, when := range at {
		logFryRun(t, repo, a.AccountID, repository.FryRun{
			SpotID:     "mock-03",
			SpotName:   "Night Owl Fries",
			MealDay:    when,
			MealWindow: []string{"morning", "midday", "evening", "late_night"}[i%4],
			At:         when,
		})
	}
}
