import { PhotoKind } from "@fries-in-the-bag/api";
import type { Option } from "../components/option.ts";
import { jpegBytes } from "./photo.ts";

/** Which part of the kitchen a shot shows. The player picks it before taking the shot. */
export type ShotKind = "fridge" | "freezer" | "pantry";

const LABELS: Record<ShotKind, string> = {
  fridge: "Fridge",
  freezer: "Freezer",
  pantry: "Pantry",
};

/** The kinds in the order the camera offers them. */
export const KINDS: readonly Option<ShotKind>[] = (["fridge", "freezer", "pantry"] as const).map(
  (value) => ({ value, label: LABELS[value] }),
);

/** kindLabel is the word a photo's thumbnail shows for what it is of. */
export function kindLabel(kind: ShotKind): string {
  return LABELS[kind];
}

/** A scan takes one to three photos (docs/contracts.md). */
export const MAX_SHOTS = 3;

/** Shot is one photo, already shrunk for the trip, and the part of the kitchen it shows. */
export interface Shot {
  /** The shrunk JPEG on disk, for the thumbnail and the judging screen. */
  uri: string;
  /** The same JPEG in base64, as the manipulator wrote it, for the request. */
  base64: string;
  kind: ShotKind;
}

const WIRE: Record<ShotKind, PhotoKind> = {
  fridge: PhotoKind.FRIDGE,
  freezer: PhotoKind.FREEZER,
  pantry: PhotoKind.PANTRY,
};

/** addShot puts a new shot after the others, and leaves the list as it is once it holds three. */
export function addShot(shots: readonly Shot[], shot: Shot): readonly Shot[] {
  return shots.length >= MAX_SHOTS ? shots : [...shots, shot];
}

/**
 * isFull says the scan takes no more photos: the ones listed and the ones still saving already
 * make three, so another would be dropped as it landed.
 */
export function isFull(listed: number, saving: number): boolean {
  return listed + saving >= MAX_SHOTS;
}

/** removeShot takes out the shot at `index`, and the rest keep their order. */
export function removeShot(shots: readonly Shot[], index: number): readonly Shot[] {
  return shots.filter((_, i) => i !== index);
}

/** scanPhotos is what CreateScan sends: each shot's bytes and kind, in the order they were taken. */
export function scanPhotos(shots: readonly Shot[]) {
  return shots.map((shot) => ({ jpeg: jpegBytes(shot.base64), kind: WIRE[shot.kind] }));
}
