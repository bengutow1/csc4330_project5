import { PhotoKind } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import {
  addShot,
  KINDS,
  kindLabel,
  removeShot,
  type Shot,
  type ShotKind,
  scanPhotos,
} from "./shots.ts";

// "/9j/4A==" is a JPEG's first four bytes, FF D8 FF E0.
const JPEG_START = "/9j/4A==";

function shot(n: number, kind: ShotKind = "fridge"): Shot {
  return { uri: `file:///shot-${n}.jpg`, base64: JPEG_START, kind };
}

describe("addShot", () => {
  it("puts a new shot after the others", () => {
    expect(addShot([shot(1)], shot(2))).toEqual([shot(1), shot(2)]);
  });

  it("keeps three at most, since a scan takes no more", () => {
    const full = [shot(1), shot(2), shot(3)];

    expect(addShot(full, shot(4))).toEqual(full);
  });

  it("never changes the list it was given", () => {
    const one = [shot(1)];

    addShot(one, shot(2));

    expect(one).toEqual([shot(1)]);
  });
});

describe("removeShot", () => {
  it("takes out only the shot at that place, and the rest keep their order", () => {
    expect(removeShot([shot(1), shot(2), shot(3)], 1)).toEqual([shot(1), shot(3)]);
  });

  it("never changes the list it was given", () => {
    const two = [shot(1), shot(2)];

    removeShot(two, 0);

    expect(two).toEqual([shot(1), shot(2)]);
  });
});

describe("kindLabel", () => {
  it("names each kind as the camera's toggle does", () => {
    expect(KINDS.map((kind) => kindLabel(kind.value))).toEqual(["Fridge", "Freezer", "Pantry"]);
  });
});

describe("scanPhotos", () => {
  it("sends each shot's bytes and kind, in the order they were taken", () => {
    expect(scanPhotos([shot(1, "pantry"), shot(2, "fridge")])).toEqual([
      { jpeg: new Uint8Array([0xff, 0xd8, 0xff, 0xe0]), kind: PhotoKind.PANTRY },
      { jpeg: new Uint8Array([0xff, 0xd8, 0xff, 0xe0]), kind: PhotoKind.FRIDGE },
    ]);
  });

  it("gives every kind the camera offers its own wire value", () => {
    const kinds = scanPhotos(KINDS.map((kind, i) => shot(i, kind.value))).map((p) => p.kind);

    expect(kinds).toEqual([PhotoKind.FRIDGE, PhotoKind.FREEZER, PhotoKind.PANTRY]);
  });
});
