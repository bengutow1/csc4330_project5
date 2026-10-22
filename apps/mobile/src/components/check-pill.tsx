import { Ionicons } from "@expo/vector-icons";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const ICON_SIZE = 16;

/**
 * CheckPill says what the server's check of a plate photo saw, in the green of a passed check:
 * "Looks like Creamy Tomato Pasta" (screen 6). It only ever reports a check the server passed.
 */
export function CheckPill(props: { label: string }) {
  return (
    <View accessible accessibilityLabel={props.label} style={styles.pill}>
      <Ionicons name="checkmark-circle-outline" size={ICON_SIZE} color={COLORS.pickle} />
      <Text style={styles.label}>{props.label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  // Centred under what it describes, as on screen 6, and as wide as its words.
  pill: {
    alignSelf: "center",
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.xs + SPACE.xxs,
    paddingVertical: SPACE.sm,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.pickleTint,
  },
  label: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight, color: COLORS.pickle },
});
