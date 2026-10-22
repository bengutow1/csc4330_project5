import Ionicons from "@expo/vector-icons/Ionicons";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The white edge a die-cut sticker keeps around its print.
const EDGE = 3;
const ICON_SIZE = 18;
const WIDTH = 150;

/**
 * Sticker is a badge as a die-cut sticker: its ribbon and name printed on mustard, a white edge
 * and a slight tilt, with how it was earned under it (the concept's look, which the design doesn't
 * draw). Stickers side by side lean opposite ways.
 */
export function Sticker(props: { name: string; line: string; lean?: "left" | "right" }) {
  return (
    <View
      accessible
      accessibilityLabel={`${props.name} badge. ${props.line}`}
      style={styles.sticker}
    >
      <View style={[styles.edge, props.lean === "right" ? styles.right : styles.left]}>
        <View style={styles.print}>
          <Ionicons name="ribbon-outline" size={ICON_SIZE} color={COLORS.mustardInk} />
          <Text style={styles.name}>{props.name}</Text>
        </View>
      </View>
      <Text style={styles.line}>{props.line}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  sticker: { maxWidth: WIDTH, alignItems: "center", gap: SPACE.sm },
  edge: {
    padding: EDGE,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
    shadowColor: COLORS.ink,
    shadowOpacity: 0.12,
    shadowRadius: 6,
    shadowOffset: { width: 0, height: 3 },
    elevation: 3,
  },
  left: { transform: [{ rotate: "-4deg" }] },
  right: { transform: [{ rotate: "3deg" }] },
  print: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.xs + SPACE.xxs,
    paddingVertical: SPACE.sm,
    paddingHorizontal: SPACE.md,
    borderRadius: RADII.lg - EDGE,
    backgroundColor: COLORS.mustardTint,
  },
  name: { ...TYPE.small, fontWeight: TYPE.heading.fontWeight, color: COLORS.mustardInk },
  line: { ...TYPE.caption, color: COLORS.muted, textAlign: "center" },
});
