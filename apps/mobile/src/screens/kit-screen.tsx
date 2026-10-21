import { Stack } from "expo-router";
import { type ReactNode, useEffect, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Card, Divider } from "@/components/card";
import { FilterChip, RemovableChip } from "@/components/chip";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { Logo } from "@/components/logo";
import { PillToggle } from "@/components/pill-toggle";
import { SegmentedControl } from "@/components/segmented-control";
import { Sheet } from "@/components/sheet";
import { Skeleton } from "@/components/skeleton";
import { UnderlineTabs } from "@/components/underline-tabs";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// Long enough to see a pending label before the pretend write lands.
const PRETEND_WRITE_MS = 1500;

const MODES = [
  { value: "photo", label: "Photo" },
  { value: "pantry", label: "Pantry" },
] as const;
const BOARDS = [
  { value: "chefs", label: "Chefs" },
  { value: "hall", label: "Hall of Fries" },
] as const;
const RANGES = [
  { value: "week", label: "This week" },
  { value: "all", label: "All time" },
] as const;
const FILTERS = ["Open now", "Under $10", "Fast food", "Walkable"];
const FOODS = ["Eggs", "Spinach", "Cheddar"];
// Not a ConnectError, so ErrorState shows the generic line, as for a server that never answered.
const PRETEND_ERROR = new Error("the kit's pretend failure");

/**
 * KitScreen shows every shared component in each of its states, for screenshots and kit.yaml.
 * Under each control a line says what it last did, so a flow can assert on visible text.
 */
