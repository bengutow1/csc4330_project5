// Author: Caden Lund
package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"

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
