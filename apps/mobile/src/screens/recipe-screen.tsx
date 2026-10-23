import Ionicons from "@expo/vector-icons/Ionicons";
import { isNotFound, type Recipe, useRecipe } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import type { ReactNode } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Card, Divider } from "@/components/card";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { IconButton } from "@/components/icon-button";
import { Skeleton } from "@/components/skeleton";
import {
  cookShowing,
  difficultyLabel,
  equipmentLabel,
  minutesLabel,
  numbered,
  stepsLabel,
} from "@/lib/recipes";
import { parseId } from "@/lib/route-id";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The design's recipe header (screen 5): a square picture this big beside the title.
const ART = 112;
const ART_ICON = 40;
const META_ICON = 14;
const ROW_ICON = 18;
const STEP_NUMBER = 24;
const BAR_HEIGHT = 44;

/**
 * RecipeScreen is C2, screen 5 of the design: one recipe's time, difficulty and points preview,
 * what the player has and what they need, the steps, and I made it, which goes to plate it.
 */
export function RecipeScreen(props: { id: string | undefined }) {
  const recipeId = parseId(props.id);
  const recipe = useRecipe(recipeId);
  const router = useRouter();
  const insets = useSafeAreaInsets();

  // A recipe in hand stays on screen through a failed refresh. The api always sends one;
  // protobuf-es types every message field as optional, so an answer without one is not found.
  const found = recipe.data?.recipe;
  const state = cookShowing({
    id: recipeId,
    answer: recipe.data === undefined ? undefined : (found ?? null),
    notFound: isNotFound(recipe.error),
    failed: recipe.isError,
  });
  let body: ReactNode;
  if (state === "not-found") {
    body = (
      <EmptyState
        title="No recipe here"
        line="This recipe doesn't exist, or it isn't yours."
        action={{ label: "Scan your fridge", onPress: () => router.dismissTo("/") }}
      />
    );
  } else if (state === "error") {
    body = (
      <ErrorState
        title="Couldn't load the recipe"
        error={recipe.error}
        retrying={recipe.isFetching}
        onRetry={() => void recipe.refetch()}
      />
    );
  } else if (state === "loading" || found === undefined) {
    body = <Loading />;
  } else {
    return (
      <View style={styles.root}>
        <Stack.Screen options={{ headerShown: false }} />
        <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
          <View style={styles.bar}>
            <IconButton icon="arrow-back" label="Back" onPress={() => router.back()} />
          </View>
          <View style={styles.page}>
            <Top recipe={found} />
            <Foods
              title="You have"
              names={found.uses}
              count={`${found.uses.length} of ${found.uses.length + found.missing.length}`}
              icon="checkmark-circle"
              color={COLORS.pickle}
            />
            {found.missing.length === 0 ? null : (
              <Foods
                title="You need"
                names={found.missing}
                count={`${found.missing.length} more`}
                icon="cart-outline"
                color={COLORS.ketchup}
              />
            )}
            <Steps steps={found.steps} />
          </View>
        </ScrollView>
        <BottomBar>
          <Button
            label="Cook step by step"
            onPress={() =>
              router.push({
                pathname: "/recipe/[id]/steps",
                params: { id: String(found.recipeId) },
              })
            }
          />
          <Button
            variant="secondary"
            label="I made it"
            onPress={() =>
              router.push({
                pathname: "/plate/[recipeId]",
                params: { recipeId: String(found.recipeId) },
              })
            }
          />
        </BottomBar>
      </View>
    );
  }

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <View style={styles.bar}>
        <IconButton icon="arrow-back" label="Back" onPress={() => router.back()} />
      </View>
      {body}
    </View>
  );
}

/** Top is the recipe's card: its title, time, difficulty and equipment, and the points preview. */
function Top(props: { recipe: Recipe }) {
  const { recipe } = props;
  const difficulty = difficultyLabel(recipe.difficulty);
  const equipment = equipmentLabel(recipe.equipment);
  return (
    <Card>
      <View style={styles.top}>
        {/* No recipe photos exist, so the tile shows a plate (docs/design.md). */}
        <View style={styles.art}>
          <Ionicons name="restaurant-outline" size={ART_ICON} color={COLORS.muted} />
        </View>
        <View style={styles.topText}>
          <Text accessibilityRole="header" style={styles.title}>
            {recipe.title}
          </Text>
          <View style={styles.meta}>
            <View style={styles.time}>
              <Ionicons name="time-outline" size={META_ICON} color={COLORS.muted} />
              <Text style={styles.metaText}>{minutesLabel(recipe.minutes)}</Text>
            </View>
            {difficulty === undefined ? null : <Text style={styles.metaText}>{difficulty}</Text>}
            {equipment === undefined ? null : <Text style={styles.metaText}>{equipment}</Text>}
          </View>
          <Text style={styles.points}>{`+${recipe.pointsPreview} pts when you plate it`}</Text>
        </View>
      </View>
    </Card>
  );
}

