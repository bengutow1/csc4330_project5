import Ionicons from "@expo/vector-icons/Ionicons";
import { errorMessage } from "@fries-in-the-bag/api";
import { StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { COLORS, SPACE, TYPE } from "@/theme";

const ICON = 32;

/**
 * ErrorState is what a screen shows when its data failed to load: what failed, in the screen's own
 * words ("Couldn't load the board"), the server's sentence for why, and Try again. Its words never
 * borrow the empty state's: an outage is not an empty board.
 */
export function ErrorState(props: {
  title: string;
  error: unknown;
  onRetry: () => void;
  /** True while the retry is in flight. */
  retrying?: boolean;
}) {
  return (
    <View style={styles.root}>
      <Ionicons name="alert-circle-outline" size={ICON} color={COLORS.ketchupInk} />
      <Text accessibilityRole="header" style={styles.title}>
        {props.title}
      </Text>
      <Text style={styles.line}>{errorMessage(props.error)}</Text>
      <Button
        label="Try again"
        pendingLabel="Trying again…"
        pending={props.retrying}
        onPress={props.onRetry}
        variant="secondary"
        size="small"
      />
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
