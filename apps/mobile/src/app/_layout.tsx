import { ApiProvider, createApiTransport, makeQueryClient } from "@fries-in-the-bag/api";
import { Stack } from "expo-router";
import { useMemo } from "react";
import { API_URL } from "@/lib/api-url";
import { COLORS } from "@/theme";

// A deep link straight to a screen still gets the home screen behind it to go back to.
export const unstable_settings = { anchor: "index" };

export default function RootLayout() {
  // Built once: a transport or query client rebuilt on render drops the whole cache. There is no
  // toast yet, so each screen shows its own errors.
  const transport = useMemo(() => createApiTransport({ baseUrl: API_URL }), []);
  const queryClient = useMemo(() => makeQueryClient(), []);

  return (
    <ApiProvider transport={transport} queryClient={queryClient}>
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: COLORS.bg },
          headerTintColor: COLORS.text,
          contentStyle: { backgroundColor: COLORS.bg },
        }}
      >
        {/* The prototype draws its own chrome. The title still labels the back button. */}
        <Stack.Screen name="index" options={{ headerShown: false, title: "Home" }} />
      </Stack>
    </ApiProvider>
  );
}
