import { Tier } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { buttonsFor, parseScanId, showing } from "./verdict.ts";

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

describe("showing", () => {
  const answer = { verdict: {} };
  const settled = { notFound: false, failed: false };

  it("keeps a verdict it has when a refresh fails, rather than trading it for an error page", () => {
    expect(showing({ id: 42n, data: answer, notFound: false, failed: true })).toBe("verdict");
    expect(showing({ id: 42n, data: answer, notFound: true, failed: true })).toBe("verdict");
  });

  it("shows the verdict once it has one", () => {
    expect(showing({ id: 42n, data: answer, ...settled })).toBe("verdict");
  });

  it("shows the error page only for a verdict that never loaded", () => {
    expect(showing({ id: 42n, data: undefined, notFound: false, failed: true })).toBe("error");
  });

  it("says not found for a scan that isn't the player's, an id that can't be one, or no verdict", () => {
    expect(showing({ id: 42n, data: undefined, notFound: true, failed: true })).toBe("not-found");
    expect(showing({ id: undefined, data: undefined, ...settled })).toBe("not-found");
    expect(showing({ id: 42n, data: {}, ...settled })).toBe("not-found");
  });

  it("is loading until the first answer", () => {
    expect(showing({ id: 42n, data: undefined, ...settled })).toBe("loading");
  });
});
