import { Diet, Difficulty, Equipment, type RecipeFilters } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import {
  cookShowing,
  DIET_CHIPS,
  difficultyLabel,
  emptyLine,
  equipmentLabel,
  filtering,
  haveLabel,
  minutesLabel,
  NO_FILTERS,
  numbered,
  parseRecipeId,
  pointsLabel,
  recipeLabel,
  stepsLabel,
  TIME_LIMITS,
  withDiet,
  withEquipment,
  withTime,
} from "./recipes.ts";

describe("the chips", () => {
  it("offers the three time limits the server takes", () => {
    expect(TIME_LIMITS).toEqual([10, 20, 45]);
  });

  it("offers every diet but none, which is no chip at all", () => {
    expect(DIET_CHIPS.map((c) => c.label)).toEqual(["Vegetarian", "Vegan", "Pescatarian"]);
  });

  it("starts with every chip off", () => {
    expect(NO_FILTERS).toEqual({ maxMinutes: 0, equipment: [], diet: Diet.NONE });
    expect(filtering(NO_FILTERS)).toBe(false);
  });
});

describe("withTime", () => {
  it("turns a time chip on", () => {
    expect(withTime(NO_FILTERS, 20).maxMinutes).toBe(20);
  });

  it("moves to another time chip, since a recipe has one time", () => {
    expect(withTime(withTime(NO_FILTERS, 20), 45).maxMinutes).toBe(45);
  });

  it("turns the chip that is on off again", () => {
    expect(withTime(withTime(NO_FILTERS, 20), 20).maxMinutes).toBe(0);
  });

  it("leaves the other chips as they were", () => {
    const vegan = withDiet(NO_FILTERS, Diet.VEGAN);
    expect(withTime(vegan, 10)).toEqual({ ...vegan, maxMinutes: 10 });
  });
});

describe("withEquipment", () => {
  it("turns each piece of equipment on and off", () => {
    const oven = withEquipment(NO_FILTERS, Equipment.OVEN);
    expect(oven.equipment).toEqual([Equipment.OVEN]);
    expect(withEquipment(oven, Equipment.OVEN).equipment).toEqual([]);
  });

  it("keeps the closed set's order, so the same chips make the same query", () => {
    const ovenFirst = withEquipment(withEquipment(NO_FILTERS, Equipment.OVEN), Equipment.MICROWAVE);
    const microwaveFirst = withEquipment(
      withEquipment(NO_FILTERS, Equipment.MICROWAVE),
      Equipment.OVEN,
    );
    expect(ovenFirst.equipment).toEqual([Equipment.MICROWAVE, Equipment.OVEN]);
    expect(microwaveFirst.equipment).toEqual(ovenFirst.equipment);
  });

  it("never changes the filters it was given", () => {
    const filters: RecipeFilters = { ...NO_FILTERS, equipment: [Equipment.STOVETOP] };
    withEquipment(filters, Equipment.OVEN);
    withEquipment(filters, Equipment.STOVETOP);
    expect(filters.equipment).toEqual([Equipment.STOVETOP]);
  });
});

describe("withDiet", () => {
  it("turns a diet chip on, and moves to another", () => {
    const vegan = withDiet(NO_FILTERS, Diet.VEGAN);
    expect(vegan.diet).toBe(Diet.VEGAN);
    expect(withDiet(vegan, Diet.PESCATARIAN).diet).toBe(Diet.PESCATARIAN);
  });

  it("turns the diet that is on off again", () => {
    expect(withDiet(withDiet(NO_FILTERS, Diet.VEGAN), Diet.VEGAN).diet).toBe(Diet.NONE);
  });
});

describe("filtering", () => {
  it("says so when any one chip is on", () => {
    expect(filtering(withTime(NO_FILTERS, 10))).toBe(true);
    expect(filtering(withEquipment(NO_FILTERS, Equipment.MICROWAVE))).toBe(true);
    expect(filtering(withDiet(NO_FILTERS, Diet.VEGETARIAN))).toBe(true);
  });
});

describe("emptyLine", () => {
  it("blames the list when no chip is on", () => {
    expect(emptyLine(NO_FILTERS)).toBe("This list doesn't add up to a meal.");
  });

  it("points at the chips when one is on, since turning it off may find a meal", () => {
    expect(emptyLine(withTime(NO_FILTERS, 10))).toBe(
      "Nothing fits these filters. Turn one off, or get fries.",
    );
  });
});

