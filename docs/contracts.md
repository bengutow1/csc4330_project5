# Shared contracts

The facts that four people's code must agree on. Every issue sends you here right after
`AGENTS.md`. Changing one is its own small PR, reviewed by the owner named below, and nobody keeps a
second copy. Owners: Caden (@cadenlund), Ben (@bengutow1), Andrew (@andrewkilp), Arthur
(@Arthur2457).

## Closed sets

Spelled the same everywhere: lowercase in Go consts, SQL checks and the app, and in proto the way
buf requires (`CATEGORY_UNSPECIFIED = 0`, then `CATEGORY_PROTEIN = 1` and so on).

| Set | Values | Proto home | Owner |
| --- | --- | --- | --- |
| Photo kind | fridge, freezer, pantry | `scan.v1` | Ben |
| Ingredient category | protein, vegetable, fruit, grain, dairy, condiment, drink, snack, leftover, other | `scan.v1` | Ben |
| Tier | certified_chef (85 to 100), home_cook (60 to 84), scrappy (35 to 59), fries_in_the_bag (0 to 34) | `scan.v1` | Ben |
| Roast level | mild, medium (the default), extra_crispy | `account.v1` | Caden |
| Equipment | microwave, stovetop, oven, air_fryer | `account.v1` | Caden |
| Diet | none, vegetarian, vegan, pescatarian | `account.v1` | Caden |
| Allergy | peanuts, tree_nuts, milk, eggs, wheat, soy, fish, shellfish, sesame | `account.v1` | Caden |
| Staple (all on by default) | salt, pepper, oil, butter, rice, pasta, flour, sugar, spices | `account.v1` | Caden |
| Recipe difficulty | easy, medium, hard | `cook.v1` | Andrew |
| Point reason | cook, scrappy_bonus, against_all_odds, streak, combo_cook | `cook.v1` | Andrew |
| Spot filter | open_now (on by default), late_night, cheap, fast_food, walkable | `fries.v1` | Arthur |
| Board and range | chefs, hall_of_fries; this_week, all_time | `board.v1` | Arthur |

A handle is 3 to 20 lowercase letters, digits and underscores, and unique (`chef_maya`). An avatar
is one emoji.

## Score, points and time

- **The score** comes from `scoring.Fridge(items)`, once, when the scan is created, and is stored on
  the scan. Editing the ingredients never changes it. The tier is computed from the score and never
  stored.
- **Points** are a ledger: one row per award with its reason, and every total is a sum over those
  rows.

| Reason | Points | When |
| --- | --- | --- |
| cook | +100 | A plate the server checked, at most one per player per meal window |
| scrappy_bonus | +50 | The recipe came from a scan that scored 35 to 59 |
| against_all_odds | +100 | The recipe came from a scan that scored under 35 |
| streak | +10 a day, at most +70 | The day's first checked plate earns +10 for each day of the current streak, today included, capped at 7 days |
| combo_cook | +50 each | Next: both players in a combo; one plate counts for both |

- **Campus time.** Every day, week and meal-window boundary uses `America/Chicago`, one const in
  `packages/go/points`. Weeks start Monday at 00:00.
- **Meal windows:** morning 05:00 to 10:59, midday 11:00 to 15:59, evening 16:00 to 21:59, late
  night 22:00 to 04:59 (it belongs to the day it started).
- **Ranks** by all-time points: Fry Cook 0, Line Cook 500, Sous Chef 1,500, Head Chef 4,000,
  Executive Chef 10,000.
- **Badges** are the concept's eleven, derived on read from a player's history and never stored. A
  new badge on the points receipt is one earned after this plate and not before it.

## The mock AI and places

- No `VISION_API_KEY` means the API uses `vision.Mock`; no `PLACES_API_KEY` means `places.Mock`.
  The boot log names whichever is live. With `APP_ENV=production` a missing key stops the boot
  instead, so a real deployment can never serve fake verdicts.
