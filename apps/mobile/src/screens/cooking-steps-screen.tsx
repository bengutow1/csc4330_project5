import { isNotFound, type Recipe, useRecipe } from "@fries-in-the-bag/api";
import { useKeepAwake } from "expo-keep-awake";
import { Stack, useFocusEffect, useRouter } from "expo-router";
import { type ReactNode, useCallback, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { CookingTimer } from "@/components/cooking-timer";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/skeleton";
import { parseId } from "@/lib/route-id";
import { COLORS, SPACE, TYPE } from "@/theme";

/** C3 reads the same recipe as C2, with one foreground instruction at a time. */
export function CookingStepsScreen({ id }: { id: string | undefined }) {
  const router = useRouter();
  const recipeId = parseId(id);
  const recipe = useRecipe(recipeId);
  const found = recipe.data?.recipe;
  const [focused, setFocused] = useState(false);
  useFocusEffect(
    useCallback(() => {
      setFocused(true);
      return () => setFocused(false);
    }, []),
  );

  let body: ReactNode;
  if (
    recipeId === undefined ||
    isNotFound(recipe.error) ||
    (recipe.data !== undefined && found === undefined)
  ) {
    body = (
      <EmptyState
        title="No recipe here"
        line="This recipe doesn't exist, or it isn't yours."
        action={{ label: "Scan your fridge", onPress: () => router.dismissTo("/") }}
      />
    );
  } else if (found !== undefined) {
    body = focused ? <CookingSteps key={String(found.recipeId)} recipe={found} /> : null;
  } else if (recipe.isError) {
    body = (
      <ErrorState
        title="Couldn't load the recipe"
        error={recipe.error}
        retrying={recipe.isFetching}
        onRetry={() => void recipe.refetch()}
      />
    );
  } else {
    body = (
      <View accessibilityLabel="Loading cooking steps" style={styles.content}>
        <Skeleton height={40} />
        <Skeleton height={180} />
        <Skeleton height={120} />
      </View>
    );
  }
  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerTitle: "Cook step by step" }} />
      {body}
    </View>
  );
}

function CookingSteps({ recipe }: { recipe: Recipe }) {
  useKeepAwake("cooking-steps", { suppressDeactivateWarnings: true });
  const router = useRouter();
  const [index, setIndex] = useState(0);
  // Older saved recipes and older servers still supply plain instructions.
  const steps =
    recipe.cookingSteps.length > 0
      ? recipe.cookingSteps
      : recipe.steps.map((instruction) => ({ instruction, timerSeconds: undefined }));
  const step = steps[index];
  if (step === undefined)
    return (
      <EmptyState
        title="No steps here"
        line="Open the recipe for its details."
        action={{ label: "Back to recipe", onPress: () => router.back() }}
      />
    );
  const last = index === steps.length - 1;

  return (
    <>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.recipe}>{recipe.title}</Text>
        <Text accessibilityRole="header" style={styles.progress}>
          {`Step ${index + 1} of ${steps.length}`}
        </Text>
        <Text style={styles.instruction}>{step.instruction}</Text>
        {step.timerSeconds === undefined ? null : (
          <CookingTimer key={`${index}:${step.timerSeconds}`} seconds={step.timerSeconds} />
        )}
      </ScrollView>
      <BottomBar>
        <Button
          label={last ? "I made it" : "Next step"}
          onPress={() =>
            last
              ? router.push({
                  pathname: "/plate/[recipeId]",
                  params: { recipeId: String(recipe.recipeId) },
                })
              : setIndex(index + 1)
          }
        />
        <Button
          label={index === 0 ? "Back to recipe" : "Previous step"}
          variant="secondary"
          onPress={() => (index === 0 ? router.back() : setIndex(index - 1))}
        />
      </BottomBar>
    </>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { flexGrow: 1, padding: SPACE.xl, gap: SPACE.xl },
  recipe: { ...TYPE.bodyStrong, color: COLORS.muted },
  progress: { ...TYPE.heading, color: COLORS.ketchup },
  instruction: { ...TYPE.title, fontSize: 32, lineHeight: 44, color: COLORS.ink },
});
