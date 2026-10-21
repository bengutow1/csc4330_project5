import { StyleSheet, Text, View } from "react-native";
import { IconButton } from "@/components/icon-button";
import { COLORS, SPACE, TYPE } from "@/theme";

const BAR_HEIGHT = 44;

/**
 * Header tops a screen that draws its own: a back arrow (a close cross ends a flow, as on the
 * points screen), then the centred title and a muted line under it. A tab's root has no arrow and
 * a bigger title. The screen keeps it clear of the status bar.
 */
export function Header(props: {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  onClose?: () => void;
  /** A tab's first screen: no arrow, and the bigger title. */
  root?: boolean;
}) {
  return (
    <View>
      {props.root ? null : (
        <View style={styles.bar}>
          {props.onBack ? (
            <IconButton icon="arrow-back" label="Back" onPress={props.onBack} />
          ) : null}
          {props.onClose ? <IconButton icon="close" label="Close" onPress={props.onClose} /> : null}
        </View>
      )}
      <View style={[styles.titles, props.root && styles.rootTitles]}>
        <Text
          accessibilityRole="header"
          style={[props.root ? TYPE.titleRoot : TYPE.title, styles.title]}
        >
          {props.title}
        </Text>
        {props.subtitle ? <Text style={styles.subtitle}>{props.subtitle}</Text> : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: "row",
    alignItems: "center",
    height: BAR_HEIGHT,
    paddingHorizontal: SPACE.sm,
  },
  titles: {
    alignItems: "center",
    gap: SPACE.xs + SPACE.xxs,
    paddingTop: SPACE.xxs,
    paddingHorizontal: SPACE.xxl + SPACE.sm,
  },
  rootTitles: { gap: SPACE.xxs, paddingTop: SPACE.sm },
  title: { color: COLORS.ink, textAlign: "center" },
  subtitle: { ...TYPE.body, color: COLORS.muted, textAlign: "center" },
});
