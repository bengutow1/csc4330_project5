import { describe, expect, it } from "vitest";
import { groupDigits } from "./format.ts";

describe("groupDigits", () => {
  it.each([
    { value: 0n, want: "0" },
    { value: 980n, want: "980" },
    { value: 1240n, want: "1,240" },
    { value: 1234567n, want: "1,234,567" },
  ])("writes $value as $want", ({ value, want }) => {
    expect(groupDigits(value)).toBe(want);
  });
});
