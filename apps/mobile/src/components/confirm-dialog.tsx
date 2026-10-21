import { Modal, StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

/**
 * ConfirmDialog is the app's own confirm, for anything that cannot be undone or that affects the
 * other player. Never Alert.alert, which does nothing on web. It asks for an answer, so tapping
 * outside does nothing. While the confirmed write is in flight, the confirm button shows its
 * pending label and cancelling is ignored: the write is already on its way.
 */
export function ConfirmDialog(props: {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
  /** For what cannot be undone, such as deleting the player's data. */
  destructive?: boolean;
  /** True while the confirmed write is in flight. */
  pending?: boolean;
  /** What the confirm button says while pending ("Deleting…"). */
  pendingLabel?: string;
}) {
  const pending = props.pending ?? false;
  const cancel = () => {
    if (!pending) {
      props.onCancel();
    }
  };

  return (
    <Modal
      visible={props.visible}
      transparent
      animationType="fade"
      statusBarTranslucent
      navigationBarTranslucent
      onRequestClose={cancel}
    >
      <View style={styles.root}>
        <View style={styles.backdrop} />
        <View accessibilityViewIsModal style={styles.card}>
          <Text accessibilityRole="header" style={styles.title}>
            {props.title}
          </Text>
          <Text style={styles.message}>{props.message}</Text>
          <View style={styles.actions}>
            <View style={styles.action}>
              <Button label="Cancel" variant="secondary" onPress={cancel} />
            </View>
            <View style={styles.action}>
              <Button
                label={props.confirmLabel}
                pendingLabel={props.pendingLabel}
                pending={pending}
                variant={props.destructive ? "destructive" : "primary"}
                onPress={props.onConfirm}
              />
            </View>
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: "center", padding: SPACE.xl },
  backdrop: { ...StyleSheet.absoluteFill, backgroundColor: COLORS.ink, opacity: 0.4 },
  card: {
    gap: SPACE.sm,
    padding: SPACE.xl,
    borderRadius: RADII.xl,
    backgroundColor: COLORS.surface,
  },
  title: { ...TYPE.heading, color: COLORS.ink },
  message: { ...TYPE.body, color: COLORS.inkSoft },
  actions: { flexDirection: "row", gap: SPACE.md, marginTop: SPACE.md },
  action: { flex: 1 },
});
