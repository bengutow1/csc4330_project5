import { describe, expect, it } from "vitest";
import { fitLongEdge, jpegBytes } from "./photo.ts";

describe("fitLongEdge", () => {
  it("brings a landscape photo down by its width", () => {
    expect(fitLongEdge({ width: 4032, height: 3024 }, 1024)).toEqual({ width: 1024 });
  });

  it("brings a portrait photo down by its height", () => {
    expect(fitLongEdge({ width: 3024, height: 4032 }, 1024)).toEqual({ height: 1024 });
  });

  it("brings a square photo down by its width", () => {
    expect(fitLongEdge({ width: 2048, height: 2048 }, 1024)).toEqual({ width: 1024 });
  });

  it("leaves a photo already within the edge alone, rather than blowing it up", () => {
    expect(fitLongEdge({ width: 1024, height: 768 }, 1024)).toBeNull();
    expect(fitLongEdge({ width: 384, height: 440 }, 1024)).toBeNull();
  });
});

describe("jpegBytes", () => {
  it("decodes base64 into the bytes it spells", () => {
    // A JPEG's first four bytes, FF D8 FF E0, which the api checks for.
    expect(jpegBytes("/9j/4A==")).toEqual(new Uint8Array([0xff, 0xd8, 0xff, 0xe0]));
  });

  it("keeps every byte value, not only the ones that are text", () => {
    const all = Uint8Array.from({ length: 256 }, (_, i) => i);
    const base64 = btoa(String.fromCharCode(...all));

    expect(jpegBytes(base64)).toEqual(all);
  });
});
