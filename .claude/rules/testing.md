---
paths:
  - "**/*_test.go"
  - "**/*.test.ts"
  - "**/*.test.tsx"
  - "apps/mobile/.maestro/**"
---

# Tests

Test heavily, and against the real thing. Every package with logic has tests, every RPC has its
success path and every error code covered, and every bug fix lands with the test that reproduced
it. As a yardstick from the projects this repo copies: a pure-logic package carries about as much
test code as source, and the handler tree at least 0.6x through integration tests.

## Go: real Postgres, never mocks

- **stdlib `testing` only.** No testify, no gomock, no mockery, no sqlmock, and no mocks of our own
  layers. A mock of the repository asserts only that we call what we call; it catches none of what
  breaks (a constraint that rejects a legal value, a unique index that does not fire).
- **testcontainers + pgtestdb.** Each DB-backed package's `main_test.go` starts ONE Postgres
  container in `TestMain`. Every test calls `newRepo(t)` and gets its own database, cloned from a
  template that pgtestdb migrates with the real goose stream and rebuilds whenever a migration
  changes. The image tag is the one in `docker/compose.yml`.

```go
const postgresImage = "postgres:18-alpine" // same tag as docker/compose.yml

var base pgtestdb.Config

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase("postgres"), tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"), tcpostgres.BasicWaitStrategies())
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres: %v\n", err)
		os.Exit(1)
	}
	// … host and mapped port, each checked the same way …
	base = pgtestdb.Config{DriverName: "pgx", Host: host, Port: port.Port(), User: "postgres",
		Password: "postgres", Database: "postgres", Options: "sslmode=disable"}
	code := m.Run()
	// Not deferred: os.Exit below would skip it.
	err = testcontainers.TerminateContainer(container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminate postgres: %v\n", err)
	}
	os.Exit(code)
}

func newRepo(t *testing.T) *repository.Repository {
	t.Helper()
	// io/fs paths may not contain "..", so the walk up lives in os.DirFS's argument.
	migrator := goosemigrator.New("migrations", goosemigrator.WithFS(os.DirFS("../../../db")))
	conf := pgtestdb.Custom(t, base, migrator)
	pool, err := pgxpool.New(t.Context(), conf.URL())
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return repository.New(pool)
}
```

- Tests that need Docker fail fast without it. Start Docker; never add a skip.
- Pure logic gets no database: pull branchy code (the score and its tier boundaries, the points
  rules, meal windows and streaks, recipe checks) into pure functions and table-test them hard,
  every boundary included (34 and 35, midnight, Monday). Do not add interfaces or seams just to
  mock something.

## Handler tests drive the serving stack

- Mount the real handler on an `httptest.Server` over a real database and real scs sessions, and
  call it through the GENERATED Connect client. Calling the Go method directly proves the happy
  path and none of what this layer exists for.
- Auth is real. One `http.Client` with a cookie jar is one person; give them a device account
  through the real RPC. Persona helpers are named for who they are (`newcomer`, `player`, `host`,
  `guest`), so "did I remember the gate" cannot be faked. Tests set
  `sessions.Cookie.Secure = false` because httptest serves plain http.
- Rate limits are off in fixtures, and one dedicated test covers them.
- Assert the wire code, and say what came back:
  `t.Fatalf("got code %v (%v), want not_found", connect.CodeOf(err), err)`.
- A validation failure asserts `InvalidArgument` AND the violation, keyed by the proto field name:

```go
func fieldViolations(t *testing.T, err error) map[string]string {
	t.Helper()
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("not a connect error: %v", err)
	}
	out := map[string]string{}
	for _, d := range cerr.Details() {
		msg, derr := d.Value()
		if derr != nil {
			t.Fatalf("decode error detail: %v", derr)
		}
		br, ok := msg.(*errorv1.BadRequest)
		if !ok {
			continue
		}
		for _, v := range br.FieldViolations {
			out[v.Field] = v.Description
		}
	}
	return out
}
```

- Integration tests live in `packages/go/handlers/*_test.go` as `package handlers_test`.

## What each change's tests cover

- **Every RPC**: the success path, asserting the response body and not merely the absence of an
  error, plus every error code the handler can return.
- **Every sentinel's mapping** in `ConnectErr`, so a sentinel that loses its arm fails a test. And
  the reverse: an unknown error comes back `internal`, with the text `internal error` and no
  violations.
- **The owner fence**: two players, and a request from one against the other's ids (a scan, a
  plate, a combo they are not in) answers `not_found`; then re-read the victim and prove nothing
  of theirs moved. A refusal that still wrote passes every other assertion. For a sweep, keep a
  map of closures with one entry per id-bearing RPC.
