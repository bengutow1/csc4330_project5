/**
 * settleAfter hands the last value pushed to onSettle once wait milliseconds pass with no newer
 * push, so a burst of changes (a word being typed) settles once, on how it ended. cancel drops a
 * value still waiting.
 */
export function settleAfter<T>(wait: number, onSettle: (value: T) => void) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  return {
    push(value: T) {
      clearTimeout(timer);
      timer = setTimeout(() => onSettle(value), wait);
    },
    cancel() {
      clearTimeout(timer);
    },
  };
}
