// Author: Caden Lund
package repository_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestANewAccountStartsWithTheDefaultKitchen: onboarding's kitchen screen opens on these values,
// and a player who skips it cooks with them.
func TestANewAccountStartsWithTheDefaultKitchen(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)

	a, err := repo.CreateAccount(t.Context(), "test_maya", "🍟")

	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if a.AccountID <= 0 || a.Handle != "test_maya" || a.Avatar != "🍟" {
		t.Errorf("account = %+v, want the id, handle and avatar it was made with", a)
	}
	if a.RoastLevel != repository.RoastMedium {
		t.Errorf("roast level = %q, want %q", a.RoastLevel, repository.RoastMedium)
	}
	if a.Diet != repository.DietNone {
		t.Errorf("diet = %q, want %q", a.Diet, repository.DietNone)
	}
	if len(a.Equipment) != 0 || len(a.Allergies) != 0 {
		t.Errorf("equipment = %v, allergies = %v, want both empty", a.Equipment, a.Allergies)
	}
	everyStaple := []string{
		repository.StapleSalt, repository.StaplePepper, repository.StapleOil,
		repository.StapleButter, repository.StapleRice, repository.StaplePasta,
		repository.StapleFlour, repository.StapleSugar, repository.StapleSpices,
	}
	if !slices.Equal(a.Staples, everyStaple) {
		t.Errorf("staples = %v, want every staple %v", a.Staples, everyStaple)
	}
}

// TestATakenHandleIsErrDuplicate: the unique constraint is the only check that holds when two
// players race for one handle, so its refusal must arrive as the sentinel the handler maps.
func TestATakenHandleIsErrDuplicate(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	createAccount(t, repo, "test_maya")

	_, err := repo.CreateAccount(t.Context(), "test_maya", "🌮")

	if !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
}

// TestAccountByIDReadsTheRowBack, and a missing id is ErrNotFound rather than an outage.
func TestAccountByIDReadsTheRowBack(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	made := createAccount(t, repo, "test_maya")

	got, err := repo.AccountByID(t.Context(), made.AccountID)
	if err != nil {
		t.Fatalf("account by id: %v", err)
	}
	if got.Handle != made.Handle || got.Avatar != made.Avatar || got.RoastLevel != made.RoastLevel ||
		!slices.Equal(got.Staples, made.Staples) {
		t.Errorf("read back %+v, want %+v", got, made)
	}

	_, err = repo.AccountByID(t.Context(), made.AccountID+1)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("missing account: err = %v, want ErrNotFound", err)
	}
}

// TestHandleTakenSeesOnlyAHeldHandle: onboarding says "taken" only when someone has it.
func TestHandleTakenSeesOnlyAHeldHandle(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)

	before, err := repo.HandleTaken(t.Context(), "test_maya")
	if err != nil {
		t.Fatalf("handle taken: %v", err)
	}
	createAccount(t, repo, "test_maya")
	after, err := repo.HandleTaken(t.Context(), "test_maya")
	if err != nil {
		t.Fatalf("handle taken: %v", err)
	}

	if before {
		t.Error("a free handle read as taken")
	}
	if !after {
		t.Error("a held handle read as free")
	}
}

// TestTheSchemaHoldsTheHandleAndAvatarShape: the handler validates first, and these checks are
// what still hold if a write ever skips it.
func TestTheSchemaHoldsTheHandleAndAvatarShape(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	cases := []struct {
		name, handle, avatar string
		ok                   bool
	}{
		{"three characters", "abc", "🍟", true},
		{"twenty characters", strings.Repeat("a", 20), "🍟", true},
		{"digits and underscores", "test_4_real", "🍟", true},
		{"the longest emoji, ten code points", "test_kiss", "👩🏻‍❤️‍💋‍👨🏼", true},
		{"two characters", "ab", "🍟", false},
		{"twenty-one characters", strings.Repeat("a", 21), "🍟", false},
		{"a capital letter", "Test_maya", "🍟", false},
		{"a space", "test maya", "🍟", false},
		{"a dash", "test-maya", "🍟", false},
		{"an accented letter", "test_josé", "🍟", false},
		{"no avatar", "test_blank", "", false},
		{"eleven code points of avatar", "test_long", strings.Repeat("🍟", 11), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := repo.CreateAccount(t.Context(), c.handle, c.avatar)
			switch {
			case c.ok && err != nil:
				t.Errorf("refused: %v", err)
			case !c.ok && !isCheckViolation(err):
				t.Errorf("err = %v, want a check violation", err)
			}
		})
	}
}

// TestTheSchemaRefusesAValueOutsideEachClosedSet: a misspelled setting stored once would follow
// the player into every recipe.
func TestTheSchemaRefusesAValueOutsideEachClosedSet(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	a := createAccount(t, repo, "test_maya")
	cases := []struct{ name, set string }{
		{"roast level", `roast_level = 'scorched'`},
		{"equipment", `equipment = array['grill']`},
		{"diet", `diet = 'keto'`},
		{"allergies", `allergies = array['gluten']`},
		{"staples", `staples = array['ketchup']`},
		{"a null inside a list", `staples = array[null]::text[]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := pool.Exec(t.Context(),
				`update app.account set `+c.set+` where account_id = $1`, a.AccountID)
			if !isCheckViolation(err) {
				t.Errorf("err = %v, want a check violation", err)
			}
		})
	}
}

// isCheckViolation reports whether err is Postgres refusing a write on a check constraint.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" // check_violation
}
