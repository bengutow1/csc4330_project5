import Ionicons from "@expo/vector-icons/Ionicons";
import type { Spot } from "@fries-in-the-bag/api";
import { StyleSheet, Text, View } from "react-native";
import { Card } from "@/components/card";
import { metaLine, openLine, travelText } from "@/lib/spots";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const TILE = 44;
const TILE_ICON = 22;
const TRAVEL_ICON = 16;
const BADGE = { paddingVertical: 2, paddingHorizontal: 7, fontSize: 11 } as const;

/**
 * SpotRow is one spot in the list, as screen 9 draws it: a tile, the name, cuisine, price and
 * distance, whether it is open, and how long the trip takes. The nearest fries get the featured
 * card and their badge. The whole card opens the spot's sheet.
 */
export function SpotRow(props: { spot: Spot; nearest: boolean; onPress: () => void }) {
  const { spot, nearest } = props;
  const meta = metaLine(spot);
  const open = openLine(spot);
  const travel = travelText(spot);
  const label = [spot.name, nearest ? "nearest fries" : "", meta, open.text, travel]
    .filter((part) => part !== "")
    .join(", ");

  return (
    <Card featured={nearest} onPress={props.onPress} accessibilityLabel={label}>
      <View style={styles.row}>
        <View style={[styles.tile, nearest && styles.nearestTile]}>
          <Ionicons
            name={nearest ? "fast-food-outline" : "restaurant-outline"}
            size={TILE_ICON}
            color={nearest ? COLORS.ketchup : COLORS.muted}
          />
        </View>
        <View style={styles.words}>
          <View style={styles.nameLine}>
            <Text style={styles.name} numberOfLines={1}>
              {spot.name}
            </Text>
            {nearest ? <Text style={styles.badge}>Nearest fries</Text> : null}
          </View>
          {meta === "" ? null : <Text style={styles.meta}>{meta}</Text>}
          <Text style={[styles.open, !open.open && styles.closed]}>{open.text}</Text>
        </View>
        {travel === "" ? null : (
          <View style={styles.travel}>
            <Ionicons
              name={spot.walkable ? "walk-outline" : "car-outline"}
              size={TRAVEL_ICON}
              color={COLORS.muted}
            />
            <Text style={styles.travelText}>{travel}</Text>
          </View>
        )}
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: SPACE.md },
  tile: {
    width: TILE,
    height: TILE,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.md,
    backgroundColor: COLORS.sunken,
  },
  nearestTile: { backgroundColor: COLORS.ketchupTint },
  words: { flex: 1, gap: SPACE.xxs, minWidth: 0 },
  nameLine: {
    flexDirection: "row",
    alignItems: "center",
    flexWrap: "wrap",
    gap: SPACE.xs + SPACE.xxs,
  },
  name: { ...TYPE.bodyStrong, color: COLORS.ink, flexShrink: 1 },
  badge: {
    ...BADGE,
    fontWeight: "700",
    overflow: "hidden",
    borderRadius: RADII.pill,
    backgroundColor: COLORS.mustardTint,
    color: COLORS.mustardInk,
  },
  meta: { ...TYPE.caption, color: COLORS.muted },
  open: { ...TYPE.caption, fontWeight: TYPE.small.fontWeight, color: COLORS.pickle },
  closed: { color: COLORS.muted },
  travel: { flexDirection: "row", alignItems: "center", gap: SPACE.xs },
  travelText: { ...TYPE.caption, fontWeight: TYPE.small.fontWeight, color: COLORS.muted },
});
