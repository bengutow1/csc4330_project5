import { skipToken } from "@connectrpc/connect-query";
import { describe, expect, it } from "vitest";
import { checkHandleInput } from "./use-account.ts";

describe("checkHandleInput", () => {
  it("waits until the handle is long enough for the server to take", () => {
    expect(checkHandleInput("")).toBe(skipToken);
    expect(checkHandleInput("ab")).toBe(skipToken);
    expect(checkHandleInput("  ab  ")).toBe(skipToken);
  });

  it("asks from three characters on, sending what was typed for the server to normalize", () => {
    expect(checkHandleInput("abc")).toEqual({ handle: "abc" });
    expect(checkHandleInput(" Chef_Maya ")).toEqual({ handle: " Chef_Maya " });
  });
});
