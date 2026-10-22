// Author: Caden Lund
package handlers_test

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1/friesv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1/scanv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/cook"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/fries"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/scan"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestANewcomerMakesAnAccountAndWhoAmIReadsItBack: the first launch's whole job. The handle is
// stored as the board shows it, the kitchen starts on its defaults, and the cookie from
// CreateAccount is what WhoAmI knows the player by.
func TestANewcomerMakesAnAccountAndWhoAmIReadsItBack(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := newcomer(t, srv.url)

	created, err := maya.CreateAccount(t.Context(), connect.NewRequest(&accountv1.CreateAccountRequest{
		Handle: "  Test_Maya ",
		Avatar: " 🌮",
	}))
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	me, err := maya.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Fatalf("who am i: %v", err)
	}

	for name, got := range map[string]*accountv1.Account{
		"create account": created.Msg.Account,
		"who am i":       me.Msg.Account,
	} {
		if got.GetHandle() != "test_maya" || got.GetAvatar() != "🌮" {
			t.Errorf("%s: handle %q, avatar %q, want test_maya and 🌮", name, got.GetHandle(), got.GetAvatar())
		}
		assertDefaultKitchen(t, name, got.GetKitchen())
	}
}

// TestTheSessionCookieAloneCarriesTheAccount: the cookie is the only credential. A second
// client holding it is the same player, and nothing else signs anyone in.
func TestTheSessionCookieAloneCarriesTheAccount(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := player(t, srv.url, "test_maya")
	sameCookie := newcomer(t, srv.url)
	stranger := newcomer(t, srv.url)
	target, err := url.Parse(srv.url)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	sameCookie.jar.SetCookies(target, maya.jar.Cookies(target))

	got, err := sameCookie.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Fatalf("who am i with maya's cookie: %v", err)
	}
	_, err = stranger.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))

	if got.Msg.Account.GetHandle() != "test_maya" {
		t.Errorf("handle = %q, want test_maya", got.Msg.Account.GetHandle())
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("a client without the cookie: got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
}

// TestWhoAmIWithoutASessionIsUnauthenticated: the app reads unauthenticated as "show
// onboarding", so a first launch must get exactly that code and not an error to toast.
func TestWhoAmIWithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})

	_, err := newcomer(t, srv.url).WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnauthenticated {
		t.Fatalf("got code %v (%v), want unauthenticated", cerr.Code(), err)
	}
	if cerr.Message() != auth.ErrNotAuthenticated.Error() {
		t.Errorf("message = %q, want %q", cerr.Message(), auth.ErrNotAuthenticated.Error())
	}
}

// TestWhoAmIAfterTheAccountIsGoneIsUnauthenticated: a session can outlive its account, and the
// next call must send the player back to onboarding rather than fail.
func TestWhoAmIAfterTheAccountIsGoneIsUnauthenticated(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := player(t, srv.url, "test_maya")
	_, err := srv.pool.Exec(t.Context(), `delete from app.account where handle = 'test_maya'`)
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}

	_, err = maya.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
}

// TestASessionThatCannotBeSavedLeavesTheHandleFree: the cookie is the account's only way in, so an
// account whose session was never saved can never be reached. It must not keep the handle, or the
// player's next try with the same name says it is taken.
func TestASessionThatCannotBeSavedLeavesTheHandleFree(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	// Without the sessions table the account's insert works and its session's save fails, as in
	// a database blip between the two.
	_, err := srv.pool.Exec(t.Context(), `drop table sessions`)
	if err != nil {
		t.Fatalf("drop sessions: %v", err)
	}
	maya := newcomer(t, srv.url)

	_, err = maya.CreateAccount(t.Context(), connect.NewRequest(&accountv1.CreateAccountRequest{
		Handle: "test_maya",
		Avatar: "🌮",
	}))

	// Unknown, not internal: the middleware's own save fails too, and its bare 500 goes out before
	// the handler's error. Any refusal before the insert would come back as another code. (CodeOf
	// says unknown for no error at all, hence the nil check.)
	if err == nil || connect.CodeOf(err) != connect.CodeUnknown {
		t.Fatalf("got code %v (%v), want unknown (the failed save's bare 500)", connect.CodeOf(err), err)
	}
	checked, err := maya.CheckHandle(t.Context(), connect.NewRequest(&accountv1.CheckHandleRequest{
		Handle: "test_maya",
	}))
	if err != nil {
		t.Fatalf("check handle: %v", err)
	}
	if !checked.Msg.Available {
		t.Error("the handle is still held by an account nobody can reach")
	}
}

