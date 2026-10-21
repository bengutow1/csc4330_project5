import { Tier } from "@fries-in-the-bag/api";
import { COLORS } from "../theme.ts";

/** TierLook is how a tier shows: its name, and the tint pair its pill is drawn in. */
export interface TierLook {
  label: string;
  fill: string;
  ink: string;
}

// Each tier's look, best first. A tier added to the wire fails to compile until it has its own.
const LOOKS: Record<Exclude<Tier, Tier.UNSPECIFIED>, TierLook> = {
  [Tier.CERTIFIED_CHEF]: { label: "Certified Chef", fill: COLORS.pickleTint, ink: COLORS.pickle },
  [Tier.HOME_COOK]: { label: "Home Cook", fill: COLORS.blueberryTint, ink: COLORS.blueberry },
  [Tier.SCRAPPY]: { label: "Scrappy", fill: COLORS.mustardTint, ink: COLORS.mustardInk },
  [Tier.FRIES_IN_THE_BAG]: {
    label: "Fries in the Bag",
    fill: COLORS.ketchupTint,
    ink: COLORS.ketchupInk,
  },
};

/**
 * tierLook is how a verdict's tier shows, or undefined when there is nothing to show: an
 * unspecified tier, or one newer than this build. Proto enums are open, so an app already on a
 * phone can be sent a tier it has never heard of, and it shows nothing rather than guess.
 */
export function tierLook(tier: Tier): TierLook | undefined {
  // Partial, because at run time the key can be any number the server sends.
  const looks: Partial<Record<Tier, TierLook>> = LOOKS;
  return looks[tier];
}
