import { PointReason } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import { awardLines, rankWords, streakLine, takenCaption } from "./receipt.ts";

describe("awardLines", () => {
  it("words each award as the receipt shows it, keeping what it paid", () => {
    const cases = [
      { reason: PointReason.COOK, points: 100, line: { label: "Home-cooked meal", points: 100 } },
      {
        reason: PointReason.SCRAPPY_BONUS,
        points: 50,
        line: { label: "Scrappy bonus", points: 50 },
      },
      {
        reason: PointReason.AGAINST_ALL_ODDS,
        points: 100,
        line: { label: "Against all odds", points: 100 },
      },
      { reason: PointReason.COMBO_COOK, points: 50, line: { label: "Combo cook", points: 50 } },
    ];

    for (const { reason, points, line } of cases) {
      expect(awardLines([{ reason, points }], 1)).toEqual([line]);
    }
  });

  it("names the streak by its days, beside the flame", () => {
    expect(awardLines([{ reason: PointReason.STREAK, points: 40 }], 4)).toEqual([
      { label: "4-day streak", points: 40, icon: "flame-outline" },
    ]);
  });

  it("keeps the server's order", () => {
    const lines = awardLines(
      [
        { reason: PointReason.COOK, points: 100 },
        { reason: PointReason.SCRAPPY_BONUS, points: 50 },
        { reason: PointReason.STREAK, points: 10 },
      ],
      1,
    );

    expect(lines.map((line) => line.label)).toEqual([
      "Home-cooked meal",
      "Scrappy bonus",
      "1-day streak",
    ]);
  });

  it("still shows what a reason newer than the app paid", () => {
    expect(awardLines([{ reason: 9 as PointReason, points: 25 }], 1)).toEqual([
      { label: "Bonus", points: 25 },
    ]);
    expect(awardLines([{ reason: PointReason.UNSPECIFIED, points: 5 }], 1)).toEqual([
      { label: "Bonus", points: 5 },
    ]);
  });
});

describe("streakLine", () => {
  it("shows the streak on its own when no award carries it, as on a second plate that day", () => {
    expect(streakLine([{ reason: PointReason.COOK, points: 100 }], 3)).toBe("3-day streak");
  });

  it("is null when an award carries the streak, so it isn't said twice", () => {
    const awards = [
      { reason: PointReason.COOK, points: 100 },
      { reason: PointReason.STREAK, points: 30 },
    ];

    expect(streakLine(awards, 3)).toBeNull();
  });

  it("is null with no streak to show", () => {
    expect(streakLine([], 0)).toBeNull();
  });
});

describe("rankWords", () => {
  const fryCook = { name: "Fry Cook", next: "Line Cook", toNext: 340 };

  it("names a new title and the one it replaced", () => {
    const lineCook = { name: "Line Cook", next: "Sous Chef", toNext: 960 };

    expect(rankWords(fryCook, lineCook)).toEqual({
      title: "You're a Line Cook now",
      line: "Up from Fry Cook",
    });
  });

  it("says an before a title that starts with a vowel", () => {
    const headChef = { name: "Head Chef", next: "Executive Chef", toNext: 20 };
    const executiveChef = { name: "Executive Chef", next: "", toNext: 0 };

    expect(rankWords(headChef, executiveChef).title).toBe("You're an Executive Chef now");
  });

  it("says how far the next title is when the title held", () => {
    expect(rankWords(fryCook, fryCook)).toEqual({
      title: "Fry Cook",
      line: "340 points to Line Cook",
    });
  });

  it("marks the thousands, and says point for one", () => {
    const sousChef = { name: "Sous Chef", next: "Head Chef", toNext: 2500 };
    const nearly = { name: "Fry Cook", next: "Line Cook", toNext: 1 };

    expect(rankWords(sousChef, sousChef).line).toBe("2,500 points to Head Chef");
    expect(rankWords(nearly, nearly).line).toBe("1 point to Line Cook");
  });

  it("says the top title is the top", () => {
    const executiveChef = { name: "Executive Chef", next: "", toNext: 0 };

    expect(rankWords(executiveChef, executiveChef)).toEqual({
      title: "Executive Chef",
      line: "The top rank",
    });
  });

  it("reads a receipt with no title before as one whose title held", () => {
    expect(rankWords(undefined, fryCook)).toEqual({
      title: "Fry Cook",
      line: "340 points to Line Cook",
    });
  });
});

describe("takenCaption", () => {
  // Local times on one day, as the phone's clock reads them.
  const at = (hour: number, minute: number) => new Date(2026, 9, 21, hour, minute);

  it("says the part of the day and the time on the clock", () => {
    expect(takenCaption(at(19, 58))).toBe("Tonight, 7:58 PM");
    expect(takenCaption(at(8, 5))).toBe("This morning, 8:05 AM");
    expect(takenCaption(at(13, 30))).toBe("This afternoon, 1:30 PM");
  });

  it("starts the morning at 5, the afternoon at noon and the night at 5 pm", () => {
    expect(takenCaption(at(4, 59))).toBe("Tonight, 4:59 AM");
    expect(takenCaption(at(5, 0))).toBe("This morning, 5:00 AM");
    expect(takenCaption(at(11, 59))).toBe("This morning, 11:59 AM");
    expect(takenCaption(at(12, 0))).toBe("This afternoon, 12:00 PM");
    expect(takenCaption(at(16, 59))).toBe("This afternoon, 4:59 PM");
    expect(takenCaption(at(17, 0))).toBe("Tonight, 5:00 PM");
  });

  it("calls the hour after midnight 12, not 0", () => {
    expect(takenCaption(at(0, 30))).toBe("Tonight, 12:30 AM");
  });
});
