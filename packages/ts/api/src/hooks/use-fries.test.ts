import { create } from "@bufbuild/protobuf";
import { skipToken } from "@connectrpc/connect-query";
import { ListSpotsResponseSchema, SpotFilter } from "@fries-in-the-bag/apigen/fries/v1/fries_pb";
import { describe, expect, it } from "vitest";
import { nextSpotsCursor, spotsInput } from "./use-fries.ts";

describe("spotsInput", () => {
  it("waits while there is neither a point nor words to search by", () => {
    expect(spotsInput({ filters: [] })).toBe(skipToken);
    expect(spotsInput({ query: "   ", filters: [SpotFilter.OPEN_NOW] })).toBe(skipToken);
  });

  it("asks around the player's point, from the first page", () => {
    expect(
      spotsInput({
        at: { lat: 30.4133, lng: -91.18 },
        filters: [SpotFilter.OPEN_NOW, SpotFilter.CHEAP],
      }),
    ).toEqual({
      lat: 30.4133,
      lng: -91.18,
      query: "",
      filters: [SpotFilter.OPEN_NOW, SpotFilter.CHEAP],
      cursor: "",
    });
  });

  it("searches by the trimmed words alone when location is off", () => {
    expect(spotsInput({ query: "  tacos near the stadium ", filters: [] })).toEqual({
      lat: undefined,
      lng: undefined,
      query: "tacos near the stadium",
      filters: [],
      cursor: "",
    });
  });
});

describe("nextSpotsCursor", () => {
  it("follows the server's cursor while there is more", () => {
    const page = create(ListSpotsResponseSchema, { nextCursor: "MjQwMDptb2NrLTA2" });
    expect(nextSpotsCursor(page)).toBe("MjQwMDptb2NrLTA2");
  });

  it("ends the list at an empty cursor", () => {
    expect(nextSpotsCursor(create(ListSpotsResponseSchema, {}))).toBeUndefined();
  });
});
