/** The fields of a combo player these helpers read, so a test can build one by hand. */
export interface Side {
  accountId: bigint;
  handle: string;
  /** Unset until this player has scanned for the combo. */
  scan?: { score: number; ingredients: readonly unknown[] };
}

/**
 * sides splits a combo's players into the caller and their friend, by the caller's handle (the
 * account the app knows has no id). Either is missing while that seat is empty or unknown.
 */
export function sides<P extends Side>(
  players: readonly P[],
  myHandle: string | undefined,
): { me?: P; them?: P } {
  return {
    me: players.find((p) => p.handle === myHandle),
    them: players.find((p) => p.handle !== myHandle),
  };
}

/** fridgeLabel names a player's fridge from the caller's side: "Your fridge", "maya's fridge". */
export function fridgeLabel(side: Side, mine: boolean): string {
  return mine ? "Your fridge" : `${side.handle}'s fridge`;
}

/** countLine counts a fridge's items, written out by hand. */
export function countLine(n: number): string {
  return `${n} item${n === 1 ? "" : "s"}`;
}

/** mergedLine is the line under the two players once both fridges are in. */
export function mergedLine(n: number): string {
  return `Fridges merged. ${n} ingredient${n === 1 ? "" : "s"} between you.`;
}

/**
 * waitingLine says what the reveal is waiting on, in the order it has to happen: a friend to join,
 * the caller's own scan, then the friend's.
 */
export function waitingLine(me: Side | undefined, them: Side | undefined): string {
  if (them === undefined) {
    return "Waiting for a friend to join…";
  }
  if (me?.scan === undefined) {
    return "Scan your fridge to see what you make together.";
  }
  return `Waiting for ${them.handle} to scan…`;
}

/** Chip is one food on the merged list, and whether it came from the caller's fridge. */
export interface Chip {
  key: string;
  name: string;
  mine: boolean;
}

/**
 * chips are the merged ingredients in the server's order, each marked by whose fridge it came
 * from. The same food from both fridges stays two chips, one each, as the server keeps it.
 */
export function chips(
  owned: readonly { accountId: bigint; ingredient?: { name: string } }[],
  myAccountId: bigint | undefined,
): Chip[] {
  return owned.map((o, i) => ({
    key: `${i}:${o.accountId}`,
    name: o.ingredient?.name ?? "",
    mine: o.accountId === myAccountId,
  }));
}

/**
 * The reveal's steps: the first fridge's score, the second's, then the combined score with its
 * chips. STEP_MS apart, so each lands before the next.
 */
export const STEPS = { first: 1, second: 2, combined: 3 } as const;
export const STEP_MS = 900;

/**
 * openingStep is where the reveal starts. A phone that watched the combo wait plays every step;
 * one that opened it after both fridges were in, or that asks for reduced motion, starts at the
 * end, showing the final state at once.
 */
export function openingStep(watchedItWait: boolean, reduceMotion: boolean): number {
  return watchedItWait && !reduceMotion ? 0 : STEPS.combined;
}