// TestATakenHandleIsAFieldErrorOnHandle: onboarding shows it under the input, so it must arrive
// as a violation on handle, and the newcomer must still have no account.
func TestATakenHandleIsAFieldErrorOnHandle(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	player(t, srv.url, "test_maya")
	late := newcomer(t, srv.url)

	_, err := late.CreateAccount(t.Context(), connect.NewRequest(&accountv1.CreateAccountRequest{
		Handle: "TEST_MAYA", // the same handle once lowercased
		Avatar: "🍕",
	}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
	}
	got := fieldViolations(t, err)
	want := map[string]string{"handle": "that handle is taken"}
	if !maps.Equal(got, want) {
		t.Errorf("violations = %v, want %v", got, want)
	}
	_, err = late.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("after the refusal: got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
}

// TestAMalformedHandleOrAvatarIsRefusedFieldByField: every problem shows under its own input in
// one round trip, and nothing is stored.
func TestAMalformedHandleOrAvatarIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	cases := []struct {
		name, handle, avatar string
		want                 map[string]string
	}{
		{
			name: "both missing", handle: " ", avatar: "",
			want: map[string]string{"handle": "handle is required", "avatar": "avatar is required"},
		},
		{
			name: "too short", handle: "ab", avatar: "🍟",
			want: map[string]string{"handle": "handle must be at least 3 characters"},
		},
		{
			name: "too long", handle: strings.Repeat("a", 21), avatar: "🍟",
			want: map[string]string{"handle": "handle must be at most 20 characters"},
		},
		{
			name: "a space inside", handle: "test maya", avatar: "🍟",
			want: map[string]string{"handle": "handle can only use lowercase letters, numbers and underscores"},
		},
		{
			name: "an avatar that is a word", handle: "test_maya", avatar: "maya",
			want: map[string]string{"avatar": "avatar must be one emoji"},
		},
	}
	// The group returns once every parallel case has finished, so the count below sees them all.
	t.Run("cases", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				_, err := newcomer(t, srv.url).CreateAccount(t.Context(),
					connect.NewRequest(&accountv1.CreateAccountRequest{Handle: c.handle, Avatar: c.avatar}))

				if connect.CodeOf(err) != connect.CodeInvalidArgument {
					t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
				}
				got := fieldViolations(t, err)
				if !maps.Equal(got, c.want) {
					t.Errorf("violations = %v, want %v", got, c.want)
				}
			})
		}
	})

	var n int
	err := srv.pool.QueryRow(t.Context(), `select count(*) from app.account`).Scan(&n)
	if err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	if n != 0 {
		t.Errorf("accounts stored = %d, want 0", n)
	}
}

