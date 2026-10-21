import { Stack } from "expo-router";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, SPACE, TYPE } from "@/theme";

/**
 * PlaceholderScreen holds a route until its issue builds the real screen, so any screen can link
 * to any other before it exists. Its owner replaces the route's mount, not this file.
 */
export function PlaceholderScreen(props: { title: string; issue: number }) {
  return (
    <View style={styles.root}>
      {/* headerTitle, not title: in a tab, title would rename the tab's label too. */}
      <Stack.Screen options={{ headerTitle: props.title }} />
      <Text style={styles.title}>{props.title}</Text>
      <Text style={styles.body}>Coming in #{props.issue}.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.sm,
    padding: SPACE.xl,
    backgroundColor: COLORS.bg,
  },
  title: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  body: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
});
