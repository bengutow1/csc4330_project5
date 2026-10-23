import { Allergy, Diet, Equipment, type Kitchen, RoastLevel, Staple } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { ALLERGIES, DIETS, draftOf, EQUIPMENT, ROAST_LEVELS, STAPLES, toggled } from "./kitchen.ts";

// Every value of a wire enum but UNSPECIFIED, in number order. A numeric TS enum also maps each
// number back to its name, hence the filter.
function everyValue(set: Record<string, string | number>): number[] {
  return Object.values(set).filter((v): v is number => typeof v === "number" && v !== 0);
}

describe("the kitchen's choices", () => {
  // A value missing here could never be picked, and one out of order would jump on save, since
  // the server returns each list in the closed set's order.
  it.each([
    { name: "equipment", choices: EQUIPMENT, set: Equipment },
    { name: "diets", choices: DIETS, set: Diet },
    { name: "allergies", choices: ALLERGIES, set: Allergy },
    { name: "staples", choices: STAPLES, set: Staple },
    { name: "roast levels", choices: ROAST_LEVELS, set: RoastLevel },
  ])("lists every value of $name once, in the closed set's order", ({ choices, set }) => {
    expect(choices.map((c) => c.value)).toEqual(everyValue(set));
  });

  it("gives every roast level its own sample, with no em dash", () => {
    const samples = ROAST_LEVELS.map((level) => level.sample);
    expect(new Set(samples).size).toBe(ROAST_LEVELS.length);
    for (const sample of samples) {
      expect(sample).not.toContain("—");
    }
  });
});

describe("toggled", () => {
  it("adds a value that is out and takes out one that is in, leaving the input alone", () => {
    const list = [Staple.SALT, Staple.RICE];
    expect(toggled(list, Staple.OIL)).toEqual([Staple.SALT, Staple.RICE, Staple.OIL]);
    expect(toggled(list, Staple.SALT)).toEqual([Staple.RICE]);
    expect(list).toEqual([Staple.SALT, Staple.RICE]);
  });
});

describe("draftOf", () => {
  // Settings and onboarding save the draft whole, so a field it dropped would be switched off.
  it("opens on every field of the kitchen as stored, on lists of its own", () => {
    const kitchen: Kitchen = {
      $typeName: "account.v1.Kitchen",
      roastLevel: RoastLevel.EXTRA_CRISPY,
      equipment: [Equipment.OVEN],
      diet: Diet.VEGAN,
      allergies: [Allergy.PEANUTS, Allergy.SESAME],
      staples: [Staple.SALT],
    };

    const draft = draftOf(kitchen);

    expect(draft).toEqual({
      roastLevel: RoastLevel.EXTRA_CRISPY,
      equipment: [Equipment.OVEN],
      diet: Diet.VEGAN,
      allergies: [Allergy.PEANUTS, Allergy.SESAME],
      staples: [Staple.SALT],
    });
    expect(draft.equipment).not.toBe(kitchen.equipment);
  });

  it("opens empty before the kitchen has loaded", () => {
    expect(draftOf(undefined)).toEqual({
      roastLevel: undefined,
      equipment: [],
      diet: undefined,
      allergies: [],
      staples: [],
    });
  });
});
