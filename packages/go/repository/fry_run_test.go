// Author: Caden Lund
package repository_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestAFryRunIsStoredForItsPlayer: the Hall of Fries counts runs by account and week, so the row
// keeps whose it is, the spot as it was sent, its meal window and when it was logged.
func TestAFryRunIsStoredForItsPlayer(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	lunch := fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:05:00Z")

	id := logFryRun(t, repo, maya.AccountID, lunch)

	var accountID int64
	var spotID, spotName, day, window string
	var at time.Time
	err := pool.QueryRow(t.Context(), `
		select f.account_id, f.spot_id, f.spot_name, f.meal_day::text, f.meal_window, f.created_at
		  from app.fry_run f
		 where f.fry_run_id = $1`, id).Scan(&accountID, &spotID, &spotName, &day, &window, &at)
	if err != nil {
		t.Fatalf("read fry run %d back: %v", id, err)
	}
	if accountID != maya.AccountID || spotID != "mock-03" || spotName != "Night Owl Fries" {
		t.Errorf("stored account %d, spot %q (%q), want account %d, spot mock-03 (Night Owl Fries)",
			accountID, spotID, spotName, maya.AccountID)
	}
	if day != "2026-10-20" || window != "midday" || !at.Equal(lunch.At) {
		t.Errorf("stored %s %s at %v, want 2026-10-20 midday at %v", day, window, at, lunch.At)
	}
	runs := fryRuns(t, pool, ben.AccountID)
	if runs != 0 {
		t.Errorf("ben has %d fry runs, want 0", runs)
	}
}

