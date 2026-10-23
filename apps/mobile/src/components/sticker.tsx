import Ionicons from "@expo/vector-icons/Ionicons";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The white edge a die-cut sticker keeps around its print.
const EDGE = 3;
const ICON_SIZE = 18;
const WIDTH = 150;
// A locked sticker sits back, as the Me tab fades a locked badge.
const LOCKED_OPACITY = 0.6;

/**
 * Sticker is a badge as a die-cut sticker: its ribbon and name printed on mustard, a white edge
 * and a slight tilt, with how it was earned under it (the concept's look, which the design doesn't
 * draw). Stickers side by side lean opposite ways. A locked one is a faded blank behind a lock,
 * and its line says how to earn it.
 */
export function Sticker(props: {
  name: string;
  line: string;
  lean?: "left" | "right";
  locked?: boolean;
}) {
  const locked = props.locked === true;
  return (
    <View
      accessible
      accessibilityLabel={`${props.name} badge${locked ? ", locked" : ""}. ${props.line}`}
      style={[styles.sticker, locked && styles.locked]}
    >
      <View style={[styles.edge, props.lean === "right" ? styles.right : styles.left]}>
        <View style={[styles.print, locked && styles.printLocked]}>
          <Ionicons
            name={locked ? "lock-closed-outline" : "ribbon-outline"}
            size={ICON_SIZE}
            color={locked ? COLORS.muted : COLORS.mustardInk}
          />
          <Text style={[styles.name, locked && styles.nameLocked]}>{props.name}</Text>
        </View>
      </View>
      <Text style={styles.line}>{props.line}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  sticker: { maxWidth: WIDTH, alignItems: "center", gap: SPACE.sm },
  locked: { opacity: LOCKED_OPACITY },
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
  printLocked: { backgroundColor: COLORS.sunken },
  name: { ...TYPE.small, fontWeight: TYPE.heading.fontWeight, color: COLORS.mustardInk },
  nameLocked: { color: COLORS.muted },
  line: { ...TYPE.caption, color: COLORS.muted, textAlign: "center" },
});
