import { Image, type ImageProps } from "expo-image";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const PHOTO_SIZE = 180;
// The design's frame is a little rounder than a card, and turned like a print left on a table.
const FRAME_RADIUS = 18;
const TILT = "-3deg";

/**
 * PhotoCard shows the plate photo in a white frame, turned a little, with when it was taken under
 * it (screen 6). The label tells a screen reader what the photo shows.
 */
export function PhotoCard(props: { source: ImageProps["source"]; label: string; caption: string }) {
  return (
    <View style={styles.frame}>
      <Image
        source={props.source}
        accessible
        accessibilityRole="image"
        accessibilityLabel={props.label}
        contentFit="cover"
        style={styles.photo}
      />
      <Text style={styles.caption}>{props.caption}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  frame: {
    alignSelf: "center",
    gap: SPACE.sm,
    paddingTop: SPACE.sm + SPACE.xxs,
    paddingHorizontal: SPACE.sm + SPACE.xxs,
    paddingBottom: SPACE.md,
    borderRadius: FRAME_RADIUS,
    backgroundColor: COLORS.surface,
    shadowColor: COLORS.ink,
    shadowOpacity: 0.12,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 8 },
    elevation: 6,
    transform: [{ rotate: TILT }],
  },
  photo: { width: PHOTO_SIZE, height: PHOTO_SIZE, borderRadius: RADII.md },
  caption: { ...TYPE.caption, color: COLORS.muted, textAlign: "center" },
});
