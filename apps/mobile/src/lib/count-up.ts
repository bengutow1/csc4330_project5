/**
 * countAt is the whole number a count-up shows `elapsed` ms into a count from `from` to `to` that
 * lasts `duration` ms. It covers most of the way early, slows into the value, and shows the value
 * exactly once the time is up.
 */
export function countAt(from: number, to: number, elapsed: number, duration: number): number {
  if (elapsed >= duration) {
    return to;
  }
  const progress = Math.max(elapsed, 0) / duration;
  // Ease out (cubic): the last few numbers tick by slowly, so the eye lands on the value.
  const eased = 1 - (1 - progress) ** 3;
  return Math.round(from + (to - from) * eased);
}
