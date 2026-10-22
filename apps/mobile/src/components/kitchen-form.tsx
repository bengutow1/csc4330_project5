import { Pressable, StyleSheet, Text, View } from "react-native";
import { FilterChip } from "@/components/chip";
import { FormSection } from "@/components/form-section";
import {
  ALLERGIES,
  DIETS,
  EQUIPMENT,
  type KitchenDraft,
  ROAST_LEVELS,
  STAPLES,
  toggled,
} from "@/lib/kitchen";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

/**
 * KitchenForm asks what the player cooks with, how they eat, what they can't eat, the staples they
 * keep and how hard the roast hits. Onboarding (O3) and settings (P3) both ask it; each screen
 * holds the draft and saves it.
 */
export function KitchenForm(props: {
  draft: KitchenDraft;
  /** Takes an update rather than a value, so two quick taps never build on the same draft. */
  onChange: (update: (draft: KitchenDraft) => KitchenDraft) => void;
}) {
  const { draft, onChange } = props;

  return (
    <>
      <FormSection title="What you cook with">
        <View style={styles.wrap}>
          {EQUIPMENT.map((choice) => (
            <FilterChip
              key={choice.value}
              label={choice.label}
              selected={draft.equipment.includes(choice.value)}
              onPress={() =>
                onChange((d) => ({ ...d, equipment: toggled(d.equipment, choice.value) }))
              }
            />
          ))}
        </View>
      </FormSection>
      <FormSection title="How you eat">
        <View style={styles.wrap}>
          {DIETS.map((choice) => (
            <FilterChip
              key={choice.value}
              label={choice.label}
              selected={draft.diet === choice.value}
              onPress={() => onChange((d) => ({ ...d, diet: choice.value }))}
            />
          ))}
        </View>
      </FormSection>
      <FormSection title="Allergies" note="A recipe with one of these never reaches you.">
        <View style={styles.wrap}>
          {ALLERGIES.map((choice) => (
            <FilterChip
              key={choice.value}
              label={choice.label}
              selected={draft.allergies.includes(choice.value)}
              onPress={() =>
                onChange((d) => ({ ...d, allergies: toggled(d.allergies, choice.value) }))
              }
            />
          ))}
        </View>
      </FormSection>
      <FormSection
        title="Always in your kitchen"
        note="Staples count as on hand without a scan. Turn off what you don't keep."
      >
        <View style={styles.wrap}>
          {STAPLES.map((choice) => (
            <FilterChip
              key={choice.value}
              label={choice.label}
              selected={draft.staples.includes(choice.value)}
              onPress={() => onChange((d) => ({ ...d, staples: toggled(d.staples, choice.value) }))}
            />
          ))}
        </View>
      </FormSection>
      <FormSection title="Roast level" note="How hard the verdict goes on your fridge.">
        {ROAST_LEVELS.map((level) => (
          <RoastChoice
            key={level.value}
            label={level.label}
            sample={level.sample}
            selected={draft.roastLevel === level.value}
            onPress={() => onChange((d) => ({ ...d, roastLevel: level.value }))}
          />
        ))}
      </FormSection>
    </>
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
});
