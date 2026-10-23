import type { Row } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { type BoardKey, rowLabel } from "@/lib/board";
import { groupDigits } from "@/lib/format";
import { Avatar } from "@/screens/board/avatar";
import { COLORS, SPACE, TYPE } from "@/theme";

// The crown from screen 7, saved as an asset (docs/design.md).
export const CROWN = require("../../../assets/board-crown.svg");
const CROWN_SMALL = 16;
// Room for a two-digit rank, as screen 7 sets it; a longer one widens its row's column.
const RANK_WIDTH = 18;
// A row dims under the finger, so a tap shows it landed before the card slides up.
export const PRESSED_OPACITY = 0.6;

/**
 * BoardRow is one player's place below the podium: rank, avatar, handle and value, as screen 7
 * lists them. The player's own row says "you", and the Hall of Fries leader wears the crown.
 * Tapping a row opens that player's card (B2).
 */
export function BoardRow(props: {
  board: BoardKey;
  row: Row;
  mine: boolean;
  onOpen: (handle: string) => void;
}) {
  const { row, mine } = props;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={rowLabel(props.board, row, mine)}
      accessibilityHint="Opens their player card"
      onPress={() => props.onOpen(row.handle)}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      <Text style={styles.rank}>{row.rank}</Text>
      <Avatar avatar={row.avatar} mine={mine} size="small" />
      <View style={styles.who}>
        <Text numberOfLines={1} style={[styles.handle, mine && styles.mine]}>
          {mine ? "you" : row.handle}
        </Text>
        {row.crown ? <Image source={CROWN} style={styles.crown} /> : null}
      </View>
      <Text style={styles.value}>{groupDigits(row.value)}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.md,
    paddingVertical: SPACE.sm + SPACE.xxs,
  },
  pressed: { opacity: PRESSED_OPACITY },
  rank: { ...TYPE.bodyStrong, minWidth: RANK_WIDTH, color: COLORS.muted },
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: SPACE.xs },
  handle: { ...TYPE.body, flexShrink: 1, fontWeight: TYPE.small.fontWeight, color: COLORS.ink },
  mine: { color: COLORS.ketchupInk },
  crown: { width: CROWN_SMALL, height: CROWN_SMALL },
  value: { ...TYPE.bodyStrong, color: COLORS.ink },
});
