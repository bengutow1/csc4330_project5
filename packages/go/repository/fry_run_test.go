// Author: Caden Lund
package repository_test

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestAFryRunIsStoredForItsPlayer: the Hall of Fries counts runs by account, so the row has to
// belong to the account that logged it and keep the spot as it was sent.
func TestAFryRunIsStoredForItsPlayer(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")

	id, err := repo.CreateFryRun(t.Context(), maya.AccountID, "mock-03", "Night Owl Fries")
	if err != nil {
		t.Fatalf("create fry run: %v", err)
	}

	var accountID int64
	var spotID, spotName string
	err = pool.QueryRow(t.Context(), `
		select f.account_id, f.spot_id, f.spot_name
		  from app.fry_run f
		 where f.fry_run_id = $1`, id).Scan(&accountID, &spotID, &spotName)
	if err != nil {
		t.Fatalf("read fry run %d back: %v", id, err)
	}
	if accountID != maya.AccountID || spotID != "mock-03" || spotName != "Night Owl Fries" {
		t.Errorf("stored account %d, spot %q (%q), want account %d, spot mock-03 (Night Owl Fries)",
			accountID, spotID, spotName, maya.AccountID)
	}
	got := fryRuns(t, pool, ben.AccountID)
	if got != 0 {
		t.Errorf("ben has %d fry runs, want 0", got)
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
	for _, a := range []repository.Account{maya, maya, ben} {
		_, err := repo.CreateFryRun(t.Context(), a.AccountID, "mock-03", "Night Owl Fries")
		if err != nil {
			t.Fatalf("create fry run for %s: %v", a.Handle, err)
		}
	}

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

// TestTheSchemaHoldsTheFryRunShape: the bounds the handler's tags restate, so a value past them
// is a check violation rather than a row the board has to cope with.
func TestTheSchemaHoldsTheFryRunShape(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	cases := []struct {
		name, spotID, spotName string
		ok                     bool
	}{
		{"one character each", "m", "N", true},
		{"a 256-character id", strings.Repeat("i", 256), "Night Owl Fries", true},
		{"a 120-character name", "mock-03", strings.Repeat("ñ", 120), true},
		{"no spot id", "", "Night Owl Fries", false},
		{"no spot name", "mock-03", "", false},
		{"a 257-character id", strings.Repeat("i", 257), "Night Owl Fries", false},
		{"a 121-character name", "mock-03", strings.Repeat("ñ", 121), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := repo.CreateFryRun(t.Context(), maya.AccountID, c.spotID, c.spotName)
			switch {
			case c.ok && err != nil:
				t.Errorf("refused: %v", err)
			case !c.ok && !isCheckViolation(err):
				t.Errorf("err = %v, want a check violation", err)
			}
		})
	}
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