/**
 * Foods is one side of the ingredients, what the player has or what they still need, each name
 * marked in that side's colour.
 */
function Foods(props: {
  title: string;
  names: string[];
  count: string;
  icon: "checkmark-circle" | "cart-outline";
  color: string;
}) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHead}>
        <Text accessibilityRole="header" style={styles.sectionTitle}>
          {props.title}
        </Text>
        <Text style={[styles.sectionCount, { color: props.color }]}>{props.count}</Text>
      </View>
      <Card>
        {numbered(props.names).map((food) => (
          <View key={food.n}>
            {food.n === 1 ? null : <Divider />}
            <View style={styles.food}>
              <Text style={styles.foodName}>{food.text}</Text>
              <Ionicons name={props.icon} size={ROW_ICON} color={props.color} />
            </View>
          </View>
        ))}
      </Card>
    </View>
  );
}

/** Steps are the recipe's instructions, numbered in the server's order. */
function Steps(props: { steps: string[] }) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHead}>
        <Text accessibilityRole="header" style={styles.sectionTitle}>
          Steps
        </Text>
        <Text style={styles.stepsCount}>{stepsLabel(props.steps.length)}</Text>
      </View>
      <Card>
        {numbered(props.steps).map((step) => (
          <View key={step.n}>
            {step.n === 1 ? null : <Divider />}
            <View style={styles.step}>
              <View style={styles.stepNumber}>
                <Text style={styles.stepNumberText}>{step.n}</Text>
              </View>
              <Text style={styles.stepText}>{step.text}</Text>
            </View>
          </View>
        ))}
      </Card>
    </View>
  );
}

/** Loading is the recipe's shape while it loads: its card, then two lists. */
function Loading() {
  return (
    <View accessible accessibilityLabel="Loading the recipe" style={styles.page}>
      <Skeleton height={ART + SPACE.xl} radius={RADII.xl} />
      <Skeleton height={24} width={120} />
      <Skeleton height={160} radius={RADII.xl} />
      <Skeleton height={24} width={120} />
      <Skeleton height={200} radius={RADII.xl} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { paddingBottom: SPACE.xl },
  bar: {
    height: BAR_HEIGHT,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: SPACE.sm,
  },
  page: { gap: SPACE.lg, paddingTop: SPACE.xs, paddingHorizontal: SPACE.lg },
  top: { flexDirection: "row", gap: SPACE.md + SPACE.xxs },
  art: {
    width: ART,
    height: ART,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.md,
    backgroundColor: COLORS.sunken,
  },
  topText: { flex: 1, justifyContent: "center", gap: SPACE.sm },
  title: { ...TYPE.heading, color: COLORS.ink },
  meta: { flexDirection: "row", flexWrap: "wrap", alignItems: "center", columnGap: SPACE.md },
  time: { flexDirection: "row", alignItems: "center", gap: SPACE.xs },
  metaText: { ...TYPE.caption, color: COLORS.muted },
  // Screen 5's pill: the preview before any plate, never a points line.
  points: {
    ...TYPE.caption,
    fontWeight: TYPE.bodyStrong.fontWeight,
    alignSelf: "flex-start",
    overflow: "hidden",
    paddingVertical: SPACE.xs,
    paddingHorizontal: SPACE.sm + SPACE.xxs,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.ketchupTint,
    color: COLORS.ketchupInk,
  },
  section: { gap: SPACE.sm },
  sectionHead: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: SPACE.xxs,
  },
  sectionTitle: { ...TYPE.heading, color: COLORS.ink },
  sectionCount: { ...TYPE.caption, fontWeight: TYPE.bodyStrong.fontWeight },
  stepsCount: { ...TYPE.caption, color: COLORS.muted },
  food: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.md,
    paddingVertical: SPACE.sm + SPACE.xxs,
  },
  foodName: { ...TYPE.body, flex: 1, color: COLORS.ink },
  step: { flexDirection: "row", gap: SPACE.md, paddingVertical: SPACE.sm + SPACE.xxs },
  stepNumber: {
    width: STEP_NUMBER,
    height: STEP_NUMBER,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: STEP_NUMBER / 2,
    backgroundColor: COLORS.ketchupTint,
  },
  stepNumberText: {
    ...TYPE.caption,
    fontWeight: TYPE.heading.fontWeight,
    color: COLORS.ketchupInk,
  },
  stepText: { ...TYPE.body, flex: 1, color: COLORS.ink },
});
