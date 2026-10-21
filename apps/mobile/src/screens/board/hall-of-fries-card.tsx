import Ionicons from "@expo/vector-icons/Ionicons";
import { useBoard } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { StyleSheet, Text, View } from "react-native";
import { Card } from "@/components/card";
import { boardOf, leaderLine, rangeOf } from "@/lib/board";
import { COLORS, SPACE, TYPE } from "@/theme";

// Screen 7's fries, saved as an asset (docs/design.md).
const FRIES = require("../../../assets/board-fries.svg");
const BADGE = 40;
const ICON = 20;
const CHEVRON = 18;

/**
 * HallOfFriesCard sits under the Chefs board, as screen 7 draws it: this week's Hall of Fries
 * leader, and a tap over to that board. It reads the same query the Hall of Fries switch does, so
 * the tap lands on a board already in hand. With no fry run this week, or no answer yet, there is
 * no card: it is a shortcut, and the Hall of Fries itself says when it cannot load.
 */
export function HallOfFriesCard(props: { onPress: () => void }) {
  const fries = useBoard(boardOf("hall_of_fries"), rangeOf("this_week"));
  const leader = fries.data?.pages[0]?.rows[0];
  if (leader === undefined) {
    return null;
  }
  return (
    <View style={styles.spaced}>
      <Card
        onPress={props.onPress}
        accessibilityLabel={`Open the Hall of Fries: ${leaderLine(leader)}`}
      >
        <View style={styles.row}>
          <View style={styles.badge}>
            <Image source={FRIES} style={styles.icon} />
          </View>
          <View style={styles.titles}>
            <Text style={styles.title}>Hall of Fries</Text>
            <Text numberOfLines={1} style={styles.line}>
              {leaderLine(leader)}
            </Text>
          </View>
          <Ionicons name="chevron-forward" size={CHEVRON} color={COLORS.muted} />
        </View>
      </Card>
    </View>
  );
}

const styles = StyleSheet.create({
  spaced: { marginTop: SPACE.md },
  row: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  badge: {
    width: BADGE,
    height: BADGE,
    borderRadius: BADGE / 2,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: COLORS.mustardTint,
  },
  icon: { width: ICON, height: ICON },
  titles: { flex: 1, gap: SPACE.xxs },
  title: { ...TYPE.bodyStrong, color: COLORS.ink },
  line: { ...TYPE.caption, color: COLORS.muted },
});
