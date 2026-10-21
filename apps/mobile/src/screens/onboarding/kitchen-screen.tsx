import {
  type Allergy,
  type Diet,
  type Equipment,
  errorMessage,
  type RoastLevel,
  type Staple,
  useUpdateKitchen,
  useWhoAmI,
} from "@fries-in-the-bag/api";
import { Stack } from "expo-router";
import { type ReactNode, useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { FilterChip } from "@/components/chip";
import { Header } from "@/components/header";
import { ALLERGIES, DIETS, EQUIPMENT, ROAST_LEVELS, STAPLES, toggled } from "@/lib/kitchen";
import { useSetup } from "@/lib/setup";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

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
  const [equipment, setEquipment] = useState<Equipment[]>(() => kitchen?.equipment ?? []);
  const [diet, setDiet] = useState<Diet | undefined>(() => kitchen?.diet);
  const [allergies, setAllergies] = useState<Allergy[]>(() => kitchen?.allergies ?? []);
  const [staples, setStaples] = useState<Staple[]>(() => kitchen?.staples ?? []);
  const [roastLevel, setRoastLevel] = useState<RoastLevel | undefined>(() => kitchen?.roastLevel);
  const save = useUpdateKitchen();

  const submit = () =>
    save.mutate(
      { kitchen: { roastLevel, equipment, diet, allergies, staples } },
      { onSuccess: () => setup.finish() },
    );

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
        <Header root title="Your kitchen" subtitle="So every recipe is one you can make." />
        <Section title="What you cook with">
          <View style={styles.wrap}>
            {EQUIPMENT.map((choice) => (
              <FilterChip
                key={choice.value}
                label={choice.label}
                selected={equipment.includes(choice.value)}
                onPress={() => setEquipment((list) => toggled(list, choice.value))}
              />
            ))}
          </View>
        </Section>
        <Section title="How you eat">
          <View style={styles.wrap}>
            {DIETS.map((choice) => (
              <FilterChip
                key={choice.value}
                label={choice.label}
                selected={diet === choice.value}
                onPress={() => setDiet(choice.value)}
              />
            ))}
          </View>
        </Section>
        <Section title="Allergies" note="A recipe with one of these never reaches you.">
          <View style={styles.wrap}>
            {ALLERGIES.map((choice) => (
              <FilterChip
                key={choice.value}
                label={choice.label}
                selected={allergies.includes(choice.value)}
                onPress={() => setAllergies((list) => toggled(list, choice.value))}
              />
            ))}
          </View>
        </Section>
        <Section
          title="Always in your kitchen"
          note="Staples count as on hand without a scan. Turn off what you don't keep."
        >
          <View style={styles.wrap}>
            {STAPLES.map((choice) => (
              <FilterChip
                key={choice.value}
                label={choice.label}
                selected={staples.includes(choice.value)}
                onPress={() => setStaples((list) => toggled(list, choice.value))}
              />
            ))}
          </View>
        </Section>
        <Section title="Roast level" note="How hard the verdict goes on your fridge.">
          {ROAST_LEVELS.map((level) => (
            <RoastChoice
              key={level.value}
              label={level.label}
              sample={level.sample}
              selected={roastLevel === level.value}
              onPress={() => setRoastLevel(level.value)}
            />
          ))}
        </Section>
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

/** Section is one question on the form: its heading, a line on why it matters, its choices. */
function Section(props: { title: string; note?: string; children: ReactNode }) {
  return (
    <View style={styles.section}>
      <View style={styles.titles}>
        <Text accessibilityRole="header" style={styles.heading}>
          {props.title}
        </Text>
        {props.note ? <Text style={styles.note}>{props.note}</Text> : null}
      </View>
      {props.children}
    </View>
  );
}

/** RoastChoice is one roast level with a sample of what it sounds like. */
function RoastChoice(props: {
  label: string;
  sample: string;
  selected: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="radio"
      accessibilityLabel={props.label}
      accessibilityHint={props.sample}
      accessibilityState={{ selected: props.selected }}
      onPress={props.onPress}
      style={[styles.roast, props.selected && styles.roastSelected]}
    >
      <Text style={styles.roastLabel}>{props.label}</Text>
      <Text style={styles.sample}>“{props.sample}”</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.xxl, padding: SPACE.lg, paddingBottom: SPACE.xxl },
  section: { gap: SPACE.md },
  titles: { gap: SPACE.xs },
  heading: { ...TYPE.heading, color: COLORS.ink },
  note: { ...TYPE.caption, color: COLORS.muted },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.sm },
  roast: {
    gap: SPACE.xs,
    padding: SPACE.lg,
    borderWidth: 2,
    borderColor: COLORS.line,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
  },
  roastSelected: { borderColor: COLORS.ketchup, backgroundColor: COLORS.ketchupTint },
  roastLabel: { ...TYPE.bodyStrong, color: COLORS.ink },
  sample: { ...TYPE.caption, color: COLORS.muted },
  failed: { ...TYPE.small, color: COLORS.ketchupInk, paddingHorizontal: SPACE.lg },
});
