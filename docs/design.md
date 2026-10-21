# Design

The app's pages follow the Fries in the Bag App Flow design in `docs/design/`. Open
`docs/design/app-flow.html` in a browser to see all ten screens. For each screen,
`docs/design/screens/NN-name.jpg` is a picture of it and `NN-name.html` is its exact markup: every
size, colour and radius, readable as text (the photos are left out of it).

Where an issue's look differs from these screens (its colours, fonts, the concept link, receipt
paper or stamps), the screens win. The issue still decides what a screen does: its data, its
states and its rules. A screen the design doesn't draw is built from the same tokens and
components.

## Screens

| # | Screen | Files | Built by |
| --- | --- | --- | --- |
| 1 | Snap: the camera, and the first tab | `01-snap` | #18 |
| 2 | Ingredients found | `02-ingredients` | #19 |
| 3 | Recipe ideas | `03-recipe-ideas` | #22 |
| 4 | Nothing to cook: the fries moment | `04-nothing-to-cook` | #27 |
| 5 | Cook it: one recipe | `05-cook-it` | #22 (step-by-step cooking is #40, Next) |
| 6 | Plate it: the points | `06-plate-it` | #24, from #10's pieces |
| 7 | Leaderboard | `07-leaderboard` | #29 |
| 8 | Combo | `08-combo` | Next (#35, #36, #39) |
| 9 | Spots nearby | `09-spots-nearby` | #27 (Fry Roulette is #42, Next) |
| 10 | Profile | `10-profile` | #30 (the badge grid is #43, Next) |

Onboarding (#14) and settings (#16) aren't drawn. Each screen's caption above the phone frame
("1. Snap your fridge or pantry") belongs to the design board, not the app.

Open questions for Caden, with the default until they're answered:

- **The kitchen home.** The design has none: the app opens on the Snap tab. #15 waits until
  Caden decides whether it stays.
- **The verdict.** Screen 2 doesn't draw the score, tier or roast. By default they stay, at the
  top of screen 2 in the design's style: the score as a big number like screen 6's total, the
  tier as a pill and the roast as one line. The tier picks the big button: "Show me what I can
  cook", or "Put the fries in the bag", which opens screen 4.

## Tokens

`src/theme.ts` (#4) holds these, and no hex appears anywhere else. Illustration colours (the
medals, category icons) are in the screen files.

| Token | Hex | Used for |
| --- | --- | --- |
| `bg` | `#FBF8F4` | Screen background |
| `surface` | `#FFFFFF` | Cards, the tab bar, secondary buttons |
| `ink` | `#1E1A17` | Titles, body text, icons |
| `inkSoft` | `#2B2420` | Longer copy |
| `muted` | `#6E655D` | Subtitles, secondary text, inactive tabs and chips |
| `line` | `#EEE6DC` | Card borders, the tab bar's top border |
| `lineStrong` | `#DCD1C4` | Secondary button and chip borders |
| `divider` | `#F2EBE2` | Rows inside a card or list |
| `sunken` | `#EFE7DD` | Segmented control track, the combo code, stat tiles |
| `ketchup` | `#C8322B` | Primary buttons, selected chips and tabs, links, point totals |
| `ketchupInk` | `#A8261F` | Red text on light fills, a pressed primary button |
| `ketchupTint` | `#FBEAE7` | The selected pill's fill |
| `mustard` | `#F7C531` | The fries in the logo |
| `mustardInk` | `#8A5A00` | Text on `mustardTint` |
| `mustardTint` | `#FFF1D6` | The "Nearest fries" badge, rank and trophy circles |
| `pickle` | `#23794A` | "Open until…", passed checks |
| `pickleTint` | `#E8F4EC` | The "Looks like…" check pill |
| `blueberry` | `#1D4FA3` | The other player in Combo, avatar letters |
| `blueberryTint` | `#E7EEFA` | Fills behind `blueberry` |
| `bag` | `#D62F27` | The bag in the logo |

## Type

The system font everywhere (SF Pro on iOS), except the logo in Lilita One from
`@expo-google-fonts/lilita-one`.

| Style | Size / weight | Used for |
| --- | --- | --- |
| `logo` | Lilita One 46 and 25 | "Fries" over "in the Bag", and nothing else |
| `hero` | 52 / 800 | The big point total ("+140") |
| `title` | 24 / 800 | Screen titles ("Here's what we found"); a tab's root title is 26 |
| `heading` | 20 / 800 | Card and section headings |
| `button` | 17 / 700 | Primary buttons; secondary buttons are 16 / 700 |
| `bodyStrong` | 15 / 700 | List item names |
| `body` | 15 / 400 | Copy, and subtitles in `muted` |
| `small` | 14 / 600 | Chips, links, meta lines |
| `caption` | 13 / 400 | Lines under a row's name |
| `tab` | 11 / 600 | Tab labels, 700 when active |

## Components

#5 builds the shared ones into `src/components/`, and #10 builds the plate pieces.

- **Button.** Primary: full width, 52 high, radius 14, `ketchup` fill, white `button` text.
  Secondary: `surface` fill, 1px `lineStrong` border, `ink` at 16 / 700. A pending label ends
  with the ellipsis character.
- **Pill toggle** (screen 1, Photo / Pantry). 40 high and fully round. Selected: `ketchupTint`
  fill, `ketchup` 15 / 700 text and a 5px dot under it. Unselected: no fill, `muted` 15 / 600.
- **Filter chip** (screen 9). 40 high and fully round. Selected: `ketchup` fill, white 14 / 700.
  Unselected: `surface` fill, 1px `lineStrong` border, `ink` 14 / 600.
- **Segmented control** (screen 7, Chefs / Hall of Fries). A `sunken` track. The selected
  segment is `surface`, 36 high, radius 9, `ink` 14 / 700; the others are `muted` 14 / 600.
- **Underline tabs** (screen 7, This week / All time). Selected: `ketchup` 14 / 700 with a 2px
  `ketchup` underline. Unselected: `muted` 14 / 600.
- **Card.** `surface`, 1px `line` border, radius 16, padding 12 by 14. A featured card (screen
  9's nearest fries) has a 2px `ketchup` border. Rows inside it are split by 1px `divider`.
- **Header.** A back arrow at the top left (a close cross on screen 6), then the centred `title`
  and a `muted` `body` subtitle.
- **Bottom bar.** The screen's main action stays pinned at the bottom: padding 12 at the top, 16
  at the sides and 30 at the bottom, with a 1px `line` top border.
- **Tab bar.** `surface`, 1px `line` top border, three equal columns of an icon over a `tab`
  label: Snap, Board and Me. Active is `ketchup` at 700, inactive `muted` at 600.
- **Shutter** (screen 1). A 76px circle: `surface` fill and a 5px `ketchup` ring.
- **Photo card** (screen 6). The plate photo in a white frame, slightly tilted, with the time
  under it.
- **Check pill** (screen 6). `pickleTint` fill with `pickle` text and icon: "Looks like Creamy
  Tomato Pasta".
- **Points breakdown** (screen 6). Under the `hero` total and "points earned": a card of reason
  and amount rows, then a Total row in `ketchup`.
- **Logo** (screens 1 and 4). "Fries" over "in the Bag" in Lilita One, beside the fries in a bag.
  Its SVG is in `01-snap.html`.

## Art, photos and names

- The logo, the medals and the other drawings in the screen files are inline SVG. Save each as a
  `.svg` under `apps/mobile/assets/` and draw it with `expo-image`, which renders SVG and tints
  one-colour art with `tintColor`. `expo-image` arrives with #3 in its one native rebuild, so this
  needs neither `react-native-svg` nor a second rebuild. Plain icons come from Ionicons.
- The design's photos are placeholders. Our data has no ingredient or recipe photos, so those
  tiles show a `sunken` tile with the category's icon, at the same size and radius. The camera
  view, plate photos and "My plates" are real photos.
- Spot names come from the places provider as plain text, and the mock's are invented. The
  design's McDonald's is a placeholder, and no chain logo appears (`expo.md`).
- Copy comes from the design wherever it draws the screen, under the rules in `expo.md`: sentence
  case, no em dashes, and the ellipsis character on pending labels.

## What changes in the issues

- **#3.** `@expo-google-fonts/lilita-one` replaces `@expo-google-fonts/anton` and
  `@expo-google-fonts/space-mono`. The root background is `#FBF8F4`, and the interface style is
  light.
- **#4.** The tokens and type above replace the issue's colours and fonts. The tabs are Snap,
  Board and Me, with Snap first, and the route table in `docs/contracts.md` puts Snap at
  `(tabs)/index`.
- **#18.** The camera is the Snap tab at `(tabs)/index`, not a `scan/index` stack.
- **#5.** The components above set the look. The primary button is `ketchup`, not gold.
- **#10.** The design has no receipt paper or rubber stamp. Build the photo card, check pill,
  points breakdown, a tier pill for screen 2 and `CountUp` instead, and show them in `/kit`.
- **#15.** On hold: see the open questions.
- **#19.** Screen 2, plus the score, tier and roast by default: see the open questions.
