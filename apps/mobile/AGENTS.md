# apps/mobile

The Expo app. Everything in the root `AGENTS.md` applies here. The app's own conventions are in
`.claude/rules/expo.md` (screens, data, UI, copy) and `.claude/rules/ts-api.md` (the data layer it
calls); read both before editing anything in this folder.

Status: the shell, behind a device account. The root layout asks `WhoAmI` and guards the routes
with `Stack.Protected` in three stages: no account opens onboarding's welcome and handle screens
(O1, O2), the kitchen screen (O3) opens between making the account and saving the kitchen (the
`Setup` context in `src/lib/setup.ts`), and every other Core route needs an account. A new route
that needs an account goes into the last guard's list in `src/app/_layout.tsx`, or anyone can open
it. A route meant for anyone also goes in `OPEN_ROUTES` (`src/lib/launch.ts`), or the launch cover
hides it until the server answers. The tabs are Snap (the camera), Board and Me, with the verdict,
cooking, Fries mode and Combo as flows over them. Every Core route in `docs/contracts.md` is built
from the design (`docs/design.md`), and Next issues add their own routes. Settings (P3,
`src/app/settings.tsx`) opens from the gear on the Me tab and reuses O3's `KitchenForm`.
`src/app/status.tsx` (open it with `friesinthebag://status`) asks the API whether it can serve, with
or without an account.

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
  app installed and the API running. A flow that needs a player starts with `clearState: true`,
  then `runFlow: subflows/sign-up.yaml`.

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
