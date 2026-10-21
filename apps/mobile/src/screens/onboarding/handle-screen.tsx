import { errorMessage, fieldErrors, useCheckHandle, useCreateAccount } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useState } from "react";
import {
  KeyboardAvoidingView,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Header } from "@/components/header";
import { useSetup } from "@/lib/setup";
import { useDebouncedValue } from "@/lib/use-debounced-value";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The food emoji an avatar is picked from. Each is a single emoji to the server's check.
const AVATARS = ["🍟", "🍔", "🌮", "🍕", "🍳", "🥗", "🍜", "🍣", "🌯", "🥞", "🍩", "🧁"] as const;

// How long typing has to pause before the handle is checked.
const CHECK_DELAY_MS = 300;

// CheckHandle answers taken with a bare false, so the words are the ones CreateAccount uses for
// the same refusal.
const TAKEN = "that handle is taken";

/**
 * HandleScreen is O2: a handle, checked at each pause in typing, and an avatar from a grid of food
 * emoji. Next makes the device account and signs this phone in, and the root layout moves on to
 * the kitchen.
 */
export function HandleScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const setup = useSetup();
  const [handle, setHandle] = useState("");
  const [avatar, setAvatar] = useState<string>(AVATARS[0]);
  const settled = useDebouncedValue(handle, CHECK_DELAY_MS);
  const check = useCheckHandle(settled);
  const create = useCreateAccount();
  const invalid = fieldErrors(create.error);
  // The check speaks only for the handle it was asked about, and the server's answer to Next
  // speaks over it.
  const checked =
    settled === handle
      ? (fieldErrors(check.error).handle ?? (check.data?.available === false ? TAKEN : undefined))
      : undefined;
  const handleError = invalid.handle ?? checked;
  // A failure that belongs to no input (the server out of reach, too many tries) sits above the
  // button instead.
  const failed = create.isError && Object.keys(invalid).length === 0;

  const type = (text: string) => {
    setHandle(text);
    // The last answer to Next was about the handle as it was.
    create.reset();
  };
  const submit = () => {
    // Before the account exists, so the guards never open the tabs ahead of the kitchen.
    setup.start();
    create.mutate({ handle, avatar });
  };

  return (
    <KeyboardAvoidingView
      behavior={process.env.EXPO_OS === "ios" ? "padding" : undefined}
      style={styles.root}
    >
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView
        contentContainerStyle={[styles.content, { paddingTop: insets.top }]}
        keyboardShouldPersistTaps="handled"
      >
        <Header
          title="Pick a handle"
          subtitle="It's your name on the board."
          onBack={() => router.back()}
        />
        <View style={styles.field}>
          <TextInput
            accessibilityLabel="Handle"
            autoCapitalize="none"
            autoCorrect={false}
            onChangeText={type}
            onSubmitEditing={submit}
            placeholder="chef_maya"
            placeholderTextColor={COLORS.muted}
            returnKeyType="next"
            style={styles.input}
            value={handle}
          />
          {handleError ? <Text style={styles.error}>{handleError}</Text> : null}
        </View>
        <View style={styles.section}>
          <Text style={styles.heading}>Pick your avatar</Text>
          <View style={styles.grid}>
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
        </View>
      </ScrollView>
      {failed ? <Text style={styles.failed}>{errorMessage(create.error)}</Text> : null}
      <BottomBar>
        <Button label="Next" pendingLabel="Saving…" pending={create.isPending} onPress={submit} />
      </BottomBar>
    </KeyboardAvoidingView>
  );
}

const AVATAR_SIZE = 64;

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.xxl, padding: SPACE.lg },
  field: { gap: SPACE.sm },
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
  failed: { ...TYPE.small, color: COLORS.ketchupInk, paddingHorizontal: SPACE.lg },
  section: { gap: SPACE.md },
  heading: { ...TYPE.heading, color: COLORS.ink },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.md },
  avatar: {
    width: AVATAR_SIZE,
    height: AVATAR_SIZE,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    borderColor: COLORS.line,
    borderRadius: RADII.xl,
    backgroundColor: COLORS.surface,
  },
  picked: { borderColor: COLORS.ketchup, backgroundColor: COLORS.ketchupTint },
  emoji: { fontSize: 30 },
});
