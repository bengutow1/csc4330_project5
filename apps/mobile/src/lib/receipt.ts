import { PointReason } from "@fries-in-the-bag/api";
import type { PointsLine } from "../components/points-breakdown.tsx";

/** One award as the receipt carries it: why, and what it paid. */
export interface Award {
  reason: PointReason;
  points: number;
}

/** A rank as the receipt carries it: the title, the next one ("" at the top) and the way there. */
export interface RankStep {
  name: string;
  next: string;
  toNext: number;
}

/**
 * awardLines is a receipt's awards in words, in the server's order: screen 6's "Home-cooked meal",
 * and the streak as its days beside a flame. A reason newer than the app still shows what it paid.
 */
export function awardLines(awards: readonly Award[], streakDays: number): PointsLine[] {
  return awards.map((award) => ({ ...words(award.reason, streakDays), points: award.points }));
}

function words(reason: PointReason, streakDays: number): Omit<PointsLine, "points"> {
  switch (reason) {
    case PointReason.COOK:
      return { label: "Home-cooked meal" };
    case PointReason.SCRAPPY_BONUS:
      return { label: "Scrappy bonus" };
    case PointReason.AGAINST_ALL_ODDS:
      return { label: "Against all odds" };
    case PointReason.STREAK:
      return { label: streakLabel(streakDays), icon: "flame-outline" };
    case PointReason.COMBO_COOK:
      return { label: "Combo cook" };
    default:
      return { label: "Bonus" };
  }
}

function streakLabel(days: number) {
  return `${days}-day streak`;
}

/**
 * streakLine is the streak on a line of its own, for a receipt whose awards don't carry it: only
 * the day's first plate pays the streak. It is null when an award carries it, or there is none.
 */
export function streakLine(awards: readonly Award[], streakDays: number): string | null {
  if (streakDays <= 0 || awards.some((award) => award.reason === PointReason.STREAK)) {
    return null;
  }
  return streakLabel(streakDays);
}

/**
 * rankWords is what the rank card says: the new title and the one it replaced, or, when the title
 * held, how far the next one is.
 */
export function rankWords(
  before: RankStep | undefined,
  after: RankStep,
): { title: string; line: string } {
  if (before !== undefined && before.name !== after.name) {
    return {
      title: `You're ${article(after.name)} ${after.name} now`,
      line: `Up from ${before.name}`,
    };
  }
  if (after.next === "") {
    return { title: after.name, line: "The top rank" };
  }
  const points = after.toNext === 1 ? "point" : "points";
  return { title: after.name, line: `${grouped(after.toNext)} ${points} to ${after.next}` };
}

function article(name: string) {
  return /^[aeiou]/i.test(name) ? "an" : "a";
}

/** grouped writes a whole number with its thousands marked: 1,500. */
function grouped(n: number) {
  return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/**
 * takenCaption says when a plate photo was taken, as the photo card does: "Tonight, 7:58 PM".
 * Morning runs from 5 am to noon and the afternoon to 5 pm, and the rest is tonight.
 */
export function takenCaption(at: Date): string {
  const hour = at.getHours();
  let part = "Tonight";
  if (hour >= 5 && hour < 12) {
    part = "This morning";
  } else if (hour >= 12 && hour < 17) {
    part = "This afternoon";
  }
  const minutes = String(at.getMinutes()).padStart(2, "0");
  return `${part}, ${hour % 12 || 12}:${minutes} ${hour < 12 ? "AM" : "PM"}`;
}
