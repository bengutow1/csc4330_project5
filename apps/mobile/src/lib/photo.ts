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
