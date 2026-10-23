import { describe, expect, it } from "vitest";
import { badgeLook, base64, earnedLine, percent, platedOn, rankLine, toDate } from "./profile.ts";

describe("rankLine", () => {
  it("says how far to the next title, as screen 10 writes it", () => {
    expect(rankLine(640n, { next: "Sous Chef", toNext: 860 })).toBe(
      "640 of 1,500 points to Sous Chef",
    );
  });

  it("starts a newcomer at nothing of the first title's points", () => {
    expect(rankLine(0n, { next: "Line Cook", toNext: 500 })).toBe("0 of 500 points to Line Cook");
  });

  it("says the top title is the top", () => {
    expect(rankLine(12000n, { next: "", toNext: 0 })).toBe("12,000 points, the top title");
  });
});

describe("percent", () => {
  it.each([
    { ratio: 0.2, want: "20%" },
    { ratio: 0, want: "0%" },
    { ratio: 1, want: "100%" },
    { ratio: 1 / 3, want: "33%" },
    { ratio: 2 / 3, want: "67%" },
  ])("writes $ratio as $want", ({ ratio, want }) => {
    expect(percent(ratio)).toBe(want);
  });
});

describe("earnedLine", () => {
  it("counts the earned badges out of all of them", () => {
    const badges = [{ earned: true }, { earned: false }, { earned: true }, { earned: false }];
    expect(earnedLine(badges)).toBe("2 of 4");
  });
});

describe("badgeLook", () => {
  it("gives screen 10's badges their icons", () => {
    expect(badgeLook("On the Board").icon).toBe("trophy-outline");
    expect(badgeLook("Combo Meal").icon).toBe("people-outline");
    expect(badgeLook("Three's a Meal").icon).toBe("restaurant-outline");
  });

  it("gives a badge this app has not heard of a plain ribbon, rather than nothing", () => {
    expect(badgeLook("A Badge From The Future").icon).toBe("ribbon-outline");
  });
});

describe("base64", () => {
  it.each([
    { bytes: [], want: "" },
    { bytes: [0x4d], want: "TQ==" },
    { bytes: [0x4d, 0x61], want: "TWE=" },
    { bytes: [0x4d, 0x61, 0x6e], want: "TWFu" },
    // A JPEG's first bytes, which every plate photo starts with.
    { bytes: [0xff, 0xd8, 0xff, 0xe0], want: "/9j/4A==" },
  ])("writes $bytes as $want", ({ bytes, want }) => {
    expect(base64(new Uint8Array(bytes))).toBe(want);
  });
});

describe("platedOn", () => {
  it("writes the day a plate was plated, in the phone's own time", () => {
    expect(platedOn(new Date(2026, 9, 21, 19, 30))).toBe("Oct 21");
    expect(platedOn(new Date(2027, 0, 2, 8, 0))).toBe("Jan 2");
  });
});

describe("toDate", () => {
  it("reads a wire timestamp, nanoseconds down to the millisecond", () => {
    const at = toDate({ seconds: 1792602000n, nanos: 123456789 });
    expect(at.getTime()).toBe(1792602000123);
  });
});
