import { Pressable, StyleSheet, Text, View } from "react-native";
import type { Option } from "@/components/option";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const HEIGHT = 40;
const DOT = 5;

/**
 * PillToggle picks one of a few modes (screen 1's Photo and Pantry). The selected pill is tinted,
 * with a dot under it.
 */
export function PillToggle<T extends string>(props: {
  options: readonly Option<T>[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <View style={styles.row}>
      {props.options.map((option) => {
        const selected = option.value === props.value;
        return (
          <View key={option.value} style={styles.item}>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={option.label}
              accessibilityState={{ selected }}
              onPress={() => props.onChange(option.value)}
              style={[styles.pill, selected && styles.pillSelected]}
            >
              <Text style={selected ? styles.labelSelected : styles.label}>{option.label}</Text>
            </Pressable>
            {/* Hidden, not removed, so the row never shifts when the choice moves. */}
            <View style={[styles.dot, !selected && styles.dotHidden]} />
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "flex-start",
    gap: SPACE.md + SPACE.xxs,
  },
  item: { alignItems: "center", gap: SPACE.xs },
  pill: {
    height: HEIGHT,
    justifyContent: "center",
    paddingHorizontal: SPACE.lg,
    borderRadius: RADII.pill,
  },
  pillSelected: { backgroundColor: COLORS.ketchupTint },
  label: { ...TYPE.body, fontWeight: TYPE.small.fontWeight, color: COLORS.muted },
  labelSelected: { ...TYPE.bodyStrong, color: COLORS.ketchup },
  dot: { width: DOT, height: DOT, borderRadius: RADII.pill, backgroundColor: COLORS.ketchup },
  dotHidden: { opacity: 0 },
});
