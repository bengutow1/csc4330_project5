import { Tier } from "@fries-in-the-bag/api";

/** Where a verdict's button leads: the recipes for this scan, or the fries moment. */
export type Way = "cook" | "fries";

/** One of the verdict's two buttons: what it says and where it goes. */
export interface Choice {
  label: string;
  to: Way;
}

// The big cook button's words are screen 2's (docs/design.md). A bonus the tier earns rides on
// it in brackets, as on Cook anyway.
const COOK: Choice = { label: "Show me what I can cook", to: "cook" };
const NAH: Choice = { label: "Nah, fries", to: "fries" };

/**
 * buttonsFor is the verdict's two buttons for a tier, the big one first: the tier decides which
 * way is big. A tier the app doesn't know gets the plain cook buttons, so the player can still go
 * on.
 */
export function buttonsFor(tier: Tier): { big: Choice; small: Choice } {
  switch (tier) {
    case Tier.CERTIFIED_CHEF:
    case Tier.HOME_COOK:
      return { big: COOK, small: NAH };
    case Tier.SCRAPPY:
      return { big: { ...COOK, label: `${COOK.label} (+50)` }, small: NAH };
    case Tier.FRIES_IN_THE_BAG:
      return {
        big: { label: "Put the fries in the bag", to: "fries" },
        small: { label: "Cook anyway (+100)", to: "cook" },
      };
    default:
      return { big: COOK, small: NAH };
  }
}

/**
 * parseScanId reads a scan's id from its route: a whole number above zero, or undefined for
 * anything else, so the screen never asks the api about a scan that can't exist.
 */
export function parseScanId(param: string | undefined): bigint | undefined {
  if (param === undefined || !/^\d+$/.test(param)) {
    return undefined;
  }
  const id = BigInt(param);
  return id > 0n ? id : undefined;
}

/** What the verdict screen shows. */
export type Showing = "not-found" | "error" | "loading" | "verdict";

/**
 * showing picks what the verdict screen shows. A verdict in hand stays on screen: a refresh that
 * fails later keeps the cached answer, with the error flagged beside it, so the not-found and
 * error states are only for a verdict that never loaded. An answer with no verdict names no scan.
 */
export function showing(scan: {
  id: bigint | undefined;
  data: { verdict?: unknown } | undefined;
  notFound: boolean;
  failed: boolean;
}): Showing {
  if (scan.id === undefined) {
    return "not-found";
  }
  if (scan.data !== undefined) {
    return scan.data.verdict === undefined ? "not-found" : "verdict";
  }
  if (scan.notFound) {
    return "not-found";
  }
  return scan.failed ? "error" : "loading";
}
