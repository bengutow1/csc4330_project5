---
paths:
  - "packages/go/handlers/**/*.go"
  - "apps/api/**/*.go"
---

# Handlers and the api binary

## Two layers, no service layer

`handlers → repository`. A Connect handler struct IS the service: it gates the caller, validates,
calls the repository, runs the transaction and maps the response. There is no service or
orchestration type between them.

- All SQL lives in `packages/go/repository`. Handlers never build a query and never hold a pool
  or a `pgx.Tx`.
- Pure logic with no I/O (scoring a fridge, checking a recipe against the list) lives in a small
  domain package called from the handler. It never touches the database.
- `packages/go/handlers` is one module. Its root package holds what every service shares:
  `ConnectErr`, `HTTPStatus`, `FieldErrors`, `Validate`, `limits.go` and the `Gate`. Each product
  surface is a subpackage (`handlers/scan`, `handlers/cook`, `handlers/combo`). Logic moves to the
  root the moment a second service needs it; never keep a second copy of a rule.

## The service type

```go
// Combo implements combo.v1 ComboService.
type Combo struct {
	repo *repository.Repository
	gate handlers.Gate
	log  *slog.Logger
}

// Config is what Combo needs to exist.
type Config struct {
	Repo *repository.Repository
	Auth *auth.Auth
	Log  *slog.Logger
}

// NewCombo builds the service. A nil Log discards.
func NewCombo(cfg Config) *Combo {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Combo{repo: cfg.Repo, gate: handlers.Gate{Repo: cfg.Repo, Auth: cfg.Auth}, log: log}
}

// fail is the one way an error leaves this service.
func (h *Combo) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}
```

- The type is named for the domain noun, with a doc line naming the proto service it implements.
- Fields are unexported. An optional collaborator (the vision model, the places provider) is a
  small interface declared here, at the consumer, so a test can hand in a fake.
- Files: `combo.go` (type, Config, New, fail, gate helpers, RPCs), `validate.go` for named input
  structs, `convert.go` for the `toProtoX` mappers.

## The shape of every RPC

```go
func (h *Combo) JoinCombo(
	ctx context.Context, req *connect.Request[combov1.JoinComboRequest],
) (*connect.Response[combov1.JoinComboResponse], error) {
	const op = "join combo"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate everything the request carries
	code := strings.TrimSpace(req.Msg.Code)
	invalid := handlers.Validate(struct {
		Code string `pfield:"code" validate:"len=6,numeric"`
	}{code})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write: an ended, expired or full combo is refused inside the same statement
	combo, err := h.repo.JoinCombo(ctx, me.AccountID, code)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&combov1.JoinComboResponse{Combo: toProtoCombo(combo)}), nil
}
```

Same order, every time:

1. `const op = "<verb> <noun>"` on the first line. The same `op` goes to every error return.
2. The gate: resolve the caller from the session. An RPC that is public by design (creating a
   device account, the health probes) says so in a comment instead.
3. Normalize (trim, case-fold), then validate. What you validate is what you store, and nothing
   touches the repository before the input is known good.
