import Ionicons from "@expo/vector-icons/Ionicons";
import { type Spot, SpotFilter, useLastScan, useSpots } from "@fries-in-the-bag/api";
import { FlashList } from "@shopify/flash-list";
import { Stack, useRouter } from "expo-router";
import { type ReactNode, useState } from "react";
import { Linking, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { Button } from "@/components/button";
import { FilterChip } from "@/components/chip";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/skeleton";
import {
  emptyReason,
  SPOT_CHIPS,
  STARTING_FILTERS,
  sendableFilters,
  toggleFilter,
} from "@/lib/spots";
import { SpotRow } from "@/screens/fries/spot-row";
import { SpotSheet } from "@/screens/fries/spot-sheet";
import { usePlayerPoint } from "@/screens/fries/use-player-point";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// A row's height while the list loads.
const ROW_SKELETON = 84;
// Four rows fill a phone's first screen of the list; each needs a key of its own.
const SKELETON_ROWS = ["first", "second", "third", "fourth"] as const;
const SEARCH_HEIGHT = 48;
const CHIP_HEIGHT = 40;
const NEAREST_ICON = 14;

/**
 * SpotsScreen is the list of spots (F2, screen 9): the server's order and filters, chips that ask
 * the server again, and the nearest fries one tap away. With location off it says why the point
 * helps and offers a search box instead. When nothing is open it says so, and offers the last
 * scan's recipes. A spot opens its sheet.
 */
export function SpotsScreen() {
  const place = usePlayerPoint();
  const [filters, setFilters] = useState<readonly SpotFilter[]>(STARTING_FILTERS);
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Spot>();
  const hasPoint = place.point !== undefined;
  // With a point the list is "near me"; the search box only shows with location off.
  const searching = hasPoint || query.trim() !== "";
  const spots = useSpots({
    at: place.point,
    query: hasPoint ? "" : query,
    filters: sendableFilters(filters, hasPoint),
  });
  // Asked for as the screen opens, so it is in hand if nothing turns out to be open.
  const lastScanId = useLastScan().data?.verdict?.scanId;
  const pages = spots.data?.pages ?? [];
  const list = pages.flatMap((page) => page.spots);
  const nearest = pages[0]?.nearestFries;

  let body: ReactNode = null;
  if (place.locating || (searching && spots.isPending)) {
    body = <SpotsSkeleton />;
  } else if (spots.isError && list.length === 0) {
    body = (
      <ErrorState
        title="Couldn't find food nearby"
        error={spots.error}
        retrying={spots.isFetching}
        onRetry={() => void spots.refetch()}
      />
    );
  } else if (searching && list.length === 0) {
    body =
      emptyReason(filters) === "closed" ? (
        <ClosedState lastScanId={lastScanId} />
      ) : (
        <EmptyState
          title="Nothing fits those chips."
          line="Turn a few off to see more spots."
          action={{ label: "Clear the chips", onPress: () => setFilters([]) }}
        />
      );
  }

  const header = (
    <View style={styles.header}>
      {place.off ? (
        <LocationOff
          canAsk={place.canAsk}
          onAsk={() => void place.ask()}
          query={query}
          onQuery={setQuery}
        />
      ) : null}
      {searching ? (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.chips}
        >
          {nearest ? <NearestChip onPress={() => setPicked(nearest)} /> : null}
          {SPOT_CHIPS.filter((chip) => hasPoint || chip.filter !== SpotFilter.WALKABLE).map(
            (chip) => (
              <FilterChip
                key={chip.label}
                label={chip.label}
                selected={filters.includes(chip.filter)}
                onPress={() => setFilters(toggleFilter(filters, chip.filter))}
              />
            ),
          )}
        </ScrollView>
      ) : null}
    </View>
  );

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerTitle: "Food nearby" }} />
      <FlashList
        data={searching ? list : []}
        keyExtractor={(spot) => spot.spotId}
        renderItem={({ item }) => (
          <SpotRow
            spot={item}
            nearest={item.spotId === nearest?.spotId}
            onPress={() => setPicked(item)}
          />
        )}
        ItemSeparatorComponent={RowGap}
        ListHeaderComponent={header}
        ListEmptyComponent={body}
        onEndReached={() => {
          if (spots.hasNextPage && !spots.isFetchingNextPage) {
            void spots.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.5}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={styles.content}
        // A new search keeps the last list up, faded, until its own answer lands.
        style={spots.isPlaceholderData ? styles.stale : undefined}
      />
      {/* Keyed by spot, so each sheet starts with its own Log fry run. */}
      {picked ? (
        <SpotSheet key={picked.spotId} spot={picked} onClose={() => setPicked(undefined)} />
      ) : null}
    </View>
  );
}

