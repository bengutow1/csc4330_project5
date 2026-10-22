import { describe, expect, it } from "vitest";
import { parseId } from "./route-id.ts";

describe("parseId", () => {
  it("reads an id from its route", () => {
    expect(parseId("42")).toBe(42n);
  });

  it("reads an id too big for a JavaScript number", () => {
    expect(parseId("9007199254740993")).toBe(9007199254740993n);
  });

  it("is undefined for anything that can't name a row, so nothing is asked for", () => {
    expect(parseId(undefined)).toBeUndefined();
    expect(parseId("")).toBeUndefined();
    expect(parseId("0")).toBeUndefined();
    expect(parseId("-3")).toBeUndefined();
    expect(parseId("1.5")).toBeUndefined();
    expect(parseId("abc")).toBeUndefined();
  });
});
