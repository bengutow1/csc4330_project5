import { LilitaOne_400Regular } from "@expo-google-fonts/lilita-one";
import { ApiProvider, createApiTransport, makeQueryClient } from "@fries-in-the-bag/api";
import { useFonts } from "expo-font";
import { Stack } from "expo-router";
import { useMemo } from "react";
import { AppState } from "react-native";
import { API_URL } from "@/lib/api-url";
import { COLORS, FONTS } from "@/theme";

// A deep link straight to a screen still gets the tabs behind it to go back to.
export const unstable_settings = { anchor: "(tabs)" };

export default function RootLayout() {
  // Built once: a transport or query client rebuilt on render drops the whole cache. There is no
  // toast yet, so each screen shows its own errors.
  const transport = useMemo(() => createApiTransport({ baseUrl: API_URL }), []);
  const queryClient = useMemo(() => makeQueryClient(), []);
  // Nothing waits on this yet: only the logo uses Lilita One, and no screen draws it so far.
  useFonts({ [FONTS.logo]: LilitaOne_400Regular });

  return (
    <ApiProvider transport={transport} queryClient={queryClient} appState={AppState}>
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: COLORS.bg },
          headerTintColor: COLORS.ink,
          // The design's back control is the arrow alone, never the previous screen's name.
          headerBackButtonDisplayMode: "minimal",
          contentStyle: { backgroundColor: COLORS.bg },
        }}
      >
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
      </Stack>
    </ApiProvider>
  );
}
