import Ionicons from "@expo/vector-icons/Ionicons";
import { CameraView, type FlashMode, useCameraPermissions } from "expo-camera";
import { useIsFocused } from "expo-router";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { Linking, Pressable, StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { IconButton } from "@/components/icon-button";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The design's shutter (screen 1): a 76px circle in a 5px ketchup ring.
const SHUTTER = 76;
const RING = 5;
// Each frame guide is a corner: how long its arms are, how thick, and how far in from the edge.
const GUIDE = 28;
const GUIDE_WIDTH = 3;
const GUIDE_INSET = SPACE.lg;
// The design's controls row sits 28px in from each side.
const CONTROLS_INSET = SPACE.xl + SPACE.xs;

/** A photo straight from the camera, before it is shrunk for the trip. */
export interface CameraShot {
  uri: string;
  width: number;
  height: number;
}

/**
 * Camera is the viewfinder the Snap tab and plate it share: frame guides, the flash and the
 * shutter. It never asks for the camera on its own. It says why first and asks when the player
 * taps, and a refused camera gets the same why and the way to Settings. The preview runs only
 * while its screen is in front, because a phone runs one preview at a time and plate it opens over
 * the tabs.
 */
export function Camera(props: {
  /** Why the camera is needed, in the screen's words. */
  why: string;
  onShot: (shot: CameraShot) => void;
  /** The screen has all the photos it can use, so the shutter rests. */
  full?: boolean;
  /** Beside the shutter, where the design draws the photo library. */
  beside?: ReactNode;
  /** Between the viewfinder and the shutter: the screen's own controls. */
  children?: ReactNode;
}) {
  const [permission, requestPermission] = useCameraPermissions();
  const focused = useIsFocused();
  const camera = useRef<CameraView>(null);
  const [flash, setFlash] = useState<FlashMode>("off");
  const [ready, setReady] = useState(false);
  const [broken, setBroken] = useState(false);
  const [shooting, setShooting] = useState(false);
  const [missed, setMissed] = useState(false);
  const granted = permission?.granted === true;
  const live = granted && focused;

  // A preview that stopped starts from scratch: it warms up again, and may work this time.
  useEffect(() => {
    if (!live) {
      setReady(false);
      setBroken(false);
    }
  }, [live]);

  const shoot = async () => {
    const view = camera.current;
    if (view === null) {
      return;
    }
    setShooting(true);
    setMissed(false);
    try {
      const photo = await view.takePictureAsync();
      props.onShot({ uri: photo.uri, width: photo.width, height: photo.height });
    } catch {
      setMissed(true);
    } finally {
      setShooting(false);
    }
  };

  return (
    <View style={styles.root}>
      <View style={styles.viewfinder}>
        {permission === null ? null : granted ? (
          <>
            {live && !broken ? (
              <CameraView
                ref={camera}
                style={StyleSheet.absoluteFill}
                facing="back"
                flash={flash}
                onCameraReady={() => setReady(true)}
                onMountError={() => setBroken(true)}
              />
            ) : null}
            {broken ? (
              <View style={styles.panel}>
                <Text style={styles.panelLine}>The camera didn't start.</Text>
              </View>
            ) : (
              <Guides />
            )}
            <View style={styles.flash}>
              <IconButton
                icon={flash === "on" ? "flash-outline" : "flash-off-outline"}
                label={flash === "on" ? "Turn the flash off" : "Turn the flash on"}
                onPress={() => setFlash(flash === "on" ? "off" : "on")}
              />
            </View>
          </>
        ) : (
          <View style={styles.panel}>
            <Ionicons name="camera-outline" size={32} color={COLORS.ink} />
            <Text style={styles.panelLine}>{props.why}</Text>
            {permission.canAskAgain ? (
              <Button
                label="Allow the camera"
                size="small"
                onPress={() => void requestPermission()}
              />
            ) : (
              <>
                <Text style={styles.panelLine}>
                  The camera is off for this app. Turn it on in Settings.
                </Text>
                <Button
                  label="Open Settings"
                  size="small"
                  onPress={() => void Linking.openSettings()}
                />
              </>
            )}
          </View>
        )}
      </View>
      {props.children}
      {missed ? <Text style={styles.error}>That photo didn't take. Try again.</Text> : null}
      <View style={styles.controls}>
        <View style={styles.side}>{props.beside}</View>
        {granted ? (
          <Shutter disabled={!ready || shooting || props.full === true} onPress={shoot} />
        ) : (
          <View style={styles.shutterSpace} />
        )}
        <View style={styles.side} />
      </View>
    </View>
  );
}

/** Guides marks the four corners a kitchen should sit inside. */
function Guides() {
  return (
    <View style={styles.guides}>
      <View style={[styles.corner, styles.topLeft]} />
      <View style={[styles.corner, styles.topRight]} />
      <View style={[styles.corner, styles.bottomLeft]} />
      <View style={[styles.corner, styles.bottomRight]} />
    </View>
  );
}

/** Shutter takes the photo. It rests while the camera warms up or the screen has enough. */
function Shutter(props: { disabled: boolean; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Take photo"
      accessibilityState={{ disabled: props.disabled }}
      disabled={props.disabled}
      onPress={props.onPress}
      style={({ pressed }) => [
        styles.shutter,
        pressed && styles.shutterPressed,
        props.disabled && styles.shutterResting,
      ]}
    />
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, gap: SPACE.md },
  viewfinder: { flex: 1, minHeight: 200, overflow: "hidden", backgroundColor: COLORS.sunken },
  panel: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.md,
    padding: SPACE.xl,
  },
  panelLine: { ...TYPE.body, color: COLORS.inkSoft, textAlign: "center" },
  flash: {
    position: "absolute",
    top: SPACE.md,
    right: SPACE.md,
    borderRadius: RADII.pill,
    backgroundColor: COLORS.surface,
  },
  guides: {
    position: "absolute",
    top: GUIDE_INSET,
    right: GUIDE_INSET,
    bottom: GUIDE_INSET,
    left: GUIDE_INSET,
    pointerEvents: "none",
  },
  corner: { position: "absolute", width: GUIDE, height: GUIDE, borderColor: COLORS.surface },
  topLeft: {
    top: 0,
    left: 0,
    borderTopWidth: GUIDE_WIDTH,
    borderLeftWidth: GUIDE_WIDTH,
    borderTopLeftRadius: RADII.sm,
  },
  topRight: {
    top: 0,
    right: 0,
    borderTopWidth: GUIDE_WIDTH,
    borderRightWidth: GUIDE_WIDTH,
    borderTopRightRadius: RADII.sm,
  },
  bottomLeft: {
    bottom: 0,
    left: 0,
    borderBottomWidth: GUIDE_WIDTH,
    borderLeftWidth: GUIDE_WIDTH,
    borderBottomLeftRadius: RADII.sm,
  },
  bottomRight: {
    bottom: 0,
    right: 0,
    borderBottomWidth: GUIDE_WIDTH,
    borderRightWidth: GUIDE_WIDTH,
    borderBottomRightRadius: RADII.sm,
  },
  error: { ...TYPE.small, color: COLORS.ketchupInk, textAlign: "center" },
  controls: { flexDirection: "row", alignItems: "center", paddingHorizontal: CONTROLS_INSET },
  side: { flex: 1, alignItems: "flex-start" },
  shutter: {
    width: SHUTTER,
    height: SHUTTER,
    borderRadius: RADII.pill,
    borderWidth: RING,
    borderColor: COLORS.ketchup,
    backgroundColor: COLORS.surface,
  },
  shutterPressed: { backgroundColor: COLORS.sunken },
  shutterResting: { opacity: 0.4 },
  shutterSpace: { width: SHUTTER, height: SHUTTER },
});