export function KitScreen() {
  const insets = useSafeAreaInsets();
  const [lastPressed, setLastPressed] = useState("nothing");
  const [mode, setMode] = useState<(typeof MODES)[number]["value"]>("photo");
  const [filters, setFilters] = useState(["Open now"]);
  const [foods, setFoods] = useState(FOODS);
  const [board, setBoard] = useState<(typeof BOARDS)[number]["value"]>("chefs");
  const [range, setRange] = useState<(typeof RANGES)[number]["value"]>("week");
  const [headerTap, setHeaderTap] = useState("nothing");
  const [barTap, setBarTap] = useState("nothing");
  const [emptyTap, setEmptyTap] = useState("nothing");
  const [cardTaps, setCardTaps] = useState(0);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleted, setDeleted] = useState(false);
  const [retries, setRetries] = useState(0);
  const [retrying, setRetrying] = useState(false);

  // The pretend delete is in flight for a moment, then lands and closes the dialog.
  useEffect(() => {
    if (!deleting) {
      return;
    }
    const timer = setTimeout(() => {
      setDeleting(false);
      setDialogOpen(false);
      setDeleted(true);
    }, PRETEND_WRITE_MS);
    return () => clearTimeout(timer);
  }, [deleting]);

  useEffect(() => {
    if (!retrying) {
      return;
    }
    const timer = setTimeout(() => {
      setRetrying(false);
      setRetries((n) => n + 1);
    }, PRETEND_WRITE_MS);
    return () => clearTimeout(timer);
  }, [retrying]);

  const press = (label: string) => () => setLastPressed(label);
  const toggleFilter = (filter: string) =>
    setFilters((on) => (on.includes(filter) ? on.filter((f) => f !== filter) : [...on, filter]));
  const labelOf = (options: readonly { value: string; label: string }[], value: string) =>
    options.find((o) => o.value === value)?.label;

  return (
    <ScrollView
      style={styles.root}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + SPACE.lg }]}
    >
      {/* The kit titles itself with Header, so the only Back on screen is the demo's own. */}
      <Stack.Screen options={{ headerShown: false }} />
      <Header root title="UI kit" subtitle="Every shared component, in each state." />

      <Section title="Buttons">
        <Button label="Primary" onPress={press("Primary")} />
        <Button
          label="Secondary"
          variant="secondary"
          icon="share-outline"
          onPress={press("Secondary")}
        />
        <Button label="Destructive" variant="destructive" onPress={press("Destructive")} />
        <Button label="Judge my fridge" pending pendingLabel="Judging…" onPress={press("Judge")} />
        <Button label="Small primary" size="small" onPress={press("Small primary")} />
        <Button
          label="Small secondary"
          size="small"
          variant="secondary"
          onPress={press("Small secondary")}
        />
        <Line>{`Last pressed: ${lastPressed}`}</Line>
      </Section>

      <Section title="Pill toggle">
        <PillToggle options={MODES} value={mode} onChange={setMode} />
        <Line>{`Mode: ${labelOf(MODES, mode)}`}</Line>
      </Section>

      <Section title="Filter chips">
        <View style={styles.wrap}>
          {FILTERS.map((filter) => (
            <FilterChip
              key={filter}
              label={filter}
              selected={filters.includes(filter)}
              onPress={() => toggleFilter(filter)}
            />
          ))}
        </View>
        <Line>{`Filters: ${FILTERS.filter((f) => filters.includes(f)).join(", ") || "none"}`}</Line>
      </Section>

      <Section title="Ingredient chips">
        <View style={styles.wrap}>
          {foods.map((food) => (
            <RemovableChip
              key={food}
              label={food}
              onRemove={() => setFoods((list) => list.filter((f) => f !== food))}
            />
          ))}
        </View>
        <Line>{`Foods: ${foods.join(", ") || "none"}`}</Line>
        <Button
          label="Put them back"
          size="small"
          variant="secondary"
          onPress={() => setFoods(FOODS)}
        />
      </Section>

      <Section title="Segmented control">
        <SegmentedControl options={BOARDS} value={board} onChange={setBoard} />
        <Line>{`Board: ${labelOf(BOARDS, board)}`}</Line>
      </Section>

      <Section title="Underline tabs">
        <UnderlineTabs options={RANGES} value={range} onChange={setRange} />
        <Line>{`Range: ${labelOf(RANGES, range)}`}</Line>
      </Section>

      <Section title="Cards">
        <Card>
          <Text style={styles.strong}>A plain card</Text>
          <Text style={styles.muted}>Surface, a line border, radius 16.</Text>
        </Card>
        <Card featured>
          <Text style={styles.strong}>A featured card</Text>
          <Text style={styles.muted}>The nearest fries get a ketchup border.</Text>
        </Card>
        <Card accessibilityLabel="Open Taco Spot" onPress={() => setCardTaps((n) => n + 1)}>
          <Text style={styles.strong}>Taco Spot</Text>
          <Text style={styles.muted}>A card that is a button.</Text>
        </Card>
        <Card>
          <View style={styles.cardRow}>
            <Text style={styles.body}>Home-cooked meal</Text>
            <Text style={styles.strong}>+100</Text>
          </View>
          <Divider />
          <View style={styles.cardRow}>
            <Text style={styles.body}>4-day streak</Text>
            <Text style={styles.strong}>+40</Text>
          </View>
        </Card>
        <Line>{`Card taps: ${cardTaps}`}</Line>
      </Section>

      <Section title="Header">
        <View style={styles.frame}>
          <Header
            title="Here's what we found"
            subtitle="We identified these ingredients in your fridge and pantry."
            onBack={() => setHeaderTap("Back")}
          />
        </View>
        <View style={styles.frame}>
          <Header title="Plate it" onClose={() => setHeaderTap("Close")} />
        </View>
        <View style={styles.frame}>
          <Header root title="Leaderboard" subtitle="Campus · this week" />
        </View>
        <Line>{`Header: ${headerTap}`}</Line>
      </Section>

      <Section title="Bottom bar">
        <View style={styles.frame}>
          <BottomBar>
            <Button
              label="Show me what I can cook"
              onPress={() => setBarTap("Show me what I can cook")}
            />
          </BottomBar>
        </View>
        <View style={styles.frame}>
          <BottomBar>
            <Button
              label="Share"
              variant="secondary"
              icon="share-outline"
              onPress={() => setBarTap("Share")}
            />
            <Button label="Finish" onPress={() => setBarTap("Finish")} />
          </BottomBar>
        </View>
        <Line>{`Bottom bar: ${barTap}`}</Line>
      </Section>

      <Section title="Skeleton">
        {/* The screen around a skeleton says what is loading; here, the kit does. */}
        <View accessible accessibilityLabel="Loading skeleton" style={styles.skeletonRow}>
          <Skeleton height={44} width={44} radius={RADII.md} />
          <View style={styles.skeletonLines}>
            <Skeleton height={14} width="70%" />
            <Skeleton height={12} width="45%" />
          </View>
        </View>
      </Section>

      <Section title="Empty state">
        <View style={styles.frame}>
          <EmptyState
            title="No plates yet"
            line="Cook something and your plates land here."
            action={{ label: "Snap your fridge", onPress: () => setEmptyTap("Snap your fridge") }}
          />
        </View>
        <Line>{`Empty state: ${emptyTap}`}</Line>
      </Section>

      <Section title="Error state">
        <View style={styles.frame}>
          <ErrorState
            title="Couldn't load the board"
            error={PRETEND_ERROR}
            retrying={retrying}
            onRetry={() => setRetrying(true)}
          />
        </View>
        <Line>{`Retries: ${retries}`}</Line>
      </Section>

      <Section title="Sheet">
        <Button label="Open the sheet" variant="secondary" onPress={() => setSheetOpen(true)} />
        <Sheet visible={sheetOpen} title="Taco Spot" onClose={() => setSheetOpen(false)}>
          <Text style={styles.muted}>Mexican · $ · 2.9 km</Text>
          <Button label="Done" onPress={() => setSheetOpen(false)} />
        </Sheet>
      </Section>

      <Section title="Confirm dialog">
        <Button label="Delete my data" variant="destructive" onPress={() => setDialogOpen(true)} />
        <ConfirmDialog
          visible={dialogOpen}
          title="Delete your data?"
          message="Your scans, plates and points go for good. This can't be undone."
          confirmLabel="Delete"
          pendingLabel="Deleting…"
          destructive
          pending={deleting}
          onConfirm={() => setDeleting(true)}
          onCancel={() => setDialogOpen(false)}
        />
        <Line>{`Data: ${deleted ? "deleted" : "kept"}`}</Line>
      </Section>

      <Section title="Logo">
        <Logo />
        <Logo size="small" />
      </Section>
    </ScrollView>
  );
}

