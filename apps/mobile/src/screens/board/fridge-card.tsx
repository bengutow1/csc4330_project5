import { useFridgeOfTheWeek } from "@fries-in-the-bag/api";
import { StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { Card } from "@/components/card";
import { Skeleton } from "@/components/skeleton";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The card's height while it loads: two lines beside the badge, and the roast under them.
const FRIDGE_SKELETON = 104;
// The badge circle, as screen 7's Hall of Fries card draws its icon.
const BADGE = 40;
const BADGE_EMOJI = 20;

/**
 * FridgeCard sits above the board with the week's best fridge: whose it was, its score and its
 * roast. Before anyone scans this week there is no card.
 */
export function FridgeCard() {
  const fridge = useFridgeOfTheWeek();
  if (fridge.isPending) {
    return <Skeleton height={FRIDGE_SKELETON} radius={RADII.xl} />;
  }
  if (fridge.isError) {
    return (
      <Card>
        <View style={styles.failed}>
          <Text style={styles.failedLine}>Couldn't load the Fridge of the Week.</Text>
          <Button
            label="Try again"
            pendingLabel="Trying again…"
            pending={fridge.isFetching}
            onPress={() => void fridge.refetch()}
            variant="secondary"
            size="small"
          />
        </View>
      </Card>
    );
  }
  const best = fridge.data.fridge;
  if (best === undefined) {
    return null;
  }
  return (
    <Card>
      <View
        accessible
        accessibilityLabel={`Fridge of the Week: ${best.handle}, scored ${best.score}. ${best.roast}`}
        style={styles.body}
      >
        <View style={styles.top}>
          <View style={styles.badge}>
            <Text style={styles.badgeEmoji}>{best.avatar}</Text>
          </View>
          <View style={styles.titles}>
            <Text style={styles.title}>Fridge of the Week</Text>
            <Text numberOfLines={1} style={styles.line}>
              {best.handle} scored {best.score}
            </Text>
          </View>
        </View>
        <Text style={styles.roast}>“{best.roast}”</Text>
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  body: { gap: SPACE.sm + SPACE.xxs },
  top: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  badge: {
    width: BADGE,
    height: BADGE,
    borderRadius: BADGE / 2,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: COLORS.mustardTint,
  },
  badgeEmoji: { fontSize: BADGE_EMOJI },
  titles: { flex: 1, gap: SPACE.xxs },
  title: { ...TYPE.bodyStrong, color: COLORS.ink },
  line: { ...TYPE.caption, color: COLORS.muted },
  roast: { ...TYPE.body, color: COLORS.inkSoft },
  failed: { alignItems: "center", gap: SPACE.sm },
  failedLine: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
});
