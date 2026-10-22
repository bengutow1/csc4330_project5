// Author: Caden Lund
package repository_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// campus is the campus zone, so the test's times read the way the plates' hours are judged.
var campus = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}
	return loc
}()

// TestAPlateAndItsAwardsAreStoredAsOneLedger: a player's points are a sum over the ledger, so the
// plate and each award must land, and the total must be the sum of what was awarded.
func TestAPlateAndItsAwardsAreStoredAsOneLedger(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	recipe := plateableRecipe(t, repo, maya.AccountID, 29)
	at := time.Date(2026, 10, 21, 18, 30, 0, 0, campus)

	plateID := createPlate(t, repo, maya.AccountID, recipe, "first", at)
	awardPoints(t, repo, maya.AccountID, plateID, "2026-10-21 evening", at,
		repository.Award{Reason: repository.ReasonCook, Points: 100},
		repository.Award{Reason: repository.ReasonAgainstAllOdds, Points: 100},
		repository.Award{Reason: repository.ReasonStreak, Points: 10})

	total, err := repo.PointsTotal(t.Context(), maya.AccountID)
	if err != nil || total != 210 {
		t.Errorf("points total = %d, %v; want 210", total, err)
	}
	var photo []byte
	var stamped time.Time
	err = pool.QueryRow(t.Context(), `
		select p.photo, p.created_at
		  from app.plate p
		 where p.plate_id = $1`, plateID).Scan(&photo, &stamped)
	if err != nil {
		t.Fatalf("read the plate: %v", err)
	}
	if string(photo) != string(platePhoto("first")) || !stamped.Equal(at) {
		t.Errorf("plate = %d bytes at %v, want the photo at %v", len(photo), stamped, at)
	}
}

// TestASeenPhotoIsRefusedForAnyone: the same photo must never count twice, whoever sends it.
func TestASeenPhotoIsRefusedForAnyone(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, campus)
	createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79), "dinner", at)

	for name, who := range map[string]repository.Account{"maya again": maya, "sam": sam} {
		_, err := repo.CreatePlate(t.Context(), who.AccountID, repository.NewPlate{
			RecipeID: plateableRecipe(t, repo, who.AccountID, 79), Photo: platePhoto("dinner"),
			PhotoSHA256: hashOf("dinner"), At: at.Add(time.Hour),
		})
		if !errors.Is(err, repository.ErrPlateSeen) {
			t.Errorf("%s: err = %v, want ErrPlateSeen", name, err)
		}
	}
	if n := rows(t, pool, `select count(*) from app.plate`); n != 1 {
		t.Errorf("%d plates stored, want the first alone", n)
	}
}

// TestAnotherPlayersRecipeTakesNoPlate: the plate reaches its owner through the recipe's scan, so
// nobody can plate (and earn on) a recipe that is not theirs.
func TestAnotherPlayersRecipeTakesNoPlate(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	mayas := plateableRecipe(t, repo, maya.AccountID, 79)

	_, err := repo.CreatePlate(t.Context(), sam.AccountID, repository.NewPlate{
		RecipeID: mayas, Photo: platePhoto("sam's"), PhotoSHA256: hashOf("sam's"),
		At: time.Date(2026, 10, 21, 12, 0, 0, 0, campus),
	})

	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("sam plating maya's recipe: err = %v, want ErrNotFound", err)
	}
	if n := rows(t, pool, `select count(*) from app.plate`); n != 0 {
		t.Errorf("%d plates stored, want none", n)
	}

	// Nor award points on a plate that is not theirs.
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, campus)
	mayasPlate := createPlate(t, repo, maya.AccountID, mayas, "maya's", at)
	err = repo.AwardPoints(t.Context(), sam.AccountID, mayasPlate, "2026-10-21 midday", at,
		[]repository.Award{{Reason: repository.ReasonCook, Points: 100}})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("sam's points on maya's plate: err = %v, want ErrNotFound", err)
	}
	if n := rows(t, pool, `select count(*) from app.point_event`); n != 0 {
		t.Errorf("%d awards stored, want none", n)
	}
}

