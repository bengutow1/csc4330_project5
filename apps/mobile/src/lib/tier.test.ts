import { Tier } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { tierLook } from "./tier.ts";

const TIERS = [Tier.CERTIFIED_CHEF, Tier.HOME_COOK, Tier.SCRAPPY, Tier.FRIES_IN_THE_BAG];

describe("tierLook", () => {
  it("names each tier the way the app's words do", () => {
    expect(TIERS.map((tier) => tierLook(tier)?.label)).toEqual([
      "Certified Chef",
      "Home Cook",
      "Scrappy",
      "Fries in the Bag",
    ]);
  });

  it("draws each tier in a tint of its own", () => {
    const fills = TIERS.map((tier) => tierLook(tier)?.fill);
    expect(new Set(fills).size).toBe(TIERS.length);
  });

  it("has nothing to show for an unspecified tier", () => {
    expect(tierLook(Tier.UNSPECIFIED)).toBeUndefined();
  });

  it("has nothing to show for a tier newer than this build", () => {
    // Proto enums are open: a newer server's tier arrives as a number this build has no name for.
    expect(tierLook(5 as Tier)).toBeUndefined();
  });
});
