import type { ReactNode } from "react";
import { StyleSheet, Text, View } from "react-native";
import { COLORS, SPACE, TYPE } from "@/theme";

/** FormSection is one question on a form: its heading, a line on why it matters, its answers. */
export function FormSection(props: { title: string; note?: string; children: ReactNode }) {
  return (
    <View style={styles.section}>
      <View style={styles.titles}>
        <Text accessibilityRole="header" style={styles.heading}>
          {props.title}
        </Text>
        {props.note ? <Text style={styles.note}>{props.note}</Text> : null}
      </View>
      {props.children}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: SPACE.md },
  titles: { gap: SPACE.xs },
  heading: { ...TYPE.heading, color: COLORS.ink },
  note: { ...TYPE.caption, color: COLORS.muted },
});
