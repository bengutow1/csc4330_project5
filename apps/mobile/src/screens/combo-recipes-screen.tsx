import {
  isNotFound,
  type Recipe,
  type RecipeFilters,
  useCombo,
  useComboRecipes,
} from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { Skeleton } from "@/components/skeleton";
import { NO_FILTERS } from "@/lib/recipes";
import { parseId } from "@/lib/route-id";
import { RecipeChips, RecipeShelf } from "@/screens/recipes-screen";
import { COLORS, SPACE } from "@/theme";

/** ComboRecipesScreen keeps the server's two shelves and each food's owner together. */
export function ComboRecipesScreen(props: { id: string | undefined }) {
  const id = parseId(props.id);
  const [filters, setFilters] = useState<RecipeFilters>(NO_FILTERS);
  const combo = useCombo(id, false);
  const recipes = useComboRecipes(id, filters);
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const owners = new Map(
    combo.data?.combo?.players.map((player) => [player.accountId, player.handle]),
  );
  const open = (recipe: Recipe) =>
    router.push({
      pathname: "/combo/[id]/bring",
      params: { id: String(id), recipeId: String(recipe.recipeId) },
    });
  const missing = id === undefined || isNotFound(combo.error) || isNotFound(recipes.error);
  return (
    <ScrollView
      style={styles.root}
      contentContainerStyle={[styles.content, { paddingTop: insets.top }]}
    >
      <Stack.Screen options={{ headerShown: false }} />
      <Header
        title="Two fridges, one meal"
        subtitle="Every ingredient brings its own plus one."
        onBack={() => router.back()}
      />
      <RecipeChips filters={filters} onChange={setFilters} />
      <View style={styles.shelves}>
        {missing ? (
          <EmptyState
            title="No Combo here"
            line="This Combo doesn't exist, or you aren't in it."
            action={{ label: "Back to Snap", onPress: () => router.dismissTo("/") }}
          />
        ) : recipes.data === undefined || combo.data === undefined ? (
          recipes.isError || combo.isError ? (
            <ErrorState
              title="Couldn't load Combo recipes"
              error={recipes.error ?? combo.error}
              retrying={recipes.isFetching || combo.isFetching}
              onRetry={() => {
                void recipes.refetch();
                void combo.refetch();
              }}
            />
          ) : (
            <>
              <Skeleton height={140} />
              <Skeleton height={140} />
            </>
          )
        ) : recipes.data.makeItNow.length + recipes.data.oneItemAway.length === 0 ? (
          <EmptyState
            title="Nothing for both fridges yet"
            line="Try different filters, or find something nearby."
            action={{ label: "Find food nearby", onPress: () => router.push("/fries") }}
          />
        ) : (
          <>
            <RecipeShelf
              title="Make it now"
              recipes={recipes.data.makeItNow}
              owners={owners}
              onOpen={open}
            />
            <RecipeShelf
              title="One item away"
              recipes={recipes.data.oneItemAway}
              owners={owners}
              onOpen={open}
            />
          </>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.lg, paddingBottom: SPACE.xl },
  shelves: { gap: SPACE.xl, paddingHorizontal: SPACE.lg },
});
