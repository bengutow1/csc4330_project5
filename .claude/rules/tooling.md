---
paths:
  - "**/moon.yml"
  - ".moon/**"
  - ".prototools"
  - "prototools/**"
  - "go.work"
  - "**/go.mod"
  - "**/package.json"
  - "pnpm-workspace.yaml"
  - "**/tsconfig*.json"
  - "biome.json"
  - ".golangci.yml"
  - ".github/**"
  - "docker/**"
  - "**/Dockerfile"
---

# Tooling

## moon owns every task

- Tasks are declared explicitly in each project's `moon.yml`. `package.json` scripts are never
  inferred (`inferTasksFromScripts: false`), and the root `package.json` has no scripts.
- Shared tasks live in `.moon/tasks/<lang>.yml`, scoped with `inheritedBy` toolchains: Go projects
  get `format`, `lint`, `test`, `build` and `tidy`; TypeScript projects get `format`, `lint` and
  `typecheck`, plus `test` where they have tests.
- A new project's `moon.yml` starts with `$schema`, then:
  - `id` with a language prefix for packages (`go-repository`, `ts-api`). Apps keep their folder
    name (`api`, `mobile`).
  - `layer`: `application` (apps), `library` (packages, proto) or `tool` (db, docker, scripts).
  - `language`: `go`, `typescript` or `bash`. Non-code projects (db, docker, proto) are `bash` so
    they inherit no Go or TypeScript tasks.
  - `project: { name, description }`, where the description is one sentence on the project's job
    ("The ONLY package that imports the generated wire types").
- Schema edges are explicit: `apigen` projects `dependsOn` `proto` and the repository depends on
  `db`, both with `scope: 'development'`, so a schema change marks every consumer as affected.
- Task names are lowercase single words (`dev`, `build`, `test`, `generate`, `migrate`, `seed`).
  Variants take a kebab-case suffix (`dev-phone`).
- `dev` runs a project locally, and `moon run :dev` starts the whole stack through `deps`
  (`api:dev` depends on `db:migrate`, which depends on `docker:up`). No aggregate root task.
  `api:dev` runs under Air (`apps/api/.air.toml`), so a Go save restarts the API and leaves Metro
  running.
- Local-only or mutating tasks: `options: { cache: false, runInCI: false }`. Long-running ones:
  `preset: 'server'`, or `persistent: true`. Replacing an inherited command:
  `mergeArgs: 'replace'`.
- Live or expensive test suites are never named `test`, so `moon run :test` cannot hit them, and
  they sit behind a build tag (`//go:build e2e`).
- Non-secret config goes in the task's `env:`. Secrets never appear in a `moon.yml`.
- Go relationship inference stays off in `.moon/toolchains.yml`; the comment there says why.
- Comments in config files say why a line exists, often naming the incident behind it. Keep them
  when you edit around them.

## Pinned tools

- Every tool is pinned in `.prototools` and installed with `proto install`. Never install one
  globally or with `npx` at an unpinned version.
- A tool proto cannot pin is pinned another way (an image tag, `@x.y.z` on an npm generator), with
  a comment saying why.
- CI pins proto itself. Raise it only after `proto install` succeeds locally on the new version.

## Go workspace

- Every Go package and app is its own module, listed in `go.work`. The module path is the repo
  path plus the directory (`github.com/<owner>/<repo>/packages/go/repository`).
- A module requires its siblings at `v0.0.0` with a relative `replace` for each, so it builds
  standalone:

```
require github.com/<owner>/<repo>/packages/go/repository v0.0.0

replace github.com/<owner>/<repo>/packages/go/repository => ../repository
```

- **The workspace build hides a missing `require`.** After changing a module's imports, run
  `go mod tidy` in it and build it standalone with `GOWORK=off`. gopls, CI and image builds read
  the module's own `go.mod`.
- moon runs `go work sync` before Go tasks, which can raise a sibling module's requirement
  without adding its `go.sum` line. If a standalone build then reports a missing `go.sum` entry,
  `go mod tidy` in that module fixes it; commit both files.
- Scripts (`scripts/seed`) are standalone modules kept out of `go.work` and run with
  `GOWORK: 'off'` in their task's `env`.
- `go.work.sum` is committed.

## TypeScript workspace

- pnpm workspaces over `apps/*` and `packages/ts/*`. Expo needs `nodeLinker: hoisted` in
  `pnpm-workspace.yaml`.
- Packages are `"private": true` and `"type": "module"`, depend on siblings with `workspace:*`,
  and are source-only: `exports` point straight at `.ts` files and there is no build step.
- Every project's `tsconfig.json` extends `tsconfig.base.json` (strict,
  `noUncheckedIndexedAccess`, `verbatimModuleSyntax`) and adds only what it needs.
- In the Expo app, add a native-facing package with `expo install` so its version matches the SDK,
  never a bare `pnpm add`. Adding any dependency is a question for the team first (see AGENTS.md).
- pnpm auto-installs a missing peer at its newest version, which can belong to the next React or
  React Native. Those peers are pinned in `pnpm-workspace.yaml` `overrides` to the versions the
  Expo SDK ships; bump them when the SDK moves, and check `pnpm install` prints no peer warnings.

## Lint and format

- Go: `gofmt` and golangci-lint v2 (`.golangci.yml`). TypeScript: Biome (`biome.json`). Proto:
  `buf format` and `buf lint`.
- Generated code (`packages/go/apigen`, `packages/ts/apigen/src/gen`) is excluded from lint and
  review, and is never hand-edited.
- A suppression carries a real reason: `//nolint:gosec // bounded by maxPhotos`,
  `// biome-ignore lint/a11y/useButtonType: <reason>`. A suppression that suppresses nothing is
  deleted.

## Git hooks

- Installed by `moon sync hooks`: pre-commit formats and lints the staged projects and runs
  gitleaks, commit-msg runs `cog verify`. The commit and PR rules themselves are in `AGENTS.md`.
- The format step rewrites files without re-staging them. If it changed something, stage and
  commit again.
- `cog.toml` sets `ignore_merge_commits = true`, because CI checks out GitHub's merge ref.

## CI and images

- `moon ci` gates every PR and push to `main`: gitleaks, then `cog check`, then build, lint and
  test of the affected projects. CI never runs `buf generate`; it consumes the committed output.
- Local Postgres comes from `docker/compose.yml` (`moon run docker:up`). Its image tag is the same
  one the testcontainers use, and its credentials are local-only defaults.
- A Go image is multi-stage, built from the repo root, `CGO_ENABLED=0`,
  `go build -trimpath -ldflags='-s -w'`, and runs on `gcr.io/distroless/static-debian12:nonroot`
  with the git sha baked in as `APP_VERSION`.
