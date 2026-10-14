---
paths:
  - "proto/**/*.proto"
  - "proto/**/*.yaml"
  - "packages/go/apigen/**"
  - "packages/ts/apigen/**"
---

# The API contract

- **Proto messages are DTOs at the wire only.** Repository and domain types stay hand-written Go;
  the handler maps between them. Never let a generated type become the domain model, or every
  schema change becomes a wire change.
- One file per domain at `proto/<domain>/v1/<domain>.proto`. Packages are `<domain>.v1`, services
  `<Domain>Service`, and every rpc has its own `XRequest` / `XResponse` pair, even when one is
  empty.
- A surface only some callers may use (the combo host's controls) can be its own service
  (`ComboHostService`), so every rpc in it has one gate rule and nothing is ambiguous.
- **A list rpc carries a cursor and a limit** and returns `next_cursor`, unless the set is
  provably tiny and bounded; then say why in a comment.
- **Filters the client controls are request fields**, richly typed, so the server applies them in
  SQL. Never a "return everything and let the client narrow it" rpc.
- **Request messages never carry identity.** No `account_id` field that says who the caller is:
  the server reads that from the session. An id field names the thing being acted on (`scan_id`).
- **Requests never carry a verdict.** No score, tier, points or "plate passed" field the server
  would store: the phone sends photos and choices, and the response carries what the server
  decided.
- **Field numbers are append-only.** Never renumber, never reuse a retired number: retire it with
  `reserved N;` and a comment saying what it was. Enum values are append-only for the same reason.
- Use `optional` where "unset" and "zero" mean different things.
- Every file opens with a short header on the domain. Every rpc has a one-line comment, and field
  comments state units, bounds and what `0` or `""` means
  (`int32 max_minutes = 3; // 10, 20 or 45; 0 = any`).
- Field violations travel as an `error.v1.BadRequest` detail, keyed by the proto field name, which
  is also the name the client gives its input.
- **Codegen is committed.** `moon run proto:generate` writes the output; the `.proto` change and
  its regenerated code go in the SAME commit, because CI never runs `buf generate`. Never edit
  generated code by hand.
- buf v2, `lint: STANDARD`, `breaking: FILE`, remote plugins only. Go lands in
  `packages/go/apigen`, TypeScript in `packages/ts/apigen/src/gen`.
- Health is `health.v1.HealthService` with `Live` and `Ready` marked `NO_SIDE_EFFECTS`, so a probe
  can GET them. `Ready` logs the real failure and answers only "not ready".
