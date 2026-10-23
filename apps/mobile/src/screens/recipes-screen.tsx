import Ionicons from "@expo/vector-icons/Ionicons";
import {
  isNotFound,
  type Recipe,
  type RecipeFilters,
  useRecipes,
  useScan,
} from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { type ReactNode, useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Button } from "@/components/button";
import { Card } from "@/components/card";
import { FilterChip } from "@/components/chip";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { IconButton } from "@/components/icon-button";
import { IngredientSheet } from "@/components/ingredient-sheet";
import { Skeleton } from "@/components/skeleton";
import { EQUIPMENT } from "@/lib/kitchen";
import {
  cookShowing,
  DIET_CHIPS,
  difficultyLabel,
  emptyLine,
  equipmentLabel,
  haveLabel,
  minutesLabel,
  NO_FILTERS,
  pointsLabel,
  recipeLabel,
  TIME_LIMITS,
  withDiet,
  withEquipment,
  withTime,
} from "@/lib/recipes";
import { parseId } from "@/lib/route-id";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The design's recipe card (screen 3): this tall, with the picture this wide on its left.
const CARD_HEIGHT = 140;
const ART_WIDTH = 128;
const ART_ICON = 32;
const META_ICON = 16;
const CHEVRON = 20;
const BAR_HEIGHT = 44;

/**
 * RecipesScreen is C1, screen 3 of the design: the list the recipes come from with Edit, the
 * filter chips, and the Make it now and One item away shelves. Every chip goes to the server in
 * the request, and the shelves show what it sends back, in its order.
 */
export function RecipesScreen(props: { scanId: string | undefined }) {
  const scanId = parseId(props.scanId);
  const [filters, setFilters] = useState<RecipeFilters>(NO_FILTERS);
  const recipes = useRecipes(scanId, filters);
  const scan = useScan(scanId);
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const [fixing, setFixing] = useState(false);

  const back = () => router.back();
  // Shelves in hand stay on screen through a failed refresh.
  const state = cookShowing({
    id: scanId,
    answer: recipes.data,
    notFound: isNotFound(recipes.error) || isNotFound(scan.error),
    failed: recipes.isError,
  });
  if (state === "not-found" || scanId === undefined) {
    return (
      <Bare top={insets.top} onBack={back}>
        <EmptyState
          title="No scan here"
          line="This scan doesn't exist, or it isn't yours."
          action={{ label: "Scan your fridge", onPress: () => router.dismissTo("/") }}
        />
      </Bare>
    );
  }

  const verdict = scan.data?.verdict;
  const open = (recipe: Recipe) =>
    router.push({ pathname: "/recipe/[id]", params: { id: String(recipe.recipeId) } });

  let shelves: ReactNode;
  if (state === "error") {
    shelves = (
      <ErrorState
        title="Couldn't load your recipes"
        error={recipes.error}
        retrying={recipes.isFetching}
        onRetry={() => void recipes.refetch()}
      />
    );
  } else if (state === "loading" || recipes.data === undefined) {
    shelves = <Loading />;
  } else {
    const { makeItNow, oneItemAway } = recipes.data;
    // A changed chip shows the last shelves until its own land, and so does a refetch.
    const updating = recipes.isRefetching;
    shelves = (
      <View style={[styles.shelves, updating && styles.updating]}>
        {updating ? <Text style={styles.updatingText}>Updating…</Text> : null}
        {makeItNow.length === 0 && oneItemAway.length === 0 ? (
          <EmptyState
            title="Nothing you can make right now"
            line={emptyLine(filters)}
            action={{ label: "Put the fries in the bag", onPress: () => router.push("/fries") }}
          />
        ) : (
          <>
            <RecipeShelf title="Make it now" recipes={makeItNow} onOpen={open} />
            <RecipeShelf title="One item away" recipes={oneItemAway} onOpen={open} />
          </>
        )}
      </View>
    );
  }

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
        <Header
          title="Recipe ideas for you"
          subtitle="Here are some meals you can make with what you have."
          onBack={back}
        />
        {verdict === undefined ? (
          // The list is a courtesy above the shelves: while it loads it holds its place, and a
          // failed read leaves the shelves to say what went wrong.
          scan.isPending ? (
            <View style={styles.inset}>
              <Skeleton height={64} radius={RADII.xl} />
            </View>
          ) : null
        ) : (
          <Summary
            names={verdict.ingredients.map((item) => item.name)}
            onEdit={() => setFixing(true)}
          />
        )}
        <RecipeChips filters={filters} onChange={setFilters} />
        {shelves}
      </ScrollView>
      {/* Mounted only while open, so each opening starts from the verdict's own list. */}
      {fixing && verdict !== undefined ? (
        <IngredientSheet
          scanId={scanId}
          ingredients={verdict.ingredients}
          onClose={() => setFixing(false)}
        />
      ) : null}
    </View>
  );
}

