// Author: Caden Lund
package handlers_test

import (
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
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
