import { Board, Range } from "@fries-in-the-bag/api";

const BOARDS = { chefs: Board.CHEFS, hall_of_fries: Board.HALL_OF_FRIES } as const;
const RANGES = { this_week: Range.THIS_WEEK, all_time: Range.ALL_TIME } as const;

/** BoardKey names a board for the switches, which take string values. */
export type BoardKey = keyof typeof BOARDS;
/** RangeKey names a stretch of time for the switches. */
export type RangeKey = keyof typeof RANGES;

/** The boards, in screen 7's order. */
export const BOARD_OPTIONS: readonly { value: BoardKey; label: string }[] = [
  { value: "chefs", label: "Chefs" },
  { value: "hall_of_fries", label: "Hall of Fries" },
];

/** The stretches of time, in screen 7's order. */
export const RANGE_OPTIONS: readonly { value: RangeKey; label: string }[] = [
  { value: "this_week", label: "This week" },
  { value: "all_time", label: "All time" },
];

/** boardOf is the board the request names, so the server does the switching. */
export function boardOf(key: BoardKey) {
  return BOARDS[key];
}

/** rangeOf is the stretch of time the request names. */
export function rangeOf(key: RangeKey) {
  return RANGES[key];
}

/** subtitle is the line under the title, as screen 7 writes it ("Campus · this week"). */
export function subtitle(range: RangeKey) {
  return range === "this_week" ? "Campus · this week" : "Campus · all time";
}

/** The fields of a board row these helpers read, so a test can build one by hand. */
interface RankedRow {
  rank: number;
  handle: string;
  value: bigint;
  crown: boolean;
}

/** BoardItem is one entry in the board's list: the podium, then one entry per row below it. */
export type BoardItem<R> = { kind: "podium"; rows: readonly R[] } | { kind: "row"; row: R };

// The podium holds the board's first three rows.
const PODIUM_SIZE = 3;

/**
 * distinctRows keeps each player's first row. Pages come one at a time, and a this-week board walked
 * across Monday midnight reads its next page from the new week, where a player already shown can
 * come back; the list keys rows by handle, so each may appear once.
 */
export function distinctRows<R extends { handle: string }>(rows: readonly R[]): R[] {
  const seen = new Set<string>();
  return rows.filter((row) => {
    if (seen.has(row.handle)) {
      return false;
    }
    seen.add(row.handle);
    return true;
  });
}

/**
 * boardItems lays the rows out as screen 7 does: the first three on the podium, the rest below,
 * in the order the server ranked them.
 */
export function boardItems<R>(rows: readonly R[]): BoardItem<R>[] {
  if (rows.length === 0) {
    return [];
  }
  const below = rows.slice(PODIUM_SIZE).map((row): BoardItem<R> => ({ kind: "row", row }));
  return [{ kind: "podium", rows: rows.slice(0, PODIUM_SIZE) }, ...below];
}

/**
 * podiumOrder stands the podium's rows as screen 7 does: the second on the left, the first in the
 * middle and the third on the right. A step nobody stands on is undefined.
 */
export function podiumOrder<R>(rows: readonly R[]): (R | undefined)[] {
  return [rows[1], rows[0], rows[2]];
}

/** podiumStep is the step a rank stands on: players tied for first share the tallest. */
export function podiumStep(rank: number) {
  if (rank === 1) {
    return "first";
  }
  return rank === 2 ? "second" : "third";
}

/**
 * pinsMyRow says whether the player's own row is pinned to the bottom: they have one, and nothing
 * on screen shows it. Until the list first says what is on screen, nothing is pinned.
 */
export function pinsMyRow(
  myHandle: string | undefined,
  onScreen: readonly BoardItem<{ handle: string }>[] | undefined,
) {
  if (myHandle === undefined || onScreen === undefined) {
    return false;
  }
  return !onScreen.some((item) =>
    item.kind === "podium"
      ? item.rows.some((row) => row.handle === myHandle)
      : item.row.handle === myHandle,
  );
}

/**
 * itemsInView is what sits at the positions the list last said were in view, read from the board as
 * it is now: a switch or a refetch can change the rows there without the list saying so again.
 * Unknown stays unknown until the list first says.
 */
export function itemsInView<T>(items: readonly T[], inView: readonly number[] | undefined) {
  return inView?.flatMap((index) => {
    const item = items[index];
    return item === undefined ? [] : [item];
  });
}

/** groupDigits writes a value with thousands separators, as screen 7 does ("1,240"). */
export function groupDigits(value: bigint) {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// countText is a board's value with its unit ("1 fry run", "1,240 points").
function countText(board: BoardKey, value: bigint) {
  const unit = board === "chefs" ? "point" : "fry run";
  return `${groupDigits(value)} ${unit}${value === 1n ? "" : "s"}`;
}

/** rowLabel is what a screen reader says for a row: the rank, whose it is, and the value. */
export function rowLabel(board: BoardKey, row: RankedRow, mine: boolean) {
  const crown = row.crown ? ", wearing the crown" : "";
  return `Rank ${row.rank}, ${mine ? "you" : row.handle}, ${countText(board, row.value)}${crown}`;
}

/** leaderLine names this week's Hall of Fries leader under the Chefs board, as screen 7 does. */
export function leaderLine(leader: { handle: string; value: bigint }) {
  return `${leader.handle} leads with ${countText("hall_of_fries", leader.value)}`;
}

/** emptyCopy is what a board with nobody on it says: in week one, or before anyone ever has. */
export function emptyCopy(board: BoardKey, range: RangeKey) {
  const when = range === "this_week" ? " this week" : "";
  return board === "chefs"
    ? { title: `No chefs yet${when}.`, line: "Plate a meal and the top spot is yours." }
    : { title: `No fry runs yet${when}.`, line: "Every fridge has held up so far." };
}