// TestAPlayerSavesTheirKitchenAndWhoAmIReadsItBack: onboarding's kitchen screen saves the whole
// kitchen, and what comes back is what was stored: each list in the closed set's order, each value
// once. Another player's kitchen never moves.
func TestAPlayerSavesTheirKitchenAndWhoAmIReadsItBack(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := player(t, srv.url, "test_maya")
	ben := player(t, srv.url, "test_ben")

	saved, err := maya.UpdateKitchen(t.Context(), connect.NewRequest(&accountv1.UpdateKitchenRequest{
		Kitchen: &accountv1.Kitchen{
			RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_EXTRA_CRISPY,
			// Out of the closed set's order, and one of them twice.
			Equipment: []accountv1.Equipment{
				accountv1.Equipment_EQUIPMENT_AIR_FRYER,
				accountv1.Equipment_EQUIPMENT_STOVETOP,
				accountv1.Equipment_EQUIPMENT_AIR_FRYER,
			},
			Diet: accountv1.Diet_DIET_VEGETARIAN,
			Allergies: []accountv1.Allergy{
				accountv1.Allergy_ALLERGY_SESAME, accountv1.Allergy_ALLERGY_PEANUTS,
			},
			Staples: []accountv1.Staple{accountv1.Staple_STAPLE_RICE, accountv1.Staple_STAPLE_SALT},
		},
	}))
	if err != nil {
		t.Fatalf("update kitchen: %v", err)
	}
	me, err := maya.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Fatalf("who am i: %v", err)
	}
	theirs, err := ben.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Fatalf("who am i, another player: %v", err)
	}

	want := &accountv1.Kitchen{
		RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_EXTRA_CRISPY,
		Equipment: []accountv1.Equipment{
			accountv1.Equipment_EQUIPMENT_STOVETOP, accountv1.Equipment_EQUIPMENT_AIR_FRYER,
		},
		Diet:      accountv1.Diet_DIET_VEGETARIAN,
		Allergies: []accountv1.Allergy{accountv1.Allergy_ALLERGY_PEANUTS, accountv1.Allergy_ALLERGY_SESAME},
		Staples:   []accountv1.Staple{accountv1.Staple_STAPLE_SALT, accountv1.Staple_STAPLE_RICE},
	}
	if saved.Msg.GetAccount().GetHandle() != "test_maya" {
		t.Errorf("saved account's handle = %q, want test_maya", saved.Msg.GetAccount().GetHandle())
	}
	assertKitchen(t, "the saved account", saved.Msg.GetAccount().GetKitchen(), want)
	assertKitchen(t, "WhoAmI after the save", me.Msg.GetAccount().GetKitchen(), want)
	assertDefaultKitchen(t, "another player", theirs.Msg.GetAccount().GetKitchen())
}

// TestAKitchenValueOutsideItsSetIsRefusedOnItsField: a client on a newer enum, or a broken one, has
// to hear which field is wrong and what it may hold, every problem at once, and the stored
// kitchen must not move.
func TestAKitchenValueOutsideItsSetIsRefusedOnItsField(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := player(t, srv.url, "test_maya")
	const (
		roastLevels = "roast level must be one of mild, medium, extra_crispy"
		diets       = "diet must be one of none, vegetarian, vegan, pescatarian"
	)
	cases := []struct {
		name    string
		kitchen *accountv1.Kitchen
		want    map[string]string
	}{
		{
			name:    "an unspecified roast level",
			kitchen: &accountv1.Kitchen{Diet: accountv1.Diet_DIET_NONE},
			want:    map[string]string{"kitchen.roast_level": roastLevels},
		},
		{
			name: "a roast level from a newer client",
			kitchen: &accountv1.Kitchen{
				RoastLevel: accountv1.RoastLevel(42), Diet: accountv1.Diet_DIET_NONE,
			},
			want: map[string]string{"kitchen.roast_level": roastLevels},
		},
		{
			name: "unknown equipment",
			kitchen: &accountv1.Kitchen{
				RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_MEDIUM,
				Equipment: []accountv1.Equipment{
					accountv1.Equipment_EQUIPMENT_OVEN, accountv1.Equipment(9),
				},
				Diet: accountv1.Diet_DIET_NONE,
			},
			want: map[string]string{
				"kitchen.equipment[1]": "equipment must be one of microwave, stovetop, oven, air_fryer",
			},
		},
		{
			name:    "an unspecified diet",
			kitchen: &accountv1.Kitchen{RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_MEDIUM},
			want:    map[string]string{"kitchen.diet": diets},
		},
		{
			name: "an unspecified allergy",
			kitchen: &accountv1.Kitchen{
				RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_MEDIUM,
				Diet:       accountv1.Diet_DIET_NONE,
				Allergies:  []accountv1.Allergy{accountv1.Allergy_ALLERGY_UNSPECIFIED},
			},
			want: map[string]string{
				"kitchen.allergies[0]": "allergies must be one of peanuts, tree_nuts, milk, eggs, " +
					"wheat, soy, fish, shellfish, sesame",
			},
		},
		{
			name: "an unknown staple",
			kitchen: &accountv1.Kitchen{
				RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_MEDIUM,
				Diet:       accountv1.Diet_DIET_NONE,
				Staples:    []accountv1.Staple{accountv1.Staple_STAPLE_SALT, accountv1.Staple(77)},
			},
			want: map[string]string{
				"kitchen.staples[1]": "staples must be one of salt, pepper, oil, butter, rice, pasta, " +
					"flour, sugar, spices",
			},
		},
		{
			name:    "no kitchen at all",
			kitchen: nil,
			want:    map[string]string{"kitchen.roast_level": roastLevels, "kitchen.diet": diets},
		},
	}
	// Grouped, so the check after it runs once every parallel case is done.
	t.Run("cases", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				_, err := maya.UpdateKitchen(t.Context(), connect.NewRequest(&accountv1.UpdateKitchenRequest{
					Kitchen: c.kitchen,
				}))
				if connect.CodeOf(err) != connect.CodeInvalidArgument {
					t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
				}
				got := fieldViolations(t, err)
				if !maps.Equal(got, c.want) {
					t.Errorf("violations = %v, want %v", got, c.want)
				}
			})
		}
	})

	me, err := maya.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Fatalf("who am i: %v", err)
	}
	assertDefaultKitchen(t, "after every refusal", me.Msg.GetAccount().GetKitchen())
}

