// Author: Caden Lund
// Package auth holds the device account's session: signing a session in, and reading back whose
// it is. Handlers call it through handlers.Gate; it owns the rules, and they own the wire.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexedwards/scs/v2"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// Lifetime is how long a session lasts. The session is the account's only credential, so losing
// it loses the account: it lives a year.
const Lifetime = 365 * 24 * time.Hour

// keyAccountID is the one thing a session holds. Nothing outside this package knows the
// session's shape.
const keyAccountID = "account_id"

// Auth signs sessions in and says whose they are.
type Auth struct {
	sessions *scs.SessionManager
	repo     *repository.Repository
}

// New builds Auth over the session manager main.go configured and the shared repository.
func New(sessions *scs.SessionManager, repo *repository.Repository) *Auth {
	return &Auth{sessions: sessions, repo: repo}
}

// SignIn makes the request's session belong to accountID. It renews the token first, so a token
// handed out before the sign-in never ends up holding the account (session fixation). It saves
// the session before it returns, so a caller that wrote something for this sign-in can still undo
// it when the save fails.
func (a *Auth) SignIn(ctx context.Context, accountID int64) error {
	err := a.sessions.RenewToken(ctx)
	if err != nil {
		return fmt.Errorf("auth: renew token account=%d: %w", accountID, err)
	}
	a.sessions.Put(ctx, keyAccountID, accountID)
	// LoadAndSave saves it again as the response goes out, which rewrites the same row.
	_, _, err = a.sessions.Commit(ctx)
	if err != nil {
		return fmt.Errorf("auth: save session account=%d: %w", accountID, err)
	}
	return nil
}

// CurrentAccount is the session's account, read from the database on every call, so an account
// deleted a moment ago is gone on the next request. No session, or a session whose account no
// longer exists, is ErrNotAuthenticated.
func (a *Auth) CurrentAccount(ctx context.Context) (repository.Account, error) {
	// 0 when the session holds no account: identity ids start at 1.
	accountID := a.sessions.GetInt64(ctx, keyAccountID)
	if accountID == 0 {
		return repository.Account{}, ErrNotAuthenticated
	}
	acct, err := a.repo.AccountByID(ctx, accountID)
	if errors.Is(err, repository.ErrNotFound) {
		return repository.Account{}, ErrNotAuthenticated
	}
	if err != nil {
		return repository.Account{}, fmt.Errorf("auth: current account: %w", err)
	}
	return acct, nil
}
