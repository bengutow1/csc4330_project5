// Author: Caden Lund
package repository_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// fridge is a list as the handler stores it: trimmed names in known categories.
var fridge = []repository.ScanItem{
	{Name: "Eggs", Category: repository.CategoryProtein, Quantity: "x6"},
	{Name: "Spinach", Category: repository.CategoryVegetable, Quantity: "1 bag"},
	{Name: "Cheddar", Category: repository.CategoryDairy, Quantity: ""},
}

const roast = "Eggs, spinach and cheddar. Someone here owns a pan."

// TestCreateScanStoresTheVerdictAndItsListInOrder: the verdict screen reads back exactly what was
// judged, in the order the model listed it.
func TestCreateScanStoresTheVerdictAndItsListInOrder(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")

	created, err := repo.CreateScan(t.Context(), maya.AccountID, 79, roast, fridge)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	read, err := repo.ScanByID(t.Context(), maya.AccountID, created.ScanID)
	if err != nil {
		t.Fatalf("scan by id: %v", err)
	}

	for name, s := range map[string]repository.Scan{"created": created, "read back": read} {
		if s.ScanID == 0 || s.Score != 79 || s.Roast != roast || s.ItemsVersion != 1 {
			t.Errorf("%s: id %d, score %d, roast %q, items version %d; want an id, 79, the roast and 1",
				name, s.ScanID, s.Score, s.Roast, s.ItemsVersion)
		}
		if s.CreatedAt.IsZero() {
			t.Errorf("%s: no created_at", name)
		}
		if !slices.Equal(s.Items, fridge) {
			t.Errorf("%s: items = %v, want %v", name, s.Items, fridge)
		}
	}
}

// TestAScanThatFoundNoFoodIsStillAScan: an empty fridge is a verdict too (score 0), and reading it
// back must not mistake its empty list for a missing scan.
func TestAScanThatFoundNoFoodIsStillAScan(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")

	created, err := repo.CreateScan(t.Context(), maya.AccountID, 0, "Just the light.", nil)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	read, err := repo.ScanByID(t.Context(), maya.AccountID, created.ScanID)
	if err != nil {
		t.Fatalf("scan by id: %v", err)
	}

	if read.Score != 0 || len(read.Items) != 0 {
		t.Errorf("score %d, items %v; want 0 and none", read.Score, read.Items)
	}
}

// TestAnotherPlayersScanIsNotFoundAndStaysUntouched: the owner fence is in the SQL. Reading,
// editing or asking for the last scan as someone else finds nothing and changes nothing.
func TestAnotherPlayersScanIsNotFoundAndStaysUntouched(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	sam := createAccount(t, repo, "test_sam")
	scan := createScan(t, repo, maya.AccountID, fridge)

	_, readErr := repo.ScanByID(t.Context(), sam.AccountID, scan.ScanID)
	changed, editErr := repo.ReplaceItems(t.Context(), sam.AccountID, scan.ScanID,
		[]repository.ScanItem{{Name: "Ketchup", Category: repository.CategoryCondiment}})
	_, found, lastErr := repo.LastScan(t.Context(), sam.AccountID)

	if !errors.Is(readErr, repository.ErrNotFound) {
		t.Errorf("read as sam: err = %v, want ErrNotFound", readErr)
	}
	if editErr != nil || changed {
		t.Errorf("edit as sam: changed %v, err %v; want false and no error", changed, editErr)
	}
	if lastErr != nil || found {
		t.Errorf("last scan as sam: found %v, err %v; want none", found, lastErr)
	}
	after, err := repo.ScanByID(t.Context(), maya.AccountID, scan.ScanID)
	if err != nil {
		t.Fatalf("re-read maya's scan: %v", err)
	}
	if after.ItemsVersion != 1 || !slices.Equal(after.Items, fridge) {
		t.Errorf("maya's scan moved: items version %d, items %v", after.ItemsVersion, after.Items)
	}
}

// TestLastScanIsTheNewest: the kitchen shows the latest verdict, by when it was made, and a
// player with no scans has none rather than an error.
func TestLastScanIsTheNewest(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")

	_, foundBefore, err := repo.LastScan(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("last scan before any: %v", err)
	}
	first := createScan(t, repo, maya.AccountID, fridge)
	second := createScan(t, repo, maya.AccountID, fridge[:1])
	latest, found, err := repo.LastScan(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("last scan: %v", err)
	}
	// Restamped a minute back by the database's own clock, the second is no longer the newest.
	_, err = pool.Exec(t.Context(), `
		update app.scan
		   set created_at = created_at - interval '1 minute'
		 where scan_id = $1`, second.ScanID)
	if err != nil {
		t.Fatalf("restamp the second scan: %v", err)
	}
	restamped, _, err := repo.LastScan(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("last scan after the restamp: %v", err)
	}

	if foundBefore {
		t.Error("before any scan: found one")
	}
	if !found || latest.ScanID != second.ScanID || !slices.Equal(latest.Items, fridge[:1]) {
		t.Errorf("last scan = %d with %v, want the second, %d", latest.ScanID, latest.Items, second.ScanID)
	}
	if restamped.ScanID != first.ScanID {
		t.Errorf("after the restamp: last scan = %d, want the first, %d", restamped.ScanID, first.ScanID)
	}
}

