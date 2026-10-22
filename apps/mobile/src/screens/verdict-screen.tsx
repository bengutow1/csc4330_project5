import Ionicons from "@expo/vector-icons/Ionicons";
import { isNotFound, useScan } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { type ReactNode, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { CountUp } from "@/components/count-up";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { IconButton } from "@/components/icon-button";
import { IngredientSheet } from "@/components/ingredient-sheet";
import { Skeleton } from "@/components/skeleton";
import { TierPill } from "@/components/tier-pill";
import { categoryLook, groupByCategory } from "@/lib/categories";
import type { Item } from "@/lib/ingredients";
import { buttonsFor, parseScanId, type Way } from "@/lib/verdict";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The design's ingredient tile (screen 2): a picture this tall, three to a row.
const TILE_ART = 84;
const TILE_WIDTH = "31%";
const BAR_HEIGHT = 44;

/**
 * VerdictScreen is a scan's verdict, screen 2 with the score, tier and roast on top
 * (docs/design.md): the score counting up, the tier's pill, the roast, and what the scan spotted
 * by category. The tier decides the two ways on, cook from this fridge or put the fries in the
 * bag, and which one is big. Fix the list opens the sheet that corrects the list, which changes
 * the recipes and never the score.
 */
export function VerdictScreen(props: { id: string | undefined }) {
  const scanId = parseScanId(props.id);
  const scan = useScan(scanId);
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const [fixing, setFixing] = useState(false);

  // Back to a fresh camera, wherever the verdict was opened from.
  const scanAgain = () => router.dismissTo("/");
  const go = (way: Way) =>
    way === "cook"
      ? router.push({ pathname: "/cook/[scanId]", params: { scanId: String(scanId) } })
      : router.push("/fries");

  if (scanId === undefined || isNotFound(scan.error)) {
    return (
      <Bare top={insets.top} onBack={() => router.back()}>
        <NotFound onScan={scanAgain} />
      </Bare>
    );
  }
  if (scan.isError) {
    return (
      <Bare top={insets.top} onBack={() => router.back()}>
        <ErrorState
          title="Couldn't load the verdict"
          error={scan.error}
          retrying={scan.isFetching}
          onRetry={() => void scan.refetch()}
        />
      </Bare>
    );
  }
  if (scan.isPending) {
    return (
      <Bare top={insets.top} onBack={() => router.back()}>
        <Loading />
      </Bare>
    );
  }
  // The api always sends one; protobuf-es types every message field as optional.
  const verdict = scan.data.verdict;
  if (verdict === undefined) {
    return (
      <Bare top={insets.top} onBack={() => router.back()}>
        <NotFound onScan={scanAgain} />
      </Bare>
    );
  }

  const nothing = verdict.ingredients.length === 0;
  const buttons = buttonsFor(verdict.tier);

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
        <Header
          title="Here's what we found"
          subtitle="We identified these ingredients in your fridge and pantry."
          onBack={() => router.back()}
        />
        <View style={styles.verdict}>
          <View style={styles.score}>
            <CountUp value={verdict.score} style={styles.scoreNumber} />
            <Text style={styles.caption}>fridge score</Text>
          </View>
          <TierPill tier={verdict.tier} />
          {/* With nothing spotted, the empty state below says it. */}
          {nothing ? null : <Text style={styles.roast}>{`“${verdict.roast}”`}</Text>}
        </View>
        {nothing ? (
          <EmptyState
            title="Nothing spotted"
            line="Either your fridge is empty or the photo is."
            action={{ label: "Scan again", onPress: scanAgain }}
          />
        ) : (
          <Sections list={verdict.ingredients} />
        )}
        <Button
          label="Fix the list"
          variant="secondary"
          size="small"
          icon="create-outline"
          onPress={() => setFixing(true)}
        />
      </ScrollView>
      <BottomBar>
        <View style={styles.ways}>
          <Button label={buttons.big.label} onPress={() => go(buttons.big.to)} />
          <Button
            label={buttons.small.label}
            variant="secondary"
            size="small"
            onPress={() => go(buttons.small.to)}
          />
        </View>
      </BottomBar>
      {/* Mounted only while open, so each opening starts from the verdict's own list. */}
      {fixing ? (
        <IngredientSheet
          scanId={scanId}
          ingredients={verdict.ingredients}
          onClose={() => setFixing(false)}
        />
      ) : null}
    </View>
  );
}

