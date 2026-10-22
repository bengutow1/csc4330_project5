import type { IconName } from "@/components/icon-button";
import { groupDigits } from "./format.ts";

/** rankLine says how far a player is from the next title, as screen 10 writes it. */
export function rankLine(points: bigint, rank: { next: string; toNext: number }) {
  if (rank.next === "") {
    return `${groupDigits(points)} points, the top title`;
  }
  return `${groupDigits(points)} of ${groupDigits(points + BigInt(rank.toNext))} points to ${rank.next}`;
}

/** percent writes a share from 0 to 1 as a whole percentage ("20%"). */
export function percent(ratio: number) {
  return `${Math.round(ratio * 100)}%`;
}

/** earnedLine counts the earned badges out of all of them ("3 of 11"). */
export function earnedLine(badges: readonly { earned: boolean }[]) {
  return `${badges.filter((badge) => badge.earned).length} of ${badges.length}`;
}

/** Tone names the tint a badge's circle wears; the screen turns it into colours. */
export type Tone = "mustard" | "blueberry" | "pickle" | "ketchup";

// Each badge's icon and tint, by the name the server sends. Screen 10 draws the first three.
const BADGE_LOOKS: Record<string, { icon: IconName; tone: Tone }> = {
  "First Roast": { icon: "flame-outline", tone: "ketchup" },
  "Certified Chef": { icon: "ribbon-outline", tone: "mustard" },
  "On the Board": { icon: "trophy-outline", tone: "mustard" },
  Underdog: { icon: "trending-up-outline", tone: "pickle" },
  "Three's a Meal": { icon: "restaurant-outline", tone: "pickle" },
  "Combo Meal": { icon: "people-outline", tone: "blueberry" },
  "Clean Plate Club": { icon: "sparkles-outline", tone: "pickle" },
  "Late Night Legend": { icon: "moon-outline", tone: "blueberry" },
  "Glow Up": { icon: "trending-up-outline", tone: "ketchup" },
  "Microwave Maestro": { icon: "timer-outline", tone: "blueberry" },
  "Bag Secured": { icon: "bag-handle-outline", tone: "ketchup" },
};

/** badgeLook is a badge's icon and tint; a badge newer than this app gets a plain ribbon. */
export function badgeLook(name: string): { icon: IconName; tone: Tone } {
  return BADGE_LOOKS[name] ?? { icon: "ribbon-outline", tone: "mustard" };
}

const BASE64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

/** base64 encodes bytes, so a plate's photo can be drawn from a data URI. */
export function base64(bytes: Uint8Array) {
  let out = "";
  for (let i = 0; i < bytes.length; i += 3) {
    const a = bytes[i] ?? 0;
    const b = bytes[i + 1] ?? 0;
    const c = bytes[i + 2] ?? 0;
    const n = (a << 16) | (b << 8) | c;
    out += BASE64[(n >> 18) & 63];
    out += BASE64[(n >> 12) & 63];
    out += i + 1 < bytes.length ? BASE64[(n >> 6) & 63] : "=";
    out += i + 2 < bytes.length ? BASE64[n & 63] : "=";
  }
  return out;
}

/** photoUri is a plate's JPEG as a URI expo-image draws. */
export function photoUri(photo: Uint8Array) {
  return `data:image/jpeg;base64,${base64(photo)}`;
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/** platedOn is the day a plate was plated, in the phone's own time ("Oct 21"). */
export function platedOn(at: Date) {
  return `${MONTHS[at.getMonth()]} ${at.getDate()}`;
}

/** toDate reads a wire timestamp (a plate's plated_at) as a Date, to the millisecond. */
export function toDate(at: { seconds: bigint; nanos: number }) {
  return new Date(Number(at.seconds) * 1000 + Math.floor(at.nanos / 1_000_000));
}
