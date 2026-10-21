import Ionicons from "@expo/vector-icons/Ionicons";
import type { ComponentProps } from "react";
import { Pressable, StyleSheet } from "react-native";
import { COLORS } from "@/theme";

export type IconName = ComponentProps<typeof Ionicons>["name"];

// The smallest target a thumb hits reliably, and the design's back button.
const TARGET = 44;
const ICON = 24;

/**
 * IconButton is an icon with no words beside it, such as the back arrow or a sheet's close cross.
 * Its label says what it does, because a screen reader has nothing else to read.
 */
export function IconButton(props: { icon: IconName; label: string; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={props.label}
      onPress={props.onPress}
      style={({ pressed }) => [styles.button, pressed && styles.pressed]}
    >
      <Ionicons name={props.icon} size={ICON} color={COLORS.ink} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: { width: TARGET, height: TARGET, alignItems: "center", justifyContent: "center" },
  pressed: { opacity: 0.5 },
});
