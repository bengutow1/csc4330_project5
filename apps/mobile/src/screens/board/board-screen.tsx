import { type Row, useBoard } from "@fries-in-the-bag/api";
import { FlashList, type FlashListRef, type ViewToken } from "@shopify/flash-list";
import { useRouter } from "expo-router";
import { type ReactNode, useCallback, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Divider } from "@/components/card";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { SegmentedControl } from "@/components/segmented-control";
import { Skeleton } from "@/components/skeleton";
import { UnderlineTabs } from "@/components/underline-tabs";
import {
  BOARD_OPTIONS,
  type BoardItem,
  type BoardKey,
  boardItems,
  boardOf,
  emptyCopy,
  pinsMyRow,
  RANGE_OPTIONS,
  type RangeKey,
  rangeOf,
  subtitle,
} from "@/lib/board";
import { BoardRow } from "@/screens/board/board-row";
import { FridgeCard } from "@/screens/board/fridge-card";
import { HallOfFriesCard } from "@/screens/board/hall-of-fries-card";
import { Podium } from "@/screens/board/podium";
import { COLORS, RADII, SPACE } from "@/theme";

// A row counts as on screen once half of it shows. A module constant: the list cannot take a new
// one after it mounts.
const VIEWABILITY = { itemVisiblePercentThreshold: 50 } as const;
// The skeleton: a podium, then rows, each block with a key of its own.
const PODIUM_SKELETON = 220;
const ROW_SKELETON = 54;
const SKELETON_ROWS = ["first", "second", "third", "fourth"] as const;
// Room under the last row so the pinned row never covers it.
const PINNED_ROOM = 80;
// Screen 7's side margin.
const SIDE = SPACE.lg + SPACE.xs;

/**
 * BoardScreen is the Board tab (B1, screen 7): the Fridge of the Week, then a board the server
 * ranks and pages, switched between Chefs and the Hall of Fries and between this week and all
 * time. The first three stand on the podium, the rest list below, and the player's own row is
 * pinned to the bottom whenever it is off screen. Under Chefs, a card names this week's Hall of
 * Fries leader.
 */