// TestReplaceItemsSwapsTheListAndNeverTheScore: fixing the list re-runs the recipes, never the
// score. Each edit bumps the version, and an emptied list is still a list.
func TestReplaceItemsSwapsTheListAndNeverTheScore(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)
	fixed := []repository.ScanItem{
		{Name: "Greek yogurt", Category: repository.CategoryDairy, Quantity: "1 tub"},
		{Name: "Eggs", Category: repository.CategoryProtein, Quantity: "x6"},
	}

	changed, err := repo.ReplaceItems(t.Context(), maya.AccountID, scan.ScanID, fixed)
	if err != nil {
		t.Fatalf("replace items: %v", err)
	}
	edited := readScan(t, repo, maya.AccountID, scan.ScanID)
	emptied, err := repo.ReplaceItems(t.Context(), maya.AccountID, scan.ScanID, nil)
	if err != nil {
		t.Fatalf("empty the list: %v", err)
	}
	empty := readScan(t, repo, maya.AccountID, scan.ScanID)

	if !changed || !slices.Equal(edited.Items, fixed) || edited.ItemsVersion != 2 {
		t.Errorf("edit: changed %v, items %v, version %d; want true, %v, 2",
			changed, edited.Items, edited.ItemsVersion, fixed)
	}
	if !emptied || len(empty.Items) != 0 || empty.ItemsVersion != 3 {
		t.Errorf("emptied: changed %v, items %v, version %d; want true, none, 3",
			emptied, empty.Items, empty.ItemsVersion)
	}
	for _, s := range []repository.Scan{edited, empty} {
		if s.Score != scan.Score || s.Roast != scan.Roast {
			t.Errorf("score %d, roast %q after an edit; want %d and %q", s.Score, s.Roast, scan.Score, scan.Roast)
		}
	}
}

// TestReplaceItemsOnAScanThatIsNotThereChangesNothing: false is how the handler knows to answer
// not_found.
func TestReplaceItemsOnAScanThatIsNotThereChangesNothing(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")

	changed, err := repo.ReplaceItems(t.Context(), maya.AccountID, 999_999, fridge)

	if err != nil || changed {
		t.Errorf("changed %v, err %v; want false and no error", changed, err)
	}
}

// TestAFailedEditLeavesTheListAsItWas: the bump, the delete and the insert are one act, so a list
// the database refuses leaves the old one and its version in place.
func TestAFailedEditLeavesTheListAsItWas(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)

	_, err := repo.ReplaceItems(t.Context(), maya.AccountID, scan.ScanID,
		[]repository.ScanItem{{Name: strings.Repeat("a", 41), Category: repository.CategoryProtein}})

	if !isCheckViolation(err) {
		t.Fatalf("err = %v, want a check violation", err)
	}
	after := readScan(t, repo, maya.AccountID, scan.ScanID)
	if after.ItemsVersion != 1 || !slices.Equal(after.Items, fridge) {
		t.Errorf("after the refusal: version %d, items %v; want 1 and the old list", after.ItemsVersion, after.Items)
	}
}

// TestAFailedCreateStoresNoScan: a scan without its list would show a verdict for food that was
// never stored, so one refused item undoes the whole scan.
func TestAFailedCreateStoresNoScan(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")

	_, err := repo.CreateScan(t.Context(), maya.AccountID, 50, roast,
		[]repository.ScanItem{fridge[0], {Name: "Steak", Category: "meat"}})

	if !isCheckViolation(err) {
		t.Fatalf("err = %v, want a check violation", err)
	}
	scans := rows(t, pool, `select count(*) from app.scan`)
	items := rows(t, pool, `select count(*) from app.scan_item`)
	if scans != 0 || items != 0 {
		t.Errorf("stored %d scans and %d items, want none", scans, items)
	}
}

// TestTheSchemaHoldsTheScanShape: the handler validates first, and these checks are what still
// hold if a write ever skips it.
func TestTheSchemaHoldsTheScanShape(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	eggs := fridge[0]
	cases := []struct {
		name  string
		score int
		item  repository.ScanItem
		ok    bool
	}{
		{name: "a score of 0", score: 0, item: eggs, ok: true},
		{name: "a score of 100", score: 100, item: eggs, ok: true},
		{name: "a score below 0", score: -1, item: eggs, ok: false},
		{name: "a score over 100", score: 101, item: eggs, ok: false},
		{name: "a forty-character name", score: 50, item: item(strings.Repeat("a", 40), ""), ok: true},
		{name: "an empty name", score: 50, item: item("", ""), ok: false},
		{name: "a forty-one-character name", score: 50, item: item(strings.Repeat("a", 41), ""), ok: false},
		{name: "a twenty-character quantity", score: 50, item: item("Eggs", strings.Repeat("x", 20)), ok: true},
		{name: "a twenty-one-character quantity", score: 50, item: item("Eggs", strings.Repeat("x", 21)), ok: false},
		{name: "a category outside the set", score: 50, item: repository.ScanItem{Name: "Steak", Category: "meat"}, ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := repo.CreateScan(t.Context(), maya.AccountID, c.score, roast, []repository.ScanItem{c.item})
			switch {
			case c.ok && err != nil:
				t.Errorf("refused: %v", err)
			case !c.ok && !isCheckViolation(err):
				t.Errorf("err = %v, want a check violation", err)
			}
		})
	}
}

