import { ComboState, isNotFound, type Player, useCombo, useWhoAmI } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { type ReactNode, useEffect, useState } from "react";
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from "react-native";
import Animated, { FadeIn, useReducedMotion } from "react-native-reanimated";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Card } from "@/components/card";
import { CountUp } from "@/components/count-up";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { Skeleton } from "@/components/skeleton";
import { TierPill } from "@/components/tier-pill";
import {
  chips,
  countLine,
  fridgeLabel,
  mergedLine,
  openingStep,
  STEP_MS,
  STEPS,
  sides,
  waitingLine,
} from "@/lib/combo";
import { parseId } from "@/lib/route-id";
import { buttonsFor, type Way } from "@/lib/verdict";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// Each fridge's dot and chips, as screen 8 colours the two players.
const MINE = { dot: COLORS.ketchup, fill: COLORS.ketchupTint, ink: COLORS.ketchupInk };
const THEIRS = { dot: COLORS.blueberry, fill: COLORS.blueberryTint, ink: COLORS.blueberry };
const DOT = 10;
const FRIDGE_SKELETON = 120;

/**
 * ComboRevealScreen is X3: each fridge's score in turn, then the combined score counting up as
 * its tier lands, and the merged foods coloured by whose fridge they came from. Until the second
 * scan lands it waits, asking for the combo every 2 s (useCombo), so both phones reveal within a
 * poll of each other; a phone that opens it afterwards shows the final state at once. The
 * combined tier picks the two ways on, as a single verdict's does.
 */
export function ComboRevealScreen(props: { id: string | undefined }) {
  const comboId = parseId(props.id);
  const query = useCombo(comboId);
  const myHandle = useWhoAmI().data?.account?.handle;
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const reduceMotion = useReducedMotion();

  const combo = query.data?.combo;
  const verdict = combo?.verdict;
  const revealed = verdict !== undefined;

  // Whether this phone saw the combo still waiting: only then does the reveal play out.
  const [watched, setWatched] = useState(false);
  useEffect(() => {
    if (combo !== undefined && !revealed && combo.state === ComboState.OPEN) {
      setWatched(true);
    }
  }, [combo, revealed]);

  // The steps play once, when the verdict first lands, and never again on a later refetch.
  const [step, setStep] = useState(0);
  const playing = revealed && openingStep(watched, reduceMotion) < STEPS.combined;
  useEffect(() => {
    if (!playing) {
      return;
    }
    const order = [STEPS.first, STEPS.second, STEPS.combined];
    const timers = order.map((s, i) => setTimeout(() => setStep(s), STEP_MS * i));
    return () => timers.forEach(clearTimeout);
  }, [playing]);
  const shown = playing ? step : STEPS.combined;

  if (comboId === undefined || isNotFound(query.error)) {
    return (
      <Frame top={insets.top} onBack={() => router.back()}>
        <EmptyState
          title="No combo here"
          line="This combo doesn't exist, or you're not in it."
          action={{ label: "Back to Snap", onPress: () => router.dismissTo("/") }}
        />
      </Frame>
    );
  }
  if (combo === undefined && query.isError) {
    return (
      <Frame top={insets.top} onBack={() => router.back()}>
        <ErrorState
          title="Couldn't load the combo"
          error={query.error}
          retrying={query.isFetching}
          onRetry={() => void query.refetch()}
        />
      </Frame>
    );
  }
  if (combo === undefined) {
    return (
      <Frame top={insets.top} onBack={() => router.back()}>
        <View accessible accessibilityLabel="Loading the combo" style={styles.skeleton}>
          <Skeleton height={FRIDGE_SKELETON} radius={RADII.xl} />
          <Skeleton height={FRIDGE_SKELETON} radius={RADII.xl} />
        </View>
      </Frame>
    );
  }

  const { me, them } = sides(combo.players, myHandle);
  if (verdict === undefined) {
    if (combo.state === ComboState.ENDED) {
      return (
        <Frame top={insets.top} onBack={() => router.back()}>
          <EmptyState
            title="This combo ended"
            line="It ended before both fridges were in."
            action={{ label: "Back to Snap", onPress: () => router.dismissTo("/") }}
          />
        </Frame>
      );
    }
    return (
      <Frame top={insets.top} onBack={() => router.back()}>
        <View style={styles.waiting}>
          <View style={styles.players}>
            {[me, them].map((p, i) =>
              p ? <PlayerStatus key={String(p.accountId)} player={p} mine={i === 0} /> : null,
            )}
          </View>
          <ActivityIndicator color={COLORS.ketchup} />
          <Text accessibilityLiveRegion="polite" style={styles.waitingLine}>
            {waitingLine(me, them)}
          </Text>
        </View>
      </Frame>
    );
  }

  const ways = buttonsFor(verdict.tier);
  const go = (way: Way) => {
    if (way === "cook") {
      router.push({ pathname: "/combo/[id]/recipes", params: { id: String(comboId) } });
      return;
    }
    router.push("/fries");
  };
  const merged = chips(verdict.ingredients, me?.accountId);
  const fridges = [me, them].filter((p): p is Player => p !== undefined);

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
        <Header title="Combo" onBack={() => router.back()} />
        <View style={styles.titles}>
          <Text accessibilityRole="header" style={styles.together}>
            {`You + ${them?.handle ?? "a friend"}`}
          </Text>
          {shown >= STEPS.combined ? (
            <Text style={styles.line}>{mergedLine(merged.length)}</Text>
          ) : null}
        </View>

        <View style={styles.fridges}>
          {fridges.map((p, i) =>
            shown >= (i === 0 ? STEPS.first : STEPS.second) ? (
              <Animated.View key={String(p.accountId)} entering={FadeIn} style={styles.fridge}>
                <Card>
                  <View style={styles.fridgeBody}>
                    <View style={styles.owner}>
                      <View
                        style={[styles.dot, { backgroundColor: (i === 0 ? MINE : THEIRS).dot }]}
                      />
                      <Text numberOfLines={1} style={styles.ownerName}>
                        {fridgeLabel(p, i === 0)}
                      </Text>
                    </View>
                    <Text style={styles.fridgeScore}>{p.scan?.score ?? 0}</Text>
                    <Text style={styles.line}>{countLine(p.scan?.ingredients.length ?? 0)}</Text>
                  </View>
                </Card>
              </Animated.View>
            ) : (
              <View key={String(p.accountId)} style={styles.fridge} />
            ),
          )}
        </View>

        {shown >= STEPS.combined ? (
          <Animated.View entering={FadeIn} style={styles.combined}>
            <View style={styles.score}>
              <CountUp value={verdict.score} style={styles.scoreNumber} />
              <Text style={styles.line}>combined score</Text>
            </View>
            <TierPill tier={verdict.tier} />
            <View style={styles.chips}>
              {merged.map((c) => {
                const look = c.mine ? MINE : THEIRS;
                return (
                  <View
                    key={c.key}
                    accessible
                    accessibilityLabel={`${c.name}, from ${c.mine ? "your fridge" : `${them?.handle ?? "your friend"}'s fridge`}`}
                    style={[styles.chip, { backgroundColor: look.fill }]}
                  >
                    <Text style={[styles.chipText, { color: look.ink }]}>{c.name}</Text>
                  </View>
                );
              })}
            </View>
            {/* The key: which colour is whose. */}
            <View style={styles.key}>
              <KeyEntry color={MINE.dot} label="You" />
              <KeyEntry color={THEIRS.dot} label={them?.handle ?? "Your friend"} />
            </View>
          </Animated.View>
        ) : null}
      </ScrollView>
      {shown >= STEPS.combined ? (
        <BottomBar>
          <View style={styles.ways}>
            <Button label={ways.big.label} onPress={() => go(ways.big.to)} />
            <Button
              label={ways.small.label}
              variant="secondary"
              size="small"
              onPress={() => go(ways.small.to)}
            />
          </View>
        </BottomBar>
      ) : null}
    </View>
  );
}

