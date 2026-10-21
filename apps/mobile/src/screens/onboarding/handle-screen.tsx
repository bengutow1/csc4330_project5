import { errorMessage, fieldErrors, useCreateAccount } from "@fries-in-the-bag/api";
import { Stack } from "expo-router";
import { useState } from "react";
import { Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// A few food emoji to pick from, until #14 draws the real grid.
const AVATARS = ["🍟", "🍔", "🌮", "🍕", "🍳", "🥗"] as const;

/**
 * HandleScreen is the plain first launch: a handle and an emoji, then the server makes the device
 * account and signs this phone in, and the root layout moves on to the tabs. #14 replaces it with
 * the designed onboarding.
 */
export function HandleScreen() {
  const [handle, setHandle] = useState("");
  const [avatar, setAvatar] = useState<string>(AVATARS[0]);
  const create = useCreateAccount();
  const invalid = fieldErrors(create.error);
  // A failure that belongs to no input (the server out of reach, too many tries) goes under the
  // button instead.
  const failed = create.isError && Object.keys(invalid).length === 0;
  const submit = () => create.mutate({ handle, avatar });

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerTitle: "Pick a handle" }} />
      <Text style={styles.body}>It's your name on the board. Pick an emoji to go with it.</Text>
      <TextInput
        accessibilityLabel="Handle"
        autoCapitalize="none"
        autoCorrect={false}
        onChangeText={setHandle}
        onSubmitEditing={submit}
        placeholder="chef_maya"
        placeholderTextColor={COLORS.muted}
        returnKeyType="go"
        style={styles.input}
        value={handle}
      />
      {invalid.handle ? <Text style={styles.error}>{invalid.handle}</Text> : null}
      <View style={styles.avatars}>
        {AVATARS.map((emoji) => (
          <Pressable
            accessibilityLabel={`Pick ${emoji} as your avatar`}
            accessibilityRole="button"
            accessibilityState={{ selected: emoji === avatar }}
            key={emoji}
            onPress={() => setAvatar(emoji)}
            style={[styles.avatar, emoji === avatar && styles.picked]}
          >
            <Text style={styles.emoji}>{emoji}</Text>
          </Pressable>
        ))}
      </View>
      {invalid.avatar ? <Text style={styles.error}>{invalid.avatar}</Text> : null}
      <Pressable
        accessibilityRole="button"
        disabled={create.isPending}
        onPress={submit}
        style={({ pressed }) => [styles.button, pressed && styles.pressed]}
      >
        <Text style={styles.buttonText}>{create.isPending ? "Starting…" : "Start cooking"}</Text>
      </Pressable>
      {failed ? <Text style={styles.error}>{errorMessage(create.error)}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, padding: SPACE.xl, gap: SPACE.md, backgroundColor: COLORS.bg },
  body: { ...TYPE.body, color: COLORS.muted },
  input: {
    ...TYPE.body,
    height: 52,
    paddingHorizontal: SPACE.lg,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
    color: COLORS.ink,
  },
  error: { ...TYPE.small, color: COLORS.ketchupInk },
  avatars: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.sm },
  avatar: {
    width: 52,
    height: 52,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    borderColor: COLORS.line,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
  },
  picked: { borderColor: COLORS.ketchup, backgroundColor: COLORS.ketchupTint },
  emoji: { fontSize: 26 },
  button: {
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
