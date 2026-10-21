import { useEffect } from "react";
import { type DimensionValue, StyleSheet } from "react-native";
import Animated, {
  cancelAnimation,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withTiming,
} from "react-native-reanimated";
import { COLORS, RADII } from "@/theme";

// A slow, shallow pulse: enough to say "loading", not enough to pull the eye.
const PULSE_LOW = 0.5;
const PULSE_MS = 800;

/**
 * Skeleton stands in for content that is loading, in its shape: a screen lays out blocks the size
 * of its real rows. It pulses, and holds still when the phone asks for reduced motion. Screen
 * readers skip it; the screen around it says what is loading.
 */
export function Skeleton(props: { height: number; width?: DimensionValue; radius?: number }) {
  const reduceMotion = useReducedMotion();
  const opacity = useSharedValue(1);

  useEffect(() => {
    if (reduceMotion) {
      opacity.set(1);
      return;
    }
    opacity.set(withRepeat(withTiming(PULSE_LOW, { duration: PULSE_MS }), -1, true));
    return () => cancelAnimation(opacity);
  }, [reduceMotion, opacity]);

  const pulse = useAnimatedStyle(() => ({ opacity: opacity.get() }));

  return (
    <Animated.View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={[
        styles.block,
        {
          height: props.height,
          width: props.width ?? "100%",
          borderRadius: props.radius ?? RADII.md,
        },
        pulse,
      ]}
    />
  );
}

const styles = StyleSheet.create({
  block: { backgroundColor: COLORS.sunken },
});
