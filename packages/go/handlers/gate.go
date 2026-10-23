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
	Repo *repository.Repository // Required by the Combo gates.
}

// ComboMember hides a combo from anyone outside its current membership.
func (g Gate) ComboMember(ctx context.Context, comboID int64) (repository.Account, repository.Combo, error) {
	me, err := g.Account(ctx)
	if err != nil {
		return repository.Account{}, repository.Combo{}, err
	}
	invalid := Validate(struct {
		ComboID int64 `pfield:"combo_id" validate:"gt=0"`
	}{comboID})
	if invalid != nil {
		return me, repository.Combo{}, invalid
	}
	c, err := g.Repo.ComboByID(ctx, me.AccountID, comboID)
	return me, c, err
}

// ComboHost checks visibility before rights, so only a visible guest gets permission denied.
func (g Gate) ComboHost(ctx context.Context, comboID int64) (repository.Account, repository.Combo, error) {
	me, c, err := g.ComboMember(ctx, comboID)
	if err != nil {
		return me, c, err
	}
	if c.HostID != me.AccountID {
		return me, c, auth.ErrForbidden
	}
	return me, c, nil
}

// Account is the caller's account, re-read from the database. No session, or a session whose
// account is gone, is auth.ErrNotAuthenticated.
func (g Gate) Account(ctx context.Context) (repository.Account, error) {
	return g.Auth.CurrentAccount(ctx)
}
