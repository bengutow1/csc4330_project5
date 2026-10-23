import { create } from "@bufbuild/protobuf";
import { GetBoardResponseSchema } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { describe, expect, it } from "vitest";
import { nextBoardCursor } from "./use-board.ts";

describe("nextBoardCursor", () => {
  it("follows the server's cursor while there is more", () => {
    const page = create(GetBoardResponseSchema, { nextCursor: "MzpjaGVmX21heWE" });
    expect(nextBoardCursor(page)).toBe("MzpjaGVmX21heWE");
  });

  it("ends the board at an empty cursor", () => {
    expect(nextBoardCursor(create(GetBoardResponseSchema, {}))).toBeUndefined();
  });
});
