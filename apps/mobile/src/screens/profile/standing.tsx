import Ionicons from "@expo/vector-icons/Ionicons";
import type { Plate, Profile } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { type ReactNode, useMemo } from "react";
import { StyleSheet, Text, View } from "react-native";
import { groupDigits } from "@/lib/format";
import { percent, photoUri, platedOn, rankLine, toDate } from "@/lib/profile";
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

/**
 * Standing is where a player stands, as the Me tab and a player card both show it. A Profile and
 * a PlayerCard each carry these fields, so either draws here.
 */
export type Standing = Pick<Profile, "handle" | "avatar" | "points" | "rank" | "stats">;

/** PlayerTop is the avatar, the handle, the rank and how far to the next one (screen 10). */
export function PlayerTop(props: { player: Standing }) {
  const { player } = props;
  const progress = player.rank?.progress ?? 0;
  return (
    <View style={styles.top}>
      <View style={styles.avatar}>
        <Text style={styles.avatarEmoji}>{player.avatar}</Text>
      </View>
      <Text accessibilityRole="header" style={styles.handle}>
        @{player.handle}
      </Text>
      <View style={styles.rankPill}>
        <Ionicons name="medal-outline" size={RANK_ICON} color={COLORS.ketchupInk} />
        <Text style={styles.rankName}>{player.rank?.name}</Text>
      </View>
      <View
        accessibilityRole="progressbar"
        accessibilityLabel={`Progress to ${player.rank?.next || "the top"}`}
        accessibilityValue={{ min: 0, max: 100, now: Math.round(progress * 100) }}
        style={styles.track}
      >
        <View style={[styles.fill, { width: `${progress * 100}%` }]} />
      </View>
      <Text style={styles.rankLine}>
        {rankLine(player.points, {
          next: player.rank?.next ?? "",
          toNext: player.rank?.toNext ?? 0,
        })}
      </Text>
    </View>
  );
}

/** StatsRow is the four counts under the rank, as screen 10 sets them. */
export function StatsRow(props: { player: Standing }) {
  const stats = props.player.stats;
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

/** PlateTile is one plate's photo in a grid, drawn from the bytes the server sent. */
export function PlateTile(props: { plate: Plate }) {
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
  tileCell: { padding: SPACE.xs },
  tile: { aspectRatio: 1, width: "100%", borderRadius: RADII.lg, backgroundColor: COLORS.sunken },
});
