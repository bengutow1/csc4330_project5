import Ionicons from "@expo/vector-icons/Ionicons";
import { Pressable, StyleSheet, Text } from "react-native";
import type { IconName } from "@/components/icon-button";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

/** Primary is a screen's main action, secondary sits beside it, destructive cannot be undone. */
export type ButtonVariant = "primary" | "secondary" | "destructive";

/** Big fills its row, as every action in the design does; small is the lesser of two choices. */
export type ButtonSize = "big" | "small";

const BIG_HEIGHT = 52;
const SMALL_HEIGHT = 40;
// The design's icons beside a button's words (screen 9's Fry Roulette, screen 6's Share).
const BIG_ICON = 20;
const SMALL_ICON = 16;

const LOOKS = {
  primary: {
    fill: COLORS.ketchup,
    pressed: COLORS.ketchupInk,
    border: COLORS.ketchup,
    text: COLORS.surface,
    type: TYPE.button,
  },
  secondary: {
    fill: COLORS.surface,
    pressed: COLORS.sunken,
    border: COLORS.lineStrong,
    text: COLORS.ink,
    type: TYPE.buttonSecondary,
  },
  // The design draws no destructive button, so it takes the secondary shape and says danger in
  // ketchup.
  destructive: {
    fill: COLORS.surface,
    pressed: COLORS.ketchupTint,
    border: COLORS.ketchup,
    text: COLORS.ketchupInk,
    type: TYPE.buttonSecondary,
  },
} as const;

/**
 * Button is the app's one button. While its write is in flight it shows its pending label, which
 * ends with the ellipsis character, and ignores presses. That is the only time it is disabled:
 * the server's validation, not the button, says what a form is missing.
 */
export function Button(props: {
  label: string;
  onPress: () => void;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: IconName;
  /** True while this button's own write is in flight. */
  pending?: boolean;
  /** What it says while pending ("Judging…"). The label and an ellipsis when unset. */
  pendingLabel?: string;
}) {
  const look = LOOKS[props.variant ?? "primary"];
  const big = (props.size ?? "big") === "big";
  const pending = props.pending ?? false;
  const label = pending ? (props.pendingLabel ?? `${props.label}…`) : props.label;

  return (
    <Pressable
      accessibilityRole="button"
      // Named outright, so a screen reader never reads the icon's glyph into it.
      accessibilityLabel={label}
      accessibilityState={{ disabled: pending, busy: pending }}
      disabled={pending}
      onPress={props.onPress}
      style={({ pressed }) => [
        styles.button,
        big ? styles.big : styles.small,
        { backgroundColor: pressed ? look.pressed : look.fill, borderColor: look.border },
        pending && styles.pending,
      ]}
    >
      {props.icon ? (
        <Ionicons name={props.icon} size={big ? BIG_ICON : SMALL_ICON} color={look.text} />
      ) : null}
      <Text style={[big ? look.type : TYPE.small, { color: look.text }]}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.sm,
    borderWidth: 1,
    borderRadius: RADII.lg,
    paddingHorizontal: SPACE.lg,
  },
  big: { alignSelf: "stretch", height: BIG_HEIGHT },
  small: { alignSelf: "center", height: SMALL_HEIGHT },
  pending: { opacity: 0.7 },
});
