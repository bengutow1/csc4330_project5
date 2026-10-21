import { Image } from "expo-image";
import { useEffect } from "react";
import { StyleSheet, View } from "react-native";
import Animated, {
  Easing,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withDelay,
  withTiming,
} from "react-native-reanimated";

// Screen 4's drawing in three layers, so the fries can fall into the bag (docs/design.md).
const FRIES = require("../../../assets/fries-moment-fries.svg");
const BAG = require("../../../assets/fries-moment-bag.svg");
const BURST = require("../../../assets/fries-moment-burst.svg");

const SIZE = 170;
// The fries start just out of the frame and land with a bounce, then the bag bursts.
const DROP_FROM = -110;
const DROP_MS = 700;
const BURST_MS = 250;
const BURST_FROM_SCALE = 0.8;

/**
 * FriesBag is the fries moment's drawing: five fries drop into a paper bag, which bursts. It holds
 * the last frame when the phone asks for reduced motion. Screen readers hear one image.
 */
export function FriesBag() {
  const reduceMotion = useReducedMotion();
  const drop = useSharedValue(reduceMotion ? 0 : DROP_FROM);
  const burst = useSharedValue(reduceMotion ? 1 : 0);

  useEffect(() => {
    if (reduceMotion) {
      drop.set(0);
      burst.set(1);
      return;
    }
    drop.set(withTiming(0, { duration: DROP_MS, easing: Easing.bounce }));
    burst.set(withDelay(DROP_MS, withTiming(1, { duration: BURST_MS })));
  }, [reduceMotion, drop, burst]);

  const falling = useAnimatedStyle(() => ({ transform: [{ translateY: drop.get() }] }));
  const bursting = useAnimatedStyle(() => ({
    opacity: burst.get(),
    transform: [{ scale: BURST_FROM_SCALE + (1 - BURST_FROM_SCALE) * burst.get() }],
  }));

  return (
    <View
      accessible
      accessibilityRole="image"
      accessibilityLabel="Fries dropping into a paper bag"
      style={styles.frame}
    >
      <Animated.View style={[StyleSheet.absoluteFill, bursting]}>
        <Image source={BURST} style={styles.layer} />
      </Animated.View>
      <Animated.View style={[StyleSheet.absoluteFill, falling]}>
        <Image source={FRIES} style={styles.layer} />
      </Animated.View>
      {/* Last, so the bag's front hides the bottom of the fries. */}
      <Image source={BAG} style={[StyleSheet.absoluteFill, styles.layer]} />
    </View>
  );
}

const styles = StyleSheet.create({
  // Clipped, so the fries appear from the top edge rather than over the header.
  frame: { width: SIZE, height: SIZE, alignSelf: "center", overflow: "hidden" },
  layer: { width: SIZE, height: SIZE },
});
