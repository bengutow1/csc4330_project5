import { StyleSheet, Text, View } from "react-native";
import { COLORS } from "@/theme";

// Screen 7's circles: a list row's, a podium step's, and the leader's in the middle.
const SIDES = { small: 34, medium: 56, large: 68 } as const;
// The player's own circle is ringed, as screen 7 rings "You".
const RING = 3;
// An emoji fills about half its circle.
const EMOJI_SCALE = 0.5;

/** Avatar is a player's emoji in a circle. The player's own is ringed in ketchup. */
export function Avatar(props: { avatar: string; mine: boolean; size: keyof typeof SIDES }) {
  const side = SIDES[props.size];
  return (
    <View
      style={[
        styles.circle,
        { width: side, height: side, borderRadius: side / 2 },
        props.mine && styles.mine,
      ]}
    >
      <Text style={{ fontSize: side * EMOJI_SCALE }}>{props.avatar}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  circle: { alignItems: "center", justifyContent: "center", backgroundColor: COLORS.sunken },
  mine: { borderWidth: RING, borderColor: COLORS.ketchup, backgroundColor: COLORS.ketchupTint },
});
