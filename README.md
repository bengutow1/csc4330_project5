# Fries in the Bag

A CSC 4330 project. Snap your fridge and the app finds a meal you can cook right now. If there
isn't one, it tells you to put the fries in the bag and finds food nearby. Cooking earns points on
a campus leaderboard, and Combo mode lets two friends merge their fridges to cook together. It is
an Expo app backed by a Go API. See `AGENTS.md` for the conventions.

## Prerequisites

- **[proto](https://moonrepo.dev/docs/proto/install)**. It installs every other tool at the
  version pinned in `.prototools` (moon, Go, Node, pnpm, buf, goose and the rest), so nothing else
  goes on your machine globally. Install the version CI pins (`.github/workflows/CI.yaml`):

  ```sh
  bash <(curl -fsSL https://moonrepo.dev/install/proto.sh) 0.57.4
  ```

  On Windows, use the PowerShell command from the link.
- **[Docker Desktop](https://www.docker.com/products/docker-desktop/)**, kept running: the local
  Postgres and the Go tests both run in containers.
- **A simulator for the Expo app**: the iOS Simulator, which comes with
  [Xcode](https://apps.apple.com/us/app/xcode/id497799835) (Mac only), or the Android Emulator,
  which comes with [Android Studio](https://developer.android.com/studio) (any OS). Expo's
  [environment guide](https://docs.expo.dev/get-started/set-up-your-environment/) walks through
  either one.

## Bootstrap

```sh
git clone https://github.com/bengutow1/csc4330_project5.git fries-in-the-bag
cd fries-in-the-bag
proto install            # pins moon, go, node, pnpm, … (.prototools)
pnpm install
moon sync hooks          # install the pre-commit / commit-msg hooks
```

## Run the API

```sh
moon run api:dev         # starts Postgres, migrates it, serves on :8080, restarts on every Go save
curl 'localhost:8080/health.v1.HealthService/Ready?encoding=json&message=%7B%7D'   # {} once it can reach the database
```

## Run the app

```sh
moon run mobile:ios      # builds the app into the iOS Simulator and starts Metro (slow the first time)
moon run mobile:dev      # after that: Metro only, and the installed app reloads on save
```

On Windows or Linux, `moon run mobile:android` builds it into the Android Emulator instead. The app
reaches the API on your machine, so run `moon run api:dev` beside it.

Every other command (migrations, codegen, the image) is in the table in `AGENTS.md`.

## CI

`moon ci` gates every PR and push to main (gitleaks → cog check → affected build/lint/test).