// TestLockAccountHoldsOnlyARealAccount: the lock is how a player's plates take turns, and a
// session whose account is gone must not lock nothing and carry on.
func TestLockAccountHoldsOnlyARealAccount(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")

	err := repo.InTx(t.Context(), func(tx *repository.Repository) error {
		return tx.LockAccount(t.Context(), maya.AccountID)
	})
	if err != nil {
		t.Errorf("lock maya: %v", err)
	}
	err = repo.InTx(t.Context(), func(tx *repository.Repository) error {
		return tx.LockAccount(t.Context(), maya.AccountID+1000)
	})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("lock a missing account: err = %v, want ErrNotFound", err)
	}
}

// TestOneCookCreditPerMealWindow: a second plate in the same window must not earn the cook credit
// again, while the next window and another player's window are their own.
func TestOneCookCreditPerMealWindow(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	at := time.Date(2026, 10, 21, 18, 0, 0, 0, campus)
	cook := repository.Award{Reason: repository.ReasonCook, Points: 100}
	first := createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79), "a", at)
	awardPoints(t, repo, maya.AccountID, first, "2026-10-21 evening", at, cook)

	second := createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79), "b", at)
	err := repo.AwardPoints(t.Context(), maya.AccountID, second, "2026-10-21 evening", at, []repository.Award{cook})
	if !errors.Is(err, repository.ErrMealWindowCooked) {
		t.Errorf("a second cook credit in the window: err = %v, want ErrMealWindowCooked", err)
	}

	late := createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79), "c", at.Add(5*time.Hour))
	awardPoints(t, repo, maya.AccountID, late, "2026-10-21 late_night", at.Add(5*time.Hour), cook)
	theirs := createPlate(t, repo, sam.AccountID, plateableRecipe(t, repo, sam.AccountID, 79), "d", at)
	awardPoints(t, repo, sam.AccountID, theirs, "2026-10-21 evening", at, cook)
}

// TestTwentyPlatesRacingForOneWindowEarnOneCredit: twenty plates at once in one window must leave
// exactly one cook credit, which the index decides with no lock.
func TestTwentyPlatesRacingForOneWindowEarnOneCredit(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	at := time.Date(2026, 10, 21, 18, 0, 0, 0, campus)
	plates := make([]int64, 20)
	for i := range plates {
		plates[i] = createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79),
			fmt.Sprintf("racer %d", i), at)
	}

	var wg sync.WaitGroup
	errs := make([]error, len(plates))
	for i, plateID := range plates {
		wg.Go(func() {
			errs[i] = repo.AwardPoints(t.Context(), maya.AccountID, plateID, "2026-10-21 evening", at,
				[]repository.Award{{Reason: repository.ReasonCook, Points: 100}})
		})
	}
	wg.Wait()

	won := 0
	for i, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, repository.ErrMealWindowCooked):
			t.Errorf("racer %d: %v, want ErrMealWindowCooked or nothing", i, err)
		}
	}
	credits := rows(t, pool, `select count(*) from app.point_event where reason = 'cook'`)
	if won != 1 || credits != 1 {
		t.Errorf("%d racers won and %d credits stored, want exactly 1 of each", won, credits)
	}
}

