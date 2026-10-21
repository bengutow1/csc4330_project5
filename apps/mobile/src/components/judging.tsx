import { Image } from "expo-image";
import { useEffect, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import Animated, {
  cancelAnimation,
  Easing,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withTiming,
} from "react-native-reanimated";
import { COLORS, SPACE, TYPE } from "@/theme";

// What the judge is up to. Every line is about the kitchen, never the person in it.
const LINES = [
  "Counting condiments…",
  "Sniffing the leftovers…",
  "Squinting at the dates…",
  "Weighing the cheese situation…",
  "Asking the ketchup what happened…",
];
const LINE_MS = 1800;
const SWEEP_MS = 1400;
// The scan line, and the glow it drags behind it.
const SCAN_LINE = 3;
const GLOW = 56;

/**
 * Judging is the moment between the shutter and the verdict: the photo, a scan line sweeping down
 * it, and a line under it that changes every couple of seconds while the server reads the
 * kitchen. Under Reduce Motion the photo holds still and only the words change.
 */
export function Judging(props: { uri: string }) {
  const reduceMotion = useReducedMotion();
  const [line, setLine] = useState(0);
  const [height, setHeight] = useState(0);
  const progress = useSharedValue(0);

  useEffect(() => {
    const timer = setInterval(() => setLine((i) => (i + 1) % LINES.length), LINE_MS);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (reduceMotion) {
      return;
    }
    const down = withTiming(1, { duration: SWEEP_MS, easing: Easing.inOut(Easing.quad) });
    progress.set(withRepeat(down, -1));
    return () => cancelAnimation(progress);
  }, [reduceMotion, progress]);

  const sweep = useAnimatedStyle(() => ({
    transform: [{ translateY: progress.get() * height }],
  }));

  return (
    <View style={styles.root}>
      <View style={styles.photo} onLayout={(e) => setHeight(e.nativeEvent.layout.height)}>
        <Image
          accessibilityLabel="The photo being judged"
          contentFit="cover"
          source={{ uri: props.uri }}
          style={StyleSheet.absoluteFill}
        />
        {reduceMotion ? null : (
          <Animated.View style={[styles.sweep, sweep]}>
            <View style={styles.glow} />
            <View style={styles.scanLine} />
          </Animated.View>
        )}
      </View>
      <Text accessibilityLiveRegion="polite" style={styles.words}>
        {LINES[line]}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, gap: SPACE.md, paddingBottom: SPACE.md },
  photo: { flex: 1, minHeight: 200, overflow: "hidden", backgroundColor: COLORS.sunken },
  // Starts above the photo, so the line enters from the top edge with its glow out of sight.
  sweep: { position: "absolute", top: -GLOW, left: 0, right: 0 },
  glow: { height: GLOW, backgroundColor: COLORS.ketchupTint, opacity: 0.5 },
  scanLine: { height: SCAN_LINE, backgroundColor: COLORS.ketchup },
  words: { ...TYPE.bodyStrong, color: COLORS.ink, textAlign: "center" },
});