- **What only Postgres can do**: check and unique constraints, cascades, defaults, ordering, the
  folding of driver errors into sentinels, and `InTx` rolling back every write.
- **Races**: start 20 goroutines (`wg.Go`) at a guarded write and assert exactly N succeed (the
  last seat in a combo, two plates claiming one meal window).
- **Pagination**: walk the cursor to the end with a runaway guard (`if pages > 5 { t.Fatal(…) }`),
  force ties on the sort key, then assert no gaps, no duplicates, the order, and an empty page
  past the end.
- **"Every…" tests** for anything that must stay complete: every id-bearing RPC is fenced, every
  tier has its buttons, two hand-kept lists agree.

## Fakes, only at a true edge

- Hand-written, in the test file that first needs them (shared ones in `main_test.go`),
  mutex-guarded, and recording their calls so a test can assert what was sent.
- They satisfy a narrow interface declared at the consumer. The vision model and the places
  provider are the edges this repo has: handler tests take a fake that returns a fixed ingredient
  list, and the clients themselves are tested against an `httptest.Server` through an injected
  base URL (`go-vendors.md`).
- Comment why the fake is safe. "A fake of our own repository" is never safe.

## Time

- Code under test never calls `time.Now()` directly. A pure function takes `now time.Time`; a
  stateful component holds `now func() time.Time`, defaulted to `time.Now`. Tests pass a fixed
  `time.Date(...)`. Meal windows, streaks, the Monday reset and code expiry are exactly where a
  wall-clock test goes green at noon and red at midnight.
- Database-backed tests never compare against this machine's `now`: the container's clock can run
  a hair behind. Stamp "a minute ago" instead.
- Wait for async work by bounded polling (`for range 200 { …; time.Sleep(10 * time.Millisecond) }`),
  never a bare sleep.

## Style

- `t.Parallel()` is the first line of every test, and of every subtest whose rows are independent.
- External test packages (`package repository_test`) by default. A test of an unexported pure
  helper lives in the package itself, in `<name>_internal_test.go` when both kinds share a folder.
- Names are sentences stating the claim: `TestAFullComboRefusesTheNextJoin`,
  `TestFixingTheIngredientsNeverChangesTheScore`. The comment above each test says which failure
  it prevents.
- `t.Context()` for contexts. `t.Cleanup` for every resource. Helpers call `t.Helper()` first and
  take `t` first: `newCombo(t, repo)`, `aGuestIn(t, client, code)`.
- `t.Fatalf` for setup and preconditions (`t.Fatalf("create combo: %v", err)`), `t.Errorf` for the
  independent assertions, so every failure is reported. Messages are got first, then want, with
  the reason when it helps: `t.Errorf("code = %v, want failed_precondition (combo ended)", got)`.
- Tables are `cases := []struct{ name string; … }{ … }` ranged with `for _, c := range cases`.
- Compare with `!=` and `slices.Equal`. No assertion libraries.
- Test data uses reserved names: `test_` handles, `.test.invalid` hosts, and ingredient lists
  written out in the test, never a real photo.
- Live or expensive suites sit behind a build tag (`//go:build e2e`) and a moon task not named
  `test`.

## TypeScript

- vitest, colocated `*.test.ts`, on pure modules: error decoding, formatters, which button a tier
  makes big. No component rendering, no Testing Library, no `vi.mock`.
- Build test inputs exactly as production produces them (a real connect-query key, a wire-shaped
  `BadRequest`). A fabricated shape passes while the real one matches nothing.

## Maestro (the app)

- **Every screen gets a Maestro flow** in `apps/mobile/.maestro/flows/`, and it exercises the
  behaviour the change added (tap Judge my fridge, see the verdict), not just "the screen is
  visible". Start each flow with `clearState: true`, tap by accessibility label, assert on visible
  text, end on an `assertVisible`.
- One-off flows to prove a behaviour are fine; delete them before merging.
- Verify on the iOS simulator. Screenshots come from the simulator, never from the web build,
  which renders different layouts.

## Rules

- Write the test first and watch it fail. A test that has never failed has not been shown to test
  anything; when you fix a bug, revert the fix once and watch the new test fail.
- **Never edit a test to make it pass**, and never special-case a test input in production code.
  If the test or the plan is wrong, stop and say so.
- A bug fix lands with the failing test that reproduced it, and the test stays.
- A test must fail for the reason it claims. A collector that returns empty on ANY error passes
  for the opposite reason.
- Expected values are written by hand. No `-update` flag that regenerates them from the code under
  test: it would enshrine today's bugs.
