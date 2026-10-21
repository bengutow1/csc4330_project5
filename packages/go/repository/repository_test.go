// Author: Caden Lund
package repository_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

var errBoom = errors.New("boom")

// TestPingAnswersFromARealDatabase: the readiness probe is only as honest as Ping.
func TestPingAnswersFromARealDatabase(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)

	err := repo.Ping(t.Context())

	if err != nil {
		t.Fatalf("ping: %v", err)
	}
}

// TestPingFailsOnceTheDatabaseIsGone keeps the probe from reporting ready while nothing answers.
func TestPingFailsOnceTheDatabaseIsGone(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	pool.Close()

	err := repo.Ping(t.Context())

	if err == nil {
		t.Fatal("ping succeeded on a closed pool")
	}
	if errors.Is(err, repository.ErrNotFound) {
		t.Errorf("an outage folded into ErrNotFound: %v", err)
	}
}

// TestInTxCommitsEveryWriteWhenFnSucceeds is the happy path the other InTx tests lean on.
func TestInTxCommitsEveryWriteWhenFnSucceeds(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)

	err := repo.InTx(t.Context(), func(tx *repository.Repository) error {
		createAccount(t, tx, "test_one")
		createAccount(t, tx, "test_two")
		return nil
	})

	if err != nil {
		t.Fatalf("in tx: %v", err)
	}
	got := accountRows(t, pool)
	if got != 2 {
		t.Errorf("rows = %d, want 2", got)
	}
}

// TestInTxRollsBackEveryWriteWhenFnFails: two writes are one act, or a failure halfway leaves
// half of it behind.
func TestInTxRollsBackEveryWriteWhenFnFails(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)

	err := repo.InTx(t.Context(), func(tx *repository.Repository) error {
		createAccount(t, tx, "test_one")
		createAccount(t, tx, "test_two")
		return errBoom
	})

	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want fn's own error back", err)
	}
	got := accountRows(t, pool)
	if got != 0 {
		t.Errorf("rows = %d, want 0 (rolled back)", got)
	}
}

// TestANestedInTxCommitsOnlyWithTheOuterTransaction: an inner InTx that succeeds must not commit
// on its own, or the outer failure would leave the inner writes behind.
func TestANestedInTxCommitsOnlyWithTheOuterTransaction(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)

	err := repo.InTx(t.Context(), func(tx *repository.Repository) error {
		createAccount(t, tx, "test_one")
		err := tx.InTx(t.Context(), func(inner *repository.Repository) error {
			createAccount(t, inner, "test_two")
			return nil
		})
		if err != nil {
			t.Fatalf("inner in tx: %v", err)
		}
		return errBoom
	})

	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want the outer fn's error", err)
	}
	got := accountRows(t, pool)
	if got != 0 {
		t.Errorf("rows = %d, want 0 (the inner commit is only a savepoint)", got)
	}
}

// TestAFailedNestedInTxUndoesOnlyItsOwnWork pins the savepoint behaviour: the outer transaction
// survives an inner failure and still decides what commits.
func TestAFailedNestedInTxUndoesOnlyItsOwnWork(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)

	err := repo.InTx(t.Context(), func(tx *repository.Repository) error {
		createAccount(t, tx, "test_one")
		err := tx.InTx(t.Context(), func(inner *repository.Repository) error {
			createAccount(t, inner, "test_two")
			return errBoom
		})
		if !errors.Is(err, errBoom) {
			t.Fatalf("inner err = %v, want errBoom", err)
		}
		return nil // the outer act carries on without the inner one, deliberately
	})

	if err != nil {
		t.Fatalf("in tx: %v", err)
	}
	got := accountRows(t, pool)
	if got != 1 {
		t.Errorf("rows = %d, want 1 (only the outer write)", got)
	}
}

// TestInTxRefusesAHandleThatCannotBegin: running fn without a transaction would give up the
// guarantee InTx was called for, with nothing to say so.
func TestInTxRefusesAHandleThatCannotBegin(t *testing.T) {
	t.Parallel()
	repo := repository.New(noBegin{})
	ran := false

	err := repo.InTx(t.Context(), func(*repository.Repository) error {
		ran = true
		return nil
	})

	if err == nil {
		t.Error("InTx succeeded on a handle with no transactions")
	}
	if ran {
		t.Error("fn ran outside any transaction")
	}
}

// noBegin is a DBTX that cannot open a transaction. InTx refuses it before calling any method,
// so the nil embedded interface is never reached.
type noBegin struct{ repository.DBTX }

func createAccount(t *testing.T, repo *repository.Repository, handle string) repository.Account {
	t.Helper()
	a, err := repo.CreateAccount(t.Context(), handle, "🍟")
	if err != nil {
		t.Fatalf("create account %s: %v", handle, err)
	}
	return a
}

func accountRows(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), `select count(*) from app.account`).Scan(&n)
	if err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	return n
}
