import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

/**
 * LaunchCover sits over every route until the app knows whose phone this is, so a returning
 * player never sees onboarding flash past. When the server can't be reached it says so, rather
 * than reading the silence as "no account" and offering to make a second one.
 */
export function LaunchCover(props: { failed: boolean; retrying: boolean; onRetry: () => void }) {
  return (
    // Modal to VoiceOver as well as to the eye: the routes underneath stay mounted.
    <View accessibilityViewIsModal style={[StyleSheet.absoluteFill, styles.root]}>
      {props.failed ? (
        <>
          <Text style={styles.title}>Couldn't reach the server</Text>
          <Text style={styles.body}>Check your connection, then try again.</Text>
          <Pressable
            accessibilityRole="button"
            disabled={props.retrying}
            onPress={props.onRetry}
            style={({ pressed }) => [styles.button, pressed && styles.pressed]}
          >
            <Text style={styles.buttonText}>{props.retrying ? "Trying…" : "Try again"}</Text>
          </Pressable>
        </>
      ) : (
        <ActivityIndicator color={COLORS.ketchup} />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.md,
    padding: SPACE.xl,
    backgroundColor: COLORS.bg,
  },
  title: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  body: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
  button: {
    alignSelf: "stretch",
    height: 52,
    marginTop: SPACE.sm,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.ketchup,
    alignItems: "center",
    justifyContent: "center",
  },
  pressed: { backgroundColor: COLORS.ketchupInk },
  buttonText: { ...TYPE.button, color: COLORS.surface },
});
