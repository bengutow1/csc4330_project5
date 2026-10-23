import { useProfile } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import type { ReactNode } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { ErrorState } from "@/components/error-state";
import { Header } from "@/components/header";
import { Skeleton } from "@/components/skeleton";
import { Sticker } from "@/components/sticker";
import { earnedLine } from "@/lib/profile";
import { COLORS, RADII, SPACE } from "@/theme";

// Two stickers to a row, leaning opposite ways, as the receipt sets them side by side.
const PER_ROW = 2;
const STICKER_SKELETON = 90;
const SKELETON_ROWS = ["first", "second", "third"] as const;

/**
 * BadgesScreen is P2: every badge as a sticker, in the server's order, opened from the Me tab.
 * An earned one is printed; a locked one is a faded blank that says how to earn it. Which are
 * earned is the server's call.
 */
export function BadgesScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const profile = useProfile();
  const badges = profile.data?.profile?.badges;

  let body: ReactNode;
  if (profile.isPending) {
    body = (
      <View style={styles.skeleton}>
        {SKELETON_ROWS.map((row) => (
          <Skeleton key={row} height={STICKER_SKELETON} radius={RADII.lg} />
        ))}
      </View>
    );
  } else if (badges === undefined) {
    body = (
      <ErrorState
        title="Couldn't load your badges"
        error={profile.error}
        retrying={profile.isFetching}
        onRetry={() => void profile.refetch()}
      />
    );
  } else {
    body = (
      <View style={styles.grid}>
        {badges.map((badge, i) => (
          <View key={badge.name} style={styles.cell}>
            <Sticker
              name={badge.name}
              line={badge.howToEarn}
              lean={i % 2 === 0 ? "left" : "right"}
              locked={!badge.earned}
            />
          </View>
        ))}
      </View>
    );
  }

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView contentContainerStyle={styles.content}>
        <Header
          title="Badges"
          subtitle={badges === undefined ? undefined : `${earnedLine(badges)} earned`}
          onBack={() => router.back()}
        />
        {body}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  // No top padding: the header sits right under the status bar, as the design draws it.
  content: { gap: SPACE.xl, paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xxl },
  skeleton: { gap: SPACE.md },
  grid: { flexDirection: "row", flexWrap: "wrap", rowGap: SPACE.xl },
  cell: { width: `${100 / PER_ROW}%`, alignItems: "center" },
});
