import { Pressable, StyleSheet, Text, View } from "react-native";
import type { Option } from "@/components/option";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const SEGMENT_HEIGHT = 36;

/**
 * SegmentedControl switches a screen between whole views (screen 7's Chefs and Hall of Fries).
 * The segments share the track's width, and the selected one is lifted onto the surface.
 */
export function SegmentedControl<T extends string>(props: {
  options: readonly Option<T>[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <View accessibilityRole="tablist" style={styles.track}>
      {props.options.map((option) => {
        const selected = option.value === props.value;
        return (
          <Pressable
            key={option.value}
            accessibilityRole="tab"
            accessibilityLabel={option.label}
            accessibilityState={{ selected }}
            onPress={() => props.onChange(option.value)}
            style={[styles.segment, selected && styles.segmentSelected]}
          >
            <Text style={selected ? styles.labelSelected : styles.label}>{option.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  track: {
    flexDirection: "row",
    gap: SPACE.xs,
    padding: SPACE.xs,
    borderRadius: RADII.md,
    backgroundColor: COLORS.sunken,
  },
  segment: {
    flex: 1,
    height: SEGMENT_HEIGHT,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.sm,
  },
  segmentSelected: {
    backgroundColor: COLORS.surface,
    shadowColor: COLORS.ink,
    shadowOpacity: 0.12,
    shadowRadius: 3,
    shadowOffset: { width: 0, height: 1 },
    elevation: 1,
  },
  label: { ...TYPE.small, color: COLORS.muted },
  labelSelected: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight, color: COLORS.ink },
});
