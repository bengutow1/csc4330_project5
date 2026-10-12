---
paths:
  - "apps/mobile/**"
---

# The Expo app

## Before writing code

- Read the versioned Expo docs for the SDK pinned in `apps/mobile/package.json`
  (`https://docs.expo.dev/versions/v<major>.0.0/`); Expo changes between SDKs and training data
  lags.
- Develop in a build of this app (`moon run mobile:ios`, which runs `expo run:ios`), not Expo Go.
  Expo Go's fixed native modules drift from the real build. Adding a native module means
  rebuilding it.
- `ios/` and `android/` are generated (continuous native generation): never committed, never
  edited by hand. Native settings go in the app config and config plugins.

## Structure

- Expo Router, file based. Routes live in `src/app/` and NOTHING else does: any file there becomes
  a route. Everything else sits beside it under `src/` (`src/screens`, `src/components`,
  `src/lib`), imported through the `@/*` alias for `./src/*`.
- **A route file is a mount.** It reads the URL params and renders one screen component. The
  screen owns its data, its header, and its loading, empty and error states.

```tsx
export default function RecipeRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <RecipeScreen id={id} />;
}
```

- Route files use a default export because expo-router requires it; everything else uses named
  exports. Typed routes are on: navigate with
  `{ pathname: "/recipe/[id]", params: { id: String(recipe.id) } }`.
- Three tabs (Kitchen, Board, Me) live in a `(tabs)` group. Scanning, cooking, Fries mode and
  Combo are stacks that open over them.
- The root and tab layouts set `unstable_settings` anchors, so a deep link still gets a back stack.
- **Gate with `<Stack.Protected guard={…}>`** in the root layout (onboarding until the device
  account exists), never a `useEffect` that calls `router.replace`: the effect races expo-router's
  own URL resolution on a cold start and bounces deep links. Always render the `<Stack>`; returning
  `null` while loading unmounts the route tree.
- **Pure logic lives in its own file with no React** (which button a tier makes big, the lines of
  a receipt) and never mutates its input. Derived values are computed during render, not stored
  in state.
- Platform branches use `process.env.EXPO_OS`. A `*.web.tsx` or `*.ios.tsx` sibling lives only in
  the app, never in a shared package, and carries a comment justifying it.

## Data

- **Every server call goes through a hook from `packages/ts/api`** (see `ts-api.md`). Never
  hand-roll `fetch` against the API. The root layout builds the transport and the query client
  once (`useMemo`) and hands both to `ApiProvider`, and `useLiveInvalidation()` is mounted once,
  in the layout that wraps the signed-in screens.
- The app talks to our API and nothing else. It never calls an AI model or a places API and never
  holds a vendor key: those calls go through the server (`go-vendors.md`).
- The base URL is `EXPO_PUBLIC_API_URL` when set (a phone on the LAN), else
  `http://10.0.2.2:8080` on the Android emulator, `http://localhost:8080` on the iOS simulator and
  `/` on web. On a phone, `localhost` is the phone.
- Server state lives in the query cache. Never copy it into `useState` or a client store; read it
  from the query.
- The server decides the score, the tier, points and whether a plate counts. The app sends photos
  and choices, renders what comes back, and never computes a number it would send.
- The session is an http-only cookie and the device account's only credential. Never store a
  token in JS or AsyncStorage. AsyncStorage holds device preferences only, and the code survives
  it being empty or throwing.

## Forms

- No form library. Hold inputs in `useState`, submit through the mutation hook, and render
  `fieldErrors(mutation.error)` under the input with the same proto field name. Show
  `errorMessage` inline only when there are no field violations.
- A submit button is disabled only while its write is in flight, never because the form looks
  incomplete: the server's validation is the one truth and says what is missing.

## UI

- Colours come from theme tokens. No hex values in component code.
- `expo-image` for images; `FlashList` for any list that can grow long. An infinite list uses plain
  `onEndReached` with `onEndReachedThreshold={0.5}` and an `isFetchingNextPage` check. If a list
  runs away, find the explicit scroll call causing it; never stack guards.
- Loading, empty and error are three different states: one `<Skeleton>` shaped like the real
  content, one `<EmptyState>` with a way forward, and error copy that never borrows the empty
  state's words ("Couldn't load the board", not "No chefs yet").
- An error shows in ONE place: the global toast, or inline beside the field it belongs to. Field
  validation errors from the server render inline under their field.
- Every icon-only control has `accessibilityRole="button"` and an `accessibilityLabel` that says
  what it DOES.
- Confirm anything that affects the other player (ending a combo, removing a player) with the
  app's own dialog component. Never `Alert.alert` (a no-op on web) or `window.confirm`.
- Hiding a control is a courtesy, not a permission: the server checks every call again.
- Ask for each permission at first use, never at launch: the camera on the scan screen, location
  at the fries moment, notifications after the first points receipt. A denied permission gets a
  screen that says why it was needed and offers the way around it (a search box instead of
  location).
- Fast food in general, no chain in particular: no logos or mascots, and spot names stay plain
  text.
- Dev-only diagnostics sit behind `__DEV__`.

## Copy

- No em dashes in any copy.
- Deadpan and short. Roasts and every other joke target the fridge, never the person.
- Sentence case for titles, buttons and toasts ("Judge my fridge", "Combo ended"). Pending labels
  use the single ellipsis character: "Judging…". Stamps and receipts are art, not copy, and may
  shout (CERTIFIED CHEF).
- Never claim what the server has not confirmed: no points line before the receipt arrives.
- "Improve the UI" means declutter and add breathing room, not a redesign and not more
  information. Ask before anything bigger.
