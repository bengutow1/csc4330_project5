import { StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { COLORS, SPACE, TYPE } from "@/theme";

/**
 * EmptyState is what a screen shows before there is anything to show, always with a way forward:
 * a title, one line and one action. An error never borrows its words.
 */
export function EmptyState(props: {
  title: string;
  line: string;
  action: { label: string; onPress: () => void };
}) {
  return (
    <View style={styles.root}>
      <Text accessibilityRole="header" style={styles.title}>
        {props.title}
      </Text>
      <Text style={styles.line}>{props.line}</Text>
      <Button label={props.action.label} onPress={props.action.onPress} size="small" />
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    alignItems: "center",
    gap: SPACE.sm,
    paddingVertical: SPACE.xxl,
    paddingHorizontal: SPACE.xl,
  },
  title: { ...TYPE.heading, color: COLORS.ink, textAlign: "center" },
  line: { ...TYPE.body, color: COLORS.muted, textAlign: "center", marginBottom: SPACE.sm },
});
