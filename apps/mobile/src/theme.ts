import type { TextStyle } from "react-native";

/**
 * The design's colour tokens (docs/design.md): components take colours from here, and no hex
 * value appears anywhere else in the app. Illustration colours live in the screen files.
 */
export const COLORS = {
  bg: "#FBF8F4",
  surface: "#FFFFFF",
  ink: "#1E1A17",
  inkSoft: "#2B2420",
  muted: "#6E655D",
  line: "#EEE6DC",
  lineStrong: "#DCD1C4",
  divider: "#F2EBE2",
  sunken: "#EFE7DD",
  ketchup: "#C8322B",
  ketchupInk: "#A8261F",
  ketchupTint: "#FBEAE7",
  mustard: "#F7C531",
  mustardInk: "#8A5A00",
  mustardTint: "#FFF1D6",
  pickle: "#23794A",
  pickleTint: "#E8F4EC",
  blueberry: "#1D4FA3",
  blueberryTint: "#E7EEFA",
  bag: "#D62F27",
} as const;

/** Spacing in points, on the steps the design's padding and gaps use most. */
export const SPACE = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
} as const;

/** Corner radii: the segmented control (sm), tiles (md), buttons (lg), cards (xl) and pills. */
export const RADII = {
  sm: 9,
  md: 12,
  lg: 14,
  xl: 16,
  pill: 999,
} as const;

/** The one custom font, Lilita One, for the logo and nothing else. The root layout loads it. */
export const FONTS = {
  logo: "LilitaOne_400Regular",
} as const;

/**
 * The design's type styles. Everything but the logo is the system font (SF Pro on iOS). A style
 * carries no colour: a component pairs it with a colour token.
 */
export const TYPE = {
  logo: { fontFamily: FONTS.logo, fontSize: 46 },
  logoSmall: { fontFamily: FONTS.logo, fontSize: 25 },
  hero: { fontSize: 52, fontWeight: "800" },
  title: { fontSize: 24, fontWeight: "800" },
  titleRoot: { fontSize: 26, fontWeight: "800" },
  heading: { fontSize: 20, fontWeight: "800" },
  button: { fontSize: 17, fontWeight: "700" },
  buttonSecondary: { fontSize: 16, fontWeight: "700" },
  bodyStrong: { fontSize: 15, fontWeight: "700" },
  body: { fontSize: 15, fontWeight: "400" },
  small: { fontSize: 14, fontWeight: "600" },
  caption: { fontSize: 13, fontWeight: "400" },
  tab: { fontSize: 11, fontWeight: "600" },
  tabActive: { fontSize: 11, fontWeight: "700" },
} as const satisfies Record<string, TextStyle>;
