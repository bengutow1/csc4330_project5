import { describe, expect, it } from "vitest";
import {
  chips,
  countLine,
  fridgeLabel,
  mergedLine,
  openingStep,
  STEPS,
  sides,
  waitingLine,
} from "./combo.ts";

const BEN = { accountId: 1n, handle: "test_ben", scan: { score: 70, ingredients: [1, 2, 3] } };
const MAYA = { accountId: 2n, handle: "test_maya", scan: { score: 20, ingredients: [1] } };
const MAYA_UNSCANNED = { accountId: 2n, handle: "test_maya" };

describe("sides", () => {
  it("finds the caller by handle, whichever seat they hold", () => {
    expect(sides([BEN, MAYA], "test_maya")).toEqual({ me: MAYA, them: BEN });
    expect(sides([BEN, MAYA], "test_ben")).toEqual({ me: BEN, them: MAYA });
  });

  it("has no friend while the second seat is empty", () => {
    expect(sides([BEN], "test_ben")).toEqual({ me: BEN, them: undefined });
  });
});

describe("fridgeLabel", () => {
  it("says your fridge for the caller and names the friend's", () => {
    expect(fridgeLabel(BEN, true)).toBe("Your fridge");
    expect(fridgeLabel(MAYA, false)).toBe("test_maya's fridge");
  });
});

describe("countLine and mergedLine", () => {
  it("count by hand, singular for one", () => {
    expect(countLine(1)).toBe("1 item");
    expect(countLine(9)).toBe("9 items");
    expect(mergedLine(1)).toBe("Fridges merged. 1 ingredient between you.");
    expect(mergedLine(16)).toBe("Fridges merged. 16 ingredients between you.");
  });
});

describe("waitingLine", () => {
  it("waits for a friend, then the caller's scan, then the friend's", () => {
    expect(waitingLine(BEN, undefined)).toBe("Waiting for a friend to join…");
    expect(waitingLine({ accountId: 1n, handle: "test_ben" }, MAYA)).toBe(
      "Scan your fridge to see what you make together.",
    );
    expect(waitingLine(BEN, MAYA_UNSCANNED)).toBe("Waiting for test_maya to scan…");
  });
});

describe("chips", () => {
  it("marks each food by whose fridge it came from, in the server's order", () => {
    const owned = [
      { accountId: 1n, ingredient: { name: "Eggs" } },
      { accountId: 2n, ingredient: { name: "Bell peppers" } },
      { accountId: 1n, ingredient: { name: "Cheddar" } },
    ];
    expect(chips(owned, 1n).map((c) => `${c.name}:${c.mine ? "mine" : "theirs"}`)).toEqual([
      "Eggs:mine",
      "Bell peppers:theirs",
      "Cheddar:mine",
    ]);
  });

  it("keeps a food both fridges hold as two chips with keys of their own", () => {
    const both = chips(
      [
        { accountId: 1n, ingredient: { name: "Eggs" } },
        { accountId: 2n, ingredient: { name: "Eggs" } },
      ],
      1n,
    );
    expect(both.map((c) => c.mine)).toEqual([true, false]);
    expect(new Set(both.map((c) => c.key)).size).toBe(2);
  });
});

describe("openingStep", () => {
  it("plays every step on a phone that watched the combo wait", () => {
    expect(openingStep(true, false)).toBe(0);
  });

  it("shows the final state at once on a phone that opened it late", () => {
    expect(openingStep(false, false)).toBe(STEPS.combined);
  });

  it("shows the final state at once when the phone asks for reduced motion", () => {
    expect(openingStep(true, true)).toBe(STEPS.combined);
  });
});
