---
paths:
  - "packages/go/repository/**/*.go"
  - "db/**/*.sql"
---

# The SQL layer

All SQL lives in `packages/go/repository`, one module that the api, workers and scripts all
import. The package has sqlc's boilerplate shape (`DBTX`, `New`, `WithTx`) but every query is
written by hand with pgx. No sqlc generation, no ORM, no query builder: the queries are the thing
that breaks, and a generated layer hides the predicate that matters.

## The boilerplate

```go
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
```

- **`InTx(ctx, fn func(*Repository) error)`** is how a handler makes two writes one act. It begins,
  defers `_ = tx.Rollback(context.WithoutCancel(ctx)) // no-op once committed`, runs `fn` on the
  tx-bound repository and commits. Called on a repository that is already inside a transaction,
  it opens a savepoint instead (pgx's nested `Begin`): a failure in `fn` undoes only `fn`'s work,
  and the outer transaction still decides what commits. Inside `fn`, use the `*Repository` you
  were handed: the outer one silently writes outside the transaction.
- Handlers never see `*pgxpool.Pool` or `pgx.Tx`. A method that takes a pool cannot join a
  transaction.
- One file per concern (`account.go`, `scan.go`, `combo.go`), each opening with a short comment
  on the tables it owns and their invariants.
- `Ping` is a real query (`select 1`), not `pool.Ping`: a pool can hold a live connection to a
  server that is rejecting statements.

## Signatures

- **`(ctx, <owner id>, <ids>, <args>)`.** Every query on a player's rows takes their account id,
  and every query on a combo's rows takes the combo id, as an explicit argument that is `$1` in
  the `where` clause. A query that can read or change someone else's rows is a bug, not a style
  nit. The fence is in the SQL, never in a handler's memory.
- Names: `ScanByID`, `ComboByCode`, `ListScans`, `CreateScan`, `UpdateKitchen`, `DeleteScanPhotos`.
- **Create** returns the created row (`returning` the columns) or, for a bare insert, its id.
- **Update and Delete return `(bool, error)`**: true means a row changed. The handler turns
  `false` into `not_found`; the repository does not guess what a miss means.
- More than three same-typed parameters become a struct (`KitchenEdit`). Five positional strings
  is a call site where two can be swapped and nothing will say so.
- Row types are plain structs with a comment on every field whose meaning is not obvious. A value
  the database leaves null reads as a documented zero:
  `EndedAt *time.Time // nil = the combo is still open`, or `""` stored with `nullif($n, '')` and
  read back with `coalesce(col, '')`.
- A closed set of values (a photo's kind, a combo's state) is a block of exported consts
  mirroring the check constraint, so callers never retype the literals.

## Errors

Driver internals never leave this package. `known(op, err)` folds them into this package's
sentinels, and the handler switch maps those:

```go
func known(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s (%s): %w", op, pgErr.ConstraintName, ErrDuplicate)
	}
	return fmt.Errorf("%s: %w", op, err)
}
```

- Every error return goes through `known` with an op that carries the ids:
  `known(fmt.Sprintf("scan by id account=%d scan=%d", accountID, scanID), err)`.
- The SQLSTATE strings live in that one helper, nowhere else.
- `errors.go` holds one `var ( … )` block of sentinels, each with a doc comment saying when it
  fires. Generic ones (`ErrNotFound`, `ErrDuplicate`) plus specific refusals whose text names the
  fix (`ErrComboFull = errors.New("that combo is full")`).
- Where no row is a valid answer rather than a missing thing, use the unexported `isNoRows(err)`
  so the caller does not import pgx.

## Queries

- Lowercase SQL in a raw string that starts on a new line, tab indented, keywords right-aligned
  into a river, short table aliases, `$n` params:

```go
err := r.db.QueryRow(ctx, `
	select s.scan_id, s.score, s.roast, s.created_at
	  from app.scan s
	 where s.account_id = $1
	   and s.scan_id = $2`, accountID, scanID).Scan(&s.ScanID, &s.Score, &s.Roast, &s.CreatedAt)
```

- **Columns are written inline in every query.** No shared `scanColumns` constant and no scan
  helper shared across queries: each query reads on its own, and adding a column touching every
  query is the accepted cost.
- Select only what the caller uses. Filter and sort in SQL, not in Go: rows fetched and discarded
  were still read, and a client cannot filter what it never received.
- No N+1: batch with `= any($1)`, `unnest` or a lateral join instead of a query in a loop.
- Predicates hit an index. Watch for `like '%x'`, a function on the indexed column, and an `OR`
  that defeats the index.
- A list that can grow pages with a keyset cursor over one ordered index: a row-value comparison
  (`(created_at, plate_id) < ($2, $3)`) matching the index's direction, fetching `limit+1` to
  know whether there is more. Never `OFFSET`, and never OR-expand the row comparison.
- Literal search is `strpos(lower(col), lower($2)) > 0`. If it must be `ilike`, escape `%` and `_`
  in the input first, so a typed `%` is not a wildcard.
- Row loops: `defer rows.Close()`, `for rows.Next()`, then check `rows.Err()`.
- **Multi-statement invariants run in one transaction** and take the right lock (`for update`, an
  advisory lock) when a check-then-write can race: two friends taking a combo's last seat, two
  plates claiming the same meal window. Better still is one guarded statement
  (`update … where state = 'open'`, `insert … where (select count(*) …) < $3`).
- Writes that change something a client reads call `r.notify(ctx, <family>Poke(comboID))` beside
  the statement. See `live-updates.md`.

## Schema and migrations

- One linear goose stream in `db/migrations`, `NNNNN_<concern>.sql`, one file per concern, each
  with `-- +goose Up` and a matching `-- +goose Down`. Create a new one with `moon run db:create`.
- Until there is a real deployment, edit the existing migration in place rather than stacking a
  new one. Goose tracks versions, not content, so after editing an applied migration say so and
  rebuild the local database. Tests are unaffected: every run migrates a fresh template.
- Tables live in the `app` schema, named in the singular (`app.account`, `app.scan`). The key is
  `<table>_id bigint generated always as identity primary key`.
- Lowercase DDL with aligned columns. `text` plus `check (kind in ('fridge', 'freezer',
  'pantry'))`, never a Postgres enum type. Check constraints carry shape too (lengths, regexes,
  ranges, columns that are null together).
- `created_at timestamptz not null default now()` on every table. Mutable tables add `updated_at`
  and the shared `set_updated_at` trigger from the first migration.
- Every foreign key states its `on delete` explicitly. No soft-delete columns: a reversible state
  is a status value, and a delete is real.
- Indexes are named `<table>_<cols>_idx`, unique ones `<table>_<what>_key`.
- A check constraint and the validator tag that restates it stay in step by hand. Say so in a
  comment on both.
- One fact lives in one column. Derive on read rather than storing a second copy that can drift:
  a player's points are a sum over their point events, one row per award with its reason, and a
  scan's tier comes from its score, so neither is a column.
