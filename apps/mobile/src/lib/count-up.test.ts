import { describe, expect, it } from "vitest";
import { countAt } from "./count-up.ts";

describe("countAt", () => {
  it("starts on the number it counts from", () => {
    expect(countAt(0, 140, 0, 900)).toBe(0);
    expect(countAt(25, 140, 0, 900)).toBe(25);
  });

  it("lands exactly on the value once the time is up, and stays there", () => {
    expect(countAt(0, 140, 900, 900)).toBe(140);
    expect(countAt(0, 140, 5000, 900)).toBe(140);
  });

  it("covers most of the way early and slows into the value", () => {
    // Halfway through the time, an ease-out has covered 87.5% of the way.
    expect(countAt(0, 80, 450, 900)).toBe(70);
    expect(countAt(0, 80, 800, 900)).toBe(80);
  });

  it("shows only whole numbers, never going back or past the value", () => {
    let last = 0;
    for (let elapsed = 0; elapsed <= 900; elapsed += 10) {
      const shown = countAt(0, 79, elapsed, 900);
      expect(Number.isInteger(shown)).toBe(true);
      expect(shown).toBeGreaterThanOrEqual(last);
      expect(shown).toBeLessThanOrEqual(79);
      last = shown;
    }
  });

  it("counts down when the value is lower", () => {
    expect(countAt(200, 0, 0, 900)).toBe(200);
    expect(countAt(200, 0, 450, 900)).toBe(25);
    expect(countAt(200, 0, 900, 900)).toBe(0);
  });

  it("shows the value at once when the count takes no time", () => {
    expect(countAt(0, 79, 0, 0)).toBe(79);
  });

  it("holds the starting number when the clock reads before the start", () => {
    expect(countAt(10, 79, -16, 900)).toBe(10);
  });
});
