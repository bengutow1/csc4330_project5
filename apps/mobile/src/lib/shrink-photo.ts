import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import { type Cut, fitLongEdge, type Size } from "./photo.ts";

/** How small a photo leaves the phone: its long edge in pixels, and its JPEG quality. */
export interface PhotoSpec {
  longEdge: number;
  quality: number;
}

/** A scan photo: 1024 px on the long edge at quality 0.7 (docs/contracts.md). */
export const SCAN_PHOTO: PhotoSpec = { longEdge: 1024, quality: 0.7 };

/** A plate photo: 640 px on the long edge at quality 0.6 (docs/contracts.md). */
export const PLATE_PHOTO: PhotoSpec = { longEdge: 640, quality: 0.6 };

/** A file on the phone, or an image already loaded (a bundled asset, through expo-image). */
export type PhotoSource = Parameters<typeof ImageManipulator.manipulate>[0];

/**
 * shrinkPhoto re-encodes a photo as a JPEG no bigger than the spec, for a request to carry, keeping
 * only the cut when there is one. It returns the new file, to show, and its base64, to send.
 */
export async function shrinkPhoto(source: PhotoSource, size: Size, spec: PhotoSpec, cut?: Cut) {
  const context = ImageManipulator.manipulate(source);
  if (cut) {
    context.crop(cut);
  }
  const fit = fitLongEdge(cut ?? size, spec.longEdge);
  if (fit) {
    context.resize(fit);
  }
  const image = await context.renderAsync();
  const saved = await image.saveAsync({
    format: SaveFormat.JPEG,
    compress: spec.quality,
    base64: true,
  });
  if (saved.base64 === undefined) {
    throw new Error("the image manipulator saved no base64");
  }
  return { uri: saved.uri, base64: saved.base64 };
}
