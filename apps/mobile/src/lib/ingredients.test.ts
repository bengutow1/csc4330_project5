import { Category } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import {
  COMMON,
  type Item,
  listErrors,
  listToSave,
  refusalOf,
  suggest,
  withIngredient,
  withoutIngredient,
} from "./ingredients.ts";

function item(name: string, category: Category = Category.OTHER): Item {
  return { name, category, quantity: "" };
}

describe("suggest", () => {
  it("offers the common ingredients whose name starts with what was typed", () => {
    expect(suggest("chick", []).map((s) => s.name)).toEqual([
      "Chicken",
      "Chicken breast",
      "Chickpeas",
    ]);
  });

  it("also offers a later word's match, after the names that start with it", () => {
    expect(suggest("pep", []).map((s) => s.name)).toEqual(["Pepperoni", "Bell peppers"]);
  });

  it("ignores case and the spaces around what was typed", () => {
    expect(suggest("  GREEK ", []).map((s) => s.name)).toEqual(["Greek yogurt"]);
  });

  it("offers nothing until something is typed", () => {
    expect(suggest("", [])).toEqual([]);
    expect(suggest("   ", [])).toEqual([]);
  });

  it("leaves out what the list already has, whatever its case", () => {
    expect(suggest("chick", [item("chicken")]).map((s) => s.name)).toEqual([
      "Chicken breast",
      "Chickpeas",
    ]);
  });

  it("offers five at most", () => {
    expect(suggest("s", [])).toHaveLength(5);
  });
});

describe("COMMON", () => {
  it("names each ingredient once, so a suggestion is never offered twice", () => {
    const names = COMMON.map((common) => common.name.toLowerCase());

    expect(new Set(names).size).toBe(names.length);
  });

  it("leaves the staples out: they are set once in the profile, never added to a scan", () => {
    const staples = [
      "salt",
      "pepper",
      "oil",
      "butter",
      "rice",
      "pasta",
      "flour",
      "sugar",
      "spices",
    ];
    const names = COMMON.map((common) => common.name.toLowerCase());

    expect(names.filter((name) => staples.includes(name))).toEqual([]);
  });
});

describe("withIngredient", () => {
  it("adds a common ingredient under its own category and spelling", () => {
    expect(withIngredient([], "greek yogurt")).toEqual([item("Greek yogurt", Category.DAIRY)]);
  });

  it("adds anything else under Other, as typed but trimmed", () => {
    expect(withIngredient([], "  Grandma's chili ")).toEqual([item("Grandma's chili")]);
  });

  it("puts the new one last", () => {
    expect(withIngredient([item("Eggs", Category.PROTEIN)], "Lemons")).toEqual([
      item("Eggs", Category.PROTEIN),
      item("Lemons", Category.FRUIT),
    ]);
  });

  it("leaves the list as it was for a name it already has, whatever its case", () => {
    const list = [item("Eggs", Category.PROTEIN)];

    expect(withIngredient(list, "EGGS")).toBe(list);
  });

  it("leaves the list as it was for an empty name", () => {
    const list = [item("Eggs", Category.PROTEIN)];

    expect(withIngredient(list, "   ")).toBe(list);
  });

  it("never changes the list it was given", () => {
    const list = [item("Eggs", Category.PROTEIN)];

    withIngredient(list, "Lemons");

    expect(list).toEqual([item("Eggs", Category.PROTEIN)]);
  });
});

describe("listToSave", () => {
  const draft = [item("Eggs", Category.PROTEIN)];

  it("sends a name still in the add field with the list, as return would have added it", () => {
    expect(listToSave(draft, " Kimchi ")).toEqual([item("Eggs", Category.PROTEIN), item("Kimchi")]);
  });

  it("sends the draft itself when the field adds nothing, so a refusal still lines up with it", () => {
    expect(listToSave(draft, "")).toBe(draft);
    expect(listToSave(draft, "eggs")).toBe(draft);
  });
});

describe("withoutIngredient", () => {
  it("takes out only the ingredient at that place", () => {
    const list = [item("Eggs"), item("Milk"), item("Lemons")];

    expect(withoutIngredient(list, 1)).toEqual([item("Eggs"), item("Lemons")]);
  });

  it("never changes the list it was given", () => {
    const list = [item("Eggs"), item("Milk")];

    withoutIngredient(list, 0);

    expect(list).toEqual([item("Eggs"), item("Milk")]);
  });
});

describe("listErrors", () => {
  // Keyed as the api's buildEdit keys them: the whole list, or one ingredient by its index.
  it("puts each message where it belongs: on the list, or on one ingredient by its place", () => {
    const errors = listErrors({
      ingredients: "ingredients must be at most 40 items",
      "ingredients[0].name": "name is required",
      "ingredients[2].name": "name must be at most 40 characters",
    });

    expect(errors.list).toBe("ingredients must be at most 40 items");
    expect(errors.items).toEqual(
      new Map([
        [0, "name is required"],
        [2, "name must be at most 40 characters"],
      ]),
    );
  });

  it("shows an ingredient's first problem when it has two", () => {
    const errors = listErrors({
      "ingredients[1].name": "name is required",
      "ingredients[1].quantity": "quantity must be at most 20 characters",
    });

    expect(errors.items).toEqual(new Map([[1, "name is required"]]));
  });

  it("is empty when nothing was refused", () => {
    expect(listErrors({})).toEqual({ list: undefined, items: new Map() });
  });
});

describe("refusalOf", () => {
  const sent = [item("Eggs", Category.PROTEIN), item("Milk", Category.DAIRY)];
  const refused = new Error("refused");

  it("is the refusal while the draft is still the list that was sent", () => {
    expect(refusalOf(sent, sent, refused)).toBe(refused);
  });

  it("is null once the draft changed, since the refusal names places in the list that was sent", () => {
    expect(refusalOf(withoutIngredient(sent, 0), sent, refused)).toBeNull();
    expect(refusalOf(withIngredient(sent, "Lemons"), sent, refused)).toBeNull();
  });

  it("keeps the refusal when an add changed nothing, as a name already on the list does", () => {
    expect(refusalOf(withIngredient(sent, "eggs"), sent, refused)).toBe(refused);
  });

  it("is null before anything was sent", () => {
    expect(refusalOf(sent, null, refused)).toBeNull();
  });
});
