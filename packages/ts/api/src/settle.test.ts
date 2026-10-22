import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { settleAfter } from "./settle.ts";

describe("settleAfter", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("settles once, on the last value, when typing never pauses long enough", () => {
    const settled: string[] = [];
    const settle = settleAfter<string>(300, (value) => settled.push(value));

    for (const typed of ["t", "ta", "tac", "taco", "tacos"]) {
      settle.push(typed);
      vi.advanceTimersByTime(299);
    }
    expect(settled).toEqual([]);

    vi.advanceTimersByTime(1);
    expect(settled).toEqual(["tacos"]);
  });

  it("settles every value that is left alone long enough", () => {
    const settled: string[] = [];
    const settle = settleAfter<string>(300, (value) => settled.push(value));

    settle.push("fries");
    vi.advanceTimersByTime(300);
    settle.push("fries near campus");
    vi.advanceTimersByTime(300);

    expect(settled).toEqual(["fries", "fries near campus"]);
  });

  it("drops a value still waiting when cancelled", () => {
    const settled: string[] = [];
    const settle = settleAfter<string>(300, (value) => settled.push(value));

    settle.push("tacos");
    settle.cancel();
    vi.advanceTimersByTime(1000);

    expect(settled).toEqual([]);
  });
});
