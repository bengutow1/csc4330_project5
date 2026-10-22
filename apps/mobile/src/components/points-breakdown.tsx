import { Ionicons } from "@expo/vector-icons";
import type { ReactNode } from "react";
import { StyleSheet, Text, View } from "react-native";
import Animated, { FadeIn } from "react-native-reanimated";
import { Card, Divider } from "@/components/card";
import type { IconName } from "@/components/icon-button";
import { COLORS, SPACE, TYPE } from "@/theme";

// Each row prints this long after the one above it, and the total prints last.
const PRINT_MS = 200;
const FADE_MS = 240;
const ICON_SIZE = 16;

/** PointsLine is one award on a breakdown: what it was for, and what it paid. */
export interface PointsLine {
  label: string;
  points: number;
  /** A glyph before the label, like the streak's flame. */
  icon?: IconName;
}

/**
 * PointsBreakdown lists what a plate earned, one award to a row, then the total (screen 6). The
 * rows print one by one, and are simply there when the phone asks for reduced motion. The total
 * is the server's, never added up here.
 */
export function PointsBreakdown(props: { lines: PointsLine[]; total: number }) {
  return (
    <Card>
      {props.lines.map((line, i) => (
        <Printed key={line.label} order={i}>
          <View
            accessible
            accessibilityLabel={`${line.label}, ${points(line.points)}`}
            style={[styles.row, i === 0 && styles.first]}
          >
            <View style={styles.reason}>
              {line.icon && <Ionicons name={line.icon} size={ICON_SIZE} color={COLORS.ketchup} />}
              <Text style={styles.label}>{line.label}</Text>
            </View>
            <Text style={styles.points}>{points(line.points)}</Text>
          </View>
        </Printed>
      ))}
      <Printed order={props.lines.length}>
        <View
          accessible
          accessibilityLabel={`Total, ${points(props.total)}`}
          style={[styles.row, props.lines.length === 0 && styles.first, styles.last]}
        >
          <Text style={styles.totalLabel}>Total</Text>
          <Text style={styles.total}>{points(props.total)}</Text>
        </View>
      </Printed>
    </Card>
  );
}

/** Printed fades a row in after the rows above it, under a divider unless it comes first. */
function Printed(props: { order: number; children: ReactNode }) {
  return (
    <Animated.View entering={FadeIn.delay(props.order * PRINT_MS).duration(FADE_MS)}>
      {props.order > 0 && <Divider />}
      {props.children}
    </Animated.View>
  );
}

function points(n: number) {
  return `+${n}`;
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: SPACE.md,
    paddingVertical: SPACE.sm + SPACE.xxs,
  },
  // The card's own padding is the space above the first row and below the total.
  first: { paddingTop: 0 },
  last: { paddingBottom: 0 },
  reason: { flexDirection: "row", alignItems: "center", gap: SPACE.xs + SPACE.xxs, flexShrink: 1 },
  label: { ...TYPE.body, color: COLORS.ink, flexShrink: 1 },
  points: { ...TYPE.bodyStrong, color: COLORS.ink },
  totalLabel: { ...TYPE.body, fontWeight: TYPE.heading.fontWeight, color: COLORS.ink },
  total: { ...TYPE.body, fontWeight: TYPE.heading.fontWeight, color: COLORS.ketchup },
});
