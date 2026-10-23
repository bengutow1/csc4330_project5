import { errorMessage, useUpdateKitchen, useWhoAmI } from "@fries-in-the-bag/api";
import { Stack } from "expo-router";
import { useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Header } from "@/components/header";
import { KitchenForm } from "@/components/kitchen-form";
import { draftOf } from "@/lib/kitchen";
import { useSetup } from "@/lib/setup";
import { COLORS, SPACE, TYPE } from "@/theme";

/**
 * KitchenScreen is O3: what the player cooks with, how they eat, what they can't eat, the staples
 * they keep and how hard the roast hits. Save stores the whole kitchen and opens the tabs.
 */
export function KitchenScreen() {
  const insets = useSafeAreaInsets();
  const setup = useSetup();
  // The form opens on the account's kitchen, which a new account has on its defaults. The guard
  // opens this screen only once WhoAmI holds the account.
  const kitchen = useWhoAmI().data?.account?.kitchen;
  const [draft, setDraft] = useState(() => draftOf(kitchen));
  const save = useUpdateKitchen();

  const submit = () => save.mutate({ kitchen: draft }, { onSuccess: () => setup.finish() });

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={styles.content}>
        <Header root title="Your kitchen" subtitle="So every recipe is one you can make." />
        <KitchenForm draft={draft} onChange={setDraft} />
      </ScrollView>
      {save.isError ? <Text style={styles.failed}>{errorMessage(save.error)}</Text> : null}
      <BottomBar>
        <Button
          label="Save my kitchen"
          pendingLabel="Saving…"
          pending={save.isPending}
          onPress={submit}
        />
      </BottomBar>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  // No top padding: the header sits right under the status bar, as the design draws it.
  content: { gap: SPACE.xxl, paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xxl },
  failed: { ...TYPE.small, color: COLORS.ketchupInk, paddingHorizontal: SPACE.lg },
});
