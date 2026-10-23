/** TimerState keeps only the current foreground countdown, never a background alarm. */
export type TimerState = { remainingMs: number; startedAt?: number };

/** Measures elapsed time so delayed interval callbacks do not extend cooking. */
export function remainingMilliseconds(timer: TimerState, now: number): number {
  const elapsed = timer.startedAt === undefined ? 0 : Math.max(0, now - timer.startedAt);
  return Math.max(0, timer.remainingMs - elapsed);
}

/** Rounds up so a running timer never displays zero before it finishes. */
export function timerLabel(milliseconds: number): string {
  const seconds = Math.ceil(milliseconds / 1000);
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
}