- `vision.Mock` holds five fridges (one per tier, plus an empty one) built from the concept's sample
  roasts and ingredients, recipes for each (including one "one item away" recipe and one invalid
  recipe the server must drop), and a plate verdict. Each call waits 2 seconds so the judging
  screen is real. It cycles through its fridges in order; `VISION_MOCK=<name>` pins one
  (`certified_chef`, `home_cook`, `scrappy`, `fries_in_the_bag`, `empty`, `error`), and
  `VISION_MOCK_PLATE=reject` fails every plate check.
- `places.Mock` places about ten made-up spots around whatever location the request sends, so
  distances look right anywhere: fast food open late, one closed spot, two cheap ones, three
  walkable ones. `PLACES_MOCK=closed` closes them all.
- The API tests use the same fixtures through a fake, so tests and local dev agree.

## Photos

- The app shrinks every photo before it leaves the phone (`expo-image-manipulator`): scans to
  1024 px on the long edge at JPEG quality 0.7, up to three per scan; plates to 640 px at 0.6.
- Photos travel as `bytes` fields in the RPC; there is no upload route. The API refuses anything
  that is not a JPEG or is over 2 MB.
- Scan photos are never stored: request, vision model, gone. Plate photos are stored in Postgres
  with their sha256, and a hash the server has already seen is refused. The profile gets plate
  thumbnails back as bytes in `ListPlates`.

## Migrations

Numbers are reserved so parallel branches never collide. Each owner edits only their own file, in
place, until the demo (`go-repository.md`).

| File | Owner | Tables |
| --- | --- | --- |
| `00002_accounts.sql` | Caden | `app.account`, plus the sessions table exactly as the pgxstore README defines it |
| `00003_scans.sql` | Ben | `app.scan`, `app.scan_item` |
| `00004_recipes.sql` | Andrew | `app.recipe` |
| `00005_plates.sql` | Andrew | `app.plate`, `app.point_event` |
| `00006_fries.sql` | Arthur | `app.fry_run` |
| `00007_combos.sql` | Caden (Next) | `app.combo`, `app.combo_member`, and `combo_id` on `app.scan` |
| `00008_crews.sql` | Ben (Next) | `app.crew`, `app.crew_member` |

Every table that references `app.account` declares `on delete cascade`, so deleting an account
(#16) removes everything the player had.

## Routes

Issue #4 created every Core route as a placeholder, so any screen could link to any other before it
was built, and each owner replaced their own with its screen.

| Route under `src/app/` | Screen | Issue |
| --- | --- | --- |
| `(tabs)/index` | Snap: the camera, then judging (S1, S2) | #18 |
| `(tabs)/board` | Board (B1) | #29 |
| `(tabs)/me` | Profile (P1) | #30 |
| `onboarding/index`, `onboarding/handle`, `onboarding/kitchen` | O1, O2, O3 | #6, #14 |
| `settings` | Settings (P3) | #16 |
| `scan/[id]` | Verdict, with the edit sheet (S3, S4) | #19 |
| `cook/[scanId]` | Recipes (C1) | #22 |
| `recipe/[id]` | Recipe (C2) | #22 |
| `plate/[recipeId]` | Plate it, then the receipt (C4, C5) | #24 |
| `fries/index` | Fries moment (F1) | #27 |
| `fries/spots` | Spots, with the spot sheet (F2, F3) | #27 |
| `crews` | Crews: your crews, make one, join with a code (B3) | #38 |
| `combo/[id]/reveal` | Combo reveal: both scores, then the combined one (X3) | #36 |
| `combo/[id]/recipes` | Combo recipes with ingredient owners (X4) | #39 |
| `combo/[id]/bring` | Kitchen choice and the guest's bring checklist (X5) | #39 |

The app opens on the Snap tab, as the design draws it (`docs/design.md`), so the kitchen home (K1,
#15) has no route while it is on hold. Next issues add their own routes (`combo/…`,
`recipe/[id]/steps`, `fries/roulette`, `badges`), and `status` stays as it is.

## Freshness

There is no live push. A screen's data refreshes when the screen opens, when the app comes back to
the foreground (an `AppState` subscription the root layout hands to `ApiProvider`, issue #4), and
after the player's own writes. The one exception is Combo: the lobby and the reveal ask for the
combo every 2 seconds while they are open, and stop once it is revealed or ended.
