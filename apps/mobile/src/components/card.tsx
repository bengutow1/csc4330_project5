import type { ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { COLORS, RADII, SPACE } from "@/theme";

/**
 * Card holds one thing on a list or one block of a screen: a spot, a points breakdown. A featured
 * card (screen 9's nearest fries) has a ketchup border. With onPress the whole card is a button,
 * and its label says where it goes.
 */
export function Card(props: {
  children: ReactNode;
  featured?: boolean;
  onPress?: () => void;
  accessibilityLabel?: string;
}) {
  const style = [styles.card, props.featured && styles.featured];
  if (props.onPress) {
    return (
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={props.accessibilityLabel}
        onPress={props.onPress}
        style={({ pressed }) => [style, pressed && styles.pressed]}
      >
        {props.children}
      </Pressable>
    );
  }
  return <View style={style}>{props.children}</View>;
}

/** Divider splits the rows inside a card or a list. */
export function Divider() {
  return <View style={styles.divider} />;
}

const styles = StyleSheet.create({
  card: {
    paddingVertical: SPACE.md,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderWidth: 1,
    borderColor: COLORS.line,
    borderRadius: RADII.xl,
    backgroundColor: COLORS.surface,
  },
  featured: { borderWidth: 2, borderColor: COLORS.ketchup },
  pressed: { backgroundColor: COLORS.bg },
  divider: { height: 1, backgroundColor: COLORS.divider },
});
