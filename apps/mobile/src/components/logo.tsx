import { LilitaOne_400Regular } from "@expo-google-fonts/lilita-one";
import { useFonts } from "expo-font";
import { Image } from "expo-image";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, FONTS, SPACE, TYPE } from "@/theme";

// The fries in a bag, drawn in the screen files and saved as an asset (docs/design.md).
const FRIES = require("../../assets/logo-fries.svg");

// Lilita One sits tight: the design sets both lines at 92% of their size.
const LEADING = 0.92;

// Large heads the Snap tab (screen 1); small sits in the fries moment's top bar (screen 4).
const SIZES = {
  large: {
    fries: TYPE.logo.fontSize,
    bag: TYPE.logoSmall.fontSize,
    art: { width: 48, height: 58 },
  },
  small: { fries: 30, bag: 16, art: { width: 30, height: 36 } },
} as const;

/** Logo is "Fries" over "in the Bag" in Lilita One, beside the fries in a bag. */
export function Logo(props: { size?: keyof typeof SIZES }) {
  // The logo is Lilita One's only user, so it loads the font. Text drawn before its font lands
  // keeps the fallback even after, so the wordmark mounts afresh once the font is in: O1 draws
  // the logo at launch.
  const [fontLoaded] = useFonts({ [FONTS.logo]: LilitaOne_400Regular });
  const size = SIZES[props.size ?? "large"];
  return (
    <View
      accessible
      accessibilityRole="image"
      accessibilityLabel="Fries in the Bag"
      style={styles.row}
    >
      <View key={fontLoaded ? "font" : "fallback"}>
        <Text style={[styles.word, { fontSize: size.fries, lineHeight: size.fries * LEADING }]}>
          Fries
        </Text>
        <Text
          style={[
            styles.word,
            styles.indent,
            { fontSize: size.bag, lineHeight: size.bag * LEADING },
          ]}
        >
          in the Bag
        </Text>
      </View>
      <Image source={FRIES} style={[styles.art, size.art]} contentFit="contain" />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "flex-start", gap: SPACE.xs },
  word: { fontFamily: TYPE.logo.fontFamily, color: COLORS.ink },
  indent: { marginLeft: SPACE.xxs },
  art: { marginTop: SPACE.xxs },
});
