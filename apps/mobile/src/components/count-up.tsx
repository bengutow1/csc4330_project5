import { useEffect, useRef, useState } from "react";
import { type StyleProp, StyleSheet, Text, type TextStyle } from "react-native";
import { useReducedMotion } from "react-native-reanimated";
import { countAt } from "@/lib/count-up";

// Long enough to read as counting, short enough not to hold up the verdict.
const COUNT_MS = 900;

/**
 * CountUp counts up to its value, for the score and the points totals. A screen reader hears only
 * the value it lands on, and the value is simply there when the phone asks for reduced motion.
 * The prefix goes before the number ("+" on points).
 */
export function CountUp(props: { value: number; prefix?: string; style?: StyleProp<TextStyle> }) {
  const reduceMotion = useReducedMotion();
  const value = props.value;
  const [shown, setShown] = useState(reduceMotion ? value : 0);
  // Where the next count starts: the number on screen when the value changes.
  const onScreen = useRef(shown);

  useEffect(() => {
    const from = onScreen.current;
    if (reduceMotion || from === value) {
      onScreen.current = value;
      setShown(value);
      return;
    }
    const start = performance.now();
    let frame = requestAnimationFrame(function tick() {
      const n = countAt(from, value, performance.now() - start, COUNT_MS);
      onScreen.current = n;
      setShown(n);
      if (n !== value) {
        frame = requestAnimationFrame(tick);
      }
    });
    return () => cancelAnimationFrame(frame);
  }, [value, reduceMotion]);

  const prefix = props.prefix ?? "";
  return (
    <Text accessibilityLabel={`${prefix}${value}`} style={[styles.number, props.style]}>
      {`${prefix}${shown}`}
    </Text>
  );
}

const styles = StyleSheet.create({
  // Every digit the same width, so the number holds still while it counts.
  number: { fontVariant: ["tabular-nums"] },
});
