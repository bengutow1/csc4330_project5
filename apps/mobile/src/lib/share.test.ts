import { describe, expect, it } from "vitest";
import { byline, CARD, captureSize, previewScale, STORY } from "./share.ts";

describe("CARD", () => {
  it("keeps a story's 9 by 16 shape", () => {
    expect(CARD).toEqual({ width: 360, height: 640 });
    expect(CARD.width / CARD.height).toBeCloseTo(9 / 16);
  });
});

describe("captureSize", () => {
  it("asks iOS for points, which it renders at the screen's scale, so it comes out a story", () => {
    for (const ratio of [2, 3]) {
      const size = captureSize("ios", ratio);
      expect(size.width * ratio).toBeCloseTo(STORY.width);
      expect(size.height * ratio).toBeCloseTo(STORY.height);
    }
  });

  it("asks Android for the story's own pixels, which it scales the snapshot to as given", () => {
    // A Pixel 7 is 2.625x: dividing there gave a 411 by 731 card.
    for (const ratio of [2, 2.625, 3.5]) {
      expect(captureSize("android", ratio)).toEqual(STORY);
    }
  });

  it("is the card itself on a 3x iPhone, so nothing is resized", () => {
    expect(captureSize("ios", 3)).toEqual(CARD);
  });
});

describe("previewScale", () => {
  it("fits the card into a short room by its height", () => {
    expect(previewScale(CARD, { width: 360, height: 320 })).toBe(0.5);
  });

  it("fits the card into a narrow room by its width", () => {
    expect(previewScale(CARD, { width: 180, height: 1000 })).toBe(0.5);
  });

  it("never grows the card past its own size", () => {
    expect(previewScale(CARD, { width: 1000, height: 2000 })).toBe(1);
  });
});

describe("byline", () => {
  it("names the player by their handle", () => {
    expect(byline("chef_maya")).toBe("@chef_maya");
  });

  it("names no one while the handle is unknown, rather than a stray @", () => {
    expect(byline(undefined)).toBeUndefined();
    expect(byline("")).toBeUndefined();
  });
});
