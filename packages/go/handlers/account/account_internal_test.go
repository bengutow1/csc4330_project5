// Author: Caden Lund
package account

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestAFailedUndoIsLoggedEvenWhenTheClientHungUp: CreateAccount answers a client that hung up with
// canceled, which ConnectErr never logs, so a delete that failed there would leave an account
// holding its handle with no trace of why.
func TestAFailedUndoIsLoggedEvenWhenTheClientHungUp(t *testing.T) {
	t.Parallel()
	// Never dialed: a closed pool refuses every query at once.
	pool, err := pgxpool.New(t.Context(), "postgres://test.invalid:1/none")
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	pool.Close()
	var logged bytes.Buffer
	h := NewAccount(Config{
		Repo: repository.New(pool),
		Log:  slog.New(slog.NewTextHandler(&logged, nil)),
	})
	ctx, hangUp := context.WithCancel(t.Context())
	hangUp()

	h.undoCreate(ctx, 42)

	line := logged.String()
	if !strings.Contains(line, "level=WARN") || !strings.Contains(line, "account_id=42") {
		t.Errorf("log = %q, want a WARN line naming account 42", line)
	}
}