// TestUpdateKitchenWithoutASessionIsUnauthenticated: only a player has a kitchen, and the app reads
// unauthenticated as "back to onboarding".
func TestUpdateKitchenWithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	stranger := newcomer(t, srv.url)

	_, err := stranger.UpdateKitchen(t.Context(), connect.NewRequest(&accountv1.UpdateKitchenRequest{
		Kitchen: &accountv1.Kitchen{
			RoastLevel: accountv1.RoastLevel_ROAST_LEVEL_MILD,
			Diet:       accountv1.Diet_DIET_NONE,
		},
	}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
}

// TestDeletingAnAccountTakesEverythingOfTheirsAndNothingElse: delete my data has to leave no row of
// the player's in any table and no cookie of theirs working, while another player's rows and
// session stay exactly as they were.
func TestDeletingAnAccountTakesEverythingOfTheirsAndNothingElse(t *testing.T) {
	t.Parallel()
	srv := everyServiceStack(t)
	maya := player(t, srv.url, "test_maya")
	ben := player(t, srv.url, "test_ben")
	mayaScan, benScan := keepEverything(t, srv, maya), keepEverything(t, srv, ben)
	mayaID, benID := accountIDOf(t, srv, "test_maya"), accountIDOf(t, srv, "test_ben")
	// Every table the setup writes, so a step that wrote nothing can't pass for a clean delete.
	mayasBefore := holdings(t, srv, mayaID, mayaScan)
	for _, table := range []string{
		"app.account (account_id)", "app.scan (account_id)", "app.plate (account_id)",
		"app.point_event (account_id)", "app.fry_run (account_id)", "app.scan_item (scan_id)",
		"app.recipe_set (scan_id)", "app.recipe (scan_id)",
	} {
		if mayasBefore[table] == 0 {
			t.Fatalf("%s holds none of maya's rows before the delete", table)
		}
	}
	bensBefore := holdings(t, srv, benID, benScan)
	target := serverURL(t, srv)
	mayaCookies := maya.jar.Cookies(target)

	_, err := maya.DeleteAccount(t.Context(), connect.NewRequest(&accountv1.DeleteAccountRequest{}))

	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	for table, n := range holdings(t, srv, mayaID, mayaScan) {
		if n != 0 {
			t.Errorf("%s still holds %d of maya's rows", table, n)
		}
	}
	bensAfter := holdings(t, srv, benID, benScan)
	if !maps.Equal(bensAfter, bensBefore) {
		t.Errorf("ben's rows = %v, want them as they were, %v", bensAfter, bensBefore)
	}
	if len(maya.jar.Cookies(target)) != 0 {
		t.Error("maya's phone still holds a cookie: the response should have cleared it")
	}
	stale := newcomer(t, srv.url)
	stale.jar.SetCookies(target, mayaCookies)
	_, err = stale.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("maya's old cookie: got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
	_, err = ben.WhoAmI(t.Context(), connect.NewRequest(&accountv1.WhoAmIRequest{}))
	if err != nil {
		t.Errorf("ben after maya's delete: %v", err)
	}
}

// TestDeleteAccountWithoutASessionIsUnauthenticated: only a player has data to delete, and the app
// reads unauthenticated as "back to onboarding".
func TestDeleteAccountWithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	stranger := newcomer(t, srv.url)

	_, err := stranger.DeleteAccount(t.Context(), connect.NewRequest(&accountv1.DeleteAccountRequest{}))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnauthenticated || cerr.Message() != auth.ErrNotAuthenticated.Error() {
		t.Fatalf("got %v %q, want unauthenticated", cerr.Code(), cerr.Message())
	}
}

// TestDeletingOneAccountFromManyPhonesAtOnceDeletesItOnce: phones holding one cookie can all tap
// delete. Exactly one call deletes, and every other is refused as signed out or not found, never
// as an internal error.
func TestDeletingOneAccountFromManyPhonesAtOnceDeletesItOnce(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	maya := player(t, srv.url, "test_maya")
	target := serverURL(t, srv)
	phones := make([]person, 20)
	for i := range phones {
		phones[i] = newcomer(t, srv.url)
		phones[i].jar.SetCookies(target, maya.jar.Cookies(target))
	}

	errs := make([]error, len(phones))
	var wg sync.WaitGroup
	for i, phone := range phones {
		wg.Go(func() {
			_, errs[i] = phone.DeleteAccount(t.Context(), connect.NewRequest(&accountv1.DeleteAccountRequest{}))
		})
	}
	wg.Wait()

	deleted := 0
	for _, err := range errs {
		switch connect.CodeOf(err) {
		case connect.CodeUnauthenticated, connect.CodeNotFound:
		default:
			if err != nil {
				t.Errorf("got code %v (%v), want success, unauthenticated or not_found", connect.CodeOf(err), err)
				continue
			}
			deleted++
		}
	}
	if deleted != 1 {
		t.Errorf("%d calls deleted the account, want exactly 1", deleted)
	}
	left := count(t, srv, `select count(*) from app.account`)
	if left != 0 {
		t.Errorf("accounts left = %d, want 0", left)
	}
}

// TestCheckHandleSaysWhetherAHandleIsFree: the answer onboarding shows while the player types,
// on the same rule CreateAccount holds them to.
func TestCheckHandleSaysWhetherAHandleIsFree(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{})
	asker := newcomer(t, srv.url)
	check := func(handle string) (*connect.Response[accountv1.CheckHandleResponse], error) {
		return asker.CheckHandle(t.Context(), connect.NewRequest(&accountv1.CheckHandleRequest{Handle: handle}))
	}

	before, err := check("test_maya")
	if err != nil {
		t.Fatalf("check a free handle: %v", err)
	}
	player(t, srv.url, "test_maya")
	after, err := check(" Test_Maya")
	if err != nil {
		t.Fatalf("check a taken handle: %v", err)
	}
	_, malformed := check("test maya")

	if !before.Msg.Available {
		t.Error("a free handle came back taken")
	}
	if after.Msg.Available {
		t.Error("a taken handle, typed in another case, came back free")
	}
	if connect.CodeOf(malformed) != connect.CodeInvalidArgument {
		t.Fatalf("a malformed handle: got code %v (%v), want invalid_argument", connect.CodeOf(malformed), malformed)
	}
	got := fieldViolations(t, malformed)
	want := map[string]string{"handle": "handle can only use lowercase letters, numbers and underscores"}
	if !maps.Equal(got, want) {
		t.Errorf("violations = %v, want %v", got, want)
	}
}