/** PlayerStatus is one seat while the combo waits: whose fridge, and whether it is in yet. */
function PlayerStatus(props: { player: Player; mine: boolean }) {
  const scanned = props.player.scan !== undefined;
  const look = props.mine ? MINE : THEIRS;
  return (
    <View style={styles.status}>
      <View style={styles.owner}>
        <View style={[styles.dot, { backgroundColor: look.dot }]} />
        <Text style={styles.ownerName}>{fridgeLabel(props.player, props.mine)}</Text>
      </View>
      <Text style={styles.line}>{scanned ? "Scanned" : "Not scanned yet"}</Text>
    </View>
  );
}

/** KeyEntry is one colour in the chips' key. */
function KeyEntry(props: { color: string; label: string }) {
  return (
    <View style={styles.owner}>
      <View style={[styles.dot, { backgroundColor: props.color }]} />
      <Text style={styles.line}>{props.label}</Text>
    </View>
  );
}

/** Frame is a state with no reveal to show yet: the header, then the state itself. */
function Frame(props: { top: number; onBack: () => void; children: ReactNode }) {
  return (
    <View style={[styles.root, { paddingTop: props.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <Header title="Combo" onBack={props.onBack} />
      {props.children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.lg, paddingBottom: SPACE.xl },
  titles: { alignItems: "center", gap: SPACE.xs, paddingHorizontal: SPACE.lg },
  together: { ...TYPE.title, color: COLORS.ink, textAlign: "center" },
  line: { ...TYPE.caption, color: COLORS.muted },
  fridges: { flexDirection: "row", gap: SPACE.md, paddingHorizontal: SPACE.lg },
  fridge: { flex: 1 },
  fridgeBody: { gap: SPACE.xs },
  owner: { flexDirection: "row", alignItems: "center", gap: SPACE.xs + SPACE.xxs },
  dot: { width: DOT, height: DOT, borderRadius: DOT / 2 },
  ownerName: { ...TYPE.bodyStrong, color: COLORS.ink, flexShrink: 1 },
  fridgeScore: { ...TYPE.title, color: COLORS.ink },
  combined: { alignItems: "center", gap: SPACE.md, paddingHorizontal: SPACE.lg },
  score: { alignItems: "center" },
  scoreNumber: { ...TYPE.hero, color: COLORS.ink },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "center",
    gap: SPACE.xs + SPACE.xxs,
  },
  chip: { paddingVertical: SPACE.xs, paddingHorizontal: SPACE.sm, borderRadius: RADII.pill },
  chipText: { ...TYPE.small },
  key: { flexDirection: "row", gap: SPACE.lg },
  ways: { gap: SPACE.sm },
  waiting: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.lg,
    padding: SPACE.xl,
  },
  players: { alignSelf: "stretch", gap: SPACE.md },
  status: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  waitingLine: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
  skeleton: { flexDirection: "row", gap: SPACE.md, padding: SPACE.lg },
});
