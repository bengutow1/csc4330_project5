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