// TestCookDaysAreTheMealWindowsDays: the streak counts the day each window belongs to, so a
// plate at 00:30 belongs to the night before, and a day with two plates counts once.
func TestCookDaysAreTheMealWindowsDays(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	cook := repository.Award{Reason: repository.ReasonCook, Points: 100}
	for i, w := range []string{"2026-10-19 evening", "2026-10-20 late_night", "2026-10-20 midday", "2026-10-22 morning"} {
		plateID := createPlate(t, repo, maya.AccountID, plateableRecipe(t, repo, maya.AccountID, 79),
			fmt.Sprintf("day %d", i), time.Date(2026, 10, 19+i, 12, 0, 0, 0, campus))
		awardPoints(t, repo, maya.AccountID, plateID, w, time.Date(2026, 10, 19+i, 12, 0, 0, 0, campus), cook)
	}

	days, err := repo.CookDays(t.Context(), maya.AccountID)

	if err != nil {
		t.Fatalf("cook days: %v", err)
	}
	if want := []string{"2026-10-19", "2026-10-20", "2026-10-22"}; !slices.Equal(days, want) {
		t.Errorf("cook days = %v, want %v", days, want)
	}
}

// TestPlayerStatsCountWhatTheBadgesRead: each badge reads one count, so each must count only
// what its feat is.
func TestPlayerStatsCountWhatTheBadgesRead(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	cook := repository.Award{Reason: repository.ReasonCook, Points: 100}

	none, err := repo.PlayerStats(t.Context(), maya.AccountID)
	if err != nil || none != (repository.PlayerStats{}) {
		t.Fatalf("a new player's stats = %+v, %v; want all zero", none, err)
	}

	// Scans that rise by 45 within a week, and a fall that never counts as a rise.
	stamped := []int64{
		scanAt(t, pool, repo, maya.AccountID, 30, "2026-10-12 12:00"),
		scanAt(t, pool, repo, maya.AccountID, 75, "2026-10-15 12:00"),
		scanAt(t, pool, repo, maya.AccountID, 20, "2026-10-16 12:00"),
		// A rise of 80 that took more than a week.
		scanAt(t, pool, repo, maya.AccountID, 100, "2026-10-24 12:00"),
	}

	// Four plates: three days in a row, then a gap.
	plate := func(score int, uses, missing, equipment []string, window string, at time.Time) {
		t.Helper()
		recipeID := recipeWith(t, repo, maya.AccountID, score, uses, missing, equipment)
		plateID := createPlate(t, repo, maya.AccountID, recipeID, window, at)
		awardPoints(t, repo, maya.AccountID, plateID, window, at, cook)
	}
	threeUses := []string{"Eggs", "Butter", "Salt"}
	fourUses := []string{"Eggs", "Spinach", "Cheddar", "Butter"}
	microwave := []string{repository.EquipmentMicrowave}
	plate(29, threeUses, nil, microwave, "2026-10-12 late_night", time.Date(2026, 10, 13, 0, 30, 0, 0, campus))
	plate(79, fourUses, nil, microwave, "2026-10-13 evening", time.Date(2026, 10, 13, 18, 0, 0, 0, campus))
	plate(79, fourUses, nil, []string{repository.EquipmentMicrowave, repository.EquipmentStovetop}, "2026-10-14 late_night", time.Date(2026, 10, 14, 23, 0, 0, 0, campus))
	plate(34, fourUses, nil, nil, "2026-10-17 morning", time.Date(2026, 10, 17, 8, 0, 0, 0, campus))
	// 04:30 is still late night, but past 4 am; and three uses and a missing thing take four.
	plate(79, threeUses, []string{"Bread"}, nil, "2026-10-17 late_night", time.Date(2026, 10, 18, 4, 30, 0, 0, campus))
	// Sam's plate counts for Sam alone.
	samsRecipe := recipeWith(t, repo, sam.AccountID, 10, threeUses, nil, microwave)
	samsPlate := createPlate(t, repo, sam.AccountID, samsRecipe, "sam", time.Date(2026, 10, 13, 1, 0, 0, 0, campus))
	awardPoints(t, repo, sam.AccountID, samsPlate, "2026-10-12 late_night", time.Date(2026, 10, 13, 1, 0, 0, 0, campus), cook)
	// The plates' own scans go back to September, so no rise runs through them.
	_, err = pool.Exec(t.Context(), `
		update app.scan
		   set created_at = '2026-09-01 12:00 America/Chicago'
		 where account_id = $1
		   and scan_id <> all ($2)`, maya.AccountID, stamped)
	if err != nil {
		t.Fatalf("move the plates' scans: %v", err)
	}

	got, err := repo.PlayerStats(t.Context(), maya.AccountID)

	if err != nil {
		t.Fatalf("player stats: %v", err)
	}
	want := repository.PlayerStats{
		// The four scans above, plus the one behind each of the five plates' recipes.
		Scans:     9,
		BestScore: 100,
		Cooks:     5,
		// The plates from scans that scored 29 and 34.
		CooksUnder35: 2,
		// The egg dish that takes three things, and not the one that lacks bread.
		CooksWithThreeOrFewer: 1,
		// Three days in a row: the 12th, 13th and 14th.
		LongestStreak: 3,
		// 00:30 counts; 23:00 and 04:30 are late night but outside midnight to 4 am.
		LateNightPlates: 1,
		// The two that need a microwave and nothing else.
		MicrowaveOnlyCooks: 2,
		BiggestWeeklyRise:  45,
	}
	if got != want {
		t.Errorf("stats = %+v\nwant %+v", got, want)
	}
}

