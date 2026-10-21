import { createConnectQueryKey, skipToken } from "@connectrpc/connect-query";
import { Diet, Equipment } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { partialMatchKey } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { RECIPES, type RecipeFilters, recipeInput, recipesInput } from "./use-cook.ts";

const ANY: RecipeFilters = { maxMinutes: 0, equipment: [], diet: Diet.NONE };

describe("recipesInput", () => {
  it("waits until there is a scan to ask about", () => {
    expect(recipesInput(undefined, ANY)).toBe(skipToken);
    expect(recipesInput(0n, ANY)).toBe(skipToken);
  });

  it("sends every filter, so the server narrows the shelves", () => {
    const quick: RecipeFilters = {
      maxMinutes: 10,
      equipment: [Equipment.MICROWAVE],
      diet: Diet.VEGAN,
    };
    expect(recipesInput(42n, quick)).toEqual({
      scanId: 42n,
      maxMinutes: 10,
      equipment: [Equipment.MICROWAVE],
      diet: Diet.VEGAN,
    });
  });
});

describe("recipeInput", () => {
  it("waits until there is a recipe to ask about", () => {
    expect(recipeInput(undefined)).toBe(skipToken);
    expect(recipeInput(0n)).toBe(skipToken);
  });

  it("asks for a real recipe by its id", () => {
    expect(recipeInput(7n)).toEqual({ recipeId: 7n });
  });
});

describe("RECIPES", () => {
  // Built as the hooks build them, through a real transport, so a key the family missed would
  // show here rather than as shelves that never refresh.
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });
  const shelvesKey = (filters: RecipeFilters) =>
    createConnectQueryKey({
      schema: CookService.method.listRecipes,
      input: recipesInput(42n, filters),
      transport,
      cardinality: "finite",
    });

  it("covers every recipe read, so one invalidation refreshes them all", () => {
    const oneRecipe = createConnectQueryKey({
      schema: CookService.method.getRecipe,
      input: recipeInput(7n),
      transport,
      cardinality: "finite",
    });
    expect(partialMatchKey(shelvesKey(ANY), RECIPES)).toBe(true);
    expect(partialMatchKey(oneRecipe, RECIPES)).toBe(true);
  });

  it("keeps each filter's shelves apart", () => {
    const vegan = shelvesKey({ ...ANY, diet: Diet.VEGAN });
    expect(vegan).not.toEqual(shelvesKey(ANY));
  });

  it("leaves other services' reads alone", () => {
    const lastScan = createConnectQueryKey({
      schema: ScanService.method.getLastScan,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(partialMatchKey(lastScan, RECIPES)).toBe(false);
  });
});
