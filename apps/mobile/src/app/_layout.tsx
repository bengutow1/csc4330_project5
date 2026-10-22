import { LilitaOne_400Regular } from "@expo-google-fonts/lilita-one";
import {
  ApiProvider,
  createApiTransport,
  isUnauthenticated,
  makeQueryClient,
  useWhoAmI,
} from "@fries-in-the-bag/api";
import { useFonts } from "expo-font";
import { Stack, useSegments } from "expo-router";
import { useMemo } from "react";
import { AppState } from "react-native";
import { API_URL } from "@/lib/api-url";
import { launchCoverShows } from "@/lib/launch";
import { LaunchCover } from "@/screens/launch-cover";
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
      <RootStack />
    </ApiProvider>
  );
}

/**
 * RootStack sits inside ApiProvider so it can ask whose phone this is, and opens each route only
 * on the right side of the answer: onboarding without an account, everything else with one.
 */
function RootStack() {
  const whoAmI = useWhoAmI();
  const [route] = useSegments();
  // A failed refetch keeps the account it had, so a dropped connection signs nobody out. A dead
  // session does: the query client resets WhoAmI when any call comes back unauthenticated.
  const signedIn = whoAmI.data !== undefined;
  // Only the server saying "no session" means onboarding. Still asking, or never reaching it, is
  // neither answer, so the cover stays up over the guarded routes.
  const covered = launchCoverShows({
    signedIn,
    noSession: isUnauthenticated(whoAmI.error),
    route,
  });

  return (
    <>
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: COLORS.bg },
          headerTintColor: COLORS.ink,
          // The design's back control is the arrow alone, never the previous screen's name.
          headerBackButtonDisplayMode: "minimal",
          contentStyle: { backgroundColor: COLORS.bg },
        }}
      >
        {/* First, so a guard that turns the tabs away lands here. */}
        <Stack.Protected guard={!signedIn}>
          <Stack.Screen name="onboarding/index" />
          <Stack.Screen name="onboarding/handle" />
          <Stack.Screen name="onboarding/kitchen" />
        </Stack.Protected>
        {/* Every route that needs an account. A route in neither list is open to anyone, and goes
            in OPEN_ROUTES (src/lib/launch.ts) so the launch cover leaves it alone. */}
        <Stack.Protected guard={signedIn}>
          <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
          <Stack.Screen name="scan/[id]" />
          <Stack.Screen name="cook/[scanId]" />
          <Stack.Screen name="recipe/[id]" />
          <Stack.Screen name="plate/[recipeId]" />
          <Stack.Screen name="fries/index" />
          <Stack.Screen name="fries/spots" />
          <Stack.Screen name="settings" />
        </Stack.Protected>
      </Stack>
      {covered ? (
        <LaunchCover
          failed={whoAmI.isError}
          retrying={whoAmI.isFetching}
          onRetry={() => void whoAmI.refetch()}
        />
      ) : null}
    </>
  );
}