describe("the card's words", () => {
  const omelette = {
    title: "Spinach Cheddar Omelette",
    minutes: 10,
    difficulty: Difficulty.EASY,
    equipment: [Equipment.STOVETOP],
    uses: ["Eggs", "Spinach", "Cheddar", "Butter", "Salt", "Pepper"],
    missing: [],
    pointsPreview: 100,
  };
  const eggOnToast = {
    title: "Sad Egg on Toast",
    minutes: 8,
    difficulty: Difficulty.EASY,
    equipment: [Equipment.STOVETOP],
    uses: ["Egg", "Butter", "Salt"],
    missing: ["Bread"],
    pointsPreview: 200,
  };

  it("counts what the player has of everything the recipe takes", () => {
    expect(haveLabel(omelette)).toBe("Have 6 of 6");
    expect(haveLabel(eggOnToast)).toBe("Have 3 of 4");
  });

  it("writes the time in minutes", () => {
    expect(minutesLabel(25)).toBe("25 min");
  });

  it("names each difficulty, and none the app doesn't know", () => {
    expect(difficultyLabel(Difficulty.EASY)).toBe("Easy");
    expect(difficultyLabel(Difficulty.MEDIUM)).toBe("Medium");
    expect(difficultyLabel(Difficulty.HARD)).toBe("Hard");
    expect(difficultyLabel(Difficulty.UNSPECIFIED)).toBeUndefined();
    // Proto enums are open, so a newer server can send one this app has never seen.
    expect(difficultyLabel(9 as Difficulty)).toBeUndefined();
  });

  it("names the equipment in the server's order, and says when there is none", () => {
    expect(equipmentLabel([Equipment.STOVETOP])).toBe("Stovetop");
    expect(equipmentLabel([Equipment.OVEN, Equipment.AIR_FRYER])).toBe("Oven, air fryer");
    expect(equipmentLabel([])).toBe("No equipment");
  });

  it("names no equipment it doesn't know, rather than claim none is needed", () => {
    expect(equipmentLabel([9 as Equipment, Equipment.OVEN])).toBe("Oven");
    expect(equipmentLabel([9 as Equipment])).toBeUndefined();
  });

  it("writes the points preview as the server sent it", () => {
    expect(pointsLabel(100)).toBe("+100 pts");
    expect(pointsLabel(200)).toBe("+200 pts");
  });

  it("counts the steps", () => {
    expect(stepsLabel(1)).toBe("1 step");
    expect(stepsLabel(4)).toBe("4 steps");
  });

  it("reads a card out whole, for a screen reader", () => {
    expect(recipeLabel(omelette)).toBe(
      "Spinach Cheddar Omelette, 10 minutes, Easy, Stovetop, Have 6 of 6, 100 points",
    );
    expect(recipeLabel({ ...eggOnToast, minutes: 1, difficulty: Difficulty.UNSPECIFIED })).toBe(
      "Sad Egg on Toast, 1 minute, Stovetop, Have 3 of 4, 200 points",
    );
  });
});

describe("numbered", () => {
  it("numbers a recipe's list from one, in the server's order, repeats and all", () => {
    expect(numbered(["Stir.", "Taste.", "Stir."])).toEqual([
      { n: 1, text: "Stir." },
      { n: 2, text: "Taste." },
      { n: 3, text: "Stir." },
    ]);
    expect(numbered([])).toEqual([]);
  });
});

describe("parseRecipeId", () => {
  it("reads a whole number above zero", () => {
    expect(parseRecipeId("7")).toBe(7n);
    expect(parseRecipeId("9007199254740993")).toBe(9007199254740993n);
  });

  it("reads anything else as no recipe, so the screen never asks about one that can't exist", () => {
    for (const param of [undefined, "", "0", "-3", "1.5", "7a", " 7", "abc"]) {
      expect(parseRecipeId(param)).toBeUndefined();
    }
  });
});

describe("cookShowing", () => {
  const shelves = { makeItNow: [], oneItemAway: [] };
  const settled = { notFound: false, failed: false };

  it("keeps an answer it has when a refresh fails, rather than trading it for an error page", () => {
    expect(cookShowing({ id: 7n, answer: shelves, notFound: false, failed: true })).toBe("answer");
    expect(cookShowing({ id: 7n, answer: shelves, notFound: true, failed: true })).toBe("answer");
  });

  it("shows the answer once it has one, empty shelves too", () => {
    expect(cookShowing({ id: 7n, answer: shelves, ...settled })).toBe("answer");
  });

  it("shows the error page only for an answer that never loaded", () => {
    expect(cookShowing({ id: 7n, answer: undefined, notFound: false, failed: true })).toBe("error");
  });

  it("says not found for one that isn't the player's, an id that can't be one, or no recipe", () => {
    expect(cookShowing({ id: 7n, answer: undefined, notFound: true, failed: true })).toBe(
      "not-found",
    );
    expect(cookShowing({ id: undefined, answer: undefined, ...settled })).toBe("not-found");
    expect(cookShowing({ id: 7n, answer: null, ...settled })).toBe("not-found");
  });

  it("is loading until the first answer", () => {
    expect(cookShowing({ id: 7n, answer: undefined, ...settled })).toBe("loading");
  });
});
