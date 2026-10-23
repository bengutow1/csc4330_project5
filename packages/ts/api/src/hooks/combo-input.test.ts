import { skipToken } from "@connectrpc/connect-query";
import { Diet, Equipment } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { expect, it } from "vitest";
import { comboRecipesInput } from "./use-cook.ts";

it("waits for a combo and sends every filter without a solo scan id", () => {
  const filters = { maxMinutes: 20, equipment: [Equipment.STOVETOP], diet: Diet.VEGAN };
  expect(comboRecipesInput(undefined, filters)).toBe(skipToken);
  expect(comboRecipesInput(0n, filters)).toBe(skipToken);
  expect(comboRecipesInput(7n, filters)).toEqual({ comboId: 7n, ...filters });
});