4. Load and check what the request names. A thing that does not exist and a thing that belongs to
   someone else (another player's scan, a combo you are not in) answer identically: `not_found`.
5. The write. More than one statement runs in
   `h.repo.InTx(ctx, func(tx *repository.Repository) error { … })`, and every read and write in
   it goes through `tx`. A write that matched nothing (`false`) returns
   `h.fail(op, repository.ErrNotFound)`. A vendor call (the vision model, places) happens before
   this step, never inside the transaction, which would hold a connection open for seconds.
6. When the response carries the row, re-read it, so the client sees what was stored.
7. After the commit, best-effort side effects. They log their failure and never return it: the
   write already happened.
8. Map with `toProtoX` from `convert.go` and return.

Every RPC signature uses the three-line form above.

## The gate

- No Connect interceptors and no `context.WithValue` for identity (see `go-auth.md`).
  `handlers.Gate` is the one implementation of the fence, and services wrap it in helpers named
  for what they mean (`h.gate.Account(ctx)`, `h.gate.ComboMember(ctx, comboID)`,
  `h.gate.ComboHost(ctx, comboID)`).
- **A request field is never authority.** A field naming an account that disagrees with the
  session is refused, not quietly replaced: a silently ignored input is how a client spends
  months doing the wrong thing with confidence.
- Check visibility before rights. A caller who cannot see the thing gets `not_found`; a caller who
  can see it but may not change it gets `permission_denied`. A refusal never confirms that
  something exists.
- Rate limits live inside the handler, with `handlers.ErrTooManyRequests` mapped to
  `resource_exhausted`. Creating a device account and anything that takes a guessable value (a
  combo or crew code) are limited per client IP, because device accounts are free to make. An RPC
  that spends vendor money (a scan, a plate check) is limited per account.

## One error boundary

`handlers.ConnectErr(log, op, err)`, reached through each service's `h.fail`, is the ONLY place
an error becomes a wire code. Never choose a Connect code at the call site: that is how the
redaction boundary springs a leak.

It is one tagless switch, arms in this order:

1. A deliberate `*connect.Error` passes through untouched (a refusal the handler built on
   purpose). Rare; prefer a sentinel.
2. `context.Canceled` → `Canceled`, with no ERROR log: the client hung up.
3. `context.DeadlineExceeded` → `DeadlineExceeded`.
4. `FieldErrors` → `InvalidArgument`, with every violation in an `error.v1.BadRequest` detail. A
   detail that fails to marshal degrades to code and message, never to a failed response.
5. The sentinels, grouped by owning package. Each arm returns the SENTINEL, never `err`: the wrap
   carries ids and driver detail that must not reach a caller.
6. `default`: `log.Error(op, "err", err)` and a generic `internal error`. The only ERROR log in
   the switch.

`errors.As` targets are declared as `var`s above the switch. Every non-obvious arm has a one-line
comment saying why that code and not its neighbour's.

Choosing a code:

- `InvalidArgument`: what they sent is wrong, and no change in the world would make it right.
- `NotFound`: the thing they named does not exist, or they cannot see it.
- `AlreadyExists`: a unique thing is taken (that handle is taken).
- `FailedPrecondition`: the request is fine, the world is not ready or has moved on (the combo
  already ended, the code expired, no vision key is configured). The message names what to do
  instead.
- `PermissionDenied`: only when somebody holds the right and this caller does not (only the host
  can end the combo or remove a player).
- `Unauthenticated`: no session, or a dead one.
- `ResourceExhausted`: a limit was hit (the combo is full, too many requests).
- `Unavailable`: an outside service failed or timed out, or the server is not ready. Trying again
  later may work.

A new known condition is three things in one change: a sentinel exported by its package, one arm
in the switch, and a test that pins the mapping. A sentinel with no arm reaches the client as a
redacted `internal error`.

Raw HTTP routes, if any, map errors through `handlers.HTTPStatus`, which lives beside
`ConnectErr` in the same file so the two cannot quietly disagree. Never a private status switch.

## Sentinels

```go
var (
	// ErrComboEnded fires when someone joins a combo the host has already ended.
	ErrComboEnded = errors.New("that combo has ended, ask for a new code")
	// ErrComboFull fires when the combo already has both of its players.
	ErrComboFull = errors.New("that combo is full")
)
```

- One `var ( … )` block in the owning package's `errors.go`, a doc comment on each.
- The message goes on screen verbatim: lowercase, no trailing period, written for the player,
  naming the fix when there is one. No em dashes.
- They are the ONLY errors callers may branch on.

## Validation

- validator/v10 through `handlers.Validate`, on an anonymous struct at the call site. `pfield`
  names the PROTO field, which is what lands in the error and what the client keys inputs by.
- It accumulates every violation so the form shows all its problems in one round trip. Never
  hand-roll checks that stop at the first failure.
- A bigger input gets a named struct in the service's `validate.go` and a
  `buildX(in) (row, FieldErrors)` that normalizes first and validates second.
- When a rule cannot be a tag (a list, a cross-field rule), accumulate into `FieldErrors` through
  a local `refuse(field, msg)` closure, with indexed paths (`items[2].name`).
- Trim before you judge. Ids use `gt=0`, which reads "is required". Lengths count runes.
- Shared bounds are named consts in `handlers/limits.go`, each with a one-line reason. A tag
  cannot name a const, so the tag restates the number; keep the two in step.
- Restate the database's check constraints in tags, so a bad value becomes a field message
  instead of a redacted internal error.
- An unknown enum or vocabulary value is refused with a message naming the valid values.
- Validation lives here, at the handler. The repository trusts shape and relies on the database
  for integrity (uniqueness, foreign keys); a check-then-insert in Go is a race.

## Endpoint hygiene

- **No unbounded list.** Anything that can grow takes a cursor and a limit and returns
  `next_cursor`, set only when there is more. The limit is clamped to a default and a maximum
  declared as consts. A set that is provably tiny and bounded (a combo's two players) may skip
  paging, with a comment saying why.
- The cursor is opaque to the client and validated on the way back in. A malformed cursor is
  refused (`InvalidArgument` on `cursor`), never treated as "start over": re-serving page one
  turns an infinite scroll into a loop. Totals ride the first page only.
- Refuse rather than answer wrong: a backwards range is refused, and a request over a cap is
  refused, not truncated.
- Filters and sorts the client controls are request fields applied in SQL (AGENTS.md).
- A mutation that matched nothing answers `not_found`, never a 200.
- A repeat call that changes nothing is a silent success with no side effects.
- Proto messages are DTOs at the wire only. Never let a generated type become the domain model.
- chi anchors a param regexp as `(^…)|(…$)`, so `{kind:a|b}` also matches `abc`. Group the
  alternation: `{kind:(a|b)}`.

## apps/api/main.go

`main.go` is wiring only: construct, mount, drain. Adding a service is three lines.

- The environment is read here and nowhere else (see `go.md`). Dev-only relaxations key off an
  explicit opt-in, `localDev := appEnv == "development" || appEnv == "local"`, so an unset or
  misspelled `APP_ENV` runs hardened.
- `log := logging.New("api")`, `slog.SetDefault(log)`, then the logger is injected everywhere.
- `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
- One pool and one `repository.New(pool)` shared by every service. Migrations never run at boot;
  `moon run db:migrate` does that.
- A missing optional dependency logs a warning and its RPCs answer `failed_precondition`. A
  missing required one is `log.Error(...)` then `os.Exit(1)`. The vendor keys are their own case:
  without one the API serves that vendor's mock, and with `APP_ENV=production` it refuses to boot
  (`go-vendors.md`). Never pass a typed-nil pointer into an interface parameter.
- Middleware: `logging.RequestLogger(log)` outermost, then `logging.Recoverer(log)` so a panic is
  logged as the 500. `sessions.LoadAndSave` wraps the whole router.
- Every Connect service is mounted with the same read cap:
  `r.Mount(scanv1connect.NewScanServiceHandler(scanSvc, readCap))`, with
  `readCap := connect.WithReadMaxBytes(4 << 20)`.
- The server sets `ReadHeaderTimeout` 10 s, `ReadTimeout` 60 s (three photos on a slow phone
  connection), `WriteTimeout` 90 s (longer than any vendor deadline) and `IdleTimeout` 2 minutes.
  There are no long-lived streams: a request that needs more time is a design problem, not a
  timeout to raise.
- Health is `healthz.NewHealth(log, repo.Ping)`.

The drain, in this shape:

```go
errc := make(chan error, 1)
go func() { errc <- srv.ListenAndServe() }()
select {
case err := <-errc:
	log.Error("server stopped", "err", err)
	os.Exit(1)
case <-ctx.Done():
}
stop() // a second SIGTERM force-quits
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := srv.Shutdown(shutdownCtx)
if err != nil {
	_ = srv.Close() // the drain timed out; drop what is left
}
```
