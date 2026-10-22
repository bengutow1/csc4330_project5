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
)

// known folds driver errors into the sentinels above and wraps everything else. op carries the
// ids for the server log; it never reaches a caller.
func known(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return fmt.Errorf("%s (%s): %w", op, pgErr.ConstraintName, ErrDuplicate)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// isNoRows reports whether a read matched no row, for the reads where none is an answer rather
// than a missing thing (a player's last scan before their first, a fry run already logged).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
