import Ionicons from "@expo/vector-icons/Ionicons";
import { errorMessage, fieldErrors, isUnreachable, useCreateScan } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Button } from "@/components/button";
import { Camera, type CameraShot } from "@/components/camera";
import { IconButton } from "@/components/icon-button";
import { Judging } from "@/components/judging";
import { Logo } from "@/components/logo";
import { PillToggle } from "@/components/pill-toggle";
import type { Size } from "@/lib/photo";
import {
  addShot,
  isFull,
  KINDS,
  kindLabel,
  removeShot,
  type Shot,
  type ShotKind,
  scanPhotos,
} from "@/lib/shots";
import { type PhotoSource, SCAN_PHOTO, shrinkPhoto } from "@/lib/shrink-photo";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

// The simulator has no camera, so a development build offers a bundled fridge to judge instead.
const SAMPLE_PHOTO = __DEV__ ? require("../../assets/dev/fridge.jpg") : null;
// What the viewfinder plays in a development build, where the simulator has no camera.
const DEMO_VIEW = __DEV__ ? require("../../assets/dev/fridge-view.gif") : undefined;

const WHY =
  "The camera sees what's in your fridge, freezer or pantry. Photos are judged, then deleted.";
const NO_CONNECTION = "Scans need a connection.";
const THUMB = 56;
const REMOVE = 24;
// Grows the small remove cross to the 44px a thumb hits reliably.
const REMOVE_SLOP = (44 - REMOVE) / 2;

/**
 * SnapScreen is the first tab: up to three photos of the kitchen, each tagged with the part it
 * shows, then the judging moment while the server reads them, then the verdict. A failed scan
 * keeps its photos, says why and offers Try again.
 */
export function SnapScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const create = useCreateScan();
  const [kind, setKind] = useState<ShotKind>("fridge");
  const [shots, setShots] = useState<readonly Shot[]>([]);
  const [lost, setLost] = useState(false);
  // Photos taken but not yet in the list, while they come off the camera and shrink.
  const [saving, setSaving] = useState(0);
  // Judge was tapped, and the scan waits for those photos so it carries every one taken before.
  const [asked, setAsked] = useState(false);
  const first = shots[0];

  // Other photos make another scan, so the last one's failure no longer applies. Nothing changes
  // the list while a scan is in flight: the scan waited for every save, then the camera hid.
  const change = (next: (list: readonly Shot[]) => readonly Shot[]) => {
    create.reset();
    setShots(next);
  };

  // Counted from the moment the shutter is pressed. The kind is read then too, so a shot keeps the
  // tag it was taken under while it shrinks.
  const keep = async (photo: Promise<{ source: PhotoSource; size: Size }>) => {
    const shotKind = kind;
    setLost(false);
    setSaving((n) => n + 1);
    try {
      const { source, size } = await photo;
      const shrunk = await shrinkPhoto(source, size, SCAN_PHOTO);
      change((list) => addShot(list, { ...shrunk, kind: shotKind }));
    } catch {
      setLost(true);
    } finally {
      setSaving((n) => n - 1);
    }
  };
  const fromCamera = (shot: Promise<CameraShot>) =>
    keep(shot.then((taken) => ({ source: taken.uri, size: taken })));
  const fromSample = () =>
    keep(Image.loadAsync(SAMPLE_PHOTO).then((image) => ({ source: image, size: image })));

  // The scan goes once nothing is left to save. It runs here, after the render that saw the last
  // photo land, so it sends the list as it is now rather than as it was at the tap.
  useEffect(() => {
    if (!asked || saving > 0) {
      return;
    }
    setAsked(false);
    // A photo that didn't save stops the scan here, so the player sees why before judging without it.
    if (lost) {
      return;
    }
    create.mutate(
      { photos: scanPhotos(shots) },
      {
        onSuccess: ({ verdict }) => {
          // The api always sends one; protobuf-es types every message field as optional.
          if (verdict === undefined) {
            return;
          }
          // Back from the verdict is a fresh camera, never the judging moment again.
          create.reset();
          setShots([]);
          router.push({ pathname: "/scan/[id]", params: { id: String(verdict.scanId) } });
        },
      },
    );
  }, [asked, saving, lost, shots, create, router]);

  // Photos still saving count toward the three, or a fourth would be taken and then dropped.
  const full = asked || isFull(shots.length, saving);
  const invalid = fieldErrors(create.error);
  const failure = create.isError
    ? (invalid.photos ?? (isUnreachable(create.error) ? NO_CONNECTION : errorMessage(create.error)))
    : null;

  return (
    <View style={[styles.root, { paddingTop: insets.top + SPACE.sm }]}>
      <View style={styles.head}>
        <Logo />
        <View>
          <Text style={styles.tagline}>See what you have. Cook something great.</Text>
          <Text style={styles.tagline}>Or put the fries in the bag.</Text>
        </View>
      </View>
      {create.isPending && first !== undefined ? (
        <Judging uri={first.uri} />
      ) : (
        // Once Judge is tapped the shutter rests, but the camera stays until every photo it took
        // has landed: a capture whose camera is torn down may never finish.
        <Camera
          why={WHY}
          demo={DEMO_VIEW}
          onShot={fromCamera}
          full={full}
          beside={
            SAMPLE_PHOTO && !full ? (
              <IconButton icon="image-outline" label="Use a sample photo" onPress={fromSample} />
            ) : null
          }
        >
          <PillToggle options={KINDS} value={kind} onChange={setKind} />
          {shots.length > 0 ? (
            <Shots shots={shots} onRemove={(i) => change((list) => removeShot(list, i))} />
          ) : null}
          {lost ? <Text style={styles.error}>That photo didn't save. Try again.</Text> : null}
        </Camera>
      )}
      {failure ? <Text style={styles.error}>{failure}</Text> : null}
      {first !== undefined ? (
        <View style={styles.action}>
          <Button
            label={create.isError ? "Try again" : "Judge my fridge"}
            pending={asked || create.isPending}
            pendingLabel="Judging…"
            onPress={() => {
              setLost(false);
              setAsked(true);
            }}
          />
        </View>
      ) : null}
    </View>
  );
}

