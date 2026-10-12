---
paths:
  - "packages/ts/**"
---

# The client data layer

`packages/ts/apigen` holds the generated wire types and nothing else. `packages/ts/api` is the one
data layer every client uses: the transport, the query client, the hooks, error decoding, uploads
and live invalidation. Apps render; they do not talk to the network.

## Boundaries

- **Only `packages/ts/api` imports `apigen`.** An enum or type the app needs is re-exported from
  `api` by name, so the app never imports a wire type directly.
- **`packages/ts/api` is universal.** It never imports `react-native` or `react-dom` and never
  touches a DOM global. Platform differences arrive as arguments from the app's root, never as
  `.web.ts` / `.native.ts` forks in a shared package.
- `src/index.ts` re-exports every public name explicitly. No `export *`.
- Packages are source-only (`exports` point at `.ts` files). Relative imports inside a package
  carry the extension (`from "./errors.ts"`), and type-only imports are marked (`import type`),
  because `verbatimModuleSyntax` is on.

## Transport and query client

```ts
export function createApiTransport(opts: ApiTransportOptions): Transport {
  return createConnectTransport({
    baseUrl: opts.baseUrl,
    // Sessions are http-only cookies, so every request carries credentials.
    fetch: (input, init) =>
      (opts.fetch ?? globalThis.fetch)(input, { ...init, credentials: "include" }),
  });
}
```

- A factory, not a module singleton. The app builds the transport and the query client ONCE (in
  the root layout, inside `useMemo`) and hands both to `ApiProvider`: one rebuilt on render drops
  the whole cache. The app never imports connect-query or TanStack Query itself.
- The optional `fetch` is how a platform's own fetch arrives. React Native's built-in fetch cannot
  stream a response body, so a server stream (live updates) needs `expo/fetch` passed in.
- `makeQueryClient({ error, success })` is the toast seam: the data layer decides WHEN to notify,
  the app decides HOW. Defaults: `staleTime` 60 s, `retry: 1`, refetch on focus. Once accounts
  exist, any `Unauthenticated` error resets `whoAmI`, which sends the app back to onboarding.
- A success toast is opt-in per mutation through the typed `meta.successToast` (a string, or
  `(data, vars) => string`), and only where nothing visible changes on its own.

## Hooks

- One file per proto service in `src/hooks/`. Queries are `useScan` / `useScans`, mutations
  `useVerbNoun` (`useJoinCombo`), and each family has one invalidator, `useInvalidateScans()`.
- connect-query v2 over TanStack Query v5:
  `useQuery(ScanService.method.getScan, input, opts)`.
- **Never hand-build or index into a query key.** A whole family is
  `createConnectQueryKey({ schema, cardinality: undefined })`, which matches finite and infinite
  queries alike; narrow with `input` only for a per-entity read.
- Every filter goes into the input, and so into the key. Filtered or paged reads use
  `placeholderData: keepPreviousData` and report `isPlaceholderData` rather than a stale count.
- `skipToken` until the inputs are real: never send a request keyed on `""` or `0`. Fixed inputs
  are frozen module constants, so the key is stable.
- `retry: false` where a 401 or 403 is a legitimate answer (`useWhoAmI`).
- A mutation invalidates through its family's shared invalidator (see `live-updates.md`). To stale
  another service's family, import that service's schema, never its hooks.
- Creating an account or deleting one uses `queryClient.resetQueries()`, not
  `invalidateQueries`: invalidation keeps the old data on screen, so the next person briefly sees
  the previous one's.
- Render the server's order as given. Never re-sort or re-filter a server collection (the board,
  the recipe shelves), and never re-derive what the server decided (a score, a tier, points,
  whether a plate counted).

## Errors

```ts
// The codes whose text the server wrote for a person, mirrored from handlers.ConnectErr.
export function errorMessage(err: unknown): string {
  if (err instanceof ConnectError) {
    switch (err.code) {
      case Code.Unauthenticated:
      case Code.PermissionDenied:
      case Code.InvalidArgument:
      case Code.NotFound:
      case Code.AlreadyExists:
      case Code.FailedPrecondition:
      case Code.ResourceExhausted:
        return err.rawMessage;
      default:
        return "Something went wrong. Try again.";
    }
  }
  return "Something went wrong. Try again.";
}
```

- `fieldErrors(err)` returns `Record<string, string>` from the `error.v1.BadRequest` detail, keyed
  by proto field name. Inputs are named after those fields, so there is no mapping table. An empty
  record means "fall back to `errorMessage`".
- Small predicates (`isUnauthenticated`, `isNotFound`, `isFailedPrecondition`) keep screens from
  matching on codes. Never render "couldn't ask" as "doesn't exist".
- Raw, non-RPC routes (photo uploads, if they need one) go through one module in `api`
  (`upload.ts`) that sends credentials and reports "never reached the api" when the response is
  not ours. Never an inline `fetch` in a screen.

## Style

- File names are kebab-case; hook files are `use-*.ts`. Functions and components are
  `export function name()`, named exports only. Constants are SCREAMING_SNAKE.
- A doc comment sits above each exported symbol and says why it exists.
- `int64` ids arrive as `bigint`: `String(id)` for React keys and route params, and `BigInt(param)`
  inside try/catch when reading one back.
- Plurals are written by hand: `${n} ${noun}${n === 1 ? "" : "s"}`.
