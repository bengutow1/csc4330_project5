import Ionicons from "@expo/vector-icons/Ionicons";
import type { Badge } from "@fries-in-the-bag/api";
import { StyleSheet, Text, View } from "react-native";
import { badgeLook, earnedLine, type Tone } from "@/lib/profile";
import { COLORS, SPACE, TYPE } from "@/theme";

// Screen 10's badge circles, four to a row.
const CIRCLE = 56;
const ICON = 26;
const PER_ROW = 4;
// A locked badge sits back, as the design fades it.
const LOCKED_OPACITY = 0.6;

const TONES: Record<Tone, { fill: string; ink: string }> = {
  mustard: { fill: COLORS.mustardTint, ink: COLORS.mustardInk },
  blueberry: { fill: COLORS.blueberryTint, ink: COLORS.blueberry },
  pickle: { fill: COLORS.pickleTint, ink: COLORS.pickle },
  ketchup: { fill: COLORS.ketchupTint, ink: COLORS.ketchupInk },
};

/**
 * BadgeShelf is every badge in the server's order, four to a row, with how many are earned. An
 * earned badge wears its icon in its tint; a locked one is faded behind a lock and says how to
 * earn it to a screen reader.
 */
export function BadgeShelf(props: { badges: readonly Badge[] }) {
  return (
    <View>
      <View style={styles.heading}>
        <Text accessibilityRole="header" style={styles.title}>
          Badges
        </Text>
        <Text style={styles.count}>{earnedLine(props.badges)}</Text>
      </View>
      <View style={styles.grid}>
        {props.badges.map((badge) => (
          <BadgeItem key={badge.name} badge={badge} />
        ))}
      </View>
    </View>
  );
}

function BadgeItem(props: { badge: Badge }) {
  const { badge } = props;
  const look = badgeLook(badge.name);
  const tone = TONES[look.tone];
  return (
    <View
      accessible
      accessibilityLabel={
        badge.earned ? `${badge.name}, earned` : `${badge.name}, locked. ${badge.howToEarn}`
      }
      style={[styles.item, !badge.earned && styles.locked]}
    >
      <View style={[styles.circle, { backgroundColor: badge.earned ? tone.fill : COLORS.sunken }]}>
        <Ionicons
          name={badge.earned ? look.icon : "lock-closed-outline"}
          size={ICON}
          color={badge.earned ? tone.ink : COLORS.muted}
        />
      </View>
      <Text style={[styles.name, !badge.earned && styles.nameLocked]}>{badge.name}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  heading: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginTop: SPACE.lg + SPACE.xs,
    marginBottom: SPACE.sm + SPACE.xxs,
    marginHorizontal: SPACE.xxs,
  },
  title: { ...TYPE.heading, fontSize: TYPE.button.fontSize, color: COLORS.ink },
  count: { ...TYPE.caption, color: COLORS.muted },
  grid: { flexDirection: "row", flexWrap: "wrap", rowGap: SPACE.md },
  item: { width: `${100 / PER_ROW}%`, alignItems: "center", gap: SPACE.xs + SPACE.xxs },
  locked: { opacity: LOCKED_OPACITY },
  circle: {
    width: CIRCLE,
    height: CIRCLE,
    borderRadius: CIRCLE / 2,
    alignItems: "center",
    justifyContent: "center",
  },
  name: {
    ...TYPE.tab,
    fontWeight: TYPE.small.fontWeight,
    color: COLORS.ink,
    textAlign: "center",
  },
  nameLocked: { color: COLORS.muted },
});
