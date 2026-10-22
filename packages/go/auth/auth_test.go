// Author: Caden Lund
package auth_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestCurrentAccountWithoutASessionIsNotAuthenticated: a first launch has no session, and has to
// reach onboarding rather than an error.
func TestCurrentAccountWithoutASessionIsNotAuthenticated(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	sessions := newSessions(pool)
	a := auth.New(sessions, repository.New(pool))

	_, err := a.CurrentAccount(request(t, sessions, ""))

	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("err = %v, want ErrNotAuthenticated", err)
	}
}

// TestASignedInSessionNamesItsAccountOnTheNextRequest: the cookie is the only credential, so the
// account has to come back from the stored session, not from anything the request says.
func TestASignedInSessionNamesItsAccountOnTheNextRequest(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	token := signIn(t, a, sessions, "", maya.AccountID)

	got, err := a.CurrentAccount(request(t, sessions, token))

	if err != nil {
		t.Fatalf("current account: %v", err)
	}
	if got.AccountID != maya.AccountID || got.Handle != "test_maya" {
		t.Errorf("current account = %+v, want test_maya's", got)
	}
}

// TestSignInNeverKeepsATokenFromBeforeIt: a token handed out before the sign-in (one an attacker
// planted, say) must not end up holding the account. That is session fixation.
func TestSignInNeverKeepsATokenFromBeforeIt(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	// A session that already exists, so its token is one the store knows.
	before := request(t, sessions, "")
	sessions.Put(before, "planted", true)
	planted := commit(t, before, sessions)

	token := signIn(t, a, sessions, planted, maya.AccountID)

	if token == planted {
		t.Fatal("sign in kept the token from before it")
	}
	_, err := a.CurrentAccount(request(t, sessions, planted))
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Errorf("the planted token after sign in: err = %v, want ErrNotAuthenticated", err)
	}
}

// TestSignInSavesTheSessionBeforeItReturns: a caller that wrote something for the sign-in (the
// account CreateAccount just made) has to hear of a failed save while it can still undo it, not
// after the response has gone out.
func TestSignInSavesTheSessionBeforeItReturns(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	ctx := request(t, sessions, "")

	err := a.SignIn(ctx, maya.AccountID)

	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	// No commit here, so the next request finds the session only if SignIn saved it.
	got, err := a.CurrentAccount(request(t, sessions, sessions.Token(ctx)))
	if err != nil {
		t.Fatalf("current account on the next request: %v", err)
	}
	if got.AccountID != maya.AccountID {
		t.Errorf("current account = %d, want %d", got.AccountID, maya.AccountID)
	}
}

// TestSignInFailsWhenTheSessionCannotBeSaved: the failed save is the caller's cue to undo, so it
// has to come back as an error.
func TestSignInFailsWhenTheSessionCannotBeSaved(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	_, err := pool.Exec(t.Context(), `drop table sessions`)
	if err != nil {
		t.Fatalf("drop sessions: %v", err)
	}

	err = a.SignIn(request(t, sessions, ""), maya.AccountID)

	if err == nil {
		t.Fatal("sign in reported success with no session saved")
	}
}

// TestASessionWhoseAccountIsGoneIsNotAuthenticated: a deleted account has to land on the next
// request as signed out, not as an internal error.
func TestASessionWhoseAccountIsGoneIsNotAuthenticated(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	token := signIn(t, a, sessions, "", maya.AccountID)
	_, err := pool.Exec(t.Context(), `delete from app.account where account_id = $1`, maya.AccountID)
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}

	_, err = a.CurrentAccount(request(t, sessions, token))

	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("err = %v, want ErrNotAuthenticated", err)
	}
}

// TestCurrentAccountReadsTheRowAfresh: the session holds only the id, so a change to the account
// shows on the very next request.
func TestCurrentAccountReadsTheRowAfresh(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	token := signIn(t, a, sessions, "", maya.AccountID)
	_, err := pool.Exec(t.Context(),
		`update app.account set roast_level = 'extra_crispy' where account_id = $1`, maya.AccountID)
	if err != nil {
		t.Fatalf("update account: %v", err)
	}

	got, err := a.CurrentAccount(request(t, sessions, token))

	if err != nil {
		t.Fatalf("current account: %v", err)
	}
	if got.RoastLevel != repository.RoastExtraCrispy {
		t.Errorf("roast level = %q, want %q", got.RoastLevel, repository.RoastExtraCrispy)
	}
}

