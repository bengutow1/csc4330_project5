# live-poll

Live Poll: create a question, share the room code, and watch the votes come in live on every
phone. An Expo app backed by its own Node server (WebSockets + SQLite). CSC 2730 project.

## Bootstrap

```sh
proto install            # pins moon, go, node, pnpm, … (.prototools)
pnpm install
moon sync hooks          # install the pre-commit / commit-msg hooks
```

## CI

`moon ci` gates every PR and push to main (gitleaks → cog check → affected build/lint/test).
