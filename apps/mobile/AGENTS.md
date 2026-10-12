# apps/mobile

The Expo app. Everything in the root `AGENTS.md` applies here. The app's own conventions are in
`.claude/rules/expo.md` (screens, data, UI, copy) and `.claude/rules/ts-api.md` (the data layer it
calls); read both before editing anything in this folder.

Status: `src/screens/home-screen.tsx` is the demo prototype, mounted at `/`. It runs on mock data,
calls no server, and is replaced screen by screen as the mockups land: the tabs become Kitchen,
Board and Me, with scanning, cooking, Fries mode and Combo as flows over them. `src/app/status.tsx`
(open it with `friesinthebag://status`) asks the API whether it can serve, and is the one screen
wired to the server so far.

## Running it

- `moon run mobile:ios` builds the app into the iOS simulator and starts Metro. It is slow the
  first time, and needed again only when something native changes (app.json, the icon, a package
  with native code). `moon run mobile:android` does the same for the Android emulator.
- After that, `moon run mobile:dev` is the fast loop: Metro alone, and the installed app reloads on
  save. Stop `dev` before running `ios` or `android`, which start their own Metro.
- The app reaches the API at `localhost:8080` on the iOS simulator and `10.0.2.2:8080` on the
  Android emulator, so run `moon run api:dev` beside it. On a real phone, set
  `EXPO_PUBLIC_API_URL` to the Mac's LAN address.
- Maestro flows live in `.maestro/flows/`: `maestro test .maestro/flows` from this folder, with the
  app installed and the API running.

## Expo has changed: do not trust your training data

Expo ships breaking changes every SDK release. Before writing code that touches an Expo, EAS or
React Native API:

1. Read the major version of `expo` in `package.json`.
2. Read the matching versioned docs: `https://docs.expo.dev/versions/v<major>.0.0/`.
3. For anything else, start at `https://docs.expo.dev/llms.txt` and follow its links to the page
   you need. Never answer from memory.

## Rules

- Routes live in `src/app/` and nothing else does; everything else sits beside it in `src/`.
- `ios/` and `android/` are generated (continuous native generation). Never create or edit them;
  configure native behaviour in the app config and config plugins.
- Develop in a build of this app (`moon run mobile:ios`), not Expo Go. After adding a library with
  native code, rebuild it.
- Add a package with `pnpm exec expo install <package>` from this folder so its version matches the
  SDK, and only after the team agreed to the dependency.
- Prefer the Expo module over a third-party library when both exist.
- Run lint and typecheck before calling a change done, and show the screen working on the iOS
  simulator.