// TestThePublicRpcsAreLimitedPerClientIP: device accounts are free to make, so the two rpcs that
// need no session stop answering one address once it spends its limit.
func TestThePublicRpcsAreLimitedPerClientIP(t *testing.T) {
	t.Parallel()
	srv := accountStack(t, account.Limits{
		Create: handlers.NewLimiter(2, time.Hour),
		Check:  handlers.NewLimiter(3, time.Hour),
	})

	// Every client here calls from 127.0.0.1, so they share one budget per rpc.
	var creates []error
	for _, handle := range []string{"test_one", "test_two", "test_three"} {
		_, err := newcomer(t, srv.url).CreateAccount(t.Context(),
			connect.NewRequest(&accountv1.CreateAccountRequest{Handle: handle, Avatar: "🍟"}))
		creates = append(creates, err)
	}
	var checks []error
	for range 4 {
		_, err := newcomer(t, srv.url).CheckHandle(t.Context(),
			connect.NewRequest(&accountv1.CheckHandleRequest{Handle: "test_four"}))
		checks = append(checks, err)
	}

	if creates[0] != nil || creates[1] != nil {
		t.Fatalf("creates within the limit: %v, %v", creates[0], creates[1])
	}
	assertTooManyRequests(t, "the third create", creates[2])
	if slices.ContainsFunc(checks[:3], func(err error) bool { return err != nil }) {
		t.Fatalf("checks within the limit: %v", checks[:3])
	}
	assertTooManyRequests(t, "the fourth check", checks[3])
}