/**
 * LocationOff is what the list shows without a point: one line on why the point helps, the way to
 * turn it on, and a search box that looks by words instead.
 */
function LocationOff(props: {
  canAsk: boolean;
  onAsk: () => void;
  query: string;
  onQuery: (query: string) => void;
}) {
  return (
    <View style={styles.off}>
      <Text style={styles.offLine}>
        With your location, the closest food comes first. It's only used for this search, never
        stored.
      </Text>
      {props.canAsk ? (
        <Button label="Use my location" variant="secondary" size="small" onPress={props.onAsk} />
      ) : (
        <Button
          label="Open settings"
          variant="secondary"
          size="small"
          onPress={() => void Linking.openSettings()}
        />
      )}
      <TextInput
        accessibilityLabel="Search for food"
        autoCorrect={false}
        onChangeText={props.onQuery}
        placeholder="Search a place or a dish"
        placeholderTextColor={COLORS.muted}
        returnKeyType="search"
        style={styles.search}
        value={props.query}
      />
    </View>
  );
}

/**
 * ClosedState is the list when nothing near is open. With a last scan it links to that fridge's
 * recipes, for the least sad thing it can make; before a first scan it offers one instead.
 */
function ClosedState(props: { lastScanId: bigint | undefined }) {
  const router = useRouter();
  const { lastScanId } = props;
  if (lastScanId === undefined) {
    return (
      <EmptyState
        title="Even the fries are closed."
        line="Nothing near you is open right now. Your fridge might have one more meal in it."
        action={{ label: "Snap your fridge", onPress: () => router.navigate("/") }}
      />
    );
  }
  return (
    <EmptyState
      title="Even the fries are closed."
      line="Nothing near you is open right now. Your last fridge can still make something."
      action={{
        label: "Cook the least sad thing",
        onPress: () =>
          router.navigate({ pathname: "/cook/[scanId]", params: { scanId: String(lastScanId) } }),
      }}
    />
  );
}

/** NearestChip jumps straight to the nearest fries, the namesake joke, whatever chips are on. */
function NearestChip(props: { onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Nearest fries"
      onPress={props.onPress}
      style={({ pressed }) => [styles.nearestChip, pressed && styles.pressed]}
    >
      <Ionicons name="fast-food-outline" size={NEAREST_ICON} color={COLORS.mustardInk} />
      <Text style={styles.nearestChipText}>Nearest fries</Text>
    </Pressable>
  );
}

/** SpotsSkeleton stands in for the first rows while the list loads. */
function SpotsSkeleton() {
  return (
    <View style={styles.skeleton}>
      {SKELETON_ROWS.map((row) => (
        <Skeleton key={row} height={ROW_SKELETON} radius={RADII.xl} />
      ))}
    </View>
  );
}

function RowGap() {
  return <View style={styles.gap} />;
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xl },
  header: { gap: SPACE.md, paddingTop: SPACE.xs, paddingBottom: SPACE.md + SPACE.xxs },
  chips: { gap: SPACE.sm },
  off: { gap: SPACE.md, alignItems: "center", paddingTop: SPACE.sm },
  offLine: { ...TYPE.body, color: COLORS.inkSoft, textAlign: "center" },
  search: {
    ...TYPE.body,
    alignSelf: "stretch",
    height: SEARCH_HEIGHT,
    paddingHorizontal: SPACE.lg,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
    color: COLORS.ink,
  },
  nearestChip: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.xs + SPACE.xxs,
    height: CHIP_HEIGHT,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.mustardTint,
  },
  nearestChipText: {
    ...TYPE.small,
    fontWeight: TYPE.bodyStrong.fontWeight,
    color: COLORS.mustardInk,
  },
  pressed: { opacity: 0.7 },
  skeleton: { gap: SPACE.sm + SPACE.xxs },
  gap: { height: SPACE.sm + SPACE.xxs },
  stale: { opacity: 0.6 },
});
