import { SpotFilter } from "@fries-in-the-bag/api";

/** The chips on the spots screen, in screen 9's order (Late night joins it from #27). */
export const SPOT_CHIPS = [
  { filter: SpotFilter.OPEN_NOW, label: "Open now" },
  { filter: SpotFilter.LATE_NIGHT, label: "Late night" },
  { filter: SpotFilter.CHEAP, label: "Under $10" },
  { filter: SpotFilter.FAST_FOOD, label: "Fast food" },
  { filter: SpotFilter.WALKABLE, label: "Walkable" },
] as const;

/** The chips a search starts with: Open now is on, as the concept asks. */
export const STARTING_FILTERS: readonly SpotFilter[] = [SpotFilter.OPEN_NOW];

/** QUERY_MAX is the longest search ListSpots takes (its max=60 on query): keep the two in step. */
export const QUERY_MAX = 60;

// Screen 9's times put a car at about 400 m a minute in town and a walker at 80.
const DRIVE_METERS_PER_MINUTE = 400;
const WALK_METERS_PER_MINUTE = 80;

/** distanceText writes a distance as screen 9 does ("650 m", "2.3 km"); "" when there is none. */
export function distanceText(meters: number) {
  if (meters <= 0) {
    return "";
  }
  if (meters < 1000) {
    return `${meters} m`;
  }
  // One decimal, without a trailing ".0".
  return `${Number((meters / 1000).toFixed(1))} km`;
}

/** priceText is the price level in dollar signs; "" when the provider does not say. */
export function priceText(level: number) {
  return "$".repeat(Math.max(0, level));
}

/** metaLine is a spot's cuisine, price and distance ("Mexican · $ · 2.9 km"), leaving out what it lacks. */
export function metaLine(spot: { category: string; priceLevel: number; distanceMeters: number }) {
  return [spot.category, priceText(spot.priceLevel), distanceText(spot.distanceMeters)]
    .filter((part) => part !== "")
    .join(" · ");
}

/** clockText reads the server's 24-hour "HH:MM" as a person says it ("2 AM", "9:30 PM", "midnight"). */
export function clockText(at: string) {
  const [hours, minutes] = at.split(":").map(Number);
  if (
    hours === undefined ||
    minutes === undefined ||
    Number.isNaN(hours) ||
    Number.isNaN(minutes)
  ) {
    return "";
  }
  if (minutes === 0 && hours === 0) {
    return "midnight";
  }
  if (minutes === 0 && hours === 12) {
    return "noon";
  }
  const half = hours < 12 ? "AM" : "PM";
  const hour = hours % 12 === 0 ? 12 : hours % 12;
  const time = minutes === 0 ? `${hour}` : `${hour}:${String(minutes).padStart(2, "0")}`;
  return `${time} ${half}`;
}

/** openLine is whether a spot is open and until when, as screen 9 writes it in pickle green. */
export function openLine(spot: { openNow: boolean; closesAt: string }) {
  if (!spot.openNow) {
    return { text: "Closed now", open: false };
  }
  const until = clockText(spot.closesAt);
  return { text: until === "" ? "Open now" : `Open until ${until}`, open: true };
}

/** travelText is how long the trip takes: on foot to a walkable spot, by car otherwise; "" with no distance. */
export function travelText(spot: { distanceMeters: number; walkable: boolean }) {
  if (spot.distanceMeters <= 0) {
    return "";
  }
  const pace = spot.walkable ? WALK_METERS_PER_MINUTE : DRIVE_METERS_PER_MINUTE;
  const minutes = Math.max(1, Math.ceil(spot.distanceMeters / pace));
  return spot.walkable ? `${minutes} min walk` : `${minutes} min`;
}

/**
 * directionsUrl hands the trip to the phone's own maps: Apple Maps on iOS, Google Maps elsewhere,
 * walking to a walkable spot and driving to the rest.
 */
export function directionsUrl(spot: { lat: number; lng: number; walkable: boolean }, os: string) {
  const to = `${spot.lat},${spot.lng}`;
  if (os === "ios") {
    return `https://maps.apple.com/?daddr=${to}&dirflg=${spot.walkable ? "w" : "d"}`;
  }
  const mode = spot.walkable ? "walking" : "driving";
  return `https://www.google.com/maps/dir/?api=1&destination=${to}&travelmode=${mode}`;
}

/** toggleFilter is the chips with one flipped, in the chips' order, so equal searches send equal requests. */
export function toggleFilter(on: readonly SpotFilter[], filter: SpotFilter) {
  const next = on.includes(filter) ? on.filter((f) => f !== filter) : [...on, filter];
  return SPOT_CHIPS.map((chip) => chip.filter).filter((f) => next.includes(f));
}

/** sendableFilters drops Walkable without a point: the server measures from one and refuses it otherwise. */
export function sendableFilters(on: readonly SpotFilter[], hasPoint: boolean) {
  return hasPoint ? [...on] : on.filter((f) => f !== SpotFilter.WALKABLE);
}

/** OffAction is what the list's location-off button does. */
export type OffAction = "look again" | "ask" | "settings";

/**
 * offAction says what the list's location-off button does: look again when the app may already
 * look, since then it was the phone that could not tell where it was; ask while the player can
 * still be asked; and send them to settings once they have said no for good.
 */
export function offAction(place: { granted: boolean; canAsk: boolean }): OffAction {
  if (place.granted) {
    return "look again";
  }
  return place.canAsk ? "ask" : "settings";
}

/**
 * emptyReason says which empty state an empty list shows, from the chips and words it sent. Typed
 * words that found nothing are the search's to answer for, chips or not. Without words, the hour
 * is to blame only when Open now is all that narrowed the list ("Even the fries are closed.");
 * otherwise the chips are, with a way to clear them.
 */
export function emptyReason(on: readonly SpotFilter[], words: string) {
  if (words.trim() !== "") {
    return "searched";
  }
  return on.length === 1 && on[0] === SpotFilter.OPEN_NOW ? "closed" : "filtered";
}

// Four decimals of a degree is about 11 m: finer than a spot's distance needs.
const SNAP = 1e4;

/**
 * snapPoint rounds the phone's point to about 11 m, so a phone that moves a step between the fries
 * moment and the list asks the same question, and the list's answer comes from the cache.
 */
export function snapPoint(lat: number, lng: number) {
  return { lat: Math.round(lat * SNAP) / SNAP, lng: Math.round(lng * SNAP) / SNAP };
}