func assertTooManyRequests(t *testing.T, what string, err error) {
	t.Helper()
	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeResourceExhausted {
		t.Errorf("%s: code = %v, want resource_exhausted", what, cerr.Code())
	}
	if cerr.Message() != handlers.ErrTooManyRequests.Error() {
		t.Errorf("%s: message = %q, want %q", what, cerr.Message(), handlers.ErrTooManyRequests.Error())
	}
}

func assertKitchen(t *testing.T, name string, got, want *accountv1.Kitchen) {
	t.Helper()
	if got.GetRoastLevel() != want.GetRoastLevel() || got.GetDiet() != want.GetDiet() ||
		!slices.Equal(got.GetEquipment(), want.GetEquipment()) ||
		!slices.Equal(got.GetAllergies(), want.GetAllergies()) ||
		!slices.Equal(got.GetStaples(), want.GetStaples()) {
		t.Errorf("%s: kitchen = %v, want %v", name, got, want)
	}
}

func assertDefaultKitchen(t *testing.T, name string, k *accountv1.Kitchen) {
	t.Helper()
	if k.GetRoastLevel() != accountv1.RoastLevel_ROAST_LEVEL_MEDIUM {
		t.Errorf("%s: roast level = %v, want medium", name, k.GetRoastLevel())
	}
	if k.GetDiet() != accountv1.Diet_DIET_NONE {
		t.Errorf("%s: diet = %v, want none", name, k.GetDiet())
	}
	if len(k.GetEquipment()) != 0 || len(k.GetAllergies()) != 0 {
		t.Errorf("%s: equipment %v, allergies %v, want both empty", name, k.GetEquipment(), k.GetAllergies())
	}
	everyStaple := []accountv1.Staple{
		accountv1.Staple_STAPLE_SALT, accountv1.Staple_STAPLE_PEPPER, accountv1.Staple_STAPLE_OIL,
		accountv1.Staple_STAPLE_BUTTER, accountv1.Staple_STAPLE_RICE, accountv1.Staple_STAPLE_PASTA,
		accountv1.Staple_STAPLE_FLOUR, accountv1.Staple_STAPLE_SUGAR, accountv1.Staple_STAPLE_SPICES,
	}
	if !slices.Equal(k.GetStaples(), everyStaple) {
		t.Errorf("%s: staples = %v, want every staple", name, k.GetStaples())
	}
}

// accountStack mounts AccountService behind the session middleware, over the test's own
// database, which holds the sessions too.
func accountStack(t *testing.T, limits account.Limits) stack {
	t.Helper()
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		svc := account.NewAccount(account.Config{Repo: repo, Auth: authn, Limits: limits})
		mux.Handle(accountv1connect.NewAccountServiceHandler(svc))
	})
}

// person is one phone: a client and the cookie jar it keeps between calls.
type person struct {
	accountv1connect.AccountServiceClient
	jar *cookiejar.Jar
}

