/** Write is what a screen knows of one of its mutations: its error, and when it was last sent. */
export interface Write {
  error: unknown;
  /** When the write was last sent, in ms, as TanStack Query reports it; 0 before the first. */
  submittedAt: number;
}

/**
 * lastWriteError is the error of the write tried last, or null while that one is fine or still
 * out. A screen with two writes shows only this, so an older failure never speaks over a newer try.
 */
export function lastWriteError(...writes: Write[]): unknown {
  let last: Write | undefined;
  for (const write of writes) {
    if (write.submittedAt > 0 && (last === undefined || write.submittedAt >= last.submittedAt)) {
      last = write;
    }
  }
  return last?.error ?? null;
}
