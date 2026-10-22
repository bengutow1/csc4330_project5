import type { Spot } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { AccessibilityInfo, ScrollView, StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { EmptyState } from "@/components/empty-state";
import { SpotSheet } from "@/screens/fries/spot-sheet";
import { useLoadedSpots } from "@/screens/fries/spots-context";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const REEL_TICKS = 20;
const TICK_MS = 90;

/** Spins only over the loaded, filtered spots. A pick never logs a fry run. */
export function RouletteScreen() {
  const router = useRouter();
  const { list, spots, searching } = useLoadedSpots();
  const [reduceMotion, setReduceMotion] = useState<boolean>();
  const [pick, setPick] = useState<Spot>();
  const [rolling, setRolling] = useState(false);
  const [showSheet, setShowSheet] = useState(false);
  const timer = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  useEffect(() => () => clearInterval(timer.current), []);

  useEffect(() => {
    let live = true;
    AccessibilityInfo.isReduceMotionEnabled().then(
      (enabled) => {
        if (live) setReduceMotion(enabled);
      },
      () => {
        if (live) setReduceMotion(true);
      },
    );
    const subscription = AccessibilityInfo.addEventListener("reduceMotionChanged", setReduceMotion);
    return () => {
      live = false;
      subscription.remove();
    };
  }, []);

  // Enabling Reduce Motion during a spin stops it immediately, too.
  useEffect(() => {
    if (reduceMotion !== false) {
      clearInterval(timer.current);
      setRolling(false);
    }
  }, [reduceMotion]);

  function spin() {
    const result = list[Math.floor(Math.random() * list.length)];
    if (result === undefined) return;
    clearInterval(timer.current);
    setShowSheet(false);
    if (reduceMotion !== false) {
      setPick(result);
      return;
    }
    setRolling(true);
    let tick = 0;
    timer.current = setInterval(() => {
      tick += 1;
      if (tick >= REEL_TICKS) {
        clearInterval(timer.current);
        setPick(result);
        setRolling(false);
      } else {
        setPick(list[tick % list.length]);
      }
    }, TICK_MS);
  }

  const available = searching && list.length > 0 && !spots.isPlaceholderData;
  return (
    <ScrollView contentContainerStyle={styles.content}>
      <Stack.Screen options={{ headerTitle: "Fry Roulette" }} />
      {available ? (
        <>
          <Text style={styles.title}>Can't pick? Let it spin.</Text>
          <Text style={styles.line}>Your spots. Your filters. One place to eat.</Text>
          <View style={styles.reel} accessibilityLiveRegion={rolling ? "none" : "polite"}>
            <Text style={styles.emoji} accessible={false}>
              🍟
            </Text>
            <Text style={styles.pick}>{pick?.name ?? "Where are we eating?"}</Text>
            <Text style={styles.line}>
              {rolling ? "Spinning…" : pick ? "Here's your pick." : "Give it a spin."}
            </Text>
          </View>
          {pick && !rolling ? <Button label="Go" onPress={() => setShowSheet(true)} /> : null}
          <Button
            label={pick ? "Spin again" : "Spin"}
            icon="shuffle"
            variant={pick ? "secondary" : "primary"}
            pending={rolling}
            pendingLabel="Spinning…"
            onPress={spin}
          />
          {showSheet && pick ? (
            <SpotSheet key={pick.spotId} spot={pick} onClose={() => setShowSheet(false)} />
          ) : null}
        </>
      ) : (
        <EmptyState
          title="Find your spots first."
          line="Open Food nearby and choose your filters before spinning."
          action={{ label: "Food nearby", onPress: () => router.replace("/fries/spots") }}
        />
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { flexGrow: 1, padding: SPACE.xl, gap: SPACE.lg, backgroundColor: COLORS.bg },
  title: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  line: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
  reel: {
    minHeight: 240,
    justifyContent: "center",
    gap: SPACE.lg,
    padding: SPACE.xl,
    backgroundColor: COLORS.mustardTint,
    borderRadius: RADII.xl,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
  },
  emoji: { ...TYPE.hero, textAlign: "center" },
  pick: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
});
