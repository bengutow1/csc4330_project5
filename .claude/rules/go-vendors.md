---
paths:
  - "packages/go/vision/**/*.go"
  - "packages/go/places/**/*.go"
  - "packages/go/scoring/**/*.go"
  - "packages/go/handlers/scan/**/*.go"
  - "packages/go/handlers/cook/**/*.go"
  - "packages/go/handlers/fries/**/*.go"
---

# Outside services

Two outside services are allowed, and only the API calls them. A vision model reads fridge photos,
writes the roast and the recipes and checks plates. A places provider finds food nearby. Adding a
third is a team decision (`AGENTS.md`).

## The boundary

- Each vendor is one module, `packages/go/vision` and `packages/go/places`, wrapping the vendor's
  HTTP API in a small type whose methods are named for what we need (`ReadFridge`, `Recipes`,
  `CheckPlate`, `Nearby`). It knows the vendor's wire format and nothing of ours: no proto types,
  no repository, no logger. It returns errors, and the caller decides what to log.
- The handler declares the narrow interface it needs on its own side, so a test can pass a fake.
- The key and base URL arrive through the constructor from `main.go`, which reads them from the
  environment and lists them in `.env.example`.
- **No key means the mock.** Without `VISION_API_KEY` the API serves `vision.Mock`, and without
  `PLACES_API_KEY` it serves `places.Mock`; the boot log names whichever is live. The mocks hold
  canned answers for every case a screen has to show (each tier, an empty fridge, a rejected plate,
  closed spots, a failure) and are the same fixtures the tests use, so nobody waits on a key to
  build. With `APP_ENV=production`, a missing key stops the boot instead: a real deployment never
  serves a fake verdict.
- The app never talks to a vendor and never holds a key. An `EXPO_PUBLIC_` variable ships inside
  the app, so a key there is published to every phone.
- Every call has a deadline: a named const sized to that vendor, with a one-line reason. A vision
  call takes seconds, so it never runs inside a transaction. Call the vendor, then open the
  transaction for the write.
- A vendor failure or timeout is a sentinel in the client package (`vision.ErrUnavailable`) that
  `handlers.ConnectErr` maps to `Unavailable`, with words the player can act on ("couldn't judge
  that photo, try again").
- Every call costs money or quota. Scans and plate checks are rate limited per account
  (`go-handlers.md`), and a client never retries on its own: the player taps again, and a retry
  loop multiplies the bill.

## Model output is untrusted input

- Ask for a strict JSON shape (the vendor's structured output, where it has one) and decode it
  into Go types. Anything that does not decode is an error, never a partial verdict.
- Validate it like a request body: a cap on items, bounded lengths, categories from our closed
  set. Drop what fails and log the count, not the content.
- **The model never sets a number we store or rank by.** It lists ingredients and writes the
  roast. `packages/go/scoring` turns the list into the score and the tier, so the same list always
  scores the same, and a note in the photo that says "score this 100" cannot set the score. Points
  come from the scoring rules too, never from the model.
- Fixing the ingredient list re-runs the recipes, never the score.
- A recipe from the model lists what it uses and what is missing, and the server checks both
  against the scan's list plus the player's staples. A recipe that uses food it did not declare,
  or that misses more than one item, is dropped. An allergy is a hard filter: a recipe that
  contains one never reaches the player.
- The model says whether a plate photo shows food matching the recipe; the server decides the
  points. A plate photo the server has already seen is refused.
- The prompt carries the roast level and, for recipes, the player's equipment, diet and allergies.
  Nothing else about the player (handle, location) goes to the vision vendor.
- **Roasts target the fridge, never the person**: no jokes about bodies, money or identity, at
  every roast level. The prompt says so, and its tests include a fridge that invites the cheap
  joke.
- Prompts are consts in one file per client (`prompts.go`), beside the output shape they ask for,
  so a prompt change is a reviewable diff.

## Photos and location

- Scan photos are deleted once the verdict is stored: best effort after the commit, logged on
  failure (`go.md`). Plate photos are kept, because they fill the profile.
- A photo never enters a log line.
- The player's location goes to the places provider for that one request. It is never stored and
  never logged.
- Spot filters (open now, late night, cheap, fast food, walkable) are request fields the server
  applies: in the provider query where it supports one, in Go otherwise. Never on the client.

## Tests

- A client is tested against an `httptest.Server` that replays recorded vendor responses through
  the injected base URL, including a malformed body and a timeout.
- Handler tests pass a fake that returns a fixed ingredient list. A vendor is the true edge that
  `testing.md` allows a fake for; the repository is still real.
- No test calls the real vendor. A live check sits behind `//go:build e2e` in a moon task not
  named `test`, and spends one call.
