import { Stack, useRouter } from "expo-router";
import { StyleSheet, Text, View } from "react-native";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Logo } from "@/components/logo";
import { COLORS, SPACE, TYPE } from "@/theme";

/** WelcomeScreen is a first launch's first screen (O1): the logo, the pitch, and the way in. */
export function WelcomeScreen() {
  const router = useRouter();

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerShown: false }} />
      <View style={styles.body}>
        <Logo />
        <Text style={styles.pitch}>
          “There's nothing to eat” becomes a decision in under a minute.
        </Text>
      </View>
      <BottomBar>
        <Button label="Get started" onPress={() => router.push("/onboarding/handle")} />
      </BottomBar>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  body: { flex: 1, justifyContent: "center", gap: SPACE.xl, padding: SPACE.xl },
  pitch: { ...TYPE.heading, color: COLORS.ink },
});
