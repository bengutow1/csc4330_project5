import type { ReactNode } from "react";
import { KeyboardAvoidingView, Modal, Pressable, StyleSheet, Text, View } from "react-native";
import Animated, { SlideInDown } from "react-native-reanimated";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { IconButton } from "@/components/icon-button";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const SLIDE_MS = 250;
const GRABBER = { width: 36, height: 5 };

/**
 * Sheet is a bottom sheet built on React Native's Modal: the ingredient editor, the spot sheet.
 * Tapping outside it, its close cross and Android's back button all call onClose, and the screen
 * decides what closing means. It slides up, or simply appears when the phone asks for reduced
 * motion.
 */
export function Sheet(props: {
  visible: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const insets = useSafeAreaInsets();
  return (
    <Modal
      visible={props.visible}
      transparent
      animationType="fade"
      statusBarTranslucent
      navigationBarTranslucent
      onRequestClose={props.onClose}
    >
      {/* Lifts the sheet over the keyboard, so a field inside it stays in sight on iOS. */}
      <KeyboardAvoidingView
        behavior={process.env.EXPO_OS === "ios" ? "padding" : undefined}
        style={styles.root}
      >
        {/* For touch only: a screen reader closes the sheet with its cross. */}
        <Pressable
          accessible={false}
          importantForAccessibility="no"
          onPress={props.onClose}
          style={styles.backdrop}
        />
        <Animated.View
          entering={SlideInDown.duration(SLIDE_MS)}
          accessibilityViewIsModal
          style={[styles.panel, { paddingBottom: insets.bottom + SPACE.lg }]}
        >
          <View style={styles.grabber} />
          <View style={styles.head}>
            <Text accessibilityRole="header" style={styles.title}>
              {props.title}
            </Text>
            <IconButton icon="close" label="Close" onPress={props.onClose} />
          </View>
          {props.children}
        </Animated.View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: "flex-end" },
  backdrop: { ...StyleSheet.absoluteFill, backgroundColor: COLORS.ink, opacity: 0.4 },
  panel: {
    gap: SPACE.md,
    paddingTop: SPACE.sm,
    paddingHorizontal: SPACE.lg,
    borderTopLeftRadius: RADII.xl,
    borderTopRightRadius: RADII.xl,
    backgroundColor: COLORS.surface,
  },
  grabber: {
    alignSelf: "center",
    width: GRABBER.width,
    height: GRABBER.height,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.lineStrong,
  },
  head: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  title: { ...TYPE.heading, color: COLORS.ink, flexShrink: 1 },
});
