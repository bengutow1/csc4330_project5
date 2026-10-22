/** A photo's size in pixels, as the camera or the manipulator reports it. */
export interface Size {
  width: number;
  height: number;
}

/** The resize that fits a photo: one edge named, so the other keeps the ratio. */
export type Fit = { width: number } | { height: number };

/**
 * fitLongEdge is the resize that brings a photo's long edge down to `longEdge`. A photo already
 * within it is left alone (null): blowing it up would only add bytes.
 */
export function fitLongEdge(size: Size, longEdge: number): Fit | null {
  if (Math.max(size.width, size.height) <= longEdge) {
    return null;
  }
  return size.width >= size.height ? { width: longEdge } : { height: longEdge };
}

/** jpegBytes turns the manipulator's base64 into the bytes a request carries. */
export function jpegBytes(base64: string): Uint8Array {
  return Uint8Array.from(atob(base64), (char) => char.charCodeAt(0));
}

/** The part of a photo to keep: its top-left corner, then its size. */
export interface Cut extends Size {
  originX: number;
  originY: number;
}

// The most a sample plate loses off each edge.
const MAX_TRIM = 32;

/**
 * sampleCut keeps all of a sample photo but a random sliver off each edge, one draw an edge (left,
 * top, right, bottom). The api never takes the same plate photo twice, so this makes each plate
 * the development build's sample stands in for a photo it hasn't seen. random is Math.random, or a
 * stand-in in a test.
 */
export function sampleCut(size: Size, random: () => number): Cut {
  // A small photo loses less, so something is always left.
  const most = Math.min(MAX_TRIM, Math.floor((Math.min(size.width, size.height) - 1) / 2));
  const trim = () => Math.floor(random() * (most + 1));
  const left = trim();
  const top = trim();
  return {
    originX: left,
    originY: top,
    width: size.width - left - trim(),
    height: size.height - top - trim(),
  };
}
