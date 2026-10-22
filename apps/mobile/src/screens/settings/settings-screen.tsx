import { errorMessage, useDeleteAccount, useUpdateKitchen, useWhoAmI } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useState } from "react";
import { Linking, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { FormSection } from "@/components/form-section";
import { Header } from "@/components/header";
import { KitchenForm } from "@/components/kitchen-form";
import { draftOf } from "@/lib/kitchen";
import { lastWriteError } from "@/lib/last-write";
import { COLORS, SPACE, TYPE } from "@/theme";

/**
 * SettingsScreen is P3: everything onboarding asked, to change, the phone's location setting, and
 * deleting the player's data for good. Save stores the whole kitchen and goes back. Delete asks
 * first, and once the account is gone the guards open onboarding.
 */
export function SettingsScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  // The guards open this screen only with an account, so the kitchen is already loaded.
  const kitchen = useWhoAmI().data?.account?.kitchen;
  const [draft, setDraft] = useState(() => draftOf(kitchen));
  const [confirming, setConfirming] = useState(false);
  const save = useUpdateKitchen();
  const remove = useDeleteAccount();
  // The write tried last reports above Save. A failed delete closes its dialog to show why.
  const failed = lastWriteError(save, remove);

  const submit = () => save.mutate({ kitchen: draft }, { onSuccess: () => router.back() });
  const deleteData = () => remove.mutate({}, { onError: () => setConfirming(false) });

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={styles.content}>
        <Header title="Settings" onBack={() => router.back()} />
        <KitchenForm draft={draft} onChange={setDraft} />
        <FormSection
          title="Location"
          note="Fries mode asks for it to find spots nearby. Your phone keeps the setting."
        >
          <Button
            label="Open phone settings"
            variant="secondary"
            onPress={() => void Linking.openSettings()}
          />
        </FormSection>
        <FormSection title="Your data" note="Your account, scans, plates, fry runs and points.">
          <Button
            label="Delete my data"
            variant="destructive"
            onPress={() => setConfirming(true)}
          />
        </FormSection>
      </ScrollView>
      {failed ? <Text style={styles.failed}>{errorMessage(failed)}</Text> : null}
      <BottomBar>
        <Button label="Save" pendingLabel="Saving…" pending={save.isPending} onPress={submit} />
      </BottomBar>
      <ConfirmDialog
        visible={confirming}
        title="Delete your data?"
        message="Your account, scans, plates, fry runs and points go for good. This can't be undone."
        confirmLabel="Delete"
        pendingLabel="Deleting…"
        pending={remove.isPending}
        destructive
        onConfirm={deleteData}
        onCancel={() => setConfirming(false)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  // No top padding: the header sits right under the status bar, as the design draws it.
  content: { gap: SPACE.xxl, paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xxl },
  failed: { ...TYPE.small, color: COLORS.ketchupInk, paddingHorizontal: SPACE.lg },
});
