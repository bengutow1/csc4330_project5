import { useEffect, useState } from "react";

/**
 * useDebouncedValue is value once it has held still for delayMs, so a check that would run on
 * every keystroke runs at each pause in typing instead.
 */
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setSettled(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return settled;
}
