import Ionicons from "@expo/vector-icons/Ionicons";
import { errorMessage, type Spot, useLogFryRun } from "@fries-in-the-bag/api";
import { Linking, StyleSheet, Text, View } from "react-native";
import { Button } from "@/components/button";
import { Sheet } from "@/components/sheet";
import { directionsUrl, metaLine, openLine, travelText } from "@/lib/spots";
import { COLORS, SPACE, TYPE } from "@/theme";

const LOGGED_ICON = 18;

/** openDirections hands the trip to the phone's own maps. */
export function openDirections(spot: Spot) {
  // An https link the phone can always open, in its maps app or its browser, so a refusal here
  // would be the system's own and there is nothing the player could do about it.
  Linking.openURL(directionsUrl(spot, process.env.EXPO_OS ?? "")).catch(() => undefined);
}

/**
 * SpotSheet is one spot up close (F3): when it is open until, how far and how pricey, where it is,
 * Directions in the phone's own maps, and Log fry run. The app has no toast yet, so the sheet says
 * itself that the run landed, or why it did not. Mount one per spot (key it by spot id), so a run
 * logged at one spot never shows on the next.
 */
export function SpotSheet(props: { spot: Spot; onClose: () => void }) {
  const { spot } = props;
  const log = useLogFryRun();
  const meta = metaLine(spot);
  const open = openLine(spot);
  const travel = travelText(spot);

  return (
    <Sheet visible title={spot.name} onClose={props.onClose}>
      <View style={styles.facts}>
        {meta === "" ? null : <Text style={styles.meta}>{meta}</Text>}
        <Text style={[styles.open, !open.open && styles.closed]}>{open.text}</Text>
        {travel === "" ? null : <Text style={styles.meta}>{travel}</Text>}
        {spot.address === "" ? null : <Text style={styles.address}>{spot.address}</Text>}
      </View>
      <Button label="Get directions" icon="navigate" onPress={() => openDirections(spot)} />
      {log.isSuccess ? (
        <View accessibilityLiveRegion="polite" style={styles.logged}>
          <Ionicons name="checkmark-circle" size={LOGGED_ICON} color={COLORS.pickle} />
          <Text style={styles.loggedText}>Fry run logged.</Text>
        </View>
      ) : (
        <Button
          label="Log fry run"
          pendingLabel="Logging…"
          variant="secondary"
          pending={log.isPending}
          onPress={() => log.mutate({ spotId: spot.spotId, spotName: spot.name })}
        />
      )}
      {log.isError ? <Text style={styles.error}>{errorMessage(log.error)}</Text> : null}
    </Sheet>
  );
}

const styles = StyleSheet.create({
  facts: { gap: SPACE.xs },
  meta: { ...TYPE.body, color: COLORS.muted },
  open: { ...TYPE.small, color: COLORS.pickle },
  closed: { color: COLORS.muted },
  address: { ...TYPE.body, color: COLORS.inkSoft },
  logged: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: SPACE.xs },
  loggedText: { ...TYPE.small, color: COLORS.pickle },
  error: { ...TYPE.small, color: COLORS.ketchupInk, textAlign: "center" },
});
