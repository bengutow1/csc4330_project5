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
 * captureSize is what to ask the capture for, so the picture comes out a story on this platform.
 * iOS takes points and renders them at the screen's scale, so it gets the story over the ratio
 * (the Expo docs' captureRef example). Android takes pixels and scales the snapshot to exactly
 * that, so it gets the story as is: dividing there shrank a Pixel 7's card to 411 by 731.
 */
export function captureSize(os: string | undefined, pixelRatio: number): Box {
  if (os === "android") {
    return STORY;
  }
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
