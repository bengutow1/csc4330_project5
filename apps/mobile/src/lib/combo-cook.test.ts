import { describe, expect, it } from "vitest";
import { bringItems } from "./combo-cook";

describe("bringItems", () => {
  const owners = [
    { accountId: 1n, ingredient: "Eggs" },
    { accountId: 2n, ingredient: "Peas" },
    { accountId: 1n, ingredient: "Rice" },
    { accountId: 2n, ingredient: "Rice" },
  ];
  it("asks the guest to bring only food the chosen kitchen lacks", () => {
    expect(bringItems(owners, 2n, 1n)).toEqual(["Peas"]);
    expect(bringItems(owners, 1n, 2n)).toEqual(["Eggs"]);
  });
  it("gives the host and an unknown player no packing list", () => {
    expect(bringItems(owners, 1n, 1n)).toEqual([]);
    expect(bringItems(owners, undefined, 1n)).toEqual([]);
    expect(bringItems(owners, 2n, undefined)).toEqual([]);
  });
});
