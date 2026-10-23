import Ionicons from "@expo/vector-icons/Ionicons";
import { StyleSheet, Text, View } from "react-native";
import { Card } from "@/components/card";
import { CheckPill } from "@/components/check-pill";
import { CountUp } from "@/components/count-up";
import { PhotoCard } from "@/components/photo-card";
import { PointsBreakdown } from "@/components/points-breakdown";
import { Sticker } from "@/components/sticker";
import {
  type Award,
  awardLines,
  type RankStep,
  rankWords,
  streakLine,
  takenCaption,
} from "@/lib/receipt";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// Screen 6's trophy circle beside the rank.
const TROPHY = 40;

/** What a passed plate earned, as the api's receipt carries it. */
export interface ReceiptData {
  /** What the check saw: "Looks like Creamy Tomato Pasta". */
  message: string;
  lines: readonly Award[];
  total: number;
  streakDays: number;
  newBadges: readonly { name: string; howToEarn: string }[];
  rankBefore?: RankStep;
  rankAfter?: RankStep;
}

/**
 * Receipt is what a passed plate earned (screen 6): the photo, what the check saw, the total
 * counting up over the awards printing one by one, the streak, any new badge as a sticker, and
 * where the plate left the player's rank. Every number on it is the server's.
 */
export function Receipt(props: { photo: { uri: string; at: Date }; receipt: ReceiptData }) {
  const receipt = props.receipt;
  const streak = streakLine(receipt.lines, receipt.streakDays);
  const rank = receipt.rankAfter ? rankWords(receipt.rankBefore, receipt.rankAfter) : null;

  return (
    <View style={styles.receipt}>
      <PhotoCard
        source={{ uri: props.photo.uri }}
        label="Your plate"
        caption={takenCaption(props.photo.at)}
      />
      <CheckPill label={receipt.message} />
      <View style={styles.total}>
        <CountUp value={receipt.total} prefix="+" style={styles.totalNumber} />
        <Text style={styles.caption}>points earned</Text>
      </View>
      <PointsBreakdown
        lines={awardLines(receipt.lines, receipt.streakDays)}
        total={receipt.total}
      />
      {streak ? (
        <View style={styles.streak}>
          <Ionicons name="flame-outline" size={16} color={COLORS.ketchup} />
          <Text style={styles.streakText}>{streak}</Text>
        </View>
      ) : null}
      {receipt.newBadges.length > 0 ? (
        <View style={styles.badges}>
          <Text accessibilityRole="header" style={styles.heading}>
            {receipt.newBadges.length === 1 ? "New badge" : "New badges"}
          </Text>
          <View style={styles.stickers}>
            {receipt.newBadges.map((badge, i) => (
              <Sticker
                key={badge.name}
                name={badge.name}
                line={badge.howToEarn}
                lean={i % 2 === 0 ? "left" : "right"}
              />
            ))}
          </View>
        </View>
      ) : null}
      {rank ? (
        <Card>
          <View style={styles.rank}>
            <View style={styles.trophy}>
              <Ionicons name="trophy-outline" size={20} color={COLORS.mustardInk} />
            </View>
            <View style={styles.rankWords}>
              <Text style={styles.rankTitle}>{rank.title}</Text>
              <Text style={styles.rankLine}>{rank.line}</Text>
            </View>
          </View>
        </Card>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  receipt: { gap: SPACE.lg, paddingHorizontal: SPACE.xl },
  total: { alignItems: "center", gap: SPACE.xs },
  totalNumber: { ...TYPE.hero, color: COLORS.ketchup },
  caption: { ...TYPE.body, color: COLORS.muted },
  streak: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.xs + SPACE.xxs,
  },
  streakText: { ...TYPE.bodyStrong, color: COLORS.ink },
  badges: { alignItems: "center", gap: SPACE.md },
  heading: { ...TYPE.heading, color: COLORS.ink },
  stickers: {
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "center",
    gap: SPACE.lg,
  },
  rank: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  trophy: {
    width: TROPHY,
    height: TROPHY,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.pill,
    backgroundColor: COLORS.mustardTint,
  },
  rankWords: { flex: 1, gap: SPACE.xxs },
  rankTitle: { ...TYPE.bodyStrong, color: COLORS.ink },
  rankLine: { ...TYPE.caption, color: COLORS.muted },
});
