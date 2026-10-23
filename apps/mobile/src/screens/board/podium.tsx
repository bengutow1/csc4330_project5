import type { Row } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { StyleSheet, Text, View } from "react-native";
import { type BoardKey, podiumOrder, podiumStep, rowLabel } from "@/lib/board";
import { groupDigits } from "@/lib/format";
import { Avatar } from "@/screens/board/avatar";
import { CROWN } from "@/screens/board/board-row";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The podium's three places, left to right.
const COLUMNS = ["left", "middle", "right"] as const;
const CROWN_SIZE = 24;
// Screen 7's steps: the tallest for first, then second, then third.
const STEP_HEIGHTS = { first: 80, second: 56, third: 40 } as const;
const STEP_NUMBER = 22;

/**
 * Podium stands the board's first three rows as screen 7 does: first in the middle on the tallest
 * step, second on the left, third on the right. A step's height comes from its rank, so players
 * tied for first stand equally tall, and both wear the crown.
 */
export function Podium(props: { board: BoardKey; rows: readonly Row[]; myHandle?: string }) {
  const order = podiumOrder(props.rows);
  return (
    <View style={styles.podium}>
      {COLUMNS.map((column, i) => {
        const row = order[i];
        if (row === undefined) {
          return <View key={column} style={styles.column} />;
        }
        return (
          <Place
            key={column}
            board={props.board}
            row={row}
            mine={row.handle === props.myHandle}
            middle={column === "middle"}
          />
        );
      })}
    </View>
  );
}

function Place(props: { board: BoardKey; row: Row; mine: boolean; middle: boolean }) {
  const { row, mine } = props;
  const step = podiumStep(row.rank);
  return (
    <View accessible accessibilityLabel={rowLabel(props.board, row, mine)} style={styles.column}>
      {/* First place wears the crown on both boards, as screen 7 draws it. The rows below
          follow the server's flag, which crowns only the Hall of Fries leader. */}
      {row.rank === 1 ? <Image source={CROWN} style={styles.crown} /> : null}
      <Avatar avatar={row.avatar} mine={mine} size={props.middle ? "large" : "medium"} />
      <Text numberOfLines={1} style={[styles.handle, mine && styles.mine]}>
        {mine ? "you" : row.handle}
      </Text>
      <Text style={styles.value}>{groupDigits(row.value)}</Text>
      <View
        style={[styles.step, { height: STEP_HEIGHTS[step] }, step === "first" && styles.stepFirst]}
      >
        <Text style={[styles.stepNumber, step === "first" && styles.stepNumberFirst]}>
          {row.rank}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  podium: { flexDirection: "row", alignItems: "flex-end", gap: SPACE.sm + SPACE.xxs },
  column: { flex: 1, alignItems: "center", gap: SPACE.xs },
  crown: { width: CROWN_SIZE, height: CROWN_SIZE },
  handle: {
    ...TYPE.caption,
    marginTop: SPACE.xxs,
    fontWeight: TYPE.bodyStrong.fontWeight,
    color: COLORS.ink,
  },
  mine: { color: COLORS.ketchupInk },
  value: { ...TYPE.caption, color: COLORS.muted },
  step: {
    alignSelf: "stretch",
    alignItems: "center",
    justifyContent: "center",
    marginTop: SPACE.xs,
    borderTopLeftRadius: RADII.md,
    borderTopRightRadius: RADII.md,
    borderWidth: 1,
    borderBottomWidth: 0,
    borderColor: COLORS.line,
    backgroundColor: COLORS.surface,
  },
  stepFirst: { borderWidth: 0, backgroundColor: COLORS.ketchup },
  stepNumber: { ...TYPE.title, fontSize: STEP_NUMBER, color: COLORS.muted },
  stepNumberFirst: { fontSize: TYPE.title.fontSize, color: COLORS.surface },
});
