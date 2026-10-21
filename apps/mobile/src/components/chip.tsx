import Ionicons from "@expo/vector-icons/Ionicons";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const HEIGHT = 40;
const TICK = 14;
const CROSS = 18;

/**
 * FilterChip narrows a list the server filters (screen 9's Open now, Under $10). A selected chip is
 * filled ketchup with a tick, so it reads as on without relying on colour alone.
 */
export function FilterChip(props: { label: string; selected: boolean; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={props.label}
      accessibilityState={{ selected: props.selected }}
      onPress={props.onPress}
      style={[styles.chip, props.selected ? styles.on : styles.off]}
    >
      {props.selected ? <Ionicons name="checkmark" size={TICK} color={COLORS.surface} /> : null}
      <Text style={props.selected ? styles.onText : styles.offText}>{props.label}</Text>
    </Pressable>
  );
}

/**
 * RemovableChip is one food on a list the player can fix (the edit sheet). Only its cross is a
 * button, and its label names the food it removes.
 */
export function RemovableChip(props: { label: string; onRemove: () => void }) {
  return (
    <View style={[styles.chip, styles.off, styles.removable]}>
      <Text style={styles.offText}>{props.label}</Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`Remove ${props.label}`}
        hitSlop={SPACE.sm}
        onPress={props.onRemove}
      >
        <Ionicons name="close-circle" size={CROSS} color={COLORS.muted} />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    alignSelf: "flex-start",
    gap: SPACE.xs + SPACE.xxs,
    height: HEIGHT,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderWidth: 1,
    borderRadius: RADII.pill,
  },
  on: { backgroundColor: COLORS.ketchup, borderColor: COLORS.ketchup },
  off: { backgroundColor: COLORS.surface, borderColor: COLORS.lineStrong },
  // The cross sits closer to the edge than the words do.
  removable: { paddingRight: SPACE.sm },
  onText: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight, color: COLORS.surface },
  offText: { ...TYPE.small, color: COLORS.ink },
});
