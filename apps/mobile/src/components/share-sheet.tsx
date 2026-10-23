import * as Sharing from "expo-sharing";
import { type ReactNode, useRef, useState } from "react";
import { PixelRatio, StyleSheet, Text, useWindowDimensions, View } from "react-native";
import { captureRef } from "react-native-view-shot";
import { Button } from "@/components/button";
import { Sheet } from "@/components/sheet";
import { CARD, captureSize, previewScale } from "@/lib/share";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The preview leaves the sheet room for its title and the Share button on a short phone.
const PREVIEW_HEIGHT = 0.55;

const UNAVAILABLE = "Sharing isn't available on this device.";
const FAILED = "Couldn't share that card. Try again.";

/**
 * ShareSheet is C6: a preview of a share card, then the phone's own share sheet with the card as
 * a story-sized picture. The card is captured at full size; only the preview is shrunk to fit.
 */
export function ShareSheet(props: {
  visible: boolean;
  title: string;
  /** What a screen reader hears for the preview ("Your verdict card"). */
  label: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const window = useWindowDimensions();
  const cardRef = useRef<View>(null);
  const [sharing, setSharing] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const scale = previewScale(CARD, {
    width: window.width - SPACE.lg * 2,
    height: window.height * PREVIEW_HEIGHT,
  });

  const share = async () => {
    setFailed(null);
    setSharing(true);
    try {
      const available = await Sharing.isAvailableAsync();
      if (!available) {
        setFailed(UNAVAILABLE);
        return;
      }
      const uri = await captureRef(cardRef, {
        format: "png",
        result: "tmpfile",
        ...captureSize(process.env.EXPO_OS, PixelRatio.get()),
      });
      await Sharing.shareAsync(uri, {
        mimeType: "image/png",
        UTI: "public.png",
        dialogTitle: props.title,
      });
    } catch {
      setFailed(FAILED);
    } finally {
      setSharing(false);
    }
  };

  const close = () => {
    setFailed(null);
    props.onClose();
  };

  return (
    <Sheet visible={props.visible} title={props.title} onClose={close}>
      <View
        accessible
        accessibilityRole="image"
        accessibilityLabel={props.label}
        style={[styles.preview, { width: CARD.width * scale, height: CARD.height * scale }]}
      >
        {/* The scale is on a parent, so the preview shrinks and the capture never does. */}
        <View style={[styles.scaler, { transform: [{ scale }] }]}>
          {/* Not collapsable, or Android flattens the view away and there is nothing to capture. */}
          <View ref={cardRef} collapsable={false}>
            {props.children}
          </View>
        </View>
      </View>
      <Button
        label="Share"
        icon="share-outline"
        pending={sharing}
        pendingLabel="Sharing…"
        onPress={() => void share()}
      />
      {failed ? (
        <Text accessibilityLiveRegion="polite" style={styles.error}>
          {failed}
        </Text>
      ) : null}
    </Sheet>
  );
}

const styles = StyleSheet.create({
  preview: {
    alignSelf: "center",
    overflow: "hidden",
    borderWidth: 1,
    borderColor: COLORS.line,
    borderRadius: RADII.xl,
  },
  scaler: { width: CARD.width, height: CARD.height, transformOrigin: "top left" },
  error: { ...TYPE.small, color: COLORS.ketchupInk, textAlign: "center" },
});
