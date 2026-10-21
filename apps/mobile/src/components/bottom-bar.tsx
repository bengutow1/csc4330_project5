import { Children, isValidElement, type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { COLORS, SPACE } from "@/theme";

/**
 * BottomBar pins a screen's main action to its foot, clear of the home indicator. Two actions sit
 * side by side and share the width (screen 6's Share and Done). It is for a screen with no tab bar
 * under it, since it pads for the bottom of the phone itself.
 */
export function BottomBar(props: { children: ReactNode }) {
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.bar, { paddingBottom: Math.max(insets.bottom, SPACE.lg) }]}>
      {/* toArray drops a null child, so an action left out leaves no empty half. */}
      {Children.toArray(props.children).map((child) => (
        <View key={isValidElement(child) ? child.key : String(child)} style={styles.slot}>
          {child}
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: "row",
    gap: SPACE.md,
    paddingTop: SPACE.md,
    paddingHorizontal: SPACE.lg,
    borderTopWidth: 1,
    borderTopColor: COLORS.line,
    backgroundColor: COLORS.bg,
  },
  slot: { flex: 1 },
});
