import { isNotFound, usePlayerCard } from "@fries-in-the-bag/api";
import type { ReactNode } from "react";
import { ScrollView, StyleSheet, Text, useWindowDimensions, View } from "react-native";
import { ErrorState } from "@/components/error-state";
import { Sheet } from "@/components/sheet";
import { Skeleton } from "@/components/skeleton";
import { BadgeShelf } from "@/screens/profile/badge-shelf";
import { PlateTile, PlayerTop, StatsRow } from "@/screens/profile/standing";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The card takes at most this share of the screen and scrolls inside it, so the board stays in
// sight above.
const MAX_SHARE = 0.75;
// Three plates to a row, as the Me tab sets them.
const PER_ROW = 3;
// The skeleton: the top of the card, then its stats row.
const TOP_SKELETON = 190;
const STATS_SKELETON = 64;

/**
 * PlayerCardSheet is B2: the card a board row opens, saying how that player got where they are.
 * Their rank and the way to the next title, their stats, their newest plates and every badge, all
 * from the server. handle is the row's player, or undefined while no card is open.
 */
export function PlayerCardSheet(props: { handle: string | undefined; onClose: () => void }) {
  const { height } = useWindowDimensions();
  const card = usePlayerCard(props.handle);

  let body: ReactNode;
  if (card.isPending) {
    body = (
      <View style={styles.skeleton}>
        <Skeleton height={TOP_SKELETON} radius={RADII.xl} />
        <Skeleton height={STATS_SKELETON} radius={RADII.lg} />
      </View>
    );
  } else if (isNotFound(card.error)) {
    // The handle is gone (its player deleted their data): nothing a retry could bring back.
    body = (
      <View style={styles.gone}>
        <Text style={styles.goneTitle}>That player isn't here anymore.</Text>
        <Text style={styles.none}>Their card left with their data.</Text>
      </View>
    );
  } else if (card.data?.card === undefined) {
    body = (
      <ErrorState
        title="Couldn't load this player"
        error={card.error}
        retrying={card.isFetching}
        onRetry={() => void card.refetch()}
      />
    );
  } else {
    const player = card.data.card;
    body = (
      <View>
        <PlayerTop player={player} />
        <StatsRow player={player} />
        <Text accessibilityRole="header" style={styles.heading}>
          Recent plates
        </Text>
        {player.recentPlates.length === 0 ? (
          <Text style={styles.none}>No plates yet.</Text>
        ) : (
          <View style={styles.grid}>
            {player.recentPlates.map((plate) => (
              <View key={String(plate.plateId)} style={styles.cell}>
                <PlateTile plate={plate} />
              </View>
            ))}
          </View>
        )}
        <BadgeShelf badges={player.badges} />
      </View>
    );
  }

  return (
    <Sheet visible={props.handle !== undefined} title="Player card" onClose={props.onClose}>
      <ScrollView style={{ maxHeight: height * MAX_SHARE }} contentContainerStyle={styles.content}>
        {body}
      </ScrollView>
    </Sheet>
  );
}

const styles = StyleSheet.create({
  content: { paddingBottom: SPACE.md },
  skeleton: { gap: SPACE.md },
  heading: {
    ...TYPE.heading,
    fontSize: TYPE.button.fontSize,
    color: COLORS.ink,
    marginTop: SPACE.lg + SPACE.xs,
    marginBottom: SPACE.sm,
    marginHorizontal: SPACE.xxs,
  },
  none: { ...TYPE.caption, color: COLORS.muted, marginHorizontal: SPACE.xxs },
  gone: { alignItems: "center", gap: SPACE.xs, paddingVertical: SPACE.xl },
  goneTitle: { ...TYPE.bodyStrong, color: COLORS.ink },
  grid: { flexDirection: "row", flexWrap: "wrap" },
  cell: { width: `${100 / PER_ROW}%` },
});
