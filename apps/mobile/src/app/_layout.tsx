import {
  ApiProvider,
  createApiTransport,
  isUnauthenticated,
  makeQueryClient,
  useWhoAmI,
} from "@fries-in-the-bag/api";
import { Stack, useSegments } from "expo-router";
import { useMemo, useState } from "react";
import { AppState } from "react-native";
import { API_URL } from "@/lib/api-url";
import { launchCoverShows } from "@/lib/launch";
import { type Setup, SetupContext } from "@/lib/setup";
import { LaunchCover } from "@/screens/launch-cover";
import { COLORS } from "@/theme";

// A deep link straight to a screen still gets the tabs behind it to go back to.
export const unstable_settings = { anchor: "(tabs)" };

export default function RootLayout() {
  // Built once: a transport or query client rebuilt on render drops the whole cache. There is no
  // toast yet, so each screen shows its own errors.
  const transport = useMemo(() => createApiTransport({ baseUrl: API_URL }), []);
  const queryClient = useMemo(() => makeQueryClient(), []);

  return (
    <ApiProvider transport={transport} queryClient={queryClient} appState={AppState}>
      <RootStack />
    </ApiProvider>
  );
}

/**
 * RootStack sits inside ApiProvider so it can ask whose phone this is, and opens each route only
 * at the right stage: the welcome and the handle without an account, the kitchen between making
 * the account and saving it, everything else after.
 */
function RootStack() {
  const whoAmI = useWhoAmI();
  const [route] = useSegments();
  // Only this launch's onboarding sets it. A player who quits on the kitchen screen comes back to
  // the tabs, with the kitchen on its defaults.
  const [settingUp, setSettingUp] = useState(false);
  const setup = useMemo<Setup>(
    () => ({ start: () => setSettingUp(true), finish: () => setSettingUp(false) }),
    [],
  );
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
    <SetupContext.Provider value={setup}>
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
        </Stack.Protected>
        {/* The one route open while setting up: the handle screen closes as the account arrives,
            the tabs are shut, so the guards land here. */}
        <Stack.Protected guard={signedIn && settingUp}>
          <Stack.Screen name="onboarding/kitchen" />
        </Stack.Protected>
        {/* Every route that needs an account. A route in none of these lists is open to anyone,
            and goes in OPEN_ROUTES (src/lib/launch.ts) so the launch cover leaves it alone. */}
        <Stack.Protected guard={signedIn && !settingUp}>
          <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
          <Stack.Screen name="scan/[id]" />
          <Stack.Screen name="cook/[scanId]" />
          <Stack.Screen name="combo/[id]/recipes" />
          <Stack.Screen name="combo/[id]/bring" />
          <Stack.Screen name="recipe/[id]" />
          <Stack.Screen name="recipe/[id]/steps" />
          <Stack.Screen name="plate/[recipeId]" />
          <Stack.Screen name="fries" options={{ headerShown: false }} />
          <Stack.Screen name="settings" />
          <Stack.Screen name="crews" />
          <Stack.Screen name="combo/[id]/reveal" />
        </Stack.Protected>
      </Stack>
      {covered ? (
        <LaunchCover
          failed={whoAmI.isError}
          retrying={whoAmI.isFetching}
          onRetry={() => void whoAmI.refetch()}
        />
      ) : null}
    </SetupContext.Provider>
  );
}
