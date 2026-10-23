import { usePlates, useProfile } from "@fries-in-the-bag/api";
import { FlashList } from "@shopify/flash-list";
import { useRouter } from "expo-router";
import { type ReactNode, useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { IconButton } from "@/components/icon-button";
import { Skeleton } from "@/components/skeleton";
import { BadgeShelf } from "@/screens/profile/badge-shelf";
import { PlateTile, PlayerTop, StatsRow } from "@/screens/profile/standing";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// Three plates to a row, and one row until See all.
const COLUMNS = 3;
const FIRST_ROW = 3;
// The skeleton: the top of the profile, then a row of stats.
const TOP_SKELETON = 210;
const STATS_SKELETON = 64;

/**
 * ProfileScreen is the Me tab (P1, screen 10): the player's avatar and handle, their rank and the
 * way to the next one, the stats row, their plates newest first and the badge shelf. The server
 * works out every number. The plates show one row until See all, then page in as the grid scrolls.
 */
export function ProfileScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const profile = useProfile();
  const plates = usePlates();
  const [seeAll, setSeeAll] = useState(false);
  const loaded = plates.data?.pages.flatMap((page) => page.plates) ?? [];
  const shown = seeAll ? loaded : loaded.slice(0, FIRST_ROW);
  const top = { paddingTop: insets.top };

  // The gear shows in every state, so Settings (Delete my data among it) stays in reach while the
  // profile loads or when it won't.
  const gear = (
    <View style={styles.bar}>
      <IconButton
        icon="settings-outline"
        label="Settings"
        onPress={() => router.navigate("/settings")}
      />
    </View>
  );

  const me = profile.data?.profile;
  if (profile.isPending) {
    return (
      <View style={[styles.root, styles.padded, top]}>
        {gear}
        <View style={styles.skeleton}>
          <Skeleton height={TOP_SKELETON} radius={RADII.xl} />
          <Skeleton height={STATS_SKELETON} radius={RADII.lg} />
        </View>
      </View>
    );
  }
  if (me === undefined) {
    return (
      <View style={[styles.root, styles.padded, top]}>
        {gear}
        <ErrorState
          title="Couldn't load your profile"
          error={profile.error}
          retrying={profile.isFetching}
          onRetry={() => void profile.refetch()}
        />
      </View>
    );
  }

  let empty: ReactNode = null;
  if (plates.isPending) {
    empty = <Skeleton height={STATS_SKELETON * 2} radius={RADII.lg} />;
  } else if (plates.isError) {
    empty = (
      <ErrorState
        title="Couldn't load your plates"
        error={plates.error}
        retrying={plates.isFetching}
        onRetry={() => void plates.refetch()}
      />
    );
  } else {
    empty = (
      <EmptyState
        title="No plates yet."
        line="Cook something from your fridge, plate it, and it lands here."
        action={{ label: "Snap your fridge", onPress: () => router.navigate("/") }}
      />
    );
  }

  const header = (
    <View>
      {gear}
      <PlayerTop player={me} />
      <StatsRow player={me} />
      <View style={styles.heading}>
        <Text accessibilityRole="header" style={styles.headingTitle}>
          My plates
        </Text>
        {!seeAll && (loaded.length > FIRST_ROW || plates.hasNextPage) ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="See all your plates"
            onPress={() => setSeeAll(true)}
            hitSlop={SPACE.sm}
          >
            <Text style={styles.link}>See all</Text>
          </Pressable>
        ) : null}
      </View>
    </View>
  );

  return (
    <View style={styles.root}>
      <FlashList
        data={shown}
        numColumns={COLUMNS}
        keyExtractor={(plate) => String(plate.plateId)}
        renderItem={({ item }) => <PlateTile plate={item} />}
        ListHeaderComponent={header}
        ListEmptyComponent={empty}
        ListFooterComponent={
          <BadgeShelf badges={me.badges} onSeeAll={() => router.navigate("/badges")} />
        }
        onEndReached={() => {
          if (seeAll && plates.hasNextPage && !plates.isFetchingNextPage) {
            void plates.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.5}
        contentContainerStyle={{ ...styles.padded, ...top, paddingBottom: SPACE.xl }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  padded: { paddingHorizontal: SPACE.lg },
  skeleton: { gap: SPACE.md },
  bar: { flexDirection: "row", justifyContent: "flex-end" },
  heading: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginTop: SPACE.lg + SPACE.xs,
    marginBottom: SPACE.sm + SPACE.xxs,
    marginHorizontal: SPACE.xxs,
  },
  headingTitle: { ...TYPE.heading, fontSize: TYPE.button.fontSize, color: COLORS.ink },
  link: { ...TYPE.small, color: COLORS.ketchup },
});