/** Shots shows the photos taken so far, each with what it is of and a way to drop it. */
function Shots(props: { shots: readonly Shot[]; onRemove: (index: number) => void }) {
  return (
    <View style={styles.shots}>
      {props.shots.map((shot, index) => (
        <View key={shot.uri} style={styles.shot}>
          <Image
            accessible
            accessibilityLabel={`Photo ${index + 1}, ${kindLabel(shot.kind)}`}
            contentFit="cover"
            source={{ uri: shot.uri }}
            style={styles.thumb}
          />
          <Text style={styles.kind}>{kindLabel(shot.kind)}</Text>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Remove photo ${index + 1}`}
            hitSlop={REMOVE_SLOP}
            onPress={() => props.onRemove(index)}
            style={({ pressed }) => [styles.remove, pressed && styles.removePressed]}
          >
            <Ionicons name="close" size={14} color={COLORS.ink} />
          </Pressable>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, gap: SPACE.md, paddingBottom: SPACE.md, backgroundColor: COLORS.bg },
  head: { alignItems: "center", gap: SPACE.sm, paddingHorizontal: SPACE.xl },
  tagline: { ...TYPE.body, color: COLORS.inkSoft, textAlign: "center" },
  error: {
    ...TYPE.small,
    color: COLORS.ketchupInk,
    textAlign: "center",
    paddingHorizontal: SPACE.xl,
  },
  action: { paddingHorizontal: SPACE.lg },
  shots: { flexDirection: "row", justifyContent: "center", gap: SPACE.lg },
  shot: { alignItems: "center", gap: SPACE.xs },
  thumb: {
    width: THUMB,
    height: THUMB,
    borderRadius: RADII.md,
    backgroundColor: COLORS.sunken,
  },
  kind: { ...TYPE.caption, color: COLORS.muted },
  // On the thumbnail's corner, half over its edge.
  remove: {
    position: "absolute",
    top: -REMOVE / 3,
    right: -REMOVE / 3,
    width: REMOVE,
    height: REMOVE,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: RADII.pill,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
    backgroundColor: COLORS.surface,
  },
  removePressed: { backgroundColor: COLORS.sunken },
});
