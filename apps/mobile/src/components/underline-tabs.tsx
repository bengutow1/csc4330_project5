import { Pressable, StyleSheet, Text, View } from "react-native";
import type { Option } from "@/components/option";
import { COLORS, SPACE, TYPE } from "@/theme";

const UNDERLINE = 2;

/**
 * UnderlineTabs narrows what a view shows, a step below a segmented control (screen 7's This week
 * and All time). The selected tab is ketchup and underlined.
 */
export function UnderlineTabs<T extends string>(props: {
  options: readonly Option<T>[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <View accessibilityRole="tablist" style={styles.row}>
      {props.options.map((option) => {
        const selected = option.value === props.value;
        return (
          <Pressable
            key={option.value}
            accessibilityRole="tab"
            accessibilityLabel={option.label}
            accessibilityState={{ selected }}
            onPress={() => props.onChange(option.value)}
            style={[styles.tab, selected && styles.tabSelected]}
          >
            <Text style={selected ? styles.labelSelected : styles.label}>{option.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", justifyContent: "center", gap: SPACE.xl - SPACE.xxs },
  tab: {
    paddingTop: SPACE.sm + SPACE.xxs,
    paddingBottom: SPACE.sm,
    paddingHorizontal: SPACE.xxs,
    // A transparent line keeps every tab the same height as the selected one.
    borderBottomWidth: UNDERLINE,
    borderBottomColor: "transparent",
  },
  tabSelected: { borderBottomColor: COLORS.ketchup },
  label: { ...TYPE.small, color: COLORS.muted },
  labelSelected: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight, color: COLORS.ketchup },
});
