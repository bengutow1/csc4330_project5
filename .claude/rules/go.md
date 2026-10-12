---
paths:
  - "**/*.go"
---

# Go style

## File header

Every new `.go` file (tests included) starts with an author line naming whoever wrote it (the
name in `git config user.name`). The package's primary file also carries the package doc comment;
every other file, and every `_test.go`, has the author line only.

```go
// Author: Caden Lund
// Package scoring turns a fridge's ingredient list into its score and tier.
package scoring
```

Never add Created, Updated or Notes lines, and never rewrite another author's header.

## Error checks: always two statements

```go
combo, err := h.repo.ComboByCode(ctx, code)
if err != nil {
	return nil, h.fail(op, err)
}

err = json.Unmarshal(body, &msg)
if err != nil {
	return err
}
```

Never the inline scoped form `if err := f(); err != nil`, not even for a single-use call, not
even for `handlers.Validate`. The variable is always `err`; do not invent `e`, `derr` or `verr`
unless shadowing genuinely forces it. When you edit a function that still has the inline form,
convert it.

## Wrapping and comparing

- Wrap with `%w` and a lowercase op phrase that carries the ids for the log:
  `fmt.Errorf("add combo member combo=%d: %w", comboID, err)`. No `failed to` or `error:` prefix. A
  package at the edge of the system prefixes its own name (`fmt.Errorf("storage: put object: %w",
  err)`).
- Compare with `errors.Is` / `errors.As`, never `==` or a type assertion (errorlint enforces it).
- Packages export sentinels for the conditions callers may branch on. Everything else stays a
  wrapped error. See `go-handlers.md` for where they become wire codes.

## Never drop an error silently

- A deliberate discard is `_ =` with a comment saying why it is safe:
  `defer func() { _ = tx.Rollback(ctx) }() // no-op once committed`.
- An error you swallow on purpose (best-effort work after a commit, cleanup) is still logged:
  `Error` for a real failure, `Warn` for best-effort cleanup. Returning nil and dropping the cause
  makes a real outage look like success.
- Every `//nolint:<linter>` carries a reason: `//nolint:gosec // bounded by maxPhotos`.

## Switch statements

- A tagless `switch {}` beats an `if / else if` chain for mutually exclusive branches.
- A switch over an enum names every value. The `default` arm is for the value nobody expected:
  it logs or returns an error, it never silently picks a behaviour. A default that quietly
  absorbs a newly added value is a bug waiting for that value.
- Group cases that share a body (`case TierChef, TierHomeCook:`, which both lead with cooking) and
  say why they are grouped when it is not obvious.
- The error switch is its own pattern: one per API, in `handlers.ConnectErr` (see
  `go-handlers.md`).

## Logging

- `log/slog` only, structured. `packages/go/logging` builds it (`logging.New("api")`): JSON by
  default, text with `LOG_FORMAT=text`, debug with `LOG_LEVEL=debug`, every line stamped with the
  service and version. `main` sets it as the default and injects it; packages never call
  `slog.Default()` or the global `slog.Info`.
- Messages are short lowercase phrases, usually the `op`. Keys are snake_case and the error key is
  always `"err"`: `h.log.Warn("delete scan photos", "scan_id", scanID, "err", err)`.
- Client hang-ups and deliberate refusals are not ERROR lines.
- Never log secrets, session tokens, vendor keys, combo codes, photos or a player's location. Log
  the id instead (`"account_id"`). The request logger records the path, never the query string.
- A client for something outside the repo takes no logger. It returns errors and the caller
  decides what to log.

## Constructors

- `NewX(...) *X` returns the concrete pointer; a package's primary type uses `New`.
- More than two dependencies arrive as a `Config` struct (`NewScan(cfg Config) *Scan`), and a nil
  `Log` in it means discard (`slog.New(slog.DiscardHandler)`). Otherwise positional, with
  `*slog.Logger` last. Either way the field is stored as `log`.
- An optional collaborator is a small interface declared at the consumer, so a test can pass a
  fake and `main` can pass nothing. Never pass a typed-nil pointer into an interface parameter.

## Context and config

- `ctx context.Context` is always the first parameter and never stored in a struct. Work that
  must outlive the request uses `context.WithoutCancel(ctx)` with its own timeout.
- Environment variables are read only in `main.go` with `os.Getenv` and passed into
  constructors as plain values or a `Config` struct. A missing required one is `log.Error(...)`
  then `os.Exit(1)`. A missing optional one logs a warning and the feature reports itself
  unavailable. The one exception is `logging.New`, which reads `LOG_FORMAT`, `LOG_LEVEL` and
  `APP_VERSION` itself so every binary logs the same way.
- Defaults fall to the safe side: anything other than an explicit local-dev `APP_ENV` behaves
  like production. A dev-only toggle refuses to boot in production.
- Every outbound HTTP client has a timeout, a named const sized to that vendor
  (`placesTimeout = 6 * time.Second`): a hung remote must not hold a request open.

## State, goroutines and time

- No `init()`. No mutable package-level state: package vars are sentinels, regexps, read-only
  tables, or values computed once (`sync.OnceValue`). A mutex is a struct field.
- Panic only for an impossible condition at startup (crypto/rand failing). Never in a request path.
- Never start a bare `go func()` from a handler. Work detached from a request goes through a
  bounded runner that drops work when full. A background loop is started from `main`, and when
  several replicas run it, each claim is one SQL statement only one replica can win.
- Logic never calls `time.Now()` directly. A pure function takes `now time.Time`; a stateful
  component holds `now func() time.Time` defaulted to `time.Now`. See `testing.md`.

## Naming and layout

- Initialisms stay upper case: `ScanID`, `BaseURL`. Generated proto fields keep theirs
  (`ScanId`); the mapper converts.
- Receivers are one letter by role: `h` handlers, `r` the repository, `a` auth, `c` a client,
  `g` a gate, `s` a store or sender.
- No `Service`, `Svc`, `Impl` or `Manager` suffixes, and no stutter: the package name already
  says what it is (`scoring.Points`, not `scoring.ScoringPoints`).
- Two named functions beat one function with a bool mode flag.
- Named results only where they document meaning (`(changed bool, err error)`).
- gofmt formats. Imports come in three groups: stdlib, third-party, this module.
- Every Go package is its own module listed in `go.work`. After changing a module's imports,
  build it standalone with `GOWORK=off`: the workspace build hides a missing `require`.
