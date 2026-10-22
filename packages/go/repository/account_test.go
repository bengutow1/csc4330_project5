// Author: Caden Lund
package repository_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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

// TestDeleteAccountFreesTheHandle: CreateAccount undoes an account whose session could not be
// saved, and the handle has to come free with it. A second delete finds nothing to delete.
func TestDeleteAccountFreesTheHandle(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")

	deleted, err := repo.DeleteAccount(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	again, err := repo.DeleteAccount(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("delete account again: %v", err)
	}
	taken, err := repo.HandleTaken(t.Context(), "test_maya")
	if err != nil {
		t.Fatalf("handle taken: %v", err)
	}

	if !deleted || again {
		t.Errorf("deleted = %v, then %v, want true, then false", deleted, again)
	}
	if taken {
		t.Error("the deleted account's handle still reads as taken")
	}
	_, err = repo.AccountByID(t.Context(), ben.AccountID)
	if err != nil {
		t.Errorf("another account after the delete: %v", err)
	}
}

// TestUpdateKitchenStoresExactlyWhatWasSent: onboarding and settings send the whole kitchen, so
// the row ends up holding just that, a nil list stored as empty, and nobody else's row moves.
func TestUpdateKitchenStoresExactlyWhatWasSent(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	edit := repository.KitchenEdit{
		RoastLevel: repository.RoastExtraCrispy,
		Equipment:  []string{repository.EquipmentStovetop, repository.EquipmentAirFryer},
		Diet:       repository.DietVegetarian,
		Allergies:  nil,
		Staples:    []string{repository.StapleSalt, repository.StapleRice},
	}

	updated, err := repo.UpdateKitchen(t.Context(), maya.AccountID, edit)
	if err != nil {
		t.Fatalf("update kitchen: %v", err)
	}
	got, err := repo.AccountByID(t.Context(), maya.AccountID)
	if err != nil {
		t.Fatalf("account by id: %v", err)
	}
	untouched, err := repo.AccountByID(t.Context(), ben.AccountID)
	if err != nil {
		t.Fatalf("account by id: %v", err)
	}

	if !updated {
		t.Error("updated = false, want true")
	}
	if got.RoastLevel != edit.RoastLevel || got.Diet != edit.Diet {
		t.Errorf("roast level, diet = %q, %q, want %q, %q", got.RoastLevel, got.Diet,
			edit.RoastLevel, edit.Diet)
	}
	if !slices.Equal(got.Equipment, edit.Equipment) || !slices.Equal(got.Staples, edit.Staples) {
		t.Errorf("equipment, staples = %v, %v, want %v, %v", got.Equipment, got.Staples,
			edit.Equipment, edit.Staples)
	}
	if got.Allergies == nil || len(got.Allergies) != 0 {
		t.Errorf("allergies = %#v, want an empty list", got.Allergies)
	}
	if untouched.RoastLevel != ben.RoastLevel || !slices.Equal(untouched.Staples, ben.Staples) {
		t.Errorf("another player's kitchen = %+v, want it as made, %+v", untouched, ben)
	}
}

// TestUpdateKitchenOnAMissingAccountSaysSo: a session can outlive its account, and the handler
// turns false into not_found rather than a quiet success.
func TestUpdateKitchenOnAMissingAccountSaysSo(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	edit := repository.KitchenEdit{RoastLevel: repository.RoastMild, Diet: repository.DietVegan}

	updated, err := repo.UpdateKitchen(t.Context(), maya.AccountID+1, edit)

	if err != nil {
		t.Fatalf("update kitchen: %v", err)
	}
	if updated {
		t.Error("updated = true for an account that does not exist")
	}
}

// TestUpdateKitchenCannotStoreAWordOutsideItsSet: the handler maps from the wire first, and the
// check constraints are what still hold if one day it does not.
func TestUpdateKitchenCannotStoreAWordOutsideItsSet(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := createAccount(t, repo, "test_maya")
	edit := repository.KitchenEdit{
		RoastLevel: repository.RoastMedium,
		Equipment:  []string{"toaster"},
		Diet:       repository.DietNone,
	}

	_, err := repo.UpdateKitchen(t.Context(), maya.AccountID, edit)

	if !isCheckViolation(err) {
		t.Fatalf("err = %v, want a check violation", err)
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

// TestDeletingAnAccountCascadesThroughEveryTableThatHangsOffIt: DeleteAccount is one statement, so
// a table that names an account without on delete cascade, or one that blocks the delete further
// down (a scan's items), would fail delete my data for every player with a row in it. The catalog
// is read, so tables that land later are held to it too.
func TestDeletingAnAccountCascadesThroughEveryTableThatHangsOffIt(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	// owned is app.account and every table its delete cascades into, directly or through another.
	// confdeltype: c is cascade. r (restrict) refuses at once, partway down the cascade. a (no
	// action) is checked once the whole delete is done, so one from a table the account doesn't
	// own refuses it. One from an owned table refuses only if another player's row points at this
	// player's, which the schema can't show: the rpc that writes the row keeps it to the caller's
	// own (PlateMeal plates only the caller's recipes).
	rows, err := pool.Query(t.Context(), `
		with recursive owned(rel) as (
		       select 'app.account'::regclass
		        union
		       select con.conrelid
		         from pg_constraint con
		         join owned o on con.confrelid = o.rel
		        where con.contype = 'f'
		          and con.confdeltype = 'c')
		select con.conrelid::regclass::text || ' ' || con.conname,
		       con.confrelid = 'app.account'::regclass,
		       con.confdeltype,
		       con.conrelid in (select rel from owned)
		  from pg_constraint con
		  join owned o on con.confrelid = o.rel
		 where con.contype = 'f'`)
	if err != nil {
		t.Fatalf("foreign keys under app.account: %v", err)
	}
	type key struct {
		name         string
		toAccount    bool
		onDeleteRule string
		fromOwned    bool
	}
	keys, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (key, error) {
		var k key
		err := row.Scan(&k.name, &k.toAccount, &k.onDeleteRule, &k.fromOwned)
		return k, err
	})
	if err != nil {
		t.Fatalf("foreign keys under app.account: %v", err)
	}

	// app.scan references the account today, so an empty result would mean the query is broken.
	if len(keys) == 0 {
		t.Fatal("no foreign key references app.account, which proves nothing")
	}
	for _, k := range keys {
		switch {
		case k.toAccount && k.onDeleteRule != "c":
			t.Errorf("%s references app.account without on delete cascade (docs/contracts.md)", k.name)
		case k.onDeleteRule == "r":
			t.Errorf("%s would refuse an account's delete as it cascades down", k.name)
		case k.onDeleteRule == "a" && !k.fromOwned:
			t.Errorf("%s would refuse an account's delete: its rows outlive the cascade", k.name)
		}
	}
}

// isCheckViolation reports whether err is Postgres refusing a write on a check constraint.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" // check_violation
}
