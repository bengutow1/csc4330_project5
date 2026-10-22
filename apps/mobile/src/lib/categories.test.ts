import { Category } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { categoryLook, groupByCategory } from "./categories.ts";
import type { Item } from "./ingredients.ts";

// The ten the server sends, in the order docs/contracts.md lists them.
const ALL = [
  Category.PROTEIN,
  Category.VEGETABLE,
  Category.FRUIT,
  Category.GRAIN,
  Category.DAIRY,
  Category.CONDIMENT,
  Category.DRINK,
  Category.SNACK,
  Category.LEFTOVER,
  Category.OTHER,
];

function item(name: string, category: Category): Item {
  return { name, category, quantity: "" };
}

describe("categoryLook", () => {
  it("names every category the server sends", () => {
    expect(ALL.map((category) => categoryLook(category).label)).toEqual([
      "Protein",
      "Vegetables",
      "Fruit",
      "Grains",
      "Dairy",
      "Condiments",
      "Drinks",
      "Snacks",
      "Leftovers",
      "Other",
    ]);
  });

  it("gives every category an icon of its own", () => {
    const icons = new Set(ALL.map((category) => categoryLook(category).icon));

    expect(icons.size).toBe(ALL.length);
  });

  it("files a category the app doesn't know under Other", () => {
    expect(categoryLook(Category.UNSPECIFIED)).toEqual(categoryLook(Category.OTHER));
    expect(categoryLook(11 as Category)).toEqual(categoryLook(Category.OTHER));
  });
});

describe("groupByCategory", () => {
  it("groups in the order each category first appears, keeping the server's order in each", () => {
    const list = [
      item("Chicken", Category.PROTEIN),
      item("Spinach", Category.VEGETABLE),
      item("Eggs", Category.PROTEIN),
      item("Bell peppers", Category.VEGETABLE),
      item("Cheddar", Category.DAIRY),
    ];

    expect(groupByCategory(list)).toEqual([
      {
        category: Category.PROTEIN,
        items: [
          { item: list[0], index: 0 },
          { item: list[2], index: 2 },
        ],
      },
      {
        category: Category.VEGETABLE,
        items: [
          { item: list[1], index: 1 },
          { item: list[3], index: 3 },
        ],
      },
      { category: Category.DAIRY, items: [{ item: list[4], index: 4 }] },
    ]);
  });

  it("puts a category the app doesn't know in the same group as Other", () => {
    const list = [item("Mystery container", Category.OTHER), item("Moon cheese", 11 as Category)];

    expect(groupByCategory(list)).toEqual([
      {
        category: Category.OTHER,
        items: [
          { item: list[0], index: 0 },
          { item: list[1], index: 1 },
        ],
      },
    ]);
  });

  it("has no groups for an empty list", () => {
    expect(groupByCategory([])).toEqual([]);
  });
});