/** Summary is the list the recipes come from, and Edit, which opens the sheet that fixes it. */
function Summary(props: { names: string[]; onEdit: () => void }) {
  return (
    <View style={styles.inset}>
      <Card>
        <View style={styles.summary}>
          <View style={styles.summaryText}>
            <Text style={styles.summaryLabel}>Cooking from</Text>
            <Text numberOfLines={2} style={styles.summaryNames}>
              {props.names.length === 0 ? "Nothing on the list yet" : props.names.join(", ")}
            </Text>
          </View>
          <Button
            label="Edit"
            variant="secondary"
            size="small"
            icon="create-outline"
            onPress={props.onEdit}
          />
        </View>
      </Card>
    </View>
  );
}

/**
 * RecipeChips are the filters: a time, the equipment the player has, and a diet. The phone only says
 * which are on; the server narrows the shelves.
 */
export function RecipeChips(props: {
  filters: RecipeFilters;
  onChange: (change: (filters: RecipeFilters) => RecipeFilters) => void;
}) {
  const { filters, onChange } = props;
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.chips}
    >
      {TIME_LIMITS.map((minutes) => (
        <FilterChip
          key={`time-${minutes}`}
          label={minutesLabel(minutes)}
          selected={filters.maxMinutes === minutes}
          onPress={() => onChange((f) => withTime(f, minutes))}
        />
      ))}
      {EQUIPMENT.map((choice) => (
        <FilterChip
          key={`equipment-${choice.value}`}
          label={choice.label}
          selected={filters.equipment.includes(choice.value)}
          onPress={() => onChange((f) => withEquipment(f, choice.value))}
        />
      ))}
      {DIET_CHIPS.map((choice) => (
        <FilterChip
          key={`diet-${choice.value}`}
          label={choice.label}
          selected={filters.diet === choice.value}
          onPress={() => onChange((f) => withDiet(f, choice.value))}
        />
      ))}
    </ScrollView>
  );
}

/** RecipeShelf is a heading over recipe cards. An empty shelf shows nothing. */
export function RecipeShelf(props: {
  title: string;
  recipes: Recipe[];
  onOpen: (recipe: Recipe) => void;
  owners?: ReadonlyMap<bigint, string>;
}) {
  if (props.recipes.length === 0) {
    return null;
  }
  return (
    <View style={styles.shelf}>
      <Text accessibilityRole="header" style={styles.shelfTitle}>
        {props.title}
      </Text>
      {props.recipes.map((recipe) => (
        <RecipeCard
          key={String(recipe.recipeId)}
          recipe={recipe}
          owners={props.owners}
          onPress={() => props.onOpen(recipe)}
        />
      ))}
    </View>
  );
}

/**
 * RecipeCard is one recipe on a shelf: its time, difficulty and equipment, how much of it the
 * player has, and the points plating it would earn. The whole card opens the recipe.
 */
