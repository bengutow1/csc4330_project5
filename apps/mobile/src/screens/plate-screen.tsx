import { errorMessage, isNotFound, isUnreachable, usePlateMeal } from "@fries-in-the-bag/api";
import { Image } from "expo-image";
import { Stack, useRouter } from "expo-router";
import { type ReactNode, useEffect, useState } from "react";
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { BottomBar } from "@/components/bottom-bar";
import { Button } from "@/components/button";
import { Camera, type CameraShot } from "@/components/camera";
import { EmptyState } from "@/components/empty-state";
import { Header } from "@/components/header";
import { IconButton } from "@/components/icon-button";
import { PhotoCard } from "@/components/photo-card";
import { Receipt } from "@/components/receipt";
import { type Cut, jpegBytes, type Size, sampleCut } from "@/lib/photo";
import { takenCaption } from "@/lib/receipt";
import { parseId } from "@/lib/route-id";
import { type PhotoSource, PLATE_PHOTO, shrinkPhoto } from "@/lib/shrink-photo";
import { COLORS, SPACE, TYPE } from "@/theme";

// The simulator has no camera, so a development build offers the same bundled photo as Snap.
const SAMPLE_PHOTO = __DEV__ ? require("../../assets/dev/fridge.jpg") : null;

const WHY = "The camera checks the dish you made. Plate photos are kept for your profile.";
const NO_CONNECTION = "Plates need a connection.";
const BAR_HEIGHT = 44;

/** The plate photo as it left the phone, and when it was taken. */
interface PlatePhoto {
  uri: string;
  base64: string;
  at: Date;
}

/**
 * PlateScreen is plate it, then the receipt (C4, C5): one photo of the dish, which the server
 * checks against the recipe, then what the plate paid. A plate the check turns down says why and
 * offers another photo. Skip, Done and the close cross all end on Snap, the app's kitchen while
 * the kitchen home is on hold (docs/design.md).
 */
export function PlateScreen(props: { recipeId: string | undefined }) {
  const recipeId = parseId(props.recipeId);
  if (recipeId === undefined) {
    return <NotFound />;
  }
  return <Plating recipeId={recipeId} />;
}

