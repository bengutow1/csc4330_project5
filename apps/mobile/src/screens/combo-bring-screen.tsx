import { isNotFound, useCombo, useRecipe, useWhoAmI } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Button } from "@/components/button";
import { FilterChip } from "@/components/chip";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { Skeleton } from "@/components/skeleton";
import { bringItems } from "@/lib/combo-cook";
import { parseId } from "@/lib/route-id";
import { COLORS, SPACE, TYPE } from "@/theme";

/** ComboBringScreen chooses a kitchen and keeps this phone's packing checks local. */
export function ComboBringScreen(props: { id: string | undefined; recipeId: string | undefined }) {
  const id = parseId(props.id);
  const recipeId = parseId(props.recipeId);
  const combo = useCombo(id, false);
  const recipe = useRecipe(recipeId);
  const me = useWhoAmI();
  const [kitchen, setKitchen] = useState<bigint>();
  const [checked, setChecked] = useState<ReadonlySet<string>>(new Set());
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const players = combo.data?.combo?.players ?? [];
  const viewer = players.find((player) => player.handle === me.data?.account?.handle)?.accountId;
  const dish = recipe.data?.recipe;
  const invalid =
    id === undefined ||
    recipeId === undefined ||
    isNotFound(combo.error) ||
    isNotFound(recipe.error) ||
    (dish !== undefined && dish.comboId !== id);
  const items = bringItems(dish?.ingredientOwners ?? [], viewer, kitchen);
  return (
    <ScrollView
      style={styles.root}
      contentContainerStyle={[styles.content, { paddingTop: insets.top }]}
    >
      <Stack.Screen options={{ headerShown: false }} />
      <Header
        title="Pick a kitchen"
        subtitle={dish?.title ?? "One meal, one place to cook."}
        onBack={() => router.back()}
      />
      <View style={styles.body}>
        {invalid ? (
          <EmptyState
            title="No Combo meal here"
            line="Choose a recipe from your Combo."
            action={{ label: "Back to Snap", onPress: () => router.dismissTo("/") }}
          />
        ) : combo.data === undefined || dish === undefined || me.data === undefined ? (
          combo.isError || recipe.isError || me.isError ? (
            <ErrorState
              title="Couldn't load the kitchen choices"
              error={combo.error ?? recipe.error ?? me.error}
              retrying={combo.isFetching || recipe.isFetching || me.isFetching}
              onRetry={() => {
                void combo.refetch();
                void recipe.refetch();
                void me.refetch();
              }}
            />
          ) : (
            <Skeleton height={200} />
          )
        ) : (
          <>
            {players.map((player) => (
              <FilterChip
                key={String(player.accountId)}
                label={`Cook at @${player.handle}'s kitchen`}
                selected={kitchen === player.accountId}
                onPress={() => {
                  setKitchen(player.accountId);
                  setChecked(new Set());
                }}
              />
            ))}
            {kitchen === undefined ? (
              <Text style={styles.copy}>
                Agree on a kitchen with your partner, then choose it here.
              </Text>
            ) : (
              <>
                <Text style={styles.title}>
                  {kitchen === viewer ? "You're hosting" : "Bring with you"}
                </Text>
                <Text style={styles.copy}>
                  {kitchen === viewer
                    ? "Your partner can check what to bring on their phone."
                    : items.length === 0
                      ? "The kitchen already has the ingredients you need."
                      : "Pack these ingredients from your fridge."}
                </Text>
                {items.map((item) => (
                  <Pressable
                    key={item}
                    accessibilityRole="checkbox"
                    accessibilityLabel={`Pack ${item}`}
                    accessibilityState={{ checked: checked.has(item) }}
                    onPress={() =>
                      setChecked((before) => {
                        const next = new Set(before);
                        if (next.has(item)) next.delete(item);
                        else next.add(item);
                        return next;
                      })
                    }
                    style={styles.item}
                  >
                    <Text style={styles.copy}>
                      {checked.has(item) ? "✓" : "○"} {item}
                    </Text>
                  </Pressable>
                ))}
                <Button
                  label="Cook this recipe"
                  onPress={() =>
                    router.push({ pathname: "/recipe/[id]", params: { id: String(recipeId) } })
                  }
                />
              </>
            )}
          </>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.lg, paddingBottom: SPACE.xl },
  body: { gap: SPACE.lg, paddingHorizontal: SPACE.lg },
  title: { ...TYPE.heading, color: COLORS.ink },
  copy: { ...TYPE.body, color: COLORS.ink },
  item: { paddingVertical: SPACE.md },
});
