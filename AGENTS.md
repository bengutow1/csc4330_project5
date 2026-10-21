# Fries in the Bag

An Expo app backed by a Go API. Snap your fridge and the app finds a meal you can cook right now.
If there isn't one, it tells you to put the fries in the bag and finds food nearby. Cooking earns
points on a campus leaderboard, and Combo mode lets two friends merge their fridges to cook
together. `apps/` holds deployables, `packages/` holds shared code.

Status: device accounts and scans, and nothing to cook yet. The API serves the health probes,
`AccountService` (make a device account, check a handle, who am I, save the kitchen) and
`ScanService` (judge a kitchen's photos with the vision mock, read a verdict, fix its list), with
sessions in Postgres. The Go packages hold what every feature builds on (the error boundary,
validation, the gate, the limiter, the session, the repository, logging), and the schema holds the
`app` schema, its shared trigger, `app.account`, the sessions table and `app.scan` with its items:
copy their shape. `packages/ts/api` holds the transport, the query client, the foreground refetch,
error decoding and the health, account and scan hooks.
`apps/mobile` is the shell: a first launch goes through onboarding (welcome, handle and avatar,
kitchen), then three tabs and a placeholder for every other Core route, with `/status` wired to
the API end to end. Each issue builds its screen from the design in `docs/design.md`.

## Words we use

Code, copy, proto and tests use these names, so a feature reads the same in Go, on the wire and in
the app.

| Word | Meaning |
| --- | --- |
| Scan | Up to three photos (fridge, freezer, pantry) that the vision model reads into an ingredient list. |
| Verdict | What a scan returns: a score from 0 to 100, its tier, a roast and the ingredients spotted. |
| Tier | Certified Chef (85 to 100), Home Cook (60 to 84), Scrappy (35 to 59) or Fries in the Bag (0 to 34). It decides which button is big. |
| Staples | Salt, oil, rice and the like, set once in the profile and never scanned. |
| Cook mode | Recipes ranked by what you have, one recipe walked through, then plated. |
| Plate | A photo of the finished dish. The server checks it, and a checked plate earns points. |
| Points | Earned by cooking, with bonuses for cooking out of a bad fridge. Never earned by owning food. |
| Fries mode | The fries moment, then spots nearby. A fry run logged there earns no points. |
| Spot | A place to eat nearby, from the places provider. |
| Combo | Two players merge fridges with a code, get one combined verdict and cook together. |
| Board | Chefs (points this week or all time), the Hall of Fries (fry runs) and the Fridge of the Week. |
| Rank, badge | A title earned with all-time points; a sticker for one feat. |
| Core, Next, Stretch | When a feature ships: Core is the demo path, Next is Combo and sharing, Stretch is after. |

## Commands

Run everything through moon, never raw `go` / `pnpm` / `expo`. The tasks carry the env, the task
deps and the caching; a bare `go test` skips all three.

| What | Command |
| --- | --- |
| Install the pinned toolchain | `proto install` |
| Install hooks | `moon sync hooks` |
| Fast check | `moon run :build :lint :typecheck :test` |
| Full check (what CI gates on) | `moon ci` |
| Run the API on :8080, restarting on every Go save (starts Postgres and migrates first) | `moon run api:dev` |
| Build the app into the iOS simulator (first run, or after a native change) | `moon run mobile:ios` |
| The same for the Android emulator | `moon run mobile:android` |
| Run the app once it is installed (Metro only, reloads on save) | `moon run mobile:dev` |
| Start or stop local Postgres | `moon run docker:up`, `moon run docker:down` |
| Apply migrations | `moon run db:migrate` |
| New migration | `moon run db:create -- <name> sql` |
| Rebuild the local database after editing a migration (wipes it) | `moon run db:fresh` |
| Regenerate the wire types after a `.proto` change | `moon run proto:generate` |
| Build the API image | `moon run api:image` |

The Go tests start their own Postgres through testcontainers, so Docker must be running. Go
projects have no `typecheck` task: `build` is the Go equivalent. How to name and wire a new task is
in `.claude/rules/tooling.md`.

## Rules by area

The detailed conventions live in `.claude/rules/`, one file per area. Claude Code and VS Code
Copilot load the matching file by themselves when you touch its paths. **Any other agent: read
the matching file before you edit those paths.**

| Before you touch | Read |
| --- | --- |
| Any `.go` file | `.claude/rules/go.md` |
| `packages/go/handlers/**`, `apps/api/**` | `.claude/rules/go-handlers.md` |
| `packages/go/repository/**`, `db/**` | `.claude/rules/go-repository.md` |
| `packages/go/auth/**`, `packages/go/policy/**` | `.claude/rules/go-auth.md` |
| `packages/go/vision/**`, `packages/go/places/**`, anything that calls them | `.claude/rules/go-vendors.md` |
| `proto/**` | `.claude/rules/proto.md` |
| Any test or Maestro flow | `.claude/rules/testing.md` |
| `packages/ts/**` | `.claude/rules/ts-api.md` |
| `apps/mobile/**` | `.claude/rules/expo.md` |
| `moon.yml`, `.prototools`, `go.mod`, `package.json`, CI, Docker | `.claude/rules/tooling.md` |

## How the system is shaped

- **Thin apps, logic in packages.** `apps/api/main.go` only wires: it reads the environment,
  builds the logger and the pool, constructs the packages, mounts the handlers and drains on
  shutdown. Anything that decides something lives under `packages/`.
- **Every Go package is its own module** at `packages/go/<name>`, listed in `go.work`.
- **Two Go layers, no service layer.** A Connect handler is the service: it validates, gates the
  caller, calls the repository (in one transaction when it writes twice) and maps the response.
  All SQL lives in `packages/go/repository`. Pure logic with no I/O (scoring a fridge, the points
  rules, checking a recipe against the list) lives in a small domain package that the handler
  calls.
- A new product surface is a subpackage of `handlers` (`handlers/scan`, `handlers/combo`). A new
  Go module is only for a separable capability: a vendor client (`packages/go/vision`), a pure
  algorithm (`packages/go/scoring`).
- **Outside services sit behind the API.** The vision model and the places provider are called
  only from Go, each through its own client module, with keys in the server's environment. The
  app talks to our API and nothing else.
- **The contract is proto.** `proto/<domain>/v1/*.proto` generates Go into `packages/go/apigen`
  and TypeScript into `packages/ts/apigen`, both committed. Generated types are DTOs at the wire
  and never become the domain model.
- **`packages/ts/api` is the client's one data layer**: transport, query client, hooks and error
  mapping. It is the only package that imports `apigen`, and it never imports `react-native`,
  `react-dom` or a DOM global. The app imports hooks from it and renders.
- **No live push.** Screens refetch when they open, when the app returns to the foreground and
  after their own writes. The only polling is the Combo lobby and reveal, which ask for the combo
  every 2 s while open and stop once it is revealed or ended.

## Invariants

- **The server is the only authority.** The score, the tier, points, streaks, ranks, badges and
  whether a plate counts are decided by the API. The phone sends photos and choices, never a
  number it wants recorded.
- **The model reads, our code judges.** The vision model lists what it sees and writes the roast.
  `packages/go/scoring` turns the list into the score, so the same list always scores the same,
  and fixing the list changes the recipes, never the score. A recipe the model writes is checked
  against the list before anyone sees it.
- **Identity comes from the session, never the request.** Accounts are device accounts: the first
  launch picks a handle, and the session cookie is the only credential (no email, no password). A
  request body never names who is asking.
- **The owner fence is in the SQL.** Every query on a player's data takes their account id, and
  every query on a combo's data takes the combo id, in the `where` clause. Another player's scan
  answers `not_found`.
- **Filtering and sorting happen on the server.** If a list can be narrowed, the request carries
  the filter and the server applies it: the SQL for our data, the provider query for spots.
  Filtering a fetched collection on the client narrows only the loaded page, and it keeps the
  fetch unbounded.
- **One error boundary.** `handlers.ConnectErr` is the only place an error becomes a wire code.
- **Two outside services, no more.** A vision model and a places provider, called only from the
  API. Adding a third is a decision to ask about, not a routine change.
- **Roasts target the fridge, never the person.** No jokes about bodies, money or identity, at any
  roast level.
- **Scan photos are deleted once the verdict is stored.** Plate photos are kept, because they fill
  the profile.
- **New apps under `apps/`, shared code under `packages/<lang>/`.** Pin every tool in
  `.prototools`; never install one globally.
- **No committed `.env`.** `.env.example` documents which variables exist. Vendor keys live only
  in the server's environment, never in a `moon.yml` and never in an `EXPO_PUBLIC_` variable,
  which ships inside the app.
- **Proto changes and their regenerated output go in the SAME commit.** CI consumes committed
  codegen and never runs `buf generate`.

## Working here

- **Scope:** stay inside what was asked and make routine decisions yourself. Ask only when the
  request could reasonably mean two different pieces of work. If you see a better approach, say
  so in a sentence and still do what was asked.
- **Done means** the fast check passes and the behaviour is shown working with real output: a
  command transcript, a Maestro run, a simulator screenshot. Not a claim that it works.
- **Bugs:** reproduce with a failing test, state the root cause in two or three sentences, then
  fix the cause where it lives, not where it surfaced. Check whether the same mistake exists
  elsewhere. Never stack defensive guards around a symptom.
- **No new dependency without asking.** Name the package, what it replaces, and why the repo's
  existing tools do not cover it.
- **Frontend tooling churns.** Before installing or upgrading anything in the Expo app, look up
  the current official install command for the pinned SDK. Scaffold fresh rather than patching a
  half-broken install, and stop after two failed attempts.
- **Stuck after two real attempts:** stop and report what you tried.

## Commits and pull requests

- **Conventional Commits**, enforced by `cog verify` (commit-msg hook) and `cog check` (CI).
  Humans author with `czg`; when committing programmatically write the message directly, because
  `czg` expects a TTY.
- The subject is `type(scope): ` and then a lowercase, present-tense sentence about behaviour,
  with no trailing period: `fix(api): a full combo refuses the next join`. The scope is one
  product area or moon project, never a file path.
- The body is prose wrapped near 72 columns: what was wrong or missing, what changed, and why.
  Name any alternative you rejected and the reason, and say how you verified the change.
- One change per pull request, branched off `main`, never stacked on another open PR. CI must be
  green and every CodeRabbit finding fixed or answered before merge.

## House style (every language)

- **No em dashes** in anything written here: UI copy, error messages, comments, docs, commit
  messages. Recast the sentence with a period, colon, comma or parentheses. The one exception is
  a lone `"—"` used as the empty-value glyph in a table cell.
- **Comments are short and say why**, not what. One-line numbered step comments inside a longer
  function are welcome (`// 1. Load the scan`). No multi-paragraph essays. A comment that no
  longer matches the code is a bug: fix it in the same change.
- **Build exactly what was asked.** No speculative features, no controls with no present use. If
  an addition looks valuable, offer it in one line instead of building it.
- **Errors never leak internals.** A caller sees a sentence written for them; ids, SQL and driver
  detail stay in the server log.
