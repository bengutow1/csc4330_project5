import { Tier } from "@fries-in-the-bag/api";
import * as Haptics from "expo-haptics";
import { useEffect } from "react";
import { StyleSheet, Text } from "react-native";
import Animated, { Easing, Keyframe, useReducedMotion } from "react-native-reanimated";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The landing ends on the thud.
const LAND_MS = 280;

// Each tier's name and tint, best first. A tier added to the wire fails to compile until it has
// its own.
const LOOKS: Record<
  Exclude<Tier, Tier.UNSPECIFIED>,
  { label: string; fill: string; ink: string }
> = {
  [Tier.CERTIFIED_CHEF]: { label: "Certified Chef", fill: COLORS.pickleTint, ink: COLORS.pickle },
  [Tier.HOME_COOK]: { label: "Home Cook", fill: COLORS.blueberryTint, ink: COLORS.blueberry },
  [Tier.SCRAPPY]: { label: "Scrappy", fill: COLORS.mustardTint, ink: COLORS.mustardInk },
  [Tier.FRIES_IN_THE_BAG]: {
    label: "Fries in the Bag",
    fill: COLORS.ketchupTint,
    ink: COLORS.ketchupInk,
  },
};

// Brought down like a stamp: big and turned, gathering speed, then flat on the page.
const LANDING = new Keyframe({
  0: { opacity: 0, transform: [{ scale: 1.6 }, { rotate: "-8deg" }] },
  100: {
    opacity: 1,
    transform: [{ scale: 1 }, { rotate: "0deg" }],
    easing: Easing.in(Easing.quad),
  },
}).duration(LAND_MS);

/**
 * TierPill names a verdict's tier in that tier's tint, at the top of the verdict (screen 2). It
 * lands like the stamp it replaces, with a slight turn and a thud, and simply appears, still
 * thudding, when the phone asks for reduced motion. The server decides the tier; this shows it.
 */
export function TierPill(props: { tier: Tier }) {
  const reduceMotion = useReducedMotion();
  const tier = props.tier;

  useEffect(() => {
    // Every verdict has a tier, so unspecified means there is nothing to land.
    if (tier === Tier.UNSPECIFIED) {
      return;
    }
    const timer = setTimeout(thud, reduceMotion ? 0 : LAND_MS);
    return () => clearTimeout(timer);
  }, [tier, reduceMotion]);

  if (tier === Tier.UNSPECIFIED) {
    return null;
  }
  const look = LOOKS[tier];
  return (
    <Animated.View
      entering={LANDING}
      accessible
      accessibilityLabel={`${look.label} tier`}
      style={[styles.pill, { backgroundColor: look.fill }]}
    >
      <Text style={[styles.label, { color: look.ink }]}>{look.label}</Text>
    </Animated.View>
  );
}

// A heavy tap as the pill lands. A phone that cannot buzz just stays quiet.
function thud() {
  Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy).catch(() => undefined);
}

const styles = StyleSheet.create({
  // Centred under the score, and as wide as its words.
  pill: {
    alignSelf: "center",
    paddingVertical: SPACE.sm,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderRadius: RADII.pill,
  },
  label: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight },
});
