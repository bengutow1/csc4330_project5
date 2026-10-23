import { describe, expect, it } from "vitest";
import { remainingMilliseconds, timerLabel } from "./cooking-timer.ts";

describe("the cooking timer", () => {
  it("counts elapsed time rather than interval callbacks", () => {
    expect(remainingMilliseconds({ remainingMs: 6000, startedAt: 1000 }, 4500)).toBe(2500);
  });
  it("holds a paused timer still", () => {
    expect(remainingMilliseconds({ remainingMs: 2500 }, 90000)).toBe(2500);
  });
  it("stops at zero when a tick arrives late", () => {
    expect(remainingMilliseconds({ remainingMs: 2000, startedAt: 1000 }, 9999)).toBe(0);
  });
  it("does not add time if the clock moves backwards", () => {
    expect(remainingMilliseconds({ remainingMs: 2000, startedAt: 1000 }, 500)).toBe(2000);
  });
  it.each([
    { ms: 0, want: "0:00" },
    { ms: 1, want: "0:01" },
    { ms: 60000, want: "1:00" },
    { ms: 240001, want: "4:01" },
  ])("labels $ms ms as $want", ({ ms, want }) => {
    expect(timerLabel(ms)).toBe(want);
  });
});
