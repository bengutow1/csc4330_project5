import { useEffect, useMemo, useState } from "react";
import { settleAfter } from "./settle.ts";

/**
 * useSettled is value once it has stopped changing for wait milliseconds, so a request keyed on
 * what the player types goes out at each pause rather than on every key.
 */
export function useSettled<T>(value: T, wait: number) {
  const [settled, setSettled] = useState(value);
  // One settler for the hook's life, so each change cancels the one before it.
  const settler = useMemo(() => settleAfter<T>(wait, setSettled), [wait]);
  useEffect(() => {
    settler.push(value);
  }, [settler, value]);
  useEffect(() => settler.cancel, [settler]);
  return settled;
}