// TestEveryCategoryConstIsOneTheSchemaTakes: the consts mirror app.scan_item's check by hand, so a
// misspelled one would pass review and fail on its first scan.
func TestEveryCategoryConstIsOneTheSchemaTakes(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	every := []string{
		repository.CategoryProtein, repository.CategoryVegetable, repository.CategoryFruit,
		repository.CategoryGrain, repository.CategoryDairy, repository.CategoryCondiment,
		repository.CategoryDrink, repository.CategorySnack, repository.CategoryLeftover,
		repository.CategoryOther,
	}
	items := make([]repository.ScanItem, 0, len(every))
	for _, category := range every {
		items = append(items, repository.ScanItem{Name: "A " + category, Category: category})
	}

	scan, err := repo.CreateScan(t.Context(), maya.AccountID, 50, roast, items)

	if err != nil {
		t.Fatalf("a list with every category: %v", err)
	}
	if !slices.Equal(scan.Items, items) {
		t.Errorf("items = %v, want %v", scan.Items, items)
	}
}

// TestTwoItemsCannotShareAPlace: the list's order is its positions, so a duplicate would make it
// ambiguous.
func TestTwoItemsCannotShareAPlace(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	scan := createScan(t, repo, maya.AccountID, fridge)

	_, err := pool.Exec(t.Context(), `
		insert into app.scan_item (scan_id, name, category, quantity, position)
		values ($1, 'Ketchup', 'condiment', '', 1)`, scan.ScanID)

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" { // unique_violation
		t.Errorf("err = %v, want a unique violation", err)
	}
}

// TestDeletingAnAccountTakesItsScansWithIt: a player who deletes their data leaves no verdict and
// no list behind.
func TestDeletingAnAccountTakesItsScansWithIt(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	createScan(t, repo, maya.AccountID, fridge)

	_, err := pool.Exec(t.Context(), `delete from app.account where account_id = $1`, maya.AccountID)

	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	scans := rows(t, pool, `select count(*) from app.scan`)
	items := rows(t, pool, `select count(*) from app.scan_item`)
	if scans != 0 || items != 0 {
		t.Errorf("%d scans and %d items left, want none", scans, items)
	}
}

// TestBackdateScanMovesOnlyTheOwnersScan: the demo seed lays last week's scans down by moving
// them back, and the move is fenced like every write: another player's scan id changes nothing.
func TestBackdateScanMovesOnlyTheOwnersScan(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	mine := createScan(t, repo, maya.AccountID, fridge)
	bens := createScan(t, repo, ben.AccountID, fridge)
	lastWeek := mine.CreatedAt.AddDate(0, 0, -7).Truncate(time.Second)

	moved, err := repo.BackdateScan(t.Context(), maya.AccountID, mine.ScanID, lastWeek)
	if err != nil {
		t.Fatalf("backdate maya's scan: %v", err)
	}
	stolen, err := repo.BackdateScan(t.Context(), maya.AccountID, bens.ScanID, lastWeek)
	if err != nil {
		t.Fatalf("backdate ben's scan as maya: %v", err)
	}

	mayas := readScan(t, repo, maya.AccountID, mine.ScanID).CreatedAt
	bensAfter := readScan(t, repo, ben.AccountID, bens.ScanID).CreatedAt
	if !moved || !mayas.Equal(lastWeek) {
		t.Errorf("maya's scan: moved %v, made at %v, want moved to %v", moved, mayas, lastWeek)
	}
	if stolen || !bensAfter.Equal(bens.CreatedAt) {
		t.Errorf("ben's scan through maya: moved %v, made at %v, want untouched at %v", stolen, bensAfter, bens.CreatedAt)
	}
}

func createScan(t *testing.T, repo *repository.Repository, accountID int64, items []repository.ScanItem) repository.Scan {
	t.Helper()
	s, err := repo.CreateScan(t.Context(), accountID, 79, roast, items)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	return s
}

func readScan(t *testing.T, repo *repository.Repository, accountID, scanID int64) repository.Scan {
	t.Helper()
	s, err := repo.ScanByID(t.Context(), accountID, scanID)
	if err != nil {
		t.Fatalf("scan by id: %v", err)
	}
	return s
}

// item is a protein named name, for the cases where only the name and quantity matter.
func item(name, quantity string) repository.ScanItem {
	return repository.ScanItem{Name: name, Category: repository.CategoryProtein, Quantity: quantity}
}

func rows(t *testing.T, pool *pgxpool.Pool, count string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), count).Scan(&n)
	if err != nil {
		t.Fatalf("%s: %v", count, err)
	}
	return n
}
