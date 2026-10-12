// Author: Caden Lund
package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestKnownTurnsNoRowsIntoErrNotFound: handlers map ErrNotFound and nothing from pgx, so a driver
// error that slipped through would reach the client as an internal error.
func TestKnownTurnsNoRowsIntoErrNotFound(t *testing.T) {
	t.Parallel()

	err := known("scan by id account=7 scan=3", pgx.ErrNoRows)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("the pgx error still leaves the package: %v", err)
	}
	// The op stays for the server log.
	want := "scan by id account=7 scan=3: not found"
	if err.Error() != want {
		t.Errorf("message = %q, want %q", err.Error(), want)
	}
}

// TestKnownTurnsAUniqueViolationIntoErrDuplicate, keeping the constraint name, which says which
// uniqueness failed when a table has several.
func TestKnownTurnsAUniqueViolationIntoErrDuplicate(t *testing.T) {
	t.Parallel()
	violation := &pgconn.PgError{Code: "23505", ConstraintName: "account_handle_key"}

	err := known("create account", violation)

	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
	want := "create account (account_handle_key): already exists"
	if err.Error() != want {
		t.Errorf("message = %q, want %q", err.Error(), want)
	}
}

// TestKnownLeavesEveryOtherErrorUnfolded: a foreign key violation or a dead connection is a bug or
// an outage, never a condition a caller should branch on, so it must reach the default arm.
func TestKnownLeavesEveryOtherErrorUnfolded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
	}{
		{name: "foreign key violation", err: &pgconn.PgError{Code: "23503", ConstraintName: "scan_account_id_fkey"}},
		{name: "check violation", err: &pgconn.PgError{Code: "23514", ConstraintName: "scan_photo_kind_check"}},
		{name: "connection refused", err: errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := known("add scan account=7", c.err)

			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrDuplicate) {
				t.Errorf("folded into a sentinel: %v", err)
			}
			if !errors.Is(err, c.err) {
				t.Errorf("the cause is lost, so the log cannot show it: %v", err)
			}
		})
	}
}