export function BoardScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const list = useRef<FlashListRef<BoardItem<Row>>>(null);
  const [boardKey, setBoardKey] = useState<BoardKey>("chefs");
  const [rangeKey, setRangeKey] = useState<RangeKey>("this_week");
  const board = useBoard(boardOf(boardKey), rangeOf(rangeKey));
  const pages = board.data?.pages ?? [];
  const items = boardItems(pages.flatMap((page) => page.rows));
  const myRow = pages[0]?.myRow;
  // What the list last said is on screen; unknown until it first says.
  const [onScreen, setOnScreen] = useState<readonly BoardItem<Row>[]>();
  const pinned = pinsMyRow(myRow?.handle, onScreen) ? myRow : undefined;

  // Stable, because the list cannot take a new one after it mounts.
  const onViewableItemsChanged = useCallback(
    (info: { viewableItems: ViewToken<BoardItem<Row>>[] }) =>
      setOnScreen(info.viewableItems.map((token) => token.item)),
    [],
  );

  let body: ReactNode = null;
  if (board.isPending) {
    body = <BoardSkeleton />;
  } else if (board.isError && items.length === 0) {
    body = (
      <ErrorState
        title="Couldn't load the board"
        error={board.error}
        retrying={board.isFetching}
        onRetry={() => void board.refetch()}
      />
    );
  } else if (items.length === 0) {
    const copy = emptyCopy(boardKey, rangeKey);
    body = (
      <EmptyState
        title={copy.title}
        line={copy.line}
        action={{ label: "Snap your fridge", onPress: () => router.navigate("/") }}
      />
    );
  }

  const header = (
    <View style={styles.header}>
      <Header root title="Leaderboard" subtitle={subtitle(rangeKey)} />
      <FridgeCard />
      <View>
        <SegmentedControl options={BOARD_OPTIONS} value={boardKey} onChange={setBoardKey} />
        <UnderlineTabs options={RANGE_OPTIONS} value={rangeKey} onChange={setRangeKey} />
      </View>
    </View>
  );

  const last = items.length - 1;
  return (
    <View style={styles.root}>
      <FlashList
        ref={list}
        data={items}
        keyExtractor={(item) => (item.kind === "podium" ? "podium" : `row:${item.row.handle}`)}
        getItemType={(item) => item.kind}
        renderItem={({ item, index }) =>
          item.kind === "podium" ? (
            <View>
              <Podium board={boardKey} rows={item.rows} myHandle={myRow?.handle} />
              {/* Nobody below the podium: the board's card is only its floor. */}
              {index === last ? <View style={styles.floor} /> : null}
            </View>
          ) : (
            <View
              style={[
                styles.rowCard,
                index === 1 && styles.rowCardFirst,
                index === last && styles.rowCardLast,
              ]}
            >
              <BoardRow board={boardKey} row={item.row} mine={item.row.handle === myRow?.handle} />
              {index === last ? null : <Divider />}
            </View>
          )
        }
        // Which row is last, and whose, changes as pages land: every visible row draws again.
        extraData={`${boardKey}:${items.length}:${myRow?.handle ?? ""}`}
        ListHeaderComponent={header}
        ListEmptyComponent={body}
        ListFooterComponent={
          boardKey === "chefs" ? (
            <HallOfFriesCard
              onPress={() => {
                // The card names this week's leader, so the board it opens is this week's too,
                // shown from its podium rather than from where the Chefs list ended.
                setBoardKey("hall_of_fries");
                setRangeKey("this_week");
                list.current?.scrollToTop({ animated: true });
              }}
            />
          ) : null
        }
        onEndReached={() => {
          if (board.hasNextPage && !board.isFetchingNextPage) {
            void board.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.5}
        viewabilityConfig={VIEWABILITY}
        onViewableItemsChanged={onViewableItemsChanged}
        contentContainerStyle={{
          ...styles.content,
          paddingTop: insets.top + SPACE.xs,
          paddingBottom: pinned ? PINNED_ROOM : SPACE.xl,
        }}
        // A switch keeps the last board up, faded, until its own answer lands.
        style={board.isPlaceholderData ? styles.stale : undefined}
      />
      {pinned ? (
        <View testID="pinned-row" style={[styles.pinned, board.isPlaceholderData && styles.stale]}>
          <BoardRow board={boardKey} row={pinned} mine />
        </View>
      ) : null}
    </View>
  );
}

/** BoardSkeleton stands in for the podium and the first rows while the board loads. */
function BoardSkeleton() {
  return (
    <View style={styles.skeleton}>
      <Skeleton height={PODIUM_SKELETON} radius={RADII.xl} />
      {SKELETON_ROWS.map((row) => (
        <Skeleton key={row} height={ROW_SKELETON} radius={RADII.md} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { paddingHorizontal: SIDE },
  header: { gap: SPACE.md + SPACE.xxs, paddingBottom: SPACE.md + SPACE.xxs },
  rowCard: {
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderLeftWidth: 1,
    borderRightWidth: 1,
    borderColor: COLORS.line,
    backgroundColor: COLORS.surface,
  },
  rowCardFirst: { borderTopWidth: 1 },
  rowCardLast: {
    borderBottomWidth: 1,
    borderBottomLeftRadius: RADII.xl,
    borderBottomRightRadius: RADII.xl,
  },
  floor: {
    height: SPACE.md,
    borderWidth: 1,
    borderColor: COLORS.line,
    borderBottomLeftRadius: RADII.xl,
    borderBottomRightRadius: RADII.xl,
    backgroundColor: COLORS.surface,
  },
  pinned: {
    position: "absolute",
    left: SIDE,
    right: SIDE,
    bottom: SPACE.sm,
    paddingHorizontal: SPACE.md + SPACE.xxs,
    borderWidth: 1,
    borderColor: COLORS.line,
    borderRadius: RADII.xl,
    backgroundColor: COLORS.surface,
    shadowColor: COLORS.ink,
    shadowOpacity: 0.12,
    shadowRadius: 6,
    shadowOffset: { width: 0, height: 2 },
    elevation: 3,
  },
  skeleton: { gap: SPACE.sm + SPACE.xxs },
  stale: { opacity: 0.6 },
});
