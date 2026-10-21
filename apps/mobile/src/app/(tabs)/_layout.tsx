import Ionicons from "@expo/vector-icons/Ionicons";
import { Tabs } from "expo-router/js-tabs";
import type { ComponentProps } from "react";
import { type ColorValue, Text } from "react-native";
import { COLORS, TYPE } from "@/theme";

// A deep link straight to Board or Me still has Snap, the first tab, behind it.
export const unstable_settings = { anchor: "index" };

type IconName = ComponentProps<typeof Ionicons>["name"];

// One tab's options. The design draws the same outline icon whether the tab is active or not: only
// the colour changes, and the label's weight. The accessibility label is set to the name alone,
// or iOS reads the icon's glyph into it (", , Board").
function tab(title: string, icon: IconName) {
  return {
    title,
    tabBarAccessibilityLabel: title,
    tabBarIcon: ({ color }: { color: ColorValue }) => (
      <Ionicons name={icon} size={24} color={color} />
    ),
  };
}

export default function TabLayout() {
  return (
    <Tabs
      screenOptions={{
        // Each tab's screen draws its own title, as the design does.
        headerShown: false,
        tabBarActiveTintColor: COLORS.ketchup,
        tabBarInactiveTintColor: COLORS.muted,
        tabBarStyle: {
          backgroundColor: COLORS.surface,
          borderTopColor: COLORS.line,
          borderTopWidth: 1,
        },
        tabBarLabel: ({ focused, color, children }) => (
          <Text style={[focused ? TYPE.tabActive : TYPE.tab, { color }]}>{children}</Text>
        ),
      }}
    >
      <Tabs.Screen name="index" options={tab("Snap", "camera-outline")} />
      <Tabs.Screen name="board" options={tab("Board", "trophy-outline")} />
      <Tabs.Screen name="me" options={tab("Me", "person-outline")} />
    </Tabs>
  );
}