func createPlate(t *testing.T, repo *repository.Repository, accountID, recipeID int64, photo string, at time.Time) int64 {
	t.Helper()
	id, err := repo.CreatePlate(t.Context(), accountID, repository.NewPlate{
		RecipeID: recipeID, Photo: platePhoto(photo), PhotoSHA256: hashOf(photo), At: at,
	})
	if err != nil {
		t.Fatalf("create plate %q: %v", photo, err)
	}
	return id
}

func awardPoints(
	t *testing.T, repo *repository.Repository, accountID, plateID int64, window string, at time.Time,
	awards ...repository.Award,
) {
	t.Helper()
	err := repo.AwardPoints(t.Context(), accountID, plateID, window, at, awards)
	if err != nil {
		t.Fatalf("award points in %s: %v", window, err)
	}
}

// plateableRecipe is a recipe on a fresh scan of the given score, ready to be plated.
func plateableRecipe(t *testing.T, repo *repository.Repository, accountID int64, score int) int64 {
	t.Helper()
	return recipeWith(t, repo, accountID, score, spinachSoup.Uses, nil, spinachSoup.Equipment)
}

// recipeWith makes a scan of the given score and a one-recipe set on it, and returns the recipe.
func recipeWith(t *testing.T, repo *repository.Repository, accountID int64, score int, uses, missing, equipment []string) int64 {
	t.Helper()
	s, err := repo.CreateScan(t.Context(), accountID, score, roast, fridge)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	r := spinachSoup
	r.Uses, r.Missing, r.Equipment = uses, missing, equipment
	createRecipes(t, repo, accountID, s.ScanID, 1, []repository.NewRecipe{r})
	got := listRecipes(t, repo, accountID, s.ScanID, 1, repository.RecipeFilter{})
	return got[0].RecipeID
}

// scanAt makes a scan, stamps it at a campus time with a score, and returns it, for the rises a
// week apart.
func scanAt(t *testing.T, pool *pgxpool.Pool, repo *repository.Repository, accountID int64, score int, at string) int64 {
	t.Helper()
	when, err := time.ParseInLocation("2006-01-02 15:04", at, campus)
	if err != nil {
		t.Fatalf("parse %q: %v", at, err)
	}
	s := createScan(t, repo, accountID, fridge)
	_, err = pool.Exec(t.Context(), `
		update app.scan
		   set score = $2, created_at = $3
		 where scan_id = $1`, s.ScanID, score, when)
	if err != nil {
		t.Fatalf("stamp scan: %v", err)
	}
	return s.ScanID
}

// platePhoto is a JPEG's opening bytes, then a name, so each name is a photo of its own.
func platePhoto(name string) []byte {
	return append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, name...)
}

func hashOf(name string) []byte {
	sum := sha256.Sum256(platePhoto(name))
	return sum[:]
}
