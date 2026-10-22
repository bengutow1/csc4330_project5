import Ionicons from "@expo/vector-icons/Ionicons";
import { errorMessage, fieldErrors, useEditIngredients, useWhoAmI } from "@fries-in-the-bag/api";
import { useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { Button } from "@/components/button";
import { OnChip, RemovableChip } from "@/components/chip";
import { Sheet } from "@/components/sheet";
import { categoryLook, groupByCategory } from "@/lib/categories";
import {
  type Item,
  listErrors,
  listToSave,
  refusalOf,
  suggest,
  withIngredient,
  withoutIngredient,
} from "@/lib/ingredients";
import { STAPLES } from "@/lib/kitchen";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The chips scroll inside the sheet past this height, so Save stays in reach.
const LIST_HEIGHT = 280;

/**
 * IngredientSheet fixes a verdict's list: every ingredient as a chip under its category, a cross
 * to take one out, and a field that suggests common ingredients as the player types. The kitchen's
 * staples show as on below the list, since they count without being scanned. Save sends the whole
 * list, a name still in the field included. The score never moves: the server judged the fridge
 * once, and the list only shapes the recipes.
 */
export function IngredientSheet(props: {
  scanId: bigint;
  /** The list as the verdict has it, which the sheet starts from. */
  ingredients: readonly Item[];
  onClose: () => void;
}) {
  const edit = useEditIngredients();
  const kitchen = useWhoAmI().data?.account?.kitchen;
  const staples = STAPLES.filter((staple) => kitchen?.staples.includes(staple.value));
  const [draft, setDraft] = useState<readonly Item[]>(props.ingredients);
  const [typed, setTyped] = useState("");
  // The list the last Save sent. A refusal names ingredients by their place in it, so it shows only
  // while the draft is still that list: an edit after it would move its messages onto others.
  const [sent, setSent] = useState<readonly Item[] | null>(null);
  const refusal = refusalOf(draft, sent, edit.error);
  const errors = listErrors(fieldErrors(refusal));
  // A refusal no field explains (the server out of reach) goes under Save.
  const failed = refusal !== null && errors.list === undefined && errors.items.size === 0;

  const add = (name: string) => {
    setDraft((list) => withIngredient(list, name));
    setTyped("");
  };
  const save = () => {
    // The draft becomes what was sent, so a refusal lines up with the chips on screen.
    const list = listToSave(draft, typed);
    setDraft(list);
    setTyped("");
    setSent(list);
    edit.mutate({ scanId: props.scanId, ingredients: [...list] }, { onSuccess: props.onClose });
  };

  return (
    <Sheet visible title="Fix the list" onClose={props.onClose}>
      <Text style={styles.note}>Fixing the list changes your recipes, not your score.</Text>
      <ScrollView style={styles.list} contentContainerStyle={styles.listContent}>
        {draft.length === 0 ? <Text style={styles.empty}>Nothing on the list.</Text> : null}
        {groupByCategory(draft).map((group) => (
          <View key={group.category} style={styles.group}>
            <Text style={styles.groupTitle}>{categoryLook(group.category).label}</Text>
            <View style={styles.chips}>
              {group.items.map(({ item, index }) => (
                <RemovableChip
                  key={`${index}-${item.name}`}
                  label={item.name}
                  onRemove={() => setDraft((list) => withoutIngredient(list, index))}
                />
              ))}
            </View>
            {group.items.map(({ item, index }) => {
              const message = errors.items.get(index);
              return message === undefined ? null : (
                <Text key={`error-${index}`} style={styles.error}>
                  {`${item.name}: ${message}`}
                </Text>
              );
            })}
          </View>
        ))}
        {staples.length === 0 ? null : (
          <View style={styles.group}>
            <Text style={styles.groupTitle}>Staples</Text>
            <Text style={styles.hint}>Always on, from your kitchen.</Text>
            <View style={styles.chips}>
              {staples.map((staple) => (
                <OnChip key={staple.value} label={staple.label} />
              ))}
            </View>
          </View>
        )}
      </ScrollView>
      {errors.list === undefined ? null : <Text style={styles.error}>{errors.list}</Text>}
      <TextInput
        accessibilityLabel="Add an ingredient"
        autoCorrect={false}
        onChangeText={setTyped}
        onSubmitEditing={() => add(typed)}
        placeholder="Add an ingredient"
        placeholderTextColor={COLORS.muted}
        returnKeyType="done"
        style={styles.input}
        value={typed}
      />
      {suggest(typed, draft).map((common) => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Add ${common.name}`}
          key={common.name}
          onPress={() => add(common.name)}
          style={({ pressed }) => [styles.suggestion, pressed && styles.pressed]}
        >
          <Ionicons name={categoryLook(common.category).icon} size={18} color={COLORS.muted} />
          <Text style={styles.suggestionName}>{common.name}</Text>
        </Pressable>
      ))}
      {failed ? <Text style={styles.error}>{errorMessage(refusal)}</Text> : null}
      <Button label="Save" pending={edit.isPending} pendingLabel="Saving…" onPress={save} />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  note: { ...TYPE.body, color: COLORS.muted },
  list: { maxHeight: LIST_HEIGHT },
  listContent: { gap: SPACE.lg },
  empty: { ...TYPE.body, color: COLORS.muted },
  group: { gap: SPACE.sm },
  groupTitle: { ...TYPE.bodyStrong, color: COLORS.ink },
  hint: { ...TYPE.caption, color: COLORS.muted },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.sm },
  error: { ...TYPE.small, color: COLORS.ketchupInk },
  input: {
    ...TYPE.body,
    height: 48,
    paddingHorizontal: SPACE.lg,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
    color: COLORS.ink,
  },
  suggestion: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.sm,
    minHeight: 44,
    paddingHorizontal: SPACE.sm,
  },
  pressed: { backgroundColor: COLORS.sunken, borderRadius: RADII.md },
  suggestionName: { ...TYPE.body, color: COLORS.ink },
});