// TestEndSessionsEndsEverySessionOfTheAccountAndNoOther: deleting an account has to sign out every
// session that held it, this request's with its cookie, while another player's stays signed in.
func TestEndSessionsEndsEverySessionOfTheAccountAndNoOther(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := newSessions(pool)
	a := auth.New(sessions, repo)
	maya := createAccount(t, repo, "test_maya")
	ben := createAccount(t, repo, "test_ben")
	here := signIn(t, a, sessions, "", maya.AccountID)
	// A second session holding maya, as a copied cookie would be.
	elsewhere := signIn(t, a, sessions, "", maya.AccountID)
	bens := signIn(t, a, sessions, "", ben.AccountID)
	ctx := request(t, sessions, here)
	bensRow := storedSession(t, pool, bens)

	err := a.EndSessions(ctx, maya.AccountID)

	if err != nil {
		t.Fatalf("end sessions: %v", err)
	}
	// Iterate hands each stored session to the callback and saves none back, so the store holds
	// ben's session exactly as it was, and no empty session for each one ended.
	left := storedSessions(t, pool)
	if !slices.Equal(left, []sessionRow{bensRow}) {
		t.Errorf("session store = %v, want only ben's, unchanged: %v", left, bensRow)
	}
	if sessions.Status(ctx) != scs.Destroyed {
		t.Errorf("this request's session status = %v, want destroyed, so the response clears its cookie",
			sessions.Status(ctx))
	}
	// Maya's account still exists here, so signed out means the session itself is gone.
	cases := []struct {
		name, token string
		want        int64 // the account the session still names, 0 for none
	}{
		{"this request's session", here, 0},
		{"maya's other session", elsewhere, 0},
		{"ben's session", bens, ben.AccountID},
	}
	for _, c := range cases {
		got, err := a.CurrentAccount(request(t, sessions, c.token))
		switch {
		case c.want == 0 && !errors.Is(err, auth.ErrNotAuthenticated):
			t.Errorf("%s: current account = %d, %v, want signed out", c.name, got.AccountID, err)
		case c.want != 0 && (err != nil || got.AccountID != c.want):
			t.Errorf("%s: current account = %d, %v, want %d", c.name, got.AccountID, err, c.want)
		}
	}
}

// sessionRow is one row of the session store, as scs's pgxstore keeps it.
type sessionRow struct {
	token  string
	data   string
	expiry time.Time
}

// storedSessions is every row in the session store, by token.
func storedSessions(t *testing.T, pool *pgxpool.Pool) []sessionRow {
	t.Helper()
	rows, err := pool.Query(t.Context(), `select token, data, expiry from sessions order by token`)
	if err != nil {
		t.Fatalf("read the session store: %v", err)
	}
	got, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (sessionRow, error) {
		var r sessionRow
		var data []byte
		err := row.Scan(&r.token, &data, &r.expiry)
		r.data = string(data)
		return r, err
	})
	if err != nil {
		t.Fatalf("read the session store: %v", err)
	}
	return got
}

// storedSession is the store's row for token.
func storedSession(t *testing.T, pool *pgxpool.Pool, token string) sessionRow {
	t.Helper()
	for _, r := range storedSessions(t, pool) {
		if r.token == token {
			return r
		}
	}
	t.Fatalf("no stored session for token %q", token)
	return sessionRow{}
}

// request loads the session a token names, the way LoadAndSave does when a request arrives. ""
// is a request with no cookie.
func request(t *testing.T, sessions *scs.SessionManager, token string) context.Context {
	t.Helper()
	ctx, err := sessions.Load(t.Context(), token)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	return ctx
}

// commit saves the session the way LoadAndSave does before the response, and returns the token
// the cookie carries.
func commit(t *testing.T, ctx context.Context, sessions *scs.SessionManager) string {
	t.Helper()
	token, _, err := sessions.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return token
}

// signIn runs one request that signs the session token names in as accountID, and returns the
// token the response's cookie carries.
func signIn(t *testing.T, a *auth.Auth, sessions *scs.SessionManager, token string, accountID int64) string {
	t.Helper()
	ctx := request(t, sessions, token)
	err := a.SignIn(ctx, accountID)
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	return commit(t, ctx, sessions)
}

func createAccount(t *testing.T, repo *repository.Repository, handle string) repository.Account {
	t.Helper()
	a, err := repo.CreateAccount(t.Context(), handle, "🍟")
	if err != nil {
		t.Fatalf("create account %s: %v", handle, err)
	}
	return a
}
