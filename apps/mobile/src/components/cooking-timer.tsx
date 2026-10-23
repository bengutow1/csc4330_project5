import * as Haptics from "expo-haptics";
import { useEffect, useState } from "react";
import { AppState, StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { remainingMilliseconds, type TimerState, timerLabel } from "@/lib/cooking-timer";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const TICK_MS = 100;

/** Mount per step: leaving it discards its foreground-only timer. */
export function CookingTimer({ seconds }: { seconds: number }) {
  const [timer, setTimer] = useState<TimerState>({ remainingMs: seconds * 1000 });
  const [now, setNow] = useState(() => performance.now());
  const remaining = remainingMilliseconds(timer, now);
  const running = timer.startedAt !== undefined;

  useEffect(() => {
    if (timer.startedAt === undefined) return;
    let live = true;
    const subscription = AppState.addEventListener("change", (state) => {
      if (state !== "active") {
        live = false;
        const time = performance.now();
        setNow(time);
        setTimer({ remainingMs: remainingMilliseconds(timer, time) });
      }
    });
    const interval = setInterval(() => {
      if (!live || AppState.currentState !== "active") return;
      const time = performance.now();
      setNow(time);
      if (remainingMilliseconds(timer, time) === 0) {
        live = false;
        setTimer({ remainingMs: 0 });
        // Haptics can be unavailable on a device; the visible completion still works.
        Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success).catch(() => undefined);
      }
    }, TICK_MS);
    return () => {
      live = false;
      clearInterval(interval);
      subscription.remove();
    };
  }, [timer]);

  function toggle() {
    const time = performance.now();
    setNow(time);
    setTimer(
      running
        ? { remainingMs: remainingMilliseconds(timer, time) }
        : { remainingMs: timer.remainingMs, startedAt: time },
    );
  }

  return (
    <View style={styles.card}>
      <Text style={styles.time} accessibilityLabel={`Timer ${timerLabel(remaining)}`}>
        {timerLabel(remaining)}
      </Text>
      {remaining === 0 ? (
        <Text accessibilityRole="alert" style={styles.finished}>
          Timer finished
        </Text>
      ) : (
        <Button
          label={
            running
              ? "Pause timer"
              : timer.remainingMs < seconds * 1000
                ? "Resume timer"
                : "Start timer"
          }
          variant="secondary"
          onPress={toggle}
        />
      )}
      <Text style={styles.note}>Pauses when the app leaves the foreground.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    padding: SPACE.lg,
    gap: SPACE.md,
    borderRadius: RADII.xl,
    backgroundColor: COLORS.mustardTint,
  },
  time: { ...TYPE.hero, color: COLORS.ink, textAlign: "center", fontVariant: ["tabular-nums"] },
  finished: { ...TYPE.heading, color: COLORS.pickle, textAlign: "center" },
  note: { ...TYPE.caption, color: COLORS.muted, textAlign: "center" },
});
