import { Stack } from "expo-router";
import { SpotsProvider } from "@/screens/fries/spots-context";
import { COLORS } from "@/theme";

export default function FriesLayout() {
  return (
    <SpotsProvider>
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: COLORS.bg },
          headerTintColor: COLORS.ink,
          headerBackButtonDisplayMode: "minimal",
        }}
      />
    </SpotsProvider>
  );
}
