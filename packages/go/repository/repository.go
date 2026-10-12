// Author: Caden Lund
// Package repository is the data layer: every SQL statement in the system lives here, written by
// hand against pgx and exposed as methods on one Repository.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the subset of pgx both *pgxpool.Pool and pgx.Tx satisfy, so the same
// Repository methods run standalone or inside a transaction (via WithTx).
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository holds a database handle (a pool, or a tx) and exposes every query as a method.
type Repository struct {
	db DBTX
}

// New builds a Repository over a connection pool (or any DBTX).
func New(db DBTX) *Repository {
	return &Repository{db: db}
}

// WithTx returns a Repository whose queries run inside tx.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: tx}
}

// beginner is a handle that can open a transaction. The pool and a pgx.Tx both can; on a
// transaction, Begin opens a savepoint.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// InTx runs fn in one transaction, so a handler's writes land together or not at all. fn gets a
// Repository bound to the transaction: use it, because the outer one writes outside the
// transaction. Called inside a transaction, InTx opens a savepoint instead, so a failure in fn
// undoes only fn's work and the outer transaction still decides what commits.
func (r *Repository) InTx(ctx context.Context, fn func(*Repository) error) error {
	db, ok := r.db.(beginner)
	if !ok {
		// Running fn without a transaction would quietly give up the guarantee it was called for.
		return errors.New("repository: this handle cannot begin a transaction")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return known("begin transaction", err)
	}
	// WithoutCancel, so a client that hangs up mid-write still unwinds the transaction.
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }() // no-op once committed

	err = fn(r.WithTx(tx))
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return known("commit transaction", err)
	}
	return nil
}

// Ping proves the database answers a query. It backs the readiness probe, and it runs a real
// statement rather than pool.Ping: a pool can hold a live connection to a server that refuses
// statements.
func (r *Repository) Ping(ctx context.Context) error {
	var one int
	err := r.db.QueryRow(ctx, `select 1`).Scan(&one)
	if err != nil {
		return known("ping", err)
	}
	return nil
}
