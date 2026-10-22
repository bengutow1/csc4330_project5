import { type Href, Stack, useRouter } from "expo-router";
import { StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { IconButton, type IconName } from "@/components/icon-button";
import { COLORS, SPACE, TYPE } from "@/theme";

/**
 * PlaceholderScreen holds a route until its issue builds the real screen, so any screen can link
 * to any other before it exists. Its owner replaces the route's mount, not this file.
 */
export function PlaceholderScreen(props: {
  title: string;
  issue: number;
  /** A way on that the real screen keeps, drawn at the top right: the profile's settings gear. */
  action?: { icon: IconName; label: string; href: Href };
}) {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const action = props.action;

  return (
    <View style={styles.root}>
      {/* headerTitle, not title: in a tab, title would rename the tab's label too. */}
      <Stack.Screen options={{ headerTitle: props.title }} />
      {action ? (
        <View style={[styles.action, { top: insets.top + SPACE.xs }]}>
          <IconButton
            icon={action.icon}
            label={action.label}
            onPress={() => router.push(action.href)}
          />
        </View>
      ) : null}
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
  action: { position: "absolute", right: SPACE.sm },
  title: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  body: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
});