// TestOneFryRunCountsPerMealWindow: a second run in the same window, at the same spot or another,
// returns the first run's id and stores nothing, so tapping again cannot pad the Hall of Fries.
// The next window, or another player in the same one, logs a run of its own.
func TestOneFryRunCountsPerMealWindow(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")

	lunch := logFryRun(t, repo, maya.AccountID,
		fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:05:00Z"))
	again := logFryRun(t, repo, maya.AccountID,
		fryRun(t, "mock-05", "Drumstick Drive-Thru", "2026-10-20", "midday", "2026-10-20T19:30:00Z"))
	dinner := logFryRun(t, repo, maya.AccountID,
		fryRun(t, "mock-05", "Drumstick Drive-Thru", "2026-10-20", "evening", "2026-10-20T23:00:00Z"))
	bensLunch := logFryRun(t, repo, ben.AccountID,
		fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:06:00Z"))

	if again != lunch {
		t.Errorf("a second run at lunch got id %d, want lunch's %d", again, lunch)
	}
	if dinner == lunch || bensLunch == lunch || bensLunch == dinner {
		t.Errorf("ids lunch %d, dinner %d, ben's lunch %d, want three runs", lunch, dinner, bensLunch)
	}
	var spotID string
	err := pool.QueryRow(t.Context(), `
		select f.spot_id
		  from app.fry_run f
		 where f.fry_run_id = $1`, lunch).Scan(&spotID)
	if err != nil {
		t.Fatalf("read lunch back: %v", err)
	}
	if spotID != "mock-03" {
		t.Errorf("lunch is at %s, want mock-03: the second run changes nothing", spotID)
	}
	runs := fryRuns(t, pool, maya.AccountID)
	if runs != 2 {
		t.Errorf("maya has %d fry runs, want 2 (lunch and dinner)", runs)
	}
}

// TestRacingTapsStoreOneFryRun: twenty taps landing at once still store one run, and every tap
// gets its id, because the unique constraint decides and not a check before the insert.
func TestRacingTapsStoreOneFryRun(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	lunch := fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:05:00Z")

	var wg sync.WaitGroup
	ids := make([]int64, 20)
	errs := make([]error, 20)
	for i := range ids {
		wg.Go(func() {
			ids[i], errs[i] = repo.LogFryRun(t.Context(), maya.AccountID, lunch)
		})
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("tap %d: %v", i, err)
		}
		if ids[i] != ids[0] {
			t.Errorf("tap %d got id %d, tap 0 got %d, want one run for all", i, ids[i], ids[0])
		}
	}
	runs := fryRuns(t, pool, maya.AccountID)
	if runs != 1 {
		t.Errorf("maya has %d fry runs, want 1", runs)
	}
}

// TestDeletingAnAccountDeletesItsFryRuns: deleting an account (#16) has to leave nothing of the
// player behind, so the foreign key cascades.
func TestDeletingAnAccountDeletesItsFryRuns(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	logFryRun(t, repo, maya.AccountID, fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:05:00Z"))
	logFryRun(t, repo, maya.AccountID, fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "late_night", "2026-10-21T04:30:00Z"))
	logFryRun(t, repo, ben.AccountID, fryRun(t, "mock-03", "Night Owl Fries", "2026-10-20", "midday", "2026-10-20T17:06:00Z"))

	_, err := pool.Exec(t.Context(), `delete from app.account where account_id = $1`, maya.AccountID)
	if err != nil {
		t.Fatalf("delete maya: %v", err)
	}

	left := fryRuns(t, pool, maya.AccountID)
	if left != 0 {
		t.Errorf("maya still has %d fry runs after her account went, want 0", left)
	}
	kept := fryRuns(t, pool, ben.AccountID)
	if kept != 1 {
		t.Errorf("ben has %d fry runs, want his 1 untouched", kept)
	}
}

// TestTheSchemaHoldsTheFryRunShape: the bounds the handler's tags restate and the four meal
// windows, so a value past them is a check violation rather than a row the board has to cope
// with. Each case logs in its own window, so none returns an earlier case's run.
func TestTheSchemaHoldsTheFryRunShape(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	cases := []struct {
		name, spotID, spotName, day, window string
		ok                                  bool
	}{
		{"one character each", "m", "N", "2026-10-01", "midday", true},
		{"a 256-character id", strings.Repeat("i", 256), "Night Owl Fries", "2026-10-02", "midday", true},
		{"a 120-character name", "mock-03", strings.Repeat("ñ", 120), "2026-10-03", "midday", true},
		{"every window: morning", "mock-03", "Night Owl Fries", "2026-10-04", "morning", true},
		{"every window: evening", "mock-03", "Night Owl Fries", "2026-10-04", "evening", true},
		{"every window: late night", "mock-03", "Night Owl Fries", "2026-10-04", "late_night", true},
		{"no spot id", "", "Night Owl Fries", "2026-10-05", "midday", false},
		{"no spot name", "mock-03", "", "2026-10-06", "midday", false},
		{"a 257-character id", strings.Repeat("i", 257), "Night Owl Fries", "2026-10-07", "midday", false},
		{"a 121-character name", "mock-03", strings.Repeat("ñ", 121), "2026-10-08", "midday", false},
		{"a window that is not one", "mock-03", "Night Owl Fries", "2026-10-09", "brunch", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			run := fryRun(t, c.spotID, c.spotName, c.day, c.window, c.day+"T12:00:00Z")
			_, err := repo.LogFryRun(t.Context(), maya.AccountID, run)
			switch {
			case c.ok && err != nil:
				t.Errorf("refused: %v", err)
			case !c.ok && !isCheckViolation(err):
				t.Errorf("err = %v, want a check violation", err)
			}
		})
	}
}

// fryRun is a trip to spotID on the campus day and meal window given, logged at the RFC 3339
// time at. The repository stores what it is told: points.MealWindow is the handler's to call.
func fryRun(t *testing.T, spotID, spotName, day, window, at string) repository.FryRun {
	t.Helper()
	mealDay, err := time.Parse(time.DateOnly, day)
	if err != nil {
		t.Fatalf("test day %q: %v", day, err)
	}
	when, err := time.Parse(time.RFC3339, at)
	if err != nil {
		t.Fatalf("test time %q: %v", at, err)
	}
	return repository.FryRun{
		SpotID:     spotID,
		SpotName:   spotName,
		MealDay:    mealDay,
		MealWindow: window,
		At:         when,
	}
}

func logFryRun(t *testing.T, repo *repository.Repository, accountID int64, run repository.FryRun) int64 {
	t.Helper()
	id, err := repo.LogFryRun(t.Context(), accountID, run)
	if err != nil {
		t.Fatalf("log fry run %+v: %v", run, err)
	}
	return id
}

func fryRuns(t *testing.T, pool *pgxpool.Pool, accountID int64) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), `
		select count(*)
		  from app.fry_run f
		 where f.account_id = $1`, accountID).Scan(&n)
	if err != nil {
		t.Fatalf("count fry runs account=%d: %v", accountID, err)
	}
	return n
}
