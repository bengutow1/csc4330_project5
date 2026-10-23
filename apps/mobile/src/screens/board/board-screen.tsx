import { type Row, useBoard, useMyCrews } from "@fries-in-the-bag/api";
import { FlashList, type FlashListRef, type ViewToken } from "@shopify/flash-list";
import { useRouter } from "expo-router";
import { type ReactNode, useCallback, useRef, useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Divider } from "@/components/card";
import { FilterChip } from "@/components/chip";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { IconButton } from "@/components/icon-button";
import { SegmentedControl } from "@/components/segmented-control";
import { Skeleton } from "@/components/skeleton";
import { UnderlineTabs } from "@/components/underline-tabs";
import {
  BOARD_OPTIONS,
  type BoardItem,
  type BoardKey,
  boardItems,
  boardOf,
  distinctRows,
  emptyCopy,
  itemsInView,
  pinsMyRow,
  RANGE_OPTIONS,
  type RangeKey,
  rangeOf,
  subtitle,
} from "@/lib/board";
import { CAMPUS, scopeOf } from "@/lib/crews";
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
 * Fries leader. Chips switch the board between the campus and each of the player's crews (B3),
 * whose members the server ranks among themselves; the crew icon opens their crews.
 */
export function BoardScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const list = useRef<FlashListRef<BoardItem<Row>>>(null);
  const [boardKey, setBoardKey] = useState<BoardKey>("chefs");
  const [rangeKey, setRangeKey] = useState<RangeKey>("this_week");
  const crews = useMyCrews();
  const [picked, setPicked] = useState(CAMPUS);
  // A crew the player has since left falls back to the campus, during render.
  const scope = scopeOf(crews.data?.crews, picked);
  const campus = scope === CAMPUS;
  const board = useBoard(boardOf(boardKey), rangeOf(rangeKey), scope);
  const pages = board.data?.pages ?? [];
  const items = boardItems(distinctRows(pages.flatMap((page) => page.rows)));
  const myRow = pages[0]?.myRow;
  // Where the list last said it is looking, as positions read against the board as it is now: a
  // switch or a refetch can change the rows there without the list saying so again.
  const [inView, setInView] = useState<readonly number[]>();
  const pinned = pinsMyRow(myRow?.handle, itemsInView(items, inView)) ? myRow : undefined;

  // Stable, because the list cannot take a new one after it mounts.
  const onViewableItemsChanged = useCallback(
    (info: { viewableItems: ViewToken<BoardItem<Row>>[] }) =>
      setInView(info.viewableItems.flatMap((token) => (token.index === null ? [] : [token.index]))),
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

  const myCrews = crews.data?.crews ?? [];
  const scopeName = myCrews.find((crew) => crew.crewId === scope)?.name;
  const header = (
    <View style={styles.header}>
      <View style={styles.bar}>
        <IconButton icon="people-outline" label="Crews" onPress={() => router.navigate("/crews")} />
      </View>
      <Header root title="Leaderboard" subtitle={subtitle(rangeKey, scopeName)} />
      {/* The campus and then each crew. With no crews, the campus is the only board. */}
      {myCrews.length > 0 ? (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.scopes}
        >
          <FilterChip label="Campus" selected={campus} onPress={() => setPicked(CAMPUS)} />
          {myCrews.map((crew) => (
            <FilterChip
              key={String(crew.crewId)}
              label={crew.name}
              selected={scope === crew.crewId}
              onPress={() => setPicked(crew.crewId)}
            />
          ))}
        </ScrollView>
      ) : null}
      {/* The Fridge of the Week is the whole campus's, so a crew's board leaves it out. */}
      {campus ? <FridgeCard /> : null}
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
        extraData={`${boardKey}:${scope}:${items.length}:${myRow?.handle ?? ""}`}
        ListHeaderComponent={header}
        ListEmptyComponent={body}
        ListFooterComponent={
          // The card names the campus's leader, so a crew's board leaves it out.
          boardKey === "chefs" && campus ? (
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
  bar: { flexDirection: "row", justifyContent: "flex-end" },
  scopes: { gap: SPACE.sm },
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
