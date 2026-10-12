import { isUnavailable, useReady } from "@fries-in-the-bag/api";
import { Stack } from "expo-router";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { API_URL } from "@/lib/api-url";
import { COLORS } from "@/theme";

/**
 * StatusScreen asks the api whether it can serve, through the same transport, query client and
 * hook every other screen uses, so it doubles as the check that the whole stack is wired up.
 */
export function StatusScreen() {
  const ready = useReady();

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ title: "Server status" }} />
      {ready.isPending ? (
        <View style={styles.row}>
          <ActivityIndicator color={COLORS.gold} />
          <Text style={styles.body}>Checking the server…</Text>
        </View>
      ) : ready.isError ? (
        <Text style={[styles.verdict, { color: COLORS.red }]}>
          {/* Unavailable means the server answered and said no; anything else, it never answered. */}
          {isUnavailable(ready.error)
            ? "The server is up but not ready"
            : "Couldn't reach the server"}
        </Text>
      ) : (
        <Text style={[styles.verdict, { color: COLORS.green }]}>Server ready</Text>
      )}
      {__DEV__ ? <Text style={styles.dim}>{API_URL}</Text> : null}
      <Pressable
        accessibilityRole="button"
        disabled={ready.isFetching}
        onPress={() => void ready.refetch()}
        style={({ pressed }) => [styles.button, pressed && styles.pressed]}
      >
        <Text style={styles.buttonText}>{ready.isFetching ? "Checking…" : "Check again"}</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, padding: 24, gap: 16, backgroundColor: COLORS.bg },
  row: { flexDirection: "row", alignItems: "center", gap: 12 },
  body: { color: COLORS.text, fontSize: 16 },
  verdict: { fontSize: 22, fontWeight: "800" },
  dim: { color: COLORS.dim, fontSize: 13 },
  button: {
    backgroundColor: COLORS.gold,
    paddingVertical: 15,
    borderRadius: 16,
    alignItems: "center",
  },
  pressed: { opacity: 0.7 },
  buttonText: { color: COLORS.bg, fontSize: 17, fontWeight: "800" },
});