function Plating(props: { recipeId: bigint }) {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const plate = usePlateMeal();
  const [photo, setPhoto] = useState<PlatePhoto | null>(null);
  const [saving, setSaving] = useState(false);
  const [lost, setLost] = useState(false);

  const leave = () => router.dismissTo("/");

  // Timed from the shutter, though the photo lands a moment later.
  const keep = async (shot: Promise<{ source: PhotoSource; size: Size; cut?: Cut }>) => {
    const at = new Date();
    setLost(false);
    setSaving(true);
    try {
      const { source, size, cut } = await shot;
      const shrunk = await shrinkPhoto(source, size, PLATE_PHOTO, cut);
      setPhoto({ ...shrunk, at });
    } catch {
      setLost(true);
    } finally {
      setSaving(false);
    }
  };

  // Plate it takes one photo, so it goes once it lands. It goes from here, after a render, so a
  // photo that lands after the player skipped is never sent. Each send spends one of the
  // player's checks, and none is sent twice: another try is another photo.
  const { isIdle, mutate } = plate;
  useEffect(() => {
    if (photo !== null && isIdle) {
      mutate({ recipeId: props.recipeId, photo: jpegBytes(photo.base64) });
    }
  }, [photo, isIdle, mutate, props.recipeId]);
  const fromCamera = (shot: Promise<CameraShot>) =>
    keep(shot.then((taken) => ({ source: taken.uri, size: taken })));
  const fromSample = () =>
    keep(
      Image.loadAsync(SAMPLE_PHOTO).then((image) => ({
        source: image,
        size: image,
        cut: sampleCut(image, Math.random),
      })),
    );
  const again = () => {
    plate.reset();
    setPhoto(null);
  };

  // One photo at a time: from the press until it is sent, the shutter and the sample rest.
  const busy = saving || photo !== null;
  const receipt = plate.data?.receipt;
  if (photo !== null && plate.isPending) {
    return (
      <Frame top={insets.top}>
        <Header title="Plate it" />
        <View style={styles.middle}>
          <PhotoCard
            source={{ uri: photo.uri }}
            label="Your plate"
            caption={takenCaption(photo.at)}
          />
          <View style={styles.checking}>
            <ActivityIndicator color={COLORS.ketchup} />
            <Text accessibilityLiveRegion="polite" style={styles.checkingLine}>
              Checking your plate…
            </Text>
          </View>
        </View>
      </Frame>
    );
  }
  if (photo !== null && receipt?.passed) {
    return (
      <View style={styles.root}>
        <Stack.Screen options={{ headerShown: false }} />
        <ScrollView contentContainerStyle={[styles.content, { paddingTop: insets.top }]}>
          <View style={styles.bar}>
            <IconButton icon="close" label="Close" onPress={leave} />
          </View>
          <Receipt photo={photo} receipt={receipt} />
        </ScrollView>
        <BottomBar>
          <Button label="Done" onPress={leave} />
        </BottomBar>
      </View>
    );
  }
  if (photo !== null && plate.isSuccess) {
    // Turned down: the plate earned nothing and the server stored nothing. The api always sends
    // a receipt; protobuf-es types every message field as optional.
    const message = receipt?.message ?? errorMessage(undefined);
    return (
      <Retry top={insets.top} photo={photo} message={message} onRetry={again} onSkip={leave} />
    );
  }
  if (plate.isError) {
    if (isNotFound(plate.error)) {
      return <NotFound />;
    }
    const message = isUnreachable(plate.error) ? NO_CONNECTION : errorMessage(plate.error);
    return (
      <Retry top={insets.top} photo={photo} message={message} onRetry={again} onSkip={leave} />
    );
  }
  return (
    <Frame top={insets.top}>
      <Header title="Plate it" subtitle="Snap what you made." onClose={leave} />
      {/* The camera stays until its photo has landed: a capture whose camera is torn down may
          never finish. */}
      <Camera
        why={WHY}
        onShot={fromCamera}
        full={busy}
        beside={
          SAMPLE_PHOTO && !busy ? (
            <IconButton icon="image-outline" label="Use a sample photo" onPress={fromSample} />
          ) : null
        }
      >
        {lost ? <Text style={styles.error}>That photo didn't save. Try again.</Text> : null}
      </Camera>
      <BottomBar>
        <Button label="Skip (no points)" variant="secondary" onPress={leave} />
      </BottomBar>
    </Frame>
  );
}

/** Retry is a plate that didn't pay: the photo, why, and another go or a way out. */
function Retry(props: {
  top: number;
  photo: PlatePhoto | null;
  message: string;
  onRetry: () => void;
  onSkip: () => void;
}) {
  return (
    <Frame top={props.top}>
      <Header title="Plate it" onClose={props.onSkip} />
      <View style={styles.middle}>
        {props.photo ? (
          <PhotoCard
            source={{ uri: props.photo.uri }}
            label="Your plate"
            caption={takenCaption(props.photo.at)}
          />
        ) : null}
        <Text style={styles.message}>{props.message}</Text>
      </View>
      <BottomBar>
        <View style={styles.ways}>
          <Button label="Try again" onPress={props.onRetry} />
          <Button
            label="Skip (no points)"
            variant="secondary"
            size="small"
            onPress={props.onSkip}
          />
        </View>
      </BottomBar>
    </Frame>
  );
}

/** NotFound answers a recipe that isn't this player's, or never was. */
function NotFound() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const leave = () => router.dismissTo("/");
  return (
    <Frame top={insets.top}>
      <View style={styles.bar}>
        <IconButton icon="close" label="Close" onPress={leave} />
      </View>
      <EmptyState
        title="No recipe here"
        line="This recipe doesn't exist, or it isn't yours."
        action={{ label: "Scan your fridge", onPress: leave }}
      />
    </Frame>
  );
}

/** Frame is a step of plating with no native header, clear of the status bar. */
function Frame(props: { top: number; children: ReactNode }) {
  return (
    <View style={[styles.root, { paddingTop: props.top }]}>
      <Stack.Screen options={{ headerShown: false }} />
      {props.children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, gap: SPACE.md, backgroundColor: COLORS.bg },
  content: { gap: SPACE.sm, paddingBottom: SPACE.xl },
  bar: {
    height: BAR_HEIGHT,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: SPACE.sm,
  },
  middle: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: SPACE.xl,
    paddingHorizontal: SPACE.xl,
  },
  checking: { alignItems: "center", gap: SPACE.sm },
  checkingLine: { ...TYPE.body, color: COLORS.muted },
  message: { ...TYPE.bodyStrong, color: COLORS.ink, textAlign: "center" },
  error: {
    ...TYPE.small,
    color: COLORS.ketchupInk,
    textAlign: "center",
    paddingHorizontal: SPACE.xl,
  },
  ways: { gap: SPACE.sm },
});
