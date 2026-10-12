import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import {
  ActivityIndicator,
  LogBox,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";

LogBox.ignoreAllLogs();

const C = {
  bg: "#140F0A",
  card: "#221A12",
  line: "#3A2D1F",
  gold: "#FFC23D",
  red: "#FF5A3C",
  green: "#4ADE80",
  text: "#FFF6E8",
  dim: "#B8A68C",
};

type Tab = "snap" | "board" | "me";

const VERDICTS = [
  {
    roast: true,
    score: 23,
    title: "ROASTED 🔥",
    line: "Ketchup, one egg and a lemon from last semester. You're one bad week away from putting the fries in the bag.",
    items: ["Ketchup", "1 egg", "Old lemon", "Energy drinks ×4", "Mystery container"],
    recipes: [
      {
        name: "Sad Egg on Toast",
        time: "5 min",
        emoji: "🍳",
        steps: "Fry the egg. Toast the bread. Reflect on your choices.",
      },
      {
        name: "Lemon Water (Fancy)",
        time: "1 min",
        emoji: "🍋",
        steps: "Squeeze lemon into water. Call it a detox.",
      },
    ],
  },
  {
    roast: false,
    score: 91,
    title: "CERTIFIED CHEF 👨‍🍳",
    line: "Fresh veggies, real protein and labeled leftovers? Your fridge has its life together. Respect.",
    items: ["Chicken", "Spinach", "Bell peppers", "Greek yogurt", "Eggs", "Cheddar"],
    recipes: [
      {
        name: "Chicken Fajita Bowl",
        time: "25 min",
        emoji: "🌯",
        steps: "Sear chicken and peppers, season, serve over rice.",
      },
      {
        name: "Spinach Cheddar Omelette",
        time: "10 min",
        emoji: "🍳",
        steps: "Whisk eggs, wilt spinach, fold in cheddar.",
      },
      {
        name: "Yogurt Parfait",
        time: "5 min",
        emoji: "🥣",
        steps: "Layer yogurt, fruit and granola.",
      },
    ],
  },
];

const BOARD = [
  { name: "chef_maya", score: 97, emoji: "🥗" },
  { name: "gymbro_tyler", score: 88, emoji: "🍗" },
  { name: "meal_prep_sam", score: 76, emoji: "🥦" },
  { name: "ramen_king", score: 41, emoji: "🍜" },
  { name: "dorm_fridge_dan", score: 12, emoji: "🧃" },
];

const RANGES = ["Today", "Week", "All time"];

function Button({
  label,
  onPress,
  ghost,
}: {
  label: string;
  onPress: () => void;
  ghost?: boolean;
}) {
  return (
    <Pressable
      onPress={onPress}
      style={({ pressed }) => [styles.button, ghost && styles.ghost, pressed && styles.pressed]}
    >
      <Text style={[styles.buttonText, ghost && { color: C.text }]}>{label}</Text>
    </Pressable>
  );
}

export function HomeScreen() {
  const [tab, setTab] = useState<Tab>("snap");
  const [phase, setPhase] = useState<"idle" | "scanning" | "result">("idle");
  const [snaps, setSnaps] = useState(0);
  const [posted, setPosted] = useState<number | null>(null);
  const [range, setRange] = useState("Today");
  const [open, setOpen] = useState<string | null>(null);
  // biome-ignore lint/style/noNonNullAssertion: always in range, the index is taken modulo the length
  const verdict = VERDICTS[(snaps + VERDICTS.length - 1) % VERDICTS.length]!;

  const snap = () => {
    setPhase("scanning");
    setOpen(null);
    setTimeout(() => {
      setSnaps((n) => n + 1);
      setPhase("result");
    }, 1800);
  };

  const offset = RANGES.indexOf(range) * 2;
  const board = [
    ...BOARD.map((p) => ({ ...p, score: Math.min(99, p.score + offset), me: false })),
    ...(posted === null ? [] : [{ name: "you", score: posted, emoji: "🍟", me: true }]),
  ].sort((a, b) => b.score - a.score);

  const snapScreen =
    phase === "result" ? (
      <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
        <View style={[styles.scoreCard, { borderColor: verdict.roast ? C.red : C.green }]}>
          <Text style={[styles.verdict, { color: verdict.roast ? C.red : C.green }]}>
            {verdict.title}
          </Text>
          <Text style={styles.score}>
            {verdict.score}
            <Text style={styles.outOf}>/100</Text>
          </Text>
          <Text style={styles.roastLine}>“{verdict.line}”</Text>
        </View>
        <Text style={styles.section}>Spotted in your fridge</Text>
        <View style={styles.chips}>
          {verdict.items.map((item) => (
            <View key={item} style={styles.chip}>
              <Text style={styles.chipText}>{item}</Text>
            </View>
          ))}
        </View>
        <Text style={styles.section}>Recipes you can make</Text>
        {verdict.recipes.map((r) => (
          <Pressable
            key={r.name}
            onPress={() => setOpen(open === r.name ? null : r.name)}
            style={({ pressed }) => [styles.recipe, pressed && styles.pressed]}
          >
            <View style={styles.recipeRow}>
              <Text style={styles.recipeEmoji}>{r.emoji}</Text>
              <View style={{ flex: 1 }}>
                <Text style={styles.recipeName}>{r.name}</Text>
                <Text style={styles.dim}>{r.time}</Text>
              </View>
              <Text style={styles.chev}>{open === r.name ? "⌄" : "›"}</Text>
            </View>
            {open === r.name && <Text style={styles.steps}>{r.steps}</Text>}
          </Pressable>
        ))}
        <View style={{ height: 8 }} />
        <Button
          label="🏆  Post to leaderboard"
          onPress={() => {
            setPosted(verdict.score);
            setTab("board");
          }}
        />
        <Button label="Snap again" ghost onPress={() => setPhase("idle")} />
      </ScrollView>
    ) : (
      <View style={styles.center}>
        <Text style={styles.logo}>🍟 Fries in the Bag</Text>
        <Text style={styles.sub}>Show us your fridge. The AI will judge you.</Text>
        <View style={styles.viewfinder}>
          {phase === "scanning" ? (
            <>
              <ActivityIndicator size="large" color={C.gold} />
              <Text style={styles.scanText}>Judging your fridge…</Text>
            </>
          ) : (
            <>
              <Text style={{ fontSize: 72 }}>🧊</Text>
              <Text style={styles.scanText}>Point your camera at your fridge</Text>
            </>
          )}
          <View
            style={[styles.corner, { top: 14, left: 14, borderTopWidth: 4, borderLeftWidth: 4 }]}
          />
          <View
            style={[styles.corner, { top: 14, right: 14, borderTopWidth: 4, borderRightWidth: 4 }]}
          />
          <View
            style={[
              styles.corner,
              { bottom: 14, left: 14, borderBottomWidth: 4, borderLeftWidth: 4 },
            ]}
          />
          <View
            style={[
              styles.corner,
              { bottom: 14, right: 14, borderBottomWidth: 4, borderRightWidth: 4 },
            ]}
          />
        </View>
        <Button label={phase === "scanning" ? "Analyzing…" : "📸  Snap my fridge"} onPress={snap} />
        <Text style={styles.hint}>Score high and get recipes. Score low and get roasted.</Text>
      </View>
    );

  const boardScreen = (
    <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
      <Text style={styles.title}>Leaderboard</Text>
      <Text style={styles.subLeft}>Best fridges on campus</Text>
      <View style={styles.segment}>
        {RANGES.map((r) => (
          <Pressable
            key={r}
            onPress={() => setRange(r)}
            style={[styles.segItem, range === r && styles.segOn]}
          >
            <Text style={[styles.segText, range === r && { color: C.bg }]}>{r}</Text>
          </Pressable>
        ))}
      </View>
      {board.map((p, i) => (
        <View key={p.name} style={[styles.row, p.me && styles.rowMe]}>
          <Text style={styles.rank}>{["🥇", "🥈", "🥉"][i] ?? `#${i + 1}`}</Text>
          <Text style={styles.avatar}>{p.emoji}</Text>
          <Text style={styles.name}>{p.me ? "you 👈" : p.name}</Text>
          <Text style={[styles.rowScore, { color: p.score >= 50 ? C.green : C.red }]}>
            {p.score}
          </Text>
        </View>
      ))}
      {posted === null && <Text style={styles.hint}>Snap your fridge to get on the board.</Text>}
    </ScrollView>
  );

  const meScreen = (
    <ScrollView
      contentContainerStyle={[styles.scroll, { alignItems: "center" }]}
      showsVerticalScrollIndicator={false}
    >
      <Text style={{ fontSize: 72 }}>🍟</Text>
      <Text style={styles.title}>@you</Text>
      <Text style={styles.sub}>Fridge rank: {snaps >= 2 ? "Line Cook" : "Fry Cook"}</Text>
      <View style={styles.stats}>
        {[
          ["Snaps", String(snaps)],
          ["Best", snaps >= 2 ? "91" : snaps === 1 ? "23" : "–"],
          ["Roasts", String(Math.ceil(snaps / 2))],
        ].map(([label, value]) => (
          <View key={label} style={styles.stat}>
            <Text style={styles.statValue}>{value}</Text>
            <Text style={styles.dim}>{label}</Text>
          </View>
        ))}
      </View>
      <Text style={[styles.section, { alignSelf: "flex-start" }]}>Badges</Text>
      {[
        ["🔥", "First Roast", snaps >= 1],
        ["👨‍🍳", "Certified Chef", snaps >= 2],
        ["🏆", "On the Board", posted !== null],
      ].map(([emoji, label, earned]) => (
        <View key={String(label)} style={[styles.row, { opacity: earned ? 1 : 0.35 }]}>
          <Text style={styles.avatar}>{emoji}</Text>
          <Text style={styles.name}>{label}</Text>
          <Text style={styles.dim}>{earned ? "Earned" : "Locked"}</Text>
        </View>
      ))}
    </ScrollView>
  );

  return (
    <View style={styles.root}>
      <StatusBar style="light" />
      <View style={styles.body}>
        {tab === "snap" ? snapScreen : tab === "board" ? boardScreen : meScreen}
      </View>
      <View style={styles.tabs}>
        {(
          [
            ["snap", "📸", "Snap"],
            ["board", "🏆", "Leaderboard"],
            ["me", "👤", "Me"],
          ] as const
        ).map(([id, icon, label]) => (
          <Pressable key={id} onPress={() => setTab(id)} style={styles.tab}>
            <Text style={{ fontSize: 22, opacity: tab === id ? 1 : 0.45 }}>{icon}</Text>
            <Text style={[styles.tabText, tab === id && { color: C.gold }]}>{label}</Text>
          </Pressable>
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: C.bg, paddingTop: 64 },
  body: { flex: 1 },
  center: {
    flex: 1,
    paddingHorizontal: 24,
    alignItems: "center",
    justifyContent: "center",
    gap: 14,
  },
  scroll: { padding: 20, gap: 12, paddingBottom: 40 },
  logo: { fontSize: 32, fontWeight: "900", color: C.gold, letterSpacing: -0.5 },
  title: { fontSize: 30, fontWeight: "900", color: C.text },
  sub: { fontSize: 15, color: C.dim, textAlign: "center" },
  subLeft: { fontSize: 15, color: C.dim, marginTop: -6 },
  hint: { fontSize: 13, color: C.dim, textAlign: "center", marginTop: 4 },
  viewfinder: {
    width: "100%",
    aspectRatio: 0.85,
    borderRadius: 28,
    backgroundColor: C.card,
    alignItems: "center",
    justifyContent: "center",
    gap: 14,
    marginVertical: 8,
  },
  corner: { position: "absolute", width: 34, height: 34, borderColor: C.gold, borderRadius: 6 },
  scanText: { color: C.text, fontSize: 16, fontWeight: "600" },
  button: {
    width: "100%",
    backgroundColor: C.gold,
    paddingVertical: 17,
    borderRadius: 18,
    alignItems: "center",
  },
  ghost: { backgroundColor: "transparent", borderWidth: 1.5, borderColor: C.line },
  pressed: { opacity: 0.7, transform: [{ scale: 0.98 }] },
  buttonText: { color: C.bg, fontSize: 18, fontWeight: "800" },
  scoreCard: {
    backgroundColor: C.card,
    borderRadius: 24,
    padding: 22,
    alignItems: "center",
    borderWidth: 2,
  },
  verdict: { fontSize: 22, fontWeight: "900", letterSpacing: 1 },
  score: { fontSize: 76, fontWeight: "900", color: C.text },
  outOf: { fontSize: 24, color: C.dim, fontWeight: "700" },
  roastLine: {
    color: C.text,
    fontSize: 16,
    textAlign: "center",
    lineHeight: 23,
    fontStyle: "italic",
  },
  section: {
    color: C.gold,
    fontSize: 13,
    fontWeight: "800",
    letterSpacing: 1.2,
    textTransform: "uppercase",
    marginTop: 8,
  },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  chip: {
    backgroundColor: C.card,
    borderColor: C.line,
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 7,
  },
  chipText: { color: C.text, fontSize: 14 },
  recipe: { backgroundColor: C.card, borderRadius: 18, padding: 14 },
  recipeRow: { flexDirection: "row", alignItems: "center", gap: 12 },
  recipeEmoji: { fontSize: 30 },
  recipeName: { color: C.text, fontSize: 16, fontWeight: "700" },
  steps: { color: C.dim, fontSize: 14, lineHeight: 20, marginTop: 10 },
  chev: { color: C.dim, fontSize: 26 },
  dim: { color: C.dim, fontSize: 13 },
  segment: {
    flexDirection: "row",
    backgroundColor: C.card,
    borderRadius: 14,
    padding: 4,
    marginVertical: 6,
  },
  segItem: { flex: 1, paddingVertical: 9, borderRadius: 10, alignItems: "center" },
  segOn: { backgroundColor: C.gold },
  segText: { color: C.dim, fontWeight: "700" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    backgroundColor: C.card,
    borderRadius: 16,
    padding: 14,
    width: "100%",
  },
  rowMe: { borderWidth: 2, borderColor: C.gold },
  rank: { width: 34, fontSize: 20, color: C.dim, fontWeight: "800", textAlign: "center" },
  avatar: { fontSize: 26 },
  name: { flex: 1, color: C.text, fontSize: 16, fontWeight: "700" },
  rowScore: { fontSize: 22, fontWeight: "900" },
  stats: { flexDirection: "row", gap: 10, marginVertical: 10 },
  stat: {
    flex: 1,
    backgroundColor: C.card,
    borderRadius: 16,
    paddingVertical: 14,
    alignItems: "center",
  },
  statValue: { color: C.text, fontSize: 26, fontWeight: "900" },
  tabs: {
    flexDirection: "row",
    borderTopWidth: 1,
    borderTopColor: C.line,
    backgroundColor: C.bg,
    paddingTop: 10,
    paddingBottom: 30,
  },
  tab: { flex: 1, alignItems: "center", gap: 2 },
  tabText: { color: C.dim, fontSize: 11, fontWeight: "700" },
});
