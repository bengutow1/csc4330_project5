import { Tier } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { buttonsFor, parseScanId } from "./verdict.ts";

describe("buttonsFor", () => {
  // Every tier written out by hand: the big button, then the one beside it.
  const cases = [
    {
      name: "certified chef",
      tier: Tier.CERTIFIED_CHEF,
      big: { label: "Show me what I can cook", to: "cook" },
      small: { label: "Nah, fries", to: "fries" },
    },
    {
      name: "home cook",
      tier: Tier.HOME_COOK,
      big: { label: "Show me what I can cook", to: "cook" },
      small: { label: "Nah, fries", to: "fries" },
    },
    {
      name: "scrappy",
      tier: Tier.SCRAPPY,
      big: { label: "Show me what I can cook (+50)", to: "cook" },
      small: { label: "Nah, fries", to: "fries" },
    },
    {
      name: "fries in the bag",
      tier: Tier.FRIES_IN_THE_BAG,
      big: { label: "Put the fries in the bag", to: "fries" },
      small: { label: "Cook anyway (+100)", to: "cook" },
    },
  ];

  it.each(cases)("gives $name its two buttons", ({ tier, big, small }) => {
    expect(buttonsFor(tier)).toEqual({ big, small });
  });

  it("gives a tier the app doesn't know the plain cook buttons, so the player can still go on", () => {
    const plain = {
      big: { label: "Show me what I can cook", to: "cook" },
      small: { label: "Nah, fries", to: "fries" },
    };

    expect(buttonsFor(Tier.UNSPECIFIED)).toEqual(plain);
    expect(buttonsFor(5 as Tier)).toEqual(plain);
  });
});

describe("parseScanId", () => {
  it("reads a scan's id from its route", () => {
    expect(parseScanId("42")).toBe(42n);
  });

  it("reads an id too big for a JavaScript number", () => {
    expect(parseScanId("9007199254740993")).toBe(9007199254740993n);
  });

  it("is undefined for anything that can't name a scan, so nothing is asked for", () => {
    expect(parseScanId(undefined)).toBeUndefined();
    expect(parseScanId("")).toBeUndefined();
    expect(parseScanId("0")).toBeUndefined();
    expect(parseScanId("-3")).toBeUndefined();
    expect(parseScanId("1.5")).toBeUndefined();
    expect(parseScanId("abc")).toBeUndefined();
  });
});