function RecipeCard(props: {
  recipe: Recipe;
  onPress: () => void;
  owners?: ReadonlyMap<bigint, string>;
}) {
  const { recipe } = props;
  const difficulty = difficultyLabel(recipe.difficulty);
  const equipment = equipmentLabel(recipe.equipment);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={recipeLabel(recipe)}
      onPress={props.onPress}
      style={({ pressed }) => [styles.card, pressed && styles.cardPressed]}
    >
      {/* No recipe photos exist, so the tile shows a plate (docs/design.md). */}
      <View style={styles.art}>
        <Ionicons name="restaurant-outline" size={ART_ICON} color={COLORS.muted} />
      </View>
      <View style={styles.cardBody}>
        <Text numberOfLines={2} style={styles.cardTitle}>
          {recipe.title}
        </Text>
        <View style={styles.meta}>
          <Ionicons name="time-outline" size={META_ICON} color={COLORS.ink} />
          <Text style={styles.metaText}>{minutesLabel(recipe.minutes)}</Text>
          {difficulty === undefined ? null : <Text style={styles.metaMuted}>{difficulty}</Text>}
        </View>
        {equipment === undefined ? null : <Text style={styles.caption}>{equipment}</Text>}
        <Text style={styles.caption}>{haveLabel(recipe)}</Text>
        {recipe.onlyTogether ? <Text style={styles.points}>Only together</Text> : null}
        {props.owners === undefined
          ? null
          : recipe.ingredientOwners.map((item) => (
              <Text key={`${item.accountId}-${item.ingredient}`} style={styles.caption}>
                {item.ingredient}: @{props.owners?.get(item.accountId)}
              </Text>
            ))}
        <Text style={styles.points}>{pointsLabel(recipe.pointsPreview)}</Text>
      </View>
      <View style={styles.chevron}>
        <Ionicons name="chevron-forward" size={CHEVRON} color={COLORS.muted} />
      </View>
    </Pressable>
  );
}

/** Loading is the shelves' shape while the server finds recipes: a heading and two cards. */
function Loading() {
  return (
    <View accessible accessibilityLabel="Finding recipes" style={styles.shelves}>
      <Skeleton height={24} width={140} />
      <Skeleton height={CARD_HEIGHT} radius={RADII.xl} />
      <Skeleton height={CARD_HEIGHT} radius={RADII.xl} />
    </View>
  );
}

/** Bare frames a state with no shelves to show: the back arrow, then the state itself. */
function Bare(props: { top: number; onBack: () => void; children: ReactNode }) {
  return (
    <View style={[styles.root, { paddingTop: props.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <View style={styles.bar}>
        <IconButton icon="arrow-back" label="Back" onPress={props.onBack} />
      </View>
      {props.children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.lg, paddingBottom: SPACE.xl },
  bar: {
    height: BAR_HEIGHT,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: SPACE.sm,
  },
  inset: { paddingHorizontal: SPACE.lg },
  summary: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  summaryText: { flex: 1, gap: SPACE.xxs },
  summaryLabel: { ...TYPE.caption, color: COLORS.muted },
  summaryNames: { ...TYPE.body, color: COLORS.ink },
  chips: { gap: SPACE.sm, paddingHorizontal: SPACE.lg },
  shelves: { gap: SPACE.xl, paddingHorizontal: SPACE.lg },
  // The last shelves stay readable while new ones load, but plainly not the answer yet.
  updating: { opacity: 0.5 },
  updatingText: { ...TYPE.small, color: COLORS.muted, textAlign: "center" },
  shelf: { gap: SPACE.md },
  shelfTitle: { ...TYPE.heading, color: COLORS.ink },
  card: {
    flexDirection: "row",
    minHeight: CARD_HEIGHT,
    borderWidth: 1,
    borderColor: COLORS.line,
    borderRadius: RADII.xl,
    overflow: "hidden",
    backgroundColor: COLORS.surface,
  },
  cardPressed: { backgroundColor: COLORS.bg },
  art: {
    width: ART_WIDTH,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: COLORS.sunken,
  },
  cardBody: {
    flex: 1,
    justifyContent: "center",
    gap: SPACE.xs + SPACE.xxs,
    paddingVertical: SPACE.md,
    paddingLeft: SPACE.md + SPACE.xxs,
    paddingRight: SPACE.xs + SPACE.xxs,
  },
  cardTitle: { ...TYPE.bodyStrong, color: COLORS.ink },
  meta: { flexDirection: "row", alignItems: "center", gap: SPACE.xs },
  metaText: { ...TYPE.small, color: COLORS.ink },
  metaMuted: { ...TYPE.small, color: COLORS.muted, marginLeft: SPACE.sm },
  caption: { ...TYPE.caption, color: COLORS.muted },
  // Screen 5's points pill, smaller: the preview before any plate, never a points line.
  points: {
    ...TYPE.caption,
    fontWeight: TYPE.bodyStrong.fontWeight,
    alignSelf: "flex-start",
    overflow: "hidden",
    paddingVertical: SPACE.xxs,
    paddingHorizontal: SPACE.sm,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.ketchupTint,
    color: COLORS.ketchupInk,
  },
  chevron: { justifyContent: "center", paddingRight: SPACE.md },
});
