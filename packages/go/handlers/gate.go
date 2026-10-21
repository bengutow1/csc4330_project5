// Author: Caden Lund
package handlers

import (
	"context"

	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// Gate resolves who is calling from the session. Every rpc that acts for a player opens with one
// of its methods, so an rpc that forgot its gate shows by the absence.
type Gate struct {
	Auth *auth.Auth
}

// Account is the caller's account, re-read from the database. No session, or a session whose
// account is gone, is auth.ErrNotAuthenticated.
func (g Gate) Account(ctx context.Context) (repository.Account, error) {
	return g.Auth.CurrentAccount(ctx)
}
