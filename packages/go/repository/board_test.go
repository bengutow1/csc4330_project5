// Author: Caden Lund
package repository_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// monday is the start of the week these tests count from, at noon UTC so each day's run is on that
// day wherever the clock is read.
var monday = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

// TestTheHallOfFriesRanksTiesTogether: players with as many runs share a rank, the next rank skips
// past them, and rows that share a rank go by handle.
func TestTheHallOfFriesRanksTiesTogether(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	runs(t, repo, createAccount(t, repo, "test_maya"), 3, monday)
	runs(t, repo, createAccount(t, repo, "test_ben"), 3, monday)
	runs(t, repo, createAccount(t, repo, "test_cal"), 2, monday)
	runs(t, repo, createAccount(t, repo, "test_dee"), 1, monday)
	createAccount(t, repo, "test_eve") // no runs, so not on the board

	got, err := repo.HallOfFries(t.Context(), monday.Add(-time.Hour), nil, 10)
	if err != nil {
		t.Fatalf("hall of fries: %v", err)
	}

	want := []string{"1 test_ben 3", "1 test_maya 3", "3 test_cal 2", "4 test_dee 1"}
	rows := describe(got)
	if !slices.Equal(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

// TestTheHallOfFriesPagesWithAKeyset: a page resumes after the last row it showed, across a tie,
// and the ranks stay the whole board's, not the page's.
func TestTheHallOfFriesPagesWithAKeyset(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	runs(t, repo, createAccount(t, repo, "test_maya"), 3, monday)
	runs(t, repo, createAccount(t, repo, "test_ben"), 3, monday)
	runs(t, repo, createAccount(t, repo, "test_cal"), 2, monday)
	runs(t, repo, createAccount(t, repo, "test_dee"), 1, monday)
	since := monday.Add(-time.Hour)

	first, err := repo.HallOfFries(t.Context(), since, nil, 1)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("first page is empty, want test_ben")
	}
	last := first[len(first)-1]
	rest, err := repo.HallOfFries(t.Context(), since, &repository.BoardKey{Value: last.Value, Handle: last.Handle}, 10)
	if err != nil {
		t.Fatalf("next page: %v", err)
	}

	firstRows := describe(first)
	if !slices.Equal(firstRows, []string{"1 test_ben 3"}) {
		t.Errorf("first page = %v, want test_ben alone", firstRows)
	}
	restRows := describe(rest)
	if !slices.Equal(restRows, []string{"1 test_maya 3", "3 test_cal 2", "4 test_dee 1"}) {
		t.Errorf("next page = %v, want the rest from test_maya, ranks unchanged", restRows)
	}
}

// TestHallOfFriesRowIsTheCallersOwn: a player's row has their place on the whole board, and a
// player with no runs in the range has none.
func TestHallOfFriesRowIsTheCallersOwn(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	runs(t, repo, createAccount(t, repo, "test_maya"), 3, monday)
	runs(t, repo, createAccount(t, repo, "test_ben"), 3, monday)
	dee := createAccount(t, repo, "test_dee")
	runs(t, repo, dee, 1, monday)
	eve := createAccount(t, repo, "test_eve")
	since := monday.Add(-time.Hour)

	row, found, err := repo.HallOfFriesRow(t.Context(), dee.AccountID, since)
	if err != nil {
		t.Fatalf("dee's row: %v", err)
	}
	if !found || row.Rank != 3 || row.Handle != "test_dee" || row.Value != 1 {
		t.Errorf("dee's row = %+v (found %v), want rank 3 with 1 run", row, found)
	}
	_, found, err = repo.HallOfFriesRow(t.Context(), eve.AccountID, since)
	if err != nil {
		t.Fatalf("eve's row: %v", err)
	}
	if found {
		t.Error("eve has a row, want none: she has no runs")
	}
}

// TestTheHallOfFriesCountsFromItsStart: runs logged before the range starts are left out, so a
// new week starts everyone at nothing.
func TestTheHallOfFriesCountsFromItsStart(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	runs(t, repo, maya, 2, monday.AddDate(0, 0, -7)) // last week
	runs(t, repo, maya, 1, monday)
	ben := createAccount(t, repo, "test_ben")
	runs(t, repo, ben, 3, monday.AddDate(0, 0, -7))
	thisWeek := monday.Add(-time.Hour)

	got, err := repo.HallOfFries(t.Context(), thisWeek, nil, 10)
	if err != nil {
		t.Fatalf("this week: %v", err)
	}
	allTime, err := repo.HallOfFries(t.Context(), time.Time{}, nil, 10)
	if err != nil {
		t.Fatalf("all time: %v", err)
	}

	weekRows := describe(got)
	if !slices.Equal(weekRows, []string{"1 test_maya 1"}) {
		t.Errorf("this week = %v, want test_maya with this week's 1", weekRows)
	}
	allRows := describe(allTime)
	if !slices.Equal(allRows, []string{"1 test_ben 3", "1 test_maya 3"}) {
		t.Errorf("all time = %v, want both with 3", allRows)
	}
}

// TestTheFridgeOfTheWeekIsTheBestScanSinceItsStart: the highest score since the week started wins,
// a scan made the moment the week starts counts, and a better one from the minute before does not.
func TestTheFridgeOfTheWeekIsTheBestScanSinceItsStart(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_cal"), 99, monday.Add(-time.Minute))
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_maya"), 95, monday)
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_ben"), 91, monday.Add(time.Hour))

	got, found, err := repo.FridgeOfTheWeek(t.Context(), monday)
	if err != nil {
		t.Fatalf("fridge of the week: %v", err)
	}

	want := repository.FridgeRow{Handle: "test_maya", Avatar: "🍟", Score: 95, Roast: "a 95 fridge"}
	if !found || got != want {
		t.Errorf("fridge = %+v (found %v), want %+v", got, found, want)
	}
}

// TestTheFridgeOfTheWeekGoesToTheFirstOnATie: of two scans with the best score, the one made first
// keeps the fridge, so matching a score later never takes it.
func TestTheFridgeOfTheWeekGoesToTheFirstOnATie(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	// test_ben's scan is stored first and sorts first by handle, but test_maya's was made earlier.
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_ben"), 80, monday.Add(2*time.Hour))
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_maya"), 80, monday.Add(time.Hour))

	got, found, err := repo.FridgeOfTheWeek(t.Context(), monday)
	if err != nil {
		t.Fatalf("fridge of the week: %v", err)
	}

	if !found || got.Handle != "test_maya" {
		t.Errorf("fridge = %+v (found %v), want test_maya's, the first 80", got, found)
	}
}

// TestNoFridgeOfTheWeekBeforeAnyoneScans: a week nobody has scanned in has no fridge, even when
// last week had one.
func TestNoFridgeOfTheWeekBeforeAnyoneScans(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	fridgeAt(t, pool, repo, createAccount(t, repo, "test_cal"), 99, monday.AddDate(0, 0, -3))

	got, found, err := repo.FridgeOfTheWeek(t.Context(), monday)
	if err != nil {
		t.Fatalf("fridge of the week: %v", err)
	}

	if found {
		t.Errorf("fridge = %+v, want none: nobody has scanned this week", got)
	}
}

// runs logs n fry runs for a, one a day at noon from the day start falls on, so each is in a meal
// window of its own.
func runs(t *testing.T, repo *repository.Repository, a repository.Account, n int, start time.Time) {
	t.Helper()
	for i := range n {
		at := start.AddDate(0, 0, i)
		logFryRun(t, repo, a.AccountID, repository.FryRun{
			SpotID:     "mock-03",
			SpotName:   "Night Owl Fries",
			MealDay:    at,
			MealWindow: "midday",
			At:         at,
		})
	}
}

// describe writes rows as "rank handle value", so a whole board compares in one line.
func describe(rows []repository.BoardRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%d %s %d", r.Rank, r.Handle, r.Value))
	}
	return out
}

// fridgeAt stores a scan of score for a, with a roast that names the score, made at at.
func fridgeAt(t *testing.T, pool *pgxpool.Pool, repo *repository.Repository, a repository.Account, score int, at time.Time) {
	t.Helper()
	s, err := repo.CreateScan(t.Context(), a.AccountID, score, fmt.Sprintf("a %d fridge", score), nil)
	if err != nil {
		t.Fatalf("create scan for %s: %v", a.Handle, err)
	}
	_, err = pool.Exec(t.Context(), `update app.scan set created_at = $2 where scan_id = $1`, s.ScanID, at)
	if err != nil {
		t.Fatalf("stamp scan %d: %v", s.ScanID, err)
	}
}