function Section(props: { title: string; children: ReactNode }) {
  return (
    <View style={styles.section}>
      <Text accessibilityRole="header" style={styles.heading}>
        {props.title}
      </Text>
      {props.children}
    </View>
  );
}

/** Line says what a control last did, for the kit flow to assert. */
function Line(props: { children: string }) {
  return <Text style={styles.line}>{props.children}</Text>;
}

const styles = StyleSheet.create({
  root: { backgroundColor: COLORS.bg },
  content: { gap: SPACE.xxl, padding: SPACE.lg, paddingBottom: SPACE.xxl * 2 },
  section: { gap: SPACE.md },
  heading: { ...TYPE.heading, color: COLORS.ink },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: SPACE.sm },
  line: { ...TYPE.caption, color: COLORS.muted },
  strong: { ...TYPE.bodyStrong, color: COLORS.ink },
  body: { ...TYPE.body, color: COLORS.ink },
  muted: { ...TYPE.caption, color: COLORS.muted },
  cardRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: SPACE.sm + SPACE.xxs,
  },
  // A dashed edge marks where a demo's own box ends, for the parts that fill their container.
  frame: {
    borderWidth: 1,
    borderStyle: "dashed",
    borderColor: COLORS.lineStrong,
    borderRadius: RADII.md,
    overflow: "hidden",
  },
  skeletonRow: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  skeletonLines: { flex: 1, gap: SPACE.sm },
});