/** Sections lays the list out by category: a heading, then a tile for each ingredient. */
function Sections(props: { list: readonly Item[] }) {
  return (
    <View style={styles.sections}>
      {groupByCategory(props.list).map((group) => {
        const look = categoryLook(group.category);
        return (
          <View key={group.category} style={styles.section}>
            <View style={styles.sectionHead}>
              <Ionicons name={look.icon} size={18} color={COLORS.ink} />
              <Text accessibilityRole="header" style={styles.sectionTitle}>
                {look.label}
              </Text>
            </View>
            <View style={styles.tiles}>
              {group.items.map(({ item, index }) => (
                <View
                  accessible
                  accessibilityLabel={item.quantity ? `${item.name}, ${item.quantity}` : item.name}
                  key={`${index}-${item.name}`}
                  style={styles.tile}
                >
                  {/* No ingredient photos exist, so the tile shows its category (docs/design.md). */}
                  <View style={styles.art}>
                    <Ionicons name={look.icon} size={30} color={COLORS.muted} />
                  </View>
                  <Text numberOfLines={2} style={styles.name}>
                    {item.name}
                  </Text>
                  {item.quantity ? <Text style={styles.quantity}>{item.quantity}</Text> : null}
                </View>
              ))}
            </View>
          </View>
        );
      })}
    </View>
  );
}

/** NotFound answers a scan that isn't this player's, or never was. */
function NotFound(props: { onScan: () => void }) {
  return (
    <EmptyState
      title="No scan here"
      line="This scan doesn't exist, or it isn't yours."
      action={{ label: "Scan your fridge", onPress: props.onScan }}
    />
  );
}

/** Loading is the verdict's shape while it loads: the score, the pill, the roast, two rows. */
function Loading() {
  return (
    <View accessible accessibilityLabel="Loading the verdict" style={styles.loading}>
      <View style={styles.verdict}>
        <Skeleton height={56} width={96} />
        <Skeleton height={32} width={140} radius={RADII.pill} />
        <Skeleton height={16} width="80%" />
      </View>
      {[0, 1].map((row) => (
        <View key={row} style={styles.tiles}>
          <Skeleton height={TILE_ART} width={TILE_WIDTH} radius={RADII.sm} />
          <Skeleton height={TILE_ART} width={TILE_WIDTH} radius={RADII.sm} />
          <Skeleton height={TILE_ART} width={TILE_WIDTH} radius={RADII.sm} />
        </View>
      ))}
    </View>
  );
}

/** Bare frames a state with no verdict to show: the back arrow, then the state itself. */
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
  verdict: { alignItems: "center", gap: SPACE.md, paddingHorizontal: SPACE.xl },
  score: { alignItems: "center" },
  scoreNumber: { ...TYPE.hero, color: COLORS.ink },
  caption: { ...TYPE.caption, color: COLORS.muted },
  roast: { ...TYPE.body, color: COLORS.inkSoft, textAlign: "center" },
  sections: { gap: SPACE.sm + SPACE.xxs },
  // A band across the screen, as the design draws each category.
  section: {
    gap: SPACE.sm,
    paddingTop: SPACE.sm + SPACE.xxs,
    paddingBottom: SPACE.md + SPACE.xxs,
    paddingHorizontal: SPACE.lg,
    borderTopWidth: 1,
    borderBottomWidth: 1,
    borderColor: COLORS.line,
    backgroundColor: COLORS.surface,
  },
  sectionHead: { flexDirection: "row", alignItems: "center", gap: SPACE.sm, minHeight: 32 },
  sectionTitle: { ...TYPE.bodyStrong, color: COLORS.ink },
  tiles: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.sm + SPACE.xxs },
  tile: { width: TILE_WIDTH, alignItems: "center", gap: SPACE.xs + SPACE.xxs },
  art: {
    alignSelf: "stretch",
    height: TILE_ART,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.sm,
    backgroundColor: COLORS.sunken,
  },
  name: { ...TYPE.caption, color: COLORS.ink, textAlign: "center" },
  quantity: { ...TYPE.caption, color: COLORS.muted, textAlign: "center" },
  ways: { gap: SPACE.sm },
  loading: { gap: SPACE.xl, padding: SPACE.lg },
});
