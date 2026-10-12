---
paths:
  - "packages/go/auth/**/*.go"
  - "packages/go/policy/**/*.go"
  - "packages/go/handlers/account/**/*.go"
---

# Auth and identity

The server decides who a request belongs to, from the session, every time. A request body never
names its own identity.

Accounts are device accounts. The first launch picks a handle and an avatar, the server creates the
account and signs the session in, and from then on the session cookie is the only credential.
There is no email and no password, so clearing the app's data loses the account; that is accepted
for a class project. Never add a second credential to work around it (a token in AsyncStorage, a
device id the client sends): ask first.

`packages/go/auth` holds the session and the rules around it. Handlers call it; it owns the rules,
they own the wire. It is not a service layer and holds no flow beyond auth itself.

## Sessions

- `alexedwards/scs/v2` with `pgxstore`: the session row lives in Postgres and the client holds an
  opaque id in an http-only cookie. No JWTs.
- Losing the session loses the account, so it lives long: `sessions.Lifetime` is a year, set in
  `main.go` with the other cookie settings.
- **`RenewToken` first, whenever privilege changes.** Creating the account is the sign-in, so it
  renews before it stores the account id. Skipping it is session fixation.
- Session keys are unexported consts in `auth`. Nothing else knows the session's shape; it is read
  through `CurrentAccount` only.
- Store Unix seconds (`time.Now().Unix()`), not `time.Time`: the session is a gob blob.
- `CurrentAccount` re-reads the row on every request, so a deleted account lands on the next call.
  A session naming a row that no longer exists is `ErrNotAuthenticated`, not an internal error.
- Inside a server stream the session is READ-ONLY: scs commits the cookie at the first byte.
- Cookie flags (Secure, SameSite, domain) are set in `main.go`, where the environment is known.
- "Delete my data" removes the account's rows and destroys its sessions through
  `sessions.Iterate`, never by reaching into the sessions table.

## Handles

- A handle is the player's public name on the board: 3 to 20 lowercase letters, digits and
  underscores (`chef_maya`). The validator tag and the table's check constraint say the same
  thing, with a comment on both.
- Trim and lowercase before lookup AND insert. Uniqueness is the unique index, never a
  check-then-insert: a taken handle comes back as `repository.ErrDuplicate`, and the handler turns
  it into a field error on `handle` ("that handle is taken") so onboarding shows it under the input.

## Gates, not middleware

- No Connect interceptors and no `context.WithValue` for identity. Every RPC opens with an explicit
  gate that resolves the caller from the session (`h.gate.Account(ctx)`,
  `h.gate.ComboMember(ctx, comboID)`, `h.gate.ComboHost(ctx, comboID)`). It is a call in each RPC
  on purpose: an RPC that forgets its gate is visible by the absence.
- Membership and the host are re-read per request, then a pure `policy` function decides. Fail
  closed: no session is `ErrNotAuthenticated`, the wrong standing is `ErrForbidden`.
- The repository stays authorization-free. It scopes every query by the account or combo id it is
  handed; the handler decides who may ask.

## Combo and crew codes

- **Six digits**: `rand.Int(rand.Reader, big.NewInt(1_000_000))` formatted `%06d`, never a modulo
  of a bigger number. Unique among live codes; a collision on insert retries with a fresh code.
- A combo code dies two hours after it was made, or when the host ends the combo. The join's own
  SQL enforces it (`expires_at > now()`), never a cleanup job the join trusts to have run.
- A code is an invite, not a credential: it is stored as is so the lobby can show it again, and
  wrong, expired and full may answer differently, because the join screen shows each one.
- Guessing six digits is cheap, so joining is rate limited per client IP (`go-handlers.md`).

## Dev switches and seed data

- A dev-only switch relaxes behaviour, never data, and `main.go` refuses to boot with it set while
  `APP_ENV=production`.
- Demo data (players on the board, a combo in progress) comes from an idempotent seed script run
  by a moon task, never from a boot step or a migration.

## Sentinels

Auth sentinels are written for the person on screen and map through `handlers.ConnectErr` like
every other sentinel:

```go
var (
	// ErrNotAuthenticated: no session, or a session whose account is gone.
	ErrNotAuthenticated = errors.New("not signed in")
	// ErrForbidden: signed in, but not allowed to do this.
	ErrForbidden = errors.New("not allowed")
)
```

`ErrNotAuthenticated` maps to `Unauthenticated` (the app returns to onboarding, no toast) and
`ErrForbidden` to `PermissionDenied`.
