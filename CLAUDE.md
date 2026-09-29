# live-poll

Live Poll, a CSC 2730 project: an Expo app where one person creates a question and gets a room
code, everyone else joins with the code and votes, and the results update live on every phone.
Its own Node server (WebSockets + SQLite) holds the rooms and votes — no third-party services.
`apps/` holds deployable applications, `packages/` holds shared libraries.

## Toolchain

The toolchain (moon, go, node, pnpm, …) is pinned in `.prototools` and installed with:

```sh
proto install
```

Don't install tools globally by hand — pin them in `.prototools` so everyone gets the same versions.

## Tasks

Tasks live in `.moon/tasks/<lang>.yml` and are inherited by every project. Prefer them over
per-project scripts.

```sh
moon ci                 # build / test / lint affected projects
moon run <project>:<task>
```

## Commits

This repo uses [Conventional Commits](https://www.conventionalcommits.org), enforced by a
`commit-msg` hook (`cog verify`) and CI (`cog check`).

- **Humans:** author with the interactive prompt — `czg` (or `git cz`).
- **When committing programmatically:** write a conventional message directly (e.g.
  `feat(app): add join screen`). Do not run `czg` non-interactively — it expects a TTY.

## Conventions

- New apps go under `apps/`, shared code under `packages/<lang>/`.
- Pin tool versions in `.prototools`, never ad-hoc installs.
- Let the moon tasks (`format`, `lint`, `typecheck`, `test`, `build`) drive checks.
- Secrets come from Infisical (`infisical run -- …`), never a committed `.env`.
