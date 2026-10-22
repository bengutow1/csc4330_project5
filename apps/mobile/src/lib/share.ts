/** A share card's size in pixels: a story, 9 by 16. */
export const STORY = { width: 1080, height: 1920 } as const;

/** A width and height, in points unless said otherwise. */
export interface Box {
  width: number;
  height: number;
}

/**
 * CARD is the size a card is laid out at, in points: a third of a story, so a 3x phone captures
 * it pixel for pixel, and it fits the design's 360-point-wide screens.
 */
export const CARD: Box = { width: STORY.width / 3, height: STORY.height / 3 };

/**
 * captureSize is what to ask the capture for on a phone with this many pixels to a point. The
 * capture's size is in points and comes out in pixels, so a story is the story's pixels over the
 * ratio (the Expo docs' captureRef example).
 */
export function captureSize(pixelRatio: number): Box {
  return { width: STORY.width / pixelRatio, height: STORY.height / pixelRatio };
}

/**
 * previewScale shrinks the card to fit the room the preview has, keeping its shape, and never
 * grows it past its own size.
 */
export function previewScale(card: Box, room: Box): number {
  return Math.min(1, room.width / card.width, room.height / card.height);
}

/** byline is how a card names its player: the handle, and nothing else about them. */
export function byline(handle: string | undefined): string | undefined {
  return handle ? `@${handle}` : undefined;
}
