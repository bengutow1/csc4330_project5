// Author: Caden Lund
package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The only errors callers may branch on. Driver internals never leave this package: known folds
// them into these, and handlers.ConnectErr maps these to wire codes. Each message goes on screen
// verbatim.
var (
	// ErrNotFound fires when the row a caller named does not exist, or is not theirs to see.
	ErrNotFound = errors.New("not found")
	// ErrDuplicate fires when a unique constraint refuses a write.
	ErrDuplicate = errors.New("already exists")
	// ErrPlateSeen fires when a plate's photo has been plated before, by anyone.
	ErrPlateSeen = errors.New("we've seen this plate before")
	// ErrMealWindowCooked fires when the player already has this meal window's cook credit.
	ErrMealWindowCooked = errors.New("you already cooked this meal window")
	// ErrComboCodeMissing fires when no invite has this code.
	ErrComboCodeMissing = errors.New("no combo has that code")
	// ErrComboExpired fires when an invite's two hours have elapsed.
	ErrComboExpired = errors.New("that code has expired, ask for a new one")
	// ErrComboEnded fires when the host has ended the combo.
	ErrComboEnded = errors.New("that combo has already ended")
	// ErrComboFull fires when the guest seat is occupied.
	ErrComboFull = errors.New("that combo is full")
	// ErrNoCrewForCode fires when no crew holds the code a player typed to join.
	ErrNoCrewForCode = errors.New("no crew has that code, check it and try again")
	// ErrTooManyCrews fires when a player already in the most crews one can be in makes or joins
	// another.
	ErrTooManyCrews = errors.New("you're in as many crews as you can be, leave one first")
)

// uniqueRefusals are the unique constraints whose refusal the player reads in words of its own,
// rather than as ErrDuplicate.
var uniqueRefusals = map[string]error{
	"plate_photo_sha256_key":                 ErrPlateSeen,
	"point_event_account_id_meal_window_key": ErrMealWindowCooked,
}

// known folds driver errors into the sentinels above and wraps everything else. op carries the
// ids for the server log; it never reaches a caller.
func known(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		refusal, ok := uniqueRefusals[pgErr.ConstraintName]
		if !ok {
			refusal = ErrDuplicate
		}
		return fmt.Errorf("%s (%s): %w", op, pgErr.ConstraintName, refusal)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// isNoRows reports whether a read matched no row, for the reads where none is an answer rather
// than a missing thing (a player's last scan before their first, a fry run already logged).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
