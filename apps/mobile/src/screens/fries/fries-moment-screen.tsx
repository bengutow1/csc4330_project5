import Ionicons from "@expo/vector-icons/Ionicons";
import { useSpots } from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Card } from "@/components/card";
import { Logo } from "@/components/logo";
import { Skeleton } from "@/components/skeleton";
import { distanceText, STARTING_FILTERS, travelText } from "@/lib/spots";
import { FriesBag } from "@/screens/fries/fries-bag";
import { openDirections } from "@/screens/fries/spot-sheet";
import { usePlayerPoint } from "@/screens/fries/use-player-point";
import { COLORS, SPACE, TYPE } from "@/theme";

const CARD_ICON = 26;
const TRAVEL_ICON = 18;
// The nearest-fries card's height while it loads: two lines of text and its padding.
const CARD_SKELETON = 72;

/**
 * FriesMomentScreen is the fries moment (F1, screen 4): fries drop into a bag, and the screen
 * admits there's nothing to cook. Find food asks for location the first time and opens the list.
 * Once the app may look, screen 4's nearest-fries card comes in with Get directions.
 */
export function FriesMomentScreen() {
  const router = useRouter();
  const place = usePlayerPoint();
  // Sent only once there is a point; the list asks the same question, so it opens on the cache.
  const spots = useSpots({ at: place.point, filters: STARTING_FILTERS });
  const nearest = spots.data?.pages[0]?.nearestFries;
  const loadingNearest =
    place.granted && (place.locating || (place.point !== undefined && spots.isPending));

  const findFood = async () => {
    // Location is asked here, at the moment it's needed, and never before.
    if (!place.granted && place.canAsk) {
      await place.ask();
    }
    router.push("/fries/spots");
  };

  return (
    <View style={styles.root}>
      <Stack.Screen options={{ headerTitle: () => <Logo size="small" /> }} />
      <ScrollView contentContainerStyle={styles.content}>
        <FriesBag />
        <Text accessibilityRole="header" style={styles.title}>
          Looks like there's nothing to cook here.
        </Text>
        <Text style={styles.line}>Time to put the fries in the bag.</Text>
        {loadingNearest ? <Skeleton height={CARD_SKELETON} /> : null}
        {nearest ? (
          <Card>
            <View style={styles.nearest}>
              <Ionicons name="car-outline" size={CARD_ICON} color={COLORS.ketchup} />
              <View style={styles.nearestWords}>
                <Text style={styles.nearestName}>{nearest.name}</Text>
                <Text
                  style={styles.nearestAway}
                >{`${distanceText(nearest.distanceMeters)} away`}</Text>
              </View>
              <View style={styles.travel}>
                <Ionicons
                  name={nearest.walkable ? "walk-outline" : "car-outline"}
                  size={TRAVEL_ICON}
                  color={COLORS.muted}
                />
                <Text style={styles.travelText}>{travelText(nearest)}</Text>
              </View>
            </View>
          </Card>
        ) : null}
      </ScrollView>
      <BottomBar>
        {nearest ? (
          <Button label="Find food" variant="secondary" onPress={() => void findFood()} />
        ) : null}
        {nearest ? (
          <Button label="Get directions" icon="navigate" onPress={() => openDirections(nearest)} />
        ) : (
          <Button label="Find food" onPress={() => void findFood()} />
        )}
      </BottomBar>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.md, paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xl },
  title: { ...TYPE.title, color: COLORS.ink, textAlign: "center", marginTop: SPACE.sm },
  line: { ...TYPE.body, color: COLORS.inkSoft, textAlign: "center" },
  nearest: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  nearestWords: { flex: 1, gap: SPACE.xxs },
  nearestName: { ...TYPE.bodyStrong, color: COLORS.ink },
  nearestAway: { ...TYPE.small, fontWeight: TYPE.body.fontWeight, color: COLORS.muted },
  travel: { flexDirection: "row", alignItems: "center", gap: SPACE.xs },
  travelText: { ...TYPE.small, color: COLORS.muted },
});
