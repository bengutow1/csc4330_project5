import Ionicons from "@expo/vector-icons/Ionicons";
import { type Plate, type Profile, usePlates, useProfile } from "@fries-in-the-bag/api";
import { FlashList } from "@shopify/flash-list";
import { Image } from "expo-image";
import { useRouter } from "expo-router";
import { type ReactNode, useMemo, useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { IconButton } from "@/components/icon-button";
import { Skeleton } from "@/components/skeleton";
import { groupDigits } from "@/lib/format";
import { percent, photoUri, platedOn, rankLine, toDate } from "@/lib/profile";
import { BadgeShelf } from "@/screens/profile/badge-shelf";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// Screen 10's sizes.
const AVATAR = 76;
const AVATAR_EMOJI = 36;
const BAR_WIDTH = 240;
const BAR_HEIGHT = 8;
const STAT_VALUE = 20;
const STREAK_ICON = 18;
const RANK_ICON = 14;
// The rank pill's height above and below its words, and the gap between the streak's flame and
// its number.
const PILL_PAD = 5;
const FLAME_GAP = 3;
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
      <ProfileTop me={me} />
      <StatsRow me={me} />
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
        ListFooterComponent={<BadgeShelf badges={me.badges} />}
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

/** ProfileTop is the avatar, the handle, the rank and how far to the next one. */
function ProfileTop(props: { me: Profile }) {
  const { me } = props;
  const progress = me.rank?.progress ?? 0;
  return (
    <View style={styles.top}>
      <View style={styles.avatar}>
        <Text style={styles.avatarEmoji}>{me.avatar}</Text>
      </View>
      <Text accessibilityRole="header" style={styles.handle}>
        @{me.handle}
      </Text>
      <View style={styles.rankPill}>
        <Ionicons name="medal-outline" size={RANK_ICON} color={COLORS.ketchupInk} />
        <Text style={styles.rankName}>{me.rank?.name}</Text>
      </View>
      <View
        accessibilityRole="progressbar"
        accessibilityLabel={`Progress to ${me.rank?.next || "the top"}`}
        accessibilityValue={{ min: 0, max: 100, now: Math.round(progress * 100) }}
        style={styles.track}
      >
        <View style={[styles.fill, { width: `${progress * 100}%` }]} />
      </View>
      <Text style={styles.rankLine}>
        {rankLine(me.points, { next: me.rank?.next ?? "", toNext: me.rank?.toNext ?? 0 })}
      </Text>
    </View>
  );
}

/** StatsRow is the four counts under the rank, as screen 10 sets them. */
function StatsRow(props: { me: Profile }) {
  const stats = props.me.stats;
  const cooks = stats?.cooks ?? 0;
  const streak = stats?.streakDays ?? 0;
  const fryRuns = stats?.fryRuns ?? 0;
  const ratio = percent(stats?.fryRatio ?? 0);
  return (
    <View style={styles.stats}>
      <Stat value={groupDigits(BigInt(cooks))} label="Meals cooked" />
      <Stat
        value={String(streak)}
        label="Day streak"
        icon={<Ionicons name="flame" size={STREAK_ICON} color={COLORS.ketchup} />}
      />
      <Stat value={groupDigits(BigInt(fryRuns))} label="Fry runs" />
      <Stat value={ratio} label="Fry ratio" />
    </View>
  );
}

function Stat(props: { value: string; label: string; icon?: ReactNode }) {
  return (
    <View accessible accessibilityLabel={`${props.label}: ${props.value}`} style={styles.stat}>
      <View style={styles.statValueRow}>
        {props.icon}
        <Text style={styles.statValue}>{props.value}</Text>
      </View>
      <Text style={styles.statLabel}>{props.label}</Text>
    </View>
  );
}

/** PlateTile is one plate's photo in the grid, drawn from the bytes the server sent. */
function PlateTile(props: { plate: Plate }) {
  const { plate } = props;
  // Encoding a photo is real work, so each tile does it once per photo.
  const uri = useMemo(() => photoUri(plate.photo), [plate.photo]);
  const day = plate.platedAt ? `, plated ${platedOn(toDate(plate.platedAt))}` : "";
  return (
    <View accessible accessibilityLabel={`${plate.recipeTitle}${day}`} style={styles.tileCell}>
      <Image source={{ uri }} style={styles.tile} contentFit="cover" />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  padded: { paddingHorizontal: SPACE.lg },
  skeleton: { gap: SPACE.md },
  bar: { flexDirection: "row", justifyContent: "flex-end" },
  top: { alignItems: "center", gap: SPACE.xs + SPACE.xxs },
  avatar: {
    width: AVATAR,
    height: AVATAR,
    borderRadius: AVATAR / 2,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: COLORS.ketchupTint,
  },
  avatarEmoji: { fontSize: AVATAR_EMOJI },
  handle: { ...TYPE.heading, fontSize: TYPE.title.fontSize - SPACE.xxs, color: COLORS.ink },
  rankPill: {
    flexDirection: "row",
    alignItems: "center",
    gap: SPACE.xs + SPACE.xxs,
    paddingVertical: PILL_PAD,
    paddingHorizontal: SPACE.sm + SPACE.xxs,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.ketchupTint,
  },
  rankName: { ...TYPE.caption, fontWeight: TYPE.bodyStrong.fontWeight, color: COLORS.ketchupInk },
  track: {
    width: BAR_WIDTH,
    height: BAR_HEIGHT,
    marginTop: SPACE.sm,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.sunken,
    overflow: "hidden",
  },
  fill: { height: "100%", borderRadius: RADII.pill, backgroundColor: COLORS.ketchup },
  rankLine: { ...TYPE.caption, color: COLORS.muted },
  stats: { flexDirection: "row", gap: SPACE.sm, marginTop: SPACE.lg + SPACE.xxs },
  stat: {
    flex: 1,
    alignItems: "center",
    gap: SPACE.xxs,
    paddingVertical: SPACE.sm + SPACE.xxs,
    paddingHorizontal: SPACE.xs,
    borderWidth: 1,
    borderColor: COLORS.line,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
  },
  statValueRow: { flexDirection: "row", alignItems: "center", gap: FLAME_GAP },
  statValue: { ...TYPE.heading, fontSize: STAT_VALUE, color: COLORS.ink },
  statLabel: {
    ...TYPE.tab,
    fontWeight: TYPE.small.fontWeight,
    color: COLORS.muted,
    textAlign: "center",
  },
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
  tileCell: { padding: SPACE.xs },
  tile: { aspectRatio: 1, width: "100%", borderRadius: RADII.lg, backgroundColor: COLORS.sunken },
});
