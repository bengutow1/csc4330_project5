import type { Tier } from "@fries-in-the-bag/api";
import type { ImageProps } from "expo-image";
import type { ReactNode } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Logo } from "@/components/logo";
import { PhotoCard } from "@/components/photo-card";
import { byline, CARD } from "@/lib/share";
import { tierLook } from "@/lib/tier";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// A story is seen from across a room, so the card's numbers run bigger than the screen's hero.
const BIG_NUMBER = 96;

/**
 * RoastCard is a verdict made for sharing (C6): the logo, the score, the tier and the roast, and
 * the player's handle, the one thing about them a card carries. Every number on it is the
 * server's.
 */
export function RoastCard(props: {
  score: number;
  tier: Tier;
  roast: string;
  handle: string | undefined;
}) {
  const look = tierLook(props.tier);
  return (
    <Frame handle={props.handle}>
      <View style={styles.middle}>
        <View style={styles.number}>
          <Text style={[styles.big, { color: COLORS.ink }]}>{props.score}</Text>
          <Text style={styles.caption}>fridge score</Text>
        </View>
        {/* Drawn still: the screen's pill lands with a thud, and a card has no time to land. */}
        {look ? (
          <View style={[styles.pill, { backgroundColor: look.fill }]}>
            <Text style={[styles.pillLabel, { color: look.ink }]}>{look.label}</Text>
          </View>
        ) : null}
        <Text style={styles.roast}>{`“${props.roast}”`}</Text>
      </View>
    </Frame>
  );
}

/**
 * PlateCard is a passed plate made for sharing (C6): the logo, the photo, the recipe, the points
 * it earned and the player's handle. The points are the server's total, never worked out here.
 */
export function PlateCard(props: {
  photo: ImageProps["source"];
  caption: string;
  recipe: string | undefined;
  total: number;
  handle: string | undefined;
}) {
  return (
    <Frame handle={props.handle}>
      <View style={styles.middle}>
        <PhotoCard source={props.photo} label="Your plate" caption={props.caption} />
        {props.recipe ? <Text style={styles.recipe}>{props.recipe}</Text> : null}
        <View style={styles.number}>
          <Text style={[styles.big, { color: COLORS.ketchup }]}>{`+${props.total}`}</Text>
          <Text style={styles.caption}>points earned</Text>
        </View>
      </View>
    </Frame>
  );
}

/** Frame is what every card shares: the story's size, the logo on top and the handle below. */
function Frame(props: { handle: string | undefined; children: ReactNode }) {
  const name = byline(props.handle);
  return (
    <View style={styles.card}>
      <Logo />
      {props.children}
      <Text style={styles.byline}>{name ?? " "}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    width: CARD.width,
    height: CARD.height,
    alignItems: "center",
    justifyContent: "space-between",
    paddingTop: SPACE.xxl + SPACE.lg,
    paddingBottom: SPACE.xxl,
    paddingHorizontal: SPACE.xl,
    backgroundColor: COLORS.bg,
  },
  middle: { alignItems: "center", gap: SPACE.lg },
  number: { alignItems: "center" },
  big: { ...TYPE.hero, fontSize: BIG_NUMBER, lineHeight: BIG_NUMBER * 1.1 },
  caption: { ...TYPE.body, color: COLORS.muted },
  pill: {
    paddingVertical: SPACE.sm,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderRadius: RADII.pill,
  },
  pillLabel: { ...TYPE.small, fontWeight: TYPE.bodyStrong.fontWeight },
  roast: { ...TYPE.heading, fontWeight: "700", color: COLORS.inkSoft, textAlign: "center" },
  recipe: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  byline: { ...TYPE.bodyStrong, color: COLORS.muted },
});