// newcomer is a fresh install: no cookie, no account.
func newcomer(t *testing.T, url string) person {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := accountv1connect.NewAccountServiceClient(&http.Client{Jar: jar}, url)
	return person{AccountServiceClient: client, jar: jar}
}

// serverURL is the stack's address, the key a phone's cookie jar files the session under.
func serverURL(t *testing.T, srv stack) *url.URL {
	t.Helper()
	u, err := url.Parse(srv.url)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	return u
}

// everyServiceStack mounts every service that writes a player's rows, the way main.go mounts them,
// so one player can hold a row in every table.
func everyServiceStack(t *testing.T) stack {
	t.Helper()
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		mux.Handle(accountv1connect.NewAccountServiceHandler(
			account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		mux.Handle(scanv1connect.NewScanServiceHandler(
			scan.NewScan(scan.Config{Repo: repo, Auth: authn, Vision: newFakeVision(kitchenFridge)})))
		mux.Handle(cookv1connect.NewCookServiceHandler(
			cook.NewCook(cook.Config{Repo: repo, Auth: authn, Vision: &fakeChef{recipes: kitchenRecipes}})))
		mux.Handle(friesv1connect.NewFriesServiceHandler(
			fries.NewFries(fries.Config{Repo: repo, Auth: authn, Spots: usualStreet(t), Log: discard()})))
	})
}

// keepEverything gives p a row in every table a feature writes, through the real rpcs: a scan and
// its items, the recipes for it, a plate of one of them with the points it paid, and a fry run.
// It returns the scan's id.
func keepEverything(t *testing.T, srv stack, p person) int64 {
	t.Helper()
	phone := &http.Client{Jar: p.jar}
	cook := cookPlayer{
		scan: scanv1connect.NewScanServiceClient(phone, srv.url),
		cook: cookv1connect.NewCookServiceClient(phone, srv.url),
	}
	scanID := scanFridge(t, cook).GetScanId()
	shelves := listRecipes(t, cook, &cookv1.ListRecipesRequest{ScanId: scanID})
	plateMeal(t, cook, firstOn(t, shelves.GetMakeItNow()).GetRecipeId(), fmt.Sprintf("plate of scan %d", scanID))
	_, err := friesv1connect.NewFriesServiceClient(phone, srv.url).LogFryRun(t.Context(),
		connect.NewRequest(nightOwlRun()))
	if err != nil {
		t.Fatalf("log fry run: %v", err)
	}
	return scanID
}

// holdings counts a player's rows in every app table, keyed by the table and the column that makes
// the row theirs: their account's id, or their scan's. A table that lands later is counted without
// anyone listing it here.
func holdings(t *testing.T, srv stack, accountID, scanID int64) map[string]int {
	t.Helper()
	out := map[string]int{}
	for column, id := range map[string]int64{"account_id": accountID, "scan_id": scanID} {
		rows, err := srv.pool.Query(t.Context(), `
			select c.table_name
			  from information_schema.columns c
			 where c.table_schema = 'app'
			   and c.column_name = $1`, column)
		if err != nil {
			t.Fatalf("tables with %s: %v", column, err)
		}
		tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatalf("tables with %s: %v", column, err)
		}
		for _, table := range tables {
			var n int
			err := srv.pool.QueryRow(t.Context(),
				`select count(*) from `+pgx.Identifier{"app", table}.Sanitize()+
					` where `+pgx.Identifier{column}.Sanitize()+` = $1`, id).Scan(&n)
			if err != nil {
				t.Fatalf("count app.%s: %v", table, err)
			}
			out[fmt.Sprintf("app.%s (%s)", table, column)] = n
		}
	}
	return out
}

// player is a newcomer who has made their account through the real rpc.
func player(t *testing.T, url, handle string) person {
	t.Helper()
	p := newcomer(t, url)
	_, err := p.CreateAccount(t.Context(), connect.NewRequest(&accountv1.CreateAccountRequest{
		Handle: handle,
		Avatar: "🍟",
	}))
	if err != nil {
		t.Fatalf("create account %s: %v", handle, err)
	}
	return p
}
